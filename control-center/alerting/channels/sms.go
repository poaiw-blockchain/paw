package channels

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/paw/control-center/alerting"
)

// SMSChannel sends alerts via SMS (Twilio)
type SMSChannel struct {
	config *alerting.SMSChannelConfig
	client *http.Client
}

// NewSMSChannel creates a new SMS channel
func NewSMSChannel(config *alerting.SMSChannelConfig) *SMSChannel {
	return &SMSChannel{
		config: config,
		client: &http.Client{},
	}
}

// Send sends an alert via SMS
func (s *SMSChannel) Send(alert *alerting.Alert, channel *alerting.Channel) error {
	// Get recipient phone number from channel config
	to, ok := channel.Config["to"].(string)
	if !ok {
		return fmt.Errorf("SMS recipient not configured")
	}

	// Build message
	message := s.buildMessage(alert)

	// Send via Twilio
	return s.sendTwilioSMS(to, message)
}

// buildMessage builds the SMS message
func (s *SMSChannel) buildMessage(alert *alerting.Alert) string {
	// SMS messages should be concise
	prefix := ""
	switch alert.Severity {
	case alerting.SeverityCritical:
		prefix = "[CRITICAL]"
	case alerting.SeverityWarning:
		prefix = "[WARNING]"
	case alerting.SeverityInfo:
		prefix = "[INFO]"
	}

	// Keep it short (SMS limit is typically 160 characters)
	message := fmt.Sprintf("%s %s: %s", prefix, alert.RuleName, alert.Message)

	// Truncate if too long
	if len(message) > 160 {
		message = message[:157] + "..."
	}

	return message
}

// sendTwilioSMS sends an SMS via Twilio API
func (s *SMSChannel) sendTwilioSMS(to, message string) error {
	// Build API URL
	apiURL := fmt.Sprintf("%s/Accounts/%s/Messages.json",
		s.config.APIEndpoint,
		s.config.AccountSID,
	)

	// Build form data
	data := url.Values{}
	data.Set("To", to)
	data.Set("From", s.config.FromNumber)
	data.Set("Body", message)

	// Create request
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.config.AccountSID, s.config.AuthToken)

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Twilio API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response to check for errors
	var twilioResp TwilioResponse
	if err := json.NewDecoder(resp.Body).Decode(&twilioResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if twilioResp.ErrorCode != 0 {
		return fmt.Errorf("Twilio error %d: %s", twilioResp.ErrorCode, twilioResp.ErrorMessage)
	}

	return nil
}

// Type returns the channel type
func (s *SMSChannel) Type() alerting.ChannelType {
	return alerting.ChannelTypeSMS
}

// TwilioResponse represents a Twilio API response
type TwilioResponse struct {
	SID          string `json:"sid"`
	Status       string `json:"status"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}
