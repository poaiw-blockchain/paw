# PAW Production Roadmap

**Status:** Build PASSING | **Chain:** Cosmos SDK 0.50.9 + CometBFT | **Modules:** DEX, Oracle, Compute

**Last Audit:** December 2024 | **Next Review:** Before Testnet

---

## CRITICAL SECURITY FINDINGS (Must Fix Before Testnet)

### CRITICAL-1: IBC Replay Attack Protection Missing
**Status:** ✅ Completed (2025-12-01)
**Files:** `x/dex/keeper/ibc_aggregation.go`, `x/oracle/keeper/ibc_prices.go`, `x/compute/keeper/ibc_compute.go`
**Issue:** Cross-chain packets lack nonce tracking. Attackers can replay successful acknowledgements.
**Impact:** Fund theft via duplicate swaps, oracle manipulation.

**Agent Instructions:**
1. Add `Nonce uint64` field to all IBC packet data structs in `x/*/types/ibc_packets.go`
2. Create `NonceStore` in each keeper with prefix key `nonce/{channel}/{sender}`
3. In `OnRecvPacket`, check `if storedNonce >= packetNonce { return ErrorAck }`, then store new nonce
4. In ACK handlers, include nonce in acknowledgement payload
5. Add tests: send same packet twice, verify second is rejected
6. Run `go test ./x/dex/... ./x/oracle/... ./x/compute/... -v`
**Tests:** `go test ./x/dex/... ./x/oracle/... ./x/compute/... -v`

---

### CRITICAL-2: ZK Proof Verification Stubbed Out
**Status:** ✅ Completed (2025-12-01)
**File:** `x/compute/keeper/ibc_compute.go:1000-1021`
**Issue:** `verifyGroth16Pairing()` performs pairing operations but DISCARDS result. Always returns success.
**Impact:** Providers can submit fake proofs and claim escrow without doing work.

**Agent Instructions:**
1. Open `x/compute/keeper/ibc_compute.go`, find `verifyGroth16Pairing` function
2. Import `github.com/consensys/gnark/backend/groth16`
3. Load verifying key: `vk := k.circuitManager.GetVerifyingKey(ctx, circuitID)`
4. Replace stub with: `return groth16.Verify(proof, vk, publicInputs)`
5. In `verifyIBCZKProof`, remove the "fallback" path that skips verification
6. Add test in `x/compute/keeper/zk_verification_test.go`: submit invalid proof, verify rejection
7. Run `go test ./x/compute/... -v -run TestZKProof`
**Tests:** `go test ./x/compute/... -v -run TestZKProof`

---

### CRITICAL-3: Escrow Double-Refund Vulnerability
**Status:** ✅ Completed (2025-12-01)
**File:** `x/compute/keeper/ibc_compute.go:622-649`
**Issue:** Timeout and ACK handlers can both call `refundEscrow()` without checking if already refunded.
**Impact:** Fund theft via race condition.

**Agent Instructions:**
1. In `x/compute/keeper/escrow.go`, add status field to escrow struct: `Status string` (values: "locked", "released", "refunded")
2. In `refundEscrow()`, add check: `if escrow.Status != "locked" { return nil }` at top
3. Set `escrow.Status = "refunded"` BEFORE sending coins
4. In `releaseEscrow()`, add same check and set `Status = "released"`
5. Add test: call both timeout and ack handler for same packet, verify only one succeeds
6. Run `go test ./x/compute/keeper/... -v -run TestEscrow`
**Tests:** `go test ./x/compute/keeper/... -v -run TestEscrow`

---

### CRITICAL-4: DEX Fee Calculation Bug - Fees Not Collected
**Status:** ✅ Completed (2025-12-01)
**File:** `x/dex/keeper/swap.go:68`
**Issue:** Fee is calculated but never deducted from reserves. Protocol loses ALL swap fees.
**Impact:** Financial loss, broken protocol economics.

