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

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"status/pkg/api"
	"status/pkg/config"
	"status/pkg/health"
	"status/pkg/incidents"
	"status/pkg/metrics"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize components
	healthMonitor := health.NewMonitor(cfg)
	incidentManager := incidents.NewManager(cfg)
	metricsCollector := metrics.NewCollector(cfg)

	// Start background monitoring
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go healthMonitor.Start(ctx)
	go metricsCollector.Start(ctx)
	go incidentManager.Start(ctx)

	// Setup HTTP router
	router := mux.NewRouter()
	apiHandler := api.NewHandler(healthMonitor, incidentManager, metricsCollector)

	// API routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiHandler.RegisterRoutes(apiRouter)

	// Static file serving
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./frontend")))

	// CORS configuration
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	// HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Printf("Starting PAW Status Server on port %d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
