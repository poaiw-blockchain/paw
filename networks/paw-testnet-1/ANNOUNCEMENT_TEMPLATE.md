# paw-testnet-1 External Validator Announcement (Template)

Fill placeholders before publishing.

- Network: paw-testnet-1
- Genesis: <https://networks.paw.xyz/paw-testnet-1/genesis.json>
- SHA256: `0ad9a1be3badff543e777501c74d577249cfc0c13a0759e5b90c544a8688d106`
- Peers: <https://networks.paw.xyz/paw-testnet-1/peers.txt> (persistent peers; seeds empty by design)
- Manifest: <https://networks.paw.xyz/paw-testnet-1/paw-testnet-1-manifest.json>
- Bundle (tar.gz): <https://networks.paw.xyz/paw-testnet-1/paw-testnet-1-artifacts.tar.gz>
- RPC: `https://rpc1.paw-testnet.io`
- REST: `https://api.paw-testnet.io`
- gRPC: `https://grpc.paw-testnet.io:443`
- Explorer: `https://explorer.paw-testnet.io`
- Faucet: `https://faucet.paw-testnet.io`
- Status: `https://status.paw-testnet.io` (RPC/REST/gRPC/Explorer/Faucet/Metrics live probes)
- Start block height: TBD (announce window)
- Tarball: <https://networks.paw-testnet.io/paw-testnet-1-artifacts.tar.gz> (sha256=78b27d1c02196531b7907773874520447e0be2bee4b95b781085c9e11b6a90de; see `paw-testnet-1-artifacts.sha256`)

Validator steps:
1) `pawd init <moniker> --chain-id paw-testnet-1 --home ~/.paw`
2) `curl -L -o ~/.paw/config/genesis.json https://networks.paw.xyz/paw-testnet-1/genesis.json`
3) `sha256sum ~/.paw/config/genesis.json` (must match SHA above)
4) Configure peers from `peers.txt` into `config.toml`
5) `pawd start --home ~/.paw`
6) Request faucet funds, then submit create-validator tx.

Post and pin this message in the validator channel; update URLs if hosting differs.***
