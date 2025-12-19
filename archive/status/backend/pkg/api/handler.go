package api

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"status/pkg/health"
	"status/pkg/incidents"
	"status/pkg/metrics"
)

// Handler handles HTTP API requests
type Handler struct {
	healthMonitor    *health.Monitor
	incidentManager  *incidents.Manager
	metricsCollector *metrics.Collector
}

// NewHandler creates a new API handler
func NewHandler(
	healthMonitor *health.Monitor,
	incidentManager *incidents.Manager,
	metricsCollector *metrics.Collector,
) *Handler {
	return &Handler{
		healthMonitor:    healthMonitor,
		incidentManager:  incidentManager,
		metricsCollector: metricsCollector,
	}
}

// RegisterRoutes registers all API routes
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Health endpoints
	router.HandleFunc("/health", h.HandleHealth).Methods("GET")
	router.HandleFunc("/status", h.HandleStatus).Methods("GET")

	// Incident endpoints
	router.HandleFunc("/incidents", h.HandleGetIncidents).Methods("GET")
	router.HandleFunc("/incidents", h.HandleCreateIncident).Methods("POST")
	router.HandleFunc("/incidents/{id}", h.HandleGetIncident).Methods("GET")
	router.HandleFunc("/incidents/{id}/update", h.HandleUpdateIncident).Methods("POST")

	// Metrics endpoints
	router.HandleFunc("/metrics", h.HandleGetMetrics).Methods("GET")
	router.HandleFunc("/metrics/summary", h.HandleGetMetricsSummary).Methods("GET")

	// Status history
	router.HandleFunc("/status/history", h.HandleStatusHistory).Methods("GET")

	// Subscribe endpoints
	router.HandleFunc("/subscribe", h.HandleSubscribe).Methods("POST")
	router.HandleFunc("/unsubscribe", h.HandleUnsubscribe).Methods("POST")

	// RSS feed
	router.HandleFunc("/status/rss", h.HandleRSSFeed).Methods("GET")
}

// HandleHealth returns basic health check
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	response := h.healthMonitor.HealthCheck()
	h.respondJSON(w, http.StatusOK, response)
}

// HandleStatus returns overall system status
func (h *Handler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	status := h.healthMonitor.GetStatus()
	h.respondJSON(w, http.StatusOK, status)
}

// HandleGetIncidents returns all incidents
func (h *Handler) HandleGetIncidents(w http.ResponseWriter, r *http.Request) {
	incidents := h.incidentManager.GetAllIncidents()
	h.respondJSON(w, http.StatusOK, incidents)
}

// HandleGetIncident returns a specific incident
func (h *Handler) HandleGetIncident(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid incident ID")
		return
	}

	incident, err := h.incidentManager.GetIncident(id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, incident)
}

// CreateIncidentRequest represents a request to create an incident
type CreateIncidentRequest struct {
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Severity    incidents.Severity `json:"severity"`
	Components  []string           `json:"components"`
}

// HandleCreateIncident creates a new incident
func (h *Handler) HandleCreateIncident(w http.ResponseWriter, r *http.Request) {
	var req CreateIncidentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.Title == "" {
		h.respondError(w, http.StatusBadRequest, "Title is required")
		return
	}

	if req.Severity != incidents.SeverityCritical &&
		req.Severity != incidents.SeverityMajor &&
		req.Severity != incidents.SeverityMinor {
		h.respondError(w, http.StatusBadRequest, "Invalid severity level")
		return
	}

	incident, err := h.incidentManager.CreateIncident(
		req.Title,
		req.Description,
		req.Severity,
		req.Components,
	)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, incident)
}

// UpdateIncidentRequest represents a request to update an incident
type UpdateIncidentRequest struct {
	Message string                   `json:"message"`
	Status  incidents.IncidentStatus `json:"status"`
}

