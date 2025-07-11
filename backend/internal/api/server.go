package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ksred/claude-session-manager/internal/claude"
	"github.com/ksred/claude-session-manager/internal/config"
	"github.com/sirupsen/logrus"
)

// Server represents the API server
type Server struct {
	config         *config.Config
	router         *gin.Engine
	logger         *logrus.Logger
	wsHub          *WebSocketHub
	sessionsCache  []claude.Session
	sessionsMutex  sync.RWMutex
	sessionWatcher *claude.SessionWatcher
	sessionRepo    *claude.SessionRepository
	chatHandler    ChatMessageHandler
}

// NewServer creates a new API server instance
func NewServer(cfg *config.Config) *Server {
	// Set Gin mode based on debug setting
	if cfg.Features.DebugMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	logger := logrus.StandardLogger()

	// Create WebSocket hub if enabled
	var wsHub *WebSocketHub
	if cfg.Features.EnableWebSocket {
		wsHub = NewWebSocketHub(logger)
	}

	server := &Server{
		config:        cfg,
		router:        router,
		logger:        logger,
		wsHub:         wsHub,
		sessionsCache: []claude.Session{},
		sessionRepo:   claude.NewSessionRepository(cfg.Claude.HomeDirectory),
	}

	// Start WebSocket hub if enabled
	if server.wsHub != nil {
		// TODO: This server implementation needs context support for proper shutdown
		// For now, create a context that never cancels
		ctx := context.Background()
		go server.wsHub.Run(ctx)
	}

	// Initialize sessions cache
	if err := server.initializeSessionsCache(); err != nil {
		logger.WithError(err).Error("Failed to initialize sessions cache")
	}

	// Setup file watcher if enabled
	if cfg.Features.EnableFileWatcher {
		if err := server.setupFileWatcher(); err != nil {
			logger.WithError(err).Error("Failed to setup file watcher")
		}
	}

	// Start periodic cache refresh based on config
	if cfg.Claude.CacheRefreshRate > 0 {
		refreshInterval := time.Duration(cfg.Claude.CacheRefreshRate) * time.Minute
		go server.startPeriodicCacheRefresh(refreshInterval)
	}

	// Setup middleware
	server.setupMiddleware()

	// Setup routes
	server.setupRoutes()

	return server
}

// Start starts the server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	s.logger.WithFields(logrus.Fields{
		"address": addr,
		"port":    s.config.Server.Port,
		"host":    s.config.Server.Host,
	}).Info("Starting server")

	// Configure timeouts
	server := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  time.Duration(s.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.Server.WriteTimeout) * time.Second,
	}

	return server.ListenAndServe()
}

// setupMiddleware configures all middleware
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// CORS middleware if enabled
	if s.config.Server.CORS.Enabled {
		s.router.Use(CORSMiddleware(s.config))
	}

	// Logging middleware
	s.router.Use(LoggingMiddleware(s.logger))
}

// healthHandler handles health check requests
// @Summary Health check
// @Description Check the health status of the Claude Session Manager API
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse "Service is healthy"
// @Router /health [get]
func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "claude-session-manager",
	})
}

// initializeSessionsCache loads initial sessions into cache
func (s *Server) initializeSessionsCache() error {
	sessions, err := claude.DiscoverSessionsWithConfig(s.config)
	if err != nil {
		return err
	}

	s.sessionsMutex.Lock()
	s.sessionsCache = sessions
	s.sessionsMutex.Unlock()

	s.logger.WithField("session_count", len(sessions)).Info("Initialized sessions cache")
	return nil
}

// setupFileWatcher initializes the file system watcher for session files
func (s *Server) setupFileWatcher() error {
	watcher, err := claude.NewSessionWatcher(func(sessions []claude.Session) {
		// Update cache when sessions change
		s.sessionsMutex.Lock()
		s.sessionsCache = sessions
		s.sessionsMutex.Unlock()

		s.logger.WithField("session_count", len(sessions)).Debug("Sessions cache updated")

		// Broadcast update to all WebSocket clients if enabled
		if s.wsHub != nil {
			s.wsHub.BroadcastUpdate("sessions_updated", gin.H{
				"total_sessions": len(sessions),
				"timestamp":      time.Now().Unix(),
			})
		}
	})

	if err != nil {
		return err
	}

	// Set event callback for individual file events
	watcher.SetEventCallback(func(event claude.WatchEvent) {
		s.logger.WithFields(logrus.Fields{
			"event_type": event.Type,
			"session_id": event.SessionID,
		}).Debug("Session file event")

		// Broadcast specific event to WebSocket clients
		messageType := ""
		switch event.Type {
		case "created":
			messageType = "session_created"
		case "modified":
			messageType = "session_update"
		case "deleted":
			messageType = "session_deleted"
		}

		if messageType != "" && s.wsHub != nil {
			data := gin.H{
				"session_id": event.SessionID,
				"timestamp":  event.Timestamp.Unix(),
			}

			// Include session data for new/update events
			if event.Session != nil {
				data["session"] = sessionToResponse(*event.Session)
			}

			s.wsHub.BroadcastUpdate(messageType, data)
		}
	})

	s.sessionWatcher = watcher
	return watcher.Start()
}

// startPeriodicCacheRefresh refreshes the cache periodically as a backup
func (s *Server) startPeriodicCacheRefresh(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.initializeSessionsCache(); err != nil {
			s.logger.WithError(err).Error("Failed to refresh sessions cache")
		}
	}
}

// getSessionsFromCache returns sessions from cache
func (s *Server) getSessionsFromCache() ([]claude.Session, error) {
	s.sessionsMutex.RLock()
	defer s.sessionsMutex.RUnlock()

	// Make a copy to avoid race conditions
	sessions := make([]claude.Session, len(s.sessionsCache))
	copy(sessions, s.sessionsCache)

	return sessions, nil
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	if s.sessionWatcher != nil {
		return s.sessionWatcher.Stop()
	}
	return nil
}
