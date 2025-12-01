package simulation

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	computetypes "github.com/paw-chain/paw/x/compute/types"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
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
	GetValidator(ctx context.Context, addr sdk.ValAddress) (validator stakingtypes.ValidatorI, err error)
	GetAllValidators(ctx context.Context) ([]stakingtypes.ValidatorI, error)
	IterateBondedValidatorsByPower(ctx context.Context, fn func(index int64, validator stakingtypes.ValidatorI) (stop bool)) error
}

// DEXKeeper defines the expected DEX keeper interface for simulation
type DEXKeeper interface {
	GetAllPools(ctx context.Context) ([]dextypes.Pool, error)
	GetPool(ctx context.Context, poolID uint64) (*dextypes.Pool, error)
	GetParams(ctx context.Context) (dextypes.Params, error)
}

// OracleKeeper defines the expected Oracle keeper interface for simulation
type OracleKeeper interface {
	GetAllPrices(ctx context.Context) ([]oracletypes.Price, error)
	GetValidatorPrices(ctx context.Context, asset string) ([]oracletypes.ValidatorPrice, error)
	GetParams(ctx context.Context) (oracletypes.Params, error)
}

// ComputeKeeper defines the expected Compute keeper interface for simulation
type ComputeKeeper interface {
	GetAllRequests(ctx context.Context) ([]computetypes.Request, error)
	GetAllProviders(ctx context.Context) ([]computetypes.Provider, error)
	GetParams(ctx context.Context) (computetypes.Params, error)
}
