package types

// Event types for the Compute module
// All event types use lowercase with underscore separator (module_action format)
const (
	// Request events
	EventTypeComputeRequest         = "compute_request"
	EventTypeComputeRequestAccepted = "compute_request_accepted"
	EventTypeComputeRequestRejected = "compute_request_rejected"

	// Result events
	EventTypeComputeResult         = "compute_result"
	EventTypeComputeResultVerified = "compute_result_verified"
	EventTypeComputeResultRejected = "compute_result_rejected"

	// Provider events
	EventTypeComputeProviderRegistered   = "compute_provider_registered"
	EventTypeComputeProviderUnregistered = "compute_provider_unregistered"
	EventTypeComputeProviderSlashed      = "compute_provider_slashed"
	EventTypeComputeProviderJailed       = "compute_provider_jailed"

	// Dispute events
	EventTypeComputeDispute         = "compute_dispute"
	EventTypeComputeDisputeResolved = "compute_dispute_resolved"

	// Escrow events
	EventTypeComputeEscrowCreated  = "compute_escrow_created"
	EventTypeComputeEscrowReleased = "compute_escrow_released"
	EventTypeComputeEscrowRefunded = "compute_escrow_refunded"

	// Verification events
	EventTypeComputeVerification       = "compute_verification"        // #nosec G101 - event identifiers, not credentials
	EventTypeComputeVerificationPassed = "compute_verification_passed" // #nosec G101 - event identifiers, not credentials
	EventTypeComputeVerificationFailed = "compute_verification_failed" // #nosec G101 - event identifiers, not credentials

	// ZK proof events
	EventTypeComputeZKProof         = "compute_zk_proof"
	EventTypeComputeZKProofVerified = "compute_zk_proof_verified"
	EventTypeComputeZKProofFailed   = "compute_zk_proof_failed"

	// Reputation events
	EventTypeComputeReputationUpdate = "compute_reputation_update"

	// Cross-chain events
	EventTypeComputeCrossChainJob         = "compute_cross_chain_job"
	EventTypeComputeCrossChainJobComplete = "compute_cross_chain_job_complete"
	EventTypeComputeCrossChainJobFailed   = "compute_cross_chain_job_failed"

	// Circuit breaker events
	EventTypeComputeCircuitBreakerTripped = "compute_circuit_breaker_tripped"
	EventTypeComputeCircuitBreakerReset   = "compute_circuit_breaker_reset"

	// Monitoring events
	EventTypeComputeAnomalyDetected = "compute_anomaly_detected"
	EventTypeComputeAlert           = "compute_alert"
)

// Event attribute keys for the Compute module
// All attribute keys use lowercase with underscore separator
const (
	// Request attributes
	AttributeKeyRequestID = "request_id"
	AttributeKeyRequester = "requester"
	AttributeKeyCodeHash  = "code_hash"
	AttributeKeyInput     = "input"
	AttributeKeyInputHash = "input_hash"

	// Result attributes
	AttributeKeyResult     = "result"
	AttributeKeyResultHash = "result_hash"
	AttributeKeyOutput     = "output"
	AttributeKeyOutputHash = "output_hash"

	// Provider attributes
	AttributeKeyProvider       = "provider"
	AttributeKeyProviderID     = "provider_id"
	AttributeKeyProviderName   = "provider_name"
	AttributeKeyProviderStake  = "provider_stake"
	AttributeKeyProviderStatus = "provider_status"

	// Dispute attributes
	AttributeKeyDisputeID     = "dispute_id"
	AttributeKeyDisputer      = "disputer"
	AttributeKeyDisputeReason = "dispute_reason"
	AttributeKeyResolution    = "resolution"

	// Escrow attributes
	AttributeKeyEscrowID     = "escrow_id"
	AttributeKeyEscrowAmount = "escrow_amount"
	AttributeKeyPayment      = "payment"
	AttributeKeyAmount       = "amount"
	AttributeKeyDenom        = "denom"

	// Verification attributes
	AttributeKeyVerificationID     = "verification_id"
	AttributeKeyVerificationScore  = "verification_score"
	AttributeKeyVerificationStatus = "verification_status"
	AttributeKeyProofType          = "proof_type"
	AttributeKeyProofHash          = "proof_hash"

	// ZK proof attributes
	AttributeKeyZKProof     = "zk_proof"
	AttributeKeyZKPublicKey = "zk_public_key"
	AttributeKeyZKNonce     = "zk_nonce"
	AttributeKeyZKTimestamp = "zk_timestamp"

	// Reputation attributes
	AttributeKeyReputation      = "reputation"
	AttributeKeyReputationDelta = "reputation_delta"
	AttributeKeySuccessRate     = "success_rate"
	AttributeKeyTotalJobs       = "total_jobs"

	// Timing attributes
	AttributeKeyTimestamp     = "timestamp"
	AttributeKeyBlockHeight   = "block_height"
	AttributeKeyDeadline      = "deadline"
	AttributeKeyExecutionTime = "execution_time"

	// Cross-chain attributes (excluding IBC-specific ones in types.go)
	AttributeKeyJobID       = "job_id"
	AttributeKeyTargetChain = "target_chain"
	AttributeKeySourceChain = "source_chain"

	// Circuit breaker attributes
	AttributeKeyCircuitState = "circuit_state"
	AttributeKeyFailureCount = "failure_count"
	AttributeKeyThreshold    = "threshold"

	// Monitoring attributes
	AttributeKeyAnomalyType = "anomaly_type"
	AttributeKeyAlertLevel  = "alert_level"
	AttributeKeyMetric      = "metric"
	AttributeKeyValue       = "value"

	// Status attributes
	AttributeKeyStatus = "status"
	AttributeKeyReason = "reason"
	AttributeKeyError  = "error"
	AttributeKeyActor  = "actor"
	AttributeKeyScore  = "score"
)

// Circuit breaker event types
const (
	EventTypeCircuitBreakerOpen     = "compute_circuit_breaker_open"
	EventTypeCircuitBreakerClose    = "compute_circuit_breaker_close"
	EventTypeJobCancellation        = "compute_job_cancellation"
	EventTypeReputationOverride     = "compute_reputation_override"
	EventTypeReputationOverrideClear = "compute_reputation_override_clear"
)
