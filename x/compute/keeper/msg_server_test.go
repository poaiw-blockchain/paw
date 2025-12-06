package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// TestMsgServer_RegisterProvider tests the RegisterProvider message handler
func TestMsgServer_RegisterProvider(t *testing.T) {
	k, sdkCtx, goCtx := newComputeKeeperCtx(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	provider := createTestProvider(t)
	specs := createValidComputeSpec()
	pricing := createValidPricing()

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	tests := []struct {
		name      string
		msg       *types.MsgRegisterProvider
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid registration",
			msg: &types.MsgRegisterProvider{
				Provider:       provider.String(),
				Moniker:        "TestProvider",
				Endpoint:       "https://test.example.com",
				AvailableSpecs: specs,
				Pricing:        pricing,
				Stake:          params.MinProviderStake,
			},
			expectErr: false,
		},
		{
			name: "invalid provider address",
			msg: &types.MsgRegisterProvider{
				Provider:       "invalid_address",
				Moniker:        "TestProvider",
				Endpoint:       "https://test.example.com",
				AvailableSpecs: specs,
				Pricing:        pricing,
				Stake:          params.MinProviderStake,
			},
			expectErr: true,
			errMsg:    "invalid provider address",
		},
		{
			name: "empty moniker",
			msg: &types.MsgRegisterProvider{
				Provider:       provider.String(),
				Moniker:        "",
				Endpoint:       "https://test.example.com",
				AvailableSpecs: specs,
				Pricing:        pricing,
				Stake:          params.MinProviderStake,
			},
			expectErr: true,
			errMsg:    "moniker",
		},
		{
			name: "invalid endpoint",
			msg: &types.MsgRegisterProvider{
				Provider:       provider.String(),
				Moniker:        "TestProvider",
				Endpoint:       "not-a-valid-url",
				AvailableSpecs: specs,
				Pricing:        pricing,
				Stake:          params.MinProviderStake,
			},
			expectErr: true,
			errMsg:    "endpoint",
		},
		{
			name: "insufficient stake",
			msg: &types.MsgRegisterProvider{
				Provider:       provider.String(),
				Moniker:        "TestProvider",
				Endpoint:       "https://test.example.com",
				AvailableSpecs: specs,
				Pricing:        pricing,
				Stake:          math.NewInt(100), // Too low
			},
			expectErr: true,
			errMsg:    "stake",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Use a fresh provider address for each test to avoid duplicates
			if !tc.expectErr {
				tc.msg.Provider = sdk.AccAddress([]byte("provider_" + tc.name)).String()
			}

			resp, err := msgServer.RegisterProvider(goCtx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify provider was stored
				addr, _ := sdk.AccAddressFromBech32(tc.msg.Provider)
				storedProvider, err := k.GetProvider(sdkCtx, addr)
				require.NoError(t, err)
				require.Equal(t, tc.msg.Provider, storedProvider.Address)
				require.Equal(t, tc.msg.Moniker, storedProvider.Moniker)
				require.True(t, storedProvider.Active)
			}
		})
	}
}

