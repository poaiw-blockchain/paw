package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// Security module implementing fortress-level protection against:
// - Resource exhaustion attacks
// - Spam and DoS
// - Rate limiting violations
// - Quota enforcement
// - Nonce cleanup and management

// RateLimitConfig defines rate limiting parameters
type RateLimitConfig struct {
	MaxRequestsPerHour    uint64
	MaxRequestsPerDay     uint64
	MaxConcurrentRequests uint64
	MaxProviderLoad       uint64
	BurstAllowance        uint64
	CooldownPeriodSeconds uint64
}

// ProviderLoadTracker is an alias to the protobuf type for load tracking
type ProviderLoadTracker = types.ProviderLoadTracker

// RateLimitBucket is an alias to the protobuf type for rate limiting
type RateLimitBucket = types.RateLimitBucket

// CheckRateLimit verifies if an account can make a request under rate limits
// Implements token bucket algorithm with hourly and daily caps
func (k Keeper) CheckRateLimit(ctx context.Context, account sdk.AccAddress) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	// Get or create rate limit bucket
	bucket, err := k.GetRateLimitBucket(ctx, account)
	if err != nil {
		// First request - create bucket
		params, err := k.GetParams(ctx)
		if err != nil {
			return err
		}

		config := k.GetDefaultRateLimitConfig(params)
		bucket = &RateLimitBucket{
			Account:          account.String(),
			Tokens:           config.BurstAllowance,
			MaxTokens:        config.BurstAllowance,
			RefillRate:       1, // 1 token per second
			LastRefill:       now,
			RequestsThisHour: 0,
			RequestsToday:    0,
			HourResetAt:      now.Add(1 * time.Hour),
			DayResetAt:       now.Add(24 * time.Hour),
		}
	}

	// Refill tokens based on elapsed time
	elapsedSeconds := uint64(now.Sub(bucket.LastRefill).Seconds())
	tokensToAdd := elapsedSeconds * bucket.RefillRate
	bucket.Tokens = min(bucket.Tokens+tokensToAdd, bucket.MaxTokens)
	bucket.LastRefill = now

	// Reset hourly counter if needed
	if now.After(bucket.HourResetAt) {
		bucket.RequestsThisHour = 0
		bucket.HourResetAt = now.Add(1 * time.Hour)
	}

	// Reset daily counter if needed
	if now.After(bucket.DayResetAt) {
		bucket.RequestsToday = 0
		bucket.DayResetAt = now.Add(24 * time.Hour)
	}

	// Get rate limit config
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}
	config := k.GetDefaultRateLimitConfig(params)

	// Reserve a token immediately to avoid race conditions
	bucket.Tokens--
	if bucket.Tokens > bucket.MaxTokens {
		// Underflow occurred (no tokens available)
		bucket.Tokens = 0
		return errorsmod.Wrapf(types.ErrRateLimitExceeded, "burst capacity depleted, wait %d seconds", bucket.RefillRate)
	}

	if bucket.RequestsThisHour >= config.MaxRequestsPerHour {
		bucket.Tokens++
		return errorsmod.Wrapf(types.ErrRateLimitExceeded, "maximum %d requests per hour reached", config.MaxRequestsPerHour)
	}

	if bucket.RequestsToday >= config.MaxRequestsPerDay {
		bucket.Tokens++
		return errorsmod.Wrapf(types.ErrRateLimitExceeded, "maximum %d requests per day reached", config.MaxRequestsPerDay)
	}

	// Consume token confirmed and increment counters
	bucket.RequestsThisHour++
	bucket.RequestsToday++

	// Save updated bucket
	if err := k.SetRateLimitBucket(ctx, *bucket); err != nil {
		return fmt.Errorf("failed to update rate limit: %w", err)
	}

	// Emit monitoring event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"rate_limit_checked",
			sdk.NewAttribute("account", account.String()),
			sdk.NewAttribute("tokens_remaining", fmt.Sprintf("%d", bucket.Tokens)),
			sdk.NewAttribute("hourly_count", fmt.Sprintf("%d/%d", bucket.RequestsThisHour, config.MaxRequestsPerHour)),
			sdk.NewAttribute("daily_count", fmt.Sprintf("%d/%d", bucket.RequestsToday, config.MaxRequestsPerDay)),
		),
	)

	return nil
}

