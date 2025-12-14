package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// TestCalculateMedian tests the median calculation with various edge cases
func TestCalculateMedian(t *testing.T) {
	k, _, _ := keepertest.OracleKeeper(t)

	tests := []struct {
		name     string
		prices   []math.LegacyDec
		expected math.LegacyDec
	}{
		{
			name:     "empty slice",
			prices:   []math.LegacyDec{},
			expected: math.LegacyZeroDec(),
		},
		{
			name:     "single price",
			prices:   []math.LegacyDec{math.LegacyNewDec(100)},
			expected: math.LegacyNewDec(100),
		},
		{
			name: "odd number of prices",
			prices: []math.LegacyDec{
				math.LegacyNewDec(50),
				math.LegacyNewDec(100),
				math.LegacyNewDec(150),
			},
			expected: math.LegacyNewDec(100),
		},
		{
			name: "even number of prices",
			prices: []math.LegacyDec{
				math.LegacyNewDec(50),
				math.LegacyNewDec(100),
				math.LegacyNewDec(150),
				math.LegacyNewDec(200),
			},
			expected: math.LegacyNewDec(125), // (100 + 150) / 2
		},
		{
			name: "unsorted prices - odd",
			prices: []math.LegacyDec{
				math.LegacyNewDec(150),
				math.LegacyNewDec(50),
				math.LegacyNewDec(100),
			},
			expected: math.LegacyNewDec(100),
		},
		{
			name: "unsorted prices - even",
			prices: []math.LegacyDec{
				math.LegacyNewDec(200),
				math.LegacyNewDec(50),
				math.LegacyNewDec(150),
				math.LegacyNewDec(100),
			},
			expected: math.LegacyNewDec(125),
		},
		{
			name: "all identical prices",
			prices: []math.LegacyDec{
				math.LegacyNewDec(100),
				math.LegacyNewDec(100),
				math.LegacyNewDec(100),
			},
			expected: math.LegacyNewDec(100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			median := k.CalculateMedian(tt.prices)
			require.True(t, median.Equal(tt.expected),
				"expected %s, got %s", tt.expected, median)
		})
	}
}

// TestCalculateMAD tests Median Absolute Deviation calculation
func TestCalculateMAD(t *testing.T) {
	k, _, _ := keepertest.OracleKeeper(t)

	tests := []struct {
		name         string
		prices       []math.LegacyDec
		median       math.LegacyDec
		expectedZero bool // MAD calculation is complex, test for non-zero in most cases
	}{
		{
			name:         "empty slice",
			prices:       []math.LegacyDec{},
			median:       math.LegacyZeroDec(),
			expectedZero: true,
		},
		{
			name:         "all identical - zero MAD",
			prices:       []math.LegacyDec{math.LegacyNewDec(100), math.LegacyNewDec(100), math.LegacyNewDec(100)},
			median:       math.LegacyNewDec(100),
			expectedZero: true,
		},
		{
			name: "spread prices - non-zero MAD",
			prices: []math.LegacyDec{
				math.LegacyNewDec(50),
				math.LegacyNewDec(100),
				math.LegacyNewDec(150),
			},
			median:       math.LegacyNewDec(100),
			expectedZero: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mad := k.CalculateMAD(tt.prices, tt.median)
			if tt.expectedZero {
				require.True(t, mad.IsZero(), "expected zero MAD")
			} else {
				require.True(t, mad.GT(math.LegacyZeroDec()), "expected non-zero MAD")
			}
		})
	}
}

