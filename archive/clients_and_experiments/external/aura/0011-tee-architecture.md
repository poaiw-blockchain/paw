# RFC-0011: TEE Architecture for Secure API Key Aggregation

- **Author(s):** Security & Infrastructure Team
- **Status:** Draft
- **Created:** 2025-11-12
- **Target Release:** Phase 2 - Shared Compute

## Summary

This RFC specifies the Trusted Execution Environment (TEE) architecture for PAW's secure API key aggregation and shared compute infrastructure. The design enables users to donate API keys for communal use while maintaining cryptographic guarantees of confidentiality and integrity through hardware-based isolation. The primary implementation uses AWS Nitro Enclaves with Intel SGX as a secondary option for bare-metal deployments.

## Motivation

PAW's vision includes democratizing access to AI compute through pooled resources. However, this requires solving a fundamental trust problem: how can users safely donate API keys to a shared pool without exposing them to theft, misuse, or unauthorized access?

**Key Challenges:**

1. **Confidentiality**: API keys must never be visible to operators, validators, or other users
2. **Integrity**: Key usage must be accurately metered and cryptographically provable
3. **Accountability**: Donors must receive fair compensation for their contributed compute
4. **Auditability**: The system must be transparently verifiable by external parties
5. **Availability**: The system must handle failures gracefully without compromising keys

**Goals:**

- Protect donated API keys from all parties including infrastructure operators
- Enable verifiable proof-of-exhaustion for fair reward distribution
- Support multiple API providers (OpenAI, Anthropic, Google, etc.)
- Maintain security even if host OS or network is compromised
- Provide transparent, auditable security guarantees

## TEE Technology Choice

### Primary: AWS Nitro Enclaves

**Selection Rationale:**

- **Hardware Root of Trust**: Cryptographic attestation backed by AWS Nitro hardware security module
- **No Known Side Channels**: Unlike Intel SGX, Nitro has not been compromised by speculative execution attacks
- **Auditability**: Public PCR measurements and open-source attestation verification
- **Operational Simplicity**: Managed infrastructure reduces operational burden
- **Availability**: Global deployment on standard EC2 instances
- **Cost Efficiency**: Pay-per-use model with no additional hardware requirements

**Security Properties:**

- Memory encryption at hardware level (AES-256-XTS)
- CPU instruction isolation (separate CPU cores assigned to enclave)
- No persistent storage (keys exist only in volatile memory)
- Cryptographic attestation document signed by Nitro hardware key
- Network isolation (only allowed connections are explicitly whitelisted)

### Secondary: Intel SGX (Software Guard Extensions)

**Use Case**: Bare-metal deployments, edge computing, regulatory requirements

**Trade-offs:**

- **Pros**: Widely available, fine-grained memory protection, mature tooling
- **Cons**: History of side-channel vulnerabilities (Spectre, Foreshadow, SGAxe, etc.), smaller enclave page cache (EPC), complex attestation process
- **Mitigation**: Require latest microcode, disable hyperthreading, implement constant-time operations

### Why Not AMD SEV (Secure Encrypted Virtualization)

**Rejected Due To:**

- Limited remote attestation support (SEV-SNP required but not widely available)
- Weaker isolation model (VM-level vs process-level)
- Attestation cannot verify code identity (only VM measurement)
- Less mature ecosystem and tooling
- Fewer academic security audits

### Comparison Table

| Feature                    | AWS Nitro               | Intel SGX             | AMD SEV             |
| -------------------------- | ----------------------- | --------------------- | ------------------- |
| **Attestation Strength**   | Hardware-signed PCRs    | EPID/DCAP signatures  | VM measurement only |
| **Side-Channel History**   | None known              | Multiple (patched)    | Some (SEVO)         |
| **Memory Isolation**       | Dedicated CPU cores     | EPC (128-256 MB)      | Full VM memory      |
| **Code Verification**      | SHA-384 hash of enclave | MRENCLAVE measurement | VM hash             |
| **Availability**           | AWS regions worldwide   | Specific CPU models   | AMD EPYC only       |
| **Auditability**           | Public PCRs             | Public measurements   | Limited             |
| **Operational Complexity** | Low (managed)           | Medium                | High                |
| **Cost Model**             | Pay-per-use             | Hardware purchase     | Hardware purchase   |
| **Max Enclave Size**       | Entire EC2 instance     | 128-256 MB            | Entire VM           |
| **Network Isolation**      | Built-in firewall       | Application-level     | VM-level            |
| **Recommendation**         | **Primary**             | Secondary             | Not recommended     |

