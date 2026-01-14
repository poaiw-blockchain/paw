package channels

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
	"time"

	"github.com/paw/control-center/alerting"
)

// EmailChannel sends alerts via email
type EmailChannel struct {
	config *alerting.EmailChannelConfig
}

// NewEmailChannel creates a new email channel
func NewEmailChannel(config *alerting.EmailChannelConfig) *EmailChannel {
	return &EmailChannel{
		config: config,
	}
}

// Send sends an alert via email
func (e *EmailChannel) Send(alert *alerting.Alert, channel *alerting.Channel) error {
	// Get recipient(s) from channel config
	to, ok := channel.Config["to"].(string)
	if !ok {
		return fmt.Errorf("email recipient not configured")
	}

	recipients := strings.Split(to, ",")
	for i, recipient := range recipients {
		recipients[i] = strings.TrimSpace(recipient)
	}

	// Build email message
	subject := e.buildSubject(alert)
	body := e.buildBody(alert, channel)

	// Send email
	return e.sendEmail(recipients, subject, body)
}

// buildSubject builds the email subject
func (e *EmailChannel) buildSubject(alert *alerting.Alert) string {
	prefix := ""
	switch alert.Severity {
	case alerting.SeverityCritical:
		prefix = "[CRITICAL]"
	case alerting.SeverityWarning:
		prefix = "[WARNING]"
	case alerting.SeverityInfo:
		prefix = "[INFO]"
	}

	return fmt.Sprintf("%s PAW Alert: %s", prefix, alert.RuleName)
}

// buildBody builds the email body
func (e *EmailChannel) buildBody(alert *alerting.Alert, channel *alerting.Channel) string {
	// Check for custom template
	useHTML := true
	if format, ok := channel.Config["format"].(string); ok && format == "text" {
		useHTML = false
	}

	if useHTML {
		return e.buildHTMLBody(alert)
	}
	return e.buildTextBody(alert)
}

// buildTextBody builds a plain text email body
func (e *EmailChannel) buildTextBody(alert *alerting.Alert) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("Alert: %s\n", alert.RuleName))
	buf.WriteString(strings.Repeat("=", 50) + "\n\n")
	buf.WriteString(fmt.Sprintf("Severity: %s\n", alert.Severity))
	buf.WriteString(fmt.Sprintf("Status: %s\n", alert.Status))
	buf.WriteString(fmt.Sprintf("Source: %s\n", alert.Source))
	buf.WriteString(fmt.Sprintf("Message: %s\n\n", alert.Message))

	if alert.Description != "" {
		buf.WriteString(fmt.Sprintf("Description: %s\n\n", alert.Description))
	}

	buf.WriteString(fmt.Sprintf("Value: %.2f\n", alert.Value))
	buf.WriteString(fmt.Sprintf("Threshold: %.2f\n\n", alert.Threshold))

	if len(alert.Labels) > 0 {
		buf.WriteString("Labels:\n")
		for k, v := range alert.Labels {
			buf.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
		buf.WriteString("\n")
	}

	buf.WriteString(fmt.Sprintf("Triggered at: %s\n", alert.CreatedAt.Format(time.RFC1123)))
	buf.WriteString(fmt.Sprintf("Alert ID: %s\n", alert.ID))

	return buf.String()
}

// buildHTMLBody builds an HTML email body
func (e *EmailChannel) buildHTMLBody(alert *alerting.Alert) string {
	tmpl := template.Must(template.New("email").Parse(emailHTMLTemplate))

	data := map[string]interface{}{
		"Alert":         alert,
		"SeverityColor": e.getSeverityColor(alert.Severity),
		"StatusBadge":   e.getStatusBadge(alert.Status),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		// Fall back to text if template fails
		return e.buildTextBody(alert)
	}

	return buf.String()
}

// getSeverityColor returns a color for the severity level
func (e *EmailChannel) getSeverityColor(severity alerting.Severity) string {
	switch severity {
	case alerting.SeverityCritical:
		return "#dc3545" // Red
	case alerting.SeverityWarning:
		return "#ffc107" // Yellow
	case alerting.SeverityInfo:
		return "#17a2b8" // Blue
	default:
		return "#6c757d" // Gray
	}
}

// getStatusBadge returns a status badge style
func (e *EmailChannel) getStatusBadge(status alerting.Status) string {
	switch status {
	case alerting.StatusActive:
		return "background-color: #dc3545; color: white; padding: 4px 8px; border-radius: 4px;"
	case alerting.StatusAcknowledged:
		return "background-color: #ffc107; color: black; padding: 4px 8px; border-radius: 4px;"
	case alerting.StatusResolved:
		return "background-color: #28a745; color: white; padding: 4px 8px; border-radius: 4px;"
	default:
		return "background-color: #6c757d; color: white; padding: 4px 8px; border-radius: 4px;"
	}
}

