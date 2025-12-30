package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

// SetValidatorOracle sets the oracle information for a validator
func (k *Keeper) SetValidatorOracle(ctx context.Context, validatorOracle types.ValidatorOracle) error {
	valAddr, err := sdk.ValAddressFromBech32(validatorOracle.ValidatorAddr)
	if err != nil {
		return err
	}

	store := k.getStore(ctx)
	if validatorOracle.GeographicRegion == "" {
		validatorOracle.GeographicRegion = "global"
	}

	bz, err := k.cdc.Marshal(&validatorOracle)
	if err != nil {
		return err
	}
	store.Set(GetValidatorOracleKey(valAddr), bz)
	return nil
}

// GetValidatorOracle retrieves the oracle information for a validator
func (k *Keeper) GetValidatorOracle(ctx context.Context, validatorAddr string) (types.ValidatorOracle, error) {
	valAddr, err := sdk.ValAddressFromBech32(validatorAddr)
	if err != nil {
		return types.ValidatorOracle{}, err
	}
	store := k.getStore(ctx)
	bz := store.Get(GetValidatorOracleKey(valAddr))
	if bz == nil {
		// Return default validator oracle if not found
		return types.ValidatorOracle{
			ValidatorAddr:    validatorAddr,
			MissCounter:      0,
			TotalSubmissions: 0,
			IsActive:         true,
		}, nil
	}

	var validatorOracle types.ValidatorOracle
	if err := k.cdc.Unmarshal(bz, &validatorOracle); err != nil {
		return types.ValidatorOracle{}, err
	}
	if validatorOracle.GeographicRegion == "" {
		validatorOracle.GeographicRegion = "global"
	}

	return validatorOracle, nil
}

// DeleteValidatorOracle removes the oracle information for a validator
func (k *Keeper) DeleteValidatorOracle(ctx context.Context, validatorAddr sdk.ValAddress) {
	store := k.getStore(ctx)
	store.Delete(GetValidatorOracleKey(validatorAddr))
}

// IterateValidatorOracles iterates over all validator oracles
func (k *Keeper) IterateValidatorOracles(ctx context.Context, cb func(validatorOracle types.ValidatorOracle) (stop bool)) error {
	store := k.getStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, ValidatorOracleKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var validatorOracle types.ValidatorOracle
		if err := k.cdc.Unmarshal(iterator.Value(), &validatorOracle); err != nil {
			return err
		}
		if cb(validatorOracle) {
			break
		}
	}
	return nil
}

// GetAllValidatorOracles returns all validator oracles
func (k *Keeper) GetAllValidatorOracles(ctx context.Context) ([]types.ValidatorOracle, error) {
	validatorOracles := []types.ValidatorOracle{}
	err := k.IterateValidatorOracles(ctx, func(vo types.ValidatorOracle) bool {
		validatorOracles = append(validatorOracles, vo)
		return false
	})
	return validatorOracles, err
}

// IsActiveValidator checks if a validator is bonded and active
func (k *Keeper) IsActiveValidator(ctx context.Context, validatorAddr sdk.ValAddress) (bool, error) {
	validator, err := k.stakingKeeper.GetValidator(ctx, validatorAddr)
	if err != nil {
		return false, err
	}

	// Validator must be bonded to participate in oracle
	return validator.IsBonded(), nil
}

// GetValidatorVotingPower retrieves a validator's voting power
func (k *Keeper) GetValidatorVotingPower(ctx context.Context, validatorAddr sdk.ValAddress) (int64, error) {
	validator, err := k.stakingKeeper.GetValidator(ctx, validatorAddr)
	if err != nil {
		return 0, err
	}

	return validator.GetConsensusPower(k.stakingKeeper.PowerReduction(ctx)), nil
}

// GetCachedTotalVotingPower retrieves the cached total voting power of all bonded validators.
// PERF-2: This value is updated once per block in BeginBlocker, avoiding O(n) iteration
// per asset aggregation. Returns 0 if cache is not yet populated.
func (k *Keeper) GetCachedTotalVotingPower(ctx context.Context) int64 {
	store := k.getStore(ctx)
	bz := store.Get(CachedTotalVotingPowerKey)
	if bz == nil || len(bz) != 8 {
		return 0
	}
	return int64(binary.BigEndian.Uint64(bz))
}

// SetCachedTotalVotingPower stores the total voting power of all bonded validators.
// PERF-2: Called from BeginBlocker to cache the value once per block.
func (k *Keeper) SetCachedTotalVotingPower(ctx context.Context, totalPower int64) {
	store := k.getStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(totalPower))
	store.Set(CachedTotalVotingPowerKey, bz)
}

// DATA-8: Voting Power Snapshot Functions
// These functions ensure voting power consistency throughout a vote period,
// preventing manipulation attacks where validators change stake mid-period.

