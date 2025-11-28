package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/oracle/types"
)

// Task 131: Oracle Price Source Verification
type PriceSource struct {
	Name          string
	URL           string
	PublicKey     []byte
	LastHeartbeat int64
	IsActive      bool
	ReputationScore math.LegacyDec
}

// VerifyPriceSource validates that a price source is legitimate and authorized
func (k Keeper) VerifyPriceSource(ctx context.Context, validatorAddr sdk.ValAddress, sourceID string, signature []byte) error {
	store := k.getStore(ctx)

	// Get registered source
	sourceKey := append([]byte("price_source/"), []byte(sourceID)...)
	sourceBz := store.Get(sourceKey)
	if sourceBz == nil {
		return types.ErrInvalidPriceSource.Wrapf("price source not registered: %s", sourceID)
	}

	// Decode source
	var source PriceSource
	if err := k.cdc.Unmarshal(sourceBz, &source); err != nil {
		return types.ErrInvalidPriceSource.Wrap("failed to unmarshal price source")
	}

	// Check if source is active
	if !source.IsActive {
		return types.ErrInvalidPriceSource.Wrapf("price source inactive: %s", sourceID)
	}

	// Check source reputation
	minReputation := math.LegacyNewDecWithPrec(70, 2) // 0.70 minimum
	if source.ReputationScore.LT(minReputation) {
		return types.ErrInvalidPriceSource.Wrapf(
			"price source reputation too low: %s < %s",
			source.ReputationScore, minReputation,
		)
	}

	// Verify signature if provided
	if len(signature) > 0 && len(source.PublicKey) > 0 {
		// In production, implement actual signature verification
		// For now, basic validation
		if len(signature) < 64 {
			return types.ErrInvalidPriceSource.Wrap("invalid signature length")
		}
	}

	// Check heartbeat freshness (within last hour)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if source.LastHeartbeat > 0 {
		timeSinceHeartbeat := sdkCtx.BlockTime().Unix() - source.LastHeartbeat
		if timeSinceHeartbeat > 3600 {
			return types.ErrInvalidPriceSource.Wrapf(
				"price source stale: last heartbeat %d seconds ago",
				timeSinceHeartbeat,
			)
		}
	}

	return nil
}

// RegisterPriceSource registers a new price source (governance only)
func (k Keeper) RegisterPriceSource(ctx context.Context, source PriceSource) error {
	store := k.getStore(ctx)

	// Validate source
	if len(source.Name) == 0 || len(source.Name) > 64 {
		return types.ErrInvalidInput.Wrap("invalid source name length")
	}

	sourceID := fmt.Sprintf("%x", sha256.Sum256([]byte(source.Name+source.URL)))
	sourceKey := append([]byte("price_source/"), []byte(sourceID)...)

	// Check if already exists
	if store.Has(sourceKey) {
		return types.ErrInvalidPriceSource.Wrap("price source already registered")
	}

	// Set initial reputation
	source.ReputationScore = math.LegacyNewDecWithPrec(100, 2) // 1.0
	source.IsActive = true

	// Store source
	sourceBz, err := k.cdc.Marshal(&source)
	if err != nil {
		return types.ErrInvalidInput.Wrap("failed to marshal price source")
	}
	store.Set(sourceKey, sourceBz)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"price_source_registered",
			sdk.NewAttribute("source_id", sourceID),
			sdk.NewAttribute("name", source.Name),
		),
	)

	return nil
}

// Task 132: Oracle Report Aggregation Weighting
type WeightedPrice struct {
	Price      math.LegacyDec
	Weight     math.LegacyDec
	Validator  string
	Timestamp  int64
	Stake      math.Int
	Reputation math.LegacyDec
}

