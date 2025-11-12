package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// handleGetPrice returns the current price
func (s *Server) handleGetPrice(c *gin.Context) {
	s.tradingService.mu.RLock()
	priceData := PriceResponse{
		CurrentPrice:     s.tradingService.currentPrice,
		Change24h:        s.tradingService.priceChange24h,
		ChangePercent24h: (s.tradingService.priceChange24h / s.tradingService.currentPrice) * 100,
		High24h:          s.tradingService.high24h,
		Low24h:           s.tradingService.low24h,
		Volume24h:        s.tradingService.volume24h,
		LastUpdated:      time.Now(),
	}
	s.tradingService.mu.RUnlock()

	c.JSON(http.StatusOK, priceData)
}

// handleGetMarketStats returns comprehensive market statistics
func (s *Server) handleGetMarketStats(c *gin.Context) {
	s.tradingService.mu.RLock()

	// Calculate total liquidity from pools
	var totalLiquidity float64
	if s.poolService != nil {
		s.poolService.mu.RLock()
		for _, pool := range s.poolService.pools {
			// Simple calculation - in production, convert to USD properly
			totalLiquidity += 1000.0 // Placeholder
		}
		s.poolService.mu.RUnlock()
	}

	stats := MarketStats{
		Price:                 s.tradingService.currentPrice,
		Volume24h:             s.tradingService.volume24h,
		VolumeChange24h:       0.0, // Calculate from historical data
		High24h:               s.tradingService.high24h,
		Low24h:                s.tradingService.low24h,
		PriceChange24h:        s.tradingService.priceChange24h,
		PriceChangePercent24h: (s.tradingService.priceChange24h / s.tradingService.currentPrice) * 100,
		MarketCap:             s.tradingService.currentPrice * 50000000, // Total supply * price
		TotalLiquidity:        totalLiquidity,
		TotalTrades:           int64(len(s.tradingService.trades)),
		LastUpdated:           time.Now(),
	}
	s.tradingService.mu.RUnlock()

	c.JSON(http.StatusOK, stats)
}

// handleGet24HStats returns 24-hour statistics
func (s *Server) handleGet24HStats(c *gin.Context) {
	s.tradingService.mu.RLock()
	defer s.tradingService.mu.RUnlock()

	// Calculate trades in last 24 hours
	cutoff := time.Now().Add(-24 * time.Hour)
	var trades24h int
	var volume24h float64

	for _, trade := range s.tradingService.trades {
		if trade.Timestamp.After(cutoff) {
			trades24h++
			volume24h += trade.Amount
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"price":          s.tradingService.currentPrice,
		"change_24h":     s.tradingService.priceChange24h,
		"change_percent": (s.tradingService.priceChange24h / s.tradingService.currentPrice) * 100,
		"high_24h":       s.tradingService.high24h,
		"low_24h":        s.tradingService.low24h,
		"volume_24h":     volume24h,
		"trades_24h":     trades24h,
		"last_updated":   time.Now(),
	})
}
