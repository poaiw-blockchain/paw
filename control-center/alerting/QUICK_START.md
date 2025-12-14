# PAW Alert Manager - Quick Start Guide

## 5-Minute Setup

### 1. Prerequisites
```bash
# Ensure Docker and Docker Compose are installed
docker --version
docker-compose --version
```

### 2. Clone and Setup
```bash
cd control-center/alerting
./setup.sh
```

This automatically:
- ✅ Creates configuration files
- ✅ Generates JWT secret
- ✅ Starts PostgreSQL, Redis, and Prometheus
- ✅ Launches Alert Manager
- ✅ Verifies health

### 3. Verify Installation
```bash
# Check health
curl http://localhost:11210/health

# Expected response:
# {"status":"healthy","service":"paw-alert-manager","version":"1.0.0"}
```

## Create Your First Alert Rule

### Via API
```bash
curl -X POST http://localhost:11210/api/v1/alerts/rules/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "High CPU Alert",
    "description": "Alert when CPU usage exceeds 80%",
    "source": "infrastructure",
    "severity": "warning",
    "enabled": true,
    "rule_type": "threshold",
    "conditions": [{
      "metric_name": "node_cpu_usage_percent",
      "operator": "gt",
      "threshold": 80.0
    }],
    "evaluation_interval": "30s",
    "for_duration": "2m",
    "channels": ["webhook-test"]
  }'
```

### Via Example Files
```bash
# Load all example rules
cat examples/example-rules.json | jq -c '.[]' | while read rule; do
  curl -X POST http://localhost:11210/api/v1/alerts/rules/create \
    -H "Content-Type: application/json" \
    -d "$rule"
done
```

## Configure Notification Channels

### Email Channel
```bash
curl -X POST http://localhost:11210/api/v1/alerts/channels/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Operations Email",
    "type": "email",
    "enabled": true,
    "config": {
      "to": "ops@yourcompany.com",
      "format": "html"
    }
  }'
```

### Webhook (Slack)
```bash
curl -X POST http://localhost:11210/api/v1/alerts/channels/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Slack DevOps",
    "type": "webhook",
    "enabled": true,
    "config": {
      "url": "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
      "template": "slack"
    }
  }'
```

### Test Channel
```bash
curl -X POST http://localhost:11210/api/v1/alerts/channels/CHANNEL_ID/test
```

## Common Operations

### List Active Alerts
```bash
curl http://localhost:11210/api/v1/alerts?status=active
```

### Acknowledge Alert
```bash
curl -X POST http://localhost:11210/api/v1/alerts/ALERT_ID/acknowledge
```

### Resolve Alert
```bash
curl -X POST http://localhost:11210/api/v1/alerts/ALERT_ID/resolve
```

### View Statistics
```bash
curl http://localhost:11210/api/v1/alerts/stats | jq
```

### List All Rules
```bash
curl http://localhost:11210/api/v1/alerts/rules | jq
```

## Environment Configuration

Edit `.env` file for your environment:

```bash
# Production settings
DATABASE_URL=postgres://user:pass@prod-db:5432/paw_control_center
REDIS_URL=redis://prod-redis:6379/0
ENVIRONMENT=production

# Email settings
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=alerts@yourcompany.com
SMTP_PASSWORD=your-app-password

# SMS settings (Twilio)
TWILIO_ACCOUNT_SID=ACxxxxxx
TWILIO_AUTH_TOKEN=xxxxxx
TWILIO_FROM_NUMBER=+1234567890
```

## Monitoring

### View Logs
```bash
docker-compose logs -f alert-manager
```

### Check Service Status
```bash
docker-compose ps
```

### Database Access
```bash
docker exec -it paw-alerting-postgres psql -U postgres -d paw_control_center
```

### Redis CLI
```bash
docker exec -it paw-alerting-redis redis-cli
```

## Troubleshooting

