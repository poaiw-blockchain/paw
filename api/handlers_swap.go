package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SwapService handles atomic swap operations
type SwapService struct {
	clientCtx client.Context
	swaps     map[string]*AtomicSwap
	mu        sync.RWMutex
}

// NewSwapService creates a new swap service
func NewSwapService(clientCtx client.Context) *SwapService {
	return &SwapService{
		clientCtx: clientCtx,
		swaps:     make(map[string]*AtomicSwap),
	}
}

// handlePrepareSwap prepares an atomic swap
func (s *Server) handlePrepareSwap(c *gin.Context) {
	userAddress, exists := c.Get("address")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	var req PrepareSwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Details: err.Error(),
		})
		return
	}

	// Validate addresses
	initiatorAddr, err := sdk.AccAddressFromBech32(userAddress.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid initiator address",
			Details: err.Error(),
		})
		return
	}

	counterpartyAddr, err := sdk.AccAddressFromBech32(req.CounterpartyAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid counterparty address",
			Details: err.Error(),
		})
		return
	}

	// Generate secret and hash lock if not provided
	var secret, hashLock string
	if req.HashLock == "" {
		secretBytes := make([]byte, 32)
		if _, err := rand.Read(secretBytes); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to generate secret",
				Details: err.Error(),
			})
			return
		}
		secret = hex.EncodeToString(secretBytes)

		hash := sha256.Sum256(secretBytes)
		hashLock = hex.EncodeToString(hash[:])
	} else {
		hashLock = req.HashLock
	}

	// Set default timelock if not provided
	timeLockDuration := req.TimeLockDuration
	if timeLockDuration == 0 {
		timeLockDuration = 3600 // 1 hour default
	}

	timeLock := time.Now().Unix() + timeLockDuration
	expiresAt := time.Now().Add(time.Duration(timeLockDuration) * time.Second)

	// Create atomic swap
	swapID := generateSwapID()
	swap := &AtomicSwap{
		ID:                    swapID,
		Initiator:             initiatorAddr.String(),
		Counterparty:          counterpartyAddr.String(),
		SendAmount:            req.SendAmount,
		SendDenom:             req.SendDenom,
		ReceiveAmount:         req.ReceiveAmount,
		ReceiveDenom:          req.ReceiveDenom,
		HashLock:              hashLock,
		Secret:                "", // Keep secret private initially
		TimeLock:              timeLock,
		Status:                "pending",
		InitiatorCommitted:    false,
		CounterpartyCommitted: false,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
		ExpiresAt:             expiresAt,
	}

	// Store swap
	s.swapService.mu.Lock()
	s.swapService.swaps[swapID] = swap
	s.swapService.mu.Unlock()

	// Return response with secret only to initiator
	response := PrepareSwapResponse{
		SwapID:   swapID,
		HashLock: hashLock,
		TimeLock: timeLock,
		Status:   swap.Status,
		ExpiresAt: expiresAt,
	}

	// Include secret only if we generated it
	if secret != "" {
		response.Secret = secret
	}

	c.JSON(http.StatusCreated, response)
}

