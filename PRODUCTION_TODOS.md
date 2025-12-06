# PAW Production Todos - Comprehensive Engineering Review

**Generated:** 2025-12-05
**Review Type:** Full Codebase Audit (4 Parallel Agents)
**Scope:** Security, Test Coverage, Code Quality, Infrastructure

---

## Executive Summary

| Category | Critical | High | Medium | Low | Total |
|----------|----------|------|--------|-----|-------|
| Security & Crypto | 2 | 6 | 4 | 5 | 17 |
| Test Coverage | 4 | 8 | 6 | 0 | 18 |
| Code Quality | 0 | 0 | 4 | 5 | 9 |
| Infrastructure | 6 | 15 | 25 | 0 | 46 |
| **TOTAL** | **12** | **29** | **39** | **10** | **90** |

**Overall Status:** Build PASSING, Tests PASSING, but significant work needed for production readiness.

---

## CRITICAL PRIORITY (Must Fix Before Testnet)

### SEC-CRIT-1: Standard Deviation Returns Variance Instead of StdDev
- **File:** `x/oracle/keeper/security.go:1047-1062`
- **Function:** `calculateStdDev()`
- **Issue:** Function returns variance directly without taking square root. All statistical outlier detection is broken.
- **Impact:** Oracle price manipulation possible - outliers NOT detected
- **Fix:** Add `sqrt()` call before return: `return variance.ApproxSqrt()`
- [ ] Fix `calculateStdDev()` to return actual standard deviation
- [ ] Add unit tests for statistical calculations with known values

### SEC-CRIT-2: ApproxSqrt Silent Failure Returns Zero
- **File:** `x/oracle/keeper/oracle_advanced.go:1496`
- **Issue:** When `ApproxSqrt()` fails, function silently returns ZERO, disabling all volatility checks
- **Impact:** Attackers can trigger precision errors to bypass oracle validation
- **Fix:** Log error and return conservative estimate (5-10% default volatility), add circuit breaker trigger
- [ ] Handle ApproxSqrt errors with fallback values
- [ ] Add circuit breaker trigger on sqrt failures

### SEC-CRIT-3: Merkle Proof Order Not Enforced
- **File:** `x/compute/keeper/verification.go:524-532`
- **Function:** `validateMerkleProof()`
- **Issue:** Hash concatenation order not enforced - allows proof forgery
- **Impact:** Fraudulent computation proofs can be validated
- **Fix:** Add canonical ordering: `if bytes.Compare(currentHash, sibling) < 0 { ... }`
- [ ] Implement canonical Merkle proof ordering
- [ ] Add attack vector tests for Merkle proof manipulation

### SEC-CRIT-4: Ed25519 Public Key Not Validated Before Use
- **File:** `x/compute/keeper/verification.go:488-503`
- **Function:** `verifyEd25519Signature()`
- **Issue:** Public key cast without validation, no binding to provider identity
- **Impact:** Small subgroup attacks, key substitution attacks possible
- **Fix:** Validate key length/format, verify key matches registered provider
- [ ] Add Ed25519 public key validation before signature verification
- [ ] Verify public key belongs to claimed provider

### TEST-CRIT-1: DEX Limit Order Engine Has NO Tests
- **File:** `x/dex/keeper/limit_orders.go`
- **Functions:** `PlaceLimitOrder`, `CancelLimitOrder`, `MatchLimitOrder`, `MatchAllOrders`, `ProcessExpiredOrders`
- **Impact:** Core DEX functionality completely untested
- [ ] Create `x/dex/keeper/limit_orders_test.go` with 50+ test cases
- [ ] Test happy paths, error paths, edge cases
- [ ] Test order matching price logic

### TEST-CRIT-2: DEX Security Functions Only 44% Tested
- **File:** `x/dex/keeper/security.go`
- **Missing:** `WithReentrancyGuard`, `ValidatePoolInvariant`, `CheckCircuitBreaker`
- **Impact:** Security protections not verified to work
- [ ] Add reentrancy guard tests (verify nested calls blocked)
- [ ] Add invariant validation tests (verify k=x*y enforcement)
- [ ] Add circuit breaker trigger/recovery tests

