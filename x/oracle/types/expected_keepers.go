package types

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// StakingKeeper defines the expected interface for the Staking module.
type StakingKeeper interface {
	GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error)
	IterateBondedValidatorsByPower(ctx context.Context, fn func(index int64, validator stakingtypes.ValidatorI) (stop bool)) error
	PowerReduction(ctx context.Context) math.Int
}

// SlashingKeeper defines the expected interface for the Slashing module.
type SlashingKeeper interface {
	Slash(ctx context.Context, consAddr sdk.ConsAddress, slashFactor math.LegacyDec, infractionHeight, power int64) error
	SlashWithInfractionReason(ctx context.Context, consAddr sdk.ConsAddress, slashFactor math.LegacyDec, infractionHeight, power int64, infraction stakingtypes.Infraction) error
}
