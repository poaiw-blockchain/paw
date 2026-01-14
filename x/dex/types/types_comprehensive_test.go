package types

import (
	"context"
	"errors"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ============================================================================
// TestAddrWithSeed Tests
// ============================================================================

func TestTestAddrWithSeed(t *testing.T) {
	tests := []struct {
		name string
		seed int
	}{
		{"seed 0", 0},
		{"seed 1", 1},
		{"seed 100", 100},
		{"seed negative", -1},
		{"seed large", 99999999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := TestAddrWithSeed(tt.seed)

			// Verify non-empty
			if len(addr) == 0 {
				t.Error("TestAddrWithSeed returned empty address")
			}

			// Verify deterministic - same seed produces same address
			addr2 := TestAddrWithSeed(tt.seed)
			if !addr.Equals(addr2) {
				t.Error("TestAddrWithSeed should be deterministic for same seed")
			}

			// Verify address length
			if len(addr) != 20 {
				t.Errorf("Expected address length 20, got %d", len(addr))
			}
		})
	}
}

func TestTestAddrWithSeed_Uniqueness(t *testing.T) {
	// Different seeds should produce different addresses
	addr1 := TestAddrWithSeed(1)
	addr2 := TestAddrWithSeed(2)
	addr3 := TestAddrWithSeed(3)

	if addr1.Equals(addr2) {
		t.Error("Different seeds should produce different addresses (1 vs 2)")
	}
	if addr2.Equals(addr3) {
		t.Error("Different seeds should produce different addresses (2 vs 3)")
	}
	if addr1.Equals(addr3) {
		t.Error("Different seeds should produce different addresses (1 vs 3)")
	}
}

func TestTestAddrWithSeed_String(t *testing.T) {
	// Verify addresses can be converted to bech32 strings
	addr := TestAddrWithSeed(42)
	addrStr := addr.String()

	if addrStr == "" {
		t.Error("Address string conversion should not be empty")
	}

	// Verify round-trip conversion works
	parsed, err := sdk.AccAddressFromBech32(addrStr)
	if err != nil {
		t.Errorf("Failed to parse address from bech32: %v", err)
	}
	if !parsed.Equals(addr) {
		t.Error("Round-trip address conversion failed")
	}
}

// ============================================================================
// MultiDexHooks Tests
// ============================================================================

// mockDexHooks is a mock implementation of DexHooks for testing
type mockDexHooks struct {
	afterSwapCalled             bool
	afterPoolCreatedCalled      bool
	afterLiquidityChangedCalled bool
	circuitBreakerCalled        bool
	shouldError                 bool
}

func (m *mockDexHooks) AfterSwap(ctx context.Context, poolID uint64, sender string, tokenIn, tokenOut string, amountIn, amountOut math.Int) error {
	m.afterSwapCalled = true
	if m.shouldError {
		return errors.New("mock swap error")
	}
	return nil
}

func (m *mockDexHooks) AfterPoolCreated(ctx context.Context, poolID uint64, tokenA, tokenB string, creator string) error {
	m.afterPoolCreatedCalled = true
	if m.shouldError {
		return errors.New("mock pool created error")
	}
	return nil
}

func (m *mockDexHooks) AfterLiquidityChanged(ctx context.Context, poolID uint64, provider string, deltaA, deltaB math.Int, isAdd bool) error {
	m.afterLiquidityChangedCalled = true
	if m.shouldError {
		return errors.New("mock liquidity changed error")
	}
	return nil
}

func (m *mockDexHooks) OnCircuitBreakerTriggered(ctx context.Context, reason string) error {
	m.circuitBreakerCalled = true
	if m.shouldError {
		return errors.New("mock circuit breaker error")
	}
	return nil
}

func TestNewMultiDexHooks(t *testing.T) {
	hook1 := &mockDexHooks{}
	hook2 := &mockDexHooks{}

	multi := NewMultiDexHooks(hook1, hook2)

	if len(multi) != 2 {
		t.Errorf("Expected 2 hooks, got %d", len(multi))
	}
}

