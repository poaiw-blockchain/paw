package keeper

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	accountkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/setup"
	"github.com/paw-chain/paw/x/compute/types"
)

func TestCeremonyKeySinkPersistsKeysInKeeper(t *testing.T) {
	t.Parallel()

	k, sdkCtx := newCeremonyKeeper(t)
	goCtx := sdk.WrapSDKContext(sdkCtx)

	circuit := &testEqualityCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(t, err)

	beacon := deterministicCeremonyBeacon{}
	sink := NewCeremonyKeySink(k)
	ceremony := setup.NewMPCCeremony("mpc-keeper-integration", ccs, setup.SecurityLevel256, beacon, sink)

	participants := []struct {
		id  string
		key []byte
	}{
		{"alice", randomTestBytes(t, 32)},
		{"bob", randomTestBytes(t, 32)},
		{"charlie", randomTestBytes(t, 32)},
	}

	for _, p := range participants {
		require.NoError(t, ceremony.RegisterParticipant(p.id, p.key))
	}
	require.NoError(t, ceremony.StartCeremony())

	for _, p := range participants {
		_, err := ceremony.Contribute(p.id, randomTestBytes(t, 64))
		require.NoError(t, err)
	}

	_, _, err = ceremony.Finalize(goCtx)
	require.NoError(t, err)

	params, err := k.GetCircuitParams(goCtx, "mpc-keeper-integration")
	require.NoError(t, err)
	require.Equal(t, "mpc-keeper-integration", params.CircuitId)
	require.NotEmpty(t, params.VerifyingKey.VkData)

	store := sdkCtx.KVStore(k.storeKey)
	pkKey := []byte("zk_proving_key_mpc-keeper-integration")
	require.NotNil(t, store.Get(pkKey))
}

func newCeremonyKeeper(t *testing.T) (*Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	ms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	k := NewKeeper(
		cdc,
		storeKey,
		nil,
		accountkeeper.AccountKeeper{},
		nil,
		slashingkeeper.Keeper{},
		nil,
		nil,
		"",
		capabilitykeeper.ScopedKeeper{},
	)

	header := cmtproto.Header{Time: time.Now().UTC()}
	sdkCtx := sdk.NewContext(ms, header, false, log.NewNopLogger())
	sdkCtx = sdkCtx.WithContext(context.Background())

	return k, sdkCtx
}

type testEqualityCircuit struct {
	A frontend.Variable `gnark:",public"`
	B frontend.Variable `gnark:",public"`
}

func (c *testEqualityCircuit) Define(api frontend.API) error {
	api.AssertIsEqual(c.A, c.B)
	return nil
}

type deterministicCeremonyBeacon struct{}

func (deterministicCeremonyBeacon) GetRandomness(round uint64) ([]byte, error) {
	return []byte{byte(round & 0xff)}, nil
}

func (deterministicCeremonyBeacon) VerifyRandomness(round uint64, randomness []byte) bool {
	return len(randomness) > 0 && randomness[0] == byte(round&0xff)
}

func randomTestBytes(t *testing.T, size int) []byte {
	t.Helper()
	buf := make([]byte, size)
	_, err := rand.Read(buf)
	require.NoError(t, err)
	return buf
}
