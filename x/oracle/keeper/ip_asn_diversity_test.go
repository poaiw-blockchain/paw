package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/app"
	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// IPASNDiversitySuite tests IP and ASN diversity enforcement
type IPASNDiversitySuite struct {
	suite.Suite
	app        *app.PAWApp
	ctx        sdk.Context
	keeper     *keeper.Keeper
	validators []sdk.ValAddress
}

func TestIPASNDiversitySuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(IPASNDiversitySuite))
}

func (suite *IPASNDiversitySuite) SetupTest() {
	testApp, ctx := keepertest.SetupTestApp(suite.T())
	suite.app = testApp
	suite.ctx = ctx
	suite.keeper = testApp.OracleKeeper

	// Setup staking params
	stakingParams, err := suite.app.StakingKeeper.GetParams(suite.ctx)
	suite.Require().NoError(err)
	stakingParams.BondDenom = "upaw"
	stakingParams.MaxValidators = 100
	err = suite.app.StakingKeeper.SetParams(suite.ctx, stakingParams)
	suite.Require().NoError(err)

	// Prefund bonded pool
	bondFund := sdk.NewCoins(sdk.NewInt64Coin("upaw", 1_000_000_000_000))
	suite.Require().NoError(suite.app.BankKeeper.MintCoins(suite.ctx, minttypes.ModuleName, bondFund))
	suite.Require().NoError(suite.app.BankKeeper.SendCoinsFromModuleToModule(suite.ctx, minttypes.ModuleName, stakingtypes.BondedPoolName, bondFund))

	// Setup test validators
	suite.setupValidators()

	// Initialize oracle params with IP/ASN limits
	params := types.DefaultParams()
	params.MaxValidatorsPerIp = 3
	params.MaxValidatorsPerAsn = 5
	err = suite.keeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)
}

func (suite *IPASNDiversitySuite) setupValidators() {
	// Create 10 validators
	suite.validators = make([]sdk.ValAddress, 10)
	powers := []int64{1000, 900, 800, 700, 600, 500, 400, 300, 200, 100}

	for i := 0; i < 10; i++ {
		privKey := secp256k1.GenPrivKey()
		pubKey := privKey.PubKey()
		valAddr := sdk.ValAddress(pubKey.Address())
		suite.validators[i] = valAddr

		pkAny, err := codectypes.NewAnyWithValue(pubKey)
		suite.Require().NoError(err)

		tokens := sdk.TokensFromConsensusPower(powers[i], sdk.DefaultPowerReduction)

		validator := stakingtypes.Validator{
			OperatorAddress: valAddr.String(),
			ConsensusPubkey: pkAny,
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          tokens,
			DelegatorShares: sdkmath.LegacyNewDecFromInt(tokens),
			Commission: stakingtypes.Commission{
				CommissionRates: stakingtypes.CommissionRates{
					Rate:          sdkmath.LegacyMustNewDecFromStr("0.1"),
					MaxRate:       sdkmath.LegacyMustNewDecFromStr("0.2"),
					MaxChangeRate: sdkmath.LegacyMustNewDecFromStr("0.01"),
				},
				UpdateTime: suite.ctx.BlockTime(),
			},
			MinSelfDelegation: sdkmath.OneInt(),
		}

		err = suite.app.StakingKeeper.SetValidator(suite.ctx, validator)
		suite.Require().NoError(err)

		err = suite.app.StakingKeeper.SetValidatorByPowerIndex(suite.ctx, validator)
		suite.Require().NoError(err)

		err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
		suite.Require().NoError(err)

		err = suite.app.StakingKeeper.SetLastValidatorPower(suite.ctx, valAddr, powers[i])
		suite.Require().NoError(err)

		// Self-delegation to back validator tokens
		delegator := sdk.AccAddress(valAddr)
		delegation := stakingtypes.Delegation{
			DelegatorAddress: delegator.String(),
			ValidatorAddress: valAddr.String(),
			Shares:           sdkmath.LegacyNewDecFromInt(tokens),
		}
		suite.Require().NoError(suite.app.StakingKeeper.SetDelegation(suite.ctx, delegation))
	}
}

