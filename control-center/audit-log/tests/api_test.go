package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"paw/control-center/audit-log/api"
	"paw/control-center/audit-log/storage"
	"paw/control-center/audit-log/types"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_QueryLogs(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	handler, cleanup := setupTestAPI(t)
	defer cleanup()

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/v1/audit/logs?limit=10&offset=0", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Contains(t, response, "entries")
	assert.Contains(t, response, "total")
}

func TestAPI_GetLog(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	handler, cleanup := setupTestAPI(t)
	defer cleanup()

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// First, create an entry
	stor := getTestStorage(t)
	entry := &types.AuditLogEntry{
		Timestamp: time.Now().UTC(),
		EventType: types.EventTypeLogin,
		UserEmail: "test@example.com",
		Action:    "Test action",
		Result:    types.ResultSuccess,
		Severity:  types.SeverityInfo,
		Hash:      "testhash",
	}
	err := stor.Insert(nil, entry)
	require.NoError(t, err)

	// Get the entry
	req := httptest.NewRequest("GET", "/api/v1/audit/logs/"+entry.ID, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var retrieved types.AuditLogEntry
	err = json.NewDecoder(w.Body).Decode(&retrieved)
	require.NoError(t, err)
	assert.Equal(t, entry.ID, retrieved.ID)
}

func TestAPI_SearchLogs(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	handler, cleanup := setupTestAPI(t)
	defer cleanup()

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Create search request
	filters := types.QueryFilters{
		EventType: []types.EventType{types.EventTypeLogin},
		Limit:     10,
	}

	body, _ := json.Marshal(filters)
	req := httptest.NewRequest("POST", "/api/v1/audit/logs/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Contains(t, response, "entries")
	assert.Contains(t, response, "total")
}

func TestAPI_ExportLogs_CSV(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	handler, cleanup := setupTestAPI(t)
	defer cleanup()

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Create export request
	exportReq := types.ExportRequest{
		Format: "csv",
		Filters: types.QueryFilters{
			Limit: 10,
		},
	}

	body, _ := json.Marshal(exportReq)
	req := httptest.NewRequest("POST", "/api/v1/audit/logs/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
}

func TestAPI_ExportLogs_JSON(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	handler, cleanup := setupTestAPI(t)
	defer cleanup()

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Create export request
	exportReq := types.ExportRequest{
		Format: "json",
		Filters: types.QueryFilters{
			Limit: 10,
		},
	}

	body, _ := json.Marshal(exportReq)
	req := httptest.NewRequest("POST", "/api/v1/audit/logs/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestAPI_GetStats(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	handler, cleanup := setupTestAPI(t)
	defer cleanup()

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/v1/audit/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var stats types.AuditStats
	err := json.NewDecoder(w.Body).Decode(&stats)
	require.NoError(t, err)
	assert.NotNil(t, stats.EventsByType)
}

func TestAPI_GetTimeline(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	handler, cleanup := setupTestAPI(t)
	defer cleanup()

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/v1/audit/timeline?limit=10", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var timeline []types.TimelineEntry
	err := json.NewDecoder(w.Body).Decode(&timeline)
	require.NoError(t, err)
}

func TestAPI_VerifyIntegrity(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	handler, cleanup := setupTestAPI(t)
	defer cleanup()

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Create verify request
	verifyReq := map[string]interface{}{
		"limit": 100,
	}

	body, _ := json.Marshal(verifyReq)
	req := httptest.NewRequest("POST", "/api/v1/audit/integrity/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var report types.IntegrityReport
	err := json.NewDecoder(w.Body).Decode(&report)
	require.NoError(t, err)
}

func TestAPI_DetectTampering(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	handler, cleanup := setupTestAPI(t)
	defer cleanup()

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Create tampering detection request
	detectReq := map[string]interface{}{
		"limit": 100,
	}

	body, _ := json.Marshal(detectReq)
	req := httptest.NewRequest("POST", "/api/v1/audit/integrity/detect-tampering", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Contains(t, response, "alerts")
	assert.Contains(t, response, "entries_checked")
}

// setupTestAPI creates a test API handler
func setupTestAPI(t *testing.T) (*api.Handler, func()) {
	stor, storCleanup := setupTestStorage(t)

	handler := api.NewHandler(stor)

	cleanup := func() {
		storCleanup()
	}

	return handler, cleanup
}

// getTestStorage returns a test storage instance
func getTestStorage(t *testing.T) *storage.PostgresStorage {
	connString := "postgres://testuser:testpass@localhost:5432/audit_test?sslmode=disable"
	stor, err := storage.NewPostgresStorage(connString)
	require.NoError(t, err)
	return stor
}
