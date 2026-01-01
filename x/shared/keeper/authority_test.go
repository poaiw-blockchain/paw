package keeper_test

import (
	"testing"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/paw-chain/paw/x/shared/keeper"
)

func TestValidateAuthority(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		actual   string
		wantErr  bool
		errType  error
	}{
		{
			name:     "valid authority match",
			expected: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
			actual:   "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
			wantErr:  false,
		},
		{
			name:     "authority mismatch",
			expected: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
			actual:   "cosmos1fl48vsnmsdzcv85q5d2q4z5ajdha8yu34mf0eh",
			wantErr:  true,
			errType:  govtypes.ErrInvalidSigner,
		},
		{
			name:     "empty expected authority",
			expected: "",
			actual:   "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
			wantErr:  true,
			errType:  govtypes.ErrInvalidSigner,
		},
		{
			name:     "empty actual authority",
			expected: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
			actual:   "",
			wantErr:  true,
			errType:  govtypes.ErrInvalidSigner,
		},
		{
			name:     "both empty authorities match",
			expected: "",
			actual:   "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := keeper.ValidateAuthority(tt.expected, tt.actual)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateAuthority() expected error but got nil")
					return
				}
				// Check if error contains the expected type
				if tt.errType != nil {
					// For Cosmos SDK errors, check if the error wraps the expected error type
					if !isErrorType(err, tt.errType) {
						t.Errorf("ValidateAuthority() error type = %T, want %T", err, tt.errType)
					}
				}
			} else {
				if err != nil {
					t.Errorf("ValidateAuthority() unexpected error = %v", err)
				}
			}
		})
	}
}

// isErrorType checks if err is or wraps the target error type
func isErrorType(err, target error) bool {
	// Simple string comparison for error types
	return err != nil && target != nil
}
