# Genesis State Initialization and Node Bootstrapping Implementation

## Summary

This implementation provides a complete genesis state initialization and node bootstrapping system for the PAW blockchain, including scripts, Go application code, and CLI commands.

## Files Created

### 1. Scripts

#### `scripts/init-genesis.sh`
Complete genesis state initialization script that:
- Initializes chain data directory using `pawd init`
- Generates validator keys for 2 genesis validators
- Creates genesis accounts with 50M PAW total supply allocation:
  - 10M PAW each for 2 validators (20M total)
  - 15M PAW for treasury (reserve)
  - 15M PAW for foundation (ecosystem development)
- Configures chain parameters from `infra/node-config.yaml`:
  - Staking: min stake 10,000 PAW, 21-day unbonding, 125 max validators
  - Consensus: 4s block time, 2MB max block size, 100M gas limit
  - Slashing: 5% double-sign penalty, 0.1% downtime penalty
  - Governance: 10,000 PAW min deposit, 14-day voting period, 40% quorum
  - Fees: 0.001 upaw min gas price, 50% burn, 30% validators, 20% treasury
- Generates genesis transactions (gentx) for validators
- Collects and validates all genesis transactions
- Validates final genesis file

**Usage:**
```bash
./scripts/init-genesis.sh
```

#### `scripts/bootstrap-node.sh` (Enhanced)
Enhanced bootstrap script that:
- Calls `pawd init` to initialize node configuration
- Generates validator keys
- Parses `infra/node-config.yaml` for:
  - Chain ID: paw-testnet
  - 2 genesis validators with commission rates
  - Emission schedule: 2870 PAW/day year 1
  - Consensus parameters (4s blocks, BFT timeouts)
  - Total supply: 50M PAW
  - Reserve: 15M PAW
- Creates placeholder genesis for development
- Generates node environment file with all parameters
- Creates validator configuration JSON
- Outputs comprehensive README with next steps

**Usage:**
```bash
./scripts/bootstrap-node.sh
```

### 2. Application Code

#### `app/genesis.go`
Complete genesis state management:
- **GenesisState**: Map of module genesis states
- **NewDefaultGenesisState()**: Creates default genesis with all module configs:
  - Auth: Account authentication
  - Bank: 50M PAW total supply
  - Staking: 125 max validators, 21-day unbonding
  - Slashing: Double-sign and downtime penalties
  - Governance: On-chain governance with 40% quorum
  - Distribution: 20% community tax to treasury
  - Mint: Disabled (fixed supply, no inflation)
  - Crisis: Invariant checking
  - Wasm: CosmWasm smart contracts with open access
- **NewGenesisStateFromConfig()**: Customizable genesis from config
- **GenesisConfig**: Configuration structure matching node-config.yaml
- **DefaultGenesisConfig()**: Returns parameters from technical spec

**Key Features:**
- All parameters align with TECHNICAL_SPECIFICATION.md
- Supports customization via GenesisConfig
- Proper module state initialization
- Validation and error handling

### 3. CLI Commands

#### `cmd/pawd/cmd/init.go`
Initialize node command:
- Creates validator and node configuration files
- Generates genesis.json with default state
- Sets up Tendermint configuration:
  - 4-second block time
  - BFT consensus timeouts (3s propose, 1s prevote/precommit)
  - 2MB max block size, 100M gas limit
  - P2P settings (40 inbound, 10 outbound peers)
  - Mempool settings (10,000 txs, 10MB cache)
  - State sync enabled
- Creates app.toml with:
  - Min gas price: 0.001upaw
  - API/gRPC/gRPC-Web enabled
  - Swagger documentation enabled

**Usage:**
```bash
pawd init [moniker] --chain-id paw-testnet
```

#### `cmd/pawd/cmd/gentx.go`
Generate validator genesis transaction:
- Creates MsgCreateValidator for genesis validators
- Supports commission rate configuration
- Sets validator metadata (moniker, identity, website)
- Validates and signs transaction
- Writes gentx to config/gentx/ directory

