package types

import (
	"testing"

	"cosmossdk.io/math"
)

// ============================================================================
// Pool Type Tests
// ============================================================================

func TestPool_Fields(t *testing.T) {
	pool := Pool{
		Id:          1,
		TokenA:      "upaw",
		TokenB:      "uatom",
		ReserveA:    math.NewInt(1000000),
		ReserveB:    math.NewInt(2000000),
		TotalShares: math.NewInt(1414213),
		Creator:     "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
	}

	// Verify all fields
	if pool.Id != 1 {
		t.Errorf("Expected Id 1, got %d", pool.Id)
	}
	if pool.TokenA != "upaw" {
		t.Errorf("Expected TokenA 'upaw', got %s", pool.TokenA)
	}
	if pool.TokenB != "uatom" {
		t.Errorf("Expected TokenB 'uatom', got %s", pool.TokenB)
	}
	if !pool.ReserveA.Equal(math.NewInt(1000000)) {
		t.Errorf("ReserveA mismatch")
	}
	if !pool.ReserveB.Equal(math.NewInt(2000000)) {
		t.Errorf("ReserveB mismatch")
	}
	if !pool.TotalShares.Equal(math.NewInt(1414213)) {
		t.Errorf("TotalShares mismatch")
	}
	if pool.Creator != "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q" {
		t.Errorf("Creator mismatch")
	}
}

func TestPool_GetMethods(t *testing.T) {
	pool := Pool{
		Id:          42,
		TokenA:      "upaw",
		TokenB:      "uatom",
		ReserveA:    math.NewInt(1000000),
		ReserveB:    math.NewInt(2000000),
		TotalShares: math.NewInt(1414213),
		Creator:     "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
	}

	if pool.GetId() != 42 {
		t.Errorf("GetId() expected 42, got %d", pool.GetId())
	}
	if pool.GetTokenA() != "upaw" {
		t.Errorf("GetTokenA() expected 'upaw', got %s", pool.GetTokenA())
	}
	if pool.GetTokenB() != "uatom" {
		t.Errorf("GetTokenB() expected 'uatom', got %s", pool.GetTokenB())
	}
	if pool.GetCreator() != "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q" {
		t.Errorf("GetCreator() mismatch")
	}
}

func TestPool_Reset(t *testing.T) {
	pool := Pool{
		Id:          1,
		TokenA:      "upaw",
		TokenB:      "uatom",
		ReserveA:    math.NewInt(1000000),
		ReserveB:    math.NewInt(2000000),
		TotalShares: math.NewInt(1414213),
		Creator:     "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
	}

	pool.Reset()

	if pool.Id != 0 {
		t.Errorf("After Reset, Id should be 0, got %d", pool.Id)
	}
	if pool.TokenA != "" {
		t.Errorf("After Reset, TokenA should be empty, got %s", pool.TokenA)
	}
	if pool.TokenB != "" {
		t.Errorf("After Reset, TokenB should be empty, got %s", pool.TokenB)
	}
	if pool.Creator != "" {
		t.Errorf("After Reset, Creator should be empty, got %s", pool.Creator)
	}
}

func TestPool_String(t *testing.T) {
	pool := Pool{
		Id:          1,
		TokenA:      "upaw",
		TokenB:      "uatom",
		ReserveA:    math.NewInt(1000000),
		ReserveB:    math.NewInt(2000000),
		TotalShares: math.NewInt(1414213),
		Creator:     "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
	}

	str := pool.String()
	if str == "" {
		t.Error("Pool.String() should not be empty")
	}
}

func TestPool_ProtoMessage(t *testing.T) {
	pool := Pool{}
	// This just verifies the method exists - it's a marker method
	pool.ProtoMessage()
}

func TestPool_ZeroValues(t *testing.T) {
	pool := Pool{
		Id:          0,
		TokenA:      "",
		TokenB:      "",
		ReserveA:    math.ZeroInt(),
		ReserveB:    math.ZeroInt(),
		TotalShares: math.ZeroInt(),
		Creator:     "",
	}

	if pool.GetId() != 0 {
		t.Error("Zero pool should have Id 0")
	}
	if pool.GetTokenA() != "" {
		t.Error("Zero pool should have empty TokenA")
	}
}

