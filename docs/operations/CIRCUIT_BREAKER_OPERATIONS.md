# Circuit Breaker Operations Guide

**Version**: 1.0
**Last Updated**: 2025-12-14

## Overview

Circuit breakers are emergency safety mechanisms that allow operators to pause module operations in response to security incidents, anomalies, or operational issues. PAW implements circuit breakers across three critical modules: DEX, Oracle, and Compute.

## Architecture

### Hierarchy Levels

1. **Global Module Circuit Breakers** - Pause all operations for an entire module
2. **Entity-Specific Circuit Breakers** - Pause operations for specific pools, feeds, or providers
3. **Emergency Controls** - Additional controls like price overrides, job cancellation, reputation overrides

### Authorization

Circuit breaker operations require governance authority. Only addresses with governance permissions can activate or deactivate circuit breakers.

---

## DEX Module Circuit Breakers

### Global DEX Circuit Breaker

**Purpose**: Pause all DEX trading, liquidity, and swap operations across the entire module.

#### Activation

```bash
# Via governance proposal
pawd tx gov submit-proposal \
  --title "Emergency DEX Circuit Breaker" \
  --description "Activating DEX circuit breaker due to detected price manipulation" \
  --type Text \
  --deposit 10000000upaw \
  --from validator

# Query current state
pawd query dex circuit-breaker-state
```

**When to Activate**:
- Price manipulation detected across multiple pools
- Critical bug discovered in swap logic
- Systemic liquidity drain attack
- IBC-related exploit affecting DEX
- Database corruption affecting pool state

#### Deactivation

```bash
# After issue resolution
pawd tx gov submit-proposal \
  --title "Resume DEX Operations" \
  --description "Circuit breaker deactivation - exploit patched, pools verified" \
  --type Text \
  --deposit 10000000upaw \
  --from validator
```

**Pre-Deactivation Checklist**:
- [ ] Root cause identified and patched
- [ ] Pool states verified and consistent
- [ ] TWAP calculations validated
- [ ] Liquidity provider balances reconciled
- [ ] Test transactions executed on private fork
- [ ] Monitoring alerts configured

### Pool-Specific Circuit Breakers

**Purpose**: Isolate problematic pools while keeping the rest of the DEX operational.

#### Activation

```bash
# Pause specific pool (pool ID 5)
pawd tx dex pause-pool 5 \
  --reason "Abnormal price volatility detected" \
  --from validator

# Verify pool is paused
pawd query dex pool 5 | grep circuit_breaker
```

**When to Activate**:
- Single pool shows abnormal price movements (>20% deviation from TWAP)
- Suspected wash trading or price manipulation in specific pool
- Pool reserves imbalance exceeding safety thresholds
- IBC timeout issues specific to pool's token pairs
- Liquidity provider withdrawals exceed 50% in single block

#### Deactivation

```bash
# Resume pool operations
pawd tx dex resume-pool 5 \
  --reason "Volatility resolved, reserves normalized" \
  --from validator

# Verify pool is active
pawd query dex pool 5
```

**Pre-Deactivation Checklist**:
- [ ] Pool reserves within expected ranges (40-60% split)
- [ ] TWAP stabilized for at least 100 blocks
- [ ] No pending dispute resolutions
- [ ] Liquidity provider consent obtained (if applicable)
- [ ] Test swap executed successfully

### Monitoring Commands

```bash
# List all paused pools
pawd query dex list-pools --filter-paused

# Get circuit breaker event history
pawd query txs --events 'dex_circuit_breaker_open.pool_id=5'
pawd query txs --events 'dex_circuit_breaker_close.pool_id=5'

# Monitor pool health metrics
pawd query dex pool-metrics 5
```

---

## Oracle Module Circuit Breakers

### Global Oracle Circuit Breaker

**Purpose**: Pause all oracle price feeds, voting, and updates.

#### Activation

```bash
# Emergency oracle pause
pawd tx gov submit-proposal \
  --title "Oracle Emergency Halt" \
  --description "Halting oracle due to coordinated validator collusion" \
  --type Text \
  --deposit 10000000upaw \
  --from validator

# Check oracle status
pawd query oracle circuit-breaker-state
```

**When to Activate**:
- Coordinated validator voting manipulation detected
- External price feed API compromise
- Systemic price deviation across all pairs (>10%)
- Slashing event cascade (>33% validators slashed)
- Geographic diversity failure (all validators in single region)

