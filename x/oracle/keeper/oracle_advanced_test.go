package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/keeper"
)

// Helper function to create a test validator address
func createTestValAddr(t *testing.T, index int) sdk.ValAddress {
	addr := make([]byte, 20)
	copy(addr, []byte("testvaloper______"))
	addr[19] = byte(index)
	return sdk.ValAddress(addr)
}

// TestAggregateWeightedPrices tests stake and reputation weighted aggregation
func TestAggregateWeightedPrices(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)

	tests := []struct {
		name      string
		prices    []keeper.WeightedPrice
		expectErr bool
		errMsg    string
	}{
		{
			name:      "empty prices",
			prices:    []keeper.WeightedPrice{},
			expectErr: true,
			errMsg:    "no prices to aggregate",
		},
		{
			name: "single price",
			prices: []keeper.WeightedPrice{
				{
					Price:      math.LegacyNewDec(50000),
					Validator:  "val1",
					Stake:      math.NewInt(1000000),
					Reputation: math.LegacyOneDec(),
				},
			},
			expectErr: false,
		},
		{
			name: "multiple prices with equal weights",
			prices: []keeper.WeightedPrice{
				{
					Price:      math.LegacyNewDec(50000),
					Validator:  "val1",
					Stake:      math.NewInt(1000000),
					Reputation: math.LegacyOneDec(),
				},
				{
					Price:      math.LegacyNewDec(52000),
					Validator:  "val2",
					Stake:      math.NewInt(1000000),
					Reputation: math.LegacyOneDec(),
				},
			},
			expectErr: false,
		},
		{
			name: "multiple prices with different weights",
			prices: []keeper.WeightedPrice{
				{
					Price:      math.LegacyNewDec(50000),
					Validator:  "val1",
					Stake:      math.NewInt(2000000), // Higher stake
					Reputation: math.LegacyOneDec(),
				},
				{
					Price:      math.LegacyNewDec(52000),
					Validator:  "val2",
					Stake:      math.NewInt(1000000),
					Reputation: math.LegacyOneDec(),
				},
			},
			expectErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := k.AggregateWeightedPrices(ctx, "BTC", tc.prices)

			if tc.expectErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					require.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				require.True(t, result.IsPositive())
			}
		})
	}
}

// TestDetectOutliersAdvanced tests outlier detection
func TestDetectOutliersAdvanced(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)

	tests := []struct {
		name           string
		prices         []math.LegacyDec
		config         keeper.OutlierDetectionConfig
		expectErr      bool
		errMsg         string
		minOutliers    int
		maxOutliers    int
	}{
		{
			name:      "insufficient data points",
			prices:    []math.LegacyDec{math.LegacyNewDec(100), math.LegacyNewDec(101)},
			config:    keeper.OutlierDetectionConfig{MinDataPoints: 5, StdDevMultiplier: math.LegacyNewDec(3), UseMAD: true},
			expectErr: true,
			errMsg:    "need at least",
		},
		{
			name: "no outliers - tight distribution",
			prices: []math.LegacyDec{
				math.LegacyNewDec(100),
				math.LegacyNewDec(101),
				math.LegacyNewDec(102),
				math.LegacyNewDec(100),
				math.LegacyNewDec(101),
			},
			config:      keeper.OutlierDetectionConfig{MinDataPoints: 5, StdDevMultiplier: math.LegacyNewDec(3), UseMAD: true},
			expectErr:   false,
			minOutliers: 0,
			maxOutliers: 0,
		},
		{
			name: "one obvious outlier",
			prices: []math.LegacyDec{
				math.LegacyNewDec(100),
				math.LegacyNewDec(101),
				math.LegacyNewDec(102),
				math.LegacyNewDec(100),
				math.LegacyNewDec(1000), // Outlier
			},
			config:      keeper.OutlierDetectionConfig{MinDataPoints: 5, StdDevMultiplier: math.LegacyNewDec(3), UseMAD: true},
			expectErr:   false,
			minOutliers: 1,
			maxOutliers: 1,
		},
		{
			name: "using standard deviation - no outliers detected",
			prices: []math.LegacyDec{
				math.LegacyNewDec(100),
				math.LegacyNewDec(101),
				math.LegacyNewDec(102),
				math.LegacyNewDec(100),
				math.LegacyNewDec(103),
			},
			config:      keeper.OutlierDetectionConfig{MinDataPoints: 5, StdDevMultiplier: math.LegacyNewDec(2), UseMAD: false},
			expectErr:   false,
			minOutliers: 0,
			maxOutliers: 0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			outliers, err := k.DetectOutliersAdvanced(ctx, tc.prices, tc.config)

			if tc.expectErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					require.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
				require.GreaterOrEqual(t, len(outliers), tc.minOutliers)
				require.LessOrEqual(t, len(outliers), tc.maxOutliers)
			}
		})
	}
}

