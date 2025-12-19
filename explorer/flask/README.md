# PAW Flask Blockchain Explorer

A production-ready blockchain explorer built with Flask, featuring real-time blockchain data visualization, RPC integration, and comprehensive analytics.

## Features

- **Real-time Blockchain Data**: Live blocks, transactions, and network statistics
- **Multi-Module Support**: DEX, Oracle, and Compute module visualization
- **RPC Integration**: Direct integration with PAW node RPC and indexer API
- **Performance Optimized**: Nginx reverse proxy with caching and compression
- **Production Ready**: Docker deployment with health checks, metrics, and logging
- **Responsive UI**: Mobile-friendly Bootstrap 5 interface
- **API Proxy**: Caching API proxy for improved performance

## Architecture

```
┌─────────────┐      ┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Client    │─────▶│    Nginx    │─────▶│    Flask    │─────▶│   Indexer   │
│  (Browser)  │      │  (Port 11080)│      │  (Port 5000)│      │  API (8080) │
└─────────────┘      └─────────────┘      └─────────────┘      └─────────────┘
                            │                     │                     │
                            │                     └────────────────────▶│
                            │                     RPC Node (26657)      │
                            │                                           │
                            ▼                                           ▼
                     Cache (nginx)                              PostgreSQL
```

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Running PAW node (RPC on port 26657)
- Running indexer API (port 8080)

### Deployment

1. **Clone and navigate to the directory**:
   ```bash
   cd /home/hudson/blockchain-projects/paw/explorer/flask
   ```

2. **Create environment file**:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Build and start services**:
   ```bash
   docker-compose up -d
   ```

4. **Verify deployment**:
   ```bash
   # Check service health
   curl http://localhost:11080/health

   # Check readiness
   curl http://localhost:11080/health/ready

   # View logs
   docker-compose logs -f
   ```

5. **Access the explorer**:
   - Web UI: http://localhost:11080
   - API: http://localhost:11080/api/v1/stats
   - Metrics: http://localhost:11080/metrics

### Staging Sandbox

For a self-contained staging stack (nginx → Flask → stub indexer → Postgres/Redis/Prometheus) use:
```bash
cd /home/hudson/blockchain-projects/paw/explorer/flask
./deploy-staging.sh
```
Ports: explorer `11083`, indexer `11081`, Postgres `11432`, Prometheus `11091`. The staging indexer is a stub that only exposes `/health` and empty datasets, so UI data will be sparse until the full pipeline is wired.

Run a quick smoke against the staging stack:
```bash
./staging-smoke.sh
```

### Development Mode

For local development without Docker:

1. **Install dependencies**:
   ```bash
   pip install -r requirements.txt
   ```

2. **Set environment variables**:
   ```bash
   export FLASK_APP=app.py
   export FLASK_ENV=development
   export FLASK_DEBUG=true
   export INDEXER_API_URL=http://localhost:8080
   export RPC_URL=http://localhost:26657
   ```

3. **Run the application**:
   ```bash
   python app.py
   # Or with Flask CLI
   flask run --host=0.0.0.0 --port=5000
   ```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `FLASK_SECRET_KEY` | `dev-secret-key...` | Flask secret key (change in production) |
| `INDEXER_API_URL` | `http://paw-indexer:8080` | Indexer API endpoint |
| `RPC_URL` | `http://paw-node:26657` | PAW node RPC endpoint |
| `GRPC_URL` | `paw-node:9090` | PAW node gRPC endpoint |
| `REQUEST_TIMEOUT` | `30` | HTTP request timeout (seconds) |
| `MAX_ITEMS_PER_PAGE` | `100` | Maximum items per page |
| `DEFAULT_ITEMS_PER_PAGE` | `20` | Default items per page |
| `GUNICORN_WORKERS` | `4` | Number of Gunicorn workers |
| `GUNICORN_THREADS` | `2` | Threads per worker |
| `LOG_LEVEL` | `info` | Logging level |

### Nginx Configuration

The nginx reverse proxy provides:

- **Rate Limiting**: 100 req/s general, 50 req/s API, 10 req/s search
- **Caching**: 1 minute HTML, 30 seconds API responses
- **Compression**: Gzip compression for text content
- **Security Headers**: XSS, frame options, content type sniffing protection
- **Connection Limits**: Max 10 concurrent connections per IP

Edit `nginx.conf` to customize these settings.

## API Endpoints

All API endpoints are proxied through nginx with caching:

### Blocks
- `GET /api/v1/blocks` - List latest blocks
- `GET /api/v1/blocks/{height}` - Get block by height

### Transactions
- `GET /api/v1/transactions` - List latest transactions
- `GET /api/v1/transactions/{hash}` - Get transaction by hash

### Stats
- `GET /api/v1/stats` - Network statistics

### Search
- `GET /api/v1/search?q={query}` - Search blocks, transactions, addresses

## Web Pages

### Main Pages
- `/` - Dashboard with network overview
- `/blocks` - Block list
- `/block/{height}` - Block details
- `/transactions` - Transaction list
- `/tx/{hash}` - Transaction details
- `/account/{address}` - Account details
- `/validators` - Validator list
- `/validator/{address}` - Validator details

