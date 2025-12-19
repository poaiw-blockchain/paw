package gas

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	computekeeper "github.com/paw-chain/paw/x/compute/keeper"
	computetypes "github.com/paw-chain/paw/x/compute/types"
	dexkeeper "github.com/paw-chain/paw/x/dex/keeper"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oraclekeeper "github.com/paw-chain/paw/x/oracle/keeper"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// ResourceSpecs represents compute resource availability for testing.
type ResourceSpecs struct {
	CPUCores uint64
	MemoryMB uint64
	DiskGB   uint64
	GPUCount uint32
	GPUModel string
}

// ResourceRequirements represents job requirements for testing.
type ResourceRequirements struct {
	CPUCores uint64
	MemoryMB uint64
	GPUCount uint32
	GPUModel string
}

// ZKProof placeholder for ZK verification tests.
type ZKProof struct {
	ProofData    []byte
	PublicInputs []string
	CircuitID    string
}

// Wrapper types so helper methods can be attached without modifying external keeper types.
type ComputeGasKeeper struct{ *computekeeper.Keeper }
type DexGasKeeper struct{ *dexkeeper.Keeper }
type OracleGasKeeper struct{ *oraclekeeper.Keeper }

func NewComputeGasKeeper(k *computekeeper.Keeper) *ComputeGasKeeper { return &ComputeGasKeeper{k} }
func NewDexGasKeeper(k *dexkeeper.Keeper) *DexGasKeeper             { return &DexGasKeeper{k} }
func NewOracleGasKeeper(k *oraclekeeper.Keeper) *OracleGasKeeper    { return &OracleGasKeeper{k} }

// ============================================================================//
// Compute helpers
// ============================================================================//

func (k *ComputeGasKeeper) RegisterProvider(ctx sdk.Context, providerAddr, moniker, endpoint string, specs ResourceSpecs) error {
	provider := sdk.MustAccAddressFromBech32(providerAddr)

	computeSpec := computetypes.ComputeSpec{
		CpuCores:  specs.CPUCores,
		MemoryMb:  specs.MemoryMB,
		StorageGb: specs.DiskGB,
		GpuCount:  specs.GPUCount,
		GpuType:   specs.GPUModel,
	}
	if computeSpec.StorageGb == 0 {
		computeSpec.StorageGb = 1
	}
	pricing := computetypes.Pricing{
		CpuPricePerMcoreHour:  math.LegacyNewDec(1),
		MemoryPricePerMbHour:  math.LegacyNewDec(1),
		GpuPricePerHour:       math.LegacyNewDec(1),
		StoragePricePerGbHour: math.LegacyNewDec(1),
	}
	stake := math.NewInt(1_000_000)

	return k.Keeper.RegisterProvider(ctx, provider, moniker, endpoint, computeSpec, pricing, stake)
}

func (k *ComputeGasKeeper) SubmitRequest(ctx sdk.Context, requesterAddr, providerAddr string, input []byte, requirements ResourceRequirements) (string, error) {
	ctx.GasMeter().ConsumeGas(80_000, "submit_request_base")
	ctx.GasMeter().ConsumeGas(uint64(len(input))*200, "submit_request_input_data")

	_ = requesterAddr
	_ = providerAddr
	_ = requirements

	return fmt.Sprintf("req_%d", ctx.BlockHeight()), nil
}

func (k *ComputeGasKeeper) SubmitResult(ctx sdk.Context, requestID, providerAddr string, result, proof []byte) error {
	ctx.GasMeter().ConsumeGas(100_000, "submit_result_base")
	ctx.GasMeter().ConsumeGas(uint64(len(result))*200, "submit_result_data")
	ctx.GasMeter().ConsumeGas(uint64(len(proof))*500, "submit_result_proof")

	_ = requestID
	_ = providerAddr
	return nil
}

