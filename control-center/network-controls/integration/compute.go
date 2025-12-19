package integration

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	computekeeper "github.com/paw-chain/paw/x/compute/keeper"
	computetypes "github.com/paw-chain/paw/x/compute/types"
)

// ComputeIntegration provides integration with the Compute module
type ComputeIntegration struct {
	keeper *computekeeper.Keeper
}

// NewComputeIntegration creates a new Compute integration
func NewComputeIntegration(keeper *computekeeper.Keeper) *ComputeIntegration {
	return &ComputeIntegration{
		keeper: keeper,
	}
}

// Pause pauses all Compute operations
func (c *ComputeIntegration) Pause(ctx sdk.Context, actor, reason string) error {
	return c.keeper.OpenCircuitBreaker(sdk.WrapSDKContext(ctx), actor, reason)
}

// Resume resumes all Compute operations
func (c *ComputeIntegration) Resume(ctx sdk.Context, actor, reason string) error {
	return c.keeper.CloseCircuitBreaker(sdk.WrapSDKContext(ctx), actor, reason)
}

// IsBlocked checks if Compute operations are blocked
func (c *ComputeIntegration) IsBlocked(ctx sdk.Context) bool {
	return c.keeper.IsCircuitBreakerOpen(sdk.WrapSDKContext(ctx))
}

// GetState retrieves the circuit breaker state
func (c *ComputeIntegration) GetState(ctx sdk.Context) (bool, string, string) {
	return c.keeper.GetCircuitBreakerState(sdk.WrapSDKContext(ctx))
}

// PauseProvider pauses a specific provider
func (c *ComputeIntegration) PauseProvider(ctx sdk.Context, providerAddr, actor, reason string) error {
	return c.keeper.OpenProviderCircuitBreaker(sdk.WrapSDKContext(ctx), providerAddr, actor, reason)
}

// ResumeProvider resumes a specific provider
func (c *ComputeIntegration) ResumeProvider(ctx sdk.Context, providerAddr, actor, reason string) error {
	return c.keeper.CloseProviderCircuitBreaker(sdk.WrapSDKContext(ctx), providerAddr, actor, reason)
}

// IsProviderBlocked checks if a provider is blocked
func (c *ComputeIntegration) IsProviderBlocked(ctx sdk.Context, providerAddr string) bool {
	return c.keeper.IsProviderCircuitBreakerOpen(sdk.WrapSDKContext(ctx), providerAddr)
}

// CancelJob cancels a specific job
func (c *ComputeIntegration) CancelJob(ctx sdk.Context, jobID, actor, reason string) error {
	return c.keeper.CancelJob(sdk.WrapSDKContext(ctx), jobID, actor, reason)
}

// IsJobCancelled checks if a job is cancelled
func (c *ComputeIntegration) IsJobCancelled(ctx sdk.Context, jobID string) bool {
	return c.keeper.IsJobCancelled(sdk.WrapSDKContext(ctx), jobID)
}

// OverrideReputation sets a temporary reputation override
func (c *ComputeIntegration) OverrideReputation(ctx sdk.Context, providerAddr string, score int64, actor, reason string) error {
	return c.keeper.SetReputationOverride(sdk.WrapSDKContext(ctx), providerAddr, score, actor, reason)
}

// ClearReputationOverride removes a reputation override
func (c *ComputeIntegration) ClearReputationOverride(ctx sdk.Context, providerAddr string) {
	c.keeper.ClearReputationOverride(sdk.WrapSDKContext(ctx), providerAddr)
}

// GetReputationWithOverride retrieves reputation with override check
func (c *ComputeIntegration) GetReputationWithOverride(ctx sdk.Context, providerAddr string) (int64, bool) {
	return c.keeper.GetReputationWithOverride(sdk.WrapSDKContext(ctx), providerAddr)
}

// GetAllProviders retrieves all providers (for status reporting)
func (c *ComputeIntegration) GetAllProviders(ctx sdk.Context) ([]string, error) {
	goCtx := sdk.WrapSDKContext(ctx)
	addresses := make([]string, 0)
	err := c.keeper.IterateProviders(goCtx, func(provider computetypes.Provider) (bool, error) {
		addresses = append(addresses, provider.Address)
		return false, nil
	})
	return addresses, err
}

// GetActiveJobs retrieves all active jobs
func (c *ComputeIntegration) GetActiveJobs(ctx sdk.Context) ([]string, error) {
	goCtx := sdk.WrapSDKContext(ctx)
	jobIDs := make([]string, 0)
	err := c.keeper.IterateRequests(goCtx, func(req computetypes.Request) (bool, error) {
		if req.Status == computetypes.REQUEST_STATUS_PENDING || req.Status == computetypes.REQUEST_STATUS_PROCESSING || req.Status == computetypes.REQUEST_STATUS_ASSIGNED {
			jobIDs = append(jobIDs, strconv.FormatUint(req.Id, 10))
		}
		return false, nil
	})
	return jobIDs, err
}

// EmergencyJobTermination terminates all jobs for a provider
func (c *ComputeIntegration) EmergencyJobTermination(ctx sdk.Context, providerAddr, actor, reason string) error {
	// Pause the provider
	if err := c.PauseProvider(ctx, providerAddr, actor, reason); err != nil {
		return fmt.Errorf("failed to pause provider: %w", err)
	}

	// Get all active jobs for this provider
	goCtx := sdk.WrapSDKContext(ctx)
	var cancelErr error
	err := c.keeper.IterateRequests(goCtx, func(req computetypes.Request) (bool, error) {
		if cancelErr != nil {
			return true, cancelErr
		}
		if req.Provider == providerAddr && (req.Status == computetypes.REQUEST_STATUS_PENDING || req.Status == computetypes.REQUEST_STATUS_PROCESSING || req.Status == computetypes.REQUEST_STATUS_ASSIGNED) {
			jobID := strconv.FormatUint(req.Id, 10)
			if err := c.keeper.CancelJob(goCtx, jobID, actor, "emergency termination: "+reason); err != nil {
				cancelErr = fmt.Errorf("failed to cancel job %s: %w", jobID, err)
				return true, cancelErr
			}
		}
		return false, nil
	})
	if cancelErr != nil {
		return cancelErr
	}
	if err != nil {
		return err
	}

	return nil
}

// BulkCancelJobs cancels multiple jobs at once
func (c *ComputeIntegration) BulkCancelJobs(ctx sdk.Context, jobIDs []string, actor, reason string) error {
	for _, jobID := range jobIDs {
		if err := c.keeper.CancelJob(sdk.WrapSDKContext(ctx), jobID, actor, reason); err != nil {
			return fmt.Errorf("failed to cancel job %s: %w", jobID, err)
		}
	}
	return nil
}

// ValidateProviderHealth checks provider health before operations
func (c *ComputeIntegration) ValidateProviderHealth(ctx sdk.Context, providerAddr string) error {
	// Check if provider operations are allowed
	if err := c.keeper.CheckProviderCircuitBreaker(sdk.WrapSDKContext(ctx), providerAddr); err != nil {
		return err
	}

	// Additional health checks could go here
	// For example: reputation thresholds, uptime, response times, etc.

	return nil
}
