# HIGH-2: GeoIP Database Mandatory for Mainnet

## Summary
Implemented mandatory GeoIP database validation for mainnet deployments to prevent validator collusion and ensure geographic diversity.

## Changes Made

### 1. Proto Definition Updates
**File**: `proto/paw/oracle/v1/oracle.proto`
- Added `require_geographic_diversity` boolean parameter (field 13)
- When true: GeoIP database must be available and validators must have valid regions
- When false: GeoIP database is optional (for testnet environments)

### 2. Parameter Updates
**File**: `x/oracle/types/params.go`
- Updated `DefaultParams()` to include `RequireGeographicDiversity: false` for testnet compatibility
- Added `MainnetParams()` function that returns mainnet-ready parameters:
  - `RequireGeographicDiversity: true`
  - `MinGeographicRegions: 3` (requires at least 3 distinct geographic regions)

### 3. Genesis Validation
**File**: `x/oracle/types/genesis.go`
- Enhanced `Validate()` to enforce geographic diversity constraints when enabled
- Validates:
  - `MinGeographicRegions` must be positive when diversity is required
  - `AllowedRegions` cannot be empty when diversity is required
  - `MinGeographicRegions` cannot exceed number of `AllowedRegions`

### 4. Keeper Initialization Updates
**File**: `x/oracle/keeper/keeper.go`
- Added `ValidateGeoIPAvailability()` method to verify GeoIP database is functional
- Tests database with known public IP (8.8.8.8) to ensure it can resolve regions
- GeoIP manager initialization remains non-fatal during construction (validated at genesis)

### 5. InitGenesis Enforcement
**File**: `x/oracle/keeper/genesis.go`
- Enhanced `InitGenesis()` to validate GeoIP database when `RequireGeographicDiversity` is true
- Provides clear error messages with instructions for configuring GeoIP database
- Validates existing validator oracles have valid geographic regions in allowed list
- Ensures no validator has empty region when diversity is required

## Error Messages
When GeoIP database is required but unavailable, operators receive:
```
geographic diversity is required but GeoIP database is not available: [error details]

Please configure GEOIP_DB_PATH environment variable or place GeoLite2-Country.mmdb in a standard location:
  - /usr/share/GeoIP/GeoLite2-Country.mmdb
  - /var/lib/GeoIP/GeoLite2-Country.mmdb
  - ./GeoLite2-Country.mmdb

For mainnet deployment, geographic diversity is mandatory to prevent validator collusion
```

## Testing
Created comprehensive test coverage:

### Genesis Tests (`x/oracle/keeper/genesis_geoip_test.go`)
1. **TestInitGenesis_RequireGeographicDiversity_NotEnabled**: Verifies genesis succeeds when diversity not required
2. **TestInitGenesis_RequireGeographicDiversity_NoValidators**: Tests with diversity required but no validators
3. **TestInitGenesis_RequireGeographicDiversity_InvalidRegion**: Validates rejection of invalid regions
4. **TestInitGenesis_RequireGeographicDiversity_EmptyRegion**: Validates rejection of empty regions
5. **TestValidateGenesis_RequireGeographicDiversity**: Table-driven tests covering all validation scenarios
6. **TestMainnetParams**: Verifies mainnet parameters enforce diversity

### Keeper Tests (`x/oracle/keeper/keeper_geoip_test.go`)
1. **TestValidateGeoIPAvailability**: Tests GeoIP validation function
2. **TestValidateGeoIPAvailability_Integration**: Integration test (skipped if DB unavailable)

### Test Results
```
=== RUN   TestInitGenesis_RequireGeographicDiversity_NotEnabled
--- PASS: TestInitGenesis_RequireGeographicDiversity_NotEnabled (0.00s)
=== RUN   TestInitGenesis_RequireGeographicDiversity_NoValidators
--- PASS: TestInitGenesis_RequireGeographicDiversity_NoValidators (0.00s)
=== RUN   TestInitGenesis_RequireGeographicDiversity_InvalidRegion
--- PASS: TestInitGenesis_RequireGeographicDiversity_InvalidRegion (0.00s)
=== RUN   TestInitGenesis_RequireGeographicDiversity_EmptyRegion
--- PASS: TestInitGenesis_RequireGeographicDiversity_EmptyRegion (0.00s)
=== RUN   TestValidateGenesis_RequireGeographicDiversity
--- PASS: TestValidateGenesis_RequireGeographicDiversity (0.00s)
=== RUN   TestMainnetParams
--- PASS: TestMainnetParams (0.00s)
=== RUN   TestValidateGeoIPAvailability
--- PASS: TestValidateGeoIPAvailability (0.00s)
```

## Usage

### For Testnet (Default)
```go
params := types.DefaultParams()
// RequireGeographicDiversity: false
// Chain starts without GeoIP database requirement
```

### For Mainnet
```go
params := types.MainnetParams()
// RequireGeographicDiversity: true
// MinGeographicRegions: 3
// GeoIP database is mandatory
```

### Governance Parameter Update
Chains can enable geographic diversity via governance:
```bash
pawd tx gov submit-proposal param-change proposal.json
```

Where proposal.json includes:
```json
{
  "changes": [{
    "subspace": "oracle",
    "key": "RequireGeographicDiversity",
    "value": "true"
  }]
}
```

## Security Impact

### Before
- Silent failure: validators could run without geographic diversity
- No enforcement of GeoIP database availability
- Risk of validator collusion from single geographic region

### After
- **Mainnet**: GeoIP database is mandatory, genesis fails without it
- **Clear error messages** guide operators to proper configuration
- **Governance control** allows enabling/disabling per chain requirements
- **Backward compatible**: Testnet environments remain unaffected

## Migration Path
1. **Existing testnets**: No action required (default: diversity not required)
2. **New mainnet**: Use `types.MainnetParams()` in genesis
3. **Existing mainnet**: Enable via governance proposal to enforce gradually

## Files Modified
- `proto/paw/oracle/v1/oracle.proto`
- `x/oracle/types/params.go`
- `x/oracle/types/genesis.go`
- `x/oracle/keeper/keeper.go`
- `x/oracle/keeper/genesis.go`

## Files Created
- `x/oracle/keeper/genesis_geoip_test.go`
- `x/oracle/keeper/keeper_geoip_test.go`

## Protobuf Generation
Regenerated protobuf files:
- `x/oracle/types/oracle.pb.go`
- `x/oracle/types/oracle.pulsar.go`

## Next Steps for Mainnet Launch
1. Download MaxMind GeoLite2-Country database
2. Place in standard location or set GEOIP_DB_PATH
3. Use `types.MainnetParams()` for genesis configuration
4. Verify all validators have valid geographic regions before launch
5. Monitor geographic distribution post-launch

## Related Issues
- Addresses security vulnerability where validators could collude from single region
- Implements defense-in-depth for oracle module
- Supports compliance with decentralization requirements
