# Audit Log Quick Reference

## Server Initialization

```go
cfg := auditlog.Config{
    DatabaseURL: "postgres://user:pass@host/db",
    HTTPPort:    8081,
    EnableCORS:  true,
}
server, _ := auditlog.NewServer(cfg)
go server.Start()
```

## Middleware Integration

```go
auditLogger := server.GetMiddleware()
router.Use(auditLogger.Middleware)
```

## Manual Logging

```go
auditLogger.LogAction(ctx, middleware.AuditAction{
    EventType:     types.EventTypeParamUpdate,
    UserEmail:     "admin@example.com",
    Action:        "Update parameters",
    Resource:      "oracle",
    Result:        types.ResultSuccess,
    Severity:      types.SeverityInfo,
})
```

## Common Event Types

| Event | Constant |
|-------|----------|
| Login | `types.EventTypeLogin` |
| Logout | `types.EventTypeLogout` |
| Update Params | `types.EventTypeParamUpdate` |
| Pause Circuit | `types.EventTypeCircuitPause` |
| Emergency Pause | `types.EventTypeEmergencyPause` |
| Create Alert | `types.EventTypeAlertRuleCreated` |
| Schedule Upgrade | `types.EventTypeUpgradeScheduled` |
| Assign Role | `types.EventTypeRoleAssigned` |

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/audit/logs` | Query logs with filters |
| GET | `/api/v1/audit/logs/{id}` | Get single log entry |
| POST | `/api/v1/audit/logs/search` | Advanced search |
| POST | `/api/v1/audit/logs/export` | Export to CSV/JSON |
| GET | `/api/v1/audit/stats` | Get statistics |
| GET | `/api/v1/audit/timeline` | Get timeline view |
| GET | `/api/v1/audit/user/{id}` | Get user activity |
| POST | `/api/v1/audit/integrity/verify` | Verify integrity |
| POST | `/api/v1/audit/integrity/detect-tampering` | Detect tampering |

## Query Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `event_type` | string[] | Filter by event type |
| `user_email` | string | Filter by user email |
| `action` | string | Filter by action (partial) |
| `resource` | string | Filter by resource |
| `result` | string | Filter by result |
| `severity` | string | Filter by severity |
| `start_time` | RFC3339 | Start of time range |
| `end_time` | RFC3339 | End of time range |
| `search` | string | Full-text search |
| `limit` | int | Results per page |
| `offset` | int | Pagination offset |
| `sort_by` | string | Sort field |
| `sort_order` | string | ASC/DESC |

## Common Queries

### Recent Failed Actions
```bash
curl "http://localhost:8081/api/v1/audit/logs?result=failure&limit=50"
```

### Critical Events
```bash
curl "http://localhost:8081/api/v1/audit/logs?severity=critical&limit=100"
```

### User Activity
```bash
curl "http://localhost:8081/api/v1/audit/user/user123?limit=100"
```

### Export Last Week
```bash
curl -X POST http://localhost:8081/api/v1/audit/logs/export \
  -d '{"format":"csv","filters":{"start_time":"2025-01-01T00:00:00Z","limit":1000}}'
```

### Verify Integrity
```bash
curl -X POST http://localhost:8081/api/v1/audit/integrity/verify \
  -d '{"limit":1000}'
```

## Database Maintenance

### Archive Old Logs
```sql
SELECT archive_old_audit_logs(365); -- Archive logs older than 1 year
```

### Refresh Statistics
```sql
SELECT refresh_audit_stats();
```

### Check Table Size
```sql
SELECT pg_size_pretty(pg_total_relation_size('audit_log'));
```

## Integrity Verification

### Verify Hash Chain
```go
calc := integrity.NewHashCalculator()
report, err := calc.VerifyChain(entries)
if !report.Verified {
    log.Printf("Tampering detected: %v", report.Errors)
}
```

### Detect Tampering
```go
alerts, err := calc.DetectTampering(entries)
for _, alert := range alerts {
    log.Printf("Alert: %s - %s", alert.AlertType, alert.Description)
}
```

## Result Types

- `types.ResultSuccess` - Action succeeded
- `types.ResultFailure` - Action failed
- `types.ResultPartial` - Action partially succeeded

## Severity Levels

- `types.SeverityInfo` - Informational
- `types.SeverityWarning` - Warning
- `types.SeverityCritical` - Critical

## Export Formats

### CSV
```json
{
  "format": "csv",
  "filters": { "limit": 1000 },
  "fields": ["timestamp", "event_type", "user_email", "action", "result"]
}
```

### JSON
```json
{
  "format": "json",
  "filters": { "limit": 1000 },
  "fields": ["timestamp", "event_type", "user_email", "action", "result", "metadata"]
}
```

## Context Values Required

```go
ctx = context.WithValue(ctx, "user_id", "user123")
ctx = context.WithValue(ctx, "user_email", "admin@example.com")
ctx = context.WithValue(ctx, "user_role", "admin")
ctx = context.WithValue(ctx, "session_id", "session123")
```

## Configuration

```go
type Config struct {
    DatabaseURL string  // PostgreSQL connection string
    HTTPPort    int     // HTTP port for API
    EnableCORS  bool    // Enable CORS
}
```

## Database Connection String

```
postgres://user:password@host:port/database?sslmode=disable
```

Production:
```
postgres://user:password@host:port/database?sslmode=require
```

## Testing

```bash
# Unit tests
go test ./control-center/audit-log/... -v

# Integration tests
export AUDIT_DB_URL="postgres://test:test@localhost/audit_test"
go test ./control-center/audit-log/tests -v

# Coverage
go test ./control-center/audit-log/... -coverprofile=coverage.out
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Missing entries | Check middleware applied, user context set |
| Slow queries | Add indexes, archive old logs |
| Integrity failures | Investigate immediately, check for tampering |
| Export timeout | Reduce time range, use pagination |

## Performance Tips

1. Use indexes for common query patterns
2. Archive logs older than retention period
3. Refresh materialized views regularly
4. Use connection pooling
5. Enable query result caching
6. Monitor database size

## Security Best Practices

1. Enable TLS for all connections
2. Restrict audit log access to admins
3. Schedule regular integrity checks
4. Monitor for tampering attempts
5. Export to SIEM system
6. Implement access logging for audit logs
7. Use strong database credentials
8. Enable database audit logging