// sendEmail sends an email using SMTP
func (e *EmailChannel) sendEmail(to []string, subject, body string) error {
	// Build message
	msg := e.buildMessage(to, subject, body)

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", e.config.SMTPHost, e.config.SMTPPort)

	var auth smtp.Auth
	if e.config.Username != "" {
		auth = smtp.PlainAuth("", e.config.Username, e.config.Password, e.config.SMTPHost)
	}

	// Send with TLS if configured
	if e.config.UseTLS {
		return e.sendWithTLS(addr, auth, to, msg)
	}

	// Send with STARTTLS if configured
	if e.config.UseStartTLS {
		return e.sendWithSTARTTLS(addr, auth, to, msg)
	}

	// Send without encryption (not recommended for production)
	return smtp.SendMail(addr, auth, e.config.FromAddress, to, msg)
}

// buildMessage builds the complete email message
func (e *EmailChannel) buildMessage(to []string, subject, body string) []byte {
	var buf bytes.Buffer

	fromName := e.config.FromName
	if fromName == "" {
		fromName = "PAW Alert Manager"
	}

	buf.WriteString(fmt.Sprintf("From: %s <%s>\r\n", fromName, e.config.FromAddress))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(body)

	return buf.Bytes()
}

// sendWithTLS sends email using TLS
func (e *EmailChannel) sendWithTLS(addr string, auth smtp.Auth, to []string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: e.config.SMTPHost,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect with TLS: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, e.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Quit()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	if err := client.Mail(e.config.FromAddress); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return nil
}

// sendWithSTARTTLS sends email using STARTTLS
func (e *EmailChannel) sendWithSTARTTLS(addr string, auth smtp.Auth, to []string, msg []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Quit()

	if err := client.StartTLS(&tls.Config{ServerName: e.config.SMTPHost}); err != nil {
		return fmt.Errorf("STARTTLS failed: %w", err)
	}

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	if err := client.Mail(e.config.FromAddress); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return w.Close()
}

// Type returns the channel type
func (e *EmailChannel) Type() alerting.ChannelType {
	return alerting.ChannelTypeEmail
}

// emailHTMLTemplate is the HTML template for emails
const emailHTMLTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body {
            font-family: Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .header {
            background-color: {{ .SeverityColor }};
            color: white;
            padding: 20px;
            border-radius: 8px 8px 0 0;
        }
        .content {
            background-color: #f8f9fa;
            padding: 20px;
            border: 1px solid #dee2e6;
            border-top: none;
            border-radius: 0 0 8px 8px;
        }
        .field {
            margin: 10px 0;
        }
        .label {
            font-weight: bold;
            display: inline-block;
            width: 120px;
        }
        .value {
            display: inline-block;
        }
        .footer {
            margin-top: 20px;
            padding-top: 20px;
            border-top: 1px solid #dee2e6;
            font-size: 12px;
            color: #6c757d;
        }
        .badge {
            {{ .StatusBadge }}
        }
    </style>
</head>
<body>
    <div class="header">
        <h2>{{ .Alert.RuleName }}</h2>
        <p>{{ .Alert.Message }}</p>
    </div>
    <div class="content">
        <div class="field">
            <span class="label">Severity:</span>
            <span class="value">{{ .Alert.Severity }}</span>
        </div>
        <div class="field">
            <span class="label">Status:</span>
            <span class="value badge">{{ .Alert.Status }}</span>
        </div>
        <div class="field">
            <span class="label">Source:</span>
            <span class="value">{{ .Alert.Source }}</span>
        </div>
        {{ if .Alert.Description }}
        <div class="field">
            <span class="label">Description:</span>
            <span class="value">{{ .Alert.Description }}</span>
        </div>
        {{ end }}
        <div class="field">
            <span class="label">Value:</span>
            <span class="value">{{ printf "%.2f" .Alert.Value }}</span>
        </div>
        <div class="field">
            <span class="label">Threshold:</span>
            <span class="value">{{ printf "%.2f" .Alert.Threshold }}</span>
        </div>
        <div class="field">
            <span class="label">Triggered at:</span>
            <span class="value">{{ .Alert.CreatedAt.Format "2006-01-02 15:04:05 MST" }}</span>
        </div>
    </div>
    <div class="footer">
        <p>Alert ID: {{ .Alert.ID }}</p>
        <p>This is an automated alert from PAW Alert Manager</p>
    </div>
</body>
</html>
`