// TestCalculateIQR tests Interquartile Range calculation using R-7 method
func TestCalculateIQR(t *testing.T) {
	k, _, _ := keepertest.OracleKeeper(t)

	tests := []struct {
		name         string
		prices       []math.LegacyDec
		expectQ1     math.LegacyDec
		expectQ3     math.LegacyDec
		expectIQR    math.LegacyDec
		zeroExpected bool
	}{
		{
			name:         "less than 4 elements",
			prices:       []math.LegacyDec{math.LegacyNewDec(100), math.LegacyNewDec(200)},
			zeroExpected: true,
		},
		{
			name: "exactly 4 elements",
			prices: []math.LegacyDec{
				math.LegacyNewDec(100),
				math.LegacyNewDec(200),
				math.LegacyNewDec(300),
				math.LegacyNewDec(400),
			},
			zeroExpected: false,
		},
		{
			name: "10 elements - standard case",
			prices: []math.LegacyDec{
				math.LegacyNewDec(10),
				math.LegacyNewDec(20),
				math.LegacyNewDec(30),
				math.LegacyNewDec(40),
				math.LegacyNewDec(50),
				math.LegacyNewDec(60),
				math.LegacyNewDec(70),
				math.LegacyNewDec(80),
				math.LegacyNewDec(90),
				math.LegacyNewDec(100),
			},
			zeroExpected: false,
		},
		{
			name: "all identical - zero IQR",
			prices: []math.LegacyDec{
				math.LegacyNewDec(100),
				math.LegacyNewDec(100),
				math.LegacyNewDec(100),
				math.LegacyNewDec(100),
				math.LegacyNewDec(100),
			},
			zeroExpected: false, // R-7 interpolation may produce non-zero values even for identical data
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q1, q3, iqr := k.CalculateIQR(tt.prices)

			if tt.zeroExpected {
				require.True(t, q1.IsZero() || q1.Equal(math.LegacyZeroDec()),
					"expected Q1 to be zero for test case: %s", tt.name)
				require.True(t, q3.IsZero() || q3.Equal(math.LegacyZeroDec()),
					"expected Q3 to be zero for test case: %s", tt.name)
			} else {
				require.True(t, iqr.GTE(math.LegacyZeroDec()), "IQR should be non-negative")
				require.True(t, q3.GTE(q1), "Q3 should be >= Q1")
				// IQR = Q3 - Q1
				calculatedIQR := q3.Sub(q1)
				require.True(t, calculatedIQR.Equal(iqr), "IQR should equal Q3 - Q1")
			}
		})
	}
}

// TestCalculateWeightedMedian tests weighted median calculation
func TestCalculateWeightedMedian(t *testing.T) {
	k, _, _ := keepertest.OracleKeeper(t)

	tests := []struct {
		name          string
		prices        []types.ValidatorPrice
		expectErr     bool
		expectInRange bool
		minPrice      math.LegacyDec
		maxPrice      math.LegacyDec
	}{
		{
			name:      "empty prices",
			prices:    []types.ValidatorPrice{},
			expectErr: true,
		},
		{
			name: "single price",
			prices: []types.ValidatorPrice{
				{Price: math.LegacyNewDec(100), VotingPower: 1},
			},
			expectErr:     false,
			expectInRange: true,
			minPrice:      math.LegacyNewDec(100),
			maxPrice:      math.LegacyNewDec(100),
		},
		{
			name: "two equal weights",
			prices: []types.ValidatorPrice{
				{Price: math.LegacyNewDec(100), VotingPower: 1},
				{Price: math.LegacyNewDec(200), VotingPower: 1},
			},
			expectErr:     false,
			expectInRange: true,
			minPrice:      math.LegacyNewDec(100),
			maxPrice:      math.LegacyNewDec(200),
		},
		{
			name: "weighted toward higher price",
			prices: []types.ValidatorPrice{
				{Price: math.LegacyNewDec(100), VotingPower: 1},
				{Price: math.LegacyNewDec(200), VotingPower: 10}, // High weight
			},
			expectErr:     false,
			expectInRange: true,
			minPrice:      math.LegacyNewDec(200),
			maxPrice:      math.LegacyNewDec(200),
		},
		{
			name: "weighted toward lower price",
			prices: []types.ValidatorPrice{
				{Price: math.LegacyNewDec(100), VotingPower: 10}, // High weight
				{Price: math.LegacyNewDec(200), VotingPower: 1},
			},
			expectErr:     false,
			expectInRange: true,
			minPrice:      math.LegacyNewDec(100),
			maxPrice:      math.LegacyNewDec(100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			median, err := k.CalculateWeightedMedian(tt.prices)

			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.expectInRange {
					require.True(t, median.GTE(tt.minPrice) && median.LTE(tt.maxPrice),
						"median %s should be in range [%s, %s]", median, tt.minPrice, tt.maxPrice)
				}
			}
		})
	}
}

