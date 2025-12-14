package alerting

import (
	"time"
)

// Severity defines the severity level of an alert
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Status defines the status of an alert
type Status string

const (
	StatusActive       Status = "active"
	StatusAcknowledged Status = "acknowledged"
	StatusResolved     Status = "resolved"
)

// AlertSource defines the source of an alert
type AlertSource string

const (
	SourceNetworkHealth   AlertSource = "network_health"
	SourceSecurity        AlertSource = "security"
	SourcePerformance     AlertSource = "performance"
	SourceModuleDEX       AlertSource = "module_dex"
	SourceModuleOracle    AlertSource = "module_oracle"
	SourceModuleCompute   AlertSource = "module_compute"
	SourceInfrastructure  AlertSource = "infrastructure"
)

// Alert represents an active alert
type Alert struct {
	ID            string                 `json:"id"`
	RuleID        string                 `json:"rule_id"`
	RuleName      string                 `json:"rule_name"`
	Source        AlertSource            `json:"source"`
	Severity      Severity               `json:"severity"`
	Status        Status                 `json:"status"`
	Message       string                 `json:"message"`
	Description   string                 `json:"description"`
	Labels        map[string]string      `json:"labels"`
	Annotations   map[string]string      `json:"annotations"`
	Value         float64                `json:"value"`
	Threshold     float64                `json:"threshold"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	AcknowledgedAt *time.Time            `json:"acknowledged_at,omitempty"`
	AcknowledgedBy string                `json:"acknowledged_by,omitempty"`
	ResolvedAt    *time.Time             `json:"resolved_at,omitempty"`
	ResolvedBy    string                 `json:"resolved_by,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AlertRule defines a rule that triggers alerts
type AlertRule struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Source          AlertSource            `json:"source"`
	Severity        Severity               `json:"severity"`
	Enabled         bool                   `json:"enabled"`
	RuleType        RuleType               `json:"rule_type"`
	Conditions      []Condition            `json:"conditions"`
	CompositeOp     CompositeOperator      `json:"composite_op,omitempty"` // For composite rules
	EvaluationInterval time.Duration       `json:"evaluation_interval"`
	ForDuration     time.Duration          `json:"for_duration"` // Alert must be active for this duration
	Labels          map[string]string      `json:"labels"`
	Annotations     map[string]string      `json:"annotations"`
	Channels        []string               `json:"channels"` // Channel IDs to notify
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// RuleType defines the type of alert rule
type RuleType string

const (
	RuleTypeThreshold    RuleType = "threshold"
	RuleTypeRateOfChange RuleType = "rate_of_change"
	RuleTypePattern      RuleType = "pattern"
	RuleTypeComposite    RuleType = "composite"
)

// Condition defines a single condition in an alert rule
type Condition struct {
	MetricName string             `json:"metric_name"`
	Operator   ComparisonOperator `json:"operator"`
	Threshold  float64            `json:"threshold"`
	Duration   time.Duration      `json:"duration,omitempty"` // For rate of change
}

// ComparisonOperator defines comparison operators for conditions
type ComparisonOperator string

const (
	OpGreaterThan       ComparisonOperator = "gt"
	OpGreaterThanOrEqual ComparisonOperator = "gte"
	OpLessThan          ComparisonOperator = "lt"
	OpLessThanOrEqual   ComparisonOperator = "lte"
	OpEquals            ComparisonOperator = "eq"
	OpNotEquals         ComparisonOperator = "ne"
)

// CompositeOperator defines logical operators for composite rules
type CompositeOperator string

const (
	OpAND CompositeOperator = "AND"
	OpOR  CompositeOperator = "OR"
)

// Channel defines a notification channel
type Channel struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        ChannelType            `json:"type"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config"`
	Filters     []ChannelFilter        `json:"filters,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ChannelType defines the type of notification channel
type ChannelType string

const (
	ChannelTypeWebhook ChannelType = "webhook"
	ChannelTypeEmail   ChannelType = "email"
	ChannelTypeSMS     ChannelType = "sms"
	ChannelTypeSlack   ChannelType = "slack"
	ChannelTypeDiscord ChannelType = "discord"
)

// ChannelFilter defines filters for channel routing
type ChannelFilter struct {
	Field    string   `json:"field"`    // severity, source, etc.
	Operator string   `json:"operator"` // eq, ne, in, etc.
	Values   []string `json:"values"`
}

// AlertStats represents alert statistics
type AlertStats struct {
	TotalAlerts       int              `json:"total_alerts"`
	ActiveAlerts      int              `json:"active_alerts"`
	AcknowledgedAlerts int             `json:"acknowledged_alerts"`
	ResolvedAlerts    int              `json:"resolved_alerts"`
	BySeverity        map[Severity]int `json:"by_severity"`
	BySource          map[AlertSource]int `json:"by_source"`
	ByStatus          map[Status]int   `json:"by_status"`
	MeanTimeToAcknowledge time.Duration `json:"mean_time_to_acknowledge"`
	MeanTimeToResolve     time.Duration `json:"mean_time_to_resolve"`
}

// Notification represents a notification sent to a channel
type Notification struct {
	ID          string      `json:"id"`
	AlertID     string      `json:"alert_id"`
	ChannelID   string      `json:"channel_id"`
	ChannelType ChannelType `json:"channel_type"`
	SentAt      time.Time   `json:"sent_at"`
	Success     bool        `json:"success"`
	Error       string      `json:"error,omitempty"`
	RetryCount  int         `json:"retry_count"`
}

// EscalationPolicy defines how alerts should be escalated
type EscalationPolicy struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Enabled     bool                `json:"enabled"`
	Levels      []EscalationLevel   `json:"levels"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// EscalationLevel defines a level in an escalation policy
type EscalationLevel struct {
	Level       int           `json:"level"`
	DelayAfter  time.Duration `json:"delay_after"` // Delay after previous level
	Channels    []string      `json:"channels"`    // Channel IDs
	RequireAck  bool          `json:"require_ack"` // Require acknowledgement before next level
}

// MetricValue represents a metric value for rule evaluation
type MetricValue struct {
	Name      string                 `json:"name"`
	Value     float64                `json:"value"`
	Timestamp time.Time              `json:"timestamp"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EvaluationResult represents the result of evaluating a rule
type EvaluationResult struct {
	RuleID    string       `json:"rule_id"`
	Triggered bool         `json:"triggered"`
	Value     float64      `json:"value"`
	Threshold float64      `json:"threshold"`
	Message   string       `json:"message"`
	Timestamp time.Time    `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}
