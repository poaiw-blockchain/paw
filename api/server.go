package api

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Server represents the main API server
type Server struct {
	router         *gin.Engine
	clientCtx      client.Context
	config         *Config
	wsHub          *WebSocketHub
	authService    *AuthService
	tradingService *TradingService
	walletService  *WalletService
	swapService    *SwapService
	poolService    *PoolService
	rateLimiter    *AdvancedRateLimiter
	auditLogger    *AuditLogger
}

// Config holds server configuration
type Config struct {
	Host            string
	Port            string
	ChainID         string
	NodeURI         string
	JWTSecret       []byte
	CORSOrigins     []string
	RateLimitRPS    int // Deprecated: Use RateLimitConfig
	MaxConnections  int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	TLSEnabled      bool
	TLSCertFile     string
	TLSKeyFile      string

	// Advanced rate limiting
	RateLimitConfig *RateLimitConfig

	// Audit logging
	AuditLogDir  string
	AuditEnabled bool
}

// DefaultConfig returns default server configuration
func DefaultConfig() *Config {
	return &Config{
		Host:            "0.0.0.0",
		Port:            "5000",
		ChainID:         "paw-1",
		NodeURI:         "tcp://localhost:26657",
		CORSOrigins:     []string{"http://localhost:3000", "http://localhost:8080"},
		RateLimitRPS:    100,
		MaxConnections:  1000,
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		ShutdownTimeout: 10 * time.Second,
		RateLimitConfig: DefaultRateLimitConfig(),
		AuditLogDir:     "./logs/audit",
		AuditEnabled:    true,
	}
}

// NewServer creates a new API server instance
func NewServer(clientCtx client.Context, config *Config) (*Server, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Generate JWT secret if not provided (using cryptographically secure random)
	if len(config.JWTSecret) == 0 {
		// Generate 32 bytes (256 bits) of cryptographically secure random data
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return nil, fmt.Errorf("failed to generate JWT secret: %w", err)
		}
		config.JWTSecret = secret

		// Log warning that secret should be configured explicitly
		fmt.Printf("WARNING: JWT secret generated randomly. For production, set explicit JWT secret via environment variable or config file.\n")
		fmt.Printf("Generated JWT secret (hex): %s\n", hex.EncodeToString(secret))
		fmt.Println("Save this secret and configure it explicitly to maintain session continuity across restarts.")
	}

	// Initialize audit logger
	auditLogger, err := NewAuditLogger(config.AuditLogDir, config.AuditEnabled)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize audit logger: %w", err)
	}

	// Initialize WebSocket hub
	wsHub := NewWebSocketHub()

	// Initialize services
	authService := NewAuthService(config.JWTSecret)
	tradingService := NewTradingService(clientCtx, wsHub)
	walletService := NewWalletService(clientCtx)
	swapService := NewSwapService(clientCtx)
	poolService := NewPoolService()

	// Initialize advanced rate limiter
	var rateLimiter *AdvancedRateLimiter
	if config.RateLimitConfig != nil && config.RateLimitConfig.Enabled {
		rateLimiter, err = NewAdvancedRateLimiter(config.RateLimitConfig, auditLogger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize rate limiter: %w", err)
		}
	}

	server := &Server{
		clientCtx:      clientCtx,
		config:         config,
		wsHub:          wsHub,
		authService:    authService,
		tradingService: tradingService,
		walletService:  walletService,
		swapService:    swapService,
		poolService:    poolService,
		rateLimiter:    rateLimiter,
		auditLogger:    auditLogger,
	}

	// Setup router
	server.setupRouter()

	return server, nil
}