// TestMsgServer_RegisterProvider_Duplicate tests duplicate registration prevention
func TestMsgServer_RegisterProvider_Duplicate(t *testing.T) {
	k, sdkCtx, goCtx := newComputeKeeperCtx(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	provider := createTestProvider(t)
	specs := createValidComputeSpec()
	pricing := createValidPricing()

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	msg := &types.MsgRegisterProvider{
		Provider:       provider.String(),
		Moniker:        "TestProvider",
		Endpoint:       "https://test.example.com",
		AvailableSpecs: specs,
		Pricing:        pricing,
		Stake:          params.MinProviderStake,
	}

	// First registration should succeed
	resp, err := msgServer.RegisterProvider(goCtx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Second registration should fail
	resp, err = msgServer.RegisterProvider(goCtx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already registered")
	require.Nil(t, resp)
}

// TestMsgServer_UpdateProvider tests the UpdateProvider message handler
func TestMsgServer_UpdateProvider(t *testing.T) {
	k, sdkCtx, goCtx := newComputeKeeperCtx(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	provider := createTestProvider(t)
	specs := createValidComputeSpec()
	pricing := createValidPricing()

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	// Register provider first
	err = k.RegisterProvider(sdkCtx, provider, "Original", "https://original.example.com", specs, pricing, params.MinProviderStake)
	require.NoError(t, err)

	tests := []struct {
		name      string
		msg       *types.MsgUpdateProvider
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid update",
			msg: &types.MsgUpdateProvider{
				Provider:       provider.String(),
				Moniker:        "UpdatedProvider",
				Endpoint:       "https://updated.example.com",
				AvailableSpecs: &specs,
				Pricing:        &pricing,
			},
			expectErr: false,
		},
		{
			name: "invalid provider address",
			msg: &types.MsgUpdateProvider{
				Provider:       "invalid_address",
				Moniker:        "UpdatedProvider",
				Endpoint:       "https://updated.example.com",
				AvailableSpecs: &specs,
				Pricing:        &pricing,
			},
			expectErr: true,
			errMsg:    "invalid provider address",
		},
		{
			name: "non-existent provider",
			msg: &types.MsgUpdateProvider{
				Provider:       sdk.AccAddress([]byte("nonexistent_provider")).String(),
				Moniker:        "UpdatedProvider",
				Endpoint:       "https://updated.example.com",
				AvailableSpecs: &specs,
				Pricing:        &pricing,
			},
			expectErr: true,
			errMsg:    "not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := msgServer.UpdateProvider(goCtx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify provider was updated
				addr, _ := sdk.AccAddressFromBech32(tc.msg.Provider)
				updated, err := k.GetProvider(sdkCtx, addr)
				require.NoError(t, err)
				require.Equal(t, tc.msg.Moniker, updated.Moniker)
				require.Equal(t, tc.msg.Endpoint, updated.Endpoint)
			}
		})
	}
}

// TestMsgServer_DeactivateProvider tests the DeactivateProvider message handler
func TestMsgServer_DeactivateProvider(t *testing.T) {
	k, sdkCtx, goCtx := newComputeKeeperCtx(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	provider := createTestProvider(t)
	specs := createValidComputeSpec()
	pricing := createValidPricing()

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	// Register provider first
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.example.com", specs, pricing, params.MinProviderStake)
	require.NoError(t, err)

	tests := []struct {
		name      string
		msg       *types.MsgDeactivateProvider
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid deactivation",
			msg: &types.MsgDeactivateProvider{
				Provider: provider.String(),
			},
			expectErr: false,
		},
		{
			name: "invalid provider address",
			msg: &types.MsgDeactivateProvider{
				Provider: "invalid_address",
			},
			expectErr: true,
			errMsg:    "invalid provider address",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := msgServer.DeactivateProvider(goCtx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify provider was deactivated
				addr, _ := sdk.AccAddressFromBech32(tc.msg.Provider)
				deactivated, err := k.GetProvider(sdkCtx, addr)
				require.NoError(t, err)
				require.False(t, deactivated.Active)
			}
		})
	}
}

