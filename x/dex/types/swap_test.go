package types

import (
	"testing"

	"cosmossdk.io/math"
)

func TestSwapResult_Fields(t *testing.T) {
	// Test SwapResult structure and field types
	result := SwapResult{
		AmountIn:     math.NewInt(1000000),
		AmountOut:    math.NewInt(990000),
		Route:        "pool-1",
		Slippage:     math.LegacyMustNewDecFromStr("0.01"),
		PriceImpact:  math.LegacyMustNewDecFromStr("0.005"),
		Fee:          math.NewInt(3000),
		NewPoolPrice: math.LegacyMustNewDecFromStr("1.5"),
	}

	// Verify AmountIn
	if !result.AmountIn.Equal(math.NewInt(1000000)) {
		t.Errorf("Expected AmountIn 1000000, got %s", result.AmountIn.String())
	}

	// Verify AmountOut
	if !result.AmountOut.Equal(math.NewInt(990000)) {
		t.Errorf("Expected AmountOut 990000, got %s", result.AmountOut.String())
	}

	// Verify Route
	if result.Route != "pool-1" {
		t.Errorf("Expected Route 'pool-1', got %s", result.Route)
	}

	// Verify Slippage
	expectedSlippage := math.LegacyMustNewDecFromStr("0.01")
	if !result.Slippage.Equal(expectedSlippage) {
		t.Errorf("Expected Slippage %s, got %s", expectedSlippage, result.Slippage)
	}

	// Verify PriceImpact
	expectedPriceImpact := math.LegacyMustNewDecFromStr("0.005")
	if !result.PriceImpact.Equal(expectedPriceImpact) {
		t.Errorf("Expected PriceImpact %s, got %s", expectedPriceImpact, result.PriceImpact)
	}

	// Verify Fee
	if !result.Fee.Equal(math.NewInt(3000)) {
		t.Errorf("Expected Fee 3000, got %s", result.Fee.String())
	}

	// Verify NewPoolPrice
	expectedPrice := math.LegacyMustNewDecFromStr("1.5")
	if !result.NewPoolPrice.Equal(expectedPrice) {
		t.Errorf("Expected NewPoolPrice %s, got %s", expectedPrice, result.NewPoolPrice)
	}
}

func TestSwapResult_ZeroValues(t *testing.T) {
	// Test SwapResult with zero values
	result := SwapResult{
		AmountIn:     math.ZeroInt(),
		AmountOut:    math.ZeroInt(),
		Route:        "",
		Slippage:     math.LegacyZeroDec(),
		PriceImpact:  math.LegacyZeroDec(),
		Fee:          math.ZeroInt(),
		NewPoolPrice: math.LegacyZeroDec(),
	}

	if !result.AmountIn.IsZero() {
		t.Error("Expected AmountIn to be zero")
	}

	if !result.AmountOut.IsZero() {
		t.Error("Expected AmountOut to be zero")
	}

	if result.Route != "" {
		t.Errorf("Expected empty Route, got %s", result.Route)
	}

	if !result.Slippage.IsZero() {
		t.Error("Expected Slippage to be zero")
	}

	if !result.PriceImpact.IsZero() {
		t.Error("Expected PriceImpact to be zero")
	}

	if !result.Fee.IsZero() {
		t.Error("Expected Fee to be zero")
	}

	if !result.NewPoolPrice.IsZero() {
		t.Error("Expected NewPoolPrice to be zero")
	}
}

func TestSwapResult_LargeValues(t *testing.T) {
	// Test SwapResult with large values
	largeInt := math.NewIntFromUint64(^uint64(0)) // Max uint64
	largeDec := math.LegacyMustNewDecFromStr("999999999999.999999999999999999")

	result := SwapResult{
		AmountIn:     largeInt,
		AmountOut:    largeInt,
		Route:        "very-long-route-name-with-multiple-hops-pool-1-pool-2-pool-3",
		Slippage:     largeDec,
		PriceImpact:  largeDec,
		Fee:          largeInt,
		NewPoolPrice: largeDec,
	}

	if !result.AmountIn.Equal(largeInt) {
		t.Error("Failed to store large AmountIn")
	}

	if !result.AmountOut.Equal(largeInt) {
		t.Error("Failed to store large AmountOut")
	}

	if len(result.Route) == 0 {
		t.Error("Failed to store long route")
	}
}

