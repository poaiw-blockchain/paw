# PAW Unified Control Center - User Guide

## Overview

The PAW Unified Control Center is a comprehensive operational dashboard that integrates all monitoring, administrative, and testing capabilities into a single unified platform. It provides enterprise-grade controls for managing the PAW blockchain network.

## Quick Start

### Prerequisites

- Docker and Docker Compose installed
- Access to PAW blockchain node (RPC endpoint)
- PostgreSQL database for audit logging
- Redis for session management (optional but recommended)

### Deploy Control Center

```bash
cd control-center

# Set JWT secret (IMPORTANT: Change in production!)
export JWT_SECRET="your-secure-random-secret-here"

# Start all services
docker-compose up -d

# Check service status
docker-compose ps

# View logs
docker-compose logs -f control-center-backend
```

### Access Control Center

- **Frontend Dashboard**: http://localhost:11200
- **Backend API**: http://localhost:11201
- **Prometheus**: http://localhost:11090
- **Grafana**: http://localhost:11030
- **Alertmanager**: http://localhost:11093

### Default Login Credentials

**SuperAdmin:**
- Email: `admin@paw.network`
- Password: `admin123`
- **CHANGE IMMEDIATELY IN PRODUCTION!**

**Admin:**
- Email: `operator@paw.network`
- Password: `operator123`

## Architecture

The Control Center consists of the following components:

1. **Backend API** (Go) - Authentication, authorization, admin operations, WebSocket
2. **Frontend** (HTML/CSS/JS) - User interface based on existing testing dashboard
3. **PostgreSQL** - Audit log storage
4. **Redis** - Session management and caching
5. **Prometheus** - Metrics collection
6. **Grafana** - Dashboard visualization
7. **Alertmanager** - Alert routing and management

## Features

### 1. Role-Based Access Control (RBAC)

Four role levels with hierarchical permissions:

| Role | Permissions |
|------|-------------|
| **Viewer** | Read-only access to all dashboards and metrics |
| **Operator** | Viewer + Run test scenarios, send test transactions |
| **Admin** | Operator + Modify parameters, control circuit breakers, manage alerts |
| **SuperAdmin** | Admin + Emergency controls, user management, system configuration |

### 2. Dashboard Overview

Real-time network metrics:
- Block production health (average block time, missed blocks)
- Validator health (active validators, total voting power)
- Transaction metrics (TPS, success rate)
- Consensus metrics (consensus rounds, timeouts)
- System resources (CPU, memory, disk)

### 3. Module Controls

#### DEX Module
- **Pause/Resume**: Temporarily halt all DEX operations
- **Emergency Price Freeze**: Lock prices during market anomalies
- **Force Settlement**: Trigger emergency settlement of all positions
- **Parameter Tuning**: Adjust swap fees, slippage limits, pool parameters

#### Oracle Module
- **Pause/Resume**: Halt oracle price feed updates
- **Override Price Feeds**: Manually set prices for specific assets
- **Disable Validator Rewards/Slashing**: Temporarily disable incentives
- **Emergency Price Lock**: Lock all prices at current values
- **Force Aggregation**: Trigger immediate price aggregation

#### Compute Module
- **Pause/Resume**: Halt new compute requests
- **Blacklist Provider**: Block specific compute providers
- **Rate Limit Requests**: Throttle request rate
- **Emergency Refund**: Trigger mass refund of pending requests
- **Force Result Submission**: Bypass timeouts for critical computations

### 4. Circuit Breaker System

Circuit breakers protect modules during emergencies:

**States:**
- **CLOSED**: Normal operation (all transactions processed)
- **OPEN**: Module paused (transactions rejected with clear error)
- **HALF_OPEN**: Testing recovery (limited operations allowed)

**Controls:**
- Manual pause/resume
- Auto-recovery with configurable timeout
- Audit trail of all state changes
- Real-time WebSocket notifications

### 5. Emergency Controls

**CRITICAL: Requires SuperAdmin role + 2FA authentication**

