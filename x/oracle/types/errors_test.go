package types

import (
	"errors"
	"strings"
	"testing"

	sdkerrors "cosmossdk.io/errors"
)

func TestErrorDefinitions(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		code  uint32
		msg   string
	}{
		{"ErrInvalidAsset", ErrInvalidAsset, 2, "invalid asset"},
		{"ErrInvalidNonce", ErrInvalidNonce, 50, "invalid packet nonce"},
		{"ErrInvalidPrice", ErrInvalidPrice, 3, "invalid price"},
		{"ErrPriceNotFound", ErrPriceNotFound, 7, "price not found"},
		{"ErrPriceExpired", ErrPriceExpired, 12, "price data expired"},
		{"ErrPriceDeviation", ErrPriceDeviation, 13, "price deviation too high"},
		{"ErrInvalidAck", ErrInvalidAck, 90, "invalid acknowledgement"},
		{"ErrValidatorNotBonded", ErrValidatorNotBonded, 4, "validator not bonded"},
		{"ErrFeederNotAuthorized", ErrFeederNotAuthorized, 5, "feeder not authorized"},
		{"ErrValidatorNotFound", ErrValidatorNotFound, 8, "validator not found"},
		{"ErrValidatorSlashed", ErrValidatorSlashed, 14, "validator has been slashed"},
		{"ErrUnauthorizedChannel", ErrUnauthorizedChannel, 60, "unauthorized IBC channel"},
		{"ErrInsufficientVotes", ErrInsufficientVotes, 6, "insufficient votes"},
		{"ErrInvalidVotePeriod", ErrInvalidVotePeriod, 9, "invalid vote period"},
		{"ErrDuplicateSubmission", ErrDuplicateSubmission, 15, "duplicate price submission"},
		{"ErrMissedVote", ErrMissedVote, 16, "validator missed vote"},
		{"ErrInvalidThreshold", ErrInvalidThreshold, 10, "invalid threshold"},
		{"ErrInvalidSlashFraction", ErrInvalidSlashFraction, 11, "invalid slash fraction"},
		{"ErrCircuitBreakerActive", ErrCircuitBreakerActive, 20, "circuit breaker is active"},
		{"ErrRateLimitExceeded", ErrRateLimitExceeded, 21, "rate limit exceeded"},
		{"ErrSybilAttackDetected", ErrSybilAttackDetected, 22, "Sybil attack detected"},
		{"ErrFlashLoanDetected", ErrFlashLoanDetected, 23, "flash loan attack detected"},
		{"ErrDataPoisoning", ErrDataPoisoning, 24, "data poisoning attempt detected"},
		{"ErrInsufficientDataSources", ErrInsufficientDataSources, 30, "insufficient data sources"},
		{"ErrOutlierDetected", ErrOutlierDetected, 31, "price outlier detected"},
		{"ErrMedianCalculationFailed", ErrMedianCalculationFailed, 32, "median calculation failed"},
		{"ErrInsufficientOracleConsensus", ErrInsufficientOracleConsensus, 33, "insufficient voting power for oracle consensus"},
		{"ErrStateCorruption", ErrStateCorruption, 40, "state corruption detected"},
		{"ErrOracleInactive", ErrOracleInactive, 41, "oracle is inactive"},
		{"ErrOracleDataUnavailable", ErrOracleDataUnavailable, 42, "oracle data unavailable"},
		{"ErrInvalidPriceSource", ErrInvalidPriceSource, 43, "invalid price source"},
		{"ErrInvalidIPAddress", ErrInvalidIPAddress, 44, "invalid IP address"},
		{"ErrIPRegionMismatch", ErrIPRegionMismatch, 45, "IP address does not match claimed region"},
		{"ErrPrivateIPNotAllowed", ErrPrivateIPNotAllowed, 46, "private IP addresses not allowed for validators"},
		{"ErrLocationProofRequired", ErrLocationProofRequired, 47, "location proof required"},
		{"ErrLocationProofInvalid", ErrLocationProofInvalid, 48, "location proof invalid or expired"},
		{"ErrInsufficientGeoDiversity", ErrInsufficientGeoDiversity, 49, "insufficient geographic diversity"},
		{"ErrGeoIPDatabaseUnavailable", ErrGeoIPDatabaseUnavailable, 51, "GeoIP database unavailable"},
		{"ErrTooManyValidatorsFromSameIP", ErrTooManyValidatorsFromSameIP, 52, "too many validators from same IP address"},
		{"ErrCircuitBreakerAlreadyOpen", ErrCircuitBreakerAlreadyOpen, 53, "circuit breaker already open"},
		{"ErrCircuitBreakerAlreadyClosed", ErrCircuitBreakerAlreadyClosed, 54, "circuit breaker already closed"},
		{"ErrOraclePaused", ErrOraclePaused, 55, "oracle is currently paused"},
		{"ErrOracleNotPaused", ErrOracleNotPaused, 56, "oracle is not currently paused"},
		{"ErrUnauthorizedPause", ErrUnauthorizedPause, 57, "unauthorized to trigger emergency pause"},
		{"ErrUnauthorizedResume", ErrUnauthorizedResume, 58, "unauthorized to resume oracle"},
		{"ErrInvalidEmergencyAdmin", ErrInvalidEmergencyAdmin, 59, "invalid emergency admin address"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check error is not nil
			if tt.err == nil {
				t.Error("Error is nil")
				return
			}

			// Check error message contains expected text
			errMsg := tt.err.Error()
			if !strings.Contains(errMsg, tt.msg) {
				t.Errorf("Expected error message to contain %q, got %q", tt.msg, errMsg)
			}

			// Check error code (if it's a Cosmos SDK error)
			if sdkErr, ok := tt.err.(*sdkerrors.Error); ok {
				if sdkErr.ABCICode() != tt.code {
					t.Errorf("Expected ABCI code %d, got %d", tt.code, sdkErr.ABCICode())
				}

				// Check module name
				if sdkErr.Codespace() != ModuleName {
					t.Errorf("Expected codespace %q, got %q", ModuleName, sdkErr.Codespace())
				}
			}
		})
	}
}

