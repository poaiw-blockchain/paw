package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/paw-chain/paw/control-center/backend/audit"
	"github.com/paw-chain/paw/control-center/backend/auth"
	"github.com/paw-chain/paw/control-center/backend/config"
	"github.com/paw-chain/paw/control-center/backend/integration"
	"github.com/paw-chain/paw/control-center/backend/websocket"
)

// Handler provides admin API handlers
type Handler struct {
	authService        *auth.Service
	auditService       *audit.Service
	integrationService *integration.Service
	wsServer           *websocket.Server
	config             *config.Config
	circuitBreakers    map[string]*CircuitBreaker
}

// CircuitBreaker represents module circuit breaker state
type CircuitBreaker struct {
	Module      string     `json:"module"`
	State       string     `json:"state"` // CLOSED, OPEN, HALF_OPEN
	Reason      string     `json:"reason"`
	TrippedAt   *time.Time `json:"tripped_at"`
	TrippedBy   string     `json:"tripped_by"`
	AutoRecover bool       `json:"auto_recover"`
	RecoverAt   *time.Time `json:"recover_at"`
}

// NewHandler creates a new admin API handler
func NewHandler(
	authService *auth.Service,
	auditService *audit.Service,
	integrationService *integration.Service,
	wsServer *websocket.Server,
	cfg *config.Config,
) *Handler {
	return &Handler{
		authService:        authService,
		auditService:       auditService,
		integrationService: integrationService,
		wsServer:           wsServer,
		config:             cfg,
		circuitBreakers: map[string]*CircuitBreaker{
			"dex":     {Module: "dex", State: "CLOSED"},
			"oracle":  {Module: "oracle", State: "CLOSED"},
			"compute": {Module: "compute", State: "CLOSED"},
		},
	}
}

// ============================================================================
// PARAMETER MANAGEMENT
// ============================================================================

// GetParams retrieves current module parameters
func (h *Handler) GetParams(c *gin.Context) {
	module := c.Param("module")

	// Validate module
	if !isValidModule(module) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid module"})
		return
	}

	// Get parameters from blockchain
	params, err := h.integrationService.GetModuleParams(module)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get params: %v", err)})
		return
	}

	c.JSON(http.StatusOK, params)
}

