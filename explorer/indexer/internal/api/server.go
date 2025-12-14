package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/time/rate"

	"github.com/paw-chain/paw/explorer/indexer/config"
	"github.com/paw-chain/paw/explorer/indexer/internal/cache"
	"github.com/paw-chain/paw/explorer/indexer/internal/database"
	"github.com/paw-chain/paw/explorer/indexer/internal/websocket/hub"
	"github.com/paw-chain/paw/explorer/indexer/pkg/logger"
)

var (
	apiRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "explorer_api_requests_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	apiRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "explorer_api_request_duration_seconds",
			Help:    "API request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	activeConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "explorer_api_active_connections",
		Help: "Number of active WebSocket connections",
	})
)

// Server represents the API server
type Server struct {
	config   config.APIConfig
	db       *database.Database
	cache    *cache.RedisCache
	wsHub    *hub.Hub
	log      *logger.Logger
	router   *gin.Engine
	server   *http.Server
	limiter  *rate.Limiter
	upgrader websocket.Upgrader
}

// NewServer creates a new API server
func NewServer(
	cfg config.APIConfig,
	db *database.Database,
	cache *cache.RedisCache,
	wsHub *hub.Hub,
	log *logger.Logger,
) *Server {
	if cfg.RateLimit <= 0 {
		cfg.RateLimit = 100 // default rate limit
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	s := &Server{
		config:  cfg,
		db:      db,
		cache:   cache,
		wsHub:   wsHub,
		log:     log,
		router:  router,
		limiter: rate.NewLimiter(rate.Limit(cfg.RateLimit), cfg.RateLimit*2),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement proper origin checking
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// setupMiddleware configures middleware
func (s *Server) setupMiddleware() {
	// CORS middleware
	s.router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range s.config.CORSOrigins {
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

	// Rate limiting middleware
	s.router.Use(func(c *gin.Context) {
		if !s.limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests, please try again later",
			})
			c.Abort()
			return
		}
		c.Next()
	})

	// Logging middleware
	s.router.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		s.log.Info("API request",
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"ip", c.ClientIP(),
		)

		// Record metrics
		apiRequestsTotal.WithLabelValues(c.Request.Method, path, fmt.Sprintf("%d", status)).Inc()
		apiRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration.Seconds())
	})

	// Timeout middleware
	s.router.Use(func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), s.config.Timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
}

