# PAW Unified Control Center - Architecture

## Overview

The Unified Control Center consolidates all monitoring, administrative, and operational capabilities into a single integrated platform. It extends the existing Testing Dashboard with enterprise-grade admin controls, real-time monitoring, and comprehensive audit logging.

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Unified Control Center                                │
│                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                     Frontend (React/Next.js)                         │    │
│  │                                                                       │    │
│  │  ┌──────────────┬───────────────┬──────────────┬──────────────┐    │    │
│  │  │  Dashboard   │    Modules    │  Monitoring  │    Admin     │    │    │
│  │  │  Overview    │   Controls    │  Integration │  Operations  │    │    │
│  │  └──────────────┴───────────────┴──────────────┴──────────────┘    │    │
│  │                                                                       │    │
│  │  Features:                                                            │    │
│  │  - Real-time metrics from analytics service                          │    │
│  │  - Module controls (DEX, Oracle, Compute circuit breakers)           │    │
│  │  - Embedded Prometheus/Grafana dashboards                            │    │
│  │  - Centralized alert management                                      │    │
│  │  - Testing controls (from archived testing dashboard)                │    │
│  │  - Admin operations (parameter changes, emergency controls)          │    │
│  │  - Audit log viewer with filtering/search                            │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                     Backend API (Go)                                 │    │
│  │                                                                       │    │
│  │  ┌──────────────┬───────────────┬──────────────┬──────────────┐    │    │
│  │  │  Read        │   Write       │    Auth      │    Audit     │    │    │
│  │  │  Endpoints   │  Endpoints    │   (RBAC)     │   Logging    │    │    │
│  │  │  (Explorer)  │  (NEW)        │              │              │    │    │
│  │  └──────────────┴───────────────┴──────────────┴──────────────┘    │    │
│  │                                                                       │    │
│  │  New Admin API:                                                       │    │
│  │  - Parameter Management (GET/POST /api/admin/params/:module)         │    │
│  │  - Circuit Breakers (POST /api/admin/circuit-breaker/:module/:action)│    │
│  │  - Emergency Controls (POST /api/admin/emergency/:action)            │    │
│  │  - Audit Log (GET/POST /api/admin/audit-log)                         │    │
│  │  - WebSocket Server (real-time updates on :8081)                     │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                     Integrations                                      │    │
│  │                                                                       │    │
│  │  ┌──────────────┬───────────────┬──────────────┬──────────────┐    │    │
│  │  │ Prometheus   │   Grafana     │  Analytics   │  Blockchain  │    │    │
│  │  │  (Metrics)   │  (Dashboards) │   Service    │     RPC      │    │    │
│  │  └──────────────┴───────────────┴──────────────┴──────────────┘    │    │
│  │                                                                       │    │
│  │  - Prometheus :9090 (metrics collection)                             │    │
│  │  - Grafana :3000 (visualization)                                     │    │
│  │  - Analytics Service (network health: 1,711 lines existing)          │    │
│  │  - Blockchain RPC :26657 (direct node control)                       │    │
│  │  - Alertmanager :9093 (alert routing)                                │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Component Details

### 1. Frontend (Extended Testing Dashboard)

**Base:** Archive testing dashboard (15,586 lines of JavaScript)
**Port:** 11200
**Technology:** HTML5, CSS3, Vanilla JavaScript (no build step)

#### Pages

1. **Dashboard Overview**
   - Real-time network metrics from analytics service
   - Block production health
   - Validator status
   - Transaction throughput
   - System resource usage
   - Alert summary

2. **Modules Page**
   - DEX Controls: Pause/Resume, Emergency Halt, Parameter Tuning
   - Oracle Controls: Pause/Resume, Override Price Feeds, Validator Management
   - Compute Controls: Pause/Resume, Provider Blacklist, Request Throttling
   - Circuit Breaker Status Indicators

3. **Monitoring Page**
   - Embedded Grafana dashboards (iframe integration)
   - Prometheus metrics visualization
   - Custom metric queries
   - Alert history

4. **Testing Page** (From Archived Dashboard)
   - Quick Actions: Send TX, Create Wallet, Delegate, etc.
   - Testing Tools: TX Simulator, Load Testing, Faucet
   - Test Scenarios: Transaction Flow, Staking Flow, etc.