- **Halt Chain**: Emergency stop of block production
- **Enable Maintenance Mode**: Graceful pause with user notifications
- **Force Upgrade**: Trigger mandatory software upgrade at specific height
- **Disable Module**: Completely disable a module (requires restart to re-enable)

### 6. Audit Logging

**Complete immutable audit trail:**

All admin actions are logged with:
- Timestamp (nanosecond precision)
- User email and role
- Action type (update_params, pause_module, etc.)
- Module affected
- Parameters changed (full JSON)
- Result (success/failure)
- IP address and user agent
- Session ID

**Audit Log Access:**
- Web UI with filtering (user, action, module, date range)
- Export to JSON or CSV
- Pagination for large datasets
- Real-time updates via WebSocket

**Storage:**
- PostgreSQL (append-only table, no deletes allowed)
- Indexed by timestamp, user, action, module
- Optional retention policy (default: indefinite)

### 7. Real-Time Updates (WebSocket)

**WebSocket Endpoint:** `ws://localhost:11202/ws/updates?token=<JWT>`

**Channels:**
1. **metrics** - Network metrics (1s interval)
2. **alerts** - Alert notifications (immediate)
3. **blocks** - New block notifications (immediate)
4. **transactions** - New transaction notifications (immediate)
5. **audit** - Audit log updates (immediate)
6. **circuit-breaker** - Circuit breaker state changes (immediate)

**Message Format:**
```json
{
  "type": "metrics",
  "data": {
    "block_height": 123456,
    "tps": 42.5,
    "peer_count": 12
  },
  "timestamp": "2025-12-14T12:00:00Z"
}
```

### 8. Alert Management

**Centralized alert management from Alertmanager:**

- View all active alerts
- Filter by severity (critical, warning, info)
- Acknowledge alerts (silence for 1 hour)
- Resolve alerts (mark as fixed)
- View alert history
- Configure alert routing (email, Slack, Discord)

**Alert Types:**
- Circuit breaker tripped (critical)
- Emergency control activated (critical)
- Failed authentication spike (warning)
- API error rate > 5% (warning)
- High block time (warning)
- Validator down (critical)

### 9. Testing Tools

**Integrated from archived testing dashboard:**

**Quick Actions:**
- Send Transaction
- Create Wallet
- Delegate Tokens
- Submit Proposal
- Swap Tokens
- Query Balance

**Testing Tools:**
- Transaction Simulator (validate before sending)
- Bulk Wallet Generator (create multiple test wallets)
- Load Testing (configurable TX rate and duration)
- Faucet Integration (request test tokens)

**Test Scenarios:**
- Transaction Flow (wallet → faucet → send TX)
- Staking Flow (query validators → delegate)
- Governance Flow (list proposals → submit → vote)
- DEX Trading Flow (query pools → swap → add liquidity)

### 10. Grafana Integration

**Embedded Grafana dashboards:**

Access via Monitoring page in Control Center:

- **Network Overview**: Block production, TPS, peer count
- **Module Health**: DEX, Oracle, Compute metrics
- **Validator Performance**: Uptime, missed blocks, voting power
- **System Resources**: CPU, memory, disk, network I/O

**Direct Access:** http://localhost:11030
- Username: `admin`
- Password: `admin`

## API Reference

### Authentication

**Login:**
```bash
curl -X POST http://localhost:11201/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@paw.network",
    "password": "admin123"
  }'
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": 1735123456,
  "user": {
    "email": "admin@paw.network",
    "role": "SuperAdmin"
  }
}
```

**Use token in subsequent requests:**
```bash
curl -H "Authorization: Bearer <token>" http://localhost:11201/api/admin/params/dex
```

### Parameter Management

**Get Current Parameters:**
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:11201/api/admin/params/dex
```

**Update Parameters:**
```bash
curl -X POST -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  http://localhost:11201/api/admin/params/dex \
  -d '{
    "swap_fee": "0.003",
    "max_slippage": "0.05"
  }'
```

**Get Parameter History:**
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:11201/api/admin/params/history?module=dex
```

### Circuit Breaker Controls