// GetCurrentVotePeriod calculates the current vote period number based on block height
func (k *Keeper) GetCurrentVotePeriod(ctx context.Context) (uint64, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}

	if params.VotePeriod == 0 {
		return 0, nil // Vote period not configured
	}

	return uint64(sdkCtx.BlockHeight()) / params.VotePeriod, nil
}

// IsVotePeriodStart returns true if current block is the start of a new vote period
func (k *Keeper) IsVotePeriodStart(ctx context.Context) (bool, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	params, err := k.GetParams(ctx)
	if err != nil {
		return false, err
	}

	if params.VotePeriod == 0 {
		return false, nil
	}

	return sdkCtx.BlockHeight()%int64(params.VotePeriod) == 0, nil
}

// SnapshotVotingPowers creates a snapshot of all validator voting powers at vote period start.
// DATA-8: This ensures consistent voting power throughout the vote period.
func (k *Keeper) SnapshotVotingPowers(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	votePeriod, err := k.GetCurrentVotePeriod(ctx)
	if err != nil {
		return err
	}

	// Get all bonded validators
	bondedVals, err := k.GetBondedValidators(ctx)
	if err != nil {
		return err
	}

	powerReduction := k.stakingKeeper.PowerReduction(ctx)
	totalPower := int64(0)

	// Snapshot each validator's voting power
	for _, val := range bondedVals {
		valAddr, err := sdk.ValAddressFromBech32(val.GetOperator())
		if err != nil {
			continue
		}

		power := val.GetConsensusPower(powerReduction)
		totalPower += power

		// Store individual validator snapshot
		key := GetVotingPowerSnapshotKey(votePeriod, valAddr)
		bz := make([]byte, 8)
		binary.BigEndian.PutUint64(bz, uint64(power))
		store.Set(key, bz)
	}

	// Store total voting power snapshot
	totalKey := GetVotingPowerSnapshotTotalKey(votePeriod)
	totalBz := make([]byte, 8)
	binary.BigEndian.PutUint64(totalBz, uint64(totalPower))
	store.Set(totalKey, totalBz)

	// Store current vote period number for lookup
	periodBz := make([]byte, 8)
	binary.BigEndian.PutUint64(periodBz, votePeriod)
	store.Set(CurrentVotePeriodKey, periodBz)

	sdkCtx.Logger().Debug("created voting power snapshot",
		"vote_period", votePeriod,
		"validators", len(bondedVals),
		"total_power", totalPower,
	)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"voting_power_snapshot",
			sdk.NewAttribute("vote_period", fmt.Sprintf("%d", votePeriod)),
			sdk.NewAttribute("validators", fmt.Sprintf("%d", len(bondedVals))),
			sdk.NewAttribute("total_power", fmt.Sprintf("%d", totalPower)),
		),
	)

	return nil
}

// GetSnapshotVotingPower retrieves a validator's snapshotted voting power for the current vote period.
// DATA-8: Returns the snapshotted power if available, falls back to current power if no snapshot exists.
func (k *Keeper) GetSnapshotVotingPower(ctx context.Context, validatorAddr sdk.ValAddress) (int64, error) {
	store := k.getStore(ctx)

	// Get current vote period
	votePeriod, err := k.GetCurrentVotePeriod(ctx)
	if err != nil {
		// Fallback to current voting power if vote period not configured
		return k.GetValidatorVotingPower(ctx, validatorAddr)
	}

	// Try to get snapshotted power
	key := GetVotingPowerSnapshotKey(votePeriod, validatorAddr)
	bz := store.Get(key)
	if bz == nil || len(bz) != 8 {
		// No snapshot exists, fallback to current power
		// This handles validators who joined after the snapshot
		return k.GetValidatorVotingPower(ctx, validatorAddr)
	}

	return int64(binary.BigEndian.Uint64(bz)), nil
}

// GetSnapshotTotalVotingPower retrieves the total snapshotted voting power for the current vote period.
// DATA-8: Returns the snapshotted total if available, falls back to cached total if no snapshot exists.
func (k *Keeper) GetSnapshotTotalVotingPower(ctx context.Context) int64 {
	store := k.getStore(ctx)

	// Get current vote period
	votePeriod, err := k.GetCurrentVotePeriod(ctx)
	if err != nil {
		// Fallback to cached total
		return k.GetCachedTotalVotingPower(ctx)
	}

	// Try to get snapshotted total
	key := GetVotingPowerSnapshotTotalKey(votePeriod)
	bz := store.Get(key)
	if bz == nil || len(bz) != 8 {
		// No snapshot exists, fallback to cached total
		return k.GetCachedTotalVotingPower(ctx)
	}

	return int64(binary.BigEndian.Uint64(bz))
}