// UpdateParams updates module parameters
func (h *Handler) UpdateParams(c *gin.Context) {
	module := c.Param("module")
	userEmail, _ := c.Get("user_email")

	// Validate module
	if !isValidModule(module) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid module"})
		return
	}

	// Parse request body
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Update parameters on blockchain
	err := h.integrationService.UpdateModuleParams(module, params)
	if err != nil {
		// Log failure
		paramsJSON, _ := json.Marshal(params)
		h.auditService.Log(audit.Entry{
			User:       userEmail.(string),
			Action:     "update_params",
			Module:     module,
			Parameters: string(paramsJSON),
			Result:     fmt.Sprintf("failed: %v", err),
			IPAddress:  c.ClientIP(),
		})

		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update params: %v", err)})
		return
	}

	// Log success
	paramsJSON, _ := json.Marshal(params)
	h.auditService.Log(audit.Entry{
		User:       userEmail.(string),
		Action:     "update_params",
		Module:     module,
		Parameters: string(paramsJSON),
		Result:     "success",
		IPAddress:  c.ClientIP(),
	})

	// Broadcast update via WebSocket
	h.wsServer.Broadcast(websocket.Message{
		Type: "params_updated",
		Data: map[string]interface{}{
			"module": module,
			"params": params,
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Parameters updated successfully"})
}

// GetParamsHistory retrieves parameter change history from audit log
func (h *Handler) GetParamsHistory(c *gin.Context) {
	filters := audit.Filters{
		Action: "update_params",
		Limit:  100,
	}

	// Optional module filter
	if module := c.Query("module"); module != "" {
		filters.Module = module
	}

	entries, total, err := h.auditService.Get(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"total":   total,
	})
}

// ============================================================================
// CIRCUIT BREAKER CONTROLS
// ============================================================================

// PauseModule pauses a module (opens circuit breaker)
func (h *Handler) PauseModule(c *gin.Context) {
	module := c.Param("module")
	userEmail, _ := c.Get("user_email")

	var req struct {
		Reason      string `json:"reason" binding:"required"`
		AutoRecover bool   `json:"auto_recover"`
		RecoverIn   int    `json:"recover_in"` // minutes
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Get circuit breaker
	cb, exists := h.circuitBreakers[module]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid module"})
		return
	}

	// Check if already paused
	if cb.State == "OPEN" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Module already paused"})
		return
	}

	// Update circuit breaker state
	now := time.Now()
	cb.State = "OPEN"
	cb.Reason = req.Reason
	cb.TrippedAt = &now
	cb.TrippedBy = userEmail.(string)
	cb.AutoRecover = req.AutoRecover

	if req.AutoRecover && req.RecoverIn > 0 {
		recoverAt := now.Add(time.Duration(req.RecoverIn) * time.Minute)
		cb.RecoverAt = &recoverAt
	}

	// Pause module on blockchain
	err := h.integrationService.PauseModule(module)
	if err != nil {
		h.auditService.Log(audit.Entry{
			User:       userEmail.(string),
			Action:     "pause_module",
			Module:     module,
			Parameters: req.Reason,
			Result:     fmt.Sprintf("failed: %v", err),
			IPAddress:  c.ClientIP(),
		})

		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to pause module: %v", err)})
		return
	}

	// Log success
	h.auditService.Log(audit.Entry{
		User:       userEmail.(string),
		Action:     "pause_module",
		Module:     module,
		Parameters: req.Reason,
		Result:     "success",
		IPAddress:  c.ClientIP(),
	})

	// Broadcast circuit breaker state change
	h.wsServer.Broadcast(websocket.Message{
		Type: "circuit_breaker_tripped",
		Data: cb,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":         "Module paused successfully",
		"circuit_breaker": cb,
	})
}

// ResumeModule resumes a module (closes circuit breaker)
func (h *Handler) ResumeModule(c *gin.Context) {
	module := c.Param("module")
	userEmail, _ := c.Get("user_email")

	// Get circuit breaker
	cb, exists := h.circuitBreakers[module]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid module"})
		return
	}

	// Check if paused
	if cb.State != "OPEN" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Module not paused"})
		return
	}

	// Resume module on blockchain
	err := h.integrationService.ResumeModule(module)
	if err != nil {
		h.auditService.Log(audit.Entry{
			User:      userEmail.(string),
			Action:    "resume_module",
			Module:    module,
			Result:    fmt.Sprintf("failed: %v", err),
			IPAddress: c.ClientIP(),
		})

		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to resume module: %v", err)})
		return
	}

	// Update circuit breaker state
	cb.State = "CLOSED"
	cb.Reason = ""
	cb.TrippedAt = nil
	cb.TrippedBy = ""
	cb.AutoRecover = false
	cb.RecoverAt = nil

	// Log success
	h.auditService.Log(audit.Entry{
		User:      userEmail.(string),
		Action:    "resume_module",
		Module:    module,
		Result:    "success",
		IPAddress: c.ClientIP(),
	})

	// Broadcast circuit breaker state change
	h.wsServer.Broadcast(websocket.Message{
		Type: "circuit_breaker_closed",
		Data: cb,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":         "Module resumed successfully",
		"circuit_breaker": cb,
	})
}

// GetCircuitBreakerStatus retrieves all circuit breaker statuses
func (h *Handler) GetCircuitBreakerStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.circuitBreakers)
}

// ============================================================================
// EMERGENCY CONTROLS
// ============================================================================

