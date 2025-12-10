package keeper

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestBuildMerkleProofPath(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	path := k.buildMerkleProofPath(sdkCtx, []byte("escrow_key"))
	require.Len(t, path, 3)
	require.NotEmpty(t, path[0])
	require.NotEmpty(t, path[1])
	require.NotEmpty(t, path[2])
}

func TestPendingDiscoveryStorage(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	k.storePendingDiscovery(sdkCtx, "channel-1", 7, "chain-X")
	store := sdkCtx.KVStore(k.storeKey)
	key := []byte("pending_discovery_7")
	require.Equal(t, []byte("chain-X"), store.Get(key))

	k.removePendingDiscovery(sdkCtx, "channel-1", 7)
	require.Nil(t, store.Get(key))
}

func TestGroth16ProofValidate_FailsOnInfinity(t *testing.T) {
	proof := &Groth16ProofBN254{}
	err := proof.Validate()
	require.Error(t, err)
}

func TestSendComputeIBCPacket_NoIBCKeeper(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	_, err := k.sendComputeIBCPacket(sdkCtx, "channel-0", []byte("data"), time.Minute)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ibc keeper not configured")
}

func TestSendComputeIBCPacket_SucceedsWithCapability(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	k.ibcKeeper = &ibckeeper.Keeper{}

	capPath := host.ChannelCapabilityPath(types.PortID, "channel-0")
	cap, err := k.scopedKeeper.NewCapability(sdkCtx, capPath)
	require.NoError(t, err)
	if err := k.scopedKeeper.ClaimCapability(sdkCtx, cap, capPath); err != nil {
		require.ErrorIs(t, err, capabilitytypes.ErrOwnerClaimed)
	}

	originalSend := sendPacketFn
	sendPacketFn = func(
		_ *Keeper,
		_ sdk.Context,
		_ *capabilitytypes.Capability,
		_ string,
		_ string,
		_ uint64,
		_ []byte,
	) (uint64, error) {
		return 99, nil
	}
	t.Cleanup(func() { sendPacketFn = originalSend })

	seq, err := k.sendComputeIBCPacket(sdkCtx, "channel-0", []byte("data"), time.Minute)
	require.NoError(t, err)
	require.Equal(t, uint64(99), seq)
}

func TestSendComputeIBCPacket_ChannelCapabilityMissing(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// provide a non-nil IBC keeper to bypass the initial nil check
	k.ibcKeeper = &ibckeeper.Keeper{}

	_, err := k.sendComputeIBCPacket(sdkCtx, "channel-0", []byte("data"), time.Minute)
	require.Error(t, err)
	require.Contains(t, err.Error(), "channel capability not found")
}

func TestSendComputeIBCPacket_SendPacketFailure(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Seed a channel capability so SendPacket would be attempted
	capPath := host.ChannelCapabilityPath(types.PortID, "channel-0")
	_, err := k.scopedKeeper.NewCapability(sdkCtx, capPath)
	require.NoError(t, err)

	k.ibcKeeper = &ibckeeper.Keeper{}

	originalSend := sendPacketFn
	sendPacketFn = func(
		_ *Keeper,
		_ sdk.Context,
		_ *capabilitytypes.Capability,
		_ string,
		_ string,
		_ uint64,
		_ []byte,
	) (uint64, error) {
		return 0, fmt.Errorf("send failure")
	}
	defer func() { sendPacketFn = originalSend }()

	_, err = k.sendComputeIBCPacket(sdkCtx, "channel-0", []byte("data"), time.Minute)
	require.Error(t, err)
	require.Contains(t, err.Error(), "send failure")
}

func TestDiscoverRemoteProviders_NoIBCKeeper(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	providers, err := k.DiscoverRemoteProviders(ctx, []string{AkashChainID}, []string{"gpu"}, math.LegacyNewDec(1))
	require.NoError(t, err)
	require.Len(t, providers, 0)
}

func TestSubmitCrossChainJob_NoIBCKeeper(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	requester := sdk.AccAddress([]byte("job_requester"))
	specs := JobRequirements{CPUCores: 1, MemoryMB: 512, StorageGB: 5}
	err := k.bankKeeper.MintCoins(sdk.UnwrapSDKContext(ctx), types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000)))
	require.NoError(t, err)
	err = k.bankKeeper.SendCoinsFromModuleToAccount(sdk.UnwrapSDKContext(ctx), types.ModuleName, requester, sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000)))
	require.NoError(t, err)

	_, err = k.SubmitCrossChainJob(ctx, "docker", []byte{0x1}, specs, AkashChainID, "provider1", requester, sdk.NewInt64Coin("upaw", 1000))
	require.Error(t, err)
}

func TestQueryCrossChainJobStatus_NoIBCKeeper(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	// store job
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	k.storeJob(sdkCtx, "job-123", &CrossChainComputeJob{
		JobID:       "job-123",
		TargetChain: AkashChainID,
		Status:      "pending",
		Requester:   sdk.AccAddress([]byte("job_req")).String(),
	})
	job, err := k.QueryCrossChainJobStatus(ctx, "job-123")
	require.NoError(t, err)
	require.Equal(t, "job-123", job.JobID)
}

func TestCreateEscrowProof(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("proof_requester"))
	amount := sdk.NewInt64Coin("upaw", 100)
	require.NoError(t, k.bankKeeper.MintCoins(sdkCtx, types.ModuleName, sdk.NewCoins(amount)))
	require.NoError(t, k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, requester, sdk.NewCoins(amount)))
	require.NoError(t, k.lockEscrow(sdkCtx, requester, amount))

	escrow := &CrossChainEscrow{
		JobID:     "job-proof-1",
		Requester: requester.String(),
		Provider:  requester.String(),
		Amount:    amount.Amount,
		Status:    "locked",
		LockedAt:  sdkCtx.BlockTime(),
	}
	k.storeEscrow(sdkCtx, escrow.JobID, escrow)

	proof, err := k.createEscrowProof(sdkCtx, escrow)
	require.NoError(t, err)
	require.NotEmpty(t, proof)
}

func TestVerifyGroth16PairingFailsWithoutCircuit(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	proof := &Groth16ProofBN254{}
	err := k.verifyGroth16Pairing(sdkCtx, []byte{}, proof, bn254.G1Affine{})
	require.Error(t, err)
}