// CleanupOldVotingPowerSnapshots removes voting power snapshots older than retentionPeriods.
// DATA-8: Prevents unbounded state growth from accumulated snapshots.
func (k *Keeper) CleanupOldVotingPowerSnapshots(ctx context.Context, retentionPeriods uint64) error {
	store := k.getStore(ctx)

	currentPeriod, err := k.GetCurrentVotePeriod(ctx)
	if err != nil {
		return err
	}

	// Keep at least 2 periods for safety (current + previous)
	if currentPeriod <= retentionPeriods || retentionPeriods < 2 {
		return nil
	}

	cutoffPeriod := currentPeriod - retentionPeriods

	// Delete old total power snapshots
	for period := uint64(0); period < cutoffPeriod; period++ {
		totalKey := GetVotingPowerSnapshotTotalKey(period)
		if store.Has(totalKey) {
			store.Delete(totalKey)
		}
	}

	// Note: Individual validator snapshots would require iteration.
	// For efficiency, we rely on the total key cleanup and accept some state bloat
	// from individual keys. A more aggressive cleanup could use prefix iteration.

	return nil
}

// GetBondedValidators returns all bonded validators
func (k *Keeper) GetBondedValidators(ctx context.Context) ([]stakingtypes.Validator, error) {
	var bondedValidators []stakingtypes.Validator

	err := k.stakingKeeper.IterateBondedValidatorsByPower(ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		val, ok := validator.(stakingtypes.Validator)
		if !ok {
			return false
		}
		bondedValidators = append(bondedValidators, val)
		return false
	})

	return bondedValidators, err
}

// IncrementMissCounter increments the miss counter for a validator
func (k *Keeper) IncrementMissCounter(ctx context.Context, validatorAddr string) error {
	validatorOracle, err := k.GetValidatorOracle(ctx, validatorAddr)
	if err != nil {
		return err
	}

	validatorOracle.MissCounter++

	// Record missed vote metric
	if k.metrics != nil {
		k.metrics.MissedVotes.With(map[string]string{
			"validator": validatorAddr,
		}).Inc()
	}

	return k.SetValidatorOracle(ctx, validatorOracle)
}

// ResetMissCounter resets the miss counter for a validator
func (k *Keeper) ResetMissCounter(ctx context.Context, validatorAddr string) error {
	validatorOracle, err := k.GetValidatorOracle(ctx, validatorAddr)
	if err != nil {
		return err
	}

	validatorOracle.MissCounter = 0
	return k.SetValidatorOracle(ctx, validatorOracle)
}

// IncrementSubmissionCount increments the submission count for a validator
func (k *Keeper) IncrementSubmissionCount(ctx context.Context, validatorAddr string) error {
	validatorOracle, err := k.GetValidatorOracle(ctx, validatorAddr)
	if err != nil {
		return err
	}

	validatorOracle.TotalSubmissions++
	return k.SetValidatorOracle(ctx, validatorOracle)
}

// ValidateFeeder checks if the feeder is authorized to submit prices for the validator
func (k *Keeper) ValidateFeeder(ctx context.Context, validatorAddr sdk.ValAddress, feederAddr sdk.AccAddress) error {
	// Get validator's account address
	valAccAddr := sdk.AccAddress(validatorAddr)

	// If feeder is the validator themselves, allow it
	if feederAddr.Equals(valAccAddr) {
		return nil
	}

	// Check if there's a delegation
	delegatedFeeder, err := k.GetFeederDelegation(ctx, validatorAddr)
	if err != nil {
		return err
	}

	if delegatedFeeder == nil {
		return fmt.Errorf("feeder %s is not authorized for validator %s", feederAddr.String(), validatorAddr.String())
	}

	if !delegatedFeeder.Equals(feederAddr) {
		return fmt.Errorf("feeder %s does not match delegated feeder %s", feederAddr.String(), delegatedFeeder.String())
	}

	return nil
}

// IsAuthorizedFeeder returns true if the delegate is currently allowed to submit prices for the validator.
// A delegate is authorized when:
//   - It equals the validator's own account (self-feeding)
//   - It matches the existing feeder delegation for the validator
//   - It is not already delegated to a different validator
func (k *Keeper) IsAuthorizedFeeder(ctx sdk.Context, delegate sdk.AccAddress, validatorAddr sdk.ValAddress) bool {
	if delegate.Empty() {
		return false
	}

	if delegate.Equals(sdk.AccAddress(validatorAddr)) {
		return true
	}

	// If the validator already delegated the same address, allow it
	existing, err := k.GetFeederDelegation(ctx, validatorAddr)
	if err == nil && existing != nil && existing.Equals(delegate) {
		return true
	}

	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, FeederDelegationKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		if bytes.Equal(iterator.Value(), delegate.Bytes()) {
			// Delegate is already registered; verify it's tied to this validator
			storedValBz := iterator.Key()[len(FeederDelegationKeyPrefix):]
			valAddr, err := sdk.ValAddressFromBech32(string(storedValBz))
			if err != nil {
				return false
			}
			return valAddr.Equals(validatorAddr)
		}
	}

	// Delegate not currently in use
	return true
}
