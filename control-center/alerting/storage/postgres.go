package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/paw/control-center/alerting"
	"github.com/redis/go-redis/v9"
)

// PostgresStorage implements alert storage using PostgreSQL
type PostgresStorage struct {
	db    *sql.DB
	redis *redis.Client
	ctx   context.Context
}

// NewPostgresStorage creates a new PostgreSQL storage backend
func NewPostgresStorage(databaseURL, redisURL string) (*PostgresStorage, error) {
	// Connect to PostgreSQL
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Connect to Redis
	var redisClient *redis.Client
	if redisURL != "" {
		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
		}

		redisClient = redis.NewClient(opt)
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			return nil, fmt.Errorf("failed to connect to Redis: %w", err)
		}
	}

	storage := &PostgresStorage{
		db:    db,
		redis: redisClient,
		ctx:   context.Background(),
	}

	// Initialize schema
	if err := storage.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return storage, nil
}

// initSchema creates the necessary database tables
func (s *PostgresStorage) initSchema() error {
	schema := `
	-- Alerts table
	CREATE TABLE IF NOT EXISTS alerts (
		id VARCHAR(255) PRIMARY KEY,
		rule_id VARCHAR(255) NOT NULL,
		rule_name VARCHAR(255) NOT NULL,
		source VARCHAR(50) NOT NULL,
		severity VARCHAR(20) NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'active',
		message TEXT NOT NULL,
		description TEXT,
		labels JSONB,
		annotations JSONB,
		value DOUBLE PRECISION,
		threshold DOUBLE PRECISION,
		metadata JSONB,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
		acknowledged_at TIMESTAMP,
		acknowledged_by VARCHAR(255),
		resolved_at TIMESTAMP,
		resolved_by VARCHAR(255)
	);

	CREATE INDEX IF NOT EXISTS idx_alerts_rule_id ON alerts(rule_id);
	CREATE INDEX IF NOT EXISTS idx_alerts_source ON alerts(source);
	CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts(severity);
	CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);
	CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at DESC);

	-- Alert rules table
	CREATE TABLE IF NOT EXISTS alert_rules (
		id VARCHAR(255) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		source VARCHAR(50) NOT NULL,
		severity VARCHAR(20) NOT NULL,
		enabled BOOLEAN NOT NULL DEFAULT true,
		rule_type VARCHAR(50) NOT NULL,
		conditions JSONB NOT NULL,
		composite_op VARCHAR(10),
		evaluation_interval BIGINT NOT NULL,
		for_duration BIGINT NOT NULL,
		labels JSONB,
		annotations JSONB,
		channels JSONB,
		metadata JSONB,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON alert_rules(enabled);
	CREATE INDEX IF NOT EXISTS idx_alert_rules_source ON alert_rules(source);

	-- Notification channels table
	CREATE TABLE IF NOT EXISTS notification_channels (
		id VARCHAR(255) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		type VARCHAR(50) NOT NULL,
		enabled BOOLEAN NOT NULL DEFAULT true,
		config JSONB NOT NULL,
		filters JSONB,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_channels_type ON notification_channels(type);
	CREATE INDEX IF NOT EXISTS idx_channels_enabled ON notification_channels(enabled);

	-- Notifications history table
	CREATE TABLE IF NOT EXISTS notifications (
		id VARCHAR(255) PRIMARY KEY,
		alert_id VARCHAR(255) NOT NULL,
		channel_id VARCHAR(255) NOT NULL,
		channel_type VARCHAR(50) NOT NULL,
		sent_at TIMESTAMP NOT NULL DEFAULT NOW(),
		success BOOLEAN NOT NULL,
		error TEXT,
		retry_count INTEGER NOT NULL DEFAULT 0,
		FOREIGN KEY (alert_id) REFERENCES alerts(id) ON DELETE CASCADE,
		FOREIGN KEY (channel_id) REFERENCES notification_channels(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_notifications_alert_id ON notifications(alert_id);
	CREATE INDEX IF NOT EXISTS idx_notifications_channel_id ON notifications(channel_id);
	CREATE INDEX IF NOT EXISTS idx_notifications_sent_at ON notifications(sent_at DESC);

	-- Escalation policies table
	CREATE TABLE IF NOT EXISTS escalation_policies (
		id VARCHAR(255) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		enabled BOOLEAN NOT NULL DEFAULT true,
		levels JSONB NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	`

	_, err := s.db.Exec(schema)
	return err
}

