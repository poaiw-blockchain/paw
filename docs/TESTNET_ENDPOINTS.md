# PAW Public Testnet Endpoints

## Chain ID

- `paw-testnet-1`

## Live Endpoints

| Service | URL |
|---------|-----|
| **RPC** | https://testnet-rpc.poaiw.org |
| **REST API** | https://testnet-api.poaiw.org |
| **gRPC** | testnet-rpc.poaiw.org:9090 |
| **Faucet** | https://testnet-faucet.poaiw.org |
| **Explorer** | https://testnet-explorer.poaiw.org |
| **Monitoring** | https://monitoring.poaiw.org |

### Direct Server Access (Development)

| Service | Address |
|---------|---------|
| Server IP | 54.39.103.49 |
| VPN IP | 10.10.0.2 |
| RPC | http://54.39.103.49:26657 |
| REST API | http://54.39.103.49:1317 |
| gRPC | 54.39.103.49:9090 |
| P2P | 54.39.103.49:26656 |

## Get Test Tokens

1. Create a wallet:
   ```bash
   pawd keys add mykey --home ~/.paw
   ```

2. Request tokens from the faucet:
   - Visit https://testnet-faucet.poaiw.org
   - Or use the API:
     ```bash
     curl -X POST https://testnet-faucet.poaiw.org/claim \
       -H "Content-Type: application/json" \
       -d '{"address": "paw1..."}'
     ```

3. Check your balance:
   ```bash
   pawd query bank balances $(pawd keys show mykey -a --home ~/.paw) --home ~/.paw
   ```

## Quick Commands

```bash
# Check node status
pawd status --home ~/.paw

# Query latest block via RPC
curl -s https://testnet-rpc.poaiw.org/status | jq '.result.sync_info'

# Query via REST API
curl -s https://testnet-api.poaiw.org/cosmos/base/tendermint/v1beta1/blocks/latest | jq '.block.header.height'

# Query DEX pools
pawd query dex pools --home ~/.paw

# Query compute parameters
pawd query compute params --home ~/.paw
```

## Chain Registry Format

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
      { "address": "testnet-rpc.poaiw.org:9090", "provider": "PAW Foundation" }
    ]
  },
  "explorers": [
    { "url": "https://testnet-explorer.poaiw.org", "kind": "PAW Explorer" }
  ]
}
```

## Status

- **Network**: Active
- **Last Updated**: 2026-01-01