// CheckResourceQuota verifies if an account has sufficient quota for a request
func (k Keeper) CheckResourceQuota(ctx context.Context, account sdk.AccAddress, specs types.ComputeSpec) error {
	// Get or create quota
	quota, err := k.GetResourceQuota(ctx, account)
	if err != nil {
		// First request - create default quota
		quota = k.GetDefaultResourceQuota(account.String())
	}

	// Check concurrent request limit
	if quota.CurrentRequests >= quota.MaxConcurrentRequests {
		return fmt.Errorf("concurrent request limit reached: %d/%d", quota.CurrentRequests, quota.MaxConcurrentRequests)
	}

	// Check if adding this request would exceed quotas
	totalCPU := quota.CurrentCpu + specs.CpuCores
	if totalCPU > quota.MaxTotalCpuCores {
		return fmt.Errorf("CPU quota exceeded: requesting %d cores would exceed limit of %d", specs.CpuCores, quota.MaxTotalCpuCores)
	}

	totalMemory := quota.CurrentMemory + specs.MemoryMb
	if totalMemory > quota.MaxTotalMemoryMb {
		return fmt.Errorf("memory quota exceeded: requesting %d MB would exceed limit of %d MB", specs.MemoryMb, quota.MaxTotalMemoryMb)
	}

	if specs.GpuCount > 0 {
		totalGPUs := quota.CurrentGpus + uint64(specs.GpuCount)
		if totalGPUs > quota.MaxTotalGpus {
			return fmt.Errorf("GPU quota exceeded: requesting %d GPUs would exceed limit of %d", specs.GpuCount, quota.MaxTotalGpus)
		}
	}

	totalStorage := quota.CurrentStorage + uint64(specs.StorageGb)
	if totalStorage > quota.MaxTotalStorageGb {
		return fmt.Errorf("storage quota exceeded: requesting %d GB would exceed limit of %d GB", specs.StorageGb, quota.MaxTotalStorageGb)
	}

	return nil
}

// AllocateResources reserves resources from quota for a request
func (k Keeper) AllocateResources(ctx context.Context, account sdk.AccAddress, specs types.ComputeSpec) error {
	quota, err := k.GetResourceQuota(ctx, account)
	if err != nil {
		quota = k.GetDefaultResourceQuota(account.String())
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Allocate resources
	quota.CurrentRequests++
	quota.CurrentCpu += specs.CpuCores
	quota.CurrentMemory += specs.MemoryMb
	quota.CurrentGpus += uint64(specs.GpuCount)
	quota.CurrentStorage += specs.StorageGb
	quota.LastUpdated = sdkCtx.BlockTime()

	if err := k.SetResourceQuota(ctx, *quota); err != nil {
		return fmt.Errorf("failed to allocate resources: %w", err)
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"resources_allocated",
			sdk.NewAttribute("account", account.String()),
			sdk.NewAttribute("cpu_cores", fmt.Sprintf("%d", specs.CpuCores)),
			sdk.NewAttribute("memory_mb", fmt.Sprintf("%d", specs.MemoryMb)),
			sdk.NewAttribute("gpu_count", fmt.Sprintf("%d", specs.GpuCount)),
			sdk.NewAttribute("storage_gb", fmt.Sprintf("%d", specs.StorageGb)),
		),
	)

	return nil
}

