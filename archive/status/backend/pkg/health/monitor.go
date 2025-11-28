package health

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"status/pkg/config"
)

// ComponentStatus represents the status of a system component
type ComponentStatus string

const (
	StatusOperational ComponentStatus = "operational"
	StatusDegraded    ComponentStatus = "degraded"
	StatusDown        ComponentStatus = "down"
)

// Component represents a monitored system component
type Component struct {
	Name         string          `json:"name"`
	Status       ComponentStatus `json:"status"`
	Description  string          `json:"description"`
	Uptime       string          `json:"uptime"`
	ResponseTime string          `json:"response_time"`
	LastChecked  time.Time       `json:"last_checked"`
	HealthURL    string          `json:"-"`
}

// OverallStatus represents the overall system status
type OverallStatus struct {
	Status     ComponentStatus `json:"overall_status"`
	Message    string          `json:"message"`
	Components []Component     `json:"components"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// Monitor handles health monitoring of all components
type Monitor struct {
	config     *config.Config
	components map[string]*Component
	mutex      sync.RWMutex
	httpClient *http.Client
	uptime     map[string]*UptimeTracker
}

// UptimeTracker tracks uptime statistics
type UptimeTracker struct {
	TotalChecks   int64
	SuccessChecks int64
	StartTime     time.Time
}

// NewMonitor creates a new health monitor
func NewMonitor(cfg *config.Config) *Monitor {
	return &Monitor{
		config:     cfg,
		components: make(map[string]*Component),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		uptime:     make(map[string]*UptimeTracker),
	}
}

// Start begins the monitoring process
func (m *Monitor) Start(ctx context.Context) {
	m.initializeComponents()

	ticker := time.NewTicker(m.config.MonitorInterval)
	defer ticker.Stop()

	// Initial check
	m.checkAllComponents()

	for {
		select {
		case <-ctx.Done():
			log.Println("Health monitor stopped")
			return
		case <-ticker.C:
			m.checkAllComponents()
		}
	}
}

// initializeComponents sets up all monitored components
func (m *Monitor) initializeComponents() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	components := []Component{
		{
			Name:        "Blockchain",
			Description: "Core blockchain network",
			HealthURL:   m.config.BlockchainRPCURL + "/health",
			Status:      StatusOperational,
		},
		{
			Name:        "API",
			Description: "REST and GraphQL API endpoints",
			HealthURL:   m.config.APIEndpoint + "/cosmos/base/tendermint/v1beta1/node_info",
			Status:      StatusOperational,
		},
		{
			Name:        "WebSocket",
			Description: "Real-time data streaming",
			HealthURL:   m.config.BlockchainRPCURL + "/status",
			Status:      StatusOperational,
		},
		{
			Name:        "Explorer",
			Description: "Block explorer interface",
			HealthURL:   m.config.ExplorerEndpoint,
			Status:      StatusOperational,
		},
		{
			Name:        "Faucet",
			Description: "Testnet token distribution",
			HealthURL:   m.config.FaucetEndpoint + "/api/v1/health",
			Status:      StatusOperational,
		},
	}

	for _, comp := range components {
		c := comp
		m.components[c.Name] = &c
		m.uptime[c.Name] = &UptimeTracker{
			StartTime: time.Now(),
		}
	}
}

// checkAllComponents performs health checks on all components
func (m *Monitor) checkAllComponents() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for name, component := range m.components {
		status, responseTime := m.checkComponent(component)

		tracker := m.uptime[name]
		tracker.TotalChecks++
		if status == StatusOperational {
			tracker.SuccessChecks++
		}

		component.Status = status
		component.ResponseTime = responseTime
		component.LastChecked = time.Now()
		component.Uptime = m.calculateUptime(tracker)

		log.Printf("Health check - %s: %s (response time: %s)", name, status, responseTime)
	}
}

// checkComponent performs a health check on a single component
func (m *Monitor) checkComponent(component *Component) (ComponentStatus, string) {
	start := time.Now()

	req, err := http.NewRequest("GET", component.HealthURL, nil)
	if err != nil {
		return StatusDown, "N/A"
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return StatusDown, "N/A"
	}
	defer resp.Body.Close()

	responseTime := time.Since(start)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if responseTime > 5*time.Second {
			return StatusDegraded, formatDuration(responseTime)
		}
		return StatusOperational, formatDuration(responseTime)
	}

	if resp.StatusCode >= 500 {
		return StatusDown, formatDuration(responseTime)
	}

	return StatusDegraded, formatDuration(responseTime)
}

// calculateUptime calculates the uptime percentage
func (m *Monitor) calculateUptime(tracker *UptimeTracker) string {
	if tracker.TotalChecks == 0 {
		return "100.00%"
	}
	uptime := float64(tracker.SuccessChecks) / float64(tracker.TotalChecks) * 100
	return fmt.Sprintf("%.2f%%", uptime)
}

// GetStatus returns the current overall status
func (m *Monitor) GetStatus() *OverallStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	components := make([]Component, 0, len(m.components))
	overallStatus := StatusOperational
	downCount := 0
	degradedCount := 0

	for _, comp := range m.components {
		components = append(components, *comp)

		if comp.Status == StatusDown {
			downCount++
		} else if comp.Status == StatusDegraded {
			degradedCount++
		}
	}

	var message string
	if downCount > 0 {
		overallStatus = StatusDown
		message = fmt.Sprintf("%d component(s) are experiencing issues", downCount)
	} else if degradedCount > 0 {
		overallStatus = StatusDegraded
		message = fmt.Sprintf("%d component(s) are experiencing degraded performance", degradedCount)
	} else {
		message = "All systems operational"
	}

	return &OverallStatus{
		Status:     overallStatus,
		Message:    message,
		Components: components,
		UpdatedAt:  time.Now(),
	}
}

// GetComponent returns a specific component status
func (m *Monitor) GetComponent(name string) (*Component, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	component, exists := m.components[name]
	if !exists {
		return nil, fmt.Errorf("component not found: %s", name)
	}

	return component, nil
}

// GetUptimeHistory returns uptime data for the past N days
func (m *Monitor) GetUptimeHistory(days int) []map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	history := make([]map[string]interface{}, days)
	now := time.Now()

	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i)

		// Simulate uptime data - in production, this would come from a database
		status := "operational"
		if i == 3 || i == 7 { // Simulate some degraded days
			status = "degraded"
		}

		history[i] = map[string]interface{}{
			"date":   date,
			"status": status,
		}
	}

	return history
}

// formatDuration formats a duration as a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.0fµs", float64(d.Microseconds()))
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// HealthCheckResponse is the response structure for health check endpoints
type HealthCheckResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

// HealthCheck returns a simple health check response
func (m *Monitor) HealthCheck() *HealthCheckResponse {
	return &HealthCheckResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
	}
}

// MarshalJSON custom JSON marshaling for Component
func (c *Component) MarshalJSON() ([]byte, error) {
	type Alias Component
	return json.Marshal(&struct {
		*Alias
		LastChecked string `json:"last_checked"`
	}{
		Alias:       (*Alias)(c),
		LastChecked: c.LastChecked.Format(time.RFC3339),
	})
}
