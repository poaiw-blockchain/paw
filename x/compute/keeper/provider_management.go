package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// TASKS 111-120: Complete provider management implementation

// TASK 111: Provider stake slashing implementation
func (k Keeper) SlashProviderStake(
	ctx context.Context,
	providerAddr sdk.AccAddress,
	slashFraction sdkmath.LegacyDec,
	reason string,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get provider info
	provider, err := k.GetProvider(ctx, providerAddr)
	if err != nil {
		return fmt.Errorf("provider not found: %w", err)
	}

	// Calculate slash amount
	slashAmount := slashFraction.MulInt(provider.Stake).TruncateInt()

	if slashAmount.IsZero() {
		return fmt.Errorf("slash amount is zero")
	}

	// Burn slashed tokens from module account
	slashCoins := sdk.NewCoins(sdk.NewCoin("upaw", slashAmount))
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, slashCoins); err != nil {
		return fmt.Errorf("failed to burn slashed tokens: %w", err)
	}

	// Update provider stake
	provider.Stake = provider.Stake.Sub(slashAmount)
	if err := k.SetProvider(ctx, *provider); err != nil {
		return fmt.Errorf("failed to update provider: %w", err)
	}

	// Record slash event
	if err := k.recordProviderSlash(ctx, providerAddr, slashAmount, reason); err != nil {
		sdkCtx.Logger().Error("failed to record provider slash", "error", err)
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"provider_slashed",
			sdk.NewAttribute("provider", providerAddr.String()),
			sdk.NewAttribute("amount", slashAmount.String()),
			sdk.NewAttribute("fraction", slashFraction.String()),
			sdk.NewAttribute("reason", reason),
		),
	)

	sdkCtx.Logger().Info("provider slashed",
		"provider", providerAddr.String(),
		"amount", slashAmount.String(),
		"reason", reason,
	)

	return nil
}

// recordProviderSlash records slash history
func (k Keeper) recordProviderSlash(
	ctx context.Context,
	provider sdk.AccAddress,
	amount sdkmath.Int,
	reason string,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	slashKey := []byte(fmt.Sprintf("slash_%s_%d", provider.String(), sdkCtx.BlockHeight()))

	slashData := map[string]interface{}{
		"provider":     provider.String(),
		"amount":       amount.String(),
		"reason":       reason,
		"block_height": sdkCtx.BlockHeight(),
		"timestamp":    sdkCtx.BlockTime().Unix(),
	}

	bz, err := json.Marshal(slashData)
	if err != nil {
		return fmt.Errorf("recordSlashEvent: marshal: %w", err)
	}

	store.Set(slashKey, bz)
	return nil
}

// TASK 112: Provider reputation decay with time-based factors
func (k Keeper) ApplyReputationDecayToAll(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	decayedCount := 0

	err := k.IterateProviderReputations(ctx, func(rep types.ProviderReputation) (bool, error) {
		// Check if decay is needed (e.g., every 24 hours)
		timeSinceDecay := now.Sub(rep.LastDecayTimestamp)
		if timeSinceDecay < 24*time.Hour {
			return false, nil // Continue to next
		}

		// Apply exponential decay
		decayFactor := 0.99 // 1% decay per day

		rep.ReliabilityScore *= decayFactor
		rep.SpeedScore *= decayFactor
		rep.AccuracyScore *= decayFactor
		rep.AvailabilityScore *= decayFactor

		// Recalculate overall score
		rep.OverallScore = uint32(
			(rep.ReliabilityScore + rep.SpeedScore + rep.AccuracyScore + rep.AvailabilityScore) * 25,
		)

		rep.LastDecayTimestamp = now

		if err := k.SetProviderReputation(ctx, rep); err != nil {
			sdkCtx.Logger().Error("failed to apply reputation decay", "provider", rep.Provider, "error", err)
			return false, nil
		}

		decayedCount++
		return false, nil
	})

	if err != nil {
		return fmt.Errorf("ApplyReputationDecayToAll: iterate reputations: %w", err)
	}

	if decayedCount > 0 {
		sdkCtx.Logger().Info("applied reputation decay",
			"count", decayedCount,
			"timestamp", now,
		)
	}

	return nil
}

