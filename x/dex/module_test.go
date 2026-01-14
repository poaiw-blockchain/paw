package dex_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex"
	"github.com/paw-chain/paw/x/dex/types"
)

// TestAppModuleBasic_Name verifies Name() returns correct module name
func TestAppModuleBasic_Name(t *testing.T) {
	amb := dex.AppModuleBasic{}
	require.Equal(t, types.ModuleName, amb.Name())
	require.Equal(t, "dex", amb.Name())
}

// TestAppModuleBasic_RegisterLegacyAminoCodec verifies codec registration doesn't panic
func TestAppModuleBasic_RegisterLegacyAminoCodec(t *testing.T) {
	amb := dex.AppModuleBasic{}
	cdc := codec.NewLegacyAmino()

	require.NotPanics(t, func() {
		amb.RegisterLegacyAminoCodec(cdc)
	})
}

// TestAppModuleBasic_RegisterInterfaces verifies interface registration doesn't panic
func TestAppModuleBasic_RegisterInterfaces(t *testing.T) {
	amb := dex.AppModuleBasic{}
	registry := codectypes.NewInterfaceRegistry()

	require.NotPanics(t, func() {
		amb.RegisterInterfaces(registry)
	})
}

// TestAppModuleBasic_DefaultGenesis verifies DefaultGenesis returns valid JSON
func TestAppModuleBasic_DefaultGenesis(t *testing.T) {
	amb := dex.AppModuleBasic{}
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	genesisJSON := amb.DefaultGenesis(cdc)
	require.NotNil(t, genesisJSON)
	require.NotEmpty(t, genesisJSON)

	// Verify it's valid JSON
	var raw map[string]interface{}
	err := json.Unmarshal(genesisJSON, &raw)
	require.NoError(t, err)

	// Verify we can unmarshal to GenesisState
	var genState types.GenesisState
	err = cdc.UnmarshalJSON(genesisJSON, &genState)
	require.NoError(t, err)

	// Verify default values
	require.Equal(t, uint64(1), genState.NextPoolId)
	require.Empty(t, genState.Pools)
	require.NotNil(t, genState.Params)
}

// TestAppModuleBasic_ValidateGenesis_Valid verifies ValidateGenesis accepts valid states
func TestAppModuleBasic_ValidateGenesis_Valid(t *testing.T) {
	amb := dex.AppModuleBasic{}
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	// Test with default genesis
	defaultGenesis := types.DefaultGenesis()
	genesisJSON, err := cdc.MarshalJSON(defaultGenesis)
	require.NoError(t, err)

	err = amb.ValidateGenesis(cdc, nil, genesisJSON)
	require.NoError(t, err)
}

// TestAppModuleBasic_ValidateGenesis_Invalid verifies ValidateGenesis rejects invalid states
func TestAppModuleBasic_ValidateGenesis_Invalid(t *testing.T) {
	amb := dex.AppModuleBasic{}
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	tests := []struct {
		name    string
		genesis *types.GenesisState
		errMsg  string
	}{
		{
			name: "negative swap fee",
			genesis: &types.GenesisState{
				Params: types.Params{
					SwapFee:                   sdkmath.LegacyMustNewDecFromStr("-0.01"),
					LpFee:                     sdkmath.LegacyMustNewDecFromStr("0.0025"),
					ProtocolFee:               sdkmath.LegacyMustNewDecFromStr("0.0005"),
					MinLiquidity:              sdkmath.NewInt(1000),
					MaxSlippagePercent:        sdkmath.LegacyMustNewDecFromStr("0.50"),
					MaxPoolDrainPercent:       sdkmath.LegacyMustNewDecFromStr("0.30"),
					FlashLoanProtectionBlocks: 100,
					PoolCreationGas:           1000,
					SwapValidationGas:         1500,
					LiquidityGas:              1200,
					RecommendedMaxSlippage:    sdkmath.LegacyMustNewDecFromStr("0.03"),
				},
				NextPoolId: 1,
			},
			errMsg: "swap fee must be in [0,1)",
		},
		{
			name: "zero next pool id",
			genesis: &types.GenesisState{
				Params:     types.DefaultParams(),
				NextPoolId: 0,
			},
			errMsg: "next pool id must be positive",
		},
		{
			name: "zero pool creation gas",
			genesis: &types.GenesisState{
				Params: types.Params{
					SwapFee:                   sdkmath.LegacyMustNewDecFromStr("0.003"),
					LpFee:                     sdkmath.LegacyMustNewDecFromStr("0.0025"),
					ProtocolFee:               sdkmath.LegacyMustNewDecFromStr("0.0005"),
					MinLiquidity:              sdkmath.NewInt(1000),
					MaxSlippagePercent:        sdkmath.LegacyMustNewDecFromStr("0.50"),
					MaxPoolDrainPercent:       sdkmath.LegacyMustNewDecFromStr("0.30"),
					FlashLoanProtectionBlocks: 100,
					PoolCreationGas:           0, // Invalid
					SwapValidationGas:         1500,
					LiquidityGas:              1200,
					RecommendedMaxSlippage:    sdkmath.LegacyMustNewDecFromStr("0.03"),
				},
				NextPoolId: 1,
			},
			errMsg: "pool creation gas must be positive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			genesisJSON, err := cdc.MarshalJSON(tc.genesis)
			require.NoError(t, err)

			err = amb.ValidateGenesis(cdc, nil, genesisJSON)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errMsg)
		})
	}
}

