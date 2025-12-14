# Geographic Location Verification System

## Overview

The PAW Oracle module implements a comprehensive geographic location verification system to ensure validators are geographically distributed. This is critical for:

1. **Decentralization**: Prevents all validators from operating in one jurisdiction
2. **Resilience**: Protects against regional network outages or government interference
3. **Sybil Attack Prevention**: Detects when multiple "validators" are actually run from the same location
4. **Regulatory Compliance**: Ensures the network isn't concentrated in a single regulatory domain

## Architecture

### Components

1. **GeoIP Manager** (`geoip.go`)
   - Uses MaxMind GeoLite2 database for IP geolocation
   - Deterministic lookups (no external API calls during consensus)
   - Maps IP addresses to geographic regions
   - Validates claimed locations against actual IP geolocation

2. **Location Proof System** (`location.go`)
   - Cryptographic proofs of validator location
   - Evidence accumulation over time
   - Detects suspicious location changes (spoofing)
   - Timestamp-based proof expiration

3. **Geographic Diversity Enforcement** (`security.go`)
   - Minimum region requirements (default: 3 regions)
   - Diversity score calculation using Shannon entropy
   - Herfindahl-Hirschman Index for concentration detection
   - Real-time monitoring and alerting

## Setup

### 1. Install GeoLite2 Database

Download the free MaxMind GeoLite2-Country database:

```bash
# Option 1: Download directly from MaxMind (requires free account)
# Sign up at: https://www.maxmind.com/en/geolite2/signup

# Option 2: Use package manager (if available)
# Ubuntu/Debian:
sudo apt-get install geoip-database-extra

# Option 3: Manual download and setup
wget https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-Country.mmdb
sudo mkdir -p /usr/share/GeoIP
sudo mv GeoLite2-Country.mmdb /usr/share/GeoIP/
```

### 2. Configure Environment

Set the `GEOIP_DB_PATH` environment variable (optional if database is in standard location):

```bash
export GEOIP_DB_PATH=/usr/share/GeoIP/GeoLite2-Country.mmdb
```

Standard search paths (tried in order):
1. `/usr/share/GeoIP/GeoLite2-Country.mmdb`
2. `/var/lib/GeoIP/GeoLite2-Country.mmdb`
3. `./GeoLite2-Country.mmdb` (current directory)
4. `$GEOIP_DB_PATH` (environment variable)

### 3. Validator Registration

When registering as a validator, provide your geographic location:

```bash
# Claim your region and provide public IP
pawd tx oracle register-validator \
  --region "north_america" \
  --ip-address "203.0.113.42" \
  --from validator
```

## Supported Regions

The system recognizes the following geographic regions:

- `north_america` - North America (US, Canada, Mexico)
- `south_america` - South America
- `europe` - Europe (including Russia west of Urals)
- `asia` - Asia (including Middle East)
- `africa` - Africa
- `oceania` - Australia, New Zealand, Pacific Islands
- `antarctica` - Antarctic research stations (unlikely)
- `unknown` - Only allowed if GeoIP unavailable

## Validation Process

### Step 1: Basic Validation

- Region must be one of the supported values
- IP address must be valid IPv4 or IPv6
- IP address must be publicly routable (no private/localhost IPs)

### Step 2: GeoIP Verification

If GeoIP database is available:

1. Look up IP address in GeoLite2 database
2. Determine actual geographic region
3. Compare with claimed region
4. Reject if mismatch detected

If GeoIP unavailable:
- Log warning
- Allow registration but flag for manual review
- Validators should configure GeoIP for production

### Step 3: Sybil Detection

- Check if IP address is already used by other validators
- Allow up to 2 validators per IP (for redundancy)
- Reject if 3+ validators claim same IP
- Emit warnings for multiple validators on same IP

### Step 4: Evidence Tracking

- Store location proof with timestamp
- Build evidence history over time
- Detect suspicious location changes
- Require consistency over 30-day periods

## Location Proof System

### Creating a Location Proof

```go
proof := types.NewLocationProof(
    validatorAddr, // Validator address
    ipAddress,     // Public IP address
    claimedRegion, // Geographic region
)
```

### Proof Validation

Location proofs are validated for:

1. **Freshness**: Must be < 24 hours old
2. **Integrity**: SHA256 hash must match computed hash
3. **Consistency**: Must match previous location claims
4. **Authenticity**: Cryptographically signed by validator

### Evidence Accumulation

The system builds evidence over time:

- **First proof**: Establishes baseline location
- **Subsequent proofs**: Must match initial claim
- **Location changes**: Trigger security review
- **Frequent changes**: Rejected as potential spoofing

Example: Detecting location jumps

