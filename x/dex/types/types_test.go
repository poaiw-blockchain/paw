package types

import (
	"testing"

	"cosmossdk.io/math"
)

func TestModuleConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"ModuleName", ModuleName, "dex"},
		{"StoreKey", StoreKey, "dex"},
		{"MemStoreKey", MemStoreKey, "mem_dex"},
		{"RouterKey", RouterKey, "dex"},
		{"QuerierRoute", QuerierRoute, "dex"},
		{"PortID", PortID, "dex"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Expected %s to be %q, got %q", tt.name, tt.expected, tt.constant)
			}
		})
	}
}

func TestIBCEventTypes_Constants(t *testing.T) {
	// Verify IBC event types are defined and not empty
	ibcEventTypes := map[string]string{
		"EventTypeChannelOpen":        EventTypeChannelOpen,
		"EventTypeChannelOpenAck":     EventTypeChannelOpenAck,
		"EventTypeChannelOpenConfirm": EventTypeChannelOpenConfirm,
		"EventTypeChannelClose":       EventTypeChannelClose,
		"EventTypePacketReceive":      EventTypePacketReceive,
		"EventTypePacketAck":          EventTypePacketAck,
		"EventTypePacketTimeout":      EventTypePacketTimeout,
	}

	for name, value := range ibcEventTypes {
		if value == "" {
			t.Errorf("%s is empty", name)
		}
	}
}

func TestIBCAttributeKeys_Constants(t *testing.T) {
	// Verify IBC attribute keys are defined and not empty
	ibcAttributeKeys := map[string]string{
		"AttributeKeyChannelID":             AttributeKeyChannelID,
		"AttributeKeyPortID":                AttributeKeyPortID,
		"AttributeKeyCounterpartyPortID":    AttributeKeyCounterpartyPortID,
		"AttributeKeyCounterpartyChannelID": AttributeKeyCounterpartyChannelID,
		"AttributeKeyPacketType":            AttributeKeyPacketType,
		"AttributeKeySequence":              AttributeKeySequence,
		"AttributeKeyAckSuccess":            AttributeKeyAckSuccess,
		"AttributeKeyPendingOperations":     AttributeKeyPendingOperations,
	}

	for name, value := range ibcAttributeKeys {
		if value == "" {
			t.Errorf("%s is empty", name)
		}
	}
}

func TestDefaultParams(t *testing.T) {
	params := DefaultParams()

	// Verify swap fee is positive
	if params.SwapFee.IsNil() || params.SwapFee.IsNegative() {
		t.Errorf("Invalid swap fee: %v", params.SwapFee)
	}

	// Verify LP fee is positive
	if params.LpFee.IsNil() || params.LpFee.IsNegative() {
		t.Errorf("Invalid LP fee: %v", params.LpFee)
	}

	// Verify protocol fee is positive
	if params.ProtocolFee.IsNil() || params.ProtocolFee.IsNegative() {
		t.Errorf("Invalid protocol fee: %v", params.ProtocolFee)
	}

	// Verify fee relationship
	totalFee := params.LpFee.Add(params.ProtocolFee)
	if totalFee.GT(params.SwapFee) {
		t.Errorf("LP fee + protocol fee (%v) exceeds swap fee (%v)", totalFee, params.SwapFee)
	}

	// Verify min liquidity
	if params.MinLiquidity.IsNil() || params.MinLiquidity.IsNegative() {
		t.Errorf("Invalid min liquidity: %v", params.MinLiquidity)
	}

	// Verify max slippage percent
	if params.MaxSlippagePercent.IsNil() || params.MaxSlippagePercent.IsNegative() {
		t.Errorf("Invalid max slippage percent: %v", params.MaxSlippagePercent)
	}

	// Verify max pool drain percent
	if params.MaxPoolDrainPercent.IsNil() || params.MaxPoolDrainPercent.IsNegative() {
		t.Errorf("Invalid max pool drain percent: %v", params.MaxPoolDrainPercent)
	}

	// Verify flash loan protection blocks
	if params.FlashLoanProtectionBlocks == 0 {
		t.Error("Flash loan protection blocks should be positive")
	}

	// Verify gas values
	if params.PoolCreationGas == 0 {
		t.Error("Pool creation gas should be positive")
	}
	if params.SwapValidationGas == 0 {
		t.Error("Swap validation gas should be positive")
	}
	if params.LiquidityGas == 0 {
		t.Error("Liquidity gas should be positive")
	}

	// Verify upgrade preserve state
	if !params.UpgradePreserveCircuitBreakerState {
		t.Error("Expected UpgradePreserveCircuitBreakerState to be true by default")
	}

	// Verify recommended max slippage
	if params.RecommendedMaxSlippage.IsNil() || params.RecommendedMaxSlippage.IsNegative() {
		t.Errorf("Invalid recommended max slippage: %v", params.RecommendedMaxSlippage)
	}

	// Verify commit-reveal defaults
	if params.EnableCommitReveal {
		t.Error("Expected EnableCommitReveal to be false by default")
	}
	if params.CommitRevealDelay == 0 {
		t.Error("CommitRevealDelay should have a default value")
	}
	if params.CommitTimeoutBlocks == 0 {
		t.Error("CommitTimeoutBlocks should have a default value")
	}
}

