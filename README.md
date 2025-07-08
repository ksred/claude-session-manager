# Claude Session Manager

A beautiful web-based dashboard for monitoring and managing Claude Code sessions in real-time.

![Claude Session Manager Dashboard](dashboard.png)

## Overview

Claude Session Manager provides a modern web interface to monitor all your active Claude Code sessions, track token usage, and manage multiple projects simultaneously. It reads session data directly from Claude's local file system and presents it through an intuitive web dashboard accessible from any browser.

## Features

- 🌐 **Web-Based Dashboard**: Access from any browser on your network
- 📊 **Real-time Monitoring**: Live updates via WebSocket connections
- 📈 **Interactive Charts**: Beautiful visualizations for token usage and costs
- 🎯 **Multi-Project Support**: Manage sessions across different projects and git worktrees
- 💰 **Cost Tracking**: Monitor token usage and estimated costs
- 🎨 **Modern UI**: Responsive design with dark/light themes
- ⚡ **High Performance**: Built with Go backend and React frontend
- 📱 **Mobile Friendly**: Fully responsive for desktop and mobile viewing

## Quick Start

### Using Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/yourusername/claude-session-manager.git
cd claude-session-manager

# Start with Docker Compose
docker-compose up -d

# Open in browser
open http://localhost:8080
```

### Manual Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/claude-session-manager.git
cd claude-session-manager

# Backend setup
cd backend
go mod download
go build -o claude-session-manager ./cmd/main.go

# Frontend setup
cd ../frontend
npm install
npm run build

# Run the backend server
cd ../backend
./claude-session-manager serve
```

## Architecture

### Technology Stack

**Backend**:
- **Language**: Go (high-performance server)
- **Web Framework**: Gin/Echo
- **WebSocket**: Gorilla WebSocket
- **Data Source**: Claude Code filesystem (`~/.claude/`)

**Frontend**:
- **Framework**: React with TypeScript
- **Build Tool**: Vite
- **Styling**: Tailwind CSS
- **Charts**: Chart.js/Recharts
- **State Management**: Zustand

## Project Structure

```
claude-session-manager/
├── backend/                   # Go backend server
│   ├── cmd/                  # Application entry points
│   │   └── main.go          
│   ├── internal/            
│   │   ├── api/             # HTTP/WebSocket handlers
│   │   ├── claude/          # Claude session parsing
│   │   └── config/          # Configuration management
│   ├── pkg/                 # Shared packages
│   ├── Dockerfile           
│   └── go.mod              
├── frontend/                  # React frontend
│   ├── src/                
│   │   ├── components/      # React components
│   │   ├── hooks/           # Custom React hooks
│   │   ├── services/        # API services
│   │   ├── store/           # State management
│   │   ├── styles/          # Global styles
│   │   └── App.tsx         
│   ├── public/             
│   ├── Dockerfile          
│   ├── package.json        
│   └── vite.config.ts      
├── docker-compose.yml         # Docker orchestration
└── README.md                  # This file
```

## Configuration

Create a configuration file at `~/.config/claude-session-manager/config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

claude:
  home_dir: "~/.claude"
  watch_interval: 2s

pricing:
  input_token_cost: 0.00001
  output_token_cost: 0.00003

ui:
  theme: "dark"
  auto_refresh: true
```

## Development

### Backend Development

```bash
cd backend
go mod download
go run ./cmd/main.go serve --dev
```

### Frontend Development

```bash
cd frontend
npm install
npm run dev
```

### Running Tests

```bash
# Backend tests
cd backend
go test ./...

# Frontend tests
cd frontend
npm test
```

## API Documentation

The backend provides a RESTful API with the following endpoints:

- `GET /api/sessions` - List all sessions
- `GET /api/sessions/:id` - Get session details
- `GET /api/metrics` - Get aggregated metrics
- `WS /ws` - WebSocket connection for real-time updates

## Browser Compatibility

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+
- Mobile browsers (iOS Safari, Chrome Mobile)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details