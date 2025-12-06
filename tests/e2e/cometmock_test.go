//go:build integration
// +build integration

package e2e_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/testutil/cometmock"
	dextypes "github.com/paw-chain/paw/x/dex/types"
)

type CometMockE2ETestSuite struct {
	suite.Suite
	app *cometmock.CometMockApp
}

func (s *CometMockE2ETestSuite) SetupTest() {
	config := cometmock.DefaultCometMockConfig()
	s.app = cometmock.SetupCometMock(s.T(), config)
}

func (s *CometMockE2ETestSuite) TearDownTest() {
	// Cleanup if needed
}

// TestBasicBlockProduction verifies basic block production
func (s *CometMockE2ETestSuite) TestBasicBlockProduction() {
	initialHeight := s.app.Height()

	// Produce 10 blocks
	s.app.NextBlocks(10)

	s.Require().Equal(initialHeight+10, s.app.Height())
}

// TestBlockTime verifies block time advances correctly
func (s *CometMockE2ETestSuite) TestBlockTime() {
	initialTime := s.app.Time()

	// Advance one block
	s.app.NextBlock()

	newTime := s.app.Time()
	s.Require().True(newTime.After(initialTime))
}

// TestBankTransfer tests bank module transfer with CometMock
func (s *CometMockE2ETestSuite) TestBankTransfer() {
	ctx := s.app.Context()

	// Create test accounts
	addr1 := sdk.AccAddress([]byte("addr1_______________"))
	addr2 := sdk.AccAddress([]byte("addr2_______________"))

	// Fund addr1
	coins := sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000000))
	err := s.app.BankKeeper.MintCoins(ctx, dextypes.ModuleName, coins)
	s.Require().NoError(err)

	err = s.app.BankKeeper.SendCoinsFromModuleToAccount(ctx, dextypes.ModuleName, addr1, coins)
	s.Require().NoError(err)

	// Verify balance
	balance := s.app.BankKeeper.GetBalance(ctx, addr1, "upaw")
	s.Require().Equal(sdk.NewInt64Coin("upaw", 1000000), balance)

	// Transfer coins
	transferAmount := sdk.NewCoins(sdk.NewInt64Coin("upaw", 500000))
	err = s.app.BankKeeper.SendCoins(ctx, addr1, addr2, transferAmount)
	s.Require().NoError(err)

	// Advance block to commit
	s.app.NextBlock()

	// Verify final balances
	ctx = s.app.Context()
	balance1 := s.app.BankKeeper.GetBalance(ctx, addr1, "upaw")
	balance2 := s.app.BankKeeper.GetBalance(ctx, addr2, "upaw")

	s.Require().Equal(sdk.NewInt64Coin("upaw", 500000), balance1)
	s.Require().Equal(sdk.NewInt64Coin("upaw", 500000), balance2)
}

