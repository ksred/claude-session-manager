// Package main provides the Claude Session Manager API server
//
// @title Claude Session Manager API
// @version 1.0.0
// @description A comprehensive API for managing and monitoring Claude.ai sessions with real-time analytics and insights.
// @termsOfService https://github.com/ksred/claude-session-manager
//
// @contact.name Claude Session Manager Support
// @contact.url https://github.com/ksred/claude-session-manager
// @contact.email support@claude-session-manager.com
//
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
//
// @host localhost:8080
// @BasePath /api/v1
// @schemes http https
//
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
//
// @x-extension-openapi {"example": "value"}
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ksred/claude-session-manager/internal/api"
	"github.com/ksred/claude-session-manager/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	appConfig *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "claude-session-manager",
	Short: "Claude Session Manager Backend Server",
	Long:  `A backend server for managing and monitoring Claude.ai sessions with real-time updates and analytics.`,
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the API server",
	Long:  `Start the Claude Session Manager API server with WebSocket support for real-time updates.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		var err error
		appConfig, err = config.LoadConfig(cfgFile)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to load configuration")
		}

		// Apply command line overrides
		applyCommandLineOverrides(cmd, appConfig)

		// Configure logging based on config
		if appConfig.Features.DebugMode {
			logrus.SetLevel(logrus.DebugLevel)
		} else {
			logrus.SetLevel(logrus.InfoLevel)
		}
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})

		// Log loaded configuration source
		if cfgFile != "" {
			logrus.WithField("config_file", cfgFile).Info("Using custom config file")
		}

		// Create server with configuration (using SQLite)
		server, err := api.NewSQLiteServer(appConfig)
		if err != nil {
			logrus.WithError(err).Fatal("Failed to create server")
		}

		// Setup graceful shutdown
		_, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle shutdown signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Start server in a goroutine
		serverErr := make(chan error, 1)
		go func() {
			logrus.WithFields(logrus.Fields{
				"port": appConfig.Server.Port,
				"host": appConfig.Server.Host,
			}).Info("Starting Claude Session Manager API server")
			serverErr <- server.Start()
		}()

		// Wait for shutdown signal or server error
		select {
		case sig := <-sigChan:
			logrus.WithField("signal", sig).Info("Received shutdown signal")
			
			logrus.Debug("Calling server.Stop()...")
			// Stop the server gracefully
			if err := server.Stop(); err != nil {
				logrus.WithError(err).Error("Error during server shutdown")
			}
			logrus.Debug("server.Stop() completed")
			
			logrus.Debug("Waiting for server goroutine to finish...")
			// Wait for the server goroutine to finish
			select {
			case err := <-serverErr:
				if err != nil && err != http.ErrServerClosed {
					logrus.WithError(err).Error("Server error during shutdown")
				}
				logrus.Debug("Server goroutine finished")
			case <-time.After(10 * time.Second):
				logrus.Error("Timeout waiting for server goroutine to finish")
			}
			
			logrus.Info("Server shutdown complete")
			return nil
			
		case err := <-serverErr:
			logrus.Debug("Received error from server goroutine")
			if err != nil && err != http.ErrServerClosed {
				logrus.WithError(err).Error("Server error")
				return err
			}
			// Server stopped for some other reason
			logrus.Info("Server stopped")
			return nil
		}

		return nil
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/claude-session-manager/config.yaml)")

	// Serve command flags
	serveCmd.Flags().IntP("port", "p", 0, "port to run the server on (overrides config)")
	serveCmd.Flags().Bool("debug", false, "enable debug logging (overrides config)")

	// Add commands
	rootCmd.AddCommand(serveCmd)
}

// Override config with command line flags after loading
func applyCommandLineOverrides(cmd *cobra.Command, cfg *config.Config) {
	// Check if port flag was explicitly set
	if portFlag := cmd.Flag("port"); portFlag != nil && portFlag.Changed {
		if port, err := cmd.Flags().GetInt("port"); err == nil && port > 0 {
			cfg.Server.Port = port
		}
	}

	// Check if debug flag was explicitly set
	if debugFlag := cmd.Flag("debug"); debugFlag != nil && debugFlag.Changed {
		if debug, err := cmd.Flags().GetBool("debug"); err == nil {
			cfg.Features.DebugMode = debug
		}
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}