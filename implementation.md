# Claude Session Manager - Implementation Plan

## Overview
Building a terminal-based session manager for Claude Code that provides real-time visibility into active sessions, token usage metrics, and project context using Go, Bubble Tea, and Lipgloss.

## Architecture Summary
- **Language**: Go with Bubble Tea TUI framework
- **Data Source**: Claude Code filesystem (`~/.claude/projects/`)
- **UI Components**: Session list, metrics panel, activity feed, token usage chart
- **Real-time Updates**: File system watching for live session monitoring

## Implementation Status

### Phase 1: Core Infrastructure ✅ COMPLETED
- [x] Requirements analysis and planning
- [x] Go project setup with dependencies (Bubble Tea, Lipgloss, fsnotify)
- [x] Session discovery from `~/.claude/projects/` directory
- [x] JSONL parsing for extracting session metadata
- [x] Core data structures (Session, TokenUsage, SessionStatus)

### Phase 2: UI Foundation ✅ COMPLETED
- [x] Bubble Tea application architecture
- [x] Main layout with sidebar and detail panels
- [x] Session list component with status indicators
- [x] Basic keyboard navigation

### Phase 3: Rich Dashboard Features ✅ COMPLETED
- [x] Metrics panel (tokens, cost, files, time)
- [x] Token usage chart with ASCII/Unicode bars
- [x] Activity feed with recent actions
- [x] Session state detection (Working/Idle/Complete/Error)

### Phase 4: Real-time & Polish ✅ COMPLETED
- [x] File system watcher for live updates
- [x] Complete keyboard navigation system
- [x] Color scheme and styling to match design
- [x] Documentation and README

## Key Components

### 1. Session Discovery Engine
- Scans `~/.claude/projects/` for `.jsonl` files
- Decodes directory paths to determine project context
- Monitors file modification times for activity detection

### 2. Dashboard UI Layout
```
┌─ Header ─────────────────────────────────────────┐
│ claude-session-manager v1.0.0    [controls]     │
├─ Main Content ──────────────────────────────────┤
│ ┌─Sessions─┐ ┌─ Session Detail ─────────────────┐ │
│ │List      │ │ Metrics | Chart | Activity      │ │
│ │          │ │                                 │ │
│ └──────────┘ └─────────────────────────────────┘ │
└──────────────────────────────────────────────────┘
```

### 3. Session State Logic
- **Working**: Recent activity (< 2 minutes)
- **Idle**: No recent activity (2-15 minutes) 
- **Complete**: Session ended gracefully
- **Error**: Last message indicates error state

### 4. Data Structures
```go
type Session struct {
    ID              string
    ProjectPath     string
    ProjectName     string
    Status          SessionStatus
    TokensUsed      TokenUsage
    FilesModified   []string
    LastActivity    time.Time
    // ... additional fields
}
```

## Technical Decisions

### Dependencies
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/fsnotify/fsnotify` - File watching
- Standard library for JSONL parsing and file operations

### Color Scheme (from PRD)
- Background: `#0d0d0d` (Deep black)
- Primary: `#00ff96` (Bright green)
- Secondary: `#00d4ff` (Cyan blue)
- Warning: `#ffbd2e` (Yellow)
- Error: `#ff5f56` (Red)

### File Structure
```
claude-session-manager/
├── cmd/main.go                 # CLI entry point
├── internal/
│   ├── claude/                 # Claude-specific logic
│   ├── ui/                     # UI components
│   └── config/                 # Configuration
├── pkg/utils/                  # Utilities
└── go.mod
```

## Next Steps
1. Initialize Go module and dependencies
2. Implement session discovery and parsing
3. Build core UI components
4. Add real-time updates and polish

## Success Criteria
- Display all active Claude sessions with status
- Show real-time token usage and costs
- Provide intuitive keyboard navigation
- Update automatically as sessions change
- Match the visual design from dashboard.png