**Agent Instructions:**
1. Open `x/dex/keeper/swap.go`, find `ExecuteSwap` function
2. After line 68 `feeAmount := ...`, add: `amountInAfterFee := amountIn.Sub(feeAmount)`
3. Use `amountInAfterFee` (not `amountIn`) in the AMM calculation
4. In `CollectSwapFees`, verify fees are sent to fee collector module account
5. Add test: execute swap, query fee collector balance, verify fees collected
6. Run `go test ./x/dex/keeper/... -v -run TestSwap`
**Tests:** `go test ./x/dex/keeper/... -v -run TestSwap`

---

### CRITICAL-5: Unsafe IBC Acknowledgement Parsing — ✅ Completed
**Files:** `x/dex/ibc_module.go`, `x/oracle/ibc_module.go`, `x/compute/ibc_module.go`
**Fix:** Added 1 MB guard before decoding acknowledgements and regression tests that craft 2 MB payloads. `go test ./x/dex/... ./x/oracle/... ./x/compute/...`

### CRITICAL-6: Missing Packet Validation in OnRecvPacket — ✅ Completed
**Files:** `x/dex/ibc_module.go`, `x/oracle/ibc_module.go`, `x/compute/ibc_module.go`, keeper params
**Fix:** Introduced authorized channel params + keeper helpers, enforced checks in `OnRecvPacket`, added governance helper script, and tests rejecting unauthorized packets. `go test ./x/dex/... ./x/oracle/... ./x/compute/...`

### CRITICAL-7: Provider Reputation Penalization Disabled — ✅ Completed
**File:** `x/compute/keeper/provider_management.go`
**Fix:** Restored the 10% reliability penalty in timeout handling, recalculated overall score, synced provider records, and added `TestHandleRequestTimeoutPenalizesReputation`. `go test ./x/compute/keeper/...`

---

### CRITICAL-8: App.go Error Silencing
**Status:** ✅ Completed (2025-12-01)
**File:** `app/app.go:649`
**Issue:** `GetSubspace()` ignores error return. Module init with invalid params fails silently.
**Impact:** Consensus failures, broken module behavior.

**Agent Instructions:**
1. Open `app/app.go`, find `GetSubspace` function (line 649)
2. Change from: `subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)`
3. Change to:
   ```go
   subspace, found := app.ParamsKeeper.GetSubspace(moduleName)
   if !found {
       panic(fmt.Sprintf("subspace not found for module: %s", moduleName))
   }
   ```
4. Run `make build` to verify no panics on startup
5. Run `make test-unit`
**Tests:** `make build`, `make test-unit`

### CRITICAL-9: State Sync Ignores Peer Reputation
**Status:** ✅ Completed (2025-12-05)
**Files:** `p2p/protocol/state_sync.go`, `p2p/protocol/state_sync_download.go`, `p2p/discovery/peer_manager.go`, `p2p/protocol/state_sync_test.go`
**Issue:** Snapshot offers hard-coded `Reliability: 1.0`, so malicious peers with poor reputation could dominate snapshot selection and chunk downloads.
**Impact:** Attackers can serve poisoned state snapshots or throttle chunk delivery, halting production sync.

**Agent Instructions:**
1. Extend the discovery `PeerManager` with `GetPeerReliability` that sources scores from the reputation manager and clamps them to `[0,1]`.
2. Feed the real reliability into snapshot offers during discovery and weight snapshot selection by total reliability, not raw peer counts.
3. Update chunk scheduling to deterministically prioritize high-reliability peers using weighted round-robin logic.
4. Add regression tests (`TestStateSyncDiscoveryIncludesReliability`, `TestStateSyncSelectionWeightsReliability`, `TestSelectPeerForChunkPrioritizesReliability`) proving snapshots/chunks follow the new weighting.
5. Run `go test ./p2p/protocol -run StateSync`.
**Tests:** `go test ./p2p/protocol -run StateSync`

---

## HIGH PRIORITY SECURITY ISSUES

### HIGH-1: DEX Pool Drain Allowed (90%)
**Status:** ✅ Completed (2025-12-03)
**File:** `x/dex/keeper/swap_secure.go:248-254`
**Issue:** Swaps can drain up to 90% of pool reserves in single tx.
**Impact:** Pool imbalance, LP losses.

**Agent Instructions:**
1. Open `x/dex/keeper/swap_secure.go`
2. Change `maxOutput` calculation from 90% to 30%:
   ```go
   maxOutput := math.LegacyNewDecFromInt(reserveOut).Mul(math.LegacyNewDecWithPrec(30, 2)).TruncateInt()
   ```
