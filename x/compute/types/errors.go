package types

import (
	"errors"

	sdkerrors "cosmossdk.io/errors"
)

// Compute module sentinel errors with recovery suggestions

var (
	// Request validation errors
	ErrInvalidRequest  = sdkerrors.Register(ModuleName, 2, "invalid compute request")
	ErrInvalidProvider = sdkerrors.Register(ModuleName, 3, "invalid provider")
	ErrInvalidResult   = sdkerrors.Register(ModuleName, 4, "invalid computation result")
	ErrInvalidProof    = sdkerrors.Register(ModuleName, 5, "invalid verification proof")

	// Provider errors
	ErrProviderNotFound   = sdkerrors.Register(ModuleName, 10, "provider not found")
	ErrProviderNotActive  = sdkerrors.Register(ModuleName, 11, "provider not active")
	ErrProviderOverloaded = sdkerrors.Register(ModuleName, 12, "provider overloaded")
	ErrInsufficientStake  = sdkerrors.Register(ModuleName, 13, "insufficient provider stake")
	ErrProviderSlashed    = sdkerrors.Register(ModuleName, 14, "provider has been slashed")

	// Request lifecycle errors
	ErrRequestNotFound         = sdkerrors.Register(ModuleName, 20, "compute request not found")
	ErrRequestExpired          = sdkerrors.Register(ModuleName, 21, "compute request has expired")
	ErrRequestAlreadyCompleted = sdkerrors.Register(ModuleName, 22, "request already completed")
	ErrRequestCancelled        = sdkerrors.Register(ModuleName, 23, "request has been cancelled")

	// Escrow errors
	ErrInsufficientEscrow = sdkerrors.Register(ModuleName, 30, "insufficient escrow balance")
	ErrEscrowLocked       = sdkerrors.Register(ModuleName, 31, "escrow is locked")
	ErrEscrowNotFound     = sdkerrors.Register(ModuleName, 32, "escrow not found")
	ErrEscrowRefundFailed = sdkerrors.Register(ModuleName, 33, "escrow refund failed")

	// Verification errors
	ErrVerificationFailed     = sdkerrors.Register(ModuleName, 40, "result verification failed")
	ErrInvalidSignature       = sdkerrors.Register(ModuleName, 41, "invalid signature")
	ErrInvalidMerkleProof     = sdkerrors.Register(ModuleName, 42, "invalid merkle proof")
	ErrInvalidStateCommitment = sdkerrors.Register(ModuleName, 43, "invalid state commitment")
	ErrReplayAttackDetected   = sdkerrors.Register(ModuleName, 44, "replay attack detected")
	ErrProofExpired           = sdkerrors.Register(ModuleName, 45, "proof has expired")

	// ZK proof errors
	ErrInvalidZKProof        = sdkerrors.Register(ModuleName, 50, "invalid zero-knowledge proof")
	ErrZKVerificationFailed  = sdkerrors.Register(ModuleName, 51, "zero-knowledge verification failed")
	ErrInvalidCircuit        = sdkerrors.Register(ModuleName, 52, "invalid verification circuit")
	ErrInvalidPublicInputs   = sdkerrors.Register(ModuleName, 53, "invalid public inputs")
	ErrInvalidWitness        = sdkerrors.Register(ModuleName, 54, "invalid witness")
	ErrProofTooLarge         = sdkerrors.Register(ModuleName, 55, "proof size exceeds maximum allowed")
	ErrInsufficientDeposit   = sdkerrors.Register(ModuleName, 56, "insufficient deposit for proof verification")
	ErrDepositTransferFailed = sdkerrors.Register(ModuleName, 57, "failed to transfer verification deposit")

	// Resource errors
	ErrInsufficientResources = sdkerrors.Register(ModuleName, 60, "insufficient computational resources")
	ErrResourceQuotaExceeded = sdkerrors.Register(ModuleName, 61, "resource quota exceeded")
	ErrInvalidResourceSpec   = sdkerrors.Register(ModuleName, 62, "invalid resource specification")

	// Security errors
	ErrUnauthorized         = sdkerrors.Register(ModuleName, 70, "unauthorized operation")
	ErrRateLimitExceeded    = sdkerrors.Register(ModuleName, 71, "rate limit exceeded")
	ErrCircuitBreakerActive = sdkerrors.Register(ModuleName, 72, "circuit breaker is active")
	ErrSuspiciousActivity   = sdkerrors.Register(ModuleName, 73, "suspicious activity detected")

	// IBC errors
	ErrInvalidPacket          = sdkerrors.Register(ModuleName, 80, "invalid IBC packet")
	ErrChannelNotFound        = sdkerrors.Register(ModuleName, 81, "IBC channel not found")
	ErrPacketTimeout          = sdkerrors.Register(ModuleName, 82, "packet timeout")
	ErrInvalidAcknowledgement = sdkerrors.Register(ModuleName, 83, "invalid acknowledgement")
	ErrInvalidNonce           = sdkerrors.Register(ModuleName, 84, "invalid packet nonce")
	ErrInvalidAck             = ErrInvalidAcknowledgement
	ErrUnauthorizedChannel    = sdkerrors.Register(ModuleName, 85, "unauthorized IBC channel")

	// Circuit breaker errors
	ErrCircuitBreakerAlreadyOpen   = sdkerrors.Register(ModuleName, 86, "circuit breaker already open")
	ErrCircuitBreakerAlreadyClosed = sdkerrors.Register(ModuleName, 87, "circuit breaker already closed")
)

