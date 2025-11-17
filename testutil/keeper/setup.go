package keeper

import (
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/paw-chain/paw/app"
)

// SetupTestApp initializes a test application with all modules and validators
func SetupTestApp(t *testing.T) (*app.PAWApp, sdk.Context) {
	db := dbm.NewMemDB()
	testApp := app.NewPAWApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		simtestutil.EmptyAppOptions{},
		baseapp.SetChainID("paw-test-1"),
	)

	// Create validator account
	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	address := sdk.AccAddress(pubKey.Address())
	valAddress := sdk.ValAddress(address)

	// Create genesis state with validator
	genesisState := createGenesisStateWithValidator(t, "paw-test-1", address, valAddress, pubKey)
	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	if err != nil {
		panic(err)
	}

	// Call InitChain to initialize all modules with validator
	_, err = testApp.InitChain(
		&abci.RequestInitChain{
			ChainId:       "paw-test-1",
			Validators:    []abci.ValidatorUpdate{},
			AppStateBytes: stateBytes,
			Time:          time.Now(),
		},
	)
	if err != nil {
		panic(err)
	}

	// Commit the initial block to finalize store setup
	_, err = testApp.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 1,
		Time:   time.Now(),
	})
	if err != nil {
		panic(err)
	}

	// Commit changes to finalize block 1
	_, err = testApp.Commit()
	if err != nil {
		panic(err)
	}

	// Start block 2 to have a proper context
	_, err = testApp.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 2,
		Time:   time.Now(),
	})
	if err != nil {
		panic(err)
	}

	// Create context from the current block
	ctx := testApp.NewUncachedContext(false, cmtproto.Header{
		ChainID: "paw-test-1",
		Height:  2,
		Time:    time.Now(),
	})

	return testApp, ctx
}

// createGenesisStateWithValidator creates genesis state with a validator
func createGenesisStateWithValidator(t *testing.T, chainID string, address sdk.AccAddress, valAddress sdk.ValAddress, pubKey cryptotypes.PubKey) map[string]json.RawMessage {
	encCfg := app.MakeEncodingConfig()
	genesisState := app.NewDefaultGenesisState(chainID)

	// Bond denom and amounts
	bondDenom := "upaw"
	accountTokens := sdk.TokensFromConsensusPower(1000, sdk.DefaultPowerReduction)
	bondedTokens := sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction)

	// Create account
	baseAcc := authtypes.NewBaseAccount(address, pubKey, 0, 0)
	accounts := []authtypes.GenesisAccount{baseAcc}

	// Update auth genesis
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), accounts)
	genesisState[authtypes.ModuleName] = encCfg.Codec.MustMarshalJSON(authGenesis)

	// Create balances for validator account and bonded pool
	bondedPoolAddress := authtypes.NewModuleAddress(stakingtypes.BondedPoolName)
	balances := []banktypes.Balance{
		{
			Address: address.String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(bondDenom, accountTokens)),
		},
		{
			Address: bondedPoolAddress.String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(bondDenom, bondedTokens)),
		},
	}

	// Update bank genesis with total supply including bonded tokens
	totalSupply := sdk.NewCoins(sdk.NewCoin(bondDenom, accountTokens.Add(bondedTokens)))
	bankGenesis := banktypes.NewGenesisState(
		banktypes.DefaultParams(),
		balances,
		totalSupply,
		[]banktypes.Metadata{},
		[]banktypes.SendEnabled{},
	)
	genesisState[banktypes.ModuleName] = encCfg.Codec.MustMarshalJSON(bankGenesis)

	// Create validator
	pkAny, err := codectypes.NewAnyWithValue(pubKey)
	if err != nil {
		panic(err)
	}

	validator := stakingtypes.Validator{
		OperatorAddress:   valAddress.String(),
		ConsensusPubkey:   pkAny,
		Jailed:            false,
		Status:            stakingtypes.Bonded,
		Tokens:            bondedTokens,
		DelegatorShares:   math.LegacyNewDecFromInt(bondedTokens),
		Description:       stakingtypes.Description{Moniker: "test-validator"},
		UnbondingHeight:   int64(0),
		UnbondingTime:     time.Unix(0, 0).UTC(),
		Commission:        stakingtypes.NewCommission(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()),
		MinSelfDelegation: math.OneInt(),
	}

	// Create delegation
	delegation := stakingtypes.Delegation{
		DelegatorAddress: address.String(),
		ValidatorAddress: valAddress.String(),
		Shares:           math.LegacyNewDecFromInt(bondedTokens),
	}

	// Update staking genesis with proper bond denom
	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenom = bondDenom
	stakingGenesis := stakingtypes.NewGenesisState(
		stakingParams,
		[]stakingtypes.Validator{validator},
		[]stakingtypes.Delegation{delegation},
	)
	genesisState[stakingtypes.ModuleName] = encCfg.Codec.MustMarshalJSON(stakingGenesis)

	return genesisState
}
