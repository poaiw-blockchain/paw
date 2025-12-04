package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the module name
	ModuleName = "dex"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_" + ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// PortID is the default port ID for dex IBC module
	PortID = "dex"

	// Event types
	EventTypeChannelOpen        = "channel_open"
	EventTypeChannelOpenAck     = "channel_open_ack"
	EventTypeChannelOpenConfirm = "channel_open_confirm"
	EventTypeChannelClose       = "channel_close"
	EventTypePacketReceive      = "packet_receive"
	EventTypePacketAck          = "packet_ack"
	EventTypePacketTimeout      = "packet_timeout"

	// Event attribute keys
	AttributeKeyChannelID             = "channel_id"
	AttributeKeyPortID                = "port_id"
	AttributeKeyCounterpartyPortID    = "counterparty_port_id"
	AttributeKeyCounterpartyChannelID = "counterparty_channel_id"
	AttributeKeyPacketType            = "packet_type"
	AttributeKeySequence              = "sequence"
	AttributeKeyAckSuccess            = "ack_success"
	AttributeKeyPendingOperations     = "pending_operations"
)

// DefaultParams returns default parameters for the dex module
func DefaultParams() Params {
	return Params{
		SwapFee:                   math.LegacyNewDecWithPrec(3, 3),  // 0.3%
		LpFee:                     math.LegacyNewDecWithPrec(25, 4), // 0.25% (of 0.3%)
		ProtocolFee:               math.LegacyNewDecWithPrec(5, 4),  // 0.05% (of 0.3%)
		MinLiquidity:              math.NewInt(1000),                // Minimum initial liquidity
		MaxSlippagePercent:        math.LegacyNewDecWithPrec(5, 2),  // 5%
		MaxPoolDrainPercent:       math.LegacyNewDecWithPrec(30, 2), // 30% of reserves
		FlashLoanProtectionBlocks: 10,
		PoolCreationGas:           1000,
		SwapValidationGas:         1500,
		LiquidityGas:              1200,
	}
}

// TestAddr returns a test address for testing purposes
func TestAddr() sdk.AccAddress {
	return sdk.AccAddress([]byte("test_address_______"))
}
