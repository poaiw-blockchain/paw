package types

import (
	"strings"
	"testing"
)

func TestEventTypes(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		prefix    string
	}{
		// Swap events
		{"EventTypeDexSwap", EventTypeDexSwap, "dex_"},
		{"EventTypeDexSwapFailed", EventTypeDexSwapFailed, "dex_"},
		{"EventTypeDexLimitOrder", EventTypeDexLimitOrder, "dex_"},
		{"EventTypeDexOrderFilled", EventTypeDexOrderFilled, "dex_"},
		{"EventTypeDexOrderCancelled", EventTypeDexOrderCancelled, ""},
		{"EventTypeDexOrderPlaced", EventTypeDexOrderPlaced, ""},
		{"EventTypeDexOrderMatched", EventTypeDexOrderMatched, ""},
		{"EventTypeDexOrderExpired", EventTypeDexOrderExpired, ""},

		// Liquidity events
		{"EventTypeDexAddLiquidity", EventTypeDexAddLiquidity, "dex_"},
		{"EventTypeDexRemoveLiquidity", EventTypeDexRemoveLiquidity, "dex_"},
		{"EventTypeDexLiquidityLocked", EventTypeDexLiquidityLocked, "dex_"},

		// Pool events
		{"EventTypeDexPoolCreated", EventTypeDexPoolCreated, "dex_"},
		{"EventTypeDexPoolUpdated", EventTypeDexPoolUpdated, "dex_"},

		// Security events
		{"EventTypeDexLargeSwap", EventTypeDexLargeSwap, "dex_"},
		{"EventTypeDexLargeLiquidityAddition", EventTypeDexLargeLiquidityAddition, "dex_"},
		{"EventTypeDexSlippageExceeded", EventTypeDexSlippageExceeded, "dex_"},
		{"EventTypeDexPotentialSandwichAttack", EventTypeDexPotentialSandwichAttack, ""},

		// Cross-chain events
		{"EventTypeDexCrossChainSwap", EventTypeDexCrossChainSwap, "dex_"},
		{"EventTypeDexCrossChainSwapTimeout", EventTypeDexCrossChainSwapTimeout, "dex_"},
		{"EventTypeDexCrossChainSwapFailed", EventTypeDexCrossChainSwapFailed, "dex_"},

		// Fee events
		{"EventTypeDexFeeCollected", EventTypeDexFeeCollected, "dex_"},
		{"EventTypeDexFeeUpdated", EventTypeDexFeeUpdated, "dex_"},

		// Circuit breaker events
		{"EventTypeCircuitBreakerOpen", EventTypeCircuitBreakerOpen, "dex_"},
		{"EventTypeCircuitBreakerClose", EventTypeCircuitBreakerClose, "dex_"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify event type is not empty
			if tt.eventType == "" {
				t.Error("Event type is empty")
			}

			// Verify lowercase with underscore format
			if tt.eventType != strings.ToLower(tt.eventType) {
				t.Errorf("Event type %q is not lowercase", tt.eventType)
			}

			// Verify no spaces
			if strings.Contains(tt.eventType, " ") {
				t.Errorf("Event type %q contains spaces", tt.eventType)
			}

			// Verify prefix if specified
			if tt.prefix != "" && !strings.HasPrefix(tt.eventType, tt.prefix) {
				t.Errorf("Event type %q does not have prefix %q", tt.eventType, tt.prefix)
			}
		})
	}
}

