package tests

import (
	"bytes"
	"testing"

	computekeeper "github.com/paw-chain/paw/x/compute/keeper"
	computetypes "github.com/paw-chain/paw/x/compute/types"
	dexkeeper "github.com/paw-chain/paw/x/dex/keeper"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oraclekeeper "github.com/paw-chain/paw/x/oracle/keeper"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"

	"github.com/stretchr/testify/require"
)

// TestModuleNamespaceUniqueness verifies that each module has a unique namespace byte
func TestModuleNamespaceUniqueness(t *testing.T) {
	namespaces := map[byte]string{
		computetypes.ModuleNamespace: "compute",
		dextypes.ModuleNamespace:     "dex",
		oracletypes.ModuleNamespace:  "oracle",
	}

	// Verify all namespaces are different
	require.Equal(t, 3, len(namespaces), "Each module must have a unique namespace")
	require.Equal(t, byte(0x01), computetypes.ModuleNamespace, "Compute module namespace must be 0x01")
	require.Equal(t, byte(0x02), dextypes.ModuleNamespace, "DEX module namespace must be 0x02")
	require.Equal(t, byte(0x03), oracletypes.ModuleNamespace, "Oracle module namespace must be 0x03")
}

// TestComputeKeysHaveNamespace verifies all compute module keys are properly namespaced
func TestComputeKeysHaveNamespace(t *testing.T) {
	keys := [][]byte{
		computekeeper.ParamsKey,
		computekeeper.ProviderKeyPrefix,
		computekeeper.RequestKeyPrefix,
		computekeeper.ResultKeyPrefix,
		computekeeper.NextRequestIDKey,
		computekeeper.RequestsByRequesterPrefix,
		computekeeper.RequestsByProviderPrefix,
		computekeeper.RequestsByStatusPrefix,
		computekeeper.ActiveProvidersPrefix,
		computekeeper.NonceKeyPrefix,
		computekeeper.GovernanceParamsKey,
		computekeeper.DisputeKeyPrefix,
		computekeeper.EvidenceKeyPrefix,
		computekeeper.SlashRecordKeyPrefix,
		computekeeper.AppealKeyPrefix,
		computekeeper.NextDisputeIDKey,
		computekeeper.NextSlashIDKey,
		computekeeper.NextAppealIDKey,
		computekeeper.DisputesByRequestPrefix,
		computekeeper.DisputesByStatusPrefix,
		computekeeper.SlashRecordsByProviderPrefix,
		computekeeper.AppealsByStatusPrefix,
		computekeeper.CircuitParamsKeyPrefix,
		computekeeper.ZKMetricsKey,
		computekeeper.VerificationProofHashPrefix,
		computekeeper.ProviderSigningKeyPrefix,
		computekeeper.RequestFinalizedPrefix,
		computekeeper.ProviderStatsKeyPrefix,
		computekeeper.EscrowTimeoutReversePrefix,
		computekeeper.NonceByHeightPrefix,
		computekeeper.ProvidersByReputationPrefix,
		computekeeper.CatastrophicFailureKeyPrefix,
		computekeeper.NextCatastrophicFailureIDKey,
		computekeeper.IBCPacketKeyPrefix,
		computekeeper.ProviderCacheKeyPrefix,
		computekeeper.ProviderCacheMetadataKey,
		computekeeper.IBCPacketNonceKeyPrefix,
	}

	for i, key := range keys {
		require.GreaterOrEqual(t, len(key), 2, "Key %d must have at least 2 bytes (namespace + prefix)", i)
		require.Equal(t, byte(0x01), key[0], "Key %d must start with compute namespace (0x01), got 0x%02x", i, key[0])
	}
}

// TestDEXKeysHaveNamespace verifies all DEX module keys are properly namespaced
func TestDEXKeysHaveNamespace(t *testing.T) {
	keys := [][]byte{
		dexkeeper.PoolKeyPrefix,
		dexkeeper.PoolCountKey,
		dexkeeper.PoolByTokensKeyPrefix,
		dexkeeper.LiquidityKeyPrefix,
		dexkeeper.ParamsKey,
		dexkeeper.CircuitBreakerKeyPrefix,
		dexkeeper.LastLiquidityActionKeyPrefix,
		dexkeeper.ReentrancyLockKeyPrefix,
		dexkeeper.PoolLPFeeKeyPrefix,
		dexkeeper.ProtocolFeeKeyPrefix,
		dexkeeper.LiquidityShareKeyPrefix,
		dexkeeper.RateLimitKeyPrefix,
		dexkeeper.RateLimitByHeightPrefix,
		dexkeeper.PoolTWAPKeyPrefix,
		dexkeeper.ActivePoolsKeyPrefix,
		dexkeeper.IBCPacketNonceKeyPrefix,
	}

	for i, key := range keys {
		require.GreaterOrEqual(t, len(key), 2, "Key %d must have at least 2 bytes (namespace + prefix)", i)
		require.Equal(t, byte(0x02), key[0], "Key %d must start with DEX namespace (0x02), got 0x%02x", i, key[0])
	}
}