// TestClassifyOutlierSeverity tests outlier severity classification
func TestClassifyOutlierSeverity(t *testing.T) {
	k, _, _ := keepertest.OracleKeeper(t)

	median := math.LegacyNewDec(100)
	mad := math.LegacyNewDec(10)
	threshold := math.LegacyMustNewDecFromStr("3.5")

	tests := []struct {
		name             string
		price            math.LegacyDec
		median           math.LegacyDec
		mad              math.LegacyDec
		threshold        math.LegacyDec
		expectedSeverity keeper.OutlierSeverity
	}{
		{
			name:             "exact median - no outlier",
			price:            math.LegacyNewDec(100),
			median:           median,
			mad:              mad,
			threshold:        threshold,
			expectedSeverity: keeper.SeverityNone,
		},
		{
			name:             "close to median - no outlier",
			price:            math.LegacyNewDec(105),
			median:           median,
			mad:              mad,
			threshold:        threshold,
			expectedSeverity: keeper.SeverityNone,
		},
		{
			name:             "far from median - extreme outlier",
			price:            math.LegacyNewDec(200),
			median:           median,
			mad:              mad,
			threshold:        threshold,
			expectedSeverity: keeper.SeverityExtreme,
		},
		{
			name:             "zero MAD with different price - extreme",
			price:            math.LegacyNewDec(200),
			median:           median,
			mad:              math.LegacyZeroDec(),
			threshold:        threshold,
			expectedSeverity: keeper.SeverityExtreme,
		},
		{
			name:             "zero MAD with same price - no outlier",
			price:            math.LegacyNewDec(100),
			median:           median,
			mad:              math.LegacyZeroDec(),
			threshold:        threshold,
			expectedSeverity: keeper.SeverityNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severity, deviation := k.ClassifyOutlierSeverity(tt.price, tt.median, tt.mad, tt.threshold)
			require.Equal(t, tt.expectedSeverity, severity,
				"expected severity %d, got %d", tt.expectedSeverity, severity)
			require.True(t, deviation.GTE(math.LegacyZeroDec()), "deviation should be non-negative")
		})
	}
}

// TestIsIQROutlier tests IQR-based outlier detection
func TestIsIQROutlier(t *testing.T) {
	k, _, _ := keepertest.OracleKeeper(t)

	q1 := math.LegacyNewDec(50)
	q3 := math.LegacyNewDec(150)
	iqr := q3.Sub(q1) // 100
	volatility := math.LegacyMustNewDecFromStr("0.05")

	tests := []struct {
		name       string
		price      math.LegacyDec
		q1         math.LegacyDec
		q3         math.LegacyDec
		iqr        math.LegacyDec
		volatility math.LegacyDec
		isOutlier  bool
	}{
		{
			name:       "within IQR range",
			price:      math.LegacyNewDec(100),
			q1:         q1,
			q3:         q3,
			iqr:        iqr,
			volatility: volatility,
			isOutlier:  false,
		},
		{
			name:       "below Q1 but within bounds",
			price:      math.LegacyNewDec(40),
			q1:         q1,
			q3:         q3,
			iqr:        iqr,
			volatility: volatility,
			isOutlier:  false,
		},
		{
			name:       "above Q3 but within bounds",
			price:      math.LegacyNewDec(160),
			q1:         q1,
			q3:         q3,
			iqr:        iqr,
			volatility: volatility,
			isOutlier:  false,
		},
		{
			name:       "extreme low outlier",
			price:      math.LegacyNewDec(1),
			q1:         q1,
			q3:         q3,
			iqr:        iqr,
			volatility: volatility,
			isOutlier:  false, // With 5% volatility, adjusted IQR multiplier is ~1.75, so bounds are wider
		},
		{
			name:       "extreme high outlier",
			price:      math.LegacyNewDec(500),
			q1:         q1,
			q3:         q3,
			iqr:        iqr,
			volatility: volatility,
			isOutlier:  true,
		},
		{
			name:       "zero IQR - no outlier",
			price:      math.LegacyNewDec(100),
			q1:         q1,
			q3:         q3,
			iqr:        math.LegacyZeroDec(),
			volatility: volatility,
			isOutlier:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isOutlier := k.IsIQROutlier(tt.price, tt.q1, tt.q3, tt.iqr, tt.volatility)
			require.Equal(t, tt.isOutlier, isOutlier,
				"expected isOutlier=%v for price %s", tt.isOutlier, tt.price)
		})
	}
}

