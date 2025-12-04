package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// ADVANCED DEX API HANDLERS
// Production-grade endpoints for DEX analytics
// ============================================================================

// handleGetPoolPriceHistory handles GET /dex/pools/:id/price-history
func (s *Server) handleGetPoolPriceHistory(c *gin.Context) {
	poolID := c.Param("id")
	interval := c.DefaultQuery("interval", "1h")
	period := c.DefaultQuery("period", "24h")

	// Parse time range from period
	start, end := parsePeriodToTimeRange(period)

	history, err := s.db.GetPoolPriceHistory(c.Request.Context(), poolID, start, end, interval)
	if err != nil {
		s.log.Error("Failed to get pool price history", "pool_id", poolID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch price history",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"price_history": history,
		"pool_id":       poolID,
		"interval":      interval,
		"period":        period,
	})
}

// handleGetPoolLiquidityChart handles GET /dex/pools/:id/liquidity-chart
func (s *Server) handleGetPoolLiquidityChart(c *gin.Context) {
	poolID := c.Param("id")
	period := c.DefaultQuery("period", "24h")

	start, end := parsePeriodToTimeRange(period)

	liquidityData, err := s.db.GetPoolLiquidityHistory(c.Request.Context(), poolID, start, end)
	if err != nil {
		s.log.Error("Failed to get pool liquidity chart", "pool_id", poolID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch liquidity chart",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"liquidity_chart": liquidityData,
		"pool_id":         poolID,
		"period":          period,
	})
}

// handleGetPoolVolumeChart handles GET /dex/pools/:id/volume-chart
func (s *Server) handleGetPoolVolumeChart(c *gin.Context) {
	poolID := c.Param("id")
	period := c.DefaultQuery("period", "7d")
	interval := c.DefaultQuery("interval", "1h")

	start, end := parsePeriodToTimeRange(period)

	volumeData, err := s.db.GetPoolVolumeHistory(c.Request.Context(), poolID, start, end, interval)
	if err != nil {
		s.log.Error("Failed to get pool volume chart", "pool_id", poolID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch volume chart",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"volume_chart": volumeData,
		"pool_id":      poolID,
		"period":       period,
		"interval":     interval,
	})
}

// handleGetPoolFees handles GET /dex/pools/:id/fees
func (s *Server) handleGetPoolFees(c *gin.Context) {
	poolID := c.Param("id")
	period := c.DefaultQuery("period", "30d")

	start, end := parsePeriodToTimeRange(period)

	feeBreakdown, err := s.db.GetPoolFeeBreakdown(c.Request.Context(), poolID, start, end)
	if err != nil {
		s.log.Error("Failed to get pool fee breakdown", "pool_id", poolID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch fee breakdown",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"fee_breakdown": feeBreakdown,
		"pool_id":       poolID,
		"period":        period,
	})
}

// handleGetPoolAPRHistory handles GET /dex/pools/:id/apr-history
func (s *Server) handleGetPoolAPRHistory(c *gin.Context) {
	poolID := c.Param("id")
	period := c.DefaultQuery("period", "30d")

	start, end := parsePeriodToTimeRange(period)

	aprHistory, err := s.db.GetPoolAPRHistory(c.Request.Context(), poolID, start, end)
	if err != nil {
		s.log.Error("Failed to get pool APR history", "pool_id", poolID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch APR history",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"apr_history": aprHistory,
		"pool_id":     poolID,
		"period":      period,
	})
}

// handleGetPoolDepth handles GET /dex/pools/:id/depth
func (s *Server) handleGetPoolDepth(c *gin.Context) {
	poolID := c.Param("id")

	pool, err := s.db.GetDEXPool(poolID)
	if err != nil {
		s.log.Error("Failed to get pool depth", "pool_id", poolID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "pool not found",
		})
		return
	}

	// Calculate liquidity depth data
	depth := map[string]interface{}{
		"pool_id":   poolID,
		"token_a":   pool.TokenA,
		"token_b":   pool.TokenB,
		"reserve_a": pool.ReserveA,
		"reserve_b": pool.ReserveB,
		"tvl":       pool.TVL,
	}

	c.JSON(http.StatusOK, gin.H{
		"depth": depth,
	})
}