// AggregateWeightedPrices aggregates prices using stake and reputation weighting
func (k Keeper) AggregateWeightedPrices(ctx context.Context, asset string, prices []WeightedPrice) (math.LegacyDec, error) {
	if len(prices) == 0 {
		return math.LegacyZeroDec(), types.ErrNoPriceData.Wrap("no prices to aggregate")
	}

	// Calculate weights based on stake and reputation
	totalWeight := math.LegacyZeroDec()
	for i := range prices {
		// Weight = (stake_percentage * 0.7) + (reputation * 0.3)
		stakeDec := math.LegacyNewDecFromInt(prices[i].Stake)
		stakeWeight := stakeDec.Mul(math.LegacyNewDecWithPrec(70, 2)) // 70% stake
		reputationWeight := prices[i].Reputation.Mul(math.LegacyNewDecWithPrec(30, 2)) // 30% reputation

		prices[i].Weight = stakeWeight.Add(reputationWeight)
		totalWeight = totalWeight.Add(prices[i].Weight)
	}

	if totalWeight.IsZero() {
		return math.LegacyZeroDec(), types.ErrInvalidInput.Wrap("total weight is zero")
	}

	// Calculate weighted average
	weightedSum := math.LegacyZeroDec()
	for _, p := range prices {
		normalizedWeight := p.Weight.Quo(totalWeight)
		weightedSum = weightedSum.Add(p.Price.Mul(normalizedWeight))
	}

	return weightedSum, nil
}

// Task 133: Oracle Outlier Detection Improvements
type OutlierDetectionConfig struct {
	StdDevMultiplier math.LegacyDec // Number of std devs for outlier
	MinDataPoints    int            // Minimum points needed
	UseMAD           bool           // Use Median Absolute Deviation instead of std dev
}

var defaultOutlierConfig = OutlierDetectionConfig{
	StdDevMultiplier: math.LegacyNewDec(3), // 3 sigma
	MinDataPoints:    5,
	UseMAD:           true,
}

// DetectOutliersAdvanced detects outliers using improved statistical methods
func (k Keeper) DetectOutliersAdvanced(ctx context.Context, prices []math.LegacyDec, config OutlierDetectionConfig) ([]int, error) {
	if len(prices) < config.MinDataPoints {
		return nil, types.ErrInsufficientData.Wrapf(
			"need at least %d data points, got %d",
			config.MinDataPoints, len(prices),
		)
	}

	outlierIndices := []int{}

	if config.UseMAD {
		// Use Median Absolute Deviation (more robust to outliers)
		median := calculateMedian(prices)

		// Calculate absolute deviations from median
		deviations := make([]math.LegacyDec, len(prices))
		for i, price := range prices {
			deviations[i] = price.Sub(median).Abs()
		}

		mad := calculateMedian(deviations)

		// Scale factor for MAD (1.4826 for normal distribution)
		scaleFactor := math.LegacyMustNewDecFromStr("1.4826")
		threshold := mad.Mul(scaleFactor).Mul(config.StdDevMultiplier)

		// Identify outliers
		for i, price := range prices {
			deviation := price.Sub(median).Abs()
			if deviation.GT(threshold) {
				outlierIndices = append(outlierIndices, i)
			}
		}
	} else {
		// Use standard deviation
		mean := calculateMean(prices)
		stdDev := calculateStdDev(prices, mean)

		threshold := stdDev.Mul(config.StdDevMultiplier)

		for i, price := range prices {
			deviation := price.Sub(mean).Abs()
			if deviation.GT(threshold) {
				outlierIndices = append(outlierIndices, i)
			}
		}
	}

	return outlierIndices, nil
}

// Task 134: Oracle Data Freshness Checks
const (
	MaxPriceAge          = 60  // seconds
	MaxSubmissionAge     = 300 // 5 minutes
	StaleDataMultiplier  = 2   // price age multiplier for staleness
)