5. **Admin Page** (RBAC Required)
   - Parameter Changes (per module)
   - Emergency Controls (halt chain, maintenance mode)
   - User Management (add/remove admins)
   - System Configuration

6. **Alerts Page**
   - Centralized alert feed from Alertmanager
   - Alert filtering and search
   - Acknowledge/Resolve controls
   - Alert routing configuration

7. **Audit Page**
   - Complete audit log with timestamps
   - Filtering by user/action/module
   - Export functionality (JSON/CSV)
   - Immutable log integrity verification

#### Components (Reused + New)

From Testing Dashboard:
- NetworkSelector.js - Network switching with status
- LogViewer.js - Real-time log display
- MetricsDisplay.js - System metrics visualization
- QuickActions.js - Quick action buttons and modals

New Components:
- AdminControls.js - Admin-only operations
- CircuitBreakerPanel.js - Module circuit breaker controls
- AlertManager.js - Alert management interface
- AuditLogViewer.js - Audit log display and filtering
- GrafanaEmbed.js - Embedded Grafana dashboard

### 2. Backend API

**Language:** Go
**Port:** 11201
**Framework:** Gin (already used in explorer API)

#### Directory Structure

```
control-center/backend/
├── admin-api/
│   ├── handlers.go         # Admin API handlers
│   ├── params.go           # Parameter management
│   ├── circuit_breaker.go  # Circuit breaker controls
│   ├── emergency.go        # Emergency operations
│   └── middleware.go       # Admin-only middleware
├── auth/
│   ├── jwt.go              # JWT token management
│   ├── rbac.go             # Role-based access control
│   ├── twofa.go            # 2FA for critical operations
│   └── session.go          # Session management
├── audit/
│   ├── logger.go           # Audit logging service
│   ├── storage.go          # Database integration
│   └── export.go           # Export functionality
├── websocket/
│   ├── server.go           # WebSocket server
│   ├── handlers.go         # WS message handlers
│   └── broadcaster.go      # Real-time event broadcast
├── integration/
│   ├── prometheus.go       # Prometheus API client
│   ├── grafana.go          # Grafana API client
│   ├── analytics.go        # Analytics service client
│   ├── alertmanager.go     # Alertmanager API client
│   └── blockchain.go       # Blockchain RPC client
└── main.go                 # Application entry point
```

#### API Endpoints

**Read Endpoints (Existing from Explorer API):**
- GET /api/blocks - Recent blocks
- GET /api/transactions - Recent transactions
- GET /api/validators - Validator list
- GET /api/proposals - Governance proposals
- GET /api/pools - DEX liquidity pools
- GET /api/network/health - Network health metrics
- GET /api/metrics - Real-time metrics

**Write Endpoints (NEW - Admin Only):**

Parameter Management:
- GET /api/admin/params/:module - Get current params
- POST /api/admin/params/:module - Update params (requires admin role)
- GET /api/admin/params/history - Audit log of param changes

Circuit Breaker Controls:
- POST /api/admin/circuit-breaker/dex/pause - Pause DEX module
- POST /api/admin/circuit-breaker/dex/resume - Resume DEX module
- POST /api/admin/circuit-breaker/oracle/pause - Pause Oracle module
- POST /api/admin/circuit-breaker/oracle/resume - Resume Oracle module
- POST /api/admin/circuit-breaker/compute/pause - Pause Compute module
- POST /api/admin/circuit-breaker/compute/resume - Resume Compute module
- GET /api/admin/circuit-breaker/status - Get all circuit breaker statuses

Emergency Controls (Requires SuperAdmin + 2FA):
- POST /api/admin/emergency/halt-chain - Emergency chain halt
- POST /api/admin/emergency/enable-maintenance - Enable maintenance mode
- POST /api/admin/emergency/force-upgrade - Trigger forced upgrade
- POST /api/admin/emergency/disable-module/:module - Disable module

Audit Logging:
- GET /api/admin/audit-log - Get audit entries (with pagination/filtering)
- POST /api/admin/audit-log - Create audit entry (automatic on all admin actions)
- GET /api/admin/audit-log/export - Export audit log (JSON/CSV)