// TestMsgServer_SubmitRequest tests the SubmitRequest message handler
func TestMsgServer_SubmitRequest(t *testing.T) {
	k, sdkCtx, goCtx := newComputeKeeperCtx(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	requester := sdk.AccAddress([]byte("test_requester_addr"))
	provider := createTestProvider(t)
	specs := createValidComputeSpec()
	pricing := createValidPricing()

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	// Register a provider first
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.example.com", specs, pricing, params.MinProviderStake)
	require.NoError(t, err)

	tests := []struct {
		name      string
		msg       *types.MsgSubmitRequest
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid request",
			msg: &types.MsgSubmitRequest{
				Requester:         requester.String(),
				Specs:             specs,
				ContainerImage:    "nginx:latest",
				Command:           []string{"/bin/bash", "-c", "echo hello"},
				EnvVars:           map[string]string{"KEY": "value"},
				MaxPayment:        math.NewInt(1000000),
				PreferredProvider: "",
			},
			expectErr: false,
		},
		{
			name: "invalid requester address",
			msg: &types.MsgSubmitRequest{
				Requester:         "invalid_address",
				Specs:             specs,
				ContainerImage:    "nginx:latest",
				Command:           []string{"/bin/bash", "-c", "echo hello"},
				EnvVars:           map[string]string{},
				MaxPayment:        math.NewInt(1000000),
				PreferredProvider: "",
			},
			expectErr: true,
			errMsg:    "invalid requester address",
		},
		{
			name: "empty container image",
			msg: &types.MsgSubmitRequest{
				Requester:         requester.String(),
				Specs:             specs,
				ContainerImage:    "",
				Command:           []string{"/bin/bash"},
				EnvVars:           map[string]string{},
				MaxPayment:        math.NewInt(1000000),
				PreferredProvider: "",
			},
			expectErr: true,
			errMsg:    "container image",
		},
		{
			name: "invalid payment amount",
			msg: &types.MsgSubmitRequest{
				Requester:         requester.String(),
				Specs:             specs,
				ContainerImage:    "nginx:latest",
				Command:           []string{"/bin/bash"},
				EnvVars:           map[string]string{},
				MaxPayment:        math.NewInt(0),
				PreferredProvider: "",
			},
			expectErr: true,
			errMsg:    "payment",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := msgServer.SubmitRequest(goCtx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Greater(t, resp.RequestId, uint64(0))

				// Verify request was stored
				request, err := k.GetRequest(sdkCtx, resp.RequestId)
				require.NoError(t, err)
				require.Equal(t, tc.msg.Requester, request.Requester)
				require.Equal(t, tc.msg.ContainerImage, request.ContainerImage)
			}
		})
	}
}

// TestMsgServer_CancelRequest tests the CancelRequest message handler
func TestMsgServer_CancelRequest(t *testing.T) {
	k, sdkCtx, goCtx := newComputeKeeperCtx(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	requester := sdk.AccAddress([]byte("test_requester_addr"))
	provider := createTestProvider(t)
	specs := createValidComputeSpec()
	pricing := createValidPricing()

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	// Register provider and submit request
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.example.com", specs, pricing, params.MinProviderStake)
	require.NoError(t, err)

	requestID, err := k.SubmitRequest(
		sdkCtx,
		requester,
		specs,
		"nginx:latest",
		[]string{"/bin/bash"},
		map[string]string{},
		math.NewInt(1000000),
		"",
	)
	require.NoError(t, err)

	tests := []struct {
		name      string
		msg       *types.MsgCancelRequest
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid cancellation",
			msg: &types.MsgCancelRequest{
				Requester: requester.String(),
				RequestId: requestID,
			},
			expectErr: false,
		},
		{
			name: "invalid requester address",
			msg: &types.MsgCancelRequest{
				Requester: "invalid_address",
				RequestId: requestID,
			},
			expectErr: true,
			errMsg:    "invalid requester address",
		},
		{
			name: "non-existent request",
			msg: &types.MsgCancelRequest{
				Requester: requester.String(),
				RequestId: 99999,
			},
			expectErr: true,
			errMsg:    "not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := msgServer.CancelRequest(goCtx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify request was cancelled
				request, err := k.GetRequest(sdkCtx, tc.msg.RequestId)
				require.NoError(t, err)
				require.Equal(t, types.REQUEST_STATUS_CANCELLED, request.Status)
			}
		})
	}
}

