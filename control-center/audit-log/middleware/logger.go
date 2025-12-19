package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/paw-chain/paw/control-center/audit-log/integrity"
	"github.com/paw-chain/paw/control-center/audit-log/storage"
	"github.com/paw-chain/paw/control-center/audit-log/types"
)

// AuditLogger is middleware that automatically logs all admin API requests
type AuditLogger struct {
	storage  *storage.PostgresStorage
	hashCalc *integrity.HashCalculator
}

// NewAuditLogger creates a new audit logging middleware
func NewAuditLogger(storage *storage.PostgresStorage) *AuditLogger {
	return &AuditLogger{
		storage:  storage,
		hashCalc: integrity.NewHashCalculator(),
	}
}

// Middleware wraps HTTP handlers with automatic audit logging
func (al *AuditLogger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip audit log endpoints to avoid recursion
		if strings.HasPrefix(r.URL.Path, "/api/v1/audit") {
			next.ServeHTTP(w, r)
			return
		}

		// Create response writer wrapper to capture status code
		wrapper := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Execute the actual request
		start := time.Now()
		next.ServeHTTP(wrapper, r)
		duration := time.Since(start)

		// Log the request
		go al.logRequest(r, wrapper.statusCode, duration)
	})
}

// LogAction logs a specific administrative action
func (al *AuditLogger) LogAction(ctx context.Context, action AuditAction) error {
	entry := &types.AuditLogEntry{
		Timestamp:     time.Now().UTC(),
		EventType:     action.EventType,
		UserID:        action.UserID,
		UserEmail:     action.UserEmail,
		UserRole:      action.UserRole,
		Action:        action.Action,
		Resource:      action.Resource,
		ResourceID:    action.ResourceID,
		Changes:       action.Changes,
		PreviousValue: action.PreviousValue,
		NewValue:      action.NewValue,
		IPAddress:     action.IPAddress,
		UserAgent:     action.UserAgent,
		SessionID:     action.SessionID,
		Result:        action.Result,
		ErrorMessage:  action.ErrorMessage,
		Severity:      action.Severity,
		Metadata:      action.Metadata,
	}

	// Get the hash of the most recent entry for the chain
	filters := types.QueryFilters{
		Limit:     1,
		SortBy:    "timestamp",
		SortOrder: "DESC",
	}
	entries, _, err := al.storage.Query(ctx, filters)
	if err == nil && len(entries) > 0 {
		entry.PreviousHash = entries[0].Hash
	} else {
		// Genesis entry
		entry.PreviousHash = "0000000000000000000000000000000000000000000000000000000000000000"
	}

	// Calculate hash for this entry
	hash, err := al.hashCalc.CalculateHash(entry)
	if err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}
	entry.Hash = hash

	// Insert into storage
	return al.storage.Insert(ctx, entry)
}

// logRequest logs an HTTP request
func (al *AuditLogger) logRequest(r *http.Request, statusCode int, duration time.Duration) {
	// Extract user information from context
	userEmail, _ := r.Context().Value("user_email").(string)
	userID, _ := r.Context().Value("user_id").(string)
	userRole, _ := r.Context().Value("user_role").(string)
	sessionID, _ := r.Context().Value("session_id").(string)

	// Determine event type and action based on path and method
	eventType, action := al.determineEventType(r)

	// Determine result
	result := types.ResultSuccess
	severity := types.SeverityInfo
	if statusCode >= 400 {
		result = types.ResultFailure
		if statusCode >= 500 {
			severity = types.SeverityCritical
		} else {
			severity = types.SeverityWarning
		}
	}

	auditAction := AuditAction{
		EventType: eventType,
		UserID:    userID,
		UserEmail: userEmail,
		UserRole:  userRole,
		Action:    action,
		Resource:  al.extractResource(r),
		IPAddress: al.getClientIP(r),
		UserAgent: r.UserAgent(),
		SessionID: sessionID,
		Result:    result,
		Severity:  severity,
		Metadata: map[string]interface{}{
			"method":      r.Method,
			"path":        r.URL.Path,
			"status_code": statusCode,
			"duration_ms": duration.Milliseconds(),
		},
	}

	// Log the action
	al.LogAction(r.Context(), auditAction)
}

// determineEventType determines the event type based on the request
func (al *AuditLogger) determineEventType(r *http.Request) (types.EventType, string) {
	path := r.URL.Path
	method := r.Method

	// Authentication
	if strings.Contains(path, "/auth/login") {
		return types.EventTypeLogin, "User login"
	}
	if strings.Contains(path, "/auth/logout") {
		return types.EventTypeLogout, "User logout"
	}

	// Parameters
	if strings.Contains(path, "/admin/params") {
		if method == "PUT" || method == "PATCH" {
			return types.EventTypeParamUpdate, "Update module parameters"
		}
	}

	// Circuit breaker
	if strings.Contains(path, "/admin/circuit") {
		if strings.Contains(path, "/pause") {
			return types.EventTypeCircuitPause, "Pause circuit breaker"
		}
		if strings.Contains(path, "/resume") {
			return types.EventTypeCircuitResume, "Resume circuit breaker"
		}
	}

	// Emergency
	if strings.Contains(path, "/admin/emergency") {
		if strings.Contains(path, "/pause") {
			return types.EventTypeEmergencyPause, "Emergency pause"
		}
		if strings.Contains(path, "/resume") {
			return types.EventTypeEmergencyResume, "Emergency resume"
		}
		return types.EventTypeEmergencyAction, "Emergency action"
	}

	// Alerts
	if strings.Contains(path, "/admin/alerts") {
		switch method {
		case "POST":
			return types.EventTypeAlertRuleCreated, "Create alert rule"
		case "PUT", "PATCH":
			return types.EventTypeAlertRuleUpdated, "Update alert rule"
		case "DELETE":
			return types.EventTypeAlertRuleDeleted, "Delete alert rule"
		}
	}

	// Upgrades
	if strings.Contains(path, "/admin/upgrade") {
		if method == "POST" {
			return types.EventTypeUpgradeScheduled, "Schedule network upgrade"
		}
		if method == "DELETE" {
			return types.EventTypeUpgradeCancelled, "Cancel network upgrade"
		}
	}

	// Access control
	if strings.Contains(path, "/admin/access") || strings.Contains(path, "/admin/roles") {
		if method == "POST" {
			return types.EventTypeRoleAssigned, "Assign role"
		}
		if method == "DELETE" {
			return types.EventTypeRoleRevoked, "Revoke role"
		}
	}

	// Default
	return types.EventType(fmt.Sprintf("admin.%s", strings.ToLower(method))), fmt.Sprintf("%s %s", method, path)
}

// extractResource extracts the resource name from the request path
func (al *AuditLogger) extractResource(r *http.Request) string {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) >= 3 {
		return parts[2] // Typically /api/v1/resource
	}
	return ""
}

// getClientIP extracts the client IP address
func (al *AuditLogger) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// AuditAction represents an action to be audited
type AuditAction struct {
	EventType     types.EventType
	UserID        string
	UserEmail     string
	UserRole      string
	Action        string
	Resource      string
	ResourceID    string
	Changes       map[string]interface{}
	PreviousValue interface{}
	NewValue      interface{}
	IPAddress     string
	UserAgent     string
	SessionID     string
	Result        types.Result
	ErrorMessage  string
	Severity      types.Severity
	Metadata      map[string]interface{}
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
