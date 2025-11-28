package reputation

import (
	"context"
	"sync"
	"time"

	"cosmossdk.io/log"
)

// Monitor provides monitoring and observability for the reputation system
type Monitor struct {
	manager *Manager
	metrics *Metrics
	logger  log.Logger

	// Alert thresholds
	config MonitorConfig

	// Alert tracking
	alerts    []Alert
	alertsMu  sync.RWMutex
	maxAlerts int

	// Health status
	lastHealthCheck time.Time
	healthStatus    HealthStatus

	stopChan chan struct{}
	wg       sync.WaitGroup
}

// MonitorConfig configures monitoring
type MonitorConfig struct {
	// Alert thresholds
	HighBanRateThreshold    float64       // Bans per hour
	LowAvgScoreThreshold    float64       // Average score threshold
	HighSubnetConcentration float64       // Max % of peers from one subnet
	AlertCooldown           time.Duration // Min time between same alert type

	// Health check settings
	HealthCheckInterval  time.Duration
	MaxUnhealthyDuration time.Duration

	// Metrics collection
	MetricsCollectionInterval time.Duration
}

// DefaultMonitorConfig returns default monitor config
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		HighBanRateThreshold:    10.0, // 10 bans/hour
		LowAvgScoreThreshold:    60.0,
		HighSubnetConcentration: 0.30, // 30%
		AlertCooldown:           1 * time.Hour,

		HealthCheckInterval:  5 * time.Minute,
		MaxUnhealthyDuration: 30 * time.Minute,

		MetricsCollectionInterval: 1 * time.Minute,
	}
}

// Alert represents a system alert
type Alert struct {
	ID        string    `json:"id"`
	Type      AlertType `json:"type"`
	Severity  Severity  `json:"severity"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data,omitempty"`
}

// AlertType defines alert types
type AlertType int

const (
	AlertTypeHighBanRate AlertType = iota
	AlertTypeLowAvgScore
	AlertTypeSubnetConcentration
	AlertTypeGeographicImbalance
	AlertTypeSystemError
	AlertTypeStorageError
)

func (at AlertType) String() string {
	switch at {
	case AlertTypeHighBanRate:
		return "high_ban_rate"
	case AlertTypeLowAvgScore:
		return "low_avg_score"
	case AlertTypeSubnetConcentration:
		return "subnet_concentration"
	case AlertTypeGeographicImbalance:
		return "geographic_imbalance"
	case AlertTypeSystemError:
		return "system_error"
	case AlertTypeStorageError:
		return "storage_error"
	default:
		return "unknown"
	}
}

// Severity defines alert severity
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// HealthStatus represents system health
type HealthStatus struct {
	Healthy        bool      `json:"healthy"`
	LastCheck      time.Time `json:"last_check"`
	Issues         []string  `json:"issues,omitempty"`
	TotalPeers     int       `json:"total_peers"`
	BannedPeers    int       `json:"banned_peers"`
	AvgScore       float64   `json:"avg_score"`
	StorageHealthy bool      `json:"storage_healthy"`
	UptimeSeconds  int64     `json:"uptime_seconds"`
}

// NewMonitor creates a new reputation monitor
func NewMonitor(manager *Manager, metrics *Metrics, config MonitorConfig, logger log.Logger) *Monitor {
	m := &Monitor{
		manager:         manager,
		metrics:         metrics,
		logger:          logger,
		config:          config,
		alerts:          make([]Alert, 0),
		maxAlerts:       1000,
		lastHealthCheck: time.Now(),
		healthStatus:    HealthStatus{Healthy: true},
		stopChan:        make(chan struct{}),
	}

	m.startBackgroundTasks()
	return m
}

// GetHealth returns current health status
func (m *Monitor) GetHealth() HealthStatus {
	m.healthStatus.LastCheck = m.lastHealthCheck
	return m.healthStatus
}

// GetAlerts returns recent alerts
func (m *Monitor) GetAlerts(since time.Time, alertType *AlertType, severity *Severity) []Alert {
	m.alertsMu.RLock()
	defer m.alertsMu.RUnlock()

	var filtered []Alert
	for _, alert := range m.alerts {
		if alert.Timestamp.Before(since) {
			continue
		}

		if alertType != nil && alert.Type != *alertType {
			continue
		}

		if severity != nil && alert.Severity != *severity {
			continue
		}

		filtered = append(filtered, alert)
	}

	return filtered
}

// ClearAlerts clears old alerts
func (m *Monitor) ClearAlerts(olderThan time.Time) {
	m.alertsMu.Lock()
	defer m.alertsMu.Unlock()

	var kept []Alert
	for _, alert := range m.alerts {
		if alert.Timestamp.After(olderThan) {
			kept = append(kept, alert)
		}
	}

	m.alerts = kept
}

// Close stops the monitor
func (m *Monitor) Close() error {
	close(m.stopChan)
	m.wg.Wait()
	return nil
}

// Internal methods

func (m *Monitor) startBackgroundTasks() {
	// Health check task
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(m.config.HealthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.performHealthCheck()
			case <-m.stopChan:
				return
			}
		}
	}()

	// Metrics collection task
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(m.config.MetricsCollectionInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.collectMetrics()
			case <-m.stopChan:
				return
			}
		}
	}()

	// Alert checking task
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.checkAlerts()
			case <-m.stopChan:
				return
			}
		}
	}()
}

