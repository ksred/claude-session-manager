package api

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ksred/claude-session-manager/internal/config"
	"github.com/ksred/claude-session-manager/internal/database"
	"github.com/sirupsen/logrus"
)

// SQLiteServer represents the API server using SQLite database
type SQLiteServer struct {
	config         *config.Config
	router         *gin.Engine
	logger         *logrus.Logger
	wsHub          *WebSocketHub
	db             *database.Database
	sessionRepo    *database.SessionRepository
	fileWatcher    *database.ClaudeFileWatcher
	sqliteHandlers *SQLiteHandlers
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewSQLiteServer creates a new API server instance using SQLite
func NewSQLiteServer(cfg *config.Config) (*SQLiteServer, error) {
	// Set Gin mode based on debug setting
	if cfg.Features.DebugMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	logger := logrus.StandardLogger()

	// Create database
	dbPath := filepath.Join(cfg.Claude.HomeDirectory, "sessions.db")
	db, err := database.NewDatabase(database.Config{
		DatabasePath: dbPath,
		Logger:       logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create session repository
	sessionRepo := database.NewSessionRepository(db, logger)

	// Create WebSocket hub if enabled
	var wsHub *WebSocketHub
	if cfg.Features.EnableWebSocket {
		wsHub = NewWebSocketHub(logger)
	}

	ctx, cancel := context.WithCancel(context.Background())

	server := &SQLiteServer{
		config:         cfg,
		router:         router,
		logger:         logger,
		wsHub:          wsHub,
		db:             db,
		sessionRepo:    sessionRepo,
		sqliteHandlers: NewSQLiteHandlers(sessionRepo, logger),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Start WebSocket hub if enabled
	if server.wsHub != nil {
		go server.wsHub.Run()
	}

	// Import existing data (this can take a while) - run in background
	go func() {
		logger.Info("Starting background import of existing session data (press Ctrl+C to cancel)")
		if err := server.importExistingData(); err != nil {
			if err == context.Canceled {
				logger.Info("Background import cancelled")
			} else {
				logger.WithError(err).Error("Failed to import existing data")
			}
		} else {
			logger.Info("Background import completed - all historical sessions loaded")
		}
	}()

	// Setup file watcher if enabled - but start it after import to avoid database locks
	if cfg.Features.EnableFileWatcher {
		go func() {
			// Wait for import to finish before starting file watcher
			time.Sleep(2 * time.Minute) // Give import time to complete
			logger.Info("Starting file watcher after import delay")
			if err := server.setupFileWatcher(); err != nil {
				logger.WithError(err).Error("Failed to setup file watcher")
			}
		}()
	}

	// Setup middleware
	server.setupMiddleware()

	// Setup routes
	server.setupRoutes()

	return server, nil
}

// Start starts the server
func (s *SQLiteServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	s.logger.WithFields(logrus.Fields{
		"address": addr,
		"port":    s.config.Server.Port,
		"host":    s.config.Server.Host,
	}).Info("Starting Claude Session Manager API server")

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
func (s *SQLiteServer) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// CORS middleware if enabled
	if s.config.Server.CORS.Enabled {
		s.router.Use(CORSMiddleware(s.config))
	}

	// Logging middleware
	s.router.Use(LoggingMiddleware(s.logger))
}

// setupRoutes configures all API routes using SQLite handlers
func (s *SQLiteServer) setupRoutes() {
	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", s.healthHandler)

		// Session routes using SQLite handlers
		sessions := v1.Group("/sessions")
		{
			sessions.GET("", s.sqliteHandlers.GetSessionsHandler)
			sessions.GET("/:id", s.sqliteHandlers.GetSessionHandler)
			sessions.GET("/active", s.sqliteHandlers.GetActiveSessionsHandler)
			sessions.GET("/recent", s.sqliteHandlers.GetRecentSessionsHandler)
		}

		// Metrics routes using SQLite handlers
		metrics := v1.Group("/metrics")
		{
			metrics.GET("/summary", s.sqliteHandlers.GetMetricsSummaryHandler)
			metrics.GET("/activity", s.sqliteHandlers.GetActivityHandler)
			metrics.GET("/usage", s.sqliteHandlers.GetUsageStatsHandler)
		}

		// Search routes using SQLite handlers
		v1.GET("/search", s.sqliteHandlers.SearchHandler)

		// Files routes
		files := v1.Group("/files")
		{
			files.GET("/recent", s.sqliteHandlers.GetRecentFilesHandler)
		}

		// Projects routes
		projects := v1.Group("/projects")
		{
			projects.GET("/:projectName/files/recent", s.sqliteHandlers.GetProjectRecentFilesHandler)
		}

		// WebSocket endpoint for real-time updates
		v1.GET("/ws", s.websocketHandler)
	}

	// Static files (if needed)
	s.router.Static("/static", "./static")
	
	// Swagger documentation
	// Note: You'll need to update the swagger imports if using this
	// s.router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// healthHandler handles health check requests
func (s *SQLiteServer) healthHandler(c *gin.Context) {
	// Check database health
	if err := s.db.Health(); err != nil {
		s.logger.WithError(err).Error("Database health check failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "Database connection failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "claude-session-manager",
		"database":  "sqlite",
		"timestamp": time.Now().Unix(),
	})
}

// websocketHandler handles WebSocket connections
func (s *SQLiteServer) websocketHandler(c *gin.Context) {
	if s.wsHub == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "WebSocket not enabled",
		})
		return
	}

	// Upgrade HTTP connection to WebSocket
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for now
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.WithError(err).Error("Failed to upgrade to WebSocket")
		return
	}

	// Create new client
	client := &WebSocketClient{
		ID:     fmt.Sprintf("client_%d", time.Now().UnixNano()),
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Hub:    s.wsHub,
		Logger: s.logger,
	}

	// Register client and start pumps
	s.wsHub.register <- client
	go client.writePump()
	go client.readPump()
}

