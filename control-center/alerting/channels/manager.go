package channels

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/paw/control-center/alerting"
	"github.com/paw/control-center/alerting/storage"
)

// NotificationChannel interface for all notification channels
type NotificationChannel interface {
	Send(alert *alerting.Alert, channel *alerting.Channel) error
	Type() alerting.ChannelType
}

// Manager manages notification channels and sending
type Manager struct {
	storage  *storage.PostgresStorage
	config   *alerting.Config
	channels map[alerting.ChannelType]NotificationChannel
	mu       sync.RWMutex
}

// NewManager creates a new notification manager
func NewManager(storage *storage.PostgresStorage, config *alerting.Config) *Manager {
	m := &Manager{
		storage:  storage,
		config:   config,
		channels: make(map[alerting.ChannelType]NotificationChannel),
	}

	// Initialize channels
	m.channels[alerting.ChannelTypeWebhook] = NewWebhookChannel(&config.WebhookConfig)
	m.channels[alerting.ChannelTypeEmail] = NewEmailChannel(&config.EmailConfig)
	m.channels[alerting.ChannelTypeSMS] = NewSMSChannel(&config.SMSConfig)

	return m
}

// SendAlert sends an alert to all configured channels
func (m *Manager) SendAlert(alert *alerting.Alert) error {
	// Get the alert rule to find which channels to use
	rule, err := m.storage.GetRule(alert.RuleID)
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	// Send to each configured channel
	var errors []error
	for _, channelID := range rule.Channels {
		if err := m.sendToChannel(alert, channelID); err != nil {
			log.Printf("Failed to send alert to channel %s: %v", channelID, err)
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send to %d/%d channels", len(errors), len(rule.Channels))
	}

	return nil
}

// sendToChannel sends an alert to a specific channel
func (m *Manager) sendToChannel(alert *alerting.Alert, channelID string) error {
	// Get channel configuration
	channel, err := m.storage.GetChannel(channelID)
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}

	// Check if channel is enabled
	if !channel.Enabled {
		log.Printf("Channel %s is disabled, skipping", channelID)
		return nil
	}

	// Check channel filters
	if !m.matchesFilters(alert, channel.Filters) {
		log.Printf("Alert does not match channel filters, skipping")
		return nil
	}

	// Get the appropriate notification channel
	m.mu.RLock()
	notifier, exists := m.channels[channel.Type]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("unsupported channel type: %s", channel.Type)
	}

	// Send with retry logic
	return m.sendWithRetry(alert, channel, notifier)
}

// sendWithRetry sends a notification with retry logic
func (m *Manager) sendWithRetry(alert *alerting.Alert, channel *alerting.Channel, notifier NotificationChannel) error {
	var lastErr error
	notification := &alerting.Notification{
		ID:          uuid.New().String(),
		AlertID:     alert.ID,
		ChannelID:   channel.ID,
		ChannelType: channel.Type,
		SentAt:      time.Now(),
		RetryCount:  0,
	}

	for attempt := 0; attempt <= m.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := m.config.RetryBackoff * time.Duration(1<<uint(attempt-1))
			log.Printf("Retrying notification after %s (attempt %d/%d)", backoff, attempt, m.config.MaxRetries)
			time.Sleep(backoff)
			notification.RetryCount = attempt
		}

		err := notifier.Send(alert, channel)
		if err == nil {
			// Success
			notification.Success = true
			m.storage.SaveNotification(notification)
			log.Printf("Successfully sent alert %s to channel %s (%s)", alert.ID, channel.Name, channel.Type)
			return nil
		}

		lastErr = err
		log.Printf("Failed to send notification (attempt %d/%d): %v", attempt+1, m.config.MaxRetries+1, err)
	}

	// All retries failed
	notification.Success = false
	notification.Error = lastErr.Error()
	m.storage.SaveNotification(notification)

	return fmt.Errorf("failed after %d retries: %w", m.config.MaxRetries, lastErr)
}

// matchesFilters checks if an alert matches channel filters
func (m *Manager) matchesFilters(alert *alerting.Alert, filters []alerting.ChannelFilter) bool {
	if len(filters) == 0 {
		return true // No filters = match all
	}

	for _, filter := range filters {
		if !m.matchFilter(alert, filter) {
			return false
		}
	}

	return true
}

// matchFilter checks if an alert matches a single filter
func (m *Manager) matchFilter(alert *alerting.Alert, filter alerting.ChannelFilter) bool {
	var fieldValue string

	switch filter.Field {
	case "severity":
		fieldValue = string(alert.Severity)
	case "source":
		fieldValue = string(alert.Source)
	case "status":
		fieldValue = string(alert.Status)
	case "rule_id":
		fieldValue = alert.RuleID
	default:
		// Check labels
		if val, ok := alert.Labels[filter.Field]; ok {
			fieldValue = val
		} else {
			return false
		}
	}

	switch filter.Operator {
	case "eq":
		return m.contains(filter.Values, fieldValue)
	case "ne":
		return !m.contains(filter.Values, fieldValue)
	case "in":
		return m.contains(filter.Values, fieldValue)
	case "not_in":
		return !m.contains(filter.Values, fieldValue)
	default:
		return false
	}
}

