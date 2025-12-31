# PAW Security Documentation

## Reporting Vulnerabilities

**Email**: security@paw-chain.io
**PGP Key**: Available at [paw-chain.io/.well-known/security.txt](https://paw-chain.io/.well-known/security.txt)

**Response Time**: 24-48 hours for initial acknowledgment
**Disclosure Policy**: 90-day coordinated disclosure

---

## Security Model

PAW is a Cosmos SDK blockchain with three primary modules:
- **compute**: Decentralized computation with escrow-based payments
- **oracle**: Cross-chain price feeds via IBC
- **dex**: Automated market maker with concentrated liquidity

### Trust Assumptions

1. **Validator Set**: 2/3+ honest validators (Byzantine fault tolerance)
2. **Cryptographic Primitives**: Ed25519, SHA-256, Groth16 ZK-SNARKs
3. **IBC Relayers**: Untrusted (all packets cryptographically verified)
4. **Compute Providers**: Untrusted (results verified via ZK proofs)

---

## Cryptographic Primitives

### Ed25519 Signature Verification
- Low-order point rejection (8 canonical points blocked)
- Strict canonical signature encoding
- Implementation: `x/compute/keeper/verification.go:540-595`

### Groth16 ZK-SNARK Verification
- BN254 curve with pairing checks
- Verification keys stored on-chain per computation type
- Implementation: `x/compute/keeper/groth16.go`

### Nonce Management
- Epoch-based rotation at 90% capacity threshold
- Block hash entropy mixing for unpredictability
- Implementation: `x/shared/nonce/manager.go`

---

## Module Security

### Compute Module

**Escrow System**
- Two-phase atomic commit using CacheContext
- Timeout-based automatic refunds via EndBlocker
- Catastrophic failure event emission for monitoring

**Provider Selection**
- Commit-reveal scheme prevents MEV/front-running
- Cryptographically secure randomness via `GenerateSecureRandomness`
- Geographic and stake-weighted diversity requirements

**Rate Limiting**
- Token bucket algorithm with safe arithmetic
- Per-address request quotas
- Explicit zero-check before token decrement

**Result Verification**
- Deadline checked before expensive verification (DoS protection)
- ZK-SNARK proof validation
- Ed25519 signature verification with low-order point rejection

### Oracle Module

**IBC Price Feeds**
- Source chain whitelist: Osmosis, Injective, Band, Slinky, UMA
- Packet type validation against whitelist
- TWAP calculations with outlier rejection

**Query Pagination**
- Maximum 1000 results per query
- Pagination key validation
- Implementation: `sanitizePagination()` helper

### DEX Module

**AMM Security**
- Constant product invariant enforcement
- Slippage protection (max 5% default)
- MEV protection via commit-reveal swaps

**Liquidity Management**
- Minimum liquidity lock (1000 units)
- Proportional withdrawal calculations
- Overflow-safe arithmetic

---

## IBC Security

### Packet Validation
- Type whitelist: `compute_request`, `compute_result`, `oracle_price`, `oracle_query`
- Channel capability verification
- Timeout enforcement

### Source Chain Whitelist
Authorized oracle sources:
- `osmosis-1` (Osmosis)
- `injective-1` (Injective)
- `band-laozi-mainnet` (Band Protocol)
- `slinky-1` (Slinky/Skip)
- `uma-mainnet` (UMA Protocol)

---

## Rate Limiting & DoS Protection

### Request Rate Limits
- Token bucket per address
- Configurable via governance
- Safe underflow protection

### Batch Request Limits
- 10,000 gas per request
- 150,000 gas maximum per batch
- Pre-execution gas estimation

### Cache Staleness
- Provider cache invalidated after 2x refresh interval
- Stale cache rejection with event emission

---

## Circuit Breakers

### Emergency Pause
- Governance-controlled module pause
- Per-operation granularity
- Automatic resume after cooldown

### Escrow Limits
- Maximum escrow amount per request
- Total escrowed value caps
- Automatic rejection above thresholds

---

## Cross-Module Error Handling

Errors crossing module boundaries emit structured events:
```
event_type: cross_module_error
attributes:
  - source_module: compute
  - target_module: bank
  - operation: escrow_refund
  - error: <error message>
  - height: <block height>
```

---

## Security Assumptions

1. **Tendermint Consensus**: Assumes correct BFT implementation
2. **Cosmos SDK**: Inherits SDK security guarantees
3. **Go Runtime**: Relies on Go's memory safety
4. **IBC Protocol**: Trusts light client verification

---

## Known Limitations

1. **Oracle Latency**: Cross-chain prices may lag 1-2 blocks
2. **ZK Verification Cost**: ~500k gas for Groth16 verification
3. **Provider Liveness**: Relies on provider availability for computation

---

## Audit History

| Date | Auditor | Scope | Status |
|------|---------|-------|--------|
| TBD | TBD | Full Protocol | Pending |

---

## Bug Bounty Program

**Planned**: $50,000+ pool
**Scope**: All on-chain modules, IBC handlers, cryptographic implementations
**Exclusions**: Documentation, testnet-only code, known issues

---

## Security Checklist (SEC-4)

- [ ] External audit by recognized firm
- [x] SEC-1 high priority items resolved
- [x] SEC-2 medium priority items resolved
- [ ] 3+ months testnet without security incidents
- [ ] Bug bounty program established
- [ ] Incident response playbook documented
- [ ] Monitoring/alerting for circuit breakers
- [x] Security assumptions documented

---

## Contact

For security inquiries: security@paw-chain.io
