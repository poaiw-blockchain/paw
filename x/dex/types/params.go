package types

import (
	"cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter store keys
var (
	KeySwapFee            = []byte("SwapFee")
	KeyLPFee              = []byte("LPFee")
	KeyProtocolFee        = []byte("ProtocolFee")
	KeyMinLiquidity       = []byte("MinLiquidity")
	KeyMaxSlippagePercent = []byte("MaxSlippagePercent")
)

// ParamKeyTable returns the param key table for the DEX module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		SwapFee:            math.LegacyNewDecWithPrec(3, 3),  // 0.3%
		LpFee:              math.LegacyNewDecWithPrec(25, 4), // 0.25%
		ProtocolFee:        math.LegacyNewDecWithPrec(5, 4),  // 0.05%
		MinLiquidity:       math.NewInt(1000),
		MaxSlippagePercent: math.LegacyNewDecWithPrec(1, 1), // 10%
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
	if err := validateSwapFee(p.SwapFee); err != nil {
		return err
	}
	if err := validateLPFee(p.LpFee); err != nil {
		return err
	}
	if err := validateProtocolFee(p.ProtocolFee); err != nil {
		return err
	}
	if err := validateMinLiquidity(p.MinLiquidity); err != nil {
		return err
	}
	if err := validateMaxSlippagePercent(p.MaxSlippagePercent); err != nil {
		return err
	}

	// Validate that LP fee + Protocol fee equals swap fee
	totalFee := p.LpFee.Add(p.ProtocolFee)
	if !totalFee.Equal(p.SwapFee) {
		return ErrInvalidParams.Wrapf("lp_fee + protocol_fee must equal swap_fee: %s + %s != %s",
			p.LpFee.String(), p.ProtocolFee.String(), p.SwapFee.String())
	}

	return nil
}

func validateSwapFee(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return ErrInvalidParams.Wrapf("invalid parameter type: %T", i)
	}

	// Swap fee must be non-negative and less than 100%
	if v.IsNegative() {
		return ErrInvalidParams.Wrap("swap fee cannot be negative")
	}
	if v.GT(math.LegacyOneDec()) {
		return ErrInvalidParams.Wrap("swap fee cannot exceed 100%")
	}
	// Reasonable upper limit: 10% (0.1)
	if v.GT(math.LegacyNewDecWithPrec(1, 1)) {
		return ErrInvalidParams.Wrap("swap fee cannot exceed 10%")
	}

	return nil
}

func validateLPFee(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return ErrInvalidParams.Wrapf("invalid parameter type: %T", i)
	}

	// LP fee must be non-negative
	if v.IsNegative() {
		return ErrInvalidParams.Wrap("lp fee cannot be negative")
	}
	// LP fee should be less than 10% (reasonable upper bound)
	if v.GT(math.LegacyNewDecWithPrec(1, 1)) {
		return ErrInvalidParams.Wrap("lp fee cannot exceed 10%")
	}

	return nil
}

func validateProtocolFee(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return ErrInvalidParams.Wrapf("invalid parameter type: %T", i)
	}

	// Protocol fee must be non-negative
	if v.IsNegative() {
		return ErrInvalidParams.Wrap("protocol fee cannot be negative")
	}
	// Protocol fee should be less than 5% (reasonable upper bound)
	if v.GT(math.LegacyNewDecWithPrec(5, 2)) {
		return ErrInvalidParams.Wrap("protocol fee cannot exceed 5%")
	}

	return nil
}

func validateMinLiquidity(i interface{}) error {
	v, ok := i.(math.Int)
	if !ok {
		return ErrInvalidParams.Wrapf("invalid parameter type: %T", i)
	}

	// Min liquidity must be positive
	if v.IsNil() || v.IsNegative() || v.IsZero() {
		return ErrInvalidParams.Wrap("min liquidity must be positive")
	}

	// Reasonable upper limit: 1 million tokens
	maxLiquidity := math.NewInt(1_000_000_000_000) // 1 million with 6 decimals
	if v.GT(maxLiquidity) {
		return ErrInvalidParams.Wrap("min liquidity exceeds maximum allowed")
	}

	return nil
}

func validateMaxSlippagePercent(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return ErrInvalidParams.Wrapf("invalid parameter type: %T", i)
	}

	// Max slippage must be non-negative
	if v.IsNegative() {
		return ErrInvalidParams.Wrap("max slippage percent cannot be negative")
	}
	// Max slippage should not exceed 100%
	if v.GT(math.LegacyOneDec()) {
		return ErrInvalidParams.Wrap("max slippage percent cannot exceed 100%")
	}
	// Reasonable minimum: 0.1% (prevent MEV attacks with too-tight slippage)
	if v.LT(math.LegacyNewDecWithPrec(1, 3)) {
		return ErrInvalidParams.Wrap("max slippage percent must be at least 0.1%")
	}

	return nil
}
