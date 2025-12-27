# Changelog

Format: [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) | [Semantic Versioning](https://semver.org/spec/v2.0.0.html)

## [Unreleased] - Targeting v1.0.0

### Breaking Changes

**Store Key Namespace Migration**
- DEX module uses `0x02` namespace prefix for all keys
- **Migration**: Run `pawd migrate v1` before upgrading nodes

**IBC Channel Authentication**
- Channel operations require explicit authorization via `x/shared/ibc/types.go`
- **Migration**: Update IBC handlers to use new auth patterns

**Event Prefixes Standardized**
- `EventTypeDexOrderCancelled` -> `dex_order_cancelled`
- `EventTypeDexOrderPlaced` -> `dex_order_placed`
- **Migration**: Update event listeners to use `dex_` prefixed events

### Security Fixes

| ID | Priority | Description |
|----|----------|-------------|
| SEC-1 | P0 | Nonce race condition - implemented reservation pattern |
| SEC-2 | P0 | Provider signing key TOFU - require explicit registration |
| SEC-3 | P0 | Private IP blocking - comprehensive RFC1918/link-local/IPv6 |
| SEC-4 | P0 | Reentrancy lock expiration - `LockExpirationBlocks=2` |
| SEC-5 | P1 | Ed25519 low-order points - all 8 points rejected |
| SEC-6 | P1 | Minimum liquidity - `MinimumLiquidity=1000` enforced |
| SEC-8 | P2 | HTTPS enforcement - `RequireHTTPS` validation |
| SEC-9 | P2 | Outlier history cleanup - periodic EndBlocker cleanup |

### Performance Fixes

| ID | Priority | Description |
|----|----------|-------------|
| PERF-1 | P0 | Iterator leak fix - IIFE pattern in `CleanupOldRateLimitData` |
| PERF-2 | P1 | Cached voting power - `GetCachedTotalVotingPower()` |
| PERF-3 | P1 | Paginated iteration - `DefaultMaxLiquidityIterations=10000` |
| PERF-4-6 | P2 | Module address cache, order archival, filtered iteration |

### Data Integrity Fixes

| ID | Priority | Description |
|----|----------|-------------|
| DATA-1 | P0 | Migration key prefix - namespaced prefixes (0x02, 0x01) |
| DATA-2 | P0 | Limit order keys - added 0x02 namespace |
| DATA-3 | P0 | SwapCommit keys - consolidated to 0x02, 0x1D |
| DATA-4 | P1 | Iterator close - IIFE pattern in invariants |
| DATA-5 | P1 | Atomic fee claims - CacheContext pattern |

### New Features

- GCP 3-node testnet with deployment automation
- Block explorer and status dashboard
- Kubernetes infrastructure with StatefulSets
- Toxiproxy chaos testing integration

### Test Coverage (1684+ lines added)

- TEST-1/2: Ante handler and BeginBlocker/EndBlocker tests
- TEST-3/4: IBC timeout, provider slashing/reputation tests
- TEST-5/6: Concurrent operations, genesis export/import tests
- TEST-7/8: 45+ integration tests, 25+ error path tests
- TEST-9/10: 12+ IBC scenarios, 8+ security tests

### Documentation

- DOC-1: ADR index (ADR-004 IBC, ADR-005 Oracle, ADR-006 Compute)
- DOC-2: API reference (`docs/api/API_REFERENCE.md`)
- DOC-3: SDK guide with TypeScript, Python, Go examples

---

## Migration Guide: Pre-release to v1.0.0

### Steps
1. Backup: `cp -r ~/.paw ~/.paw-backup`
2. Stop node: `systemctl stop pawd`
3. Update: `git checkout v1.0.0 && make install`
4. Migrate: `pawd migrate v1 --home ~/.paw`
5. Verify: `pawd start --dry-run`
6. Update event listeners to `dex_` prefixed events

### Rollback
```bash
systemctl stop pawd
rm -rf ~/.paw && cp -r ~/.paw-backup ~/.paw
# Install previous binary
systemctl start pawd
```

---

## Version History

| Version | Date | Status |
|---------|------|--------|
| v1.0.0 | TBD | In Development |
| Pre-release | 2025-12-27 | Current |

[Unreleased]: https://github.com/paw-chain/paw/compare/main...HEAD
