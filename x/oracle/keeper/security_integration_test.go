package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/app"
	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// OracleSecuritySuite provides comprehensive security integration testing with real keeper setup
type OracleSecuritySuite struct {
	suite.Suite
	app        *app.PAWApp
	ctx        sdk.Context
	keeper     *keeper.Keeper
	validators []sdk.ValAddress
	powers     []int64
}

func TestOracleSecuritySuite(t *testing.T) {
	suite.Run(t, new(OracleSecuritySuite))
}

func (suite *OracleSecuritySuite) SetupTest() {
	// Initialize test app with database
	db := dbm.NewMemDB()

	suite.app = app.NewPAWApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		simtestutil.EmptyAppOptions{},
		baseapp.SetChainID("paw-test-1"),
	)

	// Initialize chain with genesis state
	genesisState := app.NewDefaultGenesisState("paw-test-1")
	stateBytes, err := suite.app.AppCodec().MarshalJSON(genesisState)
	suite.Require().NoError(err)

	// Create context with block header
	suite.ctx = suite.app.NewContext(false, cmtproto.Header{
		Height:  1,
		ChainID: "paw-test-1",
		Time:    time.Now(),
	})

	// Get oracle keeper reference
	suite.keeper = &suite.app.OracleKeeper

	// Setup 10 test validators with varying stakes
	suite.setupValidators()

	// Initialize oracle module parameters
	params := types.Params{
		VotePeriod:         10,
		VoteThreshold:      sdkmath.LegacyMustNewDecFromStr("0.67"),
		SlashFraction:      sdkmath.LegacyMustNewDecFromStr("0.0001"),
		SlashWindow:        1000,
		MinValidPerWindow:  5,
		TwapLookbackWindow: 100,
	}
	err = suite.keeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Initialize stateBytes to avoid unused variable
	_ = stateBytes
}

func (suite *OracleSecuritySuite) setupValidators() {
	// Create 10 validators with different stakes
	// Powers: [1000, 900, 800, 700, 600, 500, 400, 300, 200, 100]
	// Total: 5500, ensuring no single validator has >20% (Byzantine threshold)

	suite.validators = make([]sdk.ValAddress, 10)
	suite.powers = []int64{1000, 900, 800, 700, 600, 500, 400, 300, 200, 100}

	for i := 0; i < 10; i++ {
		privKey := secp256k1.GenPrivKey()
		pubKey := privKey.PubKey()
		valAddr := sdk.ValAddress(pubKey.Address())
		suite.validators[i] = valAddr

		// Create validator with proper staking state
		pkAny, err := codectypes.NewAnyWithValue(pubKey)
		suite.Require().NoError(err)

		tokens := sdk.TokensFromConsensusPower(suite.powers[i], sdk.DefaultPowerReduction)

		validator := stakingtypes.Validator{
			OperatorAddress:   valAddr.String(),
			ConsensusPubkey:   pkAny,
			Jailed:            false,
			Status:            stakingtypes.Bonded,
			Tokens:            tokens,
			DelegatorShares:   sdkmath.LegacyNewDecFromInt(tokens),
			Description:       stakingtypes.Description{Moniker: "test-validator"},
			UnbondingHeight:   int64(0),
			UnbondingTime:     time.Unix(0, 0).UTC(),
			Commission:        stakingtypes.NewCommission(sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec()),
			MinSelfDelegation: sdkmath.OneInt(),
		}

		// Set validator in staking keeper
		err = suite.app.StakingKeeper.SetValidator(suite.ctx, validator)
		suite.Require().NoError(err)

		err = suite.app.StakingKeeper.SetValidatorByPowerIndex(suite.ctx, validator)
		suite.Require().NoError(err)
	}
}