// TestValidateDataFreshness tests data freshness validation
func TestValidateDataFreshness(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)

	tests := []struct {
		name        string
		asset       string
		submittedAt int64
		expectErr   bool
	}{
		{
			name:        "recent submission",
			asset:       "BTC",
			submittedAt: ctx.BlockTime().Unix() - 10, // 10 seconds ago
			expectErr:   false,
		},
		{
			name:        "stale submission",
			asset:       "ETH",
			submittedAt: ctx.BlockTime().Unix() - 400, // 400 seconds ago (over 5 min limit)
			expectErr:   true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			k2, _, ctx2 := keepertest.OracleKeeper(t)

			err := k2.ValidateDataFreshness(ctx2, tc.asset, tc.submittedAt)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "submission too old")
			} else {
				require.NoError(t, err)
			}
		})
	}

	_ = k  // Acknowledge outer keeper
	_ = ctx // Acknowledge outer context
}

// TestIsActiveOracleValidator tests active validator check
func TestIsActiveOracleValidator(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)

	validator := createTestValAddr(t, 1)

	// Initially not active
	isActive := k.IsActiveOracleValidator(ctx, validator)
	require.False(t, isActive)
}

// TestCommitRevealPrice tests commit-reveal scheme
func TestCommitRevealPrice(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)
	validator := createTestValAddr(t, 1)

	tests := []struct {
		name      string
		priceHash []byte
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid hash",
			priceHash: make([]byte, 32), // 32 bytes = valid
			expectErr: false,
		},
		{
			name:      "invalid hash - too short",
			priceHash: make([]byte, 16),
			expectErr: true,
			errMsg:    "must be 32 bytes",
		},
		{
			name:      "invalid hash - too long",
			priceHash: make([]byte, 64),
			expectErr: true,
			errMsg:    "must be 32 bytes",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			k2, _, ctx2 := keepertest.OracleKeeper(t)

			err := k2.CommitPrice(ctx2, validator, "BTC", tc.priceHash)

			if tc.expectErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					require.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}

	_ = k   // Acknowledge outer keeper
	_ = ctx // Acknowledge outer context
}

// TestTrackValidatorActivity tests validator activity tracking
func TestTrackValidatorActivity(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)
	validator := createTestValAddr(t, 1)

	// Track a successful submission
	err := k.TrackValidatorActivity(ctx, validator, true)
	require.NoError(t, err)

	// Track failed submissions
	for i := 0; i < 5; i++ {
		err := k.TrackValidatorActivity(ctx, validator, false)
		require.NoError(t, err)
	}
}

// TestSubmitEncryptedPrice tests encrypted price submission
func TestSubmitEncryptedPrice(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)
	validator := createTestValAddr(t, 1)

	tests := []struct {
		name          string
		encryptedData []byte
		nonce         []byte
		expectErr     bool
		errMsg        string
	}{
		{
			name:          "valid encrypted submission",
			encryptedData: []byte("encrypted_price_data_here"),
			nonce:         []byte("random_nonce_123"),
			expectErr:     false,
		},
		{
			name:          "empty encrypted data",
			encryptedData: []byte{},
			nonce:         []byte("random_nonce_123"),
			expectErr:     true,
			errMsg:        "encrypted data and nonce required",
		},
		{
			name:          "empty nonce",
			encryptedData: []byte("encrypted_price_data_here"),
			nonce:         []byte{},
			expectErr:     true,
			errMsg:        "encrypted data and nonce required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			k2, _, ctx2 := keepertest.OracleKeeper(t)

			err := k2.SubmitEncryptedPrice(ctx2, validator, "BTC", tc.encryptedData, tc.nonce)

			if tc.expectErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					require.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}

	_ = k   // Acknowledge outer keeper
	_ = ctx // Acknowledge outer context
}