// TestDEXPoolCreation tests DEX pool creation with CometMock
func (s *CometMockE2ETestSuite) TestDEXPoolCreation() {
	ctx := s.app.Context()

	// Create test account
	creator := sdk.AccAddress([]byte("creator_____________"))

	// Fund creator
	fundingCoins := sdk.NewCoins(
		sdk.NewInt64Coin("upaw", 10000000),
		sdk.NewInt64Coin("uusdc", 10000000),
	)

	err := s.app.BankKeeper.MintCoins(ctx, dextypes.ModuleName, fundingCoins)
	s.Require().NoError(err)

	err = s.app.BankKeeper.SendCoinsFromModuleToAccount(ctx, dextypes.ModuleName, creator, fundingCoins)
	s.Require().NoError(err)

	// Create pool
	pool, err := s.app.DEXKeeper.CreatePool(ctx, creator, "upaw", "uusdc", sdkmath.NewInt(5_000_000), sdkmath.NewInt(5_000_000))
	s.Require().NoError(err)
	s.Require().NotNil(pool)

	s.app.NextBlock()

	ctx = s.app.Context()
	storedPool, err := s.app.DEXKeeper.GetPool(ctx, pool.Id)
	s.Require().NoError(err)
	s.Require().Equal(pool.Id, storedPool.Id)

	// Fund trader for swap
	trader := sdk.AccAddress([]byte("trader______________"))
	swapIn := sdkmath.NewInt(500_000)
	swapCoins := sdk.NewCoins(sdk.NewCoin("upaw", swapIn))
	s.Require().NoError(s.app.BankKeeper.MintCoins(ctx, dextypes.ModuleName, swapCoins))
	s.Require().NoError(s.app.BankKeeper.SendCoinsFromModuleToAccount(ctx, dextypes.ModuleName, trader, swapCoins))

	// Execute swap atomically
	outAmt, err := s.app.DEXKeeper.ExecuteSwapSecure(ctx, trader, pool.Id, "upaw", "uusdc", swapIn, sdkmath.NewInt(1))
	s.Require().NoError(err)
	s.Require().True(outAmt.IsPositive())

	// Verify balances and pool reserves moved
	s.app.NextBlock()
	ctx = s.app.Context()
	traderUSDC := s.app.BankKeeper.GetBalance(ctx, trader, "uusdc")
	s.Require().True(traderUSDC.Amount.GTE(outAmt))
	updatedPool, err := s.app.DEXKeeper.GetPool(ctx, pool.Id)
	s.Require().NoError(err)
	s.Require().True(updatedPool.ReserveA.Add(updatedPool.ReserveB).GT(storedPool.ReserveA.Add(storedPool.ReserveB).Sub(swapIn)))

	s.app.NextBlock()

	// Verify pool was created
	s.Require().True(updatedPool.Id > 0)
}

// TestMultiBlockOperations tests operations across multiple blocks
func (s *CometMockE2ETestSuite) TestMultiBlockOperations() {
	ctx := s.app.Context()

	// Create and fund test account
	addr := sdk.AccAddress([]byte("test________________"))
	coins := sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000000))

	err := s.app.BankKeeper.MintCoins(ctx, dextypes.ModuleName, coins)
	s.Require().NoError(err)

	err = s.app.BankKeeper.SendCoinsFromModuleToAccount(ctx, dextypes.ModuleName, addr, coins)
	s.Require().NoError(err)

	// Perform operations over multiple blocks
	for i := 0; i < 5; i++ {
		// Each block, verify balance persists
		ctx = s.app.Context()
		balance := s.app.BankKeeper.GetBalance(ctx, addr, "upaw")
		s.Require().Equal(coins[0], balance)

		s.app.NextBlock()
	}
}

// TestFastBlockProduction tests rapid block production
func (s *CometMockE2ETestSuite) TestFastBlockProduction() {
	start := time.Now()

	// Produce 1000 blocks
	s.app.NextBlocks(1000)

	duration := time.Since(start)

	// CometMock should be much faster than real consensus
	// 1000 blocks should take less than 5 seconds
	s.Require().Less(duration, 5*time.Second)
	s.Require().Equal(int64(1001), s.app.Height())
}

// TestQueryDuringBlockProduction tests querying state during block production
func (s *CometMockE2ETestSuite) TestQueryDuringBlockProduction() {
	ctx := s.app.Context()

	// Create test account
	addr := sdk.AccAddress([]byte("query_______________"))
	coins := sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000000))

	err := s.app.BankKeeper.MintCoins(ctx, dextypes.ModuleName, coins)
	s.Require().NoError(err)

	err = s.app.BankKeeper.SendCoinsFromModuleToAccount(ctx, dextypes.ModuleName, addr, coins)
	s.Require().NoError(err)

	s.app.NextBlock()
}

func TestCometMockE2ETestSuite(t *testing.T) {
	// Only run if USE_COMETMOCK env var is set
	if os.Getenv("USE_COMETMOCK") != "true" {
		t.Skip("Skipping CometMock E2E tests - set USE_COMETMOCK=true to run")
	}

	suite.Run(t, new(CometMockE2ETestSuite))
}

// BenchmarkCometMockBlockProduction benchmarks block production speed
func BenchmarkCometMockBlockProduction(b *testing.B) {
	if os.Getenv("USE_COMETMOCK") != "true" {
		b.Skip("Skipping CometMock benchmark - set USE_COMETMOCK=true to run")
	}

	config := cometmock.DefaultCometMockConfig()
	app := cometmock.SetupCometMock(&testing.T{}, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.NextBlock()
	}
}
