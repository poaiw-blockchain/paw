package invariants_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/paw-chain/paw/app"
)

type BankInvariantsTestSuite struct {
	suite.Suite
	app *app.PAWApp
	ctx sdk.Context
}

func (s *BankInvariantsTestSuite) SetupTest() {
	db := dbm.NewMemDB()
	encCfg := app.MakeEncodingConfig()

	s.app = app.NewApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		map[int64]bool{},
		app.DefaultNodeHome,
		0,
		encCfg,
		app.GetEnabledProposals(),
		baseapp.SetChainID("paw-test-1"),
	)

	s.ctx = s.app.BaseApp.NewContext(false, tmproto.Header{
		ChainID: "paw-test-1",
		Height:  1,
	})
}

// InvariantTotalSupply checks that total supply equals sum of all account balances
func (s *BankInvariantsTestSuite) InvariantTotalSupply() (string, bool) {
	totalSupply := s.app.BankKeeper.GetSupply(s.ctx, "upaw")

	// Sum all account balances
	var accountTotal sdk.Int
	accountTotal = sdk.ZeroInt()

	s.app.AccountKeeper.IterateAccounts(s.ctx, func(account authtypes.AccountI) bool {
		balance := s.app.BankKeeper.GetBalance(s.ctx, account.GetAddress(), "upaw")
		accountTotal = accountTotal.Add(balance.Amount)
		return false
	})

	// Also add module account balances
	moduleAccounts := []string{
		authtypes.FeeCollectorName,
		"dex",
		"compute",
		"oracle",
		"bonded_tokens_pool",
		"not_bonded_tokens_pool",
	}

	for _, modName := range moduleAccounts {
		modAddr := authtypes.NewModuleAddress(modName)
		balance := s.app.BankKeeper.GetBalance(s.ctx, modAddr, "upaw")
		accountTotal = accountTotal.Add(balance.Amount)
	}

	broken := !totalSupply.Amount.Equal(accountTotal)
	msg := ""
	if broken {
		msg = sdk.FormatInvariant(
			banktypes.ModuleName,
			"total supply",
			"total supply does not equal sum of accounts\n"+
				"\ttotal supply: %s\n"+
				"\tsum of accounts: %s\n",
			totalSupply.Amount.String(),
			accountTotal.String(),
		)
	}

	return msg, broken
}

// InvariantNonNegativeBalances checks that no account has negative balance
func (s *BankInvariantsTestSuite) InvariantNonNegativeBalances() (string, bool) {
	var msg string
	var broken bool

	s.app.AccountKeeper.IterateAccounts(s.ctx, func(account authtypes.AccountI) bool {
		balances := s.app.BankKeeper.GetAllBalances(s.ctx, account.GetAddress())

		for _, balance := range balances {
			if balance.Amount.IsNegative() {
				broken = true
				msg += sdk.FormatInvariant(
					banktypes.ModuleName,
					"non-negative balances",
					"account %s has negative balance: %s\n",
					account.GetAddress().String(),
					balance.String(),
				)
			}
		}

		return false
	})

	return msg, broken
}

// InvariantDenomMetadata checks that all denoms with supply have metadata
func (s *BankInvariantsTestSuite) InvariantDenomMetadata() (string, bool) {
	totalSupply := s.app.BankKeeper.GetTotalSupply(s.ctx)
	var msg string
	var broken bool

	for _, coin := range totalSupply {
		_, found := s.app.BankKeeper.GetDenomMetaData(s.ctx, coin.Denom)
		if !found {
			broken = true
			msg += sdk.FormatInvariant(
				banktypes.ModuleName,
				"denom metadata",
				"denom %s has supply but no metadata\n",
				coin.Denom,
			)
		}
	}

	return msg, broken
}

// TestBankInvariants runs all bank invariants
func (s *BankInvariantsTestSuite) TestBankInvariants() {
	// Create some test accounts and balances
	addr1 := sdk.AccAddress([]byte("addr1_______________"))
	addr2 := sdk.AccAddress([]byte("addr2_______________"))

	// Mint coins
	coins := sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000000))
	err := s.app.BankKeeper.MintCoins(s.ctx, "dex", coins)
	s.Require().NoError(err)

	// Distribute to accounts
	err = s.app.BankKeeper.SendCoinsFromModuleToAccount(s.ctx, "dex", addr1, sdk.NewCoins(sdk.NewInt64Coin("upaw", 600000)))
	s.Require().NoError(err)

	err = s.app.BankKeeper.SendCoinsFromModuleToAccount(s.ctx, "dex", addr2, sdk.NewCoins(sdk.NewInt64Coin("upaw", 400000)))
	s.Require().NoError(err)

	// Run invariants
	msg, broken := s.InvariantTotalSupply()
	s.Require().False(broken, msg)

	msg, broken = s.InvariantNonNegativeBalances()
	s.Require().False(broken, msg)

	msg, broken = s.InvariantDenomMetadata()
	s.Require().False(broken, msg)
}