func TestSwapResult_NegativeValues(t *testing.T) {
	// While SwapResult doesn't validate, test that negative values can be stored
	// (validation should happen at higher levels)
	negativeInt := math.NewInt(-1000)
	negativeDec := math.LegacyMustNewDecFromStr("-0.5")

	result := SwapResult{
		AmountIn:     negativeInt,
		AmountOut:    negativeInt,
		Route:        "test",
		Slippage:     negativeDec,
		PriceImpact:  negativeDec,
		Fee:          negativeInt,
		NewPoolPrice: negativeDec,
	}

	if result.AmountIn.IsPositive() {
		t.Error("Expected negative AmountIn")
	}

	if result.AmountOut.IsPositive() {
		t.Error("Expected negative AmountOut")
	}

	if result.Slippage.IsPositive() {
		t.Error("Expected negative Slippage")
	}

	if result.PriceImpact.IsPositive() {
		t.Error("Expected negative PriceImpact")
	}

	if result.Fee.IsPositive() {
		t.Error("Expected negative Fee")
	}

	if result.NewPoolPrice.IsPositive() {
		t.Error("Expected negative NewPoolPrice")
	}
}

func TestSwapResult_MultiHopRoute(t *testing.T) {
	// Test SwapResult with multi-hop route
	routes := []string{
		"pool-1",
		"pool-1->pool-2",
		"pool-1->pool-2->pool-3",
		"chain-a/pool-1->chain-b/pool-2",
		"",
	}

	for _, route := range routes {
		result := SwapResult{
			AmountIn:     math.NewInt(1000000),
			AmountOut:    math.NewInt(990000),
			Route:        route,
			Slippage:     math.LegacyMustNewDecFromStr("0.01"),
			PriceImpact:  math.LegacyMustNewDecFromStr("0.005"),
			Fee:          math.NewInt(3000),
			NewPoolPrice: math.LegacyMustNewDecFromStr("1.5"),
		}

		if result.Route != route {
			t.Errorf("Expected route %q, got %q", route, result.Route)
		}
	}
}

func TestSwapResult_RealisticScenarios(t *testing.T) {
	tests := []struct {
		name        string
		amountIn    math.Int
		amountOut   math.Int
		route       string
		slippage    math.LegacyDec
		priceImpact math.LegacyDec
		fee         math.Int
		newPrice    math.LegacyDec
	}{
		{
			name:        "small swap low slippage",
			amountIn:    math.NewInt(1000000),      // 1 token with 6 decimals
			amountOut:   math.NewInt(995000),       // 0.5% slippage
			route:       "pool-1",
			slippage:    math.LegacyMustNewDecFromStr("0.005"),
			priceImpact: math.LegacyMustNewDecFromStr("0.003"),
			fee:         math.NewInt(3000),
			newPrice:    math.LegacyMustNewDecFromStr("1.003"),
		},
		{
			name:        "large swap high impact",
			amountIn:    math.NewInt(100000000),    // 100 tokens
			amountOut:   math.NewInt(90000000),     // 10% slippage
			route:       "pool-1",
			slippage:    math.LegacyMustNewDecFromStr("0.10"),
			priceImpact: math.LegacyMustNewDecFromStr("0.08"),
			fee:         math.NewInt(300000),
			newPrice:    math.LegacyMustNewDecFromStr("1.11"),
		},
		{
			name:        "multi-hop swap",
			amountIn:    math.NewInt(10000000),
			amountOut:   math.NewInt(9700000),
			route:       "pool-1->pool-2->pool-3",
			slippage:    math.LegacyMustNewDecFromStr("0.03"),
			priceImpact: math.LegacyMustNewDecFromStr("0.015"),
			fee:         math.NewInt(90000),
			newPrice:    math.LegacyMustNewDecFromStr("1.02"),
		},
		{
			name:        "zero fee swap",
			amountIn:    math.NewInt(1000000),
			amountOut:   math.NewInt(1000000),
			route:       "pool-special",
			slippage:    math.LegacyZeroDec(),
			priceImpact: math.LegacyZeroDec(),
			fee:         math.ZeroInt(),
			newPrice:    math.LegacyOneDec(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SwapResult{
				AmountIn:     tt.amountIn,
				AmountOut:    tt.amountOut,
				Route:        tt.route,
				Slippage:     tt.slippage,
				PriceImpact:  tt.priceImpact,
				Fee:          tt.fee,
				NewPoolPrice: tt.newPrice,
			}

			// Verify all fields match
			if !result.AmountIn.Equal(tt.amountIn) {
				t.Errorf("AmountIn mismatch: expected %s, got %s", tt.amountIn, result.AmountIn)
			}
			if !result.AmountOut.Equal(tt.amountOut) {
				t.Errorf("AmountOut mismatch: expected %s, got %s", tt.amountOut, result.AmountOut)
			}
			if result.Route != tt.route {
				t.Errorf("Route mismatch: expected %s, got %s", tt.route, result.Route)
			}
			if !result.Slippage.Equal(tt.slippage) {
				t.Errorf("Slippage mismatch: expected %s, got %s", tt.slippage, result.Slippage)
			}
			if !result.PriceImpact.Equal(tt.priceImpact) {
				t.Errorf("PriceImpact mismatch: expected %s, got %s", tt.priceImpact, result.PriceImpact)
			}
			if !result.Fee.Equal(tt.fee) {
				t.Errorf("Fee mismatch: expected %s, got %s", tt.fee, result.Fee)
			}
			if !result.NewPoolPrice.Equal(tt.newPrice) {
				t.Errorf("NewPoolPrice mismatch: expected %s, got %s", tt.newPrice, result.NewPoolPrice)
			}
		})
	}
}

