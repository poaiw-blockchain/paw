package fuzz

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/gogoproto/proto"

	computetypes "github.com/paw-chain/paw/x/compute/types"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// FuzzDexPoolProto tests Pool protobuf serialization/deserialization
func FuzzDexPoolProto(f *testing.F) {
	// Seed with valid pool data
	seeds := [][]byte{
		mustMarshal(&dextypes.Pool{
			Id:          1,
			TokenA:      "uatom",
			TokenB:      "uosmo",
			ReserveA:    math.NewInt(1000000),
			ReserveB:    math.NewInt(2000000),
			TotalShares: math.NewInt(1414213),
			Creator:     "cosmos1test",
		}),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var pool dextypes.Pool

		// Attempt to unmarshal
		err := proto.Unmarshal(data, &pool)
		if err != nil {
			// Invalid protobuf is acceptable
			return
		}

		// If unmarshal succeeded, validate invariants
		// INVARIANT 1: Reserves must be non-negative
		if pool.ReserveA.IsNegative() {
			t.Errorf("VIOLATION: negative ReserveA: %s", pool.ReserveA.String())
		}
		if pool.ReserveB.IsNegative() {
			t.Errorf("VIOLATION: negative ReserveB: %s", pool.ReserveB.String())
		}

		// INVARIANT 2: Total shares must be non-negative
		if pool.TotalShares.IsNegative() {
			t.Errorf("VIOLATION: negative TotalShares: %s", pool.TotalShares.String())
		}

		// INVARIANT 3: Re-marshaling should be deterministic
		remarshaled, err := proto.Marshal(&pool)
		if err != nil {
			t.Errorf("Failed to re-marshal: %v", err)
			return
		}

		// Unmarshal again to compare
		var pool2 dextypes.Pool
		err = proto.Unmarshal(remarshaled, &pool2)
		if err != nil {
			t.Errorf("Failed to unmarshal remarshaled data: %v", err)
			return
		}

		// Check equality
		if !pool.ReserveA.Equal(pool2.ReserveA) {
			t.Errorf("ReserveA mismatch after roundtrip")
		}
		if !pool.ReserveB.Equal(pool2.ReserveB) {
			t.Errorf("ReserveB mismatch after roundtrip")
		}
		if !pool.TotalShares.Equal(pool2.TotalShares) {
			t.Errorf("TotalShares mismatch after roundtrip")
		}
	})
}

// FuzzOraclePriceProto tests Price protobuf serialization/deserialization
func FuzzOraclePriceProto(f *testing.F) {
	seeds := [][]byte{
		mustMarshal(&oracletypes.Price{
			Asset:         "BTC",
			Price:         math.LegacyMustNewDecFromStr("50000.50"),
			BlockHeight:   12345,
			BlockTime:     1234567890,
			NumValidators: 10,
		}),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var price oracletypes.Price

		err := proto.Unmarshal(data, &price)
		if err != nil {
			return
		}

		// INVARIANT 1: Price must be non-negative
		if price.Price.IsNegative() {
			t.Errorf("VIOLATION: negative price: %s", price.Price.String())
		}

		// INVARIANT 2: BlockTime should be reasonable (Unix timestamp)
		if price.BlockTime < 0 {
			t.Errorf("VIOLATION: negative block time: %d", price.BlockTime)
		}

		// Test roundtrip
		remarshaled, err := proto.Marshal(&price)
		if err != nil {
			t.Errorf("Failed to re-marshal: %v", err)
			return
		}

		var price2 oracletypes.Price
		err = proto.Unmarshal(remarshaled, &price2)
		if err != nil {
			t.Errorf("Failed to unmarshal remarshaled data: %v", err)
			return
		}

		if !price.Price.Equal(price2.Price) {
			t.Errorf("Price mismatch after roundtrip")
		}
	})
}