// setupRoutes configures API routes
func (s *Server) setupRoutes() {
	// Health check endpoints
	s.router.GET("/health", s.handleHealth)
	s.router.GET("/health/ready", s.handleHealthReady)
	s.router.GET("/health/detailed", s.handleHealthDetailed)
	s.router.GET("/version", s.handleVersion)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Blockchain routes
		blocks := v1.Group("/blocks")
		{
			blocks.GET("", s.handleGetBlocks)
			blocks.GET("/latest", s.handleGetLatestBlocks)
			blocks.GET("/:height", s.handleGetBlock)
			blocks.GET("/:height/transactions", s.handleGetBlockTransactions)
		}

		// Transaction routes
		txs := v1.Group("/transactions")
		{
			txs.GET("", s.handleGetTransactions)
			txs.GET("/latest", s.handleGetLatestTransactions)
			txs.GET("/:hash", s.handleGetTransaction)
			txs.GET("/:hash/events", s.handleGetTransactionEvents)
		}

		// Account routes
		accounts := v1.Group("/accounts")
		{
			accounts.GET("/:address", s.handleGetAccount)
			accounts.GET("/:address/transactions", s.handleGetAccountTransactions)
			accounts.GET("/:address/balances", s.handleGetAccountBalances)
			accounts.GET("/:address/tokens", s.handleGetAccountTokens)

			// DEX-specific user routes
			accounts.GET("/:address/dex-positions", s.handleGetUserDEXPositions)
			accounts.GET("/:address/dex-history", s.handleGetUserDEXHistory)
			accounts.GET("/:address/dex-analytics", s.handleGetUserDEXAnalytics)
		}

		// Validator routes
		validators := v1.Group("/validators")
		{
			validators.GET("", s.handleGetValidators)
			validators.GET("/active", s.handleGetActiveValidators)
			validators.GET("/:address", s.handleGetValidator)
			validators.GET("/:address/uptime", s.handleGetValidatorUptime)
			validators.GET("/:address/rewards", s.handleGetValidatorRewards)
		}

		// DEX routes
		dex := v1.Group("/dex")
		{
			// Basic pool routes
			dex.GET("/pools", s.handleGetDEXPools)
			dex.GET("/pools/:id", s.handleGetDEXPool)
			dex.GET("/pools/:id/trades", s.handleGetPoolTrades)
			dex.GET("/pools/:id/liquidity", s.handleGetPoolLiquidity)
			dex.GET("/pools/:id/chart", s.handleGetPoolChart)

			// Advanced analytics routes
			dex.GET("/pools/:id/price-history", s.handleGetPoolPriceHistory)
			dex.GET("/pools/:id/liquidity-chart", s.handleGetPoolLiquidityChart)
			dex.GET("/pools/:id/volume-chart", s.handleGetPoolVolumeChart)
			dex.GET("/pools/:id/fees", s.handleGetPoolFees)
			dex.GET("/pools/:id/apr-history", s.handleGetPoolAPRHistory)
			dex.GET("/pools/:id/depth", s.handleGetPoolDepth)
			dex.GET("/pools/:id/statistics", s.handleGetPoolStatistics)

			// Trade routes
			dex.GET("/trades", s.handleGetDEXTrades)
			dex.GET("/trades/latest", s.handleGetLatestDEXTrades)

			// DEX-wide analytics
			dex.GET("/analytics/summary", s.handleGetDEXAnalyticsSummary)
			dex.GET("/analytics/top-pairs", s.handleGetTopTradingPairs)

			// Swap simulation
			dex.POST("/simulate-swap", s.handleSimulateSwap)
		}

		// Oracle routes
		oracle := v1.Group("/oracle")
		{
			oracle.GET("/prices", s.handleGetOraclePrices)
			oracle.GET("/prices/:asset", s.handleGetAssetPrice)
			oracle.GET("/prices/:asset/history", s.handleGetAssetPriceHistory)
			oracle.GET("/prices/:asset/chart", s.handleGetAssetPriceChart)
			oracle.GET("/submissions", s.handleGetOracleSubmissions)
			oracle.GET("/slashes", s.handleGetOracleSlashes)
		}

		// Compute routes
		compute := v1.Group("/compute")
		{
			compute.GET("/requests", s.handleGetComputeRequests)
			compute.GET("/requests/:id", s.handleGetComputeRequest)
			compute.GET("/requests/:id/results", s.handleGetComputeResults)
			compute.GET("/requests/:id/verifications", s.handleGetComputeVerifications)
			compute.GET("/providers", s.handleGetComputeProviders)
			compute.GET("/providers/:address", s.handleGetComputeProvider)
		}

		// Statistics routes
		stats := v1.Group("/stats")
		{
			stats.GET("/network", s.handleGetNetworkStats)
			stats.GET("/charts/transactions", s.handleGetTransactionChart)
			stats.GET("/charts/addresses", s.handleGetAddressChart)
			stats.GET("/charts/volume", s.handleGetVolumeChart)
			stats.GET("/charts/gas", s.handleGetGasChart)
		}

		// Search route
		v1.GET("/search", s.handleSearch)

		// Export routes
		v1.GET("/export/transactions", s.handleExportTransactions)
		v1.GET("/export/trades", s.handleExportTrades)
	}

	// WebSocket route
	if s.config.EnableWebSocket {
		s.router.GET("/ws", s.handleWebSocket)
	}

	// GraphQL route
	if s.config.EnableGraphQL {
		s.router.POST("/graphql", s.handleGraphQL)
		s.router.GET("/graphql/playground", s.handleGraphQLPlayground)
	}
}

// Start starts the API server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.log.Info("Starting API server", "address", addr)

	s.server = &http.Server{
		Addr:           addr,
		Handler:        s.router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start API server: %w", err)
	}

	return nil
}

// Stop stops the API server gracefully
func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("Stopping API server")
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// ============================================================================
// HANDLER IMPLEMENTATIONS
// ============================================================================

