// Package keeper provides shared keeper interfaces and utilities for cross-module communication.
package keeper

import (
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

// ValidateAuthority checks that the provided authority matches the expected authority.
// This is used for governance-only operations like parameter updates.
//
// Parameters:
//   - expected: the expected authority address (typically from keeper.authority)
//   - actual: the actual authority address from the message (msg.Authority)
//
// Returns:
//   - error: govtypes.ErrInvalidSigner if authority mismatch, nil otherwise
//
// Usage example:
//
//	if err := keeper.ValidateAuthority(ms.authority, msg.Authority); err != nil {
//	    return nil, err
//	}
func ValidateAuthority(expected, actual string) error {
	if expected != actual {
		return govtypes.ErrInvalidSigner.Wrapf(
			"invalid authority; expected %s, got %s",
			expected,
			actual,
		)
	}
	return nil
}
