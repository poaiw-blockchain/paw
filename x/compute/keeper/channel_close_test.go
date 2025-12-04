package keeper_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"unsafe"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	accountkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute"
	keeperpkg "github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

func TestComputeOnChanCloseConfirmRefundsEscrow(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	channelID := "channel-close-1"
	sequence := uint64(77)
	jobID := "close-job-1"
	requester := sdk.AccAddress(bytes.Repeat([]byte{0x44}, 20))
	provider := sdk.AccAddress(bytes.Repeat([]byte{0x55}, 20))
	amount := sdkmath.NewInt(3_000_000)

	store := getComputeStoreKey(t, k)
	kv := ctx.KVStore(store)

	job := keeperpkg.CrossChainComputeJob{
		JobID:        jobID,
		Requester:    requester.String(),
		Provider:     provider.String(),
		Status:       "pending",
		EscrowAmount: amount,
		SubmittedAt:  ctx.BlockTime(),
	}
	setJSON(t, kv, []byte(fmt.Sprintf("job_%s", jobID)), job)

	escrow := keeperpkg.CrossChainEscrow{
		JobID:     jobID,
		Requester: requester.String(),
		Provider:  provider.String(),
		Amount:    amount,
		Status:    "locked",
		LockedAt:  ctx.BlockTime(),
	}
	setJSON(t, kv, []byte(fmt.Sprintf("escrow_%s", jobID)), escrow)
	kv.Set([]byte(fmt.Sprintf("pending_job_%d", sequence)), []byte(jobID))

	keeperpkg.TrackPendingOperationForTest(k, ctx, keeperpkg.ChannelOperation{
		ChannelID:  channelID,
		Sequence:   sequence,
		PacketType: keeperpkg.PacketTypeSubmitJob,
		JobID:      jobID,
	})

	coins := sdk.NewCoins(sdk.NewCoin("upaw", amount))
	require.NoError(t, getComputeBankKeeper(t, k).MintCoins(ctx, types.ModuleName, coins))
	require.NoError(t, getComputeBankKeeper(t, k).SendCoinsFromModuleToAccount(ctx, types.ModuleName, requester, coins))
	orig := getComputeBankKeeper(t, k).GetBalance(ctx, requester, "upaw")
	moduleAddr := getComputeAccountKeeper(t, k).GetModuleAddress(types.ModuleName)
	require.NoError(t, getComputeBankKeeper(t, k).SendCoins(ctx, requester, moduleAddr, coins))

	require.True(t, getComputeBankKeeper(t, k).GetBalance(ctx, requester, "upaw").IsZero())

	ibcModule := compute.NewIBCModule(*k, nil)
	require.NoError(t, ibcModule.OnChanCloseConfirm(ctx, types.PortID, channelID))

	after := getComputeBankKeeper(t, k).GetBalance(ctx, requester, "upaw")
	require.Equal(t, orig.Amount, after.Amount)
	require.Len(t, k.GetPendingOperations(ctx, channelID), 0)

	foundCleanup := false
	foundClose := false
	for _, evt := range ctx.EventManager().Events() {
		switch evt.Type {
		case "compute_channel_cleanup":
			foundCleanup = true
		case types.EventTypeChannelClose:
			foundClose = true
		}
	}
	require.True(t, foundCleanup, "expected compute cleanup event")
	require.True(t, foundClose, "expected channel close event")
}

func getComputeStoreKey(t *testing.T, k *keeperpkg.Keeper) storetypes.StoreKey {
	t.Helper()
	field := reflect.ValueOf(k).Elem().FieldByName("storeKey")
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(storetypes.StoreKey)
}

func getComputeBankKeeper(t *testing.T, k *keeperpkg.Keeper) bankkeeper.Keeper {
	t.Helper()
	field := reflect.ValueOf(k).Elem().FieldByName("bankKeeper")
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(bankkeeper.Keeper)
}

func getComputeAccountKeeper(t *testing.T, k *keeperpkg.Keeper) accountkeeper.AccountKeeper {
	t.Helper()
	field := reflect.ValueOf(k).Elem().FieldByName("accountKeeper")
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(accountkeeper.AccountKeeper)
}

func setJSON(t *testing.T, store storetypes.KVStore, key []byte, v interface{}) {
	t.Helper()
	bz, err := json.Marshal(v)
	require.NoError(t, err)
	store.Set(key, bz)
}
