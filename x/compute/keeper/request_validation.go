package keeper

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const requestPaymentDenom = "upaw"

// ValidateRequesterBalance ensures the requester can cover the declared max payment.
func (k Keeper) ValidateRequesterBalance(ctx sdk.Context, requester sdk.AccAddress, maxPayment sdkmath.Int) error {
	if !maxPayment.IsPositive() {
		return sdkerrors.ErrInvalidCoins.Wrap("max payment must be positive")
	}

	required := sdk.NewCoin(requestPaymentDenom, maxPayment)
	balance := k.bankKeeper.GetBalance(ctx, requester, requestPaymentDenom)
	if balance.Amount.LT(required.Amount) {
		return sdkerrors.ErrInsufficientFunds.Wrapf(
			"requester balance %s is less than required max payment %s", balance.String(), required.String(),
		)
	}

	return nil
}
