// TEST-1.4: Oracle Price Manipulation Simulation
// Tests oracle resilience against various price manipulation attacks
package simulation_test

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/keeper"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// OracleManipulationTestSuite tests oracle resilience against attacks
type OracleManipulationTestSuite struct {
	suite.Suite
	k   *keeper.Keeper
	ctx sdk.Context

	// Simulated validators with different behaviors
	validators    []*OracleValidator
	maliciousRate float64 // Percentage of malicious validators
}

// OracleValidator represents a validator submitting oracle prices
type OracleValidator struct {
	Address      string
	VotingPower  int64
	IsMalicious  bool
	Submissions  int
	Rejections   int
	mu           sync.Mutex
}

func TestOracleManipulationTestSuite(t *testing.T) {
	suite.Run(t, new(OracleManipulationTestSuite))
}

func (suite *OracleManipulationTestSuite) SetupTest() {
	suite.k, suite.ctx = keepertest.OracleKeeper(suite.T())

	// Create 20 validators (5 malicious = 25%)
	numValidators := 20
	numMalicious := 5

	suite.validators = make([]*OracleValidator, numValidators)
	suite.maliciousRate = float64(numMalicious) / float64(numValidators)

	for i := 0; i < numValidators; i++ {
		suite.validators[i] = &OracleValidator{
			Address:     fmt.Sprintf("pawvaloper1validator%02d", i),
			VotingPower: 100,
			IsMalicious: i < numMalicious,
		}
	}
}

// ManipulationResult captures attack simulation results
type ManipulationResult struct {
	AttackType        string
	TotalSubmissions  int
	AcceptedPrices    int
	RejectedPrices    int
	PriceDeviationPct float64
	AttackSuccess     bool
	Duration          time.Duration
}

func (r ManipulationResult) String() string {
	return fmt.Sprintf("%s: Accepted=%d/%d, Deviation=%.2f%%, Attack Success=%v",
		r.AttackType, r.AcceptedPrices, r.TotalSubmissions, r.PriceDeviationPct, r.AttackSuccess)
}

// TestFlashCrashManipulation simulates sudden price crash attack
func (suite *OracleManipulationTestSuite) TestFlashCrashManipulation() {
	asset := "BTC"
	legitimatePrice := math.LegacyNewDec(50000)

	// Establish baseline price
	suite.submitLegitimatePrice(asset, legitimatePrice, 10)

	// Malicious validators attempt flash crash
	crashPrice := math.LegacyNewDec(10000) // 80% drop

	result := suite.simulateAttack(asset, "FlashCrash", func(v *OracleValidator) math.LegacyDec {
		if v.IsMalicious {
			return crashPrice
		}
		return legitimatePrice
	})

	suite.T().Logf("TEST-1.4 Flash Crash: %s", result)

	// Attack should be rejected due to deviation limits
	suite.False(result.AttackSuccess, "Flash crash attack should be rejected")
}

// TestPumpAndDump simulates coordinated price pump followed by dump
func (suite *OracleManipulationTestSuite) TestPumpAndDump() {
	asset := "ETH"
	basePrice := math.LegacyNewDec(3000)

	// Phase 1: Establish baseline
	suite.submitLegitimatePrice(asset, basePrice, 10)

	// Phase 2: Gradual pump (malicious validators submit higher prices)
	pumpPrice := math.LegacyNewDec(4500) // 50% above base

	result := suite.simulateAttack(asset, "PumpAndDump", func(v *OracleValidator) math.LegacyDec {
		if v.IsMalicious {
			// Add randomness to avoid detection
			deviation := rand.Float64() * 0.05 // ±5%
			return pumpPrice.Mul(math.LegacyNewDecWithPrec(int64(100+deviation*100), 2))
		}
		return basePrice.Add(math.LegacyNewDec(rand.Int63n(100))) // Small variation
	})

	suite.T().Logf("TEST-1.4 Pump and Dump: %s", result)

	// With 25% malicious validators, attack should fail (need >33%)
	suite.False(result.AttackSuccess, "Pump and dump should fail with <33% malicious")
}