func TestPool_LargeValues(t *testing.T) {
	largeInt := math.NewIntFromUint64(^uint64(0))

	pool := Pool{
		Id:          ^uint64(0),
		TokenA:      "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2",
		TokenB:      "ibc/123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
		ReserveA:    largeInt,
		ReserveB:    largeInt,
		TotalShares: largeInt,
		Creator:     "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
	}

	if pool.GetId() != ^uint64(0) {
		t.Error("Large pool ID not stored correctly")
	}
	if !pool.ReserveA.Equal(largeInt) {
		t.Error("Large ReserveA not stored correctly")
	}
}

// ============================================================================
// PoolTWAP Tests
// ============================================================================

func TestPoolTWAP_Fields(t *testing.T) {
	twap := PoolTWAP{
		PoolId:        1,
		TotalSeconds:  86400,
		LastTimestamp: 1700000000,
	}

	if twap.GetPoolId() != 1 {
		t.Errorf("Expected PoolId 1, got %d", twap.GetPoolId())
	}
	if twap.GetTotalSeconds() != 86400 {
		t.Errorf("Expected TotalSeconds 86400, got %d", twap.GetTotalSeconds())
	}
	if twap.GetLastTimestamp() != 1700000000 {
		t.Errorf("Expected LastTimestamp 1700000000, got %d", twap.GetLastTimestamp())
	}
}

func TestPoolTWAP_Reset(t *testing.T) {
	twap := PoolTWAP{
		PoolId:        1,
		TotalSeconds:  86400,
		LastTimestamp: 1700000000,
	}

	twap.Reset()

	if twap.PoolId != 0 {
		t.Error("After Reset, PoolId should be 0")
	}
	if twap.TotalSeconds != 0 {
		t.Error("After Reset, TotalSeconds should be 0")
	}
}

func TestPoolTWAP_String(t *testing.T) {
	twap := PoolTWAP{
		PoolId:        1,
		TotalSeconds:  86400,
		LastTimestamp: 1700000000,
	}

	str := twap.String()
	if str == "" {
		t.Error("PoolTWAP.String() should not be empty")
	}
}

// ============================================================================
// CircuitBreakerState Tests
// ============================================================================

func TestCircuitBreakerState_Fields(t *testing.T) {
	cb := CircuitBreakerState{
		Enabled:           true,
		PausedUntil:       1700100000,
		TriggeredBy:       "system",
		TriggerReason:     "price anomaly",
		NotificationsSent: 5,
		LastNotification:  1700050000,
		PersistenceKey:    "cb_pool_1",
	}

	if !cb.GetEnabled() {
		t.Error("Enabled should be true")
	}
	if cb.GetPausedUntil() != 1700100000 {
		t.Error("PausedUntil mismatch")
	}
	if cb.GetTriggeredBy() != "system" {
		t.Error("TriggeredBy mismatch")
	}
	if cb.GetTriggerReason() != "price anomaly" {
		t.Error("TriggerReason mismatch")
	}
	if cb.GetNotificationsSent() != 5 {
		t.Error("NotificationsSent mismatch")
	}
	if cb.GetLastNotification() != 1700050000 {
		t.Error("LastNotification mismatch")
	}
	if cb.GetPersistenceKey() != "cb_pool_1" {
		t.Error("PersistenceKey mismatch")
	}
}

func TestCircuitBreakerState_Reset(t *testing.T) {
	cb := CircuitBreakerState{
		Enabled:           true,
		PausedUntil:       1700100000,
		TriggeredBy:       "system",
		TriggerReason:     "price anomaly",
		NotificationsSent: 5,
		LastNotification:  1700050000,
		PersistenceKey:    "cb_pool_1",
	}

	cb.Reset()

	if cb.Enabled {
		t.Error("After Reset, Enabled should be false")
	}
	if cb.PersistenceKey != "" {
		t.Error("After Reset, PersistenceKey should be empty")
	}
}

