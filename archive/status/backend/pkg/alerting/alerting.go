package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"status/pkg/config"
	"status/pkg/incidents"
)

// AlertManager manages alerting integrations
type AlertManager struct {
	config      *config.Config
	integrations []Integration
	alertQueue  chan *Alert
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// Integration represents an alerting integration
type Integration interface {
	Name() string
	SendAlert(alert *Alert) error
	IsEnabled() bool
}

// Alert represents an alert to be sent
type Alert struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"`
	Component   string                 `json:"component"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
	Incident    *incidents.Incident    `json:"incident,omitempty"`
}

// NewAlertManager creates a new alert manager
func NewAlertManager(cfg *config.Config) *AlertManager {
	ctx, cancel := context.WithCancel(context.Background())

	am := &AlertManager{
		config:      cfg,
		integrations: make([]Integration, 0),
		alertQueue:  make(chan *Alert, 100),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Initialize integrations
	am.initializeIntegrations()

	return am
}

// initializeIntegrations sets up alert integrations
func (am *AlertManager) initializeIntegrations() {
	// Slack integration
	if am.config.SlackWebhookURL != "" {
		am.integrations = append(am.integrations, &SlackIntegration{
			webhookURL: am.config.SlackWebhookURL,
			enabled:    true,
		})
	}

	// PagerDuty integration
	if am.config.PagerDutyIntegrationKey != "" {
		am.integrations = append(am.integrations, &PagerDutyIntegration{
			integrationKey: am.config.PagerDutyIntegrationKey,
			enabled:        true,
		})
	}

	// Email integration
	if am.config.EmailSMTPHost != "" {
		am.integrations = append(am.integrations, &EmailIntegration{
			smtpHost: am.config.EmailSMTPHost,
			smtpPort: am.config.EmailSMTPPort,
			from:     am.config.EmailFrom,
			to:       am.config.EmailTo,
			enabled:  true,
		})
	}

	// Webhook integration
	if am.config.WebhookURL != "" {
		am.integrations = append(am.integrations, &WebhookIntegration{
			url:     am.config.WebhookURL,
			enabled: true,
		})
	}

	log.Printf("Initialized %d alerting integrations", len(am.integrations))
}

// Start begins processing alerts
func (am *AlertManager) Start() {
	log.Println("Starting alert manager")

	for i := 0; i < 3; i++ { // 3 worker goroutines
		go am.processAlerts()
	}
}

// Stop stops the alert manager
func (am *AlertManager) Stop() {
	log.Println("Stopping alert manager")
	am.cancel()
	close(am.alertQueue)
}

// processAlerts processes alerts from the queue
func (am *AlertManager) processAlerts() {
	for {
		select {
		case <-am.ctx.Done():
			return
		case alert, ok := <-am.alertQueue:
			if !ok {
				return
			}
			am.sendAlert(alert)
		}
	}
}

// SendAlert queues an alert to be sent
func (am *AlertManager) SendAlert(alert *Alert) {
	select {
	case am.alertQueue <- alert:
		log.Printf("Queued alert: %s", alert.Title)
	default:
		log.Printf("Alert queue full, dropping alert: %s", alert.Title)
	}
}

// sendAlert sends an alert to all enabled integrations
func (am *AlertManager) sendAlert(alert *Alert) {
	for _, integration := range am.integrations {
		if integration.IsEnabled() {
			go func(i Integration) {
				if err := i.SendAlert(alert); err != nil {
					log.Printf("Failed to send alert via %s: %v", i.Name(), err)
				} else {
					log.Printf("Alert sent successfully via %s", i.Name())
				}
			}(integration)
		}
	}
}

// CreateAlertFromIncident creates an alert from an incident
func (am *AlertManager) CreateAlertFromIncident(incident *incidents.Incident) *Alert {
	severity := "medium"
	switch incident.Severity {
	case incidents.SeverityCritical:
		severity = "critical"
	case incidents.SeverityMajor:
		severity = "high"
	case incidents.SeverityMinor:
		severity = "low"
	}

	component := "System"
	if len(incident.Components) > 0 {
		component = incident.Components[0]
	}

	return &Alert{
		ID:          fmt.Sprintf("incident-%d", incident.ID),
		Title:       incident.Title,
		Description: incident.Description,
		Severity:    severity,
		Component:   component,
		Timestamp:   incident.StartedAt,
		Incident:    incident,
		Metadata: map[string]interface{}{
			"incident_id": incident.ID,
			"status":      incident.Status,
		},
	}
}

// SlackIntegration sends alerts to Slack
type SlackIntegration struct {
	webhookURL string
	enabled    bool
}

func (s *SlackIntegration) Name() string {
	return "Slack"
}

func (s *SlackIntegration) IsEnabled() bool {
	return s.enabled
}

func (s *SlackIntegration) SendAlert(alert *Alert) error {
	color := "warning"
	switch alert.Severity {
	case "critical":
		color = "danger"
	case "high":
		color = "danger"
	case "low":
		color = "good"
	}

	payload := map[string]interface{}{
		"text": fmt.Sprintf("*%s*", alert.Title),
		"attachments": []map[string]interface{}{
			{
				"color": color,
				"fields": []map[string]interface{}{
					{
						"title": "Severity",
						"value": alert.Severity,
						"short": true,
					},
					{
						"title": "Component",
						"value": alert.Component,
						"short": true,
					},
					{
						"title": "Description",
						"value": alert.Description,
						"short": false,
					},
				},
				"footer": "PAW Status Page",
				"ts":     alert.Timestamp.Unix(),
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := http.Post(s.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// PagerDutyIntegration sends alerts to PagerDuty
type PagerDutyIntegration struct {
	integrationKey string
	enabled        bool
}

func (p *PagerDutyIntegration) Name() string {
	return "PagerDuty"
}

func (p *PagerDutyIntegration) IsEnabled() bool {
	return p.enabled
}

func (p *PagerDutyIntegration) SendAlert(alert *Alert) error {
	eventAction := "trigger"
	severity := "warning"

	switch alert.Severity {
	case "critical":
		severity = "critical"
	case "high":
		severity = "error"
	case "low":
		severity = "info"
	}

	payload := map[string]interface{}{
		"routing_key":  p.integrationKey,
		"event_action": eventAction,
		"payload": map[string]interface{}{
			"summary":  alert.Title,
			"severity": severity,
			"source":   "PAW Status Page",
			"component": alert.Component,
			"custom_details": map[string]interface{}{
				"description": alert.Description,
				"alert_id":    alert.ID,
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := http.Post("https://events.pagerduty.com/v2/enqueue", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send to PagerDuty: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PagerDuty returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// EmailIntegration sends alerts via email
type EmailIntegration struct {
	smtpHost string
	smtpPort int
	from     string
	to       []string
	enabled  bool
}

func (e *EmailIntegration) Name() string {
	return "Email"
}

func (e *EmailIntegration) IsEnabled() bool {
	return e.enabled
}

func (e *EmailIntegration) SendAlert(alert *Alert) error {
	// In production, implement actual SMTP email sending
	// For now, just log
	log.Printf("Would send email alert: %s to %v", alert.Title, e.to)
	return nil
}

// WebhookIntegration sends alerts to a generic webhook
type WebhookIntegration struct {
	url     string
	enabled bool
}

func (w *WebhookIntegration) Name() string {
	return "Webhook"
}

func (w *WebhookIntegration) IsEnabled() bool {
	return w.enabled
}

func (w *WebhookIntegration) SendAlert(alert *Alert) error {
	jsonData, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	resp, err := http.Post(w.url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