// TestOracleKeysHaveNamespace verifies all Oracle module keys are properly namespaced
func TestOracleKeysHaveNamespace(t *testing.T) {
	keys := [][]byte{
		oraclekeeper.ParamsKey,
		oraclekeeper.PriceKeyPrefix,
		oraclekeeper.ValidatorPriceKeyPrefix,
		oraclekeeper.ValidatorOracleKeyPrefix,
		oraclekeeper.PriceSnapshotKeyPrefix,
		oraclekeeper.FeederDelegationKeyPrefix,
		oraclekeeper.SubmissionByHeightPrefix,
		oraclekeeper.ValidatorAccuracyKeyPrefix,
		oraclekeeper.AccuracyBonusPoolKey,
		oraclekeeper.GeographicInfoKeyPrefix,
		oraclekeeper.OutlierHistoryKeyPrefix,
		oraclekeeper.IBCPacketNonceKeyPrefix,
	}

	for i, key := range keys {
		require.GreaterOrEqual(t, len(key), 2, "Key %d must have at least 2 bytes (namespace + prefix)", i)
		require.Equal(t, byte(0x03), key[0], "Key %d must start with Oracle namespace (0x03), got 0x%02x", i, key[0])
	}
}

// TestNoKeyCollisionsAcrossModules verifies that no keys collide across modules
func TestNoKeyCollisionsAcrossModules(t *testing.T) {
	computeKeys := [][]byte{
		computekeeper.ParamsKey,
		computekeeper.ProviderKeyPrefix,
		computekeeper.RequestKeyPrefix,
		computekeeper.ResultKeyPrefix,
		computekeeper.IBCPacketNonceKeyPrefix,
	}

	dexKeys := [][]byte{
		dexkeeper.PoolKeyPrefix,
		dexkeeper.PoolCountKey,
		dexkeeper.ParamsKey,
		dexkeeper.IBCPacketNonceKeyPrefix,
	}

	oracleKeys := [][]byte{
		oraclekeeper.ParamsKey,
		oraclekeeper.PriceKeyPrefix,
		oraclekeeper.ValidatorPriceKeyPrefix,
		oraclekeeper.IBCPacketNonceKeyPrefix,
	}

	// Check compute vs dex
	for _, ck := range computeKeys {
		for _, dk := range dexKeys {
			require.False(t, bytes.Equal(ck, dk), "Compute key %x collides with DEX key %x", ck, dk)
		}
	}

	// Check compute vs oracle
	for _, ck := range computeKeys {
		for _, ok := range oracleKeys {
			require.False(t, bytes.Equal(ck, ok), "Compute key %x collides with Oracle key %x", ck, ok)
		}
	}

	// Check dex vs oracle
	for _, dk := range dexKeys {
		for _, ok := range oracleKeys {
			require.False(t, bytes.Equal(dk, ok), "DEX key %x collides with Oracle key %x", dk, ok)
		}
	}
}

// TestIBCPacketNonceKeyUniqueness specifically tests the IBCPacketNonceKeyPrefix
// which was previously 0x0D in all three modules - now should be unique
func TestIBCPacketNonceKeyUniqueness(t *testing.T) {
	computeIBCKey := computekeeper.IBCPacketNonceKeyPrefix
	dexIBCKey := dexkeeper.IBCPacketNonceKeyPrefix
	oracleIBCKey := oraclekeeper.IBCPacketNonceKeyPrefix

	// All three keys must be different
	require.False(t, bytes.Equal(computeIBCKey, dexIBCKey), "Compute and DEX IBC nonce keys must differ")
	require.False(t, bytes.Equal(computeIBCKey, oracleIBCKey), "Compute and Oracle IBC nonce keys must differ")
	require.False(t, bytes.Equal(dexIBCKey, oracleIBCKey), "DEX and Oracle IBC nonce keys must differ")

	// Verify correct namespacing
	require.Equal(t, []byte{0x01, 0x28}, computeIBCKey, "Compute IBC nonce key must be [0x01, 0x28]")
	require.Equal(t, []byte{0x02, 0x16}, dexIBCKey, "DEX IBC nonce key must be [0x02, 0x16]")
	require.Equal(t, []byte{0x03, 0x0D}, oracleIBCKey, "Oracle IBC nonce key must be [0x03, 0x0D]")
}