3. Add governance param `MaxPoolDrainPercent` to `x/dex/types/params.go`
4. Add test: attempt 50% drain, verify rejection
5. Run `go test ./x/dex/keeper/... -v -run TestPoolDrain`

---

### HIGH-2: Oracle Outlier Detection Accepts Bad Prices
**Status:** ✅ Completed (2025-12-03)
**File:** `x/oracle/keeper/security.go:944-1012`
**Issue:** Outlier prices are DETECTED but still ACCEPTED in consensus.
**Impact:** Oracle manipulation via outlier submission.

**Agent Instructions:**
1. Open `x/oracle/keeper/security.go`, find `ImplementDataPoisoningPrevention`
2. After outlier detection (line 971), change `return nil` to:
   ```go
   return errorsmod.Wrapf(types.ErrOutlierDetected, "price %s deviates %s from median", price, deviation)
   ```
3. Add test: submit outlier price, verify rejection (not just logging)
4. Run `go test ./x/oracle/keeper/... -v -run TestOutlier`

---

### HIGH-3: Flash Loan Protection Too Weak (1 Block)
**Status:** ✅ Completed (2025-12-03)
**File:** `x/dex/keeper/dex_advanced.go:233-264`
**Issue:** `MinBlocksBetweenActions = 1` allows flash loan in 2 consecutive blocks.
**Impact:** Flash loan attacks possible.

**Agent Instructions:**
1. Open `x/dex/keeper/dex_advanced.go`
2. Change `MinBlocksBetweenActions` from 1 to 10
3. Add governance param `FlashLoanProtectionBlocks` to params
4. Add test: add liquidity block N, remove block N+1, verify rejection
5. Run `go test ./x/dex/keeper/... -v -run TestFlashLoan`

---

### HIGH-4: Rate Limiting Race Condition
**Status:** ✅ Completed (2025-12-03)
**File:** `x/compute/keeper/security.go:38-127`
**Issue:** Two concurrent txs can both pass rate limit check before either decrements.
**Impact:** Rate limit bypass.

**Agent Instructions:**
1. Open `x/compute/keeper/security.go`, find `CheckRateLimit`
2. Move token decrement BEFORE the check:
   ```go
   bucket.Tokens--
   if bucket.Tokens < 0 {
       bucket.Tokens = 0
       return ErrRateLimitExceeded
   }
   k.SetRateLimitBucket(ctx, *bucket)
   ```
3. Add test: send 2 requests same block from same account, verify second rejected if at limit
4. Run `go test ./x/compute/keeper/... -v -run TestRateLimit`

---

### HIGH-5: Attestation Verification Bypass
**Status:** ✅ Completed (2025-12-03)
**File:** `x/compute/keeper/ibc_compute.go:1023-1116`
**Issue:** `getValidatorPublicKeys()` returns empty set without error. Verification skipped.
**Impact:** Job results accepted without attestation.

**Agent Instructions:**
1. Open `x/compute/keeper/ibc_compute.go`, find `getValidatorPublicKeys`
2. Change return to error when empty:
   ```go
   if len(pubKeys) == 0 {
       return nil, fmt.Errorf("no validator public keys available for chain %s", chainID)
   }
   ```
3. In `verifyAttestations`, fail if pubKeys retrieval errors
4. Add test: call verification with no pubkeys registered, verify error
5. Run `go test ./x/compute/keeper/... -v -run TestAttestation`

---

### HIGH-6: Unused Variables After Validation (Ante Handlers)
**Files:** `app/ante/oracle_decorator.go:91`, `app/ante/compute_decorator.go:63`
**Issue:** Addresses parsed and validated but never used for actual checks.
**Impact:** Incomplete validation, security gap.

**Agent Instructions:**
1. Open `app/ante/oracle_decorator.go`
2. After `delegate` validation (line 91), add actual authorization check:
   ```go
   if !k.IsAuthorizedFeeder(ctx, delegate, msg.Validator) {
       return sdkerrors.ErrUnauthorized.Wrap("delegate not authorized for validator")
   }
   ```
