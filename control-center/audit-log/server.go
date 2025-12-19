package auditlog

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/paw-chain/paw/control-center/audit-log/api"
	"github.com/paw-chain/paw/control-center/audit-log/middleware"
	"github.com/paw-chain/paw/control-center/audit-log/storage"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// Server represents the audit log server
type Server struct {
	storage    *storage.PostgresStorage
	middleware *middleware.AuditLogger
	handler    *api.Handler
	httpServer *http.Server
}

// Config holds configuration for the audit log server
type Config struct {
	DatabaseURL string
	HTTPPort    int
	EnableCORS  bool
}

// NewServer creates a new audit log server
func NewServer(cfg Config) (*Server, error) {
	// Initialize storage
	stor, err := storage.NewPostgresStorage(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Create middleware
	mw := middleware.NewAuditLogger(stor)

	// Create API handler
	handler := api.NewHandler(stor)

	// Create HTTP server
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Apply CORS if enabled
	var httpHandler http.Handler = router
	if cfg.EnableCORS {
		c := cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
		})
		httpHandler = c.Handler(router)
	}

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      httpHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		storage:    stor,
		middleware: mw,
		handler:    handler,
		httpServer: httpServer,
	}, nil
}

// Start starts the audit log server
func (s *Server) Start() error {
	log.Printf("Starting audit log server on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("Stopping audit log server...")

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	// Close storage
	if err := s.storage.Close(); err != nil {
		return fmt.Errorf("failed to close storage: %w", err)
	}

	log.Println("Audit log server stopped")
	return nil
}

// GetMiddleware returns the audit logging middleware
func (s *Server) GetMiddleware() *middleware.AuditLogger {
	return s.middleware
}

// GetStorage returns the storage instance
func (s *Server) GetStorage() *storage.PostgresStorage {
	return s.storage
}
