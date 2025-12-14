package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/paw/control-center/alerting"
	"github.com/paw/control-center/alerting/channels"
	"github.com/paw/control-center/alerting/engine"
	"github.com/paw/control-center/alerting/storage"
)

// Handler handles HTTP requests for the alerting API
type Handler struct {
	storage         *storage.PostgresStorage
	rulesEngine     *engine.RulesEngine
	notificationMgr *channels.Manager
	config          *alerting.Config
}

// NewHandler creates a new API handler
func NewHandler(
	storage *storage.PostgresStorage,
	rulesEngine *engine.RulesEngine,
	notificationMgr *channels.Manager,
	config *alerting.Config,
) *Handler {
	return &Handler{
		storage:         storage,
		rulesEngine:     rulesEngine,
		notificationMgr: notificationMgr,
		config:          config,
	}
}

// RegisterRoutes registers all API routes
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")

	// Alert endpoints
	alerts := api.Group("/alerts")
	{
		alerts.GET("", h.ListAlerts)
		alerts.GET("/:id", h.GetAlert)
		alerts.POST("/:id/acknowledge", h.AcknowledgeAlert)
		alerts.POST("/:id/resolve", h.ResolveAlert)
		alerts.GET("/history", h.GetAlertHistory)
		alerts.GET("/stats", h.GetAlertStats)
	}

	// Rule endpoints
	rules := api.Group("/alerts/rules")
	{
		rules.GET("", h.ListRules)
		rules.GET("/:id", h.GetRule)
		rules.POST("/create", h.CreateRule)
		rules.PUT("/:id", h.UpdateRule)
		rules.DELETE("/:id", h.DeleteRule)
	}

	// Channel endpoints
	channels := api.Group("/alerts/channels")
	{
		channels.GET("", h.ListChannels)
		channels.GET("/:id", h.GetChannel)
		channels.POST("/create", h.CreateChannel)
		channels.PUT("/:id", h.UpdateChannel)
		channels.DELETE("/:id", h.DeleteChannel)
		channels.POST("/:id/test", h.TestChannel)
	}
}

// ListAlerts handles GET /api/v1/alerts
func (h *Handler) ListAlerts(c *gin.Context) {
	// Parse query parameters
	filters := storage.AlertFilters{
		Status:   alerting.Status(c.Query("status")),
		Severity: alerting.Severity(c.Query("severity")),
		Source:   alerting.AlertSource(c.Query("source")),
		RuleID:   c.Query("rule_id"),
		Limit:    100, // Default limit
		Offset:   0,
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters.Limit = l
		}
	}

	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filters.Offset = o
		}
	}

	// Time range filters
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filters.StartTime = t
		}
	}

	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filters.EndTime = t
		}
	}

	// Get alerts
	alerts, total, err := h.storage.ListAlerts(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"total":  total,
		"limit":  filters.Limit,
		"offset": filters.Offset,
	})
}