## Detailed Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                         PAW Network                          │
│  ┌──────────────┐     ┌──────────────┐     ┌─────────────┐ │
│  │  Validators  │────▶│ Attestation  │────▶│   Reward    │ │
│  │              │     │  Verifier    │     │ Distributor │ │
│  └──────────────┘     └──────────────┘     └─────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │ verify
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   AWS Nitro Enclave (TEE)                    │
│  ┌────────────────────────────────────────────────────────┐ │
│  │                  Isolated Memory Space                  │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────┐ │ │
│  │  │   Key    │  │  Proxy   │  │Attestation│ │Account-│ │ │
│  │  │ Manager  │◀─│ Service  │◀─│ Service   │ │ing Svc │ │ │
│  │  └──────────┘  └──────────┘  └──────────┘  └────────┘ │ │
│  │       │             │              │             │      │ │
│  │       ▼             ▼              ▼             ▼      │ │
│  │   [Keys in    [HTTP Client]  [Attestation]  [Usage    │ │
│  │    Memory]                      Document]     Metrics] │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
         ▲                       │                    │
         │ donate key            │ API calls          │ proofs
         │                       ▼                    ▼
    ┌─────────┐          ┌─────────────┐      ┌──────────────┐
    │  Users  │          │ API Provider│      │ Blockchain   │
    │         │          │(OpenAI, etc)│      │  (On-chain)  │
    └─────────┘          └─────────────┘      └──────────────┘
```

### Enclave Components

#### 1. Key Manager

**Responsibilities:**

- Store encrypted API keys in enclave memory
- Manage key lifecycle (donation, rotation, revocation, destruction)
- Implement secure key derivation and wrapping
- Enforce access control policies

**Implementation Details:**

```rust
struct KeyManager {
    // Enclave's long-term sealing key (derived from Nitro attestation key)
    sealing_key: [u8; 32],

    // Key pool organized by provider
    keys: HashMap<Provider, Vec<EncryptedKey>>,

    // Key metadata (usage limits, expiry, donor identity)
    metadata: HashMap<KeyFingerprint, KeyMetadata>,

    // Revocation list (CRL-style)
    revoked: HashSet<KeyFingerprint>,
}

struct EncryptedKey {
    // AES-256-GCM encrypted key material
    ciphertext: Vec<u8>,
    nonce: [u8; 12],
    tag: [u8; 16],

    // Key fingerprint (SHA-256 of plaintext key)
    fingerprint: KeyFingerprint,

    // Donor's DID for accounting
    donor_did: String,

    // Expiry timestamp
    expires_at: i64,
}

struct KeyMetadata {
    provider: Provider,
    rate_limit: RateLimit,
    max_usage_minutes: u64,
    used_minutes: u64,
    donated_at: i64,
    last_used: i64,
}
```

**Security Measures:**

- Keys encrypted at rest using AES-256-GCM with unique nonce per key
- Sealing key derived from enclave attestation (tied to specific enclave version)
- Constant-time key comparison to prevent timing attacks
- Memory zeroization on key deletion using `zeroize` crate
- No key material ever written to logs or error messages
- Key access audited with tamper-evident logs

#### 2. Proxy Service

**Responsibilities:**

- Accept authenticated API requests from PAW agents
- Select appropriate key from pool based on provider and availability
- Unwrap key, execute API call, immediately destroy session key
- Enforce rate limits and quota management
- Record usage metrics for proof-of-exhaustion

**Request Flow:**

```
1. Agent sends request: {agent_did, model, prompt_hash, nonce, signature}
2. Verify agent signature and authorization (check stake, reputation)
3. Select key from pool (round-robin, weighted by remaining quota)
4. Decrypt key in memory (exists for <100ms)
5. Make API call with decrypted key
6. Parse response, extract usage (tokens/minutes)
7. Zeroize session key from memory
8. Return response to agent
9. Update accounting ledger
10. Generate usage proof if threshold reached
```

**Security Measures:**

- TLS 1.3 with mutual authentication for agent connections
- Request replay protection via nonce validation
- Rate limiting per agent to prevent abuse
- Circuit breaker pattern if API provider returns auth errors (compromised key)
- Audit log of all requests (without sensitive data)
- Timeout-based session key destruction (max 1 second in memory)

#### 3. Attestation Service

**Responsibilities:**

- Generate remote attestation documents on demand
- Include enclave measurements (PCR values), public keys, and custom data
- Sign attestations with Nitro hardware key
- Provide attestation verification tools for external parties

**Attestation Document Structure:**

```json
{
  "module_id": "i-0123456789abcdef0-enc0123456789abcdef",
  "timestamp": 1731398400000,
  "digest": "SHA384",
  "pcrs": {
    "0": "...", // Enclave image hash
    "1": "...", // Kernel hash
    "2": "...", // Application hash
    "3": "...", // IAM role hash
    "4": "...", // Instance ID
    "8": "..." // Custom data (enclave public key)
  },
  "certificate": "...", // X.509 certificate chain to AWS root CA
  "cabundle": ["...", "...", "..."],
  "public_key": "...", // Enclave's ephemeral public key (Ed25519)
  "user_data": {
    "version": "1.0.0",
    "enclave_hash": "sha256:abc123...",
    "supported_providers": ["openai", "anthropic"],
    "max_rpm": 100
  },
  "nonce": "..." // Requester-supplied nonce for freshness
}
```

**Verification Process:**

1. Validate certificate chain against AWS Nitro root CA
2. Verify signature on attestation document
3. Check PCR0-2 match expected enclave measurements (published on )
4. Verify timestamp is recent (within 5 minutes)
5. Validate nonce matches request
6. Extract and verify enclave public key
7. Store enclave identity for subsequent proof verification

**Security Measures:**

- Attestation documents include requester nonce to prevent replay
- PCR measurements published via transparency log (Google Trillian)
- Certificate pinning for AWS root CA
- Attestation documents cryptographically bound to enclave instance
- Emergency attestation revocation mechanism (if enclave compromised)

#### 4. Accounting Service

**Responsibilities:**

- Track per-key usage at minute-level granularity
- Generate proof-of-exhaustion when usage thresholds met
- Maintain tamper-evident audit log
- Provide usage reports for donors

**Accounting Ledger:**

```rust
struct AccountingLedger {
    // Per-key usage tracking
    usage: HashMap<KeyFingerprint, UsageRecord>,