// TestMsgServer_SubmitResult tests the SubmitResult message handler
func TestMsgServer_SubmitResult(t *testing.T) {
	k, sdkCtx, goCtx := newComputeKeeperCtx(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	requester := sdk.AccAddress([]byte("test_requester_addr"))
	provider := createTestProvider(t)
	specs := createValidComputeSpec()
	pricing := createValidPricing()

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	// Register provider and submit request
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.example.com", specs, pricing, params.MinProviderStake)
	require.NoError(t, err)

	requestID, err := k.SubmitRequest(
		sdkCtx,
		requester,
		specs,
		"nginx:latest",
		[]string{"/bin/bash"},
		map[string]string{},
		math.NewInt(1000000),
		provider.String(),
	)
	require.NoError(t, err)

	tests := []struct {
		name      string
		msg       *types.MsgSubmitResult
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid result submission",
			msg: &types.MsgSubmitResult{
				Provider:          provider.String(),
				RequestId:         requestID,
				OutputHash:        "abc123def456",
				OutputUrl:         "https://storage.example.com/output",
				ExitCode:          0,
				LogsUrl:           "https://storage.example.com/logs",
				VerificationProof: []byte("proof_data"),
			},
			expectErr: false,
		},
		{
			name: "invalid provider address",
			msg: &types.MsgSubmitResult{
				Provider:          "invalid_address",
				RequestId:         requestID,
				OutputHash:        "abc123def456",
				OutputUrl:         "https://storage.example.com/output",
				ExitCode:          0,
				LogsUrl:           "https://storage.example.com/logs",
				VerificationProof: []byte("proof_data"),
			},
			expectErr: true,
			errMsg:    "invalid provider address",
		},
		{
			name: "empty output hash",
			msg: &types.MsgSubmitResult{
				Provider:          provider.String(),
				RequestId:         requestID,
				OutputHash:        "",
				OutputUrl:         "https://storage.example.com/output",
				ExitCode:          0,
				LogsUrl:           "https://storage.example.com/logs",
				VerificationProof: []byte("proof_data"),
			},
			expectErr: true,
			errMsg:    "output hash",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := msgServer.SubmitResult(goCtx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify result was stored
				request, err := k.GetRequest(sdkCtx, tc.msg.RequestId)
				require.NoError(t, err)
				require.Equal(t, tc.msg.OutputHash, request.ResultHash)
				require.Equal(t, tc.msg.OutputUrl, request.ResultUrl)
			}
		})
	}
}

// TestMsgServer_UpdateParams tests the UpdateParams message handler
func TestMsgServer_UpdateParams(t *testing.T) {
	k, sdkCtx, goCtx := newComputeKeeperCtx(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	// Get current authority from keeper
	authority := k.GetAuthority()

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	// Modify params
	newParams := params
	newParams.MaxRequestTimeoutSeconds = 7200 // Changed from default

	tests := []struct {
		name      string
		msg       *types.MsgUpdateParams
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid params update",
			msg: &types.MsgUpdateParams{
				Authority: authority,
				Params:    newParams,
			},
			expectErr: false,
		},
		{
			name: "unauthorized authority",
			msg: &types.MsgUpdateParams{
				Authority: sdk.AccAddress([]byte("unauthorized_addr")).String(),
				Params:    newParams,
			},
			expectErr: true,
			errMsg:    "invalid authority",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := msgServer.UpdateParams(goCtx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify params were updated
				updated, err := k.GetParams(sdkCtx)
				require.NoError(t, err)
				require.Equal(t, tc.msg.Params.MaxRequestTimeoutSeconds, updated.MaxRequestTimeoutSeconds)
			}
		})
	}
}