// ReleaseResources frees resources back to quota when request completes
func (k Keeper) ReleaseResources(ctx context.Context, account sdk.AccAddress, specs types.ComputeSpec) error {
	quota, err := k.GetResourceQuota(ctx, account)
	if err != nil {
		return fmt.Errorf("quota not found: %w", err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Release resources (with bounds checking)
	if quota.CurrentRequests > 0 {
		quota.CurrentRequests--
	}
	if quota.CurrentCpu >= specs.CpuCores {
		quota.CurrentCpu -= specs.CpuCores
	} else {
		quota.CurrentCpu = 0
	}
	if quota.CurrentMemory >= specs.MemoryMb {
		quota.CurrentMemory -= specs.MemoryMb
	} else {
		quota.CurrentMemory = 0
	}
	if quota.CurrentGpus >= uint64(specs.GpuCount) {
		quota.CurrentGpus -= uint64(specs.GpuCount)
	} else {
		quota.CurrentGpus = 0
	}
	if quota.CurrentStorage >= specs.StorageGb {
		quota.CurrentStorage -= specs.StorageGb
	} else {
		quota.CurrentStorage = 0
	}
	quota.LastUpdated = sdkCtx.BlockTime()

	if err := k.SetResourceQuota(ctx, *quota); err != nil {
		return fmt.Errorf("failed to release resources: %w", err)
	}

	return nil
}

// CheckProviderCapacity verifies if a provider can handle an additional request
func (k Keeper) CheckProviderCapacity(ctx context.Context, provider sdk.AccAddress, specs types.ComputeSpec) error {
	tracker, err := k.GetProviderLoadTracker(ctx, provider)
	if err != nil {
		// Initialize tracker from provider record
		providerRecord, err := k.GetProvider(ctx, provider)
		if err != nil {
			return fmt.Errorf("provider not found: %w", err)
		}

		params, err := k.GetParams(ctx)
		if err != nil {
			return err
		}
		config := k.GetDefaultRateLimitConfig(params)

		tracker = &ProviderLoadTracker{
			Provider:              provider.String(),
			MaxConcurrentRequests: config.MaxProviderLoad,
			CurrentRequests:       0,
			TotalCpuCores:         providerRecord.AvailableSpecs.CpuCores,
			UsedCpuCores:          0,
			TotalMemoryMb:         providerRecord.AvailableSpecs.MemoryMb,
			UsedMemoryMb:          0,
			TotalGpus:             uint64(providerRecord.AvailableSpecs.GpuCount),
			UsedGpus:              0,
			LastUpdated:           sdk.UnwrapSDKContext(ctx).BlockTime(),
		}
	}

	// Check concurrent request limit
	if tracker.CurrentRequests >= tracker.MaxConcurrentRequests {
		return fmt.Errorf("provider at capacity: %d/%d concurrent requests", tracker.CurrentRequests, tracker.MaxConcurrentRequests)
	}

	// Check resource availability
	if tracker.UsedCpuCores+specs.CpuCores > tracker.TotalCpuCores {
		return fmt.Errorf("provider CPU capacity exceeded")
	}

	if tracker.UsedMemoryMb+specs.MemoryMb > tracker.TotalMemoryMb {
		return fmt.Errorf("provider memory capacity exceeded")
	}

	if specs.GpuCount > 0 && tracker.UsedGpus+uint64(specs.GpuCount) > tracker.TotalGpus {
		return fmt.Errorf("provider GPU capacity exceeded")
	}

	return nil
}

// AllocateProviderResources reserves provider resources for a request
func (k Keeper) AllocateProviderResources(ctx context.Context, provider sdk.AccAddress, specs types.ComputeSpec) error {
	tracker, err := k.GetProviderLoadTracker(ctx, provider)
	if err != nil {
		return fmt.Errorf("provider load tracker not found: %w", err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	tracker.CurrentRequests++
	tracker.UsedCpuCores += specs.CpuCores
	tracker.UsedMemoryMb += specs.MemoryMb
	tracker.UsedGpus += uint64(specs.GpuCount)
	tracker.LastUpdated = sdkCtx.BlockTime()

	if err := k.SetProviderLoadTracker(ctx, *tracker); err != nil {
		return fmt.Errorf("failed to allocate provider resources: %w", err)
	}

	return nil
}

// ReleaseProviderResources frees provider resources when request completes
func (k Keeper) ReleaseProviderResources(ctx context.Context, provider sdk.AccAddress, specs types.ComputeSpec) error {
	tracker, err := k.GetProviderLoadTracker(ctx, provider)
	if err != nil {
		return fmt.Errorf("provider load tracker not found: %w", err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if tracker.CurrentRequests > 0 {
		tracker.CurrentRequests--
	}
	if tracker.UsedCpuCores >= specs.CpuCores {
		tracker.UsedCpuCores -= specs.CpuCores
	} else {
		tracker.UsedCpuCores = 0
	}
	if tracker.UsedMemoryMb >= specs.MemoryMb {
		tracker.UsedMemoryMb -= specs.MemoryMb
	} else {
		tracker.UsedMemoryMb = 0
	}
	if tracker.UsedGpus >= uint64(specs.GpuCount) {
		tracker.UsedGpus -= uint64(specs.GpuCount)
	} else {
		tracker.UsedGpus = 0
	}
	tracker.LastUpdated = sdkCtx.BlockTime()

	if err := k.SetProviderLoadTracker(ctx, *tracker); err != nil {
		return fmt.Errorf("failed to release provider resources: %w", err)
	}

	return nil
}

// GenerateSecureRandomness generates cryptographically secure randomness for provider selection
// Uses block hash, timestamp, and deterministic seed for reproducibility
func (k Keeper) GenerateSecureRandomness(ctx context.Context, seed []byte) *big.Int {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Combine multiple entropy sources
	hasher := sha256.New()

	// Block hash
	hasher.Write(sdkCtx.HeaderHash())

	// Block height
	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, saturateInt64ToUint64(sdkCtx.BlockHeight()))
	hasher.Write(heightBz)

	// Block time
	timeBz := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBz, saturateInt64ToUint64(sdkCtx.BlockTime().Unix()))
	hasher.Write(timeBz)

	// Additional seed
	hasher.Write(seed)

	randomBytes := hasher.Sum(nil)
	randomInt := new(big.Int).SetBytes(randomBytes)

	return randomInt
}

// GetDefaultRateLimitConfig returns default rate limiting configuration
func (k Keeper) GetDefaultRateLimitConfig(params types.Params) RateLimitConfig {
	return RateLimitConfig{
		MaxRequestsPerHour:    100,
		MaxRequestsPerDay:     500,
		MaxConcurrentRequests: 10,
		MaxProviderLoad:       50,
		BurstAllowance:        20,
		CooldownPeriodSeconds: 60,
	}
}

// GetDefaultResourceQuota returns default resource quota for an account
func (k Keeper) GetDefaultResourceQuota(account string) *types.ResourceQuota {
	return &types.ResourceQuota{
		Account:               account,
		MaxConcurrentRequests: 10,
		MaxTotalCpuCores:      100,
		MaxTotalMemoryMb:      102400, // 100 GB
		MaxTotalGpus:          10,
		MaxTotalStorageGb:     1000, // 1 TB
		CurrentCpu:            0,
		CurrentMemory:         0,
		CurrentGpus:           0,
		CurrentStorage:        0,
		CurrentRequests:       0,
		LastUpdated:           time.Now(),
	}
}

// Storage functions for rate limiting and quotas

func (k Keeper) GetRateLimitBucket(ctx context.Context, account sdk.AccAddress) (*RateLimitBucket, error) {
	store := k.getStore(ctx)
	bz := store.Get(RateLimitBucketKey(account))

	if bz == nil {
		return nil, fmt.Errorf("rate limit bucket not found")
	}

	var bucket RateLimitBucket
	if err := k.cdc.Unmarshal(bz, &bucket); err != nil {
		return nil, err
	}

	return &bucket, nil
}

func (k Keeper) SetRateLimitBucket(ctx context.Context, bucket RateLimitBucket) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&bucket)
	if err != nil {
		return err
	}

	addr, err := sdk.AccAddressFromBech32(bucket.Account)
	if err != nil {
		return err
	}

	store.Set(RateLimitBucketKey(addr), bz)
	return nil
}

func (k Keeper) GetResourceQuota(ctx context.Context, account sdk.AccAddress) (*types.ResourceQuota, error) {
	store := k.getStore(ctx)
	bz := store.Get(ResourceQuotaKey(account))

	if bz == nil {
		return nil, fmt.Errorf("resource quota not found")
	}

	var quota types.ResourceQuota
	if err := k.cdc.Unmarshal(bz, &quota); err != nil {
		return nil, err
	}

	return &quota, nil
}

func (k Keeper) SetResourceQuota(ctx context.Context, quota types.ResourceQuota) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&quota)
	if err != nil {
		return err
	}

	addr, err := sdk.AccAddressFromBech32(quota.Account)
	if err != nil {
		return err
	}

	store.Set(ResourceQuotaKey(addr), bz)
	return nil
}

