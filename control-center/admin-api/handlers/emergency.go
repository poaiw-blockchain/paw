package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/paw-chain/paw/control-center/admin-api/types"
)

// EmergencyHandler handles emergency operations
type EmergencyHandler struct {
	rpcClient   RPCClient
	auditLogger AuditLogger
	storage     EmergencyStorage
}

// EmergencyStorage interface for storing emergency actions
type EmergencyStorage interface {
	SaveEmergencyAction(action *types.EmergencyAction, result *types.OperationResult) error
	GetEmergencyHistory(limit int, offset int) ([]*types.AuditLog, error)
}

// NewEmergencyHandler creates a new emergency handler
func NewEmergencyHandler(rpcClient RPCClient, auditLogger AuditLogger, storage EmergencyStorage) *EmergencyHandler {
	return &EmergencyHandler{
		rpcClient:   rpcClient,
		auditLogger: auditLogger,
		storage:     storage,
	}
}

// PauseDEX handles POST /api/v1/admin/emergency/pause-dex
func (h *EmergencyHandler) PauseDEX(c *gin.Context) {
	h.pauseModule(c, "dex")
}

// PauseOracle handles POST /api/v1/admin/emergency/pause-oracle
func (h *EmergencyHandler) PauseOracle(c *gin.Context) {
	h.pauseModule(c, "oracle")
}

// PauseCompute handles POST /api/v1/admin/emergency/pause-compute
func (h *EmergencyHandler) PauseCompute(c *gin.Context) {
	h.pauseModule(c, "compute")
}

// ResumeAll handles POST /api/v1/admin/emergency/resume-all
func (h *EmergencyHandler) ResumeAll(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	var req struct {
		Reason    string `json:"reason" binding:"required"`
		MFACode   string `json:"mfa_code,omitempty"`
		Signature string `json:"signature,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Reason is required",
		})
		return
	}

	// Resume all modules
	modules := []string{"dex", "oracle", "compute"}
	results := make(map[string]interface{})
	var txHashes []string

	for _, module := range modules {
		txHash, err := h.resumeModuleOnChain(c.Request.Context(), module, getString(userID))
		if err != nil {
			results[module] = map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			}
		} else {
			results[module] = map[string]interface{}{
				"success": true,
				"tx_hash": txHash,
			}
			txHashes = append(txHashes, txHash)
		}
	}

	h.auditLogger.LogAction(
		getString(userID),
		getString(username),
		"emergency_resume_all",
		"emergency/resume-all",
		c.ClientIP(),
		map[string]interface{}{
			"reason":    req.Reason,
			"modules":   modules,
			"results":   results,
			"tx_hashes": txHashes,
		},
		true,
		nil,
	)

	c.JSON(http.StatusOK, types.OperationResult{
		Success:   true,
		Message:   "All modules resume requested",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"modules":   modules,
			"results":   results,
			"tx_hashes": txHashes,
		},
	})
}

// pauseModule is a helper function to pause a specific module
func (h *EmergencyHandler) pauseModule(c *gin.Context, module string) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	var req struct {
		Reason    string `json:"reason" binding:"required"`
		MFACode   string `json:"mfa_code,omitempty"`
		Signature string `json:"signature,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Reason is required for emergency operations",
		})
		return
	}

	// Emergency pause via RPC
	txHash, err := h.pauseModuleOnChain(c.Request.Context(), module, getString(userID))
	if err != nil {
		h.auditLogger.LogAction(
			getString(userID),
			getString(username),
			fmt.Sprintf("emergency_pause_%s_failed", module),
			fmt.Sprintf("emergency/pause-%s", module),
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
			"error":   "emergency_pause_failed",
			"message": err.Error(),
		})
		return
	}

	h.auditLogger.LogAction(
		getString(userID),
		getString(username),
		fmt.Sprintf("emergency_pause_%s", module),
		fmt.Sprintf("emergency/pause-%s", module),
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
		Message:   fmt.Sprintf("Emergency pause executed for %s module", module),
		TxHash:    txHash,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"module": module,
			"reason": req.Reason,
		},
	})
}

// Helper methods

func (h *EmergencyHandler) pauseModuleOnChain(ctx context.Context, module, signer string) (string, error) {
	// Get current module params
	params, err := h.rpcClient.GetModuleParams(ctx, module)
	if err != nil {
		return "", fmt.Errorf("failed to get %s params: %w", module, err)
	}

	// Set paused flag to true
	params["paused"] = true
	params["paused_at"] = time.Now().Unix()
	params["paused_by"] = signer
	params["pause_reason"] = "emergency"

	// Update params on chain
	txHash, err := h.rpcClient.UpdateModuleParams(ctx, module, params, signer)
	if err != nil {
		return "", fmt.Errorf("failed to pause %s: %w", module, err)
	}

	return txHash, nil
}