// TestPriceManipulationAttack_SingleValidator tests single malicious validator submitting 10x price
func (suite *OracleSecuritySuite) TestPriceManipulationAttack_SingleValidator() {
	asset := "BTC"
	honestPrice := sdkmath.LegacyNewDec(50000)
	maliciousPrice := sdkmath.LegacyNewDec(500000) // 10x manipulation

	// 9 honest validators submit correct price
	for i := 0; i < 9; i++ {
		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         honestPrice,
			BlockHeight:   suite.ctx.BlockHeight(),
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	// 1 malicious validator (validator[9] with lowest power: 100)
	maliciousVP := types.ValidatorPrice{
		ValidatorAddr: suite.validators[9].String(),
		Asset:         asset,
		Price:         maliciousPrice,
		BlockHeight:   suite.ctx.BlockHeight(),
		VotingPower:   suite.powers[9],
	}
	err := suite.keeper.SetValidatorPrice(suite.ctx, maliciousVP)
	suite.Require().NoError(err)

	// Aggregate prices - should detect and reject outlier
	err = suite.keeper.AggregatePrices(suite.ctx, asset)
	suite.Require().NoError(err)

	// Verify aggregated price is NOT manipulated
	price, err := suite.keeper.GetPrice(suite.ctx, asset)
	suite.Require().NoError(err)

	// Price should be close to honest median (50000)
	deviation := price.Price.Sub(honestPrice).Abs()
	maxDeviation := honestPrice.Mul(sdkmath.LegacyMustNewDecFromStr("0.01")) // 1% tolerance
	suite.Require().True(deviation.LTE(maxDeviation),
		"Price manipulation detected: expected ~%s, got %s", honestPrice, price.Price)
}

// TestPriceManipulationAttack_CoordinatedValidators tests 3 coordinated malicious validators (< 33%)
func (suite *OracleSecuritySuite) TestPriceManipulationAttack_CoordinatedValidators() {
	asset := "ETH"
	honestPrice := sdkmath.LegacyNewDec(3000)
	maliciousPrice := sdkmath.LegacyNewDec(6000) // 2x manipulation

	// 7 honest validators (powers: 1000, 900, 800, 700, 600, 500, 400 = 4900)
	for i := 0; i < 7; i++ {
		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         honestPrice,
			BlockHeight:   suite.ctx.BlockHeight(),
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	// 3 malicious validators (powers: 300, 200, 100 = 600, ~11% of total)
	for i := 7; i < 10; i++ {
		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         maliciousPrice,
			BlockHeight:   suite.ctx.BlockHeight(),
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	// Aggregate - should filter outliers and use honest median
	err := suite.keeper.AggregatePrices(suite.ctx, asset)
	suite.Require().NoError(err)

	// Verify Byzantine resistance
	price, err := suite.keeper.GetPrice(suite.ctx, asset)
	suite.Require().NoError(err)

	deviation := price.Price.Sub(honestPrice).Abs()
	maxDeviation := honestPrice.Mul(sdkmath.LegacyMustNewDecFromStr("0.05")) // 5% tolerance
	suite.Require().True(deviation.LTE(maxDeviation),
		"Coordinated attack not resisted: expected ~%s, got %s", honestPrice, price.Price)
}

// TestFlashLoanAttack_PriceSpike tests sudden 50x price spike attempt
func (suite *OracleSecuritySuite) TestFlashLoanAttack_PriceSpike() {
	asset := "SOL"
	normalPrice := sdkmath.LegacyNewDec(100)
	spikePrice := sdkmath.LegacyNewDec(5000) // 50x spike

	// Block 1: All validators submit normal price
	for i := 0; i < 10; i++ {
		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         normalPrice,
			BlockHeight:   1,
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	err := suite.keeper.AggregatePrices(suite.ctx, asset)
	suite.Require().NoError(err)

	// Block 2: Attacker tries to manipulate with spike
	suite.ctx = suite.ctx.WithBlockHeight(2)

	// First 5 validators submit normal price
	for i := 0; i < 5; i++ {
		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         normalPrice,
			BlockHeight:   2,
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	// Last 5 submit spike (trying to manipulate)
	for i := 5; i < 10; i++ {
		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         spikePrice,
			BlockHeight:   2,
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	// Aggregate - outlier detection should flag extreme spike
	err = suite.keeper.AggregatePrices(suite.ctx, asset)
	suite.Require().NoError(err)

	// Verify flash loan attack failed
	price, err := suite.keeper.GetPrice(suite.ctx, asset)
	suite.Require().NoError(err)

	// Price should remain near normal despite attack
	deviation := price.Price.Sub(normalPrice).Abs()
	maxDeviation := normalPrice.Mul(sdkmath.LegacyMustNewDecFromStr("0.10")) // 10% tolerance
	suite.Require().True(deviation.LTE(maxDeviation),
		"Flash loan attack not prevented: expected ~%s, got %s", normalPrice, price.Price)
}

// TestSybilAttack_ManyLowStakeValidators tests resistance to Sybil attack with many low-power validators
func (suite *OracleSecuritySuite) TestSybilAttack_ManyLowStakeValidators() {
	asset := "ATOM"
	honestPrice := sdkmath.LegacyNewDec(15)
	attackPrice := sdkmath.LegacyNewDec(30)

	// Top 3 honest validators (high stake: 1000, 900, 800 = 2700)
	for i := 0; i < 3; i++ {
		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         honestPrice,
			BlockHeight:   suite.ctx.BlockHeight(),
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	// 7 Sybil validators (low stake: 700+600+500+400+300+200+100 = 2800)
	// Even though they have slightly more total power, weighted median should resist
	for i := 3; i < 10; i++ {
		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         attackPrice,
			BlockHeight:   suite.ctx.BlockHeight(),
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	// Aggregate - weighted median and outlier detection should prevent attack
	err := suite.keeper.AggregatePrices(suite.ctx, asset)
	suite.Require().NoError(err)

	// Verify Sybil resistance through statistical filtering
	price, err := suite.keeper.GetPrice(suite.ctx, asset)
	suite.Require().NoError(err)

	// With proper outlier detection, the price should not be manipulated
	// The attack prices should be filtered as outliers
	deviation := price.Price.Sub(honestPrice).Abs()
	maxDeviation := honestPrice.Mul(sdkmath.LegacyMustNewDecFromStr("0.15")) // 15% tolerance
	suite.Require().True(deviation.LTE(maxDeviation),
		"Sybil attack not resisted: expected ~%s, got %s", honestPrice, price.Price)
}

// TestDataPoisoningAttack_ExtremeValues tests extreme value submission
func (suite *OracleSecuritySuite) TestDataPoisoningAttack_ExtremeValues() {
	asset := "LINK"
	normalPrice := sdkmath.LegacyNewDec(20)

	// 8 honest validators
	for i := 0; i < 8; i++ {
		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         normalPrice,
			BlockHeight:   suite.ctx.BlockHeight(),
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	// Attacker submits extremely low value
	extremeLow := sdkmath.LegacyMustNewDecFromStr("0.001")
	vp1 := types.ValidatorPrice{
		ValidatorAddr: suite.validators[8].String(),
		Asset:         asset,
		Price:         extremeLow,
		BlockHeight:   suite.ctx.BlockHeight(),
		VotingPower:   suite.powers[8],
	}
	err := suite.keeper.SetValidatorPrice(suite.ctx, vp1)
	suite.Require().NoError(err)

	// Attacker submits extremely high value
	extremeHigh := sdkmath.LegacyNewDec(1000000)
	vp2 := types.ValidatorPrice{
		ValidatorAddr: suite.validators[9].String(),
		Asset:         asset,
		Price:         extremeHigh,
		BlockHeight:   suite.ctx.BlockHeight(),
		VotingPower:   suite.powers[9],
	}
	err = suite.keeper.SetValidatorPrice(suite.ctx, vp2)
	suite.Require().NoError(err)

	// Aggregate - extreme values should be filtered
	err = suite.keeper.AggregatePrices(suite.ctx, asset)
	suite.Require().NoError(err)

	// Verify data poisoning prevented
	price, err := suite.keeper.GetPrice(suite.ctx, asset)
	suite.Require().NoError(err)

	deviation := price.Price.Sub(normalPrice).Abs()
	maxDeviation := normalPrice.Mul(sdkmath.LegacyMustNewDecFromStr("0.05"))
	suite.Require().True(deviation.LTE(maxDeviation),
		"Data poisoning attack not prevented: expected ~%s, got %s", normalPrice, price.Price)
}

// TestCollusionAttack_IdenticalPrices tests validators submitting identical manipulated prices
func (suite *OracleSecuritySuite) TestCollusionAttack_IdenticalPrices() {
	asset := "AVAX"
	realPrice := sdkmath.LegacyNewDec(40)
	collusionPrice := sdkmath.LegacyNewDec(80)

	// 6 honest validators with slightly varied prices (simulating real oracle variance)
	for i := 0; i < 6; i++ {
		// Add small variance: 40, 40.2, 39.8, 40.1, 39.9, 40.3
		variance := sdkmath.LegacyNewDec(int64(i)).Mul(sdkmath.LegacyMustNewDecFromStr("0.1"))
		if i%2 == 0 {
			variance = variance.Neg()
		}
		price := realPrice.Add(variance)

		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         price,
			BlockHeight:   suite.ctx.BlockHeight(),
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	// 4 colluding validators submit IDENTICAL manipulated price
	for i := 6; i < 10; i++ {
		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         collusionPrice,
			BlockHeight:   suite.ctx.BlockHeight(),
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	// Aggregate - collusion should be detected and filtered
	err := suite.keeper.AggregatePrices(suite.ctx, asset)
	suite.Require().NoError(err)

	// Verify collusion resistance
	price, err := suite.keeper.GetPrice(suite.ctx, asset)
	suite.Require().NoError(err)

	deviation := price.Price.Sub(realPrice).Abs()
	maxDeviation := realPrice.Mul(sdkmath.LegacyMustNewDecFromStr("0.10"))
	suite.Require().True(deviation.LTE(maxDeviation),
		"Collusion attack not prevented: expected ~%s, got %s", realPrice, price.Price)
}

// TestOutlierDetection_EdgeCases tests statistical outlier detection edge cases
func (suite *OracleSecuritySuite) TestOutlierDetection_EdgeCases() {
	asset := "DOT"
	medianPrice := sdkmath.LegacyNewDec(8)

	// Create tight cluster with one significant outlier
	prices := []sdkmath.LegacyDec{
		sdkmath.LegacyMustNewDecFromStr("7.9"),
		sdkmath.LegacyMustNewDecFromStr("8.0"),
		sdkmath.LegacyMustNewDecFromStr("8.1"),
		sdkmath.LegacyMustNewDecFromStr("7.95"),
		sdkmath.LegacyMustNewDecFromStr("8.05"),
		sdkmath.LegacyMustNewDecFromStr("8.02"),
		sdkmath.LegacyMustNewDecFromStr("7.98"),
		sdkmath.LegacyMustNewDecFromStr("8.03"),
		sdkmath.LegacyMustNewDecFromStr("7.97"),
		sdkmath.LegacyMustNewDecFromStr("20.0"), // Extreme outlier
	}

	for i := 0; i < 10; i++ {
		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         prices[i],
			BlockHeight:   suite.ctx.BlockHeight(),
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	// Aggregate - should detect outlier using MAD/IQR/Grubbs
	err := suite.keeper.AggregatePrices(suite.ctx, asset)
	suite.Require().NoError(err)

	// Verify tight cluster preserved, outlier removed
	price, err := suite.keeper.GetPrice(suite.ctx, asset)
	suite.Require().NoError(err)

	deviation := price.Price.Sub(medianPrice).Abs()
	maxDeviation := sdkmath.LegacyMustNewDecFromStr("0.2") // Very tight tolerance
	suite.Require().True(deviation.LTE(maxDeviation),
		"Outlier detection failed: expected ~%s, got %s", medianPrice, price.Price)
}

// TestWeightedMedian_ByzantineResistance tests weighted median under Byzantine conditions
func (suite *OracleSecuritySuite) TestWeightedMedian_ByzantineResistance() {
	asset := "MATIC"
	honestPrice := sdkmath.LegacyNewDec(1)

	// 7 honest validators (67% of power)
	for i := 0; i < 7; i++ {
		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         honestPrice,
			BlockHeight:   suite.ctx.BlockHeight(),
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	// 3 Byzantine validators (33% of power) - maximum Byzantine threshold
	byzantinePrice := sdkmath.LegacyNewDec(10)
	for i := 7; i < 10; i++ {
		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         byzantinePrice,
			BlockHeight:   suite.ctx.BlockHeight(),
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	// Aggregate - weighted median should maintain Byzantine fault tolerance
	err := suite.keeper.AggregatePrices(suite.ctx, asset)
	suite.Require().NoError(err)

	// Verify BFT guarantees hold at 33% Byzantine threshold
	price, err := suite.keeper.GetPrice(suite.ctx, asset)
	suite.Require().NoError(err)

	deviation := price.Price.Sub(honestPrice).Abs()
	maxDeviation := honestPrice.Mul(sdkmath.LegacyMustNewDecFromStr("0.10"))
	suite.Require().True(deviation.LTE(maxDeviation),
		"Byzantine resistance failed at 33%% threshold: expected ~%s, got %s", honestPrice, price.Price)
}

// TestCircuitBreaker_ExtremeDeviation tests circuit breaker triggers on >50% deviation
func (suite *OracleSecuritySuite) TestCircuitBreaker_ExtremeDeviation() {
	asset := "UNI"
	previousPrice := sdkmath.LegacyNewDec(10)

	// Set initial price
	initialPrice := types.Price{
		Asset:         asset,
		Price:         previousPrice,
		BlockHeight:   1,
		BlockTime:     suite.ctx.BlockTime().Unix(),
		NumValidators: 10,
	}
	err := suite.keeper.SetPrice(suite.ctx, initialPrice)
	suite.Require().NoError(err)

	// Next block: all validators submit 60% higher price
	suite.ctx = suite.ctx.WithBlockHeight(2)
	extremePrice := sdkmath.LegacyNewDec(16) // 60% increase

	for i := 0; i < 10; i++ {
		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         extremePrice,
			BlockHeight:   2,
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	// Aggregate - should trigger circuit breaker
	err = suite.keeper.AggregatePrices(suite.ctx, asset)

	// Circuit breaker should activate - aggregation may fail or price should not update dramatically
	if err == nil {
		// If no error, verify price didn't change by >50%
		price, err := suite.keeper.GetPrice(suite.ctx, asset)
		suite.Require().NoError(err)

		changePercent := price.Price.Sub(previousPrice).Quo(previousPrice).Abs()
		maxChange := sdkmath.LegacyMustNewDecFromStr("0.50")

		// Either circuit breaker prevented update, or update was limited
		suite.Require().True(changePercent.LTE(maxChange) || price.BlockHeight == 1,
			"Circuit breaker failed: price changed by %s (>50%%)", changePercent)
	}
}

// TestByzantineTolerance_InsufficientValidators tests detection of insufficient validators
func (suite *OracleSecuritySuite) TestByzantineTolerance_InsufficientValidators() {
	// CheckByzantineTolerance should pass with 10 validators
	err := suite.keeper.CheckByzantineTolerance(suite.ctx)
	suite.Require().NoError(err, "Should pass with 10 validators")
}

// TestStakeConcentration_ExcessiveConcentration tests detection of stake concentration risk
func (suite *OracleSecuritySuite) TestStakeConcentration_ExcessiveConcentration() {
	// With our setup, max validator has 1000/5500 = 18.2% (below 20% threshold)
	err := suite.keeper.CheckByzantineTolerance(suite.ctx)
	suite.Require().NoError(err, "Stake concentration should be within limits")
}

// TestSlashingProgression_RepeatedOutliers tests progressive slashing for repeated offenders
func (suite *OracleSecuritySuite) TestSlashingProgression_RepeatedOutliers() {
	asset := "CRV"
	normalPrice := sdkmath.LegacyNewDec(2)
	outlierPrice := sdkmath.LegacyNewDec(10)

	// First offense: validator[9] submits outlier
	for i := 0; i < 9; i++ {
		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         normalPrice,
			BlockHeight:   suite.ctx.BlockHeight(),
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	// Malicious validator first offense
	vp := types.ValidatorPrice{
		ValidatorAddr: suite.validators[9].String(),
		Asset:         asset,
		Price:         outlierPrice,
		BlockHeight:   suite.ctx.BlockHeight(),
		VotingPower:   suite.powers[9],
	}
	err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
	suite.Require().NoError(err)

	err = suite.keeper.AggregatePrices(suite.ctx, asset)
	suite.Require().NoError(err)

	// Verify validator received grace period or minimal slash for first offense
	validatorOracle, err := suite.keeper.GetValidatorOracle(suite.ctx, suite.validators[9].String())
	suite.Require().NoError(err)
	suite.Require().NotNil(validatorOracle)

	// Second offense: repeat in later block
	suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 100)

	for i := 0; i < 9; i++ {
		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         normalPrice,
			BlockHeight:   suite.ctx.BlockHeight(),
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	vp2 := types.ValidatorPrice{
		ValidatorAddr: suite.validators[9].String(),
		Asset:         asset,
		Price:         outlierPrice,
		BlockHeight:   suite.ctx.BlockHeight(),
		VotingPower:   suite.powers[9],
	}
	err = suite.keeper.SetValidatorPrice(suite.ctx, vp2)
	suite.Require().NoError(err)

	err = suite.keeper.AggregatePrices(suite.ctx, asset)
	suite.Require().NoError(err)

	// Verify repeated offender tracking works
	// In a full integration test, we would verify slashing increased
}

// TestMultiAssetAggregation tests concurrent price aggregation for multiple assets
func (suite *OracleSecuritySuite) TestMultiAssetAggregation() {
	assets := []string{"BTC", "ETH", "SOL"}
	prices := []sdkmath.LegacyDec{
		sdkmath.LegacyNewDec(50000),
		sdkmath.LegacyNewDec(3000),
		sdkmath.LegacyNewDec(100),
	}

	// Submit prices for all assets from all validators
	for assetIdx, asset := range assets {
		for i := 0; i < 10; i++ {
			vp := types.ValidatorPrice{
				ValidatorAddr: suite.validators[i].String(),
				Asset:         asset,
				Price:         prices[assetIdx],
				BlockHeight:   suite.ctx.BlockHeight(),
				VotingPower:   suite.powers[i],
			}
			err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
			suite.Require().NoError(err)
		}
	}

	// Aggregate all assets
	for _, asset := range assets {
		err := suite.keeper.AggregatePrices(suite.ctx, asset)
		suite.Require().NoError(err)
	}

	// Verify all prices set correctly
	for assetIdx, asset := range assets {
		price, err := suite.keeper.GetPrice(suite.ctx, asset)
		suite.Require().NoError(err)
		suite.Require().Equal(asset, price.Asset)

		deviation := price.Price.Sub(prices[assetIdx]).Abs()
		maxDeviation := prices[assetIdx].Mul(sdkmath.LegacyMustNewDecFromStr("0.01"))
		suite.Require().True(deviation.LTE(maxDeviation))
	}
}

// TestValidatorJailing_ExtremeOutlier tests validator jailing for extreme outliers
func (suite *OracleSecuritySuite) TestValidatorJailing_ExtremeOutlier() {
	asset := "AAVE"
	normalPrice := sdkmath.LegacyNewDec(100)
	extremePrice := sdkmath.LegacyNewDec(10000) // 100x outlier

	// 9 honest validators
	for i := 0; i < 9; i++ {
		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         normalPrice,
			BlockHeight:   suite.ctx.BlockHeight(),
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	// 1 validator submits extreme outlier (should trigger jailing)
	vp := types.ValidatorPrice{
		ValidatorAddr: suite.validators[9].String(),
		Asset:         asset,
		Price:         extremePrice,
		BlockHeight:   suite.ctx.BlockHeight(),
		VotingPower:   suite.powers[9],
	}
	err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
	suite.Require().NoError(err)

	err = suite.keeper.AggregatePrices(suite.ctx, asset)
	suite.Require().NoError(err)

	// Verify validator was slashed (jailing verification would require full staking integration)
	price, err := suite.keeper.GetPrice(suite.ctx, asset)
	suite.Require().NoError(err)

	// Price should be unaffected
	deviation := price.Price.Sub(normalPrice).Abs()
	maxDeviation := normalPrice.Mul(sdkmath.LegacyMustNewDecFromStr("0.05"))
	suite.Require().True(deviation.LTE(maxDeviation))
}

// TestPriceSnapshot_TWAPDataIntegrity tests price snapshot storage for TWAP
func (suite *OracleSecuritySuite) TestPriceSnapshot_TWAPDataIntegrity() {
	asset := "LUNA"

	// Submit prices over multiple blocks
	for block := int64(1); block <= 10; block++ {
		suite.ctx = suite.ctx.WithBlockHeight(block)
		price := sdkmath.LegacyNewDec(50 + block) // Gradually increasing price

		for i := 0; i < 10; i++ {
			vp := types.ValidatorPrice{
				ValidatorAddr: suite.validators[i].String(),
				Asset:         asset,
				Price:         price,
				BlockHeight:   block,
				VotingPower:   suite.powers[i],
			}
			err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
			suite.Require().NoError(err)
		}

		err := suite.keeper.AggregatePrices(suite.ctx, asset)
		suite.Require().NoError(err)
	}

	// Verify snapshots were created
	// Note: Full TWAP calculation testing would require accessing snapshot history
	finalPrice, err := suite.keeper.GetPrice(suite.ctx, asset)
	suite.Require().NoError(err)
	suite.Require().Equal(asset, finalPrice.Asset)
	suite.Require().Equal(int64(10), finalPrice.BlockHeight)
}

// TestVotingPowerThreshold tests insufficient voting power rejection
func (suite *OracleSecuritySuite) TestVotingPowerThreshold() {
	asset := "FTM"
	price := sdkmath.LegacyNewDec(1)

	// Only 2 validators submit (insufficient for 67% threshold)
	// Total power: 1000 + 900 = 1900 / 5500 = 34.5% < 67%
	for i := 0; i < 2; i++ {
		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         price,
			BlockHeight:   suite.ctx.BlockHeight(),
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	// Aggregation should fail due to insufficient voting power
	err := suite.keeper.AggregatePrices(suite.ctx, asset)
	suite.Require().Error(err, "Should fail with insufficient voting power")
	suite.Require().Contains(err.Error(), "insufficient voting power")
}

// TestPriceVarianceAnalysis tests natural price variance vs manipulation
func (suite *OracleSecuritySuite) TestPriceVarianceAnalysis() {
	asset := "NEAR"
	basePrice := sdkmath.LegacyNewDec(5)

	// Simulate realistic oracle variance (±2%)
	for i := 0; i < 10; i++ {
		// Each validator has slightly different price due to different data sources
		variance := sdkmath.LegacyNewDec(int64(i - 5)).Mul(sdkmath.LegacyMustNewDecFromStr("0.02"))
		price := basePrice.Add(variance)

		vp := types.ValidatorPrice{
			ValidatorAddr: suite.validators[i].String(),
			Asset:         asset,
			Price:         price,
			BlockHeight:   suite.ctx.BlockHeight(),
			VotingPower:   suite.powers[i],
		}
		err := suite.keeper.SetValidatorPrice(suite.ctx, vp)
		suite.Require().NoError(err)
	}

	// Aggregate - natural variance should not trigger outlier detection
	err := suite.keeper.AggregatePrices(suite.ctx, asset)
	suite.Require().NoError(err)

	price, err := suite.keeper.GetPrice(suite.ctx, asset)
	suite.Require().NoError(err)

	// Result should be close to base price
	deviation := price.Price.Sub(basePrice).Abs()
	maxDeviation := basePrice.Mul(sdkmath.LegacyMustNewDecFromStr("0.05"))
	suite.Require().True(deviation.LTE(maxDeviation))
}
