# RFC-0007: Atomic Swap Protocol

- **Author(s):** DEX Team
- **Status:** Draft
- **Created:** 2025-11-12
- **Target Release:** Community Testnet

## Summary

Define a trustless cross-chain atomic swap protocol using Hash Time-Locked Contracts (HTLC) to enable decentralized exchange of assets across heterogeneous blockchains without custodial intermediaries. The protocol supports Ethereum, Bitcoin, and Cosmos IBC chains with cryptographic guarantees that swaps either complete atomically or refund safely.

## Motivation & Goals

- Enable cross-chain liquidity for the AURA DEX without requiring trusted bridges or custodians.
- Provide cryptographic safety guarantees ensuring users cannot lose funds due to counterparty default.
- Support major blockchain ecosystems (Ethereum, Bitcoin, Cosmos) to maximize liquidity and user reach.
- Maintain sub-60-second swap initiation UX while preserving security through proper timeout windows.
- Generate sustainable revenue for liquidity providers through fee collection.

## Detailed Design

### HTLC Contract Structure

Each atomic swap involves two mirrored HTLC contracts deployed on source and destination chains:

**Core Components:**
- **Hash Lock:** SHA-256 hash of a secret preimage (32 bytes). Only the swap initiator knows the preimage initially.
- **Time Lock:** Absolute block height/timestamp after which refunds become possible.
- **Participants:**
  - `initiator`: Address initiating the swap (provides asset A)
  - `participant`: Address accepting the swap (provides asset B)
  - `initiator_refund_address`: Fallback address if swap fails
  - `participant_refund_address`: Fallback address if swap fails

**State Variables:**
```
struct HTLCContract {
    bytes32 hash_lock;           // SHA-256(preimage)
    uint256 time_lock;           // Unix timestamp or block height
    address initiator;
    address participant;
    address initiator_refund;
    address participant_refund;
    uint256 amount;
    address asset;               // Token contract or native asset identifier
    SwapState state;             // PENDING | REDEEMED | REFUNDED
    uint256 creation_time;
    bytes32 swap_id;             // Unique identifier derived from params
}

enum SwapState { PENDING, REDEEMED, REFUNDED }
```

### 4-Phase Protocol

#### Phase 1: Initiate
1. Initiator generates random 32-byte preimage `P` and computes `H = SHA256(P)`.
2. Initiator submits `MsgInitiateSwap` to source chain with:
   - `hash_lock = H`
   - `time_lock = current_time + 48h`
   - `participant` address
   - `amount` and `asset` to lock
   - `swap_id = SHA256(concat(initiator, participant, amount, asset, H, nonce))`
3. Source chain creates HTLC contract, locks initiator's funds, emits `SwapInitiated` event.
4. Initiator communicates `H`, `swap_id`, and HTLC details to participant off-chain (via P2P, relayer, or API).

