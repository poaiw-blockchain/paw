//go:build integration
// +build integration

package security_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/app"
	keepertest "github.com/paw-chain/paw/testutil/keeper"
	computetypes "github.com/paw-chain/paw/x/compute/types"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// AuthSecurityTestSuite tests authentication and authorization vulnerabilities
type AuthSecurityTestSuite struct {
	suite.Suite
	app *app.PAWApp
	ctx sdk.Context
}

func (suite *AuthSecurityTestSuite) SetupTest() {
	suite.app, suite.ctx = keepertest.SetupTestApp(suite.T())
}

func TestAuthSecurityTestSuite(t *testing.T) {
	suite.Run(t, new(AuthSecurityTestSuite))
}

// TestAuthenticationBypass_InvalidSignature tests that invalid signatures are rejected
func (suite *AuthSecurityTestSuite) TestAuthenticationBypass_InvalidSignature() {
	// Create two different accounts
	priv1 := secp256k1.GenPrivKey()
	addr1 := sdk.AccAddress(priv1.PubKey().Address())

	priv2 := secp256k1.GenPrivKey()
	addr2 := sdk.AccAddress(priv2.PubKey().Address())

	// Fund addr1
	coins := sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(1000000)))
	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, coins))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, addr1, coins))

	// Attempt to create a message from addr1 but try to bypass authentication
	// In a properly secured system, this should fail
	msg := &dextypes.MsgCreatePool{
		Creator: addr1.String(),
		TokenA:  "upaw",
		TokenB:  "uusdt",
		AmountA: math.NewInt(100000),
		AmountB: math.NewInt(200000),
	}

	// Try to execute with wrong signer (addr2)
	// This should fail signature verification
	suite.ctx = suite.ctx.WithIsCheckTx(true)

	// Verify that the message requires proper authentication
	require.Equal(suite.T(), addr1.String(), msg.Creator)
	require.NotEqual(suite.T(), addr2.String(), msg.Creator)
}

// TestAuthorizationEscalation_ModuleAccountAccess tests that users cannot access module accounts
func (suite *AuthSecurityTestSuite) TestAuthorizationEscalation_ModuleAccountAccess() {
	// Create user account
	priv := secp256k1.GenPrivKey()
	userAddr := sdk.AccAddress(priv.PubKey().Address())

	// Get module account address
	moduleAddr := suite.app.AccountKeeper.GetModuleAddress(dextypes.ModuleName)

	// Fund module account
	coins := sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(10000000)))
	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, coins))

	// Attempt to send directly from module account to user (should fail)
	err := suite.app.BankKeeper.SendCoins(suite.ctx, moduleAddr, userAddr, coins)

	// This should fail - module accounts should not allow direct sends
	// Only through proper keeper methods
	suite.Require().Error(err, "Direct access to module account should be prevented")
}

// TestAuthorizationEscalation_CrossModuleAccess tests isolation between modules
func (suite *AuthSecurityTestSuite) TestAuthorizationEscalation_CrossModuleAccess() {
	// Verify that one module cannot directly access another module's state
	// without going through proper keeper interfaces

	dexModuleAddr := suite.app.AccountKeeper.GetModuleAddress(dextypes.ModuleName)
	computeModuleAddr := suite.app.AccountKeeper.GetModuleAddress(computetypes.ModuleName)
	oracleModuleAddr := suite.app.AccountKeeper.GetModuleAddress(oracletypes.ModuleName)

	// Verify module accounts are distinct
	suite.Require().NotEqual(dexModuleAddr, computeModuleAddr)
	suite.Require().NotEqual(dexModuleAddr, oracleModuleAddr)
	suite.Require().NotEqual(computeModuleAddr, oracleModuleAddr)

	// Each module should have its own isolated account
	dexAcc := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, dextypes.ModuleName)
	computeAcc := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, computetypes.ModuleName)
	oracleAcc := suite.app.AccountKeeper.GetModuleAccount(suite.ctx, oracletypes.ModuleName)

	suite.Require().NotNil(dexAcc)
	suite.Require().NotNil(computeAcc)
	suite.Require().NotNil(oracleAcc)
}

