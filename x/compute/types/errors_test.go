package types

import (
	"errors"
	"strings"
	"testing"

	sdkerrors "cosmossdk.io/errors"
)

func TestErrorDefinitions(t *testing.T) {
	// Test that all errors are properly registered with unique codes
	errorTests := []struct {
		name string
		err  error
		code uint32
	}{
		{"ErrInvalidRequest", ErrInvalidRequest, 2},
		{"ErrInvalidProvider", ErrInvalidProvider, 3},
		{"ErrInvalidResult", ErrInvalidResult, 4},
		{"ErrInvalidProof", ErrInvalidProof, 5},
		{"ErrProviderNotFound", ErrProviderNotFound, 10},
		{"ErrProviderNotActive", ErrProviderNotActive, 11},
		{"ErrProviderOverloaded", ErrProviderOverloaded, 12},
		{"ErrInsufficientStake", ErrInsufficientStake, 13},
		{"ErrProviderSlashed", ErrProviderSlashed, 14},
		{"ErrRequestNotFound", ErrRequestNotFound, 20},
		{"ErrRequestExpired", ErrRequestExpired, 21},
		{"ErrRequestAlreadyCompleted", ErrRequestAlreadyCompleted, 22},
		{"ErrRequestCancelled", ErrRequestCancelled, 23},
		{"ErrInsufficientEscrow", ErrInsufficientEscrow, 30},
		{"ErrEscrowLocked", ErrEscrowLocked, 31},
		{"ErrEscrowNotFound", ErrEscrowNotFound, 32},
		{"ErrEscrowRefundFailed", ErrEscrowRefundFailed, 33},
		{"ErrVerificationFailed", ErrVerificationFailed, 40},
		{"ErrInvalidSignature", ErrInvalidSignature, 41},
		{"ErrInvalidMerkleProof", ErrInvalidMerkleProof, 42},
		{"ErrInvalidStateCommitment", ErrInvalidStateCommitment, 43},
		{"ErrReplayAttackDetected", ErrReplayAttackDetected, 44},
		{"ErrProofExpired", ErrProofExpired, 45},
		{"ErrInvalidZKProof", ErrInvalidZKProof, 50},
		{"ErrZKVerificationFailed", ErrZKVerificationFailed, 51},
		{"ErrInvalidCircuit", ErrInvalidCircuit, 52},
		{"ErrInvalidPublicInputs", ErrInvalidPublicInputs, 53},
		{"ErrInvalidWitness", ErrInvalidWitness, 54},
		{"ErrProofTooLarge", ErrProofTooLarge, 55},
		{"ErrInsufficientDeposit", ErrInsufficientDeposit, 56},
		{"ErrDepositTransferFailed", ErrDepositTransferFailed, 57},
		{"ErrInsufficientResources", ErrInsufficientResources, 60},
		{"ErrResourceQuotaExceeded", ErrResourceQuotaExceeded, 61},
		{"ErrInvalidResourceSpec", ErrInvalidResourceSpec, 62},
		{"ErrUnauthorized", ErrUnauthorized, 70},
		{"ErrRateLimitExceeded", ErrRateLimitExceeded, 71},
		{"ErrCircuitBreakerActive", ErrCircuitBreakerActive, 72},
		{"ErrSuspiciousActivity", ErrSuspiciousActivity, 73},
		{"ErrInvalidPacket", ErrInvalidPacket, 80},
		{"ErrChannelNotFound", ErrChannelNotFound, 81},
		{"ErrPacketTimeout", ErrPacketTimeout, 82},
		{"ErrInvalidAcknowledgement", ErrInvalidAcknowledgement, 83},
		{"ErrInvalidNonce", ErrInvalidNonce, 84},
		{"ErrUnauthorizedChannel", ErrUnauthorizedChannel, 85},
		{"ErrCircuitBreakerAlreadyOpen", ErrCircuitBreakerAlreadyOpen, 86},
		{"ErrCircuitBreakerAlreadyClosed", ErrCircuitBreakerAlreadyClosed, 87},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s is nil", tt.name)
			}
		})
	}
}

func TestErrorAliases(t *testing.T) {
	// Test that ErrInvalidAck is an alias for ErrInvalidAcknowledgement
	if ErrInvalidAck != ErrInvalidAcknowledgement {
		t.Error("ErrInvalidAck should be an alias for ErrInvalidAcknowledgement")
	}
}