```go
// Suspicious: 3+ region changes in 30 days
if evidence.DetectLocationJumps(3, 30*24*time.Hour) {
    return ErrLocationProofInvalid
}
```

## Geographic Diversity

### Minimum Requirements

The oracle requires:

- **Minimum Regions**: 3 distinct geographic regions
- **Minimum Diversity Score**: 0.40 (40% Shannon entropy)
- **Maximum IP Concentration**: Max 2 validators per IP

### Diversity Calculation

The system uses Shannon entropy to measure diversity:

```
H = -Σ(p_i * log2(p_i))

where p_i = proportion of validators in region i
```

Normalized to 0-1 scale:
- **1.0** = Perfect distribution (all regions equal)
- **0.0** = Complete concentration (all in one region)

### HHI Concentration Index

Herfindahl-Hirschman Index detects market concentration:

```
HHI = Σ(share_i²)

Diversity Score = 1 - HHI
```

Interpretation:
- **HHI < 0.15**: Unconcentrated (good)
- **0.15 < HHI < 0.25**: Moderate concentration
- **HHI > 0.25**: High concentration (concerning)

## Security Considerations

### Attack Vectors Mitigated

1. **Location Spoofing**
   - Validators claim false locations
   - Mitigated by: GeoIP verification, evidence tracking

2. **Sybil Attacks**
   - Multiple validators from same operator
   - Mitigated by: IP tracking, diversity requirements

3. **Regional Concentration**
   - All validators in one jurisdiction
   - Mitigated by: Minimum region requirements, diversity scoring

4. **VPN/Proxy Abuse**
   - Validators use VPNs to fake location
   - Mitigated by: Evidence consistency checks, location jump detection

### Limitations

1. **VPN Detection**: Hard to detect sophisticated VPN users
2. **Database Accuracy**: GeoLite2 ~99.8% accurate, but not perfect
3. **Cloud Providers**: Large cloud providers have multi-region IPs
4. **Privacy**: Validators must disclose approximate location

### Best Practices

For Validators:

1. **Use Real Location**: Don't use VPNs for validator operations
2. **Static IP**: Maintain consistent public IP address
3. **Update Proofs**: Submit location proofs regularly
4. **Monitor Events**: Watch for location verification warnings

For Network Operators:

1. **Monitor Diversity**: Track geographic distribution metrics
2. **Regular Audits**: Verify validator locations periodically
3. **Database Updates**: Keep GeoLite2 database current
4. **Investigate Warnings**: Review location mismatch events

## API Reference

### VerifyValidatorLocation

Validates and stores validator location information.

```go
func (k Keeper) VerifyValidatorLocation(
    ctx context.Context,
    validatorAddr string,
    claimedRegion string,
    ipAddress string,
) error
```

**Validation Steps:**
1. Input validation (non-empty, valid format)
2. Region format validation
3. IP address format validation
4. Public IP verification
5. GeoIP verification (if available)
6. Sybil detection (IP reuse check)
7. Data storage and event emission

**Errors:**
- `ErrInvalidIPAddress`: Invalid IP format or empty inputs
- `ErrPrivateIPNotAllowed`: Private/localhost IP rejected
- `ErrIPRegionMismatch`: IP doesn't match claimed region
- `ErrTooManyValidatorsFromSameIP`: Sybil attack detected
- `ErrGeoIPDatabaseUnavailable`: GeoIP lookup failed

### SubmitLocationProof

Allows validators to submit cryptographic proof of location.

```go
func (k Keeper) SubmitLocationProof(
    ctx context.Context,
    proof *types.LocationProof,
) error
```

**Validation:**
- Proof must be < 24 hours old
- Hash must match computed value
- Must be consistent with previous proofs
- No suspicious location jumps (< 3 changes in 30 days)

### TrackGeographicDiversity

Calculates current geographic distribution of validators.

```go
func (k Keeper) TrackGeographicDiversity(
    ctx context.Context,
) (*GeographicDistribution, error)
```

**Returns:**
- `RegionCounts`: Map of region -> validator count
- `TotalValidators`: Total bonded validators
- `DiversityScore`: Shannon entropy (0-1)

### ValidateGeographicDiversity

Enforces minimum geographic diversity requirements.

```go
func (k Keeper) ValidateGeographicDiversity(
    ctx context.Context,
) error
```

**Requirements:**
- Minimum 3 distinct regions
- Minimum 0.40 diversity score
- Excludes "unknown" region from count

**Errors:**
- `ErrInsufficientGeoDiversity`: Below minimum requirements

## Events

### validator_location_verified

Emitted when location is successfully verified:

```json
{
  "type": "validator_location_verified",
  "attributes": [
    {"key": "validator", "value": "cosmosvaloper1..."},
    {"key": "region", "value": "north_america"},
    {"key": "ip_address", "value": "203.0.113.42"},
    {"key": "verification_method", "value": "geoip_verified"},
    {"key": "timestamp", "value": "1234567890"}
  ]
}
```