#### Deactivation

```bash
# Resume oracle operations
pawd tx gov submit-proposal \
  --title "Resume Oracle Operations" \
  --description "Malicious validators jailed, price feeds restored" \
  --type Text \
  --deposit 10000000upaw \
  --from validator
```

**Pre-Deactivation Checklist**:
- [ ] Compromised validators jailed or removed
- [ ] Price feed sources verified and diverse
- [ ] Vote period reset and validated
- [ ] Geographic distribution restored
- [ ] External price sources operational
- [ ] Test price submissions accepted

### Feed-Specific Circuit Breakers

**Purpose**: Pause individual price feeds while maintaining others.

#### Activation

```bash
# Pause BTC/USD feed
pawd tx oracle pause-feed "BTC/USD" \
  --reason "Binance API outage affecting price accuracy" \
  --from validator

# Verify feed status
pawd query oracle feed-status "BTC/USD"
```

**When to Activate**:
- Single price feed shows extreme deviation (>15% from other sources)
- Primary price source unavailable
- Suspected oracle front-running on specific pair
- Confidence score below threshold (< 0.8)
- Vote participation below 50% for specific feed

#### Deactivation

```bash
# Resume feed
pawd tx oracle resume-feed "BTC/USD" \
  --reason "Binance API restored, price consensus achieved" \
  --from validator
```

### Emergency Price Override

**Purpose**: Manually set price for a feed during crisis (use sparingly).

#### Usage

```bash
# Set emergency price for BTC/USD (price in base units)
pawd tx oracle set-price-override "BTC/USD" 4500000000000 \
  --duration 3600 \
  --reason "Exchange outage - using Coinbase reference price" \
  --from validator

# Verify override active
pawd query oracle price-override "BTC/USD"

# Clear override manually (auto-expires after duration)
pawd tx oracle clear-price-override "BTC/USD" \
  --from validator
```

**When to Use**:
- All external price sources simultaneously unavailable
- Emergency need to prevent liquidation cascades
- Known correct price during oracle downtime
- **WARNING**: Only use with multi-sig governance approval

### Slashing Controls

#### Disable Slashing

```bash
# Temporarily disable oracle slashing
pawd tx oracle disable-slashing \
  --reason "Network upgrade in progress - prevent false slashing" \
  --from validator

# Verify slashing disabled
pawd query oracle slashing-status
```

**When to Use**:
- Coordinated network upgrade affecting validators
- Known price source outage (prevent unfair slashing)
- Emergency maintenance window
- Circuit breaker testing

#### Re-enable Slashing

```bash
# Resume normal slashing
pawd tx oracle enable-slashing \
  --reason "Network upgrade complete" \
  --from validator
```

### Monitoring Commands

```bash
# List all paused feeds
pawd query oracle list-feeds --filter-paused

# Check price override status
pawd query oracle list-overrides

# Monitor slashing events
pawd query txs --events 'oracle_slashing_disabled'

# Feed health metrics
pawd query oracle feed-metrics "BTC/USD"
```

---

## Compute Module Circuit Breakers

### Global Compute Circuit Breaker

**Purpose**: Pause all compute job submissions, assignments, and verifications.

#### Activation

```bash
# Emergency compute halt
pawd tx gov submit-proposal \
  --title "Compute Emergency Halt" \
  --description "Halting compute due to ZK proof verification exploit" \
  --type Text \
  --deposit 10000000upaw \
  --from validator

# Check compute status
pawd query compute circuit-breaker-state
```

**When to Activate**:
- ZK proof verification vulnerability discovered
- Provider collusion detected
- Escrow fund drainage attack
- IBC timeout cascade affecting compute jobs
- Systemic job result manipulation

#### Deactivation

```bash
# Resume compute operations
pawd tx gov submit-proposal \
  --title "Resume Compute Operations" \
  --description "ZK circuit patched, verification restored" \
  --type Text \
  --deposit 10000000upaw \
  --from validator
```

**Pre-Deactivation Checklist**:
- [ ] Vulnerability patched and tested
- [ ] ZK circuits regenerated if necessary
- [ ] Provider reputation scores verified
- [ ] Escrow balances reconciled
- [ ] Test job submitted and verified successfully
- [ ] All pending disputes resolved

### Provider-Specific Circuit Breakers

**Purpose**: Blacklist specific compute providers without affecting others.

