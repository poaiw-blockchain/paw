package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter store keys
var (
	KeyMinValidators  = []byte("MinValidators")
	KeyUpdateInterval = []byte("UpdateInterval")
	KeyExpiryDuration = []byte("ExpiryDuration")
)

// ParamKeyTable returns the param key table for the Oracle module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return Params{
		MinValidators:  3,
		UpdateInterval: 60,  // 1 minute
		ExpiryDuration: 300, // 5 minutes
	}
}

// ParamSetPairs implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMinValidators, &p.MinValidators, validateMinValidators),
		paramtypes.NewParamSetPair(KeyUpdateInterval, &p.UpdateInterval, validateUpdateInterval),
		paramtypes.NewParamSetPair(KeyExpiryDuration, &p.ExpiryDuration, validateExpiryDuration),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	// TODO: Implement comprehensive validation
	return nil
}

func validateMinValidators(i interface{}) error {
	// TODO: Implement validation
	return nil
}

func validateUpdateInterval(i interface{}) error {
	// TODO: Implement validation
	return nil
}

func validateExpiryDuration(i interface{}) error {
	// TODO: Implement validation
	return nil
}
