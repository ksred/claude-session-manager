package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the complete application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Claude   ClaudeConfig   `mapstructure:"claude"`
	Pricing  PricingConfig  `mapstructure:"pricing"`
	Features FeaturesConfig `mapstructure:"features"`
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Port            int      `mapstructure:"port"`
	Host            string   `mapstructure:"host"`
	ReadTimeout     int      `mapstructure:"read_timeout"`     // seconds
	WriteTimeout    int      `mapstructure:"write_timeout"`    // seconds
	ShutdownTimeout int      `mapstructure:"shutdown_timeout"` // seconds
	CORS            CORSConfig `mapstructure:"cors"`
}

// CORSConfig contains CORS settings
type CORSConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	AllowedOrigins   []string `mapstructure:"allowed_origins"`
	AllowedMethods   []string `mapstructure:"allowed_methods"`
	AllowedHeaders   []string `mapstructure:"allowed_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"`
}

// ClaudeConfig contains Claude-specific settings
type ClaudeConfig struct {
	HomeDirectory    string `mapstructure:"home_directory"`
	ProjectsPath     string `mapstructure:"projects_path"`
	WatchInterval    int    `mapstructure:"watch_interval"`    // seconds
	CacheRefreshRate int    `mapstructure:"cache_refresh_rate"` // minutes
}

// PricingConfig contains token pricing information
type PricingConfig struct {
	InputTokensPerK  float64 `mapstructure:"input_tokens_per_k"`  // Cost per 1K input tokens
	OutputTokensPerK float64 `mapstructure:"output_tokens_per_k"` // Cost per 1K output tokens
	Currency         string  `mapstructure:"currency"`
}

// FeaturesConfig contains feature flags and settings
type FeaturesConfig struct {
	EnableWebSocket      bool `mapstructure:"enable_websocket"`
	EnableFileWatcher    bool `mapstructure:"enable_file_watcher"`
	EnableMetrics        bool `mapstructure:"enable_metrics"`
	EnableProfiling      bool `mapstructure:"enable_profiling"`
	DebugMode            bool `mapstructure:"debug_mode"`
	WebSocketBatchInterval int  `mapstructure:"websocket_batch_interval"` // seconds
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	// Check for CLAUDE_DIR environment variable first
	claudeDir := os.Getenv("CLAUDE_DIR")
	if claudeDir == "" {
		// Fall back to ~/.claude if not set
		homeDir, _ := os.UserHomeDir()
		claudeDir = filepath.Join(homeDir, ".claude")
	}
	
	return &Config{
		Server: ServerConfig{
			Port:            8080,
			Host:            "0.0.0.0",
			ReadTimeout:     15,
			WriteTimeout:    15,
			ShutdownTimeout: 10,
			CORS: CORSConfig{
				Enabled:          true,
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization"},
				AllowCredentials: true,
				MaxAge:           86400,
			},
		},
		Claude: ClaudeConfig{
			HomeDirectory:    claudeDir,
			ProjectsPath:     filepath.Join(claudeDir, "projects"),
			WatchInterval:    5,
			CacheRefreshRate: 5,
		},
		Pricing: PricingConfig{
			InputTokensPerK:  0.003,  // $3.00 per million = $0.003 per 1K
			OutputTokensPerK: 0.015,  // $15.00 per million = $0.015 per 1K  
			Currency:         "USD",
		},
		Features: FeaturesConfig{
			EnableWebSocket:   true,
			EnableFileWatcher: true,
			EnableMetrics:     false,
			EnableProfiling:   false,
			DebugMode:         false,
			WebSocketBatchInterval: 20, // 20 seconds default
		},
	}
}

// LoadConfig loads configuration from multiple sources
func LoadConfig(configFile string) (*Config, error) {
	v := viper.New()
	
	// Set default values
	setDefaults(v)
	
	// Set config file
	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		// Look for config in standard locations
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		
		// Add config paths
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/claude-session-manager")
		
		// User config directory
		if homeDir, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(filepath.Join(homeDir, ".config", "claude-session-manager"))
		}
	}
	
	// Environment variables
	v.SetEnvPrefix("CSM")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	
	// Read config file if it exists
	if err := v.ReadInConfig(); err != nil {
		// It's okay if the config file doesn't exist, we have defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}
	
	// Unmarshal config
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}
	
	// Validate config
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	return &config, nil
}