    // Merkle tree of historical usage (for proofs)
    merkle_tree: MerkleTree,

    // Last checkpoint posted on-chain
    last_checkpoint: Checkpoint,
}

struct UsageRecord {
    key_fingerprint: KeyFingerprint,
    donor_did: String,
    provider: Provider,
    total_minutes: u64,
    last_updated: i64,

    // Recent usage (for rate limiting)
    usage_history: VecDeque<UsageEvent>,
}

struct UsageEvent {
    timestamp: i64,
    minutes_used: u64,
    request_hash: [u8; 32],  // For audit trail
}
```

**Proof-of-Exhaustion Generation:**

```rust
struct ProofOfExhaustion {
    // Donor identity
    donor_did: String,

    // Key identification (hashed for privacy)
    key_fingerprint: [u8; 32],

    // Usage summary
    total_minutes: u64,
    period_start: i64,
    period_end: i64,
    num_requests: u64,

    // Merkle proof of usage events
    merkle_root: [u8; 32],
    merkle_proof: Vec<[u8; 32]>,

    // Attestation
    enclave_signature: [u8; 64],  // Ed25519 signature
    attestation_document: Vec<u8>,  // Full Nitro attestation

    // Metadata
    generated_at: i64,
    nonce: [u8; 32],
}
```

**Security Measures:**

- Append-only ledger with cryptographic linking (hash chain)
- Merkle tree for efficient proof generation and verification
- Usage events signed with enclave key to prevent forgery
- Regular checkpoints posted on-chain for external verification
- Audit log replication to secure external storage (encrypted)

### Key Lifecycle

#### 1. Donation

**Process:**

```
1. User generates asymmetric keypair locally (ephemeral session key)
2. User requests enclave's public key via attestation
3. User verifies attestation (PCR values, certificate chain)
4. User encrypts API key with enclave public key (ECIES)
5. User signs donation transaction with DID key
6. User submits encrypted key + signature to enclave
7. Enclave verifies signature, decrypts key
8. Enclave encrypts key with sealing key, stores in memory
9. Enclave returns key fingerprint to user
10. Enclave posts donation proof on-chain (for reward eligibility)
```

**Encryption Scheme:**

- ECIES (Elliptic Curve Integrated Encryption Scheme)
- Curve25519 for ECDH key agreement
- AES-256-GCM for symmetric encryption
- HMAC-SHA256 for authentication

**On-Chain Record:**

```rust
struct KeyDonation {
    donor_did: String,
    key_fingerprint: [u8; 32],
    provider: Provider,
    max_usage_minutes: u64,
    expires_at: i64,
    enclave_id: String,
    donated_at: i64,
}
```

#### 2. Storage

**Memory Layout:**

- Keys stored in enclave heap (never swapped to disk)
- Each key in separate memory page with guard pages
- Memory encryption enforced by Nitro hardware
- No debugging symbols or core dumps enabled

**Redundancy:**

- Keys replicated across 3 enclaves in different availability zones
- Each enclave maintains independent attestation
- Consensus required for key revocation (2-of-3 multi-sig)

#### 3. Usage

**Session Key Lifecycle (per API call):**

```
Time 0ms:   Receive authenticated request
Time 5ms:   Select key from pool
Time 10ms:  Decrypt key into session memory
Time 15ms:  Establish TLS connection to API provider
Time 50ms:  Send API request
Time 500ms: Receive API response
Time 505ms: Parse usage metrics
Time 510ms: Zeroize session key memory (securely overwrite with random data)
Time 515ms: Update accounting ledger
Time 520ms: Return response to agent
```

**Security Invariants:**

- Plaintext key exists in memory for <1 second
- Key access serialized (no concurrent access to same key)
- Failed requests do not leak key material in error messages
- All key operations audited with tamper-evident logs

#### 4. Rotation

**User-Initiated Rotation:**

```
1. User submits revocation transaction (signed with DID key)
2. Enclave verifies signature, checks donor ownership
3. Enclave marks key as revoked (adds to CRL)
4. In-flight requests complete, new requests rejected
5. Enclave zeroizes key from memory
6. User can optionally donate new key (same process as initial donation)
```

**Automatic Rotation Triggers:**

- API provider returns authentication error (key compromised/revoked)
- Usage quota exhausted
- Expiry timestamp reached
- Enclave upgrade requires key re-encryption

#### 5. Destruction

**Guaranteed Destruction Events:**

- User revocation (immediate)
- Enclave shutdown (volatile memory cleared)
- Expiry timeout (scheduled deletion)
- Emergency shutdown (security incident)
- Key usage quota exhausted

**Secure Deletion:**

```rust
fn zeroize_key(key: &mut [u8]) {
    // Multiple overwrites to prevent forensic recovery
    for _ in 0..3 {
        key.fill(0x00);
        compiler_fence(Ordering::SeqCst);  // Prevent optimization
        key.fill(0xFF);
        compiler_fence(Ordering::SeqCst);
    }
    // Final random overwrite
    OsRng.fill_bytes(key);
    compiler_fence(Ordering::SeqCst);
}
```

### Remote Attestation Flow

#### Boot Attestation

```
1. EC2 instance launches with Nitro Enclave enabled
2. Enclave image (Docker container) loaded into isolated memory
3. Nitro hypervisor measures enclave image, generates PCR0-2
4. Enclave application boots, generates ephemeral Ed25519 keypair
5. Enclave requests attestation document from Nitro hypervisor
6. Nitro signs attestation with hardware key (stored in HSM)
7. Enclave publishes attestation document via HTTPS endpoint
8. PAW validators fetch and verify attestation
9. Validators register enclave as trusted (store public key + PCRs)
10. Enclave begins accepting key donations
```

#### Runtime Attestation

**Continuous Verification:**

- Enclaves provide fresh attestation every 5 minutes
- Validators verify attestation before accepting new proofs
- Attestation includes monotonic counter (prevents replay)
- Certificate revocation check (AWS CRL)

**Attestation Verification Code:**

```rust
fn verify_attestation(doc: &AttestationDocument) -> Result<EnclaveIdentity, Error> {
    // 1. Verify certificate chain to AWS root CA
    verify_cert_chain(&doc.certificate, &doc.cabundle, AWS_NITRO_ROOT_CA)?;

    // 2. Verify signature on attestation document
    verify_signature(&doc.digest, &doc.certificate)?;

    // 3. Check timestamp freshness (within 5 minutes)
    let now = SystemTime::now().duration_since(UNIX_EPOCH)?.as_millis();
    if (now - doc.timestamp).abs() > 300_000 {
        return Err(Error::StaleAttestation);
    }

    // 4. Verify PCR values match expected measurements
    verify_pcrs(&doc.pcrs, EXPECTED_ENCLAVE_HASH)?;

    // 5. Extract enclave public key from user_data
    let public_key = extract_public_key(&doc.public_key)?;

    // 6. Verify nonce (if provided by requester)
    if let Some(expected_nonce) = requester_nonce {
        if doc.nonce != expected_nonce {
            return Err(Error::InvalidNonce);
        }
    }

    Ok(EnclaveIdentity {
        public_key,
        module_id: doc.module_id.clone(),
        pcrs: doc.pcrs.clone(),
        verified_at: now,
    })
}
```

### API Key Pool Management

#### Pool Organization

```rust
struct KeyPool {
    // Separate pools per provider
    pools: HashMap<Provider, ProviderPool>,