### TEST-CRIT-3: Compute Rate Limiting Has NO Tests
- **File:** `x/compute/keeper/security.go`
- **Functions:** `CheckRateLimit`, burst allowance, hourly/daily caps
- **Impact:** DoS protection mechanism unverified
- [ ] Add rate limit bucket creation tests
- [ ] Add token refill mechanism tests
- [ ] Add concurrent request limiting tests

### TEST-CRIT-4: Oracle Byzantine Tolerance Check Not Tested
- **File:** `x/oracle/keeper/security.go`
- **Function:** `CheckByzantineTolerance()`
- **Impact:** Eclipse attack protection unverified
- [ ] Test rejection when < 7 validators
- [ ] Test stake concentration validation
- [ ] Test with malicious validator coalitions

### INFRA-CRIT-1: Hardcoded Development Passwords in Code
- **File:** `compose/docker-compose.dev.yml:102,161`
- **Issue:** `pawdev123` hardcoded for Grafana/PostgreSQL
- **Impact:** Anyone with repo access has dev database credentials
- [ ] Remove hardcoded passwords from compose files
- [ ] Create `.env.example` template
- [ ] Document secure credential generation

### INFRA-CRIT-2: No Helm Chart for K8s Deployments
- **Location:** Missing `/helm/` directory
- **Impact:** Cannot version-control K8s deployments, difficult upgrades
- [ ] Create Helm chart with Chart.yaml, values.yaml, templates/
- [ ] Document multi-environment configuration

### INFRA-CRIT-3: Validator Key Management Undocumented
- **Impact:** Validators at risk of key compromise, no HSM guidance
- [ ] Create `VALIDATOR_KEY_MANAGEMENT.md`
- [ ] Document air-gapped key generation
- [ ] Document HSM integration options
- [ ] Document multi-sig schemes

### INFRA-CRIT-4: No Disaster Recovery Procedures
- **File:** `k8s/validator-statefulset.yaml:354-380`
- **Issue:** Backups only on same cluster, no external backup
- [ ] Implement external backup to S3/GCS
- [ ] Create `DISASTER_RECOVERY.md` runbook
- [ ] Document node recovery from backup
- [ ] Test recovery procedures

---

## HIGH PRIORITY (Fix Before Mainnet)

### SEC-HIGH-1: Future Timestamp Not Enforced in Compute Results
- **File:** `x/compute/keeper/verification.go:69-84`
- **Issue:** Future timestamp only emits event, doesn't reject result
- [ ] Change to reject results with future timestamps

### SEC-HIGH-2: Pool State Not Validated Before Swap Operations
- **File:** `x/dex/keeper/swap_secure.go:54-57`
- **Issue:** Reserves accessed before validation, possible division by zero
- [ ] Validate pool state (reserves > 0) before any calculations

### SEC-HIGH-3: Data Poisoning Prevention Uses Wrong Statistic
- **File:** `x/oracle/keeper/security.go:975-1045`
- **Issue:** Uses variance instead of stdDev in outlier threshold
- [ ] Fix outlier detection to use actual standard deviation

### SEC-HIGH-4: Circuit Breaker Race Condition
- **File:** `x/oracle/keeper/security.go:189-226, 228-260`
- **Issue:** No synchronization between check and update
- [ ] Implement atomic compare-and-swap pattern

### SEC-HIGH-5: Unbounded Nonce Storage Growth
- **File:** `x/compute/keeper/verification.go:595-627`
- **Issue:** Used nonces stored forever with no cleanup
- [ ] Add nonce expiration window (e.g., 100 blocks)
- [ ] Implement `CleanupOldNonces()` in EndBlock

### SEC-HIGH-6: IQR Calculation Off-by-One Errors
- **File:** `x/oracle/keeper/aggregation.go:397-430`
- **Issue:** Percentile calculation doesn't match statistical standards
- [ ] Fix IQR calculation to use proper interpolation