**Pause Module:**
```bash
curl -X POST -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  http://localhost:11201/api/admin/circuit-breaker/dex/pause \
  -d '{
    "reason": "Market anomaly detected",
    "auto_recover": true,
    "recover_in": 60
  }'
```

**Resume Module:**
```bash
curl -X POST -H "Authorization: Bearer <token>" \
  http://localhost:11201/api/admin/circuit-breaker/dex/resume
```

**Get Circuit Breaker Status:**
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:11201/api/admin/circuit-breaker/status
```

### Emergency Controls

**Halt Chain (Requires 2FA):**
```bash
curl -X POST -H "Authorization: Bearer <token>" \
  -H "X-2FA-Code: 123456" \
  -H "Content-Type: application/json" \
  http://localhost:11201/api/admin/emergency/halt-chain \
  -d '{
    "reason": "Critical security vulnerability detected"
  }'
```

### Audit Log

**Query Audit Log:**
```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:11201/api/admin/audit-log?limit=50&offset=0&action=pause_module"
```

**Export Audit Log:**
```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:11201/api/admin/audit-log/export?format=csv" > audit.csv
```

### User Management

**List Users:**
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:11201/api/admin/users
```

**Create User:**
```bash
curl -X POST -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  http://localhost:11201/api/admin/users \
  -d '{
    "email": "newadmin@paw.network",
    "password": "secure_password_123",
    "role": "Admin"
  }'
```

## Security Best Practices

### Production Deployment

1. **Change Default Credentials Immediately**
   ```bash
   # Use curl or web UI to create new SuperAdmin
   # Then delete default admin@paw.network user
   ```

2. **Use Strong JWT Secret**
   ```bash
   # Generate secure random secret:
   openssl rand -base64 64

   # Set in docker-compose.yml:
   JWT_SECRET="<generated-secret>"
   ```

3. **Enable HTTPS**
   - Use nginx reverse proxy with TLS certificates
   - Redirect HTTP to HTTPS
   - Use HSTS header

4. **Configure IP Whitelist**
   ```yaml
   # In docker-compose.yml:
   - ADMIN_WHITELIST=10.0.0.0/8,192.168.0.0/16
   ```

5. **Enable 2FA for All SuperAdmins**
   - Use TOTP (Time-based One-Time Password)
   - Recommend Authy or Google Authenticator

6. **Regular Audit Log Reviews**
   - Weekly review of all admin actions
   - Alert on suspicious patterns
   - Export logs to external SIEM

7. **Backup Audit Logs**
   ```bash
   # Daily backup to S3:
   docker-compose exec postgres pg_dump -U paw paw_control_center > backup.sql
   aws s3 cp backup.sql s3://paw-backups/$(date +%Y%m%d)-audit.sql
   ```

8. **Rate Limiting**
   - Admin API: 10 req/min (configured in backend)
   - Read API: 100 req/min
   - WebSocket: 1 connection per user

9. **Session Management**
   - 30-minute token expiration
   - Automatic session timeout
   - Logout on browser close

10. **Monitoring**
    - Set up alerts for:
      - Failed authentication attempts > 5
      - Emergency controls activated
      - Circuit breaker state changes
      - Unusual audit log patterns

## Troubleshooting

### Backend API Not Starting

**Check logs:**
```bash
docker-compose logs control-center-backend
```

**Common issues:**
- Database connection failed: Check DATABASE_URL
- Redis connection failed: Check REDIS_URL
- JWT secret missing: Set JWT_SECRET environment variable

### WebSocket Connection Failed

**Check:**
1. WebSocket port open (11202)
2. Valid JWT token in query parameter
3. CORS configuration allows WebSocket upgrade

**Test connection:**
```javascript
const ws = new WebSocket('ws://localhost:11202/ws/updates?token=<JWT>');
ws.onopen = () => console.log('Connected');
ws.onerror = (e) => console.error('Error:', e);
```

### Audit Log Not Recording

**Check:**
1. PostgreSQL running and accessible
2. audit_log table created (check backend logs)
3. Database permissions correct

