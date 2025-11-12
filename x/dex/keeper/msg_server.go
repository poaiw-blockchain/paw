package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/dex/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// CreatePool handles the creation of a new liquidity pool
func (ms msgServer) CreatePool(goCtx context.Context, msg *types.MsgCreatePool) (*types.MsgCreatePoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	poolId, err := ms.Keeper.CreatePool(
		ctx,
		msg.Creator,
		msg.TokenA,
		msg.TokenB,
		msg.AmountA,
		msg.AmountB,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgCreatePoolResponse{
		PoolId: poolId,
	}, nil
}

// Swap handles token swaps
func (ms msgServer) Swap(goCtx context.Context, msg *types.MsgSwap) (*types.MsgSwapResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	amountOut, err := ms.Keeper.Swap(
		ctx,
		msg.Trader,
		msg.PoolId,
		msg.TokenIn,
		msg.TokenOut,
		msg.AmountIn,
		msg.MinAmountOut,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgSwapResponse{
		AmountOut: amountOut,
	}, nil
}

// AddLiquidity handles adding liquidity to a pool
func (ms msgServer) AddLiquidity(goCtx context.Context, msg *types.MsgAddLiquidity) (*types.MsgAddLiquidityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	shares, err := ms.Keeper.AddLiquidity(
		ctx,
		msg.Provider,
		msg.PoolId,
		msg.AmountA,
		msg.AmountB,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgAddLiquidityResponse{
		Shares: shares,
	}, nil
}

// RemoveLiquidity handles removing liquidity from a pool
func (ms msgServer) RemoveLiquidity(goCtx context.Context, msg *types.MsgRemoveLiquidity) (*types.MsgRemoveLiquidityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	amountA, amountB, err := ms.Keeper.RemoveLiquidity(
		ctx,
		msg.Provider,
		msg.PoolId,
		msg.Shares,
	)
	if err != nil {
		return nil, err
	}

	return &types.MsgRemoveLiquidityResponse{
		AmountA: amountA,
		AmountB: amountB,
	}, nil
}