// SaveAlert saves an alert to the database
func (s *PostgresStorage) SaveAlert(alert *alerting.Alert) error {
	labelsJSON, _ := json.Marshal(alert.Labels)
	annotationsJSON, _ := json.Marshal(alert.Annotations)
	metadataJSON, _ := json.Marshal(alert.Metadata)

	query := `
		INSERT INTO alerts (id, rule_id, rule_name, source, severity, status, message, description,
			labels, annotations, value, threshold, metadata, created_at, updated_at,
			acknowledged_at, acknowledged_by, resolved_at, resolved_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at,
			acknowledged_at = EXCLUDED.acknowledged_at,
			acknowledged_by = EXCLUDED.acknowledged_by,
			resolved_at = EXCLUDED.resolved_at,
			resolved_by = EXCLUDED.resolved_by
	`

	_, err := s.db.Exec(query,
		alert.ID, alert.RuleID, alert.RuleName, alert.Source, alert.Severity, alert.Status,
		alert.Message, alert.Description, labelsJSON, annotationsJSON, alert.Value, alert.Threshold,
		metadataJSON, alert.CreatedAt, alert.UpdatedAt, alert.AcknowledgedAt, alert.AcknowledgedBy,
		alert.ResolvedAt, alert.ResolvedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to save alert: %w", err)
	}

	// Cache active alerts in Redis
	if s.redis != nil && alert.Status == alerting.StatusActive {
		alertJSON, _ := json.Marshal(alert)
		s.redis.HSet(s.ctx, "alerts:active", alert.ID, alertJSON)
		s.redis.Expire(s.ctx, "alerts:active", 1*time.Hour)
	}

	return nil
}

// GetAlert retrieves an alert by ID
func (s *PostgresStorage) GetAlert(id string) (*alerting.Alert, error) {
	// Try cache first
	if s.redis != nil {
		cached, err := s.redis.HGet(s.ctx, "alerts:active", id).Result()
		if err == nil {
			var alert alerting.Alert
			if err := json.Unmarshal([]byte(cached), &alert); err == nil {
				return &alert, nil
			}
		}
	}

	// Query database
	query := `
		SELECT id, rule_id, rule_name, source, severity, status, message, description,
			labels, annotations, value, threshold, metadata, created_at, updated_at,
			acknowledged_at, acknowledged_by, resolved_at, resolved_by
		FROM alerts WHERE id = $1
	`

	var alert alerting.Alert
	var labelsJSON, annotationsJSON, metadataJSON []byte

	err := s.db.QueryRow(query, id).Scan(
		&alert.ID, &alert.RuleID, &alert.RuleName, &alert.Source, &alert.Severity, &alert.Status,
		&alert.Message, &alert.Description, &labelsJSON, &annotationsJSON, &alert.Value, &alert.Threshold,
		&metadataJSON, &alert.CreatedAt, &alert.UpdatedAt, &alert.AcknowledgedAt, &alert.AcknowledgedBy,
		&alert.ResolvedAt, &alert.ResolvedBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("alert not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	json.Unmarshal(labelsJSON, &alert.Labels)
	json.Unmarshal(annotationsJSON, &alert.Annotations)
	json.Unmarshal(metadataJSON, &alert.Metadata)

	return &alert, nil
}

// ListAlerts retrieves alerts with optional filtering
func (s *PostgresStorage) ListAlerts(filters AlertFilters) ([]*alerting.Alert, int, error) {
	// Build query
	query := `
		SELECT id, rule_id, rule_name, source, severity, status, message, description,
			labels, annotations, value, threshold, metadata, created_at, updated_at,
			acknowledged_at, acknowledged_by, resolved_at, resolved_by
		FROM alerts WHERE 1=1
	`
	countQuery := "SELECT COUNT(*) FROM alerts WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	// Apply filters
	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, filters.Status)
		argIndex++
	}

	if filters.Severity != "" {
		query += fmt.Sprintf(" AND severity = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND severity = $%d", argIndex)
		args = append(args, filters.Severity)
		argIndex++
	}

	if filters.Source != "" {
		query += fmt.Sprintf(" AND source = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND source = $%d", argIndex)
		args = append(args, filters.Source)
		argIndex++
	}

	if filters.RuleID != "" {
		query += fmt.Sprintf(" AND rule_id = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND rule_id = $%d", argIndex)
		args = append(args, filters.RuleID)
		argIndex++
	}

	if !filters.StartTime.IsZero() {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		countQuery += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, filters.StartTime)
		argIndex++
	}

	if !filters.EndTime.IsZero() {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		countQuery += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, filters.EndTime)
		argIndex++
	}

	// Get total count
	var totalCount int
	err := s.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count alerts: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC"
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}
	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
	}

	// Execute query
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query alerts: %w", err)
	}
	defer rows.Close()

	alerts := []*alerting.Alert{}
	for rows.Next() {
		var alert alerting.Alert
		var labelsJSON, annotationsJSON, metadataJSON []byte

		err := rows.Scan(
			&alert.ID, &alert.RuleID, &alert.RuleName, &alert.Source, &alert.Severity, &alert.Status,
			&alert.Message, &alert.Description, &labelsJSON, &annotationsJSON, &alert.Value, &alert.Threshold,
			&metadataJSON, &alert.CreatedAt, &alert.UpdatedAt, &alert.AcknowledgedAt, &alert.AcknowledgedBy,
			&alert.ResolvedAt, &alert.ResolvedBy,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan alert: %w", err)
		}

		json.Unmarshal(labelsJSON, &alert.Labels)
		json.Unmarshal(annotationsJSON, &alert.Annotations)
		json.Unmarshal(metadataJSON, &alert.Metadata)

		alerts = append(alerts, &alert)
	}

	return alerts, totalCount, nil
}

