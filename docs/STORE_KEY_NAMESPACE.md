# Store Key Namespace Separation

## Overview

PAW blockchain implements strict namespace separation for store keys across all modules to prevent key collisions and ensure data integrity. Each module uses a unique namespace byte prefix for all its store keys.

## Module Namespaces

| Module | Namespace Byte | Description |
|--------|---------------|-------------|
| Compute | `0x01` | Off-chain computation and ZK proof verification |
| DEX | `0x02` | Decentralized exchange and liquidity pools |
| Oracle | `0x03` | Price oracle and validator submissions |

## Key Structure

All store keys follow a two-byte prefix structure:

```
[Module Namespace (1 byte)] + [Sub-Prefix (1 byte)] + [Key Data (variable)]
```

### Example

**Compute Module ParamsKey:**
- Old format (pre-migration): `[]byte{0x01}`
- New format: `[]byte{0x01, 0x01}`
  - First `0x01`: Compute module namespace
  - Second `0x01`: Params sub-prefix

**DEX Module ParamsKey:**
- Old format: `[]byte{0x05}`
- New format: `[]byte{0x02, 0x05}`
  - First `0x02`: DEX module namespace
  - Second `0x05`: Params sub-prefix

## Complete Key Mappings

### Compute Module (Namespace: 0x01)

| Key Name | Old Prefix | New Prefix | Purpose |
|----------|-----------|------------|---------|
| ParamsKey | `0x01` | `0x01, 0x01` | Module parameters |
| ProviderKeyPrefix | `0x02` | `0x01, 0x02` | Provider storage |
| RequestKeyPrefix | `0x03` | `0x01, 0x03` | Request storage |
| ResultKeyPrefix | `0x04` | `0x01, 0x04` | Result storage |
| NextRequestIDKey | `0x05` | `0x01, 0x05` | Request ID counter |
| RequestsByRequesterPrefix | `0x06` | `0x01, 0x06` | Index by requester |
| RequestsByProviderPrefix | `0x07` | `0x01, 0x07` | Index by provider |
| RequestsByStatusPrefix | `0x08` | `0x01, 0x08` | Index by status |
| ActiveProvidersPrefix | `0x09` | `0x01, 0x09` | Active providers index |
| NonceKeyPrefix | `0x0A` | `0x01, 0x0A` | Nonce tracking |
| GovernanceParamsKey | `0x0B` | `0x01, 0x0B` | Governance parameters |
| DisputeKeyPrefix | `0x0C` | `0x01, 0x0C` | Dispute storage |
| EvidenceKeyPrefix | `0x0D` | `0x01, 0x0D` | Evidence storage |
| SlashRecordKeyPrefix | `0x0E` | `0x01, 0x0E` | Slash record storage |
| AppealKeyPrefix | `0x0F` | `0x01, 0x0F` | Appeal storage |
| NextDisputeIDKey | `0x10` | `0x01, 0x10` | Dispute ID counter |
| NextSlashIDKey | `0x11` | `0x01, 0x11` | Slash ID counter |
| NextAppealIDKey | `0x12` | `0x01, 0x12` | Appeal ID counter |
| DisputesByRequestPrefix | `0x13` | `0x01, 0x13` | Index disputes by request |
| DisputesByStatusPrefix | `0x14` | `0x01, 0x14` | Index disputes by status |
| SlashRecordsByProviderPrefix | `0x15` | `0x01, 0x15` | Index slash records by provider |
| AppealsByStatusPrefix | `0x16` | `0x01, 0x16` | Index appeals by status |
| CircuitParamsKeyPrefix | `0x17` | `0x01, 0x17` | ZK circuit parameters |
| ZKMetricsKey | `0x18` | `0x01, 0x18` | ZK verification metrics |
| VerificationProofHashPrefix | `0x19` | `0x01, 0x19` | Proof hash storage |
| ProviderSigningKeyPrefix | `0x1A` | `0x01, 0x1A` | Provider signing keys |
| RequestFinalizedPrefix | `0x1B` | `0x01, 0x1B` | Request settlement flags |
| ProviderStatsKeyPrefix | `0x1C` | `0x01, 0x1C` | Provider statistics |
| EscrowTimeoutReversePrefix | `0x1D` | `0x01, 0x1D` | Escrow timeout index |
| NonceByHeightPrefix | `0x1E` | `0x01, 0x1E` | Nonce cleanup index |
| ProvidersByReputationPrefix | `0x1F` | `0x01, 0x1F` | Reputation-sorted providers |
| CatastrophicFailureKeyPrefix | `0x23` | `0x01, 0x23` | Catastrophic failure records |
| NextCatastrophicFailureIDKey | `0x24` | `0x01, 0x24` | Failure ID counter |
| IBCPacketKeyPrefix | `0x25` | `0x01, 0x25` | IBC packet tracking |
| ProviderCacheKeyPrefix | `0x26` | `0x01, 0x26` | Provider cache |
| ProviderCacheMetadataKey | `0x27` | `0x01, 0x27` | Cache metadata |
| IBCPacketNonceKeyPrefix | `0x0D` | `0x01, 0x28` | IBC nonce tracking (FIXED COLLISION) |

### DEX Module (Namespace: 0x02)

