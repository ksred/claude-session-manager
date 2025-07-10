# Claude Session Manager

A beautiful web-based dashboard for monitoring and managing Claude Code sessions in real-time.

## Quick Start

```bash
docker run -p 80:80 -v ~/.claude:/data/claude ksred/claude-session-manager
```

Then open http://localhost in your browser.

## What's Included

- **Real-time Dashboard**: Monitor all your Claude Code sessions with live updates
- **Interactive Charts**: Beautiful visualizations for token usage and costs
- **Multi-Project Support**: Manage sessions across different projects and git worktrees
- **Cost Tracking**: Monitor token usage and estimated costs
- **Modern UI**: Responsive design with dark/light themes
- **High Performance**: Built with Go backend and React frontend

## Requirements

- Claude Code installed with sessions in `~/.claude/`
- Docker

## Environment Variables

- `CLAUDE_DIR`: Custom Claude directory path (defaults to `/data/claude` in container)
- `PORT`: Backend API port (defaults to 8080)

## Custom Claude Directory

```bash
docker run -p 80:80 \
  -e CLAUDE_DIR=/custom/claude/path \
  -v /your/custom/claude/dir:/custom/claude/path \
  ksred/claude-session-manager
```

## Features

🌐 **Web-Based Dashboard**: Access from any browser on your network  
📊 **Real-time Monitoring**: Live updates via WebSocket connections  
📈 **Interactive Charts**: Beautiful visualizations for token usage and costs  
🎯 **Multi-Project Support**: Manage sessions across different projects  
💰 **Cost Tracking**: Monitor token usage and estimated costs  
🎨 **Modern UI**: Responsive design with dark/light themes  
⚡ **High Performance**: Built with Go backend and React frontend  
📱 **Mobile Friendly**: Fully responsive for desktop and mobile viewing  

## Architecture

- **Backend**: Go with Gin framework, SQLite database
- **Frontend**: React with TypeScript, Tailwind CSS
- **Real-time**: WebSocket connections for live updates
- **Data Source**: Reads directly from Claude Code filesystem

## Source Code

Full source code and documentation available at:
https://github.com/ksred/claude-session-manager

## License

MIT License