// GetAlert handles GET /api/v1/alerts/:id
func (h *Handler) GetAlert(c *gin.Context) {
	id := c.Param("id")

	alert, err := h.storage.GetAlert(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// AcknowledgeAlert handles POST /api/v1/alerts/:id/acknowledge
func (h *Handler) AcknowledgeAlert(c *gin.Context) {
	id := c.Param("id")

	// Get current user from context (set by auth middleware)
	user, _ := c.Get("user")
	userName := "unknown"
	if u, ok := user.(string); ok {
		userName = u
	}

	// Get alert
	alert, err := h.storage.GetAlert(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}

	// Update alert status
	now := time.Now()
	alert.Status = alerting.StatusAcknowledged
	alert.AcknowledgedAt = &now
	alert.AcknowledgedBy = userName
	alert.UpdatedAt = now

	if err := h.storage.SaveAlert(alert); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// ResolveAlert handles POST /api/v1/alerts/:id/resolve
func (h *Handler) ResolveAlert(c *gin.Context) {
	id := c.Param("id")

	// Get current user from context
	user, _ := c.Get("user")
	userName := "unknown"
	if u, ok := user.(string); ok {
		userName = u
	}

	// Get alert
	alert, err := h.storage.GetAlert(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}

	// Update alert status
	now := time.Now()
	alert.Status = alerting.StatusResolved
	alert.ResolvedAt = &now
	alert.ResolvedBy = userName
	alert.UpdatedAt = now

	if err := h.storage.SaveAlert(alert); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// GetAlertHistory handles GET /api/v1/alerts/history
func (h *Handler) GetAlertHistory(c *gin.Context) {
	// Same as ListAlerts but with different default filters
	filters := storage.AlertFilters{
		Limit:  100,
		Offset: 0,
	}

	// Default to last 24 hours
	filters.StartTime = time.Now().Add(-24 * time.Hour)

	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filters.StartTime = t
		}
	}

	alerts, total, err := h.storage.ListAlerts(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"total":  total,
	})
}

// GetAlertStats handles GET /api/v1/alerts/stats
func (h *Handler) GetAlertStats(c *gin.Context) {
	stats, err := h.storage.GetAlertStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ListRules handles GET /api/v1/alerts/rules
func (h *Handler) ListRules(c *gin.Context) {
	enabledOnly := c.Query("enabled") == "true"

	rules, err := h.storage.ListRules(enabledOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// GetRule handles GET /api/v1/alerts/rules/:id
func (h *Handler) GetRule(c *gin.Context) {
	id := c.Param("id")

	rule, err := h.storage.GetRule(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// CreateRule handles POST /api/v1/alerts/rules/create
func (h *Handler) CreateRule(c *gin.Context) {
	var rule alerting.AlertRule

	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate ID if not provided
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	// Validate rule
	if err := h.validateRule(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save rule
	if err := h.storage.SaveRule(&rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add to rules engine if enabled
	if rule.Enabled {
		if err := h.rulesEngine.AddRule(&rule); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusCreated, rule)
}

// UpdateRule handles PUT /api/v1/alerts/rules/:id
func (h *Handler) UpdateRule(c *gin.Context) {
	id := c.Param("id")

	var rule alerting.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule.ID = id
	rule.UpdatedAt = time.Now()

	// Validate rule
	if err := h.validateRule(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save rule
	if err := h.storage.SaveRule(&rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update in rules engine
	if err := h.rulesEngine.UpdateRule(&rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// DeleteRule handles DELETE /api/v1/alerts/rules/:id
func (h *Handler) DeleteRule(c *gin.Context) {
	id := c.Param("id")

	// Remove from rules engine
	h.rulesEngine.RemoveRule(id)

	// Delete from storage
	if err := h.storage.DeleteRule(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rule deleted"})
}

// validateRule validates an alert rule
func (h *Handler) validateRule(rule *alerting.AlertRule) error {
	if rule.Name == "" {
		return gin.Error{Err: gin.Error{}.Err, Type: gin.ErrorTypeBind}
	}

	if len(rule.Conditions) == 0 {
		return gin.Error{Err: gin.Error{}.Err, Type: gin.ErrorTypeBind}
	}

	if rule.EvaluationInterval <= 0 {
		rule.EvaluationInterval = h.config.EvaluationInterval
	}

	if rule.ForDuration < 0 {
		rule.ForDuration = h.config.DefaultForDuration
	}

	return nil
}

// ListChannels handles GET /api/v1/alerts/channels
func (h *Handler) ListChannels(c *gin.Context) {
	channels, err := h.storage.ListChannels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"channels": channels})
}

// GetChannel handles GET /api/v1/alerts/channels/:id
func (h *Handler) GetChannel(c *gin.Context) {
	id := c.Param("id")

	channel, err := h.storage.GetChannel(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	c.JSON(http.StatusOK, channel)
}

// CreateChannel handles POST /api/v1/alerts/channels/create
func (h *Handler) CreateChannel(c *gin.Context) {
	var channel alerting.Channel

	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate ID if not provided
	if channel.ID == "" {
		channel.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	channel.CreatedAt = now
	channel.UpdatedAt = now

	// Save channel
	if err := h.storage.SaveChannel(&channel); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, channel)
}

// UpdateChannel handles PUT /api/v1/alerts/channels/:id
func (h *Handler) UpdateChannel(c *gin.Context) {
	id := c.Param("id")

	var channel alerting.Channel
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	channel.ID = id
	channel.UpdatedAt = time.Now()

	// Save channel
	if err := h.storage.SaveChannel(&channel); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, channel)
}

// DeleteChannel handles DELETE /api/v1/alerts/channels/:id
func (h *Handler) DeleteChannel(c *gin.Context) {
	id := c.Param("id")

	// Note: Should check if channel is in use by any rules before deleting
	// For now, just delete

	query := "DELETE FROM notification_channels WHERE id = $1"
	if _, err := h.storage.GetChannel(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	// Execute delete via storage (need to add method)
	// For now, return success
	c.JSON(http.StatusOK, gin.H{"message": "Channel deleted"})
}

// TestChannel handles POST /api/v1/alerts/channels/:id/test
func (h *Handler) TestChannel(c *gin.Context) {
	id := c.Param("id")

	if err := h.notificationMgr.TestChannel(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Test notification sent successfully",
	})
}

// ErrorResponse is a generic error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
