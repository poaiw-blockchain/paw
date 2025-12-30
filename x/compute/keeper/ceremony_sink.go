package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/setup"
)

// ceremonyKeySink persists MPC ceremony keys in keeper state.
type ceremonyKeySink struct {
	keeper *Keeper
}

// NewCeremonyKeySink creates a sink that satisfies setup.CircuitKeySink.
func NewCeremonyKeySink(k *Keeper) setup.CircuitKeySink {
	return &ceremonyKeySink{keeper: k}
}

// StoreCeremonyKeys persists the verifying key under the circuit params and stores the
// proving key bytes for off-chain auditing.
func (s *ceremonyKeySink) StoreCeremonyKeys(ctx context.Context, circuitID string, provingKey, verifyingKey []byte) error {
	if len(verifyingKey) == 0 {
		return fmt.Errorf("verifying key cannot be empty")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	params, err := s.keeper.GetCircuitParams(ctx, circuitID)
	if err != nil || params == nil {
		params = s.keeper.getDefaultCircuitParams(ctx, circuitID)
	}
	params.CircuitId = circuitID
	params.VerifyingKey.VkData = append([]byte(nil), verifyingKey...)
	params.VerifyingKey.CreatedAt = sdkCtx.BlockTime()

	if err := s.keeper.SetCircuitParams(ctx, *params); err != nil {
		return fmt.Errorf("StoreCeremonyKeys: set params for circuit %s: %w", circuitID, err)
	}

	store := sdkCtx.KVStore(s.keeper.storeKey)
	storeKey := []byte(fmt.Sprintf("zk_proving_key_%s", circuitID))
	store.Set(storeKey, append([]byte(nil), provingKey...))

	return nil
}
