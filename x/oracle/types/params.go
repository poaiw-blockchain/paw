package types

import (
	"fmt"

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
	if err := validateMinValidators(p.MinValidators); err != nil {
		return err
	}
	if err := validateUpdateInterval(p.UpdateInterval); err != nil {
		return err
	}
	if err := validateExpiryDuration(p.ExpiryDuration); err != nil {
		return err
	}

	// Ensure expiry duration is greater than update interval
	if p.ExpiryDuration <= p.UpdateInterval {
		return fmt.Errorf("expiry_duration (%d) must be greater than update_interval (%d)",
			p.ExpiryDuration, p.UpdateInterval)
	}

	return nil
}

func validateMinValidators(i interface{}) error {
	v, ok := i.(uint32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v < 1 {
		return fmt.Errorf("min_validators must be at least 1, got %d", v)
	}

	if v > 100 {
		return fmt.Errorf("min_validators cannot exceed 100, got %d", v)
	}

	return nil
}

func validateUpdateInterval(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v < 10 {
		return fmt.Errorf("update_interval must be at least 10 seconds, got %d", v)
	}

	if v > 3600 {
		return fmt.Errorf("update_interval cannot exceed 3600 seconds (1 hour), got %d", v)
	}

	return nil
}

func validateExpiryDuration(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v < 60 {
		return fmt.Errorf("expiry_duration must be at least 60 seconds, got %d", v)
	}

	if v > 7200 {
		return fmt.Errorf("expiry_duration cannot exceed 7200 seconds (2 hours), got %d", v)
	}

	return nil
}
