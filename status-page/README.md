# PAW Status Page

Production-ready status page for paw-mvp-1 with live health probes for RPC, REST, gRPC, Explorer, Faucet, and Metrics.

## Quick start (local)
```bash
cd status-page
go mod tidy
go run .
```
Open http://localhost:11090.

## Docker
```bash
docker compose up -d
```
Environment variables (defaults):
- `STATUS_PAGE_RPC_URL=http://localhost:26657`
- `STATUS_PAGE_REST_URL=http://localhost:1317`
- `STATUS_PAGE_GRPC_ADDR=localhost:9090`
- `STATUS_PAGE_EXPLORER_URL=http://localhost:11080`
- `STATUS_PAGE_FAUCET_HEALTH=http://localhost:8000/api/v1/health`
- `STATUS_PAGE_METRICS_URL=http://localhost:26661/metrics`
- `STATUS_PAGE_INTERVAL=15s`
- `STATUS_PAGE_TIMEOUT=6s`
- `STATUS_PAGE_PORT=11090`

## Features
- Real probes with latency + uptime tracking per component
- gRPC health check for Cosmos services
- Memory-only history capped to avoid bloat
- Self health endpoint: `/healthz`
- JSON status API: `/api/status`
- Static dashboard with auto-refresh and status badges