3. Open `app/ante/compute_decorator.go`
4. After `requester` validation (line 63), add balance check:
   ```go
   if err := k.ValidateRequesterBalance(ctx, requester, msg.MaxPayment); err != nil {
       return err
   }
   ```
5. Run `go test ./app/ante/... -v`

---

### HIGH-7: No IBC Channel Closure Cleanup
**Files:** All `ibc_module.go` files, `OnChanCloseConfirm` handlers
**Issue:** Channel close doesn't refund pending operations. Funds locked forever.
**Impact:** Locked user funds.

**Agent Instructions:**
1. In each module's `OnChanCloseConfirm`:
   ```go
   func (im IBCModule) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
       // Find all pending ops for this channel
       pendingOps := im.keeper.GetPendingOperations(ctx, channelID)
       for _, op := range pendingOps {
           if err := im.keeper.RefundOnChannelClose(ctx, op); err != nil {
               ctx.Logger().Error("failed to refund on channel close", "error", err)
           }
       }
       // Emit event
       ctx.EventManager().EmitEvent(...)
       return nil
   }
   ```
2. Implement `GetPendingOperations` and `RefundOnChannelClose` in each keeper
3. Add test: close channel with pending swap, verify refund
4. Run `go test ./x/*/... -v -run TestChannelClose`

**Progress (2025-12-06):** Added white-box helpers plus DEX/Oracle/Compute channel-close regression tests covering swap refunds, oracle source penalization, and compute escrow refunds; verified via `go test ./x/dex/keeper ./x/oracle/keeper ./x/compute/keeper -v -run 'Test.*ChanClose'`.

---

## MEDIUM PRIORITY ISSUES

### MED-1: Hardcoded Gas Values in Ante Handlers
**Status:** ✅ Completed (2025-12-03)
**Notes:** Added configurable gas params (proto/defaults/validation), ante decorator now reads them, regenerated protobufs, and `make build && make test-unit` both pass.
**File:** `app/ante/dex_decorator.go:64,96,123,142`
**Issue:** Gas consumption hardcoded (1000, 1500, 1200, 1000). Cannot adjust without code change.
**Impact:** Inflexible gas economics.

**Agent Instructions:**
1. Add gas params to `x/dex/types/params.go`:
   ```go
   PoolCreationGas uint64
   SwapValidationGas uint64
   LiquidityGas uint64
   ```
2. In ante handlers, read from params: `params.PoolCreationGas`
3. Add default values matching current hardcoded values
4. Run `make build && make test-unit`

---

### MED-2: Genesis Export Silent Errors
**Status:** ✅ Completed (2025-12-03)
**Notes:** Commission/delegation withdrawals now log explicit errors instead of dropping them; verified via `make build`.
**File:** `app/app.go:770,793`
**Issue:** Validator commission/delegation withdrawal errors ignored during genesis export.
**Impact:** Corrupted genesis state.

**Agent Instructions:**
1. Open `app/app.go`, find `prepForZeroHeightGenesis`
2. Change `_, _ = app.DistrKeeper.WithdrawValidatorCommission(...)` to:
   ```go
   if _, err := app.DistrKeeper.WithdrawValidatorCommission(ctx, valBz); err != nil {
       ctx.Logger().Error("failed to withdraw commission", "validator", valBz, "error", err)
   }
   ```
3. Apply same pattern to delegation rewards withdrawal
4. Run `make build`

---

### MED-3: Circuit Breaker String Parsing Vulnerability
**Status:** ✅ Completed (2025-12-03)
**Notes:** Circuit breaker state now persists via protobuf (`state.proto`), preventing colon-based corruption; ran `make proto-gen` and `go test ./x/oracle/... -v`.
**File:** `x/oracle/keeper/security.go:233-265`
**Issue:** State stored as fmt.Sprintf string. Colons in reason field break parsing.
**Impact:** State corruption.

**Agent Instructions:**
1. Create `CircuitBreakerState` protobuf message in `x/oracle/types/state.proto`
2. Replace string format with protobuf marshal:
   ```go
   bz, err := proto.Marshal(&state)
   store.Set(key, bz)
   ```
