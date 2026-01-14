package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/paw/control-center/alerting"
	"github.com/paw/control-center/alerting/storage"
)

// RulesEngine manages alert rule evaluation
type RulesEngine struct {
	storage       *storage.PostgresStorage
	evaluator     *Evaluator
	config        *alerting.Config
	activeRules   map[string]*alerting.AlertRule
	ruleTimers    map[string]*time.Ticker
	alertHandlers []AlertHandler
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	deduplicator  *Deduplicator
	grouper       *AlertGrouper
}

// AlertHandler is called when an alert is triggered
type AlertHandler func(alert *alerting.Alert) error

// NewRulesEngine creates a new rules engine
func NewRulesEngine(storage *storage.PostgresStorage, evaluator *Evaluator, config *alerting.Config) *RulesEngine {
	ctx, cancel := context.WithCancel(context.Background())

	engine := &RulesEngine{
		storage:       storage,
		evaluator:     evaluator,
		config:        config,
		activeRules:   make(map[string]*alerting.AlertRule),
		ruleTimers:    make(map[string]*time.Ticker),
		alertHandlers: []AlertHandler{},
		ctx:           ctx,
		cancel:        cancel,
	}

	if config.EnableDeduplication {
		engine.deduplicator = NewDeduplicator(config.DeduplicationWindow)
	}

	if config.EnableGrouping {
		engine.grouper = NewAlertGrouper(config.GroupingWindow)
		// Wire up the grouper to use handleAlert for merged alerts
		engine.grouper.SetHandler(func(mergedAlert *alerting.Alert, originalAlerts []*alerting.Alert) error {
			return engine.handleAlert(mergedAlert)
		})
	}

	return engine
}

// Start starts the rules engine
func (e *RulesEngine) Start() error {
	log.Println("Starting rules engine...")

	// Load all enabled rules from storage
	rules, err := e.storage.ListRules(true)
	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	log.Printf("Loaded %d active rules", len(rules))

	// Start evaluation loops for each rule
	for _, rule := range rules {
		if err := e.AddRule(rule); err != nil {
			log.Printf("Failed to add rule %s: %v", rule.ID, err)
		}
	}

	return nil
}

// Stop stops the rules engine
func (e *RulesEngine) Stop() {
	log.Println("Stopping rules engine...")
	e.cancel()

	e.mu.Lock()
	defer e.mu.Unlock()

	// Stop all rule timers
	for _, ticker := range e.ruleTimers {
		ticker.Stop()
	}

	e.ruleTimers = make(map[string]*time.Ticker)
	e.activeRules = make(map[string]*alerting.AlertRule)
}

// AddRule adds a rule to the engine
func (e *RulesEngine) AddRule(rule *alerting.AlertRule) error {
	if !rule.Enabled {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Stop existing timer if any
	if ticker, exists := e.ruleTimers[rule.ID]; exists {
		ticker.Stop()
	}

	// Store rule
	e.activeRules[rule.ID] = rule

	// Create evaluation ticker
	ticker := time.NewTicker(rule.EvaluationInterval)
	e.ruleTimers[rule.ID] = ticker

	// Start evaluation goroutine
	go e.evaluateRuleLoop(rule, ticker)

	log.Printf("Added rule: %s (interval: %s)", rule.Name, rule.EvaluationInterval)

	return nil
}

// RemoveRule removes a rule from the engine
func (e *RulesEngine) RemoveRule(ruleID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if ticker, exists := e.ruleTimers[ruleID]; exists {
		ticker.Stop()
		delete(e.ruleTimers, ruleID)
	}

	delete(e.activeRules, ruleID)
	log.Printf("Removed rule: %s", ruleID)
}

// UpdateRule updates an existing rule
func (e *RulesEngine) UpdateRule(rule *alerting.AlertRule) error {
	e.RemoveRule(rule.ID)
	return e.AddRule(rule)
}

// RegisterAlertHandler registers a handler for alerts
func (e *RulesEngine) RegisterAlertHandler(handler AlertHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.alertHandlers = append(e.alertHandlers, handler)
}

// evaluateRuleLoop continuously evaluates a rule
func (e *RulesEngine) evaluateRuleLoop(rule *alerting.AlertRule, ticker *time.Ticker) {
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.evaluateRule(rule)
		}
	}
}

// evaluateRule evaluates a single rule
func (e *RulesEngine) evaluateRule(rule *alerting.AlertRule) {
	result, err := e.evaluator.Evaluate(rule)
	if err != nil {
		log.Printf("Error evaluating rule %s: %v", rule.Name, err)
		return
	}

	if result.Triggered {
		alert := e.createAlert(rule, result)

		// Check deduplication
		if e.config.EnableDeduplication {
			if e.deduplicator.IsDuplicate(alert) {
				log.Printf("Alert deduplicated: %s", alert.ID)
				return
			}
			e.deduplicator.Add(alert)
		}

		// Group alerts if enabled
		if e.config.EnableGrouping {
			e.grouper.Add(alert)
			// Alerts will be flushed by grouper after grouping window
			return
		}

		// Save and handle alert immediately
		if err := e.handleAlert(alert); err != nil {
			log.Printf("Error handling alert: %v", err)
		}
	}
}

