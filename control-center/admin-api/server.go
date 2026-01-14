package adminapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/paw-chain/paw/control-center/admin-api/handlers"
	"github.com/paw-chain/paw/control-center/admin-api/middleware"
	"github.com/paw-chain/paw/control-center/admin-api/types"
)

var (
	adminAPIRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "admin_api_requests_total",
			Help: "Total number of admin API requests",
		},
		[]string{"method", "endpoint", "status", "role"},
	)

	adminAPIRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "admin_api_request_duration_seconds",
			Help:    "Admin API request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "role"},
	)

	adminAPIActiveUsers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "admin_api_active_users",
		Help: "Number of active authenticated users",
	})

	adminAPIAuthFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "admin_api_auth_failures_total",
		Help: "Total number of authentication failures",
	})
)

// Server represents the Admin API server
type Server struct {
	config           *Config
	router           *gin.Engine
	server           *http.Server
	authService      *middleware.AuthService
	rateLimiter      *middleware.RateLimiter
	rbacMiddleware   *middleware.RBACMiddleware
	paramsHandler    *handlers.ParamsHandler
	circuitHandler   *handlers.CircuitBreakerHandler
	emergencyHandler *handlers.EmergencyHandler
	upgradeHandler   *handlers.UpgradeHandler
	auditService     AuditService
	rpcClient        RPCClient
	storage          Storage
}

// Config holds the server configuration
type Config struct {
	Host                     string
	Port                     int
	JWTSecret                string
	TokenDuration            time.Duration
	WriteOperationsPerMinute int
	ReadOperationsPerMinute  int
	EnableMetrics            bool
	EnableCORS               bool
	AllowedOrigins           []string
	RPCEndpoint              string
	DatabaseURL              string
	RedisURL                 string
}

// AuditService interface for audit logging
type AuditService interface {
	LogAction(userID, username, action, resource, ipAddress string, details map[string]interface{}, success bool, err error)
	GetAuditLog(limit, offset int, filters map[string]string) ([]*types.AuditLog, error)
}

// RPCClient interface for blockchain interaction
type RPCClient interface {
	GetModuleParams(ctx context.Context, module string) (map[string]interface{}, error)
	UpdateModuleParams(ctx context.Context, module string, params map[string]interface{}, signer string) (string, error)
	GetLatestBlock(ctx context.Context) (int64, error)
}

// Storage interface for persistent storage
type Storage interface {
	handlers.StorageBackend
	handlers.CircuitBreakerStorage
	handlers.EmergencyStorage
	handlers.UpgradeStorage
}

