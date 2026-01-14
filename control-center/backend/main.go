package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/paw-chain/paw/control-center/backend/admin"
	"github.com/paw-chain/paw/control-center/backend/audit"
	"github.com/paw-chain/paw/control-center/backend/auth"
	"github.com/paw-chain/paw/control-center/backend/config"
	"github.com/paw-chain/paw/control-center/backend/integration"
	"github.com/paw-chain/paw/control-center/backend/websocket"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize services
	auditService, err := audit.NewService(cfg.DatabaseURL, cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to initialize audit service: %v", err)
	}
	defer auditService.Close()

	authService := auth.NewService(cfg.JWTSecret, cfg.TokenExpiration, auditService)

	integrationService, err := integration.NewService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize integration service: %v", err)
	}

	// Initialize WebSocket server
	wsServer := websocket.NewServer()
	go wsServer.Start(cfg.WebSocketPort)

	// Create admin API handler
	adminHandler := admin.NewHandler(authService, auditService, integrationService, wsServer, cfg)

	// Set up Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
		})
	})

	// Public endpoints (no auth required)
	public := router.Group("/api")
	{
		public.POST("/auth/login", authService.Login)
		public.POST("/auth/refresh", authService.RefreshToken)

		// Read-only endpoints (reuse from explorer API)
		public.GET("/blocks", integrationService.GetRecentBlocks)
		public.GET("/transactions", integrationService.GetRecentTransactions)
		public.GET("/validators", integrationService.GetValidators)
		public.GET("/proposals", integrationService.GetProposals)
		public.GET("/pools", integrationService.GetPools)
		public.GET("/network/health", integrationService.GetNetworkHealth)
		public.GET("/metrics", integrationService.GetMetrics)
	}

	// Protected admin endpoints (authentication required)
	adminAPI := router.Group("/api/admin")
	adminAPI.Use(authService.AuthMiddleware())
	{
		// Parameter management (Admin role required)
		params := adminAPI.Group("/params")
		params.Use(authService.RoleMiddleware("Admin"))
		{
			params.GET("/:module", adminHandler.GetParams)
			params.POST("/:module", adminHandler.UpdateParams)
			params.GET("/history", adminHandler.GetParamsHistory)
		}

		// Circuit breaker controls (Admin role required)
		cb := adminAPI.Group("/circuit-breaker")
		cb.Use(authService.RoleMiddleware("Admin"))
		{
			cb.POST("/:module/pause", adminHandler.PauseModule)
			cb.POST("/:module/resume", adminHandler.ResumeModule)
			cb.GET("/status", adminHandler.GetCircuitBreakerStatus)
		}

		// Emergency controls (SuperAdmin role + 2FA required)
		emergency := adminAPI.Group("/emergency")
		emergency.Use(authService.RoleMiddleware("SuperAdmin"))
		emergency.Use(authService.TwoFactorMiddleware())
		{
			emergency.POST("/halt-chain", adminHandler.HaltChain)
			emergency.POST("/enable-maintenance", adminHandler.EnableMaintenance)
			emergency.POST("/force-upgrade", adminHandler.ForceUpgrade)
			emergency.POST("/disable-module/:module", adminHandler.DisableModule)
		}

		// Audit log access (Admin role required)
		auditLog := adminAPI.Group("/audit-log")
		auditLog.Use(authService.RoleMiddleware("Admin"))
		{
			auditLog.GET("", adminHandler.GetAuditLog)
			auditLog.GET("/export", adminHandler.ExportAuditLog)
		}

		// Alert management (Admin role required)
		alerts := adminAPI.Group("/alerts")
		alerts.Use(authService.RoleMiddleware("Admin"))
		{
			alerts.GET("", adminHandler.GetAlerts)
			alerts.POST("/:id/acknowledge", adminHandler.AcknowledgeAlert)
			alerts.POST("/:id/resolve", adminHandler.ResolveAlert)
			alerts.GET("/config", adminHandler.GetAlertConfig)
		}

		// User management (SuperAdmin role required)
		users := adminAPI.Group("/users")
		users.Use(authService.RoleMiddleware("SuperAdmin"))
		{
			users.GET("", adminHandler.ListUsers)
			users.POST("", adminHandler.CreateUser)
			users.PUT("/:id", adminHandler.UpdateUser)
			users.DELETE("/:id", adminHandler.DeleteUser)
		}
	}

	// WebSocket endpoint (authentication required)
	router.GET("/ws/updates", authService.WSAuthMiddleware(), wsServer.HandleConnection)

	// Start HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting Control Center API on port %d", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give server 5 seconds to finish existing requests
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
