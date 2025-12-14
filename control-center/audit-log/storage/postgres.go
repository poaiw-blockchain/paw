package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"paw/control-center/audit-log/types"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// PostgresStorage implements audit log storage using PostgreSQL
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgresStorage creates a new PostgreSQL storage instance
func NewPostgresStorage(connString string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	storage := &PostgresStorage{db: db}

	// Initialize schema
	if err := storage.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return storage, nil
}

// initSchema initializes the database schema
func (s *PostgresStorage) initSchema() error {
	// Read and execute schema from embedded file or inline
	// For now, we'll assume the schema is already created
	// In production, use migrations or embed the schema file
	return nil
}

// Insert inserts a new audit log entry
func (s *PostgresStorage) Insert(ctx context.Context, entry *types.AuditLogEntry) error {
	// Generate UUID if not set
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	// Set timestamp if not set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	// Marshal JSON fields
	changesJSON, err := json.Marshal(entry.Changes)
	if err != nil {
		return fmt.Errorf("failed to marshal changes: %w", err)
	}

	previousValueJSON, err := json.Marshal(entry.PreviousValue)
	if err != nil {
		return fmt.Errorf("failed to marshal previous_value: %w", err)
	}

	newValueJSON, err := json.Marshal(entry.NewValue)
	if err != nil {
		return fmt.Errorf("failed to marshal new_value: %w", err)
	}

	metadataJSON, err := json.Marshal(entry.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO audit_log (
			id, timestamp, event_type, user_id, user_email, user_role,
			action, resource, resource_id, changes, previous_value, new_value,
			ip_address, user_agent, session_id, result, error_message,
			severity, metadata, hash, previous_hash
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18, $19, $20, $21
		)
	`

	_, err = s.db.ExecContext(ctx, query,
		entry.ID,
		entry.Timestamp,
		entry.EventType,
		nullString(entry.UserID),
		entry.UserEmail,
		nullString(entry.UserRole),
		entry.Action,
		nullString(entry.Resource),
		nullString(entry.ResourceID),
		changesJSON,
		previousValueJSON,
		newValueJSON,
		nullString(entry.IPAddress),
		nullString(entry.UserAgent),
		nullString(entry.SessionID),
		entry.Result,
		nullString(entry.ErrorMessage),
		entry.Severity,
		metadataJSON,
		entry.Hash,
		nullString(entry.PreviousHash),
	)

	if err != nil {
		return fmt.Errorf("failed to insert audit log entry: %w", err)
	}

	return nil
}

// Query queries audit log entries based on filters
func (s *PostgresStorage) Query(ctx context.Context, filters types.QueryFilters) ([]types.AuditLogEntry, int64, error) {
	// Build WHERE clause
	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIndex := 1

	if len(filters.EventType) > 0 {
		placeholders := make([]string, len(filters.EventType))
		for i, et := range filters.EventType {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, et)
			argIndex++
		}
		whereClauses = append(whereClauses, fmt.Sprintf("event_type IN (%s)", strings.Join(placeholders, ",")))
	}

	if filters.UserID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, filters.UserID)
		argIndex++
	}

	if filters.UserEmail != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("user_email = $%d", argIndex))
		args = append(args, filters.UserEmail)
		argIndex++
	}

	if filters.Action != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("action ILIKE $%d", argIndex))
		args = append(args, "%"+filters.Action+"%")
		argIndex++
	}

	if filters.Resource != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("resource = $%d", argIndex))
		args = append(args, filters.Resource)
		argIndex++
	}

	if filters.ResourceID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("resource_id = $%d", argIndex))
		args = append(args, filters.ResourceID)
		argIndex++
	}

	if filters.Result != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("result = $%d", argIndex))
		args = append(args, filters.Result)
		argIndex++
	}

	if filters.Severity != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("severity = $%d", argIndex))
		args = append(args, filters.Severity)
		argIndex++
	}

	if !filters.StartTime.IsZero() {
		whereClauses = append(whereClauses, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, filters.StartTime)
		argIndex++
	}

	if !filters.EndTime.IsZero() {
		whereClauses = append(whereClauses, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, filters.EndTime)
		argIndex++
	}

	if filters.SearchText != "" {
		whereClauses = append(whereClauses, fmt.Sprintf(
			"to_tsvector('english', coalesce(action, '') || ' ' || coalesce(resource, '') || ' ' || coalesce(error_message, '')) @@ plainto_tsquery('english', $%d)",
			argIndex))
		args = append(args, filters.SearchText)
		argIndex++
	}

	whereClause := strings.Join(whereClauses, " AND ")

	// Get total count
	var totalCount int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_log WHERE %s", whereClause)
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count entries: %w", err)
	}

	// Build ORDER BY clause
	sortBy := "timestamp"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}
	sortOrder := "DESC"
	if filters.SortOrder != "" {
		sortOrder = strings.ToUpper(filters.SortOrder)
	}

	// Build LIMIT and OFFSET
	limit := 100
	if filters.Limit > 0 {
		limit = filters.Limit
	}
	offset := 0
	if filters.Offset > 0 {
		offset = filters.Offset
	}

	// Query entries
	query := fmt.Sprintf(`
		SELECT id, timestamp, event_type, user_id, user_email, user_role,
			   action, resource, resource_id, changes, previous_value, new_value,
			   ip_address, user_agent, session_id, result, error_message,
			   severity, metadata, hash, previous_hash
		FROM audit_log
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortBy, sortOrder, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query entries: %w", err)
	}
	defer rows.Close()

	entries := []types.AuditLogEntry{}
	for rows.Next() {
		entry, err := s.scanEntry(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("row iteration error: %w", err)
	}

	return entries, totalCount, nil
}

// GetByID retrieves a single audit log entry by ID
func (s *PostgresStorage) GetByID(ctx context.Context, id string) (*types.AuditLogEntry, error) {
	query := `
		SELECT id, timestamp, event_type, user_id, user_email, user_role,
			   action, resource, resource_id, changes, previous_value, new_value,
			   ip_address, user_agent, session_id, result, error_message,
			   severity, metadata, hash, previous_hash
		FROM audit_log
		WHERE id = $1
	`

	row := s.db.QueryRowContext(ctx, query, id)
	entry, err := s.scanEntry(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("entry not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get entry: %w", err)
	}

	return &entry, nil
}

// GetStats retrieves aggregated statistics
func (s *PostgresStorage) GetStats(ctx context.Context, startTime, endTime time.Time) (*types.AuditStats, error) {
	stats := &types.AuditStats{
		EventsByType:     make(map[types.EventType]int64),
		EventsByUser:     make(map[string]int64),
		EventsByResult:   make(map[types.Result]int64),
		EventsBySeverity: make(map[types.Severity]int64),
		TimeRange: types.TimeRange{
			Start: startTime,
			End:   endTime,
		},
	}

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	if !startTime.IsZero() {
		whereClause += " AND timestamp >= $1"
		args = append(args, startTime)
	}
	if !endTime.IsZero() {
		whereClause += fmt.Sprintf(" AND timestamp <= $%d", len(args)+1)
		args = append(args, endTime)
	}

	// Total events
	query := fmt.Sprintf("SELECT COUNT(*) FROM audit_log %s", whereClause)
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&stats.TotalEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Events by type
	query = fmt.Sprintf("SELECT event_type, COUNT(*) FROM audit_log %s GROUP BY event_type", whereClause)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events by type: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var eventType types.EventType
		var count int64
		if err := rows.Scan(&eventType, &count); err != nil {
			continue
		}
		stats.EventsByType[eventType] = count
	}

	// Events by result
	query = fmt.Sprintf("SELECT result, COUNT(*) FROM audit_log %s GROUP BY result", whereClause)
	rows, err = s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events by result: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var result types.Result
		var count int64
		if err := rows.Scan(&result, &count); err != nil {
			continue
		}
		stats.EventsByResult[result] = count
	}

	// Calculate success/failure rates
	if stats.TotalEvents > 0 {
		stats.SuccessRate = float64(stats.EventsByResult[types.ResultSuccess]) / float64(stats.TotalEvents) * 100
		stats.FailureRate = float64(stats.EventsByResult[types.ResultFailure]) / float64(stats.TotalEvents) * 100
	}

	// Top users
	query = fmt.Sprintf(`
		SELECT user_id, user_email, COUNT(*) as count, MAX(timestamp) as last_seen
		FROM audit_log %s
		GROUP BY user_id, user_email
		ORDER BY count DESC
		LIMIT 10
	`, whereClause)
	rows, err = s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query top users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var activity types.UserActivity
		var userID sql.NullString
		if err := rows.Scan(&userID, &activity.UserEmail, &activity.Count, &activity.LastSeen); err != nil {
			continue
		}
		if userID.Valid {
			activity.UserID = userID.String
		}
		stats.TopUsers = append(stats.TopUsers, activity)
	}

	return stats, nil
}