// TASK 113: Provider performance tracking
type ProviderPerformanceMetrics struct {
	Provider                 string
	JobsCompleted            uint64
	JobsFailed               uint64
	AverageResponseTime      time.Duration
	AverageVerificationScore float64
	Uptime                   float64 // Percentage
	LastActiveTimestamp      time.Time
	CurrentLoad              uint64
	MaxLoad                  uint64
}

func (k Keeper) TrackProviderPerformance(
	ctx context.Context,
	providerAddr sdk.AccAddress,
	jobCompleted bool,
	responseTime time.Duration,
	verificationScore uint32,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	metricsKey := []byte(fmt.Sprintf("perf_%s", providerAddr.String()))

	var metrics ProviderPerformanceMetrics

	// Load existing metrics
	if bz := store.Get(metricsKey); bz != nil {
		var data map[string]interface{}
		if err := json.Unmarshal(bz, &data); err == nil {
			metrics.Provider = providerAddr.String()
			if jobs, ok := data["jobs_completed"].(float64); ok {
				metrics.JobsCompleted = uint64(jobs)
			}
			if failed, ok := data["jobs_failed"].(float64); ok {
				metrics.JobsFailed = uint64(failed)
			}
		}
	} else {
		metrics.Provider = providerAddr.String()
	}

	// Update metrics
	if jobCompleted {
		metrics.JobsCompleted++
	} else {
		metrics.JobsFailed++
	}

	metrics.LastActiveTimestamp = sdkCtx.BlockTime()

	// Calculate average response time
	totalJobs := metrics.JobsCompleted + metrics.JobsFailed
	if totalJobs > 0 {
		oldAvg := metrics.AverageResponseTime
		jobCount := types.SaturateUint64ToInt64(totalJobs)
		if jobCount > 0 {
			metrics.AverageResponseTime = (oldAvg*time.Duration(jobCount-1) + responseTime) / time.Duration(jobCount)
		}
	}

	// Store updated metrics
	metricsData := map[string]interface{}{
		"provider":        metrics.Provider,
		"jobs_completed":  metrics.JobsCompleted,
		"jobs_failed":     metrics.JobsFailed,
		"avg_response_ms": metrics.AverageResponseTime.Milliseconds(),
		"last_active":     metrics.LastActiveTimestamp.Unix(),
	}

	bz, err := json.Marshal(metricsData)
	if err != nil {
		return fmt.Errorf("RecordProviderPerformance: marshal metrics: %w", err)
	}

	store.Set(metricsKey, bz)

	// Emit metrics event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"provider_performance_updated",
			sdk.NewAttribute("provider", providerAddr.String()),
			sdk.NewAttribute("jobs_completed", fmt.Sprintf("%d", metrics.JobsCompleted)),
			sdk.NewAttribute("jobs_failed", fmt.Sprintf("%d", metrics.JobsFailed)),
		),
	)

	return nil
}

