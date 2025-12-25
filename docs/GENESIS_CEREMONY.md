# PAW Blockchain Genesis Ceremony Guide

## Overview

This document outlines the genesis ceremony process for launching the PAW blockchain mainnet. The genesis ceremony is a coordinated effort among validators to create and verify the initial state of the blockchain.

## Prerequisites

- Minimum 4 validators for testnet, 7+ for mainnet (BFT requirement)
- Each validator must have:
  - Server meeting hardware requirements (see `VALIDATOR_HARDWARE_REQUIREMENTS.md`)
  - `pawd` binary installed and verified
  - GPG key pair for signing
  - Secure communication channel with other validators

## Timeline

| Phase | Duration | Description |
|-------|----------|-------------|
| 1. Preparation | 1 week | Validator registration, key generation |
| 2. Gentx Collection | 3 days | Validators submit signed gentx |
| 3. Genesis Creation | 1 day | Coordinator creates genesis file |
| 4. Verification | 2 days | All validators verify genesis |
| 5. Launch | T-0 | Coordinated chain start |

## Phase 1: Preparation

### 1.1 Initialize Node

```bash
# Initialize node with your moniker
pawd init <your-moniker> --chain-id paw-1

# Generate validator keys
pawd keys add validator --keyring-backend file
```

### 1.2 Create Validator Account

```bash
# Get your validator address
pawd keys show validator -a --keyring-backend file

# Get your validator pubkey
pawd tendermint show-validator
```

### 1.3 Submit Registration

Submit to the genesis coordinator:
- Validator address
- Validator pubkey
- Self-delegation amount (minimum: 1,000,000 upaw)
- Commission rates
- Validator description/website

## Phase 2: Gentx Submission

### 2.1 Generate Gentx

```bash
# Add genesis account (coordinator will provide amounts)
pawd genesis add-genesis-account $(pawd keys show validator -a --keyring-backend file) 10000000000upaw

# Create gentx
pawd genesis gentx validator 1000000000upaw \
  --chain-id paw-1 \
  --moniker "<your-moniker>" \
  --commission-rate "0.05" \
  --commission-max-rate "0.20" \
  --commission-max-change-rate "0.01" \
  --pubkey $(pawd tendermint show-validator) \
  --keyring-backend file
```

### 2.2 Sign and Submit Gentx

```bash
# Locate your gentx file
ls ~/.paw/config/gentx/

# Sign with GPG
gpg --armor --detach-sign ~/.paw/config/gentx/gentx-*.json

# Submit both files to coordinator
# - gentx-<validator>.json
# - gentx-<validator>.json.asc
```

## Phase 3: Genesis File Creation

The genesis coordinator will:

1. Collect all gentx files
2. Verify GPG signatures
3. Add genesis accounts
4. Set chain parameters
5. Create genesis.json
6. Sign and publish genesis hash

### Chain Parameters

```json
{
  "chain_id": "paw-1",
  "genesis_time": "2025-01-15T00:00:00Z",
  "consensus_params": {
    "block": {
      "max_bytes": "22020096",
      "max_gas": "100000000"
    },
    "evidence": {
      "max_age_num_blocks": "100000",
      "max_age_duration": "172800000000000"
    }
  }
}
```

## Phase 4: Genesis Verification

### 4.1 Download Genesis

```bash
# Download from official source
wget https://github.com/paw-chain/networks/releases/download/mainnet-v1.0.0/genesis.json \
  -O ~/.paw/config/genesis.json

# Download signature
wget https://github.com/paw-chain/networks/releases/download/mainnet-v1.0.0/genesis.json.asc
```

### 4.2 Verify Checksum

```bash
# Expected checksum (announced by coordinator)
EXPECTED_HASH="<announced-sha256-hash>"

# Calculate actual hash
ACTUAL_HASH=$(sha256sum ~/.paw/config/genesis.json | awk '{print $1}')

# Verify
if [ "$EXPECTED_HASH" = "$ACTUAL_HASH" ]; then
  echo "✅ Genesis checksum VERIFIED"
else
  echo "❌ CHECKSUM MISMATCH - DO NOT START"
  exit 1
fi
```

### 4.3 Verify GPG Signature

```bash
# Import coordinator's public key
gpg --import coordinator-pubkey.asc

# Verify signature
gpg --verify genesis.json.asc ~/.paw/config/genesis.json
```

### 4.4 Validate Genesis Contents

```bash
# Validate genesis file structure
pawd genesis validate-genesis

# Verify your validator is included
grep -q "$(pawd keys show validator -a --keyring-backend file)" ~/.paw/config/genesis.json && \
  echo "✅ Your validator is in genesis" || echo "❌ Validator NOT found"
```

## Phase 5: Coordinated Launch

### 5.1 Pre-Launch Checklist

- [ ] Genesis file downloaded and verified
- [ ] GPG signature verified
- [ ] Node fully synced (if joining existing network) or genesis validated
- [ ] Persistent peers configured
- [ ] Seeds configured
- [ ] Firewall rules configured (ports 26656, 26657)
- [ ] Monitoring configured

### 5.2 Configure Peers

```bash
# Add persistent peers (provided by coordinator)
PEERS="node1@ip1:26656,node2@ip2:26656,node3@ip3:26656"
sed -i "s/persistent_peers = \"\"/persistent_peers = \"$PEERS\"/" ~/.paw/config/config.toml
```

### 5.3 Start Node at Genesis Time

```bash
# Use cosmovisor for automatic upgrades
cosmovisor run start

# Or start directly
pawd start
```

### 5.4 Verify Launch

```bash
# Check node status
pawd status | jq '.SyncInfo'

# Verify you're producing blocks (if validator)
pawd query staking validator $(pawd keys show validator --bech val -a --keyring-backend file)
```

## Security Considerations

1. **Never share your private keys** - Only share public keys and addresses
2. **Verify all downloads** - Always check checksums and GPG signatures
3. **Use secure communication** - Encrypted channels for coordinator communication
4. **Backup keys securely** - Store validator keys in secure, offline location
5. **Monitor for double-signing** - Ensure only one instance of your validator runs

## Troubleshooting

### Genesis Validation Fails

```bash
# Check for common issues
pawd genesis validate-genesis 2>&1 | head -50

# Common fixes:
# - Ensure chain-id matches
# - Verify genesis_time is in the future
# - Check all accounts have valid addresses
```

### Node Won't Start

```bash
# Check logs
journalctl -u pawd -f

# Common issues:
# - Wrong genesis file
# - Missing peers
# - Port conflicts
# - Insufficient resources
```

### Missed Genesis

If you miss the genesis block:

1. Get a snapshot from a trusted validator
2. Or state-sync from existing nodes
3. Contact the validator community for assistance

## Contact

- **Genesis Coordinator**: genesis@paw-chain.io
- **Discord**: #genesis-ceremony channel
- **Telegram**: @paw_genesis

## References

- [Validator Hardware Requirements](VALIDATOR_HARDWARE_REQUIREMENTS.md)
- [Validator Onboarding Guide](VALIDATOR_ONBOARDING_GUIDE.md)
- [Security Best Practices](SECURITY_MODEL.md)