// TestInvariantsAfterTransfers tests invariants hold after multiple transfers
func (s *BankInvariantsTestSuite) TestInvariantsAfterTransfers() {
	// Create test accounts
	accounts := make([]sdk.AccAddress, 10)
	for i := 0; i < 10; i++ {
		accounts[i] = sdk.AccAddress([]byte{byte(i), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	}

	// Mint and distribute initial coins
	totalCoins := sdk.NewCoins(sdk.NewInt64Coin("upaw", 10000000))
	err := s.app.BankKeeper.MintCoins(s.ctx, "dex", totalCoins)
	s.Require().NoError(err)

	// Distribute evenly
	perAccount := sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000000))
	for _, acc := range accounts {
		err = s.app.BankKeeper.SendCoinsFromModuleToAccount(s.ctx, "dex", acc, perAccount)
		s.Require().NoError(err)
	}

	// Perform random transfers
	err = s.app.BankKeeper.SendCoins(s.ctx, accounts[0], accounts[1], sdk.NewCoins(sdk.NewInt64Coin("upaw", 100000)))
	s.Require().NoError(err)

	err = s.app.BankKeeper.SendCoins(s.ctx, accounts[1], accounts[2], sdk.NewCoins(sdk.NewInt64Coin("upaw", 50000)))
	s.Require().NoError(err)

	err = s.app.BankKeeper.SendCoins(s.ctx, accounts[2], accounts[3], sdk.NewCoins(sdk.NewInt64Coin("upaw", 25000)))
	s.Require().NoError(err)

	// Invariants should still hold
	msg, broken := s.InvariantTotalSupply()
	s.Require().False(broken, msg)

	msg, broken = s.InvariantNonNegativeBalances()
	s.Require().False(broken, msg)
}

// TestInvariantsAfterBurnAndMint tests invariants after burn and mint operations
func (s *BankInvariantsTestSuite) TestInvariantsAfterBurnAndMint() {
	// Mint coins
	coins := sdk.NewCoins(sdk.NewInt64Coin("upaw", 5000000))
	err := s.app.BankKeeper.MintCoins(s.ctx, "dex", coins)
	s.Require().NoError(err)

	msg, broken := s.InvariantTotalSupply()
	s.Require().False(broken, msg)

	// Burn some coins
	burnCoins := sdk.NewCoins(sdk.NewInt64Coin("upaw", 2000000))
	err = s.app.BankKeeper.BurnCoins(s.ctx, "dex", burnCoins)
	s.Require().NoError(err)

	msg, broken = s.InvariantTotalSupply()
	s.Require().False(broken, msg)

	msg, broken = s.InvariantNonNegativeBalances()
	s.Require().False(broken, msg)
}

func TestBankInvariantsTestSuite(t *testing.T) {
	suite.Run(t, new(BankInvariantsTestSuite))
}

// BenchmarkBankInvariants benchmarks invariant checking performance
func BenchmarkBankInvariants(b *testing.B) {
	db := dbm.NewMemDB()
	encCfg := app.MakeEncodingConfig()

	pawApp := app.NewApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		map[int64]bool{},
		app.DefaultNodeHome,
		0,
		encCfg,
		app.GetEnabledProposals(),
		baseapp.SetChainID("paw-test-1"),
	)

	ctx := pawApp.BaseApp.NewContext(false, tmproto.Header{
		ChainID: "paw-test-1",
		Height:  1,
	})

	// Create test data
	for i := 0; i < 100; i++ {
		addr := sdk.AccAddress([]byte{byte(i), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		coins := sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000000))

		err := pawApp.BankKeeper.MintCoins(ctx, "dex", coins)
		require.NoError(b, err)

		err = pawApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, "dex", addr, coins)
		require.NoError(b, err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Run total supply invariant
		totalSupply := pawApp.BankKeeper.GetSupply(ctx, "upaw")
		var accountTotal sdk.Int
		accountTotal = sdk.ZeroInt()

		pawApp.AccountKeeper.IterateAccounts(ctx, func(account authtypes.AccountI) bool {
			balance := pawApp.BankKeeper.GetBalance(ctx, account.GetAddress(), "upaw")
			accountTotal = accountTotal.Add(balance.Amount)
			return false
		})

		_ = totalSupply.Amount.Equal(accountTotal)
	}
}
