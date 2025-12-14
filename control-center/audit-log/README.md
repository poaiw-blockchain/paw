# Audit Log System

Comprehensive audit logging system for tracking all administrative actions with cryptographic integrity verification.

## Features

- **Comprehensive Event Tracking**: Logs all administrative actions including authentication, parameter changes, circuit breaker operations, emergency actions, alerts, upgrades, and access control
- **Cryptographic Integrity**: SHA-256 hash chain ensures tamper-proof audit trail
- **Rich Query API**: Filter by event type, user, time range, resource, result, severity, and full-text search
- **Multiple Export Formats**: Export audit logs as CSV or JSON
- **Real-time Statistics**: Aggregated statistics and analytics
- **Timeline View**: Chronological view of events
- **Tampering Detection**: Automatic detection of hash chain breaks and anomalies
- **RBAC Integration**: Role-based access control for audit log access
- **Retention Policy**: Automatic archival of old logs

## Architecture

```
audit-log/
├── types/           # Type definitions and event constants
├── storage/         # PostgreSQL storage implementation
├── integrity/       # Hash chain and integrity verification
├── api/             # REST API handlers
├── middleware/      # Automatic audit logging middleware
├── export/          # Export functionality (CSV, JSON)
├── tests/           # Comprehensive test suite
└── server.go        # Server initialization
```

## Quick Start

### 1. Setup Database

```bash
# Create database
createdb audit_log

# Run migrations
psql audit_log < storage/schema.sql
```

### 2. Start Server

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    auditlog "paw/control-center/audit-log"
)

func main() {
    cfg := auditlog.Config{
        DatabaseURL: "postgres://user:pass@localhost/audit_log?sslmode=disable",
        HTTPPort:    8080,
        EnableCORS:  true,
    }

    server, err := auditlog.NewServer(cfg)
    if err != nil {
        log.Fatal(err)
    }

    // Start server
    go func() {
        if err := server.Start(); err != nil {
            log.Fatal(err)
        }
    }()

    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan

    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    server.Stop(ctx)
}
```

### 3. Integrate Middleware

```go
// In your admin API
import (
    "paw/control-center/audit-log/middleware"
)

func main() {
    // ... server setup ...

    auditLogger := server.GetMiddleware()

    // Wrap your router
    router := mux.NewRouter()
    router.Use(auditLogger.Middleware)

    // Or manually log actions
    err := auditLogger.LogAction(ctx, middleware.AuditAction{
        EventType:  types.EventTypeParamUpdate,
        UserEmail:  "admin@example.com",
        Action:     "Update oracle parameters",
        Resource:   "oracle",
        Result:     types.ResultSuccess,
        Severity:   types.SeverityInfo,
    })
}
```

## API Endpoints

### Query Logs
```bash
GET /api/v1/audit/logs?event_type=auth.login&limit=100&offset=0
```

Query parameters:
- `event_type`: Filter by event type (multiple allowed)
- `user_id`, `user_email`: Filter by user
- `action`: Filter by action (partial match)
- `resource`, `resource_id`: Filter by resource
- `result`: Filter by result (success/failure/partial)
- `severity`: Filter by severity (info/warning/critical)
- `start_time`, `end_time`: Time range (RFC3339 or Unix timestamp)
- `search`: Full-text search
- `limit`, `offset`: Pagination
- `sort_by`, `sort_order`: Sorting

### Get Single Log
```bash
GET /api/v1/audit/logs/{id}
```

### Search Logs
```bash
POST /api/v1/audit/logs/search
Content-Type: application/json

{
  "event_type": ["auth.login", "auth.logout"],
  "start_time": "2025-01-01T00:00:00Z",
  "end_time": "2025-01-31T23:59:59Z",
  "limit": 100
}
```

### Export Logs
```bash
POST /api/v1/audit/logs/export
Content-Type: application/json

{
  "format": "csv",
  "filters": {
    "start_time": "2025-01-01T00:00:00Z",
    "limit": 1000
  },
  "fields": ["timestamp", "event_type", "user_email", "action", "result"]
}
```

### Get Statistics
```bash
GET /api/v1/audit/stats?start_time=2025-01-01T00:00:00Z&end_time=2025-01-31T23:59:59Z
```

Response:
```json
{
  "total_events": 1234,
  "events_by_type": {
    "auth.login": 456,
    "param.update": 123,
    "circuit.pause": 5
  },
  "events_by_result": {
    "success": 1200,
    "failure": 34
  },
  "success_rate": 97.2,
  "failure_rate": 2.8,
  "top_users": [
    {
      "user_email": "admin@example.com",
      "count": 234,
      "last_seen": "2025-01-31T12:00:00Z"
    }
  ]
}
```

### Get Timeline
```bash
GET /api/v1/audit/timeline?limit=50&start_time=2025-01-01T00:00:00Z
```

### Get User Activity
```bash
GET /api/v1/audit/user/{user_id}?limit=100
```

### Verify Integrity
```bash
POST /api/v1/audit/integrity/verify
Content-Type: application/json

