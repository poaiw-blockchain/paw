package types

import "fmt"

// Validate validates that a PriceFeed is well-formed
func (pf PriceFeed) Validate() error {
	if pf.Asset == "" {
		return fmt.Errorf("asset symbol cannot be empty")
	}
	if pf.Price.IsNil() || pf.Price.IsNegative() || pf.Price.IsZero() {
		return fmt.Errorf("price must be positive")
	}
	if pf.Timestamp <= 0 {
		return fmt.Errorf("timestamp must be positive")
	}
	if len(pf.Validators) == 0 {
		return fmt.Errorf("must have at least one validator")
	}
	return nil
}
