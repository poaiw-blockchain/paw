# PAW Blockchain State Management

## Overview

PAW uses a hierarchical key-value store with namespace separation at the module level. Each module is assigned a unique namespace byte (0x01-0x03) as the first byte of all its store keys, ensuring complete isolation between module state spaces.

## Module Namespaces

| Module    | Namespace | Description                          |
|-----------|-----------|--------------------------------------|
| `compute` | `0x01`    | Distributed compute marketplace      |
| `dex`     | `0x02`    | Decentralized exchange and liquidity |
| `oracle`  | `0x03`    | Price oracle and validator voting    |

## Key Format Pattern

All keys follow the format:
```
[ModuleNamespace:1byte][KeyPrefix:1byte][Data:variable]
```

Where:
- **ModuleNamespace**: Module identifier (0x01, 0x02, or 0x03)
- **KeyPrefix**: Specific data type within module (0x01-0xFF)
- **Data**: Variable-length data (addresses, IDs, strings)

## Compute Module (0x01)

| Key Prefix | Name                       | Format                              | Description                        |
|------------|----------------------------|-------------------------------------|------------------------------------|
| `0x01`     | ParamsKey                  | `0x01 0x01`                         | Module parameters                  |
| `0x02`     | ComputeRequestKeyPrefix    | `0x01 0x02 + requestID`             | Compute job requests               |
| `0x03`     | ProviderKeyPrefix          | `0x01 0x03 + address`               | Compute provider registrations     |
| `0x04`     | EscrowKeyPrefix            | `0x01 0x04 + requestID`             | Payment escrow accounts            |
| `0x05`     | JobStatusKeyPrefix         | `0x01 0x05 + requestID`             | Job execution status               |
| `0x06`     | NonceTrackerKeyPrefix      | `0x01 0x06 + address`               | Request nonce tracking             |
| `0x07`     | ResultKeyPrefix            | `0x01 0x07 + requestID`             | Compute job results                |
| `0x28`     | IBCPacketNonceKeyPrefix    | `0x01 0x28 + channelID/sender`      | IBC replay protection              |

### Key Construction Functions

- **IBC Packet Nonce**: `GetIBCPacketNonceKey(channelID, sender)` → `0x01 0x28 + channelID + "/" + sender`

## DEX Module (0x02)

| Key Prefix | Name                         | Format                              | Description                        |
|------------|------------------------------|-------------------------------------|------------------------------------|
| `0x01`     | PoolKeyPrefix                | `0x02 0x01 + poolID`                | Liquidity pool state               |
| `0x02`     | PoolCountKey                 | `0x02 0x02`                         | Next pool ID counter               |
| `0x03`     | PoolByTokensKeyPrefix        | `0x02 0x03 + tokenA + tokenB`       | Pool lookup by token pair          |
| `0x04`     | LiquidityKeyPrefix           | `0x02 0x04 + poolID + address`      | Liquidity positions                |
| `0x05`     | ParamsKey                    | `0x02 0x05`                         | Module parameters                  |
| `0x06`     | CircuitBreakerKeyPrefix      | `0x02 0x06 + poolID`                | Circuit breaker state              |
| `0x07`     | LastLiquidityActionKeyPrefix | `0x02 0x07 + poolID + address`      | Last action block height           |
| `0x08`     | ReentrancyLockKeyPrefix      | `0x02 0x08 + poolID`                | Reentrancy protection locks        |
| `0x09`     | PoolLPFeeKeyPrefix           | `0x02 0x09 + poolID + token`        | LP fee accumulation per pool       |
| `0x0A`     | ProtocolFeeKeyPrefix         | `0x02 0x0A + token`                 | Protocol fee accumulation          |
| `0x0B`     | LiquidityShareKeyPrefix      | `0x02 0x0B + poolID + address`      | LP share ownership                 |
| `0x16`     | IBCPacketNonceKeyPrefix      | `0x02 0x16 + channelID/sender`      | IBC replay protection              |

### Key Construction Functions

- **Pool LP Fee**: `GetPoolLPFeeKey(poolID, token)` → `0x02 0x09 + poolID(uint64) + token`
- **Protocol Fee**: `GetProtocolFeeKey(token)` → `0x02 0x0A + token`
- **Liquidity Share**: `GetLiquidityShareKey(poolID, provider)` → `0x02 0x0B + poolID(uint64) + provider(address)`
- **IBC Packet Nonce**: `GetIBCPacketNonceKey(channelID, sender)` → `0x02 0x16 + channelID + "/" + sender`

