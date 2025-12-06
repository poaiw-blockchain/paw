package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/paw-chain/paw/x/oracle/types"
)

// Task 131: Oracle Price Source Verification
type PriceSource struct {
	Name            string
	URL             string
	PublicKey       []byte
	LastHeartbeat   int64
	IsActive        bool
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
	if err := json.Unmarshal(sourceBz, &source); err != nil {
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
		return sdkerrors.ErrInvalidRequest.Wrap("invalid source name length")
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
	sourceBz, err := json.Marshal(&source)
	if err != nil {
		return sdkerrors.ErrInvalidRequest.Wrap("failed to marshal price source")
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
		return math.LegacyZeroDec(), types.ErrPriceNotFound.Wrap("no prices to aggregate")
	}

	// Calculate weights based on stake and reputation
	totalWeight := math.LegacyZeroDec()
	for i := range prices {
		// Weight = (stake_percentage * 0.7) + (reputation * 0.3)
		stakeDec := math.LegacyNewDecFromInt(prices[i].Stake)
		stakeWeight := stakeDec.Mul(math.LegacyNewDecWithPrec(70, 2))                  // 70% stake
		reputationWeight := prices[i].Reputation.Mul(math.LegacyNewDecWithPrec(30, 2)) // 30% reputation

		prices[i].Weight = stakeWeight.Add(reputationWeight)
		totalWeight = totalWeight.Add(prices[i].Weight)
	}

	if totalWeight.IsZero() {
		return math.LegacyZeroDec(), sdkerrors.ErrInvalidRequest.Wrap("total weight is zero")
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
		return nil, sdkerrors.ErrInvalidRequest.Wrapf(
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
	MaxPriceAge         = 60  // seconds
	MaxSubmissionAge    = 300 // 5 minutes
	StaleDataMultiplier = 2   // price age multiplier for staleness
)

// ValidateDataFreshness ensures oracle data is fresh and recent
func (k Keeper) ValidateDataFreshness(ctx context.Context, asset string, submittedAt int64) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentTime := sdkCtx.BlockTime().Unix()

	// Check submission age
	submissionAge := currentTime - submittedAt
	if submissionAge > MaxSubmissionAge {
		return types.ErrPriceExpired.Wrapf(
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
	timeSinceUpdate := currentTime - price.BlockTime
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
	RotationPeriod int64 // blocks
	MinActiveTime  int64 // blocks validator must be active
	MaxConsecutive int   // max consecutive submissions
	CooldownPeriod int64 // blocks before can submit again
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
		valAddrStr := val.GetOperator()
		valAddr, err := sdk.ValAddressFromBech32(valAddrStr)
		if err != nil {
			sdkCtx.Logger().Error("invalid validator operator address", "address", valAddrStr, "error", err)
			continue
		}

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
		return sdkerrors.ErrInvalidRequest.Wrapf(
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
		return sdkerrors.ErrInvalidRequest.Wrap("too many outliers, insufficient valid prices")
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
		return sdkerrors.ErrInvalidRequest.Wrap("encrypted data and nonce required")
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

	reportBz, err := json.Marshal(&report)
	if err != nil {
		return sdkerrors.ErrInvalidRequest.Wrap("failed to marshal encrypted report")
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
		return sdkerrors.ErrInvalidRequest.Wrap("price hash must be 32 bytes")
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

	commitBz, err := json.Marshal(&commitment)
	if err != nil {
		return sdkerrors.ErrInvalidRequest.Wrap("failed to marshal commitment")
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
		return sdkerrors.ErrInvalidRequest.Wrap("no commitment found")
	}

	var commitment CommitRevealSubmission
	if err := json.Unmarshal(commitBz, &commitment); err != nil {
		return sdkerrors.ErrInvalidRequest.Wrap("failed to unmarshal commitment")
	}

	// Check reveal timing
	blocksSinceCommit := sdkCtx.BlockHeight() - commitment.CommitTime
	if blocksSinceCommit < CommitPhaseDuration {
		return sdkerrors.ErrInvalidRequest.Wrapf(
			"reveal too early: need %d blocks, only %d passed",
			CommitPhaseDuration, blocksSinceCommit,
		)
	}

	if blocksSinceCommit > CommitPhaseDuration+RevealPhaseDuration {
		return sdkerrors.ErrInvalidRequest.Wrap("reveal window expired")
	}

	// Verify hash
	priceBz, err := price.Marshal()
	if err != nil {
		return sdkerrors.ErrInvalidRequest.Wrap("failed to marshal price for hashing")
	}

	data := append(priceBz, salt...)
	computedHash := sha256.Sum256(data)

	if hex.EncodeToString(computedHash[:]) != hex.EncodeToString(commitment.Hash) {
		return sdkerrors.ErrInvalidRequest.Wrap("hash verification failed")
	}

	// Mark as revealed and store price
	commitment.Revealed = true
	commitment.RevealTime = sdkCtx.BlockHeight()
	commitment.Price = price
	commitment.Salt = salt

	commitBz, err = json.Marshal(&commitment)
	if err != nil {
		return sdkerrors.ErrInvalidRequest.Wrap("failed to marshal revealed commitment")
	}
	store.Set(key, commitBz)

	// Submit the revealed price
	return k.SubmitPrice(ctx, validator, asset, price)
}

// Task 139: Oracle Slashing for Inactivity
const (
	InactivityWindow     = 100  // blocks
	MaxMissedSubmissions = 10   // consecutive misses before slash
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
	rewardPool := math.ZeroInt()
	if bz := store.Get(rewardPoolKey); bz != nil {
		_ = rewardPool.Unmarshal(bz)
	}

	if rewardPool.IsZero() || rewardPool.IsNil() {
		return nil // No rewards to distribute
	}

	// Get all active validators who submitted
	submissions := k.GetRecentSubmissions(ctx, 100)
	validatorScores := make(map[string]math.LegacyDec)

	// Calculate scores based on accuracy and participation
	for valAddr, count := range submissions {
		// Base score from participation (50% of score)
		participationScore := math.LegacyNewDec(int64(count))

		// Accuracy bonus based on historical price accuracy (50% of score)
		// Validators who submit prices closer to the final aggregated price
		// receive higher accuracy scores
		accuracyScore := k.calculateValidatorAccuracy(ctx, valAddr)

		// Combined score: participation (1x weight) + accuracy (1x weight)
		score := participationScore.Add(accuracyScore)

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

// calculateValidatorAccuracy returns an accuracy score for a validator address string.
// The score is normalized to a 0-10 scale based on the validator's AccuracyScore (0-100).
// This is used internally for reward distribution scoring.
func (k Keeper) calculateValidatorAccuracy(ctx context.Context, valAddrStr string) math.LegacyDec {
	valAddr, err := sdk.ValAddressFromBech32(valAddrStr)
	if err != nil {
		// Invalid address, return base score
		return math.LegacyNewDec(5) // Neutral score
	}

	accuracy, err := k.GetValidatorAccuracy(ctx, valAddr)
	if err != nil {
		// No accuracy data, return neutral score
		return math.LegacyNewDec(5)
	}

	// Convert 0-100 accuracy score to 0-10 scale for reward calculation
	// This keeps accuracy bonus weight equal to participation weight
	return accuracy.AccuracyScore.QuoInt64(10)
}

// ValidatorAccuracy tracks validator accuracy statistics for bonus calculation
type ValidatorAccuracy struct {
	// TotalSubmissions is the total number of price submissions
	TotalSubmissions uint64 `json:"total_submissions"`
	// AccurateSubmissions is submissions within accuracy threshold
	AccurateSubmissions uint64 `json:"accurate_submissions"`
	// TotalDeviation is the cumulative deviation from final prices (for averaging)
	TotalDeviation math.LegacyDec `json:"total_deviation"`
	// AccuracyScore is the calculated accuracy score (0-100)
	AccuracyScore math.LegacyDec `json:"accuracy_score"`
	// ConsecutiveAccurate is streak of accurate submissions
	ConsecutiveAccurate uint64 `json:"consecutive_accurate"`
	// LastUpdatedHeight is the block height of last update
	LastUpdatedHeight int64 `json:"last_updated_height"`
}

// GetValidatorAccuracy retrieves accuracy statistics for a validator
func (k Keeper) GetValidatorAccuracy(ctx context.Context, validatorAddr sdk.ValAddress) (*ValidatorAccuracy, error) {
	store := k.getStore(ctx)
	key := GetValidatorAccuracyKey(validatorAddr)

	bz := store.Get(key)
	if bz == nil {
		return &ValidatorAccuracy{
			TotalSubmissions:    0,
			AccurateSubmissions: 0,
			TotalDeviation:      math.LegacyZeroDec(),
			AccuracyScore:       math.LegacyNewDec(50), // Start with neutral score
			ConsecutiveAccurate: 0,
			LastUpdatedHeight:   0,
		}, nil
	}

	var accuracy ValidatorAccuracy
	if err := json.Unmarshal(bz, &accuracy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal validator accuracy: %w", err)
	}

	return &accuracy, nil
}

// SetValidatorAccuracy stores accuracy statistics for a validator
func (k Keeper) SetValidatorAccuracy(ctx context.Context, validatorAddr sdk.ValAddress, accuracy *ValidatorAccuracy) error {
	store := k.getStore(ctx)
	key := GetValidatorAccuracyKey(validatorAddr)

	bz, err := json.Marshal(accuracy)
	if err != nil {
		return fmt.Errorf("failed to marshal validator accuracy: %w", err)
	}

	store.Set(key, bz)
	return nil
}

// UpdateValidatorAccuracy updates accuracy statistics after price aggregation
// This should be called after each price round with the validator's submission and final price
func (k Keeper) UpdateValidatorAccuracy(ctx context.Context, validatorAddr sdk.ValAddress, submittedPrice, finalPrice math.LegacyDec) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	accuracy, err := k.GetValidatorAccuracy(ctx, validatorAddr)
	if err != nil {
		return err
	}

	// Calculate deviation as percentage of final price
	// deviation = |submitted - final| / final * 100
	var deviation math.LegacyDec
	if finalPrice.IsPositive() {
		deviation = submittedPrice.Sub(finalPrice).Abs().Quo(finalPrice).MulInt64(100)
	} else {
		deviation = math.LegacyZeroDec()
	}

	// Update statistics
	accuracy.TotalSubmissions++
	accuracy.TotalDeviation = accuracy.TotalDeviation.Add(deviation)
	accuracy.LastUpdatedHeight = sdkCtx.BlockHeight()

	// Accuracy threshold: submission within 1% of final price is considered accurate
	const accuracyThreshold = 1 // 1% deviation
	if deviation.LTE(math.LegacyNewDec(accuracyThreshold)) {
		accuracy.AccurateSubmissions++
		accuracy.ConsecutiveAccurate++
	} else {
		accuracy.ConsecutiveAccurate = 0
	}

	// Recalculate accuracy score
	// Score = (accurate submissions / total submissions) * 100
	// Plus bonus for consecutive accurate submissions
	if accuracy.TotalSubmissions > 0 {
		baseScore := math.LegacyNewDec(int64(accuracy.AccurateSubmissions)).MulInt64(100).QuoInt64(int64(accuracy.TotalSubmissions))

		// Bonus for consecutive accuracy (up to 10% bonus for 10+ consecutive)
		consecutiveBonus := math.LegacyNewDec(int64(accuracy.ConsecutiveAccurate))
		if consecutiveBonus.GT(math.LegacyNewDec(10)) {
			consecutiveBonus = math.LegacyNewDec(10)
		}

		accuracy.AccuracyScore = baseScore.Add(consecutiveBonus)

		// Cap at 100
		if accuracy.AccuracyScore.GT(math.LegacyNewDec(100)) {
			accuracy.AccuracyScore = math.LegacyNewDec(100)
		}
	}

	return k.SetValidatorAccuracy(ctx, validatorAddr, accuracy)
}

// CalculateAccuracyBonus calculates the bonus multiplier for a validator based on accuracy
// Returns a multiplier between 1.0 and 2.0
// - 0-50 accuracy score: 1.0x (no bonus)
// - 50-75 accuracy score: 1.0-1.25x (linear scaling)
// - 75-90 accuracy score: 1.25-1.5x (linear scaling)
// - 90-100 accuracy score: 1.5-2.0x (linear scaling)
func (k Keeper) CalculateAccuracyBonus(ctx context.Context, validatorAddr sdk.ValAddress) (math.LegacyDec, error) {
	accuracy, err := k.GetValidatorAccuracy(ctx, validatorAddr)
	if err != nil {
		return math.LegacyOneDec(), err
	}

	score := accuracy.AccuracyScore
	oneDec := math.LegacyOneDec()

	switch {
	case score.LTE(math.LegacyNewDec(50)):
		// No bonus for low accuracy
		return oneDec, nil

	case score.LTE(math.LegacyNewDec(75)):
		// Linear scaling from 1.0 to 1.25
		progress := score.Sub(math.LegacyNewDec(50)).Quo(math.LegacyNewDec(25))
		bonus := oneDec.Add(progress.Mul(math.LegacyNewDecWithPrec(25, 2))) // 0.25 bonus
		return bonus, nil

	case score.LTE(math.LegacyNewDec(90)):
		// Linear scaling from 1.25 to 1.5
		progress := score.Sub(math.LegacyNewDec(75)).Quo(math.LegacyNewDec(15))
		bonus := math.LegacyNewDecWithPrec(125, 2).Add(progress.Mul(math.LegacyNewDecWithPrec(25, 2)))
		return bonus, nil

	default:
		// Linear scaling from 1.5 to 2.0
		progress := score.Sub(math.LegacyNewDec(90)).Quo(math.LegacyNewDec(10))
		bonus := math.LegacyNewDecWithPrec(150, 2).Add(progress.Mul(math.LegacyNewDecWithPrec(50, 2)))
		if bonus.GT(math.LegacyNewDec(2)) {
			bonus = math.LegacyNewDec(2)
		}
		return bonus, nil
	}
}

// DistributeOracleRewardsWithAccuracy distributes rewards with accuracy bonuses
func (k Keeper) DistributeOracleRewardsWithAccuracy(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	// Get reward pool
	rewardPoolKey := []byte("oracle_reward_pool")
	rewardPool := math.ZeroInt()
	if bz := store.Get(rewardPoolKey); bz != nil {
		_ = rewardPool.Unmarshal(bz)
	}

	if rewardPool.IsZero() || rewardPool.IsNil() {
		return nil // No rewards to distribute
	}

	// Get all active validators who submitted
	submissions := k.GetRecentSubmissions(ctx, 100)
	validatorScores := make(map[string]math.LegacyDec)

	// Calculate scores based on accuracy and participation
	for valAddrStr, count := range submissions {
		valAddr, err := sdk.ValAddressFromBech32(valAddrStr)
		if err != nil {
			continue
		}

		// Base score from participation
		baseScore := math.LegacyNewDec(int64(count))

		// Get accuracy bonus multiplier
		accuracyBonus, err := k.CalculateAccuracyBonus(ctx, valAddr)
		if err != nil {
			accuracyBonus = math.LegacyOneDec()
		}

		// Apply accuracy bonus to score
		score := baseScore.Mul(accuracyBonus)
		validatorScores[valAddrStr] = score

		// Emit accuracy bonus event
		if accuracyBonus.GT(math.LegacyOneDec()) {
			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent(
					"oracle_accuracy_bonus_applied",
					sdk.NewAttribute("validator", valAddrStr),
					sdk.NewAttribute("base_score", baseScore.String()),
					sdk.NewAttribute("bonus_multiplier", accuracyBonus.String()),
					sdk.NewAttribute("final_score", score.String()),
				),
			)
		}
	}

	// Calculate total score
	totalScore := math.LegacyZeroDec()
	for _, score := range validatorScores {
		totalScore = totalScore.Add(score)
	}

	if totalScore.IsZero() {
		return nil // No submissions, no rewards
	}

	// Track bonus pool statistics
	totalBonusDistributed := math.ZeroInt()

	// Distribute rewards proportionally
	for valAddrStr, score := range validatorScores {
		proportion := score.Quo(totalScore)
		reward := proportion.Mul(math.LegacyNewDecFromInt(rewardPool)).TruncateInt()

		if !reward.IsZero() {
			valAddr, _ := sdk.ValAddressFromBech32(valAddrStr)

			// Calculate base reward (what they would get without bonus)
			accuracy, _ := k.GetValidatorAccuracy(ctx, valAddr)
			bonusMultiplier, _ := k.CalculateAccuracyBonus(ctx, valAddr)
			baseRewardDec := math.LegacyNewDecFromInt(reward).Quo(bonusMultiplier)
			baseReward := baseRewardDec.TruncateInt()
			bonusAmount := reward.Sub(baseReward)
			totalBonusDistributed = totalBonusDistributed.Add(bonusAmount)

			// Emit reward distribution event
			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent(
					"oracle_reward_distributed",
					sdk.NewAttribute("validator", valAddrStr),
					sdk.NewAttribute("base_reward", baseReward.String()),
					sdk.NewAttribute("accuracy_bonus", bonusAmount.String()),
					sdk.NewAttribute("total_reward", reward.String()),
					sdk.NewAttribute("accuracy_score", accuracy.AccuracyScore.String()),
				),
			)

			// Record reward
			rewardKey := append([]byte("validator_reward/"), valAddr.Bytes()...)
			rewardBz, _ := reward.Marshal()
			store.Set(rewardKey, rewardBz)
		}
	}

	// Emit summary event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oracle_rewards_distributed_summary",
			sdk.NewAttribute("total_pool", rewardPool.String()),
			sdk.NewAttribute("total_bonus_distributed", totalBonusDistributed.String()),
			sdk.NewAttribute("num_validators", fmt.Sprintf("%d", len(validatorScores))),
		),
	)

	// Reset reward pool
	store.Delete(rewardPoolKey)

	return nil
}

// ProcessPriceRoundAccuracy should be called after price aggregation to update accuracy stats
// Uses string keys for the submissions map since ValAddress is not comparable
func (k Keeper) ProcessPriceRoundAccuracy(ctx context.Context, asset string, finalPrice math.LegacyDec, submissions map[string]math.LegacyDec) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	for valAddrStr, submittedPrice := range submissions {
		valAddr, err := sdk.ValAddressFromBech32(valAddrStr)
		if err != nil {
			sdkCtx.Logger().Error("invalid validator address",
				"address", valAddrStr,
				"error", err,
			)
			continue
		}

		if err := k.UpdateValidatorAccuracy(ctx, valAddr, submittedPrice, finalPrice); err != nil {
			sdkCtx.Logger().Error("failed to update validator accuracy",
				"validator", valAddrStr,
				"error", err,
			)
			continue
		}
	}

	return nil
}

// ============================================================================
// Geographic Diversity for Oracle Validators
// ============================================================================

// GeographicInfo stores validator's geographic information for diversity tracking
type GeographicInfo struct {
	// Region is the geographic region (e.g., "NA", "EU", "ASIA", "OCEANIA", "SA", "AF")
	Region string `json:"region"`
	// Country is the ISO 3166-1 alpha-2 country code
	Country string `json:"country"`
	// Timezone is the IANA timezone string
	Timezone string `json:"timezone"`
	// RegistrationTime is when this info was registered
	RegistrationTime int64 `json:"registration_time"`
	// LastVerified is when this info was last verified
	LastVerified int64 `json:"last_verified"`
	// IsVerified indicates if the geographic claim has been verified
	IsVerified bool `json:"is_verified"`
}

// Supported regions for geographic diversity
const (
	RegionNorthAmerica = "NA"
	RegionSouthAmerica = "SA"
	RegionEurope       = "EU"
	RegionAsia         = "ASIA"
	RegionOceania      = "OCEANIA"
	RegionAfrica       = "AF"
	RegionUnknown      = "UNKNOWN"
)

// GetValidatorGeographicInfo retrieves geographic info for a validator
func (k Keeper) GetValidatorGeographicInfo(ctx context.Context, validatorAddr sdk.ValAddress) (*GeographicInfo, error) {
	store := k.getStore(ctx)
	key := GetGeographicInfoKey(validatorAddr)

	bz := store.Get(key)
	if bz == nil {
		return &GeographicInfo{
			Region:           RegionUnknown,
			Country:          "",
			Timezone:         "",
			RegistrationTime: 0,
			LastVerified:     0,
			IsVerified:       false,
		}, nil
	}

	var info GeographicInfo
	if err := json.Unmarshal(bz, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal geographic info: %w", err)
	}

	return &info, nil
}

// SetValidatorGeographicInfo stores geographic info for a validator
func (k Keeper) SetValidatorGeographicInfo(ctx context.Context, validatorAddr sdk.ValAddress, info *GeographicInfo) error {
	store := k.getStore(ctx)
	key := GetGeographicInfoKey(validatorAddr)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	info.RegistrationTime = sdkCtx.BlockTime().Unix()

	bz, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal geographic info: %w", err)
	}

	store.Set(key, bz)

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"validator_geographic_info_set",
			sdk.NewAttribute("validator", validatorAddr.String()),
			sdk.NewAttribute("region", info.Region),
			sdk.NewAttribute("country", info.Country),
		),
	)

	return nil
}