3. Update getter to use proto unmarshal
4. Run `make proto-gen && go test ./x/oracle/... -v`

---

### MED-4: Load Test Functions Are Stubs
**Status:** ✅ Completed (2025-12-04)
**File:** `tests/load/gotester/main.go:214-243`
**Issue:** Load tests just sleep, don't actually send transactions.
**Impact:** Cannot measure real throughput.

**Notes:** The Go load tester now provisions funded accounts from a dedicated keyring, establishes a `cosmosclient` with the correct bech32 configuration, and uses real `MsgSend`/`MsgSwap` broadcasts (with nonce-aware metrics) instead of REST stubs. Added CLI flags for keyring management, enforced account initialization, and wired pool discovery + Cosmos SDK tx handling with comprehensive error metrics. Verified via `GOWORK=off go build ./...` under `tests/load/gotester`.

---

### MED-5: MPC Ceremony Uses Simplified Setup
**Status:** ✅ Completed (2025-12-04)
**File:** `x/compute/setup/mpc_ceremony.go:249-273`
**Issue:** Powers of tau use `g1Gen` directly instead of proper SRS.
**Impact:** Weak cryptographic setup.

**Notes:** The MPC ceremony now boots with gnark’s `groth16.Setup` for the target circuit, caches the resulting keys, and persists them through the keeper’s `CircuitKeySink` (new integration test `TestCeremonyKeySinkPersistsKeysInKeeper`). Ceremony finalization serializes/verifies keys via the sink, and a deterministic beacon-backed integration path ensures transcripts are auditable. Tests: `go test ./x/compute/setup/... -v`, `go test ./x/compute/keeper -run TestCeremonyKeySinkPersistsKeysInKeeper -v`.

---

## TEST COVERAGE GAPS (Target: >80%)

### TEST-1: Missing ABCI Handler Tests
**Priority:** CRITICAL
**Files needed:** `x/dex/keeper/abci_test.go`, `x/oracle/keeper/abci_test.go`

**Agent Instructions:**
1. Create `x/dex/keeper/abci_test.go` with tests for:
   - `TestBeginBlocker_UpdatePoolTWAPs` - verify TWAP calculation
   - `TestBeginBlocker_DistributeProtocolFees` - verify fee distribution
   - `TestEndBlocker_CircuitBreakerRecovery` - verify auto-recovery
   - `TestEndBlocker_CleanupRateLimitData` - verify state pruning
2. Create `x/oracle/keeper/abci_test.go` with tests for:
   - `TestBeginBlocker_AggregatePrices` - verify median calculation
   - `TestEndBlocker_ProcessSlashWindows` - verify slashing
3. Run `go test ./x/dex/keeper/... ./x/oracle/keeper/... -v -run TestABCI`

**Progress (2025-12-05):** `x/dex/keeper/abci_test.go` now covers all four DEX scenarios (TWAP updates, protocol fee distribution, circuit breaker recovery, and rate limit cleanup) and runs clean via `go test ./x/dex/...`. Oracle ABCI tests currently exist (`x/oracle/keeper/abci_test.go`) but are failing due to the keeper never detecting validator submissions when seeded manually (BeginBlock aggregated price sticks at lowest validator and EndBlocker never slashes). Next agent should inspect `AggregateAssetPrice` + `CheckMissedVotes` interaction under the oracle test harness and either align helpers with keeper requirements or adjust keeper logic so the tests pass (`go test ./x/oracle/keeper/...`).

---

### TEST-2: Missing Query Server Tests
**Priority:** HIGH
**Files needed:** `x/dex/keeper/query_server_test.go`, `x/oracle/keeper/query_server_test.go`

**Agent Instructions:**
1. Create query server tests covering all endpoints:
   - DEX: Params, Pool, Pools (pagination), PoolByTokens, Liquidity, SimulateSwap
   - Oracle: Price, Prices (pagination), Validator, Validators
2. Test error cases: invalid pool ID, non-existent price, pagination bounds
3. Run `go test ./x/*/keeper/... -v -run TestQuery`

**Progress (2025-12-06):** Added `x/dex/keeper/query_server_test.go` and `x/oracle/keeper/query_server_test.go` with exhaustive coverage for params, pool discovery, liquidity lookups, swap simulation, price queries, and validator pagination (including negative cases). Verified via `go test ./x/dex/keeper ./x/oracle/keeper -v -run TestQuery`.