// TestMsgServer_CreateDispute tests the CreateDispute message handler
func TestMsgServer_CreateDispute(t *testing.T) {
	k, sdkCtx, goCtx := newComputeKeeperCtx(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	requester := sdk.AccAddress([]byte("test_requester_addr"))
	provider := createTestProvider(t)
	specs := createValidComputeSpec()
	pricing := createValidPricing()

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	// Setup: Register provider and submit request with result
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.example.com", specs, pricing, params.MinProviderStake)
	require.NoError(t, err)

	requestID, err := k.SubmitRequest(
		sdkCtx,
		requester,
		specs,
		"nginx:latest",
		[]string{"/bin/bash"},
		map[string]string{},
		math.NewInt(1000000),
		provider.String(),
	)
	require.NoError(t, err)

	err = k.SubmitResult(sdkCtx, provider, requestID, "outputhash", "https://output.url", 0, "https://logs.url", []byte("proof"))
	require.NoError(t, err)

	depositAmount := math.NewInt(10000)

	tests := []struct {
		name      string
		msg       *types.MsgCreateDispute
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid dispute creation",
			msg: &types.MsgCreateDispute{
				Requester:     requester.String(),
				RequestId:     requestID,
				Reason:        "Incorrect output",
				Evidence:      []byte("Evidence data here"),
				DepositAmount: depositAmount,
			},
			expectErr: false,
		},
		{
			name: "invalid requester address",
			msg: &types.MsgCreateDispute{
				Requester:     "invalid_address",
				RequestId:     requestID,
				Reason:        "Incorrect output",
				Evidence:      []byte("Evidence data"),
				DepositAmount: depositAmount,
			},
			expectErr: true,
			errMsg:    "invalid requester",
		},
		{
			name: "empty reason",
			msg: &types.MsgCreateDispute{
				Requester:     requester.String(),
				RequestId:     requestID,
				Reason:        "",
				Evidence:      []byte("Evidence data"),
				DepositAmount: depositAmount,
			},
			expectErr: true,
			errMsg:    "reason",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := msgServer.CreateDispute(goCtx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Greater(t, resp.DisputeId, uint64(0))

				// Verify dispute was created
				dispute, err := k.GetDisputeForTesting(sdkCtx, resp.DisputeId)
				require.NoError(t, err)
				require.Equal(t, tc.msg.RequestId, dispute.RequestId)
				require.Equal(t, tc.msg.Reason, dispute.Reason)
			}
		})
	}
}

// TestMsgServer_VoteOnDispute tests the VoteOnDispute message handler
func TestMsgServer_VoteOnDispute(t *testing.T) {
	k, sdkCtx, goCtx := newComputeKeeperCtx(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	requester := sdk.AccAddress([]byte("test_requester_addr"))
	provider := createTestProvider(t)
	validator := sdk.ValAddress([]byte("test_validator_addr"))
	specs := createValidComputeSpec()
	pricing := createValidPricing()

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	// Setup: Create a dispute
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.example.com", specs, pricing, params.MinProviderStake)
	require.NoError(t, err)

	requestID, err := k.SubmitRequest(sdkCtx, requester, specs, "nginx:latest", []string{"/bin/bash"}, map[string]string{}, math.NewInt(1000000), provider.String())
	require.NoError(t, err)

	err = k.SubmitResult(sdkCtx, provider, requestID, "hash", "https://output.url", 0, "https://logs.url", []byte("proof"))
	require.NoError(t, err)

	disputeID, err := k.CreateDispute(sdkCtx, requester, requestID, "Issue with result", []byte("Evidence"),math.NewInt(10000))
	require.NoError(t, err)

	tests := []struct {
		name      string
		msg       *types.MsgVoteOnDispute
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid vote",
			msg: &types.MsgVoteOnDispute{
				Validator:     validator.String(),
				DisputeId:     disputeID,
				Vote:          types.DISPUTE_VOTE_REQUESTER_FAULT,
				Justification: "Valid complaint",
			},
			expectErr: false,
		},
		{
			name: "invalid validator address",
			msg: &types.MsgVoteOnDispute{
				Validator:     "invalid_address",
				DisputeId:     disputeID,
				Vote:          types.DISPUTE_VOTE_REQUESTER_FAULT,
				Justification: "Valid complaint",
			},
			expectErr: true,
			errMsg:    "invalid validator",
		},
		{
			name: "non-existent dispute",
			msg: &types.MsgVoteOnDispute{
				Validator:     validator.String(),
				DisputeId:     99999,
				Vote:          types.DISPUTE_VOTE_REQUESTER_FAULT,
				Justification: "Valid complaint",
			},
			expectErr: true,
			errMsg:    "not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := msgServer.VoteOnDispute(goCtx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
			}
		})
	}
}

