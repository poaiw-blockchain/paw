package channels

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/paw/control-center/alerting"
)

// WebhookChannel sends alerts via HTTP webhooks
type WebhookChannel struct {
	config *alerting.WebhookChannelConfig
	client *http.Client
}

// NewWebhookChannel creates a new webhook channel
func NewWebhookChannel(config *alerting.WebhookChannelConfig) *WebhookChannel {
	client := &http.Client{
		Timeout: config.Timeout,
	}

	if config.InsecureSkipVerify {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	return &WebhookChannel{
		config: config,
		client: client,
	}
}

// Send sends an alert via webhook
func (w *WebhookChannel) Send(alert *alerting.Alert, channel *alerting.Channel) error {
	// Get webhook URL from channel config
	webhookURL, ok := channel.Config["url"].(string)
	if !ok {
		return fmt.Errorf("webhook URL not configured")
	}

	// Prepare payload
	payload := w.buildPayload(alert, channel)

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "PAW-AlertManager/1.0")

	// Add default headers from config
	for key, value := range w.config.DefaultHeaders {
		req.Header.Set(key, value)
	}

	// Add custom headers from channel config
	if headers, ok := channel.Config["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	// Send request
	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// buildPayload builds the webhook payload
func (w *WebhookChannel) buildPayload(alert *alerting.Alert, channel *alerting.Channel) map[string]interface{} {
	// Check for custom template
	if template, ok := channel.Config["template"].(string); ok {
		switch template {
		case "pagerduty":
			return w.buildPagerDutyPayload(alert)
		case "slack":
			return w.buildSlackPayload(alert)
		case "discord":
			return w.buildDiscordPayload(alert)
		}
	}

	// Default generic payload
	return map[string]interface{}{
		"alert_id":    alert.ID,
		"rule_id":     alert.RuleID,
		"rule_name":   alert.RuleName,
		"source":      alert.Source,
		"severity":    alert.Severity,
		"status":      alert.Status,
		"message":     alert.Message,
		"description": alert.Description,
		"value":       alert.Value,
		"threshold":   alert.Threshold,
		"labels":      alert.Labels,
		"annotations": alert.Annotations,
		"created_at":  alert.CreatedAt,
		"updated_at":  alert.UpdatedAt,
		"metadata":    alert.Metadata,
	}
}

// buildPagerDutyPayload builds a PagerDuty-compatible payload
func (w *WebhookChannel) buildPagerDutyPayload(alert *alerting.Alert) map[string]interface{} {
	severity := "error"
	if alert.Severity == alerting.SeverityCritical {
		severity = "critical"
	} else if alert.Severity == alerting.SeverityWarning {
		severity = "warning"
	}

	return map[string]interface{}{
		"routing_key":  "", // Should be provided in channel config
		"event_action": "trigger",
		"dedup_key":    alert.ID,
		"payload": map[string]interface{}{
			"summary":        alert.Message,
			"source":         string(alert.Source),
			"severity":       severity,
			"timestamp":      alert.CreatedAt.Format(time.RFC3339),
			"component":      alert.RuleName,
			"custom_details": alert.Metadata,
		},
	}
}

// buildSlackPayload builds a Slack-compatible payload
func (w *WebhookChannel) buildSlackPayload(alert *alerting.Alert) map[string]interface{} {
	color := "warning"
	emoji := ":warning:"

	switch alert.Severity {
	case alerting.SeverityCritical:
		color = "danger"
		emoji = ":rotating_light:"
	case alerting.SeverityInfo:
		color = "good"
		emoji = ":information_source:"
	}

	return map[string]interface{}{
		"username": "PAW Alert Manager",
		"icon_emoji": emoji,
		"attachments": []map[string]interface{}{
			{
				"color":      color,
				"title":      fmt.Sprintf("%s Alert: %s", alert.Severity, alert.RuleName),
				"text":       alert.Message,
				"footer":     string(alert.Source),
				"ts":         alert.CreatedAt.Unix(),
				"fields": []map[string]interface{}{
					{
						"title": "Severity",
						"value": string(alert.Severity),
						"short": true,
					},
					{
						"title": "Status",
						"value": string(alert.Status),
						"short": true,
					},
					{
						"title": "Value",
						"value": fmt.Sprintf("%.2f (threshold: %.2f)", alert.Value, alert.Threshold),
						"short": false,
					},
				},
			},
		},
	}
}

// buildDiscordPayload builds a Discord-compatible payload
func (w *WebhookChannel) buildDiscordPayload(alert *alerting.Alert) map[string]interface{} {
	color := 16776960 // Yellow for warning

	switch alert.Severity {
	case alerting.SeverityCritical:
		color = 16711680 // Red
	case alerting.SeverityInfo:
		color = 3447003 // Blue
	}

	return map[string]interface{}{
		"username": "PAW Alert Manager",
		"embeds": []map[string]interface{}{
			{
				"title":       fmt.Sprintf("%s Alert: %s", alert.Severity, alert.RuleName),
				"description": alert.Message,
				"color":       color,
				"timestamp":   alert.CreatedAt.Format(time.RFC3339),
				"footer": map[string]interface{}{
					"text": string(alert.Source),
				},
				"fields": []map[string]interface{}{
					{
						"name":   "Severity",
						"value":  string(alert.Severity),
						"inline": true,
					},
					{
						"name":   "Status",
						"value":  string(alert.Status),
						"inline": true,
					},
					{
						"name":   "Value",
						"value":  fmt.Sprintf("%.2f (threshold: %.2f)", alert.Value, alert.Threshold),
						"inline": false,
					},
				},
			},
		},
	}
}

// Type returns the channel type
func (w *WebhookChannel) Type() alerting.ChannelType {
	return alerting.ChannelTypeWebhook
}

// SendBatch sends multiple alerts in a single request
func (w *WebhookChannel) SendBatch(alerts []*alerting.Alert, channel *alerting.Channel) error {
	if len(alerts) == 0 {
		return nil
	}

	// Get webhook URL from channel config
	webhookURL, ok := channel.Config["url"].(string)
	if !ok {
		return fmt.Errorf("webhook URL not configured")
	}

	// Build batch payload
	payload := w.buildBatchPayload(alerts, channel)

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal batch payload: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "PAW-AlertManager/1.0")
	req.Header.Set("X-PAW-Batch", "true")
	req.Header.Set("X-PAW-Alert-Count", fmt.Sprintf("%d", len(alerts)))

	// Add default headers from config
	for key, value := range w.config.DefaultHeaders {
		req.Header.Set(key, value)
	}

	// Add custom headers from channel config
	if headers, ok := channel.Config["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	// Send request
	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send batch webhook: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("batch webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// buildBatchPayload builds a batch webhook payload
func (w *WebhookChannel) buildBatchPayload(alerts []*alerting.Alert, channel *alerting.Channel) map[string]interface{} {
	// Check for custom template
	if template, ok := channel.Config["template"].(string); ok {
		switch template {
		case "slack":
			return w.buildSlackBatchPayload(alerts)
		case "discord":
			return w.buildDiscordBatchPayload(alerts)
		}
	}

	// Default generic batch payload
	alertPayloads := make([]map[string]interface{}, len(alerts))
	for i, alert := range alerts {
		alertPayloads[i] = w.buildPayload(alert, channel)
	}

	return map[string]interface{}{
		"batch":      true,
		"alert_count": len(alerts),
		"alerts":     alertPayloads,
		"sent_at":    time.Now().Format(time.RFC3339),
	}
}

// buildSlackBatchPayload builds a Slack-compatible batch payload
func (w *WebhookChannel) buildSlackBatchPayload(alerts []*alerting.Alert) map[string]interface{} {
	attachments := make([]map[string]interface{}, 0, len(alerts))

	// Group by severity for summary
	severityCounts := make(map[alerting.Severity]int)
	for _, alert := range alerts {
		severityCounts[alert.Severity]++
	}

	// Create summary attachment
	summaryText := fmt.Sprintf("*%d alerts triggered*\n", len(alerts))
	for sev, count := range severityCounts {
		summaryText += fmt.Sprintf("• %s: %d\n", sev, count)
	}

	attachments = append(attachments, map[string]interface{}{
		"color": "warning",
		"text":  summaryText,
	})

	// Add individual alerts (limit to first 10 to avoid payload size issues)
	maxAlerts := 10
	if len(alerts) < maxAlerts {
		maxAlerts = len(alerts)
	}

	for _, alert := range alerts[:maxAlerts] {
		color := "warning"
		if alert.Severity == alerting.SeverityCritical {
			color = "danger"
		} else if alert.Severity == alerting.SeverityInfo {
			color = "good"
		}

		attachments = append(attachments, map[string]interface{}{
			"color": color,
			"title": alert.RuleName,
			"text":  alert.Message,
			"ts":    alert.CreatedAt.Unix(),
		})
	}

	if len(alerts) > 10 {
		attachments = append(attachments, map[string]interface{}{
			"text": fmt.Sprintf("_...and %d more alerts_", len(alerts)-10),
		})
	}

	return map[string]interface{}{
		"username":    "PAW Alert Manager",
		"icon_emoji":  ":bell:",
		"attachments": attachments,
	}
}

// buildDiscordBatchPayload builds a Discord-compatible batch payload
func (w *WebhookChannel) buildDiscordBatchPayload(alerts []*alerting.Alert) map[string]interface{} {
	embeds := make([]map[string]interface{}, 0, len(alerts)+1)

	// Summary embed
	severityCounts := make(map[alerting.Severity]int)
	for _, alert := range alerts {
		severityCounts[alert.Severity]++
	}

	description := fmt.Sprintf("**%d alerts triggered**\n", len(alerts))
	for sev, count := range severityCounts {
		description += fmt.Sprintf("• %s: %d\n", sev, count)
	}

	embeds = append(embeds, map[string]interface{}{
		"title":       "PAW Alert Summary",
		"description": description,
		"color":       16776960, // Yellow
		"timestamp":   time.Now().Format(time.RFC3339),
	})

	// Add individual alerts (Discord limit: 10 embeds per message)
	maxAlerts := 9 // Leave room for summary
	if len(alerts) < maxAlerts {
		maxAlerts = len(alerts)
	}

	for _, alert := range alerts[:maxAlerts] {
		color := 16776960 // Yellow for warning
		if alert.Severity == alerting.SeverityCritical {
			color = 16711680 // Red
		} else if alert.Severity == alerting.SeverityInfo {
			color = 3447003 // Blue
		}

		embeds = append(embeds, map[string]interface{}{
			"title":       alert.RuleName,
			"description": alert.Message,
			"color":       color,
			"timestamp":   alert.CreatedAt.Format(time.RFC3339),
		})
	}

	return map[string]interface{}{
		"username": "PAW Alert Manager",
		"embeds":   embeds,
	}
}
