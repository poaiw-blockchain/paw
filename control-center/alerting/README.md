# PAW Alert Manager

Centralized alert management system for the PAW blockchain network. Provides real-time monitoring, rule-based alerting, and multi-channel notifications.

## Features

### Alert Sources
- **Network Health**: Consensus failures, validator issues, peer connectivity
- **Security**: Anomaly detection, attack patterns, unauthorized access
- **Performance**: TPS degradation, latency spikes, resource exhaustion
- **Module Alerts**: DEX volume anomalies, Oracle deviations, Compute failures
- **Infrastructure**: Disk space, memory, CPU usage

### Alert Rules Engine
- **Threshold-based**: Trigger when metric exceeds/falls below threshold
- **Rate of Change**: Detect sudden metric changes (spikes/drops)
- **Pattern Matching**: Time-series analysis and anomaly detection
- **Composite Rules**: Multiple conditions with AND/OR logic
- **Deduplication**: Prevent duplicate alerts
- **Grouping**: Consolidate similar alerts

### Notification Channels
- **Webhook**: PagerDuty, custom webhooks
- **Email**: SMTP with HTML templates
- **SMS**: Twilio integration
- **Slack**: Direct messages and channel posts
- **Discord**: Server notifications

### Alert Management
- Acknowledge alerts
- Resolve alerts
- View alert history
- Alert statistics and analytics
- Escalation policies

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Alert Manager                             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ Rules Engine │  │  Evaluator   │  │   Storage    │     │
│  │              │─>│              │─>│  (Postgres)  │     │
│  │ • Load Rules │  │ • Threshold  │  │  • Alerts    │     │
│  │ • Schedule   │  │ • Rate Change│  │  • Rules     │     │
│  │ • Evaluate   │  │ • Pattern    │  │  • Channels  │     │
│  └──────────────┘  │ • Composite  │  └──────────────┘     │
│                     └──────────────┘                        │
│                            ↓                                │
│  ┌──────────────────────────────────────────────────────┐  │
│  │          Notification Manager                        │  │
│  │                                                       │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────┐ │  │
│  │  │ Webhook  │ │  Email   │ │   SMS    │ │ Slack  │ │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └────────┘ │  │
│  └──────────────────────────────────────────────────────┘  │
│                            ↓                                │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  REST API                            │  │
│  │  • Create/Update/Delete Rules                        │  │
│  │  • Manage Channels                                   │  │
│  │  • Acknowledge/Resolve Alerts                        │  │
│  │  • View Statistics                                   │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### Installation

```bash
cd control-center/alerting
go build -o alert-manager ./cmd/alert-manager
```

### Configuration

1. Copy example config:
```bash
cp config.example.yaml config.yaml
```

2. Edit `config.yaml` with your settings

3. Set environment variables:
```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/paw_control_center"
export REDIS_URL="redis://localhost:6379/0"
export JWT_SECRET="your-secret-key"
export PROMETHEUS_URL="http://localhost:9090"
```

### Running

```bash
./alert-manager -config config.yaml
```

Or with Docker:

```bash
docker-compose up -d
```

## API Reference

### Alerts

#### List Alerts
```bash
GET /api/v1/alerts?status=active&severity=critical&limit=100
```

Response:
```json
{
  "alerts": [
    {
      "id": "alert-123",
      "rule_id": "rule-456",
      "rule_name": "High CPU Usage",
      "source": "infrastructure",
      "severity": "critical",
      "status": "active",
      "message": "CPU usage at 95%",
      "value": 95.0,
      "threshold": 90.0,
      "created_at": "2025-12-14T10:00:00Z"
    }
  ],
  "total": 1
}
```

#### Get Alert
```bash
GET /api/v1/alerts/:id
```

#### Acknowledge Alert
```bash
POST /api/v1/alerts/:id/acknowledge
```

#### Resolve Alert
```bash
POST /api/v1/alerts/:id/resolve
```

#### Get Alert Statistics
```bash
GET /api/v1/alerts/stats
```

Response:
```json
{
  "total_alerts": 150,
  "active_alerts": 5,
  "acknowledged_alerts": 10,
  "resolved_alerts": 135,
  "by_severity": {
    "critical": 2,
    "warning": 8,
    "info": 140
  },
  "mean_time_to_acknowledge": "5m30s",
  "mean_time_to_resolve": "15m45s"
}
```

### Rules

#### List Rules
```bash
GET /api/v1/alerts/rules?enabled=true
```

#### Create Rule
```bash
POST /api/v1/alerts/rules/create
Content-Type: application/json

{
  "name": "High CPU Usage",
  "description": "Alert when CPU usage exceeds 90%",
  "source": "infrastructure",
  "severity": "critical",
  "enabled": true,
  "rule_type": "threshold",
  "conditions": [
    {
      "metric_name": "node_cpu_usage_percent",
      "operator": "gt",
      "threshold": 90.0
    }
  ],
  "evaluation_interval": "30s",
  "for_duration": "2m",
  "channels": ["webhook-pagerduty", "email-ops"]
}
```

#### Update Rule
```bash
PUT /api/v1/alerts/rules/:id
```

