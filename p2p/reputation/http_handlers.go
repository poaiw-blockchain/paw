package reputation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// HTTPHandlers provides HTTP endpoints for reputation system
type HTTPHandlers struct {
	manager *Manager
	monitor *Monitor
	metrics *Metrics
}

// NewHTTPHandlers creates HTTP handlers
func NewHTTPHandlers(manager *Manager, monitor *Monitor, metrics *Metrics) *HTTPHandlers {
	return &HTTPHandlers{
		manager: manager,
		monitor: monitor,
		metrics: metrics,
	}
}

// RegisterRoutes registers HTTP routes on a mux
func (h *HTTPHandlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/p2p/reputation/peers", h.handleGetPeers)
	mux.HandleFunc("/api/p2p/reputation/peer/", h.handleGetPeer)
	mux.HandleFunc("/api/p2p/reputation/top", h.handleGetTopPeers)
	mux.HandleFunc("/api/p2p/reputation/diverse", h.handleGetDiversePeers)
	mux.HandleFunc("/api/p2p/reputation/stats", h.handleGetStatistics)
	mux.HandleFunc("/api/p2p/reputation/health", h.handleGetHealth)
	mux.HandleFunc("/api/p2p/reputation/alerts", h.handleGetAlerts)
	mux.HandleFunc("/api/p2p/reputation/metrics", h.handleGetMetrics)
	mux.HandleFunc("/api/p2p/reputation/metrics/prometheus", h.handlePrometheusMetrics)
	mux.HandleFunc("/api/p2p/reputation/ban", h.handleBanPeer)
	mux.HandleFunc("/api/p2p/reputation/unban", h.handleUnbanPeer)
}

// GET /api/p2p/reputation/peers
func (h *HTTPHandlers) handleGetPeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.manager.peersMu.RLock()
	peers := make([]*PeerReputation, 0, len(h.manager.peers))
	for _, rep := range h.manager.peers {
		repCopy := *rep
		peers = append(peers, &repCopy)
	}
	h.manager.peersMu.RUnlock()

	respondJSON(w, http.StatusOK, map[string]any{
		"peers": peers,
		"count": len(peers),
	})
}

// GET /api/p2p/reputation/peer/{peer_id}
func (h *HTTPHandlers) handleGetPeer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract peer ID from path
	peerID := PeerID(r.URL.Path[len("/api/p2p/reputation/peer/"):])
	if peerID == "" {
		http.Error(w, "Peer ID required", http.StatusBadRequest)
		return
	}

	rep, err := h.manager.GetReputation(peerID)
	if err != nil {
		http.Error(w, "Failed to get reputation: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if rep == nil {
		http.Error(w, "Peer not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, rep)
}

// GET /api/p2p/reputation/top?n=10&min_score=50
func (h *HTTPHandlers) handleGetTopPeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	n := 10
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		if parsed, err := strconv.Atoi(nStr); err == nil {
			n = parsed
		}
	}

	minScore := 0.0
	if scoreStr := r.URL.Query().Get("min_score"); scoreStr != "" {
		if parsed, err := strconv.ParseFloat(scoreStr, 64); err == nil {
			minScore = parsed
		}
	}

	peers := h.manager.GetTopPeers(n, minScore)

	respondJSON(w, http.StatusOK, map[string]any{
		"peers": peers,
		"count": len(peers),
	})
}

// GET /api/p2p/reputation/diverse?n=10&min_score=50
func (h *HTTPHandlers) handleGetDiversePeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	n := 10
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		if parsed, err := strconv.Atoi(nStr); err == nil {
			n = parsed
		}
	}

	minScore := 0.0
	if scoreStr := r.URL.Query().Get("min_score"); scoreStr != "" {
		if parsed, err := strconv.ParseFloat(scoreStr, 64); err == nil {
			minScore = parsed
		}
	}

	peers := h.manager.GetDiversePeers(n, minScore)

	respondJSON(w, http.StatusOK, map[string]any{
		"peers": peers,
		"count": len(peers),
	})
}