// GetTimeline retrieves timeline entries
func (s *PostgresStorage) GetTimeline(ctx context.Context, filters types.QueryFilters) ([]types.TimelineEntry, error) {
	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIndex := 1

	if !filters.StartTime.IsZero() {
		whereClauses = append(whereClauses, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, filters.StartTime)
		argIndex++
	}

	if !filters.EndTime.IsZero() {
		whereClauses = append(whereClauses, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, filters.EndTime)
		argIndex++
	}

	if filters.UserEmail != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("user_email = $%d", argIndex))
		args = append(args, filters.UserEmail)
		argIndex++
	}

	whereClause := strings.Join(whereClauses, " AND ")

	limit := 100
	if filters.Limit > 0 {
		limit = filters.Limit
	}

	query := fmt.Sprintf(`
		SELECT timestamp, event_type, action, user_email, resource, result,
			   CONCAT(action, ' - ', COALESCE(resource, ''), COALESCE(' (' || result || ')', '')) as description
		FROM audit_log
		WHERE %s
		ORDER BY timestamp DESC
		LIMIT $%d
	`, whereClause, argIndex)

	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query timeline: %w", err)
	}
	defer rows.Close()

	timeline := []types.TimelineEntry{}
	for rows.Next() {
		var entry types.TimelineEntry
		var resource sql.NullString
		if err := rows.Scan(&entry.Timestamp, &entry.EventType, &entry.Action, &entry.UserEmail, &resource, &entry.Result, &entry.Description); err != nil {
			continue
		}
		if resource.Valid {
			entry.Resource = resource.String
		}
		timeline = append(timeline, entry)
	}

	return timeline, nil
}

