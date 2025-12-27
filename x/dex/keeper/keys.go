package keeper

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedibc "github.com/paw-chain/paw/x/shared/ibc"
)

var (
	// ModuleNamespace is the namespace byte for the DEX module (0x02)
	// All store keys are prefixed with this byte to prevent collisions with other modules
	ModuleNamespace = byte(0x02)

	// PoolKeyPrefix is the prefix for pool store keys
	PoolKeyPrefix = []byte{0x02, 0x01}

	// PoolCountKey is the key for the next pool ID counter
	PoolCountKey = []byte{0x02, 0x02}

	// PoolByTokensKeyPrefix is the prefix for indexing pools by token pair
	PoolByTokensKeyPrefix = []byte{0x02, 0x03}

	// LiquidityKeyPrefix is the prefix for liquidity position store keys
	LiquidityKeyPrefix = []byte{0x02, 0x04}

	// ParamsKey is the key for module parameters
	ParamsKey = []byte{0x02, 0x05}

	// CircuitBreakerKeyPrefix is the prefix for circuit breaker state keys
	CircuitBreakerKeyPrefix = []byte{0x02, 0x06}

	// LastLiquidityActionKeyPrefix is the prefix for tracking last liquidity action block
	LastLiquidityActionKeyPrefix = []byte{0x02, 0x07}

	// ReentrancyLockKeyPrefix is the prefix for reentrancy protection locks
	ReentrancyLockKeyPrefix = []byte{0x02, 0x08}

	// PoolLPFeeKeyPrefix is the prefix for LP fees per pool
	PoolLPFeeKeyPrefix = []byte{0x02, 0x09}

	// ProtocolFeeKeyPrefix is the prefix for protocol fees
	ProtocolFeeKeyPrefix = []byte{0x02, 0x0A}

	// LiquidityShareKeyPrefix is the prefix for liquidity shares
	LiquidityShareKeyPrefix = []byte{0x02, 0x0B}

	// RateLimitKeyPrefix is the prefix for rate limit tracking
	RateLimitKeyPrefix = []byte{0x02, 0x0C}

	// RateLimitByHeightPrefix is the prefix for indexing rate limits by block height for cleanup
	RateLimitByHeightPrefix = []byte{0x02, 0x0D}

	// PoolTWAPKeyPrefix stores pool TWAP snapshots
	PoolTWAPKeyPrefix = []byte{0x02, 0x0E}

	// ActivePoolsKeyPrefix stores pools with recent activity for TWAP updates
	ActivePoolsKeyPrefix = []byte{0x02, 0x15}

	// IBCPacketNonceKeyPrefix is the prefix for IBC packet nonce tracking (replay protection)
	// Now properly namespaced under DEX module
	IBCPacketNonceKeyPrefix = []byte{0x02, 0x16}

	// ============================================================================
	// Limit Order Key Prefixes (0x17-0x1C)
	// Defined in limit_orders.go, documented here for reference
	// ============================================================================
	// LimitOrderKeyPrefix        = []byte{0x02, 0x17} - Primary limit order storage
	// LimitOrderCountKey         = []byte{0x02, 0x18} - Next order ID counter
	// LimitOrderByOwnerPrefix    = []byte{0x02, 0x19} - Index by owner
	// LimitOrderByPoolPrefix     = []byte{0x02, 0x1A} - Index by pool
	// LimitOrderByPricePrefix    = []byte{0x02, 0x1B} - Index by price
	// LimitOrderOpenPrefix       = []byte{0x02, 0x1C} - Index of open orders

	// ============================================================================
	// Commit-Reveal Key Prefixes (0x1D-0x1F)
	// Defined in commit_reveal.go and commit_reveal_gov.go, documented here for reference
	// ============================================================================
	// SwapCommitmentKeyPrefix      = []byte{0x02, 0x1D} - Swap commitments (unified)
	// SwapCommitmentByExpiryPrefix = []byte{0x02, 0x1E} - Index by expiry
	// SwapCommitmentByTraderPrefix = []byte{0x02, 0x1F} - Index by trader
)

// PoolKey returns the store key for a pool by ID
func PoolKey(poolID uint64) []byte {
	poolIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIDBytes, poolID)
	return append(PoolKeyPrefix, poolIDBytes...)
}