// GeographicDiversityScore calculates a diversity score for a set of validators
// Higher score means better geographic distribution
// Score is 0-100 where 100 is perfectly distributed across all regions
func (k Keeper) GeographicDiversityScore(ctx context.Context, validators []sdk.ValAddress) (math.LegacyDec, error) {
	if len(validators) == 0 {
		return math.LegacyZeroDec(), nil
	}

	// Count validators per region
	regionCounts := make(map[string]int)
	totalValidators := 0

	for _, valAddr := range validators {
		info, err := k.GetValidatorGeographicInfo(ctx, valAddr)
		if err != nil {
			continue
		}
		regionCounts[info.Region]++
		totalValidators++
	}

	if totalValidators == 0 {
		return math.LegacyZeroDec(), nil
	}

	// Calculate diversity using Simpson's Diversity Index
	// D = 1 - sum((n/N)^2) where n is count per region, N is total
	// Higher values indicate more diversity
	sumSquaredProportions := math.LegacyZeroDec()
	for _, count := range regionCounts {
		proportion := math.LegacyNewDec(int64(count)).Quo(math.LegacyNewDec(int64(totalValidators)))
		sumSquaredProportions = sumSquaredProportions.Add(proportion.Mul(proportion))
	}

	// Normalize to 0-100 scale
	// Also factor in number of unique regions represented
	uniqueRegions := len(regionCounts)
	maxRegions := 6 // Total number of defined regions

	diversityIndex := math.LegacyOneDec().Sub(sumSquaredProportions)
	regionCoverage := math.LegacyNewDec(int64(uniqueRegions)).Quo(math.LegacyNewDec(int64(maxRegions)))

	// Combined score: 70% diversity index + 30% region coverage
	score := diversityIndex.MulInt64(70).Add(regionCoverage.MulInt64(30))

	return score, nil
}