---

### TEST-3: Missing Genesis Round-Trip Tests
**Priority:** HIGH

**Agent Instructions:**
1. Add to each module's genesis_test.go:
   ```go
   func TestGenesisRoundTrip(t *testing.T) {
       // Create state
       InitGenesis(ctx, keeper, genesis)
       // Export
       exported := ExportGenesis(ctx, keeper)
       // Compare
       require.Equal(t, genesis, exported)
   }
   ```
2. Test with non-empty state (pools, prices, providers)
3. Run `go test ./x/*/keeper/... -v -run TestGenesis`

**Progress (2025-12-06):** Added non-empty Init/Export round-trip suites for DEX, Oracle, and Compute keepers (covering pools/TWAPs, price+validator state, and providers/requests/disputes respectively) and verified via `go test ./x/dex/keeper ./x/oracle/keeper ./x/compute/keeper -v -run 'Test.*GenesisRoundTrip'`.

---

### TEST-4: Missing IBC Timeout Tests
**Priority:** HIGH

**Agent Instructions:**
1. Add IBC timeout tests to each module:
   - DEX: `TestOnTimeoutSwapPacket_Refund`
   - Oracle: `TestOnTimeoutPricePacket_NoRefund`
   - Compute: `TestOnTimeoutJobPacket_RefundEscrow`
2. Verify events emitted, state cleaned up
3. Run `go test ./x/*/... -v -run TestTimeout`

**Progress (2025-12-06):** Added regression coverage for DEX swap refunds, Oracle timeout event emission, and Compute escrow refunds/job status transitions, verified via `go test ./x/dex/keeper ./x/oracle/keeper ./x/compute/keeper -v -run 'Test.*Timeout'`.

---

### TEST-5: Missing Security/Attack Vector Tests
**Priority:** HIGH

**Agent Instructions:**
1. Create `tests/security/attack_vectors_test.go` with:
   - `TestReentrancyProtection` - recursive calls fail
   - `TestIntegerOverflow` - max int amounts handled
   - `TestMEVProtection` - slippage enforced
   - `TestDuplicateSubmission` - nonce prevents replays
2. Run `go test ./tests/security/... -v`

**Progress (2025-12-06):** Added `tests/security/attack_vectors_test.go` covering reentrancy guard regression, SafeMath overflow detection in DEX/Compute plus oracle TWAP overflow guards, slippage enforcement, and nonce replay prevention. Verified via `go test ./tests/security -run TestAttackVectorsTestSuite -v`.

---

## WALLET/FRONTEND PRODUCTION GAPS

### WALLET-1: Browser Extension Wrong Branding
**Priority:** CRITICAL
**Location:** `wallet/browser-extension/`

**Agent Instructions:**
1. Search all files: `grep -r "XAI" wallet/browser-extension/`
2. Replace all "XAI" with "PAW"
3. Update `manifest.json` name, description
4. Update UI strings in all .tsx/.ts files
5. Build and test: `cd wallet/browser-extension && npm run build`

**Progress (2025-12-06):** Completed full rebrand (manifest, popup UI, README) plus background logging and env docs. Verified build via `npm run build` after dependency install; artifacts cleaned post-check.

---

### WALLET-2: Move Wallets from Archive to Production
**Priority:** HIGH

**Agent Instructions:**
1. Create `/wallet/` directory structure:
   ```
   wallet/
     core/
     desktop/
     mobile/
     browser-extension/
   ```
2. Copy from archive: `cp -r archive/wallet/* wallet/`
3. Update all import paths
4. Add to main README.md
5. Run builds for each: `npm install && npm run build`

**Progress (2025-12-06):** Created production `wallet/` tree (core, desktop, mobile, browser-extension, plus docs) copied from archive, updated the top-level README to describe usage, and wired reproducible builds. Browser extension build succeeds (`npm run build`). Core TypeScript SDK now compiles cleanly after adding proper SafeMath/ledger/trezor typings (`npm run build`). Desktop Electron build now outputs AppImage/`.deb` artifacts and exposes a dedicated `npm run build:linux:rpm` script for rpm packaging (requires `rpmbuild` on the host). Mobile React Native package now bundles deterministically for iOS/Android via `npm run build` after restoring the navigator/screens/service layer.