// TestRegisterPriceSource tests price source registration
func TestRegisterPriceSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		source    keeper.PriceSource
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid source",
			source: keeper.PriceSource{
				Name:      "Binance",
				URL:       "https://api.binance.com/v3/ticker",
				PublicKey: []byte("public_key_here"),
			},
			expectErr: false,
		},
		{
			name: "empty name",
			source: keeper.PriceSource{
				Name:      "",
				URL:       "https://api.binance.com/v3/ticker",
				PublicKey: []byte("public_key_here"),
			},
			expectErr: true,
			errMsg:    "invalid source name",
		},
		{
			name: "name too long",
			source: keeper.PriceSource{
				Name:      "this_is_a_very_long_source_name_that_exceeds_the_maximum_allowed_length_of_64_characters_limit",
				URL:       "https://api.binance.com/v3/ticker",
				PublicKey: []byte("public_key_here"),
			},
			expectErr: true,
			errMsg:    "invalid source name",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			k, _, ctx := keepertest.OracleKeeper(t)

			err := k.RegisterPriceSource(ctx, tc.source)

			if tc.expectErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					require.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestGetValidatorAccuracy tests validator accuracy retrieval
func TestGetValidatorAccuracy(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)
	validator := createTestValAddr(t, 1)

	// Get accuracy for new validator (should return default values)
	accuracy, err := k.GetValidatorAccuracy(ctx, validator)
	require.NoError(t, err)
	require.NotNil(t, accuracy)
	require.Equal(t, uint64(0), accuracy.TotalSubmissions)
	require.True(t, accuracy.AccuracyScore.Equal(math.LegacyNewDec(50))) // Neutral score
}

// TestSetValidatorAccuracy tests setting validator accuracy
func TestSetValidatorAccuracy(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)
	validator := createTestValAddr(t, 1)

	accuracy := &keeper.ValidatorAccuracy{
		TotalSubmissions:    100,
		AccurateSubmissions: 90,
		TotalDeviation:      math.LegacyNewDec(50),
		AccuracyScore:       math.LegacyNewDec(90),
		ConsecutiveAccurate: 5,
		LastUpdatedHeight:   1000,
	}

	err := k.SetValidatorAccuracy(ctx, validator, accuracy)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := k.GetValidatorAccuracy(ctx, validator)
	require.NoError(t, err)
	require.Equal(t, accuracy.TotalSubmissions, retrieved.TotalSubmissions)
	require.Equal(t, accuracy.AccurateSubmissions, retrieved.AccurateSubmissions)
	require.True(t, accuracy.AccuracyScore.Equal(retrieved.AccuracyScore))
}

// TestUpdateValidatorAccuracy tests accuracy update after price submission
func TestUpdateValidatorAccuracy(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)
	validator := createTestValAddr(t, 1)

	// Accurate submission (within 1%)
	err := k.UpdateValidatorAccuracy(ctx, validator, math.LegacyNewDec(100), math.LegacyNewDec(100))
	require.NoError(t, err)

	accuracy, err := k.GetValidatorAccuracy(ctx, validator)
	require.NoError(t, err)
	require.Equal(t, uint64(1), accuracy.TotalSubmissions)
	require.Equal(t, uint64(1), accuracy.AccurateSubmissions)
	require.Equal(t, uint64(1), accuracy.ConsecutiveAccurate)

	// Inaccurate submission (10% off)
	err = k.UpdateValidatorAccuracy(ctx, validator, math.LegacyNewDec(110), math.LegacyNewDec(100))
	require.NoError(t, err)

	accuracy, err = k.GetValidatorAccuracy(ctx, validator)
	require.NoError(t, err)
	require.Equal(t, uint64(2), accuracy.TotalSubmissions)
	require.Equal(t, uint64(1), accuracy.AccurateSubmissions) // No increase
	require.Equal(t, uint64(0), accuracy.ConsecutiveAccurate) // Reset
}

// TestCalculateAccuracyBonus tests accuracy bonus calculation
func TestCalculateAccuracyBonus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		accuracyScore math.LegacyDec
		minBonus      math.LegacyDec
		maxBonus      math.LegacyDec
	}{
		{
			name:          "low accuracy - no bonus",
			accuracyScore: math.LegacyNewDec(40),
			minBonus:      math.LegacyOneDec(),
			maxBonus:      math.LegacyOneDec(),
		},
		{
			name:          "medium accuracy - 50",
			accuracyScore: math.LegacyNewDec(50),
			minBonus:      math.LegacyOneDec(),
			maxBonus:      math.LegacyOneDec(),
		},
		{
			name:          "good accuracy - 75",
			accuracyScore: math.LegacyNewDec(75),
			minBonus:      math.LegacyNewDecWithPrec(120, 2), // ~1.2
			maxBonus:      math.LegacyNewDecWithPrec(130, 2), // ~1.3
		},
		{
			name:          "excellent accuracy - 95",
			accuracyScore: math.LegacyNewDec(95),
			minBonus:      math.LegacyNewDecWithPrec(150, 2), // 1.5
			maxBonus:      math.LegacyNewDec(2),              // 2.0 max
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			k, _, ctx := keepertest.OracleKeeper(t)
			validator := createTestValAddr(t, 1)

			// Set accuracy score
			accuracy := &keeper.ValidatorAccuracy{
				AccuracyScore: tc.accuracyScore,
			}
			err := k.SetValidatorAccuracy(ctx, validator, accuracy)
			require.NoError(t, err)

			// Calculate bonus
			bonus, err := k.CalculateAccuracyBonus(ctx, validator)
			require.NoError(t, err)
			require.True(t, bonus.GTE(tc.minBonus), "bonus %s should be >= %s", bonus, tc.minBonus)
			require.True(t, bonus.LTE(tc.maxBonus), "bonus %s should be <= %s", bonus, tc.maxBonus)
		})
	}
}