// TestAppModuleBasic_ValidateGenesis_MalformedJSON verifies ValidateGenesis rejects malformed JSON
func TestAppModuleBasic_ValidateGenesis_MalformedJSON(t *testing.T) {
	amb := dex.AppModuleBasic{}
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	err := amb.ValidateGenesis(cdc, nil, []byte("not valid json"))
	require.Error(t, err)
}

// TestAppModuleBasic_GetTxCmd verifies GetTxCmd returns non-nil command
func TestAppModuleBasic_GetTxCmd(t *testing.T) {
	amb := dex.AppModuleBasic{}
	cmd := amb.GetTxCmd()
	require.NotNil(t, cmd)
	require.Equal(t, types.ModuleName, cmd.Use)
}

// TestAppModuleBasic_GetQueryCmd verifies GetQueryCmd returns non-nil command
func TestAppModuleBasic_GetQueryCmd(t *testing.T) {
	amb := dex.AppModuleBasic{}
	cmd := amb.GetQueryCmd()
	require.NotNil(t, cmd)
	require.Equal(t, types.ModuleName, cmd.Use)
}

// TestAppModule_ModuleInterfaceCompliance verifies AppModule implements required interfaces
func TestAppModule_ModuleInterfaceCompliance(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	am := dex.NewAppModule(cdc, k, nil, nil)

	// Verify interface compliance (compile-time check)
	var _ module.AppModule = am
	var _ module.AppModuleBasic = am

	// Check Name is correct
	require.Equal(t, types.ModuleName, am.Name())

	// Check ConsensusVersion
	require.Equal(t, uint64(2), am.ConsensusVersion())

	// Verify module markers
	require.NotPanics(t, func() {
		am.IsAppModule()
		am.IsOnePerModuleType()
	})

	_ = ctx // ctx used in keeper creation
}

// TestAppModule_RegisterInvariants verifies RegisterInvariants doesn't panic
func TestAppModule_RegisterInvariants(t *testing.T) {
	k, _ := keepertest.DexKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	am := dex.NewAppModule(cdc, k, nil, nil)

	// Create a mock invariant registry
	ir := &mockInvariantRegistry{}

	require.NotPanics(t, func() {
		am.RegisterInvariants(ir)
	})

	// Verify invariants were registered
	require.Greater(t, ir.count, 0, "expected at least one invariant to be registered")
}

