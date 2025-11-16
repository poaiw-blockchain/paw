package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

type KeeperTestSuite struct {
	suite.Suite
	keeper *keeper.Keeper
	ctx    sdk.Context
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.keeper, suite.ctx = keepertest.ComputeKeeper(suite.T())
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

// TestRegisterProvider validates provider registration
func TestRegisterProvider(t *testing.T) {
	t.Skip("TODO: Implement RegisterProvider method in compute keeper")
	k, ctx := keepertest.ComputeKeeper(t)
	_ = k
	_ = ctx

	tests := []struct {
		name    string
		msg     *types.MsgRegisterProvider
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid provider registration",
			msg: &types.MsgRegisterProvider{
				Provider: "paw1provider",
				Endpoint: "https://api.compute-provider.io",
				Stake:    math.NewInt(1000000),
			},
			wantErr: false,
		},
		{
			name: "invalid endpoint",
			msg: &types.MsgRegisterProvider{
				Provider: "paw1provider",
				Endpoint: "",
				Stake:    math.NewInt(1000000),
			},
			wantErr: true,
			errMsg:  "endpoint cannot be empty",
		},
		{
			name: "insufficient stake",
			msg: &types.MsgRegisterProvider{
				Provider: "paw1provider",
				Endpoint: "https://api.compute-provider.io",
				Stake:    math.NewInt(0),
			},
			wantErr: true,
			errMsg:  "stake must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := k.RegisterProvider(ctx, tt.msg)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify provider is registered
				provider, found := k.GetProvider(ctx, tt.msg.Provider)
				require.True(t, found)
				require.Equal(t, tt.msg.Endpoint, provider.Endpoint)
				require.Equal(t, tt.msg.Stake, provider.Stake)
			}
		})
	}
}

// TestRequestCompute validates compute request submission
func TestRequestCompute(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	// Register a provider first
	keepertest.RegisterTestProvider(t, k, ctx, "paw1provider", "https://api.provider.io", math.NewInt(1000000))

	tests := []struct {
		name    string
		msg     *types.MsgRequestCompute
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid compute request",
			msg: &types.MsgRequestCompute{
				Requester: "paw1requester",
				ApiUrl:    "https://api.openai.com/v1/chat/completions",
				MaxFee:    math.NewInt(1000),
			},
			wantErr: false,
		},
		{
			name: "empty API URL",
			msg: &types.MsgRequestCompute{
				Requester: "paw1requester",
				ApiUrl:    "",
				MaxFee:    math.NewInt(1000),
			},
			wantErr: true,
			errMsg:  "API URL cannot be empty",
		},
		{
			name: "zero max fee",
			msg: &types.MsgRequestCompute{
				Requester: "paw1requester",
				ApiUrl:    "https://api.openai.com/v1/chat/completions",
				MaxFee:    math.NewInt(0),
			},
			wantErr: true,
			errMsg:  "max fee must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := k.RequestCompute(ctx, tt.msg)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Greater(t, resp.RequestId, uint64(0))

				// Verify request is stored
				request, found := k.GetRequest(ctx, resp.RequestId)
				require.True(t, found)
				require.Equal(t, tt.msg.ApiUrl, request.ApiUrl)
				require.Equal(t, types.RequestStatus_PENDING, request.Status)
			}
		})
	}
}

// TestSubmitResult validates compute result submission
func TestSubmitResult(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	// Setup: Register provider and create request
	providerAddr := "paw1provider"
	keepertest.RegisterTestProvider(t, k, ctx, providerAddr, "https://api.provider.io", math.NewInt(1000000))
	requestId := keepertest.SubmitTestRequest(t, k, ctx, "paw1requester", "https://api.openai.com/v1/chat/completions")

	tests := []struct {
		name    string
		msg     *types.MsgSubmitResult
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid result submission",
			msg: &types.MsgSubmitResult{
				Provider:  providerAddr,
				RequestId: requestId,
				Result:    `{"response": "Hello from AI"}`,
			},
			wantErr: false,
		},
		{
			name: "invalid request ID",
			msg: &types.MsgSubmitResult{
				Provider:  providerAddr,
				RequestId: 999999,
				Result:    `{"response": "Hello from AI"}`,
			},
			wantErr: true,
			errMsg:  "request not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := k.SubmitResult(ctx, tt.msg)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify request status updated
				request, found := k.GetRequest(ctx, tt.msg.RequestId)
				require.True(t, found)
				require.Equal(t, types.RequestStatus_COMPLETED, request.Status)
			}
		})
	}
}
