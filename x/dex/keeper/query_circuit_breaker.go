package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CircuitBreakerConfigResponse is the response for circuit breaker config query
type CircuitBreakerConfigResponse struct {
	Config CircuitBreakerConfig `json:"config"`
}

// CircuitBreakerStateResponse is the response for circuit breaker state query
type CircuitBreakerStateResponse struct {
	State CircuitBreakerState `json:"state"`
}

// CircuitBreakerStatesResponse is the response for all circuit breaker states query
type CircuitBreakerStatesResponse struct {
	States []CircuitBreakerState `json:"states"`
}

// CircuitBreakerStatusResponse provides a human-readable status
type CircuitBreakerStatusResponse struct {
	PoolId             uint64 `json:"pool_id"`
	IsTripped          bool   `json:"is_tripped"`
	Status             string `json:"status"` // "active", "cooldown", "gradual_resume", "normal"
	TripReason         string `json:"trip_reason"`
	SecondsUntilResume int64  `json:"seconds_until_resume"` // 0 if not in cooldown
	InGradualResume    bool   `json:"in_gradual_resume"`
	MaxSwapPercentage  string `json:"max_swap_percentage"` // e.g., "50%" or "100%"
}

// QueryCircuitBreakerConfig queries the circuit breaker configuration
func (k Keeper) QueryCircuitBreakerConfig(goCtx context.Context) (*CircuitBreakerConfigResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	config := k.GetCircuitBreakerConfig(ctx)

	return &CircuitBreakerConfigResponse{
		Config: config,
	}, nil
}

// QueryCircuitBreakerState queries the circuit breaker state for a specific pool
func (k Keeper) QueryCircuitBreakerState(goCtx context.Context, poolId uint64) (*CircuitBreakerStateResponse, error) {
	if poolId == 0 {
		return nil, status.Error(codes.InvalidArgument, "pool id cannot be zero")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	state := k.GetCircuitBreakerState(ctx, poolId)

	return &CircuitBreakerStateResponse{
		State: state,
	}, nil
}

// QueryAllCircuitBreakerStates queries all circuit breaker states
func (k Keeper) QueryAllCircuitBreakerStates(goCtx context.Context) (*CircuitBreakerStatesResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	states := k.GetAllCircuitBreakerStates(ctx)

	return &CircuitBreakerStatesResponse{
		States: states,
	}, nil
}

// QueryCircuitBreakerStatus queries the human-readable status for a pool
func (k Keeper) QueryCircuitBreakerStatus(goCtx context.Context, poolId uint64) (*CircuitBreakerStatusResponse, error) {
	if poolId == 0 {
		return nil, status.Error(codes.InvalidArgument, "pool id cannot be zero")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	state := k.GetCircuitBreakerState(ctx, poolId)
	config := k.GetCircuitBreakerConfig(ctx)
	currentTime := ctx.BlockTime().Unix()

	response := CircuitBreakerStatusResponse{
		PoolId:    poolId,
		IsTripped: state.IsTripped,
	}

	if !state.IsTripped {
		// Check if in gradual resume mode
		if state.GradualResume && state.ResumeStartedAt > 0 {
			gradualResumePeriod := int64(3600) // 1 hour
			if currentTime <= state.ResumeStartedAt+gradualResumePeriod {
				response.Status = "gradual_resume"
				response.InGradualResume = true
				response.MaxSwapPercentage = config.ResumeVolumeFactor.MulInt64(100).String() + "%"
			} else {
				response.Status = "normal"
				response.InGradualResume = false
				response.MaxSwapPercentage = "100%"
			}
		} else {
			response.Status = "normal"
			response.InGradualResume = false
			response.MaxSwapPercentage = "100%"
		}
		response.TripReason = ""
		response.SecondsUntilResume = 0
	} else {
		// Circuit breaker is tripped
		if currentTime < state.CanResumeAt {
			response.Status = "cooldown"
			response.SecondsUntilResume = state.CanResumeAt - currentTime
		} else {
			response.Status = "active"
			response.SecondsUntilResume = 0
		}
		response.TripReason = state.TripReason
		response.InGradualResume = false
		response.MaxSwapPercentage = "0%"
	}

	return &response, nil
}

// QueryActiveCircuitBreakers queries all pools with active circuit breakers
func (k Keeper) QueryActiveCircuitBreakers(goCtx context.Context) (*CircuitBreakerStatesResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	allStates := k.GetAllCircuitBreakerStates(ctx)

	// Filter for only active (tripped) circuit breakers
	activeStates := []CircuitBreakerState{}
	for _, state := range allStates {
		if state.IsTripped {
			activeStates = append(activeStates, state)
		}
	}

	return &CircuitBreakerStatesResponse{
		States: activeStates,
	}, nil
}
