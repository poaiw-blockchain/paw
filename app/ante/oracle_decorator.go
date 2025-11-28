package ante

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	oraclekeeper "github.com/paw-chain/paw/x/oracle/keeper"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// OracleDecorator validates oracle module-specific transaction requirements
type OracleDecorator struct {
	keeper oraclekeeper.Keeper
}

// NewOracleDecorator creates a new OracleDecorator
func NewOracleDecorator(keeper oraclekeeper.Keeper) OracleDecorator {
	return OracleDecorator{
		keeper: keeper,
	}
}

// AnteHandle implements the AnteDecorator interface
func (od OracleDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// Skip validation during simulation
	if simulate {
		return next(ctx, tx, simulate)
	}

	msgs := tx.GetMsgs()
	for _, msg := range msgs {
		switch msg := msg.(type) {
		case *oracletypes.MsgSubmitPrice:
			if err := od.validateSubmitPrice(ctx, msg); err != nil {
				return ctx, err
			}
		case *oracletypes.MsgRegisterOracle:
			if err := od.validateRegisterOracle(ctx, msg); err != nil {
				return ctx, err
			}
		case *oracletypes.MsgRegisterAsset:
			if err := od.validateRegisterAsset(ctx, msg); err != nil {
				return ctx, err
			}
		}
	}

	return next(ctx, tx, simulate)
}

// validateSubmitPrice performs additional validation for price submissions
func (od OracleDecorator) validateSubmitPrice(ctx sdk.Context, msg *oracletypes.MsgSubmitPrice) error {
	oracle, err := sdk.AccAddressFromBech32(msg.Oracle)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid oracle address: %s", err)
	}

	// Consume gas for validation
	ctx.GasMeter().ConsumeGas(1000, "price submission validation")

	// Verify oracle is registered and active
	oracleInfo, err := od.keeper.GetOracle(ctx, oracle)
	if err != nil {
		return sdkerrors.ErrNotFound.Wrap("oracle not registered")
	}

	if !oracleInfo.Active {
		return sdkerrors.ErrInvalidRequest.Wrap("oracle is not active")
	}

	// Verify asset is registered
	assetExists, err := od.keeper.HasAsset(ctx, msg.AssetId)
	if err != nil || !assetExists {
		return sdkerrors.ErrNotFound.Wrapf("asset %s not registered", msg.AssetId)
	}

	// Get module params
	params, err := od.keeper.GetParams(ctx)
	if err != nil {
		return fmt.Errorf("failed to get params: %w", err)
	}

	// Check price freshness - ensure price is not too old
	blockTime := ctx.BlockTime()
	if msg.Timestamp.Before(blockTime.Add(-params.MaxPriceAge)) {
		return sdkerrors.ErrInvalidRequest.Wrapf("price timestamp too old: %s", msg.Timestamp)
	}

	// Prevent future timestamps
	if msg.Timestamp.After(blockTime.Add(params.MaxClockDrift)) {
		return sdkerrors.ErrInvalidRequest.Wrapf("price timestamp in future: %s", msg.Timestamp)
	}

	// Validate price is positive
	if msg.Price.IsNil() || msg.Price.IsZero() || msg.Price.IsNegative() {
		return sdkerrors.ErrInvalidRequest.Wrap("price must be positive")
	}

	// Check for duplicate submissions in same block
	hasSubmission, err := od.keeper.HasPriceSubmission(ctx, oracle, msg.AssetId, ctx.BlockHeight())
	if err == nil && hasSubmission {
		return sdkerrors.ErrInvalidRequest.Wrap("duplicate price submission in same block")
	}

	// Rate limiting check
	submissionCount, err := od.keeper.GetOracleSubmissionCount(ctx, oracle)
	if err == nil && submissionCount >= params.MaxSubmissionsPerBlock {
		return sdkerrors.ErrInvalidRequest.Wrapf("oracle has exceeded max submissions per block: %d", params.MaxSubmissionsPerBlock)
	}

	// Check circuit breaker
	circuitBroken, err := od.keeper.IsCircuitBroken(ctx)
	if err == nil && circuitBroken {
		return sdkerrors.ErrInvalidRequest.Wrap("oracle circuit breaker triggered")
	}

	return nil
}

// validateRegisterOracle performs additional validation for oracle registration
func (od OracleDecorator) validateRegisterOracle(ctx sdk.Context, msg *oracletypes.MsgRegisterOracle) error {
	oracle, err := sdk.AccAddressFromBech32(msg.Oracle)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid oracle address: %s", err)
	}

	// Consume gas for validation
	ctx.GasMeter().ConsumeGas(1500, "oracle registration validation")

	// Check if oracle is already registered
	existingOracle, err := od.keeper.GetOracle(ctx, oracle)
	if err == nil && existingOracle != nil && existingOracle.Active {
		return sdkerrors.ErrInvalidRequest.Wrap("oracle already registered and active")
	}

	// Get module params
	params, err := od.keeper.GetParams(ctx)
	if err != nil {
		return fmt.Errorf("failed to get params: %w", err)
	}

	// Validate minimum stake
	if msg.Stake.LT(params.MinOracleStake) {
		return sdkerrors.ErrInvalidRequest.Wrapf("stake %s is less than minimum %s",
			msg.Stake.String(), params.MinOracleStake.String())
	}

	// Verify oracle is a validator (or has sufficient stake)
	validator, err := od.keeper.GetStakingKeeper().GetValidator(ctx, sdk.ValAddress(oracle))
	if err != nil && !params.AllowNonValidatorOracles {
		return sdkerrors.ErrInvalidRequest.Wrap("oracle must be a validator")
	}

	// If validator, check voting power threshold
	if err == nil {
		votingPower := validator.GetConsensusPower(od.keeper.GetStakingKeeper().PowerReduction(ctx))
		if votingPower < params.MinValidatorVotingPower {
			return sdkerrors.ErrInvalidRequest.Wrapf("validator voting power %d below minimum %d",
				votingPower, params.MinValidatorVotingPower)
		}
	}

	return nil
}

// validateRegisterAsset performs additional validation for asset registration
func (od OracleDecorator) validateRegisterAsset(ctx sdk.Context, msg *oracletypes.MsgRegisterAsset) error {
	// Consume gas for validation
	ctx.GasMeter().ConsumeGas(1000, "asset registration validation")

	// Verify authority (only governance can register assets)
	params, err := od.keeper.GetParams(ctx)
	if err != nil {
		return fmt.Errorf("failed to get params: %w", err)
	}

	if msg.Authority != params.Authority {
		return sdkerrors.ErrUnauthorized.Wrap("only governance can register assets")
	}

	// Check if asset already exists
	exists, err := od.keeper.HasAsset(ctx, msg.AssetId)
	if err == nil && exists {
		return sdkerrors.ErrInvalidRequest.Wrapf("asset %s already registered", msg.AssetId)
	}

	// Validate asset ID format
	if len(msg.AssetId) == 0 || len(msg.AssetId) > 64 {
		return sdkerrors.ErrInvalidRequest.Wrap("asset ID must be between 1 and 64 characters")
	}

	// Validate description
	if len(msg.Description) > 256 {
		return sdkerrors.ErrInvalidRequest.Wrap("description must not exceed 256 characters")
	}

	return nil
}