// TestCountValidatorsFromIP tests IP counting logic
func (suite *IPASNDiversitySuite) TestCountValidatorsFromIP() {
	// Set up validators with same IP
	testIP := "192.168.1.100"

	// Assign same IP to 3 validators
	for i := 0; i < 3; i++ {
		err := suite.keeper.SetValidatorIPAndASN(suite.ctx, suite.validators[i].String(), testIP, 0)
		suite.Require().NoError(err)
	}

	// Assign different IP to remaining validators
	for i := 3; i < 10; i++ {
		err := suite.keeper.SetValidatorIPAndASN(suite.ctx, suite.validators[i].String(), "10.0.0."+string(rune(i)), 0)
		suite.Require().NoError(err)
	}

	// Count validators from testIP
	ipDistribution, _, err := suite.keeper.GetIPAndASNDistribution(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(3, ipDistribution[testIP], "Should have 3 validators from test IP")
}

// TestCountValidatorsFromASN tests ASN counting logic
func (suite *IPASNDiversitySuite) TestCountValidatorsFromASN() {
	// Set up validators with same ASN
	testASN := uint64(15169) // Google's ASN

	// Assign same ASN to 5 validators
	for i := 0; i < 5; i++ {
		err := suite.keeper.SetValidatorIPAndASN(suite.ctx, suite.validators[i].String(), "192.168.1."+string(rune(100+i)), testASN)
		suite.Require().NoError(err)
	}

	// Assign different ASN to remaining validators
	for i := 5; i < 10; i++ {
		err := suite.keeper.SetValidatorIPAndASN(suite.ctx, suite.validators[i].String(), "10.0.0."+string(rune(i)), uint64(7224+i))
		suite.Require().NoError(err)
	}

	// Count validators from testASN
	_, asnDistribution, err := suite.keeper.GetIPAndASNDistribution(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(5, asnDistribution[testASN], "Should have 5 validators from test ASN")
}

// TestIPDiversityEnforcement tests IP diversity limits
func (suite *IPASNDiversitySuite) TestIPDiversityEnforcement() {
	testIP := "192.168.1.100"

	// Add 3 validators with same IP (at limit)
	for i := 0; i < 3; i++ {
		err := suite.keeper.SetValidatorIPAndASN(suite.ctx, suite.validators[i].String(), testIP, 0)
		suite.Require().NoError(err)
	}

	// Validation should succeed at limit
	err := suite.keeper.ValidateIPAndASNDiversity(suite.ctx, suite.validators[0].String())
	suite.Require().NoError(err, "Should allow exactly max_validators_per_ip")

	// Add 4th validator with same IP (exceeds limit)
	err = suite.keeper.SetValidatorIPAndASN(suite.ctx, suite.validators[3].String(), testIP, 0)
	suite.Require().NoError(err)

	// Validation should fail
	err = suite.keeper.ValidateIPAndASNDiversity(suite.ctx, suite.validators[3].String())
	suite.Require().Error(err, "Should reject when exceeding max_validators_per_ip")
	suite.Require().Contains(err.Error(), "SYBIL ATTACK RISK")
}

// TestASNDiversityEnforcement tests ASN diversity limits
func (suite *IPASNDiversitySuite) TestASNDiversityEnforcement() {
	testASN := uint64(15169)

	// Add 5 validators with same ASN (at limit)
	for i := 0; i < 5; i++ {
		err := suite.keeper.SetValidatorIPAndASN(suite.ctx, suite.validators[i].String(), "192.168.1."+string(rune(100+i)), testASN)
		suite.Require().NoError(err)
	}

	// Validation should succeed at limit
	err := suite.keeper.ValidateIPAndASNDiversity(suite.ctx, suite.validators[0].String())
	suite.Require().NoError(err, "Should allow exactly max_validators_per_asn")

	// Add 6th validator with same ASN (exceeds limit)
	err = suite.keeper.SetValidatorIPAndASN(suite.ctx, suite.validators[5].String(), "192.168.1.200", testASN)
	suite.Require().NoError(err)

	// Validation should fail
	err = suite.keeper.ValidateIPAndASNDiversity(suite.ctx, suite.validators[5].String())
	suite.Require().Error(err, "Should reject when exceeding max_validators_per_asn")
	suite.Require().Contains(err.Error(), "ISP CENTRALIZATION RISK")
}

// TestIPAndASNCombination tests enforcement of both IP and ASN limits
func (suite *IPASNDiversitySuite) TestIPAndASNCombination() {
	// Scenario: Multiple validators from same ASN but different IPs
	testASN := uint64(15169)

	// 3 validators from same ASN, different IPs (should pass)
	for i := 0; i < 3; i++ {
		err := suite.keeper.SetValidatorIPAndASN(suite.ctx, suite.validators[i].String(), "192.168.1."+string(rune(100+i)), testASN)
		suite.Require().NoError(err)

		err = suite.keeper.ValidateIPAndASNDiversity(suite.ctx, suite.validators[i].String())
		suite.Require().NoError(err)
	}

	// Scenario: Multiple validators from same IP but different ASNs
	testIP := "10.0.0.1"

	// 2 validators from same IP, different ASNs (should pass)
	err := suite.keeper.SetValidatorIPAndASN(suite.ctx, suite.validators[5].String(), testIP, 7224)
	suite.Require().NoError(err)

	err = suite.keeper.SetValidatorIPAndASN(suite.ctx, suite.validators[6].String(), testIP, 7225)
	suite.Require().NoError(err)

	err = suite.keeper.ValidateIPAndASNDiversity(suite.ctx, suite.validators[5].String())
	suite.Require().NoError(err)

	err = suite.keeper.ValidateIPAndASNDiversity(suite.ctx, suite.validators[6].String())
	suite.Require().NoError(err)
}

// TestEmptyIPAndASN tests handling of empty/zero IP and ASN
func (suite *IPASNDiversitySuite) TestEmptyIPAndASN() {
	// Validators with empty IP should not be counted
	err := suite.keeper.SetValidatorIPAndASN(suite.ctx, suite.validators[0].String(), "", 0)
	suite.Require().NoError(err)

	err = suite.keeper.ValidateIPAndASNDiversity(suite.ctx, suite.validators[0].String())
	suite.Require().NoError(err, "Should allow validators with empty IP/ASN")

	ipDistribution, asnDistribution, err := suite.keeper.GetIPAndASNDistribution(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().NotContains(ipDistribution, "", "Empty IP should not be in distribution")
	suite.Require().NotContains(asnDistribution, uint64(0), "Zero ASN should not be in distribution")
}

// TestGetIPAndASNDistribution tests distribution calculation
func (suite *IPASNDiversitySuite) TestGetIPAndASNDistribution() {
	// Setup diverse distribution
	testCases := []struct {
		ip  string
		asn uint64
	}{
		{"192.168.1.1", 15169}, // Google
		{"192.168.1.2", 15169}, // Google
		{"10.0.0.1", 7224},     // Comcast
		{"10.0.0.2", 7224},     // Comcast
		{"172.16.0.1", 209},    // Qwest
		{"172.16.0.2", 209},    // Qwest
		{"8.8.8.8", 15169},     // Google
		{"1.1.1.1", 13335},     // Cloudflare
		{"4.4.4.4", 3356},      // Level3
		{"9.9.9.9", 19281},     // Quad9
	}

	for i, tc := range testCases {
		err := suite.keeper.SetValidatorIPAndASN(suite.ctx, suite.validators[i].String(), tc.ip, tc.asn)
		suite.Require().NoError(err)
	}

	ipDistribution, asnDistribution, err := suite.keeper.GetIPAndASNDistribution(suite.ctx)
	suite.Require().NoError(err)

	// Verify IP distribution
	suite.Require().Equal(10, len(ipDistribution), "Should have 10 unique IPs")
	for ip := range ipDistribution {
		suite.Require().Equal(1, ipDistribution[ip], "Each IP should have exactly 1 validator")
	}

	// Verify ASN distribution
	suite.Require().Equal(6, len(asnDistribution), "Should have 6 unique ASNs")
	suite.Require().Equal(3, asnDistribution[15169], "Google ASN should have 3 validators")
	suite.Require().Equal(2, asnDistribution[7224], "Comcast ASN should have 2 validators")
	suite.Require().Equal(2, asnDistribution[209], "Qwest ASN should have 2 validators")
}

// TestDynamicIPASNUpdates tests updating IP/ASN for existing validators
func (suite *IPASNDiversitySuite) TestDynamicIPASNUpdates() {
	validatorAddr := suite.validators[0].String()

	// Initial IP/ASN
	err := suite.keeper.SetValidatorIPAndASN(suite.ctx, validatorAddr, "192.168.1.1", 15169)
	suite.Require().NoError(err)

	valOracle, err := suite.keeper.GetValidatorOracle(suite.ctx, validatorAddr)
	suite.Require().NoError(err)
	suite.Require().Equal("192.168.1.1", valOracle.IpAddress)
	suite.Require().Equal(uint64(15169), valOracle.Asn)

	// Update IP/ASN
	err = suite.keeper.SetValidatorIPAndASN(suite.ctx, validatorAddr, "10.0.0.1", 7224)
	suite.Require().NoError(err)

	valOracle, err = suite.keeper.GetValidatorOracle(suite.ctx, validatorAddr)
	suite.Require().NoError(err)
	suite.Require().Equal("10.0.0.1", valOracle.IpAddress)
	suite.Require().Equal(uint64(7224), valOracle.Asn)
}

// TestParameterValidation tests parameter validation for IP/ASN limits
func (suite *IPASNDiversitySuite) TestParameterValidation() {
	// Test with zero limits (disabled)
	params := types.DefaultParams()
	params.MaxValidatorsPerIp = 0
	params.MaxValidatorsPerAsn = 0
	err := suite.keeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Add multiple validators with same IP/ASN
	for i := 0; i < 5; i++ {
		err := suite.keeper.SetValidatorIPAndASN(suite.ctx, suite.validators[i].String(), "192.168.1.1", 15169)
		suite.Require().NoError(err)
	}

	// Validation should pass when limits are disabled
	err = suite.keeper.ValidateIPAndASNDiversity(suite.ctx, suite.validators[0].String())
	suite.Require().NoError(err, "Should allow any number when limits are disabled (0)")

	// Re-enable limits
	params.MaxValidatorsPerIp = 3
	params.MaxValidatorsPerAsn = 5
	err = suite.keeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Now validation should fail
	err = suite.keeper.ValidateIPAndASNDiversity(suite.ctx, suite.validators[0].String())
	suite.Require().Error(err, "Should fail when limits are re-enabled")
}

// TestSybilAttackScenario tests realistic Sybil attack scenario
func (suite *IPASNDiversitySuite) TestSybilAttackScenario() {
	// Attacker tries to spin up 10 validators from same location
	attackerIP := "203.0.113.1"  // TEST-NET-3 (documentation)
	attackerASN := uint64(64512) // Reserved ASN for documentation

	// Try to add 10 validators from attacker's infrastructure
	var firstError error
	for i := 0; i < 10; i++ {
		err := suite.keeper.SetValidatorIPAndASN(suite.ctx, suite.validators[i].String(), attackerIP, attackerASN)
		suite.Require().NoError(err)

		err = suite.keeper.ValidateIPAndASNDiversity(suite.ctx, suite.validators[i].String())
		if err != nil && firstError == nil {
			firstError = err
		}
	}

	// Should have detected violation
	suite.Require().Error(firstError, "Should detect Sybil attack pattern")

	// Verify distribution shows concentration
	ipDistribution, asnDistribution, err := suite.keeper.GetIPAndASNDistribution(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(10, ipDistribution[attackerIP], "All validators on attacker IP")
	suite.Require().Equal(10, asnDistribution[attackerASN], "All validators on attacker ASN")
}

// TestGeographicWithIPASN tests geographic region enforcement alongside IP/ASN
func (suite *IPASNDiversitySuite) TestGeographicWithIPASN() {
	// Setup validators with diverse geography but concentrated IP/ASN
	testIP := "192.168.1.1"
	testASN := uint64(15169)

	// All from same IP/ASN but different regions (should still fail IP/ASN check)
	regions := []string{"na", "eu", "apac", "na", "eu"}

	for i := 0; i < 5; i++ {
		// Set IP/ASN
		err := suite.keeper.SetValidatorIPAndASN(suite.ctx, suite.validators[i].String(), testIP, testASN)
		suite.Require().NoError(err)

		// Set geographic region
		valOracle, err := suite.keeper.GetValidatorOracle(suite.ctx, suite.validators[i].String())
		suite.Require().NoError(err)
		valOracle.GeographicRegion = regions[i]
		err = suite.keeper.SetValidatorOracle(suite.ctx, valOracle)
		suite.Require().NoError(err)
	}

	// IP/ASN check should fail despite geographic diversity
	err := suite.keeper.ValidateIPAndASNDiversity(suite.ctx, suite.validators[4].String())
	suite.Require().Error(err, "Should fail IP check even with geographic diversity")
}

// TestEventEmission tests that proper events are emitted
func (suite *IPASNDiversitySuite) TestEventEmission() {
	testIP := "192.168.1.1"
	testASN := uint64(15169)

	// Clear any existing events
	suite.ctx = suite.ctx.WithEventManager(sdk.NewEventManager())

	// Set IP/ASN (should emit update event)
	err := suite.keeper.SetValidatorIPAndASN(suite.ctx, suite.validators[0].String(), testIP, testASN)
	suite.Require().NoError(err)

	events := suite.ctx.EventManager().Events()
	suite.Require().NotEmpty(events)

	// Find the update event
	var foundUpdateEvent bool
	for _, event := range events {
		if event.Type == "validator_ip_asn_updated" {
			foundUpdateEvent = true
			// Verify event attributes exist
			suite.Require().Greater(len(event.Attributes), 0, "Event should have attributes")
		}
	}
	suite.Require().True(foundUpdateEvent, "Should emit validator_ip_asn_updated event")

	// Add more validators to trigger violation
	for i := 1; i < 4; i++ {
		err := suite.keeper.SetValidatorIPAndASN(suite.ctx, suite.validators[i].String(), testIP, testASN)
		suite.Require().NoError(err)
	}

	// Clear events
	suite.ctx = suite.ctx.WithEventManager(sdk.NewEventManager())

	// Trigger violation
	err = suite.keeper.ValidateIPAndASNDiversity(suite.ctx, suite.validators[3].String())
	suite.Require().Error(err)

	events = suite.ctx.EventManager().Events()
	suite.Require().NotEmpty(events)

	// Should have violation event
	var foundViolationEvent bool
	for _, event := range events {
		if event.Type == "ip_diversity_violation" {
			foundViolationEvent = true
		}
	}
	suite.Require().True(foundViolationEvent, "Should emit diversity violation event")
}