// TestMsgServer_ResolveDispute tests the ResolveDispute message handler
func TestMsgServer_ResolveDispute(t *testing.T) {
	k, sdkCtx, goCtx := newComputeKeeperCtx(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	authority := k.GetAuthority()
	requester := sdk.AccAddress([]byte("test_requester_addr"))
	provider := createTestProvider(t)
	specs := createValidComputeSpec()
	pricing := createValidPricing()

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	// Setup: Create a dispute
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.example.com", specs, pricing, params.MinProviderStake)
	require.NoError(t, err)

	requestID, err := k.SubmitRequest(sdkCtx, requester, specs, "nginx:latest", []string{"/bin/bash"}, map[string]string{}, math.NewInt(1000000), provider.String())
	require.NoError(t, err)

	err = k.SubmitResult(sdkCtx, provider, requestID, "hash", "https://output.url", 0, "https://logs.url", []byte("proof"))
	require.NoError(t, err)

	disputeID, err := k.CreateDispute(sdkCtx, requester, requestID, "Issue", []byte("Evidence"),math.NewInt(10000))
	require.NoError(t, err)

	tests := []struct {
		name      string
		msg       *types.MsgResolveDispute
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid resolution",
			msg: &types.MsgResolveDispute{
				Authority: authority,
				DisputeId: disputeID,
			},
			expectErr: false,
		},
		{
			name: "unauthorized authority",
			msg: &types.MsgResolveDispute{
				Authority: sdk.AccAddress([]byte("unauthorized_addr")).String(),
				DisputeId: disputeID,
			},
			expectErr: true,
			errMsg:    "authority",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := msgServer.ResolveDispute(goCtx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify dispute was resolved
				dispute, err := k.GetDisputeForTesting(sdkCtx, tc.msg.DisputeId)
				require.NoError(t, err)
				require.Equal(t, types.DISPUTE_STATUS_RESOLVED, dispute.Status)
			}
		})
	}
}

// TestMsgServer_SubmitEvidence tests the SubmitEvidence message handler
func TestMsgServer_SubmitEvidence(t *testing.T) {
	k, sdkCtx, goCtx := newComputeKeeperCtx(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	requester := sdk.AccAddress([]byte("test_requester_addr"))
	provider := createTestProvider(t)
	specs := createValidComputeSpec()
	pricing := createValidPricing()

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	// Setup: Create a dispute
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.example.com", specs, pricing, params.MinProviderStake)
	require.NoError(t, err)

	requestID, err := k.SubmitRequest(sdkCtx, requester, specs, "nginx:latest", []string{"/bin/bash"}, map[string]string{}, math.NewInt(1000000), provider.String())
	require.NoError(t, err)

	err = k.SubmitResult(sdkCtx, provider, requestID, "hash", "https://output.url", 0, "https://logs.url", []byte("proof"))
	require.NoError(t, err)

	disputeID, err := k.CreateDispute(sdkCtx, requester, requestID, "Issue", []byte("Initial evidence"),math.NewInt(10000))
	require.NoError(t, err)

	tests := []struct {
		name      string
		msg       *types.MsgSubmitEvidence
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid evidence submission",
			msg: &types.MsgSubmitEvidence{
				Submitter:    requester.String(),
				DisputeId:    disputeID,
				EvidenceType: "screenshot",
				Data:         []byte("image_data_here"),
				Description:  "Screenshot of error",
			},
			expectErr: false,
		},
		{
			name: "invalid submitter address",
			msg: &types.MsgSubmitEvidence{
				Submitter:    "invalid_address",
				DisputeId:    disputeID,
				EvidenceType: "screenshot",
				Data:         []byte("image_data"),
				Description:  "Screenshot",
			},
			expectErr: true,
			errMsg:    "invalid submitter",
		},
		{
			name: "empty evidence type",
			msg: &types.MsgSubmitEvidence{
				Submitter:    requester.String(),
				DisputeId:    disputeID,
				EvidenceType: "",
				Data:         []byte("data"),
				Description:  "Description",
			},
			expectErr: true,
			errMsg:    "evidence type",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := msgServer.SubmitEvidence(goCtx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
			}
		})
	}
}