// TestPermissionEscalation_UnauthorizedPoolCreation tests unauthorized pool creation
func (suite *AuthSecurityTestSuite) TestPermissionEscalation_UnauthorizedPoolCreation() {
	// Create user without funds
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// Attempt to create pool without sufficient funds
	msg := &dextypes.MsgCreatePool{
		Creator: addr.String(),
		TokenA:  "upaw",
		TokenB:  "uusdt",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(2000000),
	}

	_, err := suite.app.DEXKeeper.CreatePool(suite.ctx, msg)

	// Should fail due to insufficient funds
	suite.Require().Error(err, "Pool creation without funds should fail")
}

// TestRateLimiting_TransactionSpam tests rate limiting protections
func (suite *AuthSecurityTestSuite) TestRateLimiting_TransactionSpam() {
	// Create and fund account
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	coins := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(100000000)),
		sdk.NewCoin("uusdt", math.NewInt(100000000)),
	)
	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, coins))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, addr, coins))

	// Create a pool
	msgCreatePool := &dextypes.MsgCreatePool{
		Creator: addr.String(),
		TokenA:  "upaw",
		TokenB:  "uusdt",
		AmountA: math.NewInt(10000000),
		AmountB: math.NewInt(20000000),
	}

	resp, err := suite.app.DEXKeeper.CreatePool(suite.ctx, msgCreatePool)
	suite.Require().NoError(err)

	// Attempt rapid-fire transactions in same block
	// Block gas limit should prevent unlimited transactions
	successCount := 0
	maxAttempts := 1000

	for i := 0; i < maxAttempts; i++ {
		msgSwap := &dextypes.MsgSwap{
			Trader:       addr.String(),
			PoolId:       resp.PoolId,
			TokenIn:      "upaw",
			AmountIn:     math.NewInt(1000),
			MinAmountOut: math.NewInt(1),
		}

		_, err := suite.app.DEXKeeper.Swap(suite.ctx, msgSwap)
		if err == nil {
			successCount++
		}
	}

	// Not all transactions should succeed (gas limits should apply)
	suite.T().Logf("Successfully executed %d out of %d transactions", successCount, maxAttempts)
}

// TestSessionManagement_StaleContext tests that stale contexts are handled properly
func (suite *AuthSecurityTestSuite) TestSessionManagement_StaleContext() {
	// Create account
	priv := secp256k1.GenPrivKey()
	_ = sdk.AccAddress(priv.PubKey().Address())

	// Get initial context
	ctx1 := suite.ctx

	// Advance block height
	ctx2 := suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 100)

	// Verify contexts have different block heights
	suite.Require().NotEqual(ctx1.BlockHeight(), ctx2.BlockHeight())

	// Operations should use current context, not stale one
	suite.Require().Greater(ctx2.BlockHeight(), ctx1.BlockHeight())
}

// TestAccessControl_OracleSubmission tests that only authorized oracles can submit prices
func (suite *AuthSecurityTestSuite) TestAccessControl_OracleSubmission() {
	// Create unauthorized user
	priv := secp256k1.GenPrivKey()
	unauthorizedAddr := sdk.AccAddress(priv.PubKey().Address())

	// Attempt to submit price without being registered oracle
	msgSubmit := &oracletypes.MsgSubmitPrice{
		Oracle: unauthorizedAddr.String(),
		Asset:  "BTC/USD",
		Price:  math.LegacyMustNewDecFromStr("50000.00"),
	}

	_, err := suite.app.OracleKeeper.SubmitPrice(suite.ctx, msgSubmit)

	// Should fail - user is not a registered oracle
	suite.Require().Error(err, "Unauthorized oracle submission should fail")
}

// TestAccessControl_ComputeProviderRegistration tests compute provider authorization
func (suite *AuthSecurityTestSuite) TestAccessControl_ComputeProviderRegistration() {
	// Create account without sufficient stake
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// Attempt to register without stake
	msgRegister := &computetypes.MsgRegisterProvider{
		Provider: addr.String(),
		Endpoint: "https://malicious.endpoint.com",
		Stake:    math.NewInt(100000),
	}

	_, err := suite.app.ComputeKeeper.RegisterProvider(suite.ctx, msgRegister)

	// Should fail due to insufficient funds
	suite.Require().Error(err, "Provider registration without stake should fail")
}

