//go:build go1.18
// +build go1.18

package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func FuzzNonceUniquenessInvariant(f *testing.F) {
	f.Add(uint64(1), uint64(2))
	f.Fuzz(func(t *testing.T, n1 uint64, n2 uint64) {
		k, ctx := setupKeeperForTest(t)
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		prov := sdk.AccAddress([]byte("fuzz_provider_aaaaaaaaaaaa"))

		store := sdkCtx.KVStore(k.storeKey)
		store.Set(NonceKey(prov, n1), []byte{0x1})
		store.Set(NonceKey(prov, n2), []byte{0x2})

		_, broken := NonceUniquenessInvariant(*k)(sdkCtx)
		if n1 == n2 {
			if !broken {
				t.Fatalf("expected broken invariant for duplicate nonce %d", n1)
			}
		}
	})
}
