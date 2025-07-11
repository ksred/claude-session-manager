# SQLite Database Corruption Fix for Docker

## Problem
"database disk image is malformed" error when running in Docker containers.

## Root Causes
1. **Permission issues**: Backend runs as `claude` user but may not have write access
2. **File system incompatibility**: Docker Desktop file sharing doesn't support SQLite locking
3. **Improper shutdown**: Container killed without closing database connections
4. **Missing WAL mode**: SQLite needs WAL mode for better concurrency in containers

## Solutions Implemented

### 1. Updated Dockerfile
- Added `sqlite` package for integrity checks
- Ensured proper directory permissions for `/data/claude`
- Set environment variables for SQLite WAL mode and timeouts

### 2. Database Connection Settings
- Enabled WAL (Write-Ahead Logging) mode
- Increased busy timeout to 10 seconds
- Set synchronous mode to NORMAL (balanced safety/performance)
- Added connection pooling limits

### 3. Startup Integrity Check
- Added database integrity check on container startup
- Automatic backup of corrupted databases
- Attempt to create new database if corruption detected

### 4. Added IntegrityChecker
- Checks database integrity on startup
- Enables WAL mode
- Sets recommended pragmas for Docker
- Provides backup and repair functionality

## Docker Run Commands

### Basic usage:
```bash
docker run -d \
  -p 80:80 \
  -v ~/.claude:/data/claude \
  --name claude-session-manager \
  ksred/claude-session-manager
```

### With custom database location:
```bash
docker run -d \
  -p 80:80 \
  -v ~/my-claude-data:/data/claude \
  -e CLAUDE_DIR=/data/claude \
  --name claude-session-manager \
  ksred/claude-session-manager
```

### For better reliability, use a Docker volume:
```bash
# Create a named volume
docker volume create claude-data

# Run with named volume
docker run -d \
  -p 80:80 \
  -v claude-data:/data/claude \
  --name claude-session-manager \
  ksred/claude-session-manager
```

## Prevention Tips

1. **Use Docker volumes** instead of bind mounts when possible
2. **Graceful shutdown**: Use `docker stop` not `docker kill`
3. **Regular backups**: The integrity checker creates backups automatically
4. **Monitor logs**: Check for corruption warnings in container logs

## Recovery from Corruption

If corruption occurs:
1. The startup script will detect it and create a backup
2. A new database will be initialized
3. The corrupt database is saved with timestamp: `sessions.db.corrupt.YYYYMMDD_HHMMSS`
4. You can attempt manual recovery using SQLite tools if needed

## Testing Database Integrity

To manually check database integrity:
```bash
docker exec claude-session-manager sqlite3 /data/claude/sessions.db "PRAGMA integrity_check;"
```

Expected output: `ok`