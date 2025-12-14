# PAW Alert Manager - File Index

## Documentation Files

### Getting Started
- **QUICK_START.md** - 5-minute setup guide with common commands
- **README.md** - Complete feature guide, API reference, examples
- **DEPLOYMENT.md** - Production deployment, Kubernetes, security hardening
- **IMPLEMENTATION_SUMMARY.md** - Technical overview and architecture details

### Configuration
- **config.example.yaml** - Complete configuration template with all options
- **.env.example** - Environment variables template (create .env from this)

## Source Code Structure

### Core Types & Configuration
- **types.go** (234 lines) - Alert, Rule, Channel, and all type definitions
- **config.go** (232 lines) - Configuration loading with env var overrides
- **server.go** (152 lines) - Main HTTP server with lifecycle management

### Storage Layer (`storage/`)
- **postgres.go** (698 lines) - Complete PostgreSQL storage implementation
  - Schema auto-initialization
  - CRUD operations for alerts, rules, channels
  - Statistics and analytics queries
  - Redis caching integration

### Rules Engine (`engine/`)
- **evaluator.go** (286 lines) - Rule evaluation logic
  - Threshold rules
  - Rate of change detection
  - Composite rules (AND/OR)
  - Pattern matching (placeholder)
- **rules.go** (313 lines) - Rules engine orchestration
  - Rule loading and scheduling
  - Alert deduplication
  - Alert grouping
  - Handler registration
- **prometheus_provider.go** (133 lines) - Prometheus metrics integration
  - Instant queries
  - Range queries
  - Label-based filtering

### Notification Channels (`channels/`)
- **webhook.go** (211 lines) - Webhook notifications
  - Generic webhook support
  - PagerDuty template
  - Slack template
  - Discord template
- **email.go** (357 lines) - Email notifications
  - SMTP integration (TLS/STARTTLS)
  - HTML email templates
  - Plain text fallback
  - Multi-recipient support
- **sms.go** (99 lines) - SMS notifications
  - Twilio integration
  - Message formatting
  - Character limit handling
- **manager.go** (283 lines) - Notification orchestration
  - Unified channel interface
  - Retry logic with exponential backoff
  - Channel filtering
  - Batch notifications

### API Layer (`api/`)
- **handlers.go** (491 lines) - REST API endpoints
  - Alert management (list, get, ack, resolve)
  - Rule management (CRUD)
  - Channel management (CRUD + test)
  - Statistics and history

### Application Entry Point (`cmd/alert-manager/`)
- **main.go** (36 lines) - Application startup
  - Configuration loading
  - Metrics provider initialization
  - Server creation and startup

### Tests (`tests/`)
- **evaluator_test.go** (307 lines) - Unit tests
  - Rule evaluation tests
  - Operator tests
  - Duration tests
  - Benchmarks
- **integration_test.go** (330 lines) - Integration tests
  - Full API testing
  - Database integration
  - End-to-end scenarios

## Example Configurations

### Alert Rules (`examples/example-rules.json`)
12 production-ready alert rules covering:
1. Critical - Consensus Failure
2. Warning - High Block Latency
3. Critical - TPS Drop
4. Warning - DEX Volume Anomaly
5. Critical - Oracle Price Deviation
6. Warning - Compute Request Backlog
7. Critical - Disk Space
8. Warning - High Memory Usage
9. Critical - Validator Offline
10. Warning - Peer Count Low
11. Info - Chain Upgrade Available
12. Critical - Security: Unusual Transaction Pattern

### Notification Channels (`examples/example-channels.json`)
9 pre-configured notification channels:
1. PagerDuty Critical
2. Slack DevOps Channel
3. Slack Security Channel
4. Discord Alerts
5. Operations Email
6. Security Team Email
7. On-Call SMS
8. All Team Email (Info)
9. Custom Webhook - Internal Monitoring

## Docker & Deployment

- **Dockerfile** - Multi-stage build for production
- **docker-compose.yml** - Complete stack (AlertManager + PostgreSQL + Redis + Prometheus)
- **setup.sh** - Automated setup script

## Dependencies (`go.mod`)

Core Dependencies:
- `gin-gonic/gin` - HTTP framework
- `lib/pq` - PostgreSQL driver
- `redis/go-redis` - Redis client
- `prometheus/client_golang` - Prometheus integration
- `google/uuid` - UUID generation
- `stretchr/testify` - Testing framework

## File Statistics

- **Total Go Code**: 4,430 lines
- **Total Documentation**: 1,765 lines
- **Total Files**: 26 files
- **Test Coverage**: Unit + Integration tests for all core functionality

## Component Breakdown

### Alert Sources (5 types)
1. Network Health - Consensus, validators, peers
2. Security - Threat detection, anomalies
3. Performance - TPS, latency, throughput
4. Module Alerts - DEX, Oracle, Compute
5. Infrastructure - CPU, memory, disk

