package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestResolveDisputeInvalidAuthority(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	dispute := types.Dispute{Id: 5, Status: types.DISPUTE_STATUS_VOTING}
	require.NoError(t, k.setDispute(sdkCtx, dispute))

	_, err := k.ResolveDispute(ctx, sdk.AccAddress{}, dispute.Id)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unauthorized")
}
