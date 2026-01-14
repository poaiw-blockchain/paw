package integrity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/paw-chain/paw/control-center/audit-log/types"
)

// HashCalculator provides cryptographic hash calculations
type HashCalculator struct{}

// NewHashCalculator creates a new hash calculator
func NewHashCalculator() *HashCalculator {
	return &HashCalculator{}
}

// CalculateHash computes the SHA-256 hash of an audit log entry
func (hc *HashCalculator) CalculateHash(entry *types.AuditLogEntry) (string, error) {
	// Create a canonical representation for hashing
	hashData := struct {
		Timestamp     string      `json:"timestamp"`
		EventType     string      `json:"event_type"`
		UserID        string      `json:"user_id"`
		UserEmail     string      `json:"user_email"`
		Action        string      `json:"action"`
		Resource      string      `json:"resource"`
		ResourceID    string      `json:"resource_id"`
		Changes       interface{} `json:"changes"`
		PreviousValue interface{} `json:"previous_value"`
		NewValue      interface{} `json:"new_value"`
		Result        string      `json:"result"`
		PreviousHash  string      `json:"previous_hash"`
	}{
		Timestamp:     entry.Timestamp.UTC().Format(time.RFC3339Nano),
		EventType:     string(entry.EventType),
		UserID:        entry.UserID,
		UserEmail:     entry.UserEmail,
		Action:        entry.Action,
		Resource:      entry.Resource,
		ResourceID:    entry.ResourceID,
		Changes:       entry.Changes,
		PreviousValue: entry.PreviousValue,
		NewValue:      entry.NewValue,
		Result:        string(entry.Result),
		PreviousHash:  entry.PreviousHash,
	}

	// Marshal to JSON for consistent hashing
	jsonData, err := json.Marshal(hashData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal entry for hashing: %w", err)
	}

	// Calculate SHA-256 hash
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:]), nil
}

// VerifyHash verifies that an entry's hash is correct
func (hc *HashCalculator) VerifyHash(entry *types.AuditLogEntry) (bool, error) {
	expectedHash, err := hc.CalculateHash(entry)
	if err != nil {
		return false, err
	}

	return entry.Hash == expectedHash, nil
}

// VerifyChain verifies the integrity of a chain of audit log entries
func (hc *HashCalculator) VerifyChain(entries []types.AuditLogEntry) (*types.IntegrityReport, error) {
	if len(entries) == 0 {
		return &types.IntegrityReport{
			Verified:       true,
			EntriesChecked: 0,
			CheckedAt:      time.Now(),
		}, nil
	}

	report := &types.IntegrityReport{
		Verified:       true,
		StartID:        entries[0].ID,
		EndID:          entries[len(entries)-1].ID,
		EntriesChecked: int64(len(entries)),
		Errors:         []string{},
		CheckedAt:      time.Now(),
	}

	// Verify each entry's hash
	for i := range entries {
		entry := &entries[i]

		// Verify the hash of this entry
		valid, err := hc.VerifyHash(entry)
		if err != nil {
			report.Verified = false
			report.Errors = append(report.Errors, fmt.Sprintf("Entry %s: hash calculation error: %v", entry.ID, err))
			continue
		}

		if !valid {
			report.Verified = false
			report.Errors = append(report.Errors, fmt.Sprintf("Entry %s: hash mismatch", entry.ID))
		}

		// Verify the chain link (previous_hash matches the previous entry's hash)
		if i > 0 {
			previousEntry := &entries[i-1]
			if entry.PreviousHash != previousEntry.Hash {
				report.Verified = false
				report.Errors = append(report.Errors,
					fmt.Sprintf("Entry %s: chain broken - previous_hash (%s) does not match previous entry hash (%s)",
						entry.ID, entry.PreviousHash, previousEntry.Hash))
			}
		}
	}

	return report, nil
}

// DetectTampering detects potential tampering in audit logs
func (hc *HashCalculator) DetectTampering(entries []types.AuditLogEntry) ([]TamperAlert, error) {
	alerts := []TamperAlert{}

	for i := range entries {
		entry := &entries[i]

		// Check if hash is valid
		valid, err := hc.VerifyHash(entry)
		if err != nil {
			alerts = append(alerts, TamperAlert{
				EntryID:     entry.ID,
				Timestamp:   entry.Timestamp,
				AlertType:   TamperTypeHashError,
				Description: fmt.Sprintf("Hash calculation error: %v", err),
				Severity:    types.SeverityCritical,
			})
			continue
		}

		if !valid {
			alerts = append(alerts, TamperAlert{
				EntryID:     entry.ID,
				Timestamp:   entry.Timestamp,
				AlertType:   TamperTypeHashMismatch,
				Description: "Entry hash does not match calculated hash",
				Severity:    types.SeverityCritical,
			})
		}

		// Check chain integrity
		if i > 0 {
			previousEntry := &entries[i-1]
			if entry.PreviousHash != previousEntry.Hash {
				alerts = append(alerts, TamperAlert{
					EntryID:   entry.ID,
					Timestamp: entry.Timestamp,
					AlertType: TamperTypeChainBroken,
					Description: fmt.Sprintf("Chain broken: previous_hash does not match (expected: %s, got: %s)",
						previousEntry.Hash, entry.PreviousHash),
					Severity: types.SeverityCritical,
				})
			}

			// Check for timestamp anomalies
			if entry.Timestamp.Before(previousEntry.Timestamp) {
				alerts = append(alerts, TamperAlert{
					EntryID:   entry.ID,
					Timestamp: entry.Timestamp,
					AlertType: TamperTypeTimestampAnomaly,
					Description: fmt.Sprintf("Timestamp is before previous entry (current: %s, previous: %s)",
						entry.Timestamp, previousEntry.Timestamp),
					Severity: types.SeverityWarning,
				})
			}
		}
	}

	return alerts, nil
}

// TamperType represents the type of tampering detected
type TamperType string

const (
	TamperTypeHashMismatch     TamperType = "hash_mismatch"
	TamperTypeHashError        TamperType = "hash_error"
	TamperTypeChainBroken      TamperType = "chain_broken"
	TamperTypeTimestampAnomaly TamperType = "timestamp_anomaly"
)

// TamperAlert represents a tampering alert
type TamperAlert struct {
	EntryID     string
	Timestamp   time.Time
	AlertType   TamperType
	Description string
	Severity    types.Severity
}

// CreateGenesisEntry creates the first entry in the hash chain
func (hc *HashCalculator) CreateGenesisEntry() (*types.AuditLogEntry, error) {
	entry := &types.AuditLogEntry{
		ID:           "genesis",
		Timestamp:    time.Now().UTC(),
		EventType:    "system.genesis",
		UserID:       "system",
		UserEmail:    "system@paw.local",
		UserRole:     "system",
		Action:       "Initialize audit log",
		Resource:     "audit_log",
		ResourceID:   "genesis",
		Result:       types.ResultSuccess,
		Severity:     types.SeverityInfo,
		PreviousHash: "0000000000000000000000000000000000000000000000000000000000000000",
	}

	hash, err := hc.CalculateHash(entry)
	if err != nil {
		return nil, err
	}
	entry.Hash = hash

	return entry, nil
}