### validator_location_mismatch

Emitted when IP doesn't match claimed region (CRITICAL):

```json
{
  "type": "validator_location_mismatch",
  "attributes": [
    {"key": "validator", "value": "cosmosvaloper1..."},
    {"key": "claimed_region", "value": "north_america"},
    {"key": "actual_region", "value": "europe"},
    {"key": "ip_address", "value": "203.0.113.42"},
    {"key": "severity", "value": "CRITICAL"}
  ]
}
```

### suspicious_location_changes

Emitted when validator changes location too frequently:

```json
{
  "type": "suspicious_location_changes",
  "attributes": [
    {"key": "validator", "value": "cosmosvaloper1..."},
    {"key": "severity", "value": "HIGH"},
    {"key": "action", "value": "manual_review_required"}
  ]
}
```

### geographic_diversity_violation

Emitted when network diversity drops below minimum:

```json
{
  "type": "geographic_diversity_violation",
  "attributes": [
    {"key": "unique_regions", "value": "2"},
    {"key": "min_required", "value": "3"},
    {"key": "diversity_score", "value": "0.35"},
    {"key": "severity", "value": "high"}
  ]
}
```

## Monitoring

### Metrics

The system tracks:

- `oracle_validator_regions`: Validators per region
- `oracle_diversity_score`: Current diversity score (0-1)
- `oracle_location_proofs`: Location proof submissions
- `oracle_location_mismatches`: IP/region mismatch count

### Queries

Check geographic distribution:

```bash
# Query validator location
pawd query oracle validator-location cosmosvaloper1...

# Query geographic diversity
pawd query oracle geographic-diversity

# List validators by region
pawd query oracle validators-by-region north_america
```

## Testing

### Unit Tests

```bash
# Run all location verification tests
go test ./x/oracle/keeper -run TestLocationProof -v
go test ./x/oracle/keeper -run TestLocationEvidence -v
go test ./x/oracle/keeper -run TestGeographicDistribution -v

# Run IP validation tests
go test ./x/oracle/keeper -run TestIsValidIP -v
go test ./x/oracle/keeper -run TestIsPublicIP -v
```

### Integration Tests

```bash
# Test with actual GeoIP database
go test ./x/oracle/keeper -run TestGeoIPManager -v

# Skip if database not available
# Tests will automatically skip if GeoIP DB not found
```

### Benchmarks

```bash
# Benchmark GeoIP lookup performance
go test ./x/oracle/keeper -bench BenchmarkGeoIPLookup -v
```

## Troubleshooting

### GeoIP Database Not Found

**Error**: `GeoIP database not available for location verification`

**Solution**:
1. Download GeoLite2-Country.mmdb
2. Place in `/usr/share/GeoIP/`
3. Or set `GEOIP_DB_PATH` environment variable

### IP Region Mismatch

**Error**: `IP address resolves to region europe, but validator claims north_america`

**Causes**:
- Validator using VPN
- IP address recently changed
- GeoIP database outdated
- Cloud provider multi-region IP

**Solution**:
1. Verify actual IP location
2. Update claimed region if moved
3. Disable VPN for validator
4. Contact support if persistent

### Too Many Validators From Same IP

**Error**: `too many validators from IP 203.0.113.42: 3 (max 2 allowed)`

**Causes**:
- Multiple validators behind same NAT
- Data center with shared egress IP
- Sybil attack attempt

**Solution**:
1. Ensure validators have unique public IPs
2. Contact network operator
3. If legitimate, request governance exemption

### Insufficient Geographic Diversity

**Error**: `insufficient geographic diversity: 2 regions < 3 minimum`

**Solution**:
1. Recruit validators in additional regions
2. Wait for existing validators to register locations
3. Governance may temporarily lower threshold during bootstrap

## Future Enhancements

Planned improvements:

1. **Advanced VPN Detection**: Detect common VPN providers
2. **Latency-Based Verification**: Use network latency to verify location
3. **Peer-to-Peer Attestation**: Validators attest to each other's locations
4. **Automated Database Updates**: Auto-download latest GeoLite2 releases
5. **Regional Weighting**: Require minimum validators per region (not just 3 regions total)
6. **Governance Parameters**: Make some thresholds governance-adjustable

## References

- [MaxMind GeoLite2](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data)
- [Shannon Entropy](https://en.wikipedia.org/wiki/Entropy_(information_theory))
- [Herfindahl-Hirschman Index](https://en.wikipedia.org/wiki/Herfindahl%E2%80%93Hirschman_index)
- [Sybil Attack](https://en.wikipedia.org/wiki/Sybil_attack)