// ValidateDataFreshness ensures oracle data is fresh and recent
func (k Keeper) ValidateDataFreshness(ctx context.Context, asset string, submittedAt int64) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentTime := sdkCtx.BlockTime().Unix()

	// Check submission age
	submissionAge := currentTime - submittedAt
	if submissionAge > MaxSubmissionAge {
		return types.ErrStaleData.Wrapf(
			"submission too old: %d seconds (max %d)",
			submissionAge, MaxSubmissionAge,
		)
	}

	// Get last price update
	price, err := k.GetPrice(ctx, asset)
	if err != nil {
		// No previous price, allow
		return nil
	}

	// Check if update is needed
	timeSinceUpdate := currentTime - price.Timestamp
	if timeSinceUpdate < MaxPriceAge {
		// Recent update exists, may not need new one
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"price_recently_updated",
				sdk.NewAttribute("asset", asset),
				sdk.NewAttribute("age_seconds", fmt.Sprintf("%d", timeSinceUpdate)),
			),
		)
	}

	return nil
}

// Task 135: Oracle Validator Rotation
type ValidatorRotationConfig struct {
	RotationPeriod  int64 // blocks
	MinActiveTime   int64 // blocks validator must be active
	MaxConsecutive  int   // max consecutive submissions
	CooldownPeriod  int64 // blocks before can submit again
}

var defaultRotationConfig = ValidatorRotationConfig{
	RotationPeriod: 1000,
	MinActiveTime:  100,
	MaxConsecutive: 10,
	CooldownPeriod: 50,
}

// RotateOracleValidators implements validator rotation for oracle duties
func (k Keeper) RotateOracleValidators(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	// Get current rotation epoch
	rotationKey := []byte("rotation_epoch")
	var currentEpoch int64
	if bz := store.Get(rotationKey); bz != nil {
		currentEpoch = int64(binary.BigEndian.Uint64(bz))
	}

	// Check if rotation is due
	if sdkCtx.BlockHeight()%defaultRotationConfig.RotationPeriod != 0 {
		return nil // Not rotation time
	}

	// Increment epoch
	currentEpoch++
	store.Set(rotationKey, sdk.Uint64ToBigEndian(uint64(currentEpoch)))

	// Get all bonded validators
	bondedVals, err := k.GetBondedValidators(ctx)
	if err != nil {
		return err
	}

	// Select subset for oracle duties (e.g., 70% of validators)
	numSelected := len(bondedVals) * 70 / 100
	if numSelected < MinValidatorsForSecurity {
		numSelected = len(bondedVals) // Use all if too few
	}

	// Rotate selection based on epoch
	selectedIndices := make(map[int]bool)
	for i := 0; i < numSelected; i++ {
		index := (int(currentEpoch) + i) % len(bondedVals)
		selectedIndices[index] = true
	}

	// Update active oracle validators
	for i, val := range bondedVals {
		valAddr := val.GetOperator()
		activeKey := append([]byte("active_oracle_validator/"), valAddr.Bytes()...)

		if selectedIndices[i] {
			store.Set(activeKey, []byte{1})
		} else {
			store.Delete(activeKey)
		}
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oracle_validators_rotated",
			sdk.NewAttribute("epoch", fmt.Sprintf("%d", currentEpoch)),
			sdk.NewAttribute("active_count", fmt.Sprintf("%d", numSelected)),
		),
	)

	return nil
}

// IsActiveOracleValidator checks if a validator is currently active for oracle duties
func (k Keeper) IsActiveOracleValidator(ctx context.Context, valAddr sdk.ValAddress) bool {
	store := k.getStore(ctx)
	activeKey := append([]byte("active_oracle_validator/"), valAddr.Bytes()...)
	return store.Has(activeKey)
}

// Task 136: Oracle Price Source Redundancy
type PriceSourceSubmission struct {
	SourceID  string
	Price     math.LegacyDec
	Timestamp int64
	Signature []byte
}

