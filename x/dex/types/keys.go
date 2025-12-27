package types

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	sharedibc "github.com/paw-chain/paw/x/shared/ibc"
)

// DefaultAuthority returns the default module authority (governance module address string)
func DefaultAuthority() string {
	return authtypes.NewModuleAddress(govtypes.ModuleName).String()
}

var (
	// ModuleNamespace is the namespace byte for the DEX module (0x02)
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

	// IBCPacketNonceKeyPrefix is the prefix for IBC packet nonce tracking (replay protection)
	IBCPacketNonceKeyPrefix = []byte{0x02, 0x16}
)

// GetPoolLPFeeKey returns the store key for LP fees for a pool and token
func GetPoolLPFeeKey(poolID uint64, token string) []byte {
	poolIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIDBytes, poolID)
	key := append(PoolLPFeeKeyPrefix, poolIDBytes...)
	key = append(key, []byte(token)...)
	return key
}

// GetProtocolFeeKey returns the store key for protocol fees for a token
func GetProtocolFeeKey(token string) []byte {
	return append(ProtocolFeeKeyPrefix, []byte(token)...)
}

// GetLiquidityShareKey returns the store key for liquidity shares
func GetLiquidityShareKey(poolID uint64, provider sdk.AccAddress) []byte {
	poolIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIDBytes, poolID)
	key := append(LiquidityShareKeyPrefix, poolIDBytes...)
	key = append(key, provider.Bytes()...)
	return key
}

// GetIBCPacketNonceKey returns the store key for IBC packet nonce tracking
// Used for replay attack prevention by tracking nonce per channel/sender pair
func GetIBCPacketNonceKey(channelID, sender string) []byte {
	return sharedibc.GetIBCPacketNonceKey(IBCPacketNonceKeyPrefix, channelID, sender)
}
