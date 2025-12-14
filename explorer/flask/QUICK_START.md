# Flask Explorer - Quick Start Guide

## 60 Second Deployment

```bash
# Navigate to directory
cd /home/hudson/blockchain-projects/paw/explorer/flask

# Deploy (interactive wizard)
./deploy.sh

# Access
open http://localhost:11080
```

## Essential Commands

### Deployment
```bash
make up          # Start services
make down        # Stop services
make restart     # Restart services
make rebuild     # Rebuild from scratch
```

### Monitoring
```bash
make health      # Health check
make logs        # View logs
make monitor     # Watch resources
make metrics     # View Prometheus metrics
```

### Troubleshooting
```bash
make logs-flask  # Flask logs only
make logs-nginx  # Nginx logs only
make clean       # Remove all containers/volumes
make clean-cache # Clear nginx cache
```

## Key Endpoints

- **Web UI**: http://localhost:11080
- **Health**: http://localhost:11080/health
- **Readiness**: http://localhost:11080/health/ready
- **API**: http://localhost:11080/api/v1/stats
- **Metrics**: http://localhost:11080/metrics

## Port Assignment

**11080** - Explorer (PAW range: 11000-11999)

## Configuration

Edit `.env` file (create from `.env.example`):
```bash
FLASK_SECRET_KEY=your-random-secret-here
INDEXER_API_URL=http://paw-indexer:8080
RPC_URL=http://paw-node:26657
```

## Docker Commands

```bash
# View status
docker-compose ps

# View logs (follow)
docker-compose logs -f

# Execute shell
docker-compose exec paw-flask /bin/bash

# Stop services
docker-compose down

# Remove everything
docker-compose down -v
```

## Health Verification

```bash
# Quick health check
curl http://localhost:11080/health

# Full readiness (checks backends)
curl http://localhost:11080/health/ready

# Test API
curl http://localhost:11080/api/v1/stats | python3 -m json.tool
```

## Common Issues

**Port in use**:
```bash
netstat -tlnp | grep 11080
```

**Services not ready**:
```bash
docker-compose logs paw-flask
docker network inspect paw-network
```

**High memory**:
```bash
docker stats
export GUNICORN_WORKERS=2
make restart
```

## Directory Structure

```
explorer/flask/
├── app.py                 # Main application
├── docker-compose.yml     # Service orchestration
├── Dockerfile            # Container image
├── nginx.conf            # Reverse proxy
├── requirements.txt      # Python deps
├── .env.example          # Config template
├── templates/            # HTML templates
│   ├── base.html
│   ├── index.html
│   └── ...
├── static/               # Static files
├── Makefile              # Commands
├── deploy.sh             # Deployment script
└── verify.sh             # Verification script
```

## Documentation

- **Complete Guide**: [README.md](README.md)
- **Deployment**: [DEPLOYMENT.md](DEPLOYMENT.md)
- **Implementation**: [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md)

## Support

Run `make help` to see all available commands.

---

**Need Help?** Check logs with `make logs` or `docker-compose logs -f`
