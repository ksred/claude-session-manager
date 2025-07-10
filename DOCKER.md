# Docker Setup for Claude Session Manager

This project can be run as a single Docker container that includes both the backend API and frontend web interface.

## Quick Start

### Option 1: Using Docker Run (Recommended)

```bash
# Build the image
docker build -t ksred/claude-session-manager .

# Run the container
docker run -d \
  --name claude-session-manager \
  -p 80:80 \
  -v ~/.claude:/data/claude \
  ksred/claude-session-manager
```

The application will be available at http://localhost

### Option 2: Using Docker Compose

```bash
# Build and start the container
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the container
docker-compose down
```

## What's Included

The Docker container includes:
- **Backend API**: Go-based API server running on port 8080 internally
- **Frontend**: React-based web interface served by nginx on port 80
- **Process Manager**: Supervisor to manage both services
- **Volume Mount**: Access to your `~/.claude` directory (mounted at `/data/claude`)

## Architecture

The container uses:
- Multi-stage build to minimize image size
- Alpine Linux as the base image
- Supervisor to run both services in a single container
- Nginx as a reverse proxy for the API and to serve the frontend

## Environment Variables

- `PORT`: Backend API port (default: 8080)
- `ENV`: Environment mode (default: production)

## Ports

- Port 80: Web interface (nginx)
  - `/` - Frontend application
  - `/api/*` - Proxied to backend API
  - `/ws/*` - WebSocket connections

## Volume Mounts

- `~/.claude:/data/claude` - Mount of your Claude session data (read-write for database)

## Building from Source

```bash
# Clone the repository
git clone https://github.com/ksred/claude-session-manager.git
cd claude-session-manager

# Build the Docker image
docker build -t claude-session-manager .

# Run the container
docker run -d \
  --name claude-session-manager \
  -p 80:80 \
  -v ~/.claude:/data/claude \
  claude-session-manager
```

## Troubleshooting

### Container won't start
Check the logs:
```bash
docker logs claude-session-manager
```

### Can't access the web interface
1. Ensure port 80 is not already in use
2. Check that the container is running: `docker ps`
3. Try accessing http://localhost directly

### No session data showing
1. Verify your `~/.claude` directory exists and contains session data
2. Check that the volume mount is correct in your docker command
3. Ensure the container has read permissions for the mounted directory

## Security Notes

- The container needs read-write access to store the SQLite database in your Claude directory
- Both services run as non-privileged users where possible
- The frontend is served with security headers enabled
- All API requests are proxied through nginx

## Advanced Configuration

### Using a Different Port

```bash
docker run -d \
  --name claude-session-manager \
  -p 8080:80 \
  -v ~/.claude:/data/claude \
  ksred/claude-session-manager
```

Access at http://localhost:8080

### Custom Claude Directory

```bash
docker run -d \
  --name claude-session-manager \
  -p 80:80 \
  -v /path/to/your/claude/dir:/data/claude \
  ksred/claude-session-manager
```

### Development Mode

To run with separate containers for easier development:
1. Uncomment the separate service definitions in `docker-compose.yml`
2. Comment out the single container service
3. Run `docker-compose up -d`