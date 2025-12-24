package types

import (
	"errors"

	sdkerrors "cosmossdk.io/errors"
)

// Oracle module sentinel errors
var (
	// Asset and price errors
	ErrInvalidAsset   = sdkerrors.Register(ModuleName, 2, "invalid asset")
	ErrInvalidNonce   = sdkerrors.Register(ModuleName, 50, "invalid packet nonce")
	ErrInvalidPrice   = sdkerrors.Register(ModuleName, 3, "invalid price")
	ErrPriceNotFound  = sdkerrors.Register(ModuleName, 7, "price not found")
	ErrPriceExpired   = sdkerrors.Register(ModuleName, 12, "price data expired")
	ErrPriceDeviation = sdkerrors.Register(ModuleName, 13, "price deviation too high")
	ErrInvalidAck     = sdkerrors.Register(ModuleName, 90, "invalid acknowledgement")

	// Validator and feeder errors
	ErrValidatorNotBonded  = sdkerrors.Register(ModuleName, 4, "validator not bonded")
	ErrFeederNotAuthorized = sdkerrors.Register(ModuleName, 5, "feeder not authorized")
	ErrValidatorNotFound   = sdkerrors.Register(ModuleName, 8, "validator not found")
	ErrValidatorSlashed    = sdkerrors.Register(ModuleName, 14, "validator has been slashed")
	ErrUnauthorizedChannel = sdkerrors.Register(ModuleName, 60, "unauthorized IBC channel")

	// Vote and submission errors
	ErrInsufficientVotes   = sdkerrors.Register(ModuleName, 6, "insufficient votes")
	ErrInvalidVotePeriod   = sdkerrors.Register(ModuleName, 9, "invalid vote period")
	ErrDuplicateSubmission = sdkerrors.Register(ModuleName, 15, "duplicate price submission")
	ErrMissedVote          = sdkerrors.Register(ModuleName, 16, "validator missed vote")

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
	ErrInsufficientDataSources     = sdkerrors.Register(ModuleName, 30, "insufficient data sources")
	ErrOutlierDetected             = sdkerrors.Register(ModuleName, 31, "price outlier detected")
	ErrMedianCalculationFailed     = sdkerrors.Register(ModuleName, 32, "median calculation failed")
	ErrInsufficientOracleConsensus = sdkerrors.Register(ModuleName, 33, "insufficient voting power for oracle consensus")

	// State errors
	ErrStateCorruption = sdkerrors.Register(ModuleName, 40, "state corruption detected")
	ErrOracleInactive  = sdkerrors.Register(ModuleName, 41, "oracle is inactive")

	// Data availability errors
	ErrOracleDataUnavailable = sdkerrors.Register(ModuleName, 42, "oracle data unavailable")

	// Price source validation errors
	ErrInvalidPriceSource = sdkerrors.Register(ModuleName, 43, "invalid price source")

	// Geographic location verification errors
	ErrInvalidIPAddress          = sdkerrors.Register(ModuleName, 44, "invalid IP address")
	ErrIPRegionMismatch          = sdkerrors.Register(ModuleName, 45, "IP address does not match claimed region")
	ErrPrivateIPNotAllowed       = sdkerrors.Register(ModuleName, 46, "private IP addresses not allowed for validators")
	ErrLocationProofRequired     = sdkerrors.Register(ModuleName, 47, "location proof required")
	ErrLocationProofInvalid      = sdkerrors.Register(ModuleName, 48, "location proof invalid or expired")
	ErrInsufficientGeoDiversity  = sdkerrors.Register(ModuleName, 49, "insufficient geographic diversity")
	ErrGeoIPDatabaseUnavailable  = sdkerrors.Register(ModuleName, 51, "GeoIP database unavailable")
	ErrTooManyValidatorsFromSameIP = sdkerrors.Register(ModuleName, 52, "too many validators from same IP address")

	// Circuit breaker errors
	ErrCircuitBreakerAlreadyOpen   = sdkerrors.Register(ModuleName, 53, "circuit breaker already open")
	ErrCircuitBreakerAlreadyClosed = sdkerrors.Register(ModuleName, 54, "circuit breaker already closed")

	// Emergency pause errors
	ErrOraclePaused            = sdkerrors.Register(ModuleName, 55, "oracle is currently paused")
	ErrOracleNotPaused         = sdkerrors.Register(ModuleName, 56, "oracle is not currently paused")
	ErrUnauthorizedPause       = sdkerrors.Register(ModuleName, 57, "unauthorized to trigger emergency pause")
	ErrUnauthorizedResume      = sdkerrors.Register(ModuleName, 58, "unauthorized to resume oracle")
	ErrInvalidEmergencyAdmin   = sdkerrors.Register(ModuleName, 59, "invalid emergency admin address")
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
	ErrInvalidAsset:   "Asset symbol not recognized. Check supported asset list using query. Ensure asset denom is registered. Use correct format (e.g., 'BTC', 'ETH', 'ATOM').",
	ErrInvalidPrice:   "Price must be positive and within reasonable bounds. Check for decimal overflow. Verify price source data quality. Ensure price is recent (< 1 hour old).",
	ErrPriceNotFound:  "No price data available for this asset. Wait for next vote period (typically 30 seconds). Check if asset is actively tracked. Query available assets.",
	ErrPriceExpired:   "Price data is stale (older than max age). Wait for validators to submit new prices. Check validator liveness. Query latest update timestamp.",
	ErrPriceDeviation: "Price deviates significantly from current median (>10%). Verify your price sources are accurate. Check for market volatility. May indicate oracle manipulation attempt.",

	ErrValidatorNotBonded:  "Validator must be bonded to submit prices. Check validator status. Ensure sufficient stake. Wait for bonding period to complete (typically 21 days).",
	ErrFeederNotAuthorized: "Feeder address not authorized by validator. Validator must delegate feeder using MsgDelegateFeeder. Verify feeder address matches delegation.",
	ErrValidatorNotFound:   "Validator address not found in staking module. Check validator address format (bech32). Ensure validator is registered. Query validator set.",
	ErrValidatorSlashed:    "Validator slashed for oracle misbehavior. Cannot submit prices until penalty period expires. Check slashing status. Wait for jail period to end.",
	ErrUnauthorizedChannel: "IBC packet arrived from a channel that is not authorized for oracle traffic. Verify governance params and ensure relayers use approved channels before retrying.",

	ErrInsufficientVotes:   "Not enough validators submitted prices this period. Need >66% participation. Wait for more validators to submit. Check network connectivity.",
	ErrInvalidVotePeriod:   "Vote period parameter out of valid range (1-3600 seconds). Update params requires governance proposal. Check current param values.",
	ErrDuplicateSubmission: "Price already submitted for this vote period. Wait for next period to submit again. Each validator can submit once per period.",
	ErrMissedVote:          "Validator missed too many consecutive votes. Risk of slashing after threshold (typically 10 misses). Ensure oracle service is running. Check automation.",

	ErrInvalidThreshold:     "Vote threshold must be 0.50-1.00 (50%-100%). Update requires governance proposal. Recommended: 0.67 (67%) for Byzantine fault tolerance.",
	ErrInvalidSlashFraction: "Slash fraction must be 0.00-1.00 (0%-100%). Recommended: 0.01 (1%) for oracle violations. Update requires governance.",

	ErrCircuitBreakerActive: "Oracle circuit breaker triggered due to anomaly detection. Automatic recovery in progress. Wait for reset period. Check system status. No manual intervention needed.",
	ErrRateLimitExceeded:    "Too many price submissions in time window. Each feeder limited to 1 submission per vote period. Wait for next period. Check for duplicate submission logic.",
	ErrSybilAttackDetected:  "SECURITY: Multiple price sources from same entity detected. Diversity check failed. Use independent price sources. Ensure geographic distribution.",
	ErrFlashLoanDetected:    "SECURITY: Flash loan attack pattern in price data. Unusual price spike detected and rejected. This is expected security behavior. Submit legitimate price next period.",
	ErrDataPoisoning:        "SECURITY: Price data failed authenticity verification. Source validation failed. Check API credentials. Verify price source is legitimate. Contact price provider.",

	ErrInsufficientDataSources:     "Need minimum number of independent price sources (typically 3). Add more data sources to feeder config. Check that sources are operational.",
	ErrOutlierDetected:             "Submitted price is statistical outlier (>3 standard deviations). Verify price source accuracy. Check for API issues. Compare with other exchanges.",
	ErrMedianCalculationFailed:     "Cannot calculate median from submitted prices. Insufficient valid prices. Check validator participation. Wait for more submissions.",
	ErrInsufficientOracleConsensus: "SECURITY: Insufficient voting power after outlier filtering. After removing outliers, remaining validators have less than minimum required voting power (typically 10%). This prevents price manipulation by multiple low-stake validators. Wait for more high-stake validators to submit prices.",

	ErrStateCorruption:       "CRITICAL: Oracle state corruption detected. Automatic recovery initiated using backup. Price feeds may be temporarily unavailable. Contact validators.",
	ErrOracleDataUnavailable: "No oracle data available for requested asset. Wait for next aggregation interval or ensure feeder connectivity. Confirm asset is registered and active.",
	ErrInvalidPriceSource:    "Price source failed validation or is unregistered. Verify source registration, reputation, and heartbeat before retrying.",

	ErrInvalidIPAddress:          "IP address format is invalid. Provide a valid IPv4 or IPv6 address. Check network configuration.",
	ErrIPRegionMismatch:          "SECURITY: IP address geolocation does not match claimed region. Update claimed region to match actual location or fix IP address. This prevents location spoofing.",
	ErrPrivateIPNotAllowed:       "SECURITY: Validators must use public IP addresses. Private/localhost IPs (10.x, 192.168.x, 127.x) are not allowed. Configure proper public network access.",
	ErrLocationProofRequired:     "Geographic location proof required for validator registration. Provide verifiable location evidence. This ensures geographic diversity.",
	ErrLocationProofInvalid:      "Location proof verification failed or expired. Renew location proof with current timestamp. Ensure proof is cryptographically signed and valid.",
	ErrInsufficientGeoDiversity:  "SECURITY: Insufficient geographic diversity among validators. Need minimum 3 distinct regions. This is critical for decentralization and resilience.",
	ErrGeoIPDatabaseUnavailable:  "GeoIP database not available for location verification. Download GeoLite2-Country.mmdb and set GEOIP_DB_PATH environment variable.",
	ErrTooManyValidatorsFromSameIP: "SECURITY: Too many validators sharing same IP address (max 2). This indicates centralization risk or Sybil attack. Ensure validators are independently operated.",

	ErrOraclePaused:          "Oracle module is currently paused due to emergency. Price submissions are rejected. Existing prices can still be read but may be stale. Wait for governance to resume operations or check pause reason.",
	ErrOracleNotPaused:       "Oracle is not currently paused. Cannot resume operations that are already running. Check oracle status before retrying.",
	ErrUnauthorizedPause:     "SECURITY: Only the emergency admin or governance authority can trigger emergency pause. Contact the oracle administrator or submit a governance proposal.",
	ErrUnauthorizedResume:    "SECURITY: Only governance authority can resume oracle operations after emergency pause. This prevents abuse of pause mechanism. Submit a governance proposal to resume.",
	ErrInvalidEmergencyAdmin: "Emergency admin address format is invalid. Provide a valid bech32 address or empty string to disable admin capability. Update via governance proposal.",
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
