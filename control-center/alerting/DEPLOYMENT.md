# PAW Alert Manager - Deployment Guide

This guide covers deploying the PAW Alert Manager in production environments.

## Prerequisites

- Docker and Docker Compose
- PostgreSQL 15+
- Redis 7+
- Prometheus (for metrics)
- Valid SSL certificates (for production)

## Quick Start (Development)

```bash
# 1. Clone repository
git clone https://github.com/paw/paw-blockchain.git
cd control-center/alerting

# 2. Copy and configure
cp config.example.yaml config.yaml
# Edit config.yaml with your settings

# 3. Start services
docker-compose up -d

# 4. Verify
curl http://localhost:11210/health
```

## Production Deployment

### 1. Environment Setup

Create a `.env` file:

```bash
# Database
DATABASE_URL=postgres://user:password@db.example.com:5432/paw_control_center?sslmode=require
REDIS_URL=redis://:password@redis.example.com:6379/0

# Security
JWT_SECRET=generate-strong-random-secret-here
ADMIN_WHITELIST=10.0.0.0/8,172.16.0.0/12

# Integration URLs
PROMETHEUS_URL=http://prometheus.internal:9090
EXPLORER_URL=http://explorer-api.internal:8080
ADMIN_API_URL=http://control-center.internal:8080

# Email (SMTP)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=alerts@paw.network
SMTP_PASSWORD=your-app-specific-password
SMTP_FROM_ADDRESS=alerts@paw.network

# SMS (Twilio)
TWILIO_ACCOUNT_SID=ACxxxxxxxxxxxxxxxxxxxxxxxxxxxx
TWILIO_AUTH_TOKEN=your-auth-token
TWILIO_FROM_NUMBER=+1234567890

# Slack
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/WEBHOOK/URL
SLACK_BOT_TOKEN=xoxb-your-bot-token

# Discord
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/YOUR/WEBHOOK/URL

# Environment
ENVIRONMENT=production
```

### 2. Database Setup

```bash
# Create database
createdb paw_control_center

# Run migrations (tables are auto-created on first start)
# Or use the SQL schema directly:
psql paw_control_center < schema.sql
```

### 3. Build and Deploy

#### Option A: Docker Compose

```bash
# Build image
docker-compose build

# Start services
docker-compose up -d

# View logs
docker-compose logs -f alert-manager
```

#### Option B: Kubernetes

```yaml
# alerting-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: paw-alert-manager
  namespace: paw-control-center
spec:
  replicas: 2
  selector:
    matchLabels:
      app: alert-manager
  template:
    metadata:
      labels:
        app: alert-manager
    spec:
      containers:
      - name: alert-manager
        image: paw/alert-manager:latest
        ports:
        - containerPort: 11210
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: alerting-secrets
              key: database-url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: alerting-secrets
              key: redis-url
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: alerting-secrets
              key: jwt-secret
        livenessProbe:
          httpGet:
            path: /health
            port: 11210
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 11210
          initialDelaySeconds: 10
          periodSeconds: 5
        resources:
          requests:
            memory: "256Mi"
            cpu: "200m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: alert-manager
  namespace: paw-control-center
spec:
  selector:
    app: alert-manager
  ports:
  - protocol: TCP
    port: 11210
    targetPort: 11210
  type: ClusterIP
```

Deploy:
```bash
kubectl apply -f alerting-deployment.yaml
kubectl apply -f alerting-secrets.yaml
kubectl apply -f alerting-ingress.yaml
```

### 4. Reverse Proxy (Nginx)

```nginx
# /etc/nginx/sites-available/alerts.paw.network
upstream alert_manager {
    server localhost:11210;
    keepalive 32;
}

server {
    listen 443 ssl http2;
    server_name alerts.paw.network;

    ssl_certificate /etc/letsencrypt/live/alerts.paw.network/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/alerts.paw.network/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
    limit_req zone=api_limit burst=20 nodelay;

    location / {
        proxy_pass http://alert_manager;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    location /health {
        proxy_pass http://alert_manager;
        access_log off;
    }

    # Block admin endpoints from public
    location /api/v1/alerts/rules {
        deny all;
    }

    location /api/v1/alerts/channels {
        deny all;
    }
}

# HTTP -> HTTPS redirect
server {
    listen 80;
    server_name alerts.paw.network;
    return 301 https://$server_name$request_uri;
}
```

Enable and reload:
```bash
ln -s /etc/nginx/sites-available/alerts.paw.network /etc/nginx/sites-enabled/
nginx -t
systemctl reload nginx
```

### 5. Initial Configuration

#### Load Example Rules

```bash
# Using curl
for rule in examples/example-rules.json; do
  jq -c '.[]' $rule | while read rule_json; do
    curl -X POST http://localhost:11210/api/v1/alerts/rules/create \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer YOUR_JWT_TOKEN" \
      -d "$rule_json"
  done
done
```

#### Load Example Channels

```bash
for channel in examples/example-channels.json; do
  jq -c '.[]' $channel | while read channel_json; do
    curl -X POST http://localhost:11210/api/v1/alerts/channels/create \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer YOUR_JWT_TOKEN" \
      -d "$channel_json"
  done
done
```

### 6. Monitoring

