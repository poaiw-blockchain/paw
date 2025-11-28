package simulation

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
	computetypes "github.com/paw-chain/paw/x/compute/types"
)

// AccountKeeper defines the expected account keeper interface
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	GetModuleAddress(moduleName string) sdk.AccAddress
}

// BankKeeper defines the expected bank keeper interface
type BankKeeper interface {
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, from sdk.AccAddress, to string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, from string, to sdk.AccAddress, amt sdk.Coins) error
}

// StakingKeeper defines the expected staking keeper interface
type StakingKeeper interface {
	GetValidator(ctx context.Context, addr sdk.ValAddress) (validator sdk.ValidatorI, err error)
	GetAllValidators(ctx context.Context) (validators []sdk.ValidatorI, err error)
	IterateBondedValidatorsByPower(ctx context.Context, fn func(index int64, validator sdk.ValidatorI) (stop bool)) error
}

// DEXKeeper defines the expected DEX keeper interface for simulation
type DEXKeeper interface {
	GetAllPools(ctx sdk.Context) []dextypes.Pool
	GetPool(ctx sdk.Context, poolId uint64) (dextypes.Pool, bool)
	GetParams(ctx sdk.Context) dextypes.Params
}

// OracleKeeper defines the expected Oracle keeper interface for simulation
type OracleKeeper interface {
	GetAllPrices(ctx sdk.Context) []oracletypes.Price
	GetValidatorPrices(ctx sdk.Context, asset string) []oracletypes.ValidatorPrice
	GetParams(ctx sdk.Context) oracletypes.Params
}

// ComputeKeeper defines the expected Compute keeper interface for simulation
type ComputeKeeper interface {
	GetAllRequests(ctx sdk.Context) []computetypes.Request
	GetAllProviders(ctx sdk.Context) []computetypes.Provider
	GetParams(ctx sdk.Context) computetypes.Params
}