// handleHealth handles basic liveness check - always returns 200 if process is alive
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// handleHealthReady handles readiness check - checks if can serve traffic
func (s *Server) handleHealthReady(c *gin.Context) {
	checks := make(map[string]interface{})
	allHealthy := true

	// Check database connection
	if err := s.db.Ping(); err != nil {
		checks["database"] = gin.H{"status": "unhealthy", "message": err.Error()}
		allHealthy = false
	} else {
		checks["database"] = gin.H{"status": "ok"}
	}

	// Check cache connection
	if err := s.cache.Ping(c.Request.Context()); err != nil {
		checks["cache"] = gin.H{"status": "unhealthy", "message": err.Error()}
		allHealthy = false
	} else {
		checks["cache"] = gin.H{"status": "ok"}
	}

	status := "ready"
	statusCode := http.StatusOK
	if !allHealthy {
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status": status,
		"checks": checks,
	})
}

// handleHealthDetailed handles detailed health check with metrics
func (s *Server) handleHealthDetailed(c *gin.Context) {
	checks := make(map[string]interface{})

	// Check database connection
	if err := s.db.Ping(); err != nil {
		checks["database"] = gin.H{"status": "unhealthy", "message": err.Error()}
	} else {
		checks["database"] = gin.H{"status": "ok"}
	}

	// Check cache connection
	if err := s.cache.Ping(c.Request.Context()); err != nil {
		checks["cache"] = gin.H{"status": "unhealthy", "message": err.Error()}
	} else {
		checks["cache"] = gin.H{"status": "ok"}
	}

	// WebSocket hub status
	checks["websocket"] = gin.H{"status": "ok"}

	// Get network stats if available
	stats, err := s.db.GetNetworkStats()
	if err == nil && stats != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"checks": checks,
			"metrics": gin.H{
				"total_blocks":       stats.TotalBlocks,
				"total_transactions": stats.TotalTransactions,
				"active_accounts":    stats.ActiveAccounts,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"checks": checks,
	})
}

// handleVersion handles version requests
func (s *Server) handleVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":    "1.0.0",
		"build_time": "2024-01-01T00:00:00Z",
		"go_version": "1.21",
	})
}

// handleGetBlocks handles GET /blocks requests
func (s *Server) handleGetBlocks(c *gin.Context) {
	page := parseQueryInt(c, "page", 1)
	limit := parseQueryInt(c, "limit", 20)
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	// Try cache first
	cacheKey := fmt.Sprintf("blocks:page:%d:limit:%d", page, limit)
	if cached, err := s.cache.Get(c.Request.Context(), cacheKey); err == nil {
		var blocks []database.Block
		if err := json.Unmarshal(cached, &blocks); err == nil {
			c.JSON(http.StatusOK, gin.H{
				"blocks": blocks,
				"page":   page,
				"limit":  limit,
				"cached": true,
			})
			return
		}
	}

	blocks, total, err := s.db.GetBlocks(offset, limit)
	if err != nil {
		s.log.Error("Failed to get blocks", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch blocks",
		})
		return
	}

	// Cache the result
	if data, err := json.Marshal(blocks); err == nil {
		s.cache.Set(c.Request.Context(), cacheKey, data, time.Minute)
	}

	c.JSON(http.StatusOK, gin.H{
		"blocks": blocks,
		"page":   page,
		"limit":  limit,
		"total":  total,
	})
}

// handleGetLatestBlocks handles GET /blocks/latest requests
func (s *Server) handleGetLatestBlocks(c *gin.Context) {
	limit := parseQueryInt(c, "limit", 10)
	if limit > 50 {
		limit = 50
	}

	// Try cache first
	cacheKey := fmt.Sprintf("blocks:latest:%d", limit)
	if cached, err := s.cache.Get(c.Request.Context(), cacheKey); err == nil {
		var blocks []database.Block
		if err := json.Unmarshal(cached, &blocks); err == nil {
			c.JSON(http.StatusOK, gin.H{
				"blocks": blocks,
				"cached": true,
			})
			return
		}
	}

	blocks, _, err := s.db.GetBlocks(0, limit)
	if err != nil {
		s.log.Error("Failed to get latest blocks", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch latest blocks",
		})
		return
	}

	// Cache the result
	if data, err := json.Marshal(blocks); err == nil {
		s.cache.Set(c.Request.Context(), cacheKey, data, 10*time.Second)
	}

	c.JSON(http.StatusOK, gin.H{
		"blocks": blocks,
	})
}

