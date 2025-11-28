package invariants

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/app"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// OracleInvariantTestSuite tests oracle module invariants
// Critical for price feed accuracy and validator slashing consistency
type OracleInvariantTestSuite struct {
	suite.Suite
	app *app.PAWApp
	ctx sdk.Context
}

// SetupTest initializes the test environment before each test
func (suite *OracleInvariantTestSuite) SetupTest() {
	suite.app = app.Setup(suite.T(), false)
	suite.ctx = suite.app.BaseApp.NewContext(false)
}

// TestPriceWithinValidBounds verifies all prices are within reasonable bounds
// Prices outside bounds could indicate bugs or manipulation
func (suite *OracleInvariantTestSuite) TestPriceWithinValidBounds() {
	// Get all prices
	allPrices := suite.app.OracleKeeper.GetAllPrices(suite.ctx)

	// Define reasonable bounds for prices
	minPrice := math.LegacyNewDecWithPrec(1, 18) // 0.000000000000000001
	maxPrice := math.LegacyNewDec(1000000000)     // 1 billion

	for _, price := range allPrices {
		suite.Require().True(
			price.Price.GT(minPrice),
			"Price for %s below minimum: %s < %s",
			price.Asset,
			price.Price.String(),
			minPrice.String(),
		)

		suite.Require().True(
			price.Price.LT(maxPrice),
			"Price for %s above maximum: %s > %s",
			price.Asset,
			price.Price.String(),
			maxPrice.String(),
		)

		suite.Require().False(
			price.Price.IsNegative(),
			"Price for %s is negative: %s",
			price.Asset,
			price.Price.String(),
		)

		suite.Require().False(
			price.Price.IsZero(),
			"Price for %s is zero",
			price.Asset,
		)
	}
}

// TestAggregatePriceMatchesWeightedMedian verifies aggregate price calculation
// This ensures the aggregation algorithm is working correctly
func (suite *OracleInvariantTestSuite) TestAggregatePriceMatchesWeightedMedian() {
	// Get all prices
	allPrices := suite.app.OracleKeeper.GetAllPrices(suite.ctx)

	for _, aggregatePrice := range allPrices {
		// Get all validator price submissions for this asset
		validatorPrices := suite.app.OracleKeeper.GetValidatorPrices(suite.ctx, aggregatePrice.Asset)

		if len(validatorPrices) > 0 {
			// Calculate expected median
			expectedMedian := suite.calculateWeightedMedian(validatorPrices)

			// Allow small deviation due to precision (0.01%)
			tolerance := aggregatePrice.Price.Mul(math.LegacyNewDecWithPrec(1, 4))
			diff := aggregatePrice.Price.Sub(expectedMedian).Abs()

			suite.Require().True(
				diff.LTE(tolerance),
				"Aggregate price for %s doesn't match weighted median: aggregate=%s, expected=%s, diff=%s",
				aggregatePrice.Asset,
				aggregatePrice.Price.String(),
				expectedMedian.String(),
				diff.String(),
			)
		}
	}
}

// TestMissCounterConsistency ensures miss counters are accurate
func (suite *OracleInvariantTestSuite) TestMissCounterConsistency() {
	// Get all validator oracles
	allValidatorOracles := suite.app.OracleKeeper.GetAllValidatorOracles(suite.ctx)
	params := suite.app.OracleKeeper.GetParams(suite.ctx)

	for _, valOracle := range allValidatorOracles {
		// Miss counter should not exceed slash window
		suite.Require().True(
			valOracle.MissCounter <= params.SlashWindow,
			"Validator %s miss counter %d exceeds slash window %d",
			valOracle.ValidatorAddr,
			valOracle.MissCounter,
			params.SlashWindow,
		)

		// Total submissions should be positive if validator is active
		if valOracle.IsActive {
			suite.Require().True(
				valOracle.TotalSubmissions > 0,
				"Active validator %s has zero total submissions",
				valOracle.ValidatorAddr,
			)
		}

		// Miss counter should not exceed total opportunities
		// (approximately total submissions + miss counter = total opportunities)
		suite.Require().True(
			valOracle.MissCounter <= valOracle.TotalSubmissions+valOracle.MissCounter,
			"Validator %s miss counter inconsistent with total submissions",
			valOracle.ValidatorAddr,
		)
	}
}

