package compute_test

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
	"github.com/paw-chain/paw/x/compute"
	"github.com/paw-chain/paw/x/compute/types"
)

// TestAppModuleBasic_Name verifies Name() returns correct module name
func TestAppModuleBasic_Name(t *testing.T) {
	amb := compute.AppModuleBasic{}
	require.Equal(t, types.ModuleName, amb.Name())
	require.Equal(t, "compute", amb.Name())
}

// TestAppModuleBasic_RegisterLegacyAminoCodec verifies codec registration doesn't panic
func TestAppModuleBasic_RegisterLegacyAminoCodec(t *testing.T) {
	amb := compute.AppModuleBasic{}
	cdc := codec.NewLegacyAmino()

	require.NotPanics(t, func() {
		amb.RegisterLegacyAminoCodec(cdc)
	})
}

// TestAppModuleBasic_RegisterInterfaces verifies interface registration doesn't panic
func TestAppModuleBasic_RegisterInterfaces(t *testing.T) {
	amb := compute.AppModuleBasic{}
	registry := codectypes.NewInterfaceRegistry()

	require.NotPanics(t, func() {
		amb.RegisterInterfaces(registry)
	})
}

// TestAppModuleBasic_DefaultGenesis verifies DefaultGenesis returns valid JSON
func TestAppModuleBasic_DefaultGenesis(t *testing.T) {
	amb := compute.AppModuleBasic{}
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
	require.Equal(t, uint64(1), genState.NextRequestId)
	require.Equal(t, uint64(1), genState.NextDisputeId)
	require.Empty(t, genState.Providers)
	require.Empty(t, genState.Requests)
	require.NotNil(t, genState.Params)
}