// TASK 115: Provider availability monitoring
func (k Keeper) MonitorProviderAvailability(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	unavailableProviders := 0

	err := k.IterateProviders(ctx, func(provider types.Provider) (bool, error) {
		// Check last activity
		store := sdkCtx.KVStore(k.storeKey)
		metricsKey := []byte(fmt.Sprintf("perf_%s", provider.Address))

		if bz := store.Get(metricsKey); bz != nil {
			var data map[string]interface{}
			if err := json.Unmarshal(bz, &data); err == nil {
				if lastActive, ok := data["last_active"].(float64); ok {
					lastActiveTime := time.Unix(int64(lastActive), 0)
					inactive := now.Sub(lastActiveTime)

					// Consider provider unavailable if inactive for > 1 hour
					if inactive > 1*time.Hour {
						sdkCtx.Logger().Warn("provider inactive",
							"provider", provider.Address,
							"inactive_duration", inactive.String(),
						)

						unavailableProviders++

						// Update availability score in reputation
						// FIXED CODE-1.1: Replace MustAccAddressFromBech32 with error-handling variant
						providerAddr, addrErr := sdk.AccAddressFromBech32(provider.Address)
						if addrErr != nil {
							sdkCtx.Logger().Error("invalid provider address in state", "provider", provider.Address, "error", addrErr)
						} else if rep, err := k.GetProviderReputation(ctx, providerAddr); err == nil {
							rep.AvailabilityScore *= 0.95 // Reduce availability score by 5%
							if err := k.SetProviderReputation(ctx, *rep); err != nil {
								sdkCtx.Logger().Error("failed to persist availability score adjustment", "provider", provider.Address, "error", err)
							}
						} else {
							sdkCtx.Logger().Error("failed to load provider reputation for availability penalty", "provider", provider.Address, "error", err)
						}
					}
				}
			}
		}

		return false, nil
	})

	if err != nil {
		return fmt.Errorf("MonitorProviderAvailability: iterate providers: %w", err)
	}

	if unavailableProviders > 0 {
		sdkCtx.Logger().Info("provider availability check",
			"unavailable_count", unavailableProviders,
		)
	}

	return nil
}

// TASK 116: Resource quota enforcement

func (k Keeper) EnforceResourceQuota(
	ctx context.Context,
	providerAddr sdk.AccAddress,
	specs types.ComputeSpec,
) error {
	// Get or create quota
	quota, err := k.GetResourceQuota(ctx, providerAddr)
	if err != nil {
		// Use default if not found
		quota = k.GetDefaultResourceQuota(providerAddr.String())
	}

	// Check if quota allows new job
	if quota.CurrentRequests >= quota.MaxConcurrentRequests {
		return fmt.Errorf("provider at maximum concurrent job capacity: %d/%d",
			quota.CurrentRequests, quota.MaxConcurrentRequests)
	}

	// Check resource-specific quotas
	if specs.GpuCount > 0 && quota.CurrentGpus+uint64(specs.GpuCount) > quota.MaxTotalGpus {
		return fmt.Errorf("provider at maximum GPU capacity")
	}

	return nil
}

// UpdateResourceQuota updates provider resource usage
func (k Keeper) UpdateResourceQuota(
	ctx context.Context,
	providerAddr sdk.AccAddress,
	deltaJobs int64,
	deltaMemoryMB int64,
	deltaCPU int64,
	deltaGPU int64,
) error {
	// Get or create quota
	quota, err := k.GetResourceQuota(ctx, providerAddr)
	if err != nil {
		// Use default if not found
		quota = k.GetDefaultResourceQuota(providerAddr.String())
	}

	// Update quota
	if deltaJobs > 0 {
		quota.CurrentRequests += uint64(deltaJobs)
	} else if deltaJobs < 0 && quota.CurrentRequests >= uint64(-deltaJobs) {
		quota.CurrentRequests -= uint64(-deltaJobs)
	}

	if deltaMemoryMB > 0 {
		quota.CurrentMemory += uint64(deltaMemoryMB)
	} else if deltaMemoryMB < 0 && quota.CurrentMemory >= uint64(-deltaMemoryMB) {
		quota.CurrentMemory -= uint64(-deltaMemoryMB)
	}

	if deltaCPU > 0 {
		quota.CurrentCpu += uint64(deltaCPU)
	} else if deltaCPU < 0 && quota.CurrentCpu >= uint64(-deltaCPU) {
		quota.CurrentCpu -= uint64(-deltaCPU)
	}

	if deltaGPU > 0 {
		quota.CurrentGpus += uint64(deltaGPU)
	} else if deltaGPU < 0 && quota.CurrentGpus >= uint64(-deltaGPU) {
		quota.CurrentGpus -= uint64(-deltaGPU)
	}

	quota.LastUpdated = time.Now()

	// Store updated quota
	return k.SetResourceQuota(ctx, *quota)
}