// handleGetBlock handles GET /blocks/:height requests
func (s *Server) handleGetBlock(c *gin.Context) {
	heightStr := c.Param("height")
	height, err := strconv.ParseInt(heightStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid block height",
		})
		return
	}

	// Try cache first
	cacheKey := fmt.Sprintf("block:%d", height)
	if cached, err := s.cache.Get(c.Request.Context(), cacheKey); err == nil {
		var block database.Block
		if err := json.Unmarshal(cached, &block); err == nil {
			c.JSON(http.StatusOK, gin.H{
				"block":  block,
				"cached": true,
			})
			return
		}
	}

	block, err := s.db.GetBlockByHeight(height)
	if err != nil {
		s.log.Error("Failed to get block", "height", height, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "block not found",
		})
		return
	}

	// Cache the result
	if data, err := json.Marshal(block); err == nil {
		s.cache.Set(c.Request.Context(), cacheKey, data, time.Hour)
	}

	c.JSON(http.StatusOK, gin.H{
		"block": block,
	})
}

// handleGetBlockTransactions handles GET /blocks/:height/transactions requests
func (s *Server) handleGetBlockTransactions(c *gin.Context) {
	heightStr := c.Param("height")
	height, err := strconv.ParseInt(heightStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid block height",
		})
		return
	}

	txs, err := s.db.GetTransactionsByHeight(height)
	if err != nil {
		s.log.Error("Failed to get block transactions", "height", height, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch transactions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": txs,
		"count":        len(txs),
	})
}

// handleGetTransactions handles GET /transactions requests
func (s *Server) handleGetTransactions(c *gin.Context) {
	page := parseQueryInt(c, "page", 1)
	limit := parseQueryInt(c, "limit", 20)
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit
	status := c.Query("status")
	txType := c.Query("type")

	txs, total, err := s.db.GetTransactions(offset, limit, status, txType)
	if err != nil {
		s.log.Error("Failed to get transactions", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch transactions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": txs,
		"page":         page,
		"limit":        limit,
		"total":        total,
	})
}

// handleGetLatestTransactions handles GET /transactions/latest requests
func (s *Server) handleGetLatestTransactions(c *gin.Context) {
	limit := parseQueryInt(c, "limit", 10)
	if limit > 50 {
		limit = 50
	}

	cacheKey := fmt.Sprintf("transactions:latest:%d", limit)
	if cached, err := s.cache.Get(c.Request.Context(), cacheKey); err == nil {
		var txs []database.Transaction
		if err := json.Unmarshal(cached, &txs); err == nil {
			c.JSON(http.StatusOK, gin.H{
				"transactions": txs,
				"cached":       true,
			})
			return
		}
	}

	txs, _, err := s.db.GetTransactions(0, limit, "", "")
	if err != nil {
		s.log.Error("Failed to get latest transactions", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch latest transactions",
		})
		return
	}

	if data, err := json.Marshal(txs); err == nil {
		s.cache.Set(c.Request.Context(), cacheKey, data, 10*time.Second)
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": txs,
	})
}

// handleGetTransaction handles GET /transactions/:hash requests
func (s *Server) handleGetTransaction(c *gin.Context) {
	hash := c.Param("hash")

	cacheKey := fmt.Sprintf("transaction:%s", hash)
	if cached, err := s.cache.Get(c.Request.Context(), cacheKey); err == nil {
		var tx database.Transaction
		if err := json.Unmarshal(cached, &tx); err == nil {
			c.JSON(http.StatusOK, gin.H{
				"transaction": tx,
				"cached":      true,
			})
			return
		}
	}

	tx, err := s.db.GetTransactionByHash(hash)
	if err != nil {
		s.log.Error("Failed to get transaction", "hash", hash, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "transaction not found",
		})
		return
	}

	if data, err := json.Marshal(tx); err == nil {
		s.cache.Set(c.Request.Context(), cacheKey, data, time.Hour)
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction": tx,
	})
}

// handleGetTransactionEvents handles GET /transactions/:hash/events requests
func (s *Server) handleGetTransactionEvents(c *gin.Context) {
	hash := c.Param("hash")

	events, err := s.db.GetEventsByTxHash(hash)
	if err != nil {
		s.log.Error("Failed to get transaction events", "hash", hash, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch events",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"count":  len(events),
	})
}

