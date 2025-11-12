package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the module name
	ModuleName = "dex"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName
)

// Store key prefixes
var (
	PoolKey         = []byte{0x01} // prefix for pool store
	PoolCountKey    = []byte{0x02} // key for pool count
	LiquidityKey    = []byte{0x03} // prefix for liquidity provider shares
	PoolByTokensKey = []byte{0x04} // prefix for pool lookup by token pair
)

// GetPoolKey returns the store key for a pool
func GetPoolKey(poolId uint64) []byte {
	return append(PoolKey, sdk.Uint64ToBigEndian(poolId)...)
}

// GetLiquidityKey returns the store key for liquidity provider shares
func GetLiquidityKey(poolId uint64, provider string) []byte {
	key := append(LiquidityKey, sdk.Uint64ToBigEndian(poolId)...)
	return append(key, []byte(provider)...)
}

// GetPoolByTokensKey returns the store key for pool lookup by token pair
func GetPoolByTokensKey(tokenA, tokenB string) []byte {
	// Ensure consistent ordering
	if tokenA > tokenB {
		tokenA, tokenB = tokenB, tokenA
	}
	key := append(PoolByTokensKey, []byte(tokenA)...)
	key = append(key, []byte("/")...)
	return append(key, []byte(tokenB)...)
}
