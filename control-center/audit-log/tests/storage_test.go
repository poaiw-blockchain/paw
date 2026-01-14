package tests

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/paw-chain/paw/control-center/audit-log/storage"
	"github.com/paw-chain/paw/control-center/audit-log/types"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Truncate test tables before running tests
	connString := os.Getenv("PAW_AUDITLOG_TEST_DB")
	if connString != "" {
		db, err := sql.Open("postgres", connString)
		if err == nil {
			defer db.Close()
			// Truncate all audit tables
			db.Exec("TRUNCATE TABLE audit_log CASCADE")
			db.Exec("TRUNCATE TABLE audit_log_archive CASCADE")
			db.Exec("TRUNCATE TABLE audit_integrity_checks CASCADE")
		}
	}
	os.Exit(m.Run())
}

func TestPostgresStorage_InsertAndQuery(t *testing.T) {
	t.Parallel()

	// Skip if no test database available
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	stor, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()

	// Use unique email to isolate this test
	testEmail := "insertquery@storage-test.example.com"

	// Create test entry
	entry := &types.AuditLogEntry{
		Timestamp:  time.Now().UTC(),
		EventType:  types.EventTypeParamUpdate,
		UserEmail:  testEmail,
		UserRole:   "admin",
		Action:     "Update oracle parameters",
		Resource:   "oracle",
		ResourceID: "params",
		Result:     types.ResultSuccess,
		Severity:   types.SeverityInfo,
		Hash:       "insertquery-hash",
	}

	// Insert
	err := stor.Insert(ctx, entry)
	require.NoError(t, err)
	require.NotEmpty(t, entry.ID)

	// Query
	filters := types.QueryFilters{
		UserEmail: testEmail,
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
	// Cannot run in parallel - depends on specific data counts
	// t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	stor, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()

	// Use unique emails that won't collide with other tests
	user1Email := "qf-user1@queryfilters.test"
	user2Email := "qf-user2@queryfilters.test"

	// Insert multiple entries with unique emails
	entries := []*types.AuditLogEntry{
		{
			Timestamp: time.Now().Add(-2 * time.Hour).UTC(),
			EventType: types.EventTypeLogin,
			UserEmail: user1Email,
			Action:    "Login",
			Result:    types.ResultSuccess,
			Severity:  types.SeverityInfo,
			Hash:      "qf-hash1",
		},
		{
			Timestamp: time.Now().Add(-1 * time.Hour).UTC(),
			EventType: types.EventTypeParamUpdate,
			UserEmail: user2Email,
			Action:    "Update params",
			Result:    types.ResultFailure,
			Severity:  types.SeverityCritical,
			Hash:      "qf-hash2",
		},
		{
			Timestamp: time.Now().UTC(),
			EventType: types.EventTypeLogin,
			UserEmail: user1Email,
			Action:    "Login",
			Result:    types.ResultSuccess,
			Severity:  types.SeverityInfo,
			Hash:      "qf-hash3",
		},
	}

	for _, entry := range entries {
		err := stor.Insert(ctx, entry)
		require.NoError(t, err)
	}

	// Test filter by event type AND user email
	filters := types.QueryFilters{
		EventType: []types.EventType{types.EventTypeLogin},
		UserEmail: user1Email,
		Limit:     10,
	}
	results, total, err := stor.Query(ctx, filters)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, results, 2)

	// Test filter by result AND user email
	filters = types.QueryFilters{
		Result:    types.ResultFailure,
		UserEmail: user2Email,
		Limit:     10,
	}
	results, total, err = stor.Query(ctx, filters)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, results, 1)

	// Test time range filter with user email
	filters = types.QueryFilters{
		StartTime: time.Now().Add(-90 * time.Minute),
		EndTime:   time.Now().Add(-30 * time.Minute),
		UserEmail: user2Email,
		Limit:     10,
	}
	results, total, err = stor.Query(ctx, filters)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(1))
	assert.NotEmpty(t, results) // Ensure results are returned
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
	connString := os.Getenv("PAW_AUDITLOG_TEST_DB")
	if connString == "" {
		t.Skip("PAW_AUDITLOG_TEST_DB is not set; skipping audit-log integration tests")
	}

	stor, err := storage.NewPostgresStorage(connString)
	require.NoError(t, err)

	cleanup := func() {
		// Clean up test data
		stor.Close()
	}

	return stor, cleanup
}
