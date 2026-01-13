# paw-testnet-1 External Validator Announcement (Template)

Fill placeholders before publishing.

## Network Information
- **Network:** paw-testnet-1
- **Chain ID:** paw-testnet-1
- **Genesis SHA256:** `0ad9a1be3badff543e777501c74d577249cfc0c13a0759e5b90c544a8688d106`

## Public Endpoints
- **RPC:** `https://testnet-rpc.poaiw.org`
- **REST:** `https://testnet-api.poaiw.org`
- **gRPC:** `testnet-grpc.poaiw.org:443`
- **WebSocket:** `wss://testnet-ws.poaiw.org`
- **Explorer:** `https://testnet-explorer.poaiw.org`
- **Faucet:** `https://testnet-faucet.poaiw.org`
- **Status:** `https://status.poaiw.org`

## Artifacts
- Genesis: `https://testnet-explorer.poaiw.org/genesis.json`
- Peers: `https://testnet-explorer.poaiw.org/peers.txt`
- Chain Registry: `https://testnet-explorer.poaiw.org/chain-registry/chain.json`

## Validator Steps

1. Initialize node:
   ```bash
   pawd init <moniker> --chain-id paw-testnet-1 --home ~/.paw
   ```

2. Download and verify genesis:
   ```bash
   curl -L -o ~/.paw/config/genesis.json https://testnet-explorer.poaiw.org/genesis.json
   sha256sum ~/.paw/config/genesis.json  # must match SHA above
   ```

3. Configure peers in `~/.paw/config/config.toml`:
   ```toml
   persistent_peers = "72c594a424bfc156381860feaca3a2586173eead@54.39.103.49:11656,1780e068618ca0ffcba81574d62ab170c2ee3c8b@54.39.103.49:11756,a2b9ab78b0be7f006466131b44ede9a02fc140c4@139.99.149.160:11856,f8187d5bafe58b78b00d73b0563b65ad8c0d5fda@139.99.149.160:11956"
   ```

4. Start node:
   ```bash
   pawd start --home ~/.paw
   ```

5. Request faucet funds at https://testnet-faucet.poaiw.org

6. Submit create-validator transaction once synced.

Post and pin this message in the validator channel.