// TestGetValidatorGeographicInfo tests geographic info retrieval
func TestGetValidatorGeographicInfo(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)
	validator := createTestValAddr(t, 1)

	// Get info for new validator (should return defaults)
	info, err := k.GetValidatorGeographicInfo(ctx, validator)
	require.NoError(t, err)
	require.NotNil(t, info)
	require.Equal(t, keeper.RegionUnknown, info.Region)
	require.False(t, info.IsVerified)
}

// TestSetValidatorGeographicInfo tests setting geographic info
func TestSetValidatorGeographicInfo(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)
	validator := createTestValAddr(t, 1)

	info := &keeper.GeographicInfo{
		Region:     keeper.RegionEurope,
		Country:    "DE",
		Timezone:   "Europe/Berlin",
		IsVerified: true,
	}

	err := k.SetValidatorGeographicInfo(ctx, validator, info)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := k.GetValidatorGeographicInfo(ctx, validator)
	require.NoError(t, err)
	require.Equal(t, info.Region, retrieved.Region)
	require.Equal(t, info.Country, retrieved.Country)
	require.Equal(t, info.Timezone, retrieved.Timezone)
	// RegistrationTime is set from block time, verify it was updated (not default zero)
	require.NotEqual(t, int64(0), retrieved.RegistrationTime)
}

// TestGeographicDiversityScore tests diversity score calculation
func TestGeographicDiversityScore(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)

	tests := []struct {
		name       string
		regions    []string
		expectZero bool
	}{
		{
			name:       "empty validators",
			regions:    []string{},
			expectZero: true,
		},
		{
			name:       "single region",
			regions:    []string{keeper.RegionEurope, keeper.RegionEurope, keeper.RegionEurope},
			expectZero: false,
		},
		{
			name:       "diverse regions",
			regions:    []string{keeper.RegionEurope, keeper.RegionAsia, keeper.RegionNorthAmerica},
			expectZero: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			k2, _, ctx2 := keepertest.OracleKeeper(t)

			// Create validators with regions
			validators := make([]sdk.ValAddress, len(tc.regions))
			for i, region := range tc.regions {
				validators[i] = createTestValAddr(t, i)
				info := &keeper.GeographicInfo{Region: region}
				err := k2.SetValidatorGeographicInfo(ctx2, validators[i], info)
				require.NoError(t, err)
			}

			score, err := k2.GeographicDiversityScore(ctx2, validators)
			require.NoError(t, err)

			if tc.expectZero {
				require.True(t, score.IsZero())
			} else {
				require.True(t, score.GTE(math.LegacyZeroDec()))
			}
		})
	}

	_ = k   // Acknowledge outer keeper
	_ = ctx // Acknowledge outer context
}

// TestSelectDiverseValidators tests diverse validator selection
func TestSelectDiverseValidators(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)

	// Create validators across regions
	validators := make([]sdk.ValAddress, 12)
	regions := []string{
		keeper.RegionNorthAmerica, keeper.RegionNorthAmerica,
		keeper.RegionEurope, keeper.RegionEurope,
		keeper.RegionAsia, keeper.RegionAsia,
		keeper.RegionOceania, keeper.RegionOceania,
		keeper.RegionSouthAmerica, keeper.RegionSouthAmerica,
		keeper.RegionAfrica, keeper.RegionAfrica,
	}

	for i := 0; i < 12; i++ {
		validators[i] = createTestValAddr(t, i)
		info := &keeper.GeographicInfo{Region: regions[i]}
		err := k.SetValidatorGeographicInfo(ctx, validators[i], info)
		require.NoError(t, err)
	}

	// Select diverse validators
	selected, err := k.SelectDiverseValidators(ctx, validators, 6, 1)
	require.NoError(t, err)
	require.LessOrEqual(t, len(selected), 6)
}

