# PAW Coding Conventions

## Naming Patterns

### Go Identifiers (camelCase)
- **Private fields**: `circuitManager`, `moduleAddressCache`, `tokenGraphCache`
- **Private methods**: `getStore()`, `getModuleAccountAddress()`
- **Store key prefixes**: `LimitOrderKeyPrefix`, `PoolKeyPrefix`

### Protobuf Fields (snake_case)
- **Message fields**: `pool_id`, `token_in`, `amount_out`
- **Enum values**: `ORDER_TYPE_BUY`, `ORDER_STATUS_FILLED`

## Store Access Pattern

All keepers must use the defensive `getStore()` pattern:

```go
type kvStoreProvider interface {
    KVStore(key storetypes.StoreKey) storetypes.KVStore
}

func (k Keeper) getStore(ctx context.Context) storetypes.KVStore {
    if provider, ok := ctx.(kvStoreProvider); ok {
        return provider.KVStore(k.storeKey)
    }
    unwrapped := sdk.UnwrapSDKContext(ctx)
    return unwrapped.KVStore(k.storeKey)
}
```

This handles both `sdk.Context` and direct `kvStoreProvider` implementations.

## Serialization

- **State storage**: Always use protobuf (`k.cdc.Marshal`/`Unmarshal`)
- **JSON**: Only for external API responses, never for on-chain state

## Module Structure

```
x/<module>/
  keeper/
    keeper.go      # Core keeper with getStore()
    keys.go        # Store key prefixes
    security.go    # Security-sensitive operations
  types/
    types.go       # Constants and helpers
    errors.go      # Sentinel errors (ErrXxx)
    *.pb.go        # Generated protobuf types
```