func TestErrorWithRecovery(t *testing.T) {
	baseErr := errors.New("base error")
	recovery := "recovery suggestion"

	errWithRecovery := &ErrorWithRecovery{
		Err:      baseErr,
		Recovery: recovery,
	}

	// Test Error() method
	if errWithRecovery.Error() != baseErr.Error() {
		t.Errorf("ErrorWithRecovery.Error() = %v, want %v", errWithRecovery.Error(), baseErr.Error())
	}

	// Test Unwrap() method
	if errWithRecovery.Unwrap() != baseErr {
		t.Errorf("ErrorWithRecovery.Unwrap() = %v, want %v", errWithRecovery.Unwrap(), baseErr)
	}

	// Test that recovery suggestion is accessible
	if errWithRecovery.Recovery != recovery {
		t.Errorf("ErrorWithRecovery.Recovery = %v, want %v", errWithRecovery.Recovery, recovery)
	}
}

func TestWrapWithRecovery(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		msg          string
		wantRecovery bool
	}{
		{
			name:         "error with recovery suggestion",
			err:          ErrInvalidRequest,
			msg:          "request validation failed",
			wantRecovery: true,
		},
		{
			name:         "error without recovery suggestion",
			err:          errors.New("unknown error"),
			msg:          "something went wrong",
			wantRecovery: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapWithRecovery(tt.err, "%s", tt.msg)

			if wrapped == nil {
				t.Fatal("WrapWithRecovery returned nil")
			}

			// Check if error message contains the wrapper message
			if !strings.Contains(wrapped.Error(), tt.msg) {
				t.Errorf("WrapWithRecovery() error = %v, want error containing %v", wrapped.Error(), tt.msg)
			}

			// Check if it's an ErrorWithRecovery when expected
			if tt.wantRecovery {
				var errWithRec *ErrorWithRecovery
				if !errors.As(wrapped, &errWithRec) {
					t.Error("WrapWithRecovery() should return ErrorWithRecovery for known errors")
				} else if errWithRec.Recovery == "" {
					t.Error("ErrorWithRecovery.Recovery should not be empty for known errors")
				}
			}
		})
	}
}

func TestGetRecoverySuggestion(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantContain string
	}{
		{
			name:        "ErrInvalidRequest",
			err:         ErrInvalidRequest,
			wantContain: "Check request parameters",
		},
		{
			name:        "ErrProviderNotFound",
			err:         ErrProviderNotFound,
			wantContain: "Register as a provider",
		},
		{
			name:        "ErrInvalidSignature",
			err:         ErrInvalidSignature,
			wantContain: "Ed25519 signature",
		},
		{
			name:        "ErrUnauthorizedChannel",
			err:         ErrUnauthorizedChannel,
			wantContain: "governance-approved channel",
		},
		{
			name:        "wrapped error",
			err:         sdkerrors.Wrap(ErrInvalidProof, "proof validation failed"),
			wantContain: "Validate proof structure",
		},
		{
			name:        "unknown error",
			err:         errors.New("unknown error"),
			wantContain: "No recovery suggestion available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := GetRecoverySuggestion(tt.err)

			if suggestion == "" {
				t.Error("GetRecoverySuggestion() returned empty string")
			}

			if !strings.Contains(suggestion, tt.wantContain) {
				t.Errorf("GetRecoverySuggestion() = %v, want suggestion containing %v", suggestion, tt.wantContain)
			}
		})
	}
}