    // Global rate limiter (requests per second)
    global_limiter: RateLimiter,
}

struct ProviderPool {
    provider: Provider,

    // Available keys (sorted by remaining quota)
    available: BinaryHeap<WeightedKey>,

    // Keys currently in use (locked)
    in_use: HashMap<KeyFingerprint, LockGuard>,

    // Failed keys (temporarily unavailable)
    failed: HashMap<KeyFingerprint, FailureInfo>,

    // Per-provider rate limiter
    rate_limiter: RateLimiter,
}

struct WeightedKey {
    key: EncryptedKey,
    weight: f64,  // Based on remaining quota, success rate, latency
}
```

#### Key Selection Algorithm

**Weighted Round-Robin:**

```rust
fn select_key(pool: &mut ProviderPool, request: &Request) -> Result<EncryptedKey, Error> {
    // 1. Filter available keys
    let candidates: Vec<_> = pool.available
        .iter()
        .filter(|k| !pool.failed.contains_key(&k.key.fingerprint))
        .filter(|k| k.key.expires_at > now())
        .filter(|k| k.metadata.used_minutes < k.metadata.max_usage_minutes)
        .collect();

    if candidates.is_empty() {
        return Err(Error::NoKeysAvailable);
    }

    // 2. Calculate weights (based on remaining quota + success rate)
    let weights: Vec<f64> = candidates.iter().map(|k| {
        let quota_weight = (k.metadata.max_usage_minutes - k.metadata.used_minutes) as f64;
        let success_rate = get_success_rate(&k.key.fingerprint);
        let latency_weight = 1.0 / (get_avg_latency(&k.key.fingerprint) + 1.0);
        quota_weight * success_rate * latency_weight
    }).collect();

    // 3. Select key using weighted random sampling
    let selected_idx = weighted_random_sample(&weights)?;
    let selected_key = candidates[selected_idx].key.clone();

    // 4. Lock key (prevent concurrent use)
    pool.in_use.insert(selected_key.fingerprint, LockGuard::new());

    Ok(selected_key)
}
```

#### Rate Limiting

**Per-Key Limits:**

- Enforced by API provider (e.g., 10 requests/min for OpenAI)
- Tracked via sliding window algorithm
- Keys temporarily disabled if rate limit hit

**Per-Provider Limits:**

- Aggregate limit across all keys for same provider
- Prevents DDoS-style abuse

**Global Limits:**

- Overall enclave throughput limit (e.g., 1000 req/min)
- Prevents resource exhaustion attacks

#### Fail-Over Mechanism

**Failure Detection:**

```rust
enum KeyFailure {
    AuthenticationError,  // 401/403 from API provider
    RateLimitExceeded,    // 429 from API provider
    NetworkTimeout,       // Connection failed
    InvalidResponse,      // Malformed response
}

