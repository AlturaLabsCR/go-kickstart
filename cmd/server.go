// Copyright © 2026 NAME HERE <EMAIL ADDRESS>

package cmd

import (
	"context"
	"errors"
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
	Host            string
	Port            int
}

func runServer(cfg serverConfig) error {
	logger, err := newLogger(cfg.Dev, cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		return err
	}

	if cfg.DatabaseDSN == "" {
		return errors.New("database DSN is required")
	}

	var db database.Database
	db, err = provider.Open(context.Background(), cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	db = cached.New(db, memory.New())
	defer func() {
		if err := db.Close(context.Background()); err != nil {
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
	if err := authenticator.CleanExpiredRefreshTokens(context.Background()); err != nil {
		return err
	}

	h := handlers.NewHandler(handlers.Options{
		Logger:        logger,
		Dev:           cfg.Dev,
		DB:            db,
		Authenticator: authenticator,
		Localizer:     localizer,
		RootPrefix:    cfg.RootPrefix,
	})

	srv := &http.Server{
		Addr:    net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Handler: h,
	}

	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("listening", "addr", listener.Addr().String())
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
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