Alert Management:
- GET /api/admin/alerts - Get all alerts from Alertmanager
- POST /api/admin/alerts/:id/acknowledge - Acknowledge alert
- POST /api/admin/alerts/:id/resolve - Resolve alert
- GET /api/admin/alerts/config - Get alert routing config

User Management (SuperAdmin Only):
- GET /api/admin/users - List admin users
- POST /api/admin/users - Create new admin user
- PUT /api/admin/users/:id - Update user role
- DELETE /api/admin/users/:id - Remove admin user

WebSocket Endpoints:
- WS /ws/updates - Real-time updates stream
- WS /ws/metrics - Real-time metrics stream
- WS /ws/alerts - Real-time alert stream

### 3. Authentication & Authorization

#### Roles

1. **Viewer** (Read-only)
   - View all dashboards
   - Access monitoring data
   - Cannot modify anything

2. **Operator**
   - Viewer permissions +
   - Run test scenarios
   - Access testing tools
   - Send test transactions

3. **Admin**
   - Operator permissions +
   - Modify module parameters
   - Control circuit breakers
   - Manage alerts

4. **SuperAdmin**
   - Admin permissions +
   - Emergency controls
   - User management
   - System configuration

#### Authentication Flow

```
1. User logs in with credentials
   ↓
2. Backend validates credentials
   ↓
3. Generate JWT token with role claim
   ↓
4. Frontend stores token in localStorage
   ↓
5. All API requests include Authorization header
   ↓
6. Middleware validates token and checks role
   ↓
7. For emergency operations: require 2FA code
   ↓
8. Action executed and logged to audit log
```

#### JWT Token Structure

```json
{
  "sub": "admin@paw.network",
  "role": "Admin",
  "exp": 1735123456,
  "iat": 1735037056,
  "permissions": [
    "read:metrics",
    "write:params",
    "control:circuit-breaker"
  ]
}
```

### 4. Audit Logging

#### Audit Entry Schema

```go
type AuditEntry struct {
    ID          uint      `json:"id"`
    Timestamp   time.Time `json:"timestamp"`
    User        string    `json:"user"`
    Role        string    `json:"role"`
    Action      string    `json:"action"`
    Module      string    `json:"module"`
    Parameters  string    `json:"parameters"`  // JSON
    Result      string    `json:"result"`
    IPAddress   string    `json:"ip_address"`
    UserAgent   string    `json:"user_agent"`
    SessionID   string    `json:"session_id"`
}
```

#### Actions Logged

- All parameter changes
- Circuit breaker state changes
- Emergency control activations
- User management actions
- Failed authentication attempts
- Alert acknowledgements
- Configuration changes

#### Audit Log Storage

- Database: PostgreSQL (existing database from explorer)
- Table: `audit_log` (append-only, no deletes)
- Indexes: timestamp, user, action, module
- Retention: Indefinite (configurable)
- Export: JSON, CSV formats

### 5. Circuit Breaker Implementation

#### Circuit Breaker States

1. **CLOSED** (Normal Operation)
   - Module fully operational
   - All transactions processed
   - No restrictions

2. **OPEN** (Paused)
   - Module temporarily disabled
   - Transactions rejected with clear error
   - Existing state preserved

3. **HALF_OPEN** (Testing)
   - Limited operations allowed
   - Monitoring for recovery
   - Can transition to CLOSED or OPEN

#### Circuit Breaker Controls

**DEX Module:**
- Pause all swaps
- Pause liquidity additions/removals
- Pause pool creation
- Emergency price freeze
- Force settlement

**Oracle Module:**
- Pause price feed updates
- Override specific price feeds
- Disable validator rewards/slashing
- Emergency price lock
- Force aggregation

**Compute Module:**
- Pause new requests
- Blacklist provider
- Rate limit requests
- Emergency refund
- Force result submission

#### Implementation

```go
type CircuitBreaker struct {
    Module      string        `json:"module"`
    State       string        `json:"state"`       // CLOSED, OPEN, HALF_OPEN
    Reason      string        `json:"reason"`
    TrippedAt   *time.Time    `json:"tripped_at"`
    TrippedBy   string        `json:"tripped_by"`
    AutoRecover bool          `json:"auto_recover"`
    RecoverAt   *time.Time    `json:"recover_at"`
}
```

