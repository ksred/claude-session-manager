# Claude Session Manager Backend API

## Overview

The backend is built with Go using the Gin web framework. It provides RESTful API endpoints and WebSocket support for real-time updates.

## Running the Server

```bash
# From the backend directory
go run cmd/main.go serve

# Or build and run
go build -o claude-session-manager ./cmd/main.go
./claude-session-manager serve

# Run with options
./claude-session-manager serve --port 8080 --debug
```

## API Endpoints

Base URL: `http://localhost:8080/api/v1`

### Health Check
- `GET /health` - Server health status

### Sessions
- `GET /sessions` - Get all sessions
- `GET /sessions/:id` - Get specific session
- `GET /sessions/active` - Get active sessions
- `GET /sessions/recent?limit=10` - Get recent sessions

### Metrics
- `GET /metrics/summary` - Overall metrics summary
- `GET /metrics/activity` - Activity timeline
- `GET /metrics/usage` - Usage statistics

### Search
- `GET /search?q=query` - Search across sessions

### WebSocket
- `GET /ws` - WebSocket endpoint for real-time updates

## WebSocket Events

Connect to the WebSocket endpoint to receive real-time updates:

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws');
```

Event types:
- `session_created` - New session created
- `session_updated` - Session updated
- `message_sent` - New message in session
- `metrics_updated` - Metrics changed

## CORS

The server includes CORS middleware configured to allow requests from any origin during development. This should be restricted in production.

## Architecture

```
backend/
├── cmd/
│   └── main.go          # CLI entry point with Cobra
├── internal/
│   ├── api/
│   │   ├── server.go    # Server setup and configuration
│   │   ├── routes.go    # Route definitions
│   │   ├── handlers.go  # HTTP handlers
│   │   ├── websocket.go # WebSocket implementation
│   │   └── middleware.go # CORS and logging middleware
│   └── claude/          # Claude session management (existing)
└── go.mod               # Go module dependencies
```

## Dependencies

- `gin-gonic/gin` - Web framework
- `gorilla/websocket` - WebSocket support
- `fsnotify/fsnotify` - File system notifications
- `spf13/cobra` - CLI framework
- `spf13/viper` - Configuration management
- `sirupsen/logrus` - Structured logging