package keeper_test

import (
	"context"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/keeper"
)

// newComputeKeeperCtx returns a compute keeper with both SDK and standard contexts.
func newComputeKeeperCtx(t *testing.T) (*keeper.Keeper, sdk.Context, context.Context) {
	t.Helper()

	k, sdkCtx := keepertest.ComputeKeeper(t)
	sdkCtx = sdkCtx.WithContext(context.Background())
	sdkCtx = sdkCtx.WithBlockTime(time.Now().UTC())
	return k, sdkCtx, sdk.WrapSDKContext(sdkCtx)
}
