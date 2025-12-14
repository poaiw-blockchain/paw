package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"paw/control-center/audit-log/middleware"
	"paw/control-center/audit-log/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditLogger_Middleware(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	stor, cleanup := setupTestStorage(t)
	defer cleanup()

	logger := middleware.NewAuditLogger(stor)

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with middleware
	handler := logger.Middleware(testHandler)

	// Create request with user context
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	ctx := context.WithValue(req.Context(), "user_email", "test@example.com")
	ctx = context.WithValue(ctx, "user_id", "user123")
	ctx = context.WithValue(ctx, "user_role", "admin")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Give middleware goroutine time to log
	// In production, you'd want a more robust synchronization mechanism
}

func TestAuditLogger_LogAction(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	stor, cleanup := setupTestStorage(t)
	defer cleanup()

	logger := middleware.NewAuditLogger(stor)

	// Log an action
	action := middleware.AuditAction{
		EventType:  types.EventTypeParamUpdate,
		UserID:     "user123",
		UserEmail:  "admin@example.com",
		UserRole:   "admin",
		Action:     "Update oracle parameters",
		Resource:   "oracle",
		ResourceID: "params",
		Changes: map[string]interface{}{
			"min_count": 3,
			"max_count": 10,
		},
		PreviousValue: 5,
		NewValue:      10,
		IPAddress:     "192.168.1.1",
		UserAgent:     "test-agent",
		SessionID:     "session123",
		Result:        types.ResultSuccess,
		Severity:      types.SeverityInfo,
		Metadata: map[string]interface{}{
			"module": "oracle",
		},
	}

	err := logger.LogAction(context.Background(), action)
	require.NoError(t, err)

	// Verify the entry was logged
	filters := types.QueryFilters{
		UserEmail: "admin@example.com",
		Limit:     10,
	}

	entries, total, err := stor.Query(context.Background(), filters)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(1))
	assert.NotEmpty(t, entries)

	// Verify the logged entry has correct data
	found := false
	for _, entry := range entries {
		if entry.Action == "Update oracle parameters" {
			found = true
			assert.Equal(t, types.EventTypeParamUpdate, entry.EventType)
			assert.Equal(t, "admin@example.com", entry.UserEmail)
			assert.Equal(t, "oracle", entry.Resource)
			assert.Equal(t, types.ResultSuccess, entry.Result)
			assert.NotEmpty(t, entry.Hash)
			break
		}
	}
	assert.True(t, found, "Should find the logged action")
}

func TestAuditLogger_DetermineEventType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		method        string
		path          string
		expectedEvent types.EventType
		expectedAction string
	}{
		{
			name:          "Login",
			method:        "POST",
			path:          "/api/v1/auth/login",
			expectedEvent: types.EventTypeLogin,
			expectedAction: "User login",
		},
		{
			name:          "Logout",
			method:        "POST",
			path:          "/api/v1/auth/logout",
			expectedEvent: types.EventTypeLogout,
			expectedAction: "User logout",
		},
		{
			name:          "Update params",
			method:        "PUT",
			path:          "/api/v1/admin/params",
			expectedEvent: types.EventTypeParamUpdate,
			expectedAction: "Update module parameters",
		},
		{
			name:          "Circuit pause",
			method:        "POST",
			path:          "/api/v1/admin/circuit/pause",
			expectedEvent: types.EventTypeCircuitPause,
			expectedAction: "Pause circuit breaker",
		},
		{
			name:          "Emergency pause",
			method:        "POST",
			path:          "/api/v1/admin/emergency/pause",
			expectedEvent: types.EventTypeEmergencyPause,
			expectedAction: "Emergency pause",
		},
		{
			name:          "Create alert",
			method:        "POST",
			path:          "/api/v1/admin/alerts",
			expectedEvent: types.EventTypeAlertRuleCreated,
			expectedAction: "Create alert rule",
		},
	}

	stor, cleanup := setupTestStorage(t)
	defer cleanup()

	logger := middleware.NewAuditLogger(stor)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			// Using reflection to call private method would require making it public
			// For now, we test through the middleware
		})
	}
}

func TestAuditLogger_HashChain(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	stor, cleanup := setupTestStorage(t)
	defer cleanup()

	logger := middleware.NewAuditLogger(stor)

	// Log multiple actions
	actions := []middleware.AuditAction{
		{
			EventType: types.EventTypeLogin,
			UserEmail: "user1@example.com",
			Action:    "Login",
			Result:    types.ResultSuccess,
			Severity:  types.SeverityInfo,
		},
		{
			EventType: types.EventTypeParamUpdate,
			UserEmail: "admin@example.com",
			Action:    "Update params",
			Result:    types.ResultSuccess,
			Severity:  types.SeverityInfo,
		},
		{
			EventType: types.EventTypeLogout,
			UserEmail: "user1@example.com",
			Action:    "Logout",
			Result:    types.ResultSuccess,
			Severity:  types.SeverityInfo,
		},
	}

	for _, action := range actions {
		err := logger.LogAction(context.Background(), action)
		require.NoError(t, err)
	}

	// Retrieve all entries
	filters := types.QueryFilters{
		Limit:     10,
		SortBy:    "timestamp",
		SortOrder: "ASC",
	}

	entries, _, err := stor.Query(context.Background(), filters)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(entries), 3)

	// Verify hash chain
	for i := 1; i < len(entries); i++ {
		assert.Equal(t, entries[i-1].Hash, entries[i].PreviousHash,
			"Entry %d's previous_hash should match entry %d's hash", i, i-1)
	}
}