**Manual verification:**
```bash
docker-compose exec postgres psql -U paw -d paw_control_center
SELECT COUNT(*) FROM audit_log;
```

### Circuit Breaker Not Working

**Verify:**
1. Module name is correct (dex, oracle, compute)
2. User has Admin role
3. Circuit breaker state in memory (not persisted - resets on restart)

**Check status:**
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:11201/api/admin/circuit-breaker/status
```

### Grafana Dashboards Not Loading

**Check:**
1. Prometheus running and scraping metrics
2. Grafana provisioning directory mounted correctly
3. Data source configured in Grafana

**Access Grafana directly:**
```
http://localhost:11030
Username: admin
Password: admin
```

## Maintenance

### Backup Strategy

**Database (Audit Log):**
```bash
# Daily full backup
docker-compose exec postgres pg_dump -U paw paw_control_center > daily-backup.sql

# Restore from backup
docker-compose exec -T postgres psql -U paw paw_control_center < daily-backup.sql
```

**Configuration:**
```bash
# Backup all config files
tar -czf config-backup.tar.gz config/

# Backup docker-compose.yml
cp docker-compose.yml docker-compose.yml.backup
```

### Upgrade Procedure

```bash
# 1. Backup current state
docker-compose exec postgres pg_dump -U paw paw_control_center > pre-upgrade-backup.sql

# 2. Stop services
docker-compose down

# 3. Pull latest images
docker-compose pull

# 4. Start services
docker-compose up -d

# 5. Verify health
docker-compose ps
curl http://localhost:11201/health
```

### Log Rotation

**Audit log cleanup (if retention policy set):**
```sql
-- Delete logs older than 90 days
DELETE FROM audit_log WHERE timestamp < NOW() - INTERVAL '90 days';

-- Vacuum to reclaim space
VACUUM FULL audit_log;
```

**Docker logs:**
```bash
# Configure in docker-compose.yml:
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "3"
```

## FAQ

### Q: Can I use this in production?

A: Yes, but follow security best practices:
- Change all default credentials
- Use strong JWT secret
- Enable HTTPS
- Configure IP whitelist
- Enable 2FA for SuperAdmins
- Regular audit log reviews
- Backup audit logs daily

### Q: How do I add custom dashboards to Grafana?

A: Place dashboard JSON files in `config/grafana/dashboards/` and restart Grafana. They will be automatically provisioned.

### Q: Can I integrate with existing monitoring?

A: Yes, the Control Center exposes Prometheus metrics and can forward alerts to your existing Alertmanager or SIEM.

### Q: How do I customize the frontend?

A: The frontend is based on the testing dashboard. Modify files in `frontend/` directory and rebuild:
```bash
docker-compose build control-center-frontend
docker-compose up -d control-center-frontend
```

### Q: What happens if the Control Center goes down?

A: The blockchain continues operating normally. The Control Center is an operational tool, not part of consensus. Audit logs are persisted in PostgreSQL.

### Q: Can I run multiple Control Center instances?

A: Not recommended due to session management and circuit breaker state in memory. Use load balancer with sticky sessions if needed.

### Q: How do I reset admin password?

A: Access PostgreSQL and update user table (implementation depends on auth backend):
```bash
docker-compose exec control-center-backend \
  ./control-center-api reset-password admin@paw.network
```

### Q: Can I export all configuration?

A: Yes, all configuration is in `config/` directory:
```bash
tar -czf control-center-config.tar.gz config/
```

### Q: How do I monitor Control Center itself?

A: Prometheus scrapes Control Center API metrics. View at:
```
http://localhost:11090/targets
http://localhost:11090/graph?g0.expr=up{job="control-center-api"}
```

## Support

For issues, feature requests, or questions:

1. Check this documentation
2. Review ARCHITECTURE.md for technical details
3. Check GitHub issues: https://github.com/paw-chain/paw/issues
4. Contact PAW team: support@paw.network

## Contributing

Contributions welcome! See CONTRIBUTING.md for guidelines.

## License

MIT License - see LICENSE file for details.
