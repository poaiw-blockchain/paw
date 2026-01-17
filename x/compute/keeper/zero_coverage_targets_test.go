package keeper

import (
	"crypto/ed25519"
	"crypto/sha256"
	"testing"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

// Covers MigrateStoreKeys and migratePrefix paths.
func TestMigrateStoreKeysMovesOldPrefixes(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	store := ctx.KVStore(k.storeKey)

	oldPrefix := []byte{0x02} // maps to ProviderKeyPrefix
	oldKey := append(oldPrefix, []byte("foo")...)
	store.Set(oldKey, []byte("bar"))

	require.NoError(t, k.MigrateStoreKeys(ctx))

	newKey := append(ProviderKeyPrefix, []byte("foo")...)
	require.Equal(t, []byte("bar"), store.Get(newKey))
	require.Nil(t, store.Get(oldKey))
}

// Covers RegisterSigningKey (first registration and rotation) and DeleteRandomnessCommitment.
func TestRegisterSigningKeyAndRandomnessDeletion(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	provider := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	providerStr := provider.String()

	require.NoError(t, k.SetProvider(ctx, types.Provider{
		Address:  providerStr,
		Moniker:  "p1",
		Endpoint: "http://p1",
		Stake:    sdkmath.NewInt(1),
		Active:   true,
	}))

	// First-time registration
	priv := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	pub := priv.Public().(ed25519.PublicKey)
	require.NoError(t, k.RegisterSigningKey(ctx, provider, pub, nil))

	// Rotation with signature
	priv2 := ed25519.NewKeyFromSeed([]byte("01234567890123456789012345678901"))
	pub2 := priv2.Public().(ed25519.PublicKey)
	msg := []byte("ROTATE_KEY:" + providerStr)
	msg = append(msg, pub2...)
	hash := sha256.Sum256(msg)
	sig := ed25519.Sign(priv, hash[:])
	require.NoError(t, k.RegisterSigningKey(ctx, provider, pub2, sig))

	// Randomness commitment delete
	commit := types.RandomnessCommitment{
		Validator:      providerStr,
		CommitmentHash: []byte{0xAA},
		BlockHeight:    10,
	}
	require.NoError(t, k.SetRandomnessCommitment(ctx, commit))
	k.DeleteRandomnessCommitment(ctx, provider)
	store := k.getStore(ctx)
	require.Nil(t, store.Get(RandomnessCommitmentKey(provider)))
}

// Covers SubmitBatchRequests gas guards without executing batch.
func TestSubmitBatchRequestsGasGuard(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(1000)) // force low remaining gas

	msgSrv := NewMsgServerImpl(*k)
	req := &types.MsgSubmitBatchRequests{
		Requester: sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String(),
		Requests: []types.BatchRequestItem{
			{Specs: types.ComputeSpec{CpuCores: 1}, MaxPayment: sdkmath.NewInt(1)},
			{Specs: types.ComputeSpec{CpuCores: 1}, MaxPayment: sdkmath.NewInt(1)},
		},
	}

	_, err := msgSrv.SubmitBatchRequests(ctx, req)
	require.Error(t, err) // expected gas exceed guard

	// Empty batch error path
	_, err = msgSrv.SubmitBatchRequests(ctx, &types.MsgSubmitBatchRequests{Requester: req.Requester})
	require.Error(t, err)
}

// Covers CatastrophicFailure queries, SimulateRequest validation, countPendingRequestsForProvider.
func TestQueryCatastrophicAndSimulate(t *testing.T) {
	k, sdkCtx := setupKeeperForTest(t)
	qs := NewQueryServerImpl(*k)

	// Seed catastrophic failure
	fail := types.CatastrophicFailure{
		Id:          1,
		RequestId:   2,
		Account:     sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String(),
		Amount:      sdk.NewInt64Coin("upaw", 5).Amount,
		Reason:      "test",
		OccurredAt:  sdkCtx.BlockTime(),
		BlockHeight: sdkCtx.BlockHeight(),
	}
	require.NoError(t, k.setCatastrophicFailure(sdkCtx, fail))

	// Query single
	resp1, err := qs.CatastrophicFailure(sdkCtx, &types.QueryCatastrophicFailureRequest{FailureId: 1})
	require.NoError(t, err)
	require.Equal(t, uint64(1), resp1.Failure.Id)

	// Query all (unresolved)
	respAll, err := qs.CatastrophicFailures(sdkCtx, &types.QueryCatastrophicFailuresRequest{OnlyUnresolved: true})
	require.NoError(t, err)
	require.Len(t, respAll.Failures, 1)

	// SimulateRequest validation errors path
	simResp, err := qs.SimulateRequest(sdkCtx, &types.QuerySimulateRequestRequest{
		Specs: types.ComputeSpec{},
	})
	require.NoError(t, err)
	require.NotEmpty(t, simResp.ValidationErrors)

	// countPendingRequestsForProvider path
	provider := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	req := types.Request{
		Id:       10,
		Provider: provider.String(),
		Status:   types.REQUEST_STATUS_PENDING,
	}
	require.NoError(t, k.SetRequest(sdkCtx, req))
	store := sdkCtx.KVStore(k.storeKey)
	store.Set(RequestByProviderKey(provider, req.Id), []byte{1})

	count := qs.(*queryServer).countPendingRequestsForProvider(sdkCtx, provider.String())
	require.Equal(t, uint64(1), count)
}

// Covers migratePrefix helper via direct call.
func TestMigratePrefixHelper(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	store := ctx.KVStore(k.storeKey)

	oldPrefix := []byte{0xAA}
	oldKey := append(oldPrefix, []byte("k")...)
	store.Set(oldKey, []byte("v"))

	require.NoError(t, k.migratePrefix(store, oldPrefix, []byte{0xBB}))

	require.Nil(t, store.Get(oldKey))
	require.Equal(t, []byte("v"), store.Get(append([]byte{0xBB}, []byte("k")...)))
}