### TEST-HIGH-1: Query Server Only 21-29% Covered
- **DEX:** 3/12 query methods tested
- **Compute:** 5/24 query methods tested
- **Oracle:** 2/7 query methods tested
- [ ] Add tests for all query endpoints
- [ ] Test pagination, error cases, concurrent queries

### TEST-HIGH-2: Secure Keeper Variants Have NO Tests
- **Files:** `liquidity_secure.go`, `pool_secure.go`, `swap_secure.go`
- [ ] Create test files for all secure variants
- [ ] Test all validation and security checks

### TEST-HIGH-3: IBC OnTimeoutPacket Not Tested in Any Module
- **Impact:** State consistency on timeouts unverified
- [ ] Add DEX timeout refund tests
- [ ] Add Compute escrow refund tests
- [ ] Add Oracle timeout handling tests

### TEST-HIGH-4: Ante Decorators Have NO Tests
- **Files:** `app/ante/oracle_decorator.go`, `dex_decorator.go`, `compute_decorator.go`
- [ ] Create `app/ante/*_test.go` files
- [ ] Test all validation logic

### TEST-HIGH-5: MsgServer Handlers Not Tested
- **File:** `x/compute/keeper/msg_server.go`
- [ ] Add transaction entry point tests
- [ ] Test all message types

### TEST-HIGH-6: Aggregation & Cryptoeconomics NO Tests
- **Files:** `x/oracle/keeper/aggregation.go`, `cryptoeconomics.go`
- [ ] Test price aggregation from multiple validators
- [ ] Test weighted voting power calculation
- [ ] Test outlier detection

### INFRA-HIGH-1: No GitHub Actions CI/CD
- **Location:** Missing `.github/workflows/`
- [ ] Create `build.yml` for PR builds
- [ ] Create `test.yml` for full test suite
- [ ] Create `release.yml` for automated releases

### INFRA-HIGH-2: K8s Storage Classes Undefined
- **File:** `k8s/persistent-volumes.yaml`
- **Issue:** References storage classes that don't exist
- [ ] Create `storage-classes.yaml`
- [ ] Document cloud provider integrations

### INFRA-HIGH-3: Ingress Depends on Undocumented cert-manager
- **File:** `k8s/ingress.yaml`
- [ ] Create cert-manager setup documentation
- [ ] Parameterize domains via ConfigMap

### INFRA-HIGH-4: Prometheus Configuration Missing
- **Issue:** No `prometheus.yml` in expected locations
- [ ] Create prometheus scrape configurations
- [ ] Configure blockchain metrics endpoint

### INFRA-HIGH-5: AlertManager Rules Not Defined
- **File:** `k8s/prometheus-genesis-rules.yaml`
- [ ] Add node down/unhealthy alerts
- [ ] Add memory/disk pressure alerts
- [ ] Add consensus failure alerts

### INFRA-HIGH-6: No State Sync Configuration
- **Impact:** New nodes must sync from genesis (slow)
- [ ] Configure state sync RPC endpoints
- [ ] Document snapshot system

### INFRA-HIGH-7: Data Encryption at Rest Not Configured
- **Location:** K8s PersistentVolumes
- [ ] Add encryption to storage classes
- [ ] Document LUKS/dm-crypt for self-managed

### INFRA-HIGH-8: No Release Automation (GoReleaser)
- **File:** Missing `.goreleaser.yaml`
- [ ] Create GoReleaser configuration
- [ ] Add binary signing
- [ ] Add checksum generation

---

## MEDIUM PRIORITY (Fix Before Production Scale)

### CODE-MED-1: Event Type Strings Hardcoded
- **Files:** Multiple keeper files
- **Issue:** Should use constants from `types/events.go`
- [ ] Replace all hardcoded event strings with type constants

### CODE-MED-2: Mixed Error Handling Patterns
- **Files:** `swap.go:23,68,71,88`, `liquidity.go:48,85`
- **Issue:** Uses `fmt.Errorf()` instead of SDK error types
- [ ] Standardize to registered error types

### CODE-MED-3: Hardcoded Gas Constants Without Documentation
- **Files:** `swap_secure.go:31,37,48`, `request.go:20,44`
- [ ] Define gas constants with documentation
- [ ] Document calibration methodology

