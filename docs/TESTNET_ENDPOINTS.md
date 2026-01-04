# PAW Public Testnet Endpoints

## Chain ID

`paw-testnet-1`

## Network Status

- **Status**: Active
- **Last Updated**: 2026-01-04

---

## Chain Registry

```json
{
  "$schema": "../chain.schema.json",
  "chain_name": "paw",
  "chain_id": "paw-testnet-1",
  "pretty_name": "PAW Testnet",
  "network_type": "testnet",
  "status": "live",
  "bech32_prefix": "paw",
  "daemon_name": "pawd",
  "node_home": "$HOME/.paw",
  "key_algos": ["secp256k1"],
  "slip44": 118,
  "apis": {
    "rpc": [
      { "address": "https://testnet-rpc.poaiw.org", "provider": "PAW Foundation" }
    ],
    "rest": [
      { "address": "https://testnet-api.poaiw.org", "provider": "PAW Foundation" }
    ],
    "grpc": [
      { "address": "testnet-grpc.poaiw.org:443", "provider": "PAW Foundation" }
    ]
  },
  "explorers": [
    { "kind": "PAW Explorer", "url": "https://testnet-explorer.poaiw.org", "tx_page": "https://testnet-explorer.poaiw.org/tx/${txHash}" }
  ],
  "codebase": {
    "git_repo": "https://github.com/paw-chain/paw"
  }
}
```

---

## Public Endpoints

| Service | URL |
|---------|-----|
| RPC | https://testnet-rpc.poaiw.org |
| REST API | https://testnet-api.poaiw.org |
| gRPC | testnet-grpc.poaiw.org:443 |
| WebSocket | wss://testnet-ws.poaiw.org |
| GraphQL | https://testnet-graphql.poaiw.org/graphql |

---

## Resources

| Resource | URL |
|----------|-----|
| Explorer | https://testnet-explorer.poaiw.org |
| Faucet | https://testnet-faucet.poaiw.org |
| Documentation | https://testnet-docs.poaiw.org |
| Status | https://status.poaiw.org |
| Snapshots | https://snapshots.poaiw.org |
| Artifacts | https://artifacts.poaiw.org |

---

## Monitoring

| Dashboard | URL |
|-----------|-----|
| Grafana Home | https://monitoring.poaiw.org |
| Comprehensive | https://monitoring.poaiw.org/d/paw-comprehensive |
| Node Stats | https://monitoring.poaiw.org/d/paw-testnet-node |
| Validator | https://monitoring.poaiw.org/d/dtfjRVM7z |
| Cosmos Stats | https://monitoring.poaiw.org/d/9nXvbXO7z |
| Node Exporter | https://monitoring.poaiw.org/d/rYdddlPWk |

---

## Artifacts

Download testnet configuration files from https://artifacts.poaiw.org:

| File | Description |
|------|-------------|
| [genesis.json](https://artifacts.poaiw.org/genesis.json) | Genesis file (required) |
| [peers.txt](https://artifacts.poaiw.org/peers.txt) | Persistent peer list |
| [seeds.txt](https://artifacts.poaiw.org/seeds.txt) | Seed nodes |
| [addrbook.json](https://artifacts.poaiw.org/addrbook.json) | Address book |
| [chain.json](https://artifacts.poaiw.org/chain.json) | Chain registry metadata |

---

## Get Test Tokens

1. Create a wallet:
   ```bash
   pawd keys add mykey --home ~/.paw
   ```

2. Request tokens from the faucet:
   ```bash
   curl -X POST https://testnet-faucet.poaiw.org/faucet \
     -H "Content-Type: application/json" \
     -d '{"address": "paw1..."}'
   ```

3. Check your balance:
   ```bash
   pawd query bank balances $(pawd keys show mykey -a --home ~/.paw) --home ~/.paw
   ```

---

## Quick Commands

```bash
# Check node status
curl -s https://testnet-rpc.poaiw.org/status | jq '.result.sync_info'

# Query via REST API
curl -s https://testnet-api.poaiw.org/cosmos/auth/v1beta1/params | jq '.params'

# Query DEX pools
pawd query dex pools --home ~/.paw
```
