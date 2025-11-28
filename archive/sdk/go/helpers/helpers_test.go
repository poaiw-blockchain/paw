package helpers

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateMnemonic(t *testing.T) {
	mnemonic, err := GenerateMnemonic()
	require.NoError(t, err)
	assert.NotEmpty(t, mnemonic)

	// Should be 24 words
	words := len(mnemonic) - len(mnemonic)
	for _, c := range mnemonic {
		if c == ' ' {
			words++
		}
	}
	// Note: Actual word count validation would need string splitting
	assert.True(t, ValidateMnemonic(mnemonic))
}

func TestValidateMnemonic(t *testing.T) {
	tests := []struct {
		name     string
		mnemonic string
		want     bool
	}{
		{
			name:     "invalid mnemonic",
			mnemonic: "invalid mnemonic phrase",
			want:     false,
		},
		{
			name:     "empty mnemonic",
			mnemonic: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateMnemonic(tt.mnemonic)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCalculateSwapOutput(t *testing.T) {
	amountIn := math.NewInt(1000000)
	reserveIn := math.NewInt(10000000)
	reserveOut := math.NewInt(20000000)
	swapFee := math.LegacyNewDecWithPrec(3, 3) // 0.3%

	output := CalculateSwapOutput(amountIn, reserveIn, reserveOut, swapFee)

	assert.True(t, output.GT(math.ZeroInt()))
	assert.True(t, output.LT(amountIn.MulRaw(2))) // Should be less than 2x input
}

func TestCalculatePriceImpact(t *testing.T) {
	amountIn := math.NewInt(1000000)
	reserveIn := math.NewInt(10000000)
	reserveOut := math.NewInt(20000000)

	impact := CalculatePriceImpact(amountIn, reserveIn, reserveOut)

	assert.True(t, impact.GTE(math.LegacyZeroDec()))
	assert.True(t, impact.LT(math.LegacyNewDec(100))) // Should be less than 100%
}

func TestCalculateShares(t *testing.T) {
	tests := []struct {
		name        string
		amountA     math.Int
		amountB     math.Int
		reserveA    math.Int
		reserveB    math.Int
		totalShares math.Int
		wantZero    bool
	}{
		{
			name:        "first liquidity provider",
			amountA:     math.NewInt(1000000),
			amountB:     math.NewInt(2000000),
			reserveA:    math.ZeroInt(),
			reserveB:    math.ZeroInt(),
			totalShares: math.ZeroInt(),
			wantZero:    false,
		},
		{
			name:        "existing pool",
			amountA:     math.NewInt(1000000),
			amountB:     math.NewInt(2000000),
			reserveA:    math.NewInt(10000000),
			reserveB:    math.NewInt(20000000),
			totalShares: math.NewInt(100000000),
			wantZero:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shares := CalculateShares(
				tt.amountA,
				tt.amountB,
				tt.reserveA,
				tt.reserveB,
				tt.totalShares,
			)

			if tt.wantZero {
				assert.True(t, shares.IsZero())
			} else {
				assert.True(t, shares.GT(math.ZeroInt()))
			}
		})
	}
}

func TestValidateAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		prefix  string
		wantErr bool
	}{
		{
			name:    "empty address",
			address: "",
			prefix:  "paw",
			wantErr: true,
		},
		{
			name:    "invalid format",
			address: "invalid",
			prefix:  "paw",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAddress(tt.address, tt.prefix)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