func TestAttributeKeys(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		// Common attributes
		{"AttributeKeySender", AttributeKeySender},
		{"AttributeKeyTrader", AttributeKeyTrader},
		{"AttributeKeyProvider", AttributeKeyProvider},
		{"AttributeKeyRecipient", AttributeKeyRecipient},
		{"AttributeKeyPoolID", AttributeKeyPoolID},

		// Token attributes
		{"AttributeKeyTokenIn", AttributeKeyTokenIn},
		{"AttributeKeyTokenOut", AttributeKeyTokenOut},
		{"AttributeKeyTokenA", AttributeKeyTokenA},
		{"AttributeKeyTokenB", AttributeKeyTokenB},
		{"AttributeKeyDenom", AttributeKeyDenom},

		// Amount attributes
		{"AttributeKeyAmount", AttributeKeyAmount},
		{"AttributeKeyAmountIn", AttributeKeyAmountIn},
		{"AttributeKeyAmountOut", AttributeKeyAmountOut},
		{"AttributeKeyMinAmountOut", AttributeKeyMinAmountOut},
		{"AttributeKeyAmountA", AttributeKeyAmountA},
		{"AttributeKeyAmountB", AttributeKeyAmountB},
		{"AttributeKeyRefundedAmount", AttributeKeyRefundedAmount},

		// Liquidity attributes
		{"AttributeKeyShares", AttributeKeyShares},
		{"AttributeKeyLockedShares", AttributeKeyLockedShares},
		{"AttributeKeyTotalShares", AttributeKeyTotalShares},

		// Reserve attributes
		{"AttributeKeyReserveA", AttributeKeyReserveA},
		{"AttributeKeyReserveB", AttributeKeyReserveB},

		// Fee attributes
		{"AttributeKeyFee", AttributeKeyFee},
		{"AttributeKeyFeeAmount", AttributeKeyFeeAmount},
		{"AttributeKeyLpFee", AttributeKeyLpFee},
		{"AttributeKeyProtocolFee", AttributeKeyProtocolFee},

		// Price attributes
		{"AttributeKeyPrice", AttributeKeyPrice},
		{"AttributeKeyPriceImpact", AttributeKeyPriceImpact},
		{"AttributeKeySlippage", AttributeKeySlippage},

		// Order attributes
		{"AttributeKeyOrderID", AttributeKeyOrderID},
		{"AttributeKeyOrderType", AttributeKeyOrderType},
		{"AttributeKeyLimitPrice", AttributeKeyLimitPrice},
		{"AttributeKeyOrderStatus", AttributeKeyOrderStatus},

		// Security attributes
		{"AttributeKeyPercentage", AttributeKeyPercentage},
		{"AttributeKeySwapPercentage", AttributeKeySwapPercentage},
		{"AttributeKeyBlocksApart", AttributeKeyBlocksApart},
		{"AttributeKeyMaxAllowed", AttributeKeyMaxAllowed},
		{"AttributeKeyBlockHeight", AttributeKeyBlockHeight},
		{"AttributeKeyTimestamp", AttributeKeyTimestamp},

		// Cross-chain attributes
		{"AttributeKeySwapID", AttributeKeySwapID},
		{"AttributeKeyJobID", AttributeKeyJobID},
		{"AttributeKeyTargetChain", AttributeKeyTargetChain},
		{"AttributeKeyError", AttributeKeyError},
		{"AttributeKeyUserAddress", AttributeKeyUserAddress},

		// Status attributes
		{"AttributeKeyStatus", AttributeKeyStatus},
		{"AttributeKeyReason", AttributeKeyReason},
		{"AttributeKeyActor", AttributeKeyActor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify attribute key is not empty
			if tt.key == "" {
				t.Error("Attribute key is empty")
			}

			// Verify lowercase with underscore format
			if tt.key != strings.ToLower(tt.key) {
				t.Errorf("Attribute key %q is not lowercase", tt.key)
			}

			// Verify no spaces
			if strings.Contains(tt.key, " ") {
				t.Errorf("Attribute key %q contains spaces", tt.key)
			}

			// Verify uses underscores for multi-word keys
			if strings.Contains(tt.name, "Key") && len(tt.key) > 0 {
				words := strings.Split(tt.name, "Key")
				if len(words) > 1 && len(words[1]) > 0 {
					// Multi-word attribute should use underscores
					if !strings.Contains(tt.key, "_") && len(tt.key) > 10 {
						t.Logf("Note: Multi-word attribute %q might benefit from underscores", tt.key)
					}
				}
			}
		})
	}
}

