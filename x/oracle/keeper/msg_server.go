package keeper

import (
	"context"
	"fmt"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// SubmitPrice handles price submission from validators
func (ms msgServer) SubmitPrice(goCtx context.Context, msg *types.MsgSubmitPrice) (*types.MsgSubmitPriceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// SECURITY CHECK 1: Circuit Breaker
	circuitBreakerActive, err := ms.CheckCircuitBreaker(goCtx)
	if err != nil {
		return nil, fmt.Errorf("circuit breaker check failed: %w", err)
	}
	if circuitBreakerActive {
		return nil, fmt.Errorf("circuit breaker is active - price submissions paused")
	}

	// Validate validator address
	validatorAddr, err := sdk.ValAddressFromBech32(msg.Validator)
	if err != nil {
		return nil, fmt.Errorf("invalid validator address: %w", err)
	}

	// SECURITY CHECK 2: Rate Limiting (Sybil/Spam Prevention)
	if err := ms.CheckRateLimit(goCtx, msg.Validator); err != nil {
		ctx.Logger().Warn("rate limit exceeded", "validator", msg.Validator, "error", err)
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}

	// SECURITY CHECK 3: Sybil Attack Resistance
	if err := ms.CheckSybilAttackResistance(goCtx, validatorAddr); err != nil {
		return nil, fmt.Errorf("sybil resistance check failed: %w", err)
	}

	// Validate feeder address
	feederAddr, err := sdk.AccAddressFromBech32(msg.Feeder)
	if err != nil {
		return nil, fmt.Errorf("invalid feeder address: %w", err)
	}

	// Check if feeder is authorized
	if err := ms.ValidateFeeder(goCtx, validatorAddr, feederAddr); err != nil {
		return nil, fmt.Errorf("feeder not authorized: %w", err)
	}

	// Check if validator is active (bonded)
	isActive, err := ms.IsActiveValidator(goCtx, validatorAddr)
	if err != nil {
		return nil, err
	}
	if !isActive {
		return nil, fmt.Errorf("validator %s is not bonded", validatorAddr.String())
	}

	// Validate asset identifier
	if msg.Asset == "" {
		return nil, fmt.Errorf("asset identifier cannot be empty")
	}

	// Validate price is positive
	if msg.Price.IsNil() || msg.Price.LTE(math.LegacyZeroDec()) {
		return nil, fmt.Errorf("price must be positive")
	}

	// SECURITY CHECK 4: Data Source Authenticity (Data Poisoning Prevention)
	if err := ms.ValidateDataSourceAuthenticity(goCtx, msg.Asset, msg.Price); err != nil {
		ctx.Logger().Error("data source validation failed", "error", err)
		return nil, fmt.Errorf("data source validation failed: %w", err)
	}

	// SECURITY CHECK 5: Flash Loan Attack Resistance
	if err := ms.ValidateFlashLoanResistance(goCtx, msg.Asset, msg.Price); err != nil {
		ctx.Logger().Error("flash loan resistance check failed", "error", err)
		return nil, fmt.Errorf("flash loan resistance check failed: %w", err)
	}

	if err := ms.Keeper.SubmitPrice(goCtx, validatorAddr, msg.Asset, msg.Price, feederAddr); err != nil {
		return nil, err
	}

	return &types.MsgSubmitPriceResponse{}, nil
}

// DelegateFeedConsent handles delegation of price submission rights
func (ms msgServer) DelegateFeedConsent(goCtx context.Context, msg *types.MsgDelegateFeedConsent) (*types.MsgDelegateFeedConsentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate validator address
	validatorAddr, err := sdk.ValAddressFromBech32(msg.Validator)
	if err != nil {
		return nil, fmt.Errorf("invalid validator address: %w", err)
	}

	// Validate delegate address
	delegateAddr, err := sdk.AccAddressFromBech32(msg.Delegate)
	if err != nil {
		return nil, fmt.Errorf("invalid delegate address: %w", err)
	}

	// Check if validator exists and is bonded
	isActive, err := ms.IsActiveValidator(goCtx, validatorAddr)
	if err != nil {
		return nil, err
	}
	if !isActive {
		return nil, fmt.Errorf("validator %s is not bonded", validatorAddr.String())
	}

	// Set feeder delegation
	if err := ms.SetFeederDelegation(goCtx, validatorAddr, delegateAddr); err != nil {
		return nil, err
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"feeder_delegated",
			sdk.NewAttribute("validator", validatorAddr.String()),
			sdk.NewAttribute("delegate", delegateAddr.String()),
		),
	)

	return &types.MsgDelegateFeedConsentResponse{}, nil
}

// UpdateParams handles parameter updates (governance only)
func (ms msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate authority
	if ms.authority != msg.Authority {
		return nil, govtypes.ErrInvalidSigner.Wrapf(
			"invalid authority; expected %s, got %s",
			ms.authority,
			msg.Authority,
		)
	}

	// Validate parameters
	if err := ms.validateParams(msg.Params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	// Set new parameters
	if err := ms.SetParams(goCtx, msg.Params); err != nil {
		return nil, err
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"params_updated",
			sdk.NewAttribute("vote_period", fmt.Sprintf("%d", msg.Params.VotePeriod)),
			sdk.NewAttribute("vote_threshold", msg.Params.VoteThreshold.String()),
			sdk.NewAttribute("slash_fraction", msg.Params.SlashFraction.String()),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}

// validateParams validates oracle module parameters
func (ms msgServer) validateParams(params types.Params) error {
	if params.VotePeriod == 0 {
		return fmt.Errorf("vote period must be positive")
	}

	if params.VoteThreshold.IsNil() || params.VoteThreshold.LTE(math.LegacyZeroDec()) || params.VoteThreshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("vote threshold must be between 0 and 1")
	}

	if params.SlashFraction.IsNil() || params.SlashFraction.LT(math.LegacyZeroDec()) || params.SlashFraction.GT(math.LegacyOneDec()) {
		return fmt.Errorf("slash fraction must be between 0 and 1")
	}

	if params.SlashWindow == 0 {
		return fmt.Errorf("slash window must be positive")
	}

	if params.MinValidPerWindow == 0 {
		return fmt.Errorf("min valid per window must be positive")
	}

	if params.MinValidPerWindow > params.SlashWindow {
		return fmt.Errorf("min valid per window cannot exceed slash window")
	}

	if params.TwapLookbackWindow == 0 {
		return fmt.Errorf("twap lookback window must be positive")
	}

	if params.MinGeographicRegions == 0 {
		return fmt.Errorf("min geographic regions must be positive")
	}

	if len(params.AllowedRegions) == 0 {
		return fmt.Errorf("allowed regions must not be empty")
	}

	seen := make(map[string]struct{})
	for _, region := range params.AllowedRegions {
		region = strings.TrimSpace(region)
		if region == "" {
			return fmt.Errorf("allowed region entries must be non-empty")
		}
		if _, ok := seen[region]; ok {
			return fmt.Errorf("duplicate allowed region: %s", region)
		}
		seen[region] = struct{}{}
	}

	if params.MinGeographicRegions > uint64(len(params.AllowedRegions)) {
		return fmt.Errorf("min geographic regions cannot exceed allowed regions")
	}

	return nil
}
