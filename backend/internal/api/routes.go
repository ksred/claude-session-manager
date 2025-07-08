package api

import (
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/ksred/claude-session-manager/docs"
)

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", s.healthHandler)

		// Session routes
		sessions := v1.Group("/sessions")
		{
			sessions.GET("", s.getSessionsHandler)
			sessions.GET("/:id", s.getSessionHandler)
			sessions.GET("/active", s.getActiveSessionsHandler)
			sessions.GET("/recent", s.getRecentSessionsHandler)
		}

		// Metrics routes
		metrics := v1.Group("/metrics")
		{
			metrics.GET("/summary", s.getMetricsSummaryHandler)
			metrics.GET("/activity", s.getActivityHandler)
			metrics.GET("/usage", s.getUsageStatsHandler)
		}

		// Search routes
		v1.GET("/search", s.searchHandler)

		// WebSocket endpoint for real-time updates
		v1.GET("/ws", s.websocketHandler)
	}

	// Static files (if needed)
	s.router.Static("/static", "./static")
	
	// Swagger documentation
	s.router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}