// ErrorWithRecovery wraps an error with recovery suggestions
type ErrorWithRecovery struct {
	Err      error
	Recovery string
}

func (e *ErrorWithRecovery) Error() string {
	return e.Err.Error()
}

func (e *ErrorWithRecovery) Unwrap() error {
	return e.Err
}

// RecoverySuggestions provides actionable recovery steps for each error type
var RecoverySuggestions = map[error]string{
	ErrInvalidRequest:  "Check request parameters: ensure container image, environment variables, and resource requirements are valid. Verify request signature and timestamps.",
	ErrInvalidProvider: "Verify provider address format (bech32). Ensure provider is registered and active. Check provider's reputation score.",
	ErrInvalidResult:   "Verify result data format and hash. Check that result matches request specification. Ensure provider signature is valid.",
	ErrInvalidProof:    "Validate proof structure: signature (64 bytes), public key (32 bytes), merkle root (32 bytes). Ensure nonce is unique and timestamp is recent.",

	ErrProviderNotFound:   "Register as a provider using MsgRegisterProvider. Ensure stake amount meets minimum requirement (query params). Verify provider address is correct.",
	ErrProviderNotActive:  "Provider must activate registration. Check if provider was deactivated due to poor performance. Verify sufficient stake and no active slashing.",
	ErrProviderOverloaded: "Provider is at capacity. Wait for current jobs to complete or select a different provider. Check provider's available resources.",
	ErrInsufficientStake:  "Increase provider stake using MsgStakeProvider. Minimum stake requirement can be queried from params. Ensure tokens are available in account.",
	ErrProviderSlashed:    "Provider was slashed for misbehavior. Cannot submit new requests until penalty period expires. Check slashing status and wait for recovery period.",

	ErrRequestNotFound:         "Verify request ID is correct. Check if request was cancelled or expired. Query request status using gRPC/REST API.",
	ErrRequestExpired:          "Request exceeded timeout period. Submit a new request with longer timeout. Check network delays and provider availability.",
	ErrRequestAlreadyCompleted: "Result was already submitted and verified. Query request details to retrieve result. Cannot resubmit for completed request.",
	ErrRequestCancelled:        "Request was cancelled by requester. Submit a new request if computation is still needed. Check cancellation reason in events.",

	ErrInsufficientEscrow: "Deposit sufficient tokens for computation cost. Calculate required amount: base_price + resource_fees + verification_fees. Query provider pricing.",
	ErrEscrowLocked:       "Escrow is locked for active computation. Wait for result submission or timeout. Cannot withdraw while job is running.",
	ErrEscrowNotFound:     "No escrow found for this request. Create escrow when submitting request. Verify request ID is correct.",
	ErrEscrowRefundFailed: "Automatic refund failed. Claim refund manually using MsgClaimRefund. Check bank module balance and permissions.",

	ErrVerificationFailed:     "Result failed verification checks. Review verification logs for specific failure. Provider may be penalized. Submit new request with different provider.",
	ErrInvalidSignature:       "Ed25519 signature verification failed. Ensure signature is 64 bytes. Verify public key matches provider. Check message hash construction.",
	ErrInvalidMerkleProof:     "Merkle proof validation failed. Verify proof path has valid 32-byte nodes. Check merkle root matches. Ensure execution trace is complete.",
	ErrInvalidStateCommitment: "State commitment hash mismatch. Verify computation state is deterministic. Check for non-deterministic operations (time, randomness).",
	ErrReplayAttackDetected:   "Nonce was already used. Generate new unique nonce. Check nonce tracking storage. Ensure proof is freshly generated.",
	ErrProofExpired:           "Proof timestamp exceeded validity period. Generate new proof with current timestamp. Maximum age is configurable in params.",

	ErrInvalidZKProof:        "Zero-knowledge proof structure invalid. Verify proof matches circuit specification. Check proof size and format. Ensure correct circuit version.",
	ErrZKVerificationFailed:  "ZK proof verification failed. Public inputs may be incorrect. Verify witness is valid. Check circuit constraints are satisfied.",
	ErrInvalidCircuit:        "Verification circuit not found or invalid. Use supported circuit types. Query available circuits. Verify circuit version compatibility.",
	ErrInvalidPublicInputs:   "Public inputs don't match circuit requirements. Check input count and types. Verify input encoding. See circuit documentation.",
	ErrInvalidWitness:        "Private witness invalid or incomplete. Generate witness from valid computation trace. Verify witness satisfies all constraints.",
	ErrProofTooLarge:         "Proof exceeds maximum allowed size. Reduce proof size or check circuit configuration. Maximum proof size is defined in circuit parameters to prevent DoS attacks.",
	ErrInsufficientDeposit:   "Insufficient deposit for ZK proof verification. Query circuit params for required deposit amount. Deposit is refunded on valid proof, slashed on invalid proof.",
	ErrDepositTransferFailed: "Failed to transfer verification deposit. Check account balance and module permissions. Ensure account has sufficient funds for deposit.",

	ErrInsufficientResources: "Provider lacks required resources. Reduce CPU/memory/storage requirements. Select provider with higher capacity. Split computation into smaller jobs.",
	ErrResourceQuotaExceeded: "Account resource quota exceeded. Wait for quota reset period. Upgrade account tier. Optimize resource usage.",
	ErrInvalidResourceSpec:   "Resource specification invalid. CPU cores must be 1-64, memory 1-256GB, storage 1-1000GB. GPU/TEE are boolean flags.",

	ErrUnauthorized:         "Operation not permitted for this account. Check if you're the request owner. Verify message signer. Review access control policies.",
	ErrUnauthorizedChannel:  "IBC packet arrived via unauthorized channel. Confirm governance-approved channel list before relaying packets. Update params via proposal after channel handshake completes.",
	ErrRateLimitExceeded:    "Too many requests in time window. Wait for rate limit reset. Upgrade account tier for higher limits. Batch operations when possible.",
	ErrCircuitBreakerActive: "Module circuit breaker triggered due to anomaly. Wait for automatic recovery or admin intervention. Check system status page.",
	ErrSuspiciousActivity:   "Unusual pattern detected. Verify legitimate use case. Contact support if flagged incorrectly. Review activity logs.",

	ErrInvalidPacket:          "IBC packet validation failed. Check packet structure matches spec. Verify all required fields. Ensure correct packet type.",
	ErrChannelNotFound:        "IBC channel doesn't exist. Create channel using IBC connection. Verify channel ID. Check counterparty chain status.",
	ErrPacketTimeout:          "IBC packet timed out. Increase timeout height/timestamp. Check counterparty chain liveness. Verify relayer is running.",
	ErrInvalidAcknowledgement: "Acknowledgement validation failed. Check ack data format. Verify success/error fields. Review counterparty response.",
}

// WrapWithRecovery wraps an error with recovery suggestion
func WrapWithRecovery(err error, msg string, args ...interface{}) error {
	wrapped := sdkerrors.Wrapf(err, msg, args...)

	if suggestion, ok := RecoverySuggestions[err]; ok {
		return &ErrorWithRecovery{
			Err:      wrapped,
			Recovery: suggestion,
		}
	}

	return wrapped
}

// GetRecoverySuggestion returns the recovery suggestion for an error
func GetRecoverySuggestion(err error) string {
	// Unwrap to find the root error
	rootErr := err
	for {
		if unwrapped := errors.Unwrap(rootErr); unwrapped != nil {
			rootErr = unwrapped
		} else {
			break
		}
	}

	if suggestion, ok := RecoverySuggestions[rootErr]; ok {
		return suggestion
	}

	return "No recovery suggestion available. Check error message for details."
}

func init() {
	if _, ok := RecoverySuggestions[ErrInvalidNonce]; !ok {
		RecoverySuggestions[ErrInvalidNonce] = "Nonce must be unique and increasing. Check nonce tracking state and ensure you include the latest outbound nonce per channel."
	}
}
