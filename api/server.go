package api

import (
	"context"
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
}

// Config holds server configuration
type Config struct {
	Host            string
	Port            string
	ChainID         string
	NodeURI         string
	JWTSecret       []byte
	CORSOrigins     []string
	RateLimitRPS    int
	MaxConnections  int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
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
	}
}

// NewServer creates a new API server instance
func NewServer(clientCtx client.Context, config *Config) (*Server, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Generate JWT secret if not provided
	if len(config.JWTSecret) == 0 {
		config.JWTSecret = []byte("change-me-in-production-" + time.Now().String())
	}

	// Initialize WebSocket hub
	wsHub := NewWebSocketHub()

	// Initialize services
	authService := NewAuthService(config.JWTSecret)
	tradingService := NewTradingService(clientCtx, wsHub)
	walletService := NewWalletService(clientCtx)
	swapService := NewSwapService(clientCtx)
	poolService := NewPoolService()

	server := &Server{
		clientCtx:      clientCtx,
		config:         config,
		wsHub:          wsHub,
		authService:    authService,
		tradingService: tradingService,
		walletService:  walletService,
		swapService:    swapService,
		poolService:    poolService,
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

	// Global middleware
	s.router.Use(gin.Recovery())
	s.router.Use(LoggerMiddleware())
	s.router.Use(s.CORSMiddleware())
	s.router.Use(RateLimitMiddleware(s.config.RateLimitRPS))

	// Health check endpoint
	s.router.GET("/health", s.healthCheck)

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

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", s.config.Host, s.config.Port),
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		fmt.Printf("Starting PAW API server on %s:%s\n", s.config.Host, s.config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
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

	fmt.Println("Server exited")
	return nil
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
