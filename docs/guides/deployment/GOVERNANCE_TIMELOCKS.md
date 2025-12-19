# DEX Parameter Governance Timelocks

## Goal

Enable governance to adjust DEX security parameters without reopening unbounded risk. Changes must:
- Require supermajority support
- Publish parameter diffs before execution
- Enforce a minimum notice period (timelock)
- Support emergency cancellation if indicators spike

## Scope

Parameters that may become governable:
- `MaxPriceDeviation`
- `MaxSwapSizePercent`
- `MaxPoolDrainPercent`
- `FlashLoanProtectionBlocks`
- `PoolCreationCooldown`
- `MaxPoolsPerAddress`
- `MinPoolCreationDeposit`

Keys such as `MaxPools` (global) remain hard-coded unless validators adopt a specialized upgrade.

## Mechanism

1. **Pending Param Store**: extend `types.Params` with `PendingParams` and `PendingApplyHeight`.
2. **Proposal Flow**:
   - Msg: `MsgScheduleDexParamChange` carrying new params + `apply_height`.
   - Validation: supermajority vote + `apply_height >= current_height + MinTimelock`.
3. **Timelock Enforcement**:
   - During `EndBlocker`, if `current_height >= PendingApplyHeight`, move pending params into active store.
   - Timelock default = 10,000 blocks (~17h); overrideable via governance but not below 1,000 blocks.
4. **Cancellation Path**:
   - `MsgCancelDexParamChange` requires same vote threshold and cancels pending changes.
5. **Audit Trail**:
   - Emit events on schedule/cancel/apply.
   - Provide CLI `pawctl query dex pending-params`.

## Validation Rules

| Parameter | Allowed Range | Notes |
|-----------|---------------|-------|
| MaxPriceDeviation | 0.10 – 0.50 | percent, prevents disabling breaker |
| MaxSwapSizePercent | 0.05 – 0.20 | percent of pool reserves |
| MaxPoolDrainPercent | 0.10 – 0.50 | safeguard for single swap |
| FlashLoanProtectionBlocks | 1 – 20 | block delay |
| PoolCreationCooldown | 10 – 10,000 | block delay |
| MaxPoolsPerAddress | 1 – 100 | per window |
| MinPoolCreationDeposit | 10e6 – 1e12 | tokens, matches `MinLiquidity` order |

All fields validated in `types.Params.Validate`.

## Implementation Steps
1. Update `proto/paw/dex/v1/dex.proto` with timelock fields.
2. Regenerate code (`make proto-gen`).
3. Add keeper helpers: `ScheduleParamChange`, `ApplyPendingParams`, `CancelPendingParams`.
4. Wire `EndBlocker` to call `ApplyPendingParams`.
5. Extend CLI/REST with new msgs/queries.
6. Document governance process in `docs/guides/GOVERNANCE_PROPOSALS.md`.