struct FailureInfo {
    failure_type: KeyFailure,
    failed_at: i64,
    retry_after: i64,
    failure_count: u32,
}
```

**Fail-Over Strategy:**

1. Mark failed key as temporarily unavailable
2. Calculate exponential backoff (1min, 2min, 4min, 8min, ...)
3. Select backup key from pool
4. If authentication error, mark key as permanently revoked
5. Notify donor of key status via on-chain message

**Circuit Breaker Pattern:**

- If >50% of keys for a provider fail, circuit opens (stop using provider)
- Circuit remains open for 5 minutes (cooldown period)
- After cooldown, allow 1 test request (half-open state)
- If test succeeds, close circuit (resume normal operation)

#### Usage Tracking

**Granularity:**

- Minute-level tracking (smallest billable unit)
- Token-level tracking for fine-grained accounting
- Request-level metadata (model, latency, success/failure)

**Metrics Collected:**

```rust
struct UsageMetrics {
    // Time-based metrics
    total_minutes: u64,
    minutes_per_provider: HashMap<Provider, u64>,

    // Request-based metrics
    total_requests: u64,
    successful_requests: u64,
    failed_requests: u64,

    // Token-based metrics (for LLM providers)
    total_tokens: u64,
    prompt_tokens: u64,
    completion_tokens: u64,

    // Performance metrics
    avg_latency_ms: f64,
    p95_latency_ms: f64,
    p99_latency_ms: f64,

    // Cost estimation (based on provider pricing)
    estimated_cost_usd: f64,
}
```

### Proof-of-Exhaustion

#### Generation

**Trigger Conditions:**

- Key quota exhausted (100% of donated minutes used)
- Periodic checkpoint (every 1000 minutes of aggregate usage)
- User-requested proof (on-demand)
- Enclave shutdown (final accounting)

**Proof Structure:**

```rust
struct ProofOfExhaustion {
    // Version for future compatibility
    version: u8,

    // Key identification
    key_fingerprint: [u8; 32],  // SHA-256(API key)
    donor_did: String,
    provider: Provider,

    // Usage summary
    total_minutes: u64,
    total_requests: u64,
    total_tokens: u64,  // For LLM providers
    period_start: i64,
    period_end: i64,

    // Cryptographic proof
    merkle_root: [u8; 32],      // Root of usage events tree
    merkle_proof: Vec<[u8; 32]>, // Path to leaf
    leaf_index: u64,

    // Attestation
    enclave_public_key: [u8; 32],
    enclave_signature: [u8; 64],  // Signs entire proof
    attestation_document: Vec<u8>, // Full Nitro attestation