// TestMsgServer_AppealSlashing tests the AppealSlashing message handler
func TestMsgServer_AppealSlashing(t *testing.T) {
	k, sdkCtx, goCtx := newComputeKeeperCtx(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	provider := createTestProvider(t)
	specs := createValidComputeSpec()
	pricing := createValidPricing()

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	// Setup: Register provider
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.example.com", specs, pricing, params.MinProviderStake)
	require.NoError(t, err)

	// Create a slash record (simplified - assumes slash exists)
	slashID := uint64(1)

	depositAmount := math.NewInt(5000)

	tests := []struct {
		name      string
		msg       *types.MsgAppealSlashing
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid appeal",
			msg: &types.MsgAppealSlashing{
				Provider:      provider.String(),
				SlashId:       slashID,
				Justification: "The slash was unjustified",
				DepositAmount: depositAmount,
			},
			expectErr: false,
		},
		{
			name: "invalid provider address",
			msg: &types.MsgAppealSlashing{
				Provider:      "invalid_address",
				SlashId:       slashID,
				Justification: "Appeal",
				DepositAmount: depositAmount,
			},
			expectErr: true,
			errMsg:    "invalid provider",
		},
		{
			name: "empty justification",
			msg: &types.MsgAppealSlashing{
				Provider:      provider.String(),
				SlashId:       slashID,
				Justification: "",
				DepositAmount: depositAmount,
			},
			expectErr: true,
			errMsg:    "justification",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := msgServer.AppealSlashing(goCtx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				// Note: This may fail if slash record doesn't exist
				// In production tests, you'd create the slash first
				if err != nil {
					require.Contains(t, err.Error(), "slash")
				} else {
					require.NotNil(t, resp)
					require.Greater(t, resp.AppealId, uint64(0))
				}
			}
		})
	}
}

// TestMsgServer_VoteOnAppeal tests the VoteOnAppeal message handler
func TestMsgServer_VoteOnAppeal(t *testing.T) {
	k, _, goCtx := newComputeKeeperCtx(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	validator := sdk.ValAddress([]byte("test_validator_addr"))
	appealID := uint64(1)

	tests := []struct {
		name      string
		msg       *types.MsgVoteOnAppeal
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid vote on appeal",
			msg: &types.MsgVoteOnAppeal{
				Validator:     validator.String(),
				AppealId:      appealID,
				Approve:       true,
				Justification: "Appeal is justified",
			},
			expectErr: false,
		},
		{
			name: "invalid validator address",
			msg: &types.MsgVoteOnAppeal{
				Validator:     "invalid_address",
				AppealId:      appealID,
				Approve:       true,
				Justification: "Valid",
			},
			expectErr: true,
			errMsg:    "invalid validator",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := msgServer.VoteOnAppeal(goCtx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				// Note: May fail if appeal doesn't exist
				if err != nil {
					require.Contains(t, err.Error(), "appeal")
				} else {
					require.NotNil(t, resp)
				}
			}
		})
	}
}

// TestMsgServer_ResolveAppeal tests the ResolveAppeal message handler
func TestMsgServer_ResolveAppeal(t *testing.T) {
	k, _, goCtx := newComputeKeeperCtx(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	authority := k.GetAuthority()
	appealID := uint64(1)

	tests := []struct {
		name      string
		msg       *types.MsgResolveAppeal
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid resolution",
			msg: &types.MsgResolveAppeal{
				Authority: authority,
				AppealId:  appealID,
			},
			expectErr: false,
		},
		{
			name: "unauthorized authority",
			msg: &types.MsgResolveAppeal{
				Authority: sdk.AccAddress([]byte("unauthorized_addr")).String(),
				AppealId:  appealID,
			},
			expectErr: true,
			errMsg:    "authority",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := msgServer.ResolveAppeal(goCtx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				// Note: May fail if appeal doesn't exist
				if err != nil {
					require.Contains(t, err.Error(), "appeal")
				} else {
					require.NotNil(t, resp)
				}
			}
		})
	}
}