// FuzzComputeRequestProto tests compute Request protobuf
func FuzzComputeRequestProto(f *testing.F) {
	seeds := [][]byte{
		mustMarshal(&computetypes.Request{
			Id:             1,
			Requester:      "cosmos1requester",
			Provider:       "cosmos1provider",
			ContainerImage: "ubuntu:latest",
			Command:        []string{"echo", "hello"},
			Status:         computetypes.REQUEST_STATUS_PENDING,
			MaxPayment:     math.NewInt(1000000),
			EscrowedAmount: math.NewInt(1000000),
		}),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var request computetypes.Request

		err := proto.Unmarshal(data, &request)
		if err != nil {
			return
		}

		// INVARIANT 1: MaxPayment must be non-negative
		if request.MaxPayment.IsNegative() {
			t.Errorf("VIOLATION: negative max payment: %s", request.MaxPayment.String())
		}

		// INVARIANT 2: EscrowedAmount must be non-negative
		if request.EscrowedAmount.IsNegative() {
			t.Errorf("VIOLATION: negative escrowed amount: %s", request.EscrowedAmount.String())
		}

		// INVARIANT 3: EscrowedAmount should not exceed MaxPayment
		if request.EscrowedAmount.GT(request.MaxPayment) {
			t.Errorf("VIOLATION: escrowed (%s) > max payment (%s)",
				request.EscrowedAmount.String(), request.MaxPayment.String())
		}

		// INVARIANT 4: Status must be valid
		if request.Status < 0 || request.Status > computetypes.REQUEST_STATUS_CANCELLED {
			t.Errorf("VIOLATION: invalid status: %d", request.Status)
		}

		// Test roundtrip
		remarshaled, err := proto.Marshal(&request)
		if err != nil {
			t.Errorf("Failed to re-marshal: %v", err)
			return
		}

		var request2 computetypes.Request
		err = proto.Unmarshal(remarshaled, &request2)
		if err != nil {
			t.Errorf("Failed to unmarshal remarshaled data: %v", err)
			return
		}

		if !request.MaxPayment.Equal(request2.MaxPayment) {
			t.Errorf("MaxPayment mismatch after roundtrip")
		}
	})
}

// FuzzOracleValidatorPriceProto tests ValidatorPrice protobuf
func FuzzOracleValidatorPriceProto(f *testing.F) {
	seeds := [][]byte{
		mustMarshal(&oracletypes.ValidatorPrice{
			ValidatorAddr: "cosmosvaloper1test",
			Asset:         "ETH",
			Price:         math.LegacyMustNewDecFromStr("3000.50"),
			BlockHeight:   1234567890,
			VotingPower:   1000,
		}),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var vp oracletypes.ValidatorPrice

		err := proto.Unmarshal(data, &vp)
		if err != nil {
			return
		}

		// INVARIANT 1: Price must be positive
		if vp.Price.IsNegative() || vp.Price.IsZero() {
			t.Errorf("VIOLATION: non-positive price: %s", vp.Price.String())
		}

		// INVARIANT 2: VotingPower must be non-negative
		if vp.VotingPower < 0 {
			t.Errorf("VIOLATION: negative voting power: %d", vp.VotingPower)
		}

		// INVARIANT 3: BlockHeight must be reasonable
		if vp.BlockHeight < 0 {
			t.Errorf("VIOLATION: negative block height: %d", vp.BlockHeight)
		}

		// Test roundtrip
		remarshaled, err := proto.Marshal(&vp)
		if err != nil {
			t.Errorf("Failed to re-marshal: %v", err)
			return
		}

		var vp2 oracletypes.ValidatorPrice
		err = proto.Unmarshal(remarshaled, &vp2)
		if err != nil {
			t.Errorf("Failed to unmarshal remarshaled data: %v", err)
			return
		}

		if !vp.Price.Equal(vp2.Price) {
			t.Errorf("Price mismatch after roundtrip")
		}
		if vp.VotingPower != vp2.VotingPower {
			t.Errorf("VotingPower mismatch after roundtrip")
		}
		if vp.BlockHeight != vp2.BlockHeight {
			t.Errorf("BlockHeight mismatch after roundtrip")
		}
	})
}

