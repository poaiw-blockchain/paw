package invariants

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/paw-chain/paw/app"
)

// StakingInvariantTestSuite tests staking module invariants
// Critical for validator set consistency and bonded token tracking
type StakingInvariantTestSuite struct {
	suite.Suite
	app *app.PAWApp
	ctx sdk.Context
}

// SetupTest initializes the test environment before each test
func (suite *StakingInvariantTestSuite) SetupTest() {
	suite.app = app.Setup(suite.T(), false)
	suite.ctx = suite.app.BaseApp.NewContext(false)
}

// TestBondedTokensConsistency verifies bonded tokens match sum of bonded validators' tokens
func (suite *StakingInvariantTestSuite) TestBondedTokensConsistency() {
	// Get all validators
	var bondedTokens math.Int = math.ZeroInt()

	err := suite.app.StakingKeeper.IterateBondedValidatorsByPower(suite.ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		if validator.IsBonded() {
			bondedTokens = bondedTokens.Add(validator.GetTokens())
		}
		return false
	})
	suite.NoError(err)

	// Get bonded pool balance
	bondedPool := suite.app.StakingKeeper.GetBondedPool(suite.ctx)
	bondedPoolBalance := suite.app.BankKeeper.GetBalance(
		suite.ctx,
		bondedPool.GetAddress(),
		sdk.DefaultBondDenom,
	)

	// Bonded tokens should equal bonded pool balance
	suite.Require().True(
		bondedTokens.Equal(bondedPoolBalance.Amount),
		"Bonded tokens mismatch: validators=%s, pool=%s",
		bondedTokens.String(),
		bondedPoolBalance.Amount.String(),
	)
}

// TestDelegationSharesSum verifies delegation shares sum to validator shares
func (suite *StakingInvariantTestSuite) TestDelegationSharesSum() {
	// Get all validators
	var allValidators []stakingtypes.Validator
	err := suite.app.StakingKeeper.IterateValidators(suite.ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		val, ok := validator.(stakingtypes.Validator)
		suite.Require().True(ok)
		allValidators = append(allValidators, val)
		return false
	})
	suite.NoError(err)

	// For each validator, sum delegator shares
	for _, validator := range allValidators {
		valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
		suite.NoError(err)

		var totalDelegatorShares math.LegacyDec = math.LegacyZeroDec()

		delegations, err := suite.app.StakingKeeper.GetValidatorDelegations(suite.ctx, valAddr)
		suite.NoError(err)

		for _, delegation := range delegations {
			totalDelegatorShares = totalDelegatorShares.Add(delegation.Shares)
		}

		// Total delegator shares should equal validator's DelegatorShares
		suite.Require().True(
			totalDelegatorShares.Equal(validator.DelegatorShares),
			"Validator %s delegation shares mismatch: sum=%s, validator=%s",
			validator.OperatorAddress,
			totalDelegatorShares.String(),
			validator.DelegatorShares.String(),
		)
	}
}

// TestUnbondingQueueIntegrity verifies unbonding queue entries are valid
func (suite *StakingInvariantTestSuite) TestUnbondingQueueIntegrity() {
	// Get all unbonding delegations
	var allUnbondingDelegations []stakingtypes.UnbondingDelegation
	err := suite.app.StakingKeeper.IterateUnbondingDelegations(suite.ctx, func(index int64, ubd stakingtypes.UnbondingDelegation) (stop bool) {
		allUnbondingDelegations = append(allUnbondingDelegations, ubd)
		return false
	})
	suite.NoError(err)

	currentTime := suite.ctx.BlockTime()

	for _, ubd := range allUnbondingDelegations {
		// Check all entries
		for _, entry := range ubd.Entries {
			// Completion time should be in the future or equal to current time
			suite.Require().True(
				entry.CompletionTime.After(currentTime) || entry.CompletionTime.Equal(currentTime),
				"Unbonding entry has completion time in past: %s < %s",
				entry.CompletionTime.String(),
				currentTime.String(),
			)

			// Balance should be positive
			suite.Require().True(
				entry.Balance.GT(math.ZeroInt()),
				"Unbonding entry has non-positive balance: %s",
				entry.Balance.String(),
			)

			// Initial balance should be >= balance
			suite.Require().True(
				entry.InitialBalance.GTE(entry.Balance),
				"Unbonding initial balance less than current balance: initial=%s, current=%s",
				entry.InitialBalance.String(),
				entry.Balance.String(),
			)
		}
	}
}

// TestValidatorPowerConsistency verifies validator power equals tokens / power reduction
func (suite *StakingInvariantTestSuite) TestValidatorPowerConsistency() {
	powerReduction := stakingtypes.DefaultPowerReduction

	err := suite.app.StakingKeeper.IterateValidators(suite.ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		expectedPower := validator.GetTokens().Quo(powerReduction).Int64()
		actualPower := validator.GetConsensusPower(powerReduction)

		suite.Require().Equal(
			expectedPower,
			actualPower,
			"Validator %s power mismatch: expected=%d, actual=%d, tokens=%s",
			validator.GetOperator().String(),
			expectedPower,
			actualPower,
			validator.GetTokens().String(),
		)
		return false
	})
	suite.NoError(err)
}