// TestStaleDataAttack simulates submitting outdated prices
func (suite *OracleManipulationTestSuite) TestStaleDataAttack() {
	asset := "ATOM"
	currentPrice := math.LegacyNewDec(10)
	stalePrice := math.LegacyNewDec(15) // Old price from 24h ago

	// Establish current price
	suite.submitLegitimatePrice(asset, currentPrice, 5)

	// Malicious validators submit stale data
	staleSubmissions := 0
	rejectedStale := 0

	for _, v := range suite.validators {
		if v.IsMalicious {
			// Submit price with old timestamp (simulated)
			err := suite.submitPriceWithTimestamp(asset, stalePrice, v.Address, time.Now().Add(-24*time.Hour))
			staleSubmissions++
			if err != nil {
				rejectedStale++
			}
		}
	}

	suite.T().Logf("TEST-1.4 Stale Data Attack: Submitted=%d, Rejected=%d",
		staleSubmissions, rejectedStale)

	// Stale data should be rejected
	suite.Equal(staleSubmissions, rejectedStale, "All stale submissions should be rejected")
}

// TestFrontRunningAttack simulates front-running oracle updates
func (suite *OracleManipulationTestSuite) TestFrontRunningAttack() {
	asset := "PAW"
	currentPrice := math.LegacyNewDec(1)

	// Establish baseline
	suite.submitLegitimatePrice(asset, currentPrice, 5)

	// Simulate front-running: attacker sees pending price update and submits first
	newLegitimatePrice := math.LegacyNewDec(12) // 10% increase
	frontRunPrice := math.LegacyNewDec(15)      // Attacker's exaggerated price

	var wg sync.WaitGroup
	var frontRunAccepted bool
	var legitimateAccepted bool

	// Front-runner submits first
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, v := range suite.validators {
			if v.IsMalicious {
				err := suite.submitPrice(asset, frontRunPrice, v.Address)
				if err == nil {
					frontRunAccepted = true
				}
				break
			}
		}
	}()

	// Legitimate validators submit after small delay
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
		for _, v := range suite.validators {
			if !v.IsMalicious {
				err := suite.submitPrice(asset, newLegitimatePrice, v.Address)
				if err == nil {
					legitimateAccepted = true
				}
			}
		}
	}()

	wg.Wait()

	suite.T().Logf("TEST-1.4 Front-Running: FrontRun=%v, Legitimate=%v",
		frontRunAccepted, legitimateAccepted)

	// Legitimate price should ultimately be used (TWAP/median)
	finalPrice := suite.getFinalPrice(asset)
	deviation := finalPrice.Sub(newLegitimatePrice).Abs().Quo(newLegitimatePrice)
	suite.Less(deviation.MustFloat64(), 0.1, "Final price should be close to legitimate price")
}

// TestSybilAttack simulates many fake validator identities
func (suite *OracleManipulationTestSuite) TestSybilAttack() {
	asset := "OSMO"
	legitimatePrice := math.LegacyNewDec(5)
	attackPrice := math.LegacyNewDec(10) // 100% higher

	// Create additional "fake" validators (Sybil attack)
	numSybils := 50
	sybilValidators := make([]*OracleValidator, numSybils)
	for i := 0; i < numSybils; i++ {
		sybilValidators[i] = &OracleValidator{
			Address:     fmt.Sprintf("pawvaloper1sybil%03d", i),
			VotingPower: 1, // Low voting power (if not staked properly)
			IsMalicious: true,
		}
	}

	// Legitimate validators submit correct price
	for _, v := range suite.validators {
		if !v.IsMalicious {
			_ = suite.submitPrice(asset, legitimatePrice, v.Address)
		}
	}

	// Sybil validators flood with attack price
	sybilAccepted := 0
	for _, v := range sybilValidators {
		err := suite.submitPriceWithPower(asset, attackPrice, v.Address, v.VotingPower)
		if err == nil {
			sybilAccepted++
		}
	}

	finalPrice := suite.getFinalPrice(asset)

	suite.T().Logf("TEST-1.4 Sybil Attack: Sybil submissions=%d, Accepted=%d",
		numSybils, sybilAccepted)
	suite.T().Logf("  → Final price: %s (legitimate: %s)", finalPrice.String(), legitimatePrice.String())

	// Sybil attack should fail due to voting power weighting
	deviation := finalPrice.Sub(legitimatePrice).Abs().Quo(legitimatePrice)
	suite.Less(deviation.MustFloat64(), 0.1, "Sybil attack should not significantly affect price")
}