#### Prometheus Scraping

Add to `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'paw-alert-manager'
    static_configs:
      - targets: ['alert-manager:11210']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

#### Grafana Dashboard

Import the included Grafana dashboard:

```bash
curl -X POST http://grafana:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @grafana-dashboard.json
```

### 7. Backup and Disaster Recovery

#### Database Backup

```bash
# Daily backup script
#!/bin/bash
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
pg_dump paw_control_center > /backups/alerting_$TIMESTAMP.sql
gzip /backups/alerting_$TIMESTAMP.sql

# Retain last 30 days
find /backups -name "alerting_*.sql.gz" -mtime +30 -delete
```

Add to crontab:
```bash
0 2 * * * /usr/local/bin/backup-alerting.sh
```

#### Configuration Backup

```bash
# Backup rules and channels
curl http://localhost:11210/api/v1/alerts/rules > rules_backup.json
curl http://localhost:11210/api/v1/alerts/channels > channels_backup.json
```

### 8. High Availability Setup

For production, deploy multiple replicas:

```yaml
# docker-compose.ha.yml
version: '3.8'

services:
  alert-manager-1:
    <<: *alert-manager-common
    container_name: paw-alert-manager-1

  alert-manager-2:
    <<: *alert-manager-common
    container_name: paw-alert-manager-2

  haproxy:
    image: haproxy:latest
    ports:
      - "11210:11210"
    volumes:
      - ./haproxy.cfg:/usr/local/etc/haproxy/haproxy.cfg:ro
    depends_on:
      - alert-manager-1
      - alert-manager-2
```

HAProxy configuration:
```
global
    maxconn 4096

defaults
    mode http
    timeout connect 5s
    timeout client 50s
    timeout server 50s

frontend alert_manager_front
    bind *:11210
    default_backend alert_manager_back

backend alert_manager_back
    balance roundrobin
    option httpchk GET /health
    server alert1 alert-manager-1:11210 check
    server alert2 alert-manager-2:11210 check
```

## Security Hardening

### 1. Network Security

```bash
# Firewall rules (ufw)
ufw allow from 10.0.0.0/8 to any port 11210
ufw deny 11210
```

### 2. Secret Management

Use HashiCorp Vault or AWS Secrets Manager:

```bash
# Example with Vault
export VAULT_ADDR='http://vault.internal:8200'
vault kv put secret/paw/alerting \
  jwt_secret="$(openssl rand -base64 32)" \
  smtp_password="your-password"
```

### 3. Rate Limiting

Already configured in Nginx, but also enforce at application level in `config.yaml`:

```yaml
rate_limiting:
  enabled: true
  requests_per_minute: 60
  burst: 10
```

## Troubleshooting

### Alerts Not Triggering

```bash
# Check rules engine
docker logs paw-alert-manager | grep "Rules engine"

# Verify Prometheus connectivity
curl http://prometheus:9090/api/v1/query?query=up

# Test rule evaluation manually
curl http://localhost:11210/api/v1/alerts/rules/RULE_ID
```

### Notifications Not Sent

```bash
# Test channel
curl -X POST http://localhost:11210/api/v1/alerts/channels/CHANNEL_ID/test \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Check notification logs
docker logs paw-alert-manager | grep "notification"

# Verify channel config
curl http://localhost:11210/api/v1/alerts/channels/CHANNEL_ID
```

### Database Connection Issues

```bash
# Test connection
psql "postgres://user:pass@db.example.com:5432/paw_control_center"

# Check connection pool
docker exec paw-alert-manager ps aux | grep alert-manager

# Monitor connections
SELECT count(*) FROM pg_stat_activity WHERE datname = 'paw_control_center';
```

## Maintenance

### Upgrade Procedure

```bash
# 1. Backup
./scripts/backup.sh

# 2. Pull new image
docker pull paw/alert-manager:latest

# 3. Stop old version
docker-compose down

# 4. Start new version
docker-compose up -d

# 5. Verify
curl http://localhost:11210/health
docker logs -f paw-alert-manager
```

### Archive Old Alerts

```bash
# Archive alerts older than 90 days
psql paw_control_center <<EOF
BEGIN;
CREATE TABLE IF NOT EXISTS alerts_archive (LIKE alerts INCLUDING ALL);
INSERT INTO alerts_archive SELECT * FROM alerts WHERE created_at < NOW() - INTERVAL '90 days';
DELETE FROM alerts WHERE created_at < NOW() - INTERVAL '90 days';
COMMIT;
EOF
```

## Performance Tuning

### Database Optimization

```sql
-- Add indexes for common queries
CREATE INDEX CONCURRENTLY idx_alerts_created_at_status ON alerts(created_at, status);
CREATE INDEX CONCURRENTLY idx_alerts_severity_source ON alerts(severity, source);

-- Analyze tables
ANALYZE alerts;
ANALYZE alert_rules;
ANALYZE notifications;
```

### Connection Pooling

In `config.yaml`:
```yaml
database:
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m
```

### Redis Caching

```yaml
redis:
  cache_ttl: 5m
  max_connections: 10
```

## Support

- Documentation: https://docs.paw.network/alerting
- Issues: https://github.com/paw/paw-blockchain/issues
- Slack: #paw-alerts