func TestAttributeKeyUniqueness(t *testing.T) {
	// Collect all attribute keys
	keys := []string{
		// Common
		AttributeKeySender,
		AttributeKeyTrader,
		AttributeKeyProvider,
		AttributeKeyRecipient,
		AttributeKeyPoolID,

		// Token
		AttributeKeyTokenIn,
		AttributeKeyTokenOut,
		AttributeKeyTokenA,
		AttributeKeyTokenB,
		AttributeKeyDenom,

		// Amount
		AttributeKeyAmount,
		AttributeKeyAmountIn,
		AttributeKeyAmountOut,
		AttributeKeyMinAmountOut,
		AttributeKeyAmountA,
		AttributeKeyAmountB,
		AttributeKeyRefundedAmount,

		// Liquidity
		AttributeKeyShares,
		AttributeKeyLockedShares,
		AttributeKeyTotalShares,

		// Reserve
		AttributeKeyReserveA,
		AttributeKeyReserveB,

		// Fee
		AttributeKeyFee,
		AttributeKeyFeeAmount,
		AttributeKeyLpFee,
		AttributeKeyProtocolFee,

		// Price
		AttributeKeyPrice,
		AttributeKeyPriceImpact,
		AttributeKeySlippage,

		// Order
		AttributeKeyOrderID,
		AttributeKeyOrderType,
		AttributeKeyLimitPrice,
		AttributeKeyOrderStatus,

		// Security
		AttributeKeyPercentage,
		AttributeKeySwapPercentage,
		AttributeKeyBlocksApart,
		AttributeKeyMaxAllowed,
		AttributeKeyBlockHeight,
		AttributeKeyTimestamp,

		// Cross-chain
		AttributeKeySwapID,
		AttributeKeyJobID,
		AttributeKeyTargetChain,
		AttributeKeyError,
		AttributeKeyUserAddress,

		// Status
		AttributeKeyStatus,
		AttributeKeyReason,
		AttributeKeyActor,
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, key := range keys {
		if seen[key] {
			t.Errorf("Duplicate attribute key found: %q", key)
		}
		seen[key] = true
	}
}

func TestEventTypeUniqueness(t *testing.T) {
	// Collect all event types
	eventTypes := []string{
		// Swap
		EventTypeDexSwap,
		EventTypeDexSwapFailed,
		EventTypeDexLimitOrder,
		EventTypeDexOrderFilled,
		EventTypeDexOrderCancelled,
		EventTypeDexOrderPlaced,
		EventTypeDexOrderMatched,
		EventTypeDexOrderExpired,

		// Liquidity
		EventTypeDexAddLiquidity,
		EventTypeDexRemoveLiquidity,
		EventTypeDexLiquidityLocked,

		// Pool
		EventTypeDexPoolCreated,
		EventTypeDexPoolUpdated,

		// Security
		EventTypeDexLargeSwap,
		EventTypeDexLargeLiquidityAddition,
		EventTypeDexSlippageExceeded,
		EventTypeDexPotentialSandwichAttack,

		// Cross-chain
		EventTypeDexCrossChainSwap,
		EventTypeDexCrossChainSwapTimeout,
		EventTypeDexCrossChainSwapFailed,

		// Fee
		EventTypeDexFeeCollected,
		EventTypeDexFeeUpdated,

		// Circuit breaker
		EventTypeCircuitBreakerOpen,
		EventTypeCircuitBreakerClose,
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, eventType := range eventTypes {
		if seen[eventType] {
			t.Errorf("Duplicate event type found: %q", eventType)
		}
		seen[eventType] = true
	}
}

func TestIBCEventTypes(t *testing.T) {
	// IBC-specific event types defined in types.go
	ibcEventTypes := []string{
		EventTypeChannelOpen,
		EventTypeChannelOpenAck,
		EventTypeChannelOpenConfirm,
		EventTypeChannelClose,
		EventTypePacketReceive,
		EventTypePacketAck,
		EventTypePacketTimeout,
	}

	for _, eventType := range ibcEventTypes {
		t.Run(eventType, func(t *testing.T) {
			// Verify not empty
			if eventType == "" {
				t.Error("IBC event type is empty")
			}

			// Verify starts with dex_
			if !strings.HasPrefix(eventType, "dex_") {
				t.Errorf("IBC event type %q does not start with 'dex_'", eventType)
			}

			// Verify lowercase with underscores
			if eventType != strings.ToLower(eventType) {
				t.Errorf("IBC event type %q is not lowercase", eventType)
			}
		})
	}
}

func TestIBCAttributeKeys(t *testing.T) {
	// IBC-specific attribute keys defined in types.go
	ibcAttributeKeys := []string{
		AttributeKeyChannelID,
		AttributeKeyPortID,
		AttributeKeyCounterpartyPortID,
		AttributeKeyCounterpartyChannelID,
		AttributeKeyPacketType,
		AttributeKeySequence,
		AttributeKeyAckSuccess,
		AttributeKeyPendingOperations,
	}

	for _, key := range ibcAttributeKeys {
		t.Run(key, func(t *testing.T) {
			// Verify not empty
			if key == "" {
				t.Error("IBC attribute key is empty")
			}

			// Verify lowercase with underscores
			if key != strings.ToLower(key) {
				t.Errorf("IBC attribute key %q is not lowercase", key)
			}

			// Verify no spaces
			if strings.Contains(key, " ") {
				t.Errorf("IBC attribute key %q contains spaces", key)
			}
		})
	}
}

func TestEventNamingConsistency(t *testing.T) {
	// Verify event type naming follows consistent patterns
	swapEvents := []string{
		EventTypeDexSwap,
		EventTypeDexSwapFailed,
	}

	for _, event := range swapEvents {
		if !strings.Contains(event, "swap") {
			t.Errorf("Swap event %q does not contain 'swap'", event)
		}
	}

	liquidityEvents := []string{
		EventTypeDexAddLiquidity,
		EventTypeDexRemoveLiquidity,
		EventTypeDexLiquidityLocked,
	}

	for _, event := range liquidityEvents {
		if !strings.Contains(event, "liquidity") {
			t.Errorf("Liquidity event %q does not contain 'liquidity'", event)
		}
	}

	poolEvents := []string{
		EventTypeDexPoolCreated,
		EventTypeDexPoolUpdated,
	}

	for _, event := range poolEvents {
		if !strings.Contains(event, "pool") {
			t.Errorf("Pool event %q does not contain 'pool'", event)
		}
	}
}

func TestAttributeKeyNamingConsistency(t *testing.T) {
	// Amount attributes should contain "amount"
	amountKeys := []string{
		AttributeKeyAmount,
		AttributeKeyAmountIn,
		AttributeKeyAmountOut,
		AttributeKeyMinAmountOut,
		AttributeKeyAmountA,
		AttributeKeyAmountB,
		AttributeKeyRefundedAmount,
		AttributeKeyFeeAmount,
	}

	for _, key := range amountKeys {
		if !strings.Contains(key, "amount") {
			t.Errorf("Amount attribute key %q does not contain 'amount'", key)
		}
	}

	// Fee attributes should contain "fee"
	feeKeys := []string{
		AttributeKeyFee,
		AttributeKeyFeeAmount,
		AttributeKeyLpFee,
		AttributeKeyProtocolFee,
	}

	for _, key := range feeKeys {
		if !strings.Contains(key, "fee") {
			t.Errorf("Fee attribute key %q does not contain 'fee'", key)
		}
	}
}
