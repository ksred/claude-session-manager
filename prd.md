# Claude Code Session Manager - Product Requirements Document

## Executive Summary

Build a beautiful web-based session manager for Claude Code that provides real-time visibility into all active sessions, token usage metrics, project context, and session state. The tool will leverage Claude Code's filesystem structure to parse session data and provide an intuitive web dashboard accessible from any browser for managing multiple concurrent coding sessions.

## Problem Statement

Current pain points with Claude Code workflow:
- **No visibility** into what each terminal tab/session is working on
- **Context switching confusion** between multiple projects and git worktrees
- **No session state awareness** (working, idle, complete, error)
- **Manual session management** without central oversight
- **No usage tracking** for tokens, costs, or productivity metrics

## Product Vision

Create a `claude-session-manager` web application that serves as a **mission control center** for all Claude Code activities, providing developers with:
- Real-time session monitoring and state management
- Beautiful web dashboard with rich, interactive visualizations
- Project-aware context switching and organization
- Token usage and cost tracking with advanced charting
- Activity feeds and productivity insights
- Browser-based access from anywhere on the network
- Responsive design for desktop and mobile viewing

## Technical Architecture

### Technology Stack

**Backend**:
- **Language**: Go (for performance and concurrent session monitoring)
- **Web Framework**: [Gin](https://github.com/gin-gonic/gin) or [Echo](https://echo.labstack.com/) (high-performance HTTP)
- **WebSocket**: [Gorilla WebSocket](https://github.com/gorilla/websocket) (real-time updates)
- **Data Source**: Claude Code filesystem (`~/.claude/` directory)

**Frontend**:
- **Framework**: React with TypeScript (component-based, type-safe)
- **State Management**: Zustand or Context API
- **Styling**: Tailwind CSS (utility-first, matches design system)
- **Charts**: [Chart.js](https://www.chartjs.org/) or [Recharts](https://recharts.org/) (interactive visualizations)
- **Real-time**: Socket.io-client or native WebSocket
- **Build Tool**: Vite (fast development and building)

### Data Sources
1. **Session Files**: `~/.claude/projects/[encoded-path]/[session-uuid].jsonl`
2. **Project Config**: `CLAUDE.md`, `.claude/settings.json`, `.claude/settings.local.json`
3. **Custom Commands**: `.claude/commands/` directory
4. **Git Context**: Git worktree detection and branch information

## Core Features

### 1. Session Discovery & Monitoring

**Data Collection**:
- Scan `~/.claude/projects/` for active session files
- Parse JSONL files to extract session metadata
- Decode directory paths to determine project context
- Monitor file modification times for activity detection

**Session State Detection**:
- **Working**: Recent activity (< 2 minutes)
- **Idle**: No recent activity (2-15 minutes)
- **Complete**: Session ended gracefully
- **Error**: Last message indicates error state

### 2. Real-Time Dashboard Interface

**Web Layout Structure**:
- **Responsive Grid Layout**: Flexbox/CSS Grid for adaptive layouts
- **Navigation Bar**: Logo, global controls, settings, theme toggle
- **Sidebar**: Collapsible session list with search/filter
- **Main Dashboard**: Metrics cards, interactive charts, activity feed
- **Mobile-First**: Stacked layout on small screens, side-by-side on desktop

**Key UI Improvements over Terminal**:
- **Interactive Charts**: Hover tooltips, zoom, pan, time range selection
- **Smooth Animations**: CSS transitions for state changes, loading states
- **Rich Interactions**: Click to select, drag to reorder, multi-select
- **Glassmorphism Design**: Modern frosted glass effects with backdrop blur
- **Responsive Typography**: Dynamic font sizing based on viewport
- **Accessibility**: ARIA labels, keyboard navigation, screen reader support

### 3. Session Management Features

**Session List (Sidebar)**:
- Session name with project context
- Visual status indicators (working/idle/complete/error)
- Progress bars showing task completion
- Last activity timestamps
- Project/worktree information
- Quick keyboard navigation

**Detailed Session View**:
- Current task description from CLAUDE.md or recent activity
- Token usage metrics (input/output/total)
- Session duration and cost estimates
- Files modified count and list
- Git branch/worktree context

### 4. Metrics & Analytics

**Real-Time Metrics**:
- **Token Usage**: Input/output tokens per session
- **Cost Tracking**: Estimated costs based on token usage
- **Session Time**: Active time tracking
- **File Modifications**: Count and list of changed files
- **Project Activity**: Cross-session project insights

**Visualizations**:
- Token usage charts (hourly/daily trends)
- Session activity timeline
- Cost breakdown by project
- Productivity metrics (tokens per minute, etc.)

### 5. Project Context Integration

**Git Worktree Awareness**:
- Detect and display git worktree information
- Group sessions by base repository
- Show branch context for each session
- Visual git graph integration

**CLAUDE.md Integration**:
- Parse project descriptions and tech stacks
- Extract current project status and goals
- Display project-specific commands and workflows
- Rich markdown rendering in the UI

### 6. Web-Specific Features

**Advanced Visualizations**:
- **Token Usage Heatmap**: Daily/weekly usage patterns
- **Cost Breakdown Pie Chart**: By project, time period
- **Session Timeline**: Interactive Gantt-style view
- **Performance Metrics**: Response time graphs

**Data Export & Sharing**:
- **Export Options**: CSV, JSON, PDF reports
- **Share Links**: Generate shareable dashboard snapshots
- **Team Features**: Multi-user support with permissions
- **API Access**: RESTful API for integration

**UI Enhancements**:
- **Search Everything**: Global search across sessions, files, messages
- **Filtering**: By status, project, date range, cost
- **Bookmarks**: Save favorite sessions or views
- **Notifications**: Browser notifications for important events
- **Themes**: Dark, light, and custom color schemes

## User Interface Specification

### Color Scheme
```go
// Primary Colors
Background    = "#0d0d0d"  // Deep black
Surface       = "#1a1a1a"  // Dark grey
Border        = "#333333"  // Medium grey
Text          = "#f0f0f0"  // Light grey
TextSecondary = "#888888"  // Muted grey

// Accent Colors
Primary   = "#00ff96"  // Bright green
Secondary = "#00d4ff"  // Cyan blue
Warning   = "#ffbd2e"  // Yellow
Error     = "#ff5f56"  // Red
Success   = "#27ca3f"  // Green
```

### Component Styling (CSS/Tailwind)

**Session Status Indicators**:
```css
/* Working State */
.status-working {
  @apply bg-primary text-black px-2 py-1 rounded animate-pulse;
}

/* Idle State */
.status-idle {
  @apply bg-warning text-black px-2 py-1 rounded;
}

/* Complete State */
.status-complete {
  @apply bg-success text-black px-2 py-1 rounded;
}

/* Error State */
.status-error {
  @apply bg-error text-white px-2 py-1 rounded;
}
```

**Session Cards**:
```css
/* Base Session Card */
.session-card {
  @apply border border-gray-700 rounded-lg p-4 mb-2 
         transition-all duration-200 hover:border-gray-600
         backdrop-blur-sm bg-surface/90;
}

/* Active Session */
.session-card.active {
  @apply border-primary bg-primary/10 shadow-lg shadow-primary/20;
}

/* Glassmorphism Effects */
.glass-panel {
  @apply bg-surface/80 backdrop-blur-md border border-white/10
         shadow-xl rounded-xl;
}
```

### Navigation & Interactions

**Keyboard Shortcuts**:
| Key | Action |
|-----|--------|
| `↑/↓` | Navigate sessions |
| `←/→` | Switch tabs |
| `Enter` | Select/focus session |
| `Tab` | Cycle through panels |
| `Ctrl+R` | Refresh data |
| `/` | Focus search |
| `Esc` | Close modals/clear search |
| `1-9` | Jump to session by number |
| `Ctrl+K` | Command palette |

**Mouse/Touch Interactions**:
- Click session to select
- Double-click to expand details
- Drag to reorder sessions
- Pinch to zoom charts
- Swipe to navigate on mobile
- Right-click for context menu

## Implementation Details

### File Structure
```
claude-session-manager/
├── backend/
│   ├── cmd/
│   │   └── main.go            # Server entry point
│   ├── internal/
│   │   ├── api/               # HTTP/WebSocket handlers
│   │   │   ├── routes.go      # API route definitions
│   │   │   ├── handlers.go    # Request handlers
│   │   │   └── websocket.go   # Real-time updates
│   │   ├── claude/
│   │   │   ├── parser.go      # JSONL parsing logic
│   │   │   ├── session.go     # Session data structures
│   │   │   ├── project.go     # Project context parsing
│   │   │   └── watcher.go     # File system monitoring
│   │   └── config/
│   │       └── config.go      # Configuration management
│   ├── pkg/
│   │   └── utils/
│   │       ├── encoding.go    # Path encoding/decoding
│   │       └── git.go        # Git utilities
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── components/        # React components
│   │   │   ├── Dashboard/     # Main dashboard view
│   │   │   ├── SessionList/   # Session sidebar
│   │   │   ├── MetricsPanel/  # Metrics cards
│   │   │   ├── Charts/        # Chart components
│   │   │   └── ActivityFeed/  # Activity timeline
│   │   ├── hooks/             # Custom React hooks
│   │   ├── services/          # API/WebSocket services
│   │   ├── store/             # State management
│   │   ├── styles/            # Global styles
│   │   └── App.tsx           # Root component
│   ├── public/
│   ├── package.json
│   ├── tailwind.config.js
│   ├── tsconfig.json
│   └── vite.config.ts
├── docker-compose.yml         # Easy local development
└── README.md
```

### Data Structures

```go
type Session struct {
    ID              string             `json:"id"`
    ProjectPath     string             `json:"project_path"`
    ProjectName     string             `json:"project_name"`
    GitBranch       string             `json:"git_branch"`
    GitWorktree     string             `json:"git_worktree"`
    Status          SessionStatus      `json:"status"`
    StartTime       time.Time          `json:"start_time"`
    LastActivity    time.Time          `json:"last_activity"`
    CurrentTask     string             `json:"current_task"`
    TokensUsed      TokenUsage         `json:"tokens_used"`
    FilesModified   []string           `json:"files_modified"`
    Messages        []Message          `json:"messages"`
}

type TokenUsage struct {
    InputTokens     int     `json:"input_tokens"`
    OutputTokens    int     `json:"output_tokens"`
    TotalTokens     int     `json:"total_tokens"`
    EstimatedCost   float64 `json:"estimated_cost"`
}

type SessionStatus int

const (
    StatusWorking SessionStatus = iota
    StatusIdle
    StatusComplete
    StatusError
)
```

### Core Parsing Logic

**Session Discovery**:
```go
func DiscoverSessions() ([]Session, error) {
    claudeDir := filepath.Join(os.Getenv("HOME"), ".claude", "projects")
    var sessions []Session
    
    err := filepath.Walk(claudeDir, func(path string, info os.FileInfo, err error) error {
        if strings.HasSuffix(path, ".jsonl") && !strings.Contains(path, "summary") {
            session, err := ParseSessionFile(path)
            if err == nil {
                sessions = append(sessions, session)
            }
        }
        return nil
    })
    
    return sessions, err
}
```

**JSONL Parsing**:
```go
func ParseSessionFile(filePath string) (Session, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return Session{}, err
    }
    defer file.Close()
    
    var session Session
    scanner := bufio.NewScanner(file)
    
    for scanner.Scan() {
        var message Message
        if err := json.Unmarshal(scanner.Bytes(), &message); err == nil {
            session.Messages = append(session.Messages, message)
            updateSessionMetrics(&session, message)
        }
    }
    
    return session, nil
}
```

### Real-Time Updates

**WebSocket Communication**:
```go
// Backend WebSocket handler
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer conn.Close()
    
    // Send initial session data
    sessions, _ := claude.DiscoverSessions()
    conn.WriteJSON(map[string]interface{}{
        "type": "sessions",
        "data": sessions,
    })
    
    // Watch for changes and push updates
    go watchAndPushUpdates(conn)
}
```

**Frontend WebSocket Connection**:
```typescript
// React hook for real-time updates
export const useSessionUpdates = () => {
    const [sessions, setSessions] = useState<Session[]>([]);
    
    useEffect(() => {
        const ws = new WebSocket('ws://localhost:8080/ws');
        
        ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
            if (message.type === 'sessions') {
                setSessions(message.data);
            }
        };
        
        return () => ws.close();
    }, []);
    
    return sessions;
};
```

## UI Component Specifications

### Session List Component

**Features**:
- Virtualized scrolling for hundreds of sessions
- Real-time status indicators with smooth animations
- Project grouping with collapsible sections
- Search and filter capabilities
- Drag-and-drop reordering
- Multi-select with batch actions

**React Component Structure**:
```tsx
interface SessionListProps {
    sessions: Session[];
    selectedId: string;
    onSelect: (id: string) => void;
}

const SessionList: React.FC<SessionListProps> = ({ sessions, selectedId, onSelect }) => {
    return (
        <div className="session-list glass-panel h-full overflow-y-auto">
            <SearchBar />
            <FilterControls />
            {sessions.map(session => (
                <SessionCard 
                    key={session.id}
                    session={session}
                    isSelected={session.id === selectedId}
                    onClick={() => onSelect(session.id)}
                />
            ))}
        </div>
    );
};
```

### Metrics Panel Component

**Layout**:
```
┌─ Metrics ──────────────────────────────────────────────┐
│ ┌─Tokens──┐ ┌─Cost────┐ ┌─Files───┐ ┌─Time────────┐ │
│ │ 42.1K   │ │ $0.84   │ │ 7       │ │ 2h 15m      │ │
│ │ +2.3K/h │ │ +$0.12  │ │ 2 recent│ │ Active: 2m  │ │
│ └─────────┘ └─────────┘ └─────────┘ └─────────────┘ │
└────────────────────────────────────────────────────────┘
```

### Activity Feed Component

**Features**:
- Chronological list of recent actions
- Action type icons (⚡ working, ✓ complete, ✗ error)
- Token usage per action
- Session context for each activity
- Auto-scrolling with new activities

### Chart Component

**Interactive Chart with Chart.js**:
```tsx
interface TokenChartProps {
    data: ChartData;
    timeRange: '24h' | '7d' | '30d';
}

const TokenUsageChart: React.FC<TokenChartProps> = ({ data, timeRange }) => {
    const options = {
        responsive: true,
        plugins: {
            legend: { position: 'top' },
            title: { display: true, text: 'Token Usage' },
            tooltip: {
                callbacks: {
                    label: (context) => `${context.parsed.y.toLocaleString()} tokens`
                }
            }
        },
        scales: {
            y: { beginAtZero: true }
        },
        animation: {
            duration: 750
        }
    };
    
    return (
        <div className="chart-container glass-panel p-4">
            <div className="flex justify-between mb-4">
                <h3 className="text-lg font-semibold">Token Usage</h3>
                <TimeRangeSelector value={timeRange} />
            </div>
            <Line data={data} options={options} />
        </div>
    );
};
```

## Performance Requirements

- **Server Startup**: < 500ms to first response
- **Page Load**: < 2s initial load, < 100ms subsequent navigation
- **WebSocket Latency**: < 50ms for real-time updates
- **Memory Usage**: < 100MB backend, optimized frontend bundles
- **CPU Usage**: < 10% during normal operation
- **File I/O**: Efficient streaming parser for large JSONL files (>100MB)
- **Concurrent Users**: Support 10+ simultaneous dashboard connections
- **API Response**: < 100ms for session list, < 200ms for detailed data

## Installation & Distribution

**Installation Methods**:
```bash
# Using Docker (recommended)
docker run -d -p 8080:8080 -v ~/.claude:/root/.claude claude-session-manager

# Using Docker Compose
docker-compose up -d

# Manual installation
git clone https://github.com/user/claude-session-manager
cd claude-session-manager
make install
claude-session-manager serve

# Homebrew (future)
brew install claude-session-manager
brew services start claude-session-manager
```

**Usage**:
```bash
# Start the server
claude-session-manager serve --port 8080

# Open in browser
open http://localhost:8080

# Run with custom config
claude-session-manager serve --config config.yaml

# Run in development mode
claude-session-manager serve --dev

# API-only mode (no frontend)
claude-session-manager serve --api-only
```

**Accessing the Dashboard**:
- Local: `http://localhost:8080`
- Network: `http://[your-ip]:8080`
- Secure: Configure with TLS certificates for HTTPS

## Configuration Options

**Config File** (`~/.config/claude-session-manager/config.yaml`):
```yaml
# Server settings
server:
  host: "0.0.0.0"
  port: 8080
  cors_origins: ["http://localhost:3000"]
  tls:
    enabled: false
    cert_file: ""
    key_file: ""

# Claude Code settings
claude:
  home_dir: "~/.claude"
  projects_dir: "~/.claude/projects"
  watch_interval: 2s

# Token pricing
pricing:
  input_token_cost: 0.00001
  output_token_cost: 0.00003
  currency: "USD"

# UI settings
ui:
  theme: "dark"  # dark, light, auto
  language: "en"
  date_format: "relative"  # relative, absolute
  
# Features
features:
  auto_refresh: true
  notifications: true
  export_enabled: true
  multi_user: false

# API settings
api:
  rate_limit: 100  # requests per minute
  enable_swagger: true
  api_keys: []  # Optional API key authentication

# Database (optional, for persistence)
database:
  type: "sqlite"  # sqlite, postgres
  connection: "~/.claude-session-manager/data.db"
```

## Testing Strategy

### Unit Tests
- JSONL parsing accuracy
- Session state detection logic
- Path encoding/decoding utilities
- Metrics calculation functions

### Integration Tests
- End-to-end session discovery
- File system watcher functionality
- UI component rendering
- Keyboard navigation flows

### Performance Tests
- Large session file parsing
- Memory usage profiling
- Render performance benchmarks
- Real-time update efficiency

## Success Metrics

### User Experience
- **Time to Context**: < 3 seconds to load and understand session states
- **Session Navigation**: Instant switching between sessions
- **Error Detection**: Real-time alerts with actionable notifications
- **Mobile Experience**: Fully functional on tablets and smartphones

### Technical Performance
- **Accuracy**: 100% session detection and state classification
- **Responsiveness**: < 50ms UI interactions, < 100ms data updates
- **Reliability**: 99.9% uptime with automatic reconnection
- **Scalability**: Handle 100+ concurrent sessions without degradation

### Adoption Metrics
- **User Engagement**: Daily active usage by Claude Code developers
- **Feature Utilization**: 80%+ users leveraging charts and export features
- **Performance Satisfaction**: < 5% reports of performance issues

## Future Enhancements

### Phase 2 Features
- **Session Actions**: Stop/restart/terminate sessions from dashboard
- **Custom Dashboards**: Drag-and-drop widget customization
- **Advanced Analytics**: ML-powered insights and predictions
- **Integrations**: Slack, Discord, GitHub notifications
- **Session Recording**: Replay session interactions

### Phase 3 Features
- **Team Collaboration**: Shared dashboards with role-based access
- **Cloud Deployment**: SaaS offering with data synchronization
- **Plugin Marketplace**: Community extensions and widgets
- **Mobile Apps**: Native iOS/Android applications
- **AI Assistant**: Natural language queries about session data
- **Cost Optimization**: Recommendations for reducing token usage

## Dependencies

**Backend** (`go.mod`):
```go
module github.com/user/claude-session-manager

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/gorilla/websocket v1.5.1
    github.com/fsnotify/fsnotify v1.7.0
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.0
    gopkg.in/yaml.v3 v3.0.1
)
```

**Frontend** (`package.json`):
```json
{
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "react-router-dom": "^6.20.0",
    "chart.js": "^4.4.0",
    "react-chartjs-2": "^5.2.0",
    "zustand": "^4.4.0",
    "axios": "^1.6.0",
    "socket.io-client": "^4.6.0",
    "tailwindcss": "^3.3.0",
    "@heroicons/react": "^2.0.0"
  },
  "devDependencies": {
    "@types/react": "^18.2.0",
    "@vitejs/plugin-react": "^4.2.0",
    "typescript": "^5.3.0",
    "vite": "^5.0.0"
  }
}
```

## Implementation Timeline

### Week 1: Backend Infrastructure
- [ ] Go server setup with Gin/Echo
- [ ] Session discovery and JSONL parsing
- [ ] RESTful API endpoints
- [ ] WebSocket implementation
- [ ] File system watcher integration

### Week 2: Frontend Foundation
- [ ] React + TypeScript setup with Vite
- [ ] Component architecture and routing
- [ ] Tailwind CSS configuration
- [ ] Session list and basic dashboard layout
- [ ] WebSocket connection and state management

### Week 3: Rich Dashboard Features
- [ ] Interactive charts with Chart.js
- [ ] Metrics cards and visualizations
- [ ] Activity feed with real-time updates
- [ ] Search, filter, and sorting functionality
- [ ] Responsive design implementation

### Week 4: Polish & Deployment
- [ ] Docker containerization
- [ ] Performance optimization
- [ ] Error handling and loading states
- [ ] Documentation and API docs
- [ ] Deployment scripts and CI/CD

### Week 5: Advanced Features
- [ ] Export functionality
- [ ] User preferences and theming
- [ ] Browser notifications
- [ ] Testing and bug fixes
- [ ] Beta release preparation

---

**Ready to build!** This updated PRD transforms Claude Session Manager into a modern web application that provides a superior user experience with rich visualizations, real-time updates, and cross-platform accessibility. The web-based approach eliminates terminal limitations and opens up possibilities for advanced features and team collaboration.