func TestCircuitBreakerState_String(t *testing.T) {
	cb := CircuitBreakerState{
		Enabled: true,
	}

	str := cb.String()
	if str == "" {
		t.Error("CircuitBreakerState.String() should not be empty")
	}
}

// ============================================================================
// LimitOrder Tests
// ============================================================================

func TestLimitOrder_Fields(t *testing.T) {
	order := LimitOrder{
		Id:              1,
		Owner:           "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
		PoolId:          1,
		OrderType:       OrderType_ORDER_TYPE_BUY,
		TokenIn:         "upaw",
		TokenOut:        "uatom",
		AmountIn:        math.NewInt(1000000),
		MinAmountOut:    math.NewInt(900000),
		LimitPrice:      math.LegacyMustNewDecFromStr("1.1"),
		FilledAmount:    math.NewInt(500000),
		Status:          OrderStatus_ORDER_STATUS_PARTIALLY_FILLED,
		CreatedAt:       1700000000,
		ExpiresAt:       1700100000,
		CreatedAtHeight: 1000,
	}

	if order.GetId() != 1 {
		t.Error("Id mismatch")
	}
	if order.GetOwner() != "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q" {
		t.Error("Owner mismatch")
	}
	if order.GetPoolId() != 1 {
		t.Error("PoolId mismatch")
	}
	if order.GetOrderType() != OrderType_ORDER_TYPE_BUY {
		t.Error("OrderType mismatch")
	}
	if order.GetTokenIn() != "upaw" {
		t.Error("TokenIn mismatch")
	}
	if order.GetTokenOut() != "uatom" {
		t.Error("TokenOut mismatch")
	}
	if order.GetStatus() != OrderStatus_ORDER_STATUS_PARTIALLY_FILLED {
		t.Error("Status mismatch")
	}
	if order.GetCreatedAt() != 1700000000 {
		t.Error("CreatedAt mismatch")
	}
	if order.GetExpiresAt() != 1700100000 {
		t.Error("ExpiresAt mismatch")
	}
	if order.GetCreatedAtHeight() != 1000 {
		t.Error("CreatedAtHeight mismatch")
	}
}

func TestLimitOrder_Reset(t *testing.T) {
	order := LimitOrder{
		Id:        1,
		Owner:     "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
		OrderType: OrderType_ORDER_TYPE_BUY,
	}

	order.Reset()

	if order.Id != 0 {
		t.Error("After Reset, Id should be 0")
	}
	if order.Owner != "" {
		t.Error("After Reset, Owner should be empty")
	}
}

func TestLimitOrder_String(t *testing.T) {
	order := LimitOrder{
		Id:        1,
		Owner:     "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
		OrderType: OrderType_ORDER_TYPE_BUY,
	}

	str := order.String()
	if str == "" {
		t.Error("LimitOrder.String() should not be empty")
	}
}

// ============================================================================
// SwapCommit Tests
// ============================================================================

func TestSwapCommit_Fields(t *testing.T) {
	commit := SwapCommit{
		Trader:       "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
		SwapHash:     "abc123def456",
		CommitHeight: 100,
		ExpiryHeight: 200,
	}

	if commit.Trader != "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q" {
		t.Error("Trader mismatch")
	}
	if commit.SwapHash != "abc123def456" {
		t.Error("SwapHash mismatch")
	}
	if commit.CommitHeight != 100 {
		t.Error("CommitHeight mismatch")
	}
	if commit.ExpiryHeight != 200 {
		t.Error("ExpiryHeight mismatch")
	}
}

func TestSwapCommit_Reset(t *testing.T) {
	commit := SwapCommit{
		Trader:       "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
		SwapHash:     "abc123def456",
		CommitHeight: 100,
		ExpiryHeight: 200,
	}

	commit.Reset()

	if commit.Trader != "" {
		t.Error("After Reset, Trader should be empty")
	}
	if commit.CommitHeight != 0 {
		t.Error("After Reset, CommitHeight should be 0")
	}
}

