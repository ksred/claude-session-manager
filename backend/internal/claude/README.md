# Claude Session Parser

This package provides functionality to parse and monitor Claude Code session files according to the PRD specifications.

## Features

### Session Management
- **Session Discovery**: Automatically discovers all Claude sessions in `~/.claude/projects/`
- **JSONL Parsing**: Parses session files to extract messages, metadata, and metrics
- **Status Detection**: Determines session status (Working, Idle, Complete, Error) based on activity
- **Project Context**: Extracts project paths, names, git branches, and worktree information

### Token Usage & Cost Tracking
- Tracks input and output tokens for each session
- Calculates estimated costs based on Claude pricing
- Provides methods to update and recalculate totals

### File System Monitoring
- Real-time monitoring of session file changes
- Event callbacks for created, modified, and deleted sessions
- Debounced updates to prevent excessive notifications
- Automatic directory watching for new projects

### Git Integration
- Extracts current git branch for projects
- Detects git worktree configurations
- Handles non-git directories gracefully

## Usage

### Basic Session Discovery
```go
sessions, err := claude.DiscoverSessions()
if err != nil {
    log.Fatal(err)
}

for _, session := range sessions {
    fmt.Printf("Session %s: %s\n", session.ID, session.Status)
    fmt.Printf("Project: %s (Branch: %s)\n", session.ProjectName, session.GitBranch)
    fmt.Printf("Tokens: %d (Cost: $%.4f)\n", session.TokensUsed.TotalTokens, session.TokensUsed.EstimatedCost)
}
```

### File System Watching
```go
watcher, err := claude.NewSessionWatcher(func(sessions []claude.Session) {
    fmt.Printf("Sessions updated: %d total\n", len(sessions))
})

// Optional: Set event callback for individual events
watcher.SetEventCallback(func(event claude.WatchEvent) {
    fmt.Printf("Event: %s for session %s\n", event.Type, event.SessionID)
})

if err := watcher.Start(); err != nil {
    log.Fatal(err)
}

// Don't forget to stop when done
defer watcher.Stop()
```

### Session Methods
```go
session := sessions[0]

// Timing information
duration := session.Duration()
timeSinceActive := session.TimeSinceLastActivity()
isActive := session.IsActive()

// Message statistics
totalMessages := session.GetMessageCount()
userMessages := session.GetUserMessageCount()
assistantMessages := session.GetAssistantMessageCount()

// Error handling
if session.HasErrors() {
    errors := session.GetErrorMessages()
    for _, err := range errors {
        fmt.Printf("Error: %s\n", err.Content)
    }
}

// Get last user request
lastRequest := session.GetLastUserMessage()
```

## Data Structures

### Session
The main session structure containing all metadata and messages:
- `ID`: Unique session identifier
- `ProjectPath`: Decoded project directory path
- `ProjectName`: Display name for the project
- `GitBranch`: Current git branch (if applicable)
- `GitWorktree`: Git worktree name (if applicable)
- `Status`: Current session status
- `StartTime`: When the session began
- `LastActivity`: Most recent activity timestamp
- `CurrentTask`: Description of current work
- `TokensUsed`: Token usage and cost information
- `FilesModified`: List of modified files
- `Messages`: All session messages

### TokenUsage
Tracks token consumption and costs:
- `InputTokens`: Total input tokens used
- `OutputTokens`: Total output tokens generated
- `TotalTokens`: Combined token count
- `EstimatedCost`: Calculated cost in USD

### Message
Individual message in a session:
- `ID`: Message identifier
- `Type`: Message type (user, assistant, error, etc.)
- `Role`: Sender role
- `Content`: Message content
- `Timestamp`: When the message was created
- `Usage`: Token usage for this message
- `Meta`: Additional metadata

## Testing

Run the test suite:
```bash
go test ./internal/claude/...
```

Run with verbose output:
```bash
go test ./internal/claude/... -v
```

## Configuration

Token pricing can be adjusted by modifying the constants in `session.go`:
```go
const (
    InputTokenPricePerK  = 0.01  // $0.01 per 1K input tokens
    OutputTokenPricePerK = 0.03  // $0.03 per 1K output tokens
)
```