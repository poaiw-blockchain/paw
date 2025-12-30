package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/paw/control-center/alerting"
	"github.com/paw/control-center/alerting/api"
	"github.com/paw/control-center/alerting/channels"
	"github.com/paw/control-center/alerting/engine"
	"github.com/paw/control-center/alerting/storage"
)

// Server represents the alerting server
type Server struct {
	config          *alerting.Config
	storage         *storage.PostgresStorage
	rulesEngine     *engine.RulesEngine
	evaluator       *engine.Evaluator
	notificationMgr *channels.Manager
	metricsProvider engine.MetricsProvider
	httpServer      *http.Server
}

// NewServer creates a new alerting server
func NewServer(config *alerting.Config, metricsProvider engine.MetricsProvider) (*Server, error) {
	// Initialize storage
	storage, err := storage.NewPostgresStorage(config.DatabaseURL, config.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize evaluator
	evaluator := engine.NewEvaluator(metricsProvider)

	// Initialize rules engine
	rulesEngine := engine.NewRulesEngine(storage, evaluator, config)

	// Initialize notification manager
	notificationMgr := channels.NewManager(storage, config)

	// Register alert handler to send notifications
	rulesEngine.RegisterAlertHandler(func(alert *alerting.Alert) error {
		return notificationMgr.SendAlert(alert)
	})

	return &Server{
		config:          config,
		storage:         storage,
		rulesEngine:     rulesEngine,
		evaluator:       evaluator,
		notificationMgr: notificationMgr,
		metricsProvider: metricsProvider,
	}, nil
}

// Start starts the alerting server
func (s *Server) Start() error {
	log.Println("Starting PAW Alert Manager...")

	// Start rules engine
	if err := s.rulesEngine.Start(); err != nil {
		return fmt.Errorf("failed to start rules engine: %w", err)
	}

	// Setup HTTP server
	if s.config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Configure appropriately for production
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check endpoint
	router.GET("/health", s.healthCheck)
	router.GET("/ready", s.readyCheck)

	// API routes
	handler := api.NewHandler(s.storage, s.rulesEngine, s.notificationMgr, s.config)
	handler.RegisterRoutes(router)

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.HTTPPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Alert Manager API listening on port %d", s.config.HTTPPort)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	s.waitForShutdown()

	return nil
}

// Stop stops the alerting server
func (s *Server) Stop() error {
	log.Println("Stopping PAW Alert Manager...")

	// Stop rules engine
	s.rulesEngine.Stop()

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down HTTP server: %v", err)
	}

	// Close storage connections
	if err := s.storage.Close(); err != nil {
		log.Printf("Error closing storage: %v", err)
	}

	log.Println("PAW Alert Manager stopped")
	return nil
}

// waitForShutdown waits for interrupt signal and performs graceful shutdown
func (s *Server) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("Shutdown signal received")

	s.Stop()
}

// healthCheck handles health check requests
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "paw-alert-manager",
		"version": "1.0.0",
	})
}

// readyCheck handles readiness check requests
func (s *Server) readyCheck(c *gin.Context) {
	// Check if storage is accessible
	_, err := s.storage.GetAlertStats()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "storage not accessible",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ready",
		"service": "paw-alert-manager",
	})
}
