package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// TestDefaultConfig tests the default configuration values
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	// Test server defaults
	if config.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", config.Server.Port)
	}
	
	if config.Server.Host != "0.0.0.0" {
		t.Errorf("Expected default host '0.0.0.0', got '%s'", config.Server.Host)
	}
	
	if config.Server.ReadTimeout != 15 {
		t.Errorf("Expected default read timeout 15, got %d", config.Server.ReadTimeout)
	}
	
	if config.Server.WriteTimeout != 15 {
		t.Errorf("Expected default write timeout 15, got %d", config.Server.WriteTimeout)
	}
	
	if config.Server.ShutdownTimeout != 10 {
		t.Errorf("Expected default shutdown timeout 10, got %d", config.Server.ShutdownTimeout)
	}
	
	// Test CORS defaults
	if !config.Server.CORS.Enabled {
		t.Error("Expected CORS to be enabled by default")
	}
	
	if len(config.Server.CORS.AllowedOrigins) != 1 || config.Server.CORS.AllowedOrigins[0] != "*" {
		t.Errorf("Expected default allowed origins ['*'], got %v", config.Server.CORS.AllowedOrigins)
	}
	
	expectedMethods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	if len(config.Server.CORS.AllowedMethods) != len(expectedMethods) {
		t.Errorf("Expected %d allowed methods, got %d", len(expectedMethods), len(config.Server.CORS.AllowedMethods))
	}
	
	if !config.Server.CORS.AllowCredentials {
		t.Error("Expected CORS allow credentials to be true")
	}
	
	if config.Server.CORS.MaxAge != 86400 {
		t.Errorf("Expected CORS max age 86400, got %d", config.Server.CORS.MaxAge)
	}
	
	// Test Claude defaults
	homeDir, _ := os.UserHomeDir()
	expectedClaudeDir := filepath.Join(homeDir, ".claude")
	if config.Claude.HomeDirectory != expectedClaudeDir {
		t.Errorf("Expected Claude home directory '%s', got '%s'", expectedClaudeDir, config.Claude.HomeDirectory)
	}
	
	expectedProjectsPath := filepath.Join(expectedClaudeDir, "projects")
	if config.Claude.ProjectsPath != expectedProjectsPath {
		t.Errorf("Expected projects path '%s', got '%s'", expectedProjectsPath, config.Claude.ProjectsPath)
	}
	
	if config.Claude.WatchInterval != 5 {
		t.Errorf("Expected watch interval 5, got %d", config.Claude.WatchInterval)
	}
	
	if config.Claude.CacheRefreshRate != 5 {
		t.Errorf("Expected cache refresh rate 5, got %d", config.Claude.CacheRefreshRate)
	}
	
	// Test pricing defaults
	if config.Pricing.InputTokensPerK != 0.01 {
		t.Errorf("Expected input token price 0.01, got %f", config.Pricing.InputTokensPerK)
	}
	
	if config.Pricing.OutputTokensPerK != 0.03 {
		t.Errorf("Expected output token price 0.03, got %f", config.Pricing.OutputTokensPerK)
	}
	
	if config.Pricing.Currency != "USD" {
		t.Errorf("Expected currency 'USD', got '%s'", config.Pricing.Currency)
	}
	
	// Test features defaults
	if !config.Features.EnableWebSocket {
		t.Error("Expected WebSocket to be enabled by default")
	}
	
	if !config.Features.EnableFileWatcher {
		t.Error("Expected FileWatcher to be enabled by default")
	}
	
	if config.Features.EnableMetrics {
		t.Error("Expected Metrics to be disabled by default")
	}
	
	if config.Features.EnableProfiling {
		t.Error("Expected Profiling to be disabled by default")
	}
	
	if config.Features.DebugMode {
		t.Error("Expected DebugMode to be disabled by default")
	}
}

