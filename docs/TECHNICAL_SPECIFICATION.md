# PAW Blockchain Technical Specification

**Version:** 1.0.0
**Date:** 2025-11-12
**Status:** Draft

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [Smart Contract Layer](#2-smart-contract-layer)
3. [Consensus Protocol Details](#3-consensus-protocol-details)
4. [Transaction Format & Cryptography](#4-transaction-format--cryptography)
5. [State Model & Storage](#5-state-model--storage)
6. [Fee Market & Gas Mechanism](#6-fee-market--gas-mechanism)
7. [Block Structure](#7-block-structure)
8. [Network Protocol](#8-network-protocol)
9. [Appendices](#9-appendices)

---

## 1. Introduction

### 1.1 Overview

PAW is a manageable layer-1 blockchain designed for AI workload verification with a built-in DEX, secure API compute aggregation, and multi-device wallet support. This technical specification defines the core protocol architecture, consensus mechanism, transaction processing, state management, and network communication layers.

### 1.2 Design Goals

- **Security**: Cryptographically secure transaction signing, BFT consensus guarantees, and TEE-protected compute operations
- **Performance**: 4-second block times with instant finality
- **Scalability**: Efficient state storage with IAVL trees and fast sync capabilities
- **Developer Experience**: WASM-based smart contracts with rich Rust ecosystem support
- **Economic Sustainability**: Dynamic fee market with burn mechanisms and deflationary tokenomics

### 1.3 Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                         │
│  (Wallets, DEX Interface, Compute Agents, Mobile Apps)      │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                    Smart Contract Layer                      │
│         (CosmWasm VM, CW20/CW721, DEX Primitives)           │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                   Transaction Processing                     │
│        (Ed25519 Signatures, Protobuf Encoding)              │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                    Consensus Layer                           │
│              (Tendermint BFT, 4s blocks)                    │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                    State Management                          │
│          (IAVL Tree, Account Model, Merkle Proofs)          │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                    Network Layer                             │
│              (libp2p, Gossip, DHT Discovery)                │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Smart Contract Layer

### 2.1 Virtual Machine: CosmWasm

PAW utilizes **CosmWasm** as its smart contract execution environment, providing WASM-based contract execution with the following characteristics:

#### 2.1.1 CosmWasm Architecture

```
┌──────────────────────────────────────────────────┐
│              Contract Interface                   │
│  (instantiate, execute, query, migrate, sudo)    │
└──────────────────────────────────────────────────┘
                      │
┌──────────────────────────────────────────────────┐
│            WASM Runtime (Wasmer)                  │
│         - Deterministic execution                 │
│         - Memory sandboxing                       │
│         - Gas metering integration                │
└──────────────────────────────────────────────────┘
                      │
┌──────────────────────────────────────────────────┐
│              Host Functions                       │
│  (storage, crypto, query chain state)            │
└──────────────────────────────────────────────────┘
```

#### 2.1.2 Justification for CosmWasm

1. **Security**:
   - Sandboxed execution environment prevents unauthorized system access
   - Deterministic execution ensures identical results across all validators
   - Memory safety guarantees from WASM specification
   - No external network calls or file system access

2. **Portability**:
   - WASM bytecode runs on any platform with a compliant runtime
   - Platform-agnostic binary format
   - Easy migration between different blockchain implementations

3. **Rust Ecosystem**:
   - Access to robust Rust crates for cryptography, serialization, math
   - Strong type system prevents common programming errors
   - Excellent tooling (cargo, rustfmt, clippy)
   - Active developer community

4. **Performance**:
   - Near-native execution speed
   - Efficient bytecode compilation
   - Minimal runtime overhead

### 2.2 Contract Languages

**Primary Language**: Rust

Contracts must be written in Rust and compiled to WASM using the following toolchain:

```bash
# Required Rust version
rustc >= 1.75.0

# Target architecture
wasm32-unknown-unknown

# Optimization flags
RUSTFLAGS='-C link-arg=-s' cargo build --release --target wasm32-unknown-unknown
```

**Contract Size Limits**:
- Maximum compiled WASM size: 800 KB (uncompressed)
- Maximum instantiation message size: 256 KB
- Maximum execution message size: 128 KB

### 2.3 Gas Metering

Gas is charged per operation to prevent infinite loops and resource exhaustion:

#### 2.3.1 Base Operation Costs

| Operation | Gas Cost | Notes |
|-----------|----------|-------|
| WASM instruction | 1 | Per instruction executed |
| Storage write (per byte) | 100 | Persistent state storage |
| Storage read (per byte) | 10 | Loading from persistent state |
| Memory allocation (per page) | 1000 | 64 KB WASM page |
| Cryptographic hash (SHA256) | 500 | Per hash operation |
| Signature verification (Ed25519) | 3000 | Per signature |
| Event emission | 100 | Plus 10 per attribute |
| Contract instantiation | 50000 | Base overhead |
| Contract execution | 10000 | Base overhead |

#### 2.3.2 Gas Calculation Formula

```
total_gas = base_overhead +
            (instructions_executed × 1) +
            (storage_bytes_written × 100) +
            (storage_bytes_read × 10) +
            (crypto_operations × operation_cost)
```

#### 2.3.3 Gas Limits

- Maximum gas per transaction: 10,000,000 units
- Maximum gas per block: 100,000,000 units
- Minimum gas for contract call: 10,000 units

### 2.4 Standard Interfaces

#### 2.4.1 CW20 - Fungible Tokens

```rust
// CW20 Base Interface
pub enum ExecuteMsg {
    Transfer { recipient: String, amount: Uint128 },
    Burn { amount: Uint128 },
    Send { contract: String, amount: Uint128, msg: Binary },
    IncreaseAllowance { spender: String, amount: Uint128, expires: Option<Expiration> },
    DecreaseAllowance { spender: String, amount: Uint128, expires: Option<Expiration> },
    TransferFrom { owner: String, recipient: String, amount: Uint128 },
}

pub enum QueryMsg {
    Balance { address: String },
    TokenInfo {},
    Allowance { owner: String, spender: String },
}
```

**Standard Features**:
- Minting/burning capabilities
- Allowance mechanism for delegated transfers
- Metadata support (name, symbol, decimals)
- Supply tracking

#### 2.4.2 CW721 - Non-Fungible Tokens

```rust
// CW721 Base Interface
pub enum ExecuteMsg {
    TransferNft { recipient: String, token_id: String },
    SendNft { contract: String, token_id: String, msg: Binary },
    Approve { spender: String, token_id: String, expires: Option<Expiration> },
    Revoke { spender: String, token_id: String },
    ApproveAll { operator: String, expires: Option<Expiration> },
    RevokeAll { operator: String },
}

pub enum QueryMsg {
    OwnerOf { token_id: String, include_expired: Option<bool> },
    Approval { token_id: String, spender: String, include_expired: Option<bool> },
    Approvals { token_id: String, include_expired: Option<bool> },
    AllOperators { owner: String, include_expired: Option<bool>, start_after: Option<String>, limit: Option<u32> },
    NumTokens {},
    ContractInfo {},
    NftInfo { token_id: String },
    AllNftInfo { token_id: String, include_expired: Option<bool> },
    Tokens { owner: String, start_after: Option<String>, limit: Option<u32> },
    AllTokens { start_after: Option<String>, limit: Option<u32> },
}
```

**Standard Features**:
- Unique token identification
- Ownership tracking
- Operator approvals
- Metadata extension support
- Enumeration capabilities

#### 2.4.3 DEX Primitives

PAW includes native DEX primitives for automated market making and atomic swaps:

```rust
// Liquidity Pool Interface
pub enum ExecuteMsg {
    // Add liquidity to pool
    ProvideLiquidity {
        token_a_amount: Uint128,
        token_b_amount: Uint128,
        slippage_tolerance: Decimal,
    },

    // Remove liquidity from pool
    WithdrawLiquidity {
        lp_token_amount: Uint128,
        min_token_a: Uint128,
        min_token_b: Uint128,
    },

    // Swap tokens
    Swap {
        offer_asset: Asset,
        belief_price: Option<Decimal>,
        max_spread: Option<Decimal>,
        to: Option<String>,
    },

    // Atomic swap operations
    InitiateAtomicSwap {
        recipient: String,
        hash_lock: String,
        time_lock: u64,
        amount: Uint128,
    },

    ClaimAtomicSwap {
        swap_id: String,
        secret: String,
    },

    RefundAtomicSwap {
        swap_id: String,
    },
}

pub enum QueryMsg {
    // Pool information
    Pool {},

    // Simulate swap outcome
    Simulation { offer_asset: Asset },

    // Get swap price
    ReverseSimulation { ask_asset: Asset },

    // LP token info
    LpToken {},

    // Atomic swap status
    AtomicSwap { swap_id: String },
}
```

**Constant Product AMM Formula**:
```
x × y = k

where:
- x = reserve of token A
- y = reserve of token B
- k = constant product
```

**Fee Structure**:
- Swap fee: 0.3% (configurable via governance)
- 0.25% to liquidity providers
- 0.05% to protocol treasury

### 2.5 Contract Deployment Process

#### 2.5.1 Deployment Steps

1. **Code Upload**:
```protobuf
message MsgStoreCode {
  string sender = 1;
  bytes wasm_byte_code = 2;
  AccessConfig instantiate_permission = 3;
}
```

2. **Code Verification**:
   - Validate WASM format
   - Check size limits
   - Verify deterministic execution
   - Calculate code hash: `SHA256(wasm_byte_code)`

3. **Code Storage**:
   - Store WASM bytecode with unique code_id
   - Create code_info entry with creator, hash, instantiate_permission

4. **Contract Instantiation**:
```protobuf
message MsgInstantiateContract {
  string sender = 1;
  string admin = 2;
  uint64 code_id = 3;
  string label = 4;
  bytes msg = 5;
  repeated Coin funds = 6;
}
```

5. **Contract Address Generation**:
```
contract_address = bech32_encode("paw",
    SHA256(
        code_id ||
        instantiator_address ||
        creator_salt ||
        init_msg_hash
    )[0:20]
)
```

#### 2.5.2 Deployment Costs

- Code upload: 50,000 gas + (1000 × code_size_kb)
- Instantiation: 50,000 gas + execution gas
- Storage: 100 gas per byte of persistent state

### 2.6 Upgradability Model

**Default**: Immutable contracts

Contracts are immutable by default. Once deployed, code cannot be changed without explicit upgradability patterns.

**Optional Proxy Pattern**:

For upgradeable contracts, developers may implement a proxy pattern:

```rust
// Proxy Contract
pub struct ProxyContract {
    pub admin: Addr,
    pub implementation: Addr,
}

pub enum ExecuteMsg {
    // Forward all calls to implementation
    Execute { msg: Binary },

    // Admin-only upgrade
    Upgrade { new_implementation: Addr },

    // Admin transfer
    UpdateAdmin { new_admin: Addr },
}
```

**Upgrade Process**:
1. Deploy new implementation contract
2. Admin calls `Upgrade` with new contract address
3. Proxy redirects future calls to new implementation
4. State remains in proxy contract

**Security Considerations**:
- Timelocks: Upgrades require 7-day timelock period
- Governance approval: Critical contracts require DAO vote
- State migration: Implement explicit migration logic
- Event emission: All upgrades emit `ContractUpgraded` events

---

## 3. Consensus Protocol Details

### 3.1 Algorithm: Tendermint BFT

PAW uses Tendermint Byzantine Fault Tolerant consensus, a proven, battle-tested algorithm that provides:

- **Safety**: At most one block finalized per height
- **Liveness**: Network makes progress under 2/3 honest validators
- **Instant Finality**: No probabilistic finality; blocks are final immediately

#### 3.1.1 Tendermint Rounds

```
┌──────────────────────────────────────────────────────────┐
│                    New Height (H)                         │
└──────────────────────────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────┐
│  PROPOSE: Proposer broadcasts block proposal             │
│           Timeout: 3 seconds                              │
└──────────────────────────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────┐
│  PREVOTE: Validators vote on proposal                    │
│           Timeout: 1 second                               │
└──────────────────────────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────┐
│  PRECOMMIT: Validators commit if >2/3 prevotes           │
│             Timeout: 1 second                             │
└──────────────────────────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────┐
│  COMMIT: Block finalized if >2/3 precommits              │
│          Move to height H+1                               │
└──────────────────────────────────────────────────────────┘
```

#### 3.1.2 Round Advancement

If consensus not reached in round R:
1. Increment round: R → R+1
2. Double timeout: `timeout(R+1) = 2 × timeout(R)`
3. Select new proposer via round-robin + VRF seed
4. Repeat consensus steps

### 3.2 Block Time: 4 Seconds

**Target**: 4.0 seconds per block
**Tolerance**: ±0.5 seconds

**Timeout Parameters**:
```yaml
timeoutPropose: 3000ms      # Proposal broadcast window
timeoutPrevote: 1000ms      # Prevote collection window
timeoutPrecommit: 1000ms    # Precommit collection window
timeoutCommit: 0ms          # Proceed immediately on commit

# Round increment multipliers
timeoutProposeDelta: 500ms  # Added per failed round
timeoutPrevoteDelta: 500ms
timeoutPrecommitDelta: 500ms
```

### 3.3 Finality: Instant (BFT)

Tendermint provides **deterministic finality**:

- Once a block receives >2/3 precommit votes, it is **final**
- No reorganizations possible
- No probabilistic finality window
- Validators cannot equivocate without being slashed

**Finality Guarantees**:
- Byzantine tolerance: Up to 1/3 of voting power can be malicious
- Safety threshold: 2/3 + 1 voting power required for finalization
- Accountable safety: Evidence of double-signing is provable and slashable

### 3.4 Validator Set

#### 3.4.1 Size Parameters

- **Genesis validators**: 25
- **Maximum validators**: 125
- **Minimum validators**: 4 (network safety threshold)

**Expansion Schedule**:
```
Epoch 0-180 days:   25 validators
Epoch 180-365 days: 50 validators (governance vote required)
Epoch 365+ days:    Up to 125 validators (gradual expansion)
```

#### 3.4.2 Validator Selection

Validators are selected based on **stake-weighted voting power**:

```
voting_power(validator) = min(
    staked_amount,
    max_validator_stake
)

max_validator_stake = total_staked / (num_validators × 2)
```

This prevents single-validator dominance while maintaining stake-weighting.

**Validator Requirements**:
- Minimum stake: 100,000 PAW
- Maximum stake: Capped at 2× average validator stake
- Minimum uptime: 95% over trailing 30-day window
- Maximum missed blocks: 500 consecutive blocks

#### 3.4.3 Voting Power Distribution

```
Total Voting Power = Σ(validator_stake for all active validators)

Validator Voting Power % = (validator_stake / total_voting_power) × 100
```

**Anti-Centralization Measures**:
- No single validator can hold >15% voting power
- Top 5 validators cannot collectively hold >50% voting power
- Delegation caps: Max 20% of network stake to single validator

### 3.5 Block Proposal

#### 3.5.1 Proposer Selection

**Algorithm**: Deterministic round-robin with VRF-based seed

```
proposer_index = (previous_proposer_index + 1 + vrf_offset) mod num_validators

where:
vrf_offset = VRF(block_hash(H-1) || round) mod num_validators
```

**VRF (Verifiable Random Function)**:
- Input: Previous block hash concatenated with round number
- Output: Deterministic but unpredictable offset
- Verification: All validators can verify proposer legitimacy

#### 3.5.2 Proposal Structure

```protobuf
message Proposal {
  int64 height = 1;
  int32 round = 2;
  int32 pol_round = 3;  // Proof-of-Lock round
  bytes block_id = 4;
  google.protobuf.Timestamp timestamp = 5;
  bytes signature = 6;
}
```

**Proposal Validation**:
1. Verify proposer signature
2. Check proposer is correct for (height, round)
3. Validate block structure
4. Verify state transition validity
5. Check timestamp is within acceptable bounds

### 3.6 Timeout Parameters

```yaml
consensus_params:
  timeout_propose: 3000ms
  timeout_propose_delta: 500ms
  timeout_prevote: 1000ms
  timeout_prevote_delta: 500ms
  timeout_precommit: 1000ms
  timeout_precommit_delta: 500ms
  timeout_commit: 0ms

  # Skip empty blocks after timeout
  skip_timeout_commit: false

  # Create empty blocks
  create_empty_blocks: true
  create_empty_blocks_interval: 60s
```

**Timeout Escalation**:

For round R:
```
timeout_propose(R) = timeout_propose + (R × timeout_propose_delta)
timeout_prevote(R) = timeout_prevote + (R × timeout_prevote_delta)
timeout_precommit(R) = timeout_precommit + (R × timeout_precommit_delta)
```

### 3.7 Fork Choice Rule

**Canonical Chain Selection**:

```
canonical_chain = longest_chain with:
  1. Most validator signatures (highest cumulative voting power)
  2. Highest block height
  3. Lexicographically smallest block hash (tiebreaker)
```

**Fork Resolution**:

Since Tendermint provides instant finality, forks are impossible under normal operation. However, during:

- **Network partition**: Chain halts if <2/3 validators online
- **Recovery**: Validators sync to highest finalized block with valid +2/3 precommit proof

**Safety Mechanism**:
```
If validator sees conflicting blocks at same height:
  1. Refuse to vote on either
  2. Submit double-sign evidence
  3. Await social consensus or governance intervention
```

No automatic fork choice; the protocol prevents forks cryptographically.

---

## 4. Transaction Format & Cryptography

### 4.1 Signature Scheme: Ed25519

**Algorithm**: Edwards-curve Digital Signature Algorithm (EdDSA) using Curve25519

**Key Characteristics**:
- **Security**: 128-bit security level
- **Performance**: ~71,000 signatures/sec, ~20,000 verifications/sec (single core)
- **Key Size**: 32 bytes (public key), 32 bytes (private key)
- **Signature Size**: 64 bytes
- **Deterministic**: Same message always produces same signature (RFC 8032)

#### 4.1.1 Key Generation

```
Private Key (sk):  32 random bytes from secure CSPRNG
Public Key (pk):   edwards25519_point_multiply(sk, base_point)
```

**Derivation Path** (BIP44-compatible):
```
m / 44' / 118' / 0' / 0 / address_index

where:
- 44' = BIP44 purpose
- 118' = Cosmos coin type
- 0' = account 0
- 0 = external chain
- address_index = sequential address index
```

#### 4.1.2 Signing Process

```python
def sign_transaction(message: bytes, private_key: bytes) -> bytes:
    """
    Sign transaction using Ed25519

    Returns: 64-byte signature
    """
    # Hash message for signing
    message_hash = sha512(message)

    # Generate deterministic nonce
    r = sha512(private_key[32:64] || message)
    R = scalar_multiply(r, base_point)

    # Compute signature scalar
    k = sha512(R || public_key || message)
    s = (r + k * private_key) mod L

    return R || s  # 64 bytes
```

#### 4.1.3 Verification Process

```python
def verify_signature(message: bytes, signature: bytes, public_key: bytes) -> bool:
    """
    Verify Ed25519 signature

    Returns: True if valid, False otherwise
    """
    R = signature[0:32]
    s = signature[32:64]

    k = sha512(R || public_key || message)

    # Check: s*B = R + k*A (where A is public key, B is base point)
    return scalar_multiply(s, base_point) == R + scalar_multiply(k, public_key)
```

### 4.2 Account Format

#### 4.2.1 Address Generation

**Encoding**: Bech32 with "paw" prefix

```
1. Generate public key (33 bytes compressed)
2. Hash: SHA256(public_key)
3. Take first 20 bytes: address_bytes = hash[0:20]
4. Encode: bech32_encode("paw", address_bytes)

Example: paw1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu
```

**Address Components**:
- **Human-readable part (HRP)**: "paw"
- **Separator**: "1"
- **Data part**: Bech32-encoded 20-byte address
- **Checksum**: 6-character Bech32 checksum

#### 4.2.2 Address Validation

```python
def validate_address(address: str) -> bool:
    """Validate PAW address format"""
    # Check prefix
    if not address.startswith("paw1"):
        return False

    # Decode bech32
    hrp, data = bech32_decode(address)

    # Verify HRP
    if hrp != "paw":
        return False

    # Verify data length (20 bytes + 6 byte checksum)
    if len(data) != 20:
        return False

    return True
```

### 4.3 Serialization: Protocol Buffers (Protobuf)

#### 4.3.1 Protobuf Schema

```protobuf
syntax = "proto3";

package paw.tx.v1;

// Transaction wrapper
message Tx {
  TxBody body = 1;
  AuthInfo auth_info = 2;
  repeated bytes signatures = 3;
}

// Transaction body
message TxBody {
  repeated google.protobuf.Any messages = 1;
  string memo = 2;
  uint64 timeout_height = 3;
  repeated google.protobuf.Any extension_options = 1023;
  repeated google.protobuf.Any non_critical_extension_options = 2047;
}

// Authentication info
message AuthInfo {
  repeated SignerInfo signer_infos = 1;
  Fee fee = 2;
  Tip tip = 3;
}

// Signer information
message SignerInfo {
  google.protobuf.Any public_key = 1;
  ModeInfo mode_info = 2;
  uint64 sequence = 3;
}

// Fee structure
message Fee {
  repeated cosmos.base.v1beta1.Coin amount = 1;
  uint64 gas_limit = 2;
  string payer = 3;
  string granter = 4;
}

// Signing mode
message ModeInfo {
  oneof sum {
    Single single = 1;
    Multi multi = 2;
  }

  message Single {
    SignMode mode = 1;
  }

  message Multi {
    cosmos.crypto.multisig.v1beta1.CompactBitArray bitarray = 1;
    repeated ModeInfo mode_infos = 2;
  }
}
```

#### 4.3.2 Serialization Process

```
1. Create TxBody with messages
2. Serialize TxBody → sign_bytes
3. Sign sign_bytes with Ed25519
4. Create AuthInfo with signatures
5. Assemble final Tx
6. Serialize to bytes for broadcast
```

### 4.4 Transaction Types

#### 4.4.1 Transfer

```protobuf
message MsgSend {
  string from_address = 1;
  string to_address = 2;
  repeated cosmos.base.v1beta1.Coin amount = 3;
}

// Example
{
  "from_address": "paw1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu",
  "to_address": "paw1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu",
  "amount": [
    {
      "denom": "upaw",
      "amount": "1000000"  // 1 PAW (1e6 micro-PAW)
    }
  ]
}
```

#### 4.4.2 Stake

```protobuf
message MsgDelegate {
  string delegator_address = 1;
  string validator_address = 2;
  cosmos.base.v1beta1.Coin amount = 3;
}

// Example
{
  "delegator_address": "paw1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu",
  "validator_address": "pawvaloper1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu",
  "amount": {
    "denom": "upaw",
    "amount": "100000000000"  // 100,000 PAW minimum stake
  }
}
```

#### 4.4.3 Unstake

```protobuf
message MsgUndelegate {
  string delegator_address = 1;
  string validator_address = 2;
  cosmos.base.v1beta1.Coin amount = 3;
}
```

**Unbonding Period**: 21 days (configurable via governance)

#### 4.4.4 Vote

```protobuf
message MsgVote {
  uint64 proposal_id = 1;
  string voter = 2;
  VoteOption option = 3;
}

enum VoteOption {
  VOTE_OPTION_UNSPECIFIED = 0;
  VOTE_OPTION_YES = 1;
  VOTE_OPTION_ABSTAIN = 2;
  VOTE_OPTION_NO = 3;
  VOTE_OPTION_NO_WITH_VETO = 4;
}
```

#### 4.4.5 ContractCall

```protobuf
message MsgExecuteContract {
  string sender = 1;
  string contract = 2;
  bytes msg = 3;
  repeated cosmos.base.v1beta1.Coin funds = 4;
}

// msg is JSON-encoded contract-specific message
```

#### 4.4.6 ContractDeploy

```protobuf
message MsgStoreCode {
  string sender = 1;
  bytes wasm_byte_code = 2;
  AccessConfig instantiate_permission = 3;
}

message MsgInstantiateContract {
  string sender = 1;
  string admin = 2;
  uint64 code_id = 3;
  string label = 4;
  bytes msg = 5;
  repeated cosmos.base.v1beta1.Coin funds = 6;
}
```

### 4.5 Transaction Structure

```protobuf
message Tx {
  // Transaction body
  TxBody body = 1 {
    messages: [
      // One or more messages (Transfer, Stake, etc.)
      {
        "@type": "/paw.bank.v1.MsgSend",
        "from_address": "paw1...",
        "to_address": "paw1...",
        "amount": [{"denom": "upaw", "amount": "1000000"}]
      }
    ],
    memo: "Payment for services",
    timeout_height: 0,  // 0 = no timeout
  },

  // Authentication info
  AuthInfo auth_info = 2 {
    signer_infos: [
      {
        public_key: {
          "@type": "/cosmos.crypto.ed25519.PubKey",
          "key": "A+8f7y..."
        },
        mode_info: {
          single: {
            mode: SIGN_MODE_DIRECT
          }
        },
        sequence: 42  // Account sequence number
      }
    ],
    fee: {
      amount: [{"denom": "upaw", "amount": "5000"}],
      gas_limit: 200000,
      payer: "",
      granter: ""
    }
  },

  // Signatures (one per signer)
  signatures: [
    "7gH4K9..."  // Base64-encoded 64-byte Ed25519 signature
  ]
}
```

### 4.6 Nonce/Sequence Management

**Per-Account Sequence Counter**:

Each account maintains a monotonically increasing sequence number:

```
account = {
  address: "paw1...",
  sequence: 0,       // Starts at 0
  account_number: 123,
  pub_key: "...",
  balance: [...]
}
```

**Transaction Sequence Rules**:
1. Each transaction must include `sequence = current_account_sequence`
2. After successful execution, `account_sequence++`
3. Transactions with incorrect sequence are rejected
4. Prevents replay attacks and ensures ordering

**Sequence Handling**:
```python
def validate_sequence(tx: Tx, account: Account) -> bool:
    """Validate transaction sequence"""
    if tx.auth_info.signer_infos[0].sequence != account.sequence:
        return False
    return True

def increment_sequence(account: Account):
    """Increment sequence after successful tx"""
    account.sequence += 1
```

**Mempool Ordering**:
- Transactions in mempool ordered by (account, sequence)
- Missing sequences cause dependent txs to wait
- Maximum gap: 1000 sequences

### 4.7 Multi-Signature

#### 4.7.1 Threshold Signatures

PAW supports M-of-N threshold signatures using Ed25519:

```protobuf
message MultiSignature {
  repeated bytes signatures = 1;
}

message CompactBitArray {
  uint32 extra_bits_stored = 1;
  bytes elems = 2;
}

message LegacyAminoPubKey {
  uint32 threshold = 1;
  repeated google.protobuf.Any public_keys = 2;
}
```

#### 4.7.2 Multi-Sig Account Creation

```
1. Define participants: [pubkey1, pubkey2, pubkey3]
2. Set threshold: M = 2 (2-of-3)
3. Generate multi-sig pubkey:
   multisig_pubkey = create_multisig(threshold, [pubkey1, pubkey2, pubkey3])
4. Derive address:
   multisig_address = bech32_encode("paw", sha256(multisig_pubkey)[0:20])
```

#### 4.7.3 Multi-Sig Transaction Signing

```
1. Create transaction with multi-sig as signer
2. Participant 1 signs: sig1 = sign(tx, privkey1)
3. Participant 2 signs: sig2 = sign(tx, privkey2)
4. Combine signatures:
   multisig = MultiSignature {
     signatures: [sig1, sig2],
     bitarray: [1, 1, 0]  // which keys signed
   }
5. Broadcast transaction
```

**Validation**:
```python
def verify_multisig(tx: Tx, multisig_pubkey: MultiSig, signatures: MultiSignature) -> bool:
    """Verify multi-signature transaction"""
    signed_count = 0

    for i, sig in enumerate(signatures.signatures):
        if signatures.bitarray[i]:
            pubkey = multisig_pubkey.public_keys[i]
            if verify_signature(tx.sign_bytes, sig, pubkey):
                signed_count += 1

    return signed_count >= multisig_pubkey.threshold
```

---

## 5. State Model & Storage

### 5.1 Account Model

PAW uses an **account-based model** (not UTXO).

#### 5.1.1 Account Structure

```go
type BaseAccount struct {
    Address       string    // Bech32 address
    PubKey        *Any      // Public key (optional, set on first tx)
    AccountNumber uint64    // Unique account identifier
    Sequence      uint64    // Nonce for replay protection
}

type Account struct {
    BaseAccount
    Balances      []Coin    // Token balances
    Code          []byte    // Contract code (if contract account)
    Storage       KVStore   // Contract storage (if contract account)
}
```

**Account Types**:

1. **Externally Owned Account (EOA)**:
   - Controlled by private key
   - Can send transactions
   - No associated code

2. **Contract Account**:
   - Created by deploying contract
   - Has associated WASM code
   - Has persistent storage
   - Executed by transactions or other contracts

#### 5.1.2 State Representation

```
Global State = {
    "accounts": {
        "paw1abc...": {
            "account_number": 0,
            "sequence": 5,
            "pub_key": "A1B2C3...",
            "balances": [
                {"denom": "upaw", "amount": "1000000000"}
            ]
        },
        "paw1xyz...": { ... }
    },

    "validators": {
        "pawvaloper1abc...": {
            "operator_address": "pawvaloper1abc...",
            "consensus_pubkey": "...",
            "jailed": false,
            "status": "BONDED",
            "tokens": "100000000000",
            "delegator_shares": "100000000000",
            "commission": { ... }
        }
    },

    "contracts": {
        "paw1contract...": {
            "code_id": 1,
            "creator": "paw1abc...",
            "admin": "paw1abc...",
            "label": "My DEX Pool",
            "created": 12345
        }
    },

    "contract_storage": {
        "paw1contract...": {
            "key1": "value1",
            "key2": "value2"
        }
    }
}
```

### 5.2 Storage: IAVL Tree

**IAVL** = Immutable AVL tree (from Cosmos SDK)

#### 5.2.1 IAVL Tree Properties

- **Self-balancing**: O(log n) operations
- **Immutable**: Each update creates new version
- **Merkle tree**: Every node has hash of children
- **Versioned**: Supports historical state queries
- **Prunable**: Old versions can be deleted

#### 5.2.2 IAVL Node Structure

```go
type Node struct {
    key       []byte
    value     []byte
    version   int64
    height    int8
    size      int64
    hash      []byte

    leftHash  []byte
    leftNode  *Node

    rightHash []byte
    rightNode *Node
}
```

**Hash Calculation**:
```
node.hash = SHA256(
    node.height ||
    node.size ||
    node.version ||
    node.key ||
    node.value ||
    node.leftHash ||
    node.rightHash
)
```

#### 5.2.3 Tree Operations

**Insertion**:
```
1. Find insertion point (O(log n))
2. Create new node with value
3. Rebalance tree (AVL rotations)
4. Update hashes along path to root
5. Increment version
```

**Query**:
```
1. Traverse tree from root (O(log n))
2. Compare keys at each node
3. Return value if found, nil otherwise
```

**Merkle Proof**:
```
1. Find target key
2. Collect sibling hashes along path to root
3. Return proof = [(sibling_hash, is_left), ...]
4. Verifier can reconstruct root hash
```

### 5.3 State Commitment

#### 5.3.1 Merkle Root in Block Header

Each block header contains the **AppHash** (application state root):

```go
type Header struct {
    // ... other fields
    AppHash     []byte    // IAVL tree root hash
    // ...
}
```

**Computation**:
```
app_hash = root_node.hash

where root_node is the root of the IAVL tree after applying block N
```

#### 5.3.2 Multi-Store Architecture

PAW uses multiple IAVL trees for different modules:

```
CommitMultiStore
├── BankStore (IAVL)      // Account balances
├── StakingStore (IAVL)   // Validator/delegation state
├── WasmStore (IAVL)      // Contract code/storage
├── GovStore (IAVL)       // Governance proposals
├── ParamsStore (IAVL)    // Chain parameters
└── ... other stores
```

**Multi-Store Root Hash**:
```
multi_store_hash = SimpleMerkleRoot([
    sha256("bank" || bank_store.root_hash),
    sha256("staking" || staking_store.root_hash),
    sha256("wasm" || wasm_store.root_hash),
    ...
])
```

### 5.4 Storage Rent

**Phase 1**: No storage rent

Storage is free (paid via gas during writes). This simplifies initial launch.

**Future Phases** (governance decision):
- Periodic rent charged per byte of storage
- Accounts must maintain minimum balance
- Unpaid rent → state pruning or archival

### 5.5 Pruning

#### 5.5.1 Default Pruning Strategy

**Keep Last**: 100,000 blocks (configurable)

```yaml
pruning:
  strategy: "custom"
  keep_recent: 100000
  keep_every: 10000  # Keep every 10,000th block (archival)
  interval: 10       # Prune every 10 blocks
```

**Pruning Strategies**:

1. **Nothing**: Keep all historical states (archive node)
   ```yaml
   pruning: "nothing"
   ```

2. **Everything**: Keep only current state (light node)
   ```yaml
   pruning: "everything"
   ```

3. **Default**: Keep last 100,000 blocks
   ```yaml
   pruning: "default"
   ```

4. **Custom**: User-defined
   ```yaml
   pruning: "custom"
   keep_recent: 100000
   keep_every: 10000
   interval: 10
   ```

#### 5.5.2 Pruning Process

```
Every 10 blocks:
  current_height = blockchain.height
  prune_height = current_height - keep_recent

  if prune_height > 0 and prune_height % keep_every != 0:
    iavl_tree.DeleteVersion(prune_height)
```

### 5.6 State Sync

#### 5.6.1 Fast Sync Protocol

**Tendermint State Sync** allows new nodes to sync quickly:

```
1. Request snapshot from peers
2. Download state snapshot at height H
3. Verify snapshot against block header AppHash
4. Apply snapshot to local store
5. Resume normal block sync from H+1
```

**Snapshot Structure**:
```protobuf
message Snapshot {
  uint64 height = 1;
  uint32 format = 2;
  uint32 chunks = 3;
  bytes hash = 4;
  bytes metadata = 5;
}

message SnapshotChunk {
  uint64 height = 1;
  uint32 format = 2;
  uint32 chunk = 3;
  bytes data = 4;
}
```

#### 5.6.2 State Sync Configuration

```yaml
statesync:
  enable: true
  rpc_servers: "https://rpc1.paw.network:443,https://rpc2.paw.network:443"
  trust_height: 12345
  trust_hash: "ABC123..."
  trust_period: "168h"  # 7 days

  # Snapshot configuration
  snapshot_interval: 1000  # Take snapshot every 1000 blocks
  snapshot_keep_recent: 2  # Keep last 2 snapshots
```

#### 5.6.3 Sync Process

```
1. Node starts with no state
2. Connects to RPC servers
3. Fetches light client state at trust_height
4. Requests available snapshots
5. Downloads snapshot chunks in parallel
6. Verifies each chunk against snapshot hash
7. Applies snapshot to local store
8. Verifies final state root matches AppHash
9. Resumes block sync from snapshot height
```

**Sync Time Estimate**:
- Traditional sync: ~7 days for 1M blocks
- State sync: ~30 minutes for same height

---

## 6. Fee Market & Gas Mechanism

### 6.1 Gas Model

#### 6.1.1 Per-Operation Gas Units

Gas is consumed for every operation that uses computational or storage resources:

| Operation Category | Operation | Gas Cost |
|-------------------|-----------|----------|
| **Storage** | Write 1 byte | 100 |
| | Read 1 byte | 10 |
| | Delete 1 byte | 50 |
| **Crypto** | SHA256 hash | 500 |
| | Ed25519 verify | 3,000 |
| | Secp256k1 verify | 5,000 |
| **Bank** | Send tokens | 10,000 |
| | Multi-send | 10,000 + (5,000 × outputs) |
| **Staking** | Delegate | 20,000 |
| | Undelegate | 20,000 |
| | Redelegate | 25,000 |
| **Contract** | Instantiate | 50,000 + execution |
| | Execute | 10,000 + execution |
| | Query | 1,000 + execution |
| **Governance** | Submit proposal | 50,000 |
| | Vote | 5,000 |

#### 6.1.2 WASM Execution Costs

| WASM Operation | Gas Multiplier |
|----------------|----------------|
| Basic instruction | 1 |
| Memory load | 2 |
| Memory store | 3 |
| Function call | 10 |
| Host function call | 100 |

**Example Calculation**:
```
Contract execution gas =
    10,000 (base) +
    (instructions_executed × 1) +
    (memory_ops × 2-3) +
    (host_calls × 100) +
    storage_gas
```

### 6.2 Fee Calculation

#### 6.2.1 Formula

```
total_fee = gas_used × gas_price

where:
- gas_used: Actual gas consumed during execution
- gas_price: User-specified price per gas unit (in upaw)
```

**Fee Specification**:
```protobuf
message Fee {
  repeated Coin amount = 1;      // Total fee amount
  uint64 gas_limit = 2;           // Maximum gas allowed
  string payer = 3;               // Fee payer (optional)
  string granter = 4;             // Fee granter (optional)
}
```

**Effective Gas Price**:
```
gas_price = total_fee_amount / gas_limit

Example:
  fee_amount = 5000 upaw
  gas_limit = 200,000
  gas_price = 5000 / 200,000 = 0.025 upaw/gas
```

#### 6.2.2 Gas Limit vs Gas Used

- **Gas Limit**: Maximum gas transaction can consume (user-specified)
- **Gas Used**: Actual gas consumed during execution
- **Refund**: `(gas_limit - gas_used) × gas_price` returned to sender

**Validation**:
```
if gas_used > gas_limit:
    // Transaction fails, fee still charged
    charge_fee(gas_limit × gas_price)
    revert_state_changes()
else:
    // Transaction succeeds
    charge_fee(gas_used × gas_price)
    refund((gas_limit - gas_used) × gas_price)
```

### 6.3 Dynamic Fees (EIP-1559-style)

#### 6.3.1 Base Fee + Priority Tip

```
total_fee = (base_fee + priority_tip) × gas_used

where:
- base_fee: Protocol-determined minimum (burned)
- priority_tip: User-specified extra (to validators)
```

**Fee Structure**:
```go
type DynamicFee struct {
    BaseFee      sdk.Dec   // Minimum fee (burned)
    PriorityTip  sdk.Dec   // Extra tip (to validators)
    GasLimit     uint64
}
```

#### 6.3.2 Base Fee Adjustment

Base fee adjusts based on block fullness:

```
target_gas_per_block = max_gas_per_block / 2
actual_gas_used = gas_used_in_previous_block

if actual_gas_used > target_gas_per_block:
    // Blocks too full, increase base fee
    base_fee_next = base_fee_current × 1.125
elif actual_gas_used < target_gas_per_block:
    // Blocks underutilized, decrease base fee
    base_fee_next = base_fee_current × 0.875
else:
    // On target, no change
    base_fee_next = base_fee_current

# Cap adjustments
max_change_denominator = 8
base_fee_delta = abs(base_fee_next - base_fee_current)
max_delta = base_fee_current / max_change_denominator

if base_fee_delta > max_delta:
    base_fee_delta = max_delta
```

**Parameters**:
```yaml
fee_market:
  target_block_utilization: 0.5      # 50% target
  max_base_fee_change: 12.5%         # Per block
  min_base_fee: 0.001 upaw/gas
  max_base_fee: 1000 upaw/gas
```

### 6.4 Minimum Gas Price

**Global Minimum**: 0.001 upaw per gas unit

```
min_gas_price = 0.001 upaw/gas

Transaction valid only if:
  user_gas_price >= max(min_gas_price, current_base_fee)
```

**Validator Configuration**:
```yaml
# In validator config
minimum-gas-prices: "0.001upaw"

# Validators can set higher minimums
minimum-gas-prices: "0.01upaw"  # This validator rejects lower fees
```

### 6.5 Fee Distribution

**Split**: 50% burned / 30% validators / 20% treasury

```go
func DistributeFees(totalFee sdk.Coins) {
    burnAmount := totalFee.MulDec(0.50)
    validatorAmount := totalFee.MulDec(0.30)
    treasuryAmount := totalFee.MulDec(0.20)

    // Burn tokens (deflationary)
    BurnCoins(burnAmount)

    // Distribute to validators proportionally by voting power
    DistributeToValidators(validatorAmount)

    // Send to community treasury
    SendToTreasury(treasuryAmount)
}
```

**Per-Validator Distribution**:
```
validator_fee_share = (validator_voting_power / total_voting_power) × validator_pool

Then:
  commission = validator_fee_share × commission_rate
  delegator_pool = validator_fee_share - commission

  per_delegator_share = delegator_pool × (delegator_stake / total_delegated_to_validator)
```

### 6.6 Contract Execution Costs Table

| Contract Operation | Gas Cost | Notes |
|-------------------|----------|-------|
| **Storage Operations** | | |
| Store 32 bytes (new key) | 20,000 | First write to key |
| Store 32 bytes (existing key) | 5,000 | Overwrite existing |
| Load 32 bytes | 1,000 | Read from storage |
| Remove key | 3,000 | Delete from storage |
| **Token Operations** | | |
| CW20 transfer | 25,000 | Base cost |
| CW20 approve | 20,000 | Set allowance |
| CW20 transfer_from | 30,000 | Delegated transfer |
| CW20 mint | 25,000 | Requires minter role |
| CW20 burn | 20,000 | Destroy tokens |
| **NFT Operations** | | |
| CW721 mint | 30,000 | Create NFT |
| CW721 transfer | 25,000 | Transfer ownership |
| CW721 approve | 20,000 | Set operator |
| CW721 burn | 20,000 | Destroy NFT |
| **DEX Operations** | | |
| Provide liquidity | 150,000 | Add to pool |
| Withdraw liquidity | 100,000 | Remove from pool |
| Swap | 120,000 | Execute trade |
| Initiate atomic swap | 80,000 | Create HTLC |
| Claim atomic swap | 50,000 | Reveal secret |
| Refund atomic swap | 40,000 | Timeout refund |
| **Query Operations** | | |
| Query balance | 1,000 | Read-only |
| Query pool info | 2,000 | Read-only |
| Simulate swap | 5,000 | Calculation |

**Example DEX Swap Gas Calculation**:
```
Swap 1000 PAW for USDC:

Base execution cost:     10,000 gas
Swap logic:             120,000 gas
Storage updates (2×):    10,000 gas
Event emissions (3×):       300 gas
Pool state read:          2,000 gas
Balance updates:         10,000 gas
─────────────────────────────────
Total:                  152,300 gas

Fee (at 0.001 upaw/gas): 152.3 upaw ≈ 0.0001523 PAW
```

---

## 7. Block Structure

### 7.1 Block Header Format

```protobuf
message Header {
  // Basic block metadata
  cosmos.base.tendermint.v1beta1.Version version = 1;
  string chain_id = 2;
  int64 height = 3;
  google.protobuf.Timestamp time = 4;

  // Previous block info
  BlockID last_block_id = 5;

  // Hashes
  bytes last_commit_hash = 6;    // Previous block commit
  bytes data_hash = 7;            // Merkle root of transactions
  bytes validators_hash = 8;      // Hash of current validator set
  bytes next_validators_hash = 9; // Hash of next validator set
  bytes consensus_hash = 10;      // Consensus params hash
  bytes app_hash = 11;            // Application state root (IAVL)
  bytes last_results_hash = 12;   // ABCIResults from last block

  // Consensus info
  bytes evidence_hash = 13;       // Evidence of misbehavior
  bytes proposer_address = 14;    // Block proposer
}
```

#### 7.1.1 Header Field Details

**Block Hashes**:

```go
// Block ID (uniquely identifies block)
type BlockID struct {
    Hash          []byte  // SHA256 of block header
    PartSetHeader PartSetHeader
}

// Data hash (Merkle root of transactions)
data_hash = SimpleMerkleRoot(transactions)

// Validators hash
validators_hash = SimpleMerkleRoot([
    SHA256(validator1.pubkey || validator1.voting_power),
    SHA256(validator2.pubkey || validator2.voting_power),
    ...
])

// App hash (state root from IAVL)
app_hash = iavl_tree.RootHash()

// Last commit hash
last_commit_hash = SimpleMerkleRoot(commit_signatures)
```

**Block Header Hash**:
```
header_hash = SHA256(
    version ||
    chain_id ||
    height ||
    time ||
    last_block_id ||
    last_commit_hash ||
    data_hash ||
    validators_hash ||
    next_validators_hash ||
    consensus_hash ||
    app_hash ||
    last_results_hash ||
    evidence_hash ||
    proposer_address
)
```

### 7.2 Block Body

```protobuf
message Block {
  Header header = 1;
  Data data = 2;
  EvidenceList evidence = 3;
  Commit last_commit = 4;
}

message Data {
  repeated bytes txs = 1;  // Serialized transactions
}

message EvidenceList {
  repeated Evidence evidence = 1;
}
```

#### 7.2.1 Transactions

```
Block contains ordered list of transactions:
[
  tx1: 0x12ab34cd...  // Protobuf-encoded Tx
  tx2: 0x56ef78ab...
  tx3: 0x90cd12ef...
  ...
]
```

**Transaction Ordering**:
1. Priority: Higher gas price = higher priority
2. Sequence: Transactions from same account ordered by sequence
3. First-seen: Among equal priority, first-seen first

#### 7.2.2 Evidence

Evidence of validator misbehavior:

```protobuf
message DuplicateVoteEvidence {
  Vote vote_a = 1;
  Vote vote_b = 2;
  int64 total_voting_power = 3;
  int64 validator_power = 4;
  google.protobuf.Timestamp timestamp = 5;
}

message LightClientAttackEvidence {
  ConflictingBlock conflicting_block = 1;
  int64 common_height = 2;
  repeated Validator byzantine_validators = 3;
  int64 total_voting_power = 4;
  google.protobuf.Timestamp timestamp = 5;
}
```

**Slashing Conditions**:
- Double signing: Validator signs two different blocks at same height
- Light client attack: Validator signs conflicting headers
- Downtime: Validator misses >500 consecutive blocks

**Penalties**:
- Double sign: 5% stake slashed + permanent jail
- Light client attack: 20% stake slashed + permanent jail
- Downtime: 0.01% stake slashed + temporary jail (unjail after 10 minutes)

### 7.3 Maximum Block Size

**Limit**: 2 MB (2,097,152 bytes)

```yaml
consensus_params:
  block:
    max_bytes: 2097152        # 2 MB
    max_gas: 100000000        # 100M gas units
    time_iota_ms: 1000        # 1 second precision
```

**Size Breakdown**:
```
Block size =
    header_size +
    transactions_size +
    evidence_size +
    commit_signatures_size

Typical breakdown:
  Header: ~200 bytes
  Transactions: ~1.9 MB (variable)
  Evidence: ~100 bytes (usually empty)
  Commit: ~10 KB (25 validators × ~400 bytes)
```

**Transaction Capacity**:
```
Average transaction size: 300 bytes
Max transactions per block: ~6,500 txs

High-value transactions: 1 KB
Max transactions per block: ~1,900 txs
```

### 7.4 Maximum Gas Per Block

**Limit**: 100,000,000 gas units

```go
const MaxGasPerBlock = 100_000_000

func ValidateBlock(block *Block) error {
    totalGas := 0
    for _, tx := range block.Transactions {
        totalGas += tx.GasUsed
    }

    if totalGas > MaxGasPerBlock {
        return ErrBlockGasLimitExceeded
    }

    return nil
}
```

**Throughput Calculation**:

```
Simple transfers (10,000 gas each):
  100,000,000 / 10,000 = 10,000 transfers/block
  10,000 txs/block ÷ 4 sec/block = 2,500 TPS

Complex contract calls (150,000 gas each):
  100,000,000 / 150,000 = 666 calls/block
  666 txs/block ÷ 4 sec/block = 166 TPS

Mixed workload (avg 50,000 gas):
  100,000,000 / 50,000 = 2,000 txs/block
  2,000 txs/block ÷ 4 sec/block = 500 TPS
```

**Gas Limit Governance**:

The max gas per block is governable:

```protobuf
message ConsensusParams {
  BlockParams block = 1;
  EvidenceParams evidence = 2;
  ValidatorParams validator = 3;
  VersionParams version = 4;
}

message BlockParams {
  int64 max_bytes = 1;
  int64 max_gas = 2;     // Governable parameter
}
```

Governance proposals can adjust `max_gas` to scale throughput.

---

## 8. Network Protocol

### 8.1 P2P: libp2p

PAW uses **libp2p** for peer-to-peer networking.

#### 8.1.1 libp2p Stack

```
┌─────────────────────────────────────┐
│        Application Layer            │
│  (Tendermint Consensus, Mempool)    │
└─────────────────────────────────────┘
              │
┌─────────────────────────────────────┐
│         Protocol Layer              │
│  - GossipSub (message propagation)  │
│  - Request/Response (state sync)    │
│  - DHT (peer discovery)             │
└─────────────────────────────────────┘
              │
┌─────────────────────────────────────┐
│       Transport Layer               │
│  - TCP, QUIC                        │
│  - Multiplexing (mplex, yamux)      │
│  - Encryption (Noise, TLS)          │
└─────────────────────────────────────┘
```

#### 8.1.2 Peer Identity

Each node has a unique **PeerID**:

```
1. Generate Ed25519 keypair
2. PeerID = Multihash(public_key)
3. Multiaddr = /ip4/192.168.1.1/tcp/26656/p2p/PeerID

Example:
  /ip4/192.168.1.1/tcp/26656/p2p/12D3KooWGzxzKZYveHXtpG6AsrUJBcWxHBFS2HsEoGTxrMLvKXtf
```

#### 8.1.3 Protocol IDs

```
/paw/consensus/1.0.0    # Consensus messages
/paw/mempool/1.0.0      # Transaction gossip
/paw/blocksync/1.0.0    # Block synchronization
/paw/statesync/1.0.0    # State synchronization
/paw/evidence/1.0.0     # Evidence propagation
```

### 8.2 Peer Discovery

#### 8.2.1 Discovery Methods

**1. Seed Nodes** (bootstrap):
```yaml
seeds:
  - "seed1.paw.network:26656"
  - "seed2.paw.network:26656"
  - "seed3.paw.network:26656"
```

**2. DHT (Distributed Hash Table)**:

```
1. Node joins DHT
2. Periodically queries DHT for peers
3. DHT returns closest peers by XOR distance
4. Node connects to discovered peers
```

**3. Persistent Peers**:
```yaml
persistent_peers:
  - "node1@192.168.1.1:26656"
  - "node2@192.168.1.2:26656"
```

#### 8.2.2 Peer Exchange (PEX)

```
Peer A connects to Peer B
→ A requests peers from B
← B sends list of known peers
→ A connects to new peers
→ A shares peers with B

Discovery process repeats, network forms mesh topology
```

**PEX Message**:
```protobuf
message PexRequest {
}

message PexResponse {
  repeated PexAddress addrs = 1;
}

message PexAddress {
  string id = 1;    // PeerID
  string ip = 2;
  uint32 port = 3;
}
```

### 8.3 Block Propagation

#### 8.3.1 Gossip Protocol

**GossipSub** for efficient block propagation:

```
Proposer creates block
│
├─> Gossip to connected peers (fanout = 8)
│   │
│   ├─> Peer 1 gossips to their peers
│   ├─> Peer 2 gossips to their peers
│   └─> Peer 3 gossips to their peers
│       │
│       └─> Exponential propagation
│
└─> All peers receive block in O(log n) time
```

**GossipSub Parameters**:
```yaml
gossipsub:
  D: 6           # Desired peer count
  D_low: 4       # Minimum peer count
  D_high: 12     # Maximum peer count
  D_lazy: 6      # Gossip peer count
  fanout: 8      # Broadcast fanout
  gossip_factor: 0.25
  heartbeat_interval: 1s
```

#### 8.3.2 Block Message Format

```protobuf
message BlockGossip {
  int64 height = 1;
  bytes block = 2;      // Serialized Block
  bytes peer_id = 3;
}
```

**Propagation Steps**:
```
1. Proposer creates block at height H
2. Serialize block to protobuf
3. Publish to "/paw/blocks/1.0.0" topic
4. Peers validate and re-gossip
5. Invalid blocks dropped and peer scored down
```

#### 8.3.3 Block Parts

Large blocks split into parts for efficient transfer:

```protobuf
message PartSetHeader {
  uint32 total = 1;
  bytes hash = 2;
}

message Part {
  uint32 index = 1;
  bytes bytes = 2;
  tendermint.crypto.Proof proof = 3;
}
```

**Part Size**: 64 KB

```
Block (1.5 MB) split into parts:
  1,500,000 bytes ÷ 65,536 bytes/part = 23 parts

Peers can request missing parts in parallel
```

### 8.4 Transaction Mempool

#### 8.4.1 Mempool Structure

**Priority Queue** ordered by gas price:

```
Mempool = PriorityQueue<Transaction> {
  comparator: (tx1, tx2) => tx1.gas_price > tx2.gas_price
}

High priority (10 upaw/gas): [tx1, tx5, tx9]
Medium priority (5 upaw/gas): [tx2, tx6, tx10]
Low priority (1 upaw/gas):    [tx3, tx4, tx7, tx8]
```

**Mempool Limits**:
```yaml
mempool:
  size: 10000              # Max transactions
  max_txs_bytes: 10485760  # 10 MB total
  max_tx_bytes: 1048576    # 1 MB per tx
  cache_size: 100000       # Recently seen txs (dedup)
```

#### 8.4.2 Transaction Lifecycle

```
1. User broadcasts tx to node
   │
2. Node validates tx (signature, balance, gas)
   │
3. If valid, add to local mempool
   │
4. Gossip tx to peers
   │
5. Peers validate and add to their mempools
   │
6. Proposer selects txs for next block
   │
7. Block committed, txs removed from mempool
```

#### 8.4.3 Mempool Gossip

```protobuf
message TxMessage {
  bytes tx = 1;
}
```

**Gossip Strategy**:
```
Node receives new tx
│
├─> Validate tx
│   │
│   ├─ Invalid → Drop & penalize sender
│   │
│   └─ Valid → Add to mempool
│       │
│       └─> Gossip to sqrt(N) random peers
```

**Deduplication**:
```go
type MempoolCache struct {
    cache map[string]bool  // tx_hash -> seen
    size  int
}

func (m *Mempool) CheckTx(tx Tx) error {
    txHash := SHA256(tx)

    if m.cache[txHash] {
        return ErrTxAlreadySeen
    }

    m.cache[txHash] = true
    return nil
}
```

### 8.5 Sync Protocol

#### 8.5.1 Block Sync (Fast Sync)

**Tendermint Fast Sync** for catching up:

```
New node joins network
│
├─> Request peers for blockchain height
│
├─> Discover highest known block
│
├─> Download blocks in parallel
│   │
│   ├─> Request blocks [1-100]
│   ├─> Request blocks [101-200]
│   ├─> Request blocks [201-300]
│   └─> ...
│
├─> Validate each block
│
└─> Apply blocks to state sequentially
```

**Sync Phases**:

```
Phase 1: Fast Sync (blocks only)
  - Download and verify blocks
  - Skip full block execution
  - Verify block hashes and signatures
  - Apply state updates from AppHash

Phase 2: Consensus Sync (real-time)
  - Join consensus at current height
  - Participate in block validation
  - Full transaction execution
```

#### 8.5.2 State Sync

**Snapshot-based sync** for instant bootstrap:

```protobuf
message SnapshotRequest {
  uint64 height = 1;
}

message SnapshotResponse {
  uint64 height = 1;
  uint32 format = 2;
  repeated bytes chunks = 3;
  bytes metadata = 4;
}
```

**Sync Process**:
```
1. Request latest snapshot
2. Download snapshot chunks in parallel
3. Verify chunk hashes
4. Apply snapshot to local state
5. Verify state root matches block AppHash
6. Resume block sync from snapshot height
```

**Snapshot Generation**:
```
Every 1,000 blocks:
  1. Export IAVL tree state
  2. Compress snapshot
  3. Split into chunks
  4. Advertise snapshot availability
```

#### 8.5.3 Sync Request/Response

```protobuf
// Block sync
message BlockRequest {
  int64 height = 1;
}

message BlockResponse {
  tendermint.types.Block block = 1;
}

// No block available
message NoBlockResponse {
  int64 height = 1;
}

// Status sync
message StatusRequest {
}

message StatusResponse {
  int64 height = 1;
  bytes base = 2;
}
```

---

## 9. Appendices

### 9.1 Glossary

| Term | Definition |
|------|------------|
| **Account Model** | State model where each account has a balance, unlike UTXO model |
| **AMM** | Automated Market Maker - algorithm for decentralized trading |
| **Bech32** | Bitcoin address encoding format used by Cosmos chains |
| **BFT** | Byzantine Fault Tolerant - consensus that works with malicious nodes |
| **CosmWasm** | WebAssembly-based smart contract platform for Cosmos |
| **CW20** | Cosmos WASM fungible token standard |
| **CW721** | Cosmos WASM non-fungible token standard |
| **DEX** | Decentralized Exchange |
| **DHT** | Distributed Hash Table - decentralized key-value store |
| **Ed25519** | Elliptic curve signature algorithm |
| **Finality** | Guarantee that block cannot be reverted |
| **Gas** | Computational cost unit |
| **GossipSub** | Pub/sub protocol for efficient message propagation |
| **HTLC** | Hash Time Locked Contract - atomic swap primitive |
| **IAVL** | Immutable AVL tree - versioned Merkle tree |
| **libp2p** | Modular peer-to-peer networking library |
| **Mempool** | Memory pool of pending transactions |
| **Protobuf** | Protocol Buffers - binary serialization format |
| **Slashing** | Penalty for validator misbehavior |
| **Tendermint** | BFT consensus algorithm |
| **TEE** | Trusted Execution Environment |
| **VRF** | Verifiable Random Function |
| **WASM** | WebAssembly - portable bytecode format |

### 9.2 Network Parameters Summary

```yaml
# Chain Configuration
chain_id: "paw-1"
denomination: "upaw"
decimals: 6  # 1 PAW = 1,000,000 upaw

# Consensus
block_time: 4s
max_validators: 125
min_validators: 4
byzantine_tolerance: 33%  # Up to 1/3 malicious

# Block Limits
max_block_size: 2097152      # 2 MB
max_gas_per_block: 100000000 # 100M gas
max_tx_size: 1048576         # 1 MB

# Fees
min_gas_price: 0.001 upaw/gas
fee_distribution:
  burn: 50%
  validators: 30%
  treasury: 20%

# Staking
min_validator_stake: 100000000000 upaw  # 100,000 PAW
unbonding_period: 1814400s              # 21 days
max_validators_per_delegator: 100

# Governance
min_deposit: 1000000000 upaw  # 1,000 PAW
voting_period: 604800s         # 7 days
quorum: 33.4%
threshold: 50%
veto_threshold: 33.4%

# State Sync
snapshot_interval: 1000        # blocks
snapshot_keep_recent: 2
state_sync_enable: true

# P2P
max_peers: 50
seed_mode: false
pex: true
persistent_peers_max_dial_period: 0s
```

### 9.3 API Endpoints

#### 9.3.1 RPC Endpoints

```
# Tendermint RPC
https://rpc.paw.network:443

GET  /status                    # Node status
GET  /block?height=<height>     # Block at height
GET  /blockchain?minHeight=<h>&maxHeight=<h>  # Block range
GET  /tx?hash=<hash>            # Transaction by hash
GET  /validators                # Current validator set
POST /broadcast_tx_async        # Broadcast tx (async)
POST /broadcast_tx_sync         # Broadcast tx (sync)
POST /broadcast_tx_commit       # Broadcast tx (wait for commit)
```

#### 9.3.2 REST API

```
# Cosmos REST API
https://api.paw.network:443

GET  /cosmos/bank/v1beta1/balances/{address}
GET  /cosmos/staking/v1beta1/validators
GET  /cosmos/staking/v1beta1/delegations/{delegator}
GET  /cosmos/gov/v1beta1/proposals
POST /cosmos/tx/v1beta1/simulate
POST /cosmos/tx/v1beta1/txs

# CosmWasm
GET  /cosmwasm/wasm/v1/code
GET  /cosmwasm/wasm/v1/code/{code_id}
GET  /cosmwasm/wasm/v1/contract/{address}
POST /cosmwasm/wasm/v1/contract/{address}/smart
```

#### 9.3.3 gRPC Services

```
# gRPC endpoints
grpc.paw.network:9090

cosmos.bank.v1beta1.Query
cosmos.staking.v1beta1.Query
cosmos.gov.v1beta1.Query
cosmwasm.wasm.v1.Query
```

### 9.4 Contract Examples

#### 9.4.1 CW20 Token Contract

```rust
use cosmwasm_std::{
    entry_point, to_binary, Binary, Deps, DepsMut, Env,
    MessageInfo, Response, StdResult, Uint128,
};
use cw20::{Cw20ExecuteMsg, Cw20QueryMsg};

#[entry_point]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    // Initialize token with name, symbol, supply
    let total_supply = msg.initial_balances
        .iter()
        .map(|b| b.amount)
        .sum();

    let token_info = TokenInfo {
        name: msg.name,
        symbol: msg.symbol,
        decimals: msg.decimals,
        total_supply,
        mint: msg.mint.map(|m| m.minter),
    };

    TOKEN_INFO.save(deps.storage, &token_info)?;

    // Set initial balances
    for balance in msg.initial_balances {
        BALANCES.save(
            deps.storage,
            &deps.api.addr_validate(&balance.address)?,
            &balance.amount,
        )?;
    }

    Ok(Response::default())
}

#[entry_point]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: Cw20ExecuteMsg,
) -> Result<Response, ContractError> {
    match msg {
        Cw20ExecuteMsg::Transfer { recipient, amount } => {
            execute_transfer(deps, env, info, recipient, amount)
        }
        Cw20ExecuteMsg::Burn { amount } => {
            execute_burn(deps, env, info, amount)
        }
        // ... other operations
    }
}
```

#### 9.4.2 DEX Liquidity Pool

```rust
#[entry_point]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    match msg {
        ExecuteMsg::Swap {
            offer_asset,
            belief_price,
            max_spread,
            to,
        } => {
            // Constant product AMM: x × y = k
            let pool = POOL.load(deps.storage)?;

            let offer_pool = pool.get_pool_amount(&offer_asset.info);
            let ask_pool = pool.get_pool_amount(&ask_asset.info);

            // Calculate swap amount
            let return_amount = compute_swap(
                offer_pool,
                ask_pool,
                offer_asset.amount,
                pool.fee_rate,
            )?;

            // Check slippage
            assert_slippage_tolerance(
                belief_price,
                return_amount,
                offer_asset.amount,
                max_spread,
            )?;

            // Execute swap
            let recipient = to.unwrap_or(info.sender.to_string());

            Ok(Response::new()
                .add_message(send_tokens(recipient, return_amount))
                .add_attribute("action", "swap")
                .add_attribute("offer_amount", offer_asset.amount)
                .add_attribute("return_amount", return_amount))
        }
        // ... other operations
    }
}

fn compute_swap(
    offer_pool: Uint128,
    ask_pool: Uint128,
    offer_amount: Uint128,
    fee_rate: Decimal,
) -> StdResult<Uint128> {
    // x × y = k (constant product)
    let k = offer_pool * ask_pool;

    // Apply fee
    let offer_amount_after_fee =
        offer_amount * (Decimal::one() - fee_rate);

    // New offer pool
    let new_offer_pool = offer_pool + offer_amount_after_fee;

    // New ask pool (maintain k)
    let new_ask_pool = k / new_offer_pool;

    // Return amount
    let return_amount = ask_pool - new_ask_pool;

    Ok(return_amount)
}
```

### 9.5 Transaction Examples

#### 9.5.1 Simple Transfer

```json
{
  "body": {
    "messages": [
      {
        "@type": "/cosmos.bank.v1beta1.MsgSend",
        "from_address": "paw1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu",
        "to_address": "paw1abc123def456ghi789jkl012mno345pqr678st",
        "amount": [
          {
            "denom": "upaw",
            "amount": "1000000"
          }
        ]
      }
    ],
    "memo": "Payment for services",
    "timeout_height": "0",
    "extension_options": [],
    "non_critical_extension_options": []
  },
  "auth_info": {
    "signer_infos": [
      {
        "public_key": {
          "@type": "/cosmos.crypto.ed25519.PubKey",
          "key": "A+8f7yKZ1234abcd..."
        },
        "mode_info": {
          "single": {
            "mode": "SIGN_MODE_DIRECT"
          }
        },
        "sequence": "42"
      }
    ],
    "fee": {
      "amount": [
        {
          "denom": "upaw",
          "amount": "5000"
        }
      ],
      "gas_limit": "200000",
      "payer": "",
      "granter": ""
    }
  },
  "signatures": [
    "7gH4K9... [64 bytes Ed25519 signature]"
  ]
}
```

#### 9.5.2 Contract Execution

```json
{
  "body": {
    "messages": [
      {
        "@type": "/cosmwasm.wasm.v1.MsgExecuteContract",
        "sender": "paw1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5lzv7xu",
        "contract": "paw1contract123address456...",
        "msg": "eyJzd2FwIjp7Im9mZmVyX2Fzc2V0Ijp7ImluZm8iOnsibmF0aXZlX3Rva2VuIjp7ImRlbm9tIjoidXBhdyJ9fSwiYW1vdW50IjoiMTAwMDAwMCJ9fX0=",
        "funds": [
          {
            "denom": "upaw",
            "amount": "1000000"
          }
        ]
      }
    ],
    "memo": "DEX swap",
    "timeout_height": "0",
    "extension_options": [],
    "non_critical_extension_options": []
  },
  "auth_info": {
    "signer_infos": [
      {
        "public_key": {
          "@type": "/cosmos.crypto.ed25519.PubKey",
          "key": "A+8f7yKZ1234abcd..."
        },
        "mode_info": {
          "single": {
            "mode": "SIGN_MODE_DIRECT"
          }
        },
        "sequence": "43"
      }
    ],
    "fee": {
      "amount": [
        {
          "denom": "upaw",
          "amount": "50000"
        }
      ],
      "gas_limit": "2000000",
      "payer": "",
      "granter": ""
    }
  },
  "signatures": [
    "9kL2M... [64 bytes Ed25519 signature]"
  ]
}
```

### 9.6 State Machine Diagram

```
                    ┌─────────────────┐
                    │   Genesis       │
                    │   (Height 0)    │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │  Initial State  │
                    │  - Validators   │
                    │  - Accounts     │
                    │  - Params       │
                    └────────┬────────┘
                             │
                ┌────────────┴────────────┐
                │                         │
                ▼                         ▼
       ┌─────────────────┐      ┌─────────────────┐
       │  Block Proposal │      │   Sync State    │
       │  (Proposer)     │      │  (New Node)     │
       └────────┬────────┘      └────────┬────────┘
                │                         │
                ▼                         │
       ┌─────────────────┐               │
       │  Prevote Round  │               │
       │  (Validators)   │               │
       └────────┬────────┘               │
                │                         │
                ▼                         │
       ┌─────────────────┐               │
       │ Precommit Round │               │
       │  (Validators)   │               │
       └────────┬────────┘               │
                │                         │
                ▼                         │
       ┌─────────────────┐               │
       │  Commit Block   │◄──────────────┘
       │  (Finalized)    │
       └────────┬────────┘
                │
                ▼
       ┌─────────────────┐
       │  Apply Txs to   │
       │  State (IAVL)   │
       └────────┬────────┘
                │
                ▼
       ┌─────────────────┐
       │  Compute New    │
       │  AppHash        │
       └────────┬────────┘
                │
                ▼
       ┌─────────────────┐
       │  Height++       │
       │  (Next Block)   │
       └────────┬────────┘
                │
                └──────► (Loop)
```

### 9.7 References

1. **Tendermint Core**: https://docs.tendermint.com/
2. **Cosmos SDK**: https://docs.cosmos.network/
3. **CosmWasm**: https://docs.cosmwasm.com/
4. **libp2p**: https://docs.libp2p.io/
5. **Ed25519**: RFC 8032 - https://www.rfc-editor.org/rfc/rfc8032
6. **Bech32**: BIP 173 - https://github.com/bitcoin/bips/blob/master/bip-0173.mediawiki
7. **Protocol Buffers**: https://protobuf.dev/
8. **EIP-1559**: https://eips.ethereum.org/EIPS/eip-1559
9. **IAVL Tree**: https://github.com/cosmos/iavl
10. **GossipSub**: https://github.com/libp2p/specs/tree/master/pubsub/gossipsub

---

**Document Control**

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0.0 | 2025-11-12 | PAW Core Team | Initial technical specification |

**Review Status**: Draft
**Approval Required**: Engineering Lead, Security Auditor, Community Review

---

*End of Technical Specification*
