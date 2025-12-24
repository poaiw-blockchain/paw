package keeper

import (
	"context"
	"encoding/binary"
	"fmt"
	stdmath "math"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// SubmitRequest creates a new compute request and escrows payment
func (k Keeper) SubmitRequest(ctx context.Context, requester sdk.AccAddress, specs types.ComputeSpec, containerImage string, command []string, envVars map[string]string, maxPayment math.Int, preferredProvider string) (uint64, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Consume gas for request validation
	// GAS_REQUEST_VALIDATION = 2000 gas
	// Calibration: Input validation overhead covering:
	// - ComputeSpec validation (CPU/memory/storage bounds checking) (~800 gas)
	// - Container image string validation (~400 gas)
	// - Command array iteration and validation (~400 gas)
	// - Payment amount validation (~400 gas)
	// Total accounts for multiple validation functions and type assertions
	sdkCtx.GasMeter().ConsumeGas(2000, "compute_request_validation")

	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get params: %w", err)
	}

	// Validate specs
	specs, err = k.validateComputeSpec(specs, params, false)
	if err != nil {
		return 0, fmt.Errorf("invalid compute specs: %w", err)
	}

	// Validate container image
	if containerImage == "" {
		return 0, fmt.Errorf("container image is required")
	}

	// Validate max payment
	if maxPayment.IsZero() || maxPayment.IsNegative() {
		return 0, fmt.Errorf("max payment must be greater than zero")
	}

	// Consume gas for provider search - proportional to available providers
	// GAS_PROVIDER_SEARCH = 3000 gas
	// Calibration: Provider matching algorithm overhead:
	// - Iterate through registered providers (~500 gas per provider, assuming 3-5 providers)
	// - Check resource availability (CPU/memory/storage) for each (~400 gas per provider)
	// - Reputation score lookup and comparison (~300 gas per provider)
	// - Preferred provider lookup if specified (~500 gas)
	// - Selection logic and fallback handling (~500 gas)
	// Total calibrated for typical network with 3-5 compute providers
	sdkCtx.GasMeter().ConsumeGas(3000, "compute_provider_search")

	// Find a suitable provider
	provider, err := k.FindSuitableProvider(ctx, specs, preferredProvider)
	if err != nil {
		return 0, fmt.Errorf("failed to find suitable provider: %w", err)
	}

	// Consume gas for cost estimation
	// GAS_COST_ESTIMATION = 1500 gas
	// Calibration: Cost calculation overhead:
	// - Provider pricing retrieval from state (~500 gas)
	// - Resource-based cost calculation (CPU × rate + memory × rate + storage × rate) (~600 gas)
	// - Duration/timeout multiplier application (~200 gas)
	// - Additional fees calculation (network, priority) (~200 gas)
	// Total covers arithmetic operations on multiple pricing parameters
	sdkCtx.GasMeter().ConsumeGas(1500, "compute_cost_estimation")

	// Estimate cost
	estimatedCost, _, err := k.EstimateCost(ctx, provider, specs)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate cost: %w", err)
	}

	// Check if max payment is sufficient
	if maxPayment.LT(estimatedCost) {
		return 0, fmt.Errorf("max payment %s is less than estimated cost %s", maxPayment.String(), estimatedCost.String())
	}

	// Consume gas for escrow operation
	// GAS_PAYMENT_ESCROW = 2000 gas
	// Calibration: Bank module escrow operation overhead:
	// - Requester balance lookup (~400 gas)
	// - Module account lookup (~400 gas)
	// - Balance checks and validation (~300 gas)
	// - SendCoinsFromAccountToModule internal call (~700 gas for state changes)
	// - Escrow amount tracking update (~200 gas)
	// Lower than swap transfers due to single-direction flow
	sdkCtx.GasMeter().ConsumeGas(2000, "compute_payment_escrow")

	// Escrow payment from requester
	coins := sdk.NewCoins(sdk.NewCoin("upaw", maxPayment))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(sdkCtx, requester, types.ModuleName, coins); err != nil {
		return 0, fmt.Errorf("failed to escrow payment: %w", err)
	}

	// Consume gas for state write
	// GAS_REQUEST_STORAGE = 1000 gas
	// Calibration: Request record storage overhead:
	// - Request ID generation and increment (~200 gas)
	// - Request struct marshaling (smaller than Pool due to fewer fields) (~400 gas)
	// - KVStore Set operation (~300 gas)
	// - Index creation (3 indexes: by requester, by provider, by status) (~100 gas)
	// Total lower than pool updates due to simpler data structure
	sdkCtx.GasMeter().ConsumeGas(1000, "compute_request_storage")

	// Get next request ID
	requestID, err := k.getNextRequestID(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get next request ID: %w", err)
	}

	// Create request record
	now := sdkCtx.BlockTime()
	request := types.Request{
		Id:             requestID,
		Requester:      requester.String(),
		Provider:       provider.String(),
		Specs:          specs,
		ContainerImage: containerImage,
		Command:        command,
		EnvVars:        envVars,
		Status:         types.REQUEST_STATUS_ASSIGNED,
		MaxPayment:     maxPayment,
		EscrowedAmount: maxPayment,
		CreatedAt:      now,
		AssignedAt:     &now,
	}

	// Store request
	if err := k.SetRequest(ctx, request); err != nil {
		return 0, fmt.Errorf("failed to store request: %w", err)
	}

	// Create indexes
	if err := k.setRequestIndexes(ctx, request); err != nil {
		return 0, fmt.Errorf("failed to create request indexes: %w", err)
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"request_submitted",
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
			sdk.NewAttribute("requester", requester.String()),
			sdk.NewAttribute("provider", provider.String()),
			sdk.NewAttribute("max_payment", maxPayment.String()),
		),
	)

	return requestID, nil
}