// Archive archives old entries
func (s *PostgresStorage) Archive(ctx context.Context, retentionDays int) (int64, error) {
	query := "SELECT archive_old_audit_logs($1)"
	var count int64
	err := s.db.QueryRowContext(ctx, query, retentionDays).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to archive logs: %w", err)
	}
	return count, nil
}

// Close closes the database connection
func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// scanEntry scans a database row into an AuditLogEntry
func (s *PostgresStorage) scanEntry(scanner interface {
	Scan(dest ...interface{}) error
}) (types.AuditLogEntry, error) {
	var entry types.AuditLogEntry
	var userID, userRole, resource, resourceID, ipAddress, userAgent, sessionID, errorMessage, previousHash sql.NullString
	var changesJSON, previousValueJSON, newValueJSON, metadataJSON []byte

	err := scanner.Scan(
		&entry.ID,
		&entry.Timestamp,
		&entry.EventType,
		&userID,
		&entry.UserEmail,
		&userRole,
		&entry.Action,
		&resource,
		&resourceID,
		&changesJSON,
		&previousValueJSON,
		&newValueJSON,
		&ipAddress,
		&userAgent,
		&sessionID,
		&entry.Result,
		&errorMessage,
		&entry.Severity,
		&metadataJSON,
		&entry.Hash,
		&previousHash,
	)

	if err != nil {
		return entry, err
	}

	// Handle nullable fields
	if userID.Valid {
		entry.UserID = userID.String
	}
	if userRole.Valid {
		entry.UserRole = userRole.String
	}
	if resource.Valid {
		entry.Resource = resource.String
	}
	if resourceID.Valid {
		entry.ResourceID = resourceID.String
	}
	if ipAddress.Valid {
		entry.IPAddress = ipAddress.String
	}
	if userAgent.Valid {
		entry.UserAgent = userAgent.String
	}
	if sessionID.Valid {
		entry.SessionID = sessionID.String
	}
	if errorMessage.Valid {
		entry.ErrorMessage = errorMessage.String
	}
	if previousHash.Valid {
		entry.PreviousHash = previousHash.String
	}

	// Unmarshal JSON fields
	if len(changesJSON) > 0 && string(changesJSON) != "null" {
		json.Unmarshal(changesJSON, &entry.Changes)
	}
	if len(previousValueJSON) > 0 && string(previousValueJSON) != "null" {
		json.Unmarshal(previousValueJSON, &entry.PreviousValue)
	}
	if len(newValueJSON) > 0 && string(newValueJSON) != "null" {
		json.Unmarshal(newValueJSON, &entry.NewValue)
	}
	if len(metadataJSON) > 0 && string(metadataJSON) != "null" {
		json.Unmarshal(metadataJSON, &entry.Metadata)
	}

	return entry, nil
}

// nullString converts a string to sql.NullString
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