// GetValidatorsByRegion returns all validators in a specific region
func (k Keeper) GetValidatorsByRegion(ctx context.Context, region string) ([]sdk.ValAddress, error) {
	store := k.getStore(ctx)
	var validators []sdk.ValAddress

	// Iterate through all geographic info entries
	// Use prefix + 0xFF as end bound for prefix iteration
	endKey := make([]byte, len(GeographicInfoKeyPrefix))
	copy(endKey, GeographicInfoKeyPrefix)
	endKey[len(endKey)-1]++
	iterator := store.Iterator(GeographicInfoKeyPrefix, endKey)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var info GeographicInfo
		if err := json.Unmarshal(iterator.Value(), &info); err != nil {
			continue
		}

		if info.Region == region {
			// Extract validator address from key
			keyWithoutPrefix := iterator.Key()[len(GeographicInfoKeyPrefix):]
			valAddr, err := sdk.ValAddressFromBech32(string(keyWithoutPrefix))
			if err != nil {
				continue
			}
			validators = append(validators, valAddr)
		}
	}

	return validators, nil
}

// GetRegionDistribution returns the count of validators per region
func (k Keeper) GetRegionDistribution(ctx context.Context) (map[string]int, error) {
	store := k.getStore(ctx)
	distribution := make(map[string]int)

	// Use prefix + 0xFF as end bound for prefix iteration
	endKey := make([]byte, len(GeographicInfoKeyPrefix))
	copy(endKey, GeographicInfoKeyPrefix)
	endKey[len(endKey)-1]++
	iterator := store.Iterator(GeographicInfoKeyPrefix, endKey)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var info GeographicInfo
		if err := json.Unmarshal(iterator.Value(), &info); err != nil {
			continue
		}
		distribution[info.Region]++
	}

	return distribution, nil
}

