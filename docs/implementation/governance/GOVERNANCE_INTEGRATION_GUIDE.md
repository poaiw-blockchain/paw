# Governance Integration Guide

## Quick Start: Integrating Governance into x/compute Module

### Step 1: Regenerate Protobuf Files

Once build tools are installed:

```bash
# Install protobuf tools (if not already installed)
go install github.com/cosmos/gogoproto/protoc-gen-gocosmos@latest
go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@latest

# Generate protobuf code
cd proto
buf generate --template buf.gen.gocosmos.yaml
```

### Step 2: Update App Wiring

File: `app/app.go`

**Find the compute keeper initialization** and update it to include staking and slashing keepers:

```go
// OLD CODE:
app.ComputeKeeper = computekeeper.NewKeeper(
    appCodec,
    keys[computetypes.StoreKey],
    app.BankKeeper,
    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
)

// NEW CODE:
app.ComputeKeeper = computekeeper.NewKeeper(
    appCodec,
    keys[computetypes.StoreKey],
    app.BankKeeper,
    app.StakingKeeper,  // ADD THIS
    app.SlashingKeeper, // ADD THIS
    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
)
```

### Step 3: Add EndBlocker Logic

File: `x/compute/module.go`

Add the EndBlock method to the AppModule:

```go
// EndBlock executes end block logic for the compute module
func (am AppModule) EndBlock(ctx sdk.Context) error {
    // Process dispute lifecycle transitions (evidence → voting → resolved)
    if err := am.keeper.ProcessDisputeLifecycle(ctx); err != nil {
        ctx.Logger().Error("failed to process dispute lifecycle", "error", err)
    }

    // Process appeal lifecycle transitions (voting → resolved)
    if err := am.keeper.ProcessAppealLifecycle(ctx); err != nil {
        ctx.Logger().Error("failed to process appeal lifecycle", "error", err)
    }

    return nil
}
```

### Step 4: Initialize Governance Parameters in Genesis

File: `x/compute/types/genesis.go` (or create if missing)

Add default governance params:

```go
func DefaultGovernanceParams() GovernanceParams {
    return GovernanceParams{
        DisputeDeposit:          math.NewInt(1000000),           // 1 PAW token
        EvidencePeriodSeconds:   259200,                         // 3 days
        VotingPeriodSeconds:     604800,                         // 7 days
        QuorumPercentage:        math.LegacyNewDecWithPrec(40, 2),  // 40%
        ConsensusThreshold:      math.LegacyNewDecWithPrec(67, 2),  // 67%
        SlashPercentage:         math.LegacyNewDecWithPrec(10, 2),  // 10%
        AppealDepositPercentage: math.LegacyNewDecWithPrec(10, 2),  // 10%
        MaxEvidenceSize:         10485760,                       // 10MB
    }
}

func DefaultGenesisState() *GenesisState {
    return &GenesisState{
        Params:           DefaultParams(),
        GovernanceParams: DefaultGovernanceParams(),  // ADD THIS
        Providers:        []Provider{},
        Requests:         []Request{},
        Results:          []Result{},
        Disputes:         []Dispute{},              // ADD THIS
        SlashRecords:     []SlashRecord{},          // ADD THIS
        Appeals:          []Appeal{},               // ADD THIS
        NextRequestId:    1,
        NextDisputeId:    1,                        // ADD THIS
        NextSlashId:      1,                        // ADD THIS
        NextAppealId:     1,                        // ADD THIS
    }
}
```

### Step 5: Update Genesis Initialization

File: `x/compute/keeper/genesis.go`

```go
func (k Keeper) InitGenesis(ctx sdk.Context, data types.GenesisState) {
    // Existing initialization...
    if err := k.SetParams(ctx, data.Params); err != nil {
        panic(err)
    }

    // NEW: Initialize governance params
    if err := k.SetGovernanceParams(ctx, data.GovernanceParams); err != nil {
        panic(err)
    }

    // NEW: Initialize counters
    k.SetNextDisputeID(ctx, data.NextDisputeId)
    k.SetNextSlashID(ctx, data.NextSlashId)
    k.SetNextAppealID(ctx, data.NextAppealId)

    // NEW: Initialize disputes, slash records, appeals
    for _, dispute := range data.Disputes {
        k.SetDispute(ctx, dispute)
    }
    for _, slashRecord := range data.SlashRecords {
        k.SetSlashRecord(ctx, slashRecord)
    }
    for _, appeal := range data.Appeals {
        k.SetAppeal(ctx, appeal)
    }

    // ... rest of existing initialization
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
    // Existing export...
    params, _ := k.GetParams(ctx)
    govParams, _ := k.GetGovernanceParams(ctx)  // NEW

    // Collect disputes
    var disputes []types.Dispute
    k.IterateDisputes(ctx, func(dispute types.Dispute) bool {
        disputes = append(disputes, dispute)
        return false
    })

    // Collect slash records
    var slashRecords []types.SlashRecord
    k.IterateSlashRecords(ctx, func(record types.SlashRecord) bool {
        slashRecords = append(slashRecords, record)
        return false
    })

    // Collect appeals
    var appeals []types.Appeal
    k.IterateAppeals(ctx, func(appeal types.Appeal) bool {
        appeals = append(appeals, appeal)
        return false
    })

    return &types.GenesisState{
        Params:           params,
        GovernanceParams: govParams,              // NEW
        Providers:        /* existing */,
        Requests:         /* existing */,
        Results:          /* existing */,
        Disputes:         disputes,               // NEW
        SlashRecords:     slashRecords,           // NEW
        Appeals:          appeals,                // NEW
        NextRequestId:    k.GetNextRequestID(ctx),
        NextDisputeId:    k.GetNextDisputeID(ctx),   // NEW
        NextSlashId:      k.GetNextSlashID(ctx),     // NEW
        NextAppealId:     k.GetNextAppealID(ctx),    // NEW
    }
}
```