// SaveRule saves an alert rule
func (s *PostgresStorage) SaveRule(rule *alerting.AlertRule) error {
	conditionsJSON, _ := json.Marshal(rule.Conditions)
	labelsJSON, _ := json.Marshal(rule.Labels)
	annotationsJSON, _ := json.Marshal(rule.Annotations)
	channelsJSON, _ := json.Marshal(rule.Channels)
	metadataJSON, _ := json.Marshal(rule.Metadata)

	query := `
		INSERT INTO alert_rules (id, name, description, source, severity, enabled, rule_type,
			conditions, composite_op, evaluation_interval, for_duration, labels, annotations,
			channels, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			source = EXCLUDED.source,
			severity = EXCLUDED.severity,
			enabled = EXCLUDED.enabled,
			rule_type = EXCLUDED.rule_type,
			conditions = EXCLUDED.conditions,
			composite_op = EXCLUDED.composite_op,
			evaluation_interval = EXCLUDED.evaluation_interval,
			for_duration = EXCLUDED.for_duration,
			labels = EXCLUDED.labels,
			annotations = EXCLUDED.annotations,
			channels = EXCLUDED.channels,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`

	_, err := s.db.Exec(query,
		rule.ID, rule.Name, rule.Description, rule.Source, rule.Severity, rule.Enabled,
		rule.RuleType, conditionsJSON, rule.CompositeOp, rule.EvaluationInterval.Nanoseconds(),
		rule.ForDuration.Nanoseconds(), labelsJSON, annotationsJSON, channelsJSON,
		metadataJSON, rule.CreatedAt, rule.UpdatedAt,
	)

	return err
}

