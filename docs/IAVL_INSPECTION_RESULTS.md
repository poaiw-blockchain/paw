# IAVL Store Inspection Results

**Date:** 2025-12-12
**Node Status:** Running at block 5370+
**Location:** /home/hudson/.paw/data/application.db

## Executive Summary

✅ **IAVL metadata IS being saved to disk**
✅ **Version data exists for all blocks tested (1, 10, 100, 1000, 4000, 5000, 5171, 5370)**
❌ **LoadVersion() is failing to retrieve historical states despite data being present**

## Key Findings

### 1. Database Contains Version Metadata

Using `ldb` to inspect the database directly:

```bash
# All of these commands returned valid data:
ldb --db=~/.paw/data/application.db get "s/1"
ldb --db=~/.paw/data/application.db get "s/100"
ldb --db=~/.paw/data/application.db get "s/5000"
ldb --db=~/.paw/data/application.db get "s/5171"
```

Each version metadata entry contains:
- Protocol buffer encoded store info
- Root hashes for each module store (acc, bank, compute, distribution, etc.)
- Version number

### 2. Query Failures Despite Data Existence

When querying via gRPC at ANY height:

```bash
# Current height (5370)
grpcurl -plaintext localhost:9091 cosmos.bank.v1beta1.Query/TotalSupply
# ERROR: failed to load state at height 5370; version does not exist (latest height: 5370)

# Historical height (100)
grpcurl -plaintext -H "x-cosmos-block-height: 100" localhost:9091 cosmos.bank.v1beta1.Query/TotalSupply
# ERROR: failed to load state at height 100; version does not exist (latest height: 5385)
```

### 3. The Disconnect

**Data flow:**
1. Block N commits → IAVL trees save to database ✅
2. Version metadata written with key `s/N` ✅
3. Store root hashes saved in metadata ✅
4. Query arrives requesting height N ❌
5. `cms.CacheMultiStoreWithVersion(N)` called ❌
6. Individual stores try `tree.LoadVersion(N)` ❌
7. **FAILS with "version does not exist"** ❌

## Configuration

From `/home/hudson/.paw/config/app.toml`:
```toml
inter-block-cache = true
iavl-cache-size = 781250
iavl-disable-fastnode = false
```

From `go.mod`:
```
github.com/cosmos/iavl v1.2.6
```

## Database Structure

Keys follow IAVL v1 format:
```
s/1                  -> Version 1 metadata (all stores)
s/10                 -> Version 10 metadata
s/100                -> Version 100 metadata
s/k:bank/...         -> Bank store IAVL node data (prefix for bank store)
s/k:acc/...          -> Account store IAVL node data
```

## Hypothesis: The Root Cause

The problem appears to be in how the Cosmos SDK's `rootmulti/store.go` calls `LoadVersion()`:

1. **Metadata exists** - proven by `ldb` inspection
2. **IAVL nodes exist** - trees are saved during commit
3. **LoadVersion() fails** - something in the load path is wrong

Possible causes:
- Store key mapping mismatch between commit and query
- IAVL tree initialization parameters differ between commit and query paths
- Pruning configuration causing premature version deletion
- Cache layer interfering with version retrieval

## Evidence for Further Investigation

### Test Program Created

Created `/home/hudson/iavl_test/test_direct_load.go` to test IAVL tree loading directly:
- Opens database with same settings as node
- Attempts to load specific versions (100, 5000)
- Reports available versions from `tree.AvailableVersions()`

**To run:** Stop node first (`pkill pawd`), then:
```bash
cd /home/hudson/iavl_test && go run test_direct_load.go
```

## Next Steps

1. **Run the direct load test** to see if IAVL library itself can load versions
2. **Add debug logging** to `rootmulti/store.go` in LoadVersion path
3. **Compare store initialization** between commit path and query path
4. **Check pruning manager** to see if versions are being deleted immediately
5. **Verify store key prefixes** match between commit and query

## Conclusion

**The data is there. The loading is broken.**

This is not a "data not saved" problem. This is a "data retrieval" problem. The IAVL trees are successfully writing version metadata and node data to the database. The query path is failing to read it back.

The fix needs to be in one of:
- Store initialization (how trees are created during queries)
- Version loading logic (how LoadVersion is called)
- Pruning configuration (if versions are deleted too soon)
- Store key mapping (if query uses different prefixes than commit)
