# Admin API - Quick Reference

## Start Server

```bash
cd control-center/admin-api
go run server.go
# Server starts on http://localhost:11201
```

## Quick Test

```bash
# 1. Login
curl -X POST http://localhost:11201/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
# Save the token from response

# 2. Get DEX params
curl -X GET http://localhost:11201/api/v1/admin/params/dex \
  -H "Authorization: Bearer YOUR_TOKEN"

# 3. Update params
curl -X POST http://localhost:11201/api/v1/admin/params/dex \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "module": "dex",
    "params": {"swap_fee_rate": "0.0025"},
    "reason": "Testing"
  }'
```

## Default Users

| Username | Password    | Role      | Can Do |
|----------|-------------|-----------|--------|
| admin    | admin123    | admin     | Params, Circuit Breaker, Upgrades |
| operator | operator123 | operator  | Params, Circuit Breaker (limited) |
| readonly | readonly123 | readonly  | View only |

## Key Endpoints

### Authentication
```
POST /api/v1/auth/login      # Get token
POST /api/v1/auth/refresh    # Renew token
POST /api/v1/auth/logout     # Invalidate token
```

### Parameters
```
GET  /api/v1/admin/params/:module          # Get params
POST /api/v1/admin/params/:module          # Update params
POST /api/v1/admin/params/:module/reset    # Reset to defaults
GET  /api/v1/admin/params/history          # View history
```

### Circuit Breaker
```
POST /api/v1/admin/circuit-breaker/:module/pause    # Pause module
POST /api/v1/admin/circuit-breaker/:module/resume   # Resume module
GET  /api/v1/admin/circuit-breaker/status           # View status
```

### Emergency (SuperUser only)
```
POST /api/v1/admin/emergency/pause-dex       # Emergency halt DEX
POST /api/v1/admin/emergency/pause-oracle    # Emergency halt Oracle
POST /api/v1/admin/emergency/pause-compute   # Emergency halt Compute
POST /api/v1/admin/emergency/resume-all      # Resume all
```

### Upgrades
```
POST /api/v1/admin/upgrade/schedule    # Schedule upgrade
POST /api/v1/admin/upgrade/cancel      # Cancel upgrade
GET  /api/v1/admin/upgrade/status      # View status
```

## Using Go Client

```go
import "github.com/paw-chain/paw/control-center/admin-api/client"

// Create client
c, _ := client.NewClient(&client.Config{
    BaseURL:  "http://localhost:11201",
    Username: "admin",
    Password: "admin123",
})

// Get params
params, _ := c.GetModuleParams("dex")

// Update params
c.UpdateModuleParams("dex", map[string]interface{}{
    "swap_fee_rate": "0.0025",
}, "Lower fees")

// Pause module
c.PauseModule("dex", "Maintenance", false)

// Resume module
c.ResumeModule("dex", "Maintenance complete")

// Schedule upgrade
c.ScheduleUpgrade("v2.0", 1000000, "Major upgrade")
```

## Rate Limits

| Role      | Write (per min) | Read (per min) |
|-----------|----------------|----------------|
| SuperUser | 20             | 100            |
| Admin     | 15             | 100            |
| Operator  | 10             | 100            |
| ReadOnly  | 0              | 100            |

## Permissions

| Permission           | ReadOnly | Operator | Admin | SuperUser |
|---------------------|----------|----------|-------|-----------|
| read:params         | ✓        | ✓        | ✓     | ✓         |
| update:params       |          | ✓        | ✓     | ✓         |
| pause:module        |          | ✓        | ✓     | ✓         |
| resume:module       |          | ✓        | ✓     | ✓         |
| schedule:upgrade    |          |          | ✓     | ✓         |
| emergency:halt      |          |          |       | ✓         |
| read:audit          | ✓        |          | ✓     | ✓         |
| manage:users        |          |          | ✓     | ✓         |
| multisig:sign       |          |          | ✓     | ✓         |

## Environment Variables

```bash
# Server
export HTTP_PORT=11201
export WEBSOCKET_PORT=11202
export ENVIRONMENT=production

# Auth
export JWT_SECRET=your-secret-key
export TOKEN_EXPIRATION=30m

# Rate Limits
export RATE_LIMIT_ADMIN=10
export RATE_LIMIT_READ=100

# Blockchain
export RPC_URL=http://localhost:11001

# Storage
export DATABASE_URL=postgresql://user:pass@localhost/admin_api
export REDIS_URL=redis://localhost:6379
```

## Common Tasks

### Change User Role
```go
// Via client (when user management is implemented)
client.UpdateUser("user-id", map[string]interface{}{
    "role": "operator",
})
```

### View Audit Logs
```bash
curl -X GET http://localhost:11201/api/v1/admin/audit-log \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -G -d "limit=50" -d "action=update_params"
```

### Emergency Halt
```bash
curl -X POST http://localhost:11201/api/v1/admin/emergency/pause-dex \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Critical vulnerability detected",
    "mfa_code": "123456"
  }'
```

## Testing

```bash
# Run all tests
go test ./control-center/admin-api/... -v -race -cover

# Run specific tests
go test ./control-center/admin-api/tests -run TestAuth -v

# Check coverage
go test ./control-center/admin-api/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Troubleshooting

### 401 Unauthorized
- Token expired → Use `/auth/refresh`
- Invalid token → Login again
- Check `Authorization: Bearer <token>` format

### 403 Forbidden
- Check user role
- Verify permissions for endpoint
- Review RBAC configuration

### 429 Too Many Requests
- Rate limit exceeded
- Wait 60 seconds
- Contact admin for higher limits

### 500 Internal Server Error
- Check server logs
- Verify RPC connection
- Ensure database is accessible

## Metrics (Prometheus)

```
admin_api_requests_total{method,endpoint,status,role}
admin_api_request_duration_seconds{method,endpoint,role}
admin_api_active_users
admin_api_auth_failures_total
```

Access at: `http://localhost:11201/metrics`

## Security Checklist

- [ ] Change default passwords
- [ ] Set strong JWT_SECRET
- [ ] Enable HTTPS/TLS
- [ ] Configure IP whitelist
- [ ] Enable MFA for critical ops
- [ ] Review audit logs regularly
- [ ] Set production rate limits
- [ ] Configure backup/redundancy

## Support

- Full docs: `README.md`
- Examples: `examples/main.go`
- Tests: `tests/`
- Issues: Check server logs first
