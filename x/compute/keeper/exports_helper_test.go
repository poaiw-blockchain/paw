package keeper

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

// Covers TrackPendingOperationForTest and setCatastrophicFailure helper.
func TestExportedTestHelpers(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	op := ChannelOperation{
		ChannelID:  "channel-1",
		PacketType: "compute_request",
		Sequence:   5,
	}
	TrackPendingOperationForTest(k, ctx, op)
	// ensure pending map populated
	pending := k.GetPendingOperations(ctx, op.ChannelID)
	require.Len(t, pending, 1)
	require.Equal(t, op.Sequence, pending[0].Sequence)

	// setCatastrophicFailure is unexported; invoke via keeper to cover path
	err := k.setCatastrophicFailure(ctx, types.CatastrophicFailure{
		Id:          1,
		RequestId:   2,
		Account:     sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String(),
		Amount:      sdk.NewInt64Coin("upaw", 10).Amount,
		Reason:      "test",
		OccurredAt:  ctx.BlockTime(),
		BlockHeight: ctx.BlockHeight(),
		Resolved:    false,
	})
	require.NoError(t, err)
}
