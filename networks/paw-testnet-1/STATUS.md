# paw-testnet-1 Artifact Status

**Last updated:** 2026-01-13T04:20:00Z

## Current Repository Contents
- `genesis.json`: Canonical 4-validator genesis (`paw-testnet-1`)
- `genesis.sha256`: SHA256 checksum (0ad9a1be3badff543e777501c74d577249cfc0c13a0759e5b90c544a8688d106)
- `peers.txt`: Persistent peers with actual node IDs and IPs
- `paw-testnet-1-manifest.json`: Machine-readable manifest with full validator details
- `chain.json`: Chain registry format configuration

## Public Endpoints (live)
- RPC: https://testnet-rpc.poaiw.org
- REST: https://testnet-api.poaiw.org
- gRPC: testnet-grpc.poaiw.org:443
- WebSocket: wss://testnet-ws.poaiw.org
- Faucet: https://testnet-faucet.poaiw.org
- Explorer: https://testnet-explorer.poaiw.org
- Status: https://status.poaiw.org

## Validator Infrastructure

| Validator | Server | IP | P2P | RPC | gRPC | REST |
|-----------|--------|-----|-----|-----|------|------|
| val1 | paw-testnet | 54.39.103.49 | 11656 | 11657 | 11090 | 11317 |
| val2 | paw-testnet | 54.39.103.49 | 11756 | 11757 | 11190 | 11417 |
| val3 | services-testnet | 139.99.149.160 | 11856 | 11857 | 11290 | 11517 |
| val4 | services-testnet | 139.99.149.160 | 11956 | 11957 | 11390 | 11617 |

## Artifact Notes
- Testnet is live with 4 validators across 2 OVH servers
- Load balancing via nginx on paw-testnet server
- All public endpoints use poaiw.org domain with Let's Encrypt SSL

Refer to `docs/guides/deployment/PUBLIC_TESTNET.md` for the full deployment runbook.