### Step 6: Build and Test

```bash
# Build the chain
make build

# Run tests (when written)
make test

# Initialize test chain
./build/pawd init test-node --chain-id test-1

# Start the chain
./build/pawd start
```

---

## Usage Examples

### 1. File a Dispute (CLI)

```bash
# Create dispute against a provider
pawd tx compute create-dispute \
  --request-id=100 \
  --reason="Provider did not deliver computation results" \
  --deposit=1000000upaw \
  --from=requester \
  --chain-id=paw-1

# Submit additional evidence
pawd tx compute submit-evidence \
  --dispute-id=1 \
  --evidence-type="execution_logs" \
  --data-file=./logs.json \
  --description="Server logs showing timeout" \
  --from=requester \
  --chain-id=paw-1
```

### 2. Validator Voting (CLI)

```bash
# Vote on dispute
pawd tx compute vote-on-dispute \
  --dispute-id=1 \
  --vote=PROVIDER_FAULT \
  --justification="Evidence clearly shows provider abandoned request" \
  --from=validator1 \
  --chain-id=paw-1
```

### 3. Query Disputes (CLI)

```bash
# Get all disputes
pawd query compute disputes

# Get specific dispute
pawd query compute dispute 1

# Get disputes by status
pawd query compute disputes-by-status VOTING

# Get evidence for dispute
pawd query compute evidence 1
```

### 4. Appeal Slashing (CLI)

```bash
# Appeal a slash decision
pawd tx compute appeal-slashing \
  --slash-id=5 \
  --justification="Network outage beyond our control, ISP confirmed" \
  --deposit=100000upaw \
  --from=provider \
  --chain-id=paw-1

# Validator votes on appeal
pawd tx compute vote-on-appeal \
  --appeal-id=1 \
  --approve=true \
  --justification="Network outage was documented and confirmed" \
  --from=validator1 \
  --chain-id=paw-1
```

---

## API Usage (gRPC/REST)

### Create Dispute (REST)

```bash
POST /paw/compute/v1/create-dispute
{
  "requester": "paw1...",
  "request_id": "100",
  "reason": "Provider failed to deliver",
  "evidence": "base64_encoded_data",
  "deposit_amount": "1000000"
}
```

### Query Disputes (REST)

```bash
GET /paw/compute/v1/disputes
GET /paw/compute/v1/disputes/1
GET /paw/compute/v1/disputes/status/VOTING
GET /paw/compute/v1/disputes/request/100
```

### Query Evidence (REST)

```bash
GET /paw/compute/v1/disputes/1/evidence
```

---

## Governance Parameter Updates

Update parameters via governance proposal:

```bash
# Submit parameter change proposal
pawd tx gov submit-proposal update-governance-params \
  --dispute-deposit=2000000 \
  --evidence-period=345600 \
  --voting-period=604800 \
  --quorum=0.45 \
  --consensus-threshold=0.70 \
  --slash-percentage=0.15 \
  --appeal-deposit-percentage=0.10 \
  --max-evidence-size=20971520 \
  --title="Update Compute Governance Parameters" \
  --description="Increase dispute deposit and quorum requirements" \
  --deposit=10000000upaw \
  --from=proposer
```

---

## Monitoring and Events

### Key Events to Index

Subscribe to these events for monitoring:

```go
// Dispute events
"dispute_created"
"dispute_vote_cast"
"dispute_resolved"
"evidence_submitted"

// Appeal events
"appeal_created"
"appeal_vote_cast"
"appeal_approved"
"appeal_rejected"

// Slash events
"slash_record_created"
"provider_slashed"
"provider_stake_restored"
```

### Example Event Listener

```go
// Listen for new disputes
client.Subscribe(ctx, "disputes", "dispute_created.dispute_id EXISTS")

// Listen for resolved disputes
client.Subscribe(ctx, "resolutions", "dispute_resolved.resolution EXISTS")
```

---

## Testing Strategy

### Unit Tests to Write

