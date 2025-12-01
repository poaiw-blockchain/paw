//go:build integration
// +build integration

package invariants

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/paw-chain/paw/app"
)

// BankInvariantTestSuite tests bank module invariants to ensure state consistency
type BankInvariantTestSuite struct {
	suite.Suite
	app *app.PAWApp
	ctx sdk.Context
}

// SetupTest initializes the test environment before each test
func (suite *BankInvariantTestSuite) SetupTest() {
	suite.app = app.Setup(suite.T(), false)
	suite.ctx = suite.app.BaseApp.NewContext(false)
}

// TestTotalSupplyInvariant verifies that the sum of all account balances equals total supply
// This is a critical invariant - if violated, it indicates token creation/destruction bugs
func (suite *BankInvariantTestSuite) TestTotalSupplyInvariant() {
	// Initialize test accounts with known balances
	testAccounts := []struct {
		address sdk.AccAddress
		coins   sdk.Coins
	}{
		{
			address: sdk.AccAddress([]byte("test_account_1______")),
			coins:   sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(1000000))),
		},
		{
			address: sdk.AccAddress([]byte("test_account_2______")),
			coins:   sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(2000000))),
		},
		{
			address: sdk.AccAddress([]byte("test_account_3______")),
			coins:   sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(3000000))),
		},
	}

	// Fund test accounts
	for _, acc := range testAccounts {
		suite.NoError(suite.app.BankKeeper.MintCoins(suite.ctx, banktypes.ModuleName, acc.coins))
		suite.NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(
			suite.ctx,
			banktypes.ModuleName,
			acc.address,
			acc.coins,
		))
	}

	// Calculate total of all account balances
	totalBalances := sdk.NewCoins()
	suite.app.AccountKeeper.IterateAccounts(suite.ctx, func(account authtypes.AccountI) bool {
		balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, account.GetAddress())
		totalBalances = totalBalances.Add(balances...)
		return false
	})

	// Get total supply from bank keeper
	totalSupply := suite.app.BankKeeper.GetTotalSupply(suite.ctx)

	// Verify invariant: sum of balances must equal total supply
	suite.Require().True(
		totalBalances.IsEqual(totalSupply),
		"Total supply invariant violated: balances=%s, supply=%s",
		totalBalances.String(),
		totalSupply.String(),
	)
}

// TestNonNegativeBalancesInvariant ensures no account has negative balances
// Negative balances should be impossible, but bugs in token operations could cause this
func (suite *BankInvariantTestSuite) TestNonNegativeBalancesInvariant() {
	// Create test accounts
	addr1 := sdk.AccAddress([]byte("test_addr_1_________"))
	addr2 := sdk.AccAddress([]byte("test_addr_2_________"))

	// Fund accounts
	initialCoins := sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(1000000)))
	suite.NoError(suite.app.BankKeeper.MintCoins(suite.ctx, banktypes.ModuleName, initialCoins))
	suite.NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(
		suite.ctx,
		banktypes.ModuleName,
		addr1,
		initialCoins,
	))

	// Perform some transfers
	transferCoins := sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(500000)))
	suite.NoError(suite.app.BankKeeper.SendCoins(suite.ctx, addr1, addr2, transferCoins))

	// Check all accounts have non-negative balances
	suite.app.AccountKeeper.IterateAccounts(suite.ctx, func(account authtypes.AccountI) bool {
		balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, account.GetAddress())
		for _, coin := range balances {
			suite.Require().True(
				!coin.Amount.IsNegative(),
				"Negative balance detected for account %s: %s",
				account.GetAddress().String(),
				coin.String(),
			)
		}
		return false
	})
}

// TestModuleAccountConsistencyInvariant verifies module accounts have correct balances
// Module accounts are special accounts that hold tokens for modules
func (suite *BankInvariantTestSuite) TestModuleAccountConsistencyInvariant() {
	moduleAccounts := []string{
		authtypes.FeeCollectorName,
		banktypes.ModuleName,
	}

	for _, moduleName := range moduleAccounts {
		moduleAddr := suite.app.AccountKeeper.GetModuleAddress(moduleName)
		suite.Require().NotNil(moduleAddr, "Module account not found: %s", moduleName)

		// Get module account
		moduleAcc := suite.app.AccountKeeper.GetAccount(suite.ctx, moduleAddr)
		suite.Require().NotNil(moduleAcc, "Module account not in account keeper: %s", moduleName)

		// Verify module account has correct type
		_, ok := moduleAcc.(authtypes.ModuleAccountI)
		suite.Require().True(ok, "Account is not a module account: %s", moduleName)

		// Verify balances are non-negative
		balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, moduleAddr)
		for _, coin := range balances {
			suite.Require().False(
				coin.Amount.IsNegative(),
				"Module account %s has negative balance: %s",
				moduleName,
				coin.String(),
			)
		}
	}
}