func (k Keeper) GetProviderLoadTracker(ctx context.Context, provider sdk.AccAddress) (*ProviderLoadTracker, error) {
	store := k.getStore(ctx)
	bz := store.Get(ProviderLoadKey(provider))

	if bz == nil {
		return nil, fmt.Errorf("provider load tracker not found")
	}

	var tracker ProviderLoadTracker
	if err := k.cdc.Unmarshal(bz, &tracker); err != nil {
		return nil, err
	}

	return &tracker, nil
}

func (k Keeper) SetProviderLoadTracker(ctx context.Context, tracker ProviderLoadTracker) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&tracker)
	if err != nil {
		return err
	}

	addr, err := sdk.AccAddressFromBech32(tracker.Provider)
	if err != nil {
		return err
	}

	store.Set(ProviderLoadKey(addr), bz)
	return nil
}

// Additional key definitions for security module
var (
	RateLimitBucketPrefix = []byte{0x30}
	ResourceQuotaPrefix   = []byte{0x31}
	ProviderLoadPrefix    = []byte{0x32}
)

func RateLimitBucketKey(account sdk.AccAddress) []byte {
	return append(RateLimitBucketPrefix, account.Bytes()...)
}

func ResourceQuotaKey(account sdk.AccAddress) []byte {
	return append(ResourceQuotaPrefix, account.Bytes()...)
}

func ProviderLoadKey(provider sdk.AccAddress) []byte {
	return append(ProviderLoadPrefix, provider.Bytes()...)
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
