package keeper

import (
	"encoding/binary"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

var (
	// ProviderPrefix is the KVStore key prefix for compute providers
	ProviderPrefix = []byte{0x04}

	// ComputeRequestPrefix is the KVStore key prefix for compute requests
	ComputeRequestPrefix = []byte{0x05}

	// NextRequestIDKey is the KVStore key for the next request ID counter
	NextRequestIDKey = []byte{0x06}
)

// RegisterProvider registers a new compute provider with the required stake.
//
// This function validates the provider registration message, ensures the stake
// meets the minimum requirements, creates a provider record, and stores it in
// the module state.
//
// Parameters:
//   - ctx: SDK context for blockchain state access
//   - msg: Provider registration message containing provider address, endpoint, and stake
//
// Returns:
//   - *MsgRegisterProviderResponse: Empty response on success
//   - error: Validation error if message is invalid or stake is insufficient
//
// Events Emitted:
//   - provider_registered: Contains provider address, endpoint, and stake amount
//
// State Changes:
//   - Creates new Provider record in KVStore
func (k Keeper) RegisterProvider(ctx sdk.Context, msg *types.MsgRegisterProvider) (*types.MsgRegisterProviderResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Check minimum stake requirement
	params := k.GetParams(ctx)
	if msg.Stake.LT(params.MinStake) {
		return nil, fmt.Errorf("stake %s is less than minimum required %s", msg.Stake.String(), params.MinStake.String())
	}

	// Create provider record
	provider := types.Provider{
		Address:  msg.Provider,
		Endpoint: msg.Endpoint,
		Stake:    msg.Stake,
		Active:   true,
	}

	// Store provider
	k.SetProvider(ctx, provider)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"provider_registered",
			sdk.NewAttribute("provider", msg.Provider),
			sdk.NewAttribute("endpoint", msg.Endpoint),
			sdk.NewAttribute("stake", msg.Stake.String()),
		),
	)

	return &types.MsgRegisterProviderResponse{}, nil
}

// RequestCompute creates a new compute request for task execution.
//
// This function validates the compute request, generates a unique request ID,
// stores the request in pending state, and emits an event for provider discovery.
//
// Parameters:
//   - ctx: SDK context for blockchain state access
//   - msg: Compute request message containing requester, API URL, and maximum fee
//
// Returns:
//   - *MsgRequestComputeResponse: Response containing the generated request_id
//   - error: Validation error if message is invalid
//
// Events Emitted:
//   - compute_requested: Contains request_id, requester, api_url, and max_fee
//
// State Changes:
//   - Creates new ComputeRequest record in PENDING status
//   - Increments NextRequestID counter
func (k Keeper) RequestCompute(ctx sdk.Context, msg *types.MsgRequestCompute) (*types.MsgRequestComputeResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Get next request ID
	requestId := k.GetNextRequestID(ctx)

	// Create request
	request := types.ComputeRequest{
		Id:        requestId,
		Requester: msg.Requester,
		ApiUrl:    msg.ApiUrl,
		MaxFee:    msg.MaxFee,
		Status:    types.RequestStatus_PENDING,
	}

	// Store request
	k.SetRequest(ctx, request)

	// Increment next request ID
	k.SetNextRequestID(ctx, requestId+1)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"compute_requested",
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestId)),
			sdk.NewAttribute("requester", msg.Requester),
			sdk.NewAttribute("api_url", msg.ApiUrl),
			sdk.NewAttribute("max_fee", msg.MaxFee.String()),
		),
	)

	return &types.MsgRequestComputeResponse{RequestId: requestId}, nil
}

// SubmitResult submits the result of a completed compute task.
//
// This function verifies the provider is registered and active, validates the
// request exists and is still pending, stores the result, and updates the
// request status to COMPLETED.
//
// Parameters:
//   - ctx: SDK context for blockchain state access
//   - msg: Result submission message containing request_id, provider, and result data
//
// Returns:
//   - *MsgSubmitResultResponse: Empty response on success
//   - error: Validation error or if request not found, already completed, or provider invalid
//
// Events Emitted:
//   - result_submitted: Contains request_id and provider address
//
// State Changes:
//   - Updates ComputeRequest status to COMPLETED
//   - Stores result data and provider address
//
// Errors:
//   - Request not found
//   - Request not in PENDING status
//   - Provider not registered
//   - Provider not active
func (k Keeper) SubmitResult(ctx sdk.Context, msg *types.MsgSubmitResult) (*types.MsgSubmitResultResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Get request
	request, found := k.GetRequest(ctx, msg.RequestId)
	if !found {
		return nil, fmt.Errorf("request not found: %d", msg.RequestId)
	}

	// Check if request is still pending
	if request.Status != types.RequestStatus_PENDING {
		return nil, fmt.Errorf("request %d is not pending (status: %d)", msg.RequestId, int(request.Status))
	}

	// Verify provider is registered
	provider, found := k.GetProvider(ctx, msg.Provider)
	if !found {
		return nil, fmt.Errorf("provider not found: %s", msg.Provider)
	}

	// Verify provider is active
	if !provider.Active {
		return nil, fmt.Errorf("provider %s is not active", msg.Provider)
	}

	// Update request with result
	request.Result = msg.Result
	request.Provider = msg.Provider
	request.Status = types.RequestStatus_COMPLETED

	// Store updated request
	k.SetRequest(ctx, request)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"result_submitted",
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", msg.RequestId)),
			sdk.NewAttribute("provider", msg.Provider),
		),
	)

	return &types.MsgSubmitResultResponse{}, nil
}

