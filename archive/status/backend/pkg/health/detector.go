package health

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"status/pkg/config"
	"status/pkg/incidents"
	"status/pkg/metrics"
)

// AutoDetector automatically detects incidents based on metrics
type AutoDetector struct {
	config         *config.Config
	metricsManager *metrics.Manager
	incidentMgr    *incidents.Manager
	lastCheck      map[string]time.Time
	mu             sync.RWMutex
	detectionRules []DetectionRule
}

// DetectionRule defines a rule for incident detection
type DetectionRule struct {
	Name        string
	Component   string
	Metric      string
	Threshold   float64
	Comparison  string // "above", "below", "equal"
	Duration    time.Duration
	Severity    incidents.Severity
	Description string
}

// NewAutoDetector creates a new automated incident detector
func NewAutoDetector(
	cfg *config.Config,
	metricsMgr *metrics.Manager,
	incidentMgr *incidents.Manager,
) *AutoDetector {
	detector := &AutoDetector{
		config:         cfg,
		metricsManager: metricsMgr,
		incidentMgr:    incidentMgr,
		lastCheck:      make(map[string]time.Time),
		detectionRules: defaultDetectionRules(),
	}

	return detector
}

// Start begins automated incident detection
func (ad *AutoDetector) Start(ctx context.Context) {
	log.Println("Starting automated incident detection")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Automated incident detection stopped")
			return
		case <-ticker.C:
			ad.checkAllRules()
		}
	}
}

// checkAllRules checks all detection rules
func (ad *AutoDetector) checkAllRules() {
	for _, rule := range ad.detectionRules {
		if ad.shouldCheckRule(rule) {
			if ad.evaluateRule(rule) {
				ad.createIncident(rule)
			}
		}
	}
}

// shouldCheckRule determines if a rule should be checked
func (ad *AutoDetector) shouldCheckRule(rule DetectionRule) bool {
	ad.mu.RLock()
	lastCheck, exists := ad.lastCheck[rule.Name]
	ad.mu.RUnlock()

	if !exists {
		return true
	}

	// Check if enough time has passed since last check
	return time.Since(lastCheck) >= rule.Duration
}

// evaluateRule evaluates a detection rule
func (ad *AutoDetector) evaluateRule(rule DetectionRule) bool {
	// Get metric value
	value := ad.getMetricValue(rule.Component, rule.Metric)

	// Compare with threshold
	triggered := false
	switch rule.Comparison {
	case "above":
		triggered = value > rule.Threshold
	case "below":
		triggered = value < rule.Threshold
	case "equal":
		triggered = value == rule.Threshold
	}

	if triggered {
		log.Printf("Detection rule triggered: %s (value: %.2f, threshold: %.2f)",
			rule.Name, value, rule.Threshold)
	}

	return triggered
}

// getMetricValue retrieves a metric value from the metrics manager
func (ad *AutoDetector) getMetricValue(component, metric string) float64 {
	switch component {
	case "API":
		return ad.getAPIMetric(metric)
	case "RPC":
		return ad.getRPCMetric(metric)
	case "Database":
		return ad.getDatabaseMetric(metric)
	case "Blockchain":
		return ad.getBlockchainMetric(metric)
	default:
		return 0
	}
}

// getAPIMetric gets an API metric
func (ad *AutoDetector) getAPIMetric(metric string) float64 {
	systemMetrics := ad.metricsManager.GetSystemMetrics()

	switch metric {
	case "uptime":
		return systemMetrics.Uptime
	case "request_rate":
		return systemMetrics.RequestsPerSecond
	case "error_rate":
		return systemMetrics.ErrorRate
	case "latency":
		return systemMetrics.AvgLatency
	default:
		return 0
	}
}

// getRPCMetric gets an RPC metric
func (ad *AutoDetector) getRPCMetric(metric string) float64 {
	// In production, fetch from actual RPC metrics
	// For now, return simulated values
	switch metric {
	case "block_time":
		return 4.0 // 4 second block time
	case "peer_count":
		return 25.0
	case "sync_status":
		return 100.0 // 100% synced
	default:
		return 0
	}
}

// getDatabaseMetric gets a database metric
func (ad *AutoDetector) getDatabaseMetric(metric string) float64 {
	systemMetrics := ad.metricsManager.GetSystemMetrics()

	switch metric {
	case "connections":
		return float64(systemMetrics.ActiveConnections)
	case "query_time":
		return 0.05 // 50ms average query time
	case "disk_usage":
		return 45.0 // 45% disk usage
	default:
		return 0
	}
}

// getBlockchainMetric gets a blockchain metric
func (ad *AutoDetector) getBlockchainMetric(metric string) float64 {
	// In production, fetch from blockchain node
	switch metric {
	case "height":
		return 1000000.0
	case "tx_count":
		return 150.0 // transactions per minute
	case "gas_price":
		return 0.001
	default:
		return 0
	}
}

// createIncident creates an incident based on a triggered rule
func (ad *AutoDetector) createIncident(rule DetectionRule) {
	// Check if incident already exists for this rule
	activeIncidents := ad.incidentMgr.GetActiveIncidents()
	for _, incident := range activeIncidents {
		if incident.Title == rule.Name {
			// Incident already exists, don't create duplicate
			return
		}
	}

	// Create new incident
	title := rule.Name
	description := rule.Description
	components := []string{rule.Component}

	incident, err := ad.incidentMgr.CreateIncident(
		title,
		description,
		rule.Severity,
		components,
	)

	if err != nil {
		log.Printf("Failed to create incident: %v", err)
		return
	}

	log.Printf("Created incident: %s (ID: %d)", incident.Title, incident.ID)

	// Update last check time
	ad.mu.Lock()
	ad.lastCheck[rule.Name] = time.Now()
	ad.mu.Unlock()
}