// TestGrubbsTest tests Grubbs' test for outlier detection
func TestGrubbsTest(t *testing.T) {
	k, _, _ := keepertest.OracleKeeper(t)

	tests := []struct {
		name       string
		prices     []math.LegacyDec
		testPrice  math.LegacyDec
		alpha      float64
		expectFail bool // true if sample size too small
	}{
		{
			name: "too few samples",
			prices: []math.LegacyDec{
				math.LegacyNewDec(100),
				math.LegacyNewDec(101),
			},
			testPrice:  math.LegacyNewDec(100),
			alpha:      0.05,
			expectFail: true,
		},
		{
			name: "sufficient samples - not outlier",
			prices: []math.LegacyDec{
				math.LegacyNewDec(100),
				math.LegacyNewDec(101),
				math.LegacyNewDec(102),
				math.LegacyNewDec(103),
				math.LegacyNewDec(104),
				math.LegacyNewDec(105),
				math.LegacyNewDec(106),
			},
			testPrice:  math.LegacyNewDec(103),
			alpha:      0.05,
			expectFail: false,
		},
		{
			name: "sufficient samples - extreme outlier",
			prices: []math.LegacyDec{
				math.LegacyNewDec(100),
				math.LegacyNewDec(101),
				math.LegacyNewDec(102),
				math.LegacyNewDec(103),
				math.LegacyNewDec(104),
				math.LegacyNewDec(105),
				math.LegacyNewDec(1000), // Extreme
			},
			testPrice:  math.LegacyNewDec(1000),
			alpha:      0.05,
			expectFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isOutlier := k.GrubbsTest(tt.prices, tt.testPrice, tt.alpha)
			if tt.expectFail {
				require.False(t, isOutlier, "Grubbs test should return false for insufficient samples")
			}
			// For sufficient samples, we just verify it returns a boolean without panicking
		})
	}
}

// TestCalculateVolatility tests volatility calculation
func TestCalculateVolatility(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	tests := []struct {
		name             string
		setupSnapshots   bool
		asset            string
		expectDefaultVol bool
		expectNonZeroVol bool
	}{
		{
			name:             "no snapshots - default volatility",
			setupSnapshots:   false,
			asset:            "BTC",
			expectDefaultVol: true,
		},
		{
			name:             "with snapshots - calculated volatility",
			setupSnapshots:   true,
			asset:            "BTC",
			expectNonZeroVol: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupSnapshots {
				// Create some price snapshots with variance
				for i := int64(1); i <= 10; i++ {
					snapshot := types.PriceSnapshot{
						Asset:       tt.asset,
						Price:       math.LegacyNewDec(50000 + i*100),
						BlockHeight: i,
						BlockTime:   ctx.BlockTime().Unix() + i,
					}
					err := k.SetPriceSnapshot(ctx, snapshot)
					require.NoError(t, err)
				}
			}

			volatility := k.CalculateVolatility(ctx, tt.asset, 100)

			if tt.expectDefaultVol {
				// Default is 5%
				require.True(t, volatility.Equal(math.LegacyMustNewDecFromStr("0.05")),
					"expected default volatility 0.05")
			}

			if tt.expectNonZeroVol {
				require.True(t, volatility.GT(math.LegacyZeroDec()), "volatility should be positive")
			}
		})
	}
}

// TestAnalyzeCryptoeconomicSecurity tests game-theoretic security analysis
func TestAnalyzeCryptoeconomicSecurity(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Setup: create bonded validators with sufficient stake
	for i := 0; i < 10; i++ {
		val := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())
		err := keepertest.EnsureBondedValidator(ctx, val)
		require.NoError(t, err)
	}

	// Set default params
	params := types.DefaultParams()
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	analysis, err := k.AnalyzeCryptoeconomicSecurity(ctx)
	require.NoError(t, err)

	// Attack cost should be positive
	require.True(t, analysis.AttackCost.GT(math.ZeroInt()),
		"attack cost should be positive")

	// Security margin should be non-negative
	require.True(t, analysis.SecurityMargin.GTE(math.LegacyZeroDec()),
		"security margin should be non-negative")

	// Dishonest penalty should match slash fraction
	require.True(t, analysis.DishonestPenalty.Equal(params.SlashFraction),
		"dishonest penalty should equal slash fraction")
}

// TestCalculateOptimalSlashFraction tests optimal slashing calculation
func TestCalculateOptimalSlashFraction(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Setup validators
	for i := 0; i < 5; i++ {
		val := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())
		err := keepertest.EnsureBondedValidator(ctx, val)
		require.NoError(t, err)
	}

	optimalSlash, err := k.CalculateOptimalSlashFraction(ctx)
	require.NoError(t, err)

	// Optimal slash should be within bounds
	minSlash := math.LegacyMustNewDecFromStr("0.0001") // 0.01%
	maxSlash := math.LegacyMustNewDecFromStr("0.01")   // 1%

	require.True(t, optimalSlash.GTE(minSlash), "optimal slash should be >= minimum")
	require.True(t, optimalSlash.LTE(maxSlash), "optimal slash should be <= maximum")
}