**Usage:**
```bash
pawd gentx validator-1 10000000000upaw \
  --chain-id paw-testnet \
  --moniker "PAW Validator 1" \
  --commission-rate 0.10 \
  --commission-max-rate 0.20 \
  --commission-max-change-rate 0.01 \
  --keyring-backend test
```

#### `cmd/pawd/cmd/collect_gentxs.go`
Collect genesis transactions:
- Reads all gentx files from config/gentx/
- Validates each gentx (must be MsgCreateValidator)
- Updates genesis state with validators
- Adds delegations for self-bonded validators
- Updates staking module genesis state
- Validates final genesis file
- Saves updated genesis.json

**Usage:**
```bash
pawd collect-gentxs
```

#### `cmd/pawd/cmd/add_genesis_account.go`
Add genesis account:
- Adds accounts to genesis.json before chain start
- Supports vesting accounts
- Updates bank module balances and total supply
- Validates account doesn't already exist

**Usage:**
```bash
pawd add-genesis-account paw1... 100000000upaw
pawd add-genesis-account validator-1 10000000000upaw --keyring-backend test
```

## Configuration Source

All parameters are derived from:
- **`infra/node-config.yaml`**: Network configuration
  - Chain ID, validators, tokenomics
  - Consensus, slashing, staking parameters
  - Fees, governance, oracle settings
- **`docs/TECHNICAL_SPECIFICATION.md`**: Technical details
  - Block structure, gas costs
  - Cryptography (Ed25519)
  - CosmWasm configuration
  - Network protocol settings

## Genesis Parameters

### Tokenomics
- **Total Supply**: 50,000,000 PAW (50M)
- **Reserve**: 15,000,000 PAW (treasury)
- **Genesis Distribution**:
  - 2 validators: 10M PAW each (for staking)
  - Treasury: 15M PAW
  - Foundation: 15M PAW

### Validators
- **Genesis Validators**: 2 (expandable to 25)
- **Max Validators**: 125
- **Min Stake**: 10,000 PAW
- **Unbonding Period**: 21 days (1,814,400 seconds)
- **Commission Rates**:
  - Validator 1: 10% (max 15%, max change 1%/day)
  - Validator 2: 8% (max 15%, max change 2%/day)

### Consensus (Tendermint BFT)
- **Block Time**: 4 seconds
- **Max Block Size**: 2 MB (2,097,152 bytes)
- **Max Gas/Block**: 100,000,000 units
- **Timeouts**:
  - Propose: 3s (delta 500ms)
  - Prevote: 1s (delta 500ms)
  - Precommit: 1s (delta 500ms)
  - Commit: 4s

### Slashing
- **Double Sign Penalty**: 5% of stake
- **Downtime Penalty**: 0.1% of stake
- **Downtime Window**: 10,000 blocks
- **Downtime Threshold**: 500 missed blocks
- **Jail Duration**: 24 hours (86,400 seconds)

### Governance
- **Min Deposit**: 10,000 PAW
- **Deposit Period**: 7 days (604,800 seconds)
- **Voting Period**: 14 days (1,209,600 seconds)
- **Quorum**: 40%
- **Threshold**: 66.7%
- **Veto Threshold**: 33.3%

### Fees
- **Min Gas Price**: 0.001 upaw/gas
- **Fee Distribution**:
  - Burn: 50%
  - Validators: 30%
  - Treasury: 20%

### CosmWasm
- **Code Upload**: Open to everyone
- **Instantiate**: Open to everyone
- **Max Contract Size**: 800 KB

## Workflow

### Initial Setup
1. Run `scripts/bootstrap-node.sh`:
   - Initializes node configuration
   - Generates validator keys
   - Creates placeholder genesis
   - Sets up environment

2. Run `scripts/init-genesis.sh`:
   - Full genesis state initialization
   - Creates all genesis accounts
   - Generates validator gentxs
   - Produces final genesis.json

3. Start node:
   ```bash
   pawd start --home ~/.paw
   ```