#### Delete Rule
```bash
DELETE /api/v1/alerts/rules/:id
```

### Channels

#### List Channels
```bash
GET /api/v1/alerts/channels
```

#### Create Channel
```bash
POST /api/v1/alerts/channels/create
Content-Type: application/json

{
  "name": "PagerDuty Webhook",
  "type": "webhook",
  "enabled": true,
  "config": {
    "url": "https://events.pagerduty.com/v2/enqueue",
    "template": "pagerduty",
    "headers": {
      "Authorization": "Token token=YOUR_API_KEY"
    }
  },
  "filters": [
    {
      "field": "severity",
      "operator": "in",
      "values": ["critical", "warning"]
    }
  ]
}
```

#### Test Channel
```bash
POST /api/v1/alerts/channels/:id/test
```

## Example Rules

### Network Health Monitoring

```json
{
  "name": "Consensus Failure",
  "source": "network_health",
  "severity": "critical",
  "rule_type": "threshold",
  "conditions": [
    {
      "metric_name": "consensus_failed_rounds",
      "operator": "gt",
      "threshold": 3.0
    }
  ],
  "evaluation_interval": "10s",
  "for_duration": "1m",
  "channels": ["pagerduty", "sms-oncall"]
}
```

### Performance Monitoring

```json
{
  "name": "TPS Degradation",
  "source": "performance",
  "severity": "warning",
  "rule_type": "rate_of_change",
  "conditions": [
    {
      "metric_name": "transactions_per_second",
      "operator": "lt",
      "threshold": -50.0,
      "duration": "5m"
    }
  ],
  "evaluation_interval": "30s",
  "channels": ["slack-devops"]
}
```

### DEX Monitoring

```json
{
  "name": "DEX Volume Anomaly",
  "source": "module_dex",
  "severity": "warning",
  "rule_type": "composite",
  "composite_op": "AND",
  "conditions": [
    {
      "metric_name": "dex_volume_24h",
      "operator": "gt",
      "threshold": 1000000.0
    },
    {
      "metric_name": "dex_unique_users_24h",
      "operator": "lt",
      "threshold": 10.0
    }
  ],
  "channels": ["email-security"]
}
```

### Infrastructure Monitoring

```json
{
  "name": "Disk Space Critical",
  "source": "infrastructure",
  "severity": "critical",
  "rule_type": "threshold",
  "conditions": [
    {
      "metric_name": "node_disk_usage_percent",
      "operator": "gt",
      "threshold": 95.0
    }
  ],
  "evaluation_interval": "1m",
  "for_duration": "5m",
  "channels": ["pagerduty", "email-ops"]
}
```

## Integration with Prometheus

The alert manager integrates with Prometheus for metric collection:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'paw-node'
    static_configs:
      - targets: ['localhost:26660']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

## Security

### Authentication
All API endpoints require JWT authentication:

```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  http://localhost:11210/api/v1/alerts
```

### IP Whitelisting
Configure allowed IPs in `config.yaml`:

```yaml
admin_whitelist:
  - 192.168.0.0/16
  - 10.0.0.0/8
```

### HTTPS
In production, use a reverse proxy (nginx/traefik) for HTTPS:

```nginx
server {
    listen 443 ssl;
    server_name alerts.paw.network;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:11210;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Performance Tuning

### Rule Evaluation
- Keep `evaluation_interval` reasonable (10-60s)
- Use `for_duration` to prevent flapping
- Limit concurrent evaluations with `max_concurrent_evals`

### Database
- Use PostgreSQL connection pooling
- Add indexes on frequently queried columns
- Archive old alerts regularly

### Notifications
- Enable batching for high-volume alerts
- Use deduplication to reduce noise
- Configure appropriate retry policies

## Monitoring

The alert manager exposes its own metrics:

```
# Alert statistics
alertmanager_alerts_total{severity="critical"} 5
alertmanager_alerts_total{severity="warning"} 20

# Rule evaluation
alertmanager_evaluations_total{rule="high-cpu"} 1000
alertmanager_evaluation_duration_seconds{rule="high-cpu"} 0.05

# Notifications
alertmanager_notifications_total{channel="pagerduty",success="true"} 50
alertmanager_notifications_total{channel="pagerduty",success="false"} 2
```

## Troubleshooting

### Alerts Not Triggering
1. Check rule is enabled: `GET /api/v1/alerts/rules/:id`
2. Verify metric exists in Prometheus
3. Check evaluation logs
4. Test rule manually

### Notifications Not Sent
1. Test channel: `POST /api/v1/alerts/channels/:id/test`
2. Check notification history
3. Verify channel configuration
4. Check retry count

### High Memory Usage
1. Reduce alert retention period
2. Enable deduplication
3. Batch notifications
4. Archive old alerts

## Development

### Running Tests
```bash
go test ./... -v
```

### Running Integration Tests
```bash
# Start test database
docker-compose -f docker-compose.test.yml up -d

# Run tests
go test ./tests -tags=integration -v
```

### Building
```bash
go build -o alert-manager ./cmd/alert-manager
```

## License

Copyright 2025 PAW Network
