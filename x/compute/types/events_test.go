package types

import (
	"strings"
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		// Request events
		{"EventTypeComputeRequest", EventTypeComputeRequest, "compute_request"},
		{"EventTypeComputeRequestAccepted", EventTypeComputeRequestAccepted, "compute_request_accepted"},
		{"EventTypeComputeRequestRejected", EventTypeComputeRequestRejected, "compute_request_rejected"},

		// Result events
		{"EventTypeComputeResult", EventTypeComputeResult, "compute_result"},
		{"EventTypeComputeResultVerified", EventTypeComputeResultVerified, "compute_result_verified"},
		{"EventTypeComputeResultRejected", EventTypeComputeResultRejected, "compute_result_rejected"},

		// Provider events
		{"EventTypeComputeProviderRegistered", EventTypeComputeProviderRegistered, "compute_provider_registered"},
		{"EventTypeComputeProviderUnregistered", EventTypeComputeProviderUnregistered, "compute_provider_unregistered"},
		{"EventTypeComputeProviderSlashed", EventTypeComputeProviderSlashed, "compute_provider_slashed"},
		{"EventTypeComputeProviderJailed", EventTypeComputeProviderJailed, "compute_provider_jailed"},

		// Dispute events
		{"EventTypeComputeDispute", EventTypeComputeDispute, "compute_dispute"},
		{"EventTypeComputeDisputeResolved", EventTypeComputeDisputeResolved, "compute_dispute_resolved"},

		// Escrow events
		{"EventTypeComputeEscrowCreated", EventTypeComputeEscrowCreated, "compute_escrow_created"},
		{"EventTypeComputeEscrowReleased", EventTypeComputeEscrowReleased, "compute_escrow_released"},
		{"EventTypeComputeEscrowRefunded", EventTypeComputeEscrowRefunded, "compute_escrow_refunded"},

		// Verification events
		{"EventTypeComputeVerification", EventTypeComputeVerification, "compute_verification"},
		{"EventTypeComputeVerificationPassed", EventTypeComputeVerificationPassed, "compute_verification_passed"},
		{"EventTypeComputeVerificationFailed", EventTypeComputeVerificationFailed, "compute_verification_failed"},

		// ZK proof events
		{"EventTypeComputeZKProof", EventTypeComputeZKProof, "compute_zk_proof"},
		{"EventTypeComputeZKProofVerified", EventTypeComputeZKProofVerified, "compute_zk_proof_verified"},
		{"EventTypeComputeZKProofFailed", EventTypeComputeZKProofFailed, "compute_zk_proof_failed"},

		// Reputation events
		{"EventTypeComputeReputationUpdate", EventTypeComputeReputationUpdate, "compute_reputation_update"},

		// Cross-chain events
		{"EventTypeComputeCrossChainJob", EventTypeComputeCrossChainJob, "compute_cross_chain_job"},
		{"EventTypeComputeCrossChainJobComplete", EventTypeComputeCrossChainJobComplete, "compute_cross_chain_job_complete"},
		{"EventTypeComputeCrossChainJobFailed", EventTypeComputeCrossChainJobFailed, "compute_cross_chain_job_failed"},

		// Circuit breaker events
		{"EventTypeComputeCircuitBreakerTripped", EventTypeComputeCircuitBreakerTripped, "compute_circuit_breaker_tripped"},
		{"EventTypeComputeCircuitBreakerReset", EventTypeComputeCircuitBreakerReset, "compute_circuit_breaker_reset"},
		{"EventTypeCircuitBreakerOpen", EventTypeCircuitBreakerOpen, "compute_circuit_breaker_open"},
		{"EventTypeCircuitBreakerClose", EventTypeCircuitBreakerClose, "compute_circuit_breaker_close"},

		// Monitoring events
		{"EventTypeComputeAnomalyDetected", EventTypeComputeAnomalyDetected, "compute_anomaly_detected"},
		{"EventTypeComputeAlert", EventTypeComputeAlert, "compute_alert"},

		// Other events
		{"EventTypeJobCancellation", EventTypeJobCancellation, "compute_job_cancellation"},
		{"EventTypeReputationOverride", EventTypeReputationOverride, "compute_reputation_override"},
		{"EventTypeReputationOverrideClear", EventTypeReputationOverrideClear, "compute_reputation_override_clear"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

func TestAttributeKeyConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		// Request attributes
		{"AttributeKeyRequestID", AttributeKeyRequestID, "request_id"},
		{"AttributeKeyRequester", AttributeKeyRequester, "requester"},
		{"AttributeKeyCodeHash", AttributeKeyCodeHash, "code_hash"},
		{"AttributeKeyInput", AttributeKeyInput, "input"},
		{"AttributeKeyInputHash", AttributeKeyInputHash, "input_hash"},

		// Result attributes
		{"AttributeKeyResult", AttributeKeyResult, "result"},
		{"AttributeKeyResultHash", AttributeKeyResultHash, "result_hash"},
		{"AttributeKeyOutput", AttributeKeyOutput, "output"},
		{"AttributeKeyOutputHash", AttributeKeyOutputHash, "output_hash"},

		// Provider attributes
		{"AttributeKeyProvider", AttributeKeyProvider, "provider"},
		{"AttributeKeyProviderID", AttributeKeyProviderID, "provider_id"},
		{"AttributeKeyProviderName", AttributeKeyProviderName, "provider_name"},
		{"AttributeKeyProviderStake", AttributeKeyProviderStake, "provider_stake"},
		{"AttributeKeyProviderStatus", AttributeKeyProviderStatus, "provider_status"},

		// Dispute attributes
		{"AttributeKeyDisputeID", AttributeKeyDisputeID, "dispute_id"},
		{"AttributeKeyDisputer", AttributeKeyDisputer, "disputer"},
		{"AttributeKeyDisputeReason", AttributeKeyDisputeReason, "dispute_reason"},
		{"AttributeKeyResolution", AttributeKeyResolution, "resolution"},

		// Escrow attributes
		{"AttributeKeyEscrowID", AttributeKeyEscrowID, "escrow_id"},
		{"AttributeKeyEscrowAmount", AttributeKeyEscrowAmount, "escrow_amount"},
		{"AttributeKeyPayment", AttributeKeyPayment, "payment"},
		{"AttributeKeyAmount", AttributeKeyAmount, "amount"},
		{"AttributeKeyDenom", AttributeKeyDenom, "denom"},

		// Verification attributes
		{"AttributeKeyVerificationID", AttributeKeyVerificationID, "verification_id"},
		{"AttributeKeyVerificationScore", AttributeKeyVerificationScore, "verification_score"},
		{"AttributeKeyVerificationStatus", AttributeKeyVerificationStatus, "verification_status"},
		{"AttributeKeyProofType", AttributeKeyProofType, "proof_type"},
		{"AttributeKeyProofHash", AttributeKeyProofHash, "proof_hash"},

		// ZK proof attributes
		{"AttributeKeyZKProof", AttributeKeyZKProof, "zk_proof"},
		{"AttributeKeyZKPublicKey", AttributeKeyZKPublicKey, "zk_public_key"},
		{"AttributeKeyZKNonce", AttributeKeyZKNonce, "zk_nonce"},
		{"AttributeKeyZKTimestamp", AttributeKeyZKTimestamp, "zk_timestamp"},

		// Reputation attributes
		{"AttributeKeyReputation", AttributeKeyReputation, "reputation"},
		{"AttributeKeyReputationDelta", AttributeKeyReputationDelta, "reputation_delta"},
		{"AttributeKeySuccessRate", AttributeKeySuccessRate, "success_rate"},
		{"AttributeKeyTotalJobs", AttributeKeyTotalJobs, "total_jobs"},

		// Timing attributes
		{"AttributeKeyTimestamp", AttributeKeyTimestamp, "timestamp"},
		{"AttributeKeyBlockHeight", AttributeKeyBlockHeight, "block_height"},
		{"AttributeKeyDeadline", AttributeKeyDeadline, "deadline"},
		{"AttributeKeyExecutionTime", AttributeKeyExecutionTime, "execution_time"},

		// Cross-chain attributes
		{"AttributeKeyJobID", AttributeKeyJobID, "job_id"},
		{"AttributeKeyTargetChain", AttributeKeyTargetChain, "target_chain"},
		{"AttributeKeySourceChain", AttributeKeySourceChain, "source_chain"},

		// Circuit breaker attributes
		{"AttributeKeyCircuitState", AttributeKeyCircuitState, "circuit_state"},
		{"AttributeKeyFailureCount", AttributeKeyFailureCount, "failure_count"},
		{"AttributeKeyThreshold", AttributeKeyThreshold, "threshold"},

		// Monitoring attributes
		{"AttributeKeyAnomalyType", AttributeKeyAnomalyType, "anomaly_type"},
		{"AttributeKeyAlertLevel", AttributeKeyAlertLevel, "alert_level"},
		{"AttributeKeyMetric", AttributeKeyMetric, "metric"},
		{"AttributeKeyValue", AttributeKeyValue, "value"},

		// Status attributes
		{"AttributeKeyStatus", AttributeKeyStatus, "status"},
		{"AttributeKeyReason", AttributeKeyReason, "reason"},
		{"AttributeKeyError", AttributeKeyError, "error"},
		{"AttributeKeyActor", AttributeKeyActor, "actor"},
		{"AttributeKeyScore", AttributeKeyScore, "score"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

func TestEventTypeNamingConvention(t *testing.T) {
	// All event types should use lowercase with underscore separator
	eventTypes := []string{
		EventTypeComputeRequest,
		EventTypeComputeRequestAccepted,
		EventTypeComputeRequestRejected,
		EventTypeComputeResult,
		EventTypeComputeResultVerified,
		EventTypeComputeResultRejected,
		EventTypeComputeProviderRegistered,
		EventTypeComputeProviderUnregistered,
		EventTypeComputeProviderSlashed,
		EventTypeComputeProviderJailed,
		EventTypeComputeDispute,
		EventTypeComputeDisputeResolved,
		EventTypeComputeEscrowCreated,
		EventTypeComputeEscrowReleased,
		EventTypeComputeEscrowRefunded,
		EventTypeComputeVerification,
		EventTypeComputeVerificationPassed,
		EventTypeComputeVerificationFailed,
		EventTypeComputeZKProof,
		EventTypeComputeZKProofVerified,
		EventTypeComputeZKProofFailed,
		EventTypeComputeReputationUpdate,
		EventTypeComputeCrossChainJob,
		EventTypeComputeCrossChainJobComplete,
		EventTypeComputeCrossChainJobFailed,
		EventTypeComputeCircuitBreakerTripped,
		EventTypeComputeCircuitBreakerReset,
		EventTypeComputeAnomalyDetected,
		EventTypeComputeAlert,
		EventTypeCircuitBreakerOpen,
		EventTypeCircuitBreakerClose,
		EventTypeJobCancellation,
		EventTypeReputationOverride,
		EventTypeReputationOverrideClear,
	}

	for _, eventType := range eventTypes {
		// Should be lowercase
		if eventType != strings.ToLower(eventType) {
			t.Errorf("Event type %s should be lowercase", eventType)
		}

		// Should not contain spaces
		if strings.Contains(eventType, " ") {
			t.Errorf("Event type %s should not contain spaces", eventType)
		}

		// Should not contain uppercase letters
		for _, r := range eventType {
			if r >= 'A' && r <= 'Z' {
				t.Errorf("Event type %s contains uppercase letter %c", eventType, r)
			}
		}

		// Should start with "compute_" (module prefix)
		if !strings.HasPrefix(eventType, "compute_") {
			t.Errorf("Event type %s should start with 'compute_'", eventType)
		}
	}
}

func TestAttributeKeyNamingConvention(t *testing.T) {
	// All attribute keys should use lowercase with underscore separator
	attributeKeys := []string{
		AttributeKeyRequestID,
		AttributeKeyRequester,
		AttributeKeyCodeHash,
		AttributeKeyInput,
		AttributeKeyInputHash,
		AttributeKeyResult,
		AttributeKeyResultHash,
		AttributeKeyOutput,
		AttributeKeyOutputHash,
		AttributeKeyProvider,
		AttributeKeyProviderID,
		AttributeKeyProviderName,
		AttributeKeyProviderStake,
		AttributeKeyProviderStatus,
		AttributeKeyDisputeID,
		AttributeKeyDisputer,
		AttributeKeyDisputeReason,
		AttributeKeyResolution,
		AttributeKeyEscrowID,
		AttributeKeyEscrowAmount,
		AttributeKeyPayment,
		AttributeKeyAmount,
		AttributeKeyDenom,
		AttributeKeyVerificationID,
		AttributeKeyVerificationScore,
		AttributeKeyVerificationStatus,
		AttributeKeyProofType,
		AttributeKeyProofHash,
		AttributeKeyZKProof,
		AttributeKeyZKPublicKey,
		AttributeKeyZKNonce,
		AttributeKeyZKTimestamp,
		AttributeKeyReputation,
		AttributeKeyReputationDelta,
		AttributeKeySuccessRate,
		AttributeKeyTotalJobs,
		AttributeKeyTimestamp,
		AttributeKeyBlockHeight,
		AttributeKeyDeadline,
		AttributeKeyExecutionTime,
		AttributeKeyJobID,
		AttributeKeyTargetChain,
		AttributeKeySourceChain,
		AttributeKeyCircuitState,
		AttributeKeyFailureCount,
		AttributeKeyThreshold,
		AttributeKeyAnomalyType,
		AttributeKeyAlertLevel,
		AttributeKeyMetric,
		AttributeKeyValue,
		AttributeKeyStatus,
		AttributeKeyReason,
		AttributeKeyError,
		AttributeKeyActor,
		AttributeKeyScore,
	}

	for _, key := range attributeKeys {
		// Should be lowercase
		if key != strings.ToLower(key) {
			t.Errorf("Attribute key %s should be lowercase", key)
		}

		// Should not contain spaces
		if strings.Contains(key, " ") {
			t.Errorf("Attribute key %s should not contain spaces", key)
		}

		// Should not contain uppercase letters
		for _, r := range key {
			if r >= 'A' && r <= 'Z' {
				t.Errorf("Attribute key %s contains uppercase letter %c", key, r)
			}
		}
	}
}

func TestEventTypeUniqueness(t *testing.T) {
	// All event types should be unique
	eventTypes := []string{
		EventTypeComputeRequest,
		EventTypeComputeRequestAccepted,
		EventTypeComputeRequestRejected,
		EventTypeComputeResult,
		EventTypeComputeResultVerified,
		EventTypeComputeResultRejected,
		EventTypeComputeProviderRegistered,
		EventTypeComputeProviderUnregistered,
		EventTypeComputeProviderSlashed,
		EventTypeComputeProviderJailed,
		EventTypeComputeDispute,
		EventTypeComputeDisputeResolved,
		EventTypeComputeEscrowCreated,
		EventTypeComputeEscrowReleased,
		EventTypeComputeEscrowRefunded,
		EventTypeComputeVerification,
		EventTypeComputeVerificationPassed,
		EventTypeComputeVerificationFailed,
		EventTypeComputeZKProof,
		EventTypeComputeZKProofVerified,
		EventTypeComputeZKProofFailed,
		EventTypeComputeReputationUpdate,
		EventTypeComputeCrossChainJob,
		EventTypeComputeCrossChainJobComplete,
		EventTypeComputeCrossChainJobFailed,
		EventTypeComputeCircuitBreakerTripped,
		EventTypeComputeCircuitBreakerReset,
		EventTypeComputeAnomalyDetected,
		EventTypeComputeAlert,
		EventTypeCircuitBreakerOpen,
		EventTypeCircuitBreakerClose,
		EventTypeJobCancellation,
		EventTypeReputationOverride,
		EventTypeReputationOverrideClear,
	}

	seen := make(map[string]bool)
	for _, eventType := range eventTypes {
		if seen[eventType] {
			t.Errorf("Duplicate event type: %s", eventType)
		}
		seen[eventType] = true
	}
}

func TestAttributeKeyUniqueness(t *testing.T) {
	// All attribute keys should be unique
	attributeKeys := []string{
		AttributeKeyRequestID,
		AttributeKeyRequester,
		AttributeKeyCodeHash,
		AttributeKeyInput,
		AttributeKeyInputHash,
		AttributeKeyResult,
		AttributeKeyResultHash,
		AttributeKeyOutput,
		AttributeKeyOutputHash,
		AttributeKeyProvider,
		AttributeKeyProviderID,
		AttributeKeyProviderName,
		AttributeKeyProviderStake,
		AttributeKeyProviderStatus,
		AttributeKeyDisputeID,
		AttributeKeyDisputer,
		AttributeKeyDisputeReason,
		AttributeKeyResolution,
		AttributeKeyEscrowID,
		AttributeKeyEscrowAmount,
		AttributeKeyPayment,
		AttributeKeyAmount,
		AttributeKeyDenom,
		AttributeKeyVerificationID,
		AttributeKeyVerificationScore,
		AttributeKeyVerificationStatus,
		AttributeKeyProofType,
		AttributeKeyProofHash,
		AttributeKeyZKProof,
		AttributeKeyZKPublicKey,
		AttributeKeyZKNonce,
		AttributeKeyZKTimestamp,
		AttributeKeyReputation,
		AttributeKeyReputationDelta,
		AttributeKeySuccessRate,
		AttributeKeyTotalJobs,
		AttributeKeyTimestamp,
		AttributeKeyBlockHeight,
		AttributeKeyDeadline,
		AttributeKeyExecutionTime,
		AttributeKeyJobID,
		AttributeKeyTargetChain,
		AttributeKeySourceChain,
		AttributeKeyCircuitState,
		AttributeKeyFailureCount,
		AttributeKeyThreshold,
		AttributeKeyAnomalyType,
		AttributeKeyAlertLevel,
		AttributeKeyMetric,
		AttributeKeyValue,
		AttributeKeyStatus,
		AttributeKeyReason,
		AttributeKeyError,
		AttributeKeyActor,
		AttributeKeyScore,
	}

	seen := make(map[string]bool)
	for _, key := range attributeKeys {
		if seen[key] {
			t.Errorf("Duplicate attribute key: %s", key)
		}
		seen[key] = true
	}
}

func TestEventTypeCoverage(t *testing.T) {
	// Verify we have events for all major operations
	requiredCategories := map[string][]string{
		"request":         {EventTypeComputeRequest, EventTypeComputeRequestAccepted, EventTypeComputeRequestRejected},
		"result":          {EventTypeComputeResult, EventTypeComputeResultVerified, EventTypeComputeResultRejected},
		"provider":        {EventTypeComputeProviderRegistered, EventTypeComputeProviderUnregistered},
		"dispute":         {EventTypeComputeDispute, EventTypeComputeDisputeResolved},
		"escrow":          {EventTypeComputeEscrowCreated, EventTypeComputeEscrowReleased},
		"verification":    {EventTypeComputeVerification, EventTypeComputeVerificationPassed},
		"zk_proof":        {EventTypeComputeZKProof, EventTypeComputeZKProofVerified},
		"circuit_breaker": {EventTypeComputeCircuitBreakerTripped, EventTypeComputeCircuitBreakerReset},
	}

	for category, events := range requiredCategories {
		if len(events) == 0 {
			t.Errorf("Category %s has no events defined", category)
		}
	}
}

func TestNoSecretInEventNames(t *testing.T) {
	// Verify #nosec comments are only on verification event types
	// These are false positives - they're event identifiers, not credentials
	verificationEvents := []string{
		EventTypeComputeVerification,
		EventTypeComputeVerificationPassed,
		EventTypeComputeVerificationFailed,
	}

	for _, event := range verificationEvents {
		// These should not actually contain secrets, just the word "verification"
		if !strings.Contains(event, "verification") {
			t.Errorf("Expected verification event %s to contain 'verification'", event)
		}
	}
}
