package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/paw-chain/paw/control-center/audit-log/export"
	"github.com/paw-chain/paw/control-center/audit-log/integrity"
	"github.com/paw-chain/paw/control-center/audit-log/storage"
	"github.com/paw-chain/paw/control-center/audit-log/types"

	"github.com/gorilla/mux"
)

// Handler handles audit log API requests
type Handler struct {
	storage *storage.PostgresStorage
	hashCalc *integrity.HashCalculator
}

// NewHandler creates a new API handler
func NewHandler(storage *storage.PostgresStorage) *Handler {
	return &Handler{
		storage:  storage,
		hashCalc: integrity.NewHashCalculator(),
	}
}

// RegisterRoutes registers all API routes
func (h *Handler) RegisterRoutes(r *mux.Router) {
	api := r.PathPrefix("/api/v1/audit").Subrouter()

	api.HandleFunc("/logs", h.QueryLogs).Methods("GET")
	api.HandleFunc("/logs/{id}", h.GetLog).Methods("GET")
	api.HandleFunc("/logs/search", h.SearchLogs).Methods("POST")
	api.HandleFunc("/logs/export", h.ExportLogs).Methods("POST")
	api.HandleFunc("/stats", h.GetStats).Methods("GET")
	api.HandleFunc("/timeline", h.GetTimeline).Methods("GET")
	api.HandleFunc("/user/{user_id}", h.GetUserActivity).Methods("GET")
	api.HandleFunc("/integrity/verify", h.VerifyIntegrity).Methods("POST")
	api.HandleFunc("/integrity/detect-tampering", h.DetectTampering).Methods("POST")
}

// QueryLogs handles GET /api/v1/audit/logs
func (h *Handler) QueryLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	filters := h.parseFilters(r)

	entries, total, err := h.storage.Query(ctx, filters)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to query logs", err)
		return
	}

	response := map[string]interface{}{
		"entries": entries,
		"total":   total,
		"limit":   filters.Limit,
		"offset":  filters.Offset,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// GetLog handles GET /api/v1/audit/logs/:id
func (h *Handler) GetLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	entry, err := h.storage.GetByID(ctx, id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "Log entry not found", err)
		return
	}

	h.respondJSON(w, http.StatusOK, entry)
}

// SearchLogs handles POST /api/v1/audit/logs/search
func (h *Handler) SearchLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var filters types.QueryFilters
	if err := json.NewDecoder(r.Body).Decode(&filters); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	entries, total, err := h.storage.Query(ctx, filters)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to search logs", err)
		return
	}

	response := map[string]interface{}{
		"entries": entries,
		"total":   total,
		"filters": filters,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// ExportLogs handles POST /api/v1/audit/logs/export
func (h *Handler) ExportLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req types.ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Default format to CSV if not specified
	if req.Format == "" {
		req.Format = "csv"
	}

	entries, _, err := h.storage.Query(ctx, req.Filters)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to query logs for export", err)
		return
	}

	var data []byte
	var contentType string
	var filename string

	switch req.Format {
	case "csv":
		exporter := export.NewCSVExporter()
		data, err = exporter.Export(entries, req.Fields)
		contentType = "text/csv"
		filename = "audit_logs_" + time.Now().Format("20060102_150405") + ".csv"
	case "json":
		exporter := export.NewJSONExporter()
		data, err = exporter.Export(entries, req.Fields)
		contentType = "application/json"
		filename = "audit_logs_" + time.Now().Format("20060102_150405") + ".json"
	default:
		h.respondError(w, http.StatusBadRequest, "Unsupported export format", nil)
		return
	}

	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to export logs", err)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Write(data)
}

// GetStats handles GET /api/v1/audit/stats
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse time range
	startTime, _ := parseTime(r.URL.Query().Get("start_time"))
	endTime, _ := parseTime(r.URL.Query().Get("end_time"))

	// Default to last 24 hours if not specified
	if startTime.IsZero() {
		startTime = time.Now().Add(-24 * time.Hour)
	}
	if endTime.IsZero() {
		endTime = time.Now()
	}

	stats, err := h.storage.GetStats(ctx, startTime, endTime)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to get statistics", err)
		return
	}

	h.respondJSON(w, http.StatusOK, stats)
}