// NewServer creates a new Admin API server
func NewServer(config *Config, auditService AuditService, rpcClient RPCClient, storage Storage) *Server {
	if config.TokenDuration == 0 {
		config.TokenDuration = 30 * time.Minute
	}
	if config.WriteOperationsPerMinute == 0 {
		config.WriteOperationsPerMinute = 10
	}
	if config.ReadOperationsPerMinute == 0 {
		config.ReadOperationsPerMinute = 100
	}

	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Create services and middleware
	authService := middleware.NewAuthService(config.JWTSecret, config.TokenDuration, auditService)
	rateLimiter := middleware.NewRateLimiter(middleware.RateLimitConfig{
		WriteOperationsPerMinute: config.WriteOperationsPerMinute,
		ReadOperationsPerMinute:  config.ReadOperationsPerMinute,
		BurstMultiplier:          2,
		CleanupInterval:          10 * time.Minute,
	})
	rbacMiddleware := middleware.NewRBACMiddleware(auditService)

	// Create handlers
	paramsHandler := handlers.NewParamsHandler(rpcClient, auditService, storage)
	circuitHandler := handlers.NewCircuitBreakerHandler(rpcClient, auditService, storage)
	emergencyHandler := handlers.NewEmergencyHandler(rpcClient, auditService, storage)
	upgradeHandler := handlers.NewUpgradeHandler(rpcClient, auditService, storage)

	s := &Server{
		config:           config,
		router:           router,
		authService:      authService,
		rateLimiter:      rateLimiter,
		rbacMiddleware:   rbacMiddleware,
		paramsHandler:    paramsHandler,
		circuitHandler:   circuitHandler,
		emergencyHandler: emergencyHandler,
		upgradeHandler:   upgradeHandler,
		auditService:     auditService,
		rpcClient:        rpcClient,
		storage:          storage,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// setupMiddleware configures global middleware
func (s *Server) setupMiddleware() {
	// CORS middleware
	if s.config.EnableCORS {
		s.router.Use(func(c *gin.Context) {
			origin := c.Request.Header.Get("Origin")
			if origin == "" {
				origin = "*"
			}

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range s.config.AllowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
				c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
				c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			}

			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}

			c.Next()
		})
	}

	// Logging middleware
	s.router.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		// Get role for metrics
		role := "anonymous"
		if roleValue, exists := c.Get("role"); exists {
			if r, ok := roleValue.(types.Role); ok {
				role = string(r)
			}
		}

		// Record metrics
		adminAPIRequestsTotal.WithLabelValues(c.Request.Method, path, fmt.Sprintf("%d", status), role).Inc()
		adminAPIRequestDuration.WithLabelValues(c.Request.Method, path, role).Observe(duration.Seconds())
	})

	// Request timeout middleware
	s.router.Use(func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now(),
		})
	})

	// Metrics endpoint (if enabled)
	if s.config.EnableMetrics {
		s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	// API v1 routes
	v1 := s.router.Group("/api/v1/admin")

	// Apply authentication to all admin endpoints
	v1.Use(s.authService.AuthMiddleware())
	v1.Use(s.rateLimiter.RateLimitMiddleware())

	// Parameter Management (Admin role required)
	params := v1.Group("/params")
	params.Use(s.rbacMiddleware.RequireRole(types.RoleAdmin))
	{
		params.GET("/:module", s.paramsHandler.GetParams)
		params.POST("/:module", s.paramsHandler.UpdateParams)
		params.POST("/:module/reset", s.paramsHandler.ResetParams)
		params.GET("/history", s.paramsHandler.GetParamsHistory)
	}

	// Circuit Breaker Controls (Admin role required)
	circuit := v1.Group("/circuit-breaker")
	circuit.Use(s.rbacMiddleware.RequireRole(types.RoleAdmin))
	{
		circuit.POST("/:module/pause", s.circuitHandler.PauseModule)
		circuit.POST("/:module/resume", s.circuitHandler.ResumeModule)
		circuit.GET("/status", s.circuitHandler.GetStatus)
	}

	// Emergency Controls (SuperUser role required)
	emergency := v1.Group("/emergency")
	emergency.Use(s.rbacMiddleware.RequireRole(types.RoleSuperUser))
	{
		emergency.POST("/pause-dex", s.emergencyHandler.PauseDEX)
		emergency.POST("/pause-oracle", s.emergencyHandler.PauseOracle)
		emergency.POST("/pause-compute", s.emergencyHandler.PauseCompute)
		emergency.POST("/resume-all", s.emergencyHandler.ResumeAll)
	}

	// Network Upgrade (Admin role required)
	upgrade := v1.Group("/upgrade")
	upgrade.Use(s.rbacMiddleware.RequireRole(types.RoleAdmin))
	{
		upgrade.POST("/schedule", s.upgradeHandler.ScheduleUpgrade)
		upgrade.POST("/cancel", s.upgradeHandler.CancelUpgrade)
		upgrade.GET("/status", s.upgradeHandler.GetUpgradeStatus)
	}

	// Authentication endpoints (no auth required)
	auth := s.router.Group("/api/v1/auth")
	{
		auth.POST("/login", s.handleLogin)
		auth.POST("/refresh", s.authService.AuthMiddleware(), s.handleRefresh)
		auth.POST("/logout", s.authService.AuthMiddleware(), s.handleLogout)
	}
}

// handleLogin handles user authentication
func (s *Server) handleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		adminAPIAuthFailures.Inc()
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Authenticate user
	user, err := s.authService.Authenticate(req.Username, req.Password)
	if err != nil {
		adminAPIAuthFailures.Inc()
		s.auditService.LogAction("", req.Username, "login_failed", "auth", c.ClientIP(), map[string]interface{}{
			"username": req.Username,
			"error":    err.Error(),
		}, false, err)

		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "authentication_failed",
			"message": "Invalid credentials",
		})
		return
	}

	// Generate token
	token, err := s.authService.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "token_generation_failed",
			"message": "Failed to generate authentication token",
		})
		return
	}

	adminAPIActiveUsers.Inc()

	s.auditService.LogAction(user.ID, user.Username, "login", "auth", c.ClientIP(), map[string]interface{}{
		"username": req.Username,
		"role":     user.Role,
	}, true, nil)

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"user_id":    user.ID,
		"username":   user.Username,
		"role":       user.Role,
		"expires_at": time.Now().Add(s.config.TokenDuration),
	})
}

// handleRefresh handles token refresh
func (s *Server) handleRefresh(c *gin.Context) {
	userID, _ := c.Get("user_id")

	user, err := s.authService.GetUser(getString(userID))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "user_not_found",
			"message": "User not found",
		})
		return
	}

	token, err := s.authService.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "token_generation_failed",
			"message": "Failed to generate authentication token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_at": time.Now().Add(s.config.TokenDuration),
	})
}

// handleLogout handles user logout
func (s *Server) handleLogout(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	adminAPIActiveUsers.Dec()

	s.auditService.LogAction(
		getString(userID),
		getString(username),
		"logout",
		"auth",
		c.ClientIP(),
		nil,
		true,
		nil,
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// Start starts the API server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.server = &http.Server{
		Addr:           addr,
		Handler:        s.router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Stop stops the API server gracefully
func (s *Server) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// Helper function
func getString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