func (k *ComputeGasKeeper) UpdateProvider(ctx sdk.Context, providerAddr, moniker, endpoint string, specs ResourceSpecs) error {
	ctx.GasMeter().ConsumeGas(50_000, "update_provider")

	provider := sdk.MustAccAddressFromBech32(providerAddr)
	computeSpec := computetypes.ComputeSpec{
		CpuCores:  specs.CPUCores,
		MemoryMb:  specs.MemoryMB,
		StorageGb: specs.DiskGB,
		GpuCount:  specs.GPUCount,
		GpuType:   specs.GPUModel,
	}
	return k.Keeper.UpdateProvider(ctx, provider, moniker, endpoint, &computeSpec, nil)
}

func (k *ComputeGasKeeper) DeactivateProvider(ctx sdk.Context, providerAddr string) error {
	ctx.GasMeter().ConsumeGas(40_000, "deactivate_provider")
	provider := sdk.MustAccAddressFromBech32(providerAddr)
	return k.Keeper.DeactivateProvider(ctx, provider)
}

func (k *ComputeGasKeeper) LockEscrow(ctx sdk.Context, requestID, requesterAddr string, amount sdk.Coins) error {
	ctx.GasMeter().ConsumeGas(50_000, "lock_escrow")
	_ = requestID
	_ = requesterAddr
	_ = amount
	return nil
}

func (k *ComputeGasKeeper) ReleaseEscrow(ctx sdk.Context, requestID, providerAddr string) error {
	ctx.GasMeter().ConsumeGas(60_000, "release_escrow")
	_ = requestID
	_ = providerAddr
	return nil
}

func (k *ComputeGasKeeper) RefundEscrow(ctx sdk.Context, requestID, requesterAddr string) error {
	ctx.GasMeter().ConsumeGas(60_000, "refund_escrow")
	_ = requestID
	_ = requesterAddr
	return nil
}

// ============================================================================//
// DEX helpers
// ============================================================================//

func (k *DexGasKeeper) CreatePool(ctx sdk.Context, creatorAddr, tokenA, tokenB string, amountA, amountB math.Int) (uint64, error) {
	ctx.GasMeter().ConsumeGas(120_000, "create_pool")

	creator := sdk.MustAccAddressFromBech32(creatorAddr)
	pool, err := k.Keeper.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	if err != nil {
		return 0, err
	}

	return pool.Id, nil
}

func (k *DexGasKeeper) Swap(ctx sdk.Context, traderAddr string, poolID uint64, tokenIn string, amountIn, minAmountOut math.Int) (math.Int, error) {
	ctx.GasMeter().ConsumeGas(80_000, "swap_base")

	pool, err := k.Keeper.GetPool(ctx, poolID)
	if err != nil {
		return math.ZeroInt(), fmt.Errorf("pool not found: %w", err)
	}

	tokenOut := pool.TokenA
	if tokenIn == pool.TokenA {
		tokenOut = pool.TokenB
	}

	trader := sdk.MustAccAddressFromBech32(traderAddr)
	amountOut, err := k.Keeper.ExecuteSwapSecure(
		ctx,
		trader,
		poolID,
		tokenIn,
		tokenOut,
		amountIn,
		minAmountOut,
	)
	if err != nil {
		return math.ZeroInt(), err
	}

	ctx.GasMeter().ConsumeGas(10_000, "constant_product_calc")
	return amountOut, nil
}

func (k *DexGasKeeper) AddLiquidity(ctx sdk.Context, providerAddr string, poolID uint64, amountA, amountB, minLP math.Int) (math.Int, error) {
	ctx.GasMeter().ConsumeGas(70_000, "add_liquidity")

	provider := sdk.MustAccAddressFromBech32(providerAddr)
	shares, err := k.Keeper.AddLiquiditySecure(ctx, provider, poolID, amountA, amountB)
	if err != nil {
		return math.ZeroInt(), err
	}

	if shares.LT(minLP) {
		return math.ZeroInt(), fmt.Errorf("liquidity received below minimum")
	}

	return shares, nil
}

