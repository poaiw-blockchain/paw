# Oracle Module

## Purpose

The Oracle module provides a secure, decentralized price feed system with validator-based consensus, geographic diversity enforcement, and time-weighted average pricing (TWAP) to deliver manipulation-resistant price data for DeFi applications on the PAW blockchain.

## Key Features

- **Validator Price Feeds**: Active validators submit price data weighted by stake
- **Price Aggregation**: Median and stake-weighted consensus with outlier filtering
- **TWAP Oracle**: Time-weighted average prices resistant to flash attacks
- **Geographic Diversity**: Enforces minimum regional distribution to prevent localized manipulation
- **Slashing Protection**: Validators penalized for missed submissions or downtime
- **Feed Delegation**: Validators can delegate price submission to dedicated feeder addresses
- **Emergency Pause**: Admin or governance can pause oracle during anomalies
- **IBC Integration**: Cross-chain price synchronization via authorized channels

## Key Types

### Price
- `asset`: Asset identifier (e.g., "BTC/USD")
- `price`: Consensus price value
- `block_height`: When price was last updated
- `block_time`: Timestamp of last update
- `num_validators`: Number of validators who submitted

### ValidatorPrice
- `validator_addr`: Validator address
- `asset`: Asset being priced
- `price`: Submitted price value
- `block_height`: Submission block height
- `voting_power`: Validator's voting power at submission

### ValidatorOracle
- `validator_addr`: Validator address
- `miss_counter`: Consecutive missed submissions
- `total_submissions`: Total price submissions
- `is_active`: Currently participating in oracle
- `geographic_region`: Validator's region (e.g., "na", "eu", "apac")
- `ip_address`: IP for diversity tracking
- `asn`: Autonomous System Number for ISP diversity

### PriceSnapshot
- `asset`: Asset identifier
- `price`: Price at snapshot
- `block_height`: Snapshot block height
- `block_time`: Snapshot timestamp

## Key Messages

- **MsgSubmitPrice**: Validator submits price for an asset
- **MsgDelegateFeedConsent**: Validator delegates price submission to another address
- **MsgUpdateParams**: Update module parameters (governance only)
- **MsgEmergencyPauseOracle**: Pause all oracle operations (admin or governance)
- **MsgResumeOracle**: Resume normal oracle operations (governance only)

## Configuration Parameters

### Voting Parameters
- `vote_period`: Blocks per voting period (default: 30 blocks)
- `vote_threshold`: Minimum votes required (default: 67%)
- `slash_fraction`: Penalty for missed votes (default: 1%)
- `slash_window`: Tracking window for misses (default: 10000 blocks)
- `min_valid_per_window`: Minimum valid submissions (default: 100)

### TWAP Parameters
- `twap_lookback_window`: Blocks for TWAP calculation (default: 1000 blocks)

### Geographic Diversity
- `allowed_regions`: Permitted regions (e.g., ["na", "eu", "apac", "sa", "af", "me"])
- `min_geographic_regions`: Minimum distinct regions required (default: 1)
- `min_voting_power_for_consensus`: Minimum stake after outlier filtering (default: 10%)
- `max_validators_per_ip`: Max validators from same IP (default: 3)
- `max_validators_per_asn`: Max validators from same ISP (default: 5)
- `require_geographic_diversity`: Enforce GeoIP database availability (default: false for testnet, true for mainnet)
- `diversity_check_interval`: Blocks between diversity checks (default: 100 blocks)
- `diversity_warning_threshold`: Minimum acceptable diversity score (default: 0.40)
- `enforce_runtime_diversity`: Reject registrations violating diversity (default: false for testnet, true for mainnet)

### GeoIP Cache
- `geoip_cache_ttl_seconds`: Cache entry TTL (default: 3600s)
- `geoip_cache_max_entries`: Maximum cached IPs (default: 1000)

### Emergency Controls
- `emergency_admin`: Address authorized for emergency pause (empty to disable)

### IBC
- `authorized_channels`: Permitted IBC channels for price packets
- `nonce_ttl_seconds`: Replay protection window (default: 604800s = 7 days)

## Geographic Diversity Enforcement

The oracle prevents manipulation by requiring price feeds from geographically distributed validators:

1. **GeoIP Lookup**: Validators' IP addresses mapped to regions using MaxMind GeoLite2 database
2. **Diversity Score**: Calculated using Herfindahl-Hirschman Index (HHI) inverse
3. **Runtime Checks**: Periodic verification in BeginBlocker
4. **Warnings**: Events emitted when diversity falls below threshold
5. **Enforcement**: Optionally reject validator registrations that violate diversity requirements

**Testnet Mode**: `require_geographic_diversity = false` allows testing without GeoIP database
**Mainnet Mode**: `require_geographic_diversity = true` mandates valid regional distribution

---

**Module Path:** `github.com/paw-chain/paw/x/oracle`
**Maintainers:** PAW Core Development Team
**Last Updated:** 2025-12-25