// importExistingData imports existing JSONL files into the database using incremental import
func (s *SQLiteServer) importExistingData() error {
	s.logger.Info("Starting initial data import from JSONL files (press Ctrl+C to cancel)")
	
	// Use incremental importer to avoid re-processing files
	incrementalImporter := database.NewIncrementalImporter(s.ctx, s.sessionRepo, s.db, s.logger)
	if err := incrementalImporter.ImportClaudeDirectory(s.config.Claude.HomeDirectory, false); err != nil {
		if err == context.Canceled {
			s.logger.Info("Import cancelled by user")
			return nil
		}
		return fmt.Errorf("failed to import existing data: %w", err)
	}

	s.logger.Info("Initial data import completed")
	return nil
}

// setupFileWatcher initializes the file system watcher for session files
func (s *SQLiteServer) setupFileWatcher() error {
	var err error
	s.fileWatcher, err = database.NewFileWatcher(
		s.config.Claude.HomeDirectory,
		s.sessionRepo,
		s.logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}

	// Start the file watcher
	if err := s.fileWatcher.Start(s.ctx); err != nil {
		return fmt.Errorf("failed to start file watcher: %w", err)
	}

	s.logger.Info("File watcher started successfully")
	return nil
}

// Stop gracefully stops the server
func (s *SQLiteServer) Stop() error {
	s.logger.Info("Stopping Claude Session Manager server")

	// Cancel context to stop background processes
	if s.cancel != nil {
		s.cancel()
	}

	// Stop file watcher
	if s.fileWatcher != nil {
		s.fileWatcher.Stop()
	}

	// Stop WebSocket hub (note: WebSocketHub doesn't have a Stop method, just close channels)
	if s.wsHub != nil {
		// WebSocket hub will be stopped when context is cancelled
	}

	// Close database
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close database")
			return err
		}
	}

	s.logger.Info("Server stopped successfully")
	return nil
}

// GetDatabase returns the database instance (for testing or admin operations)
func (s *SQLiteServer) GetDatabase() *database.Database {
	return s.db
}

// GetSessionRepository returns the session repository (for testing)
func (s *SQLiteServer) GetSessionRepository() *database.SessionRepository {
	return s.sessionRepo
}