    // Metadata
    generated_at: i64,
    nonce: [u8; 32],
}
```

#### Verification (On-Chain)

**Validator Logic:**

```rust
fn verify_proof_of_exhaustion(proof: &ProofOfExhaustion) -> Result<(), Error> {
    // 1. Verify attestation document
    let enclave_id = verify_attestation(&proof.attestation_document)?;

    // 2. Verify enclave public key matches attestation
    if enclave_id.public_key != proof.enclave_public_key {
        return Err(Error::PublicKeyMismatch);
    }

    // 3. Verify signature on proof
    let proof_bytes = serialize_proof_for_signing(proof);
    verify_ed25519_signature(
        &proof.enclave_signature,
        &proof_bytes,
        &proof.enclave_public_key
    )?;

    // 4. Verify merkle proof
    verify_merkle_proof(
        &proof.merkle_root,
        &proof.merkle_proof,
        proof.leaf_index,
        &hash_usage_summary(proof)
    )?;

    // 5. Check proof is fresh (not replayed)
    if proof.generated_at < last_checkpoint_time(proof.donor_did) {
        return Err(Error::StaleProof);
    }

    // 6. Verify donor ownership of key
    verify_key_donation(&proof.donor_did, &proof.key_fingerprint)?;

    // 7. Calculate reward
    let reward = calculate_reward(proof.total_minutes, proof.provider);

    // 8. Mint reward tokens to donor
    mint_rewards(&proof.donor_did, reward)?;

    Ok(())
}
```

**Anti-Replay Protection:**

- Proofs include monotonic counter (increments per proof)
- Validators track highest counter per donor
- Reject proofs with counter ≤ last seen counter

**Reward Calculation:**

```rust
fn calculate_reward(minutes: u64, provider: Provider) -> u64 {
    // Base reward per minute (in PAW tokens)
    let base_rate = get_provider_rate(provider);  // e.g., 0.1 PAW/min

    // Apply multipliers
    let scarcity_multiplier = get_scarcity_multiplier(provider);  // Higher if provider undersupplied
    let quality_multiplier = get_quality_multiplier(key_fingerprint);  // Based on uptime, latency

    // Total reward
    (minutes as f64 * base_rate * scarcity_multiplier * quality_multiplier) as u64
}
```

### Security Model

#### Threat Model

**In-Scope Threats:**

1. **Malicious Infrastructure Operator**: AWS account owner attempts to extract keys
2. **Compromised Host OS**: EC2 instance OS compromised by attacker
3. **Network Adversary**: Man-in-the-middle attacks on enclave communication
4. **Malicious Agent**: PAW agent attempts to abuse shared keys
5. **Side-Channel Attacks**: Timing, cache, speculative execution attacks
6. **Rollback Attacks**: Attacker replays old enclave versions with vulnerabilities

**Out-of-Scope Threats:**

- Physical attacks on AWS data centers (rely on AWS physical security)
- Compromise of TEE hardware manufacturer (Intel/AWS)
- Quantum computer attacks (use post-quantum crypto in Phase 3)
- Social engineering of API key donors

#### Trust Assumptions

**What We Trust:**

1. **AWS Nitro Hardware**: Nitro security module correctly implements memory encryption and attestation
2. **AWS Infrastructure**: AWS does not backdoor Nitro firmware or collude to extract keys
3. **Cryptographic Primitives**: AES-256-GCM, Ed25519, SHA-256 are secure
4. **Certificate Authorities**: AWS CA correctly issues certificates for Nitro attestation
5. **Rust Memory Safety**: Rust compiler prevents memory corruption vulnerabilities

**What We Don't Trust:**

- Host operating system (may be compromised)
- Network (may be monitored or manipulated)
- Other enclaves on same host
- External API providers (they see decrypted requests but not other keys)
- PAW validators (they verify but cannot extract keys)

#### Security Guarantees

**Confidentiality:**

- API keys never leave enclave in plaintext
- Memory encryption prevents physical memory attacks
- Network encryption (TLS 1.3) prevents eavesdropping
- No key material in logs, error messages, or debug output

**Integrity:**

- Attestation proves enclave code has not been tampered with
- Signatures on proofs-of-exhaustion prevent forgery
- Merkle trees ensure usage logs are append-only
- Audit trail enables detection of unauthorized access

**Availability:**

- Multi-AZ deployment (3 enclaves in different zones)
- Graceful degradation (if 1 enclave fails, others continue)
- Circuit breaker prevents cascade failures
- Emergency shutdown mechanism if attack detected

#### Limitations & Mitigations

**Known Limitations:**

1. **Side-Channel Attacks**: Speculative execution vulnerabilities (Spectre, Meltdown)
   - **Mitigation**: Use constant-time operations, disable hyperthreading, apply CPU microcode updates

2. **Denial of Service**: Attacker floods enclave with requests
   - **Mitigation**: Rate limiting, proof-of-stake requirement for agents, resource quotas

3. **Key Compromise**: API provider leaks keys from their side
   - **Mitigation**: Not solvable by TEE; donors should use dedicated keys for PAW

4. **Attestation Replay**: Attacker replays old attestation from vulnerable enclave
   - **Mitigation**: Attestation includes timestamp + nonce, validators check freshness

5. **Enclave Upgrade Attacks**: Malicious enclave version deployed
   - **Mitigation**: Transparency log of PCR values, community code review, governance vote for upgrades

6. **Cross-Enclave Attacks**: Attacker uses one enclave to attack another
   - **Mitigation**: Enclaves isolated at CPU level, separate memory spaces

### Audit Trail

#### Logging Architecture

```rust
struct AuditLog {
    // Append-only log of security events
    events: Vec<AuditEvent>,

    // Merkle tree for tamper detection
    merkle_tree: MerkleTree,

    // Last checkpoint (posted on-chain)
    last_checkpoint: Checkpoint,
}