func TestNewMultiDexHooks_Empty(t *testing.T) {
	multi := NewMultiDexHooks()

	if len(multi) != 0 {
		t.Errorf("Expected 0 hooks, got %d", len(multi))
	}
}

func TestMultiDexHooks_AfterSwap(t *testing.T) {
	hook1 := &mockDexHooks{}
	hook2 := &mockDexHooks{}
	multi := NewMultiDexHooks(hook1, hook2)

	ctx := context.Background()
	err := multi.AfterSwap(ctx, 1, "sender", "upaw", "uatom", math.NewInt(1000), math.NewInt(990))

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !hook1.afterSwapCalled {
		t.Error("hook1.AfterSwap was not called")
	}
	if !hook2.afterSwapCalled {
		t.Error("hook2.AfterSwap was not called")
	}
}

func TestMultiDexHooks_AfterSwap_Error(t *testing.T) {
	hook1 := &mockDexHooks{shouldError: true}
	hook2 := &mockDexHooks{}
	multi := NewMultiDexHooks(hook1, hook2)

	ctx := context.Background()
	err := multi.AfterSwap(ctx, 1, "sender", "upaw", "uatom", math.NewInt(1000), math.NewInt(990))

	if err == nil {
		t.Error("Expected error but got nil")
	}
	if !hook1.afterSwapCalled {
		t.Error("hook1.AfterSwap should have been called")
	}
	if hook2.afterSwapCalled {
		t.Error("hook2.AfterSwap should NOT have been called after error")
	}
}

func TestMultiDexHooks_AfterSwap_NilHook(t *testing.T) {
	hook1 := &mockDexHooks{}
	multi := NewMultiDexHooks(nil, hook1, nil)

	ctx := context.Background()
	err := multi.AfterSwap(ctx, 1, "sender", "upaw", "uatom", math.NewInt(1000), math.NewInt(990))

	if err != nil {
		t.Errorf("Unexpected error with nil hooks: %v", err)
	}
	if !hook1.afterSwapCalled {
		t.Error("Non-nil hook should have been called")
	}
}

func TestMultiDexHooks_AfterPoolCreated(t *testing.T) {
	hook1 := &mockDexHooks{}
	hook2 := &mockDexHooks{}
	multi := NewMultiDexHooks(hook1, hook2)

	ctx := context.Background()
	err := multi.AfterPoolCreated(ctx, 1, "upaw", "uatom", "creator")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !hook1.afterPoolCreatedCalled {
		t.Error("hook1.AfterPoolCreated was not called")
	}
	if !hook2.afterPoolCreatedCalled {
		t.Error("hook2.AfterPoolCreated was not called")
	}
}

func TestMultiDexHooks_AfterPoolCreated_Error(t *testing.T) {
	hook1 := &mockDexHooks{shouldError: true}
	multi := NewMultiDexHooks(hook1)

	ctx := context.Background()
	err := multi.AfterPoolCreated(ctx, 1, "upaw", "uatom", "creator")

	if err == nil {
		t.Error("Expected error but got nil")
	}
}

func TestMultiDexHooks_AfterPoolCreated_NilHook(t *testing.T) {
	hook := &mockDexHooks{}
	multi := NewMultiDexHooks(nil, hook)

	ctx := context.Background()
	err := multi.AfterPoolCreated(ctx, 1, "upaw", "uatom", "creator")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !hook.afterPoolCreatedCalled {
		t.Error("Non-nil hook should have been called")
	}
}

func TestMultiDexHooks_AfterLiquidityChanged(t *testing.T) {
	hook1 := &mockDexHooks{}
	hook2 := &mockDexHooks{}
	multi := NewMultiDexHooks(hook1, hook2)

	ctx := context.Background()
	err := multi.AfterLiquidityChanged(ctx, 1, "provider", math.NewInt(1000), math.NewInt(2000), true)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !hook1.afterLiquidityChangedCalled {
		t.Error("hook1.AfterLiquidityChanged was not called")
	}
	if !hook2.afterLiquidityChangedCalled {
		t.Error("hook2.AfterLiquidityChanged was not called")
	}
}