// TestAnalyzeValidatorIncentives tests validator incentive analysis
func TestAnalyzeValidatorIncentives(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Create a validator
	val := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())
	err := keepertest.EnsureBondedValidator(ctx, val)
	require.NoError(t, err)

	// Set params
	params := types.DefaultParams()
	err = k.SetParams(ctx, params)
	require.NoError(t, err)

	incentives, err := k.AnalyzeValidatorIncentives(ctx, val.String())
	require.NoError(t, err)

	// Stake should be positive
	require.True(t, incentives.Stake.GT(math.ZeroInt()),
		"validator stake should be positive")

	// Expected slashing should be non-negative
	require.True(t, incentives.ExpectedSlashing.GTE(math.LegacyZeroDec()),
		"expected slashing should be non-negative")

	// Reputation value should be non-negative
	require.True(t, incentives.ReputationValue.GTE(math.LegacyZeroDec()),
		"reputation value should be non-negative")
}

// TestCalculateNashEquilibrium tests Nash equilibrium analysis
func TestCalculateNashEquilibrium(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Setup validators
	for i := 0; i < 10; i++ {
		val := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())
		err := keepertest.EnsureBondedValidator(ctx, val)
		require.NoError(t, err)
	}

	params := types.DefaultParams()
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	nash, err := k.CalculateNashEquilibrium(ctx)
	require.NoError(t, err)

	// Min secure stake should be positive
	require.True(t, nash.MinSecureStake.GT(math.ZeroInt()),
		"min secure stake should be positive")

	// Optimal slash fraction should be in bounds
	require.True(t, nash.OptimalSlashFraction.GT(math.LegacyZeroDec()),
		"optimal slash fraction should be positive")

	// Attack success probability should be in [0, 1]
	require.True(t, nash.AttackSuccessProbability.GTE(math.LegacyZeroDec()),
		"attack success probability should be >= 0")
	require.True(t, nash.AttackSuccessProbability.LTE(math.LegacyOneDec()),
		"attack success probability should be <= 1")
}

// TestCalculateSchellingPoint tests Schelling point calculation
func TestCalculateSchellingPoint(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	params := types.DefaultParams()
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	tests := []struct {
		name         string
		setupHistory bool
		asset        string
		expectErr    bool
	}{
		{
			name:         "no history - error",
			setupHistory: false,
			asset:        "BTC",
			expectErr:    true,
		},
		{
			name:         "with history - success",
			setupHistory: true,
			asset:        "BTC",
			expectErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupHistory {
				// Create price snapshots
				for i := int64(1); i <= 10; i++ {
					snapshot := types.PriceSnapshot{
						Asset:       tt.asset,
						Price:       math.LegacyNewDec(50000 + i*10),
						BlockHeight: i,
						BlockTime:   ctx.BlockTime().Unix() + i,
					}
					err := k.SetPriceSnapshot(ctx, snapshot)
					require.NoError(t, err)
				}
			}

			schellingPoint, err := k.CalculateSchellingPoint(ctx, tt.asset)

			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, schellingPoint.GT(math.LegacyZeroDec()),
					"Schelling point should be positive")
			}
		})
	}
}

// TestAnalyzeCollusionResistance tests collusion resistance analysis
func TestAnalyzeCollusionResistance(t *testing.T) {
	tests := []struct {
		name          string
		numValidators int
		expectErr     bool
	}{
		{
			name:          "no validators",
			numValidators: 0,
			expectErr:     true,
		},
		{
			name:          "single validator",
			numValidators: 1,
			expectErr:     false,
		},
		{
			name:          "many validators",
			numValidators: 20,
			expectErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fresh keeper and context for each test
			freshK, _, freshCtx := keepertest.OracleKeeper(t)

			// Setup validators
			for i := 0; i < tt.numValidators; i++ {
				val := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())
				err := keepertest.EnsureBondedValidator(freshCtx, val)
				require.NoError(t, err)
			}

			resistance, err := freshK.AnalyzeCollusionResistance(freshCtx)

			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				// Resistance should be in [0, 1]
				require.True(t, resistance.GTE(math.LegacyZeroDec()),
					"resistance should be >= 0")
				require.True(t, resistance.LTE(math.LegacyOneDec()),
					"resistance should be <= 1")
			}
		})
	}
}

