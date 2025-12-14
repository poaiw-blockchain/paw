package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/paw-chain/paw/control-center/admin-api/types"
)

// CircuitBreakerHandler handles circuit breaker operations
type CircuitBreakerHandler struct {
	rpcClient   RPCClient
	auditLogger AuditLogger
	storage     CircuitBreakerStorage
}

// CircuitBreakerStorage interface for storing circuit breaker state
type CircuitBreakerStorage interface {
	SaveCircuitBreakerStatus(status *types.CircuitBreakerStatus) error
	GetCircuitBreakerStatus(module string) (*types.CircuitBreakerStatus, error)
	GetAllCircuitBreakerStatuses() ([]*types.CircuitBreakerStatus, error)
}

// NewCircuitBreakerHandler creates a new circuit breaker handler
func NewCircuitBreakerHandler(rpcClient RPCClient, auditLogger AuditLogger, storage CircuitBreakerStorage) *CircuitBreakerHandler {
	return &CircuitBreakerHandler{
		rpcClient:   rpcClient,
		auditLogger: auditLogger,
		storage:     storage,
	}
}

// PauseModule handles POST /api/v1/admin/circuit-breaker/:module/pause
func (h *CircuitBreakerHandler) PauseModule(c *gin.Context) {
	module := c.Param("module")
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	var req struct {
		Reason     string `json:"reason" binding:"required"`
		AutoResume bool   `json:"auto_resume"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Reason is required",
		})
		return
	}

	// Validate module name
	if !isValidModule(module) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_module",
			"message": fmt.Sprintf("Invalid module name: %s", module),
		})
		return
	}

	// Check if module is already paused
	currentStatus, err := h.storage.GetCircuitBreakerStatus(module)
	if err == nil && currentStatus != nil && currentStatus.Paused {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "already_paused",
			"message": fmt.Sprintf("Module %s is already paused", module),
			"status":  currentStatus,
		})
		return
	}

	// Pause the module via RPC
	txHash, err := h.pauseModuleOnChain(c.Request.Context(), module, getString(userID))
	if err != nil {
		h.auditLogger.LogAction(
			getString(userID),
			getString(username),
			"pause_module_failed",
			fmt.Sprintf("circuit-breaker/%s", module),
			c.ClientIP(),
			map[string]interface{}{
				"module": module,
				"reason": req.Reason,
				"error":  err.Error(),
			},
			false,
			err,
		)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_pause_module",
			"message": err.Error(),
		})
		return
	}

	// Update circuit breaker status
	status := &types.CircuitBreakerStatus{
		Module:     module,
		Paused:     true,
		PausedAt:   time.Now(),
		PausedBy:   getString(username),
		Reason:     req.Reason,
		AutoResume: req.AutoResume,
	}

	if err := h.storage.SaveCircuitBreakerStatus(status); err != nil {
		// Log error but don't fail the operation
		h.auditLogger.LogAction(
			getString(userID),
			getString(username),
			"save_circuit_status_failed",
			fmt.Sprintf("circuit-breaker/%s", module),
			c.ClientIP(),
			map[string]interface{}{
				"error": err.Error(),
			},
			false,
			err,
		)
	}

	h.auditLogger.LogAction(
		getString(userID),
		getString(username),
		"pause_module",
		fmt.Sprintf("circuit-breaker/%s", module),
		c.ClientIP(),
		map[string]interface{}{
			"module":      module,
			"reason":      req.Reason,
			"auto_resume": req.AutoResume,
			"tx_hash":     txHash,
		},
		true,
		nil,
	)

	c.JSON(http.StatusOK, types.OperationResult{
		Success:   true,
		Message:   fmt.Sprintf("Module %s has been paused", module),
		TxHash:    txHash,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"module": module,
			"status": status,
		},
	})
}

// ResumeModule handles POST /api/v1/admin/circuit-breaker/:module/resume
func (h *CircuitBreakerHandler) ResumeModule(c *gin.Context) {
	module := c.Param("module")
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Reason is required",
		})
		return
	}

	// Validate module name
	if !isValidModule(module) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_module",
			"message": fmt.Sprintf("Invalid module name: %s", module),
		})
		return
	}

	// Check if module is paused
	currentStatus, err := h.storage.GetCircuitBreakerStatus(module)
	if err != nil || currentStatus == nil || !currentStatus.Paused {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "not_paused",
			"message": fmt.Sprintf("Module %s is not currently paused", module),
		})
		return
	}

	// Resume the module via RPC
	txHash, err := h.resumeModuleOnChain(c.Request.Context(), module, getString(userID))
	if err != nil {
		h.auditLogger.LogAction(
			getString(userID),
			getString(username),
			"resume_module_failed",
			fmt.Sprintf("circuit-breaker/%s", module),
			c.ClientIP(),
			map[string]interface{}{
				"module": module,
				"reason": req.Reason,
				"error":  err.Error(),
			},
			false,
			err,
		)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_resume_module",
			"message": err.Error(),
		})
		return
	}

	// Update circuit breaker status
	status := &types.CircuitBreakerStatus{
		Module:     module,
		Paused:     false,
		AutoResume: false,
	}

	if err := h.storage.SaveCircuitBreakerStatus(status); err != nil {
		// Log error but don't fail the operation
		h.auditLogger.LogAction(
			getString(userID),
			getString(username),
			"save_circuit_status_failed",
			fmt.Sprintf("circuit-breaker/%s", module),
			c.ClientIP(),
			map[string]interface{}{
				"error": err.Error(),
			},
			false,
			err,
		)
	}

	h.auditLogger.LogAction(
		getString(userID),
		getString(username),
		"resume_module",
		fmt.Sprintf("circuit-breaker/%s", module),
		c.ClientIP(),
		map[string]interface{}{
			"module":  module,
			"reason":  req.Reason,
			"tx_hash": txHash,
		},
		true,
		nil,
	)

	c.JSON(http.StatusOK, types.OperationResult{
		Success:   true,
		Message:   fmt.Sprintf("Module %s has been resumed", module),
		TxHash:    txHash,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"module": module,
			"status": status,
		},
	})
}

// GetStatus handles GET /api/v1/admin/circuit-breaker/status
func (h *CircuitBreakerHandler) GetStatus(c *gin.Context) {
	module := c.Query("module")

	if module != "" {
		// Get status for specific module
		status, err := h.storage.GetCircuitBreakerStatus(module)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": fmt.Sprintf("No circuit breaker status found for module %s", module),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": status,
		})
		return
	}

	// Get all circuit breaker statuses
	statuses, err := h.storage.GetAllCircuitBreakerStatuses()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_fetch_statuses",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"statuses": statuses,
		"count":    len(statuses),
	})
}

// Helper methods to interact with blockchain

func (h *CircuitBreakerHandler) pauseModuleOnChain(ctx context.Context, module, signer string) (string, error) {
	// This would call the appropriate RPC method to pause the module
	// For example, using a governance proposal or circuit breaker transaction
	// Placeholder implementation
	return fmt.Sprintf("0x%s-pause-%d", module, time.Now().Unix()), nil
}

func (h *CircuitBreakerHandler) resumeModuleOnChain(ctx context.Context, module, signer string) (string, error) {
	// This would call the appropriate RPC method to resume the module
	// Placeholder implementation
	return fmt.Sprintf("0x%s-resume-%d", module, time.Now().Unix()), nil
}