// TestSlashCounterAccuracy verifies slash counters match actual slashing events
func (suite *OracleInvariantTestSuite) TestSlashCounterAccuracy() {
	allValidatorOracles := suite.app.OracleKeeper.GetAllValidatorOracles(suite.ctx)
	params := suite.app.OracleKeeper.GetParams(suite.ctx)

	for _, valOracle := range allValidatorOracles {
		// If miss counter exceeds threshold, validator should be slashed/jailed
		if valOracle.MissCounter > params.MinValidPerWindow {
			valAddr, err := sdk.ValAddressFromBech32(valOracle.ValidatorAddr)
			suite.NoError(err)

			// Check if validator is jailed in slashing keeper
			validator, err := suite.app.StakingKeeper.GetValidator(suite.ctx, valAddr)
			if err == nil {
				// Validator should be jailed or have reduced power
				suite.Require().True(
					validator.IsJailed() || validator.GetTokens().LT(math.NewInt(1000000)),
					"Validator %s exceeded miss threshold but not jailed: misses=%d, threshold=%d",
					valOracle.ValidatorAddr,
					valOracle.MissCounter,
					params.MinValidPerWindow,
				)
			}
		}
	}
}

// TestValidatorPriceVotingPowerConsistency ensures voting power matches staking
func (suite *OracleInvariantTestSuite) TestValidatorPriceVotingPowerConsistency() {
	// Get all validator price submissions
	allPrices := suite.app.OracleKeeper.GetAllPrices(suite.ctx)

	for _, price := range allPrices {
		validatorPrices := suite.app.OracleKeeper.GetValidatorPrices(suite.ctx, price.Asset)

		for _, valPrice := range validatorPrices {
			valAddr, err := sdk.ValAddressFromBech32(valPrice.ValidatorAddr)
			suite.NoError(err)

			// Get actual validator
			validator, err := suite.app.StakingKeeper.GetValidator(suite.ctx, valAddr)
			if err == nil {
				// Voting power in price submission should match or be close to actual voting power
				actualPower := validator.GetConsensusPower(sdk.DefaultPowerReduction)

				// Allow for small differences due to timing
				suite.Require().True(
					abs(valPrice.VotingPower-actualPower) <= 1,
					"Validator %s voting power mismatch: oracle=%d, actual=%d",
					valPrice.ValidatorAddr,
					valPrice.VotingPower,
					actualPower,
				)
			}
		}
	}
}

// TestPriceBlockHeightMonotonic ensures price block heights are monotonically increasing
func (suite *OracleInvariantTestSuite) TestPriceBlockHeightMonotonic() {
	allPrices := suite.app.OracleKeeper.GetAllPrices(suite.ctx)

	currentHeight := suite.ctx.BlockHeight()

	for _, price := range allPrices {
		// Price block height should not be in the future
		suite.Require().True(
			price.BlockHeight <= currentHeight,
			"Price for %s has future block height: %d > %d",
			price.Asset,
			price.BlockHeight,
			currentHeight,
		)

		// Price block height should be positive
		suite.Require().True(
			price.BlockHeight > 0,
			"Price for %s has non-positive block height: %d",
			price.Asset,
			price.BlockHeight,
		)
	}
}

