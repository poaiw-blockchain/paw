package types

import (
	"time"
)

// EventType represents the type of audit event
type EventType string

const (
	// Authentication events
	EventTypeLogin            EventType = "auth.login"
	EventTypeLogout           EventType = "auth.logout"
	EventTypeLoginFailed      EventType = "auth.login_failed"
	EventTypePasswordChanged  EventType = "auth.password_changed"
	EventTypeTokenRefreshed   EventType = "auth.token_refreshed"
	EventTypeSessionExpired   EventType = "auth.session_expired"

	// Parameter changes
	EventTypeParamUpdate       EventType = "param.update"
	EventTypeParamBulkUpdate   EventType = "param.bulk_update"
	EventTypeParamReset        EventType = "param.reset"

	// Circuit breaker operations
	EventTypeCircuitPause      EventType = "circuit.pause"
	EventTypeCircuitResume     EventType = "circuit.resume"
	EventTypeCircuitTriggered  EventType = "circuit.triggered"

	// Emergency operations
	EventTypeEmergencyPause    EventType = "emergency.pause"
	EventTypeEmergencyResume   EventType = "emergency.resume"
	EventTypeEmergencyAction   EventType = "emergency.action"

	// Alert management
	EventTypeAlertRuleCreated  EventType = "alert.rule_created"
	EventTypeAlertRuleUpdated  EventType = "alert.rule_updated"
	EventTypeAlertRuleDeleted  EventType = "alert.rule_deleted"
	EventTypeAlertAcknowledged EventType = "alert.acknowledged"
	EventTypeAlertResolved     EventType = "alert.resolved"

	// Network upgrades
	EventTypeUpgradeScheduled  EventType = "upgrade.scheduled"
	EventTypeUpgradeExecuted   EventType = "upgrade.executed"
	EventTypeUpgradeCancelled  EventType = "upgrade.cancelled"
	EventTypeUpgradeFailed     EventType = "upgrade.failed"

	// Access control
	EventTypeRoleAssigned      EventType = "access.role_assigned"
	EventTypeRoleRevoked       EventType = "access.role_revoked"
	EventTypePermissionGranted EventType = "access.permission_granted"
	EventTypePermissionRevoked EventType = "access.permission_revoked"

	// System operations
	EventTypeSystemRestart     EventType = "system.restart"
	EventTypeSystemShutdown    EventType = "system.shutdown"
	EventTypeConfigUpdate      EventType = "system.config_update"
	EventTypeDatabaseMigration EventType = "system.db_migration"
)

// Result represents the outcome of an audited action
type Result string

const (
	ResultSuccess Result = "success"
	ResultFailure Result = "failure"
	ResultPartial Result = "partial"
)

// Severity represents the importance of an audit event
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// AuditLogEntry represents a complete audit log entry
type AuditLogEntry struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	EventType     EventType              `json:"event_type"`
	UserID        string                 `json:"user_id"`
	UserEmail     string                 `json:"user_email"`
	UserRole      string                 `json:"user_role"`
	Action        string                 `json:"action"`
	Resource      string                 `json:"resource"`
	ResourceID    string                 `json:"resource_id"`
	Changes       map[string]interface{} `json:"changes,omitempty"`
	PreviousValue interface{}            `json:"previous_value,omitempty"`
	NewValue      interface{}            `json:"new_value,omitempty"`
	IPAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent"`
	SessionID     string                 `json:"session_id"`
	Result        Result                 `json:"result"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Severity      Severity               `json:"severity"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Hash          string                 `json:"hash"`           // Cryptographic hash
	PreviousHash  string                 `json:"previous_hash"`  // Hash chain
}

// QueryFilters represents filters for querying audit logs
type QueryFilters struct {
	EventType  []EventType `json:"event_type,omitempty"`
	UserID     string      `json:"user_id,omitempty"`
	UserEmail  string      `json:"user_email,omitempty"`
	Action     string      `json:"action,omitempty"`
	Resource   string      `json:"resource,omitempty"`
	ResourceID string      `json:"resource_id,omitempty"`
	Result     Result      `json:"result,omitempty"`
	Severity   Severity    `json:"severity,omitempty"`
	StartTime  time.Time   `json:"start_time,omitempty"`
	EndTime    time.Time   `json:"end_time,omitempty"`
	SearchText string      `json:"search_text,omitempty"`
	Limit      int         `json:"limit,omitempty"`
	Offset     int         `json:"offset,omitempty"`
	SortBy     string      `json:"sort_by,omitempty"`
	SortOrder  string      `json:"sort_order,omitempty"`
}

// AuditStats represents aggregated statistics
type AuditStats struct {
	TotalEvents       int64            `json:"total_events"`
	EventsByType      map[EventType]int64 `json:"events_by_type"`
	EventsByUser      map[string]int64 `json:"events_by_user"`
	EventsByResult    map[Result]int64 `json:"events_by_result"`
	EventsBySeverity  map[Severity]int64 `json:"events_by_severity"`
	SuccessRate       float64          `json:"success_rate"`
	FailureRate       float64          `json:"failure_rate"`
	TopUsers          []UserActivity   `json:"top_users"`
	TopActions        []ActionCount    `json:"top_actions"`
	TimeRange         TimeRange        `json:"time_range"`
}

// UserActivity represents user activity statistics
type UserActivity struct {
	UserID    string `json:"user_id"`
	UserEmail string `json:"user_email"`
	Count     int64  `json:"count"`
	LastSeen  time.Time `json:"last_seen"`
}

// ActionCount represents action count statistics
type ActionCount struct {
	Action string `json:"action"`
	Count  int64  `json:"count"`
}

// TimeRange represents a time range
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// TimelineEntry represents an entry in the audit timeline
type TimelineEntry struct {
	Timestamp   time.Time   `json:"timestamp"`
	EventType   EventType   `json:"event_type"`
	Action      string      `json:"action"`
	UserEmail   string      `json:"user_email"`
	Resource    string      `json:"resource"`
	Result      Result      `json:"result"`
	Description string      `json:"description"`
}

// ExportRequest represents a request to export audit logs
type ExportRequest struct {
	Filters QueryFilters `json:"filters"`
	Format  string       `json:"format"` // csv, json, xml
	Fields  []string     `json:"fields,omitempty"`
}

// IntegrityReport represents a hash chain integrity report
type IntegrityReport struct {
	Verified      bool      `json:"verified"`
	StartID       string    `json:"start_id"`
	EndID         string    `json:"end_id"`
	EntriesChecked int64    `json:"entries_checked"`
	Errors        []string  `json:"errors,omitempty"`
	CheckedAt     time.Time `json:"checked_at"`
}
