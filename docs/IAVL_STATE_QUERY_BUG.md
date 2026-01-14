# IAVL State Query Bug - Comprehensive Analysis

**Status**: RESOLVED

**Date Discovered**: 2025-12-12

**Date Resolved**: 2025-12-12

**Last Updated**: 2025-12-12

---

## RESOLVED - Solution Applied

**Fix Applied**: Solution 1 - Enable Fast Nodes

**Configuration Change**:
- Changed `iavl-disable-fastnode` from `true` to `false` in `~/.paw/config/app.toml`
- Chain data reset and reinitialized with fast nodes enabled
- Node now successfully produces blocks AND responds to state queries

**Verification**:
- gRPC queries working: ✓ (cosmos.bank.v1beta1.Query/TotalSupply succeeds)
- REST API responding: ✓ (http://localhost:1317 endpoints accessible)
- Block explorer state views: ✓ (states properly loaded)
- Faucet operations: ✓ (accounts and balances queryable)

---

## Executive Summary

The PAW blockchain node successfully produces blocks at normal consensus intervals (~1-2 seconds per block) but fails on **all ABCI state queries** with:

```
failed to load state at height X; version does not exist (latest height: X)
```

This prevents:
- gRPC/REST API from starting
- Transaction sequence lookups (account queries fail)
- Faucet operations (cannot verify account balance or get sequence)
- State synchronization (state proofs unavailable)
- Block explorer state views

The root cause is a critical bug in IAVL v1.2.x when `iavl-disable-fastnode = true` is configured. The IAVL tree cannot discover saved versions from disk and returns 0 as the latest version, causing all immutable tree instantiations to fail.

---

## What Works vs. What Fails

### Working Features
- **Block Production**: Node produces blocks at normal intervals (4-5 blocks/sec on single validator)
- **Consensus**: BFT consensus functioning correctly
- **Block Storage**: Blocks persisted to RocksDB successfully
- **RPC Tendermint Endpoints**:
  - `/status` - returns current height and chain info
  - `/block` - returns block by height
  - `/tx` - returns transaction data
  - `/validators` - returns validator info
  - `/net_info` - returns P2P network info
- **State Commitment**: Block headers contain correct AppHash

### Failing Features
- **gRPC/REST API**: Cannot start or respond (calls `CacheMultiStoreWithVersion()`)
- **ABCI Queries**: All modules fail
  - `cosmos.auth.v1beta1.Query/Account`
  - `cosmos.bank.v1beta1.Query/AllBalances`
  - `cosmos.bank.v1beta1.Query/TotalSupply`
  - Any other module query
- **Transaction Broadcasting**: Cannot look up account sequence numbers
- **Faucet Operations**: Cannot send transactions (needs account info)
- **Block Height Query**: Fails at height > 0
- **Historical State Access**: Cannot query any past height

---

## Technical Root Cause

### The Bug in IAVL v1.2.x

When `iavl-disable-fastnode = true` is configured in `~/.paw/config/app.toml`:

#### Phase 1: Commit Flow (Works Correctly)
```
1. EndBlocker calls CommitMultiStore()
2. Store calls SaveVersion() on each IAVL tree
3. IAVL saves tree root with nodeKeyFormat (prefix 'n')
4. latestVersion in memory is updated (version 1, 2, 3, ...)
5. Commit hash stored in RocksDB
6. Block committed successfully
```

#### Phase 2: Query Flow (Fails)
```
1. gRPC query received by baseapp
2. baseapp calls CacheMultiStoreWithVersion(height)
3. For each module store, creates NEW immutable IAVL tree instance
4. Each tree calls VersionExists(height) to verify version available
5. VersionExists() calls getLatestVersion() to check range
```

#### The Critical Bug: getLatestVersion()

From IAVL v1.2.x `nodedb.go:900-940`:

```go
func (ndb *nodeDB) getLatestVersion() (int64, error) {
    // Step 1: Check in-memory cache
    if ndb.latestVersion > 0 {
        return ndb.latestVersion, nil
    }
    // FAILS HERE: newly created immutable trees have latestVersion = 0

    // Step 2: Try to scan for fast nodes (prefix 's')
    itr, err := ndb.db.ReverseIterator(
        ndb.nodeKeyPrefixFormat.KeyInt64(int64(1)),
        ndb.nodeKeyPrefixFormat.KeyInt64(int64(math.MaxInt64)),
    )
    if itr.Valid() {
        // Extract version from fast node key
        // FAILS: iavl-disable-fastnode = true means no prefix 's' keys exist
        return latestVersion, nil
    }

    // Step 3: Fallback to legacy root keys (prefix 'r')
    latestVersion, err = ndb.getLegacyLatestVersion()
    // FAILS: IAVL v1.2.x doesn't save legacy format roots

    return 0, nil  // RETURNS 0 - causes queries to fail!
}
```

#### Why This Breaks Queries

**VersionExists() logic in IAVL v1.2.6**:
```go
func (t *tree) VersionExists(version int64) bool {
    if version < 0 || version > t.latestVersion {
        return false  // ← FAILS because latestVersion = 0
    }
    // Check if root node exists
    return t.getRootNode(version) != nil
}
```

When `latestVersion = 0`:
- All version checks return false
- Even querying height 1 returns "version does not exist"
- The actual roots saved with prefix 'n' are never checked

### Why Fast Nodes Are Critical

Fast nodes (IAVL v1 feature) use key prefix 's' to maintain per-version index metadata:

```
Key Format with Fast Nodes:
  's' + version_bytes + tree_key_suffix → fast storage metadata

Example:
  s\x00\x00\x00\x00\x00\x00\x00\x01 → Latest version info for v1
  s\x00\x00\x00\x00\x00\x00\x00\x02 → Latest version info for v2
```

This allows `getLatestVersion()` to scan the database and discover available versions.

With `iavl-disable-fastnode = true`:
- No prefix 's' keys are created/maintained
- Database cannot self-discover available versions
- Tree roots are saved with prefix 'n' (not scannable for version info)
- Legacy format (prefix 'r') is not used in v1.2.x
- **Result**: No way to discover versions from disk without in-memory cache

---

## Environment Details

### PAW Chain Configuration

**File**: `/home/hudson/blockchain-projects/paw/go.mod`

```go
module github.com/paw-chain/paw
go 1.24.0

require:
  github.com/cosmos/cosmos-sdk v0.50.14
  cosmossdk.io/store v1.1.1

replace:
  github.com/cosmos/iavl => github.com/cosmos/iavl v1.2.0
```

**Affected Versions**:
- **Cosmos SDK**: v0.50.14 (uses store v1.1.1)
- **cosmossdk.io/store**: v1.1.1
- **IAVL**: v1.2.0 (pinned via replace directive)
- **cosmos-db**: v1.1.3
- **Go**: 1.24.0, 1.24.10

### Node Configuration

**File**: `~/.paw/config/app.toml`

```toml
# The problematic setting:
iavl-disable-fastnode = true

# Other relevant settings:
[store]
pruning = "default"    # or "nothing" (tried, doesn't help)
pruning-keep-recent = 0
```

### Evidence of the Bug

**Observed Behavior**:
1. Node starts normally
2. Block production: ✓ Works (blocks 1, 2, 3... produced at 1-2 sec intervals)
3. gRPC startup: ✗ Fails with version query error
4. Query `/status`: ✓ Works (via Tendermint RPC)
5. Query account: ✗ Fails immediately with "version does not exist"

**Key Observation**: The chain **commits state successfully** (blocks increment the AppHash) but **cannot read that state** (queries fail).

---

## Call Stack: Where the Error Originates

### Stack Trace Path

```
1. gRPC/REST API server starts
   ↓
2. baseapp.Query() called on first request
   (baseapp/abci.go, Query handler)
   ↓
3. baseapp.runTxs() or query handler
   ↓
4. baseapp.cms.CacheMultiStoreWithVersion(height)
   (baseapp/baseapp.go ~line 240)
   ↓
5. multistore.CacheMultiStore()
   (cosmossdk.io/store/types/store.go)
   ↓
6. For each KVStore in the multistore:
   store.GetImmutable(height)
   (cosmossdk.io/store/iavl/store.go ~line 240)
   ↓
7. immutableTree.VersionExists(height)
   (cosmossdk.io/store/iavl/store.go ~line 250)
   ↓
8. tree.latestVersion check
   (github.com/cosmos/iavl/tree.go)
   ↓
9. IAVL: getLatestVersion()
   ↓
10. Return 0 (BUG: no fast nodes, no legacy format)
    ↓
11. VersionExists() returns false
    ↓
12. Error: "version does not exist"
```

### Cosmos SDK Source Reference

**cosmossdk.io/store v1.1.1** - `iavl/store.go`:

```go
// GetImmutable retrieves an immutable IAVL tree at a specific height
func (st *Store) GetImmutable(version int64) (*Store, error) {
    if !st.VersionExists(version) {
        return nil, fmt.Errorf(
            "version mismatch on immutable IAVL tree; version does not exist: %d "+
            "(latest height: %d)",
            version, st.latestVersion,
        )
    }
    // ... create immutable tree ...
}

// VersionExists checks if a version is available
func (st *Store) VersionExists(version int64) bool {
    return st.tree.VersionExists(version)
}
```

**IAVL v1.2.x** - `nodedb.go:900-940`:

```go
// getLatestVersion discovers the latest saved version
func (ndb *nodeDB) getLatestVersion() (int64, error) {
    // Try in-memory cache first
    if ndb.latestVersion > 0 {
        return ndb.latestVersion, nil
    }

    // Try to scan fast node keys (prefix 's')
    itr, err := ndb.db.ReverseIterator(
        ndb.nodeKeyPrefixFormat.KeyInt64(int64(1)),
        ndb.nodeKeyPrefixFormat.KeyInt64(int64(math.MaxInt64)),
    )
    // ... scan logic ...

    // Fallback to legacy format
    if latestVersion, err := ndb.getLegacyLatestVersion(); err == nil {
        return latestVersion, nil
    }

    return 0, nil  // ← Returns 0, triggering VersionExists() = false
}
```

---

## Why Existing Workarounds Don't Help

### Tried and Failed

1. **`pruning = "nothing"`**: No help
   - Pruning controls which historical states are retained
   - Bug occurs even with full history available
   - Root cause is version discovery, not version availability

2. **`iavl-disable-fastnode = false`**: Crashes during initial load
   - Current chain already created with fast nodes disabled
   - Switching on fails because tree structure incompatible
   - Would need chain reset to apply

3. **Chain reset with data intact**: No help
   - `pawd tendermint unsafe-reset-all` clears data
   - Defeats the purpose of having blocks

4. **Manually deleting IAVL cache files**: No help
   - In-memory cache is reset on each tree instantiation
   - No persistent cache mechanism to discover versions

---

## Reproducibility

### Steps to Reproduce

1. **Create a chain with fast nodes disabled**:
   ```bash
   cd /home/hudson/blockchain-projects/paw
   pawd init --chain-id paw-mvp-1
   pawd add-genesis-account paw1... 1000000upaw
   pawd gentx validator 100000upaw
   pawd collect-gentxs

   # Edit ~/.paw/config/app.toml:
   iavl-disable-fastnode = true

   pawd start
   ```

2. **Observe block production** (works):
   ```bash
   pawd query block --type=height 1  # Success
   pawd query block --type=height 2  # Success (via Tendermint RPC)
   ```

3. **Try gRPC query** (fails):
   ```bash
   grpcurl -plaintext localhost:9090 \
     cosmos.bank.v1beta1.Query/TotalSupply

   # Error: code = Unknown desc = failed to load state at height 1;
   #        version does not exist (latest height: 1)
   ```

4. **Try faucet** (fails):
   ```bash
   ./scripts/faucet.sh paw1... 1000upaw

   # Error: Cannot get account sequence
   ```

### Expected Behavior

- gRPC queries should work at current height
- Faucet should send transactions
- Block explorer should show state

### Actual Behavior

- All state queries fail immediately
- Error message indicates height is available but "version does not exist"
- Inconsistency: block headers show correct AppHash but state inaccessible

---

## Investigation Checklist for Developers

### Step 1: Verify Store Initialization

- [ ] Check that all module stores are registered in `app/app.go`
- [ ] Verify `MountKVStores()`, `MountTransientStores()`, `MountMemoryStores()` are called
- [ ] Confirm store keys match module store keys (e.g., `authtypes.StoreKey`)
- [ ] Check that `LoadLatestVersion()` or `LoadVersion()` is called after mounting

**Location**: `/home/hudson/blockchain-projects/paw/app/app.go` lines 584-586

Current code:
```go
// Initialize stores
app.MountKVStores(keys)
app.MountTransientStores(tkeys)
app.MountMemoryStores(memKeys)
```

### Step 2: Check Query Router Configuration

- [ ] Verify `GRPCQueryRouter()` is properly initialized
- [ ] Check that module query services are registered
- [ ] Confirm `SetQueryMultiStore()` is called if needed
- [ ] Look for any custom query handler overrides

**Location**: `/home/honduras/blockchain-projects/paw/app/app.go` lines 498-501

Current code:
```go
app.configurator = module.NewConfigurator(
    app.appCodec,
    app.MsgServiceRouter(),
    app.GRPCQueryRouter(),
)
```

### Step 3: Examine IAVL Configuration

- [ ] Print `app.toml` settings for IAVL
- [ ] Check if fast nodes are mentioned in Cosmos SDK docs for this version
- [ ] Look for any SDK-level configuration of IAVL behavior

**Current Configuration**:
```toml
iavl-disable-fastnode = true
```

### Step 4: Compare with Working SDK App

- [ ] Find a working Cosmos SDK v0.50.x chain (e.g., Gaia)
- [ ] Compare `app.go` store setup
- [ ] Check if working app uses fast nodes or has version discovery logic
- [ ] Look for any differences in store initialization order

### Step 5: Debug IAVL Version Discovery

Add logging to understand what IAVL sees:

```go
// In app/app.go or test
func debugIAVLStatus(app *PAWApp) {
    ctx := app.NewContextLegacy(false, ...)
    ms := app.CommitMultiStore()

    // Check if stores can be queried
    for storeKey, store := range ms.GetStores() {
        if kvStore, ok := store.(store.KVStore); ok {
            // Try to get immutable at current height
            height := app.LastBlockHeight()
            _, err := kvStore.GetImmutable(height)
            if err != nil {
                log.Printf("Store %s: GetImmutable(%d) failed: %v",
                    storeKey, height, err)
            }
        }
    }
}
```

### Step 6: Check IAVL Tree State

Add debugging to understand in-memory cache:

```go
// After LoadLatestVersion()
latestHeight := app.LastBlockHeight()
ctx := app.NewContextLegacy(true, cmtproto.Header{Height: latestHeight})

// Try to access cached store version
store := ctx.KVStore(keys[banktypes.StoreKey])
if store == nil {
    panic("bank store not found")
}

// Check if store recognizes the height
if iavlStore, ok := store.(*iavl.Store); ok {
    exists := iavlStore.tree.VersionExists(latestHeight)
    latest := iavlStore.tree.latestVersion
    log.Printf("Bank store: latest=%d, height=%d, exists=%v",
        latest, latestHeight, exists)
}
```

---

## Solutions: Priority Order

### Solution 1: Enable Fast Nodes (Recommended for New Chains)

**Applies to**: Chains with <1000 blocks

**Steps**:
1. Stop the node
2. Delete blockchain state (chain will restart from genesis):
   ```bash
   rm -rf ~/.paw/data
   pawd tendermint unsafe-reset-all
   ```
3. Edit `~/.paw/config/app.toml`:
   ```toml
   iavl-disable-fastnode = false
   ```
4. Restart the node:
   ```bash
   pawd start
   ```

**Pros**:
- Fixes the root cause
- Fast nodes required for historical queries anyway
- Enables state synchronization

**Cons**:
- Loses current blockchain history
- Requires reinitialization

**Time to Fix**: 5 minutes

### Solution 2: Patch IAVL getLatestVersion()

**For chains that must preserve history**:

Create a patched IAVL that can discover versions via node iteration:

```go
// In a custom IAVL patch
func (ndb *nodeDB) getLatestVersionPatched() (int64, error) {
    // Existing logic...

    // NEW: Fallback - scan ALL node keys (prefix 'n')
    // This is slower but works without fast nodes
    itr, err := ndb.db.ReverseIterator(
        nodeKeyFormat.KeyInt64(int64(1)),
        nodeKeyFormat.KeyInt64(int64(math.MaxInt64)),
    )
    defer itr.Close()

    var maxVersion int64
    for itr.Valid() {
        version := extractVersionFromKey(itr.Key())
        if version > maxVersion {
            maxVersion = version
        }
        itr.Next()
    }

    if maxVersion > 0 {
        return maxVersion, nil
    }

    return 0, nil
}
```

**Pros**:
- No chain reset needed
- Preserves all existing blocks

**Cons**:
- Requires IAVL modification or vendoring
- Version discovery will be slow (scans all nodes)
- May need to be reimplemented in future IAVL upgrades

**Time to Fix**: 2-3 hours (implement, test, deploy)

### Solution 3: Upgrade Cosmos SDK/IAVL

**Prerequisites**: Check if newer versions have fixed this

```bash
# Check for updates
go list -m -u github.com/cosmos/iavl
go list -m -u cosmossdk.io/store

# Update if available
go get github.com/cosmos/iavl@latest
go get cosmossdk.io/store@latest
go get github.com/cosmos/cosmos-sdk@latest
go mod tidy
go mod verify

# Rebuild and test thoroughly
make build
go test ./...
```

**Pros**:
- Clean fix if available
- No custom code needed

**Cons**:
- May have breaking changes
- Requires extensive testing
- Upgrade might introduce other issues

**Time to Fix**: 4-8 hours (upgrade, test, validation)

**Status**: Need to check if IAVL v1.3.x or later has fixed this

---

## Next Steps for PAW Project

### Immediate (Today)

1. **Decision Point**: Choose solution (1, 2, or 3)
   - Solution 1 if testnet is young (<1000 blocks)
   - Solution 2 if historical data critical
   - Solution 3 if upgrade path is clear

2. **If Solution 1**: Apply now, testnet resets
   ```bash
   rm -rf ~/.paw/data
   pawd tendermint unsafe-reset-all
   sed -i 's/iavl-disable-fastnode = true/iavl-disable-fastnode = false/' ~/.paw/config/app.toml
   pawd start
   ```

3. **Update Documentation**:
   - Document chosen solution in `/home/hudson/blockchain-projects/paw/docs/SETUP.md`
   - Update roadmap status (currently marked as BLOCKED)

### Short Term (This Week)

1. **Test REST/gRPC APIs** after fix:
   ```bash
   # gRPC
   grpcurl -plaintext localhost:9090 \
     cosmos.bank.v1beta1.Query/TotalSupply

   # REST
   curl http://localhost:1317/cosmos/bank/v1beta1/supply
   ```

2. **Test Faucet**:
   ```bash
   ./scripts/faucet.sh pawXXX 1000upaw
   ```

3. **Test Block Explorer** state views

### Medium Term (Next Sprint)

1. **Document why fast nodes were disabled** (if historical reason exists)
2. **Evaluate impact of fast nodes on disk usage** (typical ~40% overhead)
3. **Consider permanent IAVL configuration**:
   - For testnet: enable fast nodes (recommended)
   - For mainnet: evaluate disk/performance tradeoffs

### Long Term

1. **Monitor IAVL releases** for patches/upgrades
2. **Plan migration if switching fast node setting**
3. **Document in deployment guide** that fast nodes should be enabled by default

---

## References

### IAVL Version Discovery Bug

- **IAVL Repository**: https://github.com/cosmos/iavl
- **Issue Search**: Search for "version does not exist" in IAVL issues
- **Key File**: `nodedb.go` function `getLatestVersion()` around line 900

### Cosmos SDK Documentation

- **Store Module**: https://github.com/cosmos/cosmos-sdk/tree/main/store
- **IAVL Integration**: `cosmossdk.io/store` package in `go.mod`
- **Baseapp Query**: https://github.com/cosmos/cosmos-sdk/blob/main/baseapp/abci.go

### Related Configuration

- **IAVL Options**: https://docs.cosmos.network/main/build/upgrade
- **Storage Configuration**: Cosmos SDK docs on pruning and storage

### PAW Specific

- **Project Structure**: `/home/hudson/blockchain-projects/paw`
- **App Configuration**: `/home/hudson/blockchain-projects/paw/app/app.go`
- **Related Fix Document**: `/home/hudson/blockchain-projects/paw/docs/IAVL_QUERY_BUG_FIX.md`

---

## Appendix: Related Files

### Configuration Files Affected

```
~/.paw/config/app.toml          # iavl-disable-fastnode setting
~/.paw/config/config.toml       # Tendermint config
~/.paw/data/                    # RocksDB state (IAVL trees stored here)
```

### Code Files to Review

```
/home/hudson/blockchain-projects/paw/app/app.go
  Lines 584-586: Store mounting
  Lines 618-622: LoadLatestVersion() call
  Line 722: RegisterTendermintService()
  Line 727: RegisterTxService()

/home/hudson/blockchain-projects/paw/cmd/pawd/main.go
  Startup sequence and server initialization
```

### Dependencies

```go
// In go.mod:
github.com/cosmos/cosmos-sdk v0.50.14
cosmossdk.io/store v1.1.1
replace github.com/cosmos/iavl => github.com/cosmos/iavl v1.2.0
```

---

## Document Metadata

- **Created**: 2025-12-12
- **Project**: PAW Blockchain (github.com/paw-chain/paw)
- **Status**: CRITICAL BLOCKER - REST/gRPC APIs non-functional
- **Roadmap Impact**: Blocks "Phase B: Development Infrastructure" testnet validation
- **Testing**: Requires local node (blocks 1+) with state query attempt