// TestCoinDenomValidationInvariant ensures all coins have valid denominations
// Invalid denoms could cause consensus failures or unexpected behavior
func (suite *BankInvariantTestSuite) TestCoinDenomValidationInvariant() {
	// Valid denomination test
	validCoin := sdk.NewCoin("upaw", math.NewInt(1000))
	suite.NoError(validCoin.Validate())

	// Check all balances in state have valid denoms
	suite.app.AccountKeeper.IterateAccounts(suite.ctx, func(account authtypes.AccountI) bool {
		balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, account.GetAddress())
		suite.NoError(balances.Validate(), "Invalid balances for account %s", account.GetAddress())
		return false
	})

	// Check total supply has valid denoms
	totalSupply := suite.app.BankKeeper.GetTotalSupply(suite.ctx)
	suite.NoError(totalSupply.Validate(), "Invalid total supply denoms")
}

// TestSupplyTracking verifies supply tracking remains accurate after operations
func (suite *BankInvariantTestSuite) TestSupplyTrackingInvariant() {
	denom := "upaw"

	// Get initial supply
	initialSupply := suite.app.BankKeeper.GetSupply(suite.ctx, denom)

	// Mint new coins
	mintAmount := sdk.NewCoins(sdk.NewCoin(denom, math.NewInt(1000000)))
	suite.NoError(suite.app.BankKeeper.MintCoins(suite.ctx, banktypes.ModuleName, mintAmount))

	// Verify supply increased
	afterMintSupply := suite.app.BankKeeper.GetSupply(suite.ctx, denom)
	expectedSupply := initialSupply.Amount.Add(math.NewInt(1000000))
	suite.Require().True(
		afterMintSupply.Amount.Equal(expectedSupply),
		"Supply tracking after mint incorrect: expected=%s, got=%s",
		expectedSupply.String(),
		afterMintSupply.Amount.String(),
	)

	// Burn coins
	burnAmount := sdk.NewCoins(sdk.NewCoin(denom, math.NewInt(500000)))
	suite.NoError(suite.app.BankKeeper.BurnCoins(suite.ctx, banktypes.ModuleName, burnAmount))

	// Verify supply decreased
	afterBurnSupply := suite.app.BankKeeper.GetSupply(suite.ctx, denom)
	expectedSupply = expectedSupply.Sub(math.NewInt(500000))
	suite.Require().True(
		afterBurnSupply.Amount.Equal(expectedSupply),
		"Supply tracking after burn incorrect: expected=%s, got=%s",
		expectedSupply.String(),
		afterBurnSupply.Amount.String(),
	)
}

// TestSendCoinsInvariant verifies sending coins maintains total supply
func (suite *BankInvariantTestSuite) TestSendCoinsInvariant() {
	addr1 := sdk.AccAddress([]byte("sender______________"))
	addr2 := sdk.AccAddress([]byte("recipient___________"))

	// Setup initial balances
	initialCoins := sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(2000000)))
	suite.NoError(suite.app.BankKeeper.MintCoins(suite.ctx, banktypes.ModuleName, initialCoins))
	suite.NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(
		suite.ctx,
		banktypes.ModuleName,
		addr1,
		initialCoins,
	))

	// Record total supply before send
	supplyBefore := suite.app.BankKeeper.GetTotalSupply(suite.ctx)

	// Send coins
	sendCoins := sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(1000000)))
	suite.NoError(suite.app.BankKeeper.SendCoins(suite.ctx, addr1, addr2, sendCoins))

	// Verify total supply unchanged
	supplyAfter := suite.app.BankKeeper.GetTotalSupply(suite.ctx)
	suite.Require().True(
		supplyBefore.IsEqual(supplyAfter),
		"Total supply changed after send: before=%s, after=%s",
		supplyBefore.String(),
		supplyAfter.String(),
	)

	// Verify balances are correct
	balance1 := suite.app.BankKeeper.GetBalance(suite.ctx, addr1, "upaw")
	balance2 := suite.app.BankKeeper.GetBalance(suite.ctx, addr2, "upaw")
	suite.Require().True(
		balance1.Amount.Equal(math.NewInt(1000000)),
		"Sender balance incorrect: expected=1000000, got=%s",
		balance1.Amount.String(),
	)
	suite.Require().True(
		balance2.Amount.Equal(math.NewInt(1000000)),
		"Recipient balance incorrect: expected=1000000, got=%s",
		balance2.Amount.String(),
	)
}

