# PAW Mainnet Genesis Configuration Guide

This guide explains how to use the PAW mainnet genesis file and provides detailed documentation for each configuration section.

## Table of Contents

1. [Overview](#overview)
2. [Genesis File Location](#genesis-file-location)
3. [How to Use](#how-to-use)
4. [Configuration Sections](#configuration-sections)
5. [Token Distribution](#token-distribution)
6. [Validator Setup](#validator-setup)
7. [Module Parameters](#module-parameters)
8. [Security Considerations](#security-considerations)
9. [Troubleshooting](#troubleshooting)

## Overview

The `genesis-mainnet.json` file defines the initial state of the PAW blockchain mainnet. It includes:

- **Chain ID**: `paw-mainnet-1`
- **Genesis Time**: December 15, 2025, 00:00:00 UTC
- **Total Supply**: 50,000,000 PAW (50,000,000,000,000 upaw)
- **Initial Validators**: 25 validators (to be added via gentx)
- **Block Time**: 4 seconds (configured in consensus params)

## Genesis File Location

```
config/genesis-mainnet.json
```

This file should be copied to your node's configuration directory:

```bash
cp config/genesis-mainnet.json ~/.paw/config/genesis.json
```

## How to Use

### 1. Initialize Node

```bash
# Initialize your node with the mainnet chain ID
pawd init <moniker> --chain-id paw-mainnet-1

# Copy the genesis file
cp config/genesis-mainnet.json ~/.paw/config/genesis.json
```

### 2. Add Genesis Validators (Pre-Genesis)

If you're a genesis validator, create your gentx:

```bash
# Create validator key
pawd keys add validator

# Add genesis account with minimum stake
pawd add-genesis-account $(pawd keys show validator -a) 10000000000upaw

# Create gentx (adjust commission rates as needed)
pawd gentx validator 10000000000upaw \
  --chain-id paw-mainnet-1 \
  --commission-rate=0.10 \
  --commission-max-rate=0.20 \
  --commission-max-change-rate=0.01 \
  --min-self-delegation=10000000000

# Collect gentxs from all genesis validators
pawd collect-gentxs
```

### 3. Validate Genesis File

Before starting the network, validate the genesis file:

```bash
pawd validate-genesis
```

### 4. Start the Node

```bash
pawd start
```

## Configuration Sections

### 1. Basic Metadata

```json
{
  "genesis_time": "2025-12-15T00:00:00Z",
  "chain_id": "paw-mainnet-1",
  "initial_height": "1"
}
```

- **genesis_time**: Network launch timestamp (UTC)
- **chain_id**: Unique identifier for the mainnet
- **initial_height**: Starting block height (always 1 for new chains)

### 2. Consensus Parameters

```json
"consensus_params": {
  "block": {
    "max_bytes": "2097152",      // 2 MB max block size
    "max_gas": "100000000"       // 100M gas per block
  },
  "evidence": {
    "max_age_num_blocks": "100000",
    "max_age_duration": "172800000000000",  // 48 hours
    "max_bytes": "1048576"                  // 1 MB
  },
  "validator": {
    "pub_key_types": ["ed25519"]  // Validator key algorithm
  }
}
```

**Key Settings:**
- **Block Size**: 2 MB allows ~1000+ transactions per block
- **Block Gas**: 100M gas supports high throughput
- **Evidence Age**: 48 hours for slashing evidence submission
- **Key Type**: Ed25519 for validator signatures

### 3. Auth Module

Manages accounts and transaction authentication.

```json
"auth": {
  "params": {
    "max_memo_characters": "256",
    "tx_sig_limit": "7",
    "tx_size_cost_per_byte": "10",
    "sig_verify_cost_ed25519": "590",
    "sig_verify_cost_secp256k1": "1000"
  }
}
```

**Parameters:**
- **max_memo_characters**: Maximum memo length in transactions
- **tx_sig_limit**: Maximum signatures per transaction (multisig)
- **tx_size_cost_per_byte**: Gas cost per byte of transaction
- **sig_verify_cost_***: Gas costs for signature verification

**Genesis Accounts:**
Seven core accounts are pre-configured for token distribution (see Token Distribution section).

### 4. Bank Module

Manages token balances and transfers.

```json
"bank": {
  "params": {
    "default_send_enabled": true
  },
  "denom_metadata": [
    {
      "base": "upaw",
      "display": "paw",
      "symbol": "PAW"
    }
  ]
}
```

**Token Denomination:**
- **Base Unit**: `upaw` (micro-PAW)
- **Display Unit**: `paw`
- **Conversion**: 1 PAW = 1,000,000 upaw

**Total Supply Breakdown:**
See [Token Distribution](#token-distribution) section below.

### 5. Staking Module

Manages validator staking and delegation.

```json
"staking": {
  "params": {
    "unbonding_time": "1814400s",        // 21 days
    "max_validators": 125,                // Up to 125 active validators
    "max_entries": 7,                     // Max unbonding/redelegation entries
    "historical_entries": 10000,
    "bond_denom": "upaw",
    "min_commission_rate": "0.05"         // 5% minimum commission
  }
}
```

**Key Parameters:**
- **Unbonding Period**: 21 days for security
- **Max Validators**: 125 active validators (starts with 25 genesis validators)
- **Min Commission**: 5% prevents race-to-zero commission
- **Historical Entries**: 10,000 blocks for IBC light client verification

### 6. Distribution Module

Manages staking rewards and community pool.

```json
"distribution": {
  "params": {
    "community_tax": "0.02",              // 2% to community pool
    "withdraw_addr_enabled": true
  }
}
```

**Parameters:**
- **Community Tax**: 2% of rewards go to governance-controlled pool
- **Withdraw Address**: Delegators can set custom reward withdrawal addresses

### 7. Governance Module

Manages on-chain governance proposals.

```json
"gov": {
  "deposit_params": {
    "min_deposit": [{"denom": "upaw", "amount": "10000000000"}],  // 10,000 PAW
    "max_deposit_period": "604800s"                                // 7 days
  },
  "voting_params": {
    "voting_period": "1209600s"                                    // 14 days
  },
  "tally_params": {
    "quorum": "0.40",                     // 40% participation required
    "threshold": "0.667",                 // 66.7% yes votes needed
    "veto_threshold": "0.333"             // 33.3% no-with-veto kills proposal
  }
}
```

**Governance Process:**
1. **Deposit Phase**: 7 days to reach 10,000 PAW deposit
2. **Voting Phase**: 14 days for voting
3. **Quorum**: 40% of bonded tokens must vote
4. **Threshold**: 66.7% yes votes required (excluding abstain)
5. **Veto**: 33.3% no-with-veto rejects and burns deposits

### 8. Mint Module

Manages token inflation and block rewards.

```json
"mint": {
  "params": {
    "mint_denom": "upaw",
    "inflation_max": "0.05",              // 5% max inflation
    "inflation_min": "0.00",              // Can go to 0%
    "goal_bonded": "0.67",                // Target 67% bonding rate
    "blocks_per_year": "7884000"          // ~4 second blocks
  }
}
```

**Emission Schedule:**
- **Year 1**: 2,870 PAW/day (1.5x multiplier for first 180 days)
- **Year 2**: 1,435 PAW/day (50% reduction)
- **Year 3+**: 717 PAW/day, annual halving with oracle gating

**Inflation Mechanics:**
- Inflation adjusts to maintain 67% bonding rate
- Capped at 5% annual inflation
- Can decrease to 0% if bonding exceeds target

### 9. Slashing Module

Manages validator punishment for misbehavior.

```json
"slashing": {
  "params": {
    "signed_blocks_window": "10000",     // Track last 10,000 blocks
    "min_signed_per_window": "0.95",     // Must sign 95% of blocks
    "downtime_jail_duration": "86400s",  // 24 hour jail for downtime
    "slash_fraction_double_sign": "0.05", // 5% slash for double-sign
    "slash_fraction_downtime": "0.001"    // 0.1% slash for downtime
  }
}
```

**Slashing Conditions:**
- **Downtime**: Missing >5% of blocks in 10,000 block window
  - Penalty: 0.1% stake slashed, 24 hour jail
- **Double Sign**: Signing two blocks at same height
  - Penalty: 5% stake slashed, permanent tombstone

### 10. DEX Module (Custom)

Native decentralized exchange with AMM pools.

```json
"dex": {
  "params": {
    "swap_fee": "0.003",                 // 0.3% total swap fee
    "lp_fee": "0.0025",                  // 0.25% to liquidity providers
    "protocol_fee": "0.0005",            // 0.05% to protocol treasury
    "min_liquidity": "1000",             // Minimum tokens per pool
    "max_slippage_percent": "0.10"       // 10% max slippage protection
  }
}
```

**Initial Pools:**
- **PAW/USDC Pool**: 1M PAW / 1M USDC initial liquidity

**Features:**
- Constant product AMM (x * y = k)
- Multi-hop routing for optimal prices
- Circuit breakers for large trades
- Flash loan support

### 11. Oracle Module (Custom)

Decentralized price feeds for DeFi operations.

```json
"oracle": {
  "params": {
    "min_validators": 17,                // 2/3 of 25 genesis validators
    "update_interval": "100",            // ~7 minutes (100 blocks)
    "expiry_duration": "86400"           // 24 hours max staleness
  },
  "price_feeds": [
    {"asset": "PAW/USD", "price": "1.00"},
    {"asset": "BTC/USD", "price": "50000.00"},
    {"asset": "ETH/USD", "price": "3000.00"}
  ]
}
```

**Price Feed Mechanism:**
- Validators submit prices from multiple sources
- Median price used to prevent manipulation
- 20% deviation threshold for outlier removal
- TWAP (Time-Weighted Average Price) over 10 minutes

**Initial Price Feeds:**
- PAW/USD: $1.00 (initial peg target)
- BTC/USD and ETH/USD for DeFi collateral

### 12. Compute Module (Custom)

Secure compute task routing with TEE integration.

```json
"compute": {
  "params": {
    "min_stake": "10000000000",          // 10,000 PAW minimum
    "verification_timeout": "300",       // 5 minutes
    "max_retries": 3
  }
}
```

**Compute Agent Requirements:**
- **Minimum Stake**: 10,000 PAW to become compute provider
- **TEE Support**: AWS Nitro or Intel SGX enclaves
- **Verification**: 5 minute timeout for task completion
- **Retries**: Up to 3 attempts for failed tasks

## Token Distribution

Total Supply: **50,000,000 PAW** (50 trillion upaw)

| Category | Allocation | Amount (PAW) | Address | Vesting |
|----------|-----------|--------------|---------|---------|
| Public Sale | 14% | 7,000,000 | `paw1publicsale...` | Immediate |
| Mining & Node Rewards | 21% | 10,500,000 | `paw1miningrewards...` | On-chain emission |
| API Donor Rewards | 16.8% | 8,400,000 | `paw1apidonors...` | 4-year cliff |
| Team & Advisors | 7% | 3,500,000 | `paw1team...` | 2-year cliff |
| Foundation Treasury | 7% | 3,500,000 | `paw1foundation...` | Governance |
| Ecosystem Fund | 4.2% | 2,100,000 | `paw1ecosystem...` | Governance |
| Reserve | 30% | 15,000,000 | `paw1reserve...` | Governance unlock |

**Notes:**
- All addresses are placeholder bech32 addresses
- Replace with actual multi-sig addresses before mainnet launch
- Vested tokens use Cosmos SDK vesting accounts (configure separately)

## Validator Setup

### Minimum Validator Requirements

- **Minimum Stake**: 10,000 PAW (10,000,000,000 upaw)
- **Minimum Commission**: 5%
- **Maximum Commission**: 20% (recommended)
- **Max Commission Change**: 1% per day

### Genesis Validator Process

1. **Generate Keys**:
   ```bash
   pawd keys add validator
   pawd tendermint show-validator
   ```

2. **Add Genesis Account**:
   ```bash
   pawd add-genesis-account $(pawd keys show validator -a) 10000000000upaw
   ```

3. **Create Gentx**:
   ```bash
   pawd gentx validator 10000000000upaw \
     --chain-id paw-mainnet-1 \
     --moniker="MyValidator" \
     --commission-rate=0.10 \
     --commission-max-rate=0.20 \
     --commission-max-change-rate=0.01 \
     --min-self-delegation=10000000000 \
     --details="Professional validator service" \
     --security-contact="security@validator.com" \
     --website="https://validator.com"
   ```

4. **Submit Gentx**: Submit your gentx file to the genesis coordinator

5. **Collect All Gentxs**:
   ```bash
   pawd collect-gentxs
   pawd validate-genesis
   ```

### Post-Genesis Validators

After genesis, new validators can join by staking:

```bash
pawd tx staking create-validator \
  --amount=10000000000upaw \
  --pubkey=$(pawd tendermint show-validator) \
  --moniker="MyValidator" \
  --chain-id=paw-mainnet-1 \
  --commission-rate=0.10 \
  --commission-max-rate=0.20 \
  --commission-max-change-rate=0.01 \
  --min-self-delegation=10000000000 \
  --from=validator
```

## Module Parameters

### Modifying Parameters via Governance

All module parameters can be modified through governance proposals:

```bash
# Example: Change min deposit for governance
pawd tx gov submit-proposal param-change proposal.json --from validator

# proposal.json
{
  "title": "Reduce Governance Min Deposit",
  "description": "Lower barrier to entry for proposals",
  "changes": [
    {
      "subspace": "gov",
      "key": "depositparams",
      "value": {
        "min_deposit": [{"denom": "upaw", "amount": "5000000000"}]
      }
    }
  ],
  "deposit": "10000000000upaw"
}
```

### Critical Parameters

**DO NOT modify these without careful consideration:**
- `unbonding_time`: Reducing increases security risk
- `max_validators`: Affects decentralization and security
- `slash_fraction_*`: Affects validator risk/reward
- `bond_denom`: Cannot be changed post-genesis

## Security Considerations

### Pre-Genesis Security

1. **Key Management**:
   - Generate validator keys on air-gapped machines
   - Use hardware security modules (HSM) for production
   - Never share private keys or mnemonics

2. **Multi-Sig Addresses**:
   - Foundation, Ecosystem, and Reserve accounts should be multi-sig
   - Recommend 5-of-9 multi-sig for Foundation
   - Recommend 3-of-5 multi-sig for Ecosystem

3. **Genesis File Verification**:
   - Verify genesis hash with all validators
   - Hash should be: `sha256sum genesis.json`
   - Publish hash on official channels before launch

### Post-Genesis Security

1. **Validator Security**:
   - Use sentry nodes to protect validator
   - Enable firewall rules (only allow P2P from sentries)
   - Monitor validator uptime 24/7
   - Set up alerting for slashing conditions

2. **Network Monitoring**:
   - Monitor block production rate
   - Watch for unusual transaction patterns
   - Track validator set changes
   - Monitor oracle price feeds for manipulation

3. **Governance Vigilance**:
   - Review all governance proposals carefully
   - Coordinate with other validators on critical votes
   - Set up alerts for new proposals

## Troubleshooting

### Genesis File Validation Errors

**Error**: "genesis.json is invalid"
```bash
# Check JSON syntax
jq . config/genesis-mainnet.json

# Validate against schema
pawd validate-genesis
```

**Error**: "account balances do not match supply"
- Verify all account balances sum to total supply
- Check for duplicate addresses

**Error**: "invalid validator power"
- Ensure all gentx stakes >= min_validator_stake
- Verify bond_denom matches token denom

### Node Startup Issues

**Error**: "Wrong Block.Header.AppHash"
- Genesis file doesn't match network
- Download correct genesis from official source
- Verify genesis hash

**Error**: "peer handshake failed"
- Check chain-id matches network
- Verify genesis_time hasn't passed (for pre-genesis)
- Check network connectivity

### Common Questions

**Q: Can I change genesis_time?**
A: Only before network launch. All validators must coordinate.

**Q: How do I add more genesis validators?**
A: Collect additional gentx files and run `pawd collect-gentxs` again before genesis_time.

**Q: What if I miss signing blocks?**
A: You'll be jailed after missing >5% of blocks in a 10k block window. Unjail with:
```bash
pawd tx slashing unjail --from validator
```

**Q: Can I change my commission rate?**
A: Yes, but limited to max_change_rate (1%) per day.

**Q: How do I upgrade the network?**
A: Submit software upgrade proposal through governance.

## Additional Resources

- **Whitepaper**: `PAW Extensive whitepaper.md`
- **Node Configuration**: `config/node-config.toml.template`
- **Validator Guide**: `docs/validator-guide.md`
- **API Documentation**: `docs/api.md`
- **Security Policy**: `SECURITY.md`

## Support

- **Discord**: (Add community Discord link)
- **Documentation**: `docs/`
- **Validator Chat**: (Add validator coordination channel)

---

**Last Updated**: November 19, 2025
**Genesis Version**: 1.0
**Compatible with**: pawd v1.0.0+