// TestCoordinatedManipulation simulates all malicious validators coordinating
func (suite *OracleManipulationTestSuite) TestCoordinatedManipulation() {
	asset := "USDC"
	legitimatePrice := math.LegacyMustNewDecFromStr("1.00")
	attackPrice := math.LegacyMustNewDecFromStr("0.90") // Depeg attack

	// All malicious validators coordinate on same attack price
	maliciousSubmissions := 0
	legitimateSubmissions := 0

	for _, v := range suite.validators {
		if v.IsMalicious {
			_ = suite.submitPrice(asset, attackPrice, v.Address)
			maliciousSubmissions++
		} else {
			_ = suite.submitPrice(asset, legitimatePrice, v.Address)
			legitimateSubmissions++
		}
	}

	finalPrice := suite.getFinalPrice(asset)

	suite.T().Logf("TEST-1.4 Coordinated Attack: Malicious=%d, Legitimate=%d",
		maliciousSubmissions, legitimateSubmissions)
	suite.T().Logf("  → Final price: %s", finalPrice.String())

	// With 25% malicious vs 75% honest, attack should fail
	deviation := finalPrice.Sub(legitimatePrice).Abs().Quo(legitimatePrice)
	suite.Less(deviation.MustFloat64(), 0.05, "Coordinated attack should fail with minority")
}

// TestPriceOscillation simulates rapid price oscillation attack
func (suite *OracleManipulationTestSuite) TestPriceOscillation() {
	asset := "LINK"
	basePrice := math.LegacyNewDec(15)

	oscillations := 100
	highPrice := math.LegacyNewDec(18) // +20%
	lowPrice := math.LegacyNewDec(12)  // -20%

	acceptedOscillations := 0

	for i := 0; i < oscillations; i++ {
		for _, v := range suite.validators {
			if v.IsMalicious {
				price := highPrice
				if i%2 == 1 {
					price = lowPrice
				}
				err := suite.submitPrice(asset, price, v.Address)
				if err == nil {
					acceptedOscillations++
				}
			} else {
				// Legitimate validators submit stable price
				_ = suite.submitPrice(asset, basePrice, v.Address)
			}
		}
	}

	finalPrice := suite.getFinalPrice(asset)
	deviation := finalPrice.Sub(basePrice).Abs().Quo(basePrice)

	suite.T().Logf("TEST-1.4 Price Oscillation: Oscillations=%d, Accepted=%d",
		oscillations, acceptedOscillations)
	suite.T().Logf("  → Final price deviation: %.2f%%", deviation.MustFloat64()*100)

	// TWAP should smooth out oscillations
	suite.Less(deviation.MustFloat64(), 0.1, "TWAP should smooth oscillations")
}

// Helper methods

func (suite *OracleManipulationTestSuite) submitLegitimatePrice(asset string, price math.LegacyDec, rounds int) {
	for r := 0; r < rounds; r++ {
		for _, v := range suite.validators {
			if !v.IsMalicious {
				_ = suite.submitPrice(asset, price, v.Address)
			}
		}
		suite.advanceBlock()
	}
}

