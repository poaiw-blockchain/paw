# RFC-0008: Light Client Specification

- **Author(s):** Protocol Team
- **Status:** Draft
- **Created:** 2025-11-12
- **Target Release:** Community Testnet

## Summary

Specification for SPV (Simplified Payment Verification) light clients enabling mobile wallets and bandwidth-constrained devices to perform trustless verification without running full nodes. Based on the Tendermint light client protocol with Merkle proof verification against signed validator headers.

## Motivation & Goals

- Enable mobile wallets to verify transactions, account balances, and credential registry state without downloading full blockchain history.
- Provide cryptographically sound trust model requiring only 1/3+ honest validators by stake weight.
- Deliver sub-second proof verification on resource-constrained devices.
- Support offline-first workflows where proofs can be cached and verified against periodically synced headers.

## Detailed Design

### Trust Model

Light clients operate under the following security assumptions:

- **Adversary Model:** Assume adversary controls <1/3 of total validator stake weight.
- **Safety Guarantee:** If ≥1/3 validators are honest, they cannot be fooled into accepting invalid state.
- **Liveness Guarantee:** If ≥2/3 validators are honest and online, light client can make progress.
- **Byzantine Detection:** Conflicting signatures from the same validator at the same height constitute cryptographic evidence of misbehavior.

### Header Verification

Light clients verify block headers using validator set signatures:

**Header Structure:**

```
Header {
  version: u64,
  chain_id: string,
  height: u64,
  time: Timestamp,
  last_block_id: BlockID,
  last_commit_hash: Hash,          // SHA-256 of previous commit
  data_hash: Hash,                 // SHA-256 of transaction merkle root
  validators_hash: Hash,           // SHA-256 of current validator set
  next_validators_hash: Hash,      // SHA-256 of next validator set
  consensus_hash: Hash,
  app_hash: Hash,                  // Application state root (Merkle root)
  last_results_hash: Hash,
  evidence_hash: Hash,
  proposer_address: Address
}
```

**Signature Verification:**

```
Commit {
  height: u64,
  round: i32,
  block_id: BlockID,
  signatures: Vec<CommitSig>       // Array of validator signatures
}

CommitSig {
  block_id_flag: BlockIDFlag,      // BLOCK_ID_FLAG_COMMIT or NIL
  validator_address: Address,       // Ed25519 public key hash
  timestamp: Timestamp,
  signature: Signature              // Ed25519 signature over canonical vote
}

Vote (canonical encoding) = SignBytes(
  chain_id,
  VOTE_TYPE_PRECOMMIT,
  height,
  round,
  block_id,
  timestamp
)
```

**Verification Algorithm:**

1. Parse header `H` and commit `C` from trusted peer
2. Verify `C.height == H.height`
3. Verify `C.block_id == BlockID(H)`
4. Load last known validator set `V_prev` from local storage
5. For each signature `s` in `C.signatures`:
   - Verify `s.validator_address` exists in `V_prev` with voting power `vp`
   - Verify Ed25519 signature: `Ed25519.Verify(s.validator_address, Vote, s.signature)`
   - Accumulate verified voting power: `total_vp += vp`
6. Require `total_vp > 2/3 * total_stake`
7. If `H.next_validators_hash != H.validators_hash`:
   - Fetch new validator set, verify hash matches `H.next_validators_hash`
   - Store for next header verification

### State Proofs

Light clients verify application state using Merkle proofs against `app_hash` in verified headers.

**IAVL Tree Specification:**

- Tree structure: AVL-balanced Merkle tree (IAVL)
- Hash function: SHA-256
- Internal node hash: `SHA-256(height || size || left_hash || right_hash)`
- Leaf node hash: `SHA-256(0x00 || key_length || key || value_hash)`
- Value hash: `SHA-256(value)`

**Proof Format:**

```
MerkleProof {
  key: bytes,
  value: bytes,
  proof: IAVLProof {
    leaf_hash: Hash,
    aunts: Vec<Hash>,              // Sibling hashes from leaf to root
    path: Vec<bool>                // Left=false, Right=true
  }
}
```

**Verification Algorithm:**

