# PAW Project Guidelines

**Read `../CLAUDE.md` first** - contains all general instructions.

## Project-Specific

**Node:** `~/.paw/` (not in repo)
**Binary:** `go build -o pawd ./cmd/...`
**Init:** `./pawd init <moniker> --chain-id paw-testnet-1`
**Proto:** `make proto-gen` or `buf generate` after modifying `.proto` files

## Testnet Access
- **SSH**: `ssh paw-testnet` (54.39.103.49)
- **Chain**: paw-testnet-1 | VPN: 10.10.0.2
- **Binary**: `~/.paw/cosmovisor/genesis/bin/pawd --home ~/.paw`
- **Ports**: P2P=26656, RPC=26657, gRPC=9091, API=1317 (REST currently not responding)
- **Modules**: compute, dex, oracle (custom) + standard Cosmos

**Full docs**: `TESTNET_INFRASTRUCTURE.md`
