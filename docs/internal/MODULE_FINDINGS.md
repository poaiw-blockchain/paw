# Per-Module Findings and Regression Tests

## x/compute - Findings Fixed

| Severity | Issue | Fix |
|----------|-------|-----|
| CRITICAL | Escrow double-lock race condition | Atomic `SetEscrowStateIfNotExists()` |
| CRITICAL | Key prefix collision (0x19) | Changed NonceByHeightPrefix to 0x1E |
| HIGH | Catastrophic failure log (events only) | Persistent `CatastrophicFailure` proto type |
| HIGH | Request iteration N+1 pattern | Batch prefetching for O(n) complexity |
| HIGH | Provider selection O(n) scan | Secondary index by reputation |
| HIGH | ZK circuits lazy init | Initialize in `InitGenesis()` |
| MEDIUM | Escrow timeout index invariant | Added `EscrowTimeoutIndexInvariant()` |
| MEDIUM | IBC channel cache | Removed premature optimization |

## x/dex - Findings Fixed

| Severity | Issue | Fix |
|----------|-------|-----|
| CRITICAL | Constant product too permissive (50%) | Tightened to 99.9%-110% bounds |
| CRITICAL | Pool creation tx ordering | Transfer tokens before state update |
| HIGH | IBC ack size limit (1MB DoS) | Reduced to 256KB |
| HIGH | Pagination unbounded | Added `MaxIterationLimit = 100` |
| MEDIUM | Commit-reveal for large swaps | Threshold-based scheme (5% reserves) |
| MEDIUM | Circuit breaker in genesis | Reset runtime state on import |
| MEDIUM | Multi-hop swap batching | Atomic state updates, 40% gas savings |
| LOW | Active pools cleanup | 24h TTL in EndBlocker |

## x/oracle - Findings Fixed

| Severity | Issue | Fix |
|----------|-------|-----|
| HIGH | TWAP unbounded snapshots (14,400/day) | Capped at 1000 snapshots |
| HIGH | GeoIP silent failure | `RequireGeographicDiversity` param |
| HIGH | Missing module migrator | Created Migrator with Migrate1to2 |
| MEDIUM | Volatility cap missing | Applied same 1000-snapshot limit |
| MEDIUM | Aggregation O(4n log n) | Sort once, reuse for median/MAD/IQR |
| MEDIUM | GeoIP no caching | LRU cache with TTL |
| LOW | Slash fraction too low (1%) | Increased to 5% (mainnet 7.5%) |

## Regression Test Commitments

**Must pass before any release:**
1. `x/compute/keeper/escrow_test.go` - Escrow atomicity and timeout index
2. `x/compute/keeper/invariants_test.go` - All compute invariants
3. `x/dex/keeper/invariants_test.go` - Constant product bounds validation
4. `x/dex/keeper/commit_reveal_test.go` - MEV protection scheme
5. `x/dex/keeper/genesis_test.go` - Circuit breaker state preservation
6. `x/oracle/keeper/aggregation_test.go` - TWAP/volatility caps
7. `x/oracle/keeper/migrations_test.go` - State migration integrity
8. `tests/upgrade/upgrade_simulation_test.go` - Chain upgrade paths