// TestAppModule_InitExportGenesis_RoundTrip verifies InitGenesis + ExportGenesis round-trip
func TestAppModule_InitExportGenesis_RoundTrip(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	am := dex.NewAppModule(cdc, k, nil, nil)

	// Create a genesis state with some data
	originalGenesis := types.DefaultGenesis()
	originalGenesis.NextPoolId = 5
	originalGenesisJSON, err := cdc.MarshalJSON(originalGenesis)
	require.NoError(t, err)

	// InitGenesis
	err = am.InitGenesis(ctx, cdc, originalGenesisJSON)
	require.NoError(t, err)

	// ExportGenesis
	exportedGenesisJSON, err := am.ExportGenesis(ctx, cdc)
	require.NoError(t, err)
	require.NotNil(t, exportedGenesisJSON)

	// Verify exported genesis matches original
	var exportedGenesis types.GenesisState
	err = cdc.UnmarshalJSON(exportedGenesisJSON, &exportedGenesis)
	require.NoError(t, err)
	require.Equal(t, originalGenesis.NextPoolId, exportedGenesis.NextPoolId)
}

// TestAppModule_InitGenesis_MalformedJSON verifies InitGenesis handles malformed JSON
func TestAppModule_InitGenesis_MalformedJSON(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	am := dex.NewAppModule(cdc, k, nil, nil)

	err := am.InitGenesis(ctx, cdc, []byte("not valid json"))
	require.Error(t, err)
}

// TestAppModule_BeginBlock_EmptyState verifies BeginBlock works with empty state
func TestAppModule_BeginBlock_EmptyState(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	am := dex.NewAppModule(cdc, k, nil, nil)

	// BeginBlock should not panic with empty state
	err := am.BeginBlock(ctx)
	require.NoError(t, err)
}

// TestAppModule_EndBlock_EmptyState verifies EndBlock works with empty state
func TestAppModule_EndBlock_EmptyState(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	am := dex.NewAppModule(cdc, k, nil, nil)

	// EndBlock should not panic with empty state
	err := am.EndBlock(ctx)
	require.NoError(t, err)
}

// TestAppModule_BeginBlock_WithPools verifies BeginBlock works with active pools
func TestAppModule_BeginBlock_WithPools(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	am := dex.NewAppModule(cdc, k, nil, nil)

	// Create a test pool
	creator := types.TestAddr()
	_, err := k.CreatePool(ctx, creator, "upaw", "uusdt", sdkmath.NewInt(10000), sdkmath.NewInt(10000))
	require.NoError(t, err)

	// BeginBlock should handle existing pools
	err = am.BeginBlock(ctx)
	require.NoError(t, err)
}

// TestAppModule_EndBlock_WithPools verifies EndBlock works with active pools
func TestAppModule_EndBlock_WithPools(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	am := dex.NewAppModule(cdc, k, nil, nil)

	// Create a test pool
	creator := types.TestAddr()
	_, err := k.CreatePool(ctx, creator, "upaw", "uusdt", sdkmath.NewInt(10000), sdkmath.NewInt(10000))
	require.NoError(t, err)

	// EndBlock should handle existing pools
	err = am.EndBlock(ctx)
	require.NoError(t, err)
}

// TestAppModule_ConsensusVersion verifies ConsensusVersion returns expected value
func TestAppModule_ConsensusVersion(t *testing.T) {
	k, _ := keepertest.DexKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	am := dex.NewAppModule(cdc, k, nil, nil)

	// ConsensusVersion should return 2 (from module.go)
	require.Equal(t, uint64(2), am.ConsensusVersion())
}

// TestAppModule_WeightedOperations verifies WeightedOperations returns operations
func TestAppModule_WeightedOperations(t *testing.T) {
	k, _ := keepertest.DexKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	// Need mock account and bank keepers for weighted operations
	am := dex.NewAppModule(cdc, k, nil, nil)

	simState := module.SimulationState{
		AppParams: make(map[string]json.RawMessage),
		Cdc:       cdc,
	}

	ops := am.WeightedOperations(simState)
	require.NotNil(t, ops)
}

// TestAppModule_GenerateGenesisState verifies GenerateGenesisState doesn't panic
func TestAppModule_GenerateGenesisState(t *testing.T) {
	k, _ := keepertest.DexKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	am := dex.NewAppModule(cdc, k, nil, nil)

	simState := &module.SimulationState{}

	require.NotPanics(t, func() {
		am.GenerateGenesisState(simState)
	})
}

// mockInvariantRegistry implements sdk.InvariantRegistry for testing
type mockInvariantRegistry struct {
	count int
}

func (m *mockInvariantRegistry) RegisterRoute(moduleName string, route string, invar sdk.Invariant) {
	m.count++
}