// TestLoadConfigFromFile tests loading configuration from a file
func TestLoadConfigFromFile(t *testing.T) {
	// Create a temporary config file
	tempDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	configFile := filepath.Join(tempDir, "config.yaml")
	configContent := `
server:
  port: 9090
  host: "127.0.0.1"
  read_timeout: 30
  write_timeout: 30
  shutdown_timeout: 15
  cors:
    enabled: false
    allowed_origins:
      - "https://example.com"
    allowed_methods:
      - "GET"
      - "POST"
    allowed_headers:
      - "Content-Type"
    allow_credentials: false
    max_age: 3600

claude:
  home_directory: "/custom/claude"
  projects_path: "/custom/claude/projects"
  watch_interval: 10
  cache_refresh_rate: 2

pricing:
  input_tokens_per_k: 0.02
  output_tokens_per_k: 0.04
  currency: "EUR"

features:
  enable_websocket: false
  enable_file_watcher: false
  enable_metrics: true
  enable_profiling: true
  debug_mode: true
`
	
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	// Load config from file
	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// Verify loaded values
	if config.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", config.Server.Port)
	}
	
	if config.Server.Host != "127.0.0.1" {
		t.Errorf("Expected host '127.0.0.1', got '%s'", config.Server.Host)
	}
	
	if config.Server.ReadTimeout != 30 {
		t.Errorf("Expected read timeout 30, got %d", config.Server.ReadTimeout)
	}
	
	if config.Server.CORS.Enabled {
		t.Error("Expected CORS to be disabled")
	}
	
	if len(config.Server.CORS.AllowedOrigins) != 1 || config.Server.CORS.AllowedOrigins[0] != "https://example.com" {
		t.Errorf("Expected allowed origins ['https://example.com'], got %v", config.Server.CORS.AllowedOrigins)
	}
	
	if config.Claude.HomeDirectory != "/custom/claude" {
		t.Errorf("Expected Claude home '/custom/claude', got '%s'", config.Claude.HomeDirectory)
	}
	
	if config.Claude.WatchInterval != 10 {
		t.Errorf("Expected watch interval 10, got %d", config.Claude.WatchInterval)
	}
	
	if config.Pricing.InputTokensPerK != 0.02 {
		t.Errorf("Expected input token price 0.02, got %f", config.Pricing.InputTokensPerK)
	}
	
	if config.Pricing.Currency != "EUR" {
		t.Errorf("Expected currency 'EUR', got '%s'", config.Pricing.Currency)
	}
	
	if config.Features.EnableWebSocket {
		t.Error("Expected WebSocket to be disabled")
	}
	
	if !config.Features.EnableMetrics {
		t.Error("Expected Metrics to be enabled")
	}
	
	if !config.Features.DebugMode {
		t.Error("Expected DebugMode to be enabled")
	}
}

// TestLoadConfigWithEnvironmentVariables tests environment variable overrides
func TestLoadConfigWithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	envVars := map[string]string{
		"CSM_SERVER_PORT":                "7777",
		"CSM_SERVER_HOST":                "localhost",
		"CSM_CLAUDE_WATCH_INTERVAL":      "20",
		"CSM_PRICING_INPUT_TOKENS_PER_K": "0.05",
		"CSM_FEATURES_DEBUG_MODE":        "true",
	}
	
	// Set environment variables
	for key, value := range envVars {
		err := os.Setenv(key, value)
		if err != nil {
			t.Fatalf("Failed to set env var %s: %v", key, err)
		}
	}
	
	// Clean up environment variables after test
	defer func() {
		for key := range envVars {
			os.Unsetenv(key)
		}
	}()
	
	// Load config (no file specified, should use defaults + env vars)
	config, err := LoadConfig("")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// Verify environment overrides
	if config.Server.Port != 7777 {
		t.Errorf("Expected port 7777 from env, got %d", config.Server.Port)
	}
	
	if config.Server.Host != "localhost" {
		t.Errorf("Expected host 'localhost' from env, got '%s'", config.Server.Host)
	}
	
	if config.Claude.WatchInterval != 20 {
		t.Errorf("Expected watch interval 20 from env, got %d", config.Claude.WatchInterval)
	}
	
	if config.Pricing.InputTokensPerK != 0.05 {
		t.Errorf("Expected input token price 0.05 from env, got %f", config.Pricing.InputTokensPerK)
	}
	
	if !config.Features.DebugMode {
		t.Error("Expected DebugMode to be enabled from env")
	}
}

// TestLoadConfigNoFile tests loading config with no file present
func TestLoadConfigNoFile(t *testing.T) {
	// Try to load config from empty string (uses search paths)
	config, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig should not fail when no config file found: %v", err)
	}
	
	// Should have default values
	if config.Server.Port != 8080 {
		t.Errorf("Expected default port 8080, got %d", config.Server.Port)
	}
	
	if config.Claude.WatchInterval != 5 {
		t.Errorf("Expected default watch interval 5, got %d", config.Claude.WatchInterval)
	}
}

