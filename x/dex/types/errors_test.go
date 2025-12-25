package types

import (
	"errors"
	"testing"

	sdkerrors "cosmossdk.io/errors"
)

func TestErrorDefinitions(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantCode  uint32
		wantSpace string
	}{
		{"ErrInvalidPoolState", ErrInvalidPoolState, 2, ModuleName},
		{"ErrInsufficientLiquidity", ErrInsufficientLiquidity, 3, ModuleName},
		{"ErrInvalidInput", ErrInvalidInput, 4, ModuleName},
		{"ErrInvalidNonce", ErrInvalidNonce, 31, ModuleName},
		{"ErrInvalidAck", ErrInvalidAck, 91, ModuleName},
		{"ErrReentrancy", ErrReentrancy, 5, ModuleName},
		{"ErrInvariantViolation", ErrInvariantViolation, 6, ModuleName},
		{"ErrCircuitBreakerTriggered", ErrCircuitBreakerTriggered, 7, ModuleName},
		{"ErrSwapTooLarge", ErrSwapTooLarge, 8, ModuleName},
		{"ErrPriceImpactTooHigh", ErrPriceImpactTooHigh, 9, ModuleName},
		{"ErrFlashLoanDetected", ErrFlashLoanDetected, 10, ModuleName},
		{"ErrOverflow", ErrOverflow, 11, ModuleName},
		{"ErrUnderflow", ErrUnderflow, 12, ModuleName},
		{"ErrDivisionByZero", ErrDivisionByZero, 13, ModuleName},
		{"ErrPoolNotFound", ErrPoolNotFound, 14, ModuleName},
		{"ErrPoolAlreadyExists", ErrPoolAlreadyExists, 15, ModuleName},
		{"ErrRateLimitExceeded", ErrRateLimitExceeded, 19, ModuleName},
		{"ErrJITLiquidityDetected", ErrJITLiquidityDetected, 20, ModuleName},
		{"ErrInvalidState", ErrInvalidState, 21, ModuleName},
		{"ErrInsufficientShares", ErrInsufficientShares, 16, ModuleName},
		{"ErrSlippageTooHigh", ErrSlippageTooHigh, 17, ModuleName},
		{"ErrInvalidTokenPair", ErrInvalidTokenPair, 18, ModuleName},
		{"ErrMaxPoolsReached", ErrMaxPoolsReached, 28, ModuleName},
		{"ErrUnauthorized", ErrUnauthorized, 29, ModuleName},
		{"ErrUnauthorizedChannel", ErrUnauthorizedChannel, 92, ModuleName},
		{"ErrDeadlineExceeded", ErrDeadlineExceeded, 30, ModuleName},
		{"ErrInvalidSwapAmount", ErrInvalidSwapAmount, 22, ModuleName},
		{"ErrInvalidLiquidityAmount", ErrInvalidLiquidityAmount, 23, ModuleName},
		{"ErrStateCorruption", ErrStateCorruption, 24, ModuleName},
		{"ErrSlippageExceeded", ErrSlippageExceeded, 25, ModuleName},
		{"ErrOraclePrice", ErrOraclePrice, 26, ModuleName},
		{"ErrPriceDeviation", ErrPriceDeviation, 27, ModuleName},
		{"ErrOrderNotFound", ErrOrderNotFound, 32, ModuleName},
		{"ErrInvalidOrder", ErrInvalidOrder, 33, ModuleName},
		{"ErrOrderNotAuthorized", ErrOrderNotAuthorized, 34, ModuleName},
		{"ErrOrderNotCancellable", ErrOrderNotCancellable, 35, ModuleName},
		{"ErrCircuitBreakerAlreadyOpen", ErrCircuitBreakerAlreadyOpen, 36, ModuleName},
		{"ErrCircuitBreakerAlreadyClosed", ErrCircuitBreakerAlreadyClosed, 37, ModuleName},
		{"ErrCommitRequired", ErrCommitRequired, 40, ModuleName},
		{"ErrCommitmentNotFound", ErrCommitmentNotFound, 41, ModuleName},
		{"ErrDuplicateCommitment", ErrDuplicateCommitment, 42, ModuleName},
		{"ErrRevealTooEarly", ErrRevealTooEarly, 43, ModuleName},
		{"ErrCommitmentExpired", ErrCommitmentExpired, 44, ModuleName},
		{"ErrInsufficientDeposit", ErrInsufficientDeposit, 45, ModuleName},
		{"ErrInvalidPool", ErrInvalidPool, 46, ModuleName},
		{"ErrCommitRevealDisabled", ErrCommitRevealDisabled, 47, ModuleName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sdkErr *sdkerrors.Error
			if !errors.As(tt.err, &sdkErr) {
				t.Fatalf("Error is not an sdkerrors.Error")
			}

			if sdkErr.ABCICode() != tt.wantCode {
				t.Errorf("Expected code %d, got %d", tt.wantCode, sdkErr.ABCICode())
			}

			if sdkErr.Codespace() != tt.wantSpace {
				t.Errorf("Expected codespace %s, got %s", tt.wantSpace, sdkErr.Codespace())
			}

			// Verify error message is not empty
			if tt.err.Error() == "" {
				t.Error("Error message is empty")
			}
		})
	}
}

