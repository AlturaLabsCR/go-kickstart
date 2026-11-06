package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"app/auth"
	"app/cache/memory"
	"app/database"
	"app/database/cached"
	"app/database/provider"
	"app/handlers"
	locales "app/i18n"
	"app/mailer/noop"
	"github.com/tavocg/go-i18n"
	secrets "github.com/tavocg/go-secrets"
	"github.com/tavocg/go-secrets/otp"
)

type serverConfig struct {
	DatabaseDSN     string
	Dev             bool
	LogLevel        string
	LogFormat       string
	AuthSecret      string
	OTPSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	RootPrefix      string
	MailFrom        string
	Host            string
	Port            int
}

func runServer(cfg serverConfig) error {
	logger, err := newLogger(cfg.Dev, cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		return err
	}

	for _, secret := range []struct{ name, value string }{
		{"auth.secret", cfg.AuthSecret},
		{"otp.secret", cfg.OTPSecret},
	} {
		if len(secret.value) >= 32 {
			continue
		}
		if !cfg.Dev {
			return fmt.Errorf("%s must contain at least 32 bytes outside development mode", secret.name)
		}
		logger.Warn("short signing secret allowed only in development", "setting", secret.name, "secret_bytes", len(secret.value))
	}
	if cfg.OTPSecret != "" && cfg.OTPSecret == cfg.AuthSecret {
		return errors.New("OTP HMAC secret must differ from the JWT signing secret")
	}

	otpSecret := cfg.OTPSecret
	if otpSecret == "" {
		// Development only: the signer requires a key. Pending codes will not
		// survive a restart unless a stable OTP secret is configured.
		otpSecret, err = secrets.RandStr()
		if err != nil {
			return err
		}
	}

	otpSigner, err := otp.NewSigner(otpSecret)
	if err != nil {
		return err
	}

	var db database.Database
	db, err = provider.Open(context.Background(), cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	cacheStore := memory.New()
	defer func() {
		if err := cacheStore.Close(); err != nil {
			logger.Error("cache close error", "error", err)
		}
	}()
	db = cached.New(db, cacheStore)
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("database close error", "error", err)
		}
	}()

	localizer, err := i18n.NewLocalizer(locales.Locales())
	if err != nil {
		return err
	}

	authenticator, err := auth.NewAuthenticator(db, cfg.AuthSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	if err != nil {
		return err
	}

	// Replace the noop provider below with a go-email SMTP or Mailgun client.
	h, err := handlers.NewHandler(handlers.Options{
		Logger:        logger,
		Dev:           cfg.Dev,
		DB:            db,
		Authenticator: authenticator,
		Localizer:     localizer,
		RootPrefix:    cfg.RootPrefix,
		Mailer:        noop.Mailer{},
		MailFrom:      cfg.MailFrom,
		OTPSigner:     otpSigner,
	})
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}

	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	stopMaintenance := startDatabaseMaintenance(ctx, db, logger)
	defer stopMaintenance()

	logger.Info("listening", "addr", listener.Addr().String())
	return serveHTTP(ctx, srv, listener)
}
