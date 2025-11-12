package api

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PoolService handles liquidity pool operations
type PoolService struct {
	pools map[string]*Pool
	mu    sync.RWMutex
}

// NewPoolService creates a new pool service
func NewPoolService() *PoolService {
	ps := &PoolService{
		pools: make(map[string]*Pool),
	}

	// Initialize default pools
	ps.initializeDefaultPools()

	return ps
}

// initializeDefaultPools creates initial liquidity pools
func (ps *PoolService) initializeDefaultPools() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// PAW/USDC pool
	ps.pools["pool-1"] = &Pool{
		ID:              "pool-1",
		TokenA:          "paw",
		TokenB:          "usdc",
		ReserveA:        "1000000000000000000000", // 1000 PAW
		ReserveB:        "10000000000",             // 10,000 USDC
		LiquidityShares: "100000000000000000000",  // 100 LP tokens
		SwapFee:         0.003,                     // 0.3%
		PoolType:        "amm",
	}

	// PAW/ETH pool
	ps.pools["pool-2"] = &Pool{
		ID:              "pool-2",
		TokenA:          "paw",
		TokenB:          "eth",
		ReserveA:        "500000000000000000000", // 500 PAW
		ReserveB:        "2500000000000000000",   // 2.5 ETH
		LiquidityShares: "50000000000000000000",  // 50 LP tokens
		SwapFee:         0.003,
		PoolType:        "amm",
	}
}

// handleGetPools returns all liquidity pools
func (s *Server) handleGetPools(c *gin.Context) {
	if s.poolService == nil {
		s.poolService = NewPoolService()
	}

	s.poolService.mu.RLock()
	pools := make([]Pool, 0, len(s.poolService.pools))
	for _, pool := range s.poolService.pools {
		pools = append(pools, *pool)
	}
	s.poolService.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"pools": pools,
		"count": len(pools),
	})
}

// handleGetPool returns a specific pool
func (s *Server) handleGetPool(c *gin.Context) {
	if s.poolService == nil {
		s.poolService = NewPoolService()
	}

	poolID := c.Param("pool_id")

	s.poolService.mu.RLock()
	pool, exists := s.poolService.pools[poolID]
	s.poolService.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "Pool not found",
		})
		return
	}

	c.JSON(http.StatusOK, pool)
}

// handleGetPoolLiquidity returns pool liquidity information
func (s *Server) handleGetPoolLiquidity(c *gin.Context) {
	if s.poolService == nil {
		s.poolService = NewPoolService()
	}

	poolID := c.Param("pool_id")

	s.poolService.mu.RLock()
	pool, exists := s.poolService.pools[poolID]
	s.poolService.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "Pool not found",
		})
		return
	}

	// Calculate liquidity metrics
	reserveA, _ := strconv.ParseFloat(pool.ReserveA, 64)
	reserveB, _ := strconv.ParseFloat(pool.ReserveB, 64)

	// Calculate price (B/A)
	price := reserveB / reserveA

	c.JSON(http.StatusOK, gin.H{
		"pool_id":          pool.ID,
		"token_a":          pool.TokenA,
		"token_b":          pool.TokenB,
		"reserve_a":        pool.ReserveA,
		"reserve_b":        pool.ReserveB,
		"liquidity_shares": pool.LiquidityShares,
		"price":            price,
		"swap_fee":         pool.SwapFee,
	})
}

// handleAddLiquidity adds liquidity to a pool
func (s *Server) handleAddLiquidity(c *gin.Context) {
	if s.poolService == nil {
		s.poolService = NewPoolService()
	}

	userAddress, exists := c.Get("address")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	var req AddLiquidityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Details: err.Error(),
		})
		return
	}

	// Validate amounts
	amountA, err := sdk.NewDecFromStr(req.AmountA)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid amount A",
			Details: err.Error(),
		})
		return
	}

	amountB, err := sdk.NewDecFromStr(req.AmountB)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid amount B",
			Details: err.Error(),
		})
		return
	}

	s.poolService.mu.Lock()
	pool, exists := s.poolService.pools[req.PoolID]
	if !exists {
		s.poolService.mu.Unlock()
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "Pool not found",
		})
		return
	}

	// Calculate LP shares to mint
	// shares = (amountA / reserveA) * totalShares
	reserveA, _ := sdk.NewDecFromStr(pool.ReserveA)
	totalShares, _ := sdk.NewDecFromStr(pool.LiquidityShares)

	lpShares := amountA.Quo(reserveA).Mul(totalShares)

	// Update pool reserves
	newReserveA := reserveA.Add(amountA)
	newReserveB, _ := sdk.NewDecFromStr(pool.ReserveB)
	newReserveB = newReserveB.Add(amountB)
	newTotalShares := totalShares.Add(lpShares)

	pool.ReserveA = newReserveA.String()
	pool.ReserveB = newReserveB.String()
	pool.LiquidityShares = newTotalShares.String()

	s.poolService.mu.Unlock()

	// In production, broadcast transaction to blockchain
	txHash := "0x" + generateOrderID()

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"tx_hash":   txHash,
		"pool_id":   req.PoolID,
		"lp_shares": lpShares.String(),
		"amount_a":  req.AmountA,
		"amount_b":  req.AmountB,
		"message":   "Liquidity added successfully",
	})
}

// handleRemoveLiquidity removes liquidity from a pool
func (s *Server) handleRemoveLiquidity(c *gin.Context) {
	if s.poolService == nil {
		s.poolService = NewPoolService()
	}

	userAddress, exists := c.Get("address")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	var req RemoveLiquidityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Details: err.Error(),
		})
		return
	}

	// Validate shares
	shares, err := sdk.NewDecFromStr(req.Shares)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid shares amount",
			Details: err.Error(),
		})
		return
	}

	s.poolService.mu.Lock()
	pool, exists := s.poolService.pools[req.PoolID]
	if !exists {
		s.poolService.mu.Unlock()
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "Pool not found",
		})
		return
	}

	// Calculate amounts to withdraw
	// amountA = (shares / totalShares) * reserveA
	totalShares, _ := sdk.NewDecFromStr(pool.LiquidityShares)
	reserveA, _ := sdk.NewDecFromStr(pool.ReserveA)
	reserveB, _ := sdk.NewDecFromStr(pool.ReserveB)

	withdrawnA := shares.Quo(totalShares).Mul(reserveA)
	withdrawnB := shares.Quo(totalShares).Mul(reserveB)

	// Update pool reserves
	newReserveA := reserveA.Sub(withdrawnA)
	newReserveB := reserveB.Sub(withdrawnB)
	newTotalShares := totalShares.Sub(shares)

	pool.ReserveA = newReserveA.String()
	pool.ReserveB = newReserveB.String()
	pool.LiquidityShares = newTotalShares.String()

	s.poolService.mu.Unlock()

	// In production, broadcast transaction to blockchain
	txHash := "0x" + generateOrderID()

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"tx_hash":    txHash,
		"pool_id":    req.PoolID,
		"shares":     req.Shares,
		"received_a": withdrawnA.String(),
		"received_b": withdrawnB.String(),
		"message":    "Liquidity removed successfully",
	})
}

// Add poolService to Server struct
var poolServiceInstance *PoolService

func (s *Server) getPoolService() *PoolService {
	if s.poolService == nil {
		if poolServiceInstance == nil {
			poolServiceInstance = NewPoolService()
		}
		s.poolService = poolServiceInstance
	}
	return s.poolService
}

// Update Server struct to include poolService
type poolServiceHolder struct {
	poolService *PoolService
}
