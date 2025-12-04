package ante

import (
	"fmt"

	"cosmossdk.io/math"
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
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid creator address: %s", err)
	}

	params, err := dd.keeper.GetParams(sdk.WrapSDKContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to get params: %w", err)
	}

	ctx.GasMeter().ConsumeGas(params.PoolCreationGas, "pool creation validation")

	if msg.TokenA == "" || msg.TokenB == "" {
		return sdkerrors.ErrInvalidRequest.Wrap("token identifiers cannot be empty")
	}

	if msg.TokenA == msg.TokenB {
		return sdkerrors.ErrInvalidRequest.Wrap("pool tokens must differ")
	}

	if msg.AmountA.IsZero() || msg.AmountB.IsZero() {
		return sdkerrors.ErrInvalidRequest.Wrap("initial liquidity amounts must be positive")
	}

	if msg.AmountA.LT(params.MinLiquidity) || msg.AmountB.LT(params.MinLiquidity) {
		return sdkerrors.ErrInvalidRequest.Wrapf("initial liquidity must be at least %s", params.MinLiquidity.String())
	}

	return nil
}

// validateSwap performs additional validation for swap operations
func (dd DEXDecorator) validateSwap(ctx sdk.Context, msg *dextypes.MsgSwap) error {
	if _, err := sdk.AccAddressFromBech32(msg.Trader); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid trader address: %s", err)
	}

	params, err := dd.keeper.GetParams(sdk.WrapSDKContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to get params: %w", err)
	}

	ctx.GasMeter().ConsumeGas(params.SwapValidationGas, "swap validation")

	if msg.TokenIn == "" || msg.TokenOut == "" {
		return sdkerrors.ErrInvalidRequest.Wrap("tokens cannot be empty")
	}

	if msg.AmountIn.IsZero() {
		return sdkerrors.ErrInvalidRequest.Wrap("swap amount must be positive")
	}

	if msg.MinAmountOut.IsNegative() {
		return sdkerrors.ErrInvalidRequest.Wrap("minimum amount out cannot be negative")
	}

	if _, err := dd.keeper.GetPool(sdk.WrapSDKContext(ctx), msg.PoolId); err != nil {
		return sdkerrors.ErrNotFound.Wrapf("pool %d not found", msg.PoolId)
	}

	return nil
}

// validateAddLiquidity performs additional validation for adding liquidity
func (dd DEXDecorator) validateAddLiquidity(ctx sdk.Context, msg *dextypes.MsgAddLiquidity) error {
	if _, err := sdk.AccAddressFromBech32(msg.Provider); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid provider address: %s", err)
	}

	params, err := dd.keeper.GetParams(sdk.WrapSDKContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to get params: %w", err)
	}

	ctx.GasMeter().ConsumeGas(params.LiquidityGas, "add liquidity validation")

	if msg.AmountA.IsZero() || msg.AmountB.IsZero() {
		return sdkerrors.ErrInvalidRequest.Wrap("liquidity amounts must be positive")
	}

	if _, err := dd.keeper.GetPool(sdk.WrapSDKContext(ctx), msg.PoolId); err != nil {
		return sdkerrors.ErrNotFound.Wrapf("pool %d not found", msg.PoolId)
	}

	return nil
}

// validateRemoveLiquidity performs additional validation for removing liquidity
func (dd DEXDecorator) validateRemoveLiquidity(ctx sdk.Context, msg *dextypes.MsgRemoveLiquidity) error {
	if _, err := sdk.AccAddressFromBech32(msg.Provider); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid provider address: %s", err)
	}

	params, err := dd.keeper.GetParams(sdk.WrapSDKContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to get params: %w", err)
	}

	ctx.GasMeter().ConsumeGas(params.PoolCreationGas, "remove liquidity validation")

	if msg.Shares.IsZero() || msg.Shares.Equal(math.ZeroInt()) {
		return sdkerrors.ErrInvalidRequest.Wrap("shares to remove must be positive")
	}

	if _, err := dd.keeper.GetPool(sdk.WrapSDKContext(ctx), msg.PoolId); err != nil {
		return sdkerrors.ErrNotFound.Wrapf("pool %d not found", msg.PoolId)
	}

	return nil
}