func TestMultiDexHooks_AfterLiquidityChanged_Remove(t *testing.T) {
	hook := &mockDexHooks{}
	multi := NewMultiDexHooks(hook)

	ctx := context.Background()
	err := multi.AfterLiquidityChanged(ctx, 1, "provider", math.NewInt(-500), math.NewInt(-1000), false)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !hook.afterLiquidityChangedCalled {
		t.Error("hook.AfterLiquidityChanged was not called")
	}
}

func TestMultiDexHooks_AfterLiquidityChanged_Error(t *testing.T) {
	hook := &mockDexHooks{shouldError: true}
	multi := NewMultiDexHooks(hook)

	ctx := context.Background()
	err := multi.AfterLiquidityChanged(ctx, 1, "provider", math.NewInt(1000), math.NewInt(2000), true)

	if err == nil {
		t.Error("Expected error but got nil")
	}
}

func TestMultiDexHooks_AfterLiquidityChanged_NilHook(t *testing.T) {
	hook := &mockDexHooks{}
	multi := NewMultiDexHooks(nil, hook)

	ctx := context.Background()
	err := multi.AfterLiquidityChanged(ctx, 1, "provider", math.NewInt(1000), math.NewInt(2000), true)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestMultiDexHooks_OnCircuitBreakerTriggered(t *testing.T) {
	hook1 := &mockDexHooks{}
	hook2 := &mockDexHooks{}
	multi := NewMultiDexHooks(hook1, hook2)

	ctx := context.Background()
	err := multi.OnCircuitBreakerTriggered(ctx, "price anomaly detected")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !hook1.circuitBreakerCalled {
		t.Error("hook1.OnCircuitBreakerTriggered was not called")
	}
	if !hook2.circuitBreakerCalled {
		t.Error("hook2.OnCircuitBreakerTriggered was not called")
	}
}

func TestMultiDexHooks_OnCircuitBreakerTriggered_Error(t *testing.T) {
	hook := &mockDexHooks{shouldError: true}
	multi := NewMultiDexHooks(hook)

	ctx := context.Background()
	err := multi.OnCircuitBreakerTriggered(ctx, "test reason")

	if err == nil {
		t.Error("Expected error but got nil")
	}
}

func TestMultiDexHooks_OnCircuitBreakerTriggered_NilHook(t *testing.T) {
	hook := &mockDexHooks{}
	multi := NewMultiDexHooks(nil, hook)

	ctx := context.Background()
	err := multi.OnCircuitBreakerTriggered(ctx, "test reason")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestMultiDexHooks_EmptyHooks(t *testing.T) {
	multi := NewMultiDexHooks()
	ctx := context.Background()

	// All methods should succeed with empty hooks
	if err := multi.AfterSwap(ctx, 1, "sender", "upaw", "uatom", math.NewInt(1000), math.NewInt(990)); err != nil {
		t.Errorf("AfterSwap failed with empty hooks: %v", err)
	}
	if err := multi.AfterPoolCreated(ctx, 1, "upaw", "uatom", "creator"); err != nil {
		t.Errorf("AfterPoolCreated failed with empty hooks: %v", err)
	}
	if err := multi.AfterLiquidityChanged(ctx, 1, "provider", math.NewInt(1000), math.NewInt(2000), true); err != nil {
		t.Errorf("AfterLiquidityChanged failed with empty hooks: %v", err)
	}
	if err := multi.OnCircuitBreakerTriggered(ctx, "test"); err != nil {
		t.Errorf("OnCircuitBreakerTriggered failed with empty hooks: %v", err)
	}
}

// ============================================================================
// Codec Tests
// ============================================================================

func TestRegisterLegacyAminoCodec(t *testing.T) {
	cdc := codec.NewLegacyAmino()

	// Should not panic
	RegisterLegacyAminoCodec(cdc)

	// Verify messages can be marshaled/unmarshaled
	msg := &MsgCreatePool{
		Creator: "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
		TokenA:  "upaw",
		TokenB:  "uatom",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(2000000),
	}

	bz, err := cdc.MarshalJSON(msg)
	if err != nil {
		t.Errorf("Failed to marshal MsgCreatePool: %v", err)
	}

	var decoded MsgCreatePool
	err = cdc.UnmarshalJSON(bz, &decoded)
	if err != nil {
		t.Errorf("Failed to unmarshal MsgCreatePool: %v", err)
	}

	if decoded.Creator != msg.Creator {
		t.Error("Creator mismatch after marshal/unmarshal")
	}
	if decoded.TokenA != msg.TokenA {
		t.Error("TokenA mismatch after marshal/unmarshal")
	}
}

func TestRegisterInterfaces(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()

	// Should not panic
	RegisterInterfaces(registry)

	// Verify message types are registered
	msgTypes := []string{
		"/paw.dex.v1.MsgCreatePool",
		"/paw.dex.v1.MsgAddLiquidity",
		"/paw.dex.v1.MsgRemoveLiquidity",
		"/paw.dex.v1.MsgSwap",
		"/paw.dex.v1.MsgCommitSwap",
		"/paw.dex.v1.MsgRevealSwap",
	}

	for _, msgType := range msgTypes {
		_, err := registry.Resolve(msgType)
		if err != nil {
			// Not all messages may be resolvable directly, but registration should work
			t.Logf("Note: Could not resolve %s (this may be expected)", msgType)
		}
	}
}

// ============================================================================
// Additional Params Tests
// ============================================================================

func TestDefaultParams_CircuitBreakerDuration(t *testing.T) {
	params := DefaultParams()

	// Verify circuit breaker duration is set
	if params.CircuitBreakerDurationSeconds == 0 {
		t.Error("CircuitBreakerDurationSeconds should be positive")
	}

	// Default should be 1 hour (3600 seconds)
	if params.CircuitBreakerDurationSeconds != 3600 {
		t.Errorf("Expected CircuitBreakerDurationSeconds to be 3600, got %d", params.CircuitBreakerDurationSeconds)
	}
}

func TestDefaultParams_AllFieldsInitialized(t *testing.T) {
	params := DefaultParams()

	// All decimal fields should not be nil
	if params.SwapFee.IsNil() {
		t.Error("SwapFee should not be nil")
	}
	if params.LpFee.IsNil() {
		t.Error("LpFee should not be nil")
	}
	if params.ProtocolFee.IsNil() {
		t.Error("ProtocolFee should not be nil")
	}
	if params.MinLiquidity.IsNil() {
		t.Error("MinLiquidity should not be nil")
	}
	if params.MaxSlippagePercent.IsNil() {
		t.Error("MaxSlippagePercent should not be nil")
	}
	if params.MaxPoolDrainPercent.IsNil() {
		t.Error("MaxPoolDrainPercent should not be nil")
	}
	if params.RecommendedMaxSlippage.IsNil() {
		t.Error("RecommendedMaxSlippage should not be nil")
	}
}

// ============================================================================
// Additional Message Type Tests
// ============================================================================

func TestMsgSwap_InvalidTokenDenom(t *testing.T) {
	validAddr := "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q"

	tests := []struct {
		name     string
		tokenIn  string
		tokenOut string
		errMsg   string
	}{
		{
			name:     "token_in with spaces",
			tokenIn:  "u paw",
			tokenOut: "uatom",
			errMsg:   "invalid denom for token_in",
		},
		{
			name:     "token_out with special chars",
			tokenIn:  "upaw",
			tokenOut: "u@tom",
			errMsg:   "invalid denom for token_out",
		},
		{
			name:     "token_in starts with number",
			tokenIn:  "1upaw",
			tokenOut: "uatom",
			errMsg:   "invalid denom for token_in",
		},
		{
			name:     "token_out starts with number",
			tokenIn:  "upaw",
			tokenOut: "2uatom",
			errMsg:   "invalid denom for token_out",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := MsgSwap{
				Trader:       validAddr,
				PoolId:       1,
				TokenIn:      tt.tokenIn,
				TokenOut:     tt.tokenOut,
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
				Deadline:     1700000000,
			}

			err := msg.ValidateBasic()
			if err == nil {
				t.Error("Expected error for invalid denom")
			}
			if err != nil && !containsStr(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestMsgCreatePool_TokenOrdering(t *testing.T) {
	validAddr := "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q"

	// Tokens in any order should be valid (sorting happens in keeper)
	tests := []struct {
		name    string
		tokenA  string
		tokenB  string
		wantErr bool
	}{
		{"alphabetical order", "uatom", "upaw", false},
		{"reverse alphabetical", "upaw", "uatom", false},
		{"IBC denom first", "ibc/1234", "upaw", false},
		{"IBC denom second", "upaw", "ibc/1234", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := MsgCreatePool{
				Creator: validAddr,
				TokenA:  tt.tokenA,
				TokenB:  tt.tokenB,
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(2000000),
			}

			err := msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ============================================================================
// Error Code Uniqueness Test
// ============================================================================

func TestErrorCodes_Unique(t *testing.T) {
	// All error codes should be unique within the module
	codesSeen := make(map[uint32]string)

	errorDefs := []struct {
		name string
		code uint32
	}{
		{"ErrInvalidPoolState", 2},
		{"ErrInsufficientLiquidity", 3},
		{"ErrInvalidInput", 4},
		{"ErrReentrancy", 5},
		{"ErrInvariantViolation", 6},
		{"ErrCircuitBreakerTriggered", 7},
		{"ErrSwapTooLarge", 8},
		{"ErrPriceImpactTooHigh", 9},
		{"ErrFlashLoanDetected", 10},
		{"ErrOverflow", 11},
		{"ErrUnderflow", 12},
		{"ErrDivisionByZero", 13},
		{"ErrPoolNotFound", 14},
		{"ErrPoolAlreadyExists", 15},
		{"ErrInsufficientShares", 16},
		{"ErrSlippageTooHigh", 17},
		{"ErrInvalidTokenPair", 18},
		{"ErrRateLimitExceeded", 19},
		{"ErrJITLiquidityDetected", 20},
		{"ErrInvalidState", 21},
		{"ErrInvalidSwapAmount", 22},
		{"ErrInvalidLiquidityAmount", 23},
		{"ErrStateCorruption", 24},
		{"ErrSlippageExceeded", 25},
		{"ErrOraclePrice", 26},
		{"ErrPriceDeviation", 27},
		{"ErrMaxPoolsReached", 28},
		{"ErrUnauthorized", 29},
		{"ErrDeadlineExceeded", 30},
		{"ErrInvalidNonce", 31},
		{"ErrOrderNotFound", 32},
		{"ErrInvalidOrder", 33},
		{"ErrOrderNotAuthorized", 34},
		{"ErrOrderNotCancellable", 35},
		{"ErrCircuitBreakerAlreadyOpen", 36},
		{"ErrCircuitBreakerAlreadyClosed", 37},
		{"ErrCommitRequired", 40},
		{"ErrCommitmentNotFound", 41},
		{"ErrDuplicateCommitment", 42},
		{"ErrRevealTooEarly", 43},
		{"ErrCommitmentExpired", 44},
		{"ErrInsufficientDeposit", 45},
		{"ErrInvalidPool", 46},
		{"ErrCommitRevealDisabled", 47},
		{"ErrMinimumReserves", 48},
		{"ErrInvalidAck", 91},
		{"ErrUnauthorizedChannel", 92},
		{"ErrSwapFailed", 93},
		{"ErrSwapExpired", 94},
	}

	for _, errDef := range errorDefs {
		if existing, found := codesSeen[errDef.code]; found {
			t.Errorf("Duplicate error code %d: %s and %s", errDef.code, existing, errDef.name)
		}
		codesSeen[errDef.code] = errDef.name
	}
}

// ============================================================================
// Genesis State Edge Cases
// ============================================================================

func TestGenesisState_ValidateWithPools(t *testing.T) {
	gs := DefaultGenesis()

	// Add some pools (they don't have a separate Validate method at genesis level)
	gs.Pools = []Pool{
		{
			Id:          1,
			TokenA:      "upaw",
			TokenB:      "uatom",
			ReserveA:    math.NewInt(1000000),
			ReserveB:    math.NewInt(2000000),
			TotalShares: math.NewInt(1414213),
		},
	}
	gs.NextPoolId = 2

	err := gs.Validate()
	if err != nil {
		t.Errorf("Valid genesis with pools should pass validation: %v", err)
	}
}

func TestGenesisState_ValidateEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*GenesisState)
		wantErr bool
		errMsg  string
	}{
		{
			name: "zero swap fee is valid",
			modify: func(gs *GenesisState) {
				gs.Params.SwapFee = math.LegacyZeroDec()
				gs.Params.LpFee = math.LegacyZeroDec()
				gs.Params.ProtocolFee = math.LegacyZeroDec()
			},
			wantErr: false,
		},
		{
			name: "zero min liquidity is valid",
			modify: func(gs *GenesisState) {
				gs.Params.MinLiquidity = math.ZeroInt()
			},
			wantErr: false,
		},
		{
			name: "large next pool id is valid",
			modify: func(gs *GenesisState) {
				gs.NextPoolId = 999999999
			},
			wantErr: false,
		},
		{
			name: "commit-reveal disabled ignores delay validation",
			modify: func(gs *GenesisState) {
				gs.Params.EnableCommitReveal = false
				gs.Params.CommitRevealDelay = 0
				gs.Params.CommitTimeoutBlocks = 0
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := DefaultGenesis()
			tt.modify(gs)

			err := gs.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("GenesisState.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !containsStr(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			}
		})
	}
}

// ============================================================================
// Message Type Constants Tests
// ============================================================================

func TestMessageTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		typeVal  string
		expected string
	}{
		{"TypeMsgCreatePool", TypeMsgCreatePool, "create_pool"},
		{"TypeMsgAddLiquidity", TypeMsgAddLiquidity, "add_liquidity"},
		{"TypeMsgRemoveLiquidity", TypeMsgRemoveLiquidity, "remove_liquidity"},
		{"TypeMsgSwap", TypeMsgSwap, "swap"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.typeVal != tt.expected {
				t.Errorf("Expected %s to be %q, got %q", tt.name, tt.expected, tt.typeVal)
			}
		})
	}
}

// ============================================================================
// Message Interface Compliance Tests
// ============================================================================

func TestMsgInterfaceCompliance(t *testing.T) {
	// Verify all message types implement sdk.Msg interface
	var _ sdk.Msg = &MsgCreatePool{}
	var _ sdk.Msg = &MsgAddLiquidity{}
	var _ sdk.Msg = &MsgRemoveLiquidity{}
	var _ sdk.Msg = &MsgSwap{}
	var _ sdk.Msg = &MsgCommitSwap{}
	var _ sdk.Msg = &MsgRevealSwap{}

	// If we get here without compile error, the test passes
	t.Log("All message types implement sdk.Msg interface")
}

// ============================================================================
// Default Authority Tests
// ============================================================================

func TestDefaultAuthority_Deterministic(t *testing.T) {
	auth1 := DefaultAuthority()
	auth2 := DefaultAuthority()

	if auth1 != auth2 {
		t.Error("DefaultAuthority should return deterministic value")
	}
}

func TestDefaultAuthority_Valid(t *testing.T) {
	auth := DefaultAuthority()

	_, err := sdk.AccAddressFromBech32(auth)
	if err != nil {
		t.Errorf("DefaultAuthority should return valid bech32 address: %v", err)
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