// TestNumValidatorsReasonable ensures reported validator count is reasonable
func (suite *OracleInvariantTestSuite) TestNumValidatorsReasonable() {
	allPrices := suite.app.OracleKeeper.GetAllPrices(suite.ctx)

	// Get total bonded validators
	var totalBondedValidators uint32 = 0
	err := suite.app.StakingKeeper.IterateBondedValidatorsByPower(suite.ctx, func(index int64, validator sdk.ValidatorI) (stop bool) {
		totalBondedValidators++
		return false
	})
	suite.NoError(err)

	for _, price := range allPrices {
		// NumValidators should not exceed total bonded validators
		suite.Require().True(
			price.NumValidators <= totalBondedValidators,
			"Price for %s reports more validators than exist: reported=%d, actual=%d",
			price.Asset,
			price.NumValidators,
			totalBondedValidators,
		)

		// NumValidators should be positive if price exists
		suite.Require().True(
			price.NumValidators > 0,
			"Price for %s has zero validators",
			price.Asset,
		)
	}
}

// TestValidatorOracleAddressesValid ensures all validator addresses are valid
func (suite *OracleInvariantTestSuite) TestValidatorOracleAddressesValid() {
	allValidatorOracles := suite.app.OracleKeeper.GetAllValidatorOracles(suite.ctx)

	for _, valOracle := range allValidatorOracles {
		_, err := sdk.ValAddressFromBech32(valOracle.ValidatorAddr)
		suite.Require().NoError(
			err,
			"ValidatorOracle has invalid address: %s",
			valOracle.ValidatorAddr,
		)
	}
}

// TestPriceAssetNamesValid ensures all asset names are non-empty and valid
func (suite *OracleInvariantTestSuite) TestPriceAssetNamesValid() {
	allPrices := suite.app.OracleKeeper.GetAllPrices(suite.ctx)

	for _, price := range allPrices {
		suite.Require().NotEmpty(
			price.Asset,
			"Price has empty asset name",
		)

		// Asset name should be reasonable length (e.g., 1-64 characters)
		suite.Require().True(
			len(price.Asset) >= 1 && len(price.Asset) <= 64,
			"Price asset name has invalid length: %s (%d chars)",
			price.Asset,
			len(price.Asset),
		)
	}
}

// TestValidatorPriceSubmissionsConsistent ensures price submissions are consistent
func (suite *OracleInvariantTestSuite) TestValidatorPriceSubmissionsConsistent() {
	allPrices := suite.app.OracleKeeper.GetAllPrices(suite.ctx)

	for _, price := range allPrices {
		validatorPrices := suite.app.OracleKeeper.GetValidatorPrices(suite.ctx, price.Asset)

		for _, valPrice := range validatorPrices {
			// Price should be positive
			suite.Require().False(
				valPrice.Price.IsNegative(),
				"Validator %s submitted negative price for %s: %s",
				valPrice.ValidatorAddr,
				valPrice.Asset,
				valPrice.Price.String(),
			)

			suite.Require().False(
				valPrice.Price.IsZero(),
				"Validator %s submitted zero price for %s",
				valPrice.ValidatorAddr,
				valPrice.Asset,
			)

			// Voting power should be non-negative
			suite.Require().True(
				valPrice.VotingPower >= 0,
				"Validator %s has negative voting power: %d",
				valPrice.ValidatorAddr,
				valPrice.VotingPower,
			)

			// Block height should be positive
			suite.Require().True(
				valPrice.BlockHeight > 0,
				"Validator %s price submission has non-positive block height: %d",
				valPrice.ValidatorAddr,
				valPrice.BlockHeight,
			)

			// Asset should match
			suite.Require().Equal(
				price.Asset,
				valPrice.Asset,
				"Validator price asset mismatch",
			)
		}
	}
}