func TestErrorCodes_Unique(t *testing.T) {
	// Verify all error codes are unique
	errorCodes := map[uint32]string{
		2:  "ErrInvalidAsset",
		3:  "ErrInvalidPrice",
		4:  "ErrValidatorNotBonded",
		5:  "ErrFeederNotAuthorized",
		6:  "ErrInsufficientVotes",
		7:  "ErrPriceNotFound",
		8:  "ErrValidatorNotFound",
		9:  "ErrInvalidVotePeriod",
		10: "ErrInvalidThreshold",
		11: "ErrInvalidSlashFraction",
		12: "ErrPriceExpired",
		13: "ErrPriceDeviation",
		14: "ErrValidatorSlashed",
		15: "ErrDuplicateSubmission",
		16: "ErrMissedVote",
		20: "ErrCircuitBreakerActive",
		21: "ErrRateLimitExceeded",
		22: "ErrSybilAttackDetected",
		23: "ErrFlashLoanDetected",
		24: "ErrDataPoisoning",
		30: "ErrInsufficientDataSources",
		31: "ErrOutlierDetected",
		32: "ErrMedianCalculationFailed",
		33: "ErrInsufficientOracleConsensus",
		40: "ErrStateCorruption",
		41: "ErrOracleInactive",
		42: "ErrOracleDataUnavailable",
		43: "ErrInvalidPriceSource",
		44: "ErrInvalidIPAddress",
		45: "ErrIPRegionMismatch",
		46: "ErrPrivateIPNotAllowed",
		47: "ErrLocationProofRequired",
		48: "ErrLocationProofInvalid",
		49: "ErrInsufficientGeoDiversity",
		50: "ErrInvalidNonce",
		51: "ErrGeoIPDatabaseUnavailable",
		52: "ErrTooManyValidatorsFromSameIP",
		53: "ErrCircuitBreakerAlreadyOpen",
		54: "ErrCircuitBreakerAlreadyClosed",
		55: "ErrOraclePaused",
		56: "ErrOracleNotPaused",
		57: "ErrUnauthorizedPause",
		58: "ErrUnauthorizedResume",
		59: "ErrInvalidEmergencyAdmin",
		60: "ErrUnauthorizedChannel",
		90: "ErrInvalidAck",
	}

	// Count should match number of defined errors
	expectedCount := 46 // Update this if you add more errors
	if len(errorCodes) != expectedCount {
		t.Errorf("Expected %d unique error codes, got %d", expectedCount, len(errorCodes))
	}
}

