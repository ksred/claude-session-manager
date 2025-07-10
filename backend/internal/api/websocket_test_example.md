# WebSocket Integration Test Guide

## Overview
The WebSocket integration allows real-time updates when Claude session data changes. When the file watcher detects changes in JSONL files, it broadcasts updates through WebSocket to all connected clients.

## Update Types

### 1. Session Updates
- **session_created**: New session detected
- **session_update**: Existing session modified
- **session_deleted**: Session removed

### 2. Activity Updates
- **activity_update**: New activity logged (messages sent, files modified)

### 3. Metrics Updates
- **metrics_update**: Token usage updated for a session

## Testing the Integration

### Backend Setup
1. Ensure WebSocket is enabled in config:
```yaml
features:
  enable_websocket: true
  enable_file_watcher: true
```

2. Start the backend server:
```bash
cd backend
go run cmd/main.go
```

### Frontend Testing
The frontend will automatically connect to WebSocket at `ws://localhost:8080/api/v1/ws` and:
- Send a "subscribe" message on connection
- Send "ping" messages every 30 seconds
- Handle incoming updates by invalidating relevant React Query caches

### Manual Testing with WebSocket Client
You can test with a WebSocket client like `wscat`:

```bash
# Install wscat
npm install -g wscat

# Connect to WebSocket
wscat -c ws://localhost:8080/api/v1/ws

# Send subscribe message
{"type":"subscribe","timestamp":1234567890}

# You should receive
{"type":"subscribed","timestamp":1234567890}

# Send ping
{"type":"ping","timestamp":1234567890}

# You should receive
{"type":"pong","timestamp":1234567890}
```

### Expected Update Format

#### Session Update:
```json
{
  "type": "session_update",
  "data": {
    "session_id": "session-123",
    "session": {
      "id": "session-123",
      "title": "Project Name",
      "project_path": "/path/to/project",
      "status": "active",
      "last_activity": "2024-01-01T12:00:00Z",
      "tokens_used": {
        "input_tokens": 1000,
        "output_tokens": 500,
        "total_tokens": 1500,
        "estimated_cost": 0.015
      }
    }
  },
  "timestamp": 1234567890
}
```

#### Activity Update:
```json
{
  "type": "activity_update",
  "data": {
    "activity": {
      "timestamp": "2024-01-01T12:00:00Z",
      "type": "file_modified",
      "session_id": "session-123",
      "session_name": "Project Name",
      "details": "Modified /src/app.js using Edit"
    }
  },
  "timestamp": 1234567890
}
```

#### Metrics Update:
```json
{
  "type": "metrics_update",
  "data": {
    "session_id": "session-123",
    "usage": {
      "input_tokens": 100,
      "output_tokens": 50,
      "total_tokens": 150,
      "estimated_cost": 0.0015
    }
  },
  "timestamp": 1234567890
}
```

## Debugging

Enable debug logging by setting:
```yaml
features:
  debug_mode: true
```

This will show detailed WebSocket activity in the backend logs.