### Rule Types (4 types)
1. **Threshold** - Simple comparison (value > threshold)
2. **Rate of Change** - Detect spikes/drops
3. **Pattern** - Time-series analysis (extensible)
4. **Composite** - Multiple conditions (AND/OR)

### Notification Channels (5 types)
1. **Webhook** - Generic HTTP webhooks (PagerDuty, Slack, Discord, custom)
2. **Email** - SMTP with HTML templates
3. **SMS** - Twilio integration
4. **Slack** - Direct Slack integration (future)
5. **Discord** - Direct Discord integration (future)

### Alert Lifecycle
1. **Active** - Alert triggered and firing
2. **Acknowledged** - Alert seen by operator
3. **Resolved** - Alert condition no longer true

## API Endpoints

### Alerts (6 endpoints)
- `GET /api/v1/alerts` - List/filter alerts
- `GET /api/v1/alerts/:id` - Get specific alert
- `POST /api/v1/alerts/:id/acknowledge` - Acknowledge
- `POST /api/v1/alerts/:id/resolve` - Resolve
- `GET /api/v1/alerts/history` - View history
- `GET /api/v1/alerts/stats` - Statistics

### Rules (5 endpoints)
- `GET /api/v1/alerts/rules` - List rules
- `GET /api/v1/alerts/rules/:id` - Get rule
- `POST /api/v1/alerts/rules/create` - Create
- `PUT /api/v1/alerts/rules/:id` - Update
- `DELETE /api/v1/alerts/rules/:id` - Delete

### Channels (6 endpoints)
- `GET /api/v1/alerts/channels` - List channels
- `GET /api/v1/alerts/channels/:id` - Get channel
- `POST /api/v1/alerts/channels/create` - Create
- `PUT /api/v1/alerts/channels/:id` - Update
- `DELETE /api/v1/alerts/channels/:id` - Delete
- `POST /api/v1/alerts/channels/:id/test` - Test channel

### Health (2 endpoints)
- `GET /health` - Health check
- `GET /ready` - Readiness check

## Quick Navigation

### I want to...

**Get Started**
→ Read `QUICK_START.md`
→ Run `./setup.sh`

**Deploy to Production**
→ Read `DEPLOYMENT.md`
→ Configure `.env`
→ Run `docker-compose up -d`

**Understand Architecture**
→ Read `IMPLEMENTATION_SUMMARY.md`
→ Review `types.go`

**Create Custom Rules**
→ See `examples/example-rules.json`
→ Read rule types in `engine/evaluator.go`

**Add Notification Channel**
→ See `examples/example-channels.json`
→ Review `channels/` directory

**Run Tests**
→ `go test ./tests -v`
→ `go test ./tests -tags=integration -v`

**API Reference**
→ Read `README.md` (API Reference section)
→ See `api/handlers.go`

**Troubleshoot**
→ Check `QUICK_START.md` (Troubleshooting section)
→ Check `DEPLOYMENT.md` (Troubleshooting section)

## Development Workflow

1. **Local Development**
   ```bash
   ./setup.sh                    # Initial setup
   docker-compose up -d          # Start services
   go run cmd/alert-manager/main.go  # Run locally
   ```

2. **Testing**
   ```bash
   go test ./tests -v            # Unit tests
   go test ./tests -tags=integration -v  # Integration tests
   go test ./... -cover          # Coverage
   ```

3. **Building**
   ```bash
   go build -o alert-manager ./cmd/alert-manager
   docker build -t paw/alert-manager .
   ```

4. **Deployment**
   ```bash
   docker-compose -f docker-compose.yml up -d  # Development
   kubectl apply -f k8s/                       # Production
   ```

## Performance Targets

- Rule Evaluation: <10ms per rule
- API Response: <100ms (p95)
- Alert Detection: <evaluation_interval
- Notification Delivery: <5s (webhook), <10s (email/SMS)
- Database Queries: <50ms (p95)

## Security Checklist

- ✅ JWT authentication support
- ✅ IP whitelisting
- ✅ HTTPS ready (via reverse proxy)
- ✅ Input validation
- ✅ Parameterized SQL queries
- ✅ Secure credential storage
- ✅ Rate limiting
- ✅ CORS configuration

## Integration Points

- ✅ Prometheus (metrics source)
- ✅ PostgreSQL (persistent storage)
- ✅ Redis (caching)
- ⏳ Explorer API (prepared)
- ⏳ Admin API (prepared)
- ⏳ Audit Service (prepared)
- ⏳ Grafana (dashboards ready)

## Support Resources

- **GitHub Issues**: Report bugs and feature requests
- **Slack Channel**: `#paw-alerts` (if available)
- **Documentation**: All `.md` files in this directory
- **Examples**: `examples/` directory
- **Tests**: `tests/` directory

---

**Last Updated**: 2025-12-14
**Version**: 1.0.0
**Status**: Production Ready
