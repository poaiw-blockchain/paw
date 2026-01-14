package types

import (
	"fmt"

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

	// IBC event types (kept here for IBC-specific events)
	EventTypeChannelOpen        = "dex_channel_open"
	EventTypeChannelOpenAck     = "dex_channel_open_ack"
	EventTypeChannelOpenConfirm = "dex_channel_open_confirm"
	EventTypeChannelClose       = "dex_channel_close"
	EventTypePacketReceive      = "dex_packet_receive"
	EventTypePacketAck          = "dex_packet_ack"
	EventTypePacketTimeout      = "dex_packet_timeout"

	// IBC event attribute keys
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
		SwapFee:                            math.LegacyNewDecWithPrec(3, 3),  // 0.3%
		LpFee:                              math.LegacyNewDecWithPrec(25, 4), // 0.25% (of 0.3%)
		ProtocolFee:                        math.LegacyNewDecWithPrec(5, 4),  // 0.05% (of 0.3%)
		MinLiquidity:                       math.NewInt(1000),                // Minimum initial liquidity
		MaxSlippagePercent:                 math.LegacyNewDecWithPrec(5, 2),  // 5%
		MaxPoolDrainPercent:                math.LegacyNewDecWithPrec(30, 2), // 30% of reserves
		FlashLoanProtectionBlocks:          100,                              // SEC-18: ~10 min at 6s blocks
		PoolCreationGas:                    1000,
		SwapValidationGas:                  1500,
		LiquidityGas:                       1200,
		UpgradePreserveCircuitBreakerState: true,                            // Preserve pause state across upgrades
		RecommendedMaxSlippage:             math.LegacyNewDecWithPrec(3, 2), // 3% recommended max
		EnableCommitReveal:                 false,                           // Disabled for testnet
		CommitRevealDelay:                  10,                              // 10 blocks
		CommitTimeoutBlocks:                100,                             // 100 blocks
		CircuitBreakerDurationSeconds:      3600,                            // 1 hour
	}
}

// TestAddr returns a test address for testing purposes
func TestAddr() sdk.AccAddress {
	return sdk.AccAddress([]byte("test_address_______"))
}

// TestAddrWithSeed returns a deterministic test address based on a seed value.
// Used to generate multiple unique addresses for testing purposes.
func TestAddrWithSeed(seed int) sdk.AccAddress {
	addr := make([]byte, 20)
	copy(addr, fmt.Sprintf("test_addr_%08d___", seed))
	return sdk.AccAddress(addr)
}
