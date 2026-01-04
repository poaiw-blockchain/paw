# Validator Key Management

Comprehensive guide for secure validator key generation, storage, and management in PAW blockchain.

## Table of Contents

1. [Introduction to Validator Keys](#introduction-to-validator-keys)
2. [Air-Gapped Key Generation](#air-gapped-key-generation)
3. [HSM Integration](#hsm-integration)
4. [Key Backup and Recovery](#key-backup-and-recovery)
5. [Multi-Signature Schemes](#multi-signature-schemes)
6. [Security Best Practices](#security-best-practices)
7. [Emergency Procedures](#emergency-procedures)

---

## Introduction to Validator Keys

### Key Types Overview

PAW validators use three distinct cryptographic keys, each serving a specific purpose:

#### 1. Consensus Key (Tendermint/CometBFT Private Validator Key)

**Purpose:** Sign blocks and votes in the consensus protocol.

**Characteristics:**
- Algorithm: `ed25519`
- Location: `~/.paw/config/priv_validator_key.json`
- Critical security requirement: MUST be unique per validator
- Usage: Hot key (accessed continuously during block production)
- Compromise impact: **CRITICAL** - enables double-signing, slashing, chain halt

**File format:**
```json
{
  "address": "E1A6C23E38A41B2F0E4D8A5C90D3F2B1E6A7C8D9",
  "pub_key": {
    "type": "tendermint/PubKeyEd25519",
    "value": "Base64EncodedPublicKey=="
  },
  "priv_key": {
    "type": "tendermint/PrivKeyEd25519",
    "value": "Base64EncodedPrivateKey=="
  }
}
```

**Security requirements:**
- Never share between validators
- Never copy to multiple machines
- Use HSM or KMS for mainnet
- Monitor for double-sign attempts
- Back up encrypted in multiple secure locations

#### 2. Operator Key (Validator Operator Account)

**Purpose:** Execute staking transactions (create validator, edit validator, delegate, unbond).

**Characteristics:**
- Algorithm: `secp256k1` (Cosmos SDK standard)
- Location: Keyring (software, OS, Ledger, file)
- Usage: Cold key (used infrequently for governance and staking operations)
- Compromise impact: **HIGH** - enables unauthorized staking changes, token theft

**Management:**
```bash
# Create operator key
pawd keys add validator-operator

# Show address
pawd keys show validator-operator -a

# Export for backup (encrypted)
pawd keys export validator-operator > operator-key-backup.json
```

**Security requirements:**
- Store in hardware wallet (Ledger) for mainnet
- Use multi-sig for high-value validators
- Separate from consensus key
- Require multiple approvals for critical operations
- Regular audit of permissions

#### 3. Node Key (P2P Identity)

**Purpose:** Authenticate in P2P network (libp2p/Tendermint P2P).

**Characteristics:**
- Algorithm: `ed25519`
- Location: `~/.paw/config/node_key.json`
- Usage: Hot key (used for all P2P connections)
- Compromise impact: **LOW** - enables P2P impersonation, eclipse attacks

**File format:**
```json
{
  "priv_key": {
    "type": "tendermint/PrivKeyEd25519",
    "value": "Base64EncodedPrivateKey=="
  }
}
```

**Security requirements:**
- Unique per node
- Can be regenerated without slashing risk
- Back up for network stability
- Monitor for unauthorized usage

---

## Air-Gapped Key Generation

Air-gapped key generation is **MANDATORY** for mainnet validators and **STRONGLY RECOMMENDED** for high-value testnets.

### Required Hardware

1. **Air-gapped machine:**
   - Never connected to network
   - Minimal OS installation (Ubuntu Server, Debian, or Tails Linux)
   - Verified clean installation
   - Physical security (locked room)

2. **Secure transfer media:**
   - New USB drives (2+ for redundancy)
   - Formatted with encrypted filesystem
   - Verified malware-free

3. **Printer (optional but recommended):**
   - Dedicated printer never connected to network
   - For paper backups of recovery phrases

### Procedure

#### Step 1: Prepare Air-Gapped Environment

```bash
# On air-gapped machine (offline Ubuntu/Debian)

# Install minimal dependencies (from verified offline source)
# Transfer Go binary and pawd source via USB

# Verify binary checksums
sha256sum go1.21.linux-amd64.tar.gz
# Compare with official Go release checksums

# Install Go
sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Build pawd from source (transferred via USB)
cd paw
go build -o pawd ./cmd/...

# Verify build reproducibility
sha256sum pawd
```

#### Step 2: Generate Consensus Key (Air-Gapped)

**Option A: Using pawd init (generates all keys)**

```bash
# Initialize node (this creates priv_validator_key.json)
./pawd init validator-airgap --chain-id paw-mainnet-1

# Keys are now in ~/.paw/config/
ls -la ~/.paw/config/

# Expected files:
# - priv_validator_key.json (CRITICAL - keep offline)
# - node_key.json
# - genesis.json
# - config.toml
# - app.toml
```

**Option B: Using tendermint/cometbft directly (more control)**

```bash
# Install cometbft on air-gapped machine
# (transfer binary via USB)

# Generate only validator key
cometbft gen-validator > priv_validator_key.json

# Set proper permissions
chmod 600 priv_validator_key.json

# Inspect (DO NOT log this output)
cat priv_validator_key.json
```

**Option C: Using custom secure random source**

For maximum security, generate using hardware RNG:

```bash
# Use hardware random source (if available)
# Install rng-tools on air-gapped machine

# Generate raw key material
dd if=/dev/hwrng bs=32 count=1 > validator_seed.bin

# Create key from seed (custom tool or use pawd with seed input)
# Implementation depends on key derivation requirements
```

#### Step 3: Extract Public Key Only

**CRITICAL: Only the public key leaves the air-gapped machine for online validator.**

```bash
# Extract public key from priv_validator_key.json
cat ~/.paw/config/priv_validator_key.json | jq -r '.pub_key'

# Example output:
# {
#   "type": "tendermint/PubKeyEd25519",
#   "value": "X7g2L9c5K3h8..."
# }

# For Tendermint KMS (tmkms), export in appropriate format
# See HSM Integration section below
```

**Transfer public key to online validator:**

1. Write public key to USB (plain text - it's public)
2. Transfer USB to online validator machine
3. Configure validator to use remote signer (tmkms) or public key

#### Step 4: Generate Operator Key (Air-Gapped)

```bash
# Generate operator key
./pawd keys add validator-operator

# CRITICAL: Write down 24-word mnemonic phrase
# This is the ONLY way to recover the operator key

# Example output:
# - name: validator-operator
#   type: local
#   address: paw1...
#   pubkey: '{"@type":"/cosmos.crypto.secp256k1.PubKey","key":"A..."}'
#   mnemonic: ""
#
# **Important** write this mnemonic phrase in a safe place.
# It is the only way to recover your account if you ever forget your password.
#
# word1 word2 word3 ... word24

# IMMEDIATELY write mnemonic on paper (2+ copies)
# Store in separate secure locations

# Export encrypted backup
./pawd keys export validator-operator > operator-backup.json
# Enter strong passphrase (25+ characters, random)
```

#### Step 5: Secure Backup Creation

**Consensus key backup:**

```bash
# Create encrypted backup of consensus key
# Method 1: GPG symmetric encryption
gpg --symmetric --cipher-algo AES256 ~/.paw/config/priv_validator_key.json

# Enter strong passphrase (different from operator key)
# Output: priv_validator_key.json.gpg

# Method 2: OpenSSL encryption
openssl enc -aes-256-cbc -salt -pbkdf2 -in ~/.paw/config/priv_validator_key.json \
  -out priv_validator_key.json.enc

# Verify encryption worked
file priv_validator_key.json.enc
# Should show: data (encrypted)

# Create checksums
sha256sum priv_validator_key.json.gpg > checksums.txt
sha256sum operator-backup.json >> checksums.txt
```

**Paper backup (for operator key mnemonic):**

```txt
PAW VALIDATOR OPERATOR KEY BACKUP
Generated: 2025-12-06
Address: paw1...

MNEMONIC PHRASE (24 words):
1. word1      7. word7      13. word13    19. word19
2. word2      8. word8      14. word14    20. word20
3. word3      9. word9      15. word15    21. word21
4. word4     10. word10     16. word16    22. word22
5. word5     11. word11     17. word17    23. word23
6. word6     12. word12     18. word18    24. word24

CRITICAL SECURITY WARNINGS:
- This phrase can recover your validator operator key
- Anyone with this phrase can control your validator
- NEVER enter this phrase on any internet-connected device
- NEVER photograph or scan this document
- Store in fireproof safe or bank vault
- Create 3+ copies in separate physical locations

Backup verification checksum (mnemonic SHA256):
[checksum of mnemonic for verification]
```

#### Step 6: Secure Transfer and Storage

**Transfer to online validator (public key only):**

```bash
# On USB drive (encrypted filesystem):
# - pub_key_only.json (extracted from priv_validator_key.json)
# - operator_address.txt (public address)

# DO NOT TRANSFER:
# - priv_validator_key.json (stays air-gapped or in HSM)
# - operator mnemonic (paper only)
# - operator-backup.json (unless moving to another air-gapped machine)
```

**Storage of encrypted backups:**

1. **Primary location:** Fireproof safe in secure facility
2. **Secondary location:** Bank safety deposit box
3. **Tertiary location:** Trusted co-founder/partner secure storage
4. **Verification:** Test recovery procedure quarterly

**Destroy intermediate artifacts:**

```bash
# On air-gapped machine after backup verified:
# Securely wipe plaintext files that were exported

# Wipe USB after transfer (if it contained sensitive data)
shred -vfz -n 5 /path/to/sensitive/file

# For USBs, use hardware destruction for maximum security
```

#### Step 7: Verification

**Verify backup integrity:**

```bash
# Decrypt backup and verify checksum
gpg --decrypt priv_validator_key.json.gpg > priv_validator_key_restored.json
sha256sum priv_validator_key_restored.json

# Compare with original checksum
# Must match exactly

# Test operator key recovery
./pawd keys delete validator-operator
./pawd keys import validator-operator operator-backup.json
# Or recover from mnemonic:
./pawd keys add validator-operator --recover
# Enter mnemonic when prompted

# Verify address matches
./pawd keys show validator-operator -a
```

---

## HSM Integration

Hardware Security Modules (HSMs) provide tamper-resistant key storage and cryptographic operations. **MANDATORY for mainnet validators managing significant stake.**

### HSM Options Comparison

| HSM Solution | Security Level | Cost | Complexity | Recommendation |
|--------------|----------------|------|------------|----------------|
| YubiHSM 2 | High | $650 | Medium | Best for single validators |
| AWS CloudHSM | Very High | $1.50/hr + setup | High | Enterprise multi-validator |
| Tendermint KMS (tmkms) | High (depends on backend) | Free (software) | Medium-High | Most common for Cosmos |
| Ledger Nano (operator key only) | Medium-High | $150 | Low | Operator key signing |
| Azure Key Vault HSM | Very High | Variable | High | Enterprise with Azure infra |

### YubiHSM 2 Setup (Most Common)

YubiHSM 2 is a USB HSM device ideal for single validator setups.

#### Prerequisites

```bash
# Install YubiHSM libraries
# Ubuntu/Debian:
sudo apt-get update
sudo apt-get install yubihsm-shell yubihsm-connector

# Verify device connection
lsusb | grep Yubico
# Should show: Yubico.com Yubikey 4/5 U2F+CCID
```

#### Initial YubiHSM Setup

```bash
# Start YubiHSM connector
yubihsm-connector -d

# Connect with default credentials
yubihsm-shell

# In yubihsm-shell:
connect

# Change default password (CRITICAL - default is well-known)
session open 1 password
# Default password: "password"

# Create authentication key
put authkey 0 0 "Validator Auth Key" all <your-secure-password>

# Generate signing key for validator
generate asymmetric 0 0 validator-key-1 all \
  sign-eddsa asymmetric-key-ed ed25519

# Export public key
get public 0x0001 > validator_public_key.pem

exit
```

#### Integrate with Tendermint KMS

See [Tendermint KMS Integration](#tendermint-kms-tmkms) section below.

### AWS CloudHSM Setup

AWS CloudHSM provides FIPS 140-2 Level 3 certified HSMs. Suitable for enterprise deployments.

#### Architecture

```
┌─────────────────┐
│ Validator Node  │
│                 │
│  pawd process   │
└────────┬────────┘
         │ TCP/26658
         │ (remote signer)
         ▼
┌─────────────────┐         ┌──────────────┐
│  tmkms process  │◄────────┤ AWS CloudHSM │
│  (bastion/signer│  PKCS#11 │  Cluster     │
│   instance)     │          └──────────────┘
└─────────────────┘
```

#### Setup Procedure

```bash
# 1. Create CloudHSM cluster (AWS Console or CLI)
aws cloudhsmv2 create-cluster \
  --hsm-type hsm1.medium \
  --subnet-ids subnet-xxx subnet-yyy \
  --source-backup-id <optional>

# 2. Initialize cluster
aws cloudhsmv2 initialize-cluster --cluster-id <cluster-id>

# 3. Launch EC2 instance for tmkms (bastion)
# - Use AWS Linux 2 AMI
# - Attach to same VPC as CloudHSM
# - Security group allows CloudHSM port (2223-2225)

# 4. Install CloudHSM client on bastion
wget https://s3.amazonaws.com/cloudhsmv2-software/CloudHsmClient/EL7/cloudhsm-client-latest.el7.x86_64.rpm
sudo yum install -y ./cloudhsm-client-latest.el7.x86_64.rpm

# 5. Configure client
sudo /opt/cloudhsm/bin/configure -a <cluster-ip>

# 6. Start CloudHSM client
sudo start cloudhsm-client

# 7. Initialize HSM user
/opt/cloudhsm/bin/cloudhsm_mgmt_util /opt/cloudhsm/etc/cloudhsm_mgmt_util.cfg

# In cloudhsm_mgmt_util:
enable_e2e
loginHSM PRECO admin <initial-password>
changePswd PRECO admin <new-password>
createUser CU validator-user <user-password>
quit

# 8. Install and configure tmkms (see tmkms section)
```

### Tendermint KMS (tmkms)

Tendermint KMS is the standard remote signer for Cosmos SDK validators, supporting multiple HSM backends.

#### Architecture

```
┌──────────────────────────┐
│   Validator Node         │
│   (online, exposed)      │
│                          │
│   pawd process           │
│   priv_validator_laddr = │
│   "tcp://0.0.0.0:26658"  │
└────────────┬─────────────┘
             │
             │ Encrypted TCP
             │ (mutual auth)
             ▼
┌──────────────────────────┐       ┌─────────────┐
│   Signing Node           │       │   YubiHSM2  │
│   (firewalled)           │◄──────┤   or        │
│                          │  USB  │   CloudHSM  │
│   tmkms process          │       └─────────────┘
│   (holds consensus key)  │
└──────────────────────────┘
```

#### Installation

```bash
# Install Rust (tmkms is written in Rust)
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source $HOME/.cargo/env

# Install tmkms with YubiHSM support
cargo install tmkms --features=yubihsm --locked

# Or for AWS CloudHSM (PKCS#11):
cargo install tmkms --features=softsign --locked

# Verify installation
tmkms version
```

#### Configuration for YubiHSM

```bash
# Initialize tmkms configuration
tmkms init /etc/tmkms

# Edit configuration
sudo vi /etc/tmkms/tmkms.toml
```

**tmkms.toml configuration:**

```toml
# Tendermint KMS configuration for PAW validator

# Connection to validator node
[[chain]]
id = "paw-mainnet-1"
key_format = { type = "bech32", account_key_prefix = "pawpub", consensus_key_prefix = "pawvalconspub" }
state_file = "/var/lib/tmkms/state/paw-mainnet-1-consensus.json"

# Validator node connection
[[validator]]
addr = "tcp://validator.example.com:26658"
chain_id = "paw-mainnet-1"
reconnect = true
secret_key = "/etc/tmkms/secrets/kms-identity.key"
protocol_version = "v0.34"
max_height = "none"

# YubiHSM provider
[[providers.yubihsm]]
adapter = { type = "usb" }
auth = { key = 1, password_file = "/etc/tmkms/secrets/yubihsm-password" }
keys = [
    { chain_ids = ["paw-mainnet-1"], key = 1 }
]
serial_number = "9876543210"  # Your YubiHSM serial

# Alternative: HTTP connector
# [[providers.yubihsm]]
# adapter = { type = "http", addr = "http://localhost:12345" }

# Logging
[log]
level = "info"
format = "json"
```

**Generate tmkms identity (for encrypted connection to validator):**

```bash
# Generate secret key for tmkms <-> validator authentication
tmkms softsign keygen /etc/tmkms/secrets/kms-identity.key

# Set permissions
chmod 600 /etc/tmkms/secrets/kms-identity.key

# Extract public key for validator config
tmkms softsign pubkey /etc/tmkms/secrets/kms-identity.key
```

#### Import Consensus Key to YubiHSM

**Option 1: Generate key directly on YubiHSM (recommended):**

```bash
# Using tmkms with YubiHSM
tmkms yubihsm keys generate 1 -b

# This generates ed25519 key in slot 1
# Key never leaves HSM
```

**Option 2: Import existing key (if migrating):**

```bash
# ONLY use this if you have an existing validator key to migrate
# This temporarily exposes the private key - use air-gapped machine

# Convert Tendermint priv_validator_key.json to raw format
tmkms yubihsm keys import -i /path/to/priv_validator_key.json 1

# IMMEDIATELY destroy plaintext key after import
shred -vfz -n 10 /path/to/priv_validator_key.json
```

#### Configure Validator for Remote Signer

On validator node, edit `~/.paw/config/config.toml`:

```toml
# Disable local signing
priv_validator_key_file = ""
priv_validator_state_file = "data/priv_validator_state.json"

# Enable remote signer
priv_validator_laddr = "tcp://0.0.0.0:26658"
```

**CRITICAL SECURITY: Firewall configuration**

```bash
# Only allow tmkms signer IP to connect to port 26658
sudo ufw allow from <tmkms-ip> to any port 26658 proto tcp
sudo ufw deny 26658

# Verify
sudo ufw status numbered
```

#### Start tmkms

```bash
# Start tmkms (foreground for testing)
tmkms start -c /etc/tmkms/tmkms.toml

# Expected output:
# INFO tmkms::commands::start: tmkms 0.12.2 starting up...
# INFO tmkms::keyring: [keyring:yubihsm] added consensus key ...
# INFO tmkms::session: [paw-mainnet-1] connected to validator

# For production, use systemd service:
sudo vi /etc/systemd/system/tmkms.service
```

**tmkms systemd service:**

```ini
[Unit]
Description=Tendermint KMS for PAW Validator
After=network.target

[Service]
Type=simple
User=tmkms
WorkingDirectory=/etc/tmkms
ExecStart=/home/tmkms/.cargo/bin/tmkms start -c /etc/tmkms/tmkms.toml
Restart=on-failure
RestartSec=10
LimitNOFILE=65535

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/tmkms

[Install]
WantedBy=multi-user.target
```

```bash
# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable tmkms
sudo systemctl start tmkms

# Monitor logs
sudo journalctl -u tmkms -f
```

#### Verify Double-Sign Protection

```bash
# tmkms maintains state to prevent double-signing
cat /var/lib/tmkms/state/paw-mainnet-1-consensus.json

# Example:
# {
#   "height": 1234567,
#   "round": 0,
#   "step": 3,
#   "block_id": "ABCD...",
#   "signature": "1234..."
# }

# tmkms will REFUSE to sign any block/vote at a height/round it has already signed
```

### Ledger Hardware Wallet (Operator Key Only)

Ledger Nano S/X can store the operator key for signing staking transactions.

**Note:** Ledger cannot be used for consensus key (real-time block signing), only operator key.

#### Setup

```bash
# Install Ledger support
go get github.com/cosmos/ledger-cosmos-go

# Connect Ledger device
# Install Cosmos app on Ledger via Ledger Live

# Create operator key on Ledger
pawd keys add validator-operator --ledger

# Expected output:
# - name: validator-operator
#   type: ledger
#   address: paw1...
#   pubkey: '{"@type":"/cosmos.crypto.secp256k1.PubKey","key":"A..."}'
#   mnemonic: ""

# Sign transactions with Ledger
pawd tx staking create-validator \
  --from validator-operator \
  --ledger \
  ... other flags
# Device will prompt for approval
```

---

## Key Backup and Recovery

### Backup Strategy

Comprehensive backup strategy following 3-2-1 rule:
- **3** copies of data
- **2** different storage media
- **1** off-site location

#### Backup Locations

| Location Type | Purpose | Contents | Access Frequency |
|---------------|---------|----------|------------------|
| Primary (fireproof safe) | Immediate recovery | Encrypted consensus key, operator key backup | Emergency only |
| Secondary (bank vault) | Disaster recovery | Encrypted consensus key, operator mnemonic paper | Annual verification |
| Tertiary (trusted partner) | Ultimate fallback | Encrypted backups, sealed envelope with passphrase | Co-recovery only |

#### Encryption Standards

**Symmetric encryption (for consensus key files):**

```bash
# Use GPG with strong cipher
gpg --symmetric \
  --cipher-algo AES256 \
  --s2k-mode 3 \
  --s2k-count 65011712 \
  --compress-algo none \
  priv_validator_key.json

# Or use OpenSSL with PBKDF2
openssl enc -aes-256-cbc -salt -pbkdf2 -iter 100000 \
  -in priv_validator_key.json \
  -out priv_validator_key.json.enc
```

**Passphrase requirements:**
- Minimum 25 characters
- High entropy (use password manager generator)
- Never reuse between keys
- Store passphrase separately from encrypted file

**Asymmetric encryption (for shared backups):**

```bash
# Encrypt to multiple GPG recipients (multi-party recovery)
gpg --encrypt \
  --recipient alice@validator.com \
  --recipient bob@validator.com \
  --recipient carol@validator.com \
  priv_validator_key.json

# Requires 1 of 3 to decrypt
# For threshold encryption, use Shamir Secret Sharing (see Multi-sig section)
```

### Recovery Procedures

#### Scenario 1: Validator Machine Failure (HSM/tmkms intact)

**Assumption:** Consensus key is in HSM, operator key backed up, validator machine crashed.

```bash
# 1. Provision new validator machine
# 2. Install pawd
go build -o pawd ./cmd/...

# 3. Initialize node with same moniker
./pawd init <original-moniker> --chain-id paw-mainnet-1

# 4. Restore genesis and config
# Download current genesis
curl https://rpc.paw.network/genesis | jq '.result.genesis' > ~/.paw/config/genesis.json

# 5. Configure for remote signer (tmkms)
vi ~/.paw/config/config.toml
# Set priv_validator_laddr = "tcp://0.0.0.0:26658"

# 6. Restore operator key
pawd keys add validator-operator --recover
# Enter backed-up mnemonic

# 7. Update tmkms validator address (if IP changed)
vi /etc/tmkms/tmkms.toml
# Update [[validator]] addr

# 8. Restart tmkms
sudo systemctl restart tmkms

# 9. Start validator
pawd start

# 10. Verify signing
pawd status
# Check if catching up and signing blocks
```

**Recovery time:** 10-30 minutes
**Downtime risk:** Minimal if seeds/peers configured properly

#### Scenario 2: HSM Failure (YubiHSM destroyed)

**Assumption:** YubiHSM device physically destroyed, encrypted backup available.

**CRITICAL:** This requires exposing private key temporarily - use air-gapped machine.

```bash
# ON AIR-GAPPED MACHINE:

# 1. Decrypt consensus key backup
gpg --decrypt priv_validator_key.json.gpg > priv_validator_key.json
# Enter passphrase

# 2. Verify integrity
sha256sum priv_validator_key.json
# Compare with stored checksum

# 3. Import to NEW YubiHSM device
tmkms yubihsm keys import -i priv_validator_key.json 1

# 4. Verify import
tmkms yubihsm keys list

# 5. IMMEDIATELY destroy plaintext key
shred -vfz -n 10 priv_validator_key.json

# 6. Transfer new YubiHSM to validator infrastructure
# 7. Restart tmkms with new device
# 8. Verify connection and signing
```

**Recovery time:** 1-4 hours
**Downtime risk:** HIGH - requires encrypted backup retrieval, air-gapped procedure

**Mitigation:** Keep spare YubiHSM with imported key in secure storage

#### Scenario 3: Complete Disaster (All machines destroyed, only backups remain)

**Assumption:** Validator, signer, and HSM all destroyed. Only encrypted backups available.

```bash
# 1. Retrieve encrypted backups from all locations
# Verify checksums match

# 2. Set up NEW air-gapped machine
# Transfer encrypted backups via new USB

# 3. Decrypt backups
gpg --decrypt priv_validator_key.json.gpg > priv_validator_key.json
gpg --decrypt operator-backup.json.gpg > operator-backup.json

# 4. Verify integrity
sha256sum priv_validator_key.json operator-backup.json
# Compare with stored checksums

# 5. Import consensus key to NEW HSM
tmkms yubihsm keys import -i priv_validator_key.json 1

# 6. Destroy plaintext consensus key
shred -vfz -n 10 priv_validator_key.json

# 7. Set up new validator infrastructure
# (new VMs, new tmkms signer)

# 8. Configure tmkms with new HSM

# 9. Restore operator key on new machine
pawd keys import validator-operator operator-backup.json

# 10. Initialize new validator node
pawd init <original-moniker> --chain-id paw-mainnet-1

# 11. Configure for remote signer
# Start tmkms, start validator

# 12. Verify signing resumed
pawd status
# Check block height catching up
```

**Recovery time:** 4-24 hours (depends on hardware procurement)
**Downtime risk:** VERY HIGH - validator offline during entire recovery

#### Scenario 4: Operator Key Compromise

**Assumption:** Operator key stolen, attacker can submit staking transactions.

```bash
# IMMEDIATE ACTIONS (within minutes):

# 1. If validator is actively validating, DO NOT stop it yet
# Stopping unbonds and loses rewards

# 2. Verify what attacker did
pawd query staking validator <your-valoper-address>
pawd query distribution validator-outstanding-rewards <your-valoper-address>

# 3. If attacker has not unbonded yet, immediately bond MORE stake from secure key
# This increases your voting power over the attacker

# 4. Create NEW operator key (on air-gapped machine or Ledger)
pawd keys add validator-operator-new --ledger

# 5. Submit edit-validator to change operator
pawd tx staking edit-validator \
  --from validator-operator-new \
  --new-moniker <same-or-new> \
  --chain-id paw-mainnet-1

# WAIT - Cosmos SDK does NOT allow changing operator address
# LIMITATION: Operator address is permanent for a validator

# ONLY OPTION: Create entirely new validator
# 6. Create NEW validator with new operator key
pawd tx staking create-validator \
  --from validator-operator-new \
  --amount 1000000upaw \
  --pubkey $(pawd tendermint show-validator) \
  --moniker "MyValidator-v2" \
  --chain-id paw-mainnet-1 \
  --commission-rate 0.1 \
  --commission-max-rate 0.2 \
  --commission-max-change-rate 0.01 \
  --min-self-delegation 1

# 7. Migrate delegators to new validator (communicate via social channels)

# 8. Unbond from compromised validator (after delegators migrate)
pawd tx staking unbond <old-valoper> 1000000upaw --from validator-operator-new

# 9. Wait unbonding period (21 days typically)

# 10. Forensics: Determine how key was compromised
# - Check logs for unauthorized access
# - Review machine security
# - Audit backup procedures
```

**Recovery time:** 21+ days (unbonding period)
**Impact:** Loss of delegations, reputation damage, potential slashing if attacker double-signs

**PREVENTION IS CRITICAL: Use Ledger or multi-sig for operator key**

### Recovery Testing

**Quarterly recovery drills (MANDATORY for mainnet validators):**

```bash
# Drill 1: Restore from encrypted backup (air-gapped)
# - Retrieve encrypted backup from secondary location
# - Decrypt on air-gapped machine
# - Verify checksum
# - Re-encrypt and return to storage
# - Document time taken

# Drill 2: HSM recovery simulation
# - Use test YubiHSM (NOT production)
# - Import test key
# - Configure test tmkms
# - Connect to testnet validator
# - Verify signing
# - Document issues encountered

# Drill 3: Operator key recovery
# - Use test mnemonic (NOT production)
# - Recover key from paper backup
# - Import to pawd
# - Sign test transaction
# - Verify address matches
# - Document clarity of backup instructions

# Drill 4: Full disaster recovery (on testnet)
# - Destroy test validator VM
# - Recover from backups only
# - Restore full validator function
# - Measure total time
# - Document bottlenecks
```

**Checklist for successful recovery test:**
- [ ] Encrypted backup retrieved from storage
- [ ] Decryption successful with documented passphrase
- [ ] Checksum verification passed
- [ ] Key imported to HSM without errors
- [ ] tmkms connected successfully
- [ ] Validator signing blocks
- [ ] Operator key functional for transactions
- [ ] Total recovery time < 4 hours
- [ ] No plaintext keys left on any system
- [ ] Recovery procedure documentation updated

---

## Multi-Signature Schemes

Multi-signature (multi-sig) accounts require multiple keys to authorize transactions. Essential for high-value validators.

### Use Cases for Multi-Sig

| Scenario | Multi-Sig Configuration | Rationale |
|----------|-------------------------|-----------|
| Foundation validators | 3-of-5 board members | No single point of control |
| Institutional staking | 2-of-3 executives | Requires dual approval |
| Validator business | 2-of-2 co-founders | Mutual consent for operations |
| Emergency recovery | 3-of-7 trustees | Threshold access to backups |

### Creating Multi-Sig Operator Key

```bash
# 1. Each participant generates their key
# Participant 1:
pawd keys add participant1

# Participant 2:
pawd keys add participant2

# Participant 3:
pawd keys add participant3

# 2. Exchange PUBLIC keys (bech32 addresses)
# Participant 1 shares:
pawd keys show participant1 -a
# Example: paw1abc...

# Participant 2 shares:
pawd keys show participant2 -a
# Example: paw1def...

# Participant 3 shares:
pawd keys show participant3 -a
# Example: paw1ghi...

# 3. Create multi-sig account (any participant can do this)
pawd keys add validator-multisig \
  --multisig participant1,participant2,participant3 \
  --multisig-threshold 2

# This creates a 2-of-3 multi-sig account
# Address: paw1xyz... (deterministic from participant pubkeys)

# 4. Fund the multi-sig account
pawd tx bank send funder paw1xyz... 10000000upaw --chain-id paw-mainnet-1
```

### Signing Transactions with Multi-Sig

```bash
# SCENARIO: Create validator with multi-sig operator

# Step 1: Participant 1 generates unsigned transaction
pawd tx staking create-validator \
  --from paw1xyz... \
  --amount 5000000upaw \
  --pubkey $(pawd tendermint show-validator) \
  --moniker "MultiSig Validator" \
  --chain-id paw-mainnet-1 \
  --commission-rate 0.10 \
  --commission-max-rate 0.20 \
  --commission-max-change-rate 0.01 \
  --min-self-delegation 1000000 \
  --gas 300000 \
  --generate-only > create-validator-unsigned.json

# Step 2: Participant 1 signs
pawd tx sign create-validator-unsigned.json \
  --from participant1 \
  --multisig paw1xyz... \
  --chain-id paw-mainnet-1 \
  --output-document participant1-signature.json

# Step 3: Participant 1 sends participant1-signature.json to Participant 2

# Step 4: Participant 2 signs
pawd tx sign create-validator-unsigned.json \
  --from participant2 \
  --multisig paw1xyz... \
  --chain-id paw-mainnet-1 \
  --output-document participant2-signature.json

# Step 5: Any participant combines signatures
pawd tx multisign create-validator-unsigned.json \
  validator-multisig \
  participant1-signature.json \
  participant2-signature.json \
  --chain-id paw-mainnet-1 > create-validator-signed.json

# Step 6: Broadcast transaction
pawd tx broadcast create-validator-signed.json \
  --chain-id paw-mainnet-1

# Transaction now authorized by 2-of-3 signers
```

### Hardware Multi-Sig (Ledger)

Each participant uses Ledger hardware wallet:

```bash
# Participant 1:
pawd keys add participant1 --ledger --index 0

# Participant 2:
pawd keys add participant2 --ledger --index 0

# Participant 3:
pawd keys add participant3 --ledger --index 0

# Create multi-sig (same as above)
pawd keys add validator-multisig \
  --multisig participant1,participant2,participant3 \
  --multisig-threshold 2

# Signing requires physical Ledger devices
pawd tx sign unsigned.json \
  --from participant1 \
  --ledger \
  --multisig paw1xyz... \
  --chain-id paw-mainnet-1 \
  --output-document sig1.json
# Ledger device prompts for approval
```

### Shamir Secret Sharing (Advanced)

For consensus key backup, use Shamir Secret Sharing to split key into shares.

**Use case:** Disaster recovery requiring multiple trustees.

```bash
# Install SSSS (Shamir's Secret Sharing Scheme)
sudo apt-get install ssss

# Split consensus key into 5 shares, requiring 3 to reconstruct
cat priv_validator_key.json | ssss-split -t 3 -n 5

# Example output:
# Generating shares using a (3,5) scheme with dynamic security level.
# Enter the secret, at most 128 ASCII characters: [reads stdin]
# Using a 256 bit security level.
# 1-f8a7b3c2d1e9...
# 2-a1b2c3d4e5f6...
# 3-1a2b3c4d5e6f...
# 4-9f8e7d6c5b4a...
# 5-5e4d3c2b1a09...

# Distribute shares to 5 different trustees
# - Trustee 1: Share 1
# - Trustee 2: Share 2
# - Trustee 3: Share 3
# - Trustee 4: Share 4
# - Trustee 5: Share 5

# To recover, combine ANY 3 shares:
ssss-combine -t 3
# Enter shares from 3 trustees:
# 1-f8a7b3c2d1e9...
# 3-1a2b3c4d5e6f...
# 5-5e4d3c2b1a09...
# Outputs: original priv_validator_key.json content

# SECURITY NOTES:
# - Each share is worthless alone (< threshold)
# - Any 3 shares fully reconstruct the secret
# - Use for emergency recovery, not routine operations
# - Store shares in geographically distributed locations
# - Trustee redundancy: 5 trustees, only need 3 (tolerates 2 unavailable)
```

**Recommended threshold schemes:**

| Total Shares | Threshold | Use Case | Fault Tolerance |
|--------------|-----------|----------|-----------------|
| 3 | 2 | Small validator team | 1 share can be lost |
| 5 | 3 | Medium validator foundation | 2 shares can be lost |
| 7 | 4 | Large institutional validator | 3 shares can be lost |
| 9 | 5 | Critical infrastructure | 4 shares can be lost |

---

## Security Best Practices

### Operational Security Checklist

**Daily:**
- [ ] Monitor tmkms logs for errors
- [ ] Verify validator signing blocks
- [ ] Check for unusual transactions from operator account
- [ ] Monitor server resource usage

**Weekly:**
- [ ] Review access logs for unauthorized attempts
- [ ] Verify backup integrity (checksum)
- [ ] Update dependencies (security patches)
- [ ] Review firewall rules

**Monthly:**
- [ ] Rotate access credentials (SSH keys, API keys)
- [ ] Audit operator account transactions
- [ ] Review and update runbooks
- [ ] Test alerting systems

**Quarterly:**
- [ ] Perform recovery drill
- [ ] Security audit of infrastructure
- [ ] Review and update disaster recovery plan
- [ ] Test backup decryption from all locations

**Annually:**
- [ ] Key rotation (if policy requires)
- [ ] Third-party security audit
- [ ] Review insurance coverage
- [ ] Update trustee/multi-sig participants

### Key Rotation Schedule

**Consensus key:**
- **Mainnet:** NEVER rotate unless compromised (causes validator re-registration)
- **Testnet:** Rotate annually for practice

**Operator key:**
- **CANNOT rotate** (Cosmos SDK limitation - operator address is permanent)
- If compromised: Create new validator, migrate delegators

**Node key (P2P):**
- Rotate every 12 months
- No slashing risk
- Improves P2P network security

**SSH/Server access keys:**
- Rotate every 3 months
- Immediately upon employee departure

**API keys/tokens:**
- Rotate every 6 months
- Immediately if exposed

### Access Control

**Principle of least privilege:**

```bash
# Validator server access:
# - Production signing keys: 0 people (HSM/air-gapped only)
# - Validator SSH: 2-3 senior engineers (multi-factor auth)
# - Monitoring: DevOps team (read-only)
# - Operator key: Multi-sig participants (2-of-3 or higher)

# File permissions:
chmod 600 ~/.paw/config/priv_validator_key.json  # If not using HSM
chmod 600 ~/.paw/config/node_key.json
chmod 644 ~/.paw/config/genesis.json  # Public
chmod 644 ~/.paw/config/config.toml   # Public

# User separation:
# - tmkms runs as dedicated user (not root)
# - pawd runs as dedicated user (not root)
# - No shared service accounts
```

### Monitoring for Key Compromise

**Alerts to configure:**

```yaml
# Prometheus alerting rules

# Alert if validator double-signs
- alert: ValidatorDoubleSigning
  expr: increase(consensus_double_sign_total[5m]) > 0
  severity: critical
  annotations:
    summary: "CRITICAL: Double-sign detected - possible key compromise"
    action: "1. Immediately stop validator 2. Investigate signing infrastructure 3. Verify key uniqueness 4. Rotate keys if confirmed breach"

# Alert if validator missed blocks
- alert: ValidatorMissedBlocks
  expr: increase(consensus_missed_blocks_total[10m]) > 10
  severity: high
  annotations:
    summary: "Validator missing blocks - possible infrastructure issue"
    action: "Check tmkms connection and validator logs"

# Alert if unauthorized operator transactions
- alert: UnauthorizedOperatorTx
  expr: rate(validator_operator_transactions[1h]) > 1
  severity: high
  annotations:
    summary: "Unusual operator account activity"
    action: "Verify transactions are authorized, check for key compromise"

# Alert if tmkms connection lost
- alert: TmkmsConnectionLost
  expr: up{job="tmkms"} == 0
  for: 5m
  severity: critical
  annotations:
    summary: "tmkms signer disconnected"
    action: "Check tmkms service, network, and HSM connection"
```

**Log monitoring:**

```bash
# tmkms logs to watch for:
# - "error: connection refused" → Network issue
# - "error: signature verification failed" → Key mismatch
# - "error: HSM error" → Hardware issue
# - "error: double sign" → CRITICAL KEY COMPROMISE

# pawd logs to watch for:
# - "ERROR: failed to sign" → tmkms unreachable
# - "ERROR: unable to verify signature" → Key mismatch
# - "WARN: validator missed block" → Signing too slow

# Set up log aggregation:
# - Ship logs to centralized system (ELK, Loki, CloudWatch)
# - Alert on ERROR-level messages
# - Retain logs for forensics (90+ days)
```

### Network Security

```bash
# Firewall configuration (validator node)
# ONLY allow:
# - SSH from bastion host IP
# - 26656 (P2P) from known peers/seed nodes
# - 26658 (remote signer) from tmkms signer IP ONLY
# - 26660 (Prometheus) from monitoring IP only

sudo ufw default deny incoming
sudo ufw default allow outgoing

# SSH (from bastion only)
sudo ufw allow from <bastion-ip> to any port 22 proto tcp

# P2P (from known peers)
sudo ufw allow from <peer1-ip> to any port 26656 proto tcp
sudo ufw allow from <peer2-ip> to any port 26656 proto tcp

# Remote signer (from tmkms ONLY)
sudo ufw allow from <tmkms-ip> to any port 26658 proto tcp

# Prometheus (from monitoring)
sudo ufw allow from <monitoring-ip> to any port 26660 proto tcp

# DENY everything else
sudo ufw deny 26657  # RPC - NO public access
sudo ufw deny 1317   # REST - NO public access
sudo ufw deny 9090   # gRPC - NO public access

sudo ufw enable
sudo ufw status numbered
```

### Incident Response Plan

**Key compromise response (step-by-step):**

```
SEVERITY: CRITICAL
TIME SENSITIVITY: Immediate (< 15 minutes)

Step 1: STOP THE VALIDATOR (if double-sign risk)
$ sudo systemctl stop pawd
$ sudo systemctl stop tmkms

Step 2: DISCONNECT FROM NETWORK
$ sudo ufw deny 26656  # Block P2P
$ sudo ufw deny 26658  # Block signer

Step 3: PRESERVE FORENSIC EVIDENCE
$ sudo tar -czf /backup/incident-$(date +%s).tar.gz \
    /var/log/pawd \
    /var/log/tmkms \
    ~/.paw/config \
    /etc/tmkms
$ sudo chmod 400 /backup/incident-*.tar.gz

Step 4: NOTIFY STAKEHOLDERS
- Alert co-founders/trustees
- Notify delegators via social channels
- Report to chain security team (if applicable)

Step 5: ASSESS DAMAGE
- Check blockchain for double-signs: $ pawd query slashing signing-info
- Check for unauthorized operator txs: $ pawd query txs --events sender=<operator>
- Review access logs: $ sudo last -f /var/log/auth.log

Step 6: DETERMINE ROOT CAUSE
- Was HSM compromised? (physical access logs)
- Was tmkms server compromised? (intrusion detection)
- Was operator key stolen? (keyring access logs)
- Social engineering? (review recent communications)

Step 7: REMEDIATION
- If consensus key compromised: Migrate to new validator (see Recovery)
- If operator key compromised: Create new validator (operator address is permanent)
- If infrastructure compromised: Rebuild from clean images

Step 8: POST-INCIDENT
- Full security audit
- Update runbooks
- Conduct team training
- Implement additional controls
- Public disclosure (if warranted)
```

---

## Emergency Procedures

### Emergency Contact List

Maintain up-to-date contact information:

```yaml
# VALIDATOR_EMERGENCY_CONTACTS.yaml

validator_name: "PAW Mainnet Validator"
chain_id: "paw-mainnet-1"

primary_operator:
  name: "Alice Engineer"
  phone: "+1-555-0100"
  email: "alice@validator.com"
  pgp_key: "0xABCD1234"
  signal: "+1-555-0100"
  availability: "24/7"

secondary_operator:
  name: "Bob Engineer"
  phone: "+1-555-0101"
  email: "bob@validator.com"
  pgp_key: "0xEF567890"
  signal: "+1-555-0101"
  availability: "Business hours + on-call rotation"

security_lead:
  name: "Carol Security"
  phone: "+1-555-0102"
  email: "security@validator.com"
  pgp_key: "0x12345678"

infrastructure_provider:
  name: "AWS Support"
  account: "123456789012"
  support_level: "Enterprise"
  phone: "+1-800-AWS-HELP"
  portal: "https://console.aws.amazon.com/support"

hsm_provider:
  name: "Yubico Support"
  email: "support@yubico.com"
  account: "VALIDATOR-001"
  device_serials:
    - "9876543210"
    - "9876543211"  # backup device

chain_governance:
  discord: "https://discord.gg/DBHTc2QV"
  telegram: "https://t.me/pawvalidators"
  email: "security@paw.network"

backup_locations:
  primary: "Office fireproof safe - Combination known by Alice + Bob"
  secondary: "First National Bank - Safety deposit box #4523"
  tertiary: "Trustee Carol - Sealed envelope in home safe"

multi_sig_participants:
  - name: "Alice"
    contact: "+1-555-0100"
    key_id: "participant1"
  - name: "Bob"
    contact: "+1-555-0101"
    key_id: "participant2"
  - name: "Carol"
    contact: "+1-555-0102"
    key_id: "participant3"

escalation_procedure: |
  1. Primary operator responds within 15 minutes
  2. If no response, escalate to secondary operator
  3. If both unavailable, security lead takes control
  4. For key recovery, minimum 2 multi-sig participants required
  5. For critical decisions, convene all 3 participants

last_updated: "2025-12-06"
review_frequency: "Quarterly"
```

### Runbook: Emergency Validator Stop

```bash
#!/bin/bash
# emergency-stop.sh
# WHEN TO USE: Suspected key compromise, double-signing detected, critical infrastructure breach
# WHO CAN RUN: Primary/secondary operator only
# IMPACT: Validator stops producing blocks, enters unbonding after downtime threshold

set -e

echo "========================================="
echo "EMERGENCY VALIDATOR STOP"
echo "========================================="
echo "WARNING: This will STOP the validator immediately."
echo "The validator will miss blocks and may be slashed for downtime."
echo ""
read -p "Are you CERTAIN you want to proceed? (type YES): " confirm

if [ "$confirm" != "YES" ]; then
    echo "Aborted."
    exit 1
fi

echo ""
echo "Step 1: Stopping validator process..."
sudo systemctl stop pawd
echo "✓ pawd stopped"

echo ""
echo "Step 2: Stopping remote signer..."
sudo systemctl stop tmkms
echo "✓ tmkms stopped"

echo ""
echo "Step 3: Blocking network access..."
sudo ufw deny 26656  # P2P
sudo ufw deny 26658  # Remote signer
echo "✓ Network isolated"

echo ""
echo "Step 4: Capturing forensic data..."
INCIDENT_DIR="/backup/incident-$(date +%s)"
sudo mkdir -p $INCIDENT_DIR
sudo journalctl -u pawd --since "1 hour ago" > $INCIDENT_DIR/pawd.log
sudo journalctl -u tmkms --since "1 hour ago" > $INCIDENT_DIR/tmkms.log
sudo cp -r ~/.paw/config $INCIDENT_DIR/
sudo cp -r /etc/tmkms $INCIDENT_DIR/
sudo tar -czf $INCIDENT_DIR.tar.gz $INCIDENT_DIR
sudo chmod 400 $INCIDENT_DIR.tar.gz
echo "✓ Forensic data saved to $INCIDENT_DIR.tar.gz"

echo ""
echo "Step 5: Checking for slashing..."
pawd query slashing signing-info $(pawd tendermint show-validator | jq -r .key) || true
echo ""

echo "========================================="
echo "VALIDATOR EMERGENCY STOP COMPLETE"
echo "========================================="
echo ""
echo "NEXT STEPS:"
echo "1. Investigate root cause using forensic data"
echo "2. Determine if keys are compromised"
echo "3. Contact other operators/security team"
echo "4. Follow key rotation procedure if needed"
echo "5. DO NOT restart validator until issue resolved"
echo ""
echo "Forensic data location: $INCIDENT_DIR.tar.gz"
echo "Incident timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

### Runbook: Key Rotation (Emergency)

```bash
#!/bin/bash
# emergency-key-rotation.sh
# WHEN TO USE: Consensus key confirmed compromised
# IMPACT: Creates NEW validator with new keys, requires delegator migration

set -e

echo "========================================="
echo "EMERGENCY KEY ROTATION"
echo "========================================="
echo "WARNING: This creates a NEW validator. The old validator will be abandoned."
echo "Delegators will need to manually migrate."
echo ""
read -p "Confirm consensus key is COMPROMISED? (type COMPROMISED): " confirm

if [ "$confirm" != "COMPROMISED" ]; then
    echo "Aborted. If not compromised, use normal recovery procedures."
    exit 1
fi

echo ""
echo "Step 1: Stop old validator (if running)..."
sudo systemctl stop pawd || true
sudo systemctl stop tmkms || true

echo ""
echo "Step 2: Generate NEW consensus key on HSM..."
echo "ACTION REQUIRED: Run on air-gapped machine or directly on HSM:"
echo "  $ tmkms yubihsm keys generate 2 -b"
echo "  (Use new key slot, NOT the compromised slot 1)"
echo ""
read -p "Press ENTER when new key is generated..."

echo ""
echo "Step 3: Generate NEW operator key..."
pawd keys add validator-operator-new --ledger || pawd keys add validator-operator-new
NEW_OPERATOR=$(pawd keys show validator-operator-new -a)
echo "✓ New operator address: $NEW_OPERATOR"

echo ""
echo "Step 4: Initialize new validator node..."
pawd init "MyValidator-v2" --chain-id paw-mainnet-1

echo ""
echo "Step 5: Get new consensus public key..."
NEW_CONSENSUS_PUBKEY=$(pawd tendermint show-validator)
echo "✓ New consensus pubkey: $NEW_CONSENSUS_PUBKEY"

echo ""
echo "Step 6: Fund new operator account..."
echo "ACTION REQUIRED: Send tokens to new operator:"
echo "  Address: $NEW_OPERATOR"
echo "  Minimum: 1000000upaw + gas"
echo ""
read -p "Press ENTER when operator account is funded..."

echo ""
echo "Step 7: Create new validator..."
pawd tx staking create-validator \
  --from validator-operator-new \
  --amount 5000000upaw \
  --pubkey "$NEW_CONSENSUS_PUBKEY" \
  --moniker "MyValidator-v2" \
  --chain-id paw-mainnet-1 \
  --commission-rate 0.10 \
  --commission-max-rate 0.20 \
  --commission-max-change-rate 0.01 \
  --min-self-delegation 1000000 \
  --gas auto \
  --gas-adjustment 1.5 \
  --broadcast-mode block

echo ""
echo "Step 8: Configure tmkms for new key..."
echo "ACTION REQUIRED: Update /etc/tmkms/tmkms.toml"
echo "  [[providers.yubihsm.keys]]"
echo "  key = 2  # NEW key slot"
echo ""
read -p "Press ENTER when tmkms.toml is updated..."

echo ""
echo "Step 9: Start new validator..."
sudo systemctl start tmkms
sleep 5
sudo systemctl start pawd

echo ""
echo "Step 10: Verify signing..."
sleep 10
pawd status | jq .ValidatorInfo

echo ""
echo "========================================="
echo "EMERGENCY KEY ROTATION COMPLETE"
echo "========================================="
echo ""
echo "CRITICAL POST-ROTATION TASKS:"
echo "1. Notify all delegators via social media/email/Discord"
echo "2. Provide new validator address: [query from chain]"
echo "3. Monitor for delegator migrations"
echo "4. After 21 days, unbond from old validator"
echo "5. Investigate HOW key was compromised"
echo "6. Implement additional security controls"
echo "7. Update all documentation with new addresses"
echo "8. Report incident to chain governance (if required)"
echo ""
echo "Old operator (DO NOT USE): [previous address]"
echo "New operator: $NEW_OPERATOR"
echo "New validator: [query from chain after creation]"
```

---

## Conclusion

Validator key management is the **MOST CRITICAL** security aspect of blockchain infrastructure. A single mistake can result in:

- Permanent slashing of staked tokens
- Loss of delegator trust and reputation
- Chain halt (if major validator)
- Financial loss for delegators
- Legal liability

**Best practices summary:**

✓ Generate consensus keys on air-gapped machines or directly in HSMs
✓ NEVER share consensus keys between validators
✓ Use hardware wallets (Ledger) for operator keys
✓ Implement multi-signature for high-value validators
✓ Maintain encrypted backups in 3+ geographically distributed locations
✓ Test recovery procedures quarterly
✓ Monitor for double-signing and unauthorized transactions
✓ Rotate access credentials regularly
✓ Have documented emergency procedures
✓ Maintain 24/7 on-call coverage for mainnet validators

**Remember:** The blockchain community expects validators to operate with **PRODUCTION-GRADE security**. This documentation provides the foundation, but security is an ongoing process requiring vigilance, testing, and continuous improvement.

**NEVER compromise on key security. When in doubt, choose the more secure option.**

---

## Appendix: Quick Reference Commands

```bash
# Generate consensus key (air-gapped)
pawd init <moniker> --chain-id <chain-id>

# Generate operator key
pawd keys add validator-operator [--ledger]

# Create multi-sig operator
pawd keys add validator-multisig \
  --multisig addr1,addr2,addr3 \
  --multisig-threshold 2

# Encrypt backup
gpg --symmetric --cipher-algo AES256 priv_validator_key.json

# Import key to YubiHSM
tmkms yubihsm keys import -i priv_validator_key.json 1

# Start tmkms
tmkms start -c /etc/tmkms/tmkms.toml

# Check validator signing
pawd query slashing signing-info $(pawd tendermint show-validator | jq -r .key)

# Emergency stop
sudo systemctl stop pawd && sudo systemctl stop tmkms

# Verify backup integrity
sha256sum priv_validator_key.json.gpg
```

**Last updated:** 2025-12-06
**Review schedule:** Quarterly
**Maintainer:** PAW Blockchain Security Team
