// Copyright © 2026 NAME HERE <EMAIL ADDRESS>

package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the HTTP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServerFromConfig()
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
	viper.SetDefault("db", "data/app.sqlite")

	flags := rootCmd.PersistentFlags()
	flags.String("db", "data/app.sqlite", "database DSN")
	flags.Bool("dev", false, "enable dev mode")
	flags.String("host", "", "bind host")
	flags.Int("port", 3080, "bind port")
	flags.String("logfmt", "json", "log format")
	flags.String("loglvl", "info", "log level")
	flags.String("auth-secret", "", "JWT signing secret")
	flags.Duration("auth-access-ttl", 15*time.Minute, "access token TTL")
	flags.Duration("auth-refresh-ttl", 30*24*time.Hour, "refresh token TTL")
	flags.String("root", "", "route prefix to mount the app under")

	mustBindPersistentFlag("db", rootCmd, "db")
	mustBindPersistentFlag("dev", rootCmd, "dev")
	mustBindPersistentFlag("host", rootCmd, "host")
	mustBindPersistentFlag("port", rootCmd, "port")
	mustBindPersistentFlag("logfmt", rootCmd, "logfmt")
	mustBindPersistentFlag("loglvl", rootCmd, "loglvl")
	mustBindPersistentFlag("auth.secret", rootCmd, "auth-secret")
	mustBindPersistentFlag("auth.access-token-ttl", rootCmd, "auth-access-ttl")
	mustBindPersistentFlag("auth.refresh-token-ttl", rootCmd, "auth-refresh-ttl")
	mustBindPersistentFlag("root", rootCmd, "root")
}

func runServerFromConfig() error {
	return runServer(serverConfig{
		DatabaseDSN:     viper.GetString("db"),
		Dev:             viper.GetBool("dev"),
		LogLevel:        viper.GetString("loglvl"),
		LogFormat:       viper.GetString("logfmt"),
		AuthSecret:      viper.GetString("auth.secret"),
		AccessTokenTTL:  viper.GetDuration("auth.access-token-ttl"),
		RefreshTokenTTL: viper.GetDuration("auth.refresh-token-ttl"),
		RootPrefix:      viper.GetString("root"),
		Host:            viper.GetString("host"),
		Port:            viper.GetInt("port"),
	})
}