// CancelRequest cancels a pending request and refunds the payment
func (k Keeper) CancelRequest(ctx context.Context, requester sdk.AccAddress, requestID uint64) error {
	// Get request
	request, err := k.GetRequest(ctx, requestID)
	if err != nil {
		return fmt.Errorf("request not found: %w", err)
	}

	// Verify requester
	if request.Requester != requester.String() {
		return fmt.Errorf("unauthorized: only requester can cancel request")
	}

	// Check if request can be cancelled
	if request.Status != types.REQUEST_STATUS_PENDING &&
		request.Status != types.REQUEST_STATUS_ASSIGNED {
		return fmt.Errorf("request cannot be cancelled in status %s", request.Status.String())
	}

	// Update status
	request.Status = types.REQUEST_STATUS_CANCELLED

	// Store updated request
	if err := k.SetRequest(ctx, *request); err != nil {
		return fmt.Errorf("failed to update request: %w", err)
	}

	// Update indexes
	if err := k.updateRequestStatusIndex(ctx, *request); err != nil {
		return fmt.Errorf("failed to update request indexes: %w", err)
	}

	// Refund escrowed payment
	if !request.EscrowedAmount.IsZero() {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		coins := sdk.NewCoins(sdk.NewCoin("upaw", request.EscrowedAmount))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, requester, coins); err != nil {
			return fmt.Errorf("failed to refund payment: %w", err)
		}
	}

	// Emit event
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"request_cancelled",
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
			sdk.NewAttribute("requester", requester.String()),
			sdk.NewAttribute("refund_amount", request.EscrowedAmount.String()),
		),
	)

	return nil
}

