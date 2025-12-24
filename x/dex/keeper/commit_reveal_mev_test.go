package keeper_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// TestCommitRevealDisabledByDefault verifies commit-reveal is disabled in default genesis
func TestCommitRevealDisabledByDefault(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)
	params, err := k.GetParams(ctx)
	require.NoError(t, err)

	// Verify commit-reveal is disabled by default (testnet phase)
	require.False(t, params.EnableCommitReveal)
	require.Equal(t, uint64(10), params.CommitRevealDelay)
	require.Equal(t, uint64(100), params.CommitTimeoutBlocks)
}

// TestCommitSwapWhenDisabled verifies that commit fails when feature is disabled
func TestCommitSwapWhenDisabled(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)
	trader := types.TestAddr()

	// Create commit message
	msg := &types.MsgCommitSwap{
		Trader:   trader.String(),
		SwapHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}

	// Validate message
	require.NoError(t, msg.ValidateBasic())

	// Try to commit - should fail because feature is disabled
	msgServer := keeper.NewMsgServerImpl(*k)
	_, err := msgServer.CommitSwap(ctx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "commit-reveal feature is disabled")
}

// TestCommitRevealFullFlow tests the complete commit-reveal flow when enabled
func TestCommitRevealFullFlow(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)
	trader := types.TestAddr()

	// Enable commit-reveal
	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	params.EnableCommitReveal = true
	params.CommitRevealDelay = 2 // 2 blocks delay
	params.CommitTimeoutBlocks = 50
	require.NoError(t, k.SetParams(ctx, params))

	// Create a pool
	pool, err := k.CreatePool(ctx, trader, "upaw", "uatom",
		math.NewInt(1_000_000_000), math.NewInt(1_000_000_000))
	require.NoError(t, err)

	// Prepare swap parameters
	poolID := pool.Id
	tokenIn := "upaw"
	tokenOut := "uatom"
	amountIn := math.NewInt(10_000_000)
	minAmountOut := math.NewInt(9_500_000)
	deadline := ctx.BlockTime().Unix() + 300
	nonce := "test_nonce_123456"

	// Compute commitment hash
	h := sha256.New()
	h.Write([]byte(trader.String()))
	h.Write([]byte(fmt.Sprintf("%d", poolID)))
	h.Write([]byte(tokenIn))
	h.Write([]byte(tokenOut))
	h.Write([]byte(amountIn.String()))
	h.Write([]byte(minAmountOut.String()))
	h.Write([]byte(fmt.Sprintf("%d", deadline)))
	h.Write([]byte(nonce))
	swapHash := hex.EncodeToString(h.Sum(nil))

	// Phase 1: Commit
	commitMsg := &types.MsgCommitSwap{
		Trader:   trader.String(),
		SwapHash: swapHash,
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	commitHeight := sdkCtx.BlockHeight()

	msgServer := keeper.NewMsgServerImpl(*k)
	commitResp, err := msgServer.CommitSwap(ctx, commitMsg)
	require.NoError(t, err)
	require.Equal(t, commitHeight, commitResp.CommitHeight)
	require.Equal(t, commitHeight+2, commitResp.EarliestRevealHeight)
	require.Equal(t, commitHeight+50, commitResp.ExpiryHeight)

	// Verify commit is stored
	commit, err := k.GetSwapCommitByHash(ctx, swapHash)
	require.NoError(t, err)
	require.Equal(t, trader.String(), commit.Trader)
	require.Equal(t, swapHash, commit.SwapHash)

	// Try to reveal too early - should fail
	revealMsg := &types.MsgRevealSwap{
		Trader:       trader.String(),
		PoolId:       poolID,
		TokenIn:      tokenIn,
		TokenOut:     tokenOut,
		AmountIn:     amountIn,
		MinAmountOut: minAmountOut,
		Deadline:     deadline,
		Nonce:        nonce,
	}

	_, err = msgServer.RevealSwap(ctx, revealMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "reveal too early")

	// Advance blocks past reveal delay
	ctx = sdkCtx.WithBlockHeight(commitHeight + 3)

	// Phase 2: Reveal (should succeed now)
	revealResp, err := msgServer.RevealSwap(ctx, revealMsg)
	require.NoError(t, err)
	require.True(t, revealResp.AmountOut.GT(math.ZeroInt()))

	// Verify commit was deleted after reveal
	_, err = k.GetSwapCommitByHash(ctx, swapHash)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// TestCommitRevealHashMismatch verifies that reveal fails if hash doesn't match
func TestCommitRevealHashMismatch(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)
	trader := types.TestAddr()

	// Enable commit-reveal
	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	params.EnableCommitReveal = true
	require.NoError(t, k.SetParams(ctx, params))

	// Create commit with one hash
	commitMsg := &types.MsgCommitSwap{
		Trader:   trader.String(),
		SwapHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}

	msgServer := keeper.NewMsgServerImpl(*k)
	_, err = msgServer.CommitSwap(ctx, commitMsg)
	require.NoError(t, err)

	// Try to reveal with parameters that produce a different hash
	pool, err := k.CreatePool(ctx, trader, "upaw", "uatom",
		math.NewInt(1_000_000_000), math.NewInt(1_000_000_000))
	require.NoError(t, err)

	revealMsg := &types.MsgRevealSwap{
		Trader:       trader.String(),
		PoolId:       pool.Id,
		TokenIn:      "upaw",
		TokenOut:     "uatom",
		AmountIn:     math.NewInt(10_000_000),
		MinAmountOut: math.NewInt(9_500_000),
		Deadline:     ctx.BlockTime().Unix() + 300,
		Nonce:        "wrong_nonce",
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	ctx = sdkCtx.WithBlockHeight(sdkCtx.BlockHeight() + 10)

	_, err = msgServer.RevealSwap(ctx, revealMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// TestCommitRevealExpiry verifies that expired commits cannot be revealed
func TestCommitRevealExpiry(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)
	trader := types.TestAddr()

	// Enable commit-reveal with short timeout
	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	params.EnableCommitReveal = true
	params.CommitRevealDelay = 2
	params.CommitTimeoutBlocks = 10
	require.NoError(t, k.SetParams(ctx, params))

	// Create a pool
	pool, err := k.CreatePool(ctx, trader, "upaw", "uatom",
		math.NewInt(1_000_000_000), math.NewInt(1_000_000_000))
	require.NoError(t, err)

	// Prepare swap and commit
	poolID := pool.Id
	tokenIn := "upaw"
	tokenOut := "uatom"
	amountIn := math.NewInt(10_000_000)
	minAmountOut := math.NewInt(9_500_000)
	deadline := ctx.BlockTime().Unix() + 300
	nonce := "test_nonce_123456"

	h := sha256.New()
	h.Write([]byte(trader.String()))
	h.Write([]byte(fmt.Sprintf("%d", poolID)))
	h.Write([]byte(tokenIn))
	h.Write([]byte(tokenOut))
	h.Write([]byte(amountIn.String()))
	h.Write([]byte(minAmountOut.String()))
	h.Write([]byte(fmt.Sprintf("%d", deadline)))
	h.Write([]byte(nonce))
	swapHash := hex.EncodeToString(h.Sum(nil))

	commitMsg := &types.MsgCommitSwap{
		Trader:   trader.String(),
		SwapHash: swapHash,
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	commitHeight := sdkCtx.BlockHeight()

	msgServer := keeper.NewMsgServerImpl(*k)
	_, err = msgServer.CommitSwap(ctx, commitMsg)
	require.NoError(t, err)

	// Advance blocks past expiry
	ctx = sdkCtx.WithBlockHeight(commitHeight + 15)

	// Try to reveal - should fail due to expiry
	revealMsg := &types.MsgRevealSwap{
		Trader:       trader.String(),
		PoolId:       poolID,
		TokenIn:      tokenIn,
		TokenOut:     tokenOut,
		AmountIn:     amountIn,
		MinAmountOut: minAmountOut,
		Deadline:     deadline,
		Nonce:        nonce,
	}

	_, err = msgServer.RevealSwap(ctx, revealMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "expired")
}

// TestCleanupExpiredCommits verifies expired commits are cleaned up
func TestCleanupExpiredCommits(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)
	trader := types.TestAddr()

	// Enable commit-reveal with short timeout
	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	params.EnableCommitReveal = true
	params.CommitTimeoutBlocks = 5
	require.NoError(t, k.SetParams(ctx, params))

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	commitHeight := sdkCtx.BlockHeight()

	// Create commit
	commit := types.SwapCommit{
		Trader:       trader.String(),
		SwapHash:     "test_hash_123456789",
		CommitHeight: commitHeight,
		ExpiryHeight: commitHeight + 5,
	}

	require.NoError(t, k.SetSwapCommit(ctx, commit))

	// Verify commit exists
	_, err = k.GetSwapCommitByHash(ctx, commit.SwapHash)
	require.NoError(t, err)

	// Advance past expiry
	ctx = sdkCtx.WithBlockHeight(commitHeight + 10)

	// Run cleanup
	require.NoError(t, k.CleanupExpiredSwapCommits(ctx))

	// Verify commit was deleted
	_, err = k.GetSwapCommitByHash(ctx, commit.SwapHash)
	require.Error(t, err)
}

// TestMsgCommitSwapValidation tests message validation
func TestMsgCommitSwapValidation(t *testing.T) {
	t.Parallel()

	trader := types.TestAddr()

	tests := []struct {
		name    string
		msg     *types.MsgCommitSwap
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid commit",
			msg: &types.MsgCommitSwap{
				Trader:   trader.String(),
				SwapHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			},
			wantErr: false,
		},
		{
			name: "empty trader",
			msg: &types.MsgCommitSwap{
				Trader:   "",
				SwapHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			},
			wantErr: true,
			errMsg:  "trader address cannot be empty",
		},
		{
			name: "invalid trader",
			msg: &types.MsgCommitSwap{
				Trader:   "invalid",
				SwapHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			},
			wantErr: true,
			errMsg:  "invalid trader address",
		},
		{
			name: "empty swap hash",
			msg: &types.MsgCommitSwap{
				Trader:   trader.String(),
				SwapHash: "",
			},
			wantErr: true,
			errMsg:  "swap_hash cannot be empty",
		},
		{
			name: "invalid hash length",
			msg: &types.MsgCommitSwap{
				Trader:   trader.String(),
				SwapHash: "123abc",
			},
			wantErr: true,
			errMsg:  "swap_hash must be 64 hex characters",
		},
		{
			name: "invalid hex characters",
			msg: &types.MsgCommitSwap{
				Trader:   trader.String(),
				SwapHash: "xyz123456789abcdef0123456789abcdef0123456789abcdef0123456789abc",
			},
			wantErr: true,
			errMsg:  "must be valid hexadecimal",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.msg.ValidateBasic()
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					require.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestMsgRevealSwapValidation tests reveal message validation
func TestMsgRevealSwapValidation(t *testing.T) {
	t.Parallel()

	trader := types.TestAddr()

	tests := []struct {
		name    string
		msg     *types.MsgRevealSwap
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid reveal",
			msg: &types.MsgRevealSwap{
				Trader:       trader.String(),
				PoolId:       1,
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
				Deadline:     1234567890,
				Nonce:        "abcdefghijklmnop",
			},
			wantErr: false,
		},
		{
			name: "nonce too short",
			msg: &types.MsgRevealSwap{
				Trader:       trader.String(),
				PoolId:       1,
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
				Deadline:     1234567890,
				Nonce:        "short",
			},
			wantErr: true,
			errMsg:  "nonce must be at least 16 characters",
		},
		{
			name: "zero min amount out",
			msg: &types.MsgRevealSwap{
				Trader:       trader.String(),
				PoolId:       1,
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.ZeroInt(),
				Deadline:     1234567890,
				Nonce:        "abcdefghijklmnop",
			},
			wantErr: true,
			errMsg:  "min_amount_out must be positive",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.msg.ValidateBasic()
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					require.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