// TestOracleParamsValid ensures oracle parameters are within valid ranges
func (suite *OracleInvariantTestSuite) TestOracleParamsValid() {
	params := suite.app.OracleKeeper.GetParams(suite.ctx)

	// Vote period should be positive
	suite.Require().True(
		params.VotePeriod > 0,
		"Vote period is non-positive: %d",
		params.VotePeriod,
	)

	// Vote threshold should be between 0 and 1
	suite.Require().True(
		params.VoteThreshold.GT(math.LegacyZeroDec()) && params.VoteThreshold.LTE(math.LegacyOneDec()),
		"Vote threshold out of bounds: %s",
		params.VoteThreshold.String(),
	)

	// Slash fraction should be between 0 and 1
	suite.Require().True(
		params.SlashFraction.GTE(math.LegacyZeroDec()) && params.SlashFraction.LTE(math.LegacyOneDec()),
		"Slash fraction out of bounds: %s",
		params.SlashFraction.String(),
	)

	// Slash window should be positive
	suite.Require().True(
		params.SlashWindow > 0,
		"Slash window is non-positive: %d",
		params.SlashWindow,
	)

	// MinValidPerWindow should not exceed SlashWindow
	suite.Require().True(
		params.MinValidPerWindow <= params.SlashWindow,
		"MinValidPerWindow %d exceeds SlashWindow %d",
		params.MinValidPerWindow,
		params.SlashWindow,
	)

	// TWAP lookback window should be positive
	suite.Require().True(
		params.TwapLookbackWindow > 0,
		"TWAP lookback window is non-positive: %d",
		params.TwapLookbackWindow,
	)
}

// TestPriceTimestampsReasonable ensures price timestamps are reasonable
func (suite *OracleInvariantTestSuite) TestPriceTimestampsReasonable() {
	allPrices := suite.app.OracleKeeper.GetAllPrices(suite.ctx)
	currentTime := suite.ctx.BlockTime().Unix()

	for _, price := range allPrices {
		// Price timestamp should not be in the distant future (within 1 day)
		suite.Require().True(
			price.BlockTime <= currentTime+86400,
			"Price for %s has timestamp far in future: %d > %d",
			price.Asset,
			price.BlockTime,
			currentTime+86400,
		)

		// Price timestamp should be positive
		suite.Require().True(
			price.BlockTime > 0,
			"Price for %s has non-positive timestamp: %d",
			price.Asset,
			price.BlockTime,
		)
	}
}

// TestActiveValidatorsHaveOracleRecords ensures active validators have oracle records
func (suite *OracleInvariantTestSuite) TestActiveValidatorsHaveOracleRecords() {
	// Get all bonded validators
	var bondedValidators []sdk.ValAddress
	err := suite.app.StakingKeeper.IterateBondedValidatorsByPower(suite.ctx, func(index int64, validator sdk.ValidatorI) (stop bool) {
		bondedValidators = append(bondedValidators, validator.GetOperator())
		return false
	})
	suite.NoError(err)

	// Each bonded validator should have an oracle record
	for _, valAddr := range bondedValidators {
		valOracle := suite.app.OracleKeeper.GetValidatorOracle(suite.ctx, valAddr)

		suite.Require().NotNil(
			valOracle,
			"Bonded validator %s has no oracle record",
			valAddr.String(),
		)
	}
}

// Helper function to calculate weighted median
func (suite *OracleInvariantTestSuite) calculateWeightedMedian(prices []oracletypes.ValidatorPrice) math.LegacyDec {
	if len(prices) == 0 {
		return math.LegacyZeroDec()
	}

	// Sort prices by value
	sorted := make([]oracletypes.ValidatorPrice, len(prices))
	copy(sorted, prices)

	// Simple bubble sort for small arrays
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Price.GT(sorted[j].Price) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Calculate total voting power
	totalPower := int64(0)
	for _, p := range sorted {
		totalPower += p.VotingPower
	}

	// Find weighted median
	cumulativePower := int64(0)
	halfPower := totalPower / 2

	for _, p := range sorted {
		cumulativePower += p.VotingPower
		if cumulativePower >= halfPower {
			return p.Price
		}
	}

	// Fallback to last price
	return sorted[len(sorted)-1].Price
}

// Helper function for absolute value
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func TestOracleInvariantTestSuite(t *testing.T) {
	suite.Run(t, new(OracleInvariantTestSuite))
}
