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

	// SECURITY CHECK 0: Emergency Pause
	if err := ms.CheckEmergencyPause(goCtx); err != nil {
		return nil, fmt.Errorf("emergency pause check failed: %w", err)
	}

	// SECURITY CHECK 1: Circuit Breaker
	if err := ms.CheckCircuitBreaker(goCtx); err != nil {
		return nil, fmt.Errorf("circuit breaker check failed: %w", err)
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

	// SECURITY CHECK 3.5: Geographic Diversity (Runtime Enforcement)
	// Check if this is a new validator oracle (first-time submission)
	existingOracle, err := ms.GetValidatorOracle(goCtx, validatorAddr.String())
	if err != nil || existingOracle.TotalSubmissions == 0 {
		// This is a new validator oracle - check geographic diversity
		// Get the validator's geographic region from their oracle info
		validatorOracle, oracleErr := ms.GetValidatorOracle(goCtx, validatorAddr.String())
		if oracleErr != nil {
			// Oracle doesn't exist yet, will be created - check if we can determine region
			ctx.Logger().Warn("new validator oracle without region info",
				"validator", validatorAddr.String(),
			)
		} else if validatorOracle.GeographicRegion != "" {
			// Check if adding this validator would violate diversity constraints
			if err := ms.CheckGeographicDiversityForNewValidator(goCtx, validatorOracle.GeographicRegion); err != nil {
				ctx.Logger().Warn("geographic diversity check failed for new validator",
					"validator", validatorAddr.String(),
					"region", validatorOracle.GeographicRegion,
					"error", err,
				)
				return nil, fmt.Errorf("geographic diversity check failed: %w", err)
			}
		}
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
		return nil, fmt.Errorf("SubmitPrice: check validator status: %w", err)
	}
	if !isActive {
		return nil, fmt.Errorf("SubmitPrice: validator %s is not bonded", validatorAddr.String())
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
		return nil, fmt.Errorf("SubmitPrice: %w", err)
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
		return nil, fmt.Errorf("DelegateFeedConsent: check validator status: %w", err)
	}
	if !isActive {
		return nil, fmt.Errorf("DelegateFeedConsent: validator %s is not bonded", validatorAddr.String())
	}

	// Set feeder delegation
	if err := ms.SetFeederDelegation(goCtx, validatorAddr, delegateAddr); err != nil {
		return nil, fmt.Errorf("DelegateFeedConsent: set delegation: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeOracleFeederDelegated,
			sdk.NewAttribute(types.AttributeKeyValidator, validatorAddr.String()),
			sdk.NewAttribute(types.AttributeKeyDelegate, delegateAddr.String()),
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
		return nil, fmt.Errorf("UpdateParams: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeOracleParamsUpdated,
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

	// Validate diversity check interval (can be 0 to disable periodic checks)
	// No upper limit validation - operators can set their own check frequency

	// Validate diversity warning threshold
	if params.DiversityWarningThreshold.IsNil() ||
		params.DiversityWarningThreshold.LT(math.LegacyZeroDec()) ||
		params.DiversityWarningThreshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("diversity warning threshold must be between 0 and 1")
	}

	// Validate emergency admin address if provided
	if params.EmergencyAdmin != "" {
		if _, err := sdk.AccAddressFromBech32(params.EmergencyAdmin); err != nil {
			return fmt.Errorf("invalid emergency admin address: %w", err)
		}
	}

	return nil
}

// EmergencyPauseOracle handles emergency pause requests
func (ms msgServer) EmergencyPauseOracle(goCtx context.Context, msg *types.MsgEmergencyPauseOracle) (*types.MsgEmergencyPauseOracleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get current params to check emergency admin
	params, err := ms.GetParams(goCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get params: %w", err)
	}

	// Validate signer address
	signerAddr, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return nil, fmt.Errorf("invalid signer address: %w", err)
	}

	// Check authorization: must be either emergency admin or governance authority
	isAdmin := params.EmergencyAdmin != "" && msg.Signer == params.EmergencyAdmin
	isAuthority := msg.Signer == ms.authority

	if !isAdmin && !isAuthority {
		return nil, types.ErrUnauthorizedPause.Wrapf(
			"signer %s is not authorized to pause oracle (admin: %s, authority: %s)",
			msg.Signer,
			params.EmergencyAdmin,
			ms.authority,
		)
	}

	// Validate reason is not empty
	if strings.TrimSpace(msg.Reason) == "" {
		return nil, fmt.Errorf("pause reason cannot be empty")
	}

	// Trigger emergency pause
	if err := ms.Keeper.EmergencyPauseOracle(goCtx, signerAddr.String(), msg.Reason); err != nil {
		return nil, fmt.Errorf("EmergencyPauseOracle: %w", err)
	}

	ctx.Logger().Info(
		"oracle emergency pause activated",
		"paused_by", msg.Signer,
		"reason", msg.Reason,
		"is_admin", isAdmin,
		"is_authority", isAuthority,
	)

	return &types.MsgEmergencyPauseOracleResponse{}, nil
}

// ResumeOracle handles resume requests (governance only)
func (ms msgServer) ResumeOracle(goCtx context.Context, msg *types.MsgResumeOracle) (*types.MsgResumeOracleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate authority (only governance can resume)
	if ms.authority != msg.Authority {
		return nil, govtypes.ErrInvalidSigner.Wrapf(
			"invalid authority; expected %s, got %s",
			ms.authority,
			msg.Authority,
		)
	}

	// Validate reason is not empty
	if strings.TrimSpace(msg.Reason) == "" {
		return nil, fmt.Errorf("resume reason cannot be empty")
	}

	// Resume oracle operations
	if err := ms.Keeper.ResumeOracle(goCtx, msg.Authority, msg.Reason); err != nil {
		return nil, fmt.Errorf("ResumeOracle: %w", err)
	}

	ctx.Logger().Info(
		"oracle emergency pause lifted",
		"resumed_by", msg.Authority,
		"reason", msg.Reason,
	)

	return &types.MsgResumeOracleResponse{}, nil
}
