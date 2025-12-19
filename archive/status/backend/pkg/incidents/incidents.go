package incidents

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"status/pkg/config"
)

// Severity represents the severity level of an incident
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityMajor    Severity = "major"
	SeverityMinor    Severity = "minor"
)

// IncidentStatus represents the current status of an incident
type IncidentStatus string

const (
	StatusInvestigating IncidentStatus = "investigating"
	StatusIdentified    IncidentStatus = "identified"
	StatusMonitoring    IncidentStatus = "monitoring"
	StatusResolved      IncidentStatus = "resolved"
)

// Update represents an update to an incident
type Update struct {
	Timestamp time.Time      `json:"timestamp"`
	Message   string         `json:"message"`
	Status    IncidentStatus `json:"status,omitempty"`
}

// Incident represents a system incident
type Incident struct {
	ID          int            `json:"id"`
	Title       string         `json:"title"`
	Severity    Severity       `json:"severity"`
	Status      IncidentStatus `json:"status"`
	StartedAt   time.Time      `json:"started_at"`
	ResolvedAt  *time.Time     `json:"resolved_at,omitempty"`
	Description string         `json:"description"`
	Updates     []Update       `json:"updates"`
	Components  []string       `json:"components"`
}

// Manager handles incident management
type Manager struct {
	config      *config.Config
	incidents   map[int]*Incident
	nextID      int
	mutex       sync.RWMutex
	subscribers []string
}

// NewManager creates a new incident manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config:      cfg,
		incidents:   make(map[int]*Incident),
		nextID:      1,
		subscribers: make([]string, 0),
	}
}

// Start begins the incident manager
func (m *Manager) Start(ctx context.Context) {
	log.Println("Incident manager started")

	// Load historical incidents (in production, this would load from database)
	m.loadHistoricalIncidents()

	<-ctx.Done()
	log.Println("Incident manager stopped")
}

// loadHistoricalIncidents loads sample historical data
func (m *Manager) loadHistoricalIncidents() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	resolvedTime1 := now.Add(-7 * 24 * time.Hour).Add(2 * time.Hour)
	resolvedTime2 := now.Add(-14 * 24 * time.Hour).Add(45 * time.Minute)

	historicalIncidents := []*Incident{
		{
			ID:          1,
			Title:       "Scheduled Maintenance - Database Upgrade",
			Severity:    SeverityMinor,
			Status:      StatusResolved,
			StartedAt:   now.Add(-7 * 24 * time.Hour),
			ResolvedAt:  &resolvedTime1,
			Description: "Planned database upgrade to improve performance.",
			Components:  []string{"API", "Explorer"},
			Updates: []Update{
				{
					Timestamp: now.Add(-7 * 24 * time.Hour),
					Message:   "Maintenance window started. Expected duration: 2 hours.",
					Status:    StatusInvestigating,
				},
				{
					Timestamp: now.Add(-7 * 24 * time.Hour).Add(1 * time.Hour),
					Message:   "Database upgrade in progress. All services operational.",
					Status:    StatusMonitoring,
				},
				{
					Timestamp: now.Add(-7 * 24 * time.Hour).Add(2 * time.Hour),
					Message:   "Maintenance completed successfully. All systems normal.",
					Status:    StatusResolved,
				},
			},
		},
		{
			ID:          2,
			Title:       "API Rate Limiting Issues",
			Severity:    SeverityMajor,
			Status:      StatusResolved,
			StartedAt:   now.Add(-14 * 24 * time.Hour),
			ResolvedAt:  &resolvedTime2,
			Description: "Some users experienced rate limiting errors on API endpoints.",
			Components:  []string{"API"},
			Updates: []Update{
				{
					Timestamp: now.Add(-14 * 24 * time.Hour),
					Message:   "Investigating reports of API rate limiting issues.",
					Status:    StatusInvestigating,
				},
				{
					Timestamp: now.Add(-14 * 24 * time.Hour).Add(30 * time.Minute),
					Message:   "Issue identified. Deploying fix.",
					Status:    StatusIdentified,
				},
				{
					Timestamp: now.Add(-14 * 24 * time.Hour).Add(45 * time.Minute),
					Message:   "Fix deployed. Monitoring for stability.",
					Status:    StatusResolved,
				},
			},
		},
	}

	for _, incident := range historicalIncidents {
		m.incidents[incident.ID] = incident
		m.nextID = incident.ID + 1
	}
}