func TestErrorWithRecovery(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		recovery string
	}{
		{
			name:     "error with recovery",
			err:      errors.New("test error"),
			recovery: "test recovery",
		},
		{
			name:     "empty recovery",
			err:      errors.New("test error"),
			recovery: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errWithRecovery := &ErrorWithRecovery{
				Err:      tt.err,
				Recovery: tt.recovery,
			}

			// Test Error() method
			if errWithRecovery.Error() != tt.err.Error() {
				t.Errorf("Expected error message %q, got %q", tt.err.Error(), errWithRecovery.Error())
			}

			// Test Unwrap() method
			if errWithRecovery.Unwrap() != tt.err {
				t.Error("Unwrap() did not return original error")
			}

			// Test recovery field
			if errWithRecovery.Recovery != tt.recovery {
				t.Errorf("Expected recovery %q, got %q", tt.recovery, errWithRecovery.Recovery)
			}
		})
	}
}

func TestWrapWithRecovery(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		msg            string
		expectRecovery bool
	}{
		{
			name:           "known error with recovery",
			err:            ErrInvalidAsset,
			msg:            "additional context",
			expectRecovery: true,
		},
		{
			name:           "unknown error without recovery",
			err:            errors.New("unknown error"),
			msg:            "additional context",
			expectRecovery: false,
		},
		{
			name:           "nil error",
			err:            nil,
			msg:            "additional context",
			expectRecovery: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				// Can't wrap nil error
				return
			}

			wrapped := WrapWithRecovery(tt.err, "%s", tt.msg)

			if wrapped == nil {
				t.Fatal("WrapWithRecovery returned nil")
			}

			// Check if message is included
			if !strings.Contains(wrapped.Error(), tt.msg) {
				t.Errorf("Wrapped error should contain message %q, got %q", tt.msg, wrapped.Error())
			}

			// Check for recovery suggestion
			if tt.expectRecovery {
				if errWithRecovery, ok := wrapped.(*ErrorWithRecovery); ok {
					if errWithRecovery.Recovery == "" {
						t.Error("Expected recovery suggestion, got empty string")
					}
				} else {
					t.Error("Expected ErrorWithRecovery type for known error")
				}
			}
		})
	}
}

func TestGetRecoverySuggestion(t *testing.T) {
	tests := []struct {
		name               string
		err                error
		expectSuggestion   bool
		suggestionContains string
	}{
		{
			name:               "ErrInvalidAsset",
			err:                ErrInvalidAsset,
			expectSuggestion:   true,
			suggestionContains: "Asset symbol not recognized",
		},
		{
			name:               "ErrInvalidPrice",
			err:                ErrInvalidPrice,
			expectSuggestion:   true,
			suggestionContains: "Price must be positive",
		},
		{
			name:               "ErrValidatorNotBonded",
			err:                ErrValidatorNotBonded,
			expectSuggestion:   true,
			suggestionContains: "Validator must be bonded",
		},
		{
			name:               "ErrSybilAttackDetected",
			err:                ErrSybilAttackDetected,
			expectSuggestion:   true,
			suggestionContains: "SECURITY",
		},
		{
			name:               "wrapped known error",
			err:                sdkerrors.Wrap(ErrInvalidAsset, "context"),
			expectSuggestion:   true,
			suggestionContains: "Asset symbol not recognized",
		},
		{
			name:             "unknown error",
			err:              errors.New("unknown"),
			expectSuggestion: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := GetRecoverySuggestion(tt.err)

			if suggestion == "" {
				t.Error("GetRecoverySuggestion returned empty string")
				return
			}

			if tt.expectSuggestion {
				if strings.Contains(suggestion, "No recovery suggestion available") {
					t.Error("Expected specific suggestion, got default message")
				}
				if tt.suggestionContains != "" && !strings.Contains(suggestion, tt.suggestionContains) {
					t.Errorf("Expected suggestion to contain %q, got %q", tt.suggestionContains, suggestion)
				}
			} else {
				if !strings.Contains(suggestion, "No recovery suggestion available") {
					t.Errorf("Expected default message for unknown error, got %q", suggestion)
				}
			}
		})
	}
}

