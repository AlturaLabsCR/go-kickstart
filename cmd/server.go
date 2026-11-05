// Copyright © 2026 NAME HERE <EMAIL ADDRESS>

package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/mail"
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
)

type serverConfig struct {
	DatabaseDSN     string
	Dev             bool
	LogLevel        string
	LogFormat       string
	AuthSecret      string
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

	if len(cfg.AuthSecret) < 32 {
		logger.Warn("JWT signing secret is shorter than the recommended 32 random bytes; tokens may be forged", "secret_bytes", len(cfg.AuthSecret))
	}

	if cfg.DatabaseDSN == "" {
		return errors.New("database DSN is required")
	}

	if _, err := mail.ParseAddress(cfg.MailFrom); err != nil {
		return fmt.Errorf("invalid mail sender: %w", err)
	}

	var db database.Database
	db, err = provider.Open(context.Background(), cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	cacheStore := memory.New()
	defer cacheStore.Close()
	db = cached.New(db, cacheStore)
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("database close error", "error", err)
		}
	}()

	localizer, err := i18n.NewLocalizer(locales.Locales)
	if err != nil {
		return err
	}

	authenticator, err := auth.NewAuthenticator(db, cfg.AuthSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	if err != nil {
		return err
	}

	// Replace the noop provider below with a go-email SMTP or Mailgun client.
	h := handlers.NewHandler(handlers.Options{
		Logger:        logger,
		Dev:           cfg.Dev,
		DB:            db,
		Authenticator: authenticator,
		Localizer:     localizer,
		RootPrefix:    cfg.RootPrefix,
		Mailer:        noop.Mailer{},
		MailFrom:      cfg.MailFrom,
	})

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