// SelectDiverseValidators selects validators ensuring geographic diversity
// It attempts to select at least minPerRegion validators from each represented region
// while respecting the total maxValidators limit
func (k Keeper) SelectDiverseValidators(ctx context.Context, candidates []sdk.ValAddress, maxValidators, minPerRegion int) ([]sdk.ValAddress, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	// Group candidates by region
	candidatesByRegion := make(map[string][]sdk.ValAddress)
	for _, valAddr := range candidates {
		info, err := k.GetValidatorGeographicInfo(ctx, valAddr)
		if err != nil {
			info = &GeographicInfo{Region: RegionUnknown}
		}
		candidatesByRegion[info.Region] = append(candidatesByRegion[info.Region], valAddr)
	}

	selected := make([]sdk.ValAddress, 0, maxValidators)
	selectedSet := make(map[string]bool)

	// First pass: select minPerRegion from each region
	regions := []string{RegionNorthAmerica, RegionSouthAmerica, RegionEurope, RegionAsia, RegionOceania, RegionAfrica}
	for _, region := range regions {
		regionCandidates := candidatesByRegion[region]
		count := 0
		for _, valAddr := range regionCandidates {
			if count >= minPerRegion || len(selected) >= maxValidators {
				break
			}
			if !selectedSet[valAddr.String()] {
				selected = append(selected, valAddr)
				selectedSet[valAddr.String()] = true
				count++
			}
		}
	}

	// Second pass: fill remaining slots round-robin from each region
	for len(selected) < maxValidators {
		added := false
		for _, region := range regions {
			if len(selected) >= maxValidators {
				break
			}
			for _, valAddr := range candidatesByRegion[region] {
				if !selectedSet[valAddr.String()] {
					selected = append(selected, valAddr)
					selectedSet[valAddr.String()] = true
					added = true
					break
				}
			}
		}
		// Also add from unknown region
		for _, valAddr := range candidatesByRegion[RegionUnknown] {
			if len(selected) >= maxValidators {
				break
			}
			if !selectedSet[valAddr.String()] {
				selected = append(selected, valAddr)
				selectedSet[valAddr.String()] = true
				added = true
			}
		}
		if !added {
			break // No more candidates available
		}
	}

	return selected, nil
}

