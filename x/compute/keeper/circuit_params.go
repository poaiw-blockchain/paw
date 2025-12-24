package keeper

import (
	"context"
	"encoding/json"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// GetCircuitParams retrieves circuit parameters from state.
func (k *Keeper) GetCircuitParams(ctx context.Context, circuitID string) (*types.CircuitParams, error) {
	store := k.getStore(ctx)
	key := CircuitParamsKey(circuitID)

	bz := store.Get(key)
	if bz == nil {
		// Return default params if not found
		return k.getDefaultCircuitParams(ctx, circuitID), nil
	}

	var params types.CircuitParams
	if err := json.Unmarshal(bz, &params); err != nil {
		return nil, err
	}

	return &params, nil
}

// SetCircuitParams stores circuit parameters.
func (k *Keeper) SetCircuitParams(ctx context.Context, params types.CircuitParams) error {
	store := k.getStore(ctx)
	key := CircuitParamsKey(params.CircuitId)

	bz, err := json.Marshal(&params)
	if err != nil {
		return err
	}

	store.Set(key, bz)
	return nil
}

// getDefaultCircuitParams returns default circuit parameters.
func (k *Keeper) getDefaultCircuitParams(ctx context.Context, circuitID string) *types.CircuitParams {
	createdAt := time.Now().UTC()
	if sdkCtx, ok := ctx.(sdk.Context); ok && !sdkCtx.BlockTime().IsZero() {
		createdAt = sdkCtx.BlockTime()
	}

	return &types.CircuitParams{
		CircuitId:   circuitID,
		Description: "Compute result verification circuit using Groth16",
		VerifyingKey: types.VerifyingKey{
			CircuitId:        circuitID,
			Curve:            "bn254",
			ProofSystem:      "groth16",
			CreatedAt:        createdAt,
			PublicInputCount: 3, // RequestID, ResultHash, ProviderAddress
		},
		MaxProofSize:              1024 * 1024, // 1MB max - prevents DoS via oversized proofs
		GasCost:                   500000,      // Gas cost for verification (~0.5M gas)
		Enabled:                   true,
		VerificationDepositAmount: 1000000, // 1,000,000 upaw (1 PAW) - refunded on valid proof, slashed on invalid
	}
}

// GetZKMetrics retrieves ZK metrics from state.
func (k *Keeper) GetZKMetrics(ctx context.Context) (*types.ZKMetrics, error) {
	store := k.getStore(ctx)
	bz := store.Get([]byte("zk_metrics"))

	if bz == nil {
		// Return empty metrics instead of error for first-time use
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		return &types.ZKMetrics{
			LastUpdated: sdkCtx.BlockTime(),
		}, nil
	}

	var metrics types.ZKMetrics
	if err := json.Unmarshal(bz, &metrics); err != nil {
		return nil, err
	}

	return &metrics, nil
}

// SetZKMetrics stores ZK metrics.
func (k *Keeper) SetZKMetrics(ctx context.Context, metrics types.ZKMetrics) error {
	store := k.getStore(ctx)

	bz, err := json.Marshal(&metrics)
	if err != nil {
		return err
	}

	store.Set([]byte("zk_metrics"), bz)
	return nil
}