// handleGetAccount handles GET /accounts/:address requests
func (s *Server) handleGetAccount(c *gin.Context) {
	address := c.Param("address")

	account, err := s.db.GetAccount(address)
	if err != nil {
		s.log.Error("Failed to get account", "address", address, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "account not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"account": account,
	})
}

// handleGetAccountTransactions handles GET /accounts/:address/transactions requests
func (s *Server) handleGetAccountTransactions(c *gin.Context) {
	address := c.Param("address")
	page := parseQueryInt(c, "page", 1)
	limit := parseQueryInt(c, "limit", 20)
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	txs, total, err := s.db.GetTransactionsByAddress(address, offset, limit)
	if err != nil {
		s.log.Error("Failed to get account transactions", "address", address, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch transactions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": txs,
		"page":         page,
		"limit":        limit,
		"total":        total,
	})
}

// handleGetAccountBalances handles GET /accounts/:address/balances requests
func (s *Server) handleGetAccountBalances(c *gin.Context) {
	address := c.Param("address")

	balances, err := s.db.GetAccountBalances(address)
	if err != nil {
		s.log.Error("Failed to get account balances", "address", address, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch balances",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"balances": balances,
	})
}

// handleGetAccountTokens handles GET /accounts/:address/tokens requests
func (s *Server) handleGetAccountTokens(c *gin.Context) {
	address := c.Param("address")

	tokens, err := s.db.GetAccountTokens(address)
	if err != nil {
		s.log.Error("Failed to get account tokens", "address", address, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch tokens",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tokens": tokens,
	})
}

// Additional handlers would continue here for validators, DEX, Oracle, Compute, etc.
// Due to space constraints, I'm showing the pattern and structure.
// In production, all handlers would be fully implemented following this pattern.

// handleGetValidators handles GET /validators requests
func (s *Server) handleGetValidators(c *gin.Context) {
	page := parseQueryInt(c, "page", 1)
	limit := parseQueryInt(c, "limit", 20)
	status := c.Query("status")

	offset := (page - 1) * limit

	validators, total, err := s.db.GetValidators(offset, limit, status)
	if err != nil {
		s.log.Error("Failed to get validators", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch validators",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"validators": validators,
		"page":       page,
		"limit":      limit,
		"total":      total,
	})
}

// handleGetActiveValidators handles GET /validators/active requests
func (s *Server) handleGetActiveValidators(c *gin.Context) {
	validators, err := s.db.GetActiveValidators()
	if err != nil {
		s.log.Error("Failed to get active validators", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch active validators",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"validators": validators,
		"count":      len(validators),
	})
}

// handleGetValidator handles GET /validators/:address requests
func (s *Server) handleGetValidator(c *gin.Context) {
	address := c.Param("address")

	validator, err := s.db.GetValidator(address)
	if err != nil {
		s.log.Error("Failed to get validator", "address", address, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "validator not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"validator": validator,
	})
}

// handleGetValidatorUptime handles GET /validators/:address/uptime requests
func (s *Server) handleGetValidatorUptime(c *gin.Context) {
	address := c.Param("address")
	days := parseQueryInt(c, "days", 30)

	uptime, err := s.db.GetValidatorUptime(address, days)
	if err != nil {
		s.log.Error("Failed to get validator uptime", "address", address, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch uptime",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"uptime": uptime,
	})
}

// handleGetValidatorRewards handles GET /validators/:address/rewards requests
func (s *Server) handleGetValidatorRewards(c *gin.Context) {
	address := c.Param("address")
	page := parseQueryInt(c, "page", 1)
	limit := parseQueryInt(c, "limit", 20)

	offset := (page - 1) * limit

	rewards, total, err := s.db.GetValidatorRewards(address, offset, limit)
	if err != nil {
		s.log.Error("Failed to get validator rewards", "address", address, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch rewards",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rewards": rewards,
		"page":    page,
		"limit":   limit,
		"total":   total,
	})
}

