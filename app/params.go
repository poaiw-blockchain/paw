package app

import (
	"sync"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// Bech32PrefixAccAddr defines the Bech32 prefix of an account's address
	Bech32PrefixAccAddr = "pawtest"
	// Bech32PrefixAccPub defines the Bech32 prefix of an account's public key
	Bech32PrefixAccPub = "pawtestpub"
	// Bech32PrefixValAddr defines the Bech32 prefix of a validator's operator address
	Bech32PrefixValAddr = "pawtestvaloper"
	// Bech32PrefixValPub defines the Bech32 prefix of a validator's operator public key
	Bech32PrefixValPub = "pawtestvaloperpub"
	// Bech32PrefixConsAddr defines the Bech32 prefix of a consensus node address
	Bech32PrefixConsAddr = "pawtestvalcons"
	// Bech32PrefixConsPub defines the Bech32 prefix of a consensus node public key
	Bech32PrefixConsPub = "pawtestvalconspub"

	// CoinType is the PAW coin type as defined in SLIP44 (https://github.com/satoshilabs/slips/blob/master/slip-0044.md)
	CoinType = 118

	// BondDenom defines the native staking token denomination.
	BondDenom = "upaw"

	// DisplayDenom defines the name, symbol, and display value of the PAW token.
	DisplayDenom = "PAW"

	// DefaultGasPrice is the default gas price in upaw
	DefaultGasPrice = "0.001"
)

var (
	// DefaultMinGasPrice is the minimum gas price
	DefaultMinGasPrice = sdk.NewDecCoinFromDec(BondDenom, math.LegacyNewDecWithPrec(1, 3)) // 0.001upaw
	setConfigOnce      sync.Once
)

// SetConfig sets the configuration for the PAW network
// This function is idempotent and can be called multiple times safely
func SetConfig() {
	setConfigOnce.Do(func() {
		config := sdk.GetConfig()

		// Only set if not already configured with pawtest prefix
		if config.GetBech32AccountAddrPrefix() == Bech32PrefixAccAddr {
			return
		}

		config.SetBech32PrefixForAccount(Bech32PrefixAccAddr, Bech32PrefixAccPub)
		config.SetBech32PrefixForValidator(Bech32PrefixValAddr, Bech32PrefixValPub)
		config.SetBech32PrefixForConsensusNode(Bech32PrefixConsAddr, Bech32PrefixConsPub)
		config.SetCoinType(CoinType)
		config.Seal()
	})
}
