package keeper

import (
	"context"
	"fmt"

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
		return nil, fmt.Errorf("CreatePool: validate: %w", err)
	}

	// Parse creator address
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("CreatePool: invalid creator address: %w", err)
	}

	// Create pool using secure implementation
	pool, err := ms.Keeper.CreatePool(goCtx, creator, msg.TokenA, msg.TokenB, msg.AmountA, msg.AmountB)
	if err != nil {
		return nil, fmt.Errorf("CreatePool: %w", err)
	}

	return &types.MsgCreatePoolResponse{
		PoolId: pool.Id,
	}, nil
}

// AddLiquidity handles adding liquidity to an existing pool
func (ms msgServer) AddLiquidity(goCtx context.Context, msg *types.MsgAddLiquidity) (*types.MsgAddLiquidityResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("AddLiquidity: validate: %w", err)
	}

	// Parse provider address
	provider, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return nil, fmt.Errorf("AddLiquidity: invalid provider address: %w", err)
	}

	// Add liquidity using secure implementation
	shares, err := ms.Keeper.AddLiquidity(goCtx, provider, msg.PoolId, msg.AmountA, msg.AmountB)
	if err != nil {
		return nil, fmt.Errorf("AddLiquidity: %w", err)
	}

	return &types.MsgAddLiquidityResponse{
		Shares: shares,
	}, nil
}

// RemoveLiquidity handles removing liquidity from a pool
func (ms msgServer) RemoveLiquidity(goCtx context.Context, msg *types.MsgRemoveLiquidity) (*types.MsgRemoveLiquidityResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("RemoveLiquidity: validate: %w", err)
	}

	// Parse provider address
	provider, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return nil, fmt.Errorf("RemoveLiquidity: invalid provider address: %w", err)
	}

	// Remove liquidity using secure implementation
	amountA, amountB, err := ms.Keeper.RemoveLiquidity(goCtx, provider, msg.PoolId, msg.Shares)
	if err != nil {
		return nil, fmt.Errorf("RemoveLiquidity: %w", err)
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
		return nil, fmt.Errorf("Swap: validate: %w", err)
	}

	// Check deadline - must be after current block time
	ctx := sdk.UnwrapSDKContext(goCtx)
	if ctx.BlockTime().Unix() > msg.Deadline {
		return nil, types.ErrDeadlineExceeded
	}

	// Parse trader address
	trader, err := sdk.AccAddressFromBech32(msg.Trader)
	if err != nil {
		return nil, fmt.Errorf("Swap: invalid trader address: %w", err)
	}

	// Execute swap using secure implementation
	amountOut, err := ms.Keeper.ExecuteSwap(goCtx, trader, msg.PoolId, msg.TokenIn, msg.TokenOut, msg.AmountIn, msg.MinAmountOut)
	if err != nil {
		return nil, fmt.Errorf("Swap: %w", err)
	}

	return &types.MsgSwapResponse{
		AmountOut: amountOut,
	}, nil
}

