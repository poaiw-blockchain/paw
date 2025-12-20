# Flask Explorer Quick Start

## Deployment (30 seconds)

```bash
# Deploy
./scripts/deploy-flask-explorer.sh

# Access
http://localhost:11080
```

## Stop

```bash
./scripts/stop-flask-explorer.sh
```

## Verify

```bash
./scripts/verify-flask-explorer.sh
```

## URLs

- **Dashboard**: http://localhost:11080/
- **Validators**: http://localhost:11080/validators
- **Search**: http://localhost:11080/search?q=1
- **API Status**: http://localhost:11080/api/status
- **API Block**: http://localhost:11080/api/block/1
- **API Validators**: http://localhost:11080/api/validators

## Logs

```bash
docker logs -f paw-flask-explorer
```

## Configuration

Edit `compose/docker-compose.flask-explorer.yml`:

```yaml
environment:
  - RPC_URL=http://your-node:26657
  - CHAIN_ID=your-chain-id
```

## Requirements

- Docker
- PAW node running on port 26657
- Port 11080 available

## Full Documentation

See `FLASK_EXPLORER_GUIDE.md` for complete documentation.