#### Activation

```bash
# Pause specific provider
pawd tx compute pause-provider paw1provider... \
  --reason "Provider submitting invalid proofs - 5 failures in 100 blocks" \
  --from validator

# Verify provider paused
pawd query compute provider-status paw1provider...
```

**When to Activate**:
- Provider repeatedly submits invalid results (>5% failure rate)
- Provider timeout rate exceeds 20%
- Suspected malicious behavior (collusion, front-running)
- Provider geographic location compromised
- Reputation score below minimum threshold

#### Deactivation

```bash
# Resume provider
pawd tx compute resume-provider paw1provider... \
  --reason "Provider hardware replaced, test jobs verified" \
  --from validator
```

### Emergency Job Cancellation

**Purpose**: Cancel in-flight jobs during emergencies.

#### Usage

```bash
# Cancel specific job
pawd tx compute cancel-job job-abc123 \
  --reason "Job contains exploit targeting verification circuit" \
  --from validator

# Check job status
pawd query compute job job-abc123

# List all cancelled jobs
pawd query compute list-jobs --filter-cancelled
```

**When to Use**:
- Job discovered to contain malicious computation
- Provider assigned to job goes offline indefinitely
- Requester requests cancellation with valid reason
- Circuit breaker activation requires aborting pending jobs

### Reputation Override

**Purpose**: Temporarily override provider reputation scores.

#### Usage

```bash
# Set emergency reputation override (0-100 scale)
pawd tx compute set-reputation-override paw1provider... 0 \
  --reason "Emergency quarantine - suspected exploit attempt" \
  --from validator

# Check override status
pawd query compute reputation-override paw1provider...

# Clear override
pawd tx compute clear-reputation-override paw1provider... \
  --from validator
```

**When to Use**:
- Emergency provider quarantine (set to 0)
- Restore trusted provider after false positive (set to 100)
- Temporary boost for emergency capacity (set to 95)
- **WARNING**: Use sparingly, automatic reputation preferred

### Monitoring Commands

```bash
# List all paused providers
pawd query compute list-providers --filter-paused

# Check job cancellation status
pawd query compute list-cancelled-jobs

# Monitor circuit breaker events
pawd query txs --events 'compute_circuit_breaker_open.provider=paw1...'

# Provider health metrics
pawd query compute provider-metrics paw1provider...
```

---

## Emergency Response Scenarios

### Scenario 1: Price Manipulation Attack

**Detection**:
```bash
# Monitor abnormal TWAP deviation
pawd query dex twap-deviation --threshold 15
```

**Response**:
1. Pause affected pool: `pawd tx dex pause-pool <id>`
2. Pause oracle feed if price source compromised: `pawd tx oracle pause-feed <pair>`
3. Investigate transactions: `pawd query txs --events 'swap.pool_id=<id>'`
4. If systemic, activate global DEX circuit breaker
5. Coordinate with validators for emergency patch

**Resolution**:
1. Identify attack vector and patch
2. Revert malicious transactions if within dispute window
3. Set price override if necessary: `pawd tx oracle set-price-override`
4. Resume pool after verification
5. Post-mortem and security upgrade

### Scenario 2: Compute Provider Collusion

**Detection**:
```bash
# Monitor provider correlation
pawd query compute provider-collusion-score
```

**Response**:
1. Pause all suspected providers
2. Cancel jobs assigned to colluding providers
3. Set reputation overrides to 0 for attackers
4. Activate global compute circuit breaker if widespread
5. Emergency governance vote to jail providers

**Resolution**:
1. Redistribute jobs to trusted providers
2. Update provider selection algorithm
3. Resume operations with enhanced monitoring
4. Implement reputation decay for suspended providers

### Scenario 3: IBC Timeout Cascade

**Detection**:
```bash
# Monitor IBC timeouts
pawd query ibc channel-status --show-timeouts
```

**Response**:
1. Pause affected pools using IBC tokens
2. Pause oracle feeds for cross-chain price pairs
3. Cancel compute jobs with IBC dependencies
4. Coordinate with counterparty chain validators

**Resolution**:
1. Clear IBC packet backlog
2. Increase timeout parameters if necessary
3. Resume operations incrementally
4. Monitor packet success rate

---

## Monitoring and Alerting

### Critical Metrics