{
  "start_time": "2025-01-01T00:00:00Z",
  "end_time": "2025-01-31T23:59:59Z",
  "limit": 1000
}
```

Response:
```json
{
  "verified": true,
  "start_id": "abc123",
  "end_id": "xyz789",
  "entries_checked": 1000,
  "errors": [],
  "checked_at": "2025-01-31T12:00:00Z"
}
```

### Detect Tampering
```bash
POST /api/v1/audit/integrity/detect-tampering
Content-Type: application/json

{
  "limit": 1000
}
```

Response:
```json
{
  "alerts": [
    {
      "EntryID": "entry123",
      "Timestamp": "2025-01-15T10:30:00Z",
      "AlertType": "hash_mismatch",
      "Description": "Entry hash does not match calculated hash",
      "Severity": "critical"
    }
  ],
  "alerts_count": 1,
  "entries_checked": 1000
}
```

## Event Types

### Authentication
- `auth.login` - User login
- `auth.logout` - User logout
- `auth.login_failed` - Failed login attempt
- `auth.password_changed` - Password changed
- `auth.token_refreshed` - Token refreshed
- `auth.session_expired` - Session expired

### Parameter Changes
- `param.update` - Single parameter updated
- `param.bulk_update` - Multiple parameters updated
- `param.reset` - Parameters reset to default

### Circuit Breaker
- `circuit.pause` - Circuit breaker paused
- `circuit.resume` - Circuit breaker resumed
- `circuit.triggered` - Circuit breaker triggered

### Emergency Operations
- `emergency.pause` - Emergency pause activated
- `emergency.resume` - Emergency pause lifted
- `emergency.action` - Other emergency action

### Alert Management
- `alert.rule_created` - Alert rule created
- `alert.rule_updated` - Alert rule updated
- `alert.rule_deleted` - Alert rule deleted
- `alert.acknowledged` - Alert acknowledged
- `alert.resolved` - Alert resolved

### Network Upgrades
- `upgrade.scheduled` - Upgrade scheduled
- `upgrade.executed` - Upgrade executed
- `upgrade.cancelled` - Upgrade cancelled
- `upgrade.failed` - Upgrade failed

### Access Control
- `access.role_assigned` - Role assigned to user
- `access.role_revoked` - Role revoked from user
- `access.permission_granted` - Permission granted
- `access.permission_revoked` - Permission revoked

## Security Features

### Hash Chain Integrity

Each audit log entry contains:
- `hash`: SHA-256 hash of the entry content
- `previous_hash`: Hash of the previous entry

This creates an immutable chain where any modification breaks the chain.

### Automatic Integrity Verification

```go
import "paw/control-center/audit-log/integrity"

calc := integrity.NewHashCalculator()

// Verify a single entry
valid, err := calc.VerifyHash(entry)

// Verify a chain of entries
report, err := calc.VerifyChain(entries)

// Detect tampering
alerts, err := calc.DetectTampering(entries)
```

### Immutable Storage

Database schema enforces:
- Append-only writes (no UPDATE or DELETE)
- Automatic hash chain maintenance via triggers
- Indexed queries for performance

## Performance

- **Write Performance**: ~1000 entries/sec (with hash calculation)
- **Query Performance**: Sub-second for most queries with proper indexes
- **Storage**: ~500 bytes per entry on average
- **Retention**: Automatic archival after configurable period (default: 1 year)

## Database Maintenance

### Archive Old Logs
```sql
SELECT archive_old_audit_logs(365); -- Archive logs older than 365 days
```

### Refresh Statistics
```sql
SELECT refresh_audit_stats();
```

### Verify Integrity
```bash
# Via API
curl -X POST http://localhost:8080/api/v1/audit/integrity/verify \
  -H "Content-Type: application/json" \
  -d '{"limit": 10000}'
```

## Testing

```bash
# Run unit tests
go test ./... -v

# Run integration tests (requires PostgreSQL)
go test ./tests -v

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Best Practices

1. **Log Everything**: Use middleware to automatically log all admin API requests
2. **Rich Context**: Include user info, IP address, session ID in all log entries
3. **Detailed Changes**: Use the `changes` field to record before/after values
4. **Regular Verification**: Schedule periodic integrity verification
5. **Monitor Failures**: Alert on failed actions and integrity violations
6. **Secure Access**: Restrict audit log access to authorized administrators only
7. **Retention Policy**: Balance storage costs with compliance requirements
8. **Export Regularly**: Export logs to external SIEM systems for analysis

## Compliance

This audit logging system helps meet compliance requirements for:

- **SOC 2**: Comprehensive audit trails for security controls
- **ISO 27001**: Information security event logging
- **PCI DSS**: Requirement 10 - Track and monitor all access to network resources
- **GDPR**: Article 30 - Records of processing activities
- **HIPAA**: 164.308(a)(1)(ii)(D) - Information system activity review

## License

See project root LICENSE file.
