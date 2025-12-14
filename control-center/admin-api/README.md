## Admin API

Comprehensive administrative API for managing the PAW blockchain network. Provides secure, authenticated endpoints for parameter management, circuit breaker controls, emergency operations, and network upgrades.

## Features

### Security
- **JWT-based Authentication**: Secure token-based authentication with configurable expiration
- **Role-Based Access Control (RBAC)**: Four role levels with granular permissions
  - `superuser`: Full access including emergency operations
  - `admin`: Standard administrative operations
  - `operator`: Operational tasks like parameter updates
  - `readonly`: Read-only access to system state
- **Rate Limiting**: Configurable per-role rate limits (10 req/min for writes, 100 req/min for reads)
- **Audit Logging**: Complete audit trail of all administrative actions
- **Multi-signature Support**: Critical operations can require multiple approvals

### Core Functionality

#### 1. Parameter Management
Manage blockchain module parameters with full history tracking:
- Get current parameters for any module
- Update parameters with reason tracking
- Reset parameters to defaults
- View complete parameter change history

#### 2. Circuit Breaker Controls
Pause and resume individual modules without network downtime:
- Pause/resume DEX module
- Pause/resume Oracle module
- Pause/resume Compute module
- View circuit breaker status for all modules

#### 3. Emergency Controls
Critical emergency operations (requires superuser role):
- Emergency pause of individual modules
- Emergency resume all modules
- Requires additional MFA/signature for safety

#### 4. Network Upgrade Management
Schedule and manage network upgrades:
- Schedule upgrades at specific block heights
- Cancel pending upgrades
- View upgrade status and history

## API Endpoints

### Authentication
```bash
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
```

### Parameter Management
```bash
GET    /api/v1/admin/params/:module
POST   /api/v1/admin/params/:module
POST   /api/v1/admin/params/:module/reset
GET    /api/v1/admin/params/history
```

### Circuit Breaker
```bash
POST   /api/v1/admin/circuit-breaker/:module/pause
POST   /api/v1/admin/circuit-breaker/:module/resume
GET    /api/v1/admin/circuit-breaker/status
```

### Emergency Controls
```bash
POST   /api/v1/admin/emergency/pause-dex
POST   /api/v1/admin/emergency/pause-oracle
POST   /api/v1/admin/emergency/pause-compute
POST   /api/v1/admin/emergency/resume-all
```

### Network Upgrade
```bash
POST   /api/v1/admin/upgrade/schedule
POST   /api/v1/admin/upgrade/cancel
GET    /api/v1/admin/upgrade/status
```

## Usage Examples

### 1. Authentication
```bash
# Login
curl -X POST http://localhost:11201/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }'

# Response:
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user_id": "admin-001",
  "username": "admin",
  "role": "admin",
  "expires_at": "2024-01-01T13:30:00Z"
}
```

### 2. Get Module Parameters
```bash
curl -X GET http://localhost:11201/api/v1/admin/params/dex \
  -H "Authorization: Bearer <token>"

# Response:
{
  "module": "dex",
  "params": {
    "min_liquidity": "1000",
    "swap_fee_rate": "0.003",
    "max_slippage": "0.05",
    "enable_limit_orders": true
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### 3. Update Parameters
```bash
curl -X POST http://localhost:11201/api/v1/admin/params/dex \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "module": "dex",
    "params": {
      "swap_fee_rate": "0.0025"
    },
    "reason": "Reduce fees to increase trading volume"
  }'

# Response:
{
  "success": true,
  "message": "Parameters updated successfully for module dex",
  "tx_hash": "0x1234...5678",
  "timestamp": "2024-01-01T12:05:00Z",
  "data": {
    "module": "dex",
    "updated": {
      "swap_fee_rate": "0.0025"
    },
    "old_values": {
      "swap_fee_rate": "0.003"
    }
  }
}
```

### 4. Pause Module (Circuit Breaker)
```bash
curl -X POST http://localhost:11201/api/v1/admin/circuit-breaker/dex/pause \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Suspected vulnerability in swap logic",
    "auto_resume": false
  }'

# Response:
{
  "success": true,
  "message": "Module dex has been paused",
  "tx_hash": "0xabcd...ef01",
  "timestamp": "2024-01-01T12:10:00Z",
  "data": {
    "module": "dex",
    "status": {
      "module": "dex",
      "paused": true,
      "paused_at": "2024-01-01T12:10:00Z",
      "paused_by": "admin",
      "reason": "Suspected vulnerability in swap logic",
      "auto_resume": false
    }
  }
}
```

### 5. Emergency Pause
```bash
curl -X POST http://localhost:11201/api/v1/admin/emergency/pause-dex \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Critical vulnerability detected - immediate action required",
    "mfa_code": "123456"
  }'

# Response:
{
  "success": true,
  "message": "Emergency pause executed for dex module",
  "tx_hash": "0x9876...5432",
  "timestamp": "2024-01-01T12:15:00Z",
  "data": {
    "module": "dex",
    "reason": "Critical vulnerability detected - immediate action required"
  }
}
```

### 6. Schedule Network Upgrade
```bash
curl -X POST http://localhost:11201/api/v1/admin/upgrade/schedule \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "v2.0.0-upgrade",
    "height": 1000000,
    "info": "Major upgrade with new features and security patches"
  }'