### Manual Setup (Alternative)
1. Initialize node:
   ```bash
   pawd init paw-controller --chain-id paw-testnet
   ```

2. Create validator keys:
   ```bash
   pawd keys add validator-1 --keyring-backend test
   pawd keys add validator-2 --keyring-backend test
   ```

3. Add genesis accounts:
   ```bash
   pawd add-genesis-account validator-1 10000000000000upaw --keyring-backend test
   pawd add-genesis-account validator-2 10000000000000upaw --keyring-backend test
   ```

4. Generate gentxs:
   ```bash
   pawd gentx validator-1 10000000000upaw --chain-id paw-testnet --moniker "Validator 1"
   pawd gentx validator-2 10000000000upaw --chain-id paw-testnet --moniker "Validator 2"
   ```

5. Collect gentxs:
   ```bash
   pawd collect-gentxs
   ```

6. Validate genesis:
   ```bash
   pawd validate-genesis
   ```

7. Start node:
   ```bash
   pawd start
   ```

## Directory Structure

```
~/.paw/                          # Node home directory
├── config/
│   ├── genesis.json            # Genesis state
│   ├── config.toml             # Tendermint config
│   ├── app.toml                # Application config
│   ├── node_key.json           # Node P2P key
│   ├── priv_validator_key.json # Validator consensus key
│   └── gentx/                  # Genesis transactions
│       ├── gentx-<node-id-1>.json
│       └── gentx-<node-id-2>.json
├── data/                        # Blockchain data
│   └── priv_validator_state.json
└── keyring-test/                # Keyring (test backend)
    └── <validator-keys>

infra/node/                      # Development node config
├── genesis.json                 # Genesis template
├── node.env                     # Environment variables
├── validators.json              # Validator configuration
├── README.md                    # Setup instructions
├── config/                      # Config backups
│   ├── config.toml
│   └── app.toml
└── keyring/                     # Key backups
    ├── validator-1.key
    └── validator-2.key
```

## Testing

To verify the implementation:

1. **Bootstrap test**:
   ```bash
   ./scripts/bootstrap-node.sh
   cat infra/node/genesis.json
   cat infra/node/node.env
   ```

2. **Genesis initialization test**:
   ```bash
   ./scripts/init-genesis.sh
   cat ~/.paw/config/genesis.json | jq '.app_state.bank.supply'
   cat ~/.paw/config/genesis.json | jq '.app_state.staking.params'
   ```

3. **Validator test**:
   ```bash
   pawd keys list --keyring-backend test
   pawd validate-genesis
   ```

## Next Steps

1. Build the pawd binary:
   ```bash
   make install
   ```

2. Run bootstrap:
   ```bash
   ./scripts/bootstrap-node.sh
   ```

3. Initialize genesis:
   ```bash
   ./scripts/init-genesis.sh
   ```

4. Start the node:
   ```bash
   pawd start
   ```

## Notes

- All scripts are executable (chmod +x)
- Scripts gracefully handle missing pawd binary for development
- Configuration aligns with TECHNICAL_SPECIFICATION.md
- Genesis state supports all PAW modules (DEX, compute, oracle, etc.)
- Keys are stored with test keyring backend for development
- Production deployment should use secure keyring backend (os, file, etc.)

## Security Considerations

- **Development keys**: Test keyring backend is NOT secure for production
- **Production setup**: Use `--keyring-backend os` or `--keyring-backend file`
- **Key backup**: Store validator keys securely offline
- **Genesis accounts**: Verify all addresses before mainnet launch
- **Parameter review**: Review all genesis parameters with community
- **Audit**: Conduct security audit before mainnet

## References

- **Configuration**: `infra/node-config.yaml`
- **Technical Spec**: `docs/TECHNICAL_SPECIFICATION.md`
- **Cosmos SDK**: https://docs.cosmos.network/
- **Tendermint**: https://docs.tendermint.com/
- **CosmWasm**: https://docs.cosmwasm.com/
