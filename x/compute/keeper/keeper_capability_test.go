package keeper

import (
	"encoding/json"
	"fmt"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/stretchr/testify/require"
)

type recordingInvariantRegistry struct {
	routes []string
}

func (r *recordingInvariantRegistry) RegisterRoute(moduleName, route string, _ sdk.Invariant) {
	r.routes = append(r.routes, fmt.Sprintf("%s:%s", moduleName, route))
}

func TestKeeper_ClaimAndGetChannelCapability(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	path := host.ChannelCapabilityPath("port-cap", "channel-1")
	cap, err := k.scopedKeeper.NewCapability(sdkCtx, path)
	require.NoError(t, err)

	err = k.ClaimCapability(sdkCtx, cap, path)
	if err != nil {
		require.ErrorIs(t, err, capabilitytypes.ErrOwnerClaimed)
	}

	stored, ok := k.GetChannelCapability(sdkCtx, "port-cap", "channel-1")
	require.True(t, ok)
	require.Equal(t, cap, stored)
}

func TestBindPortAlreadyBound(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	require.NoError(t, k.BindPort(sdkCtx))
	// Second bind should be no-op and not error
	require.NoError(t, k.BindPort(sdkCtx))
}

func TestRegisterInvariantsRegistersAllRoutes(t *testing.T) {
	k, _ := setupKeeperForTest(t)
	registry := &recordingInvariantRegistry{}

	RegisterInvariants(registry, *k)

	require.ElementsMatch(t, []string{
		"compute:escrow-balance",
		"compute:provider-stake",
		"compute:request-status",
		"compute:nonce-uniqueness",
		"compute:dispute-index",
		"compute:appeal-index",
	}, registry.routes)
}

func TestRecordProviderSlashPersistsEntry(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	provider := sdk.AccAddress([]byte("record-slash-provider"))
	amount := sdkmath.NewInt(500)

	require.NoError(t, k.recordProviderSlash(sdkCtx, provider, amount, "misbehavior"))

	store := sdkCtx.KVStore(k.storeKey)
	key := []byte(fmt.Sprintf("slash_%s_%d", provider.String(), sdkCtx.BlockHeight()))
	data := store.Get(key)
	require.NotNil(t, data)

	var slashData map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &slashData))

	require.Equal(t, provider.String(), slashData["provider"])
	require.Equal(t, amount.String(), slashData["amount"])
	require.EqualValues(t, sdkCtx.BlockHeight(), slashData["block_height"])
}
