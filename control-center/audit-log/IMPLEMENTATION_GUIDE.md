# Audit Log Implementation Guide

Complete guide for implementing the audit logging system in the PAW Control Center.

## Table of Contents

1. [Overview](#overview)
2. [Database Setup](#database-setup)
3. [Integration Steps](#integration-steps)
4. [Event Types Reference](#event-types-reference)
5. [API Reference](#api-reference)
6. [Security Considerations](#security-considerations)
7. [Performance Tuning](#performance-tuning)
8. [Monitoring](#monitoring)
9. [Troubleshooting](#troubleshooting)

## Overview

The audit logging system provides:
- **Immutable audit trail** with cryptographic integrity
- **Automatic logging** via middleware
- **Comprehensive querying** and export capabilities
- **Real-time statistics** and analytics
- **Tampering detection** and alerts

## Database Setup

### Step 1: Create Database

```bash
# PostgreSQL
createdb paw_audit_log

# Or with specific user
createdb -U postgres paw_audit_log
```

### Step 2: Apply Schema

```bash
psql paw_audit_log < control-center/audit-log/storage/schema.sql
```

### Step 3: Verify Setup

```sql
-- Check tables
\dt

-- Check indexes
\di

-- Check functions
\df
```

Expected tables:
- `audit_log` - Main audit log table
- `audit_log_archive` - Archive for old entries
- `audit_integrity_checks` - Integrity verification records
- `audit_log_stats` - Materialized view for statistics

## Integration Steps

### Step 1: Initialize Audit Server

```go
import (
    auditlog "paw/control-center/audit-log"
)

func main() {
    cfg := auditlog.Config{
        DatabaseURL: os.Getenv("AUDIT_DB_URL"),
        HTTPPort:    8081,
        EnableCORS:  true,
    }

    auditServer, err := auditlog.NewServer(cfg)
    if err != nil {
        log.Fatal("Failed to initialize audit server:", err)
    }

    // Start in background
    go func() {
        if err := auditServer.Start(); err != nil {
            log.Fatal("Audit server error:", err)
        }
    }()
}
```

### Step 2: Add Middleware to Admin API

```go
import (
    "paw/control-center/audit-log/middleware"
)

func setupAdminAPI(auditServer *auditlog.Server) http.Handler {
    router := mux.NewRouter()

    // Get audit middleware
    auditMiddleware := auditServer.GetMiddleware()

    // Apply to admin routes
    adminAPI := router.PathPrefix("/admin").Subrouter()
    adminAPI.Use(auditMiddleware.Middleware)
    adminAPI.Use(authMiddleware) // Your authentication middleware

    // Define admin endpoints
    adminAPI.HandleFunc("/params", updateParamsHandler).Methods("PUT")
    adminAPI.HandleFunc("/circuit/pause", pauseCircuitHandler).Methods("POST")
    // ... more endpoints

    return router
}
```

### Step 3: Ensure User Context

Your authentication middleware must set these context values:

```go
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract from JWT or session
        user := getUserFromToken(r)

        // Set in context for audit logging
        ctx := r.Context()
        ctx = context.WithValue(ctx, "user_id", user.ID)
        ctx = context.WithValue(ctx, "user_email", user.Email)
        ctx = context.WithValue(ctx, "user_role", user.Role)
        ctx = context.WithValue(ctx, "session_id", user.SessionID)

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Step 4: Manual Logging (When Needed)

For actions that need detailed logging beyond automatic middleware:

```go
func updateOracleParams(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    auditLogger := getAuditLogger() // From server instance

    // Get current state
    oldParams := getCurrentOracleParams()

    // Parse new params
    var newParams OracleParams
    json.NewDecoder(r.Body).Decode(&newParams)

    // Apply update
    err := applyOracleParams(newParams)

    // Log with detailed change tracking
    auditLogger.LogAction(ctx, middleware.AuditAction{
        EventType:     types.EventTypeParamUpdate,
        UserID:        ctx.Value("user_id").(string),
        UserEmail:     ctx.Value("user_email").(string),
        UserRole:      ctx.Value("user_role").(string),
        Action:        "Update Oracle parameters",
        Resource:      "oracle",
        ResourceID:    "params",
        PreviousValue: oldParams,
        NewValue:      newParams,
        Changes: map[string]interface{}{
            "min_count_changed":  oldParams.MinCount != newParams.MinCount,
            "max_count_changed":  oldParams.MaxCount != newParams.MaxCount,
            "timeout_changed":    oldParams.Timeout != newParams.Timeout,
        },
        IPAddress:    getClientIP(r),
        UserAgent:    r.UserAgent(),
        SessionID:    ctx.Value("session_id").(string),
        Result:       getResult(err),
        ErrorMessage: getErrorMessage(err),
        Severity:     getSeverity(err),
        Metadata: map[string]interface{}{
            "endpoint":    r.URL.Path,
            "method":      r.Method,
            "request_id":  ctx.Value("request_id"),
        },
    })

    // Send response
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
```

## Event Types Reference

### When to Use Each Event Type

#### Authentication Events
```go
// User login
types.EventTypeLogin
// Use: When user successfully authenticates

// Login failed
types.EventTypeLoginFailed
// Use: When authentication attempt fails

// Logout
types.EventTypeLogout
// Use: When user explicitly logs out

// Password changed
types.EventTypePasswordChanged
// Use: When user changes their password

// Token refreshed
types.EventTypeTokenRefreshed
// Use: When access token is refreshed

// Session expired
types.EventTypeSessionExpired
// Use: When session times out
```

#### Parameter Changes
```go
// Single parameter update
types.EventTypeParamUpdate
// Use: When updating module parameters

// Bulk update
types.EventTypeParamBulkUpdate
// Use: When updating multiple modules at once

// Reset to defaults
types.EventTypeParamReset
// Use: When resetting parameters to default values
```

#### Circuit Breaker
```go
// Manual pause
types.EventTypeCircuitPause
// Use: When admin manually pauses circuit breaker

// Manual resume
types.EventTypeCircuitResume
// Use: When admin resumes circuit breaker

// Automatic trigger
types.EventTypeCircuitTriggered
// Use: When circuit breaker trips automatically
```

#### Emergency Operations
```go
// Emergency pause
types.EventTypeEmergencyPause
// Use: When emergency stop is activated

// Emergency resume
types.EventTypeEmergencyResume
// Use: When emergency stop is lifted

// Other emergency action
types.EventTypeEmergencyAction
// Use: For other critical emergency operations
```

## API Reference

### Query Patterns

#### Get Recent Failed Actions
```bash
curl "http://localhost:8081/api/v1/audit/logs?result=failure&limit=50"
```

#### Get Specific User's Actions
```bash
curl "http://localhost:8081/api/v1/audit/logs?user_email=admin@example.com&limit=100"
```

#### Get Critical Events
```bash
curl "http://localhost:8081/api/v1/audit/logs?severity=critical&limit=100"
```

#### Search by Text
```bash
curl "http://localhost:8081/api/v1/audit/logs?search=oracle%20params"
```

#### Get Timeline for Last 24 Hours
```bash
START=$(date -u -d '24 hours ago' +%Y-%m-%dT%H:%M:%SZ)
curl "http://localhost:8081/api/v1/audit/timeline?start_time=$START"
```

#### Export Last Week's Logs
```bash
curl -X POST http://localhost:8081/api/v1/audit/logs/export \
  -H "Content-Type: application/json" \
  -d '{
    "format": "csv",
    "filters": {
      "start_time": "2025-01-01T00:00:00Z",
      "end_time": "2025-01-07T23:59:59Z"
    }
  }' > audit_logs.csv
```

## Security Considerations

### Access Control

Only authorized administrators should access audit logs:

```go
func auditAccessMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := getUserFromContext(r.Context())

        // Check if user has audit viewer role
        if !user.HasRole("audit_viewer") && !user.HasRole("admin") {
            http.Error(w, "Forbidden", http.StatusForbidden)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

### Integrity Monitoring

Schedule regular integrity checks:

```go
func scheduleIntegrityChecks(auditServer *auditlog.Server) {
    ticker := time.NewTicker(1 * time.Hour)
    go func() {
        for range ticker.C {
            report, err := verifyIntegrity(auditServer)
            if err != nil {
                log.Printf("Integrity check failed: %v", err)
                alertAdmins("Audit log integrity check failed")
                continue
            }

            if !report.Verified {
                log.Printf("CRITICAL: Audit log tampering detected!")
                log.Printf("Errors: %v", report.Errors)
                alertAdmins("CRITICAL: Audit log tampering detected")
            }
        }
    }()
}
```

### Secure Communication

Always use TLS for audit log API:

```go
// Use HTTPS in production
httpServer := &http.Server{
    Addr:      ":8081",
    Handler:   router,
    TLSConfig: &tls.Config{
        MinVersion: tls.VersionTLS13,
    },
}
httpServer.ListenAndServeTLS("cert.pem", "key.pem")
```

## Performance Tuning

### Database Optimization

```sql
-- Create additional indexes for common queries
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at DESC);
CREATE INDEX idx_audit_log_user_action ON audit_log(user_email, action);

-- Enable parallel query execution
SET max_parallel_workers_per_gather = 4;

-- Tune autovacuum for high-write table
ALTER TABLE audit_log SET (
    autovacuum_vacuum_scale_factor = 0.01,
    autovacuum_analyze_scale_factor = 0.005
);
```

### Connection Pooling

```go
// Optimize PostgreSQL connection pool
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(5 * time.Minute)
db.SetConnMaxIdleTime(10 * time.Minute)
```

### Archival Strategy

```bash
# Daily cron job to archive old logs
0 2 * * * psql paw_audit_log -c "SELECT archive_old_audit_logs(365);"

# Weekly cron job to refresh statistics
0 3 * * 0 psql paw_audit_log -c "SELECT refresh_audit_stats();"
```

## Monitoring

### Key Metrics to Track

1. **Write Throughput**: Audit entries per second
2. **Query Latency**: Response time for common queries
3. **Storage Growth**: Database size over time
4. **Integrity Status**: Results of integrity checks
5. **Failed Actions**: Count of failed administrative actions

### Prometheus Metrics Example

```go
var (
    auditEntriesTotal = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "audit_entries_total",
            Help: "Total number of audit log entries",
        },
    )

    auditFailuresTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "audit_failures_total",
            Help: "Total number of failed actions",
        },
        []string{"event_type"},
    )

    auditQueryDuration = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "audit_query_duration_seconds",
            Help:    "Audit log query duration",
            Buckets: prometheus.DefBuckets,
        },
    )
)
```

## Troubleshooting

### Issue: High Database Load

**Symptoms**: Slow audit log queries, high CPU usage

**Solutions**:
1. Check query patterns and add appropriate indexes
2. Increase database connection pool size
3. Archive old logs more frequently
4. Consider partitioning the audit_log table by month

### Issue: Integrity Check Failures

**Symptoms**: Integrity verification reports errors

**Solutions**:
1. Immediately investigate - this indicates tampering or data corruption
2. Check database logs for any unauthorized modifications
3. Review recent database maintenance activities
4. Restore from backup if tampering confirmed

### Issue: Missing Audit Entries

**Symptoms**: Expected audit entries not appearing

**Solutions**:
1. Verify middleware is applied to all admin routes
2. Check that user context is being set correctly
3. Review application logs for audit logging errors
4. Ensure database has sufficient disk space

### Issue: Slow Exports

**Symptoms**: Export requests timing out

**Solutions**:
1. Reduce export time range or use pagination
2. Export during off-peak hours
3. Consider streaming exports for large datasets
4. Increase HTTP timeout for export endpoint

## Best Practices Checklist

- [ ] Database schema applied and verified
- [ ] Audit middleware applied to all admin routes
- [ ] User context properly set in authentication middleware
- [ ] Manual logging for critical operations
- [ ] Regular integrity checks scheduled
- [ ] Archival policy configured
- [ ] Access control enforced for audit logs
- [ ] TLS enabled for audit log API
- [ ] Monitoring and alerting configured
- [ ] Export functionality tested
- [ ] Backup and recovery procedures documented

## Next Steps

1. Integrate with SIEM system for centralized security monitoring
2. Add real-time alerting for critical events
3. Implement automated response to certain event patterns
4. Create dashboards for audit log visualization
5. Document compliance mapping to regulatory requirements