// createAlert creates an alert from evaluation result
func (e *RulesEngine) createAlert(rule *alerting.AlertRule, result *alerting.EvaluationResult) *alerting.Alert {
	alert := &alerting.Alert{
		ID:          e.generateAlertID(rule, result),
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		Source:      rule.Source,
		Severity:    rule.Severity,
		Status:      alerting.StatusActive,
		Message:     result.Message,
		Description: rule.Description,
		Labels:      rule.Labels,
		Annotations: rule.Annotations,
		Value:       result.Value,
		Threshold:   result.Threshold,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    result.Metadata,
	}

	return alert
}

// generateAlertID generates a unique alert ID based on rule and conditions
func (e *RulesEngine) generateAlertID(rule *alerting.AlertRule, result *alerting.EvaluationResult) string {
	// Create a deterministic ID based on rule ID and labels
	// This allows us to deduplicate identical alerts
	hash := sha256.New()
	hash.Write([]byte(rule.ID))
	hash.Write([]byte(result.Timestamp.Format(time.RFC3339)))

	for k, v := range rule.Labels {
		hash.Write([]byte(k))
		hash.Write([]byte(v))
	}

	return hex.EncodeToString(hash.Sum(nil))[:16]
}

// handleAlert saves alert and calls handlers
func (e *RulesEngine) handleAlert(alert *alerting.Alert) error {
	// Save to storage
	if err := e.storage.SaveAlert(alert); err != nil {
		return fmt.Errorf("failed to save alert: %w", err)
	}

	log.Printf("Alert triggered: %s - %s", alert.RuleName, alert.Message)

	// Call registered handlers
	e.mu.RLock()
	handlers := e.alertHandlers
	e.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(alert); err != nil {
			log.Printf("Alert handler error: %v", err)
		}
	}

	return nil
}

// Deduplicator prevents duplicate alerts within a time window
type Deduplicator struct {
	window time.Duration
	alerts map[string]time.Time
	mu     sync.RWMutex
}

// NewDeduplicator creates a new deduplicator
func NewDeduplicator(window time.Duration) *Deduplicator {
	d := &Deduplicator{
		window: window,
		alerts: make(map[string]time.Time),
	}

	// Clean up old entries periodically
	go d.cleanup()

	return d
}

// IsDuplicate checks if alert is a duplicate
func (d *Deduplicator) IsDuplicate(alert *alerting.Alert) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	key := d.getKey(alert)
	lastSeen, exists := d.alerts[key]

	if !exists {
		return false
	}

	return time.Since(lastSeen) < d.window
}

// Add adds an alert to the deduplicator
func (d *Deduplicator) Add(alert *alerting.Alert) {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := d.getKey(alert)
	d.alerts[key] = time.Now()
}

// getKey generates a key for deduplication
func (d *Deduplicator) getKey(alert *alerting.Alert) string {
	// Use rule ID + severity + source as dedup key
	return fmt.Sprintf("%s:%s:%s", alert.RuleID, alert.Severity, alert.Source)
}

// cleanup removes old entries
func (d *Deduplicator) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		d.mu.Lock()
		now := time.Now()
		for key, timestamp := range d.alerts {
			if now.Sub(timestamp) > d.window*2 {
				delete(d.alerts, key)
			}
		}
		d.mu.Unlock()
	}
}

// AlertGrouper groups similar alerts together
type AlertGrouper struct {
	window       time.Duration
	groups       map[string][]*alerting.Alert
	mu           sync.RWMutex
	flusher      chan string
	flushHandler GroupedAlertHandler
}

// GroupedAlertHandler handles a group of merged alerts
type GroupedAlertHandler func(mergedAlert *alerting.Alert, originalAlerts []*alerting.Alert) error

// NewAlertGrouper creates a new alert grouper
func NewAlertGrouper(window time.Duration) *AlertGrouper {
	g := &AlertGrouper{
		window:  window,
		groups:  make(map[string][]*alerting.Alert),
		flusher: make(chan string, 100),
	}

	go g.flushLoop()

	return g
}

// SetHandler sets the handler for grouped alerts
func (g *AlertGrouper) SetHandler(handler GroupedAlertHandler) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.flushHandler = handler
}

// Add adds an alert to a group
func (g *AlertGrouper) Add(alert *alerting.Alert) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := g.getGroupKey(alert)

	if _, exists := g.groups[key]; !exists {
		// New group - schedule flush
		go g.scheduleFlush(key)
	}

	g.groups[key] = append(g.groups[key], alert)
}

