package reputation

import (
	"fmt"
	"sync"
	"time"
)

// Metrics tracks reputation system metrics
type Metrics struct {
	mu sync.RWMutex

	// Event counters
	eventCounts     map[EventType]int64
	eventRates      map[EventType]float64 // events per second
	lastEventUpdate time.Time

	// Score tracking
	peerScores     map[PeerID]float64
	scoreHistory   []ScoreHistoryPoint
	maxHistorySize int

	// Ban metrics
	tempBans      int64
	permanentBans int64
	totalBans     int64
	banReasons    map[string]int64

	// Performance metrics
	avgProcessingTime time.Duration
	maxProcessingTime time.Duration
	processingCount   int64
}

// ScoreHistoryPoint represents a historical score data point
type ScoreHistoryPoint struct {
	Timestamp time.Time
	AvgScore  float64
	MinScore  float64
	MaxScore  float64
	PeerCount int
}

// NewMetrics creates a new metrics tracker
func NewMetrics() *Metrics {
	return &Metrics{
		eventCounts:     make(map[EventType]int64),
		eventRates:      make(map[EventType]float64),
		peerScores:      make(map[PeerID]float64),
		scoreHistory:    make([]ScoreHistoryPoint, 0, 1440), // 24h at 1min intervals
		maxHistorySize:  1440,
		banReasons:      make(map[string]int64),
		lastEventUpdate: time.Now(),
	}
}

// RecordEvent records an event occurrence
func (m *Metrics) RecordEvent(eventType EventType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.eventCounts[eventType]++

	// Update rates every second
	now := time.Now()
	if now.Sub(m.lastEventUpdate) >= time.Second {
		duration := now.Sub(m.lastEventUpdate).Seconds()
		for et, count := range m.eventCounts {
			m.eventRates[et] = float64(count) / duration
		}
		m.lastEventUpdate = now
		// Reset counters for next interval
		m.eventCounts = make(map[EventType]int64)
	}
}

// UpdateScore updates peer score
func (m *Metrics) UpdateScore(peerID PeerID, score float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.peerScores[peerID] = score
}

// RecordBan records a ban event
func (m *Metrics) RecordBan(banType BanType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalBans++

	switch banType {
	case BanTypeTemporary:
		m.tempBans++
	case BanTypePermanent:
		m.permanentBans++
	}
}

// RecordBanReason records the reason for a ban
func (m *Metrics) RecordBanReason(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.banReasons[reason]++
}

// RecordProcessingTime records event processing time
func (m *Metrics) RecordProcessingTime(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.processingCount++

	// Update average
	totalTime := m.avgProcessingTime * time.Duration(m.processingCount-1)
	m.avgProcessingTime = (totalTime + duration) / time.Duration(m.processingCount)

	// Update max
	if duration > m.maxProcessingTime {
		m.maxProcessingTime = duration
	}
}

// AddScoreHistoryPoint adds a point to score history
func (m *Metrics) AddScoreHistoryPoint(point ScoreHistoryPoint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.scoreHistory = append(m.scoreHistory, point)

	// Keep only max size
	if len(m.scoreHistory) > m.maxHistorySize {
		m.scoreHistory = m.scoreHistory[len(m.scoreHistory)-m.maxHistorySize:]
	}
}

// GetEventCounts returns event counts
func (m *Metrics) GetEventCounts() map[EventType]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[EventType]int64, len(m.eventCounts))
	for et, count := range m.eventCounts {
		result[et] = count
	}
	return result
}

// GetEventRates returns event rates
func (m *Metrics) GetEventRates() map[EventType]float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[EventType]float64, len(m.eventRates))
	for et, rate := range m.eventRates {
		result[et] = rate
	}
	return result
}

// GetBanMetrics returns ban statistics
func (m *Metrics) GetBanMetrics() BanMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	reasons := make(map[string]int64, len(m.banReasons))
	for reason, count := range m.banReasons {
		reasons[reason] = count
	}

	return BanMetrics{
		TotalBans:     m.totalBans,
		TempBans:      m.tempBans,
		PermanentBans: m.permanentBans,
		BanReasons:    reasons,
	}
}

// GetProcessingMetrics returns processing performance metrics
func (m *Metrics) GetProcessingMetrics() ProcessingMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return ProcessingMetrics{
		AvgProcessingTime: m.avgProcessingTime,
		MaxProcessingTime: m.maxProcessingTime,
		ProcessingCount:   m.processingCount,
	}
}

// GetScoreHistory returns score history
func (m *Metrics) GetScoreHistory() []ScoreHistoryPoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]ScoreHistoryPoint, len(m.scoreHistory))
	copy(result, m.scoreHistory)
	return result
}

// BanMetrics holds ban statistics
type BanMetrics struct {
	TotalBans     int64
	TempBans      int64
	PermanentBans int64
	BanReasons    map[string]int64
}

// ProcessingMetrics holds processing performance metrics
type ProcessingMetrics struct {
	AvgProcessingTime time.Duration
	MaxProcessingTime time.Duration
	ProcessingCount   int64
}

// ExportPrometheus exports metrics in Prometheus format
func (m *Metrics) ExportPrometheus() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	output := ""

	// Event counts
	output += "# HELP paw_p2p_reputation_events_total Total number of reputation events by type\n"
	output += "# TYPE paw_p2p_reputation_events_total counter\n"
	for et, count := range m.eventCounts {
		output += fmt.Sprintf("paw_p2p_reputation_events_total{type=\"%s\"} %d\n", et.String(), count)
	}

	// Event rates
	output += "# HELP paw_p2p_reputation_event_rate Events per second by type\n"
	output += "# TYPE paw_p2p_reputation_event_rate gauge\n"
	for et, rate := range m.eventRates {
		output += fmt.Sprintf("paw_p2p_reputation_event_rate{type=\"%s\"} %.2f\n", et.String(), rate)
	}

	// Ban metrics
	output += "# HELP paw_p2p_reputation_bans_total Total number of bans by type\n"
	output += "# TYPE paw_p2p_reputation_bans_total counter\n"
	output += fmt.Sprintf("paw_p2p_reputation_bans_total{type=\"temporary\"} %d\n", m.tempBans)
	output += fmt.Sprintf("paw_p2p_reputation_bans_total{type=\"permanent\"} %d\n", m.permanentBans)

	// Processing time
	output += "# HELP paw_p2p_reputation_processing_seconds Processing time in seconds\n"
	output += "# TYPE paw_p2p_reputation_processing_seconds gauge\n"
	output += fmt.Sprintf("paw_p2p_reputation_processing_seconds{stat=\"avg\"} %.6f\n", m.avgProcessingTime.Seconds())
	output += fmt.Sprintf("paw_p2p_reputation_processing_seconds{stat=\"max\"} %.6f\n", m.maxProcessingTime.Seconds())

	// Peer count
	output += "# HELP paw_p2p_reputation_peers Total number of peers tracked\n"
	output += "# TYPE paw_p2p_reputation_peers gauge\n"
	output += fmt.Sprintf("paw_p2p_reputation_peers %d\n", len(m.peerScores))

	return output
}