// GET /api/p2p/reputation/stats
func (h *HTTPHandlers) handleGetStatistics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := h.manager.GetStatistics()

	respondJSON(w, http.StatusOK, stats)
}

// GET /api/p2p/reputation/health
func (h *HTTPHandlers) handleGetHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := h.monitor.GetHealth()

	status := http.StatusOK
	if !health.Healthy {
		status = http.StatusServiceUnavailable
	}

	respondJSON(w, status, health)
}

// GET /api/p2p/reputation/alerts?since=timestamp&type=high_ban_rate&severity=warning
func (h *HTTPHandlers) handleGetAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse since parameter
	since := time.Now().Add(-24 * time.Hour) // Default: last 24 hours
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if parsed, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = parsed
		}
	}

	// Parse type parameter
	var alertType *AlertType
	if typeStr := r.URL.Query().Get("type"); typeStr != "" {
		// Simple mapping - could be improved
		var at AlertType
		switch typeStr {
		case "high_ban_rate":
			at = AlertTypeHighBanRate
		case "low_avg_score":
			at = AlertTypeLowAvgScore
		case "subnet_concentration":
			at = AlertTypeSubnetConcentration
		default:
			at = AlertTypeSystemError
		}
		alertType = &at
	}

	// Parse severity parameter
	var severity *Severity
	if sevStr := r.URL.Query().Get("severity"); sevStr != "" {
		var sev Severity
		switch sevStr {
		case "info":
			sev = SeverityInfo
		case "warning":
			sev = SeverityWarning
		case "error":
			sev = SeverityError
		case "critical":
			sev = SeverityCritical
		}
		severity = &sev
	}

	alerts := h.monitor.GetAlerts(since, alertType, severity)

	respondJSON(w, http.StatusOK, map[string]any{
		"alerts": alerts,
		"count":  len(alerts),
	})
}

// GET /api/p2p/reputation/metrics
func (h *HTTPHandlers) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	eventCounts := h.metrics.GetEventCounts()
	eventRates := h.metrics.GetEventRates()
	banMetrics := h.metrics.GetBanMetrics()
	processingMetrics := h.metrics.GetProcessingMetrics()
	scoreHistory := h.metrics.GetScoreHistory()

	respondJSON(w, http.StatusOK, map[string]any{
		"event_counts":       eventCounts,
		"event_rates":        eventRates,
		"ban_metrics":        banMetrics,
		"processing_metrics": processingMetrics,
		"score_history":      scoreHistory,
	})
}

// GET /api/p2p/reputation/metrics/prometheus
func (h *HTTPHandlers) handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	output := h.metrics.ExportPrometheus()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(output))
}

// POST /api/p2p/reputation/ban
// Request body: {"peer_id": "...", "duration": "24h", "reason": "..."}
func (h *HTTPHandlers) handleBanPeer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PeerID   string `json:"peer_id"`
		Duration string `json:"duration"`
		Reason   string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.PeerID == "" {
		http.Error(w, "peer_id required", http.StatusBadRequest)
		return
	}

	duration, err := time.ParseDuration(req.Duration)
	if err != nil {
		http.Error(w, "Invalid duration format", http.StatusBadRequest)
		return
	}

	if req.Reason == "" {
		req.Reason = "Manual ban via API"
	}

	if err := h.manager.BanPeer(PeerID(req.PeerID), duration, req.Reason); err != nil {
		http.Error(w, "Failed to ban peer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Peer banned successfully",
	})
}

// POST /api/p2p/reputation/unban
// Request body: {"peer_id": "..."}
func (h *HTTPHandlers) handleUnbanPeer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PeerID string `json:"peer_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.PeerID == "" {
		http.Error(w, "peer_id required", http.StatusBadRequest)
		return
	}

	if err := h.manager.UnbanPeer(PeerID(req.PeerID)); err != nil {
		http.Error(w, "Failed to unban peer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Peer unbanned successfully",
	})
}

// Helper function to respond with JSON
func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