### Alerts Not Triggering?
1. Check Prometheus connectivity:
   ```bash
   curl http://localhost:9090/api/v1/query?query=up
   ```

2. Verify rule is enabled:
   ```bash
   curl http://localhost:11210/api/v1/alerts/rules/RULE_ID | jq .enabled
   ```

3. Check logs for evaluation errors:
   ```bash
   docker-compose logs alert-manager | grep "error evaluating"
   ```

### Notifications Not Sent?
1. Test the channel:
   ```bash
   curl -X POST http://localhost:11210/api/v1/alerts/channels/CHANNEL_ID/test
   ```

2. Check notification history:
   ```bash
   docker exec -it paw-alerting-postgres psql -U postgres -d paw_control_center \
     -c "SELECT * FROM notifications ORDER BY sent_at DESC LIMIT 10;"
   ```

3. Verify SMTP/Twilio credentials in `.env`

### Service Won't Start?
1. Check port availability:
   ```bash
   lsof -i :11210
   ```

2. Verify database connection:
   ```bash
   psql "postgres://postgres:postgres@localhost:5432/paw_control_center"
   ```

3. Check Docker logs:
   ```bash
   docker-compose logs
   ```

## Next Steps

1. **Read Full Documentation**
   - `README.md` - Complete feature guide
   - `DEPLOYMENT.md` - Production deployment
   - `IMPLEMENTATION_SUMMARY.md` - Technical details

2. **Configure for Production**
   - Set up HTTPS reverse proxy
   - Configure proper JWT secret
   - Set up email/SMS credentials
   - Configure IP whitelist

3. **Create Custom Rules**
   - Review `examples/example-rules.json`
   - Understand rule types (threshold, rate_of_change, composite)
   - Set appropriate evaluation intervals

4. **Integrate with Monitoring**
   - Connect to existing Prometheus
   - Set up Grafana dashboards
   - Configure alerting for Alert Manager itself

## Useful Commands

```bash
# Stop all services
docker-compose down

# Restart services
docker-compose restart

# View resource usage
docker stats

# Clean up
docker-compose down -v  # WARNING: Removes all data

# Update to latest version
git pull
docker-compose build
docker-compose up -d

# Export configuration
curl http://localhost:11210/api/v1/alerts/rules > rules-backup.json
curl http://localhost:11210/api/v1/alerts/channels > channels-backup.json

# Import configuration
cat rules-backup.json | jq -c '.rules[]' | while read rule; do
  curl -X POST http://localhost:11210/api/v1/alerts/rules/create \
    -H "Content-Type: application/json" -d "$rule"
done
```

## Support

- **Documentation**: See `README.md` and `DEPLOYMENT.md`
- **Examples**: See `examples/` directory
- **Issues**: Check logs with `docker-compose logs`
- **Health**: `curl http://localhost:11210/health`

## Architecture Overview

```
┌─────────────────────────────────────────┐
│         Alert Manager (Port 11210)      │
├─────────────────────────────────────────┤
│                                          │
│  Rules Engine ─> Evaluator ─> Alerts   │
│       │              │            │      │
│       ▼              ▼            ▼      │
│  Prometheus    Metrics      Channels    │
│                              (Email/    │
│                               SMS/      │
│                               Webhook)  │
└─────────────────────────────────────────┘
          │              │
          ▼              ▼
    PostgreSQL        Redis
    (Persistence)    (Cache)
```

## Quick Tips

1. **Start Simple**: Begin with threshold rules on basic metrics
2. **Test Channels**: Always test channels before deploying
3. **Use for_duration**: Prevent alert flapping with 1-2 minute durations
4. **Monitor the Monitor**: Set up alerts for the Alert Manager itself
5. **Document Rules**: Use clear descriptions for all rules
6. **Regular Cleanup**: Archive old resolved alerts monthly

---

**Ready to go!** Your Alert Manager is running on http://localhost:11210

For production deployment, see `DEPLOYMENT.md`