// TestParamsKeyUniqueness verifies ParamsKey is unique across all modules
func TestParamsKeyUniqueness(t *testing.T) {
	computeParams := computekeeper.ParamsKey
	dexParams := dexkeeper.ParamsKey
	oracleParams := oraclekeeper.ParamsKey

	// All three keys must be different
	require.False(t, bytes.Equal(computeParams, dexParams), "Compute and DEX params keys must differ")
	require.False(t, bytes.Equal(computeParams, oracleParams), "Compute and Oracle params keys must differ")
	require.False(t, bytes.Equal(dexParams, oracleParams), "DEX and Oracle params keys must differ")

	// Verify correct namespacing
	require.Equal(t, []byte{0x01, 0x01}, computeParams, "Compute params key must be [0x01, 0x01]")
	require.Equal(t, []byte{0x02, 0x05}, dexParams, "DEX params key must be [0x02, 0x05]")
	require.Equal(t, []byte{0x03, 0x01}, oracleParams, "Oracle params key must be [0x03, 0x01]")
}

// TestMigrationHelpers tests the migration helper functions
func TestMigrationHelpers(t *testing.T) {
	// Test Compute module helpers
	// Old key without namespace: just the sub-prefix
	oldKey := []byte{0x23}
	newKey := computekeeper.GetNewKey(oldKey)
	require.Equal(t, []byte{0x01, 0x23}, newKey, "GetNewKey should prepend namespace")

	// Namespaced key
	namespacedKey := []byte{0x01, 0x23}
	recoveredOldKey := computekeeper.GetOldKey(namespacedKey)
	require.Equal(t, []byte{0x23}, recoveredOldKey, "GetOldKey should strip namespace")

	// Test idempotency - calling GetNewKey on already namespaced key shouldn't double-namespace
	alreadyNamespaced := []byte{0x01, 0x23}
	stillNamespaced := computekeeper.GetNewKey(alreadyNamespaced)
	require.Equal(t, []byte{0x01, 0x23}, stillNamespaced, "GetNewKey should be idempotent")

	// Test DEX module helpers
	oldKey = []byte{0x15}
	newKey = dexkeeper.GetNewKey(oldKey)
	require.Equal(t, []byte{0x02, 0x15}, newKey, "DEX GetNewKey should prepend namespace")

	namespacedKey = []byte{0x02, 0x15}
	recoveredOldKey = dexkeeper.GetOldKey(namespacedKey)
	require.Equal(t, []byte{0x15}, recoveredOldKey, "DEX GetOldKey should strip namespace")

	// Test Oracle module helpers
	oldKey = []byte{0x0A}
	newKey = oraclekeeper.GetNewKey(oldKey)
	require.Equal(t, []byte{0x03, 0x0A}, newKey, "Oracle GetNewKey should prepend namespace")

	namespacedKey = []byte{0x03, 0x0A}
	recoveredOldKey = oraclekeeper.GetOldKey(namespacedKey)
	require.Equal(t, []byte{0x0A}, recoveredOldKey, "Oracle GetOldKey should strip namespace")
}

// TestKeyPrefixStructure verifies the two-byte structure of all keys
func TestKeyPrefixStructure(t *testing.T) {
	testCases := []struct {
		name      string
		key       []byte
		namespace byte
		subPrefix byte
	}{
		{"Compute ParamsKey", computekeeper.ParamsKey, 0x01, 0x01},
		{"Compute ProviderKeyPrefix", computekeeper.ProviderKeyPrefix, 0x01, 0x02},
		{"DEX PoolKeyPrefix", dexkeeper.PoolKeyPrefix, 0x02, 0x01},
		{"DEX ParamsKey", dexkeeper.ParamsKey, 0x02, 0x05},
		{"Oracle ParamsKey", oraclekeeper.ParamsKey, 0x03, 0x01},
		{"Oracle PriceKeyPrefix", oraclekeeper.PriceKeyPrefix, 0x03, 0x02},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.GreaterOrEqual(t, len(tc.key), 2, "Key must have at least 2 bytes")
			require.Equal(t, tc.namespace, tc.key[0], "First byte must be module namespace")
			require.Equal(t, tc.subPrefix, tc.key[1], "Second byte must be sub-prefix")
		})
	}
}