---

### WALLET-3: Desktop Wallet Missing DEX UI
**Priority:** HIGH
**Location:** `wallet/desktop/`

**Agent Instructions:**
1. Create DEX component: `src/components/DEX/SwapInterface.tsx`
2. Implement: token selector, amount input, slippage settings, swap button
3. Connect to PAW chain via RPC: use existing core SDK
4. Add to main navigation
5. Test swap flow end-to-end

---

### WALLET-4: Explorer Not Production Ready
**Priority:** HIGH
**Location:** `archive/explorer/`

**Agent Instructions:**
1. Move to production: `mv archive/explorer/ explorer/`
2. Add DEX pool visualization component
3. Add Oracle price charts component
4. Add Compute job tracking component
5. Test against live pawd node
6. Add production Docker config

---

## DOCUMENTATION GAPS

### DOC-1: Validator Quickstart Guide
**Priority:** HIGH

**Agent Instructions:**
1. Create `docs/guides/VALIDATOR_QUICKSTART.md`
2. Include: hardware requirements, binary build, genesis setup, gentx, startup
3. Add systemd service file example
4. Add monitoring setup instructions
5. Test guide end-to-end on fresh machine

---

### DOC-2: DEX Trading Guide
**Priority:** MEDIUM

**Agent Instructions:**
1. Create `docs/guides/DEX_TRADING.md`
2. Include: pool creation, adding liquidity, swapping, removing liquidity
3. Add CLI examples for each operation
4. Add slippage/fee explanation
5. Add risk warnings

---

### DOC-3: API Reference
**Priority:** MEDIUM

**Agent Instructions:**
1. Generate API docs from proto files
2. Create `docs/api/README.md` with endpoint list
3. Add request/response examples for each endpoint
4. Add authentication requirements
5. Add rate limit documentation

---

## REMAINING ORIGINAL TASKS

### Phase 1: Local Testnet
- [ ] Run devnet and execute smoke tests: `docker-compose -f compose/docker-compose.devnet.yml up`
- [ ] Test validator add/remove with governance
- [ ] Test coordinated upgrade simulation

### Phase 2: Cloud Testnet
- [ ] Provision cloud infrastructure (GCP)
- [ ] Deploy K8s cluster
- [ ] Configure DNS
- [ ] Deploy public faucet and explorer
- [ ] Establish IBC channel to Cosmos Hub testnet

### Phase 3: Security Hardening
- [ ] Run `make security-audit`
- [ ] Complete internal security review
- [ ] Engage external audit firm (Trail of Bits, Halborn)
- [ ] Launch bug bounty program
- [ ] Remediate all critical/high findings

### Phase 4: Production Preparation
- [ ] Tag v1.0.0 release
- [ ] Build signed binaries for all platforms
- [ ] Finalize mainnet genesis
- [ ] Coordinate genesis ceremony with 20+ validators

### Phase 5: Mainnet Launch
- [ ] Collect gentx submissions
- [ ] Distribute final genesis with checksum
- [ ] Coordinated launch
- [ ] 24/7 monitoring first week

---

## Quick Commands

```bash
# Build
make build

# Test
make test
make test-unit
make test-coverage

# Security
make security-audit
make scan-secrets

# Local testnet
docker-compose -f compose/docker-compose.devnet.yml up

# Monitoring
docker-compose -f compose/docker-compose.monitoring.yml up -d
```

---

## Task Summary

| Priority | Count | Category |
|----------|-------|----------|
| CRITICAL | 8 | Security vulnerabilities that enable fund theft |
| HIGH | 7 | Security issues with significant impact |
| MEDIUM | 5 | Code quality and robustness issues |
| TEST | 5 | Test coverage gaps blocking production |
| WALLET | 4 | Wallet/frontend production gaps |
| DOC | 3 | Documentation gaps |
| PHASE | 5 | Original roadmap phases |

**Total New Tasks Added:** 37

**All tasks include explicit agent instructions for implementation.**