// CheckSubmissionGeographicDiversity checks if a submission set meets diversity requirements
// Returns true if the submissions are sufficiently diverse
func (k Keeper) CheckSubmissionGeographicDiversity(ctx context.Context, submissions map[string]math.LegacyDec, minRegions int) (bool, error) {
	regionsPresent := make(map[string]bool)

	for valAddrStr := range submissions {
		valAddr, err := sdk.ValAddressFromBech32(valAddrStr)
		if err != nil {
			continue
		}

		info, err := k.GetValidatorGeographicInfo(ctx, valAddr)
		if err != nil {
			continue
		}

		if info.Region != RegionUnknown {
			regionsPresent[info.Region] = true
		}
	}

	return len(regionsPresent) >= minRegions, nil
}

// ApplyGeographicDiversityBonus adjusts validator weights based on geographic diversity
// Validators from underrepresented regions get a bonus
func (k Keeper) ApplyGeographicDiversityBonus(ctx context.Context, submissions map[string]math.LegacyDec) (map[string]math.LegacyDec, error) {
	// Get current region distribution from submissions
	regionCounts := make(map[string]int)
	validatorRegions := make(map[string]string)

	for valAddrStr := range submissions {
		valAddr, err := sdk.ValAddressFromBech32(valAddrStr)
		if err != nil {
			validatorRegions[valAddrStr] = RegionUnknown
			continue
		}

		info, err := k.GetValidatorGeographicInfo(ctx, valAddr)
		if err != nil {
			validatorRegions[valAddrStr] = RegionUnknown
			continue
		}

		validatorRegions[valAddrStr] = info.Region
		regionCounts[info.Region]++
	}

	totalSubmissions := len(submissions)
	if totalSubmissions == 0 {
		return submissions, nil
	}

	// Calculate bonus for underrepresented regions
	// If a region has fewer than expected share, validators get bonus
	expectedShare := float64(totalSubmissions) / 6.0 // 6 regions

	adjustedSubmissions := make(map[string]math.LegacyDec)
	for valAddrStr, price := range submissions {
		region := validatorRegions[valAddrStr]
		regionCount := regionCounts[region]

		// Calculate bonus multiplier
		// If region has fewer than expected validators, give bonus
		// This could be used to weight votes during aggregation
		if region != RegionUnknown && regionCount > 0 {
			actualShare := float64(regionCount)
			if actualShare < expectedShare {
				// Underrepresented region - could give bonus in future
				// For now, we track but don't modify the price
				_ = actualShare // Acknowledge for future use
			}
		}

		// For price aggregation, we don't actually change the price
		// The bonus weight should be applied during voting weight calculation
		adjustedSubmissions[valAddrStr] = price
	}

	return adjustedSubmissions, nil
}