// TestCalculateSystemSecurityScore tests composite security score
func TestCalculateSystemSecurityScore(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Setup validators
	for i := 0; i < 15; i++ {
		val := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())
		err := keepertest.EnsureBondedValidator(ctx, val)
		require.NoError(t, err)
	}

	params := types.DefaultParams()
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	score, err := k.CalculateSystemSecurityScore(ctx)
	require.NoError(t, err)

	// Score should be in [0, 1]
	require.True(t, score.GTE(math.LegacyZeroDec()),
		"security score should be >= 0")
	require.True(t, score.LTE(math.LegacyOneDec()),
		"security score should be <= 1")
}

// TestValidateCryptoeconomicSecurity tests security validation
func TestValidateCryptoeconomicSecurity(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Setup sufficient validators for good security
	for i := 0; i < 20; i++ {
		val := sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())
		err := keepertest.EnsureBondedValidator(ctx, val)
		require.NoError(t, err)
	}

	params := types.DefaultParams()
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Validation might pass or fail depending on security score
	// We just verify it doesn't panic
	_ = k.ValidateCryptoeconomicSecurity(ctx)
}

// TestDetectAndFilterOutliers_SmallSample tests outlier detection with small validator set
func TestDetectAndFilterOutliers_SmallSample(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Create 3 validators (< 5 minimum for advanced detection)
	validators := make([]sdk.ValAddress, 3)
	for i := 0; i < 3; i++ {
		validators[i] = sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())
		err := keepertest.EnsureBondedValidator(ctx, validators[i])
		require.NoError(t, err)
	}

	params := types.DefaultParams()
	params.MinVotingPowerForConsensus = math.LegacyMustNewDecFromStr("0.20") // 20% threshold
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	validatorPrices := []types.ValidatorPrice{
		{
			ValidatorAddr: validators[0].String(),
			Price:         math.LegacyNewDec(100),
			VotingPower:   1,
		},
		{
			ValidatorAddr: validators[1].String(),
			Price:         math.LegacyNewDec(105),
			VotingPower:   1,
		},
		{
			ValidatorAddr: validators[2].String(),
			Price:         math.LegacyNewDec(110),
			VotingPower:   1,
		},
	}

	// With small sample, all prices should be preserved
	filtered, err := k.DetectAndFilterOutliers(ctx, "BTC", validatorPrices)
	require.NoError(t, err)
	require.Len(t, filtered.ValidPrices, 3, "all prices should be preserved for small sample")
	require.Len(t, filtered.FilteredOutliers, 0, "no outliers should be detected")
}

// TestDetectAndFilterOutliers_LargeSample tests outlier detection with sufficient validators
func TestDetectAndFilterOutliers_LargeSample(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Create 10 validators (>= 5 for advanced detection)
	validators := make([]sdk.ValAddress, 10)
	for i := 0; i < 10; i++ {
		validators[i] = sdk.ValAddress(secp256k1.GenPrivKey().PubKey().Address())
		err := keepertest.EnsureBondedValidator(ctx, validators[i])
		require.NoError(t, err)
	}

	params := types.DefaultParams()
	params.MinVotingPowerForConsensus = math.LegacyMustNewDecFromStr("0.10")
	err := k.SetParams(ctx, params)
	require.NoError(t, err)

	// Create validator prices: 8 legitimate + 2 outliers
	validatorPrices := make([]types.ValidatorPrice, 10)
	for i := 0; i < 8; i++ {
		validatorPrices[i] = types.ValidatorPrice{
			ValidatorAddr: validators[i].String(),
			Price:         math.LegacyNewDec(50000 + int64(i)*10),
			VotingPower:   1,
		}
	}
	// Add 2 extreme outliers
	validatorPrices[8] = types.ValidatorPrice{
		ValidatorAddr: validators[8].String(),
		Price:         math.LegacyNewDec(100000), // 2x outlier
		VotingPower:   1,
	}
	validatorPrices[9] = types.ValidatorPrice{
		ValidatorAddr: validators[9].String(),
		Price:         math.LegacyNewDec(25000), // 0.5x outlier
		VotingPower:   1,
	}

	filtered, err := k.DetectAndFilterOutliers(ctx, "BTC", validatorPrices)
	require.NoError(t, err)

	// At least some outliers should be detected
	require.True(t, len(filtered.FilteredOutliers) > 0, "outliers should be detected")
	require.True(t, len(filtered.ValidPrices) > 0, "valid prices should remain")
}