```rust
fn verify_merkle_proof(
  proof: &MerkleProof,
  root_hash: &Hash,
) -> Result<bool> {
  let mut current_hash = compute_leaf_hash(&proof.key, &proof.value);

  for (aunt, is_right) in proof.proof.aunts.iter().zip(&proof.proof.path) {
    current_hash = if *is_right {
      sha256(&[current_hash, *aunt].concat())
    } else {
      sha256(&[*aunt, current_hash].concat())
    };
  }

  Ok(current_hash == *root_hash)
}
```

### Sync Protocol

**1. Initial Trust Setup:**

```
TrustedHeader {
  header: Header,
  commit: Commit,
  validator_set: ValidatorSet,
  trust_source: enum {
    AppBundle,          // Hardcoded at compile time
    QRScan,            // Scanned from trusted display
    SocialRecovery,    // Provided by guardians
    GenesisFile        // From genesis.json
  }
}
```

**2. Sequential Header Sync:**

```
// Verify header at height H+1 using trusted header at H
fn verify_adjacent(
  trusted: &TrustedHeader,
  untrusted: &Header,
  untrusted_commit: &Commit,
) -> Result<TrustedHeader> {
  // Check continuity
  require(untrusted.height == trusted.header.height + 1);
  require(untrusted.last_block_id == BlockID(trusted.header));

  // Verify commit signatures
  verify_commit(untrusted, untrusted_commit, &trusted.validator_set)?;

  // Build new trusted state
  let next_vals = if untrusted.validators_hash != untrusted.next_validators_hash {
    fetch_and_verify_validator_set(untrusted.next_validators_hash)?
  } else {
    trusted.validator_set.clone()
  };

  Ok(TrustedHeader {
    header: untrusted.clone(),
    commit: untrusted_commit.clone(),
    validator_set: next_vals,
    trust_source: TrustSource::Verified
  })
}
```

**3. Bisection Algorithm (Fast Sync):**

```
// Skip from height H to H+n efficiently
fn verify_skipping(
  trusted: &TrustedHeader,
  target: &Header,
  target_commit: &Commit,
  trust_period: Duration,
) -> Result<TrustedHeader> {
  // Check target is within trust period
  require(target.time - trusted.header.time < trust_period);

  // Check validator overlap
  let overlap = compute_voting_power_overlap(
    &trusted.validator_set,
    &target_commit.signatures
  );

  if overlap > 1/3 * trusted.validator_set.total_voting_power {
    // Can verify directly (sufficient overlap)
    verify_commit(target, target_commit, &trusted.validator_set)?;
    Ok(new_trusted_header(target, target_commit))
  } else {
    // Bisect: fetch intermediate header at (H + target.height) / 2
    let mid_height = (trusted.header.height + target.height) / 2;
    let mid_header = fetch_header(mid_height)?;

    let mid_trusted = verify_skipping(trusted, &mid_header.header, &mid_header.commit, trust_period)?;
    verify_skipping(&mid_trusted, target, target_commit, trust_period)
  }
}
```

**4. Trust Period & Clock Drift:**

- **Trust Period:** Maximum time between header updates before trust expires (default: 14 days).
- **Clock Drift Tolerance:** Accept headers with timestamp within ±5 minutes of local clock.
- **Header Refresh Policy:** Fetch new header if latest is >5 minutes old or before generating proofs.

### Proof Capabilities

Light clients can trustlessly verify:

**Transaction Inclusion:**

```
TxProof {
  tx: Transaction,
  merkle_proof: MerkleProof,      // Proof in transaction merkle tree
  header: Header,                  // Signed header containing data_hash
  commit: Commit
}

verify_tx_inclusion(proof: &TxProof) -> bool {
  verify_commit(&proof.header, &proof.commit) &&
  verify_merkle_proof(&proof.merkle_proof, &proof.header.data_hash)
}
```

**Account Balance:**

```
AccountProof {
  address: Address,
  account_state: AccountState {
    nonce: u64,
    balance: u128,
    code_hash: Hash,
    storage_root: Hash
  },
  merkle_proof: MerkleProof,      // Proof in account tree
  header: Header,
  commit: Commit
}

verify_account_state(proof: &AccountProof) -> bool {
  verify_commit(&proof.header, &proof.commit) &&
  verify_merkle_proof(&proof.merkle_proof, &proof.header.app_hash)
}
```

**Verifiable Credential Status:**

```
VCStatusProof {
  vc_id: Hash,
  status: enum { Active, Revoked, Suspended },
  revocation_timestamp: Option<Timestamp>,
  merkle_proof: MerkleProof,      // Proof in VC registry tree
  header: Header,
  commit: Commit
}
```