// HaltChain performs emergency chain halt
func (h *Handler) HaltChain(c *gin.Context) {
	userEmail, _ := c.Get("user_email")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Execute emergency halt
	err := h.integrationService.HaltChain(req.Reason)
	if err != nil {
		h.auditService.Log(audit.Entry{
			User:       userEmail.(string),
			Action:     "emergency_halt_chain",
			Parameters: req.Reason,
			Result:     fmt.Sprintf("failed: %v", err),
			IPAddress:  c.ClientIP(),
		})

		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to halt chain: %v", err)})
		return
	}

	// Log critical action
	h.auditService.Log(audit.Entry{
		User:       userEmail.(string),
		Action:     "emergency_halt_chain",
		Parameters: req.Reason,
		Result:     "success",
		IPAddress:  c.ClientIP(),
	})

	// Broadcast emergency alert
	h.wsServer.Broadcast(websocket.Message{
		Type: "emergency_alert",
		Data: map[string]interface{}{
			"action": "chain_halted",
			"reason": req.Reason,
			"user":   userEmail.(string),
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Chain halted successfully"})
}

// EnableMaintenance enables maintenance mode
func (h *Handler) EnableMaintenance(c *gin.Context) {
	userEmail, _ := c.Get("user_email")

	err := h.integrationService.EnableMaintenance()
	if err != nil {
		h.auditService.Log(audit.Entry{
			User:      userEmail.(string),
			Action:    "enable_maintenance",
			Result:    fmt.Sprintf("failed: %v", err),
			IPAddress: c.ClientIP(),
		})

		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to enable maintenance: %v", err)})
		return
	}

	h.auditService.Log(audit.Entry{
		User:      userEmail.(string),
		Action:    "enable_maintenance",
		Result:    "success",
		IPAddress: c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "Maintenance mode enabled"})
}

// ForceUpgrade triggers forced chain upgrade
func (h *Handler) ForceUpgrade(c *gin.Context) {
	userEmail, _ := c.Get("user_email")

	var req struct {
		Version string `json:"version" binding:"required"`
		Height  int64  `json:"height" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err := h.integrationService.ForceUpgrade(req.Version, req.Height)
	if err != nil {
		paramsJSON, _ := json.Marshal(req)
		h.auditService.Log(audit.Entry{
			User:       userEmail.(string),
			Action:     "force_upgrade",
			Parameters: string(paramsJSON),
			Result:     fmt.Sprintf("failed: %v", err),
			IPAddress:  c.ClientIP(),
		})

		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to force upgrade: %v", err)})
		return
	}

	paramsJSON, _ := json.Marshal(req)
	h.auditService.Log(audit.Entry{
		User:       userEmail.(string),
		Action:     "force_upgrade",
		Parameters: string(paramsJSON),
		Result:     "success",
		IPAddress:  c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "Forced upgrade scheduled"})
}

// DisableModule disables a module completely
func (h *Handler) DisableModule(c *gin.Context) {
	module := c.Param("module")
	userEmail, _ := c.Get("user_email")

	err := h.integrationService.DisableModule(module)
	if err != nil {
		h.auditService.Log(audit.Entry{
			User:      userEmail.(string),
			Action:    "disable_module",
			Module:    module,
			Result:    fmt.Sprintf("failed: %v", err),
			IPAddress: c.ClientIP(),
		})

		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to disable module: %v", err)})
		return
	}

	h.auditService.Log(audit.Entry{
		User:      userEmail.(string),
		Action:    "disable_module",
		Module:    module,
		Result:    "success",
		IPAddress: c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "Module disabled successfully"})
}

// ============================================================================
// AUDIT LOG
// ============================================================================

// GetAuditLog retrieves audit log entries
func (h *Handler) GetAuditLog(c *gin.Context) {
	filters := audit.Filters{}

	// Parse query parameters
	if user := c.Query("user"); user != "" {
		filters.User = user
	}

	if action := c.Query("action"); action != "" {
		filters.Action = action
	}

	if module := c.Query("module"); module != "" {
		filters.Module = module
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters.Limit = l
		}
	} else {
		filters.Limit = 100 // Default
	}

	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filters.Offset = o
		}
	}

	// Get audit log entries
	entries, total, err := h.auditService.Get(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get audit log"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": entries,
		"total":   total,
		"limit":   filters.Limit,
		"offset":  filters.Offset,
	})
}

// ExportAuditLog exports audit log in specified format
func (h *Handler) ExportAuditLog(c *gin.Context) {
	format := c.Query("format")
	if format == "" {
		format = "json"
	}

	filters := audit.Filters{
		Limit: 10000, // Max export
	}

	data, err := h.auditService.Export(filters, format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export audit log"})
		return
	}

	// Set appropriate content type
	var contentType string
	switch format {
	case "csv":
		contentType = "text/csv"
	default:
		contentType = "application/json"
	}

	c.Data(http.StatusOK, contentType, data)
}

// ============================================================================
// ALERT MANAGEMENT
// ============================================================================

// GetAlerts retrieves all alerts from Alertmanager
func (h *Handler) GetAlerts(c *gin.Context) {
	alerts, err := h.integrationService.GetAlerts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get alerts"})
		return
	}

	c.JSON(http.StatusOK, alerts)
}

// AcknowledgeAlert acknowledges an alert
func (h *Handler) AcknowledgeAlert(c *gin.Context) {
	alertID := c.Param("id")
	userEmail, _ := c.Get("user_email")

	err := h.integrationService.AcknowledgeAlert(alertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to acknowledge alert"})
		return
	}

	h.auditService.Log(audit.Entry{
		User:       userEmail.(string),
		Action:     "acknowledge_alert",
		Parameters: alertID,
		Result:     "success",
		IPAddress:  c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "Alert acknowledged"})
}

// ResolveAlert resolves an alert
func (h *Handler) ResolveAlert(c *gin.Context) {
	alertID := c.Param("id")
	userEmail, _ := c.Get("user_email")

	err := h.integrationService.ResolveAlert(alertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve alert"})
		return
	}

	h.auditService.Log(audit.Entry{
		User:       userEmail.(string),
		Action:     "resolve_alert",
		Parameters: alertID,
		Result:     "success",
		IPAddress:  c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "Alert resolved"})
}

// GetAlertConfig retrieves alert configuration
func (h *Handler) GetAlertConfig(c *gin.Context) {
	config, err := h.integrationService.GetAlertConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get alert config"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// ============================================================================
// USER MANAGEMENT
// ============================================================================

// ListUsers lists all users
func (h *Handler) ListUsers(c *gin.Context) {
	users := h.authService.ListUsers()
	c.JSON(http.StatusOK, gin.H{"users": users})
}

// CreateUser creates a new user
func (h *Handler) CreateUser(c *gin.Context) {
	var req struct {
		Email    string    `json:"email" binding:"required,email"`
		Password string    `json:"password" binding:"required,min=8"`
		Role     auth.Role `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err := h.authService.CreateUser(req.Email, req.Password, req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	userEmail, _ := c.Get("user_email")
	h.auditService.Log(audit.Entry{
		User:       userEmail.(string),
		Action:     "create_user",
		Parameters: fmt.Sprintf("%s (%s)", req.Email, req.Role),
		Result:     "success",
		IPAddress:  c.ClientIP(),
	})

	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully"})
}

// UpdateUser updates a user
func (h *Handler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")

	var req struct {
		Role    *auth.Role `json:"role"`
		Enabled *bool      `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// For simplicity, using email as ID
	err := h.authService.UpdateUser(userID, req.Role, req.Enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	adminEmail, _ := c.Get("user_email")
	h.auditService.Log(audit.Entry{
		User:       adminEmail.(string),
		Action:     "update_user",
		Parameters: userID,
		Result:     "success",
		IPAddress:  c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// DeleteUser deletes a user
func (h *Handler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	err := h.authService.DeleteUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	adminEmail, _ := c.Get("user_email")
	h.auditService.Log(audit.Entry{
		User:       adminEmail.(string),
		Action:     "delete_user",
		Parameters: userID,
		Result:     "success",
		IPAddress:  c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// ============================================================================
// HELPERS
// ============================================================================

func isValidModule(module string) bool {
	validModules := map[string]bool{
		"dex":     true,
		"oracle":  true,
		"compute": true,
	}
	return validModules[module]
}
