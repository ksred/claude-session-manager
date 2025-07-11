package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ksred/claude-session-manager/internal/config"
	"github.com/sirupsen/logrus"
)

// CORSMiddleware returns a middleware function that handles CORS
func CORSMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set allowed origins
		origin := c.Request.Header.Get("Origin")
		allowedOrigin := ""

		// Check if origin is allowed
		for _, allowed := range cfg.Server.CORS.AllowedOrigins {
			if allowed == "*" {
				// When using credentials, we must echo the actual origin, not "*"
				if cfg.Server.CORS.AllowCredentials && origin != "" {
					allowedOrigin = origin
				} else {
					allowedOrigin = "*"
				}
				break
			} else if allowed == origin {
				allowedOrigin = origin
				break
			}
		}

		if allowedOrigin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		}

		// Set credentials
		if cfg.Server.CORS.AllowCredentials {
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Set allowed headers
		if len(cfg.Server.CORS.AllowedHeaders) > 0 {
			c.Writer.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.Server.CORS.AllowedHeaders, ", "))
		}

		// Set allowed methods
		if len(cfg.Server.CORS.AllowedMethods) > 0 {
			c.Writer.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.Server.CORS.AllowedMethods, ", "))
		}

		// Set max age
		if cfg.Server.CORS.MaxAge > 0 {
			c.Writer.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.Server.CORS.MaxAge))
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// LoggingMiddleware returns a middleware function that logs requests
func LoggingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		if raw != "" {
			path = path + "?" + raw
		}

		// Log request details
		entry := logger.WithFields(logrus.Fields{
			"status":     c.Writer.Status(),
			"method":     c.Request.Method,
			"path":       path,
			"ip":         c.ClientIP(),
			"latency":    latency,
			"user-agent": c.Request.UserAgent(),
		})

		if c.Writer.Status() >= 500 {
			entry.Error("Server error")
		} else if c.Writer.Status() >= 400 {
			entry.Warn("Client error")
		} else {
			entry.Info("Request processed")
		}
	}
}