func TestErrorWithRecovery(t *testing.T) {
	baseErr := errors.New("base error")
	recovery := "Recovery suggestion"

	errWithRecovery := &ErrorWithRecovery{
		Err:      baseErr,
		Recovery: recovery,
	}

	// Test Error() method
	if errWithRecovery.Error() != baseErr.Error() {
		t.Errorf("Expected error message %q, got %q", baseErr.Error(), errWithRecovery.Error())
	}

	// Test Unwrap() method
	if errors.Unwrap(errWithRecovery) != baseErr {
		t.Error("Unwrap() did not return base error")
	}

	// Test recovery field
	if errWithRecovery.Recovery != recovery {
		t.Errorf("Expected recovery %q, got %q", recovery, errWithRecovery.Recovery)
	}
}

func TestWrapWithRecovery(t *testing.T) {
	tests := []struct {
		name             string
		baseErr          error
		msg              string
		args             []interface{}
		expectRecovery   bool
		expectedRecovery string
	}{
		{
			name:             "error with recovery suggestion",
			baseErr:          ErrInsufficientLiquidity,
			msg:              "pool %d has insufficient liquidity",
			args:             []interface{}{1},
			expectRecovery:   true,
			expectedRecovery: RecoverySuggestions[ErrInsufficientLiquidity],
		},
		{
			name:           "error without recovery suggestion",
			baseErr:        errors.New("unknown error"),
			msg:            "wrapped error",
			args:           []interface{}{},
			expectRecovery: false,
		},
		{
			name:             "reentrancy error with recovery",
			baseErr:          ErrReentrancy,
			msg:              "reentrancy detected in pool %d",
			args:             []interface{}{5},
			expectRecovery:   true,
			expectedRecovery: RecoverySuggestions[ErrReentrancy],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapWithRecovery(tt.baseErr, tt.msg, tt.args...)

			if wrapped == nil {
				t.Fatal("WrapWithRecovery returned nil")
			}

			if tt.expectRecovery {
				var errWithRecovery *ErrorWithRecovery
				if !errors.As(wrapped, &errWithRecovery) {
					t.Fatal("Expected ErrorWithRecovery, got different type")
				}

				if errWithRecovery.Recovery != tt.expectedRecovery {
					t.Errorf("Expected recovery %q, got %q", tt.expectedRecovery, errWithRecovery.Recovery)
				}
			} else {
				var errWithRecovery *ErrorWithRecovery
				if errors.As(wrapped, &errWithRecovery) {
					t.Error("Did not expect ErrorWithRecovery for unknown error")
				}
			}
		})
	}
}

func TestGetRecoverySuggestion(t *testing.T) {
	tests := []struct {
		name               string
		err                error
		expectedSuggestion string
	}{
		{
			name:               "insufficient liquidity",
			err:                ErrInsufficientLiquidity,
			expectedSuggestion: RecoverySuggestions[ErrInsufficientLiquidity],
		},
		{
			name:               "wrapped error",
			err:                sdkerrors.Wrap(ErrReentrancy, "wrapped"),
			expectedSuggestion: RecoverySuggestions[ErrReentrancy],
		},
		{
			name:               "double wrapped error",
			err:                sdkerrors.Wrap(sdkerrors.Wrap(ErrInvariantViolation, "wrap1"), "wrap2"),
			expectedSuggestion: RecoverySuggestions[ErrInvariantViolation],
		},
		{
			name:               "unknown error",
			err:                errors.New("unknown"),
			expectedSuggestion: "No recovery suggestion available. Check error message for details.",
		},
		{
			name:               "circuit breaker triggered",
			err:                ErrCircuitBreakerTriggered,
			expectedSuggestion: RecoverySuggestions[ErrCircuitBreakerTriggered],
		},
		{
			name:               "commit required",
			err:                ErrCommitRequired,
			expectedSuggestion: RecoverySuggestions[ErrCommitRequired],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := GetRecoverySuggestion(tt.err)

			if suggestion != tt.expectedSuggestion {
				t.Errorf("Expected suggestion %q, got %q", tt.expectedSuggestion, suggestion)
			}
		})
	}
}

