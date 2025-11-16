package types_test

import (
	"testing"

	"cosmossdk.io/math"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/dex/types"
)

// TestMsgCreatePool_ValidateBasic validates MsgCreatePool message validation
func TestMsgCreatePool_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     types.MsgCreatePool
		wantErr bool
		errType error
	}{
		{
			name: "valid message",
			msg: types.MsgCreatePool{
				Creator: "paw1validaddress",
				TokenA:  "upaw",
				TokenB:  "uusdt",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(1000000),
			},
			wantErr: false,
		},
		{
			name: "invalid creator address",
			msg: types.MsgCreatePool{
				Creator: "invalid",
				TokenA:  "upaw",
				TokenB:  "uusdt",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(1000000),
			},
			wantErr: true,
			errType: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "empty token A",
			msg: types.MsgCreatePool{
				Creator: "paw1validaddress",
				TokenA:  "",
				TokenB:  "uusdt",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(1000000),
			},
			wantErr: true,
		},
		{
			name: "empty token B",
			msg: types.MsgCreatePool{
				Creator: "paw1validaddress",
				TokenA:  "upaw",
				TokenB:  "",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(1000000),
			},
			wantErr: true,
		},
		{
			name: "same tokens",
			msg: types.MsgCreatePool{
				Creator: "paw1validaddress",
				TokenA:  "upaw",
				TokenB:  "upaw",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(1000000),
			},
			wantErr: true,
		},
		{
			name: "zero amount A",
			msg: types.MsgCreatePool{
				Creator: "paw1validaddress",
				TokenA:  "upaw",
				TokenB:  "uusdt",
				AmountA: math.NewInt(0),
				AmountB: math.NewInt(1000000),
			},
			wantErr: true,
		},
		{
			name: "negative amount B",
			msg: types.MsgCreatePool{
				Creator: "paw1validaddress",
				TokenA:  "upaw",
				TokenB:  "uusdt",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(-1),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					require.ErrorIs(t, err, tt.errType)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestMsgSwap_ValidateBasic validates MsgSwap message validation
func TestMsgSwap_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     types.MsgSwap
		wantErr bool
		errType error
	}{
		{
			name: "valid swap",
			msg: types.MsgSwap{
				Trader:       "paw1trader",
				PoolId:       1,
				TokenIn:      "upaw",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
			},
			wantErr: false,
		},
		{
			name: "invalid trader address",
			msg: types.MsgSwap{
				Trader:       "invalid",
				PoolId:       1,
				TokenIn:      "upaw",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
			},
			wantErr: true,
			errType: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "zero pool id",
			msg: types.MsgSwap{
				Trader:       "paw1trader",
				PoolId:       0,
				TokenIn:      "upaw",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
			},
			wantErr: true,
		},
		{
			name: "empty token in",
			msg: types.MsgSwap{
				Trader:       "paw1trader",
				PoolId:       1,
				TokenIn:      "",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
			},
			wantErr: true,
		},
		{
			name: "zero amount in",
			msg: types.MsgSwap{
				Trader:       "paw1trader",
				PoolId:       1,
				TokenIn:      "upaw",
				AmountIn:     math.NewInt(0),
				MinAmountOut: math.NewInt(900000),
			},
			wantErr: true,
		},
		{
			name: "negative min amount out",
			msg: types.MsgSwap{
				Trader:       "paw1trader",
				PoolId:       1,
				TokenIn:      "upaw",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(-1),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					require.ErrorIs(t, err, tt.errType)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestMsgAddLiquidity_ValidateBasic validates MsgAddLiquidity message validation
func TestMsgAddLiquidity_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     types.MsgAddLiquidity
		wantErr bool
		errType error
	}{
		{
			name: "valid add liquidity",
			msg: types.MsgAddLiquidity{
				Provider: "paw1provider",
				PoolId:   1,
				AmountA:  math.NewInt(1000000),
				AmountB:  math.NewInt(2000000),
			},
			wantErr: false,
		},
		{
			name: "invalid provider address",
			msg: types.MsgAddLiquidity{
				Provider: "invalid",
				PoolId:   1,
				AmountA:  math.NewInt(1000000),
				AmountB:  math.NewInt(2000000),
			},
			wantErr: true,
			errType: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "zero pool id",
			msg: types.MsgAddLiquidity{
				Provider: "paw1provider",
				PoolId:   0,
				AmountA:  math.NewInt(1000000),
				AmountB:  math.NewInt(2000000),
			},
			wantErr: true,
		},
		{
			name: "zero amount A",
			msg: types.MsgAddLiquidity{
				Provider: "paw1provider",
				PoolId:   1,
				AmountA:  math.NewInt(0),
				AmountB:  math.NewInt(2000000),
			},
			wantErr: true,
		},
		{
			name: "negative amount B",
			msg: types.MsgAddLiquidity{
				Provider: "paw1provider",
				PoolId:   1,
				AmountA:  math.NewInt(1000000),
				AmountB:  math.NewInt(-1),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					require.ErrorIs(t, err, tt.errType)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestMsgRemoveLiquidity_ValidateBasic validates MsgRemoveLiquidity message validation
func TestMsgRemoveLiquidity_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     types.MsgRemoveLiquidity
		wantErr bool
		errType error
	}{
		{
			name: "valid remove liquidity",
			msg: types.MsgRemoveLiquidity{
				Provider: "paw1provider",
				PoolId:   1,
				Shares:   math.NewInt(1000000),
			},
			wantErr: false,
		},
		{
			name: "invalid provider address",
			msg: types.MsgRemoveLiquidity{
				Provider: "invalid",
				PoolId:   1,
				Shares:   math.NewInt(1000000),
			},
			wantErr: true,
			errType: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "zero pool id",
			msg: types.MsgRemoveLiquidity{
				Provider: "paw1provider",
				PoolId:   0,
				Shares:   math.NewInt(1000000),
			},
			wantErr: true,
		},
		{
			name: "zero liquidity shares",
			msg: types.MsgRemoveLiquidity{
				Provider: "paw1provider",
				PoolId:   1,
				Shares:   math.NewInt(0),
			},
			wantErr: true,
		},
		{
			name: "negative shares",
			msg: types.MsgRemoveLiquidity{
				Provider: "paw1provider",
				PoolId:   1,
				Shares:   math.NewInt(-1),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					require.ErrorIs(t, err, tt.errType)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
