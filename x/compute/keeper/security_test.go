package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestSizeGuardsVerifyZKProof(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	huge := make([]byte, 11*1024*1024) // >10MB absolute max
	err := k.VerifyZKProof(sdkCtx, huge, make([]byte, 60))
	require.Error(t, err)
}
