package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/paw-chain/paw/control-center/admin-api/types"
)

// ParamsHandler handles parameter management operations
type ParamsHandler struct {
	rpcClient   RPCClient
	auditLogger AuditLogger
	storage     StorageBackend
}

// RPCClient interface for interacting with the blockchain
type RPCClient interface {
	GetModuleParams(ctx context.Context, module string) (map[string]interface{}, error)
	UpdateModuleParams(ctx context.Context, module string, params map[string]interface{}, signer string) (string, error)
	GetLatestBlock(ctx context.Context) (int64, error)
}

// AuditLogger interface for logging audit events
type AuditLogger interface {
	LogAction(userID, username, action, resource, ipAddress string, details map[string]interface{}, success bool, err error)
}

// StorageBackend interface for storing parameter history
type StorageBackend interface {
	SaveParamHistory(entry *types.ParamHistoryEntry) error
	GetParamHistory(module string, limit int, offset int) ([]*types.ParamHistoryEntry, error)
	GetParamByID(id string) (*types.ParamHistoryEntry, error)
}

// NewParamsHandler creates a new parameter management handler
func NewParamsHandler(rpcClient RPCClient, auditLogger AuditLogger, storage StorageBackend) *ParamsHandler {
	return &ParamsHandler{
		rpcClient:   rpcClient,
		auditLogger: auditLogger,
		storage:     storage,
	}
}

// GetParams handles GET /api/v1/admin/params/:module
func (h *ParamsHandler) GetParams(c *gin.Context) {
	module := c.Param("module")
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	// Validate module name
	if !isValidModule(module) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_module",
			"message": fmt.Sprintf("Invalid module name: %s", module),
		})
		return
	}

	// Get current parameters from blockchain
	params, err := h.rpcClient.GetModuleParams(c.Request.Context(), module)
	if err != nil {
		h.auditLogger.LogAction(
			getString(userID),
			getString(username),
			"get_params_failed",
			fmt.Sprintf("params/%s", module),
			c.ClientIP(),
			map[string]interface{}{
				"module": module,
				"error":  err.Error(),
			},
			false,
			err,
		)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_fetch_params",
			"message": err.Error(),
		})
		return
	}

	h.auditLogger.LogAction(
		getString(userID),
		getString(username),
		"get_params",
		fmt.Sprintf("params/%s", module),
		c.ClientIP(),
		map[string]interface{}{
			"module": module,
		},
		true,
		nil,
	)

	c.JSON(http.StatusOK, gin.H{
		"module": module,
		"params": params,
		"timestamp": time.Now(),
	})
}

// UpdateParams handles POST /api/v1/admin/params/:module
func (h *ParamsHandler) UpdateParams(c *gin.Context) {
	module := c.Param("module")
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	// Validate module name
	if !isValidModule(module) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_module",
			"message": fmt.Sprintf("Invalid module name: %s", module),
		})
		return
	}

	var req types.ParamUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Validate reason is provided
	if req.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_error",
			"message": "Reason is required for parameter updates",
		})
		return
	}

	// Get current parameters for comparison
	oldParams, err := h.rpcClient.GetModuleParams(c.Request.Context(), module)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_fetch_current_params",
			"message": err.Error(),
		})
		return
	}

	// Update parameters on blockchain
	txHash, err := h.rpcClient.UpdateModuleParams(c.Request.Context(), module, req.Params, getString(userID))
	if err != nil {
		h.auditLogger.LogAction(
			getString(userID),
			getString(username),
			"update_params_failed",
			fmt.Sprintf("params/%s", module),
			c.ClientIP(),
			map[string]interface{}{
				"module": module,
				"params": req.Params,
				"reason": req.Reason,
				"error":  err.Error(),
			},
			false,
			err,
		)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_update_params",
			"message": err.Error(),
		})
		return
	}

	// Store parameter history
	for paramName, newValue := range req.Params {
		oldValue := oldParams[paramName]

		historyEntry := &types.ParamHistoryEntry{
			ID:        fmt.Sprintf("%s-%s-%d", module, paramName, time.Now().Unix()),
			Timestamp: time.Now(),
			Module:    module,
			Param:     paramName,
			OldValue:  oldValue,
			NewValue:  newValue,
			ChangedBy: getString(username),
			Reason:    req.Reason,
			TxHash:    txHash,
		}

		if err := h.storage.SaveParamHistory(historyEntry); err != nil {
			// Log error but don't fail the operation
			h.auditLogger.LogAction(
				getString(userID),
				getString(username),
				"save_history_failed",
				fmt.Sprintf("params/%s/%s", module, paramName),
				c.ClientIP(),
				map[string]interface{}{
					"error": err.Error(),
				},
				false,
				err,
			)
		}
	}

	h.auditLogger.LogAction(
		getString(userID),
		getString(username),
		"update_params",
		fmt.Sprintf("params/%s", module),
		c.ClientIP(),
		map[string]interface{}{
			"module":     module,
			"params":     req.Params,
			"old_params": oldParams,
			"reason":     req.Reason,
			"tx_hash":    txHash,
		},
		true,
		nil,
	)

	c.JSON(http.StatusOK, types.OperationResult{
		Success:   true,
		Message:   fmt.Sprintf("Parameters updated successfully for module %s", module),
		TxHash:    txHash,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"module":     module,
			"updated":    req.Params,
			"old_values": oldParams,
		},
	})
}

