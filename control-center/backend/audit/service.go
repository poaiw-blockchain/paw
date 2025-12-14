package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// Entry represents an audit log entry
type Entry struct {
	ID          uint      `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	User        string    `json:"user"`
	Role        string    `json:"role"`
	Action      string    `json:"action"`
	Module      string    `json:"module"`
	Parameters  string    `json:"parameters"`
	Result      string    `json:"result"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	SessionID   string    `json:"session_id"`
}

// Service provides audit logging functionality
type Service struct {
	db    *sql.DB
	redis *redis.Client
	ctx   context.Context
}

// NewService creates a new audit logging service
func NewService(databaseURL, redisURL string) (*Service, error) {
	// Connect to PostgreSQL
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

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

	svc := &Service{
		db:    db,
		redis: redisClient,
		ctx:   context.Background(),
	}

	// Initialize database schema
	if err := svc.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return svc, nil
}

// initSchema creates the audit log table if it doesn't exist
func (s *Service) initSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS audit_log (
		id SERIAL PRIMARY KEY,
		timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
		user_email VARCHAR(255) NOT NULL,
		user_role VARCHAR(50),
		action VARCHAR(100) NOT NULL,
		module VARCHAR(50),
		parameters TEXT,
		result VARCHAR(255),
		ip_address VARCHAR(45),
		user_agent TEXT,
		session_id VARCHAR(255)
	);

	CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON audit_log(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_audit_log_user ON audit_log(user_email);
	CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action);
	CREATE INDEX IF NOT EXISTS idx_audit_log_module ON audit_log(module);
	`

	_, err := s.db.Exec(query)
	return err
}

// Log records an audit log entry
func (s *Service) Log(entry Entry) error {
	// Set timestamp if not provided
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Insert into database
	query := `
		INSERT INTO audit_log (timestamp, user_email, user_role, action, module, parameters, result, ip_address, user_agent, session_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	err := s.db.QueryRow(
		query,
		entry.Timestamp,
		entry.User,
		entry.Role,
		entry.Action,
		entry.Module,
		entry.Parameters,
		entry.Result,
		entry.IPAddress,
		entry.UserAgent,
		entry.SessionID,
	).Scan(&entry.ID)

	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}

	// Cache recent entry in Redis for real-time display
	if s.redis != nil {
		entryJSON, _ := json.Marshal(entry)
		s.redis.LPush(s.ctx, "audit:recent", entryJSON)
		s.redis.LTrim(s.ctx, "audit:recent", 0, 99) // Keep last 100 entries
		s.redis.Expire(s.ctx, "audit:recent", 1*time.Hour)
	}

	return nil
}

// Get retrieves audit log entries with filtering and pagination
func (s *Service) Get(filters Filters) ([]Entry, int, error) {
	// Build query
	query := "SELECT id, timestamp, user_email, user_role, action, module, parameters, result, ip_address, user_agent, session_id FROM audit_log WHERE 1=1"
	countQuery := "SELECT COUNT(*) FROM audit_log WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	// Apply filters
	if filters.User != "" {
		query += fmt.Sprintf(" AND user_email = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND user_email = $%d", argIndex)
		args = append(args, filters.User)
		argIndex++
	}

	if filters.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND action = $%d", argIndex)
		args = append(args, filters.Action)
		argIndex++
	}

	if filters.Module != "" {
		query += fmt.Sprintf(" AND module = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND module = $%d", argIndex)
		args = append(args, filters.Module)
		argIndex++
	}

	if !filters.StartTime.IsZero() {
		query += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		countQuery += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, filters.StartTime)
		argIndex++
	}

	if !filters.EndTime.IsZero() {
		query += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
		countQuery += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
		args = append(args, filters.EndTime)
		argIndex++
	}

	// Get total count
	var totalCount int
	err := s.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Add pagination
	query += " ORDER BY timestamp DESC"
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
		return nil, 0, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	// Parse results
	entries := []Entry{}
	for rows.Next() {
		var entry Entry
		err := rows.Scan(
			&entry.ID,
			&entry.Timestamp,
			&entry.User,
			&entry.Role,
			&entry.Action,
			&entry.Module,
			&entry.Parameters,
			&entry.Result,
			&entry.IPAddress,
			&entry.UserAgent,
			&entry.SessionID,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log row: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, totalCount, nil
}

// GetRecent retrieves recent audit log entries from Redis cache
func (s *Service) GetRecent(limit int) ([]Entry, error) {
	if s.redis == nil {
		// Fall back to database query
		return s.Get(Filters{Limit: limit})
	}

	// Get from Redis cache
	result, err := s.redis.LRange(s.ctx, "audit:recent", 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get recent logs from cache: %w", err)
	}

	entries := []Entry{}
	for _, entryJSON := range result {
		var entry Entry
		if err := json.Unmarshal([]byte(entryJSON), &entry); err != nil {
			continue // Skip invalid entries
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// Export exports audit log entries in specified format
func (s *Service) Export(filters Filters, format string) ([]byte, error) {
	entries, _, err := s.Get(filters)
	if err != nil {
		return nil, err
	}

	switch format {
	case "json":
		return json.MarshalIndent(entries, "", "  ")
	case "csv":
		return s.exportCSV(entries)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportCSV exports entries as CSV
func (s *Service) exportCSV(entries []Entry) ([]byte, error) {
	csv := "ID,Timestamp,User,Role,Action,Module,Parameters,Result,IP Address,User Agent,Session ID\n"

	for _, entry := range entries {
		csv += fmt.Sprintf("%d,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
			entry.ID,
			entry.Timestamp.Format(time.RFC3339),
			entry.User,
			entry.Role,
			entry.Action,
			entry.Module,
			entry.Parameters,
			entry.Result,
			entry.IPAddress,
			entry.UserAgent,
			entry.SessionID,
		)
	}

	return []byte(csv), nil
}

// Close closes database and Redis connections
func (s *Service) Close() error {
	if s.redis != nil {
		s.redis.Close()
	}
	return s.db.Close()
}

// Filters represents audit log query filters
type Filters struct {
	User      string
	Action    string
	Module    string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
	Offset    int
}