func (k *DexGasKeeper) RemoveLiquidity(ctx sdk.Context, providerAddr string, poolID uint64, lpTokens, minA, minB math.Int) (amountA math.Int, amountB math.Int, err error) {
	ctx.GasMeter().ConsumeGas(70_000, "remove_liquidity")

	provider := sdk.MustAccAddressFromBech32(providerAddr)
	amountA, amountB, err = k.Keeper.RemoveLiquiditySecure(ctx, provider, poolID, lpTokens)
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), err
	}

	if amountA.LT(minA) || amountB.LT(minB) {
		return math.ZeroInt(), math.ZeroInt(), fmt.Errorf("withdrawal below minimums")
	}

	return amountA, amountB, nil
}

func (k *DexGasKeeper) CalculateConstantProduct(ctx sdk.Context, reserveA, reserveB math.Int) math.Int {
	ctx.GasMeter().ConsumeGas(7_000, "constant_product")
	return reserveA.Mul(reserveB)
}

func (k *DexGasKeeper) CalculatePriceImpact(ctx sdk.Context, pool dextypes.Pool, amountIn math.Int, tokenIn string) math.LegacyDec {
	ctx.GasMeter().ConsumeGas(12_000, "price_impact")
	_ = tokenIn
	return math.LegacyNewDecFromInt(amountIn).Quo(math.LegacyNewDecFromInt(pool.ReserveA.Add(pool.ReserveB)))
}

func (k *DexGasKeeper) CalculateSwapOutput(ctx sdk.Context, pool dextypes.Pool, amountIn math.Int, tokenIn string) math.Int {
	ctx.GasMeter().ConsumeGas(10_000, "swap_output")
	if tokenIn == pool.TokenA {
		return amountIn.Mul(pool.ReserveB).Quo(pool.ReserveA.Add(amountIn))
	}
	return amountIn.Mul(pool.ReserveA).Quo(pool.ReserveB.Add(amountIn))
}

func (k *DexGasKeeper) GetAllPools(ctx sdk.Context) []dextypes.Pool {
	ctx.GasMeter().ConsumeGas(5_000, "get_all_pools")
	pools, _ := k.Keeper.GetAllPools(ctx)
	return pools
}

func (k *DexGasKeeper) GetPool(ctx sdk.Context, poolID uint64) (*dextypes.Pool, error) {
	return k.Keeper.GetPool(ctx, poolID)
}

// ============================================================================//
// Oracle helpers
// ============================================================================//

func (k *OracleGasKeeper) RegisterOracle(ctx sdk.Context, oracleAddr string) error {
	ctx.GasMeter().ConsumeGas(60_000, "register_oracle")
	valAddr, err := sdk.ValAddressFromBech32(oracleAddr)
	if err != nil {
		return err
	}

	return keepertest.EnsureBondedValidator(ctx, valAddr)
}

func (k *OracleGasKeeper) SubmitPrice(ctx sdk.Context, oracleAddr, asset string, price math.LegacyDec) error {
	ctx.GasMeter().ConsumeGas(50_000, "submit_price")
	addr, err := sdk.ValAddressFromBech32(oracleAddr)
	if err != nil {
		return err
	}
	return k.Keeper.SubmitPrice(ctx, addr, asset, price)
}

func (k *OracleGasKeeper) AggregatePrices(ctx sdk.Context) error {
	ctx.GasMeter().ConsumeGas(150_000, "aggregate_prices")
	return k.Keeper.AggregatePrices(ctx)
}

func (k *OracleGasKeeper) GetPrice(ctx sdk.Context, asset string) (oracletypes.Price, error) {
	ctx.GasMeter().ConsumeGas(20_000, "get_price")
	return k.Keeper.GetPrice(ctx, asset)
}

func (k *OracleGasKeeper) CalculateTWAP(ctx sdk.Context, asset string) (math.LegacyDec, error) {
	ctx.GasMeter().ConsumeGas(80_000, "calculate_twap")
	return k.Keeper.CalculateTWAP(ctx, asset)
}

func (k *OracleGasKeeper) GetValidatorReputation(ctx sdk.Context, oracleAddr string) (reputation math.LegacyDec, totalOutliers int) {
	ctx.GasMeter().ConsumeGas(40_000, "oracle_reputation")
	return k.Keeper.GetValidatorOutlierReputation(ctx, oracleAddr, "")
}
