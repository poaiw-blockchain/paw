package types

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
)
