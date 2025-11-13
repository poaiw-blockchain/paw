package types

import (
	"cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter store keys
var (
	KeyMinStake            = []byte("MinStake")
	KeyVerificationTimeout = []byte("VerificationTimeout")
	KeyMaxRetries          = []byte("MaxRetries")
)

// ParamKeyTable returns the param key table for the Compute module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		MinStake:            math.NewInt(10000000000), // 10,000 PAW
		VerificationTimeout: 300,                      // 5 minutes
		MaxRetries:          3,
	}
}

// ParamSetPairs implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMinStake, &p.MinStake, validateMinStake),
		paramtypes.NewParamSetPair(KeyVerificationTimeout, &p.VerificationTimeout, validateVerificationTimeout),
		paramtypes.NewParamSetPair(KeyMaxRetries, &p.MaxRetries, validateMaxRetries),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	// TODO: Implement comprehensive validation
	return nil
}

func validateMinStake(i interface{}) error {
	// TODO: Implement validation
	return nil
}

func validateVerificationTimeout(i interface{}) error {
	// TODO: Implement validation
	return nil
}

func validateMaxRetries(i interface{}) error {
	// TODO: Implement validation
	return nil
}