// setupRouter configures the Gin router with all routes and middleware
func (s *Server) setupRouter() {
	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	s.router = gin.New()

	// Global middleware - ORDER MATTERS!
	// 1. Recovery (must be first to catch panics)
	s.router.Use(gin.Recovery())

	// 2. Security headers (set early)
	s.router.Use(SecurityHeadersMiddleware())

	// 3. Request size limiting (prevent DOS)
	s.router.Use(RequestSizeLimitMiddleware(MaxRequestSize))

	// 4. HTTPS redirect (if configured)
	if s.config.TLSEnabled {
		s.router.Use(HTTPSRedirectMiddleware())
	}

	// 5. Request ID (for tracing)
	s.router.Use(RequestIDMiddleware())

	// 6. Logging
	s.router.Use(LoggerMiddleware())

	// 7. CORS (before auth)
	s.router.Use(s.CORSMiddleware())

	// 8. Rate limiting (before expensive operations)
	if s.rateLimiter != nil {
		s.router.Use(AdvancedRateLimitMiddleware(s.rateLimiter))
	} else {
		s.router.Use(RateLimitMiddleware(s.config.RateLimitRPS))
	}

	// 9. Audit logging (if enabled)
	if s.auditLogger != nil && s.config.AuditEnabled {
		s.router.Use(AuditMiddleware(s.auditLogger))
	}

	// 10. Timeout (prevent hanging requests)
	s.router.Use(TimeoutMiddleware(30 * time.Second))

	// Health check endpoint (no auth required)
	s.router.GET("/health", s.healthCheck)

	// Rate limiter stats endpoint (for monitoring)
	if s.rateLimiter != nil {
		s.router.GET("/rate-limit/stats", s.handleRateLimitStats)
	}

	// Register all routes
	s.registerRoutes()
}

// healthCheck returns server health status
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Start WebSocket hub
	go s.wsHub.Run()

	// Create HTTP server with security configurations
	srv := &http.Server{
		Addr:           fmt.Sprintf("%s:%s", s.config.Host, s.config.Port),
		Handler:        s.router,
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Configure TLS if enabled
	if s.config.TLSEnabled {
		// Configure secure TLS settings
		srv.TLSConfig = &tls.Config{
			MinVersion:               tls.VersionTLS13, // Only TLS 1.3
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_AES_128_GCM_SHA256,
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
			},
		}
	}

	// Start server in a goroutine
	go func() {
		if s.config.TLSEnabled {
			fmt.Printf("Starting PAW API server (TLS) on %s:%s\n", s.config.Host, s.config.Port)
			if err := srv.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile); err != nil && err != http.ErrServerClosed {
				fmt.Printf("Server error: %v\n", err)
			}
		} else {
			fmt.Printf("WARNING: Starting PAW API server (HTTP - unencrypted) on %s:%s\n", s.config.Host, s.config.Port)
			fmt.Println("For production, enable TLS by setting TLSEnabled=true and providing certificate files")
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Printf("Server error: %v\n", err)
			}
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	// Close WebSocket hub
	s.wsHub.Close()

	// Close rate limiter
	if s.rateLimiter != nil {
		s.rateLimiter.Close()
	}

	// Close audit logger
	if s.auditLogger != nil {
		s.auditLogger.Close()
	}

	fmt.Println("Server exited")
	return nil
}

// handleRateLimitStats returns rate limiter statistics
func (s *Server) handleRateLimitStats(c *gin.Context) {
	if s.rateLimiter == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Rate limiter not enabled",
		})
		return
	}

	stats := s.rateLimiter.GetStats()
	c.JSON(http.StatusOK, stats)
}

// BroadcastTx broadcasts a transaction to the blockchain
func (s *Server) BroadcastTx(txBuilder client.TxBuilder, fromAddress sdk.AccAddress) (*sdk.TxResponse, error) {
	txBytes, err := s.clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, fmt.Errorf("failed to encode tx: %w", err)
	}

	// Broadcast transaction
	res, err := s.clientCtx.BroadcastTx(txBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast tx: %w", err)
	}

	return res, nil
}

// QueryWithData performs a query with data
func (s *Server) QueryWithData(path string, data []byte) ([]byte, error) {
	res, _, err := s.clientCtx.QueryWithData(path, data)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	return res, nil
}

// GetClientContext returns the client context
func (s *Server) GetClientContext() client.Context {
	return s.clientCtx
}
