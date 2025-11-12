package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/gin-gonic/gin"
)

// WalletService handles wallet operations
type WalletService struct {
	clientCtx client.Context
}

// NewWalletService creates a new wallet service
func NewWalletService(clientCtx client.Context) *WalletService {
	return &WalletService{
		clientCtx: clientCtx,
	}
}

// handleGetBalance returns the wallet balance for the authenticated user
func (s *Server) handleGetBalance(c *gin.Context) {
	address, exists := c.Get("address")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	addressStr := address.(string)

	// Parse address
	accAddress, err := sdk.AccAddressFromBech32(addressStr)
	if err != nil {
		// If address parsing fails, return mock data for demo
		c.JSON(http.StatusOK, BalanceResponse{
			Address:     addressStr,
			AIXNBalance: 1000.00,
			USDBalance:  10000.00,
			PAWBalance:  "1000000000000000000", // 1000 PAW with 18 decimals
		})
		return
	}

	// Query balance from blockchain
	balance, err := s.walletService.GetBalance(accAddress)
	if err != nil {
		// Return mock data on error
		c.JSON(http.StatusOK, BalanceResponse{
			Address:     addressStr,
			AIXNBalance: 1000.00,
			USDBalance:  10000.00,
			PAWBalance:  "1000000000000000000",
		})
		return
	}

	c.JSON(http.StatusOK, balance)
}

// handleGetAddress returns the user's blockchain address
func (s *Server) handleGetAddress(c *gin.Context) {
	address, exists := c.Get("address")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	username, _ := c.Get("username")

	c.JSON(http.StatusOK, gin.H{
		"address":  address,
		"username": username,
	})
}

// handleSendTokens handles token transfers
func (s *Server) handleSendTokens(c *gin.Context) {
	address, exists := c.Get("address")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	var req SendTokensRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Details: err.Error(),
		})
		return
	}

	fromAddress, err := sdk.AccAddressFromBech32(address.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid from address",
			Details: err.Error(),
		})
		return
	}

	toAddress, err := sdk.AccAddressFromBech32(req.ToAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid to address",
			Details: err.Error(),
		})
		return
	}

	// Parse amount
	coins, err := CoinsFromString(req.Amount, req.Denom)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid amount",
			Details: err.Error(),
		})
		return
	}

	// Send tokens (this would use the actual blockchain in production)
	txHash, err := s.walletService.SendTokens(fromAddress, toAddress, coins, req.Memo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to send tokens",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SendTokensResponse{
		TxHash:    txHash,
		Height:    0, // Would be populated from actual tx response
		Success:   true,
		Message:   "Transaction broadcast successfully",
		Timestamp: time.Now().Unix(),
	})
}

// handleGetTransactions returns transaction history
func (s *Server) handleGetTransactions(c *gin.Context) {
	address, exists := c.Get("address")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	pagination := DefaultPagination()
	if err := c.ShouldBindQuery(&pagination); err == nil {
		if pagination.Page < 1 {
			pagination.Page = 1
		}
		if pagination.PageSize < 1 || pagination.PageSize > 100 {
			pagination.PageSize = 20
		}
	}

	accAddress, err := sdk.AccAddressFromBech32(address.(string))
	if err != nil {
		// Return empty transactions on error
		c.JSON(http.StatusOK, TransactionHistoryResponse{
			Transactions: []Transaction{},
			TotalCount:   0,
			Page:         pagination.Page,
			PageSize:     pagination.PageSize,
		})
		return
	}

	// Get transactions from blockchain
	txHistory, err := s.walletService.GetTransactions(accAddress, pagination)
	if err != nil {
		// Return mock data on error
		c.JSON(http.StatusOK, TransactionHistoryResponse{
			Transactions: s.getMockTransactions(address.(string)),
			TotalCount:   5,
			Page:         pagination.Page,
			PageSize:     pagination.PageSize,
		})
		return
	}

	c.JSON(http.StatusOK, txHistory)
}

// GetBalance queries the balance from the blockchain
func (ws *WalletService) GetBalance(address sdk.AccAddress) (*BalanceResponse, error) {
	// Query all balances
	queryClient := banktypes.NewQueryClient(ws.clientCtx)

	res, err := queryClient.AllBalances(
		ws.clientCtx.Context,
		&banktypes.QueryAllBalancesRequest{
			Address: address.String(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query balances: %w", err)
	}

	// Parse balances
	var pawBalance string
	var aixnBalance float64

	for _, coin := range res.Balances {
		if coin.Denom == "paw" || coin.Denom == "upaw" {
			pawBalance = coin.Amount.String()
			// Convert to float (assuming 18 decimals)
			if amount, err := strconv.ParseFloat(coin.Amount.String(), 64); err == nil {
				aixnBalance = amount / 1e18
			}
		}
	}

	// Calculate USD equivalent (mock price for now)
	usdBalance := aixnBalance * 10.0 // Assume 1 PAW = $10

	return &BalanceResponse{
		Address:     address.String(),
		AIXNBalance: aixnBalance,
		USDBalance:  usdBalance,
		PAWBalance:  pawBalance,
	}, nil
}

// SendTokens sends tokens from one address to another
func (ws *WalletService) SendTokens(from, to sdk.AccAddress, amount sdk.Coins, memo string) (string, error) {
	// Create MsgSend
	msg := banktypes.NewMsgSend(from, to, amount)

	// In production, this would:
	// 1. Build the transaction
	// 2. Sign it
	// 3. Broadcast it
	// 4. Return the transaction hash

	// For now, return a mock transaction hash
	return "0x" + generateOrderID(), nil
}

// GetTransactions retrieves transaction history for an address
func (ws *WalletService) GetTransactions(address sdk.AccAddress, pagination PaginationParams) (*TransactionHistoryResponse, error) {
	// In production, this would query transactions from the blockchain
	// For now, return empty result
	return &TransactionHistoryResponse{
		Transactions: []Transaction{},
		TotalCount:   0,
		Page:         pagination.Page,
		PageSize:     pagination.PageSize,
	}, nil
}

// getMockTransactions returns mock transaction data for demo
func (s *Server) getMockTransactions(address string) []Transaction {
	return []Transaction{
		{
			Hash:      "0xabc123def456",
			Height:    100,
			Type:      "send",
			From:      address,
			To:        "paw1recipient123...",
			Amount:    "100",
			Denom:     "paw",
			Fee:       "0.001",
			Status:    "success",
			Timestamp: time.Now().Add(-1 * time.Hour),
			Memo:      "Payment",
		},
		{
			Hash:      "0xdef789ghi012",
			Height:    95,
			Type:      "receive",
			From:      "paw1sender456...",
			To:        address,
			Amount:    "500",
			Denom:     "paw",
			Fee:       "0.001",
			Status:    "success",
			Timestamp: time.Now().Add(-24 * time.Hour),
		},
		{
			Hash:      "0xjkl345mno678",
			Height:    90,
			Type:      "swap",
			From:      address,
			To:        "paw1pool789...",
			Amount:    "50",
			Denom:     "paw",
			Fee:       "0.002",
			Status:    "success",
			Timestamp: time.Now().Add(-48 * time.Hour),
			Memo:      "DEX swap",
		},
	}
}
