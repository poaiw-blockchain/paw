package types

import (
	"encoding/binary"

	"cosmossdk.io/math"
)

const (
	// ModuleName defines the module name
	ModuleName = "dex"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for slashing
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_dex"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

const (
	// PoolKeyPrefix is the prefix for pool store
	PoolKeyPrefix = "Pool/value/"

	// LiquidityKeyPrefix is the prefix for liquidity provider store
	LiquidityKeyPrefix = "Liquidity/value/"

	// SwapKeyPrefix is the prefix for swap records
	SwapKeyPrefix = "Swap/value/"

	// PoolCountKeyPrefix is the prefix for pool counter
	PoolCountKeyPrefix = "Pool/count/"

	// PoolByTokensKeyPrefix is the prefix for pool lookup by tokens
	PoolByTokensKeyPrefix = "Pool/by-tokens/"
)

var (
	// PoolKey is the key prefix for pools
	PoolKey = []byte{0x01}

	// PoolCountKey is the key for the pool counter
	PoolCountKey = []byte{0x02}

	// LiquidityKey is the key prefix for liquidity
	LiquidityKey = []byte{0x03}

	// PoolByTokensKey is the key prefix for pool lookup by tokens
	PoolByTokensKey = []byte{0x04}
)

// GetPoolKey returns the store key for a pool
func GetPoolKey(poolId uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, poolId)
	return bz
}

// GetPoolByTokensKey returns the store key for pool lookup by token pair
func GetPoolByTokensKey(tokenA, tokenB string) []byte {
	return append(PoolByTokensKey, []byte(tokenA+"/"+tokenB)...)
}

// GetLiquidityKey returns the store key for liquidity shares
func GetLiquidityKey(poolId uint64, provider string) []byte {
	poolIdBz := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIdBz, poolId)
	return append(append(LiquidityKey, poolIdBz...), []byte(provider)...)
}

// NewPool creates a new Pool instance
func NewPool(id uint64, tokenA, tokenB string, reserveA, reserveB math.Int, creator string) Pool {
	// Calculate initial shares (geometric mean of reserves)
	product := reserveA.Mul(reserveB)
	sqrtProduct, _ := product.ToLegacyDec().ApproxRoot(2)
	totalShares := sqrtProduct.TruncateInt()

	return Pool{
		Id:          id,
		TokenA:      tokenA,
		TokenB:      tokenB,
		ReserveA:    reserveA,
		ReserveB:    reserveB,
		TotalShares: totalShares,
		Creator:     creator,
	}
}