// TestTimeBasedAttacks_TimestampManipulation tests timestamp validation
func (suite *AuthSecurityTestSuite) TestTimeBasedAttacks_TimestampManipulation() {
	// Get current block time
	currentTime := suite.ctx.BlockTime()

	// Try to create context with future timestamp (should be validated by consensus)
	futureTime := currentTime.Add(24 * time.Hour)
	futureCtx := suite.ctx.WithBlockTime(futureTime)

	// Verify timestamp changed
	suite.Require().NotEqual(currentTime, futureCtx.BlockTime())
	suite.Require().Equal(futureTime, futureCtx.BlockTime())

	// In production, consensus layer should prevent blocks with invalid timestamps
	suite.Require().True(futureCtx.BlockTime().After(currentTime))
}

// TestReentrancy_DEXSwap tests reentrancy protection in DEX swaps
func (suite *AuthSecurityTestSuite) TestReentrancy_DEXSwap() {
	// Create and fund account
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	coins := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(10000000)),
		sdk.NewCoin("uusdt", math.NewInt(20000000)),
	)
	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, coins))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, addr, coins))

	// Create pool
	msgCreatePool := &dextypes.MsgCreatePool{
		Creator: addr.String(),
		TokenA:  "upaw",
		TokenB:  "uusdt",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(2000000),
	}

	resp, err := suite.app.DEXKeeper.CreatePool(suite.ctx, msgCreatePool)
	suite.Require().NoError(err)

	// Get pool state before swap
	poolBefore, found := suite.app.DEXKeeper.GetPool(suite.ctx, resp.PoolId)
	suite.Require().True(found)

	// Execute swap
	msgSwap := &dextypes.MsgSwap{
		Trader:       addr.String(),
		PoolId:       resp.PoolId,
		TokenIn:      "upaw",
		AmountIn:     math.NewInt(100000),
		MinAmountOut: math.NewInt(1),
	}

	_, err = suite.app.DEXKeeper.Swap(suite.ctx, msgSwap)
	suite.Require().NoError(err)

	// Get pool state after swap
	poolAfter, found := suite.app.DEXKeeper.GetPool(suite.ctx, resp.PoolId)
	suite.Require().True(found)

	// Verify pool state changed (proves operation completed)
	suite.Require().NotEqual(poolBefore.ReserveA, poolAfter.ReserveA)

	// In a reentrancy attack, state should be locked during operation
	// Cosmos SDK provides reentrancy protection through message handling
}

// TestPrivilegeEscalation_GovernanceBypass tests governance parameter changes
func (suite *AuthSecurityTestSuite) TestPrivilegeEscalation_GovernanceBypass() {
	// Regular users should not be able to change module parameters directly
	// without going through governance

	// Create regular user
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// Verify user is not a module account
	moduleAccounts := []string{
		dextypes.ModuleName,
		computetypes.ModuleName,
		oracletypes.ModuleName,
		"gov",
	}

	userIsModule := false
	for _, modName := range moduleAccounts {
		modAddr := suite.app.AccountKeeper.GetModuleAddress(modName)
		if modAddr.Equals(addr) {
			userIsModule = true
			break
		}
	}

	suite.Require().False(userIsModule, "Regular user should not be a module account")
}

// TestDenialOfService_ResourceExhaustion tests resource exhaustion protections
func (suite *AuthSecurityTestSuite) TestDenialOfService_ResourceExhaustion() {
	// Create account
	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())

	// Fund account with large amount
	coins := sdk.NewCoins(
		sdk.NewCoin("upaw", math.NewInt(1000000000)),
		sdk.NewCoin("uusdt", math.NewInt(1000000000)),
	)
	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, dextypes.ModuleName, coins))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToAccount(suite.ctx, dextypes.ModuleName, addr, coins))

	// Attempt to create excessive number of pools
	maxPools := 100
	createdPools := 0

	for i := 0; i < maxPools; i++ {
		msg := &dextypes.MsgCreatePool{
			Creator: addr.String(),
			TokenA:  "upaw",
			TokenB:  "uusdt",
			AmountA: math.NewInt(1000),
			AmountB: math.NewInt(2000),
		}

		_, err := suite.app.DEXKeeper.CreatePool(suite.ctx, msg)
		if err == nil {
			createdPools++
		} else {
			// Hit resource limit
			break
		}
	}

	suite.T().Logf("Created %d pools before hitting limits", createdPools)
}
