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

	// ParamsKey is the key for module parameters
	ParamsKey = []byte{0x05}

	// PausedKey is the key for module paused state
	PausedKey = []byte{0x06}

	// PriceObservationsKey is the key prefix for TWAP price observations
	PriceObservationsKey = []byte{0x07}

	// FlashLoanKey is the key prefix for flash loan tracking
	FlashLoanKey = []byte{0x08}

	// SwapCountKey is the key prefix for swap count tracking
	SwapCountKey = []byte{0x09}

	// LargeSwapCountKey is the key prefix for large swap count tracking
	LargeSwapCountKey = []byte{0x0A}

	// CircuitBreakerConfigKey is the key for circuit breaker configuration
	CircuitBreakerConfigKey = []byte{0x0B}

	// CircuitBreakerStateKeyPrefix is the key prefix for circuit breaker states
	CircuitBreakerStateKeyPrefix = []byte{0x0C}

	// MEVProtectionConfigKey is the key for MEV protection configuration
	MEVProtectionConfigKey = []byte{0x0D}

	// MEVMetricsKey is the key for MEV protection metrics
	MEVMetricsKey = []byte{0x0E}

	// TransactionRecordKeyPrefix is the key prefix for transaction records
	TransactionRecordKeyPrefix = []byte{0x0F}

	// RecentTransactionKeyPrefix is the key prefix for recent transactions cache
	RecentTransactionKeyPrefix = []byte{0x10}

	// LastTransactionTimestampKeyPrefix is the key prefix for last transaction timestamps
	LastTransactionTimestampKeyPrefix = []byte{0x11}

	// SandwichPatternKeyPrefix is the key prefix for sandwich attack patterns
	SandwichPatternKeyPrefix = []byte{0x12}
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

// GetPriceObservationsKey returns the store key for price observations
func GetPriceObservationsKey(poolId uint64) []byte {
	poolIdBz := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIdBz, poolId)
	return append(PriceObservationsKey, poolIdBz...)
}

// GetFlashLoanKey returns the store key for flash loan tracking
func GetFlashLoanKey(address string, poolId uint64) []byte {
	poolIdBz := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIdBz, poolId)
	return append(append(FlashLoanKey, []byte(address)...), poolIdBz...)
}

// GetSwapCountKey returns the store key for swap count tracking
func GetSwapCountKey(address string, poolId uint64, blockHeight int64) []byte {
	poolIdBz := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIdBz, poolId)
	blockBz := make([]byte, 8)
	binary.BigEndian.PutUint64(blockBz, uint64(blockHeight))
	return append(append(append(SwapCountKey, []byte(address)...), poolIdBz...), blockBz...)
}

// GetLargeSwapCountKey returns the store key for large swap count tracking
func GetLargeSwapCountKey(address string, blockHeight int64) []byte {
	blockBz := make([]byte, 8)
	binary.BigEndian.PutUint64(blockBz, uint64(blockHeight))
	return append(append(LargeSwapCountKey, []byte(address)...), blockBz...)
}

// GetCircuitBreakerStateKey returns the store key for circuit breaker state
func GetCircuitBreakerStateKey(poolId uint64) []byte {
	poolIdBz := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIdBz, poolId)
	return append(CircuitBreakerStateKeyPrefix, poolIdBz...)
}

// GetTransactionRecordKey returns the store key for a transaction record
func GetTransactionRecordKey(poolId uint64, blockHeight int64, txHash string) []byte {
	poolIdBz := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIdBz, poolId)
	blockBz := make([]byte, 8)
	binary.BigEndian.PutUint64(blockBz, uint64(blockHeight))
	return append(append(append(TransactionRecordKeyPrefix, poolIdBz...), blockBz...), []byte(txHash)...)
}

// GetPoolTransactionPrefix returns the prefix for all transactions of a pool
func GetPoolTransactionPrefix(poolId uint64) []byte {
	poolIdBz := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIdBz, poolId)
	return append(TransactionRecordKeyPrefix, poolIdBz...)
}

// GetRecentTransactionKey returns the store key for a recent transaction
func GetRecentTransactionKey(poolId uint64, blockHeight int64, txHash string) []byte {
	poolIdBz := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIdBz, poolId)
	blockBz := make([]byte, 8)
	binary.BigEndian.PutUint64(blockBz, uint64(blockHeight))
	return append(append(append(RecentTransactionKeyPrefix, poolIdBz...), blockBz...), []byte(txHash)...)
}

// GetLastTransactionTimestampKey returns the store key for last transaction timestamp
func GetLastTransactionTimestampKey(poolId uint64) []byte {
	poolIdBz := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIdBz, poolId)
	return append(LastTransactionTimestampKeyPrefix, poolIdBz...)
}

// GetSandwichPatternKey returns the store key for a sandwich pattern
func GetSandwichPatternKey(poolId uint64, blockHeight int64, txHash string) []byte {
	poolIdBz := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIdBz, poolId)
	blockBz := make([]byte, 8)
	binary.BigEndian.PutUint64(blockBz, uint64(blockHeight))
	return append(append(append(SandwichPatternKeyPrefix, poolIdBz...), blockBz...), []byte(txHash)...)
}

// GetSandwichPatternPrefix returns the prefix for all sandwich patterns of a pool
func GetSandwichPatternPrefix(poolId uint64) []byte {
	poolIdBz := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIdBz, poolId)
	return append(SandwichPatternKeyPrefix, poolIdBz...)
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