func TestDefaultParams_FeeValues(t *testing.T) {
	params := DefaultParams()

	// Verify expected fee values
	expectedSwapFee := math.LegacyNewDecWithPrec(3, 3) // 0.3%
	if !params.SwapFee.Equal(expectedSwapFee) {
		t.Errorf("Expected swap fee %v, got %v", expectedSwapFee, params.SwapFee)
	}

	expectedLpFee := math.LegacyNewDecWithPrec(25, 4) // 0.25%
	if !params.LpFee.Equal(expectedLpFee) {
		t.Errorf("Expected LP fee %v, got %v", expectedLpFee, params.LpFee)
	}

	expectedProtocolFee := math.LegacyNewDecWithPrec(5, 4) // 0.05%
	if !params.ProtocolFee.Equal(expectedProtocolFee) {
		t.Errorf("Expected protocol fee %v, got %v", expectedProtocolFee, params.ProtocolFee)
	}
}

func TestDefaultParams_LiquidityValues(t *testing.T) {
	params := DefaultParams()

	expectedMinLiquidity := math.NewInt(1000)
	if !params.MinLiquidity.Equal(expectedMinLiquidity) {
		t.Errorf("Expected min liquidity %v, got %v", expectedMinLiquidity, params.MinLiquidity)
	}
}

func TestDefaultParams_SlippageValues(t *testing.T) {
	params := DefaultParams()

	expectedMaxSlippage := math.LegacyNewDecWithPrec(5, 2) // 5%
	if !params.MaxSlippagePercent.Equal(expectedMaxSlippage) {
		t.Errorf("Expected max slippage %v, got %v", expectedMaxSlippage, params.MaxSlippagePercent)
	}

	expectedRecommendedMaxSlippage := math.LegacyNewDecWithPrec(3, 2) // 3%
	if !params.RecommendedMaxSlippage.Equal(expectedRecommendedMaxSlippage) {
		t.Errorf("Expected recommended max slippage %v, got %v", expectedRecommendedMaxSlippage, params.RecommendedMaxSlippage)
	}
}

func TestDefaultParams_PoolDrainValues(t *testing.T) {
	params := DefaultParams()

	expectedMaxPoolDrain := math.LegacyNewDecWithPrec(30, 2) // 30%
	if !params.MaxPoolDrainPercent.Equal(expectedMaxPoolDrain) {
		t.Errorf("Expected max pool drain %v, got %v", expectedMaxPoolDrain, params.MaxPoolDrainPercent)
	}
}

func TestDefaultParams_GasValues(t *testing.T) {
	params := DefaultParams()

	tests := []struct {
		name     string
		value    uint64
		expected uint64
	}{
		{"PoolCreationGas", params.PoolCreationGas, 1000},
		{"SwapValidationGas", params.SwapValidationGas, 1500},
		{"LiquidityGas", params.LiquidityGas, 1200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected %s to be %d, got %d", tt.name, tt.expected, tt.value)
			}
		})
	}
}