// CreateIncident creates a new incident
func (m *Manager) CreateIncident(title, description string, severity Severity, components []string) (*Incident, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	incident := &Incident{
		ID:          m.nextID,
		Title:       title,
		Severity:    severity,
		Status:      StatusInvestigating,
		StartedAt:   time.Now(),
		Description: description,
		Components:  components,
		Updates:     make([]Update, 0),
	}

	m.nextID++
	m.incidents[incident.ID] = incident

	// Add initial update
	incident.Updates = append(incident.Updates, Update{
		Timestamp: time.Now(),
		Message:   "Incident detected and investigation started.",
		Status:    StatusInvestigating,
	})

	log.Printf("New incident created: %d - %s", incident.ID, incident.Title)

	// Notify subscribers
	m.notifySubscribers(incident)

	return incident, nil
}

// UpdateIncident adds an update to an existing incident
func (m *Manager) UpdateIncident(id int, message string, status IncidentStatus) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	incident, exists := m.incidents[id]
	if !exists {
		return ErrIncidentNotFound
	}

	update := Update{
		Timestamp: time.Now(),
		Message:   message,
		Status:    status,
	}

	incident.Updates = append(incident.Updates, update)
	incident.Status = status

	if status == StatusResolved {
		now := time.Now()
		incident.ResolvedAt = &now
	}

	log.Printf("Incident %d updated: %s", id, message)

	// Notify subscribers
	m.notifySubscribers(incident)

	return nil
}

// GetIncident retrieves a specific incident
func (m *Manager) GetIncident(id int) (*Incident, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	incident, exists := m.incidents[id]
	if !exists {
		return nil, ErrIncidentNotFound
	}

	return incident, nil
}

// GetActiveIncidents returns all active incidents
func (m *Manager) GetActiveIncidents() []*Incident {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	active := make([]*Incident, 0)
	for _, incident := range m.incidents {
		if incident.Status != StatusResolved {
			active = append(active, incident)
		}
	}

	return active
}

// GetIncidentHistory returns resolved incidents
func (m *Manager) GetIncidentHistory(limit int) []*Incident {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	history := make([]*Incident, 0)
	for _, incident := range m.incidents {
		if incident.Status == StatusResolved {
			history = append(history, incident)
		}
	}

	// Sort by resolved time (most recent first)
	// In production, this would be done in the database query
	if len(history) > limit {
		history = history[:limit]
	}

	return history
}

// GetAllIncidents returns all incidents
func (m *Manager) GetAllIncidents() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return map[string]interface{}{
		"active":  m.GetActiveIncidents(),
		"history": m.GetIncidentHistory(10),
	}
}

// Subscribe adds an email to the subscriber list
func (m *Manager) Subscribe(email string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if already subscribed
	for _, sub := range m.subscribers {
		if sub == email {
			return nil // Already subscribed
		}
	}

	m.subscribers = append(m.subscribers, email)
	log.Printf("New subscriber: %s", email)

	return nil
}

// Unsubscribe removes an email from the subscriber list
func (m *Manager) Unsubscribe(email string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i, sub := range m.subscribers {
		if sub == email {
			m.subscribers = append(m.subscribers[:i], m.subscribers[i+1:]...)
			log.Printf("Subscriber removed: %s", email)
			return nil
		}
	}

	return ErrSubscriberNotFound
}

// notifySubscribers sends notifications to all subscribers
func (m *Manager) notifySubscribers(incident *Incident) {
	// In production, this would send emails or webhooks
	log.Printf("Notifying %d subscribers about incident: %s", len(m.subscribers), incident.Title)

	// Send webhook if configured
	if m.config.IncidentWebhookURL != "" {
		go m.sendWebhook(incident)
	}
}

// sendWebhook sends incident data to webhook URL
func (m *Manager) sendWebhook(incident *Incident) {
	// In production, implement actual webhook sending
	data, _ := json.Marshal(incident)
	log.Printf("Webhook payload: %s", string(data))
}

// Errors
var (
	ErrIncidentNotFound   = &IncidentError{Message: "incident not found"}
	ErrSubscriberNotFound = &IncidentError{Message: "subscriber not found"}
)

// IncidentError represents an incident-related error
type IncidentError struct {
	Message string
}

func (e *IncidentError) Error() string {
	return e.Message
}

// MarshalJSON custom JSON marshaling for Incident
func (i *Incident) MarshalJSON() ([]byte, error) {
	type Alias Incident
	aux := &struct {
		*Alias
		StartedAt  string  `json:"started_at"`
		ResolvedAt *string `json:"resolved_at,omitempty"`
	}{
		Alias:     (*Alias)(i),
		StartedAt: i.StartedAt.Format(time.RFC3339),
	}

	if i.ResolvedAt != nil {
		resolved := i.ResolvedAt.Format(time.RFC3339)
		aux.ResolvedAt = &resolved
	}

	return json.Marshal(aux)
}

// MarshalJSON custom JSON marshaling for Update
func (u *Update) MarshalJSON() ([]byte, error) {
	type Alias Update
	return json.Marshal(&struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias:     (*Alias)(u),
		Timestamp: u.Timestamp.Format(time.RFC3339),
	})
}
