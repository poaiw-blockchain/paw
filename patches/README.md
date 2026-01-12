# PAW Custom Patches

Patches applied to forked dependencies in `.tmp/` directory.

## IAVL Debug Log Level Fix

**File**: `.tmp/iavl/nodedb.go` (line ~974)

**Problem**: State-synced nodes log "iavl root missing" at ERROR level when querying historical versions that predate the state-sync point. This is expected behavior but creates log noise.

**Fix**: Change log level from `Error` to `Debug` and update message.

```diff
- ndb.logger.Error(
-     "iavl root missing",
+ ndb.logger.Debug(
+     "iavl root missing (expected for state-synced nodes)",
      "version", version,
      "latest_version", ndb.latestVersion,
      "first_version", ndb.firstVersion,
  )
```

**Apply manually** after cloning `.tmp/iavl`:
```bash
sed -i 's/ndb.logger.Error(/ndb.logger.Debug(/; s/"iavl root missing"/"iavl root missing (expected for state-synced nodes)"/' .tmp/iavl/nodedb.go
```

Then rebuild: `make build`