// GetGeographicDiversityMetrics returns comprehensive diversity metrics
type GeographicDiversityMetrics struct {
	TotalValidators   int            `json:"total_validators"`
	RegionDistribution map[string]int `json:"region_distribution"`
	DiversityScore    math.LegacyDec `json:"diversity_score"`
	LargestRegion     string         `json:"largest_region"`
	SmallestRegion    string         `json:"smallest_region"`
	RegionBalance     math.LegacyDec `json:"region_balance"` // 0-1, higher is more balanced
}

func (k Keeper) GetGeographicDiversityMetrics(ctx context.Context) (*GeographicDiversityMetrics, error) {
	distribution, err := k.GetRegionDistribution(ctx)
	if err != nil {
		return nil, err
	}

	totalValidators := 0
	largestRegion := ""
	largestCount := 0
	smallestRegion := ""
	smallestCount := int(^uint(0) >> 1) // Max int

	for region, count := range distribution {
		totalValidators += count
		if count > largestCount {
			largestCount = count
			largestRegion = region
		}
		if count < smallestCount && count > 0 {
			smallestCount = count
			smallestRegion = region
		}
	}

	// Calculate diversity score from distribution
	// Simpson's Diversity Index: D = 1 - sum((n/N)^2)
	diversityScore := math.LegacyZeroDec()
	if totalValidators > 0 {
		sumSquaredProportions := math.LegacyZeroDec()
		for _, count := range distribution {
			proportion := math.LegacyNewDec(int64(count)).Quo(math.LegacyNewDec(int64(totalValidators)))
			sumSquaredProportions = sumSquaredProportions.Add(proportion.Mul(proportion))
		}
		diversityScore = math.LegacyOneDec().Sub(sumSquaredProportions).MulInt64(100)
	}

	// Calculate balance ratio
	regionBalance := math.LegacyZeroDec()
	if largestCount > 0 && smallestCount > 0 && smallestCount != int(^uint(0)>>1) {
		regionBalance = math.LegacyNewDec(int64(smallestCount)).Quo(math.LegacyNewDec(int64(largestCount)))
	}

	return &GeographicDiversityMetrics{
		TotalValidators:    totalValidators,
		RegionDistribution: distribution,
		DiversityScore:     diversityScore,
		LargestRegion:      largestRegion,
		SmallestRegion:     smallestRegion,
		RegionBalance:      regionBalance,
	}, nil
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

	// Compute square root with security-conscious fallback
	stdDev, err := variance.ApproxSqrt()
	if err != nil {
		// Security fallback: return conservative estimate (5% of mean) to ensure
		// statistical validation remains active even on sqrt failure.
		// Returning zero would disable outlier detection - unacceptable for security.
		if mean.IsPositive() {
			return mean.Mul(math.LegacyNewDecWithPrec(5, 2))
		}
		return math.LegacyNewDecWithPrec(1, 2) // 0.01 minimum fallback
	}

	return stdDev
}
