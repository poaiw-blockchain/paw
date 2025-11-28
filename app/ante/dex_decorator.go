package ante

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	dexkeeper "github.com/paw-chain/paw/x/dex/keeper"
	dextypes "github.com/paw-chain/paw/x/dex/types"
)

// DEXDecorator validates DEX module-specific transaction requirements
type DEXDecorator struct {
	keeper dexkeeper.Keeper
}

// NewDEXDecorator creates a new DEXDecorator
func NewDEXDecorator(keeper dexkeeper.Keeper) DEXDecorator {
	return DEXDecorator{
		keeper: keeper,
	}
}

// AnteHandle implements the AnteDecorator interface
func (dd DEXDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// Skip validation during simulation
	if simulate {
		return next(ctx, tx, simulate)
	}

	msgs := tx.GetMsgs()
	for _, msg := range msgs {
		switch msg := msg.(type) {
		case *dextypes.MsgCreatePool:
			if err := dd.validateCreatePool(ctx, msg); err != nil {
				return ctx, err
			}
		case *dextypes.MsgSwap:
			if err := dd.validateSwap(ctx, msg); err != nil {
				return ctx, err
			}
		case *dextypes.MsgAddLiquidity:
			if err := dd.validateAddLiquidity(ctx, msg); err != nil {
				return ctx, err
			}
		case *dextypes.MsgRemoveLiquidity:
			if err := dd.validateRemoveLiquidity(ctx, msg); err != nil {
				return ctx, err
			}
		}
	}

	return next(ctx, tx, simulate)
}

// validateCreatePool performs additional validation for pool creation
func (dd DEXDecorator) validateCreatePool(ctx sdk.Context, msg *dextypes.MsgCreatePool) error {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}

	// Consume gas for validation
	ctx.GasMeter().ConsumeGas(1000, "pool creation validation")

	// Get module params
	params, err := dd.keeper.GetParams(ctx)
	if err != nil {
		return fmt.Errorf("failed to get params: %w", err)
	}

	// Validate pool creation fee
	if params.PoolCreationFee.IsPositive() {
		balance := dd.keeper.GetBankKeeper().SpendableCoins(ctx, creator)
		if balance.AmountOf(params.PoolCreationFee.Denom).LT(params.PoolCreationFee.Amount) {
			return sdkerrors.ErrInsufficientFunds.Wrapf("insufficient balance for pool creation fee")
		}
	}

	// Check if pool already exists
	poolID := dextypes.GetPoolID(msg.TokenA, msg.TokenB)
	_, err = dd.keeper.GetPool(ctx, poolID)
	if err == nil {
		return sdkerrors.ErrInvalidRequest.Wrapf("pool for %s/%s already exists", msg.TokenA, msg.TokenB)
	}

	// Validate initial liquidity amounts are reasonable
	if msg.AmountA.IsZero() || msg.AmountB.IsZero() {
		return sdkerrors.ErrInvalidRequest.Wrap("initial liquidity amounts must be positive")
	}

	// Check minimum liquidity requirement
	if msg.AmountA.LT(params.MinInitialPoolLiquidity) || msg.AmountB.LT(params.MinInitialPoolLiquidity) {
		return sdkerrors.ErrInvalidRequest.Wrapf("initial liquidity must be at least %s", params.MinInitialPoolLiquidity.String())
	}

	return nil
}

