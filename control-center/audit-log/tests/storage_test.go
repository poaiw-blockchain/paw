package tests

import (
	"context"
	"testing"
	"time"

	"paw/control-center/audit-log/storage"
	"paw/control-center/audit-log/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresStorage_InsertAndQuery(t *testing.T) {
	t.Parallel()

	// Skip if no test database available
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	stor, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()

	// Create test entry
	entry := &types.AuditLogEntry{
		Timestamp:  time.Now().UTC(),
		EventType:  types.EventTypeParamUpdate,
		UserEmail:  "test@example.com",
		UserRole:   "admin",
		Action:     "Update oracle parameters",
		Resource:   "oracle",
		ResourceID: "params",
		Result:     types.ResultSuccess,
		Severity:   types.SeverityInfo,
		Hash:       "testhash123",
	}

	// Insert
	err := stor.Insert(ctx, entry)
	require.NoError(t, err)
	require.NotEmpty(t, entry.ID)

	// Query
	filters := types.QueryFilters{
		UserEmail: "test@example.com",
		Limit:     10,
	}

	entries, total, err := stor.Query(ctx, filters)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, entries, 1)
	assert.Equal(t, entry.ID, entries[0].ID)
	assert.Equal(t, entry.Action, entries[0].Action)
}

func TestPostgresStorage_QueryFilters(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	stor, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()

	// Insert multiple entries
	entries := []*types.AuditLogEntry{
		{
			Timestamp:  time.Now().Add(-2 * time.Hour).UTC(),
			EventType:  types.EventTypeLogin,
			UserEmail:  "user1@example.com",
			Action:     "Login",
			Result:     types.ResultSuccess,
			Severity:   types.SeverityInfo,
			Hash:       "hash1",
		},
		{
			Timestamp:  time.Now().Add(-1 * time.Hour).UTC(),
			EventType:  types.EventTypeParamUpdate,
			UserEmail:  "user2@example.com",
			Action:     "Update params",
			Result:     types.ResultFailure,
			Severity:   types.SeverityCritical,
			Hash:       "hash2",
		},
		{
			Timestamp:  time.Now().UTC(),
			EventType:  types.EventTypeLogin,
			UserEmail:  "user1@example.com",
			Action:     "Login",
			Result:     types.ResultSuccess,
			Severity:   types.SeverityInfo,
			Hash:       "hash3",
		},
	}

	for _, entry := range entries {
		err := stor.Insert(ctx, entry)
		require.NoError(t, err)
	}

	// Test filter by event type
	filters := types.QueryFilters{
		EventType: []types.EventType{types.EventTypeLogin},
		Limit:     10,
	}
	results, total, err := stor.Query(ctx, filters)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, results, 2)

	// Test filter by result
	filters = types.QueryFilters{
		Result: types.ResultFailure,
		Limit:  10,
	}
	results, total, err = stor.Query(ctx, filters)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)

	// Test time range filter
	filters = types.QueryFilters{
		StartTime: time.Now().Add(-90 * time.Minute),
		EndTime:   time.Now().Add(-30 * time.Minute),
		Limit:     10,
	}
	results, total, err = stor.Query(ctx, filters)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(1))
}

func TestPostgresStorage_GetByID(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	stor, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()

	// Insert entry
	entry := &types.AuditLogEntry{
		Timestamp: time.Now().UTC(),
		EventType: types.EventTypeLogin,
		UserEmail: "test@example.com",
		Action:    "Test action",
		Result:    types.ResultSuccess,
		Severity:  types.SeverityInfo,
		Hash:      "testhash",
	}

	err := stor.Insert(ctx, entry)
	require.NoError(t, err)

	// Get by ID
	retrieved, err := stor.GetByID(ctx, entry.ID)
	require.NoError(t, err)
	assert.Equal(t, entry.ID, retrieved.ID)
	assert.Equal(t, entry.Action, retrieved.Action)

	// Get non-existent ID
	_, err = stor.GetByID(ctx, "non-existent-id")
	assert.Error(t, err)
}

func TestPostgresStorage_GetStats(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	stor, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()

	// Insert test data
	entries := []*types.AuditLogEntry{
		{
			Timestamp: time.Now().UTC(),
			EventType: types.EventTypeLogin,
			UserEmail: "user1@example.com",
			Action:    "Login",
			Result:    types.ResultSuccess,
			Severity:  types.SeverityInfo,
			Hash:      "hash1",
		},
		{
			Timestamp: time.Now().UTC(),
			EventType: types.EventTypeLogin,
			UserEmail: "user2@example.com",
			Action:    "Login",
			Result:    types.ResultFailure,
			Severity:  types.SeverityWarning,
			Hash:      "hash2",
		},
		{
			Timestamp: time.Now().UTC(),
			EventType: types.EventTypeParamUpdate,
			UserEmail: "user1@example.com",
			Action:    "Update",
			Result:    types.ResultSuccess,
			Severity:  types.SeverityInfo,
			Hash:      "hash3",
		},
	}

	for _, entry := range entries {
		err := stor.Insert(ctx, entry)
		require.NoError(t, err)
	}

	// Get stats
	stats, err := stor.GetStats(ctx, time.Now().Add(-1*time.Hour), time.Now().Add(1*time.Hour))
	require.NoError(t, err)

	assert.GreaterOrEqual(t, stats.TotalEvents, int64(3))
	assert.GreaterOrEqual(t, stats.EventsByType[types.EventTypeLogin], int64(2))
	assert.GreaterOrEqual(t, stats.EventsByType[types.EventTypeParamUpdate], int64(1))
	assert.Greater(t, stats.SuccessRate, float64(0))
}

func TestPostgresStorage_GetTimeline(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	stor, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()

	// Insert entries
	entry := &types.AuditLogEntry{
		Timestamp: time.Now().UTC(),
		EventType: types.EventTypeLogin,
		UserEmail: "test@example.com",
		Action:    "Login",
		Resource:  "auth",
		Result:    types.ResultSuccess,
		Severity:  types.SeverityInfo,
		Hash:      "testhash",
	}

	err := stor.Insert(ctx, entry)
	require.NoError(t, err)

	// Get timeline
	filters := types.QueryFilters{
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now().Add(1 * time.Hour),
		Limit:     10,
	}

	timeline, err := stor.GetTimeline(ctx, filters)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(timeline), 1)
}

// setupTestStorage creates a test storage instance
// In a real implementation, this would set up a test database
func setupTestStorage(t *testing.T) (*storage.PostgresStorage, func()) {
	// Use test database connection string
	connString := "postgres://testuser:testpass@localhost:5432/audit_test?sslmode=disable"

	stor, err := storage.NewPostgresStorage(connString)
	require.NoError(t, err)

	cleanup := func() {
		// Clean up test data
		stor.Close()
	}

	return stor, cleanup
}