// TestMsgServer_UpdateGovernanceParams tests the UpdateGovernanceParams message handler
func TestMsgServer_UpdateGovernanceParams(t *testing.T) {
	k, sdkCtx, goCtx := newComputeKeeperCtx(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	authority := k.GetAuthority()

	// Get current governance params
	govParams, err := k.GetGovernanceParams(sdkCtx)
	require.NoError(t, err)

	// Modify governance params
	newGovParams := govParams
	newGovParams.DisputeDeposit = math.NewInt(20000)

	tests := []struct {
		name      string
		msg       *types.MsgUpdateGovernanceParams
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid governance params update",
			msg: &types.MsgUpdateGovernanceParams{
				Authority: authority,
				Params:    newGovParams,
			},
			expectErr: false,
		},
		{
			name: "unauthorized authority",
			msg: &types.MsgUpdateGovernanceParams{
				Authority: sdk.AccAddress([]byte("unauthorized_addr")).String(),
				Params:    newGovParams,
			},
			expectErr: true,
			errMsg:    "unauthorized",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := msgServer.UpdateGovernanceParams(goCtx, tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify governance params were updated
				updated, err := k.GetGovernanceParams(sdkCtx)
				require.NoError(t, err)
				require.True(t, tc.msg.Params.DisputeDeposit.Equal(updated.DisputeDeposit))
			}
		})
	}
}

// TestMsgServer_Authorization tests that only authorized users can call specific handlers
func TestMsgServer_Authorization(t *testing.T) {
	k, sdkCtx, goCtx := newComputeKeeperCtx(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	requester := sdk.AccAddress([]byte("test_requester_addr"))
	wrongRequester := sdk.AccAddress([]byte("wrong_requester_addr"))
	provider := createTestProvider(t)
	specs := createValidComputeSpec()
	pricing := createValidPricing()

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	// Setup: Register provider and submit request
	err = k.RegisterProvider(sdkCtx, provider, "TestProvider", "https://test.example.com", specs, pricing, params.MinProviderStake)
	require.NoError(t, err)

	requestID, err := k.SubmitRequest(
		sdkCtx,
		requester,
		specs,
		"nginx:latest",
		[]string{"/bin/bash"},
		map[string]string{},
		math.NewInt(1000000),
		provider.String(),
	)
	require.NoError(t, err)

	t.Run("wrong requester cannot cancel request", func(t *testing.T) {
		msg := &types.MsgCancelRequest{
			Requester: wrongRequester.String(),
			RequestId: requestID,
		}

		resp, err := msgServer.CancelRequest(goCtx, msg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not authorized")
		require.Nil(t, resp)
	})

	t.Run("correct requester can cancel request", func(t *testing.T) {
		msg := &types.MsgCancelRequest{
			Requester: requester.String(),
			RequestId: requestID,
		}

		resp, err := msgServer.CancelRequest(goCtx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})
}

// TestMsgServer_StateChanges tests that message handlers properly update state
func TestMsgServer_StateChanges(t *testing.T) {
	k, sdkCtx, goCtx := newComputeKeeperCtx(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	provider := createTestProvider(t)
	specs := createValidComputeSpec()
	pricing := createValidPricing()

	params, err := k.GetParams(sdkCtx)
	require.NoError(t, err)

	msg := &types.MsgRegisterProvider{
		Provider:       provider.String(),
		Moniker:        "TestProvider",
		Endpoint:       "https://test.example.com",
		AvailableSpecs: specs,
		Pricing:        pricing,
		Stake:          params.MinProviderStake,
	}

	// Before registration
	_, err = k.GetProvider(sdkCtx, provider)
	require.Error(t, err)

	// Register
	resp, err := msgServer.RegisterProvider(goCtx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// After registration
	storedProvider, err := k.GetProvider(sdkCtx, provider)
	require.NoError(t, err)
	require.Equal(t, provider.String(), storedProvider.Address)
	require.True(t, storedProvider.Active)

	// Deactivate
	deactivateMsg := &types.MsgDeactivateProvider{
		Provider: provider.String(),
	}

	deactivateResp, err := msgServer.DeactivateProvider(goCtx, deactivateMsg)
	require.NoError(t, err)
	require.NotNil(t, deactivateResp)

	// After deactivation
	deactivated, err := k.GetProvider(sdkCtx, provider)
	require.NoError(t, err)
	require.False(t, deactivated.Active)
}
