package app

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// Bech32PrefixAccAddr defines the Bech32 prefix of an account's address
	Bech32PrefixAccAddr = "paw"
	// Bech32PrefixAccPub defines the Bech32 prefix of an account's public key
	Bech32PrefixAccPub = "pawpub"
	// Bech32PrefixValAddr defines the Bech32 prefix of a validator's operator address
	Bech32PrefixValAddr = "pawvaloper"
	// Bech32PrefixValPub defines the Bech32 prefix of a validator's operator public key
	Bech32PrefixValPub = "pawvaloperpub"
	// Bech32PrefixConsAddr defines the Bech32 prefix of a consensus node address
	Bech32PrefixConsAddr = "pawvalcons"
	// Bech32PrefixConsPub defines the Bech32 prefix of a consensus node public key
	Bech32PrefixConsPub = "pawvalconspub"

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
)

// SetConfig sets the configuration for the PAW network
func SetConfig() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(Bech32PrefixAccAddr, Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(Bech32PrefixValAddr, Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(Bech32PrefixConsAddr, Bech32PrefixConsPub)
	config.SetCoinType(CoinType)
	config.Seal()
}