// handleCommitSwap commits to an atomic swap
func (s *Server) handleCommitSwap(c *gin.Context) {
	userAddress, exists := c.Get("address")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	var req CommitSwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Details: err.Error(),
		})
		return
	}

	// Get swap
	s.swapService.mu.Lock()
	swap, exists := s.swapService.swaps[req.SwapID]
	if !exists {
		s.swapService.mu.Unlock()
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "Swap not found",
		})
		return
	}

	// Check if swap is still valid
	if swap.Status != "pending" {
		s.swapService.mu.Unlock()
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: fmt.Sprintf("Swap is not in pending state (current: %s)", swap.Status),
		})
		return
	}

	// Check if swap has expired
	if time.Now().Unix() > swap.TimeLock {
		swap.Status = "expired"
		s.swapService.mu.Unlock()
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Swap has expired",
		})
		return
	}

	// Determine if user is initiator or counterparty
	isInitiator := swap.Initiator == userAddress.(string)
	isCounterparty := swap.Counterparty == userAddress.(string)

	if !isInitiator && !isCounterparty {
		s.swapService.mu.Unlock()
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Not a participant in this swap",
		})
		return
	}

	// Verify secret if provided (for counterparty claiming)
	if req.Secret != "" {
		secretBytes, err := hex.DecodeString(req.Secret)
		if err != nil {
			s.swapService.mu.Unlock()
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid secret format",
				Details: err.Error(),
			})
			return
		}

		hash := sha256.Sum256(secretBytes)
		computedHashLock := hex.EncodeToString(hash[:])

		if computedHashLock != swap.HashLock {
			s.swapService.mu.Unlock()
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "Secret does not match hash lock",
			})
			return
		}

		swap.Secret = req.Secret
	}

	// Update commitment status
	if isInitiator {
		swap.InitiatorCommitted = true
	} else {
		swap.CounterpartyCommitted = true
	}

	// Check if both parties have committed
	completed := swap.InitiatorCommitted && swap.CounterpartyCommitted
	if completed {
		swap.Status = "committed"
	}

	swap.UpdatedAt = time.Now()
	s.swapService.mu.Unlock()

	// In production, this would broadcast a transaction to the blockchain
	txHash := "0x" + generateOrderID()

	response := CommitSwapResponse{
		SwapID:    req.SwapID,
		TxHash:    txHash,
		Status:    swap.Status,
		Completed: completed,
	}

	if completed {
		response.Message = "Swap completed successfully"
	} else {
		response.Message = "Commitment recorded, waiting for counterparty"
	}

	c.JSON(http.StatusOK, response)
}

// handleRefundSwap processes a swap refund after timelock expiry
func (s *Server) handleRefundSwap(c *gin.Context) {
	userAddress, exists := c.Get("address")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	swapID := c.Query("swap_id")
	if swapID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Swap ID is required",
		})
		return
	}

	// Get swap
	s.swapService.mu.Lock()
	swap, exists := s.swapService.swaps[swapID]
	if !exists {
		s.swapService.mu.Unlock()
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "Swap not found",
		})
		return
	}

	// Verify user is initiator
	if swap.Initiator != userAddress.(string) {
		s.swapService.mu.Unlock()
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Only initiator can request refund",
		})
		return
	}

	// Check if timelock has expired
	if time.Now().Unix() <= swap.TimeLock {
		s.swapService.mu.Unlock()
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Cannot refund before timelock expiry",
		})
		return
	}

	// Check if swap is already committed
	if swap.Status == "committed" {
		s.swapService.mu.Unlock()
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Cannot refund committed swap",
		})
		return
	}

	// Process refund
	swap.Status = "refunded"
	swap.UpdatedAt = time.Now()
	s.swapService.mu.Unlock()

	// In production, broadcast refund transaction
	txHash := "0x" + generateOrderID()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"swap_id": swapID,
		"tx_hash": txHash,
		"status":  "refunded",
		"message": "Swap refunded successfully",
	})
}

// handleGetSwapStatus returns the status of a swap
func (s *Server) handleGetSwapStatus(c *gin.Context) {
	swapID := c.Param("swap_id")

	s.swapService.mu.RLock()
	swap, exists := s.swapService.swaps[swapID]
	s.swapService.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "Swap not found",
		})
		return
	}

	// Determine if user can commit or refund
	canCommit := swap.Status == "pending" && time.Now().Unix() <= swap.TimeLock
	canRefund := swap.Status == "pending" && time.Now().Unix() > swap.TimeLock

	var message string
	if swap.Status == "pending" {
		if !swap.InitiatorCommitted {
			message = "Waiting for initiator commitment"
		} else if !swap.CounterpartyCommitted {
			message = "Waiting for counterparty commitment"
		}
	}

	c.JSON(http.StatusOK, SwapStatusResponse{
		Swap:      swap,
		CanCommit: canCommit,
		CanRefund: canRefund,
		Message:   message,
	})
}

// handleGetMySwaps returns all swaps for the authenticated user
func (s *Server) handleGetMySwaps(c *gin.Context) {
	userAddress, exists := c.Get("address")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	s.swapService.mu.RLock()
	defer s.swapService.mu.RUnlock()

	userSwaps := make([]AtomicSwap, 0)
	for _, swap := range s.swapService.swaps {
		if swap.Initiator == userAddress.(string) || swap.Counterparty == userAddress.(string) {
			userSwaps = append(userSwaps, *swap)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"swaps": userSwaps,
		"count": len(userSwaps),
	})
}

// Helper functions

func generateSwapID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "SWAP-" + hex.EncodeToString(b)[:16]
}