**Contract State:**

```
ContractStateProof {
  contract_address: Address,
  storage_key: bytes,
  storage_value: bytes,
  account_proof: MerkleProof,     // Proof contract exists
  storage_proof: MerkleProof,     // Proof in contract storage trie
  header: Header,
  commit: Commit
}
```

## API Endpoints

Full nodes expose the following RPC endpoints for light client synchronization:

**GET /light-client/headers?from={height}&to={height}**

```json
Response: {
  "headers": [
    {
      "header": {...},              // Full header structure
      "commit": {
        "height": 12345,
        "signatures": [...]         // Validator signatures
      },
      "validator_set": {            // Included on validator set changes
        "validators": [
          {
            "address": "cosmosvaloper1...",
            "pub_key": {
              "type": "tendermint/PubKeyEd25519",
              "value": "base64..."
            },
            "voting_power": 1000000,
            "proposer_priority": 0
          }
        ],
        "total_voting_power": 100000000
      }
    }
  ]
}
```

**GET /light-client/checkpoint**

```json
Response: {
  "header": {...},
  "commit": {...},
  "validator_set": {...},
  "trusted_height": 12345,
  "trusted_hash": "0x...",
  "expires_at": "2025-11-26T12:00:00Z"
}
```

**GET /light-client/tx-proof/{txid}**

```json
Response: {
  "tx": "base64...",              // Transaction bytes
  "tx_proof": {
    "data": [...],                // Merkle aunts
    "proof": {
      "total": 128,               // Total txs in block
      "index": 42,                // Index of this tx
      "leaf_hash": "0x...",
      "aunts": ["0x...", ...]
    }
  },
  "header": {...},
  "commit": {...}
}
```

**GET /light-client/account-proof/{address}?height={height}**

```json
Response: {
  "address": "cosmos1...",
  "account": {
    "nonce": 15,
    "balance": "1000000000uaura",
    "code_hash": null,
    "storage_root": "0x..."
  },
  "proof": {
    "key": "base64...",           // Account key in state tree
    "value": "base64...",         // RLP-encoded account
    "proof": {
      "leaf_hash": "0x...",
      "aunts": ["0x...", ...],
      "path": [false, true, ...]
    }
  },
  "header": {...},
  "commit": {...}
}
```

**GET /light-client/vc-status-proof/{vc_id}**

```json
Response: {
  "vc_id": "0x...",
  "status": "Active",
  "issued_at": "2025-01-15T10:00:00Z",
  "revoked_at": null,
  "proof": {...},
  "header": {...},
  "commit": {...}
}
```

## Security Model

### Threat Mitigation

**1. Byzantine Validators (<1/3 stake):**

- Cannot forge valid commits (require >2/3 signatures)
- Can withhold data but not create false state
- Detection: Conflicting signatures constitute slashable evidence

**2. Eclipse Attacks:**

- Mitigation: Connect to multiple full node endpoints
- Cross-reference headers from ≥3 independent peers
- Reject if any peer provides conflicting header at same height

**3. Long-Range Attacks:**

- Trust period bounds how far back adversary can rewrite history
- After trust period expires, require new trusted checkpoint from social layer
- Cannot attack within trust period without controlling >1/3 stake at time of attack

**4. Invalid State Transitions:**

- Fraud Proofs: Any full node can submit proof that validator set committed to invalid state transition
- Proof format: `{prev_state_root, block, next_state_root, witness}` where witness demonstrates state transition invalidity
- Light client upgrades to full verification if fraud proof confirmed

**5. Timestamp Manipulation:**

- Bounded by BFT time (median of validator timestamps)
- Light client rejects headers with timestamps >5min ahead of local clock
- Prevents time-based attacks on VC expiration, governance deadlines

### Fraud Proof Format

```
FraudProof {
  type: enum {
    InvalidStateTransition,
    InvalidValidatorSetChange,
    InvalidMerkleRoot,
    DoubleSign
  },
  malicious_header: Header,
  malicious_commit: Commit,
  witness: bytes,                  // Type-specific evidence
  proof_of_invalidity: bytes       // Cryptographic demonstration
}

DoubleSignEvidence {
  validator_address: Address,
  vote_a: Vote,                    // First signed vote
  vote_b: Vote,                    // Conflicting vote
  signature_a: Signature,
  signature_b: Signature,
  // Requires: vote_a.height == vote_b.height
  //           vote_a.round == vote_b.round
  //           vote_a.block_id != vote_b.block_id
  //           Ed25519.Verify(validator_address, vote_a, signature_a)
  //           Ed25519.Verify(validator_address, vote_b, signature_b)
}
```