// getGroupKey generates a grouping key
func (g *AlertGrouper) getGroupKey(alert *alerting.Alert) string {
	// Group by rule ID and severity
	return fmt.Sprintf("%s:%s", alert.RuleID, alert.Severity)
}

// scheduleFlush schedules a group to be flushed
func (g *AlertGrouper) scheduleFlush(key string) {
	time.Sleep(g.window)
	g.flusher <- key
}

// flushLoop processes group flushes
func (g *AlertGrouper) flushLoop() {
	for key := range g.flusher {
		g.mu.Lock()
		alerts, exists := g.groups[key]
		handler := g.flushHandler
		if exists {
			delete(g.groups, key)
		}
		g.mu.Unlock()

		if exists && len(alerts) > 0 {
			mergedAlert := g.mergeAlerts(alerts)
			log.Printf("Grouped %d alerts for key %s into merged alert %s", len(alerts), key, mergedAlert.ID)

			if handler != nil {
				if err := handler(mergedAlert, alerts); err != nil {
					log.Printf("Error handling grouped alert: %v", err)
				}
			}
		}
	}
}

// mergeAlerts combines multiple alerts into a single summary alert
func (g *AlertGrouper) mergeAlerts(alerts []*alerting.Alert) *alerting.Alert {
	if len(alerts) == 0 {
		return nil
	}

	if len(alerts) == 1 {
		return alerts[0]
	}

	// Use the first alert as the base
	first := alerts[0]

	// Find highest severity among alerts
	highestSeverity := first.Severity
	for _, alert := range alerts[1:] {
		if severityPriority(alert.Severity) > severityPriority(highestSeverity) {
			highestSeverity = alert.Severity
		}
	}

	// Aggregate values
	var totalValue, maxValue, minValue float64
	minValue = alerts[0].Value
	maxValue = alerts[0].Value
	for _, alert := range alerts {
		totalValue += alert.Value
		if alert.Value > maxValue {
			maxValue = alert.Value
		}
		if alert.Value < minValue {
			minValue = alert.Value
		}
	}
	avgValue := totalValue / float64(len(alerts))

	// Find earliest and latest timestamps
	earliest := alerts[0].CreatedAt
	latest := alerts[0].CreatedAt
	for _, alert := range alerts[1:] {
		if alert.CreatedAt.Before(earliest) {
			earliest = alert.CreatedAt
		}
		if alert.CreatedAt.After(latest) {
			latest = alert.CreatedAt
		}
	}

	// Create merged alert ID
	hash := sha256.New()
	hash.Write([]byte(first.RuleID))
	hash.Write([]byte(fmt.Sprintf("%d", len(alerts))))
	hash.Write([]byte(earliest.Format(time.RFC3339)))
	mergedID := "grp-" + hex.EncodeToString(hash.Sum(nil))[:12]

	// Build summary message
	summaryMessage := fmt.Sprintf("[GROUPED] %d alerts from rule '%s': avg=%.2f, min=%.2f, max=%.2f",
		len(alerts), first.RuleName, avgValue, minValue, maxValue)

	// Collect unique sources
	sources := make(map[alerting.AlertSource]int)
	for _, alert := range alerts {
		sources[alert.Source]++
	}

	// Build metadata with aggregation info
	metadata := map[string]interface{}{
		"grouped":       true,
		"alert_count":   len(alerts),
		"avg_value":     avgValue,
		"min_value":     minValue,
		"max_value":     maxValue,
		"first_alert":   earliest.Format(time.RFC3339),
		"last_alert":    latest.Format(time.RFC3339),
		"source_counts": sources,
	}

	// Include IDs of original alerts
	alertIDs := make([]string, len(alerts))
	for i, alert := range alerts {
		alertIDs[i] = alert.ID
	}
	metadata["original_alert_ids"] = alertIDs

	return &alerting.Alert{
		ID:          mergedID,
		RuleID:      first.RuleID,
		RuleName:    first.RuleName,
		Source:      first.Source,
		Severity:    highestSeverity,
		Status:      alerting.StatusActive,
		Message:     summaryMessage,
		Description: fmt.Sprintf("Grouped alert containing %d individual alerts", len(alerts)),
		Labels:      first.Labels,
		Annotations: first.Annotations,
		Value:       avgValue,
		Threshold:   first.Threshold,
		CreatedAt:   earliest,
		UpdatedAt:   time.Now(),
		Metadata:    metadata,
	}
}

// severityPriority returns a numeric priority for severity comparison
func severityPriority(s alerting.Severity) int {
	switch s {
	case alerting.SeverityCritical:
		return 4
	case alerting.SeverityWarning:
		return 3
	case alerting.SeverityInfo:
		return 2
	default:
		return 1
	}
}
