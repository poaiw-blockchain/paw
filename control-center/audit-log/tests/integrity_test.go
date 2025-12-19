package tests

import (
	"testing"
	"time"
	"strings"

	"github.com/paw-chain/paw/control-center/audit-log/integrity"
	"github.com/paw-chain/paw/control-center/audit-log/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashCalculator_CalculateHash(t *testing.T) {
	t.Parallel()

	calc := integrity.NewHashCalculator()

	entry := &types.AuditLogEntry{
		Timestamp:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
		EventType:    types.EventTypeLogin,
		UserID:       "user123",
		UserEmail:    "test@example.com",
		Action:       "Login",
		Resource:     "auth",
		Result:       types.ResultSuccess,
		PreviousHash: "0000000000000000000000000000000000000000000000000000000000000000",
	}

	hash1, err := calc.CalculateHash(entry)
	require.NoError(t, err)
	assert.NotEmpty(t, hash1)
	assert.Len(t, hash1, 64) // SHA-256 produces 64 hex characters

	// Same entry should produce same hash
	hash2, err := calc.CalculateHash(entry)
	require.NoError(t, err)
	assert.Equal(t, hash1, hash2)

	// Different entry should produce different hash
	entry.Action = "Logout"
	hash3, err := calc.CalculateHash(entry)
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash3)
}

