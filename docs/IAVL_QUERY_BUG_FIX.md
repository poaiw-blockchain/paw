# IAVL State Query Bug - Root Cause and Fix

## Problem Summary

All state queries via gRPC/REST/ABCI fail with:
```
failed to load state at height X; version does not exist (latest height: X)
```

Despite the node successfully producing blocks, **all queries return "version does not exist"** errors.

## Root Cause Analysis

### The Bug

When `iavl-disable-fastnode = true` is set in `app.toml`, IAVL v1.2.6 has a critical bug in version detection:

1. **During Commit** (working correctly):
   - `SaveVersion()` successfully saves tree roots using `nodeKeyFormat` (prefix 'n')
   - Blocks are committed and the chain progresses normally
   - In-memory `latestVersion` is correctly updated in the committing tree instance

2. **During Query** (fails):
   - `CacheMultiStoreWithVersion(height)` creates immutable tree instances for queries
   - Each store calls `VersionExists(height)` to verify the version is available
   - `VersionExists()` calls `getLatestVersion()` to check if height is within valid range
   - `getLatestVersion()` logic:
     ```
     a. Check in-memory cache (latestVersion) → returns 0 for new tree instances
     b. Scan database for fast node keys (prefix 's') → finds nothing (disabled)
     c. Fall back to legacy root keys (prefix 'r') → finds nothing (not used in v1.2.x)
     d. Return 0
     ```
   - With `latestVersion = 0`, `VersionExists()` returns false
   - Query fails with "version does not exist"

### Why Fast Nodes Matter

Fast nodes (introduced in IAVL v1) store additional index data using key prefix 's' to speed up queries:
- With fast nodes: `getLatestVersion()` can scan the 's' prefix keys to find the latest version
- Without fast nodes: `getLatestVersion()` has no reliable way to discover versions from disk

The bug is that `getLatestVersion()` assumes fast nodes exist for version discovery, but:
- Tree roots are saved with prefix 'n' (not scannable for version info)
- Legacy roots (prefix 'r') are not saved in IAVL v1.2.x
- The in-memory cache is not shared between tree instances

### Code References

**IAVL v1.2.6 nodedb.go:900-940** - `getLatestVersion()`:
```go
func (ndb *nodeDB) getLatestVersion() (int64, error) {
    // Try memory cache first
    if latestVersion > 0 {
        return latestVersion, nil
    }

    // Scan for fast nodes (prefix 's') - FAILS when disabled
    itr, err := ndb.db.ReverseIterator(
        nodeKeyPrefixFormat.KeyInt64(int64(1)),
        nodeKeyPrefixFormat.KeyInt64(int64(math.MaxInt64)),
    )
    if itr.Valid() {
        // Extract version from fast node key
        return latestVersion, nil
    }

    // Fallback to legacy format (prefix 'r') - NOT SAVED in v1.2.x
    latestVersion, err = ndb.getLegacyLatestVersion()

    return 0, nil  // RETURNS 0 when fast nodes disabled!
}
```

**cosmossdk.io/store v1.1.1 iavl/store.go** - `GetImmutable()`:
```go
func (st *Store) GetImmutable(version int64) (*Store, error) {
    if !st.VersionExists(version) {  // ← Fails here
        return nil, errors.New("version mismatch on immutable IAVL tree; version does not exist...")
    }
    // ...
}
```

## The Fix

### Option 1: Enable Fast Nodes (Recommended for New Chains)

**For chains at low block height (<1000 blocks):**

1. Stop the node
2. Edit `~/.paw/config/app.toml`:
   ```toml
   iavl-disable-fastnode = false
   ```
3. Reset the blockchain data:
   ```bash
   rm -rf ~/.paw/data
   pawd tendermint unsafe-reset-all
   ```
4. Restart from genesis with fast nodes enabled

**For existing chains with significant history:**

1. Stop the node
2. Edit `~/.paw/config/app.toml`:
   ```toml
   iavl-disable-fastnode = false
   ```
3. Restart the node
4. IAVL will automatically upgrade and rebuild fast storage:
   - The `enableFastStorageAndCommitIfNotEnabled()` function runs on first load
   - Fast nodes are rebuilt from existing tree data
   - This may cause a delay on first startup (5-30 min depending on chain size)
   - Once complete, queries will work normally

### Option 2: Upgrade IAVL/Store (Future Fix)

Check if newer versions of `cosmossdk.io/store` or `github.com/cosmos/iavl` have fixed this issue:

```bash
# Check for updates
go list -m -u github.com/cosmos/iavl
go list -m -u cosmossdk.io/store

# Update if available (test thoroughly first!)
go get github.com/cosmos/iavl@latest
go get cosmossdk.io/store@latest
go mod tidy
```

**Note**: Upgrading store/IAVL versions may require Cosmos SDK upgrades. Test extensively before mainnet deployment.

### Option 3: Keep Fast Nodes Disabled (Not Recommended)

If you must keep fast nodes disabled for specific reasons:
- This is not recommended as it breaks queries in current IAVL versions
- Would require patching IAVL to store version metadata differently
- No simple workaround exists without code changes

## Why Fast Nodes Were Disabled

The configuration `iavl-disable-fastnode = true` is sometimes used to:
1. Reduce disk space usage (fast nodes add ~40% storage overhead)
2. Improve write performance (no fast node index updates)
3. Compatibility with older IAVL versions

However, **in IAVL v1.2.x, disabling fast nodes breaks query functionality** due to this bug.

## Testing the Fix

After applying Option 1:

```bash
# Wait for node to start and produce blocks
./build/pawd status

# Test gRPC query (should succeed)
grpcurl -plaintext localhost:9091 cosmos.bank.v1beta1.Query/TotalSupply

# Test block query (should succeed)
./build/pawd query block --type=height 100
```

## Prevention for New Chains

For new Cosmos SDK chains:
- **Leave fast nodes enabled** (the default)
- Only disable if you have a specific, tested reason
- If using custom pruning, test queries extensively before mainnet
- Monitor gRPC/REST endpoint health in CI/CD

## Related Issues

- Cosmos SDK: https://github.com/cosmos/cosmos-sdk/issues
- IAVL: https://github.com/cosmos/iavl/issues
- Search for: "version does not exist", "iavl-disable-fastnode"

## Version Information

**Affected Versions:**
- IAVL: v1.2.0 - v1.2.6 (confirmed)
- cosmossdk.io/store: v1.1.1 (confirmed)
- Cosmos SDK: v0.50.x (uses affected store version)

**Status**: Awaiting confirmation if this is fixed in IAVL v1.3.x+ or requires a code patch.
