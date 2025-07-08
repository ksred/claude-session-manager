# Claude Session Manager Configuration Guide

## Overview

Claude Session Manager uses a flexible configuration system powered by Viper that supports:
- Configuration files (YAML)
- Environment variables
- Command-line flags

Configuration sources are loaded in the following order (later sources override earlier ones):
1. Default values (built into the application)
2. Configuration file
3. Environment variables
4. Command-line flags

## Configuration File

The default configuration file location is: `~/.config/claude-session-manager/config.yaml`

You can specify a custom configuration file using the `--config` flag:
```bash
claude-session-manager serve --config /path/to/custom/config.yaml
```

### Creating a Configuration File

Initialize a configuration file with default values:
```bash
make config-init
```

This will create a configuration file at `~/.config/claude-session-manager/config.yaml` if it doesn't already exist.

### Configuration Structure

```yaml
# HTTP Server Configuration
server:
  port: 8080                  # Server port
  host: "0.0.0.0"            # Server host
  read_timeout: 15           # Read timeout in seconds
  write_timeout: 15          # Write timeout in seconds
  shutdown_timeout: 10       # Graceful shutdown timeout in seconds
  
  # CORS Configuration
  cors:
    enabled: true
    allowed_origins:
      - "*"                  # Allow all origins (configure for production)
    allowed_methods:
      - GET
      - POST
      - PUT
      - DELETE
      - OPTIONS
    allowed_headers:
      - Origin
      - Content-Type
      - Accept
      - Authorization
    allow_credentials: true
    max_age: 86400          # CORS preflight cache duration in seconds

# Claude Configuration
claude:
  home_directory: ~/.claude      # Claude home directory
  projects_path: ~/.claude/projects  # Claude projects directory
  watch_interval: 5             # File system watch interval in seconds
  cache_refresh_rate: 5         # Cache refresh rate in minutes

# Token Pricing Configuration
pricing:
  input_tokens_per_k: 0.01      # Cost per 1,000 input tokens
  output_tokens_per_k: 0.03     # Cost per 1,000 output tokens
  currency: USD                 # Currency for cost calculations

# Feature Flags and Settings
features:
  enable_websocket: true        # Enable WebSocket support
  enable_file_watcher: true     # Enable file system watcher
  enable_metrics: false         # Enable metrics collection
  enable_profiling: false       # Enable profiling endpoints
  debug_mode: false            # Enable debug mode
```

## Environment Variables

Any configuration value can be overridden using environment variables. Use the `CSM_` prefix and replace dots with underscores:

```bash
# Override server port
export CSM_SERVER_PORT=9090

# Override debug mode
export CSM_FEATURES_DEBUG_MODE=true

# Override token pricing
export CSM_PRICING_INPUT_TOKENS_PER_K=0.015
export CSM_PRICING_OUTPUT_TOKENS_PER_K=0.045

# Override Claude home directory
export CSM_CLAUDE_HOME_DIRECTORY=/custom/path/to/claude
```

## Command-Line Flags

The most common settings can be overridden via command-line flags:

```bash
# Override port
claude-session-manager serve --port 9090

# Enable debug mode
claude-session-manager serve --debug

# Use custom config file
claude-session-manager serve --config /path/to/config.yaml
```

## Configuration Precedence Example

Given the following:
1. Default port in code: `8080`
2. Config file (`config.yaml`): `port: 8081`
3. Environment variable: `CSM_SERVER_PORT=8082`
4. Command-line flag: `--port 8083`

The server will run on port `8083` (command-line flag wins).

## Makefile Commands

The Makefile provides several configuration-related commands:

```bash
# Initialize configuration file
make config-init

# Show current configuration
make config-show

# Run with custom configuration
make run-config

# Run with default configuration
make run
```

## Production Recommendations

1. **CORS Settings**: Configure `allowed_origins` to specific domains instead of `"*"`
2. **Timeouts**: Adjust timeouts based on your deployment environment
3. **Token Pricing**: Update pricing to match current Claude API rates
4. **Debug Mode**: Ensure `debug_mode` is `false` in production
5. **File Watcher**: Consider disabling if not needed to reduce resource usage

## Troubleshooting

### Configuration Not Loading

1. Check file permissions on the config file
2. Verify the YAML syntax is valid
3. Use `--debug` flag to see configuration loading details

### Environment Variables Not Working

1. Ensure the `CSM_` prefix is used
2. Use underscores instead of dots (e.g., `CSM_SERVER_PORT`)
3. Export the variables before running the application

### Command-Line Flags Not Working

1. Place flags after the `serve` command
2. Use the correct flag format (e.g., `--port` not `-port`)
3. Check `--help` for available flags