func TestRecoverySuggestions_Coverage(t *testing.T) {
	// Verify all critical errors have recovery suggestions
	criticalErrors := []error{
		ErrInvalidPoolState,
		ErrInsufficientLiquidity,
		ErrInvalidInput,
		ErrReentrancy,
		ErrInvariantViolation,
		ErrCircuitBreakerTriggered,
		ErrSwapTooLarge,
		ErrPriceImpactTooHigh,
		ErrFlashLoanDetected,
		ErrOverflow,
		ErrUnderflow,
		ErrDivisionByZero,
		ErrPoolNotFound,
		ErrPoolAlreadyExists,
		ErrInsufficientShares,
		ErrSlippageTooHigh,
		ErrInvalidTokenPair,
		ErrMaxPoolsReached,
		ErrUnauthorized,
		ErrUnauthorizedChannel,
		ErrDeadlineExceeded,
		ErrInvalidSwapAmount,
		ErrInvalidLiquidityAmount,
		ErrStateCorruption,
		ErrSlippageExceeded,
		ErrOraclePrice,
		ErrPriceDeviation,
		ErrCommitRequired,
		ErrCommitmentNotFound,
		ErrDuplicateCommitment,
		ErrRevealTooEarly,
		ErrCommitmentExpired,
		ErrInsufficientDeposit,
	}

	for _, err := range criticalErrors {
		t.Run(err.Error(), func(t *testing.T) {
			suggestion, exists := RecoverySuggestions[err]
			if !exists {
				t.Errorf("No recovery suggestion for critical error: %v", err)
			}
			if suggestion == "" {
				t.Errorf("Empty recovery suggestion for error: %v", err)
			}
		})
	}
}

func TestRecoverySuggestions_Quality(t *testing.T) {
	// Verify recovery suggestions are meaningful
	for err, suggestion := range RecoverySuggestions {
		t.Run(err.Error(), func(t *testing.T) {
			if len(suggestion) < 20 {
				t.Errorf("Recovery suggestion too short (< 20 chars): %q", suggestion)
			}

			// Should contain actionable words
			actionable := false
			keywords := []string{
				"check", "verify", "ensure", "contact", "reduce", "increase",
				"wait", "retry", "query", "review", "split", "use",
			}

			lowerSuggestion := suggestion
			for _, keyword := range keywords {
				if contains(lowerSuggestion, keyword) || contains(lowerSuggestion, keyword[:1]+keyword[1:]) {
					actionable = true
					break
				}
			}

			if !actionable {
				t.Logf("Warning: Suggestion may not be actionable: %q", suggestion)
			}
		})
	}
}

func TestErrorUnwrapping(t *testing.T) {
	// Test deep unwrapping chain
	base := ErrInsufficientLiquidity
	wrapped1 := sdkerrors.Wrap(base, "level 1")
	wrapped2 := sdkerrors.Wrap(wrapped1, "level 2")
	wrapped3 := sdkerrors.Wrap(wrapped2, "level 3")

	suggestion := GetRecoverySuggestion(wrapped3)
	expectedSuggestion := RecoverySuggestions[ErrInsufficientLiquidity]

	if suggestion != expectedSuggestion {
		t.Errorf("Failed to unwrap deeply nested error. Expected %q, got %q", expectedSuggestion, suggestion)
	}
}

func TestErrorWithRecovery_Integration(t *testing.T) {
	// Test the full workflow
	baseErr := ErrInsufficientLiquidity
	wrapped := WrapWithRecovery(baseErr, "pool %d has insufficient liquidity", 1)

	// Verify it's an ErrorWithRecovery
	var errWithRecovery *ErrorWithRecovery
	if !errors.As(wrapped, &errWithRecovery) {
		t.Fatal("Expected ErrorWithRecovery")
	}

	// Verify we can get the recovery suggestion
	suggestion := GetRecoverySuggestion(wrapped)
	if suggestion != RecoverySuggestions[ErrInsufficientLiquidity] {
		t.Errorf("Unexpected recovery suggestion: %q", suggestion)
	}

	// Verify error message includes wrapper
	if !contains(errWithRecovery.Error(), "pool") {
		t.Error("Error message does not include wrapper text")
	}

	// Verify we can unwrap to base error
	unwrapped := errors.Unwrap(errWithRecovery)
	if unwrapped == nil {
		t.Fatal("Failed to unwrap error")
	}
}
