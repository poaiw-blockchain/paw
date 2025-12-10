//go:build go1.18
// +build go1.18

package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

func FuzzDisputeIndexInvariant(f *testing.F) {
	f.Add(uint64(1), uint64(2))
	f.Fuzz(func(t *testing.T, disputeID uint64, requestID uint64) {
		k, ctx := setupKeeperForTest(t)
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		dispute := types.Dispute{Id: disputeID, RequestId: requestID, Status: types.DISPUTE_STATUS_VOTING}
		_ = k.setDispute(sdkCtx, dispute)
		// Remove status index to simulate corruption
		sdkCtx.KVStore(k.storeKey).Delete(DisputeByStatusKey(uint32(dispute.Status), dispute.Id))

		_, broken := DisputeIndexInvariant(*k)(sdkCtx)
		if disputeID != 0 && requestID != 0 {
			// expect broken when indexes missing
			if !broken {
				t.Fatalf("expected broken dispute index for %d", disputeID)
			}
		}
	})
}