// CompleteRequest marks a request as completed and processes payment
func (k Keeper) CompleteRequest(ctx context.Context, requestID uint64, success bool) error {
	// Get request
	request, err := k.GetRequest(ctx, requestID)
	if err != nil {
		return fmt.Errorf("request not found: %w", err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()
	if k.isRequestFinalized(ctx, requestID) {
		return fmt.Errorf("request %d already settled", requestID)
	}

	if request.Status != types.REQUEST_STATUS_PROCESSING &&
		request.Status != types.REQUEST_STATUS_ASSIGNED {
		return fmt.Errorf("request cannot be completed from status %s", request.Status.String())
	}

	if success {
		if request.Status != types.REQUEST_STATUS_PROCESSING {
			return fmt.Errorf("request %d not actively processing", requestID)
		}

		// Mark as completed
		request.Status = types.REQUEST_STATUS_COMPLETED
		request.CompletedAt = &now

		// Release payment to provider
		provider, err := sdk.AccAddressFromBech32(request.Provider)
		if err != nil {
			return fmt.Errorf("invalid provider address: %w", err)
		}
		if request.EscrowedAmount.IsZero() {
			return fmt.Errorf("request %d escrow already released", requestID)
		}

		if !request.EscrowedAmount.IsZero() {
			coins := sdk.NewCoins(sdk.NewCoin("upaw", request.EscrowedAmount))
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, provider, coins); err != nil {
				return fmt.Errorf("failed to release payment: %w", err)
			}
			request.EscrowedAmount = math.ZeroInt()
		}

		// Update provider reputation (positive)
		if err := k.UpdateProviderReputation(ctx, provider, true); err != nil {
			return fmt.Errorf("failed to update provider reputation: %w", err)
		}

		// Emit event
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"request_completed",
				sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
				sdk.NewAttribute("provider", request.Provider),
				sdk.NewAttribute("payment_released", request.EscrowedAmount.String()),
			),
		)
	} else {
		// Mark as failed
		request.Status = types.REQUEST_STATUS_FAILED
		request.CompletedAt = &now

		// Refund payment to requester
		requester, err := sdk.AccAddressFromBech32(request.Requester)
		if err != nil {
			return fmt.Errorf("invalid requester address: %w", err)
		}

		if !request.EscrowedAmount.IsZero() {
			coins := sdk.NewCoins(sdk.NewCoin("upaw", request.EscrowedAmount))
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, requester, coins); err != nil {
				return fmt.Errorf("failed to refund payment: %w", err)
			}
			request.EscrowedAmount = math.ZeroInt()
		}

		// Update provider reputation (negative)
		provider, err := sdk.AccAddressFromBech32(request.Provider)
		if err != nil {
			return fmt.Errorf("invalid provider address: %w", err)
		}

		if err := k.UpdateProviderReputation(ctx, provider, false); err != nil {
			return fmt.Errorf("failed to update provider reputation: %w", err)
		}

		// Emit event
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"request_failed",
				sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
				sdk.NewAttribute("provider", request.Provider),
				sdk.NewAttribute("refund_amount", request.EscrowedAmount.String()),
			),
		)
	}

	// Store updated request
	if err := k.SetRequest(ctx, *request); err != nil {
		return fmt.Errorf("failed to update request: %w", err)
	}

	// Update indexes
	if err := k.updateRequestStatusIndex(ctx, *request); err != nil {
		return fmt.Errorf("failed to update request indexes: %w", err)
	}

	// Persist updated request state
	if err := k.SetRequest(ctx, *request); err != nil {
		return fmt.Errorf("failed to update request: %w", err)
	}

	if err := k.updateRequestStatusIndex(ctx, *request); err != nil {
		return fmt.Errorf("failed to update request indexes: %w", err)
	}

	k.markRequestFinalized(ctx, requestID)

	return nil
}

// GetRequest retrieves a request by ID
func (k Keeper) GetRequest(ctx context.Context, requestID uint64) (*types.Request, error) {
	store := k.getStore(ctx)
	bz := store.Get(RequestKey(requestID))

	if bz == nil {
		return nil, fmt.Errorf("request not found")
	}

	var request types.Request
	if err := k.cdc.Unmarshal(bz, &request); err != nil {
		return nil, err
	}

	return &request, nil
}

