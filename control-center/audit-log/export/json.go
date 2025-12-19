package export

import (
	"encoding/json"
	"fmt"

	"github.com/paw-chain/paw/control-center/audit-log/types"
)

// JSONExporter exports audit logs to JSON format
type JSONExporter struct{}

// NewJSONExporter creates a new JSON exporter
func NewJSONExporter() *JSONExporter {
	return &JSONExporter{}
}

// Export exports audit log entries to JSON
func (e *JSONExporter) Export(entries []types.AuditLogEntry, fields []string) ([]byte, error) {
	// If no specific fields requested, export everything
	if len(fields) == 0 {
		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal entries to JSON: %w", err)
		}
		return data, nil
	}

	// Export only specified fields
	filtered := make([]map[string]interface{}, len(entries))
	for i, entry := range entries {
		filtered[i] = e.filterFields(entry, fields)
	}

	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filtered entries to JSON: %w", err)
	}

	return data, nil
}

// filterFields extracts only the specified fields from an entry
func (e *JSONExporter) filterFields(entry types.AuditLogEntry, fields []string) map[string]interface{} {
	result := make(map[string]interface{})

	for _, field := range fields {
		switch field {
		case "id":
			result["id"] = entry.ID
		case "timestamp":
			result["timestamp"] = entry.Timestamp
		case "event_type":
			result["event_type"] = entry.EventType
		case "user_id":
			result["user_id"] = entry.UserID
		case "user_email":
			result["user_email"] = entry.UserEmail
		case "user_role":
			result["user_role"] = entry.UserRole
		case "action":
			result["action"] = entry.Action
		case "resource":
			result["resource"] = entry.Resource
		case "resource_id":
			result["resource_id"] = entry.ResourceID
		case "changes":
			result["changes"] = entry.Changes
		case "previous_value":
			result["previous_value"] = entry.PreviousValue
		case "new_value":
			result["new_value"] = entry.NewValue
		case "ip_address":
			result["ip_address"] = entry.IPAddress
		case "user_agent":
			result["user_agent"] = entry.UserAgent
		case "session_id":
			result["session_id"] = entry.SessionID
		case "result":
			result["result"] = entry.Result
		case "error_message":
			result["error_message"] = entry.ErrorMessage
		case "severity":
			result["severity"] = entry.Severity
		case "metadata":
			result["metadata"] = entry.Metadata
		case "hash":
			result["hash"] = entry.Hash
		case "previous_hash":
			result["previous_hash"] = entry.PreviousHash
		}
	}

	return result
}