### 6. Real-Time Updates (WebSocket)

#### WebSocket Server

- Port: 8081
- Protocol: RFC 6455 (WebSocket)
- Library: gorilla/websocket

#### Message Types

**Client → Server:**
```json
{
  "type": "subscribe",
  "channels": ["metrics", "alerts", "blocks", "transactions"]
}
```

**Server → Client:**
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

#### Channels

1. **metrics** - Real-time network metrics (1s interval)
2. **alerts** - Alert notifications (immediate)
3. **blocks** - New block notifications (immediate)
4. **transactions** - New transaction notifications (immediate)
5. **audit** - Audit log updates (immediate)
6. **circuit-breaker** - Circuit breaker state changes (immediate)

### 7. Integration Layer

#### Prometheus Integration

- Query API: http://prometheus:9090/api/v1/query
- Range Queries: http://prometheus:9090/api/v1/query_range
- Metrics: block_height, tx_rate, peer_count, consensus_status
- Custom Dashboards: Embedded in Monitoring page

#### Grafana Integration

- API: http://grafana:3000/api
- Embedding: <iframe src="http://grafana:3000/d/dashboard-id?orgId=1&theme=light">
- Dashboards: Network Overview, Module Health, Validator Performance
- Authentication: API key-based

#### Analytics Service Integration

- Endpoint: http://explorer-api:8080/api/analytics
- Metrics: NetworkHealth, BlockProductionMetrics, ValidatorHealthMetrics
- Refresh: Every 5 seconds via WebSocket
- Caching: Redis-backed (existing)

#### Alertmanager Integration

- API: http://alertmanager:9093/api/v2
- Endpoints: /alerts, /silences, /receivers
- Alert Routing: Centralized in Control Center
- Notifications: Email, Slack, Discord

#### Blockchain RPC Integration

- RPC: http://paw-node:26657
- Direct Control: Parameter updates, emergency halt
- Query: Node status, validator info, consensus state
- Transaction Broadcast: Admin operations

## Security Architecture

### 1. Authentication

- JWT tokens with 30-minute expiration
- Refresh tokens with 7-day expiration
- HTTPS only in production (nginx reverse proxy)
- Secure cookie storage (HttpOnly, SameSite=Strict)

### 2. Authorization (RBAC)

- Role-based access control enforced on every request
- Middleware checks role against required permission
- 2FA required for emergency operations
- Session timeout after 30 minutes of inactivity

### 3. Rate Limiting

- Admin API: 10 requests/minute per user
- Read API: 100 requests/minute per IP
- WebSocket: 1 connection per user
- Burst allowance: 2x rate limit

### 4. IP Whitelisting

- Configurable IP whitelist for admin access
- Default: localhost, internal network
- Production: VPN or specific IPs only

### 5. Audit Logging

- All admin actions logged
- All authentication attempts logged
- Immutable log storage (append-only)
- Regular integrity verification

### 6. Input Validation

- All parameters validated against schema
- SQL injection prevention (parameterized queries)
- XSS prevention (escaped output)
- CSRF protection (SameSite cookies)

## Deployment Architecture

### Docker Compose

```yaml
services:
  control-center-frontend:
    image: paw-control-center-frontend:latest
    ports: ["11200:80"]
    environment:
      - API_URL=http://control-center-backend:8080
      - WS_URL=ws://control-center-backend:8081
    depends_on:
      - control-center-backend

  control-center-backend:
    image: paw-control-center-backend:latest
    ports:
      - "11201:8080"  # HTTP API
      - "11202:8081"  # WebSocket
    environment:
      - RPC_URL=http://paw-node1:26657
      - PROMETHEUS_URL=http://prometheus:9090
      - GRAFANA_URL=http://grafana:3000
      - ALERTMANAGER_URL=http://alertmanager:9093
      - ANALYTICS_URL=http://explorer-api:8080
      - DATABASE_URL=postgresql://user:pass@postgres:5432/paw_explorer
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=${JWT_SECRET}
      - ADMIN_WHITELIST=192.168.0.0/16,10.0.0.0/8
    depends_on:
      - postgres
      - redis
      - prometheus
      - grafana
      - alertmanager
    volumes:
      - ./config:/app/config:ro
```

