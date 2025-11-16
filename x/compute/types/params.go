package types

import (
	"fmt"

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
	if err := validateMinStake(p.MinStake); err != nil {
		return err
	}
	if err := validateVerificationTimeout(p.VerificationTimeout); err != nil {
		return err
	}
	if err := validateMaxRetries(p.MaxRetries); err != nil {
		return err
	}
	return nil
}

func validateMinStake(i interface{}) error {
	v, ok := i.(math.Int)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() || !v.IsPositive() {
		return fmt.Errorf("min_stake must be positive")
	}

	// Minimum stake should be at least 1000 PAW (1000000000 micro-PAW)
	minAllowed := math.NewInt(1000000000)
	if v.LT(minAllowed) {
		return fmt.Errorf("min_stake must be at least %s, got %s", minAllowed, v)
	}

	// Maximum stake shouldn't exceed 1M PAW (1000000000000 micro-PAW)
	maxAllowed := math.NewInt(1000000000000)
	if v.GT(maxAllowed) {
		return fmt.Errorf("min_stake cannot exceed %s, got %s", maxAllowed, v)
	}

	return nil
}

func validateVerificationTimeout(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v < 10 {
		return fmt.Errorf("verification_timeout must be at least 10 seconds, got %d", v)
	}

	if v > 3600 {
		return fmt.Errorf("verification_timeout cannot exceed 3600 seconds (1 hour), got %d", v)
	}

	return nil
}

func validateMaxRetries(i interface{}) error {
	v, ok := i.(uint32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v < 1 {
		return fmt.Errorf("max_retries must be at least 1, got %d", v)
	}

	if v > 10 {
		return fmt.Errorf("max_retries cannot exceed 10, got %d", v)
	}

	return nil
}
