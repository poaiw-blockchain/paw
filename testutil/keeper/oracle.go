package keeper

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// OracleKeeper creates a test keeper for the Oracle module with mock dependencies
func OracleKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		&MockBankKeeper{},
		&MockStakingKeeper{},
		&MockSlashingKeeper{},
		authority.String(),
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())

	// Initialize module genesis
	k.InitGenesis(ctx, *types.DefaultGenesis())

	return k, ctx
}

// MockBankKeeper is a mock implementation of BankKeeper for testing
type MockBankKeeper struct{}

func (m *MockBankKeeper) SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (m *MockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	return nil
}

func (m *MockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (m *MockBankKeeper) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return sdk.NewCoin(denom, math.NewInt(0))
}

// MockStakingKeeper is a mock implementation of StakingKeeper for testing
type MockStakingKeeper struct{}

func (m *MockStakingKeeper) GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error) {
	return stakingtypes.Validator{}, nil
}

func (m *MockStakingKeeper) IterateBondedValidatorsByPower(ctx context.Context, fn func(index int64, validator stakingtypes.ValidatorI) (stop bool)) error {
	return nil
}

func (m *MockStakingKeeper) PowerReduction(ctx context.Context) math.Int {
	return math.NewInt(1000000)
}

// MockSlashingKeeper is a mock implementation of SlashingKeeper for testing
type MockSlashingKeeper struct{}

func (m *MockSlashingKeeper) Slash(ctx context.Context, consAddr sdk.ConsAddress, slashFactor math.LegacyDec, infractionHeight, power int64) error {
	// Mock implementation - do nothing
	return nil
}

func (m *MockSlashingKeeper) SlashWithInfractionReason(ctx context.Context, consAddr sdk.ConsAddress, slashFactor math.LegacyDec, infractionHeight, power int64, infraction stakingtypes.Infraction) error {
	// Mock implementation - do nothing
	return nil
}

// RegisterTestOracle registers a test oracle validator
// TODO: Implement when message handlers are ready
func RegisterTestOracle(t testing.TB, k *keeper.Keeper, ctx sdk.Context, validator string) {
	// msgRegister := &types.MsgRegisterOracle{
	// 	Validator: validator,
	// }
	// _, err := k.RegisterOracle(ctx, msgRegister)
	// require.NoError(t, err)
	t.Skip("RegisterOracle not implemented yet")
}

// SubmitTestPrice submits a test price feed
// TODO: Implement when message handlers are ready
func SubmitTestPrice(t testing.TB, k *keeper.Keeper, ctx sdk.Context, oracle, asset string, price math.LegacyDec) {
	// msgSubmit := &types.MsgSubmitPrice{
	// 	Oracle: oracle,
	// 	Asset:  asset,
	// 	Price:  price,
	// }
	// _, err := k.SubmitPrice(ctx, msgSubmit)
	// require.NoError(t, err)
	t.Skip("SubmitPrice not implemented yet")
}
