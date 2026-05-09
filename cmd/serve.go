/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
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

	"github.com/myrepo/myserver/database"
	"github.com/myrepo/myserver/database/provider"
	"github.com/myrepo/myserver/handlers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServer(
			viper.GetString("serve.db"),
			viper.GetBool("serve.dev"),
			viper.GetString("serve.loglvl"),
			viper.GetString("serve.logfmt"),
			viper.GetString("serve.host"),
			viper.GetInt("serve.port"),
		)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	viper.SetDefault("serve.dev", false)
	viper.SetDefault("serve.host", "")
	viper.SetDefault("serve.port", 3080)
	viper.SetDefault("serve.logfmt", "json")
	viper.SetDefault("serve.loglvl", "info")
	viper.SetDefault("serve.db", "myserver.sqlite")

	flags := serveCmd.Flags()
	flags.String("db", "myserver.sqlite", "database connection string")
	flags.Bool("dev", false, "enable development logging")
	flags.String("host", "", "host interface to bind")
	flags.Int("port", 3080, "port to listen on")
	flags.String("logfmt", "json", "log format: text or json")
	flags.String("loglvl", "info", "log level: debug, info, warn, or error")

	mustBindFlag("serve.db", serveCmd, "db")
	mustBindFlag("serve.dev", serveCmd, "dev")
	mustBindFlag("serve.host", serveCmd, "host")
	mustBindFlag("serve.port", serveCmd, "port")
	mustBindFlag("serve.logfmt", serveCmd, "logfmt")
	mustBindFlag("serve.loglvl", serveCmd, "loglvl")
}

func runServer(connStr string, dev bool, logLvl string, logFmt string, host string, port int) error {
	logger, err := newLogger(dev, logLvl, logFmt)
	if err != nil {
		return err
	}

	var dbConn database.Database
	if connStr != "" {
		dbConn, err = provider.Open(context.Background(), connStr)
		if err != nil {
			return err
		}
		defer func() {
			if err := dbConn.Close(context.Background()); err != nil {
				logger.Error("database close error", "error", err)
			}
		}()
	}

	h := handlers.NewHandler(handlers.Options{
		Logger: logger,
		Dev:    dev,
		DB:     dbConn,
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