// ============================================================================
// Params Tests
// ============================================================================

func TestParams_GetMethods(t *testing.T) {
	params := DefaultParams()

	// Test Get methods for various fields - use the actual default values
	if params.GetFlashLoanProtectionBlocks() != 100 {
		t.Errorf("FlashLoanProtectionBlocks mismatch: expected %d, got %d",
			100, params.GetFlashLoanProtectionBlocks())
	}

	if params.GetPoolCreationGas() != 1000 {
		t.Errorf("PoolCreationGas mismatch: expected 1000, got %d", params.GetPoolCreationGas())
	}

	if params.GetSwapValidationGas() != 1500 {
		t.Errorf("SwapValidationGas mismatch: expected 1500, got %d", params.GetSwapValidationGas())
	}

	if params.GetLiquidityGas() != 1200 {
		t.Errorf("LiquidityGas mismatch: expected 1200, got %d", params.GetLiquidityGas())
	}

	if params.GetEnableCommitReveal() != false {
		t.Errorf("EnableCommitReveal mismatch: expected false, got %v", params.GetEnableCommitReveal())
	}

	if params.GetCommitRevealDelay() != 10 {
		t.Errorf("CommitRevealDelay mismatch: expected 10, got %d", params.GetCommitRevealDelay())
	}

	if params.GetCommitTimeoutBlocks() != 100 {
		t.Errorf("CommitTimeoutBlocks mismatch: expected 100, got %d", params.GetCommitTimeoutBlocks())
	}

	if params.GetCircuitBreakerDurationSeconds() != 3600 {
		t.Errorf("CircuitBreakerDurationSeconds mismatch: expected 3600, got %d", params.GetCircuitBreakerDurationSeconds())
	}
}

func TestParams_Reset(t *testing.T) {
	params := DefaultParams()
	params.Reset()

	if params.FlashLoanProtectionBlocks != 0 {
		t.Error("After Reset, FlashLoanProtectionBlocks should be 0")
	}
	if params.PoolCreationGas != 0 {
		t.Error("After Reset, PoolCreationGas should be 0")
	}
}

func TestParams_String(t *testing.T) {
	params := DefaultParams()
	str := params.String()

	if str == "" {
		t.Error("Params.String() should not be empty")
	}
}

// ============================================================================
// AuthorizedChannel Tests
// ============================================================================

func TestAuthorizedChannel_Fields(t *testing.T) {
	channel := AuthorizedChannel{
		PortId:    "transfer",
		ChannelId: "channel-0",
	}

	if channel.GetPortId() != "transfer" {
		t.Error("PortId mismatch")
	}
	if channel.GetChannelId() != "channel-0" {
		t.Error("ChannelId mismatch")
	}
}

func TestAuthorizedChannel_Reset(t *testing.T) {
	channel := AuthorizedChannel{
		PortId:    "transfer",
		ChannelId: "channel-0",
	}

	channel.Reset()

	if channel.PortId != "" {
		t.Error("After Reset, PortId should be empty")
	}
	if channel.ChannelId != "" {
		t.Error("After Reset, ChannelId should be empty")
	}
}

func TestAuthorizedChannel_String(t *testing.T) {
	channel := AuthorizedChannel{
		PortId:    "transfer",
		ChannelId: "channel-0",
	}

	str := channel.String()
	if str == "" {
		t.Error("AuthorizedChannel.String() should not be empty")
	}
}

// ============================================================================
// OrderType Enum Tests
// ============================================================================

func TestOrderType_Values(t *testing.T) {
	tests := []struct {
		orderType OrderType
		expected  int32
	}{
		{OrderType_ORDER_TYPE_UNSPECIFIED, 0},
		{OrderType_ORDER_TYPE_BUY, 1},
		{OrderType_ORDER_TYPE_SELL, 2},
	}

	for _, tt := range tests {
		if int32(tt.orderType) != tt.expected {
			t.Errorf("OrderType %v expected value %d, got %d", tt.orderType, tt.expected, int32(tt.orderType))
		}
	}
}

