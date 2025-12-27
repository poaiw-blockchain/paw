# PAW Blockchain Production Roadmap

**Last Updated:** 2025-12-27
**Status:** PUBLIC RELEASE REVIEW IN PROGRESS

---

## 🔴 P0 - CRITICAL (Must Fix Before Public Launch)

### Security Issues

- [x] **SEC-1: Nonce race condition in verification** `x/compute/keeper/verification.go:678-745`
  - ✅ FIXED: Implemented nonce reservation pattern - nonce reserved BEFORE verification, upgraded to "used" after

- [x] **SEC-2: Provider signing key trust-on-first-use** `x/compute/keeper/verification.go:774-839`
  - ✅ FIXED: Providers must explicitly call RegisterSigningKey before submitting results

- [x] **SEC-3: Incomplete private IP range blocking** `x/compute/types/validation.go:328-437`
  - ✅ FIXED: Added comprehensive isPrivateOrReservedIP() covering all RFC1918, link-local, and IPv6 private ranges

- [x] **SEC-4: Reentrancy lock persistence** `x/dex/keeper/security.go:144-235`
  - ✅ FIXED: Added block-height expiration (LockExpirationBlocks=2) and CleanupStaleReentrancyLocks()

### Performance Issues

- [x] **PERF-1: Iterator leak in CleanupOldRateLimitData** `x/dex/keeper/abci.go`
  - ✅ FIXED: IIFE pattern ensures iterator.Close() per iteration

### Data Integrity Issues

- [x] **DATA-1: Migration key prefix mismatch** `x/dex/migrations/v2/migrations.go`
  - ✅ FIXED: Updated to use namespaced prefixes (0x02, 0x01, etc.)

- [x] **DATA-2: Limit order keys missing namespace** `x/dex/keeper/limit_orders.go`
  - ✅ FIXED: Added 0x02 DEX namespace prefix

- [x] **DATA-3: SwapCommit key inconsistency** `commit_reveal.go` and `commit_reveal_gov.go`
  - ✅ FIXED: Consolidated to use 0x02, 0x1D for both

### Repository Cleanup

- [x] **REPO-1: Remove K8s secrets files from version control**
  - ✅ FIXED: Removed in commit 17c15e9, added patterns to .gitignore

- [x] **REPO-2: Clean root directory artifacts**
  - ✅ FIXED: Removed 24 files in commit 17c15e9

---

## 🟡 P1 - HIGH (Should Fix Before Public Testnet)

### Security

- [x] **SEC-5: Add all 8 Ed25519 low-order points** `x/compute/keeper/verification.go:538-565`
  - ✅ FIXED: All 8 low-order points now checked (identity, order 1, order 8, order 2, order 4, 2 additional, high-bit)

- [x] **SEC-6: Minimum liquidity enforcement** `x/dex/keeper/lp_security.go:14-16`
  - ✅ FIXED: MinimumLiquidity=1000 enforced on pool creation and withdrawal protection

### Testing Gaps (150-200 missing test cases)

- [ ] **TEST-1: Ante handler tests (0% coverage)** `app/ante/ante.go`
- [ ] **TEST-2: BeginBlocker/EndBlocker tests (0% coverage)** `app/app.go`
- [ ] **TEST-3: IBC packet timeout tests (0% coverage)**
- [ ] **TEST-4: Provider slashing + reputation recovery tests**
- [ ] **TEST-5: Concurrent provider operation tests**
- [ ] **TEST-6: Genesis export/import with custom modules**
- [ ] **TEST-7: Add 45+ integration tests** for cross-module interactions
- [ ] **TEST-8: Add 25+ error path tests** for auth/access control

### Documentation

- [ ] **DOC-1: Create ADR (Architecture Decision Records)** structure
  - Missing formal design documentation for key decisions

- [ ] **DOC-2: API documentation from proto files**
  - Generate comprehensive API reference with examples

- [ ] **DOC-3: Developer SDK guide**
  - TypeScript, Python, Go client documentation missing

### Performance

- [x] **PERF-2: Cache total voting power** `x/oracle/keeper/aggregation.go`
  - ✅ FIXED: Uses GetCachedTotalVotingPower() from store

- [x] **PERF-3: Add pagination to IterateLiquidityByPool** `x/dex/keeper/liquidity.go`
  - ✅ FIXED: Added IterateLiquidityByPoolPaginated() with DefaultMaxLiquidityIterations=10000

### Data Integrity

- [x] **DATA-4: Fix iterator close in invariants** `x/dex/keeper/invariants.go`
  - ✅ FIXED: IIFE pattern for proper iterator cleanup

- [x] **DATA-5: Make fee claim atomic** `x/dex/keeper/fees.go`
  - ✅ FIXED: CacheContext pattern ensures atomicity

### Code Quality

- [x] **CODE-1: Unify IBC utilities** `x/shared/ibc/types.go`
  - ✅ FIXED: Created unified ChannelOperation in shared package