// SetRequest stores a request record
func (k Keeper) SetRequest(ctx context.Context, request types.Request) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&request)
	if err != nil {
		return err
	}

	store.Set(RequestKey(request.Id), bz)
	return nil
}

// MaxIterationLimit is the maximum number of items to return in unbounded queries
// This prevents DoS attacks via excessive iteration
const MaxIterationLimit = 100

// IterateRequests iterates over all requests with a maximum limit
func (k Keeper) IterateRequests(ctx context.Context, cb func(request types.Request) (stop bool, err error)) error {
	store := k.getStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, RequestKeyPrefix)
	defer iterator.Close()

	count := 0
	for ; iterator.Valid(); iterator.Next() {
		if count >= MaxIterationLimit {
			break // Stop at limit to prevent DoS
		}

		var request types.Request
		if err := k.cdc.Unmarshal(iterator.Value(), &request); err != nil {
			return err
		}

		stop, err := cb(request)
		if err != nil {
			return err
		}
		if stop {
			break
		}
		count++
	}

	return nil
}

// IterateRequestsByRequester iterates over requests by a specific requester with a maximum limit
func (k Keeper) IterateRequestsByRequester(ctx context.Context, requester sdk.AccAddress, cb func(request types.Request) (stop bool, err error)) error {
	store := k.getStore(ctx)
	prefix := append(RequestsByRequesterPrefix, requester.Bytes()...)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	// Batch-prefetch: collect all request IDs first
	var requestIDs []uint64
	count := 0
	for ; iterator.Valid(); iterator.Next() {
		if count >= MaxIterationLimit {
			break // Stop at limit to prevent DoS
		}

		// Extract request ID from key
		keyLen := len(prefix)
		requestID := GetRequestIDFromBytes(iterator.Key()[keyLen:])
		requestIDs = append(requestIDs, requestID)
		count++
	}

	// Fetch all requests and invoke callback
	for _, requestID := range requestIDs {
		request, err := k.GetRequest(ctx, requestID)
		if err != nil {
			continue
		}

		stop, err := cb(*request)
		if err != nil {
			return err
		}
		if stop {
			break
		}
	}

	return nil
}

// IterateRequestsByProvider iterates over requests assigned to a specific provider with a maximum limit
func (k Keeper) IterateRequestsByProvider(ctx context.Context, provider sdk.AccAddress, cb func(request types.Request) (stop bool, err error)) error {
	store := k.getStore(ctx)
	prefix := append(RequestsByProviderPrefix, provider.Bytes()...)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	// Batch-prefetch: collect all request IDs first
	var requestIDs []uint64
	count := 0
	for ; iterator.Valid(); iterator.Next() {
		if count >= MaxIterationLimit {
			break // Stop at limit to prevent DoS
		}

		// Extract request ID from key
		keyLen := len(prefix)
		requestID := GetRequestIDFromBytes(iterator.Key()[keyLen:])
		requestIDs = append(requestIDs, requestID)
		count++
	}

	// Fetch all requests and invoke callback
	for _, requestID := range requestIDs {
		request, err := k.GetRequest(ctx, requestID)
		if err != nil {
			continue
		}

		stop, err := cb(*request)
		if err != nil {
			return err
		}
		if stop {
			break
		}
	}

	return nil
}

// IterateRequestsByStatus iterates over requests with a specific status with a maximum limit
func (k Keeper) IterateRequestsByStatus(ctx context.Context, status types.RequestStatus, cb func(request types.Request) (stop bool, err error)) error {
	store := k.getStore(ctx)
	statusBz := make([]byte, 4)
	binary.BigEndian.PutUint32(statusBz, types.SaturateInt64ToUint32(int64(status)))
	prefix := append(RequestsByStatusPrefix, statusBz...)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	// Batch-prefetch: collect all request IDs first
	var requestIDs []uint64
	count := 0
	for ; iterator.Valid(); iterator.Next() {
		if count >= MaxIterationLimit {
			break // Stop at limit to prevent DoS
		}

		// Extract request ID from key
		keyLen := len(prefix)
		requestID := GetRequestIDFromBytes(iterator.Key()[keyLen:])
		requestIDs = append(requestIDs, requestID)
		count++
	}

	// Fetch all requests and invoke callback
	for _, requestID := range requestIDs {
		request, err := k.GetRequest(ctx, requestID)
		if err != nil {
			continue
		}

		stop, err := cb(*request)
		if err != nil {
			return err
		}
		if stop {
			break
		}
	}

	return nil
}