enum AuditEvent {
    KeyDonated {
        timestamp: i64,
        donor_did: String,
        key_fingerprint: [u8; 32],
        provider: Provider,
    },
    KeyUsed {
        timestamp: i64,
        key_fingerprint: [u8; 32],
        request_hash: [u8; 32],  // Does not reveal request content
        minutes_used: u64,
        success: bool,
    },
    KeyRevoked {
        timestamp: i64,
        donor_did: String,
        key_fingerprint: [u8; 32],
        reason: RevocationReason,
    },
    AttestationGenerated {
        timestamp: i64,
        enclave_id: String,
        requester: String,
    },
    SecurityIncident {
        timestamp: i64,
        incident_type: IncidentType,
        severity: Severity,
        details: String,
    },
}
```

#### Transparency Logs

**Enclave Updates:**

- Every enclave version published to  with source code
- PCR values computed and published to transparency log (Google Trillian)
- Validators monitor log for unauthorized updates
- Community can audit source → binary correspondence

**Usage Checkpoints:**

- Aggregate usage posted on-chain every 1000 minutes
- Merkle root of audit log included in checkpoint
- Anyone can request proof of specific usage event
- Enables external audits of enclave behavior

**Incident Reports:**

- Security incidents disclosed publicly (after remediation)
- Root cause analysis published
- Remediation steps documented
- Post-mortem review with community

### Failure Handling

#### Enclave Crash

**Scenario**: Enclave process crashes due to software bug or resource exhaustion

**Consequences:**

- All keys in memory are lost (volatile memory cleared)
- In-flight API requests fail
- Proofs-of-exhaustion for recent usage not yet posted

**Recovery Process:**

1. Enclave automatically restarts (EC2 Auto Scaling)
2. New enclave generates fresh attestation
3. Validators verify new attestation
4. Donors re-donate keys (if desired)
5. Usage resumes

**User Impact:**

- Donors lose recent usage data (since last checkpoint)
- Agents experience temporary service disruption (30-60 seconds)
- Donors must re-donate keys (manual action required)

**Mitigation:**

- Frequent checkpoints (every 100 minutes instead of 1000)
- Replicate audit logs to external storage (encrypted)
- Multi-enclave deployment (2-of-3 quorum for critical operations)

#### Attestation Failure

**Scenario**: Enclave attestation fails verification (invalid PCRs, expired cert, etc.)

**Consequences:**

- Validators stop accepting proofs from enclave
- New key donations rejected
- Existing keys quarantined (no new requests)

**Recovery Process:**

1. Investigate root cause (software bug, AWS incident, attack)
2. If software bug: deploy fixed enclave version
3. If AWS incident: wait for AWS resolution
4. If attack detected: execute emergency shutdown
5. Validators re-verify attestation after fix
6. Resume normal operation

**User Impact:**

- Service temporarily unavailable (duration depends on root cause)
- Donors' keys remain safe (encrypted in enclave memory)

**Mitigation:**

- Backup enclaves in different AWS regions
- Fail-over to Intel SGX enclave if Nitro unavailable
- Governance multisig can authorize emergency enclave version

#### Compromise Detection

**Detection Mechanisms:**

1. **Anomaly Detection**: Monitor usage patterns for abnormal behavior
2. **Canary Keys**: Inject fake keys, alert if used
3. **External Monitoring**: Independent monitors query enclave, verify attestations
4. **Community Bug Bounty**: Reward discovery of vulnerabilities

**Compromise Indicators:**

- Keys used without corresponding on-chain proof
- Attestation PCRs change without governance approval
- API provider reports unauthorized key usage
- Side-channel attack successful in penetration test

**Emergency Response:**

1. **Immediate**: Shutdown enclave, prevent new key donations
2. **Short-term**: Revoke all keys, notify donors
3. **Medium-term**: Conduct forensic analysis, identify root cause
4. **Long-term**: Deploy patched enclave, compensate affected users

**Slashing:**

- If enclave operator found negligent, slash their stake
- Funds used to compensate donors for lost compute
- Governance vote required for slashing decision

## Implementation

### Technology Stack

**Language**: Rust (memory safety, performance, formal verification potential)

**Key Libraries:**

- `aws-nitro-enclaves-sdk`: Official AWS SDK for Nitro Enclaves
- `ring`: Cryptographic primitives (AES-GCM, Ed25519, SHA-256)
- `zeroize`: Secure memory zeroization
- `serde`: Serialization/deserialization
- `hyper`: HTTP client for API calls
- `tokio`: Async runtime

**Build Process:**

```bash
# Build enclave Docker image
docker build -t paw-enclave:latest .

# Convert to Enclave Image Format (EIF)
nitro-cli build-enclave \
  --docker-uri paw-enclave:latest \
  --output-file paw-enclave.eif

# Extract measurements
nitro-cli describe-eif --eif-path paw-enclave.eif

# Expected output:
# {
#   "Measurements": {
#     "PCR0": "abc123...",  # Enclave image hash
#     "PCR1": "def456...",  # Kernel hash
#     "PCR2": "ghi789..."   # Application hash
#   }
# }

# Publish measurements to transparency log
publish-to-trillian paw-enclave.eif
```

### Minimal Dependencies

**Rationale**: Reduce attack surface by minimizing external dependencies

**Allowed Dependencies:**

- AWS Nitro SDK (required for attestation)
- Core cryptographic libraries (ring, zeroize)
- Async runtime (tokio)
- HTTP client (hyper with minimal features)

**Forbidden Dependencies:**

- Databases (use in-memory data structures)
- Logging frameworks (implement custom audit log)
- Heavy serialization (use serde with minimal features)
- Unnecessary utilities

**Dependency Audit:**

```bash
# Regularly audit dependencies for vulnerabilities
cargo audit

# Review dependency tree for unexpected inclusions
cargo tree