// ResetParams handles POST /api/v1/admin/params/:module/reset
func (h *ParamsHandler) ResetParams(c *gin.Context) {
	module := c.Param("module")
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	// Validate module name
	if !isValidModule(module) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_module",
			"message": fmt.Sprintf("Invalid module name: %s", module),
		})
		return
	}

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

	// Get default parameters for the module
	defaultParams := getDefaultParams(module)
	if defaultParams == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "no_defaults",
			"message": fmt.Sprintf("No default parameters defined for module %s", module),
		})
		return
	}

	// Get current parameters
	oldParams, err := h.rpcClient.GetModuleParams(c.Request.Context(), module)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_fetch_current_params",
			"message": err.Error(),
		})
		return
	}

	// Reset to defaults
	txHash, err := h.rpcClient.UpdateModuleParams(c.Request.Context(), module, defaultParams, getString(userID))
	if err != nil {
		h.auditLogger.LogAction(
			getString(userID),
			getString(username),
			"reset_params_failed",
			fmt.Sprintf("params/%s", module),
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
			"error":   "failed_to_reset_params",
			"message": err.Error(),
		})
		return
	}

	h.auditLogger.LogAction(
		getString(userID),
		getString(username),
		"reset_params",
		fmt.Sprintf("params/%s", module),
		c.ClientIP(),
		map[string]interface{}{
			"module":         module,
			"reason":         req.Reason,
			"tx_hash":        txHash,
			"old_params":     oldParams,
			"default_params": defaultParams,
		},
		true,
		nil,
	)

	c.JSON(http.StatusOK, types.OperationResult{
		Success:   true,
		Message:   fmt.Sprintf("Parameters reset to defaults for module %s", module),
		TxHash:    txHash,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"module":      module,
			"old_values":  oldParams,
			"new_values":  defaultParams,
		},
	})
}

// GetParamsHistory handles GET /api/v1/admin/params/history
func (h *ParamsHandler) GetParamsHistory(c *gin.Context) {
	module := c.Query("module")
	limit := parseIntWithDefault(c.Query("limit"), 50)
	offset := parseIntWithDefault(c.Query("offset"), 0)

	if limit > 100 {
		limit = 100
	}

	history, err := h.storage.GetParamHistory(module, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_fetch_history",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"module":  module,
		"limit":   limit,
		"offset":  offset,
	})
}

// Helper functions

func isValidModule(module string) bool {
	validModules := []string{"dex", "oracle", "compute", "staking", "gov", "slashing"}
	for _, m := range validModules {
		if m == module {
			return true
		}
	}
	return false
}

func getDefaultParams(module string) map[string]interface{} {
	// Define default parameters for each module
	defaults := map[string]map[string]interface{}{
		"dex": {
			"min_liquidity":       "1000",
			"swap_fee_rate":       "0.003",
			"max_slippage":        "0.05",
			"enable_limit_orders": true,
		},
		"oracle": {
			"voting_period":     "60",
			"vote_threshold":    "0.66",
			"slash_fraction":    "0.01",
			"min_validator_fee": "100",
		},
		"compute": {
			"min_deposit":        "1000",
			"result_timeout":     "3600",
			"verification_ratio": "0.5",
			"max_data_size":      "1048576",
		},
	}

	return defaults[module]
}

func parseIntWithDefault(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	var val int
	fmt.Sscanf(s, "%d", &val)
	if val == 0 {
		return defaultVal
	}
	return val
}

func getString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