// handleGetPoolStatistics handles GET /dex/pools/:id/statistics
func (s *Server) handleGetPoolStatistics(c *gin.Context) {
	poolID := c.Param("id")
	period := c.DefaultQuery("period", "24h")

	start, end := parsePeriodToTimeRange(period)

	stats, err := s.db.GetPoolStatistics(c.Request.Context(), poolID, period, start, end)
	if err != nil {
		s.log.Error("Failed to get pool statistics", "pool_id", poolID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch pool statistics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"statistics": stats,
		"pool_id":    poolID,
		"period":     period,
	})
}

// ============================================================================
// USER POSITION HANDLERS
// ============================================================================

// handleGetUserDEXPositions handles GET /accounts/:address/dex-positions
func (s *Server) handleGetUserDEXPositions(c *gin.Context) {
	address := c.Param("address")
	status := c.DefaultQuery("status", "active")

	positions, err := s.db.GetUserDEXPositions(c.Request.Context(), address, status)
	if err != nil {
		s.log.Error("Failed to get user DEX positions", "address", address, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch positions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"positions": positions,
		"address":   address,
		"status":    status,
		"count":     len(positions),
	})
}

// handleGetUserDEXHistory handles GET /accounts/:address/dex-history
func (s *Server) handleGetUserDEXHistory(c *gin.Context) {
	address := c.Param("address")
	page := parseQueryInt(c, "page", 1)
	limit := parseQueryInt(c, "limit", 20)
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	history, total, err := s.db.GetUserDEXHistory(c.Request.Context(), address, offset, limit)
	if err != nil {
		s.log.Error("Failed to get user DEX history", "address", address, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch history",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"address": address,
		"page":    page,
		"limit":   limit,
		"total":   total,
	})
}

// handleGetUserDEXAnalytics handles GET /accounts/:address/dex-analytics
func (s *Server) handleGetUserDEXAnalytics(c *gin.Context) {
	address := c.Param("address")

	// Check cache first
	cacheKey := fmt.Sprintf("user_dex_analytics:%s", address)
	if cached, err := s.db.GetCachedAnalytics(c.Request.Context(), cacheKey); err == nil && cached != nil {
		c.JSON(http.StatusOK, gin.H{
			"analytics": cached,
			"address":   address,
			"cached":    true,
		})
		return
	}

	analytics, err := s.db.GetUserDEXAnalytics(c.Request.Context(), address)
	if err != nil {
		s.log.Error("Failed to get user DEX analytics", "address", address, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch analytics",
		})
		return
	}

	// Cache the result for 5 minutes
	s.db.SetCachedAnalytics(c.Request.Context(), cacheKey, "user_analytics", analytics, 5*time.Minute)

	c.JSON(http.StatusOK, gin.H{
		"analytics": analytics,
		"address":   address,
	})
}

// ============================================================================
// DEX-WIDE ANALYTICS HANDLERS
// ============================================================================

// handleGetDEXAnalyticsSummary handles GET /dex/analytics/summary
func (s *Server) handleGetDEXAnalyticsSummary(c *gin.Context) {
	// Check cache first
	cacheKey := "dex_analytics_summary"
	if cached, err := s.db.GetCachedAnalytics(c.Request.Context(), cacheKey); err == nil && cached != nil {
		c.JSON(http.StatusOK, gin.H{
			"summary": cached,
			"cached":  true,
		})
		return
	}

	summary, err := s.db.GetDEXAnalyticsSummary(c.Request.Context())
	if err != nil {
		s.log.Error("Failed to get DEX analytics summary", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch summary",
		})
		return
	}

	// Cache for 1 minute
	s.db.SetCachedAnalytics(c.Request.Context(), cacheKey, "dex_summary", summary, 1*time.Minute)

	c.JSON(http.StatusOK, gin.H{
		"summary": summary,
	})
}

