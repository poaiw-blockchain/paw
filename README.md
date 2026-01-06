# PAW Blockchain

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8.svg)](https://golang.org/dl/)

Cosmos SDK–based Layer‑1 for verifiable AI compute with integrated DEX and Oracle.

Release artifacts policy: see `RELEASE_ARTIFACTS.md`.

What’s included
- DEX: AMM pools, swaps, liquidity accounting, IBC transfer support
- Compute: job escrow, assignment, verification hooks, IBC port for cross‑chain requests
- Oracle: price reporting, aggregation, slashing hooks, IBC queries

Quick start
- Prerequisites: Go 1.24+, buf CLI, Docker (optional)
- Build: `make build` (produces `./build/pawd`)
- Verify binary: `./build/pawd version`
- Strict single-node boot (canonical genesis, conflict-free ports):
  - `./build/pawd init validator --chain-id paw-testnet-1 --home ./localnode`
  - `./build/pawd add-genesis-account <addr> 1000000000upaw --home ./localnode --keyring-backend test`
  - `./build/pawd gentx validator 700000000upaw --chain-id paw-testnet-1 --home ./localnode --keyring-backend test`
  - `./build/pawd collect-gentxs --home ./localnode`
  - `./build/pawd start --home ./localnode --minimum-gas-prices 0.001upaw --grpc.address 127.0.0.1:19090 --api.address tcp://127.0.0.1:1318 --rpc.laddr tcp://127.0.0.1:26658`
- Docker localnet: `docker compose up -d`

Multi-validator testnet
- For testing consensus with 2, 3, or 4 validators, see **[docs/MULTI_VALIDATOR_TESTNET.md](docs/MULTI_VALIDATOR_TESTNET.md)** (complete guide)
- For production-like testing with sentry nodes, see **[docs/SENTRY_ARCHITECTURE.md](docs/SENTRY_ARCHITECTURE.md)** (sentry guide)
- Quick reference: **[docs/TESTNET_QUICK_REFERENCE.md](docs/TESTNET_QUICK_REFERENCE.md)** (one-page cheat sheet)
- Quick start (4 validators):
  ```bash
  docker compose -f compose/docker-compose.4nodes.yml down -v
  rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic
  ./scripts/devnet/setup-validators.sh 4
  docker compose -f compose/docker-compose.4nodes.yml up -d
  ```
- Quick start (4 validators + 2 sentries):
  ```bash
  docker compose -f compose/docker-compose.4nodes-with-sentries.yml down -v
  rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic
  ./scripts/devnet/setup-validators.sh 4
  docker compose -f compose/docker-compose.4nodes-with-sentries.yml up -d
  ```

Build from source
1. Install Go 1.24+, ensure `$GOBIN` is on your PATH.
2. Install buf and protoc plugins (`make proto-tools` or follow `scripts/protocgen.sh`).
3. Generate protobufs when `.proto` changes land: `make proto-gen`.
4. Compile the daemon: `make build` or `go build -o pawd ./cmd/...`.
5. Run tests: `make test` for the full suite or `make test-unit` for faster module checks.

Node initialization
- Default home: `~/.paw` (override via `PAW_HOME` or `--home`).
- Bootstrap a fresh validator on canonical genesis (no lenient JSON accepted):
  1. `./build/pawd init <moniker> --chain-id paw-testnet-1 --home <home>`
  2. `./build/pawd add-genesis-account <addr> 1000000000upaw --home <home> --keyring-backend test`
  3. `./build/pawd gentx <moniker> 700000000upaw --chain-id paw-testnet-1 --home <home> --keyring-backend test`
  4. `./build/pawd collect-gentxs --home <home>`
  5. Start with strict ports and min-gas-price:  
     `./build/pawd start --home <home> --minimum-gas-prices 0.001upaw --grpc.address 127.0.0.1:19090 --api.address tcp://127.0.0.1:1318 --rpc.laddr tcp://127.0.0.1:26658`
- Canonical genesis rules: integers serialized as strings, non-null `app_hash`, bond denom `upaw`. Invalid genesis must be fixed offline; runtime will not auto-heal.
- To run the localnet scripts the binary must exist at `./build/pawd`.

Repository structure
- `cmd/` CLI and daemon entrypoints (`pawd`, `pawcli`)
- `app/` application wiring, keepers, ante, params, genesis
- `x/` modules: `compute/`, `dex/`, `oracle/` (+ `privacy/` staging)
- `proto/` protobuf definitions and `buf` configs
- `docs/` technical docs, whitepaper, guides (lean and current)
- `scripts/` developer and operator scripts
- `compose/`, `docker/`, `infra/`, `k8s/` operational tooling
- `ibc/` relayer configuration and security guidance
- `wallet/` production wallet suite (core SDK, desktop, mobile, browser extension)

Wallet suite
- `wallet/core` — TypeScript SDK powering every wallet surface (HD keys, signing helpers, RPC client). Build with `npm install && npm run build`.
- `wallet/desktop` — Electron + React desktop wallet with integrated DEX tooling. Build with `npm install && npm run build` (per-platform bundles via `electron-builder`).
- `wallet/mobile` — React Native wallet for iOS and Android. Install deps with `npm install` and bundle per-platform via `npm run android` / `npm run ios` (add a `build` script for CI artifacts if desired).
- `wallet/browser-extension` — Chromium/Firefox extension for swaps + miner controls. Build with `npm install && npm run build`.
- Shared multi-chain wallet (cross-chain compatibility only): `/home/hudson/blockchain-projects/shared/wallet/multi-chain-wallet`

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
- Protobufs: `make proto-gen` after editing `.proto` files
- `make test-unit` currently skips compute/oracle security integration suites pending validator/app wiring for Cosmos SDK v0.50; functional unit suites pass.

Documentation
- Whitepaper: `docs/WHITEPAPER.md`
- Technical spec: `docs/TECHNICAL_SPECIFICATION.md`
- Testnet quick reference: `docs/TESTNET_QUICK_REFERENCE.md`
- Validator quickstart: `docs/VALIDATOR_QUICK_START.md`
- CLI reference: `docs/guides/CLI_QUICK_REFERENCE.md`
- DEX trading: `docs/guides/DEX_TRADING.md`

References
- Cosmos SDK: https://docs.cosmos.network
- CometBFT: https://cometbft.com/
- IBC: https://ibcprotocol.org/

Devnet

To join the PAW devnet, see the [networks repository](https://github.com/poaiw-blockchain/testnets) for genesis files, peer lists, and network details.

| Network | Chain ID | Status |
|---------|----------|--------|
| [paw-testnet-1](https://github.com/poaiw-blockchain/testnets/tree/main/paw-testnet-1) | `paw-testnet-1` | Devnet |

**Quick join:**
1. Build: `make build`
2. Initialize: `./build/pawd init <moniker> --chain-id paw-testnet-1`
3. Download genesis: `curl -o ~/.paw/config/genesis.json https://raw.githubusercontent.com/poaiw-blockchain/testnets/main/paw-testnet-1/genesis.json`
4. Set peers in `~/.paw/config/config.toml` from [peers.txt](https://github.com/poaiw-blockchain/testnets/blob/main/paw-testnet-1/peers.txt)
5. Start: `./build/pawd start --minimum-gas-prices 0.001upaw`
