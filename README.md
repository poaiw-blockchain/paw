# PAW Blockchain

Cosmos SDK–based Layer‑1 for verifiable AI compute with integrated DEX and Oracle.

What’s included
- DEX: AMM pools, swaps, liquidity accounting, IBC transfer support
- Compute: job escrow, assignment, verification hooks, IBC port for cross‑chain requests
- Oracle: price reporting, aggregation, slashing hooks, IBC queries

Quick start
- Prerequisites: Go 1.23+, buf CLI, Docker (optional)
- Build: `make build` (produces `./build/pawd`)
- Verify binary: `./build/pawd version`
- Strict single-node boot (canonical genesis, conflict-free ports):
  - `./build/pawd init validator --chain-id paw-test-1 --home ./localnode`
  - `./build/pawd add-genesis-account <addr> 1000000000upaw --home ./localnode --keyring-backend test`
  - `./build/pawd gentx validator 700000000upaw --chain-id paw-test-1 --home ./localnode --keyring-backend test`
  - `./build/pawd collect-gentxs --home ./localnode`
  - `./build/pawd start --home ./localnode --minimum-gas-prices 0.001upaw --grpc.address 127.0.0.1:19090 --api.address tcp://127.0.0.1:1318 --rpc.laddr tcp://127.0.0.1:26658`
- Docker localnet: `docker compose up -d`

Build from source
1. Install Go 1.23+, ensure `$GOBIN` is on your PATH.
2. Install buf and protoc plugins (`make proto-tools` or follow `scripts/protocgen.sh`).
3. Generate protobufs when `.proto` changes land: `make proto-gen`.
4. Compile the daemon: `make build` or `go build -o pawd ./cmd/...`.
5. Run unit tests: `make test-unit` (security suites for compute/oracle remain skipped until validator/staking wiring is finalized).

Node initialization
- Default home: `~/.paw` (override via `PAW_HOME` or `--home`).
- Bootstrap a fresh validator on canonical genesis (no lenient JSON accepted):
  1. `./build/pawd init <moniker> --chain-id paw-test-1 --home <home>`
  2. `./build/pawd add-genesis-account <addr> 1000000000upaw --home <home> --keyring-backend test`
  3. `./build/pawd gentx <moniker> 700000000upaw --chain-id paw-test-1 --home <home> --keyring-backend test`
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
- CLI reference: `docs/guides/CLI_QUICK_REFERENCE.md`

References
- Cosmos SDK: https://docs.cosmos.network
- CometBFT: https://cometbft.com/
- IBC: https://ibcprotocol.org/

Status
- Latest update: Nov 2025
- Chain status: Beta testnet