### CODE-MED-4: Secure/Base Keeper Code Duplication
- **Files:** `swap.go` vs `swap_secure.go`, etc.
- [ ] Refactor to composition pattern
- [ ] Reduce code duplication

### CODE-MED-5: Security Parameters Lack Justification
- **Files:** `security.go:16-31` in DEX/Oracle
- [ ] Add inline comments explaining security rationale

### TEST-MED-1: IBC Channel Lifecycle Not Tested
- [ ] Test OnChanOpenInit, OnChanOpenTry, OnChanOpenAck
- [ ] Test OnChanCloseConfirm with pending operations

### TEST-MED-2: Error Recovery Paths Not Tested
- [ ] Add explicit tests for revert operations
- [ ] Add gas metering accuracy tests

### TEST-MED-3: TWAP Advanced NO Tests
- **File:** `x/oracle/keeper/twap_advanced.go`
- [ ] Add TWAP calculation tests

### TEST-MED-4: DEX Oracle Integration NO Tests
- **File:** `x/dex/keeper/oracle_integration.go`
- [ ] Add cross-module integration tests

### INFRA-MED-1: Docker Base Images Not Pinned
- **Issue:** Using `alpine:3.18`, `3.19`, `3.20` inconsistently
- [ ] Pin to specific patch versions
- [ ] Add Trivy image scanning

### INFRA-MED-2: Docker Compose Healthchecks Inconsistent
- [ ] Add healthchecks to all services

### INFRA-MED-3: K8s RBAC Incomplete
- **File:** `k8s/monitoring-deployment.yaml:278-320`
- [ ] Create ClusterRole for pod discovery
- [ ] Add proper RoleBindings

### INFRA-MED-4: Loki Uses Filesystem Backend
- **File:** `infra/logging/loki-config.yaml:8-17`
- [ ] Create production config with S3/GCS backend
- [ ] Document persistent storage setup

### INFRA-MED-5: Grafana Dashboards Not Provisioned
- [ ] Create node metrics dashboard
- [ ] Create blockchain metrics dashboard
- [ ] Create DEX metrics dashboard

### INFRA-MED-6: No Custom Blockchain Metrics Documented
- [ ] Document metrics endpoint at `:26660/metrics`
- [ ] Document required metric names

### INFRA-MED-7: Environment Variables Not Standardized
- [ ] Standardize to `PAW_*` prefix
- [ ] Document all supported variables

### INFRA-MED-8: No Load Balancer Configuration
- [ ] Configure sticky sessions for RPC
- [ ] Document geo-distributed deployment

### DOC-MED-1: No Deployment Runbook
- [ ] Create `DEPLOYMENT_QUICKSTART.md`
- [ ] Create `DEPLOYMENT_PRODUCTION.md`
- [ ] Create `TROUBLESHOOTING.md`

### DOC-MED-2: No Validator Operator Guide
- [ ] Create `VALIDATOR_OPERATOR_GUIDE.md`
- [ ] Document staking, rewards, slashing

### DOC-MED-3: No Upgrade Procedures
- [ ] Create `UPGRADE_PROCEDURES.md`
- [ ] Document consensus upgrade governance
- [ ] Document rollback procedures

### DOC-MED-4: No API Documentation
- [ ] Create OpenAPI 3.0 specification
- [ ] Document all endpoints with examples

### DOC-MED-5: Module-Level READMEs Missing
- [ ] Create `x/dex/README.md`
- [ ] Create `x/oracle/README.md`
- [ ] Create `x/compute/README.md`

---

## LOW PRIORITY (Nice to Have)

### CODE-LOW-1: Silent Error Ignore in Revert Paths
- **Files:** `swap.go:106,112,120`
- [ ] Add defensive logging for failed reverts

### CODE-LOW-2: Missing Godoc on Public Types
- **Files:** `dex_advanced.go`, `limit_orders.go`, `ibc_aggregation.go`
- [ ] Add godoc comments to exported types

### CODE-LOW-3: Pool Reserves Precision Loss Risk
- **File:** `x/dex/keeper/swap.go:126-132`
- [ ] Add additional safety checks for accumulated precision loss