// PoolByTokensKey returns the store key for indexing a pool by its token pair
func PoolByTokensKey(tokenA, tokenB string) []byte {
	// Ensure consistent ordering: tokenA < tokenB lexicographically
	if tokenA > tokenB {
		tokenA, tokenB = tokenB, tokenA
	}
	key := append(PoolByTokensKeyPrefix, []byte(tokenA)...)
	key = append(key, []byte("/")...)
	key = append(key, []byte(tokenB)...)
	return key
}

// LiquidityKey returns the store key for a liquidity position
func LiquidityKey(poolID uint64, provider sdk.AccAddress) []byte {
	poolIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIDBytes, poolID)
	key := append(LiquidityKeyPrefix, poolIDBytes...)
	key = append(key, provider.Bytes()...)
	return key
}

// LiquidityKeyByPoolPrefix returns the prefix for all liquidity positions in a pool
func LiquidityKeyByPoolPrefix(poolID uint64) []byte {
	poolIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIDBytes, poolID)
	return append(LiquidityKeyPrefix, poolIDBytes...)
}

// CircuitBreakerKey returns the store key for circuit breaker state
func CircuitBreakerKey(poolID uint64) []byte {
	poolIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIDBytes, poolID)
	return append(CircuitBreakerKeyPrefix, poolIDBytes...)
}

// LastLiquidityActionKey returns the store key for last liquidity action block
func LastLiquidityActionKey(poolID uint64, provider sdk.AccAddress) []byte {
	poolIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIDBytes, poolID)
	key := append(LastLiquidityActionKeyPrefix, poolIDBytes...)
	key = append(key, provider.Bytes()...)
	return key
}

// ReentrancyLockKey returns the store key for a reentrancy lock
func ReentrancyLockKey(lockKey string) []byte {
	return append(ReentrancyLockKeyPrefix, []byte(lockKey)...)
}

// PoolLPFeeKey returns the store key for LP fees for a pool and token
func PoolLPFeeKey(poolID uint64, token string) []byte {
	poolIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIDBytes, poolID)
	key := append(PoolLPFeeKeyPrefix, poolIDBytes...)
	key = append(key, []byte(token)...)
	return key
}

// ProtocolFeeKey returns the store key for protocol fees for a token
func ProtocolFeeKey(token string) []byte {
	return append(ProtocolFeeKeyPrefix, []byte(token)...)
}

// LiquidityShareKey returns the store key for liquidity shares
func LiquidityShareKey(poolID uint64, provider sdk.AccAddress) []byte {
	poolIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIDBytes, poolID)
	key := append(LiquidityShareKeyPrefix, poolIDBytes...)
	key = append(key, provider.Bytes()...)
	return key
}

// RateLimitKey returns the store key for rate limit tracking
func RateLimitKey(user sdk.AccAddress, window int64) []byte {
	windowBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(windowBytes, uint64(window))
	key := append(RateLimitKeyPrefix, user.Bytes()...)
	key = append(key, windowBytes...)
	return key
}

// RateLimitByHeightKey returns the index key for rate limits by height for cleanup
func RateLimitByHeightKey(height int64, user sdk.AccAddress, window int64) []byte {
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, uint64(height))
	windowBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(windowBytes, uint64(window))
	key := append(RateLimitByHeightPrefix, heightBytes...)
	key = append(key, user.Bytes()...)
	key = append(key, windowBytes...)
	return key
}

// RateLimitByHeightPrefixForHeight returns the prefix for all rate limits at a specific height
func RateLimitByHeightPrefixForHeight(height int64) []byte {
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, uint64(height))
	return append(RateLimitByHeightPrefix, heightBytes...)
}

// PoolTWAPKey returns the store key for pool TWAP data
func PoolTWAPKey(poolID uint64) []byte {
	poolIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIDBytes, poolID)
	return append(PoolTWAPKeyPrefix, poolIDBytes...)
}

// ActivePoolKey returns the store key for tracking active pools
func ActivePoolKey(poolID uint64) []byte {
	poolIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIDBytes, poolID)
	return append(ActivePoolsKeyPrefix, poolIDBytes...)
}

// IBCPacketNonceKey returns the store key for IBC packet nonce tracking
// Used for replay attack prevention by tracking nonce per channel/sender pair
func IBCPacketNonceKey(channelID, sender string) []byte {
	return sharedibc.GetIBCPacketNonceKey(IBCPacketNonceKeyPrefix, channelID, sender)
}
