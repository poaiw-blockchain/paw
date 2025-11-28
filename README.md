# PAW Blockchain

Cosmos SDK–based Layer‑1 for verifiable AI compute with integrated DEX and Oracle.

What’s included
- DEX: AMM pools, swaps, liquidity accounting, IBC transfer support
- Compute: job escrow, assignment, verification hooks, IBC port for cross‑chain requests
- Oracle: price reporting, aggregation, slashing hooks, IBC queries

Quick start
- Prerequisites: Go 1.25+, Docker (optional)
- Build: `make build`
- Local node: `./scripts/bootstrap-node.sh && ./build/pawd start`
- Docker localnet: `docker compose up -d`

Repository structure
- `cmd/` CLI and daemon entrypoints (`pawd`, `pawcli`)
- `app/` application wiring, keepers, ante, params, genesis
- `x/` modules: `compute/`, `dex/`, `oracle/` (+ `privacy/` staging)
- `proto/` protobuf definitions and `buf` configs
- `docs/` technical docs, whitepaper, guides (lean and current)
- `scripts/` developer and operator scripts
- `compose/`, `docker/`, `infra/`, `k8s/` operational tooling
- `ibc/` relayer configuration and security guidance

Configuration
- Node home: `~/.paw` with Bech32 prefix `paw`
- Min gas price: `0.001upaw` (default app.toml override)
- Key files: `config/app.toml`, `config/config.toml`
- Env: `PAW_NETWORK`, `PAW_CHAIN_ID`, `PAW_RPC_PORT`, `PAW_REST_PORT`

Modules
- `x/dex`: pools, swaps, fees, ICS‑20 integration
- `x/compute`: requests, escrow, assignment, verification, IBC port
- `x/oracle`: price aggregation, voting, slashing integrations
- `app/`: BaseApp wiring, module manager, ante handler, upgrades

Testing
- All tests: `make test`
- Module: `go test ./x/<module>/...`

Documentation
- Whitepaper: `docs/WHITEPAPER.md`
- Technical spec: `docs/TECHNICAL_SPECIFICATION.md`
- CLI reference: `docs/guides/CLI_QUICK_REFERENCE.md`

References
- Cosmos SDK: https://docs.cosmos.network
- CometBFT: https://cometbft.com/
- IBC: https://ibcprotocol.org/

Status
- Latest update: Nov 2025
- Chain status: Beta testnet