// TestNonNegativeTokens ensures all validators have non-negative tokens
func (suite *StakingInvariantTestSuite) TestNonNegativeTokens() {
	err := suite.app.StakingKeeper.IterateValidators(suite.ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		suite.Require().False(
			validator.GetTokens().IsNegative(),
			"Validator %s has negative tokens: %s",
			validator.GetOperator().String(),
			validator.GetTokens().String(),
		)

		suite.Require().False(
			validator.GetDelegatorShares().IsNegative(),
			"Validator %s has negative delegator shares: %s",
			validator.GetOperator().String(),
			validator.GetDelegatorShares().String(),
		)
		return false
	})
	suite.NoError(err)
}

// TestDelegationAmountsPositive ensures all delegations have positive amounts
func (suite *StakingInvariantTestSuite) TestDelegationAmountsPositive() {
	err := suite.app.StakingKeeper.IterateAllDelegations(suite.ctx, func(delegation stakingtypes.Delegation) (stop bool) {
		suite.Require().True(
			delegation.Shares.GT(math.LegacyZeroDec()),
			"Delegation has non-positive shares: delegator=%s, validator=%s, shares=%s",
			delegation.DelegatorAddress,
			delegation.ValidatorAddress,
			delegation.Shares.String(),
		)
		return false
	})
	suite.NoError(err)
}

// TestNotBondedPoolConsistency verifies not-bonded pool balance matches unbonding tokens
func (suite *StakingInvariantTestSuite) TestNotBondedPoolConsistency() {
	// Sum tokens from unbonding validators
	var unbondingTokens math.Int = math.ZeroInt()

	err := suite.app.StakingKeeper.IterateValidators(suite.ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		if validator.IsUnbonding() {
			unbondingTokens = unbondingTokens.Add(validator.GetTokens())
		}
		return false
	})
	suite.NoError(err)

	// Add tokens from unbonding delegations
	err = suite.app.StakingKeeper.IterateUnbondingDelegations(suite.ctx, func(index int64, ubd stakingtypes.UnbondingDelegation) (stop bool) {
		for _, entry := range ubd.Entries {
			unbondingTokens = unbondingTokens.Add(entry.Balance)
		}
		return false
	})
	suite.NoError(err)

	// Get not-bonded pool balance
	notBondedPool := suite.app.StakingKeeper.GetNotBondedPool(suite.ctx)
	notBondedBalance := suite.app.BankKeeper.GetBalance(
		suite.ctx,
		notBondedPool.GetAddress(),
		sdk.DefaultBondDenom,
	)

	// Not-bonded pool should contain at least unbonding tokens
	// (may be more due to unbonded validators)
	suite.Require().True(
		notBondedBalance.Amount.GTE(unbondingTokens),
		"Not-bonded pool has less than unbonding tokens: pool=%s, unbonding=%s",
		notBondedBalance.Amount.String(),
		unbondingTokens.String(),
	)
}

// TestRedelegationIntegrity verifies redelegation entries are valid
func (suite *StakingInvariantTestSuite) TestRedelegationIntegrity() {
	err := suite.app.StakingKeeper.IterateRedelegations(suite.ctx, func(index int64, red stakingtypes.Redelegation) (stop bool) {
		currentTime := suite.ctx.BlockTime()

		for _, entry := range red.Entries {
			// Completion time should be in future
			suite.Require().True(
				entry.CompletionTime.After(currentTime) || entry.CompletionTime.Equal(currentTime),
				"Redelegation completion time in past: %s < %s",
				entry.CompletionTime.String(),
				currentTime.String(),
			)

			// Shares should be positive
			suite.Require().True(
				entry.SharesDst.GT(math.LegacyZeroDec()),
				"Redelegation has non-positive destination shares: %s",
				entry.SharesDst.String(),
			)

			// Initial balance should be positive
			suite.Require().True(
				entry.InitialBalance.GT(math.ZeroInt()),
				"Redelegation has non-positive initial balance: %s",
				entry.InitialBalance.String(),
			)
		}
		return false
	})
	suite.NoError(err)
}

// TestValidatorStatusConsistency ensures validator status matches bonding state
func (suite *StakingInvariantTestSuite) TestValidatorStatusConsistency() {
	err := suite.app.StakingKeeper.IterateValidators(suite.ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		status := validator.GetStatus()

		if validator.IsBonded() {
			suite.Require().Equal(
				stakingtypes.Bonded,
				status,
				"Bonded validator has wrong status: %s",
				status.String(),
			)
		}

		if validator.IsUnbonding() {
			suite.Require().Equal(
				stakingtypes.Unbonding,
				status,
				"Unbonding validator has wrong status: %s",
				status.String(),
			)
		}

		if validator.IsUnbonded() {
			suite.Require().Equal(
				stakingtypes.Unbonded,
				status,
				"Unbonded validator has wrong status: %s",
				status.String(),
			)
		}

		return false
	})
	suite.NoError(err)
}