func TestOrderType_String(t *testing.T) {
	tests := []struct {
		orderType OrderType
		contains  string
	}{
		{OrderType_ORDER_TYPE_UNSPECIFIED, "UNSPECIFIED"},
		{OrderType_ORDER_TYPE_BUY, "BUY"},
		{OrderType_ORDER_TYPE_SELL, "SELL"},
	}

	for _, tt := range tests {
		str := tt.orderType.String()
		if str == "" {
			t.Errorf("OrderType %v has empty string representation", tt.orderType)
		}
	}
}

// ============================================================================
// OrderStatus Enum Tests
// ============================================================================

func TestOrderStatus_Values(t *testing.T) {
	tests := []struct {
		status   OrderStatus
		expected int32
	}{
		{OrderStatus_ORDER_STATUS_UNSPECIFIED, 0},
		{OrderStatus_ORDER_STATUS_OPEN, 1},
		{OrderStatus_ORDER_STATUS_PARTIALLY_FILLED, 2},
		{OrderStatus_ORDER_STATUS_FILLED, 3},
		{OrderStatus_ORDER_STATUS_CANCELLED, 4},
		{OrderStatus_ORDER_STATUS_EXPIRED, 5},
	}

	for _, tt := range tests {
		if int32(tt.status) != tt.expected {
			t.Errorf("OrderStatus %v expected value %d, got %d", tt.status, tt.expected, int32(tt.status))
		}
	}
}

func TestOrderStatus_String(t *testing.T) {
	tests := []struct {
		status   OrderStatus
		contains string
	}{
		{OrderStatus_ORDER_STATUS_UNSPECIFIED, "UNSPECIFIED"},
		{OrderStatus_ORDER_STATUS_OPEN, "OPEN"},
		{OrderStatus_ORDER_STATUS_PARTIALLY_FILLED, "PARTIALLY_FILLED"},
		{OrderStatus_ORDER_STATUS_FILLED, "FILLED"},
		{OrderStatus_ORDER_STATUS_CANCELLED, "CANCELLED"},
		{OrderStatus_ORDER_STATUS_EXPIRED, "EXPIRED"},
	}

	for _, tt := range tests {
		str := tt.status.String()
		if str == "" {
			t.Errorf("OrderStatus %v has empty string representation", tt.status)
		}
	}
}

// ============================================================================
// Nil Pointer Safety Tests
// ============================================================================

func TestPool_NilPointer(t *testing.T) {
	var pool *Pool = nil

	// These should not panic - should return zero values
	if pool.GetId() != 0 {
		t.Error("Nil pool GetId should return 0")
	}
	if pool.GetTokenA() != "" {
		t.Error("Nil pool GetTokenA should return empty string")
	}
	if pool.GetTokenB() != "" {
		t.Error("Nil pool GetTokenB should return empty string")
	}
	if pool.GetCreator() != "" {
		t.Error("Nil pool GetCreator should return empty string")
	}
}

func TestParams_NilPointer(t *testing.T) {
	var params *Params = nil

	// These should not panic - should return zero values
	if params.GetFlashLoanProtectionBlocks() != 0 {
		t.Error("Nil params GetFlashLoanProtectionBlocks should return 0")
	}
	if params.GetPoolCreationGas() != 0 {
		t.Error("Nil params GetPoolCreationGas should return 0")
	}
}

func TestCircuitBreakerState_NilPointer(t *testing.T) {
	var cb *CircuitBreakerState = nil

	// These should not panic - should return zero values
	if cb.GetEnabled() != false {
		t.Error("Nil circuit breaker GetEnabled should return false")
	}
	if cb.GetPausedUntil() != 0 {
		t.Error("Nil circuit breaker GetPausedUntil should return 0")
	}
}