// CommitSwap handles swap commitment (phase 1 of commit-reveal MEV protection)
func (ms msgServer) CommitSwap(goCtx context.Context, msg *types.MsgCommitSwap) (*types.MsgCommitSwapResponse, error) {
	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("CommitSwap: validate: %w", err)
	}

	// Check if commit-reveal is enabled
	params, err := ms.Keeper.GetParams(goCtx)
	if err != nil {
		return nil, fmt.Errorf("CommitSwap: get params: %w", err)
	}

	if !params.EnableCommitReveal {
		return nil, types.ErrCommitRevealDisabled.Wrap("commit-reveal feature is disabled by governance")
	}

	// Parse trader address
	_ = msg.Trader // Validated by msg.ValidateBasic()

	ctx := sdk.UnwrapSDKContext(goCtx)

	// Store the commitment
	commit := types.SwapCommit{
		Trader:       msg.Trader,
		SwapHash:     msg.SwapHash,
		CommitHeight: ctx.BlockHeight(),
		ExpiryHeight: ctx.BlockHeight() + int64(params.CommitTimeoutBlocks),
	}

	if err := ms.Keeper.SetSwapCommit(goCtx, commit); err != nil {
		return nil, fmt.Errorf("CommitSwap: set commit: %w", err)
	}

	earliestReveal := ctx.BlockHeight() + int64(params.CommitRevealDelay)

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"swap_committed",
			sdk.NewAttribute("trader", msg.Trader),
			sdk.NewAttribute("swap_hash", msg.SwapHash),
			sdk.NewAttribute("commit_height", fmt.Sprintf("%d", ctx.BlockHeight())),
			sdk.NewAttribute("earliest_reveal_height", fmt.Sprintf("%d", earliestReveal)),
			sdk.NewAttribute("expiry_height", fmt.Sprintf("%d", commit.ExpiryHeight)),
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
		return nil, fmt.Errorf("RevealSwap: validate: %w", err)
	}

	// Check if commit-reveal is enabled
	params, err := ms.Keeper.GetParams(goCtx)
	if err != nil {
		return nil, fmt.Errorf("RevealSwap: get params: %w", err)
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
		return nil, fmt.Errorf("RevealSwap: invalid trader address: %w", err)
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
		return nil, fmt.Errorf("RevealSwap: delete commit: %w", err)
	}

	// Execute the swap using secure implementation
	amountOut, err := ms.Keeper.ExecuteSwap(goCtx, trader, msg.PoolId, msg.TokenIn, msg.TokenOut, msg.AmountIn, msg.MinAmountOut)
	if err != nil {
		return nil, fmt.Errorf("RevealSwap: execute swap: %w", err)
	}

	// Emit reveal event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"swap_revealed",
			sdk.NewAttribute("trader", msg.Trader),
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", ctx.HeaderInfo().Height)),
			sdk.NewAttribute("amount_in", msg.AmountIn.String()),
			sdk.NewAttribute("amount_out", amountOut.String()),
			sdk.NewAttribute("commit_height", fmt.Sprintf("%d", ctx.HeaderInfo().Height)),
			sdk.NewAttribute("reveal_height", fmt.Sprintf("%d", ctx.HeaderInfo().Height)),
		),
	)

	return &types.MsgRevealSwapResponse{
		AmountOut: amountOut,
	}, nil
}

// BatchSwap handles multiple swaps in a single atomic transaction
// AGENT-1: Enables batch swap operations for agents with reduced gas overhead
func (ms msgServer) BatchSwap(goCtx context.Context, msg *types.MsgBatchSwap) (*types.MsgBatchSwapResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate message
	if msg == nil {
		return nil, types.ErrInvalidInput.Wrap("nil message")
	}

	// Validate batch size (max 10 swaps per batch)
	const maxBatchSize = 10
	if len(msg.Swaps) == 0 {
		return nil, types.ErrInvalidInput.Wrap("empty swap batch")
	}
	if len(msg.Swaps) > maxBatchSize {
		return nil, types.ErrInvalidInput.Wrapf("batch size %d exceeds maximum %d", len(msg.Swaps), maxBatchSize)
	}

	// Parse trader address
	trader, err := sdk.AccAddressFromBech32(msg.Trader)
	if err != nil {
		return nil, types.ErrInvalidInput.Wrapf("invalid trader address: %v", err)
	}

	// Check deadline for all swaps
	if msg.Deadline > 0 && ctx.BlockTime().Unix() > msg.Deadline {
		return nil, types.ErrSwapExpired.Wrapf("deadline %d expired at block time %d", msg.Deadline, ctx.BlockTime().Unix())
	}

	results := make([]types.BatchSwapResult, 0, len(msg.Swaps))
	var successCount uint64

	// Execute each swap atomically - all must succeed or all fail
	for i, swapReq := range msg.Swaps {
		result := types.BatchSwapResult{
			PoolId:  swapReq.PoolId,
			Success: false,
		}

		// Execute the swap
		amountOut, swapErr := ms.Keeper.ExecuteSwap(
			goCtx,
			trader,
			swapReq.PoolId,
			swapReq.TokenIn,
			swapReq.TokenOut,
			swapReq.AmountIn,
			swapReq.MinAmountOut,
		)

		if swapErr != nil {
			// Return error - batch is atomic, all fail if one fails
			return nil, types.ErrSwapFailed.Wrapf("swap %d failed: %v", i, swapErr)
		}

		result.AmountOut = amountOut
		result.Success = true
		successCount++
		results = append(results, result)
	}

	// Emit batch swap event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"batch_swap",
			sdk.NewAttribute("trader", msg.Trader),
			sdk.NewAttribute("total_swaps", fmt.Sprintf("%d", len(msg.Swaps))),
			sdk.NewAttribute("successful_swaps", fmt.Sprintf("%d", successCount)),
		),
	)

	return &types.MsgBatchSwapResponse{
		Results:         results,
		TotalSwaps:      uint64(len(msg.Swaps)),
		SuccessfulSwaps: successCount,
	}, nil
}