```bash
# DEX Health
pawd query dex metrics | jq '.circuit_breakers_active'

# Oracle Health
pawd query oracle metrics | jq '.feeds_paused, .slashing_disabled'

# Compute Health
pawd query compute metrics | jq '.providers_paused, .jobs_cancelled'
```

### Event Subscriptions

```bash
# Subscribe to circuit breaker events
pawd subscribe --query "message.action='/paw.dex.MsgPausePool'"
pawd subscribe --query "message.action='/paw.oracle.MsgPauseFeed'"
pawd subscribe --query "message.action='/paw.compute.MsgPauseProvider'"
```

### Prometheus Metrics

```
# Circuit breaker states
paw_dex_circuit_breaker_open{module="dex"} 0
paw_dex_pools_paused_total 0
paw_oracle_circuit_breaker_open{module="oracle"} 0
paw_oracle_feeds_paused_total 0
paw_compute_circuit_breaker_open{module="compute"} 0
paw_compute_providers_paused_total 0
```

---

## Troubleshooting

### Circuit Breaker Won't Activate

**Symptoms**: Command succeeds but state unchanged

**Diagnosis**:
```bash
pawd query auth account <validator-address>
pawd query gov params
```

**Solutions**:
- Verify governance permissions
- Check if circuit breaker already active
- Ensure validator key has authority
- Verify proposal passed if using governance

### Circuit Breaker Won't Deactivate

**Symptoms**: Resume command fails or state remains paused

**Diagnosis**:
```bash
pawd query <module> circuit-breaker-state
pawd query txs --events 'message.sender=<validator>'
```

**Solutions**:
- Verify reason and actor match activation
- Check if dependent conditions still present
- Ensure proper governance approval
- Manually query state and compare with expected

### Operations Still Execute Despite Circuit Breaker

**Symptoms**: Transactions succeed when circuit breaker active

**Diagnosis**:
```bash
# Check if circuit breaker check present in handler
pawd query <module> params
grep -r "CheckCircuitBreaker" x/<module>/keeper/
```

**Solutions**:
- Verify module version includes circuit breaker checks
- Ensure ante handler installed correctly
- Check if bypass mechanism active (shouldn't exist in production)
- Review transaction logs for exemptions

### Price Override Not Applied

**Symptoms**: Override set but queries return different price

**Diagnosis**:
```bash
pawd query oracle price-override <pair>
pawd query oracle price <pair>
```

**Solutions**:
- Verify override not expired (check duration)
- Ensure using `price-with-override` query endpoint
- Check if feed circuit breaker also required
- Verify price format matches expected precision

---

## Best Practices

1. **Always Document Activation**: Include detailed reason in all circuit breaker commands
2. **Coordinate with Validators**: Announce circuit breaker activation in validator chat
3. **Test Before Deactivation**: Use private fork to verify safety before resuming
4. **Monitor Closely After Resume**: Watch metrics for 1000 blocks post-deactivation
5. **Automate Common Scenarios**: Use scripts for repeated circuit breaker patterns
6. **Maintain Runbooks**: Keep scenario-specific response procedures updated
7. **Regular Drills**: Practice circuit breaker activation quarterly
8. **Audit Logs**: Review all circuit breaker events monthly for patterns
9. **Minimize Override Usage**: Price overrides and reputation overrides should be rare
10. **Post-Mortem Required**: Document every circuit breaker activation with root cause analysis

---

## Governance Integration

All circuit breaker operations can be executed via governance proposals for added safety:

```bash
# Submit circuit breaker activation proposal
pawd tx gov submit-proposal circuit-breaker-activation \
  --module dex \
  --entity-type pool \
  --entity-id 5 \
  --reason "Emergency pool suspension" \
  --deposit 10000000upaw \
  --from validator

# Vote on proposal
pawd tx gov vote 1 yes --from validator

# Query proposal status
pawd query gov proposal 1
```

This ensures multi-validator consensus for critical operations.

---

## Reference

**Related Documentation**:
- `/docs/PARAMETER_REFERENCE.md` - Module parameter tuning
- `/x/dex/SECURITY_IMPLEMENTATION_GUIDE.md` - DEX security features
- `/x/oracle/SECURITY_AUDIT_REPORT.md` - Oracle security controls
- `/x/compute/SECURITY.md` - Compute security mechanisms

**Support**: File circuit breaker issues at `github.com/paw-chain/paw/issues` with label `circuit-breaker`