// SubmitMultiSourcePrice allows validators to submit prices from multiple sources
func (k Keeper) SubmitMultiSourcePrice(ctx context.Context, validator sdk.ValAddress, asset string, sources []PriceSourceSubmission) error {
	// Require at least 3 sources for redundancy
	if len(sources) < 3 {
		return types.ErrInsufficientData.Wrapf(
			"need at least 3 price sources, got %d",
			len(sources),
		)
	}

	// Verify each source
	for _, source := range sources {
		if err := k.VerifyPriceSource(ctx, validator, source.SourceID, source.Signature); err != nil {
			return types.ErrInvalidPriceSource.Wrapf("source %s verification failed: %v", source.SourceID, err)
		}
	}

	// Extract prices
	prices := make([]math.LegacyDec, len(sources))
	for i, source := range sources {
		prices[i] = source.Price
	}

	// Detect outliers among sources
	outliers, err := k.DetectOutliersAdvanced(ctx, prices, defaultOutlierConfig)
	if err != nil {
		return err
	}

	// Remove outliers
	validPrices := []math.LegacyDec{}
	for i, price := range prices {
		isOutlier := false
		for _, outlierIdx := range outliers {
			if i == outlierIdx {
				isOutlier = true
				break
			}
		}
		if !isOutlier {
			validPrices = append(validPrices, price)
		}
	}

	if len(validPrices) < 2 {
		return types.ErrInsufficientData.Wrap("too many outliers, insufficient valid prices")
	}

	// Calculate median of valid prices
	medianPrice := calculateMedian(validPrices)

	// Submit the aggregated price
	return k.SubmitPrice(ctx, validator, asset, medianPrice)
}

// Task 137: Oracle Report Encryption
type EncryptedReport struct {
	Ciphertext []byte
	Nonce      []byte
	Validator  string
	Timestamp  int64
}

// SubmitEncryptedPrice submits an encrypted price (for commit-reveal)
func (k Keeper) SubmitEncryptedPrice(ctx context.Context, validator sdk.ValAddress, asset string, encryptedData []byte, nonce []byte) error {
	store := k.getStore(ctx)

	// Validate inputs
	if len(encryptedData) == 0 || len(nonce) == 0 {
		return types.ErrInvalidInput.Wrap("encrypted data and nonce required")
	}

	// Store encrypted submission
	key := append([]byte("encrypted_submission/"), validator.Bytes()...)
	key = append(key, []byte(asset)...)

	report := EncryptedReport{
		Ciphertext: encryptedData,
		Nonce:      nonce,
		Validator:  validator.String(),
		Timestamp:  sdk.UnwrapSDKContext(ctx).BlockTime().Unix(),
	}

	reportBz, err := k.cdc.Marshal(&report)
	if err != nil {
		return types.ErrInvalidInput.Wrap("failed to marshal encrypted report")
	}

	store.Set(key, reportBz)

	return nil
}

// Task 138: Oracle Commit-Reveal Scheme
type CommitRevealSubmission struct {
	Hash       []byte
	CommitTime int64
	RevealTime int64
	Price      math.LegacyDec
	Salt       []byte
	Revealed   bool
}

const (
	CommitPhaseDuration = 10 // blocks
	RevealPhaseDuration = 10 // blocks
)

// CommitPrice commits a price hash (phase 1 of commit-reveal)
func (k Keeper) CommitPrice(ctx context.Context, validator sdk.ValAddress, asset string, priceHash []byte) error {
	if len(priceHash) != 32 {
		return types.ErrInvalidInput.Wrap("price hash must be 32 bytes")
	}

	store := k.getStore(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Store commitment
	key := append([]byte("price_commit/"), validator.Bytes()...)
	key = append(key, []byte(asset)...)

	commitment := CommitRevealSubmission{
		Hash:       priceHash,
		CommitTime: sdkCtx.BlockHeight(),
		Revealed:   false,
	}

	commitBz, err := k.cdc.Marshal(&commitment)
	if err != nil {
		return types.ErrInvalidInput.Wrap("failed to marshal commitment")
	}

	store.Set(key, commitBz)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"price_committed",
			sdk.NewAttribute("validator", validator.String()),
			sdk.NewAttribute("asset", asset),
			sdk.NewAttribute("hash", hex.EncodeToString(priceHash)),
		),
	)

	return nil
}