# Response:
{
  "success": true,
  "message": "Upgrade 'v2.0.0-upgrade' scheduled for height 1000000",
  "timestamp": "2024-01-01T12:20:00Z",
  "data": {
    "schedule": {
      "name": "v2.0.0-upgrade",
      "height": 1000000,
      "info": "Major upgrade with new features and security patches",
      "scheduled_at": "2024-01-01T12:20:00Z",
      "scheduled_by": "admin",
      "status": "pending"
    }
  }
}
```

## Configuration

### Environment Variables
```bash
# Server configuration
HTTP_PORT=11201
WEBSOCKET_PORT=11202
ENVIRONMENT=production

# Authentication
JWT_SECRET=your-secret-key-here
TOKEN_EXPIRATION=30m

# Rate limiting
RATE_LIMIT_ADMIN=10      # Write operations per minute
RATE_LIMIT_READ=100      # Read operations per minute

# Blockchain connection
RPC_URL=http://localhost:11001

# Database
DATABASE_URL=postgresql://user:pass@localhost/admin_api
REDIS_URL=redis://localhost:6379

# Monitoring
PROMETHEUS_URL=http://localhost:11030
GRAFANA_URL=http://localhost:11031
ALERTMANAGER_URL=http://localhost:11032
```

### Default Users
For testing/development only (change in production):

| Username | Password    | Role      | Permissions |
|----------|-------------|-----------|-------------|
| admin    | admin123    | admin     | Full administrative access |
| operator | operator123 | operator  | Operational tasks |
| readonly | readonly123 | readonly  | Read-only access |

## Security Best Practices

### Production Deployment
1. **Change Default Credentials**: Immediately change all default passwords
2. **Use Strong JWT Secret**: Generate a cryptographically secure random secret
3. **Enable HTTPS**: Always use TLS in production
4. **IP Whitelisting**: Restrict access to known admin IPs
5. **Enable MFA**: Require multi-factor authentication for critical operations
6. **Audit Logs**: Regularly review audit logs for suspicious activity
7. **Rate Limiting**: Tune rate limits based on your security requirements

### Role Assignment Guidelines
- **superuser**: Assign to 2-3 trusted individuals only
- **admin**: Assign to operations team leads
- **operator**: Assign to operations team members
- **readonly**: Assign to monitoring/analytics tools

### Emergency Procedures
1. Keep emergency contacts list updated
2. Establish multi-signature requirements for critical ops
3. Test emergency pause/resume procedures regularly
4. Document rollback procedures for parameter changes

## Testing

### Run All Tests
```bash
cd control-center/admin-api
go test ./... -v -race -cover
```

### Run Specific Test Suites
```bash
# Authentication tests
go test ./tests -run TestAuth -v

# Rate limiting tests
go test ./tests -run TestRateLimit -v

# RBAC tests
go test ./tests -run TestRBAC -v
```

### Coverage Report
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Metrics

The API exposes Prometheus metrics at `/metrics`:

- `admin_api_requests_total`: Total number of API requests
- `admin_api_request_duration_seconds`: Request duration histogram
- `admin_api_active_users`: Number of active authenticated users
- `admin_api_auth_failures_total`: Total authentication failures

## Architecture

```
admin-api/
├── server.go           # Main server implementation
├── types/              # Type definitions and RBAC permissions
│   └── types.go
├── middleware/         # Authentication, RBAC, rate limiting
│   ├── auth.go
│   ├── rbac.go
│   └── ratelimit.go
├── handlers/           # Request handlers
│   ├── params.go
│   ├── circuit_breaker.go
│   └── emergency.go
├── tests/              # Comprehensive test suite
│   ├── auth_test.go
│   ├── ratelimit_test.go
│   └── handlers_test.go
├── client/             # Go client library
│   └── client.go
└── README.md
```

## Client Library

A Go client library is provided for programmatic access:

```go
import "github.com/paw-chain/paw/control-center/admin-api/client"

// Create client
cfg := &client.Config{
    BaseURL:  "http://localhost:11201",
    Username: "admin",
    Password: "admin123",
}
client, err := client.NewClient(cfg)

// Get module params
params, err := client.GetModuleParams("dex")

// Update params
err = client.UpdateModuleParams("dex", map[string]interface{}{
    "swap_fee_rate": "0.0025",
}, "Reduce fees to increase volume")

// Pause module
err = client.PauseModule("dex", "Maintenance required", false)
```

## Troubleshooting

### Common Issues

**401 Unauthorized**
- Check that JWT token is valid and not expired
- Ensure Authorization header format: `Bearer <token>`

**403 Forbidden**
- Verify user has required role/permissions
- Check RBAC configuration

**429 Too Many Requests**
- Rate limit exceeded - wait before retrying
- Consider requesting higher rate limit if needed

**500 Internal Server Error**
- Check server logs for details
- Verify blockchain RPC connection
- Ensure database is accessible

## Contributing

When adding new endpoints:
1. Add types to `types/types.go`
2. Implement handler in `handlers/`
3. Add route in `server.go`
4. Add comprehensive tests
5. Update this README with examples
6. Update client library

## License

Copyright (c) 2024 PAW Chain
