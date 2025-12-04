package keeper

import (
	"bytes"
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

// SetValidatorOracle sets the oracle information for a validator
func (k Keeper) SetValidatorOracle(ctx context.Context, validatorOracle types.ValidatorOracle) error {
	valAddr, err := sdk.ValAddressFromBech32(validatorOracle.ValidatorAddr)
	if err != nil {
		return err
	}

	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&validatorOracle)
	if err != nil {
		return err
	}
	store.Set(GetValidatorOracleKey(valAddr), bz)
	return nil
}

// GetValidatorOracle retrieves the oracle information for a validator
func (k Keeper) GetValidatorOracle(ctx context.Context, validatorAddr string) (types.ValidatorOracle, error) {
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
	return validatorOracle, nil
}

// DeleteValidatorOracle removes the oracle information for a validator
func (k Keeper) DeleteValidatorOracle(ctx context.Context, validatorAddr sdk.ValAddress) {
	store := k.getStore(ctx)
	store.Delete(GetValidatorOracleKey(validatorAddr))
}

// IterateValidatorOracles iterates over all validator oracles
func (k Keeper) IterateValidatorOracles(ctx context.Context, cb func(validatorOracle types.ValidatorOracle) (stop bool)) error {
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
func (k Keeper) GetAllValidatorOracles(ctx context.Context) ([]types.ValidatorOracle, error) {
	validatorOracles := []types.ValidatorOracle{}
	err := k.IterateValidatorOracles(ctx, func(vo types.ValidatorOracle) bool {
		validatorOracles = append(validatorOracles, vo)
		return false
	})
	return validatorOracles, err
}

// IsActiveValidator checks if a validator is bonded and active
func (k Keeper) IsActiveValidator(ctx context.Context, validatorAddr sdk.ValAddress) (bool, error) {
	validator, err := k.stakingKeeper.GetValidator(ctx, validatorAddr)
	if err != nil {
		return false, err
	}

	// Validator must be bonded to participate in oracle
	return validator.IsBonded(), nil
}

// GetValidatorVotingPower retrieves a validator's voting power
func (k Keeper) GetValidatorVotingPower(ctx context.Context, validatorAddr sdk.ValAddress) (int64, error) {
	validator, err := k.stakingKeeper.GetValidator(ctx, validatorAddr)
	if err != nil {
		return 0, err
	}

	return validator.GetConsensusPower(k.stakingKeeper.PowerReduction(ctx)), nil
}

// GetBondedValidators returns all bonded validators
func (k Keeper) GetBondedValidators(ctx context.Context) ([]stakingtypes.Validator, error) {
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
func (k Keeper) IncrementMissCounter(ctx context.Context, validatorAddr string) error {
	validatorOracle, err := k.GetValidatorOracle(ctx, validatorAddr)
	if err != nil {
		return err
	}

	validatorOracle.MissCounter++
	return k.SetValidatorOracle(ctx, validatorOracle)
}

// ResetMissCounter resets the miss counter for a validator
func (k Keeper) ResetMissCounter(ctx context.Context, validatorAddr string) error {
	validatorOracle, err := k.GetValidatorOracle(ctx, validatorAddr)
	if err != nil {
		return err
	}

	validatorOracle.MissCounter = 0
	return k.SetValidatorOracle(ctx, validatorOracle)
}

// IncrementSubmissionCount increments the submission count for a validator
func (k Keeper) IncrementSubmissionCount(ctx context.Context, validatorAddr string) error {
	validatorOracle, err := k.GetValidatorOracle(ctx, validatorAddr)
	if err != nil {
		return err
	}

	validatorOracle.TotalSubmissions++
	return k.SetValidatorOracle(ctx, validatorOracle)
}

// ValidateFeeder checks if the feeder is authorized to submit prices for the validator
func (k Keeper) ValidateFeeder(ctx context.Context, validatorAddr sdk.ValAddress, feederAddr sdk.AccAddress) error {
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
func (k Keeper) IsAuthorizedFeeder(ctx sdk.Context, delegate sdk.AccAddress, validatorAddr sdk.ValAddress) bool {
	if delegate.Empty() {
		return false
	}

	if delegate.Equals(sdk.AccAddress(validatorAddr)) {
		return true
	}

	// If the validator already delegated the same address, allow it
	existing, err := k.GetFeederDelegation(sdk.WrapSDKContext(ctx), validatorAddr)
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