func TestRecoverySuggestionsCompleteness(t *testing.T) {
	// Verify that all major errors have recovery suggestions
	errorsToCheck := []error{
		ErrInvalidRequest,
		ErrInvalidProvider,
		ErrInvalidResult,
		ErrInvalidProof,
		ErrProviderNotFound,
		ErrProviderNotActive,
		ErrProviderOverloaded,
		ErrInsufficientStake,
		ErrProviderSlashed,
		ErrRequestNotFound,
		ErrRequestExpired,
		ErrRequestAlreadyCompleted,
		ErrRequestCancelled,
		ErrInsufficientEscrow,
		ErrEscrowLocked,
		ErrEscrowNotFound,
		ErrEscrowRefundFailed,
		ErrVerificationFailed,
		ErrInvalidSignature,
		ErrInvalidMerkleProof,
		ErrInvalidStateCommitment,
		ErrReplayAttackDetected,
		ErrProofExpired,
		ErrInvalidZKProof,
		ErrZKVerificationFailed,
		ErrInvalidCircuit,
		ErrInvalidPublicInputs,
		ErrInvalidWitness,
		ErrProofTooLarge,
		ErrInsufficientDeposit,
		ErrDepositTransferFailed,
		ErrInsufficientResources,
		ErrResourceQuotaExceeded,
		ErrInvalidResourceSpec,
		ErrUnauthorized,
		ErrUnauthorizedChannel,
		ErrRateLimitExceeded,
		ErrCircuitBreakerActive,
		ErrSuspiciousActivity,
		ErrInvalidPacket,
		ErrChannelNotFound,
		ErrPacketTimeout,
		ErrInvalidAcknowledgement,
	}

	for _, err := range errorsToCheck {
		t.Run(err.Error(), func(t *testing.T) {
			suggestion := GetRecoverySuggestion(err)
			if strings.Contains(suggestion, "No recovery suggestion available") {
				t.Errorf("Error %v is missing a recovery suggestion", err)
			}
		})
	}
}

func TestInitFunction(t *testing.T) {
	// Test that the init function properly sets up ErrInvalidNonce
	suggestion := RecoverySuggestions[ErrInvalidNonce]
	if suggestion == "" {
		t.Error("ErrInvalidNonce should have a recovery suggestion after init()")
	}

	if !strings.Contains(suggestion, "nonce") {
		t.Errorf("ErrInvalidNonce recovery suggestion should mention 'nonce', got: %s", suggestion)
	}
}

func TestUnwrapChain(t *testing.T) {
	// Test that GetRecoverySuggestion can unwrap multiple layers
	baseErr := ErrInvalidRequest
	wrapped1 := sdkerrors.Wrap(baseErr, "first wrap")
	wrapped2 := sdkerrors.Wrap(wrapped1, "second wrap")
	wrapped3 := sdkerrors.Wrap(wrapped2, "third wrap")

	suggestion := GetRecoverySuggestion(wrapped3)
	expectedSuggestion := RecoverySuggestions[ErrInvalidRequest]

	if suggestion != expectedSuggestion {
		t.Errorf("GetRecoverySuggestion() failed to unwrap chain: got %v, want %v", suggestion, expectedSuggestion)
	}
}

func BenchmarkGetRecoverySuggestion(b *testing.B) {
	err := sdkerrors.Wrap(sdkerrors.Wrap(ErrInvalidRequest, "layer1"), "layer2")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetRecoverySuggestion(err)
	}
}

func BenchmarkWrapWithRecovery(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WrapWithRecovery(ErrInvalidRequest, "request failed with ID %d", 123)
	}
}

func TestRecoverySuggestionQuality(t *testing.T) {
	// Verify that recovery suggestions are helpful and actionable
	for err, suggestion := range RecoverySuggestions {
		t.Run(err.Error(), func(t *testing.T) {
			if len(suggestion) < 20 {
				t.Errorf("Recovery suggestion too short (< 20 chars): %s", suggestion)
			}

			if len(suggestion) > 1000 {
				t.Errorf("Recovery suggestion too long (> 1000 chars): %s", suggestion)
			}

			// Should contain actionable words or be descriptive enough
			actionableWords := []string{"check", "verify", "ensure", "confirm", "review", "query", "wait", "use", "submit", "increase", "reduce", "generate", "must"}
			hasActionableWord := false
			lowerSuggestion := strings.ToLower(suggestion)
			for _, word := range actionableWords {
				if strings.Contains(lowerSuggestion, word) {
					hasActionableWord = true
					break
				}
			}

			if !hasActionableWord {
				t.Errorf("Recovery suggestion lacks actionable guidance: %s", suggestion)
			}
		})
	}
}

func TestErrorMessages(t *testing.T) {
	// Verify error messages are descriptive
	errorMessages := map[error]string{
		ErrInvalidRequest:          "invalid compute request",
		ErrInvalidProvider:         "invalid provider",
		ErrProviderNotFound:        "provider not found",
		ErrInvalidSignature:        "invalid signature",
		ErrReplayAttackDetected:    "replay attack detected",
		ErrUnauthorizedChannel:     "unauthorized IBC channel",
	}

	for err, expectedMsg := range errorMessages {
		if !strings.Contains(err.Error(), expectedMsg) {
			t.Errorf("Error %v should contain message %q", err, expectedMsg)
		}
	}
}