func TestSwapResult_PrecisionHandling(t *testing.T) {
	// Test decimal precision handling
	result := SwapResult{
		AmountIn:     math.NewInt(1000000000000000000), // 1 token with 18 decimals
		AmountOut:    math.NewInt(997000000000000000),  // High precision output
		Route:        "pool-precise",
		Slippage:     math.LegacyMustNewDecFromStr("0.003000000000000000"),
		PriceImpact:  math.LegacyMustNewDecFromStr("0.001500000000000000"),
		Fee:          math.NewInt(3000000000000000),
		NewPoolPrice: math.LegacyMustNewDecFromStr("1.003015045135406218"), // High precision price
	}

	// Verify precision is maintained
	if result.AmountIn.String() != "1000000000000000000" {
		t.Errorf("Lost precision in AmountIn: %s", result.AmountIn.String())
	}

	if result.AmountOut.String() != "997000000000000000" {
		t.Errorf("Lost precision in AmountOut: %s", result.AmountOut.String())
	}

	// Decimal precision should be maintained (up to 18 decimals in LegacyDec)
	slippageStr := result.Slippage.String()
	if len(slippageStr) == 0 {
		t.Error("Slippage string representation is empty")
	}
}

func TestSwapResult_Comparison(t *testing.T) {
	// Test comparing SwapResult values
	result1 := SwapResult{
		AmountIn:     math.NewInt(1000000),
		AmountOut:    math.NewInt(990000),
		Route:        "pool-1",
		Slippage:     math.LegacyMustNewDecFromStr("0.01"),
		PriceImpact:  math.LegacyMustNewDecFromStr("0.005"),
		Fee:          math.NewInt(3000),
		NewPoolPrice: math.LegacyMustNewDecFromStr("1.5"),
	}

	result2 := SwapResult{
		AmountIn:     math.NewInt(1000000),
		AmountOut:    math.NewInt(990000),
		Route:        "pool-1",
		Slippage:     math.LegacyMustNewDecFromStr("0.01"),
		PriceImpact:  math.LegacyMustNewDecFromStr("0.005"),
		Fee:          math.NewInt(3000),
		NewPoolPrice: math.LegacyMustNewDecFromStr("1.5"),
	}

	// Verify equality
	if !result1.AmountIn.Equal(result2.AmountIn) {
		t.Error("AmountIn should be equal")
	}
	if !result1.AmountOut.Equal(result2.AmountOut) {
		t.Error("AmountOut should be equal")
	}
	if result1.Route != result2.Route {
		t.Error("Route should be equal")
	}
	if !result1.Slippage.Equal(result2.Slippage) {
		t.Error("Slippage should be equal")
	}
	if !result1.PriceImpact.Equal(result2.PriceImpact) {
		t.Error("PriceImpact should be equal")
	}
	if !result1.Fee.Equal(result2.Fee) {
		t.Error("Fee should be equal")
	}
	if !result1.NewPoolPrice.Equal(result2.NewPoolPrice) {
		t.Error("NewPoolPrice should be equal")
	}
}
