package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

type KeeperTestSuite struct {
	suite.Suite
	keeper keeper.Keeper
	ctx    sdk.Context
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.keeper, suite.ctx = keepertest.OracleKeeper(suite.T())
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

// TestRegisterOracle validates oracle registration
func TestRegisterOracle(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	tests := []struct {
		name    string
		msg     *types.MsgRegisterOracle
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid oracle registration",
			msg: &types.MsgRegisterOracle{
				Validator: "paw1validator",
			},
			wantErr: false,
		},
		{
			name: "duplicate registration",
			msg: &types.MsgRegisterOracle{
				Validator: "paw1validator",
			},
			wantErr: true,
			errMsg:  "already registered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := k.RegisterOracle(ctx, tt.msg)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify oracle is registered
				oracle, found := k.GetOracle(ctx, tt.msg.Validator)
				require.True(t, found)
				require.Equal(t, tt.msg.Validator, oracle.Validator)
			}
		})
	}
}

// TestSubmitPrice validates price feed submission
func TestSubmitPrice(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Register oracle first
	oracleAddr := "paw1oracle"
	keepertest.RegisterTestOracle(t, k, ctx, oracleAddr)

	tests := []struct {
		name    string
		msg     *types.MsgSubmitPrice
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid price submission - BTC",
			msg: &types.MsgSubmitPrice{
				Oracle: oracleAddr,
				Asset:  "BTC/USD",
				Price:  sdk.MustNewDecFromStr("45000.00"),
			},
			wantErr: false,
		},
		{
			name: "valid price submission - ETH",
			msg: &types.MsgSubmitPrice{
				Oracle: oracleAddr,
				Asset:  "ETH/USD",
				Price:  sdk.MustNewDecFromStr("2500.00"),
			},
			wantErr: false,
		},
		{
			name: "unregistered oracle",
			msg: &types.MsgSubmitPrice{
				Oracle: "paw1unregistered",
				Asset:  "BTC/USD",
				Price:  sdk.MustNewDecFromStr("45000.00"),
			},
			wantErr: true,
			errMsg:  "oracle not registered",
		},
		{
			name: "negative price",
			msg: &types.MsgSubmitPrice{
				Oracle: oracleAddr,
				Asset:  "BTC/USD",
				Price:  sdk.MustNewDecFromStr("-100.00"),
			},
			wantErr: true,
			errMsg:  "price must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := k.SubmitPrice(ctx, tt.msg)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				// Verify price is stored
				price, found := k.GetPrice(ctx, tt.msg.Asset, tt.msg.Oracle)
				require.True(t, found)
				require.Equal(t, tt.msg.Price, price.Price)
			}
		})
	}
}

// TestGetMedianPrice validates median price calculation
func TestGetMedianPrice(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Register multiple oracles
	oracle1 := "paw1oracle1"
	oracle2 := "paw1oracle2"
	oracle3 := "paw1oracle3"

	keepertest.RegisterTestOracle(t, k, ctx, oracle1)
	keepertest.RegisterTestOracle(t, k, ctx, oracle2)
	keepertest.RegisterTestOracle(t, k, ctx, oracle3)

	// Submit prices for BTC/USD
	asset := "BTC/USD"
	keepertest.SubmitTestPrice(t, k, ctx, oracle1, asset, sdk.MustNewDecFromStr("44000.00"))
	keepertest.SubmitTestPrice(t, k, ctx, oracle2, asset, sdk.MustNewDecFromStr("45000.00"))
	keepertest.SubmitTestPrice(t, k, ctx, oracle3, asset, sdk.MustNewDecFromStr("46000.00"))

	// Get median price
	medianPrice, err := k.GetMedianPrice(ctx, asset)
	require.NoError(t, err)

	// Median of [44000, 45000, 46000] should be 45000
	expectedMedian := sdk.MustNewDecFromStr("45000.00")
	require.Equal(t, expectedMedian, medianPrice)
}

// TestPriceDeviation validates price deviation detection
func TestPriceDeviation(t *testing.T) {
	k, ctx := keepertest.OracleKeeper(t)

	// Register oracles
	oracle1 := "paw1oracle1"
	oracle2 := "paw1oracle2"

	keepertest.RegisterTestOracle(t, k, ctx, oracle1)
	keepertest.RegisterTestOracle(t, k, ctx, oracle2)

	asset := "BTC/USD"

	// Submit normal prices
	keepertest.SubmitTestPrice(t, k, ctx, oracle1, asset, sdk.MustNewDecFromStr("45000.00"))
	keepertest.SubmitTestPrice(t, k, ctx, oracle2, asset, sdk.MustNewDecFromStr("45100.00"))

	// Verify prices are within acceptable deviation
	price1, _ := k.GetPrice(ctx, asset, oracle1)
	price2, _ := k.GetPrice(ctx, asset, oracle2)

	deviation := price1.Price.Sub(price2.Price).Abs().Quo(price1.Price)
	maxDeviation := sdk.MustNewDecFromStr("0.05") // 5% max deviation

	require.True(t, deviation.LTE(maxDeviation), "Price deviation should be within limits")
}