### Privacy Considerations

- **Query Privacy:** Light client queries reveal which accounts/contracts user is interested in.
  - Mitigation: Query through Tor/VPN, batch queries, dummy queries
- **IP Correlation:** Full node can correlate IP address with queried accounts.
  - Mitigation: Use privacy-preserving RPC proxies, rotate endpoints
- **Timing Attacks:** Query patterns may reveal user behavior.
  - Mitigation: Background sync, pre-fetch commonly accessed state

## Implementation

### Rust Verification Library

Core library providing cryptographic verification primitives:

```rust
// Public API surface
pub struct LightClient {
  trusted_state: TrustedState,
  rpc_client: RpcClient,
  config: LightClientConfig,
}

impl LightClient {
  pub fn new(initial_trust: TrustedHeader, config: LightClientConfig) -> Self;

  pub async fn sync_to_latest(&mut self) -> Result<Header>;

  pub async fn verify_tx_inclusion(
    &self,
    tx_hash: Hash,
  ) -> Result<(Transaction, BlockHeight)>;

  pub async fn verify_account_state(
    &self,
    address: Address,
    height: Option<BlockHeight>,
  ) -> Result<AccountState>;

  pub async fn verify_vc_status(
    &self,
    vc_id: Hash,
  ) -> Result<VCStatus>;

  pub fn get_latest_trusted_height(&self) -> BlockHeight;

  pub fn export_checkpoint(&self) -> TrustedHeader;
}

// Configuration
pub struct LightClientConfig {
  pub trust_period: Duration,           // Default: 14 days
  pub trusting_period: Duration,        // Default: 2/3 of unbonding period
  pub clock_drift: Duration,            // Default: 5 minutes
  pub max_retries: u32,                 // Default: 3
  pub sequential_threshold: u64,        // Use sequential if gap < threshold
}
```

**Dependencies:**

- `ed25519-dalek` for signature verification
- `sha2` for hashing
- `serde` for serialization
- `tendermint` for header/commit types
- `tokio` for async runtime

### React Native Bindings

Mobile wallet integration via FFI:

```typescript
// TypeScript API
import { LightClient } from '@aura/light-client-native';

interface TrustedCheckpoint {
  header: Header;
  commit: Commit;
  validatorSet: ValidatorSet;
}

class AuraLightClient {
  async initialize(checkpoint: TrustedCheckpoint): Promise<void>;

  async syncHeaders(): Promise<Header>;

  async verifyTxInclusion(txHash: string): Promise<{
    tx: Transaction;
    height: number;
  }>;

  async getAccountBalance(address: string): Promise<string>;

  async getVCStatus(vcId: string): Promise<{
    status: 'Active' | 'Revoked' | 'Suspended';
    timestamp: string;
  }>;

  async exportCheckpoint(): Promise<TrustedCheckpoint>;
}

// React Native bridge implementation
// Platform: iOS (Swift), Android (Kotlin)
// FFI: Rust library compiled as static lib, exposed via C ABI
```

### In-App Storage

Trusted header storage on mobile devices:

**iOS (Keychain):**

```swift
// Store encrypted checkpoint in keychain
let checkpoint = try lightClient.exportCheckpoint()
let data = try JSONEncoder().encode(checkpoint)

let query: [String: Any] = [
  kSecClass as String: kSecClassGenericPassword,
  kSecAttrAccount as String: "aura.trusted.checkpoint",
  kSecValueData as String: data,
  kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlock
]

SecItemAdd(query as CFDictionary, nil)
```

**Android (EncryptedSharedPreferences):**

```kotlin
val masterKey = MasterKey.Builder(context)
  .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
  .build()

val sharedPreferences = EncryptedSharedPreferences.create(
  context,
  "aura_light_client",
  masterKey,
  EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
  EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
)

sharedPreferences.edit()
  .putString("trusted_checkpoint", checkpointJson)
  .apply()
```

## Validation Plan

### Security Audit Scope