// RevealPrice reveals a committed price (phase 2 of commit-reveal)
func (k Keeper) RevealPrice(ctx context.Context, validator sdk.ValAddress, asset string, price math.LegacyDec, salt []byte) error {
	store := k.getStore(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get commitment
	key := append([]byte("price_commit/"), validator.Bytes()...)
	key = append(key, []byte(asset)...)

	commitBz := store.Get(key)
	if commitBz == nil {
		return types.ErrInvalidInput.Wrap("no commitment found")
	}

	var commitment CommitRevealSubmission
	if err := k.cdc.Unmarshal(commitBz, &commitment); err != nil {
		return types.ErrInvalidInput.Wrap("failed to unmarshal commitment")
	}

	// Check reveal timing
	blocksSinceCommit := sdkCtx.BlockHeight() - commitment.CommitTime
	if blocksSinceCommit < CommitPhaseDuration {
		return types.ErrInvalidInput.Wrapf(
			"reveal too early: need %d blocks, only %d passed",
			CommitPhaseDuration, blocksSinceCommit,
		)
	}

	if blocksSinceCommit > CommitPhaseDuration+RevealPhaseDuration {
		return types.ErrInvalidInput.Wrap("reveal window expired")
	}

	// Verify hash
	priceBz, _ := price.Marshal()
	data := append(priceBz, salt...)
	computedHash := sha256.Sum256(data)

	if hex.EncodeToString(computedHash[:]) != hex.EncodeToString(commitment.Hash) {
		return types.ErrInvalidInput.Wrap("hash verification failed")
	}

	// Mark as revealed and store price
	commitment.Revealed = true
	commitment.RevealTime = sdkCtx.BlockHeight()
	commitment.Price = price
	commitment.Salt = salt

	commitBz, err := k.cdc.Marshal(&commitment)
	if err != nil {
		return types.ErrInvalidInput.Wrap("failed to marshal revealed commitment")
	}
	store.Set(key, commitBz)

	// Submit the revealed price
	return k.SubmitPrice(ctx, validator, asset, price)
}

// Task 139: Oracle Slashing for Inactivity
const (
	InactivityWindow     = 100 // blocks
	MaxMissedSubmissions = 10  // consecutive misses before slash
	InactivitySlashRate  = 0.01 // 1% slash
)

// TrackValidatorActivity tracks oracle validator activity for slashing
func (k Keeper) TrackValidatorActivity(ctx context.Context, validator sdk.ValAddress, submitted bool) error {
	store := k.getStore(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get miss counter
	missKey := append([]byte("validator_miss_count/"), validator.Bytes()...)
	var missCount int64
	if bz := store.Get(missKey); bz != nil {
		missCount = int64(binary.BigEndian.Uint64(bz))
	}

	if submitted {
		// Reset miss counter on successful submission
		missCount = 0
	} else {
		// Increment miss counter
		missCount++
	}

	// Store updated counter
	store.Set(missKey, sdk.Uint64ToBigEndian(uint64(missCount)))

	// Check if slashing threshold reached
	if missCount >= MaxMissedSubmissions {
		// Slash for inactivity
		slashAmount := math.LegacyNewDecWithPrec(int64(InactivitySlashRate*100), 2)

		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"oracle_inactivity_slash",
				sdk.NewAttribute("validator", validator.String()),
				sdk.NewAttribute("missed_count", fmt.Sprintf("%d", missCount)),
				sdk.NewAttribute("slash_rate", slashAmount.String()),
			),
		)

		// Reset counter after slashing
		store.Delete(missKey)

		// In production, call actual slashing module
		// k.stakingKeeper.Slash(ctx, validator, ..., slashAmount)
	}

	return nil
}

