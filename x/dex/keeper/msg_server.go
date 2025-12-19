package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the dex MsgServer interface
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// CreatePool handles the creation of a new liquidity pool
func (ms msgServer) CreatePool(goCtx context.Context, msg *types.MsgCreatePool) (*types.MsgCreatePoolResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Parse creator address
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	// Create pool using secure implementation
	pool, err := ms.Keeper.CreatePoolSecure(goCtx, creator, msg.TokenA, msg.TokenB, msg.AmountA, msg.AmountB)
	if err != nil {
		return nil, err
	}

	return &types.MsgCreatePoolResponse{
		PoolId: pool.Id,
	}, nil
}

// AddLiquidity handles adding liquidity to an existing pool
func (ms msgServer) AddLiquidity(goCtx context.Context, msg *types.MsgAddLiquidity) (*types.MsgAddLiquidityResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Parse provider address
	provider, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return nil, err
	}

	// Add liquidity using secure implementation
	shares, err := ms.Keeper.AddLiquiditySecure(goCtx, provider, msg.PoolId, msg.AmountA, msg.AmountB)
	if err != nil {
		return nil, err
	}

	return &types.MsgAddLiquidityResponse{
		Shares: shares,
	}, nil
}

// RemoveLiquidity handles removing liquidity from a pool
func (ms msgServer) RemoveLiquidity(goCtx context.Context, msg *types.MsgRemoveLiquidity) (*types.MsgRemoveLiquidityResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Parse provider address
	provider, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return nil, err
	}

	// Remove liquidity using secure implementation
	amountA, amountB, err := ms.Keeper.RemoveLiquiditySecure(goCtx, provider, msg.PoolId, msg.Shares)
	if err != nil {
		return nil, err
	}

	return &types.MsgRemoveLiquidityResponse{
		AmountA: amountA,
		AmountB: amountB,
	}, nil
}

// Swap handles token swaps
func (ms msgServer) Swap(goCtx context.Context, msg *types.MsgSwap) (*types.MsgSwapResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Check deadline - must be after current block time
	ctx := sdk.UnwrapSDKContext(goCtx)
	if ctx.BlockTime().Unix() > msg.Deadline {
		return nil, types.ErrDeadlineExceeded
	}

	// Parse trader address
	trader, err := sdk.AccAddressFromBech32(msg.Trader)
	if err != nil {
		return nil, err
	}

	// Execute swap using secure implementation
	amountOut, err := ms.Keeper.ExecuteSwapSecure(goCtx, trader, msg.PoolId, msg.TokenIn, msg.TokenOut, msg.AmountIn, msg.MinAmountOut)
	if err != nil {
		return nil, err
	}

	return &types.MsgSwapResponse{
		AmountOut: amountOut,
	}, nil
}
