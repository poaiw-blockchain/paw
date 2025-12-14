# Flask Explorer Deployment Guide

Complete deployment instructions for the PAW Flask Blockchain Explorer.

## Prerequisites

### Required Services

Before deploying the Flask Explorer, ensure these services are running:

1. **PAW Node** - Blockchain node with RPC endpoint
   - RPC: Port 26657
   - gRPC: Port 9090
   - REST API: Port 1317

2. **Indexer API** - Database indexer service
   - API: Port 8080
   - PostgreSQL: Port 5432
   - Redis: Port 6379

3. **Docker** - Container runtime
   - Docker Engine 20.10+
   - Docker Compose 2.0+

### Port Allocation

The Flask Explorer uses port **11080** (within PAW's allocated range of 11000-11999).

Ensure this port is available:
```bash
# Check if port is in use
netstat -tlnp | grep 11080

# If in use, find the process
lsof -i :11080
```

## Deployment Steps

### 1. Navigate to Flask Directory

```bash
cd /home/hudson/blockchain-projects/paw/explorer/flask
```

### 2. Configure Environment

Create `.env` file from template:
```bash
cp .env.example .env
```

Edit `.env` with your configuration:
```bash
# Minimal production configuration
FLASK_SECRET_KEY=$(openssl rand -hex 32)
INDEXER_API_URL=http://paw-indexer:8080
RPC_URL=http://paw-node:26657
GRPC_URL=paw-node:9090
```

**Important**: Change `FLASK_SECRET_KEY` to a random value in production!

### 3. Verify Network

The Flask Explorer must be on the same Docker network as the indexer and node:

```bash
# Check if paw-network exists
docker network ls | grep paw-network

# If it doesn't exist, create it
docker network create paw-network

# Verify indexer and node are on the network
docker network inspect paw-network
```

### 4. Build and Deploy

Using Docker Compose:
```bash
# Build images
docker-compose build

# Start services
docker-compose up -d

# Verify deployment
docker-compose ps
```

Expected output:
```
NAME                IMAGE                   STATUS
paw-flask           paw-flask:latest        Up 30 seconds (healthy)
paw-nginx           nginx:1.25-alpine       Up 30 seconds (healthy)
```

Using Makefile (recommended):
```bash
# Build and start
make up

# View logs
make logs-follow

# Check health
make health
```

### 5. Verify Deployment

Check all endpoints:
```bash
# Health check
curl http://localhost:11080/health

# Readiness check (verifies backend connectivity)
curl http://localhost:11080/health/ready

# Network stats
curl http://localhost:11080/api/v1/stats

# Prometheus metrics
curl http://localhost:11080/metrics
```

Open in browser:
```bash
# Linux
xdg-open http://localhost:11080

# macOS
open http://localhost:11080

# Or use make command
make open
```

## Configuration Options

### Environment Variables

All configurable via `.env` file:

#### Flask Configuration
- `FLASK_SECRET_KEY` - Session secret key (MUST change in production)
- `FLASK_ENV` - Environment (production/development)
- `FLASK_DEBUG` - Debug mode (false in production)

#### Service Endpoints
- `INDEXER_API_URL` - Indexer API URL (default: http://paw-indexer:8080)
- `RPC_URL` - Node RPC URL (default: http://paw-node:26657)
- `GRPC_URL` - Node gRPC URL (default: paw-node:9090)

#### Performance
- `REQUEST_TIMEOUT` - HTTP request timeout in seconds (default: 30)
- `MAX_ITEMS_PER_PAGE` - Maximum items per page (default: 100)
- `DEFAULT_ITEMS_PER_PAGE` - Default items per page (default: 20)
- `GUNICORN_WORKERS` - Gunicorn worker processes (default: 4)
- `GUNICORN_THREADS` - Threads per worker (default: 2)

#### Logging
- `LOG_LEVEL` - Logging level (info/debug/warning/error)

### Nginx Configuration

Edit `nginx.conf` to customize:

#### Rate Limiting
```nginx
# Current limits:
limit_req_zone ... zone=general:10m rate=100r/s;
limit_req_zone ... zone=api:10m rate=50r/s;
limit_req_zone ... zone=search:10m rate=10r/s;
```

#### Cache Settings
```nginx
proxy_cache_path /var/cache/nginx/explorer
    levels=1:2
    keys_zone=explorer_cache:100m
    max_size=1g
    inactive=60m;
```

#### SSL/TLS (for production)
Uncomment SSL sections in `nginx.conf` and add certificates:
```nginx
listen 443 ssl http2;
ssl_certificate /etc/nginx/ssl/explorer.crt;
ssl_certificate_key /etc/nginx/ssl/explorer.key;
```

## Production Considerations

### 1. Security

**CRITICAL - Before Production:**
- [ ] Change `FLASK_SECRET_KEY` to a random value
- [ ] Enable SSL/TLS in nginx
- [ ] Restrict `/metrics` endpoint to internal IPs
- [ ] Configure CORS origins properly
- [ ] Review rate limiting settings
- [ ] Set up fail2ban for repeated failures
- [ ] Configure firewall rules

**SSL Certificate Setup:**
```bash
# Using Let's Encrypt
certbot certonly --standalone -d explorer.paw.network

# Copy certificates
mkdir -p /etc/nginx/ssl
cp /etc/letsencrypt/live/explorer.paw.network/fullchain.pem /etc/nginx/ssl/explorer.crt
cp /etc/letsencrypt/live/explorer.paw.network/privkey.pem /etc/nginx/ssl/explorer.key

# Update nginx.conf (uncomment SSL sections)
# Restart nginx
docker-compose restart paw-nginx
```

### 2. Performance Tuning

**Gunicorn Workers:**
Calculate optimal workers: `(2 × CPU_cores) + 1`

```bash
# For 8 CPU cores
export GUNICORN_WORKERS=17
```

**Memory Considerations:**
- Each worker uses ~100-200MB RAM
- 4 workers + nginx ≈ 1GB RAM minimum
- Recommended: 2GB+ RAM for production

**Cache Tuning:**
```bash
# Increase nginx cache size for high traffic
# Edit nginx.conf:
keys_zone=explorer_cache:500m    # 500MB cache keys
max_size=5g                       # 5GB max cache
```

### 3. Monitoring

**Health Checks:**
```bash
# Add to monitoring system (Prometheus, Nagios, etc.)
curl -f http://localhost:11080/health || alert
curl -f http://localhost:11080/health/ready || alert
```

**Metrics Collection:**
Configure Prometheus to scrape:
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'paw-flask-explorer'
    static_configs:
      - targets: ['localhost:11080']
    metrics_path: '/metrics'
```

**Log Aggregation:**
```bash
# Ship logs to centralized logging
docker-compose logs -f | your-log-shipper

# Or use Docker logging driver
# Edit docker-compose.yml:
logging:
  driver: "syslog"
  options:
    syslog-address: "tcp://logserver:514"
```

### 4. Backup

**Configuration Backup:**
```bash
# Create backup
make backup

# Or manually
tar -czf explorer-backup-$(date +%Y%m%d).tar.gz \
  .env docker-compose.yml nginx.conf
```

**Restore:**
```bash
tar -xzf explorer-backup-YYYYMMDD.tar.gz
```

### 5. High Availability

**Multi-Instance Deployment:**

1. Deploy multiple Flask instances
2. Use external load balancer (HAProxy, nginx upstream)
3. Share session storage (Redis)

Example `docker-compose.ha.yml`:
```yaml
services:
  paw-flask-1:
    # ... same as paw-flask
  paw-flask-2:
    # ... same as paw-flask

  paw-nginx:
    # Configure upstream in nginx.conf
```

Nginx upstream config:
```nginx
upstream flask_app {
    least_conn;
    server paw-flask-1:5000 max_fails=3;
    server paw-flask-2:5000 max_fails=3;
    keepalive 32;
}
```

## Troubleshooting

### Services Won't Start

```bash
# Check logs
docker-compose logs paw-flask
docker-compose logs paw-nginx

# Check Docker daemon
systemctl status docker

# Check disk space
df -h
```

### Can't Connect to Indexer

```bash
# Verify indexer is running
curl http://localhost:8080/health

# Check network connectivity
docker network inspect paw-network | grep paw-indexer

# Test from Flask container
docker-compose exec paw-flask curl http://paw-indexer:8080/health
```

### High Memory Usage

```bash
# Check container stats
docker stats paw-flask paw-nginx

# Reduce workers
export GUNICORN_WORKERS=2
docker-compose restart paw-flask

# Check for memory leaks
docker-compose logs paw-flask | grep -i memory
```

### Nginx 502 Bad Gateway

```bash
# Check Flask is running
docker-compose ps paw-flask

# Check Flask health
docker-compose exec paw-flask curl localhost:5000/health

# Check nginx logs
docker-compose logs paw-nginx | grep error

# Restart services
docker-compose restart
```

### Slow Response Times

```bash
# Check metrics
curl http://localhost:11080/metrics | grep duration

# Check cache hit rate
curl http://localhost:11080/metrics | grep cache

# Clear nginx cache
make clean-cache

# Check indexer performance
curl http://localhost:8080/metrics
```

## Maintenance

### Daily Tasks

```bash
# Check health
make health

# Check logs for errors
make logs | grep -i error

# Monitor resources
make monitor
```

### Weekly Tasks

```bash
# Review metrics
make metrics

# Check disk usage
docker system df

# Clean unused images
docker image prune -a
```

### Monthly Tasks

```bash
# Update dependencies
docker-compose pull
make rebuild

# Backup configuration
make backup

# Review security logs
make logs | grep -i security
```

### Updates

```bash
# Pull latest code
git pull

# Rebuild and restart
make rebuild

# Verify deployment
make health
make test-endpoints
```

## Rollback Procedure

If deployment fails:

```bash
# Stop new deployment
docker-compose down

# Restore from backup
tar -xzf explorer-backup-YYYYMMDD.tar.gz

# Restart previous version
docker-compose up -d

# Verify
make health
```

## Support

For issues:
1. Check logs: `make logs`
2. Review this guide
3. Check health endpoints
4. Verify backend services (indexer, RPC)
5. Review recent changes in git log

## Appendix

### Port Reference
- **11080** - Explorer web UI and API (nginx)
- **5000** - Flask internal (not exposed)

### Network Reference
- **paw-network** - Docker bridge network (172.20.0.0/16)

### Volume Reference
- **paw-flask-logs** - Flask application logs
- **paw-nginx-logs** - Nginx access/error logs
- **paw-nginx-cache** - Nginx cache storage

### Service Dependencies
```
paw-nginx → paw-flask → paw-indexer → PostgreSQL
                      → paw-node → CometBFT
```

### Resource Requirements

**Minimum:**
- CPU: 1 core
- RAM: 1GB
- Disk: 10GB

**Recommended:**
- CPU: 2+ cores
- RAM: 2GB+
- Disk: 50GB+
- SSD storage for cache

**High Traffic:**
- CPU: 4+ cores
- RAM: 4GB+
- Disk: 100GB+
- SSD storage
- Load balancer
