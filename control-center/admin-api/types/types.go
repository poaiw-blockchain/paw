package types

import (
	"time"
)

// Role represents user roles for RBAC
type Role string

const (
	RoleAdmin     Role = "admin"
	RoleOperator  Role = "operator"
	RoleReadOnly  Role = "readonly"
	RoleSuperUser Role = "superuser" // For critical operations
)

// User represents an admin user
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Role         Role      `json:"role"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	LastLoginAt  time.Time `json:"last_login_at,omitempty"`
	RequiresMFA  bool      `json:"requires_mfa"`
	PasswordHash string    `json:"-"` // Never expose in JSON
}

// Session represents an authenticated session
type Session struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	Role      Role      `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
	IPAddress string    `json:"ip_address"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    string                 `json:"user_id"`
	Username  string                 `json:"username"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Details   map[string]interface{} `json:"details"`
	IPAddress string                 `json:"ip_address"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
}

// ParamUpdate represents a parameter update request
type ParamUpdate struct {
	Module string                 `json:"module" binding:"required"`
	Params map[string]interface{} `json:"params" binding:"required"`
	Reason string                 `json:"reason" binding:"required"`
}

// ParamHistoryEntry represents a historical parameter change
type ParamHistoryEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Module    string                 `json:"module"`
	Param     string                 `json:"param"`
	OldValue  interface{}            `json:"old_value"`
	NewValue  interface{}            `json:"new_value"`
	ChangedBy string                 `json:"changed_by"`
	Reason    string                 `json:"reason"`
	TxHash    string                 `json:"tx_hash,omitempty"`
}

// CircuitBreakerStatus represents the status of a module's circuit breaker
type CircuitBreakerStatus struct {
	Module    string    `json:"module"`
	Paused    bool      `json:"paused"`
	PausedAt  time.Time `json:"paused_at,omitempty"`
	PausedBy  string    `json:"paused_by,omitempty"`
	Reason    string    `json:"reason,omitempty"`
	AutoResume bool     `json:"auto_resume"`
}

// EmergencyAction represents an emergency action request
type EmergencyAction struct {
	Action    string `json:"action" binding:"required"`
	Module    string `json:"module,omitempty"`
	Reason    string `json:"reason" binding:"required"`
	MFACode   string `json:"mfa_code,omitempty"`
	Signature string `json:"signature,omitempty"` // For multi-sig operations
}

// UpgradeSchedule represents a scheduled network upgrade
type UpgradeSchedule struct {
	Name        string    `json:"name" binding:"required"`
	Height      int64     `json:"height" binding:"required"`
	Info        string    `json:"info"`
	ScheduledAt time.Time `json:"scheduled_at"`
	ScheduledBy string    `json:"scheduled_by"`
	Status      string    `json:"status"` // pending, active, cancelled, completed
}

// ModuleParams represents module parameters
type ModuleParams struct {
	Module      string                 `json:"module"`
	Params      map[string]interface{} `json:"params"`
	LastUpdated time.Time              `json:"last_updated"`
	UpdatedBy   string                 `json:"updated_by,omitempty"`
}

// OperationResult represents the result of an admin operation
type OperationResult struct {
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	TxHash    string                 `json:"tx_hash,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// MultiSigRequest represents a request requiring multiple signatures
type MultiSigRequest struct {
	ID          string                 `json:"id"`
	Action      string                 `json:"action"`
	Params      map[string]interface{} `json:"params"`
	RequiredSigs int                   `json:"required_sigs"`
	Signatures  []MultiSigSignature    `json:"signatures"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
	Status      string                 `json:"status"` // pending, approved, rejected, executed
}

// MultiSigSignature represents a single signature in a multi-sig request
type MultiSigSignature struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Signature string    `json:"signature"`
	SignedAt  time.Time `json:"signed_at"`
}

// RateLimitInfo represents rate limit information
type RateLimitInfo struct {
	UserID         string    `json:"user_id"`
	Endpoint       string    `json:"endpoint"`
	RequestCount   int       `json:"request_count"`
	WindowStart    time.Time `json:"window_start"`
	WindowEnd      time.Time `json:"window_end"`
	Limit          int       `json:"limit"`
	RemainingSlots int       `json:"remaining_slots"`
}

// NetworkStatus represents overall network status
type NetworkStatus struct {
	ChainID         string    `json:"chain_id"`
	LatestHeight    int64     `json:"latest_height"`
	LatestBlockTime time.Time `json:"latest_block_time"`
	ValidatorCount  int       `json:"validator_count"`
	ActiveModules   []string  `json:"active_modules"`
	PausedModules   []string  `json:"paused_modules"`
	Healthy         bool      `json:"healthy"`
}

// Permission represents a specific permission
type Permission string

const (
	PermissionReadParams     Permission = "read:params"
	PermissionUpdateParams   Permission = "update:params"
	PermissionPauseModule    Permission = "pause:module"
	PermissionResumeModule   Permission = "resume:module"
	PermissionEmergencyHalt  Permission = "emergency:halt"
	PermissionScheduleUpgrade Permission = "upgrade:schedule"
	PermissionReadAudit      Permission = "read:audit"
	PermissionManageUsers    Permission = "manage:users"
	PermissionMultiSig       Permission = "multisig:sign"
)

// RolePermissions maps roles to their permissions
var RolePermissions = map[Role][]Permission{
	RoleReadOnly: {
		PermissionReadParams,
		PermissionReadAudit,
	},
	RoleOperator: {
		PermissionReadParams,
		PermissionUpdateParams,
		PermissionPauseModule,
		PermissionResumeModule,
		PermissionReadAudit,
	},
	RoleAdmin: {
		PermissionReadParams,
		PermissionUpdateParams,
		PermissionPauseModule,
		PermissionResumeModule,
		PermissionScheduleUpgrade,
		PermissionReadAudit,
		PermissionManageUsers,
		PermissionMultiSig,
	},
	RoleSuperUser: {
		PermissionReadParams,
		PermissionUpdateParams,
		PermissionPauseModule,
		PermissionResumeModule,
		PermissionEmergencyHalt,
		PermissionScheduleUpgrade,
		PermissionReadAudit,
		PermissionManageUsers,
		PermissionMultiSig,
	},
}

// HasPermission checks if a role has a specific permission
func (r Role) HasPermission(p Permission) bool {
	perms, ok := RolePermissions[r]
	if !ok {
		return false
	}
	for _, perm := range perms {
		if perm == p {
			return true
		}
	}
	return false
}