// FuzzDexParamsProto tests DEX Params protobuf
func FuzzDexParamsProto(f *testing.F) {
	seeds := [][]byte{
		mustMarshal(&dextypes.Params{
			SwapFee:            math.LegacyMustNewDecFromStr("0.003"),
			LpFee:              math.LegacyMustNewDecFromStr("0.0025"),
			ProtocolFee:        math.LegacyMustNewDecFromStr("0.0005"),
			MinLiquidity:       math.NewInt(1000),
			MaxSlippagePercent: math.LegacyMustNewDecFromStr("0.05"),
		}),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var params dextypes.Params

		err := proto.Unmarshal(data, &params)
		if err != nil {
			return
		}

		// INVARIANT 1: All fees must be between 0 and 1
		if params.SwapFee.IsNegative() || params.SwapFee.GT(math.LegacyOneDec()) {
			t.Errorf("VIOLATION: invalid swap fee: %s", params.SwapFee.String())
		}
		if params.LpFee.IsNegative() || params.LpFee.GT(math.LegacyOneDec()) {
			t.Errorf("VIOLATION: invalid LP fee: %s", params.LpFee.String())
		}
		if params.ProtocolFee.IsNegative() || params.ProtocolFee.GT(math.LegacyOneDec()) {
			t.Errorf("VIOLATION: invalid protocol fee: %s", params.ProtocolFee.String())
		}

		// INVARIANT 2: MinLiquidity must be positive
		if params.MinLiquidity.IsNegative() || params.MinLiquidity.IsZero() {
			t.Errorf("VIOLATION: invalid min liquidity: %s", params.MinLiquidity.String())
		}

		// INVARIANT 3: MaxSlippagePercent must be reasonable
		if params.MaxSlippagePercent.IsNegative() || params.MaxSlippagePercent.GT(math.LegacyOneDec()) {
			t.Errorf("VIOLATION: invalid max slippage: %s", params.MaxSlippagePercent.String())
		}

		// Test roundtrip
		remarshaled, err := proto.Marshal(&params)
		if err != nil {
			t.Errorf("Failed to re-marshal: %v", err)
			return
		}

		var params2 dextypes.Params
		err = proto.Unmarshal(remarshaled, &params2)
		if err != nil {
			t.Errorf("Failed to unmarshal remarshaled data: %v", err)
		}
	})
}

// mustMarshal is a helper that panics on marshal error
func mustMarshal(msg proto.Message) []byte {
	bz, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return bz
}

// FuzzLargeNumbers tests handling of extremely large numbers in protos
func FuzzLargeNumbers(f *testing.F) {
	seeds := []string{
		"1000000000000000000",            // 1 quintillion
		"999999999999999999999999999999", // Very large
		"1",
		"0",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, numStr string) {
		// Try to create math.Int from string
		num, ok := math.NewIntFromString(numStr)
		if !ok {
			return // Invalid number string
		}

		// Create pool with large reserves
		pool := &dextypes.Pool{
			Id:          1,
			TokenA:      "token1",
			TokenB:      "token2",
			ReserveA:    num,
			ReserveB:    num,
			TotalShares: num,
			Creator:     "cosmos1test",
		}

		// Marshal
		data, err := proto.Marshal(pool)
		if err != nil {
			t.Errorf("Failed to marshal large numbers: %v", err)
			return
		}

		// Unmarshal
		var pool2 dextypes.Pool
		err = proto.Unmarshal(data, &pool2)
		if err != nil {
			t.Errorf("Failed to unmarshal large numbers: %v", err)
			return
		}

		// Verify
		if !pool.ReserveA.Equal(pool2.ReserveA) {
			t.Errorf("Large number corrupted: original=%s, roundtrip=%s",
				pool.ReserveA.String(), pool2.ReserveA.String())
		}
	})
}
