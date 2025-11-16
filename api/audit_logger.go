package api

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AuditLogger handles security event logging
type AuditLogger struct {
	logFile  *os.File
	mu       sync.Mutex
	enabled  bool
	logDir   string
	maxSize  int64 // Maximum log file size in bytes
	maxFiles int   // Maximum number of log files to keep
}

// AuditEvent represents a security audit event
type AuditEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"`
	Severity    string                 `json:"severity"` // "info", "warning", "critical"
	UserID      string                 `json:"user_id,omitempty"`
	Username    string                 `json:"username,omitempty"`
	IPAddress   string                 `json:"ip_address"`
	Action      string                 `json:"action"`
	Resource    string                 `json:"resource,omitempty"`
	Status      string                 `json:"status"` // "success", "failure", "blocked"
	Details     map[string]interface{} `json:"details,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	BlockHeight int64                  `json:"block_height,omitempty"`
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logDir string, enabled bool) (*AuditLogger, error) {
	if !enabled {
		return &AuditLogger{enabled: false}, nil
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	al := &AuditLogger{
		enabled:  true,
		logDir:   logDir,
		maxSize:  100 * 1024 * 1024, // 100 MB
		maxFiles: 10,
	}

	// Open or create audit log file
	if err := al.rotateLogFile(); err != nil {
		return nil, err
	}

	return al, nil
}

// Log logs an audit event
func (al *AuditLogger) Log(event AuditEvent) error {
	if !al.enabled {
		return nil
	}

	al.mu.Lock()
	defer al.mu.Unlock()

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Check if log rotation is needed
	if al.logFile != nil {
		info, err := al.logFile.Stat()
		if err == nil && info.Size() >= al.maxSize {
			al.rotateLogFile()
		}
	}

	// Marshal event to JSON
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	// Write to log file
	if _, err := al.logFile.Write(append(jsonData, '\n')); err != nil {
		return fmt.Errorf("failed to write audit log: %w", err)
	}

	// Flush immediately for critical events
	if event.Severity == "critical" {
		al.logFile.Sync()
	}

	return nil
}

// rotateLogFile rotates the log file when it reaches max size
func (al *AuditLogger) rotateLogFile() error {
	// Close existing file if open
	if al.logFile != nil {
		al.logFile.Close()
	}

	// Generate new log file name with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(al.logDir, fmt.Sprintf("audit_%s.log", timestamp))

	// Open new log file
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit log file: %w", err)
	}

	al.logFile = file

	// Clean up old log files
	go al.cleanupOldLogs()

	return nil
}

// cleanupOldLogs removes old log files exceeding maxFiles
func (al *AuditLogger) cleanupOldLogs() {
	files, err := filepath.Glob(filepath.Join(al.logDir, "audit_*.log"))
	if err != nil {
		return
	}

	// Sort files by name (which includes timestamp)
	if len(files) > al.maxFiles {
		// Remove oldest files
		for i := 0; i < len(files)-al.maxFiles; i++ {
			os.Remove(files[i])
		}
	}
}

// Close closes the audit logger
func (al *AuditLogger) Close() error {
	if !al.enabled || al.logFile == nil {
		return nil
	}

	al.mu.Lock()
	defer al.mu.Unlock()

	return al.logFile.Close()
}

// LogAuthentication logs authentication events
func (al *AuditLogger) LogAuthentication(c *gin.Context, username, userID, status, details string) {
	al.Log(AuditEvent{
		EventType: "authentication",
		Severity:  al.getSeverityForAuth(status),
		Username:  username,
		UserID:    userID,
		IPAddress: c.ClientIP(),
		Action:    c.Request.Method + " " + c.Request.URL.Path,
		Status:    status,
		Details: map[string]interface{}{
			"details": details,
		},
		UserAgent: c.Request.UserAgent(),
	})
}

// LogAuthorization logs authorization events
func (al *AuditLogger) LogAuthorization(c *gin.Context, userID, resource, action, status string) {
	al.Log(AuditEvent{
		EventType: "authorization",
		Severity:  al.getSeverityForStatus(status),
		UserID:    userID,
		IPAddress: c.ClientIP(),
		Action:    action,
		Resource:  resource,
		Status:    status,
		UserAgent: c.Request.UserAgent(),
	})
}

// LogTransaction logs blockchain transaction events
func (al *AuditLogger) LogTransaction(userID, txHash, txType string, amount int64, status, details string) {
	al.Log(AuditEvent{
		EventType: "transaction",
		Severity:  "info",
		UserID:    userID,
		Action:    txType,
		Status:    status,
		Details: map[string]interface{}{
			"tx_hash": txHash,
			"amount":  amount,
			"details": details,
		},
	})
}

// LogSecurityEvent logs generic security events
func (al *AuditLogger) LogSecurityEvent(eventType, severity, action, status, details string, metadata map[string]interface{}) {
	al.Log(AuditEvent{
		EventType: eventType,
		Severity:  severity,
		Action:    action,
		Status:    status,
		Details: map[string]interface{}{
			"details":  details,
			"metadata": metadata,
		},
	})
}

// LogAPIAccess logs API access events
func (al *AuditLogger) LogAPIAccess(c *gin.Context, userID string, statusCode int, duration time.Duration) {
	severity := "info"
	status := "success"

	if statusCode >= 400 && statusCode < 500 {
		severity = "warning"
		status = "client_error"
	} else if statusCode >= 500 {
		severity = "critical"
		status = "server_error"
	}

	al.Log(AuditEvent{
		EventType: "api_access",
		Severity:  severity,
		UserID:    userID,
		IPAddress: c.ClientIP(),
		Action:    c.Request.Method + " " + c.Request.URL.Path,
		Status:    status,
		Details: map[string]interface{}{
			"status_code":   statusCode,
			"duration_ms":   duration.Milliseconds(),
			"request_size":  c.Request.ContentLength,
			"response_size": c.Writer.Size(),
			"query_params":  c.Request.URL.Query(),
		},
		UserAgent: c.Request.UserAgent(),
		RequestID: c.GetString("request_id"),
	})
}

// LogRateLimitExceeded logs rate limit violations
func (al *AuditLogger) LogRateLimitExceeded(c *gin.Context, userID, limit string) {
	al.Log(AuditEvent{
		EventType: "rate_limit_exceeded",
		Severity:  "warning",
		UserID:    userID,
		IPAddress: c.ClientIP(),
		Action:    c.Request.Method + " " + c.Request.URL.Path,
		Status:    "blocked",
		Details: map[string]interface{}{
			"limit": limit,
		},
		UserAgent: c.Request.UserAgent(),
	})
}

// LogSuspiciousActivity logs suspicious activity
func (al *AuditLogger) LogSuspiciousActivity(c *gin.Context, userID, activityType, details string) {
	al.Log(AuditEvent{
		EventType: "suspicious_activity",
		Severity:  "critical",
		UserID:    userID,
		IPAddress: c.ClientIP(),
		Action:    activityType,
		Status:    "detected",
		Details: map[string]interface{}{
			"details": details,
		},
		UserAgent: c.Request.UserAgent(),
	})
}

// Helper functions

func (al *AuditLogger) getSeverityForAuth(status string) string {
	switch status {
	case "success":
		return "info"
	case "failure":
		return "warning"
	case "blocked":
		return "critical"
	default:
		return "info"
	}
}

func (al *AuditLogger) getSeverityForStatus(status string) string {
	switch status {
	case "success", "allowed":
		return "info"
	case "failure", "denied":
		return "warning"
	case "blocked":
		return "critical"
	default:
		return "info"
	}
}

// AuditMiddleware logs all API requests
func AuditMiddleware(auditLogger *AuditLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Log after request completes
		duration := time.Since(start)
		userID := c.GetString("user_id") // Set by auth middleware

		auditLogger.LogAPIAccess(c, userID, c.Writer.Status(), duration)
	}
}
