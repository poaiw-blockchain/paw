package integration

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	computekeeper "github.com/paw-chain/paw/x/compute/keeper"
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
	return c.keeper.OpenCircuitBreaker(ctx, actor, reason)
}

// Resume resumes all Compute operations
func (c *ComputeIntegration) Resume(ctx sdk.Context, actor, reason string) error {
	return c.keeper.CloseCircuitBreaker(ctx, actor, reason)
}

// IsBlocked checks if Compute operations are blocked
func (c *ComputeIntegration) IsBlocked(ctx sdk.Context) bool {
	return c.keeper.IsCircuitBreakerOpen(ctx)
}

// GetState retrieves the circuit breaker state
func (c *ComputeIntegration) GetState(ctx sdk.Context) (bool, string, string) {
	return c.keeper.GetCircuitBreakerState(ctx)
}

// PauseProvider pauses a specific provider
func (c *ComputeIntegration) PauseProvider(ctx sdk.Context, providerAddr, actor, reason string) error {
	return c.keeper.OpenProviderCircuitBreaker(ctx, providerAddr, actor, reason)
}

// ResumeProvider resumes a specific provider
func (c *ComputeIntegration) ResumeProvider(ctx sdk.Context, providerAddr, actor, reason string) error {
	return c.keeper.CloseProviderCircuitBreaker(ctx, providerAddr, actor, reason)
}

// IsProviderBlocked checks if a provider is blocked
func (c *ComputeIntegration) IsProviderBlocked(ctx sdk.Context, providerAddr string) bool {
	return c.keeper.IsProviderCircuitBreakerOpen(ctx, providerAddr)
}

// CancelJob cancels a specific job
func (c *ComputeIntegration) CancelJob(ctx sdk.Context, jobID, actor, reason string) error {
	return c.keeper.CancelJob(ctx, jobID, actor, reason)
}

// IsJobCancelled checks if a job is cancelled
func (c *ComputeIntegration) IsJobCancelled(ctx sdk.Context, jobID string) bool {
	return c.keeper.IsJobCancelled(ctx, jobID)
}

// OverrideReputation sets a temporary reputation override
func (c *ComputeIntegration) OverrideReputation(ctx sdk.Context, providerAddr string, score int64, actor, reason string) error {
	return c.keeper.SetReputationOverride(ctx, providerAddr, score, actor, reason)
}

// ClearReputationOverride removes a reputation override
func (c *ComputeIntegration) ClearReputationOverride(ctx sdk.Context, providerAddr string) {
	c.keeper.ClearReputationOverride(ctx, providerAddr)
}

// GetReputationWithOverride retrieves reputation with override check
func (c *ComputeIntegration) GetReputationWithOverride(ctx sdk.Context, providerAddr string) (int64, bool) {
	return c.keeper.GetReputationWithOverride(ctx, providerAddr)
}

// GetAllProviders retrieves all providers (for status reporting)
func (c *ComputeIntegration) GetAllProviders(ctx sdk.Context) ([]string, error) {
	// This would retrieve all provider addresses from the keeper
	// Implementation depends on how providers are stored in the Compute module
	providers := c.keeper.GetAllProviders(ctx)
	addresses := make([]string, len(providers))
	for i, provider := range providers {
		addresses[i] = provider.Address
	}
	return addresses, nil
}

// GetActiveJobs retrieves all active jobs
func (c *ComputeIntegration) GetActiveJobs(ctx sdk.Context) ([]string, error) {
	// This would retrieve all active job IDs from the keeper
	// Implementation depends on how jobs are stored in the Compute module
	requests := c.keeper.GetAllRequests(ctx)
	jobIDs := make([]string, 0)
	for _, req := range requests {
		if req.Status == "pending" || req.Status == "processing" {
			jobIDs = append(jobIDs, req.RequestId)
		}
	}
	return jobIDs, nil
}

// EmergencyJobTermination terminates all jobs for a provider
func (c *ComputeIntegration) EmergencyJobTermination(ctx sdk.Context, providerAddr, actor, reason string) error {
	// Pause the provider
	if err := c.PauseProvider(ctx, providerAddr, actor, reason); err != nil {
		return fmt.Errorf("failed to pause provider: %w", err)
	}

	// Get all active jobs for this provider
	requests := c.keeper.GetAllRequests(ctx)
	cancelledCount := 0

	for _, req := range requests {
		if req.Provider == providerAddr && (req.Status == "pending" || req.Status == "processing") {
			if err := c.keeper.CancelJob(ctx, req.RequestId, actor, "emergency termination: "+reason); err != nil {
				return fmt.Errorf("failed to cancel job %s: %w", req.RequestId, err)
			}
			cancelledCount++
		}
	}

	return nil
}

// BulkCancelJobs cancels multiple jobs at once
func (c *ComputeIntegration) BulkCancelJobs(ctx sdk.Context, jobIDs []string, actor, reason string) error {
	for _, jobID := range jobIDs {
		if err := c.keeper.CancelJob(ctx, jobID, actor, reason); err != nil {
			return fmt.Errorf("failed to cancel job %s: %w", jobID, err)
		}
	}
	return nil
}

// ValidateProviderHealth checks provider health before operations
func (c *ComputeIntegration) ValidateProviderHealth(ctx sdk.Context, providerAddr string) error {
	// Check if provider operations are allowed
	if err := c.keeper.CheckProviderCircuitBreaker(ctx, providerAddr); err != nil {
		return err
	}

	// Additional health checks could go here
	// For example: reputation thresholds, uptime, response times, etc.

	return nil
}
