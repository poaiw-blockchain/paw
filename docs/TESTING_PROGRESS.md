## PAW DEX Testing Progress (Jan 15, 2026)

This file mirrors AGENTS.md progress entries for quick reference inside the repo.

Recent batches:
- **Batch 12**: CancelSwapCommitment expiry index deletion verified.
- **Batch 11**: TWAP stale fallback, rate-limit no-op guard, and rate-limit pruning within cutoff window.
- **Batch 10**: Rate-limit cleanup, commit cancel trader index removal, fee_collector fixture support.
- **Batch 9**: Swap → active/TWAP side effects and oversized multihop rejection.
- **Commit lifecycle coverage**: CommitSwap (duplicate guard), CancelSwapCommitment (refund/fee + trader & expiry index removal), CleanupExpiredSwapCommits (gov path), Commit-reveal expiry error paths.
- **Batch 20**: Cross-chain aggregation cache freshness, missing-connection tolerance, local-route fallback, empty-route guard, local slippage-bound happy path, and slippage-exceeded error for ExecuteCrossChainSwap.
- **Batch 21/22**: Remote swap ACK success/failure paths, timeout refunds, channel-capability failure refund, and 5m cache staleness boundary.
- **Batch 23**: IBC mock added; remote happy-path ExecuteCrossChainSwap covered with successful SendPacket.
- **Batch 24**: Oracle stale-price fallback when no submissions, collusion/flash-loan detection hardening, circuit-breaker legacy recovery cleanup, and removal of obsolete compute getter tests (build fixed).
- **Batch 25**: Oracle price override lifecycle, override fallback, slashing disable/enable, whitelist add/remove/is checks, and feed circuit-breaker open/close coverage.
- **Batch 26**: Compute circuit-breaker lifecycle (global + provider), job cancellation persistence, reputation overrides with fallback, and ZK metrics default/storage round-trips.
- **Batch 27**: Compute migration helpers (store key migration), signing key registration/rotation, batch submit gas guards, catastrophic failure queries, simulate request validation, pending-request counting, and randomness commitment delete path.

See `/home/hudson/blockchain-projects/paw-dev-team/PAW-SDK-API-TESTING-PLAN.md` for the authoritative, date-stamped list.
