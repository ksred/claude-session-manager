# Build stage for Go backend
FROM golang:1.23-alpine AS backend-builder

# Build arguments
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

RUN apk add --no-cache git gcc musl-dev sqlite-dev

WORKDIR /backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.BuildDate=${BUILD_DATE}" \
    -a -installsuffix cgo -o claude-session-manager ./cmd/main.go

# Build stage for React frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /frontend
COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Final stage - single container with both services
FROM alpine:latest

# Labels for image metadata
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

LABEL org.opencontainers.image.title="Claude Session Manager" \
      org.opencontainers.image.description="Manage and analyze Claude AI sessions" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${GIT_COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.source="https://github.com/ksred/claude-session-manager" \
      org.opencontainers.image.vendor="ksred" \
      org.opencontainers.image.licenses="MIT"

# Install nginx, supervisor, and sqlite tools
RUN apk --no-cache add ca-certificates nginx supervisor sqlite

# Create user
RUN addgroup -g 1000 -S claude && \
    adduser -u 1000 -S claude -G claude

# Setup directories with proper permissions
RUN mkdir -p /app/backend /app/frontend /var/log/supervisor /run/nginx /data/claude && \
    chmod 755 /data && \
    chmod 755 /data/claude

# Copy backend binary
COPY --from=backend-builder /backend/claude-session-manager /app/backend/

# Copy frontend build
COPY --from=frontend-builder /frontend/dist /app/frontend/

# Copy nginx config
COPY nginx-unified.conf /etc/nginx/http.d/default.conf

# Create supervisor config directory and config
RUN mkdir -p /etc/supervisor/conf.d && \
    cat > /etc/supervisor/conf.d/supervisord.conf <<EOF
[supervisord]
nodaemon=true
user=root

[program:backend]
command=/app/backend/claude-session-manager serve
autostart=true
autorestart=true
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0
environment=PORT="8080",CLAUDE_DIR="/data/claude",SQLITE_JOURNAL_MODE="WAL",SQLITE_BUSY_TIMEOUT="10000"

[program:nginx]
command=nginx -g 'daemon off;'
autostart=true
autorestart=true
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0
EOF

# Set permissions - ensure data directory is writable by backend user
RUN chown -R claude:claude /app /data && \
    chown -R nginx:nginx /var/lib/nginx /var/log/nginx /run/nginx && \
    chmod +x /app/backend/claude-session-manager

# Expose port 80 for nginx
EXPOSE 80

# Create startup script with database integrity check
RUN cat > /app/start.sh <<'EOF'
#!/bin/sh
# Set CLAUDE_DIR from environment or default to /data/claude
export CLAUDE_DIR="${CLAUDE_DIR:-/data/claude}"
echo "Starting Claude Session Manager..."
echo "Using Claude directory: $CLAUDE_DIR"

# Ensure database directory exists and has correct permissions
mkdir -p "$CLAUDE_DIR"
chown -R claude:claude "$CLAUDE_DIR"

# Check database integrity if it exists
if [ -f "$CLAUDE_DIR/sessions.db" ]; then
    echo "Checking database integrity..."
    sqlite3 "$CLAUDE_DIR/sessions.db" "PRAGMA integrity_check;" || {
        echo "Database corruption detected, creating backup..."
        mv "$CLAUDE_DIR/sessions.db" "$CLAUDE_DIR/sessions.db.corrupt.$(date +%Y%m%d_%H%M%S)"
    }
fi

echo "Backend will be available at http://localhost:8080"
echo "Frontend will be available at http://localhost"
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
EOF

RUN chmod +x /app/start.sh

# Volume for .claude directory (can be overridden with CLAUDE_DIR env var)
VOLUME ["/data/claude"]

# Start both services
CMD ["/app/start.sh"]