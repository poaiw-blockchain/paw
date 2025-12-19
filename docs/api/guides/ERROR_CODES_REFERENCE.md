# Error Codes Reference

Complete reference for all error codes across PAW blockchain modules.

## Quick Navigation

- [DEX Module](#dex-module-errors) - Codes 2-37, 91-92
- [Oracle Module](#oracle-module-errors) - Codes 2-54, 60, 90
- [Compute Module](#compute-module-errors) - Codes 2-87

---

## Error Code Format

All errors follow Cosmos SDK `sdkerrors.Register()` pattern:

```go
ErrName = sdkerrors.Register(ModuleName, Code, "description")
```

**Usage in Code**:
```go
return types.ErrInvalidInput.Wrap("additional context")
return types.ErrInvalidInput.Wrapf("value: %s", value)
```

**On-Chain Format**:
```json
{
  "codespace": "dex",
  "code": 4,
  "message": "invalid input: amount must be positive"
}
```

---

## DEX Module Errors

**Module Name**: `dex`
**Codespace**: `dex`

### Pool State Errors (2-6)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 2 | `ErrInvalidPoolState` | Pool reserves corrupted or invalid | Query pool state, create backup checkpoint, contact validators |
| 3 | `ErrInsufficientLiquidity` | Pool lacks liquidity for operation | Reduce swap amount, check reserves, try alternative pools |
| 4 | `ErrInvalidInput` | Input parameters invalid | Check: positive amounts, valid addresses (bech32), correct denoms |
| 5 | `ErrReentrancy` | Reentrancy attack detected | **CRITICAL**: Transaction rolled back, report to security team |
| 6 | `ErrInvariantViolation` | Pool invariant (x*y=k) violated | **CRITICAL**: Pool may need emergency recovery, contact validators |

---

### Circuit Breaker & Security (7-13)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 7 | `ErrCircuitBreakerTriggered` | Emergency pause active | Wait for automatic recovery (typically 1 hour) or check status page |
| 8 | `ErrSwapTooLarge` | Swap exceeds maximum size | Split into smaller swaps, check `max_swap_size` param |
| 9 | `ErrPriceImpactTooHigh` | Price impact > 5% | Reduce amount, split across pools, use limit orders |
| 10 | `ErrFlashLoanDetected` | Flash loan attack pattern | **SECURITY**: Multiple operations in same block blocked |
| 11 | `ErrOverflow` | Arithmetic overflow | **CRITICAL**: Contact developers, should never happen |
| 12 | `ErrUnderflow` | Arithmetic underflow | **CRITICAL**: Contact developers, should never happen |
| 13 | `ErrDivisionByZero` | Division by zero attempted | **CRITICAL**: Pool reserves may be zero |

---

### Pool Management (14-18)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 14 | `ErrPoolNotFound` | Pool doesn't exist | Query available pools, create with `MsgCreatePool` |
| 15 | `ErrPoolAlreadyExists` | Duplicate pool for token pair | Use `MsgAddLiquidity` to add to existing pool |
| 16 | `ErrInsufficientShares` | Insufficient LP shares | Query share balance, reduce withdrawal amount |
| 17 | `ErrSlippageTooHigh` | Price slipped beyond tolerance | Increase slippage tolerance or wait for stability |
| 18 | `ErrInvalidTokenPair` | Invalid token pair | Tokens must differ, valid denoms, lexicographically sorted |

---

### Rate Limiting & JIT (19-20)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 19 | `ErrRateLimitExceeded` | Too many operations | Wait for rate limit reset |
| 20 | `ErrJITLiquidityDetected` | Just-in-time liquidity detected | **SECURITY**: Sandwich attack prevention |

---

### State & Amounts (21-24)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 21 | `ErrInvalidState` | State is invalid | General state error, check specific operation |
| 22 | `ErrInvalidSwapAmount` | Swap amount invalid | Amount must be positive, within min/max limits |
| 23 | `ErrInvalidLiquidityAmount` | Liquidity amount invalid | Both amounts required for initial liquidity, maintain ratio for additional |
| 24 | `ErrStateCorruption` | State corruption detected | **CRITICAL**: Automatic recovery, backup restoration |

---

### Cross-Chain & Oracle (25-27)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 25 | `ErrSlippageExceeded` | Cross-chain slippage exceeded | Retry with updated limits after liquidity refresh |
| 26 | `ErrOraclePrice` | Oracle price retrieval failed | Ensure oracle module active, prices fresh (<60s) |
| 27 | `ErrPriceDeviation` | Pool price deviates from oracle | Verify prices, adjust liquidity, potential manipulation |

---

### System Limits (28-30)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 28 | `ErrMaxPoolsReached` | Maximum pools created | Wait for cleanup or parameter update via governance |
| 29 | `ErrUnauthorized` | Caller not authorized | Verify you're the signer, check LP share ownership |
| 30 | `ErrDeadlineExceeded` | Transaction deadline passed | Increase deadline, account for network latency |

---

### IBC & Nonce (31, 91-92)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 31 | `ErrInvalidNonce` | Packet nonce validation failed | Ensure unique, increasing nonce per channel |
| 91 | `ErrInvalidAck` | Acknowledgement invalid | Check ack data format, verify success/error fields |
| 92 | `ErrUnauthorizedChannel` | IBC channel not authorized | Submit governance proposal to authorize channel |

---

### Limit Orders (32-35)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 32 | `ErrOrderNotFound` | Limit order doesn't exist | Verify order ID, may have been filled or cancelled |
| 33 | `ErrInvalidOrder` | Order parameters invalid | Check price, amounts, expiration time |
| 34 | `ErrOrderNotAuthorized` | Not the order owner | Only order creator can modify/cancel |
| 35 | `ErrOrderNotCancellable` | Order cannot be cancelled | May be partially filled or expired |

---

### Circuit Breaker State (36-37)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 36 | `ErrCircuitBreakerAlreadyOpen` | Already in emergency pause | No action needed |
| 37 | `ErrCircuitBreakerAlreadyClosed` | Already resumed operations | No action needed |

---

## Oracle Module Errors

**Module Name**: `oracle`
**Codespace**: `oracle`

### Asset & Price Errors (2-7, 12-13)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 2 | `ErrInvalidAsset` | Asset symbol not recognized | Check supported assets, use correct format (BTC, ETH, ATOM) |
| 3 | `ErrInvalidPrice` | Price invalid (non-positive, overflow) | Ensure positive, within bounds, recent (<1hr) |
| 7 | `ErrPriceNotFound` | No price data for asset | Wait for next vote period (~30s), query available assets |
| 12 | `ErrPriceExpired` | Price data stale (> max age) | Wait for validators to submit new prices |
| 13 | `ErrPriceDeviation` | Price deviates from median (>10%) | Verify price sources, check for market volatility |

---

### Validator & Feeder Errors (4-5, 8, 14)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 4 | `ErrValidatorNotBonded` | Validator must be bonded | Check status, ensure sufficient stake, wait for bonding (21 days) |
| 5 | `ErrFeederNotAuthorized` | Feeder not delegated | Validator must use `MsgDelegateFeeder` |
| 8 | `ErrValidatorNotFound` | Validator address not found | Check address format (bech32), query validator set |
| 14 | `ErrValidatorSlashed` | Slashed for misbehavior | Wait for jail period to end before submitting |

---

### Voting & Submission (6, 9, 15-16)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 6 | `ErrInsufficientVotes` | < 66% validator participation | Wait for more validators, check network connectivity |
| 9 | `ErrInvalidVotePeriod` | Vote period out of range (1-3600s) | Update via governance proposal |
| 15 | `ErrDuplicateSubmission` | Already submitted this period | Wait for next vote period |
| 16 | `ErrMissedVote` | Validator missed too many votes | Risk of slashing, ensure oracle service running |

---

### Parameter Validation (10-11)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 10 | `ErrInvalidThreshold` | Threshold not in 0.50-1.00 range | Update via governance, recommended: 0.67 |
| 11 | `ErrInvalidSlashFraction` | Slash fraction not in 0.00-1.00 | Update via governance, recommended: 0.01 |

---

### Security Errors (20-24)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 20 | `ErrCircuitBreakerActive` | Emergency pause triggered | Wait for automatic reset, check system status |
| 21 | `ErrRateLimitExceeded` | Too many submissions | 1 per vote period, wait for next period |
| 22 | `ErrSybilAttackDetected` | Multiple sources from same entity | **SECURITY**: Use independent, geographically distributed sources |
| 23 | `ErrFlashLoanDetected` | Unusual price spike detected | **SECURITY**: Submit legitimate price next period |
| 24 | `ErrDataPoisoning` | Price data failed authenticity check | **SECURITY**: Check API credentials, verify source legitimacy |

---

### Aggregation Errors (30-33)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 30 | `ErrInsufficientDataSources` | < minimum sources (typically 3) | Add more independent data sources |
| 31 | `ErrOutlierDetected` | Price is statistical outlier (>3σ) | Verify accuracy, check API issues, compare exchanges |
| 32 | `ErrMedianCalculationFailed` | Cannot calculate median | Insufficient valid prices, wait for more submissions |
| 33 | `ErrInsufficientOracleConsensus` | < minimum voting power after filtering | **SECURITY**: Wait for high-stake validators (10% minimum) |

---

### State & Data Errors (40-43)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 40 | `ErrStateCorruption` | State corruption detected | **CRITICAL**: Automatic recovery, backup restoration |
| 41 | `ErrOracleInactive` | Oracle is inactive | Check module status, contact validators |
| 42 | `ErrOracleDataUnavailable` | Data unavailable for asset | Wait for next aggregation, ensure feeder connectivity |
| 43 | `ErrInvalidPriceSource` | Source failed validation | Verify registration, reputation, heartbeat |

---

### Geographic Validation (44-49, 51-52)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 44 | `ErrInvalidIPAddress` | IP format invalid | Provide valid IPv4/IPv6 address |
| 45 | `ErrIPRegionMismatch` | IP location doesn't match claim | **SECURITY**: Update claimed region or fix IP |
| 46 | `ErrPrivateIPNotAllowed` | Private IPs not allowed | **SECURITY**: Validators need public IPs |
| 47 | `ErrLocationProofRequired` | Location proof needed | Provide verifiable location evidence |
| 48 | `ErrLocationProofInvalid` | Proof expired or invalid | Renew with current timestamp, ensure signature valid |
| 49 | `ErrInsufficientGeoDiversity` | < 3 distinct regions | **SECURITY**: Critical for decentralization |
| 51 | `ErrGeoIPDatabaseUnavailable` | GeoIP database missing | Download GeoLite2-Country.mmdb, set GEOIP_DB_PATH |
| 52 | `ErrTooManyValidatorsFromSameIP` | > max validators per IP (2) | **SECURITY**: Ensures independent operation |

---

### IBC & Nonce (50, 60, 90)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 50 | `ErrInvalidNonce` | Packet nonce invalid | Must be unique and increasing per channel |
| 60 | `ErrUnauthorizedChannel` | Channel not authorized | Verify governance params, use approved channels |
| 90 | `ErrInvalidAck` | Acknowledgement invalid | Check format, verify counterparty response |

---

### Circuit Breaker State (53-54)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 53 | `ErrCircuitBreakerAlreadyOpen` | Already paused | No action needed |
| 54 | `ErrCircuitBreakerAlreadyClosed` | Already resumed | No action needed |

---

## Compute Module Errors

**Module Name**: `compute`
**Codespace**: `compute`

### Request Validation (2-5)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 2 | `ErrInvalidRequest` | Request parameters invalid | Check: container image, env vars, resource requirements, signature |
| 3 | `ErrInvalidProvider` | Provider address invalid | Verify bech32 format, ensure registered and active |
| 4 | `ErrInvalidResult` | Result data invalid | Verify format, hash, signature from provider |
| 5 | `ErrInvalidProof` | Verification proof invalid | Validate: signature (64B), pubkey (32B), merkle root (32B) |

---

### Provider Errors (10-14)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 10 | `ErrProviderNotFound` | Provider not registered | Register with `MsgRegisterProvider`, meet minimum stake |
| 11 | `ErrProviderNotActive` | Provider inactive | Verify sufficient stake, no active slashing |
| 12 | `ErrProviderOverloaded` | Provider at capacity | Wait for jobs to complete or select different provider |
| 13 | `ErrInsufficientStake` | Stake below minimum | Increase stake with `MsgStakeProvider`, query params for minimum |
| 14 | `ErrProviderSlashed` | Slashed for misbehavior | Wait for penalty period to expire |

---

### Request Lifecycle (20-23)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 20 | `ErrRequestNotFound` | Request ID doesn't exist | Verify ID, check if cancelled or expired |
| 21 | `ErrRequestExpired` | Request exceeded timeout | Submit new request with longer timeout |
| 22 | `ErrRequestAlreadyCompleted` | Result already submitted | Query request details to retrieve result |
| 23 | `ErrRequestCancelled` | Request was cancelled | Submit new request if still needed |

---

### Escrow Errors (30-33)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 30 | `ErrInsufficientEscrow` | Insufficient tokens for cost | Deposit: base_price + resource_fees + verification_fees |
| 31 | `ErrEscrowLocked` | Locked for active computation | Wait for result or timeout before withdrawal |
| 32 | `ErrEscrowNotFound` | No escrow for request | Create escrow when submitting request |
| 33 | `ErrEscrowRefundFailed` | Automatic refund failed | Claim manually with `MsgClaimRefund` |

---

### Verification Errors (40-45)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 40 | `ErrVerificationFailed` | Result failed verification | Review logs, provider may be penalized |
| 41 | `ErrInvalidSignature` | Ed25519 signature invalid | Ensure 64 bytes, pubkey matches provider |
| 42 | `ErrInvalidMerkleProof` | Merkle proof validation failed | Verify 32-byte nodes, check root matches |
| 43 | `ErrInvalidStateCommitment` | State commitment mismatch | Ensure deterministic computation (no time/randomness) |
| 44 | `ErrReplayAttackDetected` | Nonce already used | **SECURITY**: Generate new unique nonce |
| 45 | `ErrProofExpired` | Proof timestamp too old | Generate new proof with current timestamp |

---

### ZK Proof Errors (50-57)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 50 | `ErrInvalidZKProof` | ZK proof structure invalid | Verify matches circuit spec, check size and format |
| 51 | `ErrZKVerificationFailed` | ZK verification failed | Check public inputs, verify witness validity |
| 52 | `ErrInvalidCircuit` | Circuit not found or invalid | Use supported circuit types, verify version |
| 53 | `ErrInvalidPublicInputs` | Public inputs don't match circuit | Check count, types, encoding per circuit docs |
| 54 | `ErrInvalidWitness` | Private witness invalid | Generate from valid trace, verify constraints |
| 55 | `ErrProofTooLarge` | Proof exceeds maximum size | **SECURITY**: DoS prevention, check circuit config |
| 56 | `ErrInsufficientDeposit` | Deposit too low for verification | Query circuit params, deposit refunded on valid proof |
| 57 | `ErrDepositTransferFailed` | Failed to transfer deposit | Check balance, ensure sufficient funds |

---

### Resource Errors (60-62)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 60 | `ErrInsufficientResources` | Provider lacks resources | Reduce requirements, select higher capacity provider |
| 61 | `ErrResourceQuotaExceeded` | Account quota exceeded | Wait for reset, upgrade tier, optimize usage |
| 62 | `ErrInvalidResourceSpec` | Resource spec invalid | CPU: 1-64 cores, Memory: 1-256GB, Storage: 1-1000GB |

---

### Security Errors (70-73)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 70 | `ErrUnauthorized` | Operation not permitted | Check if you're request owner, verify signer |
| 71 | `ErrRateLimitExceeded` | Too many requests | Wait for reset, upgrade tier, batch operations |
| 72 | `ErrCircuitBreakerActive` | Module paused | Wait for recovery or admin intervention |
| 73 | `ErrSuspiciousActivity` | Unusual pattern detected | Contact support if flagged incorrectly |

---

### IBC Errors (80-85)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 80 | `ErrInvalidPacket` | IBC packet validation failed | Check structure, verify required fields |
| 81 | `ErrChannelNotFound` | Channel doesn't exist | Create channel, verify ID, check counterparty |
| 82 | `ErrPacketTimeout` | Packet timed out | Increase timeout, check liveness, verify relayer |
| 83 | `ErrInvalidAcknowledgement` | Ack validation failed | Check format, verify success/error fields |
| 84 | `ErrInvalidNonce` | Packet nonce invalid | Must be unique and increasing |
| 85 | `ErrUnauthorizedChannel` | Channel not authorized | Update params via governance proposal |

---

### Circuit Breaker State (86-87)

| Code | Error Name | Description | Recovery |
|------|------------|-------------|----------|
| 86 | `ErrCircuitBreakerAlreadyOpen` | Already paused | No action needed |
| 87 | `ErrCircuitBreakerAlreadyClosed` | Already resumed | No action needed |

---

## Error Recovery Patterns

### 1. Automatic Recovery

**Errors with automatic recovery**:
- `ErrCircuitBreakerTriggered` - Cooldown period (1 hour)
- `ErrStateCorruption` - Backup restoration
- `ErrEscrowRefundFailed` - Manual claim available

**Pattern**:
```bash
# Check recovery status
pawd query dex circuit-breaker-status

# Wait for automatic recovery or trigger manual
pawd tx dex close-circuit-breaker --from validator
```

---

### 2. User-Correctable Errors

**Common fixes**:

```bash
# ErrInsufficientLiquidity - Check pool
pawd query dex pool 1

# ErrInsufficientStake - Increase stake
pawd tx compute stake-provider 1000000upaw --from provider

# ErrFeederNotAuthorized - Delegate feeder
pawd tx oracle delegate-feeder <feeder-address> --from validator
```

---

### 3. Governance-Required Errors

**Requires proposals**:
- `ErrUnauthorizedChannel` - Authorize IBC channel
- `ErrMaxPoolsReached` - Update max pools param
- `ErrInvalidVotePeriod` - Update vote period

**Example**:
```bash
# Submit parameter change proposal
pawd tx gov submit-proposal param-change proposal.json --from proposer
```

---

### 4. Critical Errors (Contact Validators)

**Immediate escalation required**:
- `ErrReentrancy` - Security incident
- `ErrOverflow` / `ErrUnderflow` - Code bug
- `ErrStateCorruption` - Data integrity issue
- `ErrInvariantViolation` - Pool corruption

**Actions**:
1. Report to validators immediately
2. Halt affected operations
3. Wait for emergency patch
4. Do NOT retry transactions

---

## Error Handling Best Practices

### In Client Code

```go
// Always check and handle errors
result, err := dexKeeper.ExecuteSwap(ctx, msg)
if err != nil {
    // Check error type
    if errors.Is(err, types.ErrInsufficientLiquidity) {
        // Handle specific error
        return fmt.Errorf("pool lacks liquidity: %w", err)
    }

    // Get recovery suggestion
    suggestion := types.GetRecoverySuggestion(err)
    return fmt.Errorf("%w\nRecovery: %s", err, suggestion)
}
```

### In Smart Contracts (CosmWasm)

```rust
// Check error codes in responses
match response.code {
    3 => Err(ContractError::InsufficientLiquidity),
    9 => Err(ContractError::PriceImpactTooHigh),
    _ => Err(ContractError::UnexpectedError(response.message))
}
```

### In CLI

```bash
# Use --output json for programmatic parsing
pawd tx dex swap ... --output json 2>&1 | jq '.raw_log'

# Check exit codes
if [ $? -ne 0 ]; then
    echo "Transaction failed, check error code"
fi
```

---

## Monitoring & Alerting

### Critical Errors to Monitor

```yaml
alerts:
  - name: ReentrancyDetected
    error: ErrReentrancy (code 5)
    severity: critical
    action: Immediate security response

  - name: StateCorruption
    error: ErrStateCorruption (code 24/40)
    severity: critical
    action: Initiate backup recovery

  - name: CircuitBreakerTriggered
    error: ErrCircuitBreakerTriggered (code 7/20/72)
    severity: warning
    action: Investigate trigger cause

  - name: FlashLoanDetected
    error: ErrFlashLoanDetected (code 10)
    severity: info
    action: Log for analysis
```

### Prometheus Metrics

```prometheus
# Count errors by type
paw_errors_total{module="dex",code="3"} 42

# Circuit breaker status
paw_circuit_breaker_active{module="dex"} 0

# State corruption events
paw_state_corruption_total{module="oracle"} 0
```

---

## Appendix: Error Code Ranges

| Module | Code Range | Usage |
|--------|------------|-------|
| DEX | 2-37 | Core DEX operations |
| DEX | 91-92 | IBC operations |
| Oracle | 2-54 | Core oracle operations |
| Oracle | 60, 90 | IBC operations |
| Compute | 2-87 | All compute operations |

**Reserved Ranges**:
- 1: Reserved (never used)
- 100+: Future expansion

---

## Related Documentation

- [Cross-Module Integration](../implementation/CROSS_MODULE_INTEGRATION.md)
- [Parameter Reference](../../PARAMETER_REFERENCE.md)
- [Governance Proposals](../guides/GOVERNANCE_PROPOSALS.md)