func TestHashCalculator_VerifyHash(t *testing.T) {
	t.Parallel()

	calc := integrity.NewHashCalculator()

	entry := &types.AuditLogEntry{
		Timestamp:    time.Now().UTC(),
		EventType:    types.EventTypeLogin,
		UserEmail:    "test@example.com",
		Action:       "Login",
		Result:       types.ResultSuccess,
		PreviousHash: "0000000000000000000000000000000000000000000000000000000000000000",
	}

	// Calculate correct hash
	hash, err := calc.CalculateHash(entry)
	require.NoError(t, err)
	entry.Hash = hash

	// Verify should pass
	valid, err := calc.VerifyHash(entry)
	require.NoError(t, err)
	assert.True(t, valid)

	// Tamper with the entry
	entry.Action = "Tampered"

	// Verify should fail
	valid, err = calc.VerifyHash(entry)
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestHashCalculator_VerifyChain(t *testing.T) {
	t.Parallel()

	calc := integrity.NewHashCalculator()

	// Create a valid chain of entries
	entries := make([]types.AuditLogEntry, 3)

	// Genesis entry
	entries[0] = types.AuditLogEntry{
		ID:           "1",
		Timestamp:    time.Now().UTC(),
		EventType:    types.EventTypeLogin,
		UserEmail:    "user1@example.com",
		Action:       "Login",
		Result:       types.ResultSuccess,
		PreviousHash: "0000000000000000000000000000000000000000000000000000000000000000",
	}
	hash0, _ := calc.CalculateHash(&entries[0])
	entries[0].Hash = hash0

	// Second entry
	entries[1] = types.AuditLogEntry{
		ID:           "2",
		Timestamp:    time.Now().UTC(),
		EventType:    types.EventTypeParamUpdate,
		UserEmail:    "admin@example.com",
		Action:       "Update params",
		Result:       types.ResultSuccess,
		PreviousHash: hash0,
	}
	hash1, _ := calc.CalculateHash(&entries[1])
	entries[1].Hash = hash1

	// Third entry
	entries[2] = types.AuditLogEntry{
		ID:           "3",
		Timestamp:    time.Now().UTC(),
		EventType:    types.EventTypeLogout,
		UserEmail:    "user1@example.com",
		Action:       "Logout",
		Result:       types.ResultSuccess,
		PreviousHash: hash1,
	}
	hash2, _ := calc.CalculateHash(&entries[2])
	entries[2].Hash = hash2

	// Verify chain - should pass
	report, err := calc.VerifyChain(entries)
	require.NoError(t, err)
	assert.True(t, report.Verified)
	assert.Empty(t, report.Errors)
	assert.Equal(t, int64(3), report.EntriesChecked)

	// Break the chain
	entries[2].PreviousHash = "wronghash"

	// Verify chain - should fail
	report, err = calc.VerifyChain(entries)
	require.NoError(t, err)
	assert.False(t, report.Verified)
	assert.NotEmpty(t, report.Errors)
}

func TestHashCalculator_DetectTampering(t *testing.T) {
	t.Parallel()

	calc := integrity.NewHashCalculator()

	// Create entries
	entries := make([]types.AuditLogEntry, 3)

	entries[0] = types.AuditLogEntry{
		ID:           "1",
		Timestamp:    time.Now().Add(-2 * time.Hour).UTC(),
		EventType:    types.EventTypeLogin,
		UserEmail:    "test@example.com",
		Action:       "Login",
		Result:       types.ResultSuccess,
		PreviousHash: "0000000000000000000000000000000000000000000000000000000000000000",
	}
	hash0, _ := calc.CalculateHash(&entries[0])
	entries[0].Hash = hash0

	entries[1] = types.AuditLogEntry{
		ID:           "2",
		Timestamp:    time.Now().Add(-1 * time.Hour).UTC(),
		EventType:    types.EventTypeParamUpdate,
		UserEmail:    "admin@example.com",
		Action:       "Update",
		Result:       types.ResultSuccess,
		PreviousHash: hash0,
	}
	hash1, _ := calc.CalculateHash(&entries[1])
	entries[1].Hash = hash1

	entries[2] = types.AuditLogEntry{
		ID:           "3",
		Timestamp:    time.Now().UTC(),
		EventType:    types.EventTypeLogout,
		UserEmail:    "test@example.com",
		Action:       "Logout",
		Result:       types.ResultSuccess,
		PreviousHash: hash1,
	}
	hash2, _ := calc.CalculateHash(&entries[2])
	entries[2].Hash = hash2

	// No tampering - should return no alerts
	alerts, err := calc.DetectTampering(entries)
	require.NoError(t, err)
	assert.Empty(t, alerts)

	// Tamper with hash
	entries[1].Hash = "tampered_hash"

	// Should detect tampering
	alerts, err = calc.DetectTampering(entries)
	require.NoError(t, err)
	assert.NotEmpty(t, alerts)
	assert.Contains(t, alerts[0].Description, "hash")

	// Fix hash but break chain
	entries[1].Hash = hash1
	entries[2].PreviousHash = "wrong_previous_hash"
	hash2Broken, _ := calc.CalculateHash(&entries[2])
	entries[2].Hash = hash2Broken

	// Should detect chain break
	alerts, err = calc.DetectTampering(entries)
	require.NoError(t, err)
	assert.NotEmpty(t, alerts)
	assert.Contains(t, strings.ToLower(alerts[0].Description), "chain")
}

func TestHashCalculator_CreateGenesisEntry(t *testing.T) {
	t.Parallel()

	calc := integrity.NewHashCalculator()

	genesis, err := calc.CreateGenesisEntry()
	require.NoError(t, err)
	assert.NotNil(t, genesis)
	assert.Equal(t, "genesis", genesis.ID)
	assert.Equal(t, types.EventType("system.genesis"), genesis.EventType)
	assert.NotEmpty(t, genesis.Hash)
	assert.Equal(t, "0000000000000000000000000000000000000000000000000000000000000000", genesis.PreviousHash)

	// Verify genesis hash
	valid, err := calc.VerifyHash(genesis)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestHashCalculator_TimestampAnomaly(t *testing.T) {
	t.Parallel()

	calc := integrity.NewHashCalculator()

	entries := make([]types.AuditLogEntry, 2)

	// First entry
	entries[0] = types.AuditLogEntry{
		ID:           "1",
		Timestamp:    time.Now().UTC(),
		EventType:    types.EventTypeLogin,
		UserEmail:    "test@example.com",
		Action:       "Login",
		Result:       types.ResultSuccess,
		PreviousHash: "0000000000000000000000000000000000000000000000000000000000000000",
	}
	hash0, _ := calc.CalculateHash(&entries[0])
	entries[0].Hash = hash0

	// Second entry with earlier timestamp (anomaly)
	entries[1] = types.AuditLogEntry{
		ID:           "2",
		Timestamp:    time.Now().Add(-1 * time.Hour).UTC(),
		EventType:    types.EventTypeLogout,
		UserEmail:    "test@example.com",
		Action:       "Logout",
		Result:       types.ResultSuccess,
		PreviousHash: hash0,
	}
	hash1, _ := calc.CalculateHash(&entries[1])
	entries[1].Hash = hash1

	// Detect tampering - should find timestamp anomaly
	alerts, err := calc.DetectTampering(entries)
	require.NoError(t, err)
	assert.NotEmpty(t, alerts)

	// Find timestamp anomaly alert
	found := false
	for _, alert := range alerts {
		if alert.AlertType == integrity.TamperTypeTimestampAnomaly {
			found = true
			assert.Equal(t, types.SeverityWarning, alert.Severity)
			break
		}
	}
	assert.True(t, found, "Should detect timestamp anomaly")
}
