package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TradingService handles trading operations
type TradingService struct {
	clientCtx client.Context
	wsHub     *WebSocketHub
	orders    map[string]*Order
	trades    []Trade
	mu        sync.RWMutex

	// Market state
	currentPrice     float64
	high24h          float64
	low24h           float64
	volume24h        float64
	priceChange24h   float64
}

// NewTradingService creates a new trading service
func NewTradingService(clientCtx client.Context, wsHub *WebSocketHub) *TradingService {
	ts := &TradingService{
		clientCtx:    clientCtx,
		wsHub:        wsHub,
		orders:       make(map[string]*Order),
		trades:       make([]Trade, 0),
		currentPrice: 10.00, // Initial price
		high24h:      10.50,
		low24h:       9.50,
	}

	// Start background price update simulation
	go ts.simulatePriceUpdates()

	return ts
}

// handleCreateOrder handles order creation
func (s *Server) handleCreateOrder(c *gin.Context) {
	// Get user from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	username, _ := c.Get("username")
	address, _ := c.Get("address")

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Details: err.Error(),
		})
		return
	}

	// Create order
	order := &Order{
		ID:        generateOrderID(),
		UserID:    userID.(string),
		Username:  username.(string),
		Type:      req.OrderType,
		Price:     req.Price,
		Amount:    req.Amount,
		Filled:    0,
		Status:    "open",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store order
	s.tradingService.mu.Lock()
	s.tradingService.orders[order.ID] = order
	s.tradingService.mu.Unlock()

	// Try to match orders
	go s.tradingService.matchOrders()

	// Broadcast order book update
	s.tradingService.broadcastOrderBookUpdate()

	c.JSON(http.StatusCreated, CreateOrderResponse{
		OrderID:   order.ID,
		OrderType: order.Type,
		Price:     order.Price,
		Amount:    order.Amount,
		Status:    order.Status,
		Timestamp: order.CreatedAt.Unix(),
	})
}

// handleGetOrderBook returns the current order book
func (s *Server) handleGetOrderBook(c *gin.Context) {
	orderBook := s.tradingService.GetOrderBook()
	c.JSON(http.StatusOK, orderBook)
}

// handleGetRecentTrades returns recent trades
func (s *Server) handleGetRecentTrades(c *gin.Context) {
	limit := 20
	if l, ok := c.GetQuery("limit"); ok {
		fmt.Sscanf(l, "%d", &limit)
	}

	s.tradingService.mu.RLock()
	trades := s.tradingService.trades
	s.tradingService.mu.RUnlock()

	// Get most recent trades
	start := len(trades) - limit
	if start < 0 {
		start = 0
	}

	recentTrades := make([]Trade, 0)
	for i := len(trades) - 1; i >= start; i-- {
		recentTrades = append(recentTrades, trades[i])
	}

	c.JSON(http.StatusOK, gin.H{
		"trades": recentTrades,
		"count":  len(recentTrades),
	})
}

// handleGetTradeHistory returns trade history with pagination
func (s *Server) handleGetTradeHistory(c *gin.Context) {
	pagination := DefaultPagination()
	if err := c.ShouldBindQuery(&pagination); err == nil {
		if pagination.Page < 1 {
			pagination.Page = 1
		}
		if pagination.PageSize < 1 || pagination.PageSize > 100 {
			pagination.PageSize = 20
		}
	}

	s.tradingService.mu.RLock()
	totalTrades := len(s.tradingService.trades)
	s.tradingService.mu.RUnlock()

	offset := (pagination.Page - 1) * pagination.PageSize
	end := offset + pagination.PageSize

	if offset >= totalTrades {
		c.JSON(http.StatusOK, TradeHistoryResponse{
			Trades:     []Trade{},
			TotalCount: totalTrades,
			Page:       pagination.Page,
			PageSize:   pagination.PageSize,
		})
		return
	}

	if end > totalTrades {
		end = totalTrades
	}

	s.tradingService.mu.RLock()
	trades := make([]Trade, end-offset)
	copy(trades, s.tradingService.trades[offset:end])
	s.tradingService.mu.RUnlock()

	c.JSON(http.StatusOK, TradeHistoryResponse{
		Trades:     trades,
		TotalCount: totalTrades,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
	})
}

// handleGetMyOrders returns user's orders
func (s *Server) handleGetMyOrders(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	s.tradingService.mu.RLock()
	defer s.tradingService.mu.RUnlock()

	userOrders := make([]Order, 0)
	for _, order := range s.tradingService.orders {
		if order.UserID == userID.(string) {
			userOrders = append(userOrders, *order)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": userOrders,
		"count":  len(userOrders),
	})
}

// handleCancelOrder cancels an order
func (s *Server) handleCancelOrder(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	orderID := c.Param("order_id")

	s.tradingService.mu.Lock()
	order, exists := s.tradingService.orders[orderID]
	if !exists {
		s.tradingService.mu.Unlock()
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Order not found"})
		return
	}

	if order.UserID != userID.(string) {
		s.tradingService.mu.Unlock()
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Not authorized to cancel this order"})
		return
	}

	if order.Status != "open" && order.Status != "partial" {
		s.tradingService.mu.Unlock()
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Order cannot be cancelled"})
		return
	}

	order.Status = "cancelled"
	order.UpdatedAt = time.Now()
	s.tradingService.mu.Unlock()

	// Broadcast order book update
	s.tradingService.broadcastOrderBookUpdate()

	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Order cancelled successfully",
		Data:    order,
	})
}