// TestAppModuleBasic_ValidateGenesis_Valid verifies ValidateGenesis accepts valid states
func TestAppModuleBasic_ValidateGenesis_Valid(t *testing.T) {
	amb := compute.AppModuleBasic{}
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
	amb := compute.AppModuleBasic{}
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	tests := []struct {
		name    string
		genesis *types.GenesisState
		errMsg  string
	}{
		{
			name: "zero min provider stake",
			genesis: &types.GenesisState{
				Params: types.Params{
					MinProviderStake:           sdkmath.NewInt(0), // Invalid
					VerificationTimeoutSeconds: 60,
					MaxRequestTimeoutSeconds:   300,
					ReputationSlashPercentage:  10,
					StakeSlashPercentage:       5,
					MinReputationScore:         50,
					EscrowReleaseDelaySeconds:  3600,
				},
				GovernanceParams: types.DefaultGovernanceParams(),
				NextRequestId:    1,
				NextDisputeId:    1,
				NextSlashId:      1,
				NextAppealId:     1,
				NextEscrowNonce:  1,
			},
			errMsg: "min provider stake must be positive",
		},
		{
			name: "zero next request id",
			genesis: &types.GenesisState{
				Params:           types.DefaultParams(),
				GovernanceParams: types.DefaultGovernanceParams(),
				NextRequestId:    0, // Invalid
				NextDisputeId:    1,
				NextSlashId:      1,
				NextAppealId:     1,
				NextEscrowNonce:  1,
			},
			errMsg: "next ids must be positive",
		},
		{
			name: "zero verification timeout",
			genesis: &types.GenesisState{
				Params: types.Params{
					MinProviderStake:           sdkmath.NewInt(1000),
					VerificationTimeoutSeconds: 0, // Invalid
					MaxRequestTimeoutSeconds:   300,
					ReputationSlashPercentage:  10,
					StakeSlashPercentage:       5,
					MinReputationScore:         50,
					EscrowReleaseDelaySeconds:  3600,
				},
				GovernanceParams: types.DefaultGovernanceParams(),
				NextRequestId:    1,
				NextDisputeId:    1,
				NextSlashId:      1,
				NextAppealId:     1,
				NextEscrowNonce:  1,
			},
			errMsg: "timeouts must be non-zero",
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
	amb := compute.AppModuleBasic{}
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	err := amb.ValidateGenesis(cdc, nil, []byte("not valid json"))
	require.Error(t, err)
}

// TestAppModuleBasic_GetTxCmd verifies GetTxCmd returns non-nil command
func TestAppModuleBasic_GetTxCmd(t *testing.T) {
	amb := compute.AppModuleBasic{}
	cmd := amb.GetTxCmd()
	require.NotNil(t, cmd)
	require.Equal(t, types.ModuleName, cmd.Use)
}

// TestAppModuleBasic_GetQueryCmd verifies GetQueryCmd returns non-nil command
func TestAppModuleBasic_GetQueryCmd(t *testing.T) {
	amb := compute.AppModuleBasic{}
	cmd := amb.GetQueryCmd()
	require.NotNil(t, cmd)
	require.Equal(t, types.ModuleName, cmd.Use)
}

// TestAppModule_ModuleInterfaceCompliance verifies AppModule implements required interfaces
func TestAppModule_ModuleInterfaceCompliance(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	am := compute.NewAppModule(cdc, k, nil, nil)

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
	k, _ := keepertest.ComputeKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	am := compute.NewAppModule(cdc, k, nil, nil)

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
	k, ctx := keepertest.ComputeKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	am := compute.NewAppModule(cdc, k, nil, nil)

	// Create a genesis state with some data
	originalGenesis := types.DefaultGenesis()
	originalGenesis.NextRequestId = 5
	originalGenesis.NextDisputeId = 3
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
	require.Equal(t, originalGenesis.NextRequestId, exportedGenesis.NextRequestId)
	require.Equal(t, originalGenesis.NextDisputeId, exportedGenesis.NextDisputeId)
}

// TestAppModule_InitGenesis_MalformedJSON verifies InitGenesis handles malformed JSON
func TestAppModule_InitGenesis_MalformedJSON(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	am := compute.NewAppModule(cdc, k, nil, nil)

	err := am.InitGenesis(ctx, cdc, []byte("not valid json"))
	require.Error(t, err)
}

// TestAppModule_BeginBlock_EmptyState verifies BeginBlock works with empty state
func TestAppModule_BeginBlock_EmptyState(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	am := compute.NewAppModule(cdc, k, nil, nil)

	// BeginBlock should not panic with empty state
	err := am.BeginBlock(ctx)
	require.NoError(t, err)
}

// TestAppModule_EndBlock_EmptyState verifies EndBlock works with empty state
func TestAppModule_EndBlock_EmptyState(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	am := compute.NewAppModule(cdc, k, nil, nil)

	// EndBlock should not panic with empty state
	err := am.EndBlock(ctx)
	require.NoError(t, err)
}

// TestAppModule_ConsensusVersion verifies ConsensusVersion returns expected value
func TestAppModule_ConsensusVersion(t *testing.T) {
	k, _ := keepertest.ComputeKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	am := compute.NewAppModule(cdc, k, nil, nil)

	// ConsensusVersion should return 2 (from module.go)
	require.Equal(t, uint64(2), am.ConsensusVersion())
}

// TestAppModule_WeightedOperations verifies WeightedOperations returns operations
func TestAppModule_WeightedOperations(t *testing.T) {
	k, _ := keepertest.ComputeKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	// Need mock account and bank keepers for weighted operations
	am := compute.NewAppModule(cdc, k, nil, nil)

	simState := module.SimulationState{
		AppParams: make(map[string]json.RawMessage),
		Cdc:       cdc,
	}

	ops := am.WeightedOperations(simState)
	require.NotNil(t, ops)
}

// TestAppModule_GenerateGenesisState verifies GenerateGenesisState doesn't panic
func TestAppModule_GenerateGenesisState(t *testing.T) {
	k, _ := keepertest.ComputeKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	am := compute.NewAppModule(cdc, k, nil, nil)

	simState := &module.SimulationState{}

	require.NotPanics(t, func() {
		am.GenerateGenesisState(simState)
	})
}

// TestAppModule_RegisterStoreDecoder verifies RegisterStoreDecoder doesn't panic
func TestAppModule_RegisterStoreDecoder(t *testing.T) {
	k, _ := keepertest.ComputeKeeper(t)
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	am := compute.NewAppModule(cdc, k, nil, nil)

	require.NotPanics(t, func() {
		am.RegisterStoreDecoder(nil)
	})
}

// mockInvariantRegistry implements sdk.InvariantRegistry for testing
type mockInvariantRegistry struct {
	count int
}

func (m *mockInvariantRegistry) RegisterRoute(moduleName string, route string, invar sdk.Invariant) {
	m.count++
}