| Key Name | Old Prefix | New Prefix | Purpose |
|----------|-----------|------------|---------|
| PoolKeyPrefix | `0x01` | `0x02, 0x01` | Pool storage |
| PoolCountKey | `0x02` | `0x02, 0x02` | Pool ID counter |
| PoolByTokensKeyPrefix | `0x03` | `0x02, 0x03` | Index pools by token pair |
| LiquidityKeyPrefix | `0x04` | `0x02, 0x04` | Liquidity positions |
| ParamsKey | `0x05` | `0x02, 0x05` | Module parameters |
| CircuitBreakerKeyPrefix | `0x06` | `0x02, 0x06` | Circuit breaker state |
| LastLiquidityActionKeyPrefix | `0x07` | `0x02, 0x07` | Last liquidity action |
| ReentrancyLockKeyPrefix | `0x08` | `0x02, 0x08` | Reentrancy locks |
| PoolLPFeeKeyPrefix | `0x09` | `0x02, 0x09` | LP fees per pool |
| ProtocolFeeKeyPrefix | `0x0A` | `0x02, 0x0A` | Protocol fees |
| LiquidityShareKeyPrefix | `0x0B` | `0x02, 0x0B` | Liquidity shares |
| RateLimitKeyPrefix | `0x0C` | `0x02, 0x0C` | Rate limiting |
| RateLimitByHeightPrefix | `0x0D` | `0x02, 0x0D` | Rate limit cleanup index |
| PoolTWAPKeyPrefix | `0x0E` | `0x02, 0x0E` | TWAP snapshots |
| ActivePoolsKeyPrefix | `0x15` | `0x02, 0x15` | Active pools |
| IBCPacketNonceKeyPrefix | `0x0D` | `0x02, 0x16` | IBC nonce tracking (FIXED COLLISION) |

### Oracle Module (Namespace: 0x03)

| Key Name | Old Prefix | New Prefix | Purpose |
|----------|-----------|------------|---------|
| ParamsKey | `0x01` | `0x03, 0x01` | Module parameters |
| PriceKeyPrefix | `0x02` | `0x03, 0x02` | Price storage |
| ValidatorPriceKeyPrefix | `0x03` | `0x03, 0x03` | Validator price submissions |
| ValidatorOracleKeyPrefix | `0x04` | `0x03, 0x04` | Validator oracle info |
| PriceSnapshotKeyPrefix | `0x05` | `0x03, 0x05` | Price snapshots |
| FeederDelegationKeyPrefix | `0x06` | `0x03, 0x06` | Feeder delegations |
| SubmissionByHeightPrefix | `0x07` | `0x03, 0x07` | Submission cleanup index |
| ValidatorAccuracyKeyPrefix | `0x08` | `0x03, 0x08` | Validator accuracy stats |
| AccuracyBonusPoolKey | `0x09` | `0x03, 0x09` | Accuracy bonus pool |
| GeographicInfoKeyPrefix | `0x0A` | `0x03, 0x0A` | Geographic information |
| OutlierHistoryKeyPrefix | `0x0C` | `0x03, 0x0C` | Outlier history |
| IBCPacketNonceKeyPrefix | `0x0D` | `0x03, 0x0D` | IBC nonce tracking (FIXED COLLISION) |

## Critical Fix: IBCPacketNonceKeyPrefix

**Problem Identified:**
All three modules (Compute, DEX, Oracle) used `IBCPacketNonceKeyPrefix = 0x0D`, causing data collisions.

**Solution:**
Each module now has a unique namespaced version:
- Compute: `[]byte{0x01, 0x28}`
- DEX: `[]byte{0x02, 0x16}`
- Oracle: `[]byte{0x03, 0x0D}`

## Migration

### For Existing Chains

Use the provided migration helpers to upgrade from old to new key format:

```go
// In upgrade handler
import (
    computekeeper "github.com/paw-chain/paw/x/compute/keeper"
    dexkeeper "github.com/paw-chain/paw/x/dex/keeper"
    oraclekeeper "github.com/paw-chain/paw/x/oracle/keeper"
)

func(UpgradeHandler(ctx sdk.Context, ...) error {
    // Migrate compute module keys
    if err := app.ComputeKeeper.MigrateStoreKeys(ctx); err != nil {
        return err
    }

    // Migrate DEX module keys
    if err := app.DEXKeeper.MigrateStoreKeys(ctx); err != nil {
        return err
    }

    // Migrate oracle module keys
    if err := app.OracleKeeper.MigrateStoreKeys(ctx); err != nil {
        return err
    }

    return nil
}
```

### Migration Helper Functions

Each module provides:
- `MigrateStoreKeys(ctx)`: Migrates all keys in the module
- `GetOldKey(namespacedKey)`: Converts new key to old format (for reading during migration)
- `GetNewKey(oldKey)`: Converts old key to new format

## Testing

Comprehensive tests verify:
1. **Module namespace uniqueness**: Each module has a unique namespace byte
2. **Key namespacing**: All keys start with the correct module namespace
3. **No collisions**: No keys collide across modules
4. **IBC nonce key uniqueness**: The previously colliding `IBCPacketNonceKeyPrefix` is now unique
5. **Migration helpers**: Helper functions correctly convert between old and new formats

Run tests:
```bash
go test ./tests/store_key_namespace_test.go -v
```

## Benefits

1. **Data Integrity**: Prevents accidental data overwrites between modules
2. **Security**: Reduces attack surface for key collision exploits
3. **Maintainability**: Clear organization of module data
4. **Debugging**: Easier to identify which module owns a key
5. **Future-Proof**: Easy to add new modules without key conflicts

## References

- Implementation: `x/{compute,dex,oracle}/keeper/keys.go`
- Migration: `x/{compute,dex,oracle}/keeper/migration.go`
- Tests: `tests/store_key_namespace_test.go`
- Analysis: `KEY_NAMESPACE_ANALYSIS.md`