// GetProvider retrieves a compute provider by address.
//
// Parameters:
//   - ctx: SDK context for blockchain state access
//   - address: Bech32 address of the provider
//
// Returns:
//   - Provider: The provider record if found
//   - bool: True if provider exists, false otherwise
func (k Keeper) GetProvider(ctx sdk.Context, address string) (types.Provider, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := append(ProviderPrefix, []byte(address)...)
	bz, err := store.Get(key)
	if err != nil {
		panic(err)
	}
	if bz == nil {
		return types.Provider{}, false
	}

	var provider types.Provider
	k.cdc.MustUnmarshal(bz, &provider)
	return provider, true
}

// SetProvider stores a provider in the KVStore.
//
// Parameters:
//   - ctx: SDK context for blockchain state access
//   - provider: Provider record to store
func (k Keeper) SetProvider(ctx sdk.Context, provider types.Provider) {
	store := k.storeService.OpenKVStore(ctx)
	key := append(ProviderPrefix, []byte(provider.Address)...)
	bz := k.cdc.MustMarshal(&provider)
	if err := store.Set(key, bz); err != nil {
		panic(err)
	}
}

// GetRequest retrieves a compute request by ID.
//
// Parameters:
//   - ctx: SDK context for blockchain state access
//   - id: Unique identifier of the compute request
//
// Returns:
//   - ComputeRequest: The request record if found
//   - bool: True if request exists, false otherwise
func (k Keeper) GetRequest(ctx sdk.Context, id uint64) (types.ComputeRequest, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := append(ComputeRequestPrefix, sdk.Uint64ToBigEndian(id)...)
	bz, err := store.Get(key)
	if err != nil {
		panic(err)
	}
	if bz == nil {
		return types.ComputeRequest{}, false
	}

	var request types.ComputeRequest
	k.cdc.MustUnmarshal(bz, &request)
	return request, true
}

// SetRequest stores a compute request in the KVStore.
//
// Parameters:
//   - ctx: SDK context for blockchain state access
//   - request: ComputeRequest record to store
func (k Keeper) SetRequest(ctx sdk.Context, request types.ComputeRequest) {
	store := k.storeService.OpenKVStore(ctx)
	key := append(ComputeRequestPrefix, sdk.Uint64ToBigEndian(request.Id)...)
	bz := k.cdc.MustMarshal(&request)
	if err := store.Set(key, bz); err != nil {
		panic(err)
	}
}

// GetNextRequestID returns the next request ID to be used.
//
// The request ID counter starts at 1 and is incremented for each new request.
//
// Parameters:
//   - ctx: SDK context for blockchain state access
//
// Returns:
//   - uint64: The next available request ID (default: 1)
func (k Keeper) GetNextRequestID(ctx sdk.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(NextRequestIDKey)
	if err != nil {
		panic(err)
	}
	if bz == nil {
		return 1
	}
	return binary.BigEndian.Uint64(bz)
}

// SetNextRequestID sets the next request ID counter.
//
// Parameters:
//   - ctx: SDK context for blockchain state access
//   - id: The next request ID value to store
func (k Keeper) SetNextRequestID(ctx sdk.Context, id uint64) {
	store := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	if err := store.Set(NextRequestIDKey, bz); err != nil {
		panic(err)
	}
}

// RegisterTestProvider is a helper function for tests to easily register providers.
//
// This function simplifies provider registration in test scenarios by creating
// and submitting a MsgRegisterProvider message.
//
// Parameters:
//   - k: Keeper instance
//   - ctx: SDK context
//   - address: Provider address
//   - endpoint: Provider API endpoint
//   - stake: Stake amount
//
// Returns:
//   - error: Registration error if any
func RegisterTestProvider(k *Keeper, ctx sdk.Context, address, endpoint string, stake math.Int) error {
	msg := &types.MsgRegisterProvider{
		Provider: address,
		Endpoint: endpoint,
		Stake:    stake,
	}
	_, err := k.RegisterProvider(ctx, msg)
	return err
}

// SubmitTestRequest is a helper function for tests to easily submit compute requests.
//
// This function simplifies compute request submission in test scenarios by creating
// and submitting a MsgRequestCompute message.
//
// Parameters:
//   - k: Keeper instance
//   - ctx: SDK context
//   - requester: Requester address
//   - apiUrl: API endpoint URL
//   - maxFee: Maximum fee willing to pay
//
// Returns:
//   - uint64: Generated request ID
//   - error: Submission error if any
func SubmitTestRequest(k *Keeper, ctx sdk.Context, requester, apiUrl string, maxFee math.Int) (uint64, error) {
	msg := &types.MsgRequestCompute{
		Requester: requester,
		ApiUrl:    apiUrl,
		MaxFee:    maxFee,
	}
	resp, err := k.RequestCompute(ctx, msg)
	if err != nil {
		return 0, err
	}
	return resp.RequestId, nil
}
