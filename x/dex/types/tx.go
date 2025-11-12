package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MsgServer defines the message server interface
type MsgServer interface {
	CreatePool(context.Context, *MsgCreatePool) (*MsgCreatePoolResponse, error)
	Swap(context.Context, *MsgSwap) (*MsgSwapResponse, error)
	AddLiquidity(context.Context, *MsgAddLiquidity) (*MsgAddLiquidityResponse, error)
	RemoveLiquidity(context.Context, *MsgRemoveLiquidity) (*MsgRemoveLiquidityResponse, error)
}

// Response types

// MsgCreatePoolResponse defines the response for CreatePool
type MsgCreatePoolResponse struct {
	PoolId uint64 `json:"pool_id"`
}

// MsgSwapResponse defines the response for Swap
type MsgSwapResponse struct {
	AmountOut sdk.Int `json:"amount_out"`
}

// MsgAddLiquidityResponse defines the response for AddLiquidity
type MsgAddLiquidityResponse struct {
	Shares sdk.Int `json:"shares"`
}

// MsgRemoveLiquidityResponse defines the response for RemoveLiquidity
type MsgRemoveLiquidityResponse struct {
	AmountA sdk.Int `json:"amount_a"`
	AmountB sdk.Int `json:"amount_b"`
}

// Placeholder for protobuf service descriptor
var _Msg_serviceDesc = struct{}{}