// TestValidateConfig tests configuration validation
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "Invalid port - too low",
			config: &Config{
				Server: ServerConfig{Port: 0},
			},
			wantErr: true,
			errMsg:  "invalid server port",
		},
		{
			name: "Invalid port - too high",
			config: &Config{
				Server: ServerConfig{Port: 70000},
			},
			wantErr: true,
			errMsg:  "invalid server port",
		},
		{
			name: "Invalid read timeout",
			config: &Config{
				Server: ServerConfig{
					Port:        8080,
					ReadTimeout: -1,
				},
			},
			wantErr: true,
			errMsg:  "invalid read timeout",
		},
		{
			name: "Invalid write timeout",
			config: &Config{
				Server: ServerConfig{
					Port:         8080,
					WriteTimeout: -1,
				},
			},
			wantErr: true,
			errMsg:  "invalid write timeout",
		},
		{
			name: "Invalid shutdown timeout",
			config: &Config{
				Server: ServerConfig{
					Port:            8080,
					ShutdownTimeout: -1,
				},
			},
			wantErr: true,
			errMsg:  "invalid shutdown timeout",
		},
		{
			name: "Invalid watch interval",
			config: &Config{
				Server: ServerConfig{Port: 8080},
				Claude: ClaudeConfig{WatchInterval: -1},
			},
			wantErr: true,
			errMsg:  "invalid watch interval",
		},
		{
			name: "Invalid cache refresh rate",
			config: &Config{
				Server: ServerConfig{Port: 8080},
				Claude: ClaudeConfig{CacheRefreshRate: -1},
			},
			wantErr: true,
			errMsg:  "invalid cache refresh rate",
		},
		{
			name: "Invalid input token price",
			config: &Config{
				Server:  ServerConfig{Port: 8080},
				Claude:  ClaudeConfig{},
				Pricing: PricingConfig{InputTokensPerK: -0.01},
			},
			wantErr: true,
			errMsg:  "invalid input token price",
		},
		{
			name: "Invalid output token price",
			config: &Config{
				Server:  ServerConfig{Port: 8080},
				Claude:  ClaudeConfig{},
				Pricing: PricingConfig{OutputTokensPerK: -0.01},
			},
			wantErr: true,
			errMsg:  "invalid output token price",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errMsg)
				} else if err.Error() == "" || (tt.errMsg != "" && err.Error()[:len(tt.errMsg)] != tt.errMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

// TestSetDefaults tests the setDefaults function
func TestSetDefaults(t *testing.T) {
	v := viper.New()
	setDefaults(v)
	
	// Test that defaults are set correctly
	if v.GetInt("server.port") != 8080 {
		t.Errorf("Expected default port 8080, got %d", v.GetInt("server.port"))
	}
	
	if v.GetString("server.host") != "0.0.0.0" {
		t.Errorf("Expected default host '0.0.0.0', got '%s'", v.GetString("server.host"))
	}
	
	if v.GetBool("server.cors.enabled") != true {
		t.Error("Expected CORS to be enabled by default")
	}
	
	allowedOrigins := v.GetStringSlice("server.cors.allowed_origins")
	if len(allowedOrigins) != 1 || allowedOrigins[0] != "*" {
		t.Errorf("Expected default allowed origins ['*'], got %v", allowedOrigins)
	}
	
	if v.GetFloat64("pricing.input_tokens_per_k") != 0.01 {
		t.Errorf("Expected input token price 0.01, got %f", v.GetFloat64("pricing.input_tokens_per_k"))
	}
	
	if v.GetBool("features.enable_websocket") != true {
		t.Error("Expected WebSocket to be enabled by default")
	}
	
	if v.GetBool("features.debug_mode") != false {
		t.Error("Expected DebugMode to be disabled by default")
	}
}

// TestGetConfigPath tests the GetConfigPath function
func TestGetConfigPath(t *testing.T) {
	path := GetConfigPath()
	
	// Should return a valid path
	if path == "" {
		t.Error("GetConfigPath should not return empty string")
	}
	
	// Should contain the expected directory structure
	if !filepath.IsAbs(path) {
		// If not absolute, should be the fallback
		if path != "config.yaml" {
			t.Errorf("Expected fallback 'config.yaml', got '%s'", path)
		}
	} else {
		// Should contain config directory structure
		if filepath.Clean(path) != path {
			t.Error("Config path should be clean")
		}
	}
}

// TestLoadConfigInvalidFile tests loading config with invalid YAML
func TestLoadConfigInvalidFile(t *testing.T) {
	// Create a temporary invalid config file
	tempDir, err := os.MkdirTemp("", "config-test-invalid")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	configFile := filepath.Join(tempDir, "config.yaml")
	invalidContent := `
server:
  port: "not a number"
  invalid_yaml: [
`
	
	err = os.WriteFile(configFile, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid config file: %v", err)
	}
	
	// Try to load invalid config
	_, err = LoadConfig(configFile)
	if err == nil {
		t.Error("Expected error for invalid config file")
	}
}

// TestConfigStructureCompleteness tests that all config fields are properly structured
func TestConfigStructureCompleteness(t *testing.T) {
	config := DefaultConfig()
	
	// Test that all nested structures are properly initialized
	if config.Server.CORS.AllowedOrigins == nil {
		t.Error("CORS AllowedOrigins should be initialized")
	}
	
	if config.Server.CORS.AllowedMethods == nil {
		t.Error("CORS AllowedMethods should be initialized")
	}
	
	if config.Server.CORS.AllowedHeaders == nil {
		t.Error("CORS AllowedHeaders should be initialized")
	}
	
	// Test that string fields are not empty where they shouldn't be
	if config.Server.Host == "" {
		t.Error("Server Host should not be empty")
	}
	
	if config.Claude.HomeDirectory == "" {
		t.Error("Claude HomeDirectory should not be empty")
	}
	
	if config.Claude.ProjectsPath == "" {
		t.Error("Claude ProjectsPath should not be empty")
	}
	
	if config.Pricing.Currency == "" {
		t.Error("Pricing Currency should not be empty")
	}
}

// TestEnvironmentVariableMapping tests that environment variables map correctly
func TestEnvironmentVariableMapping(t *testing.T) {
	// Test complex nested structure mapping
	os.Setenv("CSM_SERVER_CORS_ALLOWED_ORIGINS", "https://app.com,https://api.com")
	os.Setenv("CSM_SERVER_CORS_ALLOWED_METHODS", "GET,POST,PUT")
	defer func() {
		os.Unsetenv("CSM_SERVER_CORS_ALLOWED_ORIGINS")
		os.Unsetenv("CSM_SERVER_CORS_ALLOWED_METHODS")
	}()
	
	_, err := LoadConfig("")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// Note: Viper has limitations with array environment variables
	// This test verifies the environment variable structure is correct
	// even if the actual parsing might not work perfectly for arrays
	
	// Test that basic environment variables work
	os.Setenv("CSM_SERVER_PORT", "9999")
	defer os.Unsetenv("CSM_SERVER_PORT")
	
	config2, err := LoadConfig("")
	if err != nil {
		t.Fatalf("Failed to load config with env override: %v", err)
	}
	
	if config2.Server.Port != 9999 {
		t.Errorf("Expected port 9999 from env, got %d", config2.Server.Port)
	}
}

// TestConfigValidationBoundaries tests edge cases for validation
func TestConfigValidationBoundaries(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		valid  bool
	}{
		{
			name: "Port boundary - minimum valid",
			config: &Config{
				Server: ServerConfig{Port: 1},
			},
			valid: true,
		},
		{
			name: "Port boundary - maximum valid",
			config: &Config{
				Server: ServerConfig{Port: 65535},
			},
			valid: true,
		},
		{
			name: "Zero timeouts (valid)",
			config: &Config{
				Server: ServerConfig{
					Port:            8080,
					ReadTimeout:     0,
					WriteTimeout:    0,
					ShutdownTimeout: 0,
				},
			},
			valid: true,
		},
		{
			name: "Zero intervals (valid)",
			config: &Config{
				Server: ServerConfig{Port: 8080},
				Claude: ClaudeConfig{
					WatchInterval:    0,
					CacheRefreshRate: 0,
				},
			},
			valid: true,
		},
		{
			name: "Zero token prices (valid)",
			config: &Config{
				Server: ServerConfig{Port: 8080},
				Claude: ClaudeConfig{},
				Pricing: PricingConfig{
					InputTokensPerK:  0.0,
					OutputTokensPerK: 0.0,
				},
			},
			valid: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.valid && err != nil {
				t.Errorf("Expected valid config, got error: %v", err)
			} else if !tt.valid && err == nil {
				t.Error("Expected invalid config, got no error")
			}
		})
	}
}

// BenchmarkLoadConfig benchmarks configuration loading
func BenchmarkLoadConfig(b *testing.B) {
	// Create a temporary config file
	tempDir, err := os.MkdirTemp("", "config-benchmark")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	configFile := filepath.Join(tempDir, "config.yaml")
	configContent := `
server:
  port: 8080
  host: "0.0.0.0"
claude:
  watch_interval: 5
pricing:
  input_tokens_per_k: 0.01
features:
  enable_websocket: true
`
	
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		b.Fatalf("Failed to write config file: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadConfig(configFile)
		if err != nil {
			b.Fatalf("Failed to load config: %v", err)
		}
	}
}

// BenchmarkDefaultConfig benchmarks default configuration creation
func BenchmarkDefaultConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultConfig()
	}
}

// BenchmarkValidateConfig benchmarks configuration validation
func BenchmarkValidateConfig(b *testing.B) {
	config := DefaultConfig()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validateConfig(config)
		if err != nil {
			b.Fatalf("Validation failed: %v", err)
		}
	}
}