func (m *Monitor) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_ = ctx // Use context if needed for future async operations

	status := HealthStatus{
		Healthy:        true,
		LastCheck:      time.Now(),
		Issues:         make([]string, 0),
		StorageHealthy: true,
	}

	// Check statistics
	stats := m.manager.GetStatistics()
	status.TotalPeers = stats.TotalPeers
	status.BannedPeers = stats.BannedPeers
	status.AvgScore = stats.AvgScore

	// Check for issues
	if stats.AvgScore < m.config.LowAvgScoreThreshold && stats.TotalPeers > 10 {
		status.Healthy = false
		status.Issues = append(status.Issues, "average peer score below threshold")
	}

	banRatio := 0.0
	if stats.TotalPeers > 0 {
		banRatio = float64(stats.BannedPeers) / float64(stats.TotalPeers)
	}

	if banRatio > 0.5 && stats.TotalPeers > 10 {
		status.Healthy = false
		status.Issues = append(status.Issues, "high ban ratio (>50%)")
	}

	// Check storage health
	// Try to perform a simple operation
	_, err := m.manager.storage.LoadLatestSnapshot()
	if err != nil {
		status.StorageHealthy = false
		status.Healthy = false
		status.Issues = append(status.Issues, "storage error: "+err.Error())
	}

	m.healthStatus = status
	m.lastHealthCheck = time.Now()

	if !status.Healthy {
		m.logger.Warn("reputation system unhealthy", "issues", status.Issues)
	}
}

func (m *Monitor) collectMetrics() {
	stats := m.manager.GetStatistics()

	// Calculate average score across all peers
	m.metrics.mu.Lock()
	totalScore := 0.0
	minScore := 100.0
	maxScore := 0.0
	peerCount := 0

	for _, score := range m.metrics.peerScores {
		totalScore += score
		if score < minScore {
			minScore = score
		}
		if score > maxScore {
			maxScore = score
		}
		peerCount++
	}

	avgScore := 0.0
	if peerCount > 0 {
		avgScore = totalScore / float64(peerCount)
	}

	// Add to history
	point := ScoreHistoryPoint{
		Timestamp: time.Now(),
		AvgScore:  avgScore,
		MinScore:  minScore,
		MaxScore:  maxScore,
		PeerCount: peerCount,
	}
	m.metrics.mu.Unlock()

	m.metrics.AddScoreHistoryPoint(point)

	// Log summary
	m.logger.Debug("metrics collected",
		"total_peers", stats.TotalPeers,
		"banned_peers", stats.BannedPeers,
		"avg_score", avgScore,
	)
}

func (m *Monitor) checkAlerts() {
	// Check ban rate
	banMetrics := m.metrics.GetBanMetrics()
	// Simplified: calculate rate from total bans (would need time-windowed tracking for accuracy)
	if banMetrics.TotalBans > 0 {
		// Rough estimate: if we've seen more than threshold in the last collection
		// A proper implementation would track bans in a time window
		recentBans := banMetrics.TotalBans // This is total, not per hour
		if float64(recentBans) > m.config.HighBanRateThreshold {
			m.raiseAlert(Alert{
				Type:      AlertTypeHighBanRate,
				Severity:  SeverityWarning,
				Message:   "High ban rate detected",
				Timestamp: time.Now(),
				Data: map[string]any{
					"total_bans":     banMetrics.TotalBans,
					"temp_bans":      banMetrics.TempBans,
					"permanent_bans": banMetrics.PermanentBans,
				},
			})
		}
	}

	// Check average score
	stats := m.manager.GetStatistics()
	if stats.AvgScore < m.config.LowAvgScoreThreshold && stats.TotalPeers > 10 {
		m.raiseAlert(Alert{
			Type:      AlertTypeLowAvgScore,
			Severity:  SeverityWarning,
			Message:   "Average peer score below threshold",
			Timestamp: time.Now(),
			Data: map[string]any{
				"avg_score":   stats.AvgScore,
				"threshold":   m.config.LowAvgScoreThreshold,
				"total_peers": stats.TotalPeers,
			},
		})
	}

	// Check subnet concentration
	m.manager.statsMu.RLock()
	totalPeers := stats.TotalPeers
	for subnet, subnetStats := range m.manager.subnetStats {
		if totalPeers > 0 {
			concentration := float64(subnetStats.PeerCount) / float64(totalPeers)
			if concentration > m.config.HighSubnetConcentration {
				m.raiseAlert(Alert{
					Type:      AlertTypeSubnetConcentration,
					Severity:  SeverityWarning,
					Message:   "High peer concentration in subnet",
					Timestamp: time.Now(),
					Data: map[string]any{
						"subnet":        subnet,
						"peer_count":    subnetStats.PeerCount,
						"concentration": concentration,
					},
				})
			}
		}
	}
	m.manager.statsMu.RUnlock()
}

func (m *Monitor) raiseAlert(alert Alert) {
	// Generate alert ID
	alert.ID = time.Now().Format("20060102150405") + "-" + alert.Type.String()

	// Check cooldown (don't spam same alert type)
	m.alertsMu.RLock()
	for _, existing := range m.alerts {
		if existing.Type == alert.Type &&
			time.Since(existing.Timestamp) < m.config.AlertCooldown {
			m.alertsMu.RUnlock()
			return // Skip - too soon
		}
	}
	m.alertsMu.RUnlock()

	// Add alert
	m.alertsMu.Lock()
	m.alerts = append(m.alerts, alert)

	// Keep only max alerts
	if len(m.alerts) > m.maxAlerts {
		m.alerts = m.alerts[len(m.alerts)-m.maxAlerts:]
	}
	m.alertsMu.Unlock()

	// Log alert
	m.logger.Info("reputation alert raised",
		"type", alert.Type.String(),
		"severity", alert.Severity.String(),
		"message", alert.Message,
	)
}
