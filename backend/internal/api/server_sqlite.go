package api

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ksred/claude-session-manager/internal/chat"
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
	chatHandler    *chat.WebSocketChatHandler
	ctx            context.Context
	cancel         context.CancelFunc
	httpServer     *http.Server
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

	// Create database in the Claude directory
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

	// Clean up any stuck import processes from previous runs
	if err := db.CleanupStuckImports(); err != nil {
		logger.WithError(err).Error("Failed to cleanup stuck imports")
		// Don't fail startup for this, just log the error
	}

	// Check for files modified while the server was down
	missedFiles, err := db.CheckForMissedFiles(cfg.Claude.HomeDirectory)
	if err != nil {
		logger.WithError(err).Error("Failed to check for missed files")
		// Don't fail startup for this, just log the error
	} else if missedFiles > 0 {
		logger.WithField("missed_files", missedFiles).Info("Found files modified while server was down - will be processed during startup import")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create chat components if WebSocket is enabled
	var chatHandler *chat.WebSocketChatHandler
	if cfg.Features.EnableWebSocket && wsHub != nil {
		// Create chat repository (Database embeds *sqlx.DB, so we pass db directly)
		chatRepo := chat.NewRepository(db.DB)

		// Create CLI manager
		cliManager := chat.NewCLIManager(chatRepo)

		// Create chat handler
		chatHandler = chat.NewWebSocketChatHandler(cliManager, chatRepo, logger)

		// Set the chat handler on the WebSocket hub
		wsHub.ChatHandler = chatHandler
	}

	server := &SQLiteServer{
		config:         cfg,
		router:         router,
		logger:         logger,
		wsHub:          wsHub,
		db:             db,
		sessionRepo:    sessionRepo,
		sqliteHandlers: NewSQLiteHandlers(sessionRepo, logger),
		chatHandler:    chatHandler,
		ctx:            ctx,
		cancel:         cancel,
	}

	// Start WebSocket hub if enabled
	if server.wsHub != nil {
		// Create and set up the event batcher
		batchInterval := time.Duration(cfg.Features.WebSocketBatchInterval) * time.Second
		if batchInterval < 10*time.Second {
			batchInterval = 10 * time.Second // Minimum 10 seconds
		}
		if batchInterval > 30*time.Second {
			batchInterval = 30 * time.Second // Maximum 30 seconds
		}
		batcher := NewEventBatcher(server.wsHub, logger, batchInterval)
		server.wsHub.SetBatcher(batcher)

		// Start the batcher
		go func() {
			logger.Info("Event batcher goroutine started")
			batcher.Start(ctx)
			logger.Info("Event batcher goroutine exited")
		}()

		// Start the WebSocket hub
		go func() {
			logger.Info("WebSocket hub goroutine started")
			server.wsHub.Run(ctx)
			logger.Info("WebSocket hub goroutine exited")
		}()
	}

	// Create completion channel for import process
	importDone := make(chan struct{})

	// Import existing data (this can take a while) - run in background
	go func() {
		defer close(importDone)
		logger.Info("Import goroutine started")
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
		logger.Info("Import goroutine exited")
	}()

	// Setup file watcher if enabled - start it after import completes
	if cfg.Features.EnableFileWatcher {
		go func() {
			logger.Info("File watcher setup goroutine started")
			// Wait for import to finish before starting file watcher
			select {
			case <-ctx.Done():
				logger.Info("File watcher setup cancelled due to shutdown")
				logger.Info("File watcher setup goroutine exited")
				return
			case <-importDone:
				// Check context again before proceeding
				if ctx.Err() != nil {
					logger.Info("File watcher setup cancelled due to shutdown")
					logger.Info("File watcher setup goroutine exited")
					return
				}
				logger.Info("Starting file watcher after import completion")
				if err := server.setupFileWatcher(); err != nil {
					logger.WithError(err).Error("Failed to setup file watcher")
				}
				logger.Info("File watcher setup goroutine exited")
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
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  time.Duration(s.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.Server.WriteTimeout) * time.Second,
	}

	// ListenAndServe is blocking, so the caller should handle it appropriately
	s.logger.Debug("Calling httpServer.ListenAndServe()...")
	err := s.httpServer.ListenAndServe()
	s.logger.WithError(err).Info("httpServer.ListenAndServe() returned")
	return err
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
			sessions.GET("/:id/tokens/timeline", s.sqliteHandlers.GetSessionTokenTimelineHandler)
			sessions.GET("/:id/activity", s.sqliteHandlers.GetSessionActivityHandler)
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
			projects.GET("/:projectName/tokens/timeline", s.sqliteHandlers.GetProjectTokenTimelineHandler)
			projects.GET("/:projectName/activity", s.sqliteHandlers.GetProjectActivityHandler)
		}

		// Analytics routes
		analytics := v1.Group("/analytics")
		{
			analytics.GET("/tokens/timeline", s.sqliteHandlers.GetTokenTimelineHandler)
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

	// Set up WebSocket update callback if WebSocket is enabled
	if s.wsHub != nil {
		wsAdapter := NewWebSocketUpdateAdapter(s.wsHub, s.sessionRepo, s.logger)
		s.fileWatcher.SetUpdateCallback(wsAdapter)
		s.logger.Info("WebSocket update adapter connected to file watcher")
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
	s.logger.Info("Starting server shutdown sequence")

	// Create shutdown context with timeout from config
	shutdownTimeout := time.Duration(s.config.Server.ShutdownTimeout) * time.Second
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	// Shutdown HTTP server gracefully
	if s.httpServer != nil {
		s.logger.Info("Step 1/5: Shutting down HTTP server...")
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.WithError(err).Error("HTTP server shutdown error")
			// Force close if graceful shutdown fails
			if closeErr := s.httpServer.Close(); closeErr != nil {
				s.logger.WithError(closeErr).Error("HTTP server force close error")
			}
		} else {
			s.logger.Info("HTTP server shutdown completed successfully")
		}
	}

	// Cancel context to stop background processes
	s.logger.Info("Step 2/5: Cancelling background contexts...")
	if s.cancel != nil {
		s.cancel()
		s.logger.Info("Context cancelled - background goroutines should stop")
	}

	// Stop file watcher
	s.logger.Info("Step 3/5: Stopping file watcher...")
	if s.fileWatcher != nil {
		s.fileWatcher.Stop()
		s.logger.Info("File watcher stopped")
	} else {
		s.logger.Info("No file watcher to stop")
	}

	// Stop WebSocket hub (note: WebSocketHub doesn't have a Stop method, just close channels)
	s.logger.Info("Step 4/5: WebSocket hub status...")
	if s.wsHub != nil {
		s.logger.Info("WebSocket hub should stop via context cancellation")
		// Give it a moment to clean up
		time.Sleep(100 * time.Millisecond)
	} else {
		s.logger.Info("No WebSocket hub to stop")
	}

	// Close database
	s.logger.Info("Step 5/5: Closing database...")
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close database")
			return err
		}
		s.logger.Info("Database closed successfully")
	}

	s.logger.Info("Server shutdown sequence completed")
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
