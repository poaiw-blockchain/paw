package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter store keys
var (
	KeySwapFee              = []byte("SwapFee")
	KeyLPFee                = []byte("LPFee")
	KeyProtocolFee          = []byte("ProtocolFee")
	KeyMinLiquidity         = []byte("MinLiquidity")
	KeyMaxSlippagePercent   = []byte("MaxSlippagePercent")
)

// ParamKeyTable returns the param key table for the DEX module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		SwapFee:            sdk.NewDecWithPrec(3, 3),  // 0.3%
		LpFee:              sdk.NewDecWithPrec(25, 4), // 0.25%
		ProtocolFee:        sdk.NewDecWithPrec(5, 4),  // 0.05%
		MinLiquidity:       sdk.NewInt(1000),
		MaxSlippagePercent: sdk.NewDecWithPrec(1, 1),  // 10%
	}
}

// ParamSetPairs implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeySwapFee, &p.SwapFee, validateSwapFee),
		paramtypes.NewParamSetPair(KeyLPFee, &p.LpFee, validateLPFee),
		paramtypes.NewParamSetPair(KeyProtocolFee, &p.ProtocolFee, validateProtocolFee),
		paramtypes.NewParamSetPair(KeyMinLiquidity, &p.MinLiquidity, validateMinLiquidity),
		paramtypes.NewParamSetPair(KeyMaxSlippagePercent, &p.MaxSlippagePercent, validateMaxSlippagePercent),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	// TODO: Implement comprehensive validation
	return nil
}

func validateSwapFee(i interface{}) error {
	// TODO: Implement validation
	return nil
}

func validateLPFee(i interface{}) error {
	// TODO: Implement validation
	return nil
}

func validateProtocolFee(i interface{}) error {
	// TODO: Implement validation
	return nil
}

func validateMinLiquidity(i interface{}) error {
	// TODO: Implement validation
	return nil
}

func validateMaxSlippagePercent(i interface{}) error {
	// TODO: Implement validation
	return nil
}