// TASK 117: Provider load balancing
func (k Keeper) GetProviderLoad(ctx context.Context, providerAddr sdk.AccAddress) (uint64, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	quotaKey := []byte(fmt.Sprintf("quota_%s", providerAddr.String()))

	if bz := store.Get(quotaKey); bz != nil {
		var data map[string]interface{}
		if err := json.Unmarshal(bz, &data); err == nil {
			if current, ok := data["current_jobs"].(float64); ok {
				return uint64(current), nil
			}
		}
	}

	return 0, nil
}

// SelectProviderWithLoadBalancing selects provider considering load
func (k Keeper) SelectProviderWithLoadBalancing(
	ctx context.Context,
	specs types.ComputeSpec,
) (sdk.AccAddress, error) {
	var bestProvider sdk.AccAddress
	var minLoad uint64 = ^uint64(0) // Max uint64

	err := k.IterateProviders(ctx, func(provider types.Provider) (bool, error) {
		// FIXED CODE-1.1: Replace MustAccAddressFromBech32 with error-handling variant
		providerAddr, addrErr := sdk.AccAddressFromBech32(provider.Address)
		if addrErr != nil {
			// Skip invalid addresses in state (shouldn't happen but handle gracefully)
			return false, nil
		}

		// Check if provider meets requirements
		if !provider.Active {
			return false, nil
		}

		// Get current load
		load, err := k.GetProviderLoad(ctx, providerAddr)
		if err != nil {
			return false, nil
		}

		// Select provider with lowest load
		if load < minLoad {
			minLoad = load
			bestProvider = providerAddr
		}

		return false, nil
	})

	if err != nil {
		return nil, fmt.Errorf("SelectProviderWithLoadBalancing: iterate providers: %w", err)
	}

	if bestProvider.Empty() {
		return nil, fmt.Errorf("no available provider found")
	}

	return bestProvider, nil
}

// TASK 118: Request timeout handling
func (k Keeper) HandleRequestTimeout(ctx context.Context, requestID uint64) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get request
	request, err := k.GetRequest(ctx, requestID)
	if err != nil || request == nil {
		return fmt.Errorf("request not found: %d", requestID)
	}

	sdkCtx.Logger().Error("request timed out",
		"request_id", requestID,
		"requester", request.Requester,
	)

	// Refund escrow
	if err := k.refundEscrow(sdkCtx, fmt.Sprintf("req_%d", requestID)); err != nil {
		sdkCtx.Logger().Error("failed to refund escrow on timeout", "error", err)
	}

	now := sdkCtx.BlockTime()

	// Penalize provider reputation (10% reliability penalty) if provider is set
	if len(request.Provider) > 0 {
		providerAddr, addrErr := sdk.AccAddressFromBech32(request.Provider)
		if addrErr != nil {
			sdkCtx.Logger().Error("invalid provider address on timed-out request", "provider", request.Provider, "error", addrErr)
		} else if rep, err := k.GetProviderReputation(ctx, providerAddr); err == nil {
			rep.FailedRequests++
			rep.ReliabilityScore *= 0.90
			if rep.ReliabilityScore < 0 {
				rep.ReliabilityScore = 0
			}
			rep.LastUpdateTimestamp = now
			rep.OverallScore = recalculateOverallScore(rep)

			if err := k.SetProviderReputation(ctx, *rep); err != nil {
				sdkCtx.Logger().Error("failed to persist reputation penalty", "provider", request.Provider, "error", err)
			} else if providerRecord, err := k.GetProvider(ctx, providerAddr); err == nil {
				providerRecord.Reputation = rep.OverallScore
				if err := k.SetProvider(ctx, *providerRecord); err != nil {
					sdkCtx.Logger().Error("failed to update provider with penalized reputation", "provider", request.Provider, "error", err)
				}
			}
		} else {
			sdkCtx.Logger().Error("failed to load provider reputation for timeout penalty", "provider", request.Provider, "error", err)
		}
	}

	// Update request status
	request.Status = types.REQUEST_STATUS_FAILED // timeout -> failed
	if err := k.SetRequest(ctx, *request); err != nil {
		sdkCtx.Logger().Error("failed to persist timed-out request", "request_id", requestID, "error", err)
	}

	// Emit timeout event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"request_timeout",
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
			sdk.NewAttribute("requester", request.Requester),
		),
	)

	return nil
}

