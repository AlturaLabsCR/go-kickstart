/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/myrepo/myserver/handler"
	"github.com/spf13/cobra"
)

var serveDev bool
var serveLogFmt string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServer(serveDev, serveLogFmt)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().BoolVar(&serveDev, "dev", false, "enable development logging")
	serveCmd.Flags().StringVar(&serveLogFmt, "logfmt", "json", "log format: text or json")
}

func runServer(dev bool, logFmt string) error {
	logger, err := newLogger(dev, logFmt)
	if err != nil {
		return err
	}

	handler.SetLogger(logger)
	handler.SetDev(dev)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: handler.Mux(),
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

func newLogger(dev bool, logFmt string) (*slog.Logger, error) {
	opts := &slog.HandlerOptions{}
	if dev {
		opts.AddSource = true
		opts.Level = slog.LevelDebug
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