### CODE-LOW-4: Reentrancy Guard Implementation Unclear
- **File:** `x/dex/keeper/liquidity_secure.go:14-22`
- [ ] Document reentrancy guard implementation

### CODE-LOW-5: Security Parameters Should Be Governable
- **Files:** `security.go` constants
- [ ] Move hardcoded values to module params

### INFRA-LOW-1: Port Mapping Matrix Not Documented
- [ ] Create port mapping documentation

### INFRA-LOW-2: Resource Quota Documentation Missing
- [ ] Document sizing guidance

### INFRA-LOW-3: Cost Estimation Not Provided
- [ ] Create AWS/GCP/Azure cost estimates

### INFRA-LOW-4: SLO Targets Not Defined
- [ ] Define availability, latency, error rate targets

### INFRA-LOW-5: Performance Tuning Guide Missing
- [ ] Create `PERFORMANCE_TUNING.md`

---

## Summary by Module

### DEX Module
- [ ] 50+ limit order tests (CRITICAL)
- [ ] Security function tests (CRITICAL)
- [ ] Secure variant tests (HIGH)
- [ ] Query server tests (HIGH)
- [ ] TWAP tests (MEDIUM)
- [ ] Event constant usage (MEDIUM)

### Oracle Module
- [ ] Fix calculateStdDev (CRITICAL)
- [ ] Fix sqrt error handling (CRITICAL)
- [ ] Byzantine tolerance tests (CRITICAL)
- [ ] Aggregation tests (HIGH)
- [ ] IQR calculation fix (MEDIUM)
- [ ] Circuit breaker sync (MEDIUM)

### Compute Module
- [ ] Merkle proof ordering (CRITICAL)
- [ ] Ed25519 key validation (CRITICAL)
- [ ] Rate limit tests (CRITICAL)
- [ ] MsgServer tests (HIGH)
- [ ] Nonce cleanup (HIGH)
- [ ] Query server tests (HIGH)

### Infrastructure
- [ ] Remove hardcoded passwords (CRITICAL)
- [ ] Create Helm chart (CRITICAL)
- [ ] Disaster recovery docs (CRITICAL)
- [ ] GitHub Actions CI/CD (HIGH)
- [ ] K8s storage classes (HIGH)
- [ ] Prometheus config (HIGH)
- [ ] Alert rules (HIGH)

### Documentation
- [ ] Validator key management (CRITICAL)
- [ ] Disaster recovery (CRITICAL)
- [ ] Deployment runbook (MEDIUM)
- [ ] API documentation (MEDIUM)
- [ ] Module READMEs (MEDIUM)

---

## Recommended Order of Work

### Phase 1: Critical Security (Week 1)
1. Fix `calculateStdDev` to return actual stdDev
2. Fix Merkle proof canonical ordering
3. Add Ed25519 key validation
4. Fix ApproxSqrt error handling
5. Remove hardcoded passwords

### Phase 2: Critical Tests (Week 2)
1. DEX limit order tests (50+ cases)
2. DEX security function tests
3. Compute rate limiting tests
4. Oracle Byzantine tolerance tests

### Phase 3: High Priority Security (Week 3)
1. Future timestamp enforcement
2. Pool state validation
3. Circuit breaker synchronization
4. Nonce storage cleanup

### Phase 4: High Priority Tests (Week 4)
1. Query server coverage (all modules)
2. Secure keeper variant tests
3. IBC timeout tests
4. Ante decorator tests

### Phase 5: Infrastructure (Week 5-6)
1. Helm chart creation
2. GitHub Actions CI/CD
3. Prometheus/AlertManager setup
4. Disaster recovery documentation

### Phase 6: Documentation (Week 7)
1. Validator key management guide
2. Deployment runbook
3. Upgrade procedures
4. API documentation

---

**Estimated Total Effort:** 7-8 weeks for full production readiness

**Current Blockers for Testnet:** Critical security issues in SEC-CRIT-1, SEC-CRIT-2, SEC-CRIT-3, SEC-CRIT-4

**Current Blockers for Mainnet:** All Critical + High priority items