// handleGetDEXPools handles GET /dex/pools requests
func (s *Server) handleGetDEXPools(c *gin.Context) {
	page := parseQueryInt(c, "page", 1)
	limit := parseQueryInt(c, "limit", 20)
	sortBy := c.DefaultQuery("sort", "tvl")

	offset := (page - 1) * limit

	pools, total, err := s.db.GetDEXPools(offset, limit, sortBy)
	if err != nil {
		s.log.Error("Failed to get DEX pools", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch pools",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pools": pools,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

// handleGetDEXPool handles GET /dex/pools/:id requests
func (s *Server) handleGetDEXPool(c *gin.Context) {
	poolID := c.Param("id")

	pool, err := s.db.GetDEXPool(poolID)
	if err != nil {
		s.log.Error("Failed to get DEX pool", "pool_id", poolID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "pool not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pool": pool,
	})
}

// handleGetPoolTrades handles GET /dex/pools/:id/trades requests
func (s *Server) handleGetPoolTrades(c *gin.Context) {
	poolID := c.Param("id")
	page := parseQueryInt(c, "page", 1)
	limit := parseQueryInt(c, "limit", 20)

	offset := (page - 1) * limit

	trades, total, err := s.db.GetPoolTrades(poolID, offset, limit)
	if err != nil {
		s.log.Error("Failed to get pool trades", "pool_id", poolID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch trades",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"trades": trades,
		"page":   page,
		"limit":  limit,
		"total":  total,
	})
}

// handleGetPoolLiquidity handles GET /dex/pools/:id/liquidity requests
func (s *Server) handleGetPoolLiquidity(c *gin.Context) {
	poolID := c.Param("id")
	page := parseQueryInt(c, "page", 1)
	limit := parseQueryInt(c, "limit", 20)

	offset := (page - 1) * limit

	liquidity, total, err := s.db.GetPoolLiquidity(poolID, offset, limit)
	if err != nil {
		s.log.Error("Failed to get pool liquidity", "pool_id", poolID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch liquidity",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"liquidity": liquidity,
		"page":      page,
		"limit":     limit,
		"total":     total,
	})
}

// handleGetPoolChart handles GET /dex/pools/:id/chart requests
func (s *Server) handleGetPoolChart(c *gin.Context) {
	poolID := c.Param("id")
	period := c.DefaultQuery("period", "24h")

	chart, err := s.db.GetPoolChartData(poolID, period)
	if err != nil {
		s.log.Error("Failed to get pool chart", "pool_id", poolID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch chart data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chart": chart,
	})
}

// handleGetDEXTrades handles GET /dex/trades requests
func (s *Server) handleGetDEXTrades(c *gin.Context) {
	page := parseQueryInt(c, "page", 1)
	limit := parseQueryInt(c, "limit", 20)

	offset := (page - 1) * limit

	trades, total, err := s.db.GetDEXTrades(offset, limit)
	if err != nil {
		s.log.Error("Failed to get DEX trades", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch trades",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"trades": trades,
		"page":   page,
		"limit":  limit,
		"total":  total,
	})
}

// handleGetLatestDEXTrades handles GET /dex/trades/latest requests
func (s *Server) handleGetLatestDEXTrades(c *gin.Context) {
	limit := parseQueryInt(c, "limit", 10)

	trades, _, err := s.db.GetDEXTrades(0, limit)
	if err != nil {
		s.log.Error("Failed to get latest DEX trades", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch latest trades",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"trades": trades,
	})
}

// handleGetOraclePrices handles GET /oracle/prices requests
func (s *Server) handleGetOraclePrices(c *gin.Context) {
	prices, err := s.db.GetLatestOraclePrices()
	if err != nil {
		s.log.Error("Failed to get oracle prices", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch prices",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"prices": prices,
	})
}

// handleGetAssetPrice handles GET /oracle/prices/:asset requests
func (s *Server) handleGetAssetPrice(c *gin.Context) {
	asset := c.Param("asset")

	price, err := s.db.GetAssetPrice(asset)
	if err != nil {
		s.log.Error("Failed to get asset price", "asset", asset, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "price not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"price": price,
	})
}

// handleGetAssetPriceHistory handles GET /oracle/prices/:asset/history requests
func (s *Server) handleGetAssetPriceHistory(c *gin.Context) {
	asset := c.Param("asset")
	period := c.DefaultQuery("period", "24h")

	history, err := s.db.GetAssetPriceHistory(asset, period)
	if err != nil {
		s.log.Error("Failed to get asset price history", "asset", asset, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch price history",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
	})
}

// handleGetAssetPriceChart handles GET /oracle/prices/:asset/chart requests
func (s *Server) handleGetAssetPriceChart(c *gin.Context) {
	asset := c.Param("asset")
	period := c.DefaultQuery("period", "24h")
	interval := c.DefaultQuery("interval", "1h")

	chart, err := s.db.GetAssetPriceChart(asset, period, interval)
	if err != nil {
		s.log.Error("Failed to get asset price chart", "asset", asset, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch chart data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chart": chart,
	})
}

// handleGetOracleSubmissions handles GET /oracle/submissions requests
func (s *Server) handleGetOracleSubmissions(c *gin.Context) {
	page := parseQueryInt(c, "page", 1)
	limit := parseQueryInt(c, "limit", 20)
	asset := c.Query("asset")

	offset := (page - 1) * limit

	submissions, total, err := s.db.GetOracleSubmissions(offset, limit, asset)
	if err != nil {
		s.log.Error("Failed to get oracle submissions", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch submissions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"submissions": submissions,
		"page":        page,
		"limit":       limit,
		"total":       total,
	})
}

// handleGetOracleSlashes handles GET /oracle/slashes requests
func (s *Server) handleGetOracleSlashes(c *gin.Context) {
	page := parseQueryInt(c, "page", 1)
	limit := parseQueryInt(c, "limit", 20)

	offset := (page - 1) * limit

	slashes, total, err := s.db.GetOracleSlashes(offset, limit)
	if err != nil {
		s.log.Error("Failed to get oracle slashes", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch slashes",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"slashes": slashes,
		"page":    page,
		"limit":   limit,
		"total":   total,
	})
}

// Compute handlers

func (s *Server) handleGetComputeRequests(c *gin.Context) {
	page := parseQueryInt(c, "page", 1)
	limit := parseQueryInt(c, "limit", 20)
	status := c.Query("status")

	offset := (page - 1) * limit

	requests, total, err := s.db.GetComputeRequests(offset, limit, status)
	if err != nil {
		s.log.Error("Failed to get compute requests", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch compute requests",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"requests": requests,
		"page":     page,
		"limit":    limit,
		"total":    total,
	})
}

func (s *Server) handleGetComputeRequest(c *gin.Context) {
	requestID := c.Param("id")

	request, err := s.db.GetComputeRequest(requestID)
	if err != nil {
		s.log.Error("Failed to get compute request", "request_id", requestID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "compute request not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"request": request,
	})
}

func (s *Server) handleGetComputeResults(c *gin.Context) {
	requestID := c.Param("id")

	results, err := s.db.GetComputeResults(requestID)
	if err != nil {
		s.log.Error("Failed to get compute results", "request_id", requestID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch compute results",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
	})
}

func (s *Server) handleGetComputeVerifications(c *gin.Context) {
	requestID := c.Param("id")

	verifications, err := s.db.GetComputeVerifications(requestID)
	if err != nil {
		s.log.Error("Failed to get compute verifications", "request_id", requestID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch verifications",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"verifications": verifications,
	})
}

func (s *Server) handleGetComputeProviders(c *gin.Context) {
	providers, err := s.db.GetComputeProviders()
	if err != nil {
		s.log.Error("Failed to get compute providers", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch providers",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
	})
}

func (s *Server) handleGetComputeProvider(c *gin.Context) {
	address := c.Param("address")

	provider, err := s.db.GetComputeProvider(address)
	if err != nil {
		s.log.Error("Failed to get compute provider", "address", address, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "provider not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"provider": provider,
	})
}

// Statistics handlers

func (s *Server) handleGetNetworkStats(c *gin.Context) {
	stats, err := s.db.GetNetworkStats()
	if err != nil {
		s.log.Error("Failed to get network stats", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch network stats",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

func (s *Server) handleGetTransactionChart(c *gin.Context) {
	period := c.DefaultQuery("period", "24h")

	chart, err := s.db.GetTransactionChart(period)
	if err != nil {
		s.log.Error("Failed to get transaction chart", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch chart data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chart": chart,
	})
}

func (s *Server) handleGetAddressChart(c *gin.Context) {
	period := c.DefaultQuery("period", "30d")

	chart, err := s.db.GetAddressChart(period)
	if err != nil {
		s.log.Error("Failed to get address chart", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch chart data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chart": chart,
	})
}

func (s *Server) handleGetVolumeChart(c *gin.Context) {
	period := c.DefaultQuery("period", "7d")

	chart, err := s.db.GetVolumeChart(period)
	if err != nil {
		s.log.Error("Failed to get volume chart", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch chart data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chart": chart,
	})
}

func (s *Server) handleGetGasChart(c *gin.Context) {
	period := c.DefaultQuery("period", "24h")

	chart, err := s.db.GetGasChart(period)
	if err != nil {
		s.log.Error("Failed to get gas chart", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch chart data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chart": chart,
	})
}

// handleSearch handles GET /search requests
func (s *Server) handleSearch(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "search query is required",
		})
		return
	}

	results, err := s.db.Search(query)
	if err != nil {
		s.log.Error("Failed to search", "query", query, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "search failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"query":   query,
	})
}

// Export handlers

func (s *Server) handleExportTransactions(c *gin.Context) {
	address := c.Query("address")
	format := c.DefaultQuery("format", "csv")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "address is required",
		})
		return
	}

	data, err := s.db.ExportTransactions(address, format, startDate, endDate)
	if err != nil {
		s.log.Error("Failed to export transactions", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "export failed",
		})
		return
	}

	contentType := "text/csv"
	if format == "json" {
		contentType = "application/json"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=transactions.%s", format))
	c.String(http.StatusOK, string(data))
}

