# Store Key Namespace Analysis

## Current State (BEFORE Fix)

### Compute Module (x/compute/keeper/keys.go)
Uses prefixes: 0x01-0x25
- 0x01: ParamsKey
- 0x02: ProviderKeyPrefix
- 0x03: RequestKeyPrefix
- 0x04: ResultKeyPrefix
- 0x05: NextRequestIDKey
- 0x06-0x16: Various indexes
- 0x17-0x1F: ZK and performance keys
- 0x23-0x25: Catastrophic failure, IBC packet tracking

### DEX Module (x/dex/keeper/keys.go)
Uses prefixes: 0x01-0x15
- 0x01: PoolKeyPrefix
- 0x02: PoolCountKey
- 0x03: PoolByTokensKeyPrefix
- 0x04: LiquidityKeyPrefix
- 0x05: ParamsKey
- 0x06-0x0E: Various pool and fee keys
- 0x15: ActivePoolsKeyPrefix

### Oracle Module (x/oracle/keeper/keys.go)
Uses prefixes: 0x01-0x0C
- 0x01: ParamsKey
- 0x02: PriceKeyPrefix
- 0x03: ValidatorPriceKeyPrefix
- 0x04: ValidatorOracleKeyPrefix
- 0x05-0x0C: Various oracle indexes

### Critical Overlaps
1. **IBCPacketNonceKeyPrefix = 0x0D**: Present in ALL THREE modules (types/keys.go)
2. **ParamsKey = 0x01**: Used by ALL THREE modules
3. Multiple overlapping prefixes across modules

## Solution: Module Namespace Separation

### Namespace Scheme
- **Compute Module**: 0x01 + existing prefix (e.g., 0x0101, 0x0102, ...)
- **DEX Module**: 0x02 + existing prefix (e.g., 0x0201, 0x0202, ...)
- **Oracle Module**: 0x03 + existing prefix (e.g., 0x0301, 0x0302, ...)

### New Key Structure
```
[Module Namespace Byte] + [Original Prefix Byte] + [Key Data]
```

### Migration Strategy
1. Keep old keys accessible via migration helpers
2. Write to new namespaced keys
3. Provide clear migration path for existing chains
4. Document breaking changes in upgrade handler