// TestCheckSubmissionGeographicDiversity tests diversity check for submissions
func TestCheckSubmissionGeographicDiversity(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)

	// Create validators with known regions
	val1 := createTestValAddr(t, 1)
	val2 := createTestValAddr(t, 2)
	val3 := createTestValAddr(t, 3)

	_ = k.SetValidatorGeographicInfo(ctx, val1, &keeper.GeographicInfo{Region: keeper.RegionEurope})
	_ = k.SetValidatorGeographicInfo(ctx, val2, &keeper.GeographicInfo{Region: keeper.RegionAsia})
	_ = k.SetValidatorGeographicInfo(ctx, val3, &keeper.GeographicInfo{Region: keeper.RegionNorthAmerica})

	submissions := map[string]math.LegacyDec{
		val1.String(): math.LegacyNewDec(50000),
		val2.String(): math.LegacyNewDec(50100),
		val3.String(): math.LegacyNewDec(49900),
	}

	// Check with minimum 2 regions
	isDiverse, err := k.CheckSubmissionGeographicDiversity(ctx, submissions, 2)
	require.NoError(t, err)
	require.True(t, isDiverse)

	// Check with minimum 5 regions (should fail)
	isDiverse, err = k.CheckSubmissionGeographicDiversity(ctx, submissions, 5)
	require.NoError(t, err)
	require.False(t, isDiverse)
}

// TestGetGeographicDiversityMetrics tests metrics calculation
func TestGetGeographicDiversityMetrics(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)

	// Create validators across regions
	regions := []string{
		keeper.RegionEurope, keeper.RegionEurope, keeper.RegionEurope,
		keeper.RegionAsia, keeper.RegionAsia,
		keeper.RegionNorthAmerica,
	}

	for i, region := range regions {
		val := createTestValAddr(t, i)
		info := &keeper.GeographicInfo{Region: region}
		err := k.SetValidatorGeographicInfo(ctx, val, info)
		require.NoError(t, err)
	}

	metrics, err := k.GetGeographicDiversityMetrics(ctx)
	require.NoError(t, err)
	require.Equal(t, 6, metrics.TotalValidators)
	require.Equal(t, keeper.RegionEurope, metrics.LargestRegion)
	require.True(t, metrics.DiversityScore.IsPositive())
}

// TestDistributeOracleRewards tests reward distribution
func TestDistributeOracleRewards(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)

	// Distribution should work even with no reward pool
	err := k.DistributeOracleRewards(ctx)
	require.NoError(t, err)
}

// TestDistributeOracleRewardsWithAccuracy tests reward distribution with accuracy bonuses
func TestDistributeOracleRewardsWithAccuracy(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)

	// Distribution should work even with no reward pool
	err := k.DistributeOracleRewardsWithAccuracy(ctx)
	require.NoError(t, err)
}

// TestProcessPriceRoundAccuracy tests processing accuracy after price round
func TestProcessPriceRoundAccuracy(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)

	val1 := createTestValAddr(t, 1)
	val2 := createTestValAddr(t, 2)

	submissions := map[string]math.LegacyDec{
		val1.String(): math.LegacyNewDec(50000), // Exact
		val2.String(): math.LegacyNewDec(50500), // 1% off
	}

	finalPrice := math.LegacyNewDec(50000)

	err := k.ProcessPriceRoundAccuracy(ctx, "BTC", finalPrice, submissions)
	require.NoError(t, err)

	// Check val1 accuracy
	acc1, _ := k.GetValidatorAccuracy(ctx, val1)
	require.Equal(t, uint64(1), acc1.TotalSubmissions)
	require.Equal(t, uint64(1), acc1.AccurateSubmissions)

	// Check val2 accuracy
	acc2, _ := k.GetValidatorAccuracy(ctx, val2)
	require.Equal(t, uint64(1), acc2.TotalSubmissions)
	require.Equal(t, uint64(1), acc2.AccurateSubmissions) // 1% is within threshold
}

// TestRotateOracleValidators tests validator rotation
func TestRotateOracleValidators(t *testing.T) {
	t.Parallel()

	k, _, ctx := keepertest.OracleKeeper(t)

	// Rotation should not error even with no validators
	err := k.RotateOracleValidators(ctx)
	require.NoError(t, err)
}
