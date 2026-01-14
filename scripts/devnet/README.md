# PAW Devnet Scripts

Scripts for generating multi-validator genesis and initializing validator nodes.

## Files

### `setup-validators.sh`

Generates a multi-validator genesis file with 2, 3, or 4 validators.

**Usage:**
```bash
./scripts/devnet/setup-validators.sh <NUM_VALIDATORS>
```

**Parameters:**
- `NUM_VALIDATORS` - Number of validators (2, 3, or 4)

**What it does:**
1. Creates a base genesis with chain configuration
2. Generates validator keys for each node (node1, node2, node3, node4)
3. Creates genesis transactions (gentx) for each validator
4. Collects all gentxs into the final genesis
5. **Adds ValidatorSigningInfo for each validator** (fixes SDK v0.50.x signing info bug)
6. Validates the final genesis structure
7. Saves everything to `scripts/devnet/.state/`

**Output files:**
- `scripts/devnet/.state/genesis.json` - Final genesis file
- `scripts/devnet/.state/node1.priv_validator_key.json` - Validator 1 consensus key
- `scripts/devnet/.state/node2.priv_validator_key.json` - Validator 2 consensus key
- `scripts/devnet/.state/node3.priv_validator_key.json` - Validator 3 consensus key
- `scripts/devnet/.state/node4.priv_validator_key.json` - Validator 4 consensus key
- `scripts/devnet/.state/node*_validator.mnemonic` - Validator mnemonics

**Example:**
```bash
# Generate 4-validator genesis
./scripts/devnet/setup-validators.sh 4
```

### `setup-multivalidators.sh`

Generates the canonical 4-validator genesis used by the docker-compose stacks and Phase D rehearsal flow. Mirrors `setup-validators.sh` but always targets four validators and now creates a dedicated faucet account backed up inside `scripts/devnet/.state/`.

**Usage:**
```bash
CHAIN_ID=paw-mvp-1 ./scripts/devnet/setup-multivalidators.sh
```

**Highlights:**
- Reads `PAWD_BIN` (defaults to `<repo>/pawd`) so it works from any directory
- Captures validator, faucet, and smoke-test mnemonics inside `.state/`
- Accepts `CHAIN_ID` override so the same script can generate devnet or public-testnet genesis files
- Seeds a faucet account with 5,000,000 PAW-equivalent to power the faucet service

### `init_node.sh`

Initializes a validator node for the multi-validator testnet.

**Usage:**
```bash
./scripts/devnet/init_node.sh <NODE_NAME> <RPC_PORT> <GRPC_PORT> <API_PORT>
```

**Parameters:**
- `NODE_NAME` - Node identifier (node1, node2, node3, node4)
- `RPC_PORT` - CometBFT RPC port (26657, 26667, 26677, 26687)
- `GRPC_PORT` - gRPC port (9090, 9091, 9092, 9093)
- `API_PORT` - REST API port (1317, 1327, 1337, 1347)

**What it does:**
1. Checks if multi-validator genesis exists in `.state/genesis.json`
2. For node1: Uses pre-generated genesis and starts immediately
3. For other nodes: Waits for node1's node ID, configures persistent peers
4. Configures consensus timeouts (10s propose, 5s prevote/precommit)
5. Copies the appropriate priv_validator_key.json for this node
6. Starts the validator node with proper P2P mesh

**Important configurations:**
- **Consensus timeouts increased** to prevent "ProposalBlock is nil" errors
  - `timeout_propose = "10s"` (default 3s was too short)
  - `timeout_prevote = "5s"`
  - `timeout_precommit = "5s"`
- **P2P mesh**: All nodes connect to each other via persistent_peers
- **Unique validator keys**: Each node uses its own priv_validator_key.json

**Example:**
```bash
# Initialize node1 (called by Docker)
./scripts/devnet/init_node.sh node1 26657 9090 1317

# Initialize node2 (called by Docker)
./scripts/devnet/init_node.sh node2 26667 9091 1327
```

### `local-phase-d-rehearsal.sh`