func (h *EmergencyHandler) resumeModuleOnChain(ctx context.Context, module, signer string) (string, error) {
	// Get current module params
	params, err := h.rpcClient.GetModuleParams(ctx, module)
	if err != nil {
		return "", fmt.Errorf("failed to get %s params: %w", module, err)
	}

	// Set paused flag to false
	params["paused"] = false
	params["resumed_at"] = time.Now().Unix()
	params["resumed_by"] = signer

	// Update params on chain
	txHash, err := h.rpcClient.UpdateModuleParams(ctx, module, params, signer)
	if err != nil {
		return "", fmt.Errorf("failed to resume %s: %w", module, err)
	}

	return txHash, nil
}

// UpgradeHandler handles network upgrade operations
type UpgradeHandler struct {
	rpcClient   RPCClient
	auditLogger AuditLogger
	storage     UpgradeStorage
}

// UpgradeStorage interface for storing upgrade schedules
type UpgradeStorage interface {
	SaveUpgradeSchedule(schedule *types.UpgradeSchedule) error
	GetUpgradeSchedule(name string) (*types.UpgradeSchedule, error)
	GetAllUpgradeSchedules() ([]*types.UpgradeSchedule, error)
	UpdateUpgradeStatus(name, status string) error
}

// NewUpgradeHandler creates a new upgrade handler
func NewUpgradeHandler(rpcClient RPCClient, auditLogger AuditLogger, storage UpgradeStorage) *UpgradeHandler {
	return &UpgradeHandler{
		rpcClient:   rpcClient,
		auditLogger: auditLogger,
		storage:     storage,
	}
}

// ScheduleUpgrade handles POST /api/v1/admin/upgrade/schedule
func (h *UpgradeHandler) ScheduleUpgrade(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	var schedule types.UpgradeSchedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Validate height
	currentHeight, err := h.rpcClient.GetLatestBlock(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_get_current_height",
			"message": err.Error(),
		})
		return
	}

	if schedule.Height <= currentHeight {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_height",
			"message": fmt.Sprintf("Upgrade height must be greater than current height %d", currentHeight),
		})
		return
	}

	// Set metadata
	schedule.ScheduledAt = time.Now()
	schedule.ScheduledBy = getString(username)
	schedule.Status = "pending"

	// Save schedule
	if err := h.storage.SaveUpgradeSchedule(&schedule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_save_schedule",
			"message": err.Error(),
		})
		return
	}

	h.auditLogger.LogAction(
		getString(userID),
		getString(username),
		"schedule_upgrade",
		"upgrade/schedule",
		c.ClientIP(),
		map[string]interface{}{
			"name":   schedule.Name,
			"height": schedule.Height,
			"info":   schedule.Info,
		},
		true,
		nil,
	)

	c.JSON(http.StatusOK, types.OperationResult{
		Success:   true,
		Message:   fmt.Sprintf("Upgrade '%s' scheduled for height %d", schedule.Name, schedule.Height),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"schedule": schedule,
		},
	})
}

// CancelUpgrade handles POST /api/v1/admin/upgrade/cancel
func (h *UpgradeHandler) CancelUpgrade(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	var req struct {
		Name   string `json:"name" binding:"required"`
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Get existing schedule
	schedule, err := h.storage.GetUpgradeSchedule(req.Name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "upgrade_not_found",
			"message": fmt.Sprintf("No upgrade found with name: %s", req.Name),
		})
		return
	}

	if schedule.Status != "pending" {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "cannot_cancel",
			"message": fmt.Sprintf("Cannot cancel upgrade with status: %s", schedule.Status),
		})
		return
	}

	// Update status to cancelled
	if err := h.storage.UpdateUpgradeStatus(req.Name, "cancelled"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_cancel_upgrade",
			"message": err.Error(),
		})
		return
	}

	h.auditLogger.LogAction(
		getString(userID),
		getString(username),
		"cancel_upgrade",
		"upgrade/cancel",
		c.ClientIP(),
		map[string]interface{}{
			"name":   req.Name,
			"reason": req.Reason,
		},
		true,
		nil,
	)

	c.JSON(http.StatusOK, types.OperationResult{
		Success:   true,
		Message:   fmt.Sprintf("Upgrade '%s' has been cancelled", req.Name),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"name":   req.Name,
			"reason": req.Reason,
		},
	})
}

// GetUpgradeStatus handles GET /api/v1/admin/upgrade/status
func (h *UpgradeHandler) GetUpgradeStatus(c *gin.Context) {
	name := c.Query("name")

	if name != "" {
		// Get specific upgrade
		schedule, err := h.storage.GetUpgradeSchedule(name)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "upgrade_not_found",
				"message": fmt.Sprintf("No upgrade found with name: %s", name),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"schedule": schedule,
		})
		return
	}

	// Get all upgrades
	schedules, err := h.storage.GetAllUpgradeSchedules()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed_to_fetch_upgrades",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"schedules": schedules,
		"count":     len(schedules),
	})
}
