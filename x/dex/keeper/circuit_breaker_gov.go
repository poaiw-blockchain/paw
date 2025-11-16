package keeper

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/paw-chain/paw/x/dex/types"
)

// CircuitBreakerProposal is a governance proposal to update circuit breaker configuration
type CircuitBreakerProposal struct {
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Config      CircuitBreakerConfig `json:"config"`
}

// CircuitBreakerResumeProposal is a governance proposal to override circuit breaker and resume trading
type CircuitBreakerResumeProposal struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PoolId      uint64 `json:"pool_id"`
}

// HandleCircuitBreakerProposal handles a governance proposal to update circuit breaker config
func HandleCircuitBreakerProposal(ctx sdk.Context, k Keeper, p *CircuitBreakerProposal) error {
	if err := k.SetCircuitBreakerConfig(ctx, p.Config); err != nil {
		return err
	}

	k.Logger(ctx).Info(
		"Circuit breaker configuration updated via governance",
		"threshold_1min", p.Config.Threshold1Min.String(),
		"threshold_5min", p.Config.Threshold5Min.String(),
		"threshold_15min", p.Config.Threshold15Min.String(),
		"threshold_1hour", p.Config.Threshold1Hour.String(),
		"cooldown_period", p.Config.CooldownPeriod,
	)

	return nil
}

// HandleCircuitBreakerResumeProposal handles a governance proposal to override and resume trading
func HandleCircuitBreakerResumeProposal(ctx sdk.Context, k Keeper, p *CircuitBreakerResumeProposal) error {
	if err := k.ResumeTrading(ctx, p.PoolId, true); err != nil {
		return err
	}

	k.Logger(ctx).Info(
		"Circuit breaker override via governance",
		"pool_id", p.PoolId,
		"resumed_at", ctx.BlockHeight(),
	)

	return nil
}

// Implement ProposalHandler interface for CircuitBreakerProposal
func (p CircuitBreakerProposal) GetTitle() string       { return p.Title }
func (p CircuitBreakerProposal) GetDescription() string { return p.Description }
func (p CircuitBreakerProposal) ProposalRoute() string  { return types.RouterKey }
func (p CircuitBreakerProposal) ProposalType() string   { return "CircuitBreakerConfig" }
func (p CircuitBreakerProposal) ValidateBasic() error {
	if err := govtypes.ValidateAbstract(p); err != nil {
		return err
	}
	return validateCircuitBreakerConfig(p.Config)
}
func (p CircuitBreakerProposal) String() string {
	bz, _ := json.Marshal(p)
	return string(bz)
}

// Implement ProposalHandler interface for CircuitBreakerResumeProposal
func (p CircuitBreakerResumeProposal) GetTitle() string       { return p.Title }
func (p CircuitBreakerResumeProposal) GetDescription() string { return p.Description }
func (p CircuitBreakerResumeProposal) ProposalRoute() string  { return types.RouterKey }
func (p CircuitBreakerResumeProposal) ProposalType() string   { return "CircuitBreakerResume" }
func (p CircuitBreakerResumeProposal) ValidateBasic() error {
	if err := govtypes.ValidateAbstract(p); err != nil {
		return err
	}
	if p.PoolId == 0 {
		return fmt.Errorf("pool id cannot be zero")
	}
	return nil
}
func (p CircuitBreakerResumeProposal) String() string {
	bz, _ := json.Marshal(p)
	return string(bz)
}

// NewCircuitBreakerProposalHandler creates a governance proposal handler for circuit breaker operations
func NewCircuitBreakerProposalHandler(k Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *CircuitBreakerProposal:
			return HandleCircuitBreakerProposal(ctx, k, c)
		case *CircuitBreakerResumeProposal:
			return HandleCircuitBreakerResumeProposal(ctx, k, c)
		default:
			return fmt.Errorf("unrecognized circuit breaker proposal content type: %T", c)
		}
	}
}