func recalculateOverallScore(rep *types.ProviderReputation) uint32 {
	overall := 0.4*rep.ReliabilityScore +
		0.3*rep.AccuracyScore +
		0.2*rep.SpeedScore +
		0.1*rep.AvailabilityScore

	if overall < 0 {
		overall = 0
	} else if overall > 1 {
		overall = 1
	}

	return uint32(overall * 100)
}

// TASK 120: Request priority queue
type RequestPriority uint8

const (
	PriorityLow      RequestPriority = 0
	PriorityNormal   RequestPriority = 1
	PriorityHigh     RequestPriority = 2
	PriorityCritical RequestPriority = 3
)

type PrioritizedRequest struct {
	RequestID uint64
	Priority  RequestPriority
	Timestamp time.Time
	Fee       sdkmath.Int
}

func (k Keeper) EnqueuePrioritizedRequest(
	ctx context.Context,
	requestID uint64,
	priority RequestPriority,
	fee sdkmath.Int,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	// Store in priority-based key for efficient retrieval
	priorityKey := []byte(fmt.Sprintf("priority_%d_%d_%d", priority, sdkCtx.BlockTime().Unix(), requestID))

	requestData := PrioritizedRequest{
		RequestID: requestID,
		Priority:  priority,
		Timestamp: sdkCtx.BlockTime(),
		Fee:       fee,
	}

	data := map[string]interface{}{
		"request_id": requestID,
		"priority":   uint8(priority),
		"timestamp":  requestData.Timestamp.Unix(),
		"fee":        fee.String(),
	}

	bz, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("PrioritizeRequest: marshal: %w", err)
	}

	store.Set(priorityKey, bz)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"request_prioritized",
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
			sdk.NewAttribute("priority", fmt.Sprintf("%d", priority)),
		),
	)

	return nil
}

// DequeueHighestPriorityRequest retrieves the highest priority pending request
func (k Keeper) DequeueHighestPriorityRequest(ctx context.Context) (*PrioritizedRequest, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	// Iterate from highest to lowest priority
	for priority := PriorityCritical; ; priority-- {
		prefix := []byte(fmt.Sprintf("priority_%d_", priority))
		iterator := storetypes.KVStorePrefixIterator(store, prefix)

		if iterator.Valid() {
			var data map[string]interface{}
			if err := json.Unmarshal(iterator.Value(), &data); err == nil {
				request := &PrioritizedRequest{
					RequestID: uint64(data["request_id"].(float64)),
					Priority:  RequestPriority(data["priority"].(float64)),
				}

				// Remove from queue
				store.Delete(iterator.Key())
				if err := iterator.Close(); err != nil {
					sdkCtx.Logger().Error("failed closing iterator after dequeue", "priority", priority, "error", err)
				}

				return request, nil
			}
		}
		if err := iterator.Close(); err != nil {
			sdkCtx.Logger().Error("failed closing iterator for priority bucket", "priority", priority, "error", err)
		}

		if priority == PriorityLow {
			break
		}
	}

	return nil, fmt.Errorf("no pending requests in priority queue")
}