func TestDefaultParams_FlashLoanProtection(t *testing.T) {
	params := DefaultParams()

	expectedFlashLoanBlocks := uint64(10)
	if params.FlashLoanProtectionBlocks != expectedFlashLoanBlocks {
		t.Errorf("Expected flash loan protection blocks %d, got %d", expectedFlashLoanBlocks, params.FlashLoanProtectionBlocks)
	}
}

func TestDefaultParams_CommitReveal(t *testing.T) {
	params := DefaultParams()

	// Verify commit-reveal is disabled by default
	if params.EnableCommitReveal {
		t.Error("Expected EnableCommitReveal to be false by default")
	}

	expectedCommitRevealDelay := uint64(10)
	if params.CommitRevealDelay != expectedCommitRevealDelay {
		t.Errorf("Expected commit reveal delay %d, got %d", expectedCommitRevealDelay, params.CommitRevealDelay)
	}

	expectedCommitTimeoutBlocks := uint64(100)
	if params.CommitTimeoutBlocks != expectedCommitTimeoutBlocks {
		t.Errorf("Expected commit timeout blocks %d, got %d", expectedCommitTimeoutBlocks, params.CommitTimeoutBlocks)
	}
}

func TestTestAddr(t *testing.T) {
	addr := TestAddr()

	if len(addr) == 0 {
		t.Error("TestAddr() returned empty address")
	}

	// Verify it's a valid length for an AccAddress (should be 19 or 20 bytes)
	if len(addr) < 19 || len(addr) > 20 {
		t.Errorf("Expected address length between 19-20, got %d", len(addr))
	}

	// Verify it's deterministic
	addr2 := TestAddr()
	if !addr.Equals(addr2) {
		t.Error("TestAddr() should return the same address on multiple calls")
	}
}

func TestDefaultParams_Consistency(t *testing.T) {
	// Test that calling DefaultParams multiple times returns consistent values
	params1 := DefaultParams()
	params2 := DefaultParams()

	if !params1.SwapFee.Equal(params2.SwapFee) {
		t.Error("DefaultParams() returns inconsistent SwapFee")
	}

	if !params1.LpFee.Equal(params2.LpFee) {
		t.Error("DefaultParams() returns inconsistent LpFee")
	}

	if !params1.ProtocolFee.Equal(params2.ProtocolFee) {
		t.Error("DefaultParams() returns inconsistent ProtocolFee")
	}

	if !params1.MinLiquidity.Equal(params2.MinLiquidity) {
		t.Error("DefaultParams() returns inconsistent MinLiquidity")
	}

	if params1.FlashLoanProtectionBlocks != params2.FlashLoanProtectionBlocks {
		t.Error("DefaultParams() returns inconsistent FlashLoanProtectionBlocks")
	}
}

func TestConstantsUniqueness(t *testing.T) {
	// Verify that key constants are unique
	constants := []string{
		ModuleName,
		StoreKey,
		MemStoreKey,
		RouterKey,
		QuerierRoute,
		PortID,
	}

	seen := make(map[string]bool)
	for _, constant := range constants {
		if constant == ModuleName || constant == StoreKey || constant == RouterKey || constant == QuerierRoute || constant == PortID {
			// These can be the same ("dex")
			continue
		}
		if seen[constant] && constant != "dex" {
			t.Errorf("Duplicate constant value: %s", constant)
		}
		seen[constant] = true
	}

	// MemStoreKey should be different from others
	if MemStoreKey == ModuleName {
		t.Error("MemStoreKey should be different from ModuleName")
	}
}

func TestDefaultParams_AuthorizedChannels(t *testing.T) {
	params := DefaultParams()

	// Default params may have nil or empty authorized channels slice
	if params.AuthorizedChannels != nil && len(params.AuthorizedChannels) != 0 {
		t.Errorf("Expected 0 authorized channels by default, got %d", len(params.AuthorizedChannels))
	}
}