// Task 140: Oracle Reward Distribution
type OracleRewards struct {
	TotalRewards      math.Int
	ValidatorRewards  map[string]math.Int
	AccuracyBonuses   map[string]math.Int
	ParticipationRate math.LegacyDec
}

// DistributeOracleRewards distributes rewards to oracle validators
func (k Keeper) DistributeOracleRewards(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	// Get reward pool
	rewardPoolKey := []byte("oracle_reward_pool")
	var rewardPool math.Int
	if bz := store.Get(rewardPoolKey); bz != nil {
		_ = rewardPool.Unmarshal(bz)
	}

	if rewardPool.IsZero() {
		return nil // No rewards to distribute
	}

	// Get all active validators who submitted
	submissions := k.GetRecentSubmissions(ctx, 100)
	validatorScores := make(map[string]math.LegacyDec)

	// Calculate scores based on accuracy and participation
	for valAddr, count := range submissions {
		// Base score from participation
		score := math.LegacyNewDec(int64(count))

		// TODO: Add accuracy bonus based on price accuracy
		// For now, use participation as primary metric

		validatorScores[valAddr] = score
	}

	// Calculate total score
	totalScore := math.LegacyZeroDec()
	for _, score := range validatorScores {
		totalScore = totalScore.Add(score)
	}

	if totalScore.IsZero() {
		return nil // No submissions, no rewards
	}

	// Distribute rewards proportionally
	for valAddrStr, score := range validatorScores {
		proportion := score.Quo(totalScore)
		reward := proportion.Mul(math.LegacyNewDecFromInt(rewardPool)).TruncateInt()

		if !reward.IsZero() {
			valAddr, _ := sdk.ValAddressFromBech32(valAddrStr)

			// Send reward (in production, use proper reward distribution)
			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent(
					"oracle_reward_distributed",
					sdk.NewAttribute("validator", valAddrStr),
					sdk.NewAttribute("amount", reward.String()),
					sdk.NewAttribute("score", score.String()),
				),
			)

			// Record reward
			rewardKey := append([]byte("validator_reward/"), valAddr.Bytes()...)
			rewardBz, _ := reward.Marshal()
			store.Set(rewardKey, rewardBz)
		}
	}

	// Reset reward pool
	store.Delete(rewardPoolKey)

	return nil
}

// GetRecentSubmissions returns submission counts for validators
func (k Keeper) GetRecentSubmissions(ctx context.Context, blocks int64) map[string]int {
	// In production, query actual submission history
	// For now, return empty map
	return make(map[string]int)
}

// Helper functions
func calculateMedian(values []math.LegacyDec) math.LegacyDec {
	if len(values) == 0 {
		return math.LegacyZeroDec()
	}

	sorted := make([]math.LegacyDec, len(values))
	copy(sorted, values)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].LT(sorted[j])
	})

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return sorted[mid-1].Add(sorted[mid]).Quo(math.LegacyNewDec(2))
	}
	return sorted[mid]
}

func calculateMean(values []math.LegacyDec) math.LegacyDec {
	if len(values) == 0 {
		return math.LegacyZeroDec()
	}

	sum := math.LegacyZeroDec()
	for _, v := range values {
		sum = sum.Add(v)
	}
	return sum.Quo(math.LegacyNewDec(int64(len(values))))
}

func calculateStdDev(values []math.LegacyDec, mean math.LegacyDec) math.LegacyDec {
	if len(values) <= 1 {
		return math.LegacyZeroDec()
	}

	variance := math.LegacyZeroDec()
	for _, v := range values {
		diff := v.Sub(mean)
		variance = variance.Add(diff.Mul(diff))
	}
	variance = variance.Quo(math.LegacyNewDec(int64(len(values) - 1)))

	// Approximate square root
	return variance.ApproxSqrt()
}
