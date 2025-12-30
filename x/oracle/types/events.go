package types

// Event types for the Oracle module
// All event types use lowercase with underscore separator (module_action format)
const (
	// Price events
	EventTypeOraclePriceUpdate     = "oracle_price_update"
	EventTypeOraclePriceSubmitted  = "oracle_price_submitted"
	EventTypeOraclePriceAggregated = "oracle_price_aggregated"
	EventTypeOracleFallback        = "oracle_fallback" // DATA-12: Tiered fallback when all prices filtered

	// Voting events
	EventTypeOracleVote            = "oracle_vote"
	EventTypeOracleVoteAggregated  = "oracle_vote_aggregated"
	EventTypeOracleFeederDelegated = "oracle_feeder_delegated"

	// Security events
	EventTypeOracleSlash        = "oracle_slash"
	EventTypeOracleSlashOutlier = "oracle_slash_outlier"
	EventTypeOracleJail         = "oracle_jail"
	EventTypeOracleOutlier      = "oracle_outlier"

	// Cross-chain events
	EventTypeOracleCrossChainPrice = "oracle_cross_chain_price"
	EventTypeOraclePriceRelay      = "oracle_price_relay"

	// Parameter events
	EventTypeOracleParamsUpdated = "oracle_params_updated"
)

// Event attribute keys for the Oracle module
// All attribute keys use lowercase with underscore separator
const (
	// Asset attributes
	AttributeKeyAsset  = "asset"
	AttributeKeyAssets = "assets"
	AttributeKeyDenom  = "denom"

	// Price attributes
	AttributeKeyPrice  = "price"
	AttributeKeyPrices = "prices"
	AttributeKeyMedian = "median"
	AttributeKeyMean   = "mean"
	AttributeKeyStdDev = "std_dev"
	AttributeKeyMAD    = "mad"

	// Validator attributes
	AttributeKeyValidator        = "validator"
	AttributeKeyValidators       = "validators"
	AttributeKeyFeeder           = "feeder"
	AttributeKeyDelegate         = "delegate"
	AttributeKeyVotingPower      = "voting_power"
	AttributeKeyNumValidators    = "num_validators"
	AttributeKeyTotalVotingPower = "total_voting_power"

	// Deviation attributes
	AttributeKeyDeviation        = "deviation"
	AttributeKeyDeviationPercent = "deviation_percent"
	AttributeKeyThreshold        = "threshold"

	// Slashing attributes
	AttributeKeyReason        = "reason"
	AttributeKeyDetails       = "details"
	AttributeKeySlashFraction = "slash_fraction"
	AttributeKeySlashAmount   = "slash_amount"
	AttributeKeySeverity      = "severity"
	AttributeKeyJailed        = "jailed"

	// Block attributes
	AttributeKeyBlockHeight = "block_height"
	AttributeKeyTimestamp   = "timestamp"

	// Aggregation attributes
	AttributeKeyNumSubmissions = "num_submissions"
	AttributeKeyNumOutliers    = "num_outliers"
	AttributeKeyConfidence     = "confidence"

	// Cross-chain attributes (excluding IBC-specific ones in types.go)
	AttributeKeySourceChain = "source_chain"
	AttributeKeyTargetChain = "target_chain"

	// Status attributes
	AttributeKeyStatus = "status"
	AttributeKeyError  = "error"
	AttributeKeyActor  = "actor"

	// Circuit breaker attributes
	AttributeKeyPair     = "pair"
	AttributeKeyFeedType = "feed_type"
)

// Circuit breaker event types
const (
	EventTypeCircuitBreakerOpen  = "oracle_circuit_breaker_open"
	EventTypeCircuitBreakerClose = "oracle_circuit_breaker_close"
	EventTypePriceOverride       = "oracle_price_override"
	EventTypePriceOverrideClear  = "oracle_price_override_clear"
	EventTypeSlashingDisabled    = "oracle_slashing_disabled"
	EventTypeSlashingEnabled     = "oracle_slashing_enabled"
)

// Emergency pause event types
const (
	EventTypeEmergencyPause  = "oracle_emergency_pause"
	EventTypeEmergencyResume = "oracle_emergency_resume"
)

// Emergency pause attribute keys
const (
	AttributeKeyPausedBy     = "paused_by"
	AttributeKeyPauseReason  = "pause_reason"
	AttributeKeyResumeReason = "resume_reason"
)