### Module Pages
- `/dex` - DEX pools overview
- `/dex/pool/{id}` - Pool details
- `/oracle` - Oracle prices
- `/compute` - Compute jobs

### Utility Pages
- `/search` - Search page
- `/health` - Health check
- `/metrics` - Prometheus metrics

## Monitoring

### Health Checks

```bash
# Basic health
curl http://localhost:11080/health

# Readiness check (verifies indexer and RPC connectivity)
curl http://localhost:11080/health/ready
```

### Prometheus Metrics

Available at `http://localhost:11080/metrics`:

- `flask_explorer_requests_total` - Total requests by method, endpoint, status
- `flask_explorer_request_duration_seconds` - Request latency histogram
- `flask_explorer_active_requests` - Active requests gauge
- `flask_explorer_rpc_errors_total` - RPC error count
- `flask_explorer_cache_hits_total` - Cache hit count

### Logs

```bash
# View all logs
docker-compose logs -f

# Flask logs only
docker-compose logs -f paw-flask

# Nginx logs only
docker-compose logs -f paw-nginx
```

## Performance Tuning

### Gunicorn Workers

Calculate optimal workers: `(2 × CPU cores) + 1`

```bash
# For 4 cores
export GUNICORN_WORKERS=9
```

### Nginx Cache

Edit `nginx.conf` to adjust cache settings:

```nginx
proxy_cache_path /var/cache/nginx/explorer
    levels=1:2
    keys_zone=explorer_cache:100m    # 100MB cache keys
    max_size=1g                       # 1GB max cache size
    inactive=60m;                     # 60 min inactive timeout
```

### Redis Cache (Future)

To use Redis for application caching:

1. Add Redis to `docker-compose.yml`
2. Update `.env`:
   ```
   CACHE_TYPE=redis
   CACHE_REDIS_URL=redis://paw-redis:6379/0
   ```

## Security

### Production Checklist

- [ ] Change `FLASK_SECRET_KEY` to a random value
- [ ] Enable SSL/TLS (uncomment HTTPS section in `nginx.conf`)
- [ ] Restrict `/metrics` endpoint (uncomment IP whitelist in `nginx.conf`)
- [ ] Configure CORS origins (update `CORS_ORIGINS` in `.env`)
- [ ] Review rate limiting settings
- [ ] Enable fail2ban for repeated failures
- [ ] Set up firewall rules

### SSL/TLS Setup

1. Obtain SSL certificates (Let's Encrypt, etc.)
2. Place certificates in `/etc/nginx/ssl/`
3. Uncomment SSL sections in `nginx.conf`
4. Restart nginx: `docker-compose restart paw-nginx`

## Troubleshooting

### Services won't start

```bash
# Check Docker logs
docker-compose logs

# Verify network connectivity
docker network inspect paw-network

# Check port conflicts
netstat -tlnp | grep 11080
```

### Can't connect to indexer

```bash
# Test indexer directly
curl http://localhost:8080/health

# Check if indexer is in same network
docker network inspect paw-network | grep indexer
```

### High memory usage

```bash
# Check container stats
docker stats

# Reduce Gunicorn workers
export GUNICORN_WORKERS=2

# Restart services
docker-compose restart
```

### Nginx 502 Bad Gateway

```bash
# Check Flask is running
docker-compose ps paw-flask

# Check Flask logs
docker-compose logs paw-flask

# Test Flask directly
curl http://localhost:5000/health
```

## Development

### Adding New Pages

1. Add route in `app.py`:
   ```python
   @app.route('/new-page')
   @track_metrics
   def new_page():
       return render_template('new-page.html')
   ```

2. Create template in `templates/new-page.html`:
   ```html
   {% extends "base.html" %}
   {% block content %}
   <!-- Your content -->
   {% endblock %}
   ```

### Adding New API Endpoints

1. Add RPC client method in `app.py`:
   ```python
   def get_new_data(self) -> Optional[Dict]:
       url = f"{self.indexer_url}/api/v1/new-endpoint"
       return self._make_request(url)
   ```

2. Add route:
   ```python
   @app.route('/api/v1/new-endpoint')
   @track_metrics
   @cache.cached(timeout=60)
   def api_new_endpoint():
       data = rpc_client.get_new_data()
       return jsonify(data)
   ```

## Maintenance

### Clearing Cache

```bash
# Clear nginx cache
docker-compose exec paw-nginx rm -rf /var/cache/nginx/*

# Restart nginx to rebuild cache
docker-compose restart paw-nginx
```

### Updating

```bash
# Pull latest changes
git pull

# Rebuild images
docker-compose build --no-cache

# Restart services
docker-compose up -d
```

### Backup

```bash
# Backup configuration
tar -czf flask-explorer-backup.tar.gz .env docker-compose.yml nginx.conf

# Restore
tar -xzf flask-explorer-backup.tar.gz
```

## License

Copyright (c) 2024 PAW Chain Team. All rights reserved.
