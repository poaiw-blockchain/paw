package export

import (
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/paw-chain/paw/control-center/audit-log/types"
)

// CSVExporter exports audit logs to CSV format
type CSVExporter struct{}

// NewCSVExporter creates a new CSV exporter
func NewCSVExporter() *CSVExporter {
	return &CSVExporter{}
}

// Export exports audit log entries to CSV
func (e *CSVExporter) Export(entries []types.AuditLogEntry, fields []string) ([]byte, error) {
	// Use default fields if none specified
	if len(fields) == 0 {
		fields = []string{
			"id", "timestamp", "event_type", "user_email", "action",
			"resource", "result", "severity", "ip_address",
		}
	}

	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// Write header
	if err := writer.Write(fields); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write rows
	for _, entry := range entries {
		row := make([]string, len(fields))
		for i, field := range fields {
			row[i] = e.getFieldValue(entry, field)
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return []byte(buf.String()), nil
}

// getFieldValue extracts a field value from an audit log entry
func (e *CSVExporter) getFieldValue(entry types.AuditLogEntry, field string) string {
	switch field {
	case "id":
		return entry.ID
	case "timestamp":
		return entry.Timestamp.Format(time.RFC3339)
	case "event_type":
		return string(entry.EventType)
	case "user_id":
		return entry.UserID
	case "user_email":
		return entry.UserEmail
	case "user_role":
		return entry.UserRole
	case "action":
		return entry.Action
	case "resource":
		return entry.Resource
	case "resource_id":
		return entry.ResourceID
	case "result":
		return string(entry.Result)
	case "severity":
		return string(entry.Severity)
	case "ip_address":
		return entry.IPAddress
	case "user_agent":
		return entry.UserAgent
	case "session_id":
		return entry.SessionID
	case "error_message":
		return entry.ErrorMessage
	case "hash":
		return entry.Hash
	case "previous_hash":
		return entry.PreviousHash
	default:
		return ""
	}
}
