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
	sdked25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
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
	t.Helper()

	db := dbm.NewMemDB()
	chainID := "paw-test-1"

	testApp := app.NewPAWApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		simtestutil.EmptyAppOptions{},
		baseapp.SetChainID("paw-test-1"),
	)

	// Create a default validator so staking/slashing/genesis-dependent suites work
	valPrivKey := sdked25519.GenPrivKey()
	valPubKey := valPrivKey.PubKey()
	delAddr := sdk.AccAddress(valPubKey.Address())
	valAddr := sdk.ValAddress(delAddr)
	bondedTokens := sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction)
	pkAny, err := codectypes.NewAnyWithValue(valPubKey)
	if err != nil {
		panic(err)
	}
	validator := stakingtypes.Validator{
		OperatorAddress:   valAddr.String(),
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

	genesisState := createGenesisStateWithValidator(t, chainID, delAddr, valAddr, valPubKey)

	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	if err != nil {
		panic(err)
	}

	_, err = testApp.InitChain(
		&abci.RequestInitChain{
			ChainId: chainID,
			Validators: []abci.ValidatorUpdate{
				validator.ABCIValidatorUpdate(sdk.DefaultPowerReduction),
			},
			AppStateBytes: stateBytes,
		},
	)
	if err != nil {
		panic(err)
	}

	ctx := testApp.NewContext(true).WithBlockHeader(cmtproto.Header{
		Height:  testApp.LastBlockHeight() + 1,
		ChainID: chainID,
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

	// Create account and keep existing module accounts intact
	baseAcc := authtypes.NewBaseAccount(address, pubKey, 0, 0)
	authGenesis := authtypes.GetGenesisStateFromAppState(encCfg.Codec, genesisState)
	baseAccAny, err := codectypes.NewAnyWithValue(baseAcc)
	if err != nil {
		panic(err)
	}
	authGenesis.Accounts = append(authGenesis.Accounts, baseAccAny)
	genesisState[authtypes.ModuleName] = encCfg.Codec.MustMarshalJSON(&authGenesis)

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

	// Update bank genesis with total supply including bonded tokens while retaining existing metadata
	bankGenesis := banktypes.GetGenesisStateFromAppState(encCfg.Codec, genesisState)
	bankGenesis.Balances = append(bankGenesis.Balances, balances...)
	totalSupply := sdk.NewCoin(bondDenom, accountTokens.Add(bondedTokens))
	if bankGenesis.Supply == nil {
		bankGenesis.Supply = sdk.NewCoins()
	}
	bankGenesis.Supply = bankGenesis.Supply.Add(totalSupply)
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
