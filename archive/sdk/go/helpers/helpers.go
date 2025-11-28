package helpers

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/go-bip39"
)

// GenerateMnemonic generates a new 24-word BIP39 mnemonic
func GenerateMnemonic() (string, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", fmt.Errorf("failed to generate entropy: %w", err)
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("failed to generate mnemonic: %w", err)
	}

	return mnemonic, nil
}

// ValidateMnemonic validates a BIP39 mnemonic
func ValidateMnemonic(mnemonic string) bool {
	return bip39.IsMnemonicValid(mnemonic)
}

// FormatCoin formats a coin for display
func FormatCoin(coin sdk.Coin, decimals int) string {
	amount := coin.Amount.BigInt()
	divisor := math.NewInt(1)
	for i := 0; i < decimals; i++ {
		divisor = divisor.MulRaw(10)
	}

	quotient := math.NewIntFromBigInt(amount).Quo(divisor)
	remainder := math.NewIntFromBigInt(amount).Mod(divisor)

	return fmt.Sprintf("%s.%0*s %s",
		quotient.String(),
		decimals,
		remainder.String(),
		coin.Denom,
	)
}

// ParseCoin parses a coin string with decimals
func ParseCoin(coinStr string, denom string, decimals int) (sdk.Coin, error) {
	// Simple implementation - parse amount and convert to base units
	var amountStr string
	_, err := fmt.Sscanf(coinStr, "%s", &amountStr)
	if err != nil {
		return sdk.Coin{}, fmt.Errorf("failed to parse coin: %w", err)
	}

	amount, ok := math.NewIntFromString(amountStr)
	if !ok {
		return sdk.Coin{}, fmt.Errorf("failed to parse amount: %s", amountStr)
	}

	multiplier := math.NewInt(1)
	for i := 0; i < decimals; i++ {
		multiplier = multiplier.MulRaw(10)
	}

	baseAmount := amount.Mul(multiplier)
	return sdk.NewCoin(denom, baseAmount), nil
}

// CalculateSwapOutput calculates the output amount for a swap
func CalculateSwapOutput(amountIn, reserveIn, reserveOut math.Int, swapFee math.LegacyDec) math.Int {
	// Apply fee
	feeMultiplier := math.LegacyOneDec().Sub(swapFee)
	amountInWithFee := math.LegacyNewDecFromInt(amountIn).Mul(feeMultiplier).TruncateInt()

	// Constant product formula: amountOut = (amountInWithFee * reserveOut) / (reserveIn + amountInWithFee)
	numerator := amountInWithFee.Mul(reserveOut)
	denominator := reserveIn.Add(amountInWithFee)

	return numerator.Quo(denominator)
}

// CalculatePriceImpact calculates the price impact of a swap
func CalculatePriceImpact(amountIn, reserveIn, reserveOut math.Int) math.LegacyDec {
	amountOut := CalculateSwapOutput(amountIn, reserveIn, reserveOut, math.LegacyZeroDec())

	priceBefore := math.LegacyNewDecFromInt(reserveOut).Quo(math.LegacyNewDecFromInt(reserveIn))
	priceAfter := math.LegacyNewDecFromInt(amountOut).Quo(math.LegacyNewDecFromInt(amountIn))

	impact := priceBefore.Sub(priceAfter).Quo(priceBefore).Abs()
	return impact.Mul(math.LegacyNewDec(100)) // Convert to percentage
}

// CalculateShares calculates LP shares for liquidity addition
func CalculateShares(amountA, amountB, reserveA, reserveB, totalShares math.Int) math.Int {
	if totalShares.IsZero() {
		// First liquidity provider
		return amountA.Mul(amountB)
	}

	// Proportional to existing pool
	return amountA.Mul(totalShares).Quo(reserveA)
}

// ValidateAddress validates a bech32 address
func ValidateAddress(address, prefix string) error {
	_, err := sdk.GetFromBech32(address, prefix)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	return nil
}

// ConvertAddress converts an address from one prefix to another
func ConvertAddress(address, fromPrefix, toPrefix string) (string, error) {
	bz, err := sdk.GetFromBech32(address, fromPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to decode address: %w", err)
	}

	converted, err := sdk.Bech32ifyAddressBytes(toPrefix, bz)
	if err != nil {
		return "", fmt.Errorf("failed to encode address: %w", err)
	}

	return converted, nil
}