## Oracle Module (0x03)

| Key Prefix | Name                       | Format                              | Description                        |
|------------|----------------------------|-------------------------------------|------------------------------------|
| `0x01`     | ParamsKey                  | `0x03 0x01`                         | Module parameters                  |
| `0x02`     | PriceKeyPrefix             | `0x03 0x02 + denom`                 | Current price feeds                |
| `0x03`     | ValidatorKeyPrefix         | `0x03 0x03 + address`               | Oracle validator registration      |
| `0x04`     | FeederDelegationKeyPrefix  | `0x03 0x04 + validator`             | Feeder delegation mappings         |
| `0x05`     | MissCounterKeyPrefix       | `0x03 0x05 + validator`             | Missed vote counters               |
| `0x06`     | AggregateVoteKeyPrefix     | `0x03 0x06 + validator`             | Aggregated validator votes         |
| `0x07`     | PrevoteKeyPrefix           | `0x03 0x07 + validator`             | Prevote commitments (hash)         |
| `0x08`     | VoteKeyPrefix              | `0x03 0x08 + validator`             | Revealed votes                     |
| `0x09`     | DelegateKeyPrefix          | `0x03 0x09 + validator`             | Validator delegations              |
| `0x0A`     | SlashingKeyPrefix          | `0x03 0x0A + validator`             | Slashing penalties                 |
| `0x0B`     | TWAPKeyPrefix              | `0x03 0x0B + denom`                 | Time-weighted average prices       |
| `0x0D`     | IBCPacketNonceKeyPrefix    | `0x03 0x0D + channelID/sender`      | IBC replay protection              |
| `0x0E`     | EmergencyPauseStateKey     | `0x03 0x0E`                         | Emergency pause state              |

### Key Construction Functions

- **IBC Packet Nonce**: `GetIBCPacketNonceKey(channelID, sender)` → `0x03 0x0D + channelID + "/" + sender`

## Key Prefix Allocation Rules

### Reserved Ranges
- `0x01-0x0F`: Core module functionality (params, primary entities)
- `0x10-0x1F`: Secondary indices and lookups
- `0x20-0x2F`: Cross-module features (IBC, relayer support)
- `0x30-0xFF`: Module-specific extensions

### Current Allocation Status
- **Compute**: Uses 0x01-0x07, 0x28 (7 prefixes allocated, 248 available)
- **DEX**: Uses 0x01-0x0B, 0x16 (12 prefixes allocated, 243 available)
- **Oracle**: Uses 0x01-0x0E (14 prefixes allocated, 241 available)

## Migration Considerations

### Adding New Key Prefixes
1. Select next available prefix in appropriate range
2. Update module's `types/keys.go`
3. Add entry to this document
4. Use migration handler if existing data affected

### Modifying Existing Keys
- **NEVER** change existing key prefixes in production
- Use on-chain migration with upgrade handler
- Export state → Transform → Import pattern
- Test migration on testnet first

### Key Deprecation
1. Mark as deprecated in code comments
2. Keep prefix reserved (don't reuse)
3. Remove data via explicit migration only
4. Document deprecation in CHANGELOG

## Best Practices

### Key Design
- **Fixed-length prefixes**: Use 2 bytes (namespace + prefix) for all keys
- **Deterministic ordering**: Append IDs in big-endian for range queries
- **No variable separators**: Avoid `/` or `:` in composite keys (except IBC nonces)
- **Address encoding**: Use `sdk.AccAddress.Bytes()` directly, not string form

### Security
- **Namespace isolation**: Never share prefixes between modules
- **Replay protection**: All IBC handlers use packet nonce tracking (0xXX 0x16/0x28/0x0D)
- **Reentrancy locks**: DEX uses dedicated lock keys (0x02 0x08)
- **Validate inputs**: Check key construction inputs to prevent injection

### Performance
- **Index strategically**: PoolByTokensKeyPrefix enables O(1) lookups
- **Avoid hot keys**: Don't update global counters in high-throughput paths
- **Batch operations**: Use iterator patterns for bulk updates
- **Prune old data**: Implement retention policies for historical data

## References

- Compute keys: `x/compute/types/keys.go`
- DEX keys: `x/dex/types/keys.go`
- Oracle keys: `x/oracle/types/keys.go`
- Cosmos SDK KVStore: https://docs.cosmos.network/main/core/store
