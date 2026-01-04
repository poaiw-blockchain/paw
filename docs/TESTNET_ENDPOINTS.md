# PAW Public Testnet Endpoints

## Chain ID

- `paw-testnet-1`

---

## Public Endpoints (Updated 2026-01-03)

| Service | URL | Status |
|---------|-----|--------|
| **RPC** | https://testnet-rpc.poaiw.org | OK |
| **REST API** | https://testnet-api.poaiw.org | OK |
| **gRPC** | https://testnet-grpc.poaiw.org | OK |
| **WebSocket** | wss://testnet-ws.poaiw.org | OK |
| **GraphQL** | https://testnet-graphql.poaiw.org/graphql | OK |
| **Explorer** | https://testnet-explorer.poaiw.org | OK |
| **Faucet** | https://testnet-faucet.poaiw.org | OK |
| **Archive RPC** | https://testnet-archive.poaiw.org | OK |
| **Docs** | https://testnet-docs.poaiw.org | OK |
| **Monitoring** | https://monitoring.poaiw.org | OK |
| **Status** | https://status.poaiw.org | OK |
| **Snapshots** | https://snapshots.poaiw.org | OK |
| **Artifacts** | https://artifacts.poaiw.org | OK |

---

## Public Artifacts

Download testnet configuration files from https://artifacts.poaiw.org:

| File | URL | Description |
|------|-----|-------------|
| genesis.json | [Download](https://artifacts.poaiw.org/genesis.json) | Genesis file (required) |
| peers.txt | [Download](https://artifacts.poaiw.org/peers.txt) | Persistent peer list |
| seeds.txt | [Download](https://artifacts.poaiw.org/seeds.txt) | Seed nodes |
| addrbook.json | [Download](https://artifacts.poaiw.org/addrbook.json) | Address book |
| chain.json | [Download](https://artifacts.poaiw.org/chain.json) | Chain registry metadata |
| state_sync.md | [View](https://artifacts.poaiw.org/state_sync.md) | State sync guide |

---

### Direct Server Access (Operators)

| Service | Address |
|---------|---------|
| Server IP | 54.39.103.49 |
| VPN IP | 10.10.0.2 |
| RPC | http://127.0.0.1:26657 |
| REST API | http://127.0.0.1:1317 (not responding) |
| gRPC | 127.0.0.1:9091 |
| P2P | 0.0.0.0:26656 |
| Archive RPC | 127.0.0.1:26687 |

---

## Get Test Tokens

1. Create a wallet:
   ```bash
   pawd keys add mykey --home ~/.paw
   ```

2. Request tokens from the faucet:
   - Visit https://testnet-faucet.poaiw.org
   - Or use the API:
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

# Query via REST API (currently degraded)
curl -s https://testnet-api.poaiw.org/cosmos/auth/v1beta1/params | jq '.params'

# Query DEX pools
pawd query dex pools --home ~/.paw
```

---

## Chain Registry (Current Template)

```json
{
  "chain_name": "paw",
  "chain_id": "paw-testnet-1",
  "pretty_name": "PAW Testnet",
  "network_type": "testnet",
  "status": "live",
  "bech32_prefix": "paw",
  "daemon_name": "pawd",
  "node_home": "$HOME/.paw",
  "apis": {
    "rpc": [
      { "address": "https://testnet-rpc.poaiw.org", "provider": "PAW Foundation" }
    ],
    "rest": [
      { "address": "https://testnet-api.poaiw.org", "provider": "PAW Foundation" }
    ],
    "grpc": [
      { "address": "testnet-rpc.poaiw.org:9091", "provider": "PAW Foundation" }
    ]
  },
  "explorers": [
    { "url": "https://testnet-explorer.poaiw.org", "kind": "PAW Explorer" }
  ],
  "codebase": {
    "git_repo": "https://github.com/paw-chain/paw"
  }
}
```

---

## Status

- **Network**: Active (public endpoints partially degraded)
- **Last Updated**: 2026-01-03