// contains checks if a slice contains a value
func (m *Manager) contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// BatchSend sends multiple alerts efficiently (if batch mode enabled)
func (m *Manager) BatchSend(alerts []*alerting.Alert) error {
	if !m.config.BatchNotifications {
		// Send individually
		for _, alert := range alerts {
			if err := m.SendAlert(alert); err != nil {
				log.Printf("Failed to send alert: %v", err)
			}
		}
		return nil
	}

	// Group alerts by channel
	type channelAlerts struct {
		channel *alerting.Channel
		alerts  []*alerting.Alert
	}

	channelMap := make(map[string]*channelAlerts)

	for _, alert := range alerts {
		rule, err := m.storage.GetRule(alert.RuleID)
		if err != nil {
			continue
		}

		for _, channelID := range rule.Channels {
			if _, exists := channelMap[channelID]; !exists {
				channel, err := m.storage.GetChannel(channelID)
				if err != nil {
					continue
				}
				channelMap[channelID] = &channelAlerts{
					channel: channel,
					alerts:  []*alerting.Alert{},
				}
			}

			channelMap[channelID].alerts = append(channelMap[channelID].alerts, alert)
		}
	}

	// Send batches
	for _, ca := range channelMap {
		if !ca.channel.Enabled {
			continue
		}

		m.mu.RLock()
		notifier := m.channels[ca.channel.Type]
		m.mu.RUnlock()

		if notifier == nil {
			continue
		}

		// Check if channel supports batch sending
		if batchNotifier, ok := notifier.(BatchNotificationChannel); ok {
			// Use batch send for channels that support it
			if err := m.sendBatchWithRetry(ca.alerts, ca.channel, batchNotifier); err != nil {
				log.Printf("Batch send failed for channel %s: %v, falling back to individual", ca.channel.Name, err)
				// Fall back to individual sends
				for _, alert := range ca.alerts {
					m.sendWithRetry(alert, ca.channel, notifier)
				}
			}
		} else {
			// Send individually for channels that don't support batching
			for _, alert := range ca.alerts {
				m.sendWithRetry(alert, ca.channel, notifier)
			}
		}
	}

	return nil
}

// BatchNotificationChannel interface for channels that support batch sending
type BatchNotificationChannel interface {
	NotificationChannel
	SendBatch(alerts []*alerting.Alert, channel *alerting.Channel) error
}

// sendBatchWithRetry sends a batch of notifications with retry logic
func (m *Manager) sendBatchWithRetry(alerts []*alerting.Alert, channel *alerting.Channel, notifier BatchNotificationChannel) error {
	if len(alerts) == 0 {
		return nil
	}

	var lastErr error
	batchNotification := &alerting.Notification{
		ID:          uuid.New().String(),
		AlertID:     fmt.Sprintf("batch-%d-alerts", len(alerts)),
		ChannelID:   channel.ID,
		ChannelType: channel.Type,
		SentAt:      time.Now(),
		RetryCount:  0,
	}

	for attempt := 0; attempt <= m.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := m.config.RetryBackoff * time.Duration(1<<uint(attempt-1))
			log.Printf("Retrying batch notification after %s (attempt %d/%d)", backoff, attempt, m.config.MaxRetries)
			time.Sleep(backoff)
			batchNotification.RetryCount = attempt
		}

		err := notifier.SendBatch(alerts, channel)
		if err == nil {
			// Success
			batchNotification.Success = true
			m.storage.SaveNotification(batchNotification)
			log.Printf("Successfully sent batch of %d alerts to channel %s (%s)", len(alerts), channel.Name, channel.Type)
			return nil
		}

		lastErr = err
		log.Printf("Failed to send batch notification (attempt %d/%d): %v", attempt+1, m.config.MaxRetries+1, err)
	}

	// All retries failed
	batchNotification.Success = false
	batchNotification.Error = lastErr.Error()
	m.storage.SaveNotification(batchNotification)

	return fmt.Errorf("batch send failed after %d retries: %w", m.config.MaxRetries, lastErr)
}

// TestChannel tests a notification channel
func (m *Manager) TestChannel(channelID string) error {
	channel, err := m.storage.GetChannel(channelID)
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}

	// Create a test alert
	testAlert := &alerting.Alert{
		ID:          "test-" + uuid.New().String(),
		RuleID:      "test-rule",
		RuleName:    "Test Alert",
		Source:      alerting.SourceInfrastructure,
		Severity:    alerting.SeverityInfo,
		Status:      alerting.StatusActive,
		Message:     "This is a test alert from PAW Alert Manager",
		Description: "If you received this, the notification channel is working correctly.",
		Value:       100.0,
		Threshold:   50.0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Labels:      map[string]string{"test": "true"},
		Annotations: map[string]string{},
	}

	m.mu.RLock()
	notifier, exists := m.channels[channel.Type]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("unsupported channel type: %s", channel.Type)
	}

	return notifier.Send(testAlert, channel)
}