// GetTimeline handles GET /api/v1/audit/timeline
func (h *Handler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	filters := h.parseFilters(r)

	timeline, err := h.storage.GetTimeline(ctx, filters)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to get timeline", err)
		return
	}

	h.respondJSON(w, http.StatusOK, timeline)
}

// GetUserActivity handles GET /api/v1/audit/user/:user_id
func (h *Handler) GetUserActivity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	userID := vars["user_id"]

	filters := h.parseFilters(r)
	filters.UserID = userID

	entries, total, err := h.storage.Query(ctx, filters)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to get user activity", err)
		return
	}

	response := map[string]interface{}{
		"user_id": userID,
		"entries": entries,
		"total":   total,
	}

	h.respondJSON(w, http.StatusOK, response)
}

// VerifyIntegrity handles POST /api/v1/audit/integrity/verify
func (h *Handler) VerifyIntegrity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		StartTime time.Time `json:"start_time"`
		EndTime   time.Time `json:"end_time"`
		Limit     int       `json:"limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Default limit
	if req.Limit == 0 {
		req.Limit = 1000
	}

	filters := types.QueryFilters{
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Limit:     req.Limit,
		SortBy:    "timestamp",
		SortOrder: "ASC",
	}

	entries, _, err := h.storage.Query(ctx, filters)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to query logs for verification", err)
		return
	}

	report, err := h.hashCalc.VerifyChain(entries)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to verify integrity", err)
		return
	}

	h.respondJSON(w, http.StatusOK, report)
}

// DetectTampering handles POST /api/v1/audit/integrity/detect-tampering
func (h *Handler) DetectTampering(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		StartTime time.Time `json:"start_time"`
		EndTime   time.Time `json:"end_time"`
		Limit     int       `json:"limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Limit == 0 {
		req.Limit = 1000
	}

	filters := types.QueryFilters{
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Limit:     req.Limit,
		SortBy:    "timestamp",
		SortOrder: "ASC",
	}

	entries, _, err := h.storage.Query(ctx, filters)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to query logs for tampering detection", err)
		return
	}

	alerts, err := h.hashCalc.DetectTampering(entries)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to detect tampering", err)
		return
	}

	response := map[string]interface{}{
		"alerts":         alerts,
		"alerts_count":   len(alerts),
		"entries_checked": len(entries),
	}

	h.respondJSON(w, http.StatusOK, response)
}

// parseFilters parses query parameters into QueryFilters
func (h *Handler) parseFilters(r *http.Request) types.QueryFilters {
	q := r.URL.Query()

	filters := types.QueryFilters{
		UserID:     q.Get("user_id"),
		UserEmail:  q.Get("user_email"),
		Action:     q.Get("action"),
		Resource:   q.Get("resource"),
		ResourceID: q.Get("resource_id"),
		SearchText: q.Get("search"),
		SortBy:     q.Get("sort_by"),
		SortOrder:  q.Get("sort_order"),
	}

	// Parse event types
	if eventTypes := q["event_type"]; len(eventTypes) > 0 {
		for _, et := range eventTypes {
			filters.EventType = append(filters.EventType, types.EventType(et))
		}
	}

	// Parse result
	if result := q.Get("result"); result != "" {
		filters.Result = types.Result(result)
	}

	// Parse severity
	if severity := q.Get("severity"); severity != "" {
		filters.Severity = types.Severity(severity)
	}

	// Parse time range
	if startTime, err := parseTime(q.Get("start_time")); err == nil {
		filters.StartTime = startTime
	}
	if endTime, err := parseTime(q.Get("end_time")); err == nil {
		filters.EndTime = endTime
	}

	// Parse pagination
	if limit, err := strconv.Atoi(q.Get("limit")); err == nil && limit > 0 {
		filters.Limit = limit
	} else {
		filters.Limit = 100
	}

	if offset, err := strconv.Atoi(q.Get("offset")); err == nil && offset >= 0 {
		filters.Offset = offset
	}

	return filters
}

// parseTime parses a time string in various formats
func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	// Try RFC3339 format first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try Unix timestamp
	if timestamp, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(timestamp, 0), nil
	}

	return time.Time{}, nil
}

// respondJSON sends a JSON response
func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response
func (h *Handler) respondError(w http.ResponseWriter, status int, message string, err error) {
	response := map[string]interface{}{
		"error":   message,
		"status":  status,
	}

	if err != nil {
		response["details"] = err.Error()
	}

	h.respondJSON(w, status, response)
}