// handleGetTopTradingPairs handles GET /dex/analytics/top-pairs
func (s *Server) handleGetTopTradingPairs(c *gin.Context) {
	period := c.DefaultQuery("period", "24h")
	limit := parseQueryInt(c, "limit", 10)
	if limit > 50 {
		limit = 50
	}

	topPairs, err := s.db.GetTopTradingPairs(c.Request.Context(), period, limit)
	if err != nil {
		s.log.Error("Failed to get top trading pairs", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch top pairs",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"top_pairs": topPairs,
		"period":    period,
		"limit":     limit,
	})
}

// ============================================================================
// SWAP SIMULATION HANDLER
// ============================================================================

// handleSimulateSwap handles POST /dex/simulate-swap
func (s *Server) handleSimulateSwap(c *gin.Context) {
	var req struct {
		PoolID    string `json:"pool_id" binding:"required"`
		TokenIn   string `json:"token_in" binding:"required"`
		AmountIn  string `json:"amount_in" binding:"required"`
		TokenOut  string `json:"token_out" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	// Get pool data
	pool, err := s.db.GetDEXPool(req.PoolID)
	if err != nil {
		s.log.Error("Failed to get pool for simulation", "pool_id", req.PoolID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "pool not found",
		})
		return
	}

	// Simple constant product AMM calculation (x * y = k)
	// In production, this would call the actual chain simulation
	amountIn, err := strconv.ParseFloat(req.AmountIn, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid amount_in",
		})
		return
	}

	reserveA, _ := strconv.ParseFloat(pool.ReserveA, 64)
	reserveB, _ := strconv.ParseFloat(pool.ReserveB, 64)
	swapFee, _ := strconv.ParseFloat(pool.SwapFee, 64)

	// Determine direction
	var amountOut float64
	var priceImpact float64

	if req.TokenIn == pool.TokenA {
		// Swapping A for B
		amountInWithFee := amountIn * (1 - swapFee)
		amountOut = (reserveB * amountInWithFee) / (reserveA + amountInWithFee)
		priceImpact = (amountIn / reserveA) * 100
	} else {
		// Swapping B for A
		amountInWithFee := amountIn * (1 - swapFee)
		amountOut = (reserveA * amountInWithFee) / (reserveB + amountInWithFee)
		priceImpact = (amountIn / reserveB) * 100
	}

	effectivePrice := amountOut / amountIn

	c.JSON(http.StatusOK, gin.H{
		"simulation": map[string]interface{}{
			"pool_id":         req.PoolID,
			"token_in":        req.TokenIn,
			"token_out":       req.TokenOut,
			"amount_in":       req.AmountIn,
			"amount_out":      fmt.Sprintf("%.6f", amountOut),
			"effective_price": fmt.Sprintf("%.6f", effectivePrice),
			"price_impact":    fmt.Sprintf("%.2f%%", priceImpact),
			"swap_fee":        fmt.Sprintf("%.2f%%", swapFee*100),
			"minimum_output":  fmt.Sprintf("%.6f", amountOut*0.99), // 1% slippage
		},
	})
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// parsePeriodToTimeRange converts period string to time range
func parsePeriodToTimeRange(period string) (time.Time, time.Time) {
	end := time.Now()
	var start time.Time

	switch period {
	case "1h":
		start = end.Add(-1 * time.Hour)
	case "24h":
		start = end.Add(-24 * time.Hour)
	case "7d":
		start = end.Add(-7 * 24 * time.Hour)
	case "30d":
		start = end.Add(-30 * 24 * time.Hour)
	case "1y":
		start = end.Add(-365 * 24 * time.Hour)
	default:
		start = end.Add(-24 * time.Hour)
	}

	return start, end
}
