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

	// IBC event types (kept here for IBC-specific events)
	EventTypeChannelOpen        = "oracle_channel_open"
	EventTypeChannelOpenAck     = "oracle_channel_open_ack"
	EventTypeChannelOpenConfirm = "oracle_channel_open_confirm"
	EventTypeChannelClose       = "oracle_channel_close"
	EventTypePacketReceive      = "oracle_packet_receive"
	EventTypePacketAck          = "oracle_packet_ack"
	EventTypePacketTimeout      = "oracle_packet_timeout"

	// IBC attribute keys
	AttributeKeyChannelID             = "channel_id"
	AttributeKeyPortID                = "port_id"
	AttributeKeyCounterpartyPortID    = "counterparty_port_id"
	AttributeKeyCounterpartyChannelID = "counterparty_channel_id"
	AttributeKeyPacketType            = "packet_type"
	AttributeKeySequence              = "sequence"
	AttributeKeyAckSuccess            = "ack_success"
	AttributeKeyPendingOperations     = "pending_operations"
)