// HandleUpdateIncident updates an existing incident
func (h *Handler) HandleUpdateIncident(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid incident ID")
		return
	}

	var req UpdateIncidentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Message == "" {
		h.respondError(w, http.StatusBadRequest, "Message is required")
		return
	}

	if err := h.incidentManager.UpdateIncident(id, req.Message, req.Status); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	incident, _ := h.incidentManager.GetIncident(id)
	h.respondJSON(w, http.StatusOK, incident)
}

// HandleGetMetrics returns current metrics
func (h *Handler) HandleGetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := h.metricsCollector.GetMetrics()
	h.respondJSON(w, http.StatusOK, metrics)
}

// HandleGetMetricsSummary returns metrics summary
func (h *Handler) HandleGetMetricsSummary(w http.ResponseWriter, r *http.Request) {
	summary := h.metricsCollector.GetMetricsSummary()
	h.respondJSON(w, http.StatusOK, summary)
}

// HandleStatusHistory returns uptime history
func (h *Handler) HandleStatusHistory(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 90 {
			days = d
		}
	}

	history := h.healthMonitor.GetUptimeHistory(days)
	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"days":    days,
		"history": history,
	})
}

// SubscribeRequest represents a subscription request
type SubscribeRequest struct {
	Email       string          `json:"email"`
	Preferences map[string]bool `json:"preferences"`
}

// HandleSubscribe handles subscription requests
func (h *Handler) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	var req SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" {
		h.respondError(w, http.StatusBadRequest, "Email is required")
		return
	}

	if err := h.incidentManager.Subscribe(req.Email); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Successfully subscribed to status updates",
	})
}

// UnsubscribeRequest represents an unsubscribe request
type UnsubscribeRequest struct {
	Email string `json:"email"`
}

// HandleUnsubscribe handles unsubscribe requests
func (h *Handler) HandleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	var req UnsubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" {
		h.respondError(w, http.StatusBadRequest, "Email is required")
		return
	}

	if err := h.incidentManager.Unsubscribe(req.Email); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Successfully unsubscribed from status updates",
	})
}

// RSS Feed structures
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

// HandleRSSFeed generates RSS feed for status updates
func (h *Handler) HandleRSSFeed(w http.ResponseWriter, r *http.Request) {
	status := h.healthMonitor.GetStatus()
	allIncidents := h.incidentManager.GetAllIncidents()

	items := make([]Item, 0)

	// Add active incidents
	if active, ok := allIncidents["active"].([]*incidents.Incident); ok {
		for _, incident := range active {
			items = append(items, Item{
				Title:       fmt.Sprintf("[%s] %s", incident.Severity, incident.Title),
				Link:        fmt.Sprintf("https://status.pawchain.io/incidents/%d", incident.ID),
				Description: incident.Description,
				PubDate:     incident.StartedAt.Format(time.RFC1123),
				GUID:        fmt.Sprintf("incident-%d", incident.ID),
			})
		}
	}

	// Add recent resolved incidents
	if history, ok := allIncidents["history"].([]*incidents.Incident); ok {
		for _, incident := range history {
			items = append(items, Item{
				Title:       fmt.Sprintf("[Resolved] %s", incident.Title),
				Link:        fmt.Sprintf("https://status.pawchain.io/incidents/%d", incident.ID),
				Description: incident.Description,
				PubDate:     incident.StartedAt.Format(time.RFC1123),
				GUID:        fmt.Sprintf("incident-%d", incident.ID),
			})
		}
	}

	rss := RSS{
		Version: "2.0",
		Channel: Channel{
			Title:       "PAW Blockchain Status",
			Link:        "https://status.pawchain.io",
			Description: fmt.Sprintf("Status updates for PAW Blockchain - %s", status.Message),
			Items:       items,
		},
	}

	w.Header().Set("Content-Type", "application/rss+xml")
	xml.NewEncoder(w).Encode(rss)
}

// respondJSON sends a JSON response
func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

// respondError sends an error response
func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]interface{}{
		"error":   true,
		"message": message,
	})
}