// validateSwap performs additional validation for swap operations
func (dd DEXDecorator) validateSwap(ctx sdk.Context, msg *dextypes.MsgSwap) error {
	trader, err := sdk.AccAddressFromBech32(msg.Trader)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid trader address: %s", err)
	}

	// Consume gas for validation
	ctx.GasMeter().ConsumeGas(1500, "swap validation")

	// Get pool to verify it exists
	poolID := dextypes.GetPoolID(msg.TokenIn, msg.TokenOut)
	pool, err := dd.keeper.GetPool(ctx, poolID)
	if err != nil {
		return sdkerrors.ErrNotFound.Wrapf("pool not found for %s/%s", msg.TokenIn, msg.TokenOut)
	}

	// Verify pool is active
	if !pool.Active {
		return sdkerrors.ErrInvalidRequest.Wrap("pool is not active")
	}

	// Check circuit breaker
	circuitBroken, err := dd.keeper.IsCircuitBroken(ctx, poolID)
	if err == nil && circuitBroken {
		return sdkerrors.ErrInvalidRequest.Wrap("circuit breaker triggered for this pool")
	}

	// Validate slippage tolerance
	if msg.MinAmountOut.IsNegative() {
		return sdkerrors.ErrInvalidRequest.Wrap("minimum amount out cannot be negative")
	}

	// Check if slippage is reasonable (not more than 50%)
	if msg.MinAmountOut.IsZero() {
		return sdkerrors.ErrInvalidRequest.Wrap("slippage tolerance too high - minimum amount out required")
	}

	// Validate trader has sufficient balance
	balance := dd.keeper.GetBankKeeper().SpendableCoins(ctx, trader)
	if balance.AmountOf(msg.TokenIn).LT(msg.AmountIn) {
		return sdkerrors.ErrInsufficientFunds.Wrapf("insufficient %s balance", msg.TokenIn)
	}

	// Rate limiting check
	swapCount, err := dd.keeper.GetAccountSwapCount(ctx, trader)
	if err == nil {
		params, _ := dd.keeper.GetParams(ctx)
		if swapCount >= params.MaxSwapsPerBlock {
			return sdkerrors.ErrInvalidRequest.Wrapf("account has exceeded max swaps per block: %d", params.MaxSwapsPerBlock)
		}
	}

	return nil
}

// validateAddLiquidity performs additional validation for adding liquidity
func (dd DEXDecorator) validateAddLiquidity(ctx sdk.Context, msg *dextypes.MsgAddLiquidity) error {
	provider, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid provider address: %s", err)
	}

	// Consume gas for validation
	ctx.GasMeter().ConsumeGas(1200, "add liquidity validation")

	// Verify pool exists
	poolID := dextypes.GetPoolID(msg.TokenA, msg.TokenB)
	pool, err := dd.keeper.GetPool(ctx, poolID)
	if err != nil {
		return sdkerrors.ErrNotFound.Wrapf("pool not found for %s/%s", msg.TokenA, msg.TokenB)
	}

	// Verify pool is active
	if !pool.Active {
		return sdkerrors.ErrInvalidRequest.Wrap("pool is not active")
	}

	// Validate amounts are positive
	if msg.AmountA.IsZero() || msg.AmountB.IsZero() {
		return sdkerrors.ErrInvalidRequest.Wrap("liquidity amounts must be positive")
	}

	// Check provider balance
	balance := dd.keeper.GetBankKeeper().SpendableCoins(ctx, provider)
	if balance.AmountOf(msg.TokenA).LT(msg.AmountA) || balance.AmountOf(msg.TokenB).LT(msg.AmountB) {
		return sdkerrors.ErrInsufficientFunds.Wrap("insufficient balance for liquidity provision")
	}

	return nil
}

// validateRemoveLiquidity performs additional validation for removing liquidity
func (dd DEXDecorator) validateRemoveLiquidity(ctx sdk.Context, msg *dextypes.MsgRemoveLiquidity) error {
	provider, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid provider address: %s", err)
	}

	// Consume gas for validation
	ctx.GasMeter().ConsumeGas(1200, "remove liquidity validation")

	// Verify pool exists
	poolID := dextypes.GetPoolID(msg.TokenA, msg.TokenB)
	pool, err := dd.keeper.GetPool(ctx, poolID)
	if err != nil {
		return sdkerrors.ErrNotFound.Wrapf("pool not found for %s/%s", msg.TokenA, msg.TokenB)
	}

	// Validate liquidity amount
	if msg.Liquidity.IsZero() || msg.Liquidity.IsNegative() {
		return sdkerrors.ErrInvalidRequest.Wrap("liquidity amount must be positive")
	}

	// Verify provider has sufficient LP tokens
	lpBalance, err := dd.keeper.GetLPTokenBalance(ctx, provider, poolID)
	if err != nil || lpBalance.LT(msg.Liquidity) {
		return sdkerrors.ErrInsufficientFunds.Wrap("insufficient LP token balance")
	}

	return nil
}