- [x] **CODE-2: Standardize event prefixes** `x/dex/types/events.go`
  - ✅ FIXED: Added dex_ prefix to EventTypeDexOrderCancelled and EventTypeDexOrderPlaced

---

## 🔵 P2 - MEDIUM (Should Address)

### Security

- [ ] **SEC-7: Randomness predictability in provider selection** `x/compute/keeper/security.go:362-390`
  - Uses only on-chain data - consider VRF or commit-reveal

- [ ] **SEC-8: HTTP allowed for provider endpoints** `x/compute/types/validation.go:228-237`
  - Require HTTPS-only in production

### Testing

- [ ] **TEST-9: IBC scenario tests** (12+ scenarios needed)
  - Timeout, channel closure, error recovery

- [ ] **TEST-10: Security-focused tests** (8+ scenarios)
  - Oracle manipulation, DEX sandwich attacks

### Performance

- [ ] **PERF-4: Cache module addresses** `x/dex/keeper/swap.go`
- [ ] **PERF-5: Archive old limit orders** - prune orders > 30 days
- [ ] **PERF-6: Filter validator prices by asset in iteration** `x/oracle/keeper/abci.go`

### Documentation

- [ ] **DOC-4: Expand changelog** with migration instructions per version
- [ ] **DOC-5: Breaking change migration guides**
- [ ] **DOC-6: CosmWasm/Smart contract integration guide**

### Code Quality

- [ ] **CODE-3: Parameterize circuit breaker duration** `x/dex/keeper/security.go:283`
  - Currently hardcoded to 1 hour

- [ ] **CODE-4: Convert CircuitBreakerState to proto** - use protobuf instead of JSON
- [ ] **CODE-5: Extract GetIBCPacketNonceKey to shared package** - duplicate in 3 modules

### Repository

- [ ] **REPO-3: Move analysis documents** to `/docs/archive/`
  - 15+ files: `*SUMMARY*.md`, `CODE_PATTERN_ANALYSIS.md`, etc.

---

## 🟢 P3 - LOW (Nice to Have)

### Security

- [ ] **SEC-9: Unbounded outlier history storage** `x/oracle/keeper/slashing.go:192-238`
  - Add periodic cleanup in EndBlocker

### Performance

- [ ] **PERF-7: Top-N streaming for provider cache** `x/compute/keeper/provider_cache.go`
- [ ] **PERF-8: Parallel asset aggregation** in oracle module

### Testing

- [ ] **TEST-11: Run fuzz tests for 72+ hours continuously**
- [ ] **TEST-12: Execute simulation with 5000+ blocks**
- [ ] **TEST-13: Benchmark under sustained 1000 TPS load**

### Documentation

- [ ] **DOC-7: Add godoc to all public functions**
- [ ] **DOC-8: Create FAQ document**
- [ ] **DOC-9: Module development guide** for third-party developers

### Code Quality

- [ ] **CODE-6: Remove context value usage** `x/dex/keeper/security.go:126`
  - Replace with explicit function parameters

- [ ] **CODE-7: Fix duplicate event attribute** `x/oracle/keeper/slashing.go:389-390`
  - Duplicate "reason" key in event emission

---

## Remaining Original Tasks

### Post-Testnet

- [ ] **Upgrade path tested (v1.0 → v1.1)**
  - Full chain upgrade simulation with state migration

### Mainnet Preparation

- [ ] **External security audit completed**
  - Trail of Bits or equivalent audit firm

---

## Performance Thresholds

| Metric | Minimum | Target |
|--------|---------|--------|
| Swap Execution Gas | < 150,000 | < 100,000 |
| Oracle Aggregation Gas | < 500,000 | < 300,000 |
| Compute Request Submission | < 200,000 | < 150,000 |
| Provider Selection Latency | < 100ms | < 50ms |

---

## Review Summary (2025-12-27)

| Category | Score | Notes |
|----------|-------|-------|
| Security | 9/10 | All P0/P1 security issues resolved ✅ |
| Performance | 95% | All critical issues fixed ✅ |
| Data Integrity | 9/10 | All key prefix issues resolved ✅ |
| Test Coverage | 78% | 150-200 missing critical tests (P1 pending) |
| Documentation | 6.4/10 | ADR + SDK docs needed (P1 pending) |
| Code Quality | 9.5/10 | Unified patterns, standardized events ✅ |
| Repository | 9/10 | K8s secrets removed, artifacts cleaned ✅ |

**Status:** All P0 items complete (10/10). P1: 10/18 complete, 8 pending (tests+docs).

**Completed:** SEC-1-6, PERF-1-3, DATA-1-5, REPO-1-2, CODE-1-2 (20 items)
**Remaining:** TEST-1-8, DOC-1-3 (11 items)

*Previous 35 development items completed 2025-12-26.*
*P0/P1 security, performance, data integrity fixes completed 2025-12-27.*
