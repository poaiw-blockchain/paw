package types

import (
	"errors"

	sdkerrors "cosmossdk.io/errors"
)

// Oracle module sentinel errors
var (
	// Asset and price errors
	ErrInvalidAsset         = sdkerrors.Register(ModuleName, 2, "invalid asset")
	ErrInvalidPrice         = sdkerrors.Register(ModuleName, 3, "invalid price")
	ErrPriceNotFound        = sdkerrors.Register(ModuleName, 7, "price not found")
	ErrPriceExpired         = sdkerrors.Register(ModuleName, 12, "price data expired")
	ErrPriceDeviation       = sdkerrors.Register(ModuleName, 13, "price deviation too high")

	// Validator and feeder errors
	ErrValidatorNotBonded   = sdkerrors.Register(ModuleName, 4, "validator not bonded")
	ErrFeederNotAuthorized  = sdkerrors.Register(ModuleName, 5, "feeder not authorized")
	ErrValidatorNotFound    = sdkerrors.Register(ModuleName, 8, "validator not found")
	ErrValidatorSlashed     = sdkerrors.Register(ModuleName, 14, "validator has been slashed")

	// Vote and submission errors
	ErrInsufficientVotes    = sdkerrors.Register(ModuleName, 6, "insufficient votes")
	ErrInvalidVotePeriod    = sdkerrors.Register(ModuleName, 9, "invalid vote period")
	ErrDuplicateSubmission  = sdkerrors.Register(ModuleName, 15, "duplicate price submission")
	ErrMissedVote           = sdkerrors.Register(ModuleName, 16, "validator missed vote")

	// Parameter errors
	ErrInvalidThreshold     = sdkerrors.Register(ModuleName, 10, "invalid threshold")
	ErrInvalidSlashFraction = sdkerrors.Register(ModuleName, 11, "invalid slash fraction")

	// Security errors
	ErrCircuitBreakerActive = sdkerrors.Register(ModuleName, 20, "circuit breaker is active")
	ErrRateLimitExceeded    = sdkerrors.Register(ModuleName, 21, "rate limit exceeded")
	ErrSybilAttackDetected  = sdkerrors.Register(ModuleName, 22, "Sybil attack detected")
	ErrFlashLoanDetected    = sdkerrors.Register(ModuleName, 23, "flash loan attack detected")
	ErrDataPoisoning        = sdkerrors.Register(ModuleName, 24, "data poisoning attempt detected")

	// Aggregation errors
	ErrInsufficientDataSources = sdkerrors.Register(ModuleName, 30, "insufficient data sources")
	ErrOutlierDetected         = sdkerrors.Register(ModuleName, 31, "price outlier detected")
	ErrMedianCalculationFailed = sdkerrors.Register(ModuleName, 32, "median calculation failed")

	// State errors
	ErrStateCorruption = sdkerrors.Register(ModuleName, 40, "state corruption detected")
	ErrOracleInactive  = sdkerrors.Register(ModuleName, 41, "oracle is inactive")
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
	ErrInvalidAsset: "Asset symbol not recognized. Check supported asset list using query. Ensure asset denom is registered. Use correct format (e.g., 'BTC', 'ETH', 'ATOM').",
	ErrInvalidPrice: "Price must be positive and within reasonable bounds. Check for decimal overflow. Verify price source data quality. Ensure price is recent (< 1 hour old).",
	ErrPriceNotFound: "No price data available for this asset. Wait for next vote period (typically 30 seconds). Check if asset is actively tracked. Query available assets.",
	ErrPriceExpired: "Price data is stale (older than max age). Wait for validators to submit new prices. Check validator liveness. Query latest update timestamp.",
	ErrPriceDeviation: "Price deviates significantly from current median (>10%). Verify your price sources are accurate. Check for market volatility. May indicate oracle manipulation attempt.",

	ErrValidatorNotBonded: "Validator must be bonded to submit prices. Check validator status. Ensure sufficient stake. Wait for bonding period to complete (typically 21 days).",
	ErrFeederNotAuthorized: "Feeder address not authorized by validator. Validator must delegate feeder using MsgDelegateFeeder. Verify feeder address matches delegation.",
	ErrValidatorNotFound: "Validator address not found in staking module. Check validator address format (bech32). Ensure validator is registered. Query validator set.",
	ErrValidatorSlashed: "Validator slashed for oracle misbehavior. Cannot submit prices until penalty period expires. Check slashing status. Wait for jail period to end.",

	ErrInsufficientVotes: "Not enough validators submitted prices this period. Need >66% participation. Wait for more validators to submit. Check network connectivity.",
	ErrInvalidVotePeriod: "Vote period parameter out of valid range (1-3600 seconds). Update params requires governance proposal. Check current param values.",
	ErrDuplicateSubmission: "Price already submitted for this vote period. Wait for next period to submit again. Each validator can submit once per period.",
	ErrMissedVote: "Validator missed too many consecutive votes. Risk of slashing after threshold (typically 10 misses). Ensure oracle service is running. Check automation.",

	ErrInvalidThreshold: "Vote threshold must be 0.50-1.00 (50%-100%). Update requires governance proposal. Recommended: 0.67 (67%) for Byzantine fault tolerance.",
	ErrInvalidSlashFraction: "Slash fraction must be 0.00-1.00 (0%-100%). Recommended: 0.01 (1%) for oracle violations. Update requires governance.",

	ErrCircuitBreakerActive: "Oracle circuit breaker triggered due to anomaly detection. Automatic recovery in progress. Wait for reset period. Check system status. No manual intervention needed.",
	ErrRateLimitExceeded: "Too many price submissions in time window. Each feeder limited to 1 submission per vote period. Wait for next period. Check for duplicate submission logic.",
	ErrSybilAttackDetected: "SECURITY: Multiple price sources from same entity detected. Diversity check failed. Use independent price sources. Ensure geographic distribution.",
	ErrFlashLoanDetected: "SECURITY: Flash loan attack pattern in price data. Unusual price spike detected and rejected. This is expected security behavior. Submit legitimate price next period.",
	ErrDataPoisoning: "SECURITY: Price data failed authenticity verification. Source validation failed. Check API credentials. Verify price source is legitimate. Contact price provider.",

	ErrInsufficientDataSources: "Need minimum number of independent price sources (typically 3). Add more data sources to feeder config. Check that sources are operational.",
	ErrOutlierDetected: "Submitted price is statistical outlier (>3 standard deviations). Verify price source accuracy. Check for API issues. Compare with other exchanges.",
	ErrMedianCalculationFailed: "Cannot calculate median from submitted prices. Insufficient valid prices. Check validator participation. Wait for more submissions.",

	ErrStateCorruption: "CRITICAL: Oracle state corruption detected. Automatic recovery initiated using backup. Price feeds may be temporarily unavailable. Contact validators.",
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

	return "No recovery suggestion available. Check error message for details. Query oracle status and validator participation."
}