func (suite *OracleManipulationTestSuite) simulateAttack(asset, attackType string, priceFunc func(*OracleValidator) math.LegacyDec) ManipulationResult {
	startTime := time.Now()

	totalSubmissions := 0
	accepted := 0
	rejected := 0

	for round := 0; round < 10; round++ {
		for _, v := range suite.validators {
			price := priceFunc(v)
			err := suite.submitPrice(asset, price, v.Address)
			totalSubmissions++

			v.mu.Lock()
			v.Submissions++
			if err != nil {
				rejected++
				v.Rejections++
			} else {
				accepted++
			}
			v.mu.Unlock()
		}
		suite.advanceBlock()
	}

	finalPrice := suite.getFinalPrice(asset)
	legitimatePrice := priceFunc(suite.validators[len(suite.validators)-1]) // Non-malicious

	deviation := finalPrice.Sub(legitimatePrice).Abs().Quo(legitimatePrice)
	attackSuccess := deviation.MustFloat64() > 0.1 // >10% deviation = attack success

	return ManipulationResult{
		AttackType:        attackType,
		TotalSubmissions:  totalSubmissions,
		AcceptedPrices:    accepted,
		RejectedPrices:    rejected,
		PriceDeviationPct: deviation.MustFloat64() * 100,
		AttackSuccess:     attackSuccess,
		Duration:          time.Since(startTime),
	}
}

func (suite *OracleManipulationTestSuite) submitPrice(asset string, price math.LegacyDec, validator string) error {
	// Simulate price submission to oracle keeper
	return suite.k.SubmitPrice(suite.ctx, validator, asset, price)
}

func (suite *OracleManipulationTestSuite) submitPriceWithTimestamp(asset string, price math.LegacyDec, validator string, timestamp time.Time) error {
	// Simulate price submission with specific timestamp
	ctx := suite.ctx.WithBlockTime(timestamp)
	return suite.k.SubmitPrice(ctx, validator, asset, price)
}

func (suite *OracleManipulationTestSuite) submitPriceWithPower(asset string, price math.LegacyDec, validator string, power int64) error {
	// For Sybil test - validators with low power should be weighted less
	return suite.k.SubmitPriceWeighted(suite.ctx, validator, asset, price, power)
}

func (suite *OracleManipulationTestSuite) getFinalPrice(asset string) math.LegacyDec {
	price, err := suite.k.GetPrice(suite.ctx, asset)
	if err != nil {
		return math.LegacyZeroDec()
	}
	return price.Price
}

func (suite *OracleManipulationTestSuite) advanceBlock() {
	suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 1)
	suite.ctx = suite.ctx.WithBlockTime(suite.ctx.BlockTime().Add(5 * time.Second))
}

// TestManipulationSummary generates attack summary report
func (suite *OracleManipulationTestSuite) TestManipulationSummary() {
	suite.T().Log("\n=== TEST-1.4 ORACLE MANIPULATION SIMULATION SUMMARY ===")
	suite.T().Logf("Validator configuration:")
	suite.T().Logf("  → Total validators: %d", len(suite.validators))
	suite.T().Logf("  → Malicious rate: %.0f%%", suite.maliciousRate*100)
	suite.T().Logf("  → Byzantine threshold: 33%%")
	suite.T().Log("")
	suite.T().Log("Attack scenarios tested:")
	suite.T().Log("  ✓ Flash Crash Attack")
	suite.T().Log("  ✓ Pump and Dump")
	suite.T().Log("  ✓ Stale Data Attack")
	suite.T().Log("  ✓ Front-Running")
	suite.T().Log("  ✓ Sybil Attack")
	suite.T().Log("  ✓ Coordinated Manipulation")
	suite.T().Log("  ✓ Price Oscillation")
	suite.T().Log("")
	suite.T().Log("Defense mechanisms:")
	suite.T().Log("  → Voting power weighting")
	suite.T().Log("  → Price deviation limits")
	suite.T().Log("  → TWAP smoothing")
	suite.T().Log("  → Timestamp validation")
	suite.T().Log("  → Median aggregation")
	suite.T().Log("=== END SUMMARY ===\n")
}