// GetOrderBook returns the current order book
func (ts *TradingService) GetOrderBook() OrderBook {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	bids := make([]OrderBookEntry, 0)
	asks := make([]OrderBookEntry, 0)

	// Aggregate orders by price
	bidMap := make(map[float64]float64)
	askMap := make(map[float64]float64)

	for _, order := range ts.orders {
		if order.Status != "open" && order.Status != "partial" {
			continue
		}

		remainingAmount := order.Amount - order.Filled
		if remainingAmount <= 0 {
			continue
		}

		if order.Type == "buy" {
			bidMap[order.Price] += remainingAmount
		} else {
			askMap[order.Price] += remainingAmount
		}
	}

	// Convert to slices
	for price, amount := range bidMap {
		bids = append(bids, OrderBookEntry{
			Price:  price,
			Amount: amount,
			Total:  price * amount,
		})
	}

	for price, amount := range askMap {
		asks = append(asks, OrderBookEntry{
			Price:  price,
			Amount: amount,
			Total:  price * amount,
		})
	}

	// Sort bids (highest first)
	sort.Slice(bids, func(i, j int) bool {
		return bids[i].Price > bids[j].Price
	})

	// Sort asks (lowest first)
	sort.Slice(asks, func(i, j int) bool {
		return asks[i].Price < asks[j].Price
	})

	// Calculate spread
	var spread float64
	if len(bids) > 0 && len(asks) > 0 {
		spread = asks[0].Price - bids[0].Price
	}

	return OrderBook{
		Bids:   bids,
		Asks:   asks,
		Spread: spread,
	}
}

// matchOrders attempts to match buy and sell orders
func (ts *TradingService) matchOrders() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Get all open orders
	buyOrders := make([]*Order, 0)
	sellOrders := make([]*Order, 0)

	for _, order := range ts.orders {
		if order.Status != "open" && order.Status != "partial" {
			continue
		}

		if order.Type == "buy" {
			buyOrders = append(buyOrders, order)
		} else {
			sellOrders = append(sellOrders, order)
		}
	}

	// Sort buy orders by price (highest first)
	sort.Slice(buyOrders, func(i, j int) bool {
		return buyOrders[i].Price > buyOrders[j].Price
	})

	// Sort sell orders by price (lowest first)
	sort.Slice(sellOrders, func(i, j int) bool {
		return sellOrders[i].Price < sellOrders[j].Price
	})

	// Match orders
	for _, buyOrder := range buyOrders {
		for _, sellOrder := range sellOrders {
			if buyOrder.Price >= sellOrder.Price {
				// Orders can be matched
				buyRemaining := buyOrder.Amount - buyOrder.Filled
				sellRemaining := sellOrder.Amount - sellOrder.Filled

				if buyRemaining <= 0 || sellRemaining <= 0 {
					continue
				}

				// Calculate matched amount
				matchedAmount := buyRemaining
				if sellRemaining < matchedAmount {
					matchedAmount = sellRemaining
				}

				// Execute trade
				tradePrice := sellOrder.Price // Take the ask price
				trade := Trade{
					ID:        generateTradeID(),
					BuyerID:   buyOrder.UserID,
					SellerID:  sellOrder.UserID,
					OrderType: "buy", // Direction of taker
					Price:     tradePrice,
					Amount:    matchedAmount,
					Total:     tradePrice * matchedAmount,
					Timestamp: time.Now(),
				}

				ts.trades = append(ts.trades, trade)

				// Update orders
				buyOrder.Filled += matchedAmount
				sellOrder.Filled += matchedAmount
				buyOrder.UpdatedAt = time.Now()
				sellOrder.UpdatedAt = time.Now()

				// Update order status
				if buyOrder.Filled >= buyOrder.Amount {
					buyOrder.Status = "filled"
				} else {
					buyOrder.Status = "partial"
				}

				if sellOrder.Filled >= sellOrder.Amount {
					sellOrder.Status = "filled"
				} else {
					sellOrder.Status = "partial"
				}

				// Update market stats
				ts.currentPrice = tradePrice
				ts.volume24h += matchedAmount
				if tradePrice > ts.high24h {
					ts.high24h = tradePrice
				}
				if tradePrice < ts.low24h {
					ts.low24h = tradePrice
				}

				// Broadcast trade
				go ts.broadcastNewTrade(trade)
			}
		}
	}
}

// broadcastOrderBookUpdate broadcasts order book updates via WebSocket
func (ts *TradingService) broadcastOrderBookUpdate() {
	if ts.wsHub == nil {
		return
	}

	orderBook := ts.GetOrderBook()
	ts.wsHub.Broadcast(WSMessage{
		Type: "orderbook_update",
		Data: orderBook,
	})
}

// broadcastNewTrade broadcasts new trade via WebSocket
func (ts *TradingService) broadcastNewTrade(trade Trade) {
	if ts.wsHub == nil {
		return
	}

	ts.wsHub.Broadcast(WSMessage{
		Type: "new_trade",
		Data: trade,
	})

	// Also broadcast updated order book
	ts.broadcastOrderBookUpdate()
}

// simulatePriceUpdates simulates price updates (for demo purposes)
func (ts *TradingService) simulatePriceUpdates() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if ts.wsHub == nil {
			continue
		}

		ts.mu.RLock()
		priceData := PriceResponse{
			CurrentPrice:     ts.currentPrice,
			Change24h:        ts.priceChange24h,
			ChangePercent24h: (ts.priceChange24h / ts.currentPrice) * 100,
			High24h:          ts.high24h,
			Low24h:           ts.low24h,
			Volume24h:        ts.volume24h,
			LastUpdated:      time.Now(),
		}
		ts.mu.RUnlock()

		ts.wsHub.Broadcast(WSMessage{
			Type: "price_update",
			Data: priceData,
		})
	}
}

// Helper functions

func generateOrderID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "ORD-" + hex.EncodeToString(b)[:16]
}

func generateTradeID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "TRD-" + hex.EncodeToString(b)[:16]
}