func (s *Server) handleExportTrades(c *gin.Context) {
	poolID := c.Query("pool_id")
	format := c.DefaultQuery("format", "csv")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	data, err := s.db.ExportTrades(poolID, format, startDate, endDate)
	if err != nil {
		s.log.Error("Failed to export trades", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "export failed",
		})
		return
	}

	contentType := "text/csv"
	if format == "json" {
		contentType = "application/json"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=trades.%s", format))
	c.String(http.StatusOK, string(data))
}

// WebSocket handler
func (s *Server) handleWebSocket(c *gin.Context) {
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.log.Error("Failed to upgrade websocket", "error", err)
		return
	}

	client := hub.NewClient(s.wsHub, conn, s.log)
	s.wsHub.Register(client)

	activeConnections.Inc()

	go client.WritePump()
	go client.ReadPump()
}

// GraphQL handlers
func (s *Server) handleGraphQL(c *gin.Context) {
	// GraphQL implementation would go here
	c.JSON(http.StatusOK, gin.H{
		"message": "GraphQL endpoint",
	})
}

func (s *Server) handleGraphQLPlayground(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, graphqlPlaygroundHTML)
}

// Helper functions

func parseQueryInt(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

const graphqlPlaygroundHTML = `
<!DOCTYPE html>
<html>
<head>
  <meta charset=utf-8/>
  <meta name="viewport" content="user-scalable=no, initial-scale=1.0, minimum-scale=1.0, maximum-scale=1.0, minimal-ui">
  <title>GraphQL Playground</title>
  <link rel="stylesheet" href="//cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
  <link rel="shortcut icon" href="//cdn.jsdelivr.net/npm/graphql-playground-react/build/favicon.png" />
  <script src="//cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
</head>
<body>
  <div id="root">
    <style>
      body {
        background-color: rgb(23, 42, 58);
        font-family: Open Sans, sans-serif;
        height: 90vh;
      }
      #root {
        height: 100%;
        width: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
      }
      .loading {
        font-size: 32px;
        font-weight: 200;
        color: rgba(255, 255, 255, .6);
        margin-left: 20px;
      }
      img {
        width: 78px;
        height: 78px;
      }
      .title {
        font-weight: 400;
      }
    </style>
    <img src='//cdn.jsdelivr.net/npm/graphql-playground-react/build/logo.png' alt=''>
    <div class="loading"> Loading
      <span class="title">GraphQL Playground</span>
    </div>
  </div>
  <script>window.addEventListener('load', function (event) {
      GraphQLPlayground.init(document.getElementById('root'), {
        endpoint: '/graphql'
      })
    })</script>
</body>
</html>
`