// setDefaults sets default values in viper
func setDefaults(v *viper.Viper) {
	defaults := DefaultConfig()
	
	// Server defaults
	v.SetDefault("server.port", defaults.Server.Port)
	v.SetDefault("server.host", defaults.Server.Host)
	v.SetDefault("server.read_timeout", defaults.Server.ReadTimeout)
	v.SetDefault("server.write_timeout", defaults.Server.WriteTimeout)
	v.SetDefault("server.shutdown_timeout", defaults.Server.ShutdownTimeout)
	
	// CORS defaults
	v.SetDefault("server.cors.enabled", defaults.Server.CORS.Enabled)
	v.SetDefault("server.cors.allowed_origins", defaults.Server.CORS.AllowedOrigins)
	v.SetDefault("server.cors.allowed_methods", defaults.Server.CORS.AllowedMethods)
	v.SetDefault("server.cors.allowed_headers", defaults.Server.CORS.AllowedHeaders)
	v.SetDefault("server.cors.allow_credentials", defaults.Server.CORS.AllowCredentials)
	v.SetDefault("server.cors.max_age", defaults.Server.CORS.MaxAge)
	
	// Claude defaults
	v.SetDefault("claude.home_directory", defaults.Claude.HomeDirectory)
	v.SetDefault("claude.projects_path", defaults.Claude.ProjectsPath)
	v.SetDefault("claude.watch_interval", defaults.Claude.WatchInterval)
	v.SetDefault("claude.cache_refresh_rate", defaults.Claude.CacheRefreshRate)
	
	// Pricing defaults
	v.SetDefault("pricing.input_tokens_per_k", defaults.Pricing.InputTokensPerK)
	v.SetDefault("pricing.output_tokens_per_k", defaults.Pricing.OutputTokensPerK)
	v.SetDefault("pricing.currency", defaults.Pricing.Currency)
	
	// Features defaults
	v.SetDefault("features.enable_websocket", defaults.Features.EnableWebSocket)
	v.SetDefault("features.enable_file_watcher", defaults.Features.EnableFileWatcher)
	v.SetDefault("features.enable_metrics", defaults.Features.EnableMetrics)
	v.SetDefault("features.enable_profiling", defaults.Features.EnableProfiling)
	v.SetDefault("features.debug_mode", defaults.Features.DebugMode)
	v.SetDefault("features.websocket_batch_interval", defaults.Features.WebSocketBatchInterval)
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate server port
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}
	
	// Validate timeouts
	if config.Server.ReadTimeout < 0 {
		return fmt.Errorf("invalid read timeout: %d", config.Server.ReadTimeout)
	}
	if config.Server.WriteTimeout < 0 {
		return fmt.Errorf("invalid write timeout: %d", config.Server.WriteTimeout)
	}
	if config.Server.ShutdownTimeout < 0 {
		return fmt.Errorf("invalid shutdown timeout: %d", config.Server.ShutdownTimeout)
	}
	
	// Validate Claude settings
	if config.Claude.WatchInterval < 0 {
		return fmt.Errorf("invalid watch interval: %d", config.Claude.WatchInterval)
	}
	if config.Claude.CacheRefreshRate < 0 {
		return fmt.Errorf("invalid cache refresh rate: %d", config.Claude.CacheRefreshRate)
	}
	
	// Validate pricing
	if config.Pricing.InputTokensPerK < 0 {
		return fmt.Errorf("invalid input token price: %f", config.Pricing.InputTokensPerK)
	}
	if config.Pricing.OutputTokensPerK < 0 {
		return fmt.Errorf("invalid output token price: %f", config.Pricing.OutputTokensPerK)
	}
	
	return nil
}

// GetConfigPath returns the path where the config file should be created
func GetConfigPath() string {
	if homeDir, err := os.UserHomeDir(); err == nil {
		return filepath.Join(homeDir, ".config", "claude-session-manager", "config.yaml")
	}
	return "config.yaml"
}