End-to-end automation for the Production Roadmap Phase D "local rehearsal" requirement. It builds `pawd`, regenerates the multi-validator genesis, boots the `docker-compose.4nodes-with-sentries.yml` stack, runs smoke tests, verifies metrics on all validators + sentries, and publishes the resulting genesis/peers/checksum into `networks/<chain-id>/`.

**Usage:**
```bash
CHAIN_ID=paw-mvp-1 ./scripts/devnet/local-phase-d-rehearsal.sh
```

**What you get:**
- 4 validators + 2 sentries with Prometheus metrics enabled on every node
- Deterministic Phase D smoke test (bank + DEX + gov + oracle + compute)
- Artifacts synced into `networks/paw-mvp-1/` via the existing publish pipeline
- Optional knobs (e.g., `PAW_REHEARSAL_KEEP_STACK=1`, `REBUILD_GENESIS=0`) for iterative debugging

See `docs/guides/deployment/PUBLIC_TESTNET.md` for how this script ties into the public testnet onboarding workflow.

## Workflow

### 1. Generate Genesis (Before Docker)

```bash
# Clean old state
rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic

# Generate 4-validator genesis
./scripts/devnet/setup-validators.sh 4
```

### 2. Start Validators (Docker Compose)

```bash
# Docker Compose calls init_node.sh for each container
docker compose -f compose/docker-compose.4nodes.yml up -d
```

**Docker Compose does:**
- Mounts `scripts/devnet/.state/` into each container
- Calls `init_node.sh` with appropriate ports for each node
- node1 reads genesis and starts
- node2-4 wait for node1, configure peers, then start

### 3. Consensus Achieved

After ~30 seconds:
- All 4 nodes have connected to each other
- Genesis block committed
- Continuous block production begins

## State Directory

The `.state/` directory contains:

```
scripts/devnet/.state/
├── genesis.json                       # Final multi-validator genesis
├── node1.priv_validator_key.json      # Node 1 consensus key
├── node2.priv_validator_key.json      # Node 2 consensus key
├── node3.priv_validator_key.json      # Node 3 consensus key
├── node4.priv_validator_key.json      # Node 4 consensus key
├── node1_validator.mnemonic           # Node 1 mnemonic (24 words)
├── node2_validator.mnemonic           # Node 2 mnemonic
├── node3_validator.mnemonic           # Node 3 mnemonic
└── node4_validator.mnemonic           # Node 4 mnemonic
```

**⚠️ SECURITY:** These files contain private keys. Never commit to version control.

## Technical Details

### SDK v0.50.x Signing Info Bug

In Cosmos SDK v0.50.x, the `AfterValidatorBonded` hook (which creates `ValidatorSigningInfo`) is not called for validators that are already bonded during genesis `InitChain`. This causes the "no validator signing info found" error when the slashing module's `BeginBlocker` runs.

**Our fix:** `setup-validators.sh` manually populates the `signing_infos` array in the genesis file after `collect-gentxs`, converting validator consensus addresses from hex to bech32 format.

### Consensus Timeout Fix

Default CometBFT timeouts (3s for proposal) were too short for the chain's `PrepareProposal` execution, causing validators to vote NIL instead of voting for blocks.

**Our fix:** `init_node.sh` increases all consensus timeouts to allow sufficient time for block preparation and propagation.

## Complete Documentation

For full usage instructions, troubleshooting, and examples, see:
- **[docs/MULTI_VALIDATOR_TESTNET.md](../../docs/MULTI_VALIDATOR_TESTNET.md)** - Complete guide
- **[docs/TESTNET_QUICK_REFERENCE.md](../../docs/TESTNET_QUICK_REFERENCE.md)** - Quick reference

## Common Issues

**"genesis hash mismatch"**
- Cause: Old state not cleaned before regeneration
- Fix: `rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic`

**"no validator signing info found"**
- Cause: Genesis missing signing_infos (script bug)
- Fix: Ensure you're using the updated `setup-validators.sh`

**Stuck at height 0**
- Cause: Consensus timeouts too short or missing validators
- Fix: Verify all 4 containers are running and wait 30+ seconds
