package keeper_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"unsafe"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle"
	oraclekeeper "github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

func TestOracleOnChanCloseConfirmCleansPendingOperations(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	channelID := "channel-1"
	sequence := uint64(21)
	chainID := "osmosis-1"

	store := getOracleStoreKey(t, k)
	kv := ctx.KVStore(store)

	source := oraclekeeper.CrossChainOracleSource{
		ChainID:           chainID,
		OracleType:        "band",
		ConnectionID:      "connection-0",
		ChannelID:         channelID,
		Reputation:        sdkmath.LegacyOneDec(),
		LastHeartbeat:     ctx.BlockTime(),
		TotalQueries:      10,
		SuccessfulQueries: 9,
		Active:            true,
	}
	bz, err := json.Marshal(source)
	require.NoError(t, err)
	kv.Set([]byte(fmt.Sprintf("oracle_source_%s", chainID)), bz)

	oraclekeeper.TrackPendingOperationForTest(k, ctx, channelID, chainID, oraclekeeper.PacketTypeSubscribePrices, sequence)
	require.Len(t, k.GetPendingOperations(ctx, channelID), 1)

	ibcModule := oracle.NewIBCModule(*k, nil)
	require.NoError(t, ibcModule.OnChanCloseConfirm(ctx, types.PortID, channelID))

	require.Len(t, k.GetPendingOperations(ctx, channelID), 0)

	foundCleanup := false
	foundPenalty := false
	foundClose := false
	for _, evt := range ctx.EventManager().Events() {
		switch evt.Type {
		case "oracle_channel_cleanup":
			foundCleanup = true
		case "oracle_source_penalized":
			foundPenalty = true
		case types.EventTypeChannelClose:
			foundClose = true
		}
	}
	require.True(t, foundCleanup, "expected channel cleanup event")
	require.True(t, foundPenalty, "expected oracle source penalized event")
	require.True(t, foundClose, "expected channel close event")
}

func getOracleStoreKey(t *testing.T, k *oraclekeeper.Keeper) storetypes.StoreKey {
	t.Helper()
	field := reflect.ValueOf(k).Elem().FieldByName("storeKey")
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(storetypes.StoreKey)
}