1. **Cryptographic Verification Logic:**
   - Ed25519 signature verification implementation
   - Merkle proof verification algorithm
   - Header chain verification (sequential + bisection)
   - Validator set transition logic

2. **Attack Surface Analysis:**
   - Eclipse attack resilience
   - Long-range attack mitigation
   - Timestamp manipulation bounds
   - Denial-of-service via malformed proofs

3. **Implementation Audits:**
   - Rust library memory safety (unsafe blocks)
   - FFI boundary security (React Native bindings)
   - Mobile storage security (keychain/encrypted prefs)

### Fuzzing Targets

```rust
// Fuzz header verification with malformed inputs
#[cfg(fuzzing)]
mod fuzz_targets {
  use libfuzzer_sys::fuzz_target;

  fuzz_target!(|data: &[u8]| {
    if let Ok(header) = deserialize_header(data) {
      let _ = verify_header(&trusted_state, &header);
    }
  });

  fuzz_target!(|data: &[u8]| {
    if let Ok(proof) = deserialize_merkle_proof(data) {
      let _ = verify_merkle_proof(&proof, &root_hash);
    }
  });
}
```

**Fuzzing Parameters:**

- 1M+ executions per target
- Coverage-guided (libFuzzer)
- Sanitizers: AddressSanitizer, MemorySanitizer, UndefinedBehaviorSanitizer
- Corpus seeding: Valid headers/proofs from testnet

### Integration Tests

```rust
#[tokio::test]
async fn test_sequential_sync() {
  let genesis = fetch_genesis_checkpoint();
  let mut client = LightClient::new(genesis, default_config());

  // Sync 1000 blocks sequentially
  for height in 1..=1000 {
    client.sync_to_height(height).await.unwrap();
    assert_eq!(client.get_latest_trusted_height(), height);
  }
}

#[tokio::test]
async fn test_bisection_sync() {
  let genesis = fetch_genesis_checkpoint();
  let mut client = LightClient::new(genesis, default_config());

  // Jump to height 100,000 via bisection
  client.sync_to_height(100_000).await.unwrap();

  // Verify intermediate state
  let proof = client.verify_account_state(test_address, Some(50_000)).await.unwrap();
  assert!(proof.verify());
}

#[tokio::test]
async fn test_validator_set_change() {
  // Sync across epoch boundary where validator set changes
  let pre_epoch = fetch_checkpoint_at(epoch_height - 1);
  let mut client = LightClient::new(pre_epoch, default_config());

  client.sync_to_height(epoch_height + 100).await.unwrap();

  // Verify new validator set is trusted
  let latest = client.get_latest_trusted_state();
  assert_ne!(latest.validator_set.hash(), pre_epoch.validator_set.hash());
}
```

### Testnet Requirements

Before mainnet release:

- Deploy 100+ full nodes with light client RPC enabled
- Run 1,000+ mobile devices syncing in parallel
- Simulate network partitions and measure recovery time
- Test validator set changes every 100 blocks
- Measure sync performance: sequential (1K blocks) vs bisection (100K blocks)
- Verify fraud proof detection with intentionally malicious validator

**Success Criteria:**

- <1s proof verification on mobile (iPhone 12, Pixel 5)
- <10s sync from genesis to 1K blocks (sequential)
- <30s sync from genesis to 100K blocks (bisection)
- Zero false negatives in fraud proof detection
- Zero successful attacks in penetration testing

## Backwards Compatibility

- Light client protocol versioned independently from consensus
- Full nodes maintain compatibility with light clients on protocol version `v1.x`
- Breaking changes require coordinated upgrade: governance proposal + 3-month migration window
- Legacy light clients (v1.0) can sync alongside v2.0 clients during transition

## Open Questions

- Should we support ZK-SNARK-based succinct proofs for faster mobile verification?
- Level of proof compression (default Merkle aunts vs. compressed bitmap)?
- Support for light client DAOs (aggregate multiple light clients into quorum)?
- Should we expose fraud proof submission API to light clients or only full nodes?

## References

- [Tendermint Light Client Specification](https://github.com/tendermint/tendermint/tree/main/spec/light-client)
- [Ethereum Light Client Sync Protocol](https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/light-client/sync-protocol.md)
- [IAVL+ Tree Specification](https://github.com/cosmos/iavl/blob/master/docs/overview.md)
- [Ed25519 Signature Scheme](https://ed25519.cr.yp.to/)
