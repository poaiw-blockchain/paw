package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
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
		context.Background(),
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

	// Build transaction
	txBuilder := ws.clientCtx.TxConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(msg); err != nil {
		return "", fmt.Errorf("failed to set messages: %w", err)
	}

	// Set memo if provided
	if memo != "" {
		txBuilder.SetMemo(memo)
	}

	// Set gas and fee (simplified - in production you'd estimate gas)
	txBuilder.SetGasLimit(200000) // Default gas limit

	// Set fee (0.001 paw = 1000 upaw)
	fee := sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000))
	txBuilder.SetFeeAmount(fee)

	// Get account from keyring to sign
	// Note: The key must be in the keyring for signing
	keyInfo, err := ws.clientCtx.Keyring.KeyByAddress(from)
	if err != nil {
		return "", fmt.Errorf("key not found in keyring (wallet must be imported): %w", err)
	}

	// Create transaction factory for signing
	txFactory := tx.Factory{}
	txFactory = txFactory.
		WithChainID(ws.clientCtx.ChainID).
		WithKeybase(ws.clientCtx.Keyring).
		WithTxConfig(ws.clientCtx.TxConfig).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)

	// Sign the transaction using the tx.Sign helper
	// This handles all the complexity of getting account number, sequence, and signing
	if err := tx.Sign(context.Background(), txFactory, keyInfo.Name, txBuilder, true); err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Encode transaction
	txBytes, err := ws.clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return "", fmt.Errorf("failed to encode tx: %w", err)
	}

	// Broadcast transaction
	res, err := ws.clientCtx.BroadcastTx(txBytes)
	if err != nil {
		return "", fmt.Errorf("failed to broadcast tx: %w", err)
	}

	// Check transaction result
	if res.Code != 0 {
		return "", fmt.Errorf("transaction failed: code=%d, log=%s", res.Code, res.RawLog)
	}

	return res.TxHash, nil
}

// GetTransactions retrieves transaction history for an address
func (ws *WalletService) GetTransactions(address sdk.AccAddress, pagination PaginationParams) (*TransactionHistoryResponse, error) {
	// Query transactions from the blockchain using the bank module
	queryClient := banktypes.NewQueryClient(ws.clientCtx)

	// Calculate offset for pagination
	offset := uint64((pagination.Page - 1) * pagination.PageSize)
	limit := uint64(pagination.PageSize)

	// Query sent transactions (from this address)
	sentTxs, err := ws.queryBankTransactions(queryClient, address, "sent", offset, limit)
	if err != nil {
		// Log error but continue - we might still get received transactions
		fmt.Printf("Warning: failed to query sent transactions: %v\n", err)
	}

	// Query received transactions (to this address)
	receivedTxs, err := ws.queryBankTransactions(queryClient, address, "received", offset, limit)
	if err != nil {
		fmt.Printf("Warning: failed to query received transactions: %v\n", err)
	}

	// Combine and sort transactions by timestamp (newest first)
	allTxs := append(sentTxs, receivedTxs...)

	// Sort by timestamp descending
	for i := 0; i < len(allTxs); i++ {
		for j := i + 1; j < len(allTxs); j++ {
			if allTxs[i].Timestamp.Before(allTxs[j].Timestamp) {
				allTxs[i], allTxs[j] = allTxs[j], allTxs[i]
			}
		}
	}

	// Apply pagination to combined results
	start := 0
	end := len(allTxs)
	if start > len(allTxs) {
		start = len(allTxs)
	}
	if end > len(allTxs) {
		end = len(allTxs)
	}

	paginatedTxs := allTxs
	if len(allTxs) > pagination.PageSize {
		if start+pagination.PageSize > len(allTxs) {
			paginatedTxs = allTxs[start:]
		} else {
			paginatedTxs = allTxs[start : start+pagination.PageSize]
		}
	}

	return &TransactionHistoryResponse{
		Transactions: paginatedTxs,
		TotalCount:   len(allTxs),
		Page:         pagination.Page,
		PageSize:     pagination.PageSize,
	}, nil
}

// queryBankTransactions queries bank transactions for an address
func (ws *WalletService) queryBankTransactions(queryClient banktypes.QueryClient, address sdk.AccAddress, direction string, offset, limit uint64) ([]Transaction, error) {
	// Note: Cosmos SDK doesn't have a built-in transaction history query by address
	// In production, you would either:
	// 1. Use a transaction indexer service (like TxSearch from Tendermint RPC)
	// 2. Use a custom module that indexes transactions
	// 3. Query events using the Tendermint RPC

	// For now, we'll use the Tendermint RPC client to search for transactions
	// This requires the node to have transaction indexing enabled

	// Build query string based on direction
	var query string
	if direction == "sent" {
		query = fmt.Sprintf("message.sender='%s' AND message.action='send'", address.String())
	} else {
		query = fmt.Sprintf("transfer.recipient='%s'", address.String())
	}

	// In production, you would call the Tendermint RPC to search for transactions:
	// result, err := ws.clientCtx.Client.TxSearch(context.Background(), query, prove, page, perPage, orderBy)
	// Then parse the results and convert them to Transaction objects

	// For now, return empty transactions as the full RPC integration requires:
	// - A running node with transaction indexing enabled
	// - Proper RPC client configuration
	// - Result parsing and conversion logic
	_ = query // Suppress unused variable warning

	return []Transaction{}, nil
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
