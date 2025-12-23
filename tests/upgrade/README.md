# PAW Chain Upgrade Simulation Tests

## Overview

This directory contains comprehensive upgrade simulation tests for the PAW blockchain. These tests validate chain upgrades are safe, deterministic, and preserve state integrity during mainnet upgrades.

## Test Files

### upgrade_simulation_test.go

Production-grade upgrade simulation tests covering:

- **TestUpgradeFromV1ToV2_FullSimulation**: Complete v1→v2 upgrade with state verification
- **TestMultiVersionUpgradePath**: Sequential upgrades v1→v2→v3 with data persistence
- **TestUpgradeWithActiveOperations**: Upgrades during active DEX/compute operations
- **TestUpgradeRollbackOnMigrationFailure**: Failed migrations don't corrupt state
- **TestAppVersionBump**: App version handling during upgrades
- **TestUpgradeDeterminism**: Upgrades produce identical results
- **TestUpgradeDataIntegrity**: No data loss during migration

### upgrade_handler_test.go

Module-specific migration tests:

- **TestUpgradeFromV1ToV2**: DEX/Compute/Oracle migration validation
- **TestUpgradeRepairsComputeIndexes**: Index rebuilding verification
- **TestUpgradeRollback**: Cache context rollback on failure

### upgrade_test.go

Integration tests for upgrade handlers:

- **TestV1_1_0_Upgrade**: v1.1.0 handler execution
- **TestV1_2_0_Upgrade**: v1.2.0 handler execution
- **TestUpgradeInvariants**: Invariant checks post-upgrade
- **TestUpgradeDeterminism**: Idempotency verification

## Running Tests

```bash
# All upgrade tests
go test ./tests/upgrade/... -v

# Specific test suite
go test ./tests/upgrade/... -v -run TestUpgradeSimulationSuite

# Individual test
go test ./tests/upgrade/... -v -run TestUpgradeFromV1ToV2_FullSimulation

# With timeout for long-running tests
go test ./tests/upgrade/... -v -timeout 5m
```

## Upgrade Simulation Workflow

1. **Setup** - Create app with genesis state and validator
2. **Seed State** - Populate modules with test data (pools, providers, prices)
3. **Capture Pre-State** - Snapshot state before upgrade
4. **Schedule Upgrade** - Register upgrade plan at target height
5. **Simulate Blocks** - Progress chain to upgrade height
6. **Execute Upgrade** - Run migrations via UpgradeKeeper.ApplyUpgrade
7. **Verify Post-State** - Check data integrity, versions, functionality

## Module Migrations

### DEX Module (v1→v2)

- Rebuilds pool token-pair indexes
- Fixes reversed token ordering (lexicographic)
- Validates liquidity provider positions
- Initializes circuit breaker states
- Repairs negative reserves/shares

### Compute Module (v1→v2)

- Rebuilds request status indexes
- Rebuilds provider indexes
- Recalculates provider reputation scores
- Validates request counters
- Migrates params with security defaults

### Oracle Module (v1→v2)

- Validates price data integrity
- Rebuilds price snapshot indexes
- Initializes miss counters
- Migrates params (vote thresholds, slash fractions)
- Cleans stale snapshots

## Critical Mainnet Safety Features

### State Preservation
- Pre/post-state comparison
- Data count verification (pools, providers, prices)
- Zero data loss guarantee

### Rollback Protection
- Failed migrations use cache contexts
- State never mutates on migration errors
- Atomic upgrade execution

### Determinism
- Identical state across validators
- Reproducible migration results
- Consistent version maps

### Multi-Version Paths
- Sequential upgrade testing (v1→v2→v3)
- Data persistence across versions
- Version compatibility checks

## Adding New Upgrade Tests

1. Create migration handler in `x/{module}/migrations/v{N}/migrations.go`
2. Register migrator in `x/{module}/keeper/migrations.go`
3. Add version upgrade in `x/{module}/module.go`
4. Register upgrade handler in `app/app.go`
5. Add test case in `upgrade_simulation_test.go`:

```go
func (suite *UpgradeSimulationSuite) TestMyUpgrade() {
    // Seed pre-upgrade state
    suite.seedMyData()

    // Execute upgrade
    suite.executeUpgrade("v1.X.0", suite.ctx.BlockHeight()+1)

    // Verify post-upgrade state
    suite.verifyMyDataIntact()
    suite.verifyModuleVersions(X)
}
```

## Common Issues

### Migration Not Running
- Check module version in `ConsensusVersion()`
- Verify `RegisterMigration()` call in module.go
- Ensure version map set correctly before upgrade

### Index Not Rebuilt
- Migration must explicitly clear and rebuild indexes
- Check key prefixes match keeper implementation
- Verify iteration over all records

### State Corruption
- Always use cache context for risky operations
- Test rollback on migration failure
- Validate invariants post-migration

## Upgrade Checklist

Before mainnet upgrade:

- [ ] All migration tests pass
- [ ] Multi-version path tested
- [ ] State determinism verified
- [ ] Rollback tested
- [ ] Invariants hold post-upgrade
- [ ] Active operations tested
- [ ] Data integrity confirmed
- [ ] App version bumped correctly