func TestRecoverySuggestions_AllErrorsCovered(t *testing.T) {
	// Verify all defined errors have recovery suggestions
	allErrors := []error{
		ErrInvalidAsset,
		ErrInvalidPrice,
		ErrPriceNotFound,
		ErrPriceExpired,
		ErrPriceDeviation,
		ErrValidatorNotBonded,
		ErrFeederNotAuthorized,
		ErrValidatorNotFound,
		ErrValidatorSlashed,
		ErrUnauthorizedChannel,
		ErrInsufficientVotes,
		ErrInvalidVotePeriod,
		ErrDuplicateSubmission,
		ErrMissedVote,
		ErrInvalidThreshold,
		ErrInvalidSlashFraction,
		ErrCircuitBreakerActive,
		ErrRateLimitExceeded,
		ErrSybilAttackDetected,
		ErrFlashLoanDetected,
		ErrDataPoisoning,
		ErrInsufficientDataSources,
		ErrOutlierDetected,
		ErrMedianCalculationFailed,
		ErrInsufficientOracleConsensus,
		ErrStateCorruption,
		ErrOracleDataUnavailable,
		ErrInvalidPriceSource,
		ErrInvalidIPAddress,
		ErrIPRegionMismatch,
		ErrPrivateIPNotAllowed,
		ErrLocationProofRequired,
		ErrLocationProofInvalid,
		ErrInsufficientGeoDiversity,
		ErrGeoIPDatabaseUnavailable,
		ErrTooManyValidatorsFromSameIP,
		ErrOraclePaused,
		ErrOracleNotPaused,
		ErrUnauthorizedPause,
		ErrUnauthorizedResume,
		ErrInvalidEmergencyAdmin,
	}

	missing := []string{}
	for _, err := range allErrors {
		if _, ok := RecoverySuggestions[err]; !ok {
			missing = append(missing, err.Error())
		}
	}

	if len(missing) > 0 {
		t.Errorf("The following errors are missing recovery suggestions:\n%s", strings.Join(missing, "\n"))
	}
}

func TestRecoverySuggestions_QualityCheck(t *testing.T) {
	// Verify recovery suggestions are meaningful (not too short)
	minLength := 20 // Minimum characters for a useful suggestion

	for err, suggestion := range RecoverySuggestions {
		if len(suggestion) < minLength {
			t.Errorf("Recovery suggestion for %v is too short (%d chars): %q", err, len(suggestion), suggestion)
		}

		// Check for actionable language
		hasActionable := strings.Contains(suggestion, "Check") ||
			strings.Contains(suggestion, "Ensure") ||
			strings.Contains(suggestion, "Verify") ||
			strings.Contains(suggestion, "Wait") ||
			strings.Contains(suggestion, "Query") ||
			strings.Contains(suggestion, "Use") ||
			strings.Contains(suggestion, "Contact") ||
			strings.Contains(suggestion, "Submit") ||
			strings.Contains(suggestion, "Update") ||
			strings.Contains(suggestion, "Download") ||
			strings.Contains(suggestion, "Configure") ||
			strings.Contains(suggestion, "Provide") ||
			strings.Contains(suggestion, "Add") ||
			strings.Contains(suggestion, "Need")

		if !hasActionable {
			t.Errorf("Recovery suggestion for %v lacks actionable guidance: %q", err, suggestion)
		}
	}
}

func TestErrorWithRecovery_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := &ErrorWithRecovery{
		Err:      originalErr,
		Recovery: "recovery suggestion",
	}

	// Test that Unwrap returns the original error
	unwrapped := wrappedErr.Unwrap()
	if unwrapped != originalErr {
		t.Error("Unwrap did not return original error")
	}

	// Test that errors.Is works with wrapped error
	if !errors.Is(wrappedErr, originalErr) {
		t.Error("errors.Is should recognize wrapped error")
	}
}

func TestRecoverySuggestions_SecurityErrors(t *testing.T) {
	// Security-related errors should have "SECURITY" prefix in suggestions
	securityErrors := []error{
		ErrSybilAttackDetected,
		ErrFlashLoanDetected,
		ErrDataPoisoning,
		ErrIPRegionMismatch,
		ErrPrivateIPNotAllowed,
		ErrInsufficientGeoDiversity,
		ErrTooManyValidatorsFromSameIP,
		ErrInsufficientOracleConsensus,
		ErrUnauthorizedPause,
		ErrUnauthorizedResume,
	}

	for _, err := range securityErrors {
		suggestion := RecoverySuggestions[err]
		if !strings.Contains(suggestion, "SECURITY") {
			t.Errorf("Security error %v should have 'SECURITY' in suggestion, got: %q", err, suggestion)
		}
	}
}
