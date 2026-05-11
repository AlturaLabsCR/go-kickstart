// Copyright © 2026 NAME HERE <EMAIL ADDRESS>

package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	appauth "github.com/myrepo/myserver/auth"
	"github.com/myrepo/myserver/database"
	"github.com/myrepo/myserver/database/provider"
	"github.com/myrepo/myserver/handlers"
	locales "github.com/myrepo/myserver/i18n"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tavocg/go-auth/authenticators"
	"github.com/tavocg/go-i18n"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the HTTP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServer(
			viper.GetString("db"),
			viper.GetBool("dev"),
			viper.GetString("loglvl"),
			viper.GetString("logfmt"),
			viper.GetString("auth.secret"),
			viper.GetDuration("auth.access-token-ttl"),
			viper.GetDuration("auth.refresh-token-ttl"),
			viper.GetString("root"),
			viper.GetString("host"),
			viper.GetInt("port"),
		)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	viper.SetDefault("dev", false)
	viper.SetDefault("host", "")
	viper.SetDefault("port", 3080)
	viper.SetDefault("logfmt", "json")
	viper.SetDefault("loglvl", "info")
	viper.SetDefault("auth.secret", "")
	viper.SetDefault("auth.access-token-ttl", 15*time.Minute)
	viper.SetDefault("auth.refresh-token-ttl", 30*24*time.Hour)
	viper.SetDefault("root", "")
	viper.SetDefault("db", "data/myserver.sqlite")

	flags := serveCmd.Flags()
	flags.String("db", "data/myserver.sqlite", "database DSN")
	flags.Bool("dev", false, "enable dev mode")
	flags.String("host", "", "bind host")
	flags.Int("port", 3080, "bind port")
	flags.String("logfmt", "json", "log format")
	flags.String("loglvl", "info", "log level")
	flags.String("auth-secret", "", "JWT signing secret")
	flags.Duration("auth-access-ttl", 15*time.Minute, "access token TTL")
	flags.Duration("auth-refresh-ttl", 30*24*time.Hour, "refresh token TTL")
	flags.String("root", "", "route prefix to mount the app under")

	mustBindFlag("db", serveCmd, "db")
	mustBindFlag("dev", serveCmd, "dev")
	mustBindFlag("host", serveCmd, "host")
	mustBindFlag("port", serveCmd, "port")
	mustBindFlag("logfmt", serveCmd, "logfmt")
	mustBindFlag("loglvl", serveCmd, "loglvl")
	mustBindFlag("auth.secret", serveCmd, "auth-secret")
	mustBindFlag("auth.access-token-ttl", serveCmd, "auth-access-ttl")
	mustBindFlag("auth.refresh-token-ttl", serveCmd, "auth-refresh-ttl")
	mustBindFlag("root", serveCmd, "root")
}

func runServer(connStr string, dev bool, logLvl string, logFmt string, authSecret string, authAccessTTL time.Duration, authRefreshTTL time.Duration, rootPrefix string, host string, port int) error {
	logger, err := newLogger(dev, logLvl, logFmt)
	if err != nil {
		return err
	}

	if connStr == "" {
		return errors.New("database DSN is required")
	}

	var db database.Database
	db, err = provider.Open(context.Background(), connStr)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(context.Background()); err != nil {
			logger.Error("database close error", "error", err)
		}
	}()

	localizer, err := i18n.NewLocalizer(locales.Locales())
	if err != nil {
		return err
	}

	authenticator, err := authenticators.NewMemoryAuthenticator[*appauth.Claims](authSecret, authAccessTTL, authRefreshTTL)
	if err != nil {
		return err
	}

	h := handlers.NewHandler(handlers.Options{
		Logger:        logger,
		Dev:           dev,
		DB:            db,
		Authenticator: authenticator,
		Localizer:     localizer,
		RootPrefix:    rootPrefix,
	})

	srv := &http.Server{
		Addr:    net.JoinHostPort(host, strconv.Itoa(port)),
		Handler: h.Mux(),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}

	return nil
}

func newLogger(dev bool, logLvl string, logFmt string) (*slog.Logger, error) {
	level, err := parseLogLevel(logLvl)
	if err != nil {
		return nil, err
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}
	if dev {
		opts.AddSource = true
	}

	switch strings.ToLower(logFmt) {
	case "json":
		return slog.New(slog.NewJSONHandler(os.Stdout, opts)), nil
	case "text":
		return slog.New(slog.NewTextHandler(os.Stdout, opts)), nil
	default:
		return nil, fmt.Errorf("invalid --logfmt %q: expected text or json", logFmt)
	}
}

func parseLogLevel(logLvl string) (slog.Level, error) {
	switch strings.ToLower(logLvl) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid --loglvl %q: expected debug, info, warn, or error", logLvl)
	}
}
