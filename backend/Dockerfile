# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o claude-session-manager ./cmd/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 -S claude && \
    adduser -u 1000 -S claude -G claude

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/claude-session-manager .

# Change ownership
RUN chown -R claude:claude /app

# Switch to non-root user
USER claude

# Expose port
EXPOSE 8080

# Run the application
CMD ["./claude-session-manager", "serve"]