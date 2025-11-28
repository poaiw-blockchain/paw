package types

const (
	// ModuleName defines the module name
	ModuleName = "oracle"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// PortID is the default port ID for oracle IBC module
	PortID = "oracle"
)

// Event types for the oracle module
const (
	EventTypePriceUpdated       = "price_updated"
	EventTypePriceSubmitted     = "price_submitted"
	EventTypeFeederDelegated    = "feeder_delegated"
	EventTypeOracleSlash        = "oracle_slash"
	EventTypeOracleSlashOutlier = "oracle_slash_outlier"
	EventTypeOracleJail         = "oracle_jail"
	EventTypeParamsUpdated      = "params_updated"
	EventTypePriceAggregated    = "price_aggregated"
	EventTypeOutlierDetected    = "oracle_outlier_detected"

	// IBC event types
	EventTypeChannelOpen        = "channel_open"
	EventTypeChannelOpenAck     = "channel_open_ack"
	EventTypeChannelOpenConfirm = "channel_open_confirm"
	EventTypeChannelClose       = "channel_close"
	EventTypePacketReceive      = "packet_receive"
	EventTypePacketAck          = "packet_ack"
	EventTypePacketTimeout      = "packet_timeout"

	AttributeKeyAsset         = "asset"
	AttributeKeyPrice         = "price"
	AttributeKeyValidator     = "validator"
	AttributeKeyFeeder        = "feeder"
	AttributeKeyDelegate      = "delegate"
	AttributeKeyVotingPower   = "voting_power"
	AttributeKeyBlockHeight   = "block_height"
	AttributeKeyNumValidators = "num_validators"
	AttributeKeyReason        = "reason"
	AttributeKeySlashFraction = "slash_fraction"
	AttributeKeySeverity      = "severity"
	AttributeKeyDeviation     = "deviation"
	AttributeKeyMedian        = "median"
	AttributeKeyMAD           = "mad"
	AttributeKeyNumOutliers   = "num_outliers"
	AttributeKeyJailed        = "jailed"

	// IBC attribute keys
	AttributeKeyChannelID             = "channel_id"
	AttributeKeyPortID                = "port_id"
	AttributeKeyCounterpartyPortID    = "counterparty_port_id"
	AttributeKeyCounterpartyChannelID = "counterparty_channel_id"
	AttributeKeyPacketType            = "packet_type"
	AttributeKeySequence              = "sequence"
	AttributeKeyAckSuccess            = "ack_success"
)