// getNextRequestID gets and increments the next request ID
func (k Keeper) getNextRequestID(ctx context.Context) (uint64, error) {
	store := k.getStore(ctx)
	bz := store.Get(NextRequestIDKey)

	var nextID uint64 = 1
	if bz != nil {
		nextID = binary.BigEndian.Uint64(bz)
	}

	// Increment and store
	nextBz := make([]byte, 8)
	binary.BigEndian.PutUint64(nextBz, nextID+1)
	store.Set(NextRequestIDKey, nextBz)

	return nextID, nil
}

// setRequestIndexes creates all necessary indexes for a request
func (k Keeper) setRequestIndexes(ctx context.Context, request types.Request) error {
	store := k.getStore(ctx)

	// Index by requester
	requester, err := sdk.AccAddressFromBech32(request.Requester)
	if err != nil {
		return err
	}
	store.Set(RequestByRequesterKey(requester, request.Id), []byte{})

	// Index by provider (if assigned)
	if request.Provider != "" {
		provider, err := sdk.AccAddressFromBech32(request.Provider)
		if err != nil {
			return err
		}
		store.Set(RequestByProviderKey(provider, request.Id), []byte{})
	}

	// Index by status
	store.Set(RequestByStatusKey(types.SaturateInt64ToUint32(int64(request.Status)), request.Id), []byte{})

	return nil
}

// updateRequestStatusIndex updates the status index when a request status changes
func (k Keeper) updateRequestStatusIndex(ctx context.Context, request types.Request) error {
	store := k.getStore(ctx)

	// Remove old status indexes (try all statuses)
	for status := types.REQUEST_STATUS_PENDING; status <= types.REQUEST_STATUS_CANCELLED; status++ {
		key := RequestByStatusKey(types.SaturateInt64ToUint32(int64(status)), request.Id)
		store.Delete(key) // Ignore errors as key might not exist
	}

	// Add new status index
	store.Set(RequestByStatusKey(types.SaturateInt64ToUint32(int64(request.Status)), request.Id), []byte{})
	return nil
}

func (k Keeper) isRequestFinalized(ctx context.Context, requestID uint64) bool {
	store := k.getStore(ctx)
	return store.Has(RequestFinalizedKey(requestID))
}

func (k Keeper) markRequestFinalized(ctx context.Context, requestID uint64) {
	store := k.getStore(ctx)
	store.Set(RequestFinalizedKey(requestID), []byte{1})
}

// requestDeadline computes the absolute deadline when a request expires.
func (k Keeper) requestDeadline(ctx context.Context, request types.Request, now time.Time) (time.Time, error) {
	timeoutSeconds := request.Specs.TimeoutSeconds
	if timeoutSeconds == 0 {
		params, err := k.GetParams(ctx)
		if err != nil {
			return time.Time{}, err
		}
		timeoutSeconds = params.MaxRequestTimeoutSeconds
		if timeoutSeconds == 0 {
			timeoutSeconds = 3600
		}
	}

	baseTime := request.CreatedAt
	if request.AssignedAt != nil {
		baseTime = *request.AssignedAt
	}

	maxSeconds := uint64(stdmath.MaxInt64 / int64(time.Second))
	if timeoutSeconds > maxSeconds {
		timeoutSeconds = maxSeconds
	}

	return baseTime.Add(types.SecondsToDuration(timeoutSeconds)), nil
}