# Pin dependencies to specific versions (no wildcards)
# In Cargo.toml:
[dependencies]
ring = "=0.17.7"  # Exact version pinning
```

### Code Structure

```
paw-enclave/
├── src/
│   ├── main.rs              # Entry point, HTTP server
│   ├── key_manager.rs       # Key storage and lifecycle
│   ├── proxy.rs             # API request proxy
│   ├── attestation.rs       # Attestation generation
│   ├── accounting.rs        # Usage tracking and proofs
│   ├── crypto.rs            # Cryptographic utilities
│   ├── rate_limit.rs        # Rate limiting logic
│   └── audit.rs             # Audit logging
├── tests/
│   ├── integration_test.rs  # End-to-end tests
│   ├── security_test.rs     # Security-specific tests
│   └── fuzzing/             # Fuzz testing harnesses
├── Dockerfile               # Enclave container image
├── Cargo.toml               # Dependencies (minimal)
└── README.md                # Build and deployment instructions
```

### Formal Verification (Optional Phase 2)

**Goal**: Mathematically prove correctness of key manager

**Approach**:

1. Model key manager state machine in TLA+
2. Specify safety properties (keys never leak)
3. Specify liveness properties (keys eventually get used)
4. Use TLC model checker to verify properties
5. Extract verified code using code generation

**Example TLA+ Specification:**

```tla
VARIABLES keys, usage, revoked

KeyManagerInvariant ==
  /\ \A k \in keys : k.encrypted = TRUE  \* All keys encrypted
  /\ \A k \in keys : k.plaintext_lifetime < 1000ms  \* Short lifetime
  /\ \A k \in revoked : k \notin keys  \* Revoked keys removed

SafetyProperty ==
  []KeyManagerInvariant  \* Always true

LivenessProperty ==
  \A k \in keys : <>(k \in usage)  \* Eventually all keys get used
```

## Validation Plan

### Security Audit

**Scope**: Third-party audit by reputable security firm

**Audit Areas:**

1. **Cryptographic Implementation**: Verify correct use of primitives
2. **Memory Safety**: Review Rust unsafe blocks, foreign function interfaces
3. **Side-Channel Resistance**: Test for timing leaks, cache attacks
4. **Attestation Verification**: Ensure PCR checks are robust
5. **Access Control**: Verify authorization logic
6. **Network Security**: Review TLS configuration, certificate validation

**Timeline**: 4-6 weeks

**Deliverable**: Public audit report with findings and remediation plan

### Penetration Testing

**Objectives**: Simulate real-world attacks on enclave

**Attack Vectors:**

1. **Key Extraction**: Attempt to extract API keys from running enclave
2. **Attestation Forgery**: Try to generate fake attestation documents
3. **Denial of Service**: Flood enclave with requests
4. **Privilege Escalation**: Attempt to bypass access controls
5. **Side-Channel Exploitation**: Extract secrets via timing, cache

**Team**: Independent red team with TEE expertise

**Timeline**: 2 weeks

**Deliverable**: Penetration test report with exploits (if found)

### Bug Bounty Program

**Scope**: Ongoing public bug bounty

**Reward Tiers:**

- **Critical** ($50,000): Key extraction, attestation forgery
- **High** ($10,000): Unauthorized API access, DoS attacks
- **Medium** ($2,500): Side-channel leaks, logic bugs
- **Low** ($500): Minor issues, documentation errors

**Platform**: HackerOne or Immunefi

**Launch**: After mainnet deployment

## Backwards Compatibility

**Not Applicable**: This is a new feature, no backwards compatibility concerns.

**Forward Compatibility:**

- Proof-of-exhaustion format versioned (v1, v2, ...)
- Attestation document parsing supports multiple formats
- Enclave upgrades via governance (validators vote on new PCR values)

## Open Questions

1. **Multi-Cloud Support**: Should we support GCP Confidential VMs or Azure Confidential Computing?
   - **Consideration**: Reduces AWS lock-in but increases complexity

2. **Decentralized Attestation**: Can we move from AWS attestation to decentralized verifiable computation?
   - **Consideration**: Better for trustlessness but current tech (zkSNARKs) too slow for real-time

3. **Key Persistence**: Should keys optionally be persisted (encrypted) to survive enclave restarts?
   - **Consideration**: Improves availability but increases attack surface

4. **Post-Quantum Cryptography**: When to migrate to post-quantum algorithms?
   - **Consideration**: Wait for NIST standards to stabilize (2025-2026)

5. **Cross-Chain Proofs**: Should proofs-of-exhaustion be portable across blockchains?
   - **Consideration**: Useful for interoperability but adds complexity

## References

- [AWS Nitro Enclaves Documentation](https://docs.aws.amazon.com/enclaves/)
- [Intel SGX Developer Guide](https://www.intel.com/content/www/us/en/developer/tools/software-guard-extensions/overview.html)
- [AMD SEV-SNP Whitepaper](https://www.amd.com/system/files/TechDocs/SEV-SNP-strengthening-vm-isolation-with-integrity-protection-and-more.pdf)
- [RFC 9334: Remote Attestation Procedures Architecture](https://datatracker.ietf.org/doc/rfc9334/)
- [NIST SP 800-193: Platform Firmware Resiliency Guidelines](https://csrc.nist.gov/publications/detail/sp/800-193/final)

## Changelog

- **2025-11-12**: Initial draft