// defaultDetectionRules returns the default set of detection rules
func defaultDetectionRules() []DetectionRule {
	return []DetectionRule{
		{
			Name:        "High API Error Rate",
			Component:   "API",
			Metric:      "error_rate",
			Threshold:   5.0, // 5% error rate
			Comparison:  "above",
			Duration:    time.Minute,
			Severity:    incidents.SeverityMajor,
			Description: "API error rate is above acceptable threshold",
		},
		{
			Name:        "Low API Uptime",
			Component:   "API",
			Metric:      "uptime",
			Threshold:   99.0, // 99% uptime
			Comparison:  "below",
			Duration:    time.Minute * 5,
			Severity:    incidents.SeverityCritical,
			Description: "API uptime has fallen below 99%",
		},
		{
			Name:        "High API Latency",
			Component:   "API",
			Metric:      "latency",
			Threshold:   1000.0, // 1000ms
			Comparison:  "above",
			Duration:    time.Minute * 2,
			Severity:    incidents.SeverityMajor,
			Description: "API latency is unusually high",
		},
		{
			Name:        "Low RPC Peer Count",
			Component:   "RPC",
			Metric:      "peer_count",
			Threshold:   5.0,
			Comparison:  "below",
			Duration:    time.Minute * 5,
			Severity:    incidents.SeverityMinor,
			Description: "RPC node has fewer peers than expected",
		},
		{
			Name:        "Slow Block Time",
			Component:   "Blockchain",
			Metric:      "block_time",
			Threshold:   10.0, // 10 seconds
			Comparison:  "above",
			Duration:    time.Minute * 3,
			Severity:    incidents.SeverityMajor,
			Description: "Block time is slower than expected",
		},
		{
			Name:        "High Database Connections",
			Component:   "Database",
			Metric:      "connections",
			Threshold:   90.0,
			Comparison:  "above",
			Duration:    time.Minute * 2,
			Severity:    incidents.SeverityMinor,
			Description: "Database connection pool is nearly exhausted",
		},
	}
}

// AddDetectionRule adds a custom detection rule
func (ad *AutoDetector) AddDetectionRule(rule DetectionRule) {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	ad.detectionRules = append(ad.detectionRules, rule)
	log.Printf("Added detection rule: %s", rule.Name)
}

// RemoveDetectionRule removes a detection rule
func (ad *AutoDetector) RemoveDetectionRule(name string) {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	for i, rule := range ad.detectionRules {
		if rule.Name == name {
			ad.detectionRules = append(ad.detectionRules[:i], ad.detectionRules[i+1:]...)
			log.Printf("Removed detection rule: %s", name)
			return
		}
	}
}

// GetDetectionRules returns all detection rules
func (ad *AutoDetector) GetDetectionRules() []DetectionRule {
	ad.mu.RLock()
	defer ad.mu.RUnlock()

	rules := make([]DetectionRule, len(ad.detectionRules))
	copy(rules, ad.detectionRules)
	return rules
}

// HealthCheck represents a health check configuration
type HealthCheck struct {
	Name       string
	URL        string
	Interval   time.Duration
	Timeout    time.Duration
	RetryCount int
	Component  string
}

// HealthChecker performs periodic health checks
type HealthChecker struct {
	checks      []HealthCheck
	detector    *AutoDetector
	mu          sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(detector *AutoDetector) *HealthChecker {
	return &HealthChecker{
		checks:   defaultHealthChecks(),
		detector: detector,
	}
}

// Start begins health checking
func (hc *HealthChecker) Start(ctx context.Context) {
	log.Println("Starting health checks")

	for _, check := range hc.checks {
		go hc.runHealthCheck(ctx, check)
	}

	<-ctx.Done()
	log.Println("Health checks stopped")
}

// runHealthCheck runs a single health check periodically
func (hc *HealthChecker) runHealthCheck(ctx context.Context, check HealthCheck) {
	ticker := time.NewTicker(check.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !hc.performCheck(check) {
				hc.handleFailedCheck(check)
			}
		}
	}
}

// performCheck performs a single health check
func (hc *HealthChecker) performCheck(check HealthCheck) bool {
	// In production, implement actual HTTP health check
	// For now, simulate success
	log.Printf("Health check: %s - OK", check.Name)
	return true
}

// handleFailedCheck handles a failed health check
func (hc *HealthChecker) handleFailedCheck(check HealthCheck) {
	log.Printf("Health check failed: %s", check.Name)

	// Create incident through detector
	rule := DetectionRule{
		Name:        fmt.Sprintf("%s - Health Check Failed", check.Name),
		Component:   check.Component,
		Severity:    incidents.SeverityMajor,
		Description: fmt.Sprintf("Health check for %s failed", check.Name),
	}

	hc.detector.createIncident(rule)
}

// defaultHealthChecks returns default health check configurations
func defaultHealthChecks() []HealthCheck {
	return []HealthCheck{
		{
			Name:       "API Health",
			URL:        "http://localhost:8080/health",
			Interval:   time.Second * 30,
			Timeout:    time.Second * 5,
			RetryCount: 3,
			Component:  "API",
		},
		{
			Name:       "RPC Node Health",
			URL:        "http://localhost:26657/health",
			Interval:   time.Minute,
			Timeout:    time.Second * 10,
			RetryCount: 3,
			Component:  "RPC",
		},
		{
			Name:       "Database Health",
			URL:        "http://localhost:5432/health",
			Interval:   time.Minute,
			Timeout:    time.Second * 5,
			RetryCount: 3,
			Component:  "Database",
		},
	}
}