#### Phase 2: Lock (Participate)
1. Participant verifies initiator's HTLC exists on source chain via light client proof.
2. Participant submits `MsgParticipateSwap` to destination chain with:
   - `hash_lock = H` (same hash from initiator)
   - `time_lock = current_time + 24h` (shorter timeout)
   - `initiator` address
   - `amount` and `asset` to lock
   - `swap_id` (mirrors initiator's)
3. Destination chain creates mirrored HTLC, locks participant's funds, emits `SwapParticipated` event.

#### Phase 3: Reveal & Redeem
1. Initiator monitors destination chain for `SwapParticipated` event.
2. Initiator submits `MsgRedeemSwap` to destination chain with:
   - `swap_id`
   - `preimage = P`
3. Destination chain verifies `SHA256(P) == H`, transfers participant's locked funds to initiator, sets state to `REDEEMED`.
4. Preimage `P` is now publicly visible on destination chain.
5. Participant detects redemption (via event monitoring), extracts `P`.
6. Participant submits `MsgRedeemSwap` to source chain with `preimage = P`.
7. Source chain verifies `SHA256(P) == H`, transfers initiator's locked funds to participant, sets state to `REDEEMED`.

**Result:** Swap completes atomically. Both parties receive their counterparty's assets.

#### Phase 4: Refund (Timeout Path)
If participant never locks funds OR initiator never reveals preimage:

1. After `time_lock` expires on participant's chain (24h):
   - Participant submits `MsgRefundSwap` with `swap_id`.
   - Funds returned to `participant_refund_address`, state set to `REFUNDED`.
2. After `time_lock` expires on initiator's chain (48h):
   - Initiator submits `MsgRefundSwap` with `swap_id`.
   - Funds returned to `initiator_refund_address`, state set to `REFUNDED`.

**Safety Property:** Participant's timeout (24h) is always shorter than initiator's (48h) to prevent scenarios where initiator redeems on destination chain after participant's refund window closes.

### Message Types

#### MsgInitiateSwap
```protobuf
message MsgInitiateSwap {
    string initiator = 1;                    // Bech32 address
    string participant = 2;                  // Bech32 or chain-specific address
    string initiator_refund_address = 3;
    cosmos.base.v1beta1.Coin amount = 4;     // Amount and denom to lock
    bytes hash_lock = 5;                     // 32-byte SHA-256 hash
    uint64 time_lock = 6;                    // Unix timestamp
    string swap_id = 7;                      // Hex-encoded swap identifier
}
```

#### MsgParticipateSwap
```protobuf
message MsgParticipateSwap {
    string participant = 1;
    string initiator = 2;
    string participant_refund_address = 3;
    cosmos.base.v1beta1.Coin amount = 4;
    bytes hash_lock = 5;                     // Must match initiator's hash
    uint64 time_lock = 6;
    string swap_id = 7;                      // Must match initiator's swap_id
}
```

#### MsgRedeemSwap
```protobuf
message MsgRedeemSwap {
    string redeemer = 1;                     // Address claiming funds
    string swap_id = 2;
    bytes preimage = 3;                      // 32-byte secret revealing hash
}
```

#### MsgRefundSwap
```protobuf
message MsgRefundSwap {
    string refunder = 1;                     // Must be initiator or participant
    string swap_id = 2;
}
```

### Timeout Configuration

| Parameter | Value | Rationale |
| --------- | ----- | --------- |
| `participant_timelock` | 24 hours | Gives initiator 24h to redeem on destination chain after participant locks funds |
| `initiator_timelock` | 48 hours | 24h safety margin after participant's timeout to handle chain congestion |
| `reveal_grace_period` | 1 hour | Minimum time before participant timeout for initiator to safely reveal |
| `max_swap_duration` | 7 days | Governance-adjustable maximum timelock to prevent capital lock-up abuse |

**Clock Drift Protection:** All on-chain time comparisons use block timestamps with ±15-minute tolerance accounting for Bitcoin's variable block times.

### Cryptographic Primitives

**Hash Function:** SHA-256 (FIPS 180-4 compliant)
- Collision resistance: 2^128 security level
- Preimage resistance: 2^256 security level
- Standardized across all supported chains (Bitcoin, Ethereum, Cosmos SDK)

**Preimage Generation:**
```
preimage = CSPRNG(32 bytes)  // Cryptographically secure random number generator
hash_lock = SHA256(preimage)
```

**Swap ID Derivation (prevents replay attacks):**
```
swap_id = SHA256(
    initiator_address ||
    participant_address ||
    amount || asset ||
    hash_lock ||
    nonce ||            // Chain-specific sequence number
    chain_id
)
```

### Cross-Chain Verification

Each chain validates counterparty HTLCs via light client proofs:

#### Cosmos IBC Chains
- Use IBC light client verification (ICS-07 Tendermint).
- Query Merkle proof of HTLC state from counterparty chain.
- Verify proof against trusted consensus state.

#### Ethereum
- Maintain Ethereum light client (sync committee signatures post-merge).
- Verify receipt logs via Merkle Patricia Trie proofs.
- Storage proof validation for HTLC contract state.

#### Bitcoin
- SPV proofs for HTLC transaction inclusion.
- Verify transaction outputs using Merkle proof against block header.
- Require 6 confirmations (approximately 1 hour) before considering Bitcoin HTLC locked.

**Light Client Update Frequency:**
- Ethereum: Every epoch (~6.4 minutes)
- Bitcoin: Every 10 blocks (~100 minutes)
- Cosmos: Every block (~6 seconds)

### Fee Structure

**Swap Fee:** 0.1% of swap value
- 70% to liquidity providers
- 20% to protocol treasury
- 10% to relayers (if assisted swap)

**Fee Collection Points:**
- Deducted from initiator's locked amount on source chain.
- Distributed on swap completion (redemption).
- Refunded proportionally if swap is refunded.

**Gas Cost Estimation (Ethereum):**
- `MsgInitiateSwap`: ~80,000 gas
- `MsgParticipateSwap`: ~80,000 gas
- `MsgRedeemSwap`: ~50,000 gas
- `MsgRefundSwap`: ~40,000 gas

### Supported Chains (Phase 1)

| Chain | Asset Support | Light Client | Estimated Launch |
| ----- | ------------- | ------------ | ---------------- |
| Cosmos Hub | ATOM, IBC tokens | Native IBC | Testnet Week 1 |
| Osmosis | OSMO, IBC tokens | Native IBC | Testnet Week 1 |
| Ethereum | ETH, ERC-20 | Sync Committee | Testnet Week 3 |
| Bitcoin | BTC | SPV Client | Testnet Week 6 |

**Chain Adapters:** Abstraction layer mapping chain-specific primitives:
- Address format conversion (Bech32, Ethereum hex, Bitcoin Base58)
- Transaction signing (Secp256k1, ECDSA, Schnorr)
- Block time normalization (seconds, block heights)

## Security Considerations

### Timelock Safety Margins
- **Problem:** Participant redeems on source chain, but initiator's refund window closes before participant can redeem on destination.
- **Solution:** Participant timeout (24h) always < Initiator timeout (48h) with 24h margin accounting for:
  - Chain downtime (Byzantine faults, consensus halts)
  - Mempool congestion (high gas prices delaying inclusion)
  - Cross-chain communication latency

### Hash Collision Resistance
- **Threat:** Attacker finds second preimage `P'` where `SHA256(P') = SHA256(P)`.
- **Mitigation:** SHA-256 provides 2^256 preimage resistance. Computationally infeasible with current/near-future hardware.
- **Monitoring:** Protocol logs hash collisions (never expected to occur) for cryptographic break detection.

### Replay Attack Prevention
- **Threat:** Reusing `swap_id` or `preimage` across different swaps to drain funds.
- **Mitigation:**
  - `swap_id` includes chain-specific nonce and all swap parameters.
  - On-chain state prevents reuse: contracts check `state == PENDING` before redemption.
  - Cross-chain replay prevented by embedding `chain_id` in `swap_id`.

### Front-Running Protection
- **Threat:** Mempool watcher sees initiator's redemption transaction, extracts preimage, front-runs on source chain.
- **Solution:**
  - **Commit-Reveal Enhancement:** Optional two-step redemption:
    1. Commit: Submit `SHA256(preimage || salt)` with higher gas price.
    2. Reveal: After commitment included, reveal `preimage` and `salt`.
  - **Flashbots Integration:** Private transaction submission for redemption on Ethereum.
  - **Grace Period:** Participant must wait `reveal_grace_period` (1h) after participant's lock before redeeming, giving initiator priority.

### MEV (Maximal Extractable Value) Mitigation
- **Sandwich Attacks:** Not applicable—swaps execute at predetermined rates, no AMM slippage.
- **Preimage Extraction:** Addressed via commit-reveal or private mempools.
- **Time Bandit Attacks:** Validators reordering blocks to capture swap value. Mitigated by:
  - Short reveal windows (24h) limit reorganization profitability.
  - Slashing conditions for provable time manipulation (if consensus supports).

### Griefing Attacks
- **Capital Lock-Up Griefing:** Attacker initiates swaps, never reveals preimage, locking participant capital for 24h.
- **Mitigation:**
  - Reputation system tracking swap completion rates.
  - Mandatory participation bond (5% of swap value) slashed on repeated timeouts.
  - Rate limiting: Max 3 pending swaps per address per 24h window.

### Atomic Swap Censorship
- **Threat:** Validators censor redemption transactions to force timeouts.
- **Mitigation:**
  - Multi-relayer infrastructure ensuring redundancy.
  - Threshold encryption schemes (future): Reveal preimage automatically after timeout.
  - Governance-driven validator penalties for provable censorship.

## API Endpoints

### POST /atomic-swap/prepare
Prepares atomic swap parameters and returns unsigned HTLC transactions.

**Request:**
```json
{
  "source_chain": "cosmos-hub",
  "destination_chain": "ethereum",
  "source_asset": "uatom",
  "destination_asset": "0x...ERC20",
  "source_amount": "1000000",
  "destination_amount": "500000000000000000",
  "participant_address": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
  "initiator_refund_address": "cosmos1...",
  "participant_refund_address": "0x..."
}
```

**Response:**
```json
{
  "swap_id": "0x1a2b3c...",
  "hash_lock": "0x9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
  "initiator_timelock": 1736726400,
  "participant_timelock": 1736640000,
  "unsigned_initiator_tx": "0x...",
  "estimated_fees": {
    "source_chain_gas": "5000",
    "destination_chain_gas": "80000",
    "protocol_fee": "1000"
  },
  "preimage_storage_key": "swap:1a2b3c:preimage"
}
```

### POST /atomic-swap/commit
Submits signed HTLC transactions to initiate or participate in swap.

**Request:**
```json
{
  "swap_id": "0x1a2b3c...",
  "role": "initiator",
  "signed_tx": "0x...",
  "chain": "cosmos-hub"
}
```

**Response:**
```json
{
  "tx_hash": "0xABCDEF...",
  "confirmation_threshold": 1,
  "estimated_confirmation_time": 6,
  "status": "pending_confirmation"
}
```

### GET /atomic-swap/status/{swap_id}
Queries current swap state across both chains.

**Response:**
```json
{
  "swap_id": "0x1a2b3c...",
  "source_chain": {
    "chain": "cosmos-hub",
    "state": "PENDING",
    "tx_hash": "0x...",
    "confirmations": 12,
    "amount_locked": "1000000uatom",
    "timelock_expires_at": 1736726400
  },
  "destination_chain": {
    "chain": "ethereum",
    "state": "REDEEMED",
    "tx_hash": "0x...",
    "confirmations": 25,
    "amount_locked": "500000000000000000",
    "timelock_expires_at": 1736640000,
    "preimage_revealed": true
  },
  "next_action": {
    "role": "participant",
    "action": "redeem_on_source",
    "deadline": 1736640000,
    "estimated_gas": "50000"
  }
}
```

### WebSocket /atomic-swap/subscribe/{swap_id}
Real-time updates for swap state transitions.

**Events:**
- `swap_initiated`
- `swap_participated`
- `swap_redeemed`
- `swap_refunded`
- `counterparty_locked` (light client verified)
- `preimage_revealed`
- `timeout_approaching` (emitted at 80% of timelock)

## Validation Plan

### Testnet Phases

**Phase 1: Single-Chain Testing (Week 1-2)**
- Deploy HTLC module on AURA testnet.
- Test all message types with test tokens.
- Verify timeout mechanics using accelerated block times.
- Stress test with 1000+ concurrent swaps.

**Phase 2: Cross-Chain (Cosmos IBC) (Week 3-4)**
- Integrate with Cosmos Hub testnet and Osmosis testnet.
- Validate IBC light client proofs.
- Test failure scenarios (chain halts, validator set changes).
- Measure end-to-end swap latency.

**Phase 3: Ethereum Integration (Week 5-7)**
- Deploy Ethereum HTLC contracts on Sepolia testnet.
- Sync committee light client integration.
- Test ERC-20 and native ETH swaps.
- Gas optimization benchmarks.

**Phase 4: Bitcoin Integration (Week 8-10)**
- Bitcoin testnet SPV client deployment.
- HTLC using Bitcoin Script (OP_SHA256, OP_CHECKLOCKTIMEVERIFY).
- Multi-signature fallback testing.
- Confirmation depth sensitivity analysis.

**Phase 5: Security Audit (Week 11-12)**
- External audit of HTLC contracts and light clients.
- Formal verification of timeout safety properties (TLA+ model).
- Economic attack simulation (griefing, MEV).
- Bug bounty program launch.

### Success Criteria

- **Atomic Safety:** 100% of swaps either complete or refund (zero loss scenarios).
- **Latency:** Sub-60s for initiation, <5min for full completion (Cosmos-Cosmos).
- **Throughput:** Support 100 swaps/second on testnet.
- **Light Client Accuracy:** 99.99% proof verification success rate.
- **Gas Efficiency:** Ethereum swaps <200k gas total per participant.

### Monitoring & Observability

**On-Chain Metrics:**
- Swap completion rate by chain pair.
- Average time-to-completion vs. timeout.
- Refund reasons (participant no-show, initiator no-reveal, timeout).
- Fee collection and distribution.

**Light Client Health:**
- Sync delay (current block vs. latest verified header).
- Proof verification failures and retry rates.
- Consensus anomalies (fork detection, validator set discrepancies).

**Alerting Thresholds:**
- Completion rate <95% for any chain pair.
- Light client sync delay >10 minutes.
- Gas price spikes causing timeout failures.

## Backwards Compatibility

- **Module Versioning:** HTLC module uses semantic versioning. V1 contracts remain supported while V2 adds features (multi-party swaps).
- **Message Format:** Protobuf ensures forward compatibility via unknown field skipping.
- **Chain Upgrades:** Governance-driven migration plans for breaking changes (e.g., hash function upgrade to SHA-3).

## Open Questions

### Should we support more complex multi-party swaps?
**Use Case:** Triangular arbitrage (A→B→C→A) or 1-to-N swaps (airdrop splitting).

**Challenges:**
- Timeout ordering: Each leg needs progressively shorter timelocks, limiting depth.
- Coordination overhead: All parties must be online simultaneously for reveals.
- Increased attack surface: More participants = more griefing vectors.

**Recommendation:** Start with 2-party swaps (RFC-0007). Defer multi-party to RFC-0007-ext after production validation.

### Privacy Enhancements?
- **Current State:** All swap parameters (amounts, participants, preimages) are public on-chain.
- **Future Work:** Zero-knowledge proofs for private amounts, or integration with privacy chains (Penumbra, Zcash) requiring custom HTLC circuits.

### Adaptive Timeouts?
- **Idea:** Dynamically adjust timelock durations based on network congestion or historical completion rates.
- **Risk:** Complexity in ensuring timeout safety across chains with different adaptation algorithms.
- **Decision:** Fixed timeouts for V1, gather data for V2 improvements.

### Cross-Chain Gas Payment?
- **Problem:** User initiating Cosmos→Ethereum swap may lack ETH for gas.
- **Solution:** Gas abstraction layer where relayers pay gas, deducted from swap amount. Requires relayer incentive model (additional 0.05% fee).

## References

- Bitcoin HTLC: BIP-199 (https://github.com/bitcoin/bips/blob/master/bip-0199.mediawiki)
- Ethereum HTLC Implementations: https://github.com/chatch/hashed-timelock-contract-ethereum
- Cosmos IBC Specification: https://github.com/cosmos/ibc
- Atomic Swap Security Analysis: "On the Security and Performance of Proof of Work Blockchains" (Gervais et al., 2016)
- Lightning Network HTLC: BOLT #3 (https://github.com/lightning/bolts/blob/master/03-transactions.md)

## Implementation Checklist

- [ ] Cosmos SDK HTLC module (`x/htlc`)
- [ ] Ethereum Solidity contracts (`HTLCEth.sol`, `HTLCERC20.sol`)
- [ ] Bitcoin Script HTLC templates
- [ ] IBC light client integration (Cosmos chains)
- [ ] Ethereum sync committee light client
- [ ] Bitcoin SPV client
- [ ] API service (prepare, commit, status endpoints)
- [ ] WebSocket event service
- [ ] CLI tools for swap initiation/monitoring
- [ ] Integration tests (Cosmos-Cosmos, Cosmos-Ethereum, Cosmos-Bitcoin)
- [ ] Gas benchmarking suite
- [ ] Security audit
- [ ] Testnet deployment playbook
- [ ] User documentation (swap guides, troubleshooting)
- [ ] Relayer infrastructure setup
