# PAW Blockchain - Docker Deployment Guide

This directory contains Docker configurations for running PAW blockchain nodes and infrastructure.

## Quick Start

### Prerequisites

- Docker Engine 24.0+
- Docker Compose 2.20+
- Minimum 4GB RAM
- 100GB free disk space (recommended for full node)

### 1. Build the Image

```bash
# From project root
cd docker
docker build -t paw-chain/paw:latest -f Dockerfile ..
```

### 2. Run with Docker Compose

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f paw-node

# Stop all services
docker-compose down
```

### 3. Access Services

- **RPC**: http://localhost:26657
- **REST API**: http://localhost:1317
- **Custom API**: http://localhost:5000
- **Prometheus**: http://localhost:9091
- **Grafana**: http://localhost:3001 (admin/admin)
- **AlertManager**: http://localhost:9093

## Configuration

### Environment Variables

Create a `.env` file in the docker directory:

```bash
# Chain Configuration
CHAIN_ID=paw-1
MONIKER=my-paw-node

# Network
PERSISTENT_PEERS=node1@ip1:26656,node2@ip2:26656
SEEDS=seed1@ip1:26656

# State Sync (Optional - faster initial sync)
STATE_SYNC_ENABLED=false
STATE_SYNC_RPC_SERVERS=http://rpc1:26657,http://rpc2:26657
STATE_SYNC_TRUST_HEIGHT=1000000
STATE_SYNC_TRUST_HASH=ABC123...

# API Configuration
JWT_SECRET=your-secret-key-here
TLS_ENABLED=false

# Monitoring
GRAFANA_USER=admin
GRAFANA_PASSWORD=secure-password
```

### Custom Genesis File

Place your genesis file at `docker/genesis.json` before starting:

```bash
curl -L https://rawhubusercontent.com/paw-chain/networks/main/mainnet/genesis.json \
  -o genesis.json
```

## Docker Commands

### Node Management

```bash
# Initialize node only
docker-compose run --rm paw-node init

# Validate genesis
docker-compose run --rm paw-node validate-genesis

# Check version
docker-compose run --rm paw-node version

# Run shell
docker-compose exec paw-node sh

# View node status
docker-compose exec paw-node pawd status
```

### Data Management

```bash
# Backup node data
docker run --rm -v paw-data:/data -v $(pwd):/backup \
  alpine tar czf /backup/paw-backup-$(date +%Y%m%d).tar.gz /data

# Restore node data
docker run --rm -v paw-data:/data -v $(pwd):/backup \
  alpine tar xzf /backup/paw-backup-20231201.tar.gz -C /

# Reset node data (DANGER: deletes all data)
docker-compose down -v
```

### Logs and Debugging

```bash
# View real-time logs
docker-compose logs -f paw-node

# View last 100 lines
docker-compose logs --tail=100 paw-node

# Export logs
docker-compose logs --no-color paw-node > paw-node.log

# Check container health
docker inspect --format='{{.State.Health.Status}}' paw-node
```

## Production Deployment

### 1. Enable TLS

```bash
# Generate certificates (or use Let's Encrypt)
./scripts/generate-tls-certs.sh

# Update docker-compose.yml
environment:
  - TLS_ENABLED=true
volumes:
  - ./certs/cert.pem:/certs/cert.pem:ro
  - ./certs/key.pem:/certs/key.pem:ro
```

### 2. Configure Resources

Add resource limits to `docker-compose.yml`:

```yaml
services:
  paw-node:
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 8G
        reservations:
          cpus: '2'
          memory: 4G
```

### 3. Enable Auto-restart

```yaml
services:
  paw-node:
    restart: always # Instead of unless-stopped
```

### 4. External Volumes

For production, use named volumes or mount host directories:

```yaml
volumes:
  paw-data:
    driver: local
    driver_opts:
      type: none
      device: /mnt/paw-data
      o: bind
```

## Monitoring Setup

### Grafana Dashboards

1. Access Grafana at http://localhost:3001
2. Login with admin/admin (change password)
3. Pre-configured dashboards:
   - PAW Node Metrics
   - Blockchain Performance
   - P2P Network Status
   - Transaction Analytics

### Prometheus Metrics

Available at http://localhost:9091

Key metrics:

- `tendermint_consensus_height` - Current block height
- `tendermint_consensus_validators` - Active validators
- `tendermint_mempool_size` - Transactions in mempool
- `tendermint_p2p_peers` - Connected peers

### AlertManager

Configure alerts in `monitoring/alertmanager.yml`:

```yaml
receivers:
  - name: 'email'
    email_configs:
      - to: 'alerts@yourcompany.com'
        from: 'paw-alerts@yourcompany.com'
        smarthost: 'smtp.gmail.com:587'
```

## Troubleshooting

### Node Won't Start

```bash
# Check logs
docker-compose logs paw-node

# Verify genesis file
docker-compose run --rm paw-node validate-genesis

# Reset and reinitialize
docker-compose down
docker volume rm paw-data
docker-compose up -d
```

### Sync Issues

```bash
# Check sync status
curl http://localhost:26657/status | jq .result.sync_info

# Enable state sync for faster sync
STATE_SYNC_ENABLED=true docker-compose up -d
```

### Memory Issues

```bash
# Increase Docker memory limit
# Docker Desktop: Settings > Resources > Memory

# Or reduce cache size in app.toml
sed -i 's/iavl-cache-size = 781250/iavl-cache-size = 100000/' \
  ~/.paw/config/app.toml
```

### Network Connectivity

```bash
# Test P2P connectivity
docker-compose exec paw-node nc -zv peer-ip 26656

# Check firewall
sudo ufw allow 26656/tcp
sudo ufw allow 26657/tcp
```

## Advanced Configuration

### Running Multiple Nodes

```bash
# Scale horizontally
docker-compose up -d --scale paw-node=3

# Each node needs unique ports - use docker-compose.override.yml
```

### Custom Build Args

```dockerfile
# Build with specific Go version
docker build --build-arg GO_VERSION=1.23.1 \
  -t paw-chain/paw:custom -f Dockerfile ..
```

### Testnet Deployment

```bash
# Use testnet genesis
CHAIN_ID=paw-mvp-1 docker-compose up -d

# Connect to testnet seeds
SEEDS=testnet-seed@testnet.paw-chain.io:26656
```

## Security Best Practices

1. **Never expose RPC to public internet** without authentication
2. **Use TLS** for all production deployments
3. **Rotate JWT secrets** regularly
4. **Enable firewall** rules
5. **Monitor logs** for suspicious activity
6. **Backup regularly** and test restores
7. **Use secrets management** for sensitive data
8. **Run as non-root** (already configured)
9. **Keep images updated** with security patches
10. **Limit resource usage** to prevent DoS

## Maintenance

### Updates

```bash
# Pull latest code
 pull

# Rebuild image
docker-compose build --no-cache

# Restart with new image
docker-compose up -d
```

### Backups

```bash
# Automated backup script
./scripts/deploy/backup.sh docker
```

### Health Checks

```bash
# Automated health monitoring
while true; do
  curl -f http://localhost:26657/health || \
    docker-compose restart paw-node
  sleep 60
done
```

## Support

- Documentation: https://docs.paw-chain.io
- Issues: https://github.com/paw-chain/paw/issues
- Discord: https://discord.gg/DBHTc2QV
- Email: support@paw-chain.io