// GetRule retrieves a rule by ID
func (s *PostgresStorage) GetRule(id string) (*alerting.AlertRule, error) {
	query := `
		SELECT id, name, description, source, severity, enabled, rule_type, conditions,
			composite_op, evaluation_interval, for_duration, labels, annotations, channels,
			metadata, created_at, updated_at
		FROM alert_rules WHERE id = $1
	`

	var rule alerting.AlertRule
	var conditionsJSON, labelsJSON, annotationsJSON, channelsJSON, metadataJSON []byte
	var evalIntervalNs, forDurationNs int64

	err := s.db.QueryRow(query, id).Scan(
		&rule.ID, &rule.Name, &rule.Description, &rule.Source, &rule.Severity, &rule.Enabled,
		&rule.RuleType, &conditionsJSON, &rule.CompositeOp, &evalIntervalNs, &forDurationNs,
		&labelsJSON, &annotationsJSON, &channelsJSON, &metadataJSON, &rule.CreatedAt, &rule.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("rule not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	json.Unmarshal(conditionsJSON, &rule.Conditions)
	json.Unmarshal(labelsJSON, &rule.Labels)
	json.Unmarshal(annotationsJSON, &rule.Annotations)
	json.Unmarshal(channelsJSON, &rule.Channels)
	json.Unmarshal(metadataJSON, &rule.Metadata)

	rule.EvaluationInterval = time.Duration(evalIntervalNs)
	rule.ForDuration = time.Duration(forDurationNs)

	return &rule, nil
}

// ListRules retrieves all alert rules
func (s *PostgresStorage) ListRules(enabledOnly bool) ([]*alerting.AlertRule, error) {
	query := `
		SELECT id, name, description, source, severity, enabled, rule_type, conditions,
			composite_op, evaluation_interval, for_duration, labels, annotations, channels,
			metadata, created_at, updated_at
		FROM alert_rules
	`

	if enabledOnly {
		query += " WHERE enabled = true"
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()

	rules := []*alerting.AlertRule{}
	for rows.Next() {
		var rule alerting.AlertRule
		var conditionsJSON, labelsJSON, annotationsJSON, channelsJSON, metadataJSON []byte
		var evalIntervalNs, forDurationNs int64

		err := rows.Scan(
			&rule.ID, &rule.Name, &rule.Description, &rule.Source, &rule.Severity, &rule.Enabled,
			&rule.RuleType, &conditionsJSON, &rule.CompositeOp, &evalIntervalNs, &forDurationNs,
			&labelsJSON, &annotationsJSON, &channelsJSON, &metadataJSON, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}

		json.Unmarshal(conditionsJSON, &rule.Conditions)
		json.Unmarshal(labelsJSON, &rule.Labels)
		json.Unmarshal(annotationsJSON, &rule.Annotations)
		json.Unmarshal(channelsJSON, &rule.Channels)
		json.Unmarshal(metadataJSON, &rule.Metadata)

		rule.EvaluationInterval = time.Duration(evalIntervalNs)
		rule.ForDuration = time.Duration(forDurationNs)

		rules = append(rules, &rule)
	}

	return rules, nil
}

// DeleteRule deletes an alert rule
func (s *PostgresStorage) DeleteRule(id string) error {
	_, err := s.db.Exec("DELETE FROM alert_rules WHERE id = $1", id)
	return err
}

// SaveChannel saves a notification channel
func (s *PostgresStorage) SaveChannel(channel *alerting.Channel) error {
	configJSON, _ := json.Marshal(channel.Config)
	filtersJSON, _ := json.Marshal(channel.Filters)

	query := `
		INSERT INTO notification_channels (id, name, type, enabled, config, filters, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			enabled = EXCLUDED.enabled,
			config = EXCLUDED.config,
			filters = EXCLUDED.filters,
			updated_at = EXCLUDED.updated_at
	`

	_, err := s.db.Exec(query,
		channel.ID, channel.Name, channel.Type, channel.Enabled, configJSON, filtersJSON,
		channel.CreatedAt, channel.UpdatedAt,
	)

	return err
}

// GetChannel retrieves a channel by ID
func (s *PostgresStorage) GetChannel(id string) (*alerting.Channel, error) {
	query := `
		SELECT id, name, type, enabled, config, filters, created_at, updated_at
		FROM notification_channels WHERE id = $1
	`

	var channel alerting.Channel
	var configJSON, filtersJSON []byte

	err := s.db.QueryRow(query, id).Scan(
		&channel.ID, &channel.Name, &channel.Type, &channel.Enabled, &configJSON, &filtersJSON,
		&channel.CreatedAt, &channel.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("channel not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	json.Unmarshal(configJSON, &channel.Config)
	json.Unmarshal(filtersJSON, &channel.Filters)

	return &channel, nil
}

// ListChannels retrieves all notification channels
func (s *PostgresStorage) ListChannels() ([]*alerting.Channel, error) {
	query := `
		SELECT id, name, type, enabled, config, filters, created_at, updated_at
		FROM notification_channels
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query channels: %w", err)
	}
	defer rows.Close()

	channels := []*alerting.Channel{}
	for rows.Next() {
		var channel alerting.Channel
		var configJSON, filtersJSON []byte

		err := rows.Scan(
			&channel.ID, &channel.Name, &channel.Type, &channel.Enabled, &configJSON, &filtersJSON,
			&channel.CreatedAt, &channel.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}

		json.Unmarshal(configJSON, &channel.Config)
		json.Unmarshal(filtersJSON, &channel.Filters)

		channels = append(channels, &channel)
	}

	return channels, nil
}

// DeleteChannel deletes a notification channel
func (s *PostgresStorage) DeleteChannel(id string) error {
	_, err := s.db.Exec("DELETE FROM notification_channels WHERE id = $1", id)
	return err
}

// SaveNotification saves a notification record
func (s *PostgresStorage) SaveNotification(notification *alerting.Notification) error {
	query := `
		INSERT INTO notifications (id, alert_id, channel_id, channel_type, sent_at, success, error, retry_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := s.db.Exec(query,
		notification.ID, notification.AlertID, notification.ChannelID, notification.ChannelType,
		notification.SentAt, notification.Success, notification.Error, notification.RetryCount,
	)

	return err
}

// GetAlertStats retrieves alert statistics
func (s *PostgresStorage) GetAlertStats() (*alerting.AlertStats, error) {
	stats := &alerting.AlertStats{
		BySeverity: make(map[alerting.Severity]int),
		BySource:   make(map[alerting.AlertSource]int),
		ByStatus:   make(map[alerting.Status]int),
	}

	// Total counts
	s.db.QueryRow("SELECT COUNT(*) FROM alerts").Scan(&stats.TotalAlerts)
	s.db.QueryRow("SELECT COUNT(*) FROM alerts WHERE status = 'active'").Scan(&stats.ActiveAlerts)
	s.db.QueryRow("SELECT COUNT(*) FROM alerts WHERE status = 'acknowledged'").Scan(&stats.AcknowledgedAlerts)
	s.db.QueryRow("SELECT COUNT(*) FROM alerts WHERE status = 'resolved'").Scan(&stats.ResolvedAlerts)

	// By severity
	rows, _ := s.db.Query("SELECT severity, COUNT(*) FROM alerts GROUP BY severity")
	for rows.Next() {
		var severity alerting.Severity
		var count int
		rows.Scan(&severity, &count)
		stats.BySeverity[severity] = count
	}
	rows.Close()

	// By source
	rows, _ = s.db.Query("SELECT source, COUNT(*) FROM alerts GROUP BY source")
	for rows.Next() {
		var source alerting.AlertSource
		var count int
		rows.Scan(&source, &count)
		stats.BySource[source] = count
	}
	rows.Close()

	// By status
	rows, _ = s.db.Query("SELECT status, COUNT(*) FROM alerts GROUP BY status")
	for rows.Next() {
		var status alerting.Status
		var count int
		rows.Scan(&status, &count)
		stats.ByStatus[status] = count
	}
	rows.Close()

	// Mean time to acknowledge
	var avgAckSeconds sql.NullFloat64
	s.db.QueryRow(`
		SELECT AVG(EXTRACT(EPOCH FROM (acknowledged_at - created_at)))
		FROM alerts WHERE acknowledged_at IS NOT NULL
	`).Scan(&avgAckSeconds)
	if avgAckSeconds.Valid {
		stats.MeanTimeToAcknowledge = time.Duration(avgAckSeconds.Float64 * float64(time.Second))
	}

	// Mean time to resolve
	var avgResolveSeconds sql.NullFloat64
	s.db.QueryRow(`
		SELECT AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)))
		FROM alerts WHERE resolved_at IS NOT NULL
	`).Scan(&avgResolveSeconds)
	if avgResolveSeconds.Valid {
		stats.MeanTimeToResolve = time.Duration(avgResolveSeconds.Float64 * float64(time.Second))
	}

	return stats, nil
}

// Close closes database connections
func (s *PostgresStorage) Close() error {
	if s.redis != nil {
		s.redis.Close()
	}
	return s.db.Close()
}

// AlertFilters defines filters for listing alerts
type AlertFilters struct {
	Status    alerting.Status
	Severity  alerting.Severity
	Source    alerting.AlertSource
	RuleID    string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
	Offset    int
}