1. **Dispute Creation**:
   ```go
   TestCreateDispute_Success
   TestCreateDispute_InvalidRequest
   TestCreateDispute_InsufficientDeposit
   TestCreateDispute_NonRequester
   ```

2. **Voting**:
   ```go
   TestVoteOnDispute_ValidValidator
   TestVoteOnDispute_NonValidator
   TestVoteOnDispute_VoteUpdate
   TestTallyDisputeVotes_Quorum
   TestTallyDisputeVotes_Consensus
   ```

3. **Resolution**:
   ```go
   TestResolveDispute_SlashProvider
   TestResolveDispute_NoRefund
   TestResolveDispute_TechnicalIssue
   TestExecuteDisputeResolution_FundMovements
   ```

4. **Appeals**:
   ```go
   TestAppealSlashing_Valid
   TestAppealSlashing_AlreadyAppealed
   TestResolveAppeal_Approved
   TestResolveAppeal_Rejected
   ```

### Integration Tests

```go
func TestFullDisputeLifecycle(t *testing.T) {
    // Setup: Create request with escrow
    // Create dispute
    // Submit evidence from both parties
    // Advance block time past evidence period
    // Multiple validators vote
    // Advance block time past voting period
    // Verify resolution executed correctly
    // Verify funds moved appropriately
}
```

---

## Security Considerations

### Authority Validation

The following operations MUST be called by the governance module address:
- `MsgResolveDispute`
- `MsgResolveAppeal`
- `MsgUpdateGovernanceParams`

Verify in app.go:
```go
authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
```

### Validator Verification

All voting operations verify:
1. Validator is currently bonded
2. Validator has sufficient stake
3. Voting power is snapshotted at vote time

### Deposit Safety

- Deposits held in module account
- Only released on successful resolution
- Atomic fund movements (no partial states)

### Evidence Validation

- Size limits enforced (10MB default)
- Type validation required
- Immutable once submitted

---

## Troubleshooting

### Common Issues

**Issue**: Protobuf generation fails
```bash
# Solution: Install required tools
go install github.com/cosmos/gogoproto/protoc-gen-gocosmos@latest
go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@latest
```

**Issue**: Keeper initialization fails
```bash
# Solution: Ensure staking and slashing keepers are passed
# Check app.go for correct keeper order (staking before compute)
```

**Issue**: Voting fails with "not bonded"
```bash
# Solution: Ensure validator is fully bonded before voting
pawd query staking validator <validator-addr>
```

**Issue**: Dispute resolution fails
```bash
# Solution: Check if voting period has ended
pawd query compute dispute <dispute-id>
# Verify block time > voting_ends_at
```

---

## Performance Optimization

### Indexing Strategy

For production deployments, index these keys in your database:
- Disputes by status (for active dispute queries)
- Disputes by request ID (for request-specific lookups)
- Slash records by provider (for provider history)
- Appeals by status (for active appeal queries)

### Batch Processing

The EndBlocker processes multiple disputes/appeals per block:
- Disputes transitioning from evidence to voting
- Disputes ready for resolution
- Appeals ready for resolution

Monitor EndBlocker gas usage and adjust batch sizes if needed.

---

## Migration Guide (Existing Chains)

If adding governance to an existing chain:

1. **Create Migration Handler**:
```go
func MigrateComputeV1ToV2(ctx sdk.Context, k keeper.Keeper) error {
    // Initialize governance params
    govParams := types.DefaultGovernanceParams()
    k.SetGovernanceParams(ctx, govParams)

    // Initialize counters
    k.SetNextDisputeID(ctx, 1)
    k.SetNextSlashID(ctx, 1)
    k.SetNextAppealID(ctx, 1)

    return nil
}
```

2. **Register Upgrade**:
```go
app.UpgradeKeeper.SetUpgradeHandler(
    "v2-governance",
    func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
        if err := MigrateComputeV1ToV2(ctx, app.ComputeKeeper); err != nil {
            return nil, err
        }
        return app.mm.RunMigrations(ctx, app.configurator, vm)
    },
)
```

---

## Support and Documentation

For additional help:
- See: `GOVERNANCE_IMPLEMENTATION_SUMMARY.md` for architecture details
- Review: `x/compute/keeper/governance.go` for implementation
- Check: Proto files for message formats
- Reference: Cosmos SDK governance module for patterns

---

## Checklist Before Production

- [ ] Protobuf files regenerated
- [ ] App.go updated with new keepers
- [ ] EndBlocker implemented
- [ ] Genesis state includes governance params
- [ ] Unit tests written and passing
- [ ] Integration tests written and passing
- [ ] Parameters tuned for your network
- [ ] Events indexed for monitoring
- [ ] Documentation updated
- [ ] Security audit completed

---

## Next Steps

1. Complete protobuf generation
2. Write comprehensive test suite
3. Update CLI commands for user-friendly governance
4. Create governance proposal templates
5. Document best practices for validators
6. Set up monitoring and alerting
7. Plan governance parameter evolution

The governance system is ready for integration and testing!