### Network Topology

```
Internet
   ↓
[Nginx Reverse Proxy :443]
   ↓
   ├→ [Control Center Frontend :11200]
   │     ↓
   │  [Control Center Backend :11201]
   │     ↓
   │     ├→ [PostgreSQL :5432] (Audit Log)
   │     ├→ [Redis :6379] (Session Store)
   │     ├→ [Prometheus :9090] (Metrics)
   │     ├→ [Grafana :3000] (Dashboards)
   │     ├→ [Alertmanager :9093] (Alerts)
   │     ├→ [Explorer API :8080] (Analytics)
   │     └→ [PAW Node :26657] (Blockchain)
```

## Performance Considerations

### Caching Strategy

- Redis cache for analytics data (5-second TTL)
- In-memory cache for circuit breaker state
- Browser localStorage for user preferences
- Grafana dashboard cache (1-minute TTL)

### Database Optimization

- Indexes on audit_log: (timestamp, user, action)
- Partitioning: audit_log by month
- Read replicas for heavy query load
- Connection pooling (max 100 connections)

### WebSocket Optimization

- Message batching (max 100ms delay)
- Compression (permessage-deflate)
- Heartbeat every 30 seconds
- Auto-reconnect on disconnect

### Frontend Optimization

- Lazy loading of components
- Virtual scrolling for large tables
- Debounced search inputs
- Memoized renders

## Monitoring & Observability

### Metrics Collected

- API request rate and latency
- WebSocket connection count
- Circuit breaker state changes
- Audit log entry rate
- Authentication success/failure rate
- Session count by role

### Alerts Configured

- Circuit breaker tripped (critical)
- Emergency control activated (critical)
- Failed authentication spike (warning)
- API error rate > 5% (warning)
- WebSocket disconnect rate > 10% (warning)
- Audit log export (info)

### Logging

- Structured JSON logs
- Log levels: DEBUG, INFO, WARNING, ERROR, CRITICAL
- Centralized logging with Loki (optional)
- Log retention: 30 days

## Disaster Recovery

### Backup Strategy

- Audit log: Daily full backup to S3
- Configuration: Git repository
- Database: Continuous replication
- Recovery Time Objective (RTO): 1 hour
- Recovery Point Objective (RPO): 1 hour

### Incident Response

1. Detect issue (monitoring alerts)
2. Assess severity (critical/high/medium/low)
3. Execute emergency procedure:
   - Critical: Halt chain, notify team
   - High: Trip circuit breaker, investigate
   - Medium: Monitor, plan fix
   - Low: Log, fix in next release
4. Document in audit log
5. Post-mortem review

## Extension Points

### Adding New Module Control

1. Add circuit breaker definition in backend
2. Create handler in admin-api/circuit_breaker.go
3. Add UI controls in frontend/components/CircuitBreakerPanel.js
4. Update RBAC permissions
5. Add audit logging

### Adding New Admin Operation

1. Define endpoint in admin-api/handlers.go
2. Add RBAC middleware with required role
3. Implement audit logging
4. Add UI in frontend Admin page
5. Update documentation

### Adding New Integration

1. Create client in backend/integration/
2. Add configuration in config.yml
3. Expose metrics via /api/integration/:name
4. Add UI visualization
5. Configure alerts

## Success Metrics

### Operational

- Admin response time < 2 seconds (p95)
- Circuit breaker activation < 5 seconds
- Audit log integrity 100%
- Authentication success rate > 99%
- WebSocket uptime > 99.9%

### User Experience

- Time to find critical info < 10 seconds
- Single pane of glass for all operations
- Zero need to SSH into nodes for routine ops
- Complete audit trail for compliance

## Future Enhancements

### Phase 2
- Multi-tenancy support
- Advanced analytics (ML-based anomaly detection)
- Custom dashboard builder
- Automated incident response
- Mobile app

### Phase 3
- Distributed tracing integration
- Advanced RBAC (fine-grained permissions)
- Compliance reporting (SOC2, ISO27001)
- Multi-chain support
- API gateway integration