// TestLockedCoinsInvariant verifies locked coins don't affect spendable balances incorrectly
func (suite *BankInvariantTestSuite) TestLockedCoinsInvariant() {
	addr := sdk.AccAddress([]byte("locked_test_________"))

	// Fund account
	totalCoins := sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(1000000)))
	suite.NoError(suite.app.BankKeeper.MintCoins(suite.ctx, banktypes.ModuleName, totalCoins))
	suite.NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(
		suite.ctx,
		banktypes.ModuleName,
		addr,
		totalCoins,
	))

	// Get spendable balance (should equal total for regular accounts)
	spendable := suite.app.BankKeeper.SpendableCoins(suite.ctx, addr)
	total := suite.app.BankKeeper.GetAllBalances(suite.ctx, addr)

	suite.Require().True(
		spendable.IsEqual(total),
		"Spendable coins don't match total for unlocked account: spendable=%s, total=%s",
		spendable.String(),
		total.String(),
	)
}

// TestInvariantAfterMultipleOperations runs multiple operations and checks invariants
func (suite *BankInvariantTestSuite) TestInvariantAfterMultipleOperations() {
	addresses := make([]sdk.AccAddress, 10)
	for i := 0; i < 10; i++ {
		addresses[i] = sdk.AccAddress([]byte{byte(i), 't', 'e', 's', 't', '_', 'a', 'd', 'd', 'r', 'e', 's', 's', '_', '_', '_', '_', '_', '_', '_'})
	}

	// Fund all accounts
	for _, addr := range addresses {
		coins := sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(1000000)))
		suite.NoError(suite.app.BankKeeper.MintCoins(suite.ctx, banktypes.ModuleName, coins))
		suite.NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(
			suite.ctx,
			banktypes.ModuleName,
			addr,
			coins,
		))
	}

	// Perform random transfers
	for i := 0; i < 20; i++ {
		from := addresses[i%10]
		to := addresses[(i+1)%10]
		amount := sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(int64(1000*(i+1)))))

		// Check if sender has enough balance
		fromBalance := suite.app.BankKeeper.GetBalance(suite.ctx, from, "upaw")
		if fromBalance.Amount.GTE(amount[0].Amount) {
			suite.NoError(suite.app.BankKeeper.SendCoins(suite.ctx, from, to, amount))
		}
	}

	// Check total supply invariant after all operations
	totalBalances := sdk.NewCoins()
	suite.app.AccountKeeper.IterateAccounts(suite.ctx, func(account authtypes.AccountI) bool {
		balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, account.GetAddress())
		totalBalances = totalBalances.Add(balances...)
		return false
	})

	totalSupply := suite.app.BankKeeper.GetTotalSupply(suite.ctx)
	suite.Require().True(
		totalBalances.IsEqual(totalSupply),
		"Total supply invariant violated after multiple operations: balances=%s, supply=%s",
		totalBalances.String(),
		totalSupply.String(),
	)

	// Check all balances are non-negative
	suite.app.AccountKeeper.IterateAccounts(suite.ctx, func(account authtypes.AccountI) bool {
		balances := suite.app.BankKeeper.GetAllBalances(suite.ctx, account.GetAddress())
		for _, coin := range balances {
			suite.Require().False(
				coin.Amount.IsNegative(),
				"Negative balance after operations for %s: %s",
				account.GetAddress().String(),
				coin.String(),
			)
		}
		return false
	})
}

func TestBankInvariantTestSuite(t *testing.T) {
	suite.Run(t, new(BankInvariantTestSuite))
}