// TestJailedValidatorsNotBonded ensures jailed validators are not bonded
func (suite *StakingInvariantTestSuite) TestJailedValidatorsNotBonded() {
	err := suite.app.StakingKeeper.IterateValidators(suite.ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		if validator.IsJailed() {
			suite.Require().False(
				validator.IsBonded(),
				"Jailed validator is bonded: %s",
				validator.GetOperator().String(),
			)
		}
		return false
	})
	suite.NoError(err)
}

// TestTotalSupplyConsistency verifies bonded + not-bonded pools don't exceed supply
func (suite *StakingInvariantTestSuite) TestTotalSupplyConsistency() {
	bondDenom := sdk.DefaultBondDenom

	// Get bonded pool balance
	bondedPool := suite.app.StakingKeeper.GetBondedPool(suite.ctx)
	bondedBalance := suite.app.BankKeeper.GetBalance(suite.ctx, bondedPool.GetAddress(), bondDenom)

	// Get not-bonded pool balance
	notBondedPool := suite.app.StakingKeeper.GetNotBondedPool(suite.ctx)
	notBondedBalance := suite.app.BankKeeper.GetBalance(suite.ctx, notBondedPool.GetAddress(), bondDenom)

	// Get total supply
	totalSupply := suite.app.BankKeeper.GetSupply(suite.ctx, bondDenom)

	// Staking pools should not exceed total supply
	stakingTotal := bondedBalance.Amount.Add(notBondedBalance.Amount)
	suite.Require().True(
		stakingTotal.LTE(totalSupply.Amount),
		"Staking pools exceed total supply: staking=%s, supply=%s",
		stakingTotal.String(),
		totalSupply.Amount.String(),
	)
}

// TestUnbondingDelegationBalances verifies unbonding delegation balances are tracked correctly
func (suite *StakingInvariantTestSuite) TestUnbondingDelegationBalances() {
	err := suite.app.StakingKeeper.IterateUnbondingDelegations(suite.ctx, func(index int64, ubd stakingtypes.UnbondingDelegation) (stop bool) {
		for i, entry := range ubd.Entries {
			// Balance should not exceed initial balance
			suite.Require().True(
				entry.Balance.LTE(entry.InitialBalance),
				"Unbonding delegation entry %d has balance > initial: balance=%s, initial=%s",
				i,
				entry.Balance.String(),
				entry.InitialBalance.String(),
			)

			// Both should be positive
			suite.Require().True(
				entry.Balance.GT(math.ZeroInt()),
				"Unbonding delegation entry %d has non-positive balance: %s",
				i,
				entry.Balance.String(),
			)

			suite.Require().True(
				entry.InitialBalance.GT(math.ZeroInt()),
				"Unbonding delegation entry %d has non-positive initial balance: %s",
				i,
				entry.InitialBalance.String(),
			)
		}
		return false
	})
	suite.NoError(err)
}

// TestCommissionRatesValid ensures commission rates are within valid bounds
func (suite *StakingInvariantTestSuite) TestCommissionRatesValid() {
	err := suite.app.StakingKeeper.IterateValidators(suite.ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		commission := validator.GetCommission()

		// Rate should be between 0 and 1
		suite.Require().True(
			commission.GTE(math.LegacyZeroDec()) && commission.LTE(math.LegacyOneDec()),
			"Validator %s has invalid commission rate: %s",
			validator.GetOperator().String(),
			commission.String(),
		)

		return false
	})
	suite.NoError(err)
}

// TestMinSelfDelegation ensures validators meet minimum self-delegation
func (suite *StakingInvariantTestSuite) TestMinSelfDelegation() {
	err := suite.app.StakingKeeper.IterateValidators(suite.ctx, func(index int64, validator stakingtypes.ValidatorI) (stop bool) {
		valAddr, err := sdk.ValAddressFromBech32(validator.GetOperator().String())
		suite.NoError(err)

		// Get self-delegation
		delAddr := sdk.AccAddress(valAddr)
		delegation, err := suite.app.StakingKeeper.GetDelegation(suite.ctx, delAddr, valAddr)

		if err == nil {
			// Calculate self-delegation amount
			selfDelegation := validator.TokensFromShares(delegation.Shares).TruncateInt()

			// Should meet minimum self-delegation
			suite.Require().True(
				selfDelegation.GTE(validator.GetMinSelfDelegation()),
				"Validator %s below min self-delegation: current=%s, min=%s",
				validator.GetOperator().String(),
				selfDelegation.String(),
				validator.GetMinSelfDelegation().String(),
			)
		}

		return false
	})
	suite.NoError(err)
}

func TestStakingInvariantTestSuite(t *testing.T) {
	suite.Run(t, new(StakingInvariantTestSuite))
}
