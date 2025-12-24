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

// CommitSwap handles swap commitment (phase 1 of commit-reveal MEV protection)
func (ms msgServer) CommitSwap(goCtx context.Context, msg *types.MsgCommitSwap) (*types.MsgCommitSwapResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Check if commit-reveal is enabled
	params, err := ms.Keeper.GetParams(goCtx)
	if err != nil {
		return nil, err
	}

	if !params.EnableCommitReveal {
		return nil, types.ErrCommitRevealDisabled.Wrap("commit-reveal feature is disabled by governance")
	}

	// Parse trader address
	trader, err := sdk.AccAddressFromBech32(msg.Trader)
	if err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// Store the commitment
	commit := types.SwapCommit{
		Trader:       msg.Trader,
		SwapHash:     msg.SwapHash,
		CommitHeight: ctx.BlockHeight(),
		ExpiryHeight: ctx.BlockHeight() + int64(params.CommitTimeoutBlocks),
	}

	if err := ms.Keeper.SetSwapCommit(goCtx, commit); err != nil {
		return nil, err
	}

	earliestReveal := ctx.BlockHeight() + int64(params.CommitRevealDelay)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"swap_committed",
			sdk.NewAttribute("trader", msg.Trader),
			sdk.NewAttribute("swap_hash", msg.SwapHash),
			sdk.NewAttribute("commit_height", ctx.HeaderInfo().Height.String()),
			sdk.NewAttribute("earliest_reveal_height", ctx.HeaderInfo().Height.String()),
			sdk.NewAttribute("expiry_height", ctx.HeaderInfo().Height.String()),
		),
	)

	return &types.MsgCommitSwapResponse{
		CommitHeight:         ctx.BlockHeight(),
		EarliestRevealHeight: earliestReveal,
		ExpiryHeight:         commit.ExpiryHeight,
	}, nil
}

// RevealSwap handles swap reveal and execution (phase 2 of commit-reveal MEV protection)
func (ms msgServer) RevealSwap(goCtx context.Context, msg *types.MsgRevealSwap) (*types.MsgRevealSwapResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Check if commit-reveal is enabled
	params, err := ms.Keeper.GetParams(goCtx)
	if err != nil {
		return nil, err
	}

	if !params.EnableCommitReveal {
		return nil, types.ErrCommitRevealDisabled.Wrap("commit-reveal feature is disabled by governance")
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

	// Compute the hash from revealed parameters
	revealedHash := ms.Keeper.ComputeRevealHash(msg)

	// Get the commitment
	commit, err := ms.Keeper.GetSwapCommitByHash(goCtx, revealedHash)
	if err != nil {
		return nil, types.ErrCommitmentNotFound.Wrap("no matching commitment found for revealed parameters")
	}

	// Verify trader matches
	if commit.Trader != msg.Trader {
		return nil, types.ErrUnauthorized.Wrap("trader address does not match commitment")
	}

	// Check reveal delay has passed
	if ctx.BlockHeight() < commit.CommitHeight+int64(params.CommitRevealDelay) {
		return nil, types.ErrRevealTooEarly.Wrapf(
			"reveal allowed at block %d, current block %d",
			commit.CommitHeight+int64(params.CommitRevealDelay),
			ctx.BlockHeight(),
		)
	}

	// Check commitment hasn't expired
	if ctx.BlockHeight() >= commit.ExpiryHeight {
		return nil, types.ErrCommitmentExpired.Wrapf(
			"commitment expired at block %d, current block %d",
			commit.ExpiryHeight,
			ctx.BlockHeight(),
		)
	}

	// Delete the commitment now that it's being used
	if err := ms.Keeper.DeleteSwapCommit(goCtx, revealedHash); err != nil {
		return nil, err
	}

	// Execute the swap using secure implementation
	amountOut, err := ms.Keeper.ExecuteSwapSecure(goCtx, trader, msg.PoolId, msg.TokenIn, msg.TokenOut, msg.AmountIn, msg.MinAmountOut)
	if err != nil {
		return nil, err
	}

	// Emit reveal event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"swap_revealed",
			sdk.NewAttribute("trader", msg.Trader),
			sdk.NewAttribute("pool_id", ctx.HeaderInfo().Height.String()),
			sdk.NewAttribute("amount_in", msg.AmountIn.String()),
			sdk.NewAttribute("amount_out", amountOut.String()),
			sdk.NewAttribute("commit_height", ctx.HeaderInfo().Height.String()),
			sdk.NewAttribute("reveal_height", ctx.HeaderInfo().Height.String()),
		),
	)

	return &types.MsgRevealSwapResponse{
		AmountOut: amountOut,
	}, nil
}
