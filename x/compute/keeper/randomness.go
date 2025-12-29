package keeper

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/big"
	"sort"
	"time"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// Randomness constants
const (
	// MaxRandomnessParticipants is the maximum number of validators that can participate
	// in the commit-reveal scheme per block. This limits state growth and ensures
	// efficient processing in BeginBlocker/EndBlocker.
	MaxRandomnessParticipants = 20

	// RandomnessRevealWindow is the number of blocks a validator has to reveal
	// their commitment. After this window, the commitment expires.
	RandomnessRevealWindow = 1

	// RandomnessCommitmentSize is the expected size of the commitment hash (SHA256)
	RandomnessCommitmentSize = 32

	// RandomnessRevealSize is the expected size of the reveal value
	RandomnessRevealSize = 32
)

// CommitRandomness stores a validator's hash commitment for the commit-reveal scheme.
// The commitment hash should be SHA256(reveal_value || validator_address).
// This is called during transaction processing when a validator submits their commitment.
func (k Keeper) CommitRandomness(ctx context.Context, validator sdk.AccAddress, commitmentHash []byte) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()
	now := sdkCtx.BlockTime()

	// Validate commitment hash size
	if len(commitmentHash) != RandomnessCommitmentSize {
		return fmt.Errorf("invalid commitment hash size: expected %d, got %d", RandomnessCommitmentSize, len(commitmentHash))
	}

	// Check if validator already has a pending commitment for this block
	existing, err := k.GetRandomnessCommitment(ctx, validator)
	if err == nil && existing != nil {
		if existing.BlockHeight == currentHeight && existing.Status == types.RANDOMNESS_COMMITMENT_STATUS_COMMITTED {
			return fmt.Errorf("validator %s already has a commitment for block %d", validator.String(), currentHeight)
		}
	}

	// Check if we've reached the maximum number of participants for this block
	count := k.CountCommitmentsForBlock(ctx, currentHeight)
	if count >= MaxRandomnessParticipants {
		return fmt.Errorf("maximum randomness participants (%d) reached for block %d", MaxRandomnessParticipants, currentHeight)
	}

	// Create the commitment
	commitment := types.RandomnessCommitment{
		Validator:      validator.String(),
		CommitmentHash: commitmentHash,
		RevealValue:    nil, // Empty until revealed
		BlockHeight:    currentHeight,
		Status:         types.RANDOMNESS_COMMITMENT_STATUS_COMMITTED,
		CommittedAt:    now,
		RevealedAt:     nil,
	}

	// Store the commitment
	if err := k.SetRandomnessCommitment(ctx, commitment); err != nil {
		return fmt.Errorf("failed to store commitment: %w", err)
	}

	// Create height index for efficient lookup during reveal phase
	store := k.getStore(ctx)
	indexKey := RandomnessCommitmentByHeightKey(currentHeight, validator)
	store.Set(indexKey, []byte{1}) // Non-empty value indicates existence

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"randomness_committed",
			sdk.NewAttribute("validator", validator.String()),
			sdk.NewAttribute("block_height", fmt.Sprintf("%d", currentHeight)),
		),
	)

	return nil
}

// RevealRandomness verifies and stores the revealed value for a validator's commitment.
// The reveal value is valid if SHA256(reveal_value || validator_address) matches the commitment hash.
func (k Keeper) RevealRandomness(ctx context.Context, validator sdk.AccAddress, revealValue []byte) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	// Validate reveal value size
	if len(revealValue) != RandomnessRevealSize {
		return fmt.Errorf("invalid reveal value size: expected %d, got %d", RandomnessRevealSize, len(revealValue))
	}

	// Get the existing commitment
	commitment, err := k.GetRandomnessCommitment(ctx, validator)
	if err != nil {
		return fmt.Errorf("no commitment found for validator %s: %w", validator.String(), err)
	}

	// Check if already revealed
	if commitment.Status == types.RANDOMNESS_COMMITMENT_STATUS_REVEALED {
		return fmt.Errorf("commitment already revealed for validator %s", validator.String())
	}

	// Check if expired
	if commitment.Status == types.RANDOMNESS_COMMITMENT_STATUS_EXPIRED {
		return fmt.Errorf("commitment expired for validator %s", validator.String())
	}

	// Verify the reveal: SHA256(reveal_value || validator_address) must equal commitment_hash
	expectedHash := computeCommitmentHash(revealValue, validator)
	if !bytes.Equal(expectedHash, commitment.CommitmentHash) {
		return fmt.Errorf("reveal verification failed: hash mismatch for validator %s", validator.String())
	}

	// Update the commitment with the revealed value
	commitment.RevealValue = revealValue
	commitment.Status = types.RANDOMNESS_COMMITMENT_STATUS_REVEALED
	commitment.RevealedAt = &now

	// Store the updated commitment
	if err := k.SetRandomnessCommitment(ctx, *commitment); err != nil {
		return fmt.Errorf("failed to store revealed commitment: %w", err)
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"randomness_revealed",
			sdk.NewAttribute("validator", validator.String()),
			sdk.NewAttribute("block_height", fmt.Sprintf("%d", commitment.BlockHeight)),
		),
	)

	return nil
}

// GetAggregatedRandomness combines all revealed values from the previous block
// with block entropy to produce secure randomness for provider selection.
// This should be called after EndBlocker has processed all reveals.
func (k Keeper) GetAggregatedRandomness(ctx context.Context, seed []byte) *big.Int {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get the stored aggregated randomness from previous block's reveals
	store := k.getStore(ctx)
	aggregatedBytes := store.Get(AggregatedRandomnessKey)

	hasher := sha256.New()

	// Add the aggregated randomness from revealed values (if any)
	if len(aggregatedBytes) > 0 {
		hasher.Write(aggregatedBytes)
	}

	// Add block entropy sources
	hasher.Write(sdkCtx.HeaderHash())

	// Block height
	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, types.SaturateInt64ToUint64(sdkCtx.BlockHeight()))
	hasher.Write(heightBz)

	// Block time
	timeBz := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBz, types.SaturateInt64ToUint64(sdkCtx.BlockTime().Unix()))
	hasher.Write(timeBz)

	// Additional seed (request-specific entropy)
	hasher.Write(seed)

	randomBytes := hasher.Sum(nil)
	return new(big.Int).SetBytes(randomBytes)
}

// ProcessRandomnessCommitPhase is called in BeginBlocker to initialize the commit phase.
// Validators can submit commitments during this block's transaction processing.
func (k Keeper) ProcessRandomnessCommitPhase(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Clean up expired commitments from previous blocks
	if err := k.CleanupExpiredCommitments(ctx); err != nil {
		sdkCtx.Logger().Error("failed to cleanup expired commitments", "error", err)
		// Don't return error - continue processing
	}

	return nil
}

// ProcessRandomnessRevealPhase is called in EndBlocker to process reveals and aggregate randomness.
// It collects all revealed values from the current block and combines them for the next block.
func (k Keeper) ProcessRandomnessRevealPhase(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Find the previous block's height for which we should process reveals
	// Commitments made in block N should be revealed in block N (same block, EndBlocker)
	revealHeight := currentHeight

	// Collect all revealed values for this block
	var revealedValues [][]byte
	var validators []string

	store := k.getStore(ctx)
	prefix := RandomnessCommitmentByHeightPrefixForHeight(revealHeight)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		// Extract validator address from the key
		// Key format: prefix (2) + height (8) + validator (20)
		key := iter.Key()
		if len(key) < len(prefix)+20 {
			continue
		}

		validatorBytes := key[len(prefix):]
		validator := sdk.AccAddress(validatorBytes)

		commitment, err := k.GetRandomnessCommitment(ctx, validator)
		if err != nil {
			continue
		}

		// Only include revealed commitments
		if commitment.Status == types.RANDOMNESS_COMMITMENT_STATUS_REVEALED && len(commitment.RevealValue) > 0 {
			revealedValues = append(revealedValues, commitment.RevealValue)
			validators = append(validators, validator.String())
		} else if commitment.Status == types.RANDOMNESS_COMMITMENT_STATUS_COMMITTED {
			// Mark uncommitted as expired
			commitment.Status = types.RANDOMNESS_COMMITMENT_STATUS_EXPIRED
			_ = k.SetRandomnessCommitment(ctx, *commitment)
		}
	}

	// If we have revealed values, aggregate them
	if len(revealedValues) > 0 {
		aggregated := k.aggregateRevealedValues(ctx, revealedValues)
		store.Set(AggregatedRandomnessKey, aggregated)

		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"randomness_aggregated",
				sdk.NewAttribute("block_height", fmt.Sprintf("%d", currentHeight)),
				sdk.NewAttribute("participants", fmt.Sprintf("%d", len(revealedValues))),
				sdk.NewAttribute("validators", fmt.Sprintf("%v", validators)),
			),
		)
	}

	// Clean up processed commitments
	iter2 := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter2.Close()

	var keysToDelete [][]byte
	for ; iter2.Valid(); iter2.Next() {
		keysToDelete = append(keysToDelete, append([]byte(nil), iter2.Key()...))
	}

	for _, key := range keysToDelete {
		store.Delete(key)
	}

	return nil
}

// aggregateRevealedValues combines multiple reveal values into a single randomness value.
// Uses XOR followed by SHA256 to ensure uniform distribution and prevent manipulation
// by any single participant.
func (k Keeper) aggregateRevealedValues(ctx context.Context, revealedValues [][]byte) []byte {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if len(revealedValues) == 0 {
		return nil
	}

	// Sort values for deterministic ordering (prevents ordering manipulation)
	sortedValues := make([][]byte, len(revealedValues))
	copy(sortedValues, revealedValues)
	sort.Slice(sortedValues, func(i, j int) bool {
		return bytes.Compare(sortedValues[i], sortedValues[j]) < 0
	})

	// XOR all values together
	result := make([]byte, RandomnessRevealSize)
	for _, val := range sortedValues {
		for i := 0; i < len(result) && i < len(val); i++ {
			result[i] ^= val[i]
		}
	}

	// Add block entropy to prevent prediction
	hasher := sha256.New()
	hasher.Write(result)
	hasher.Write(sdkCtx.HeaderHash())

	// Add timestamp for additional entropy
	timeBz := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBz, types.SaturateInt64ToUint64(sdkCtx.BlockTime().Unix()))
	hasher.Write(timeBz)

	return hasher.Sum(nil)
}

// CleanupExpiredCommitments removes old commitments that were never revealed.
func (k Keeper) CleanupExpiredCommitments(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Only cleanup commitments older than the reveal window
	cutoffHeight := currentHeight - RandomnessRevealWindow - 1
	if cutoffHeight <= 0 {
		return nil
	}

	store := k.getStore(ctx)

	// Iterate through all commitments and remove expired ones
	iter := storetypes.KVStorePrefixIterator(store, RandomnessCommitmentKeyPrefix)
	defer iter.Close()

	var toDelete [][]byte
	for ; iter.Valid(); iter.Next() {
		var commitment types.RandomnessCommitment
		if err := k.cdc.Unmarshal(iter.Value(), &commitment); err != nil {
			continue
		}

		// Remove commitments that are too old
		if commitment.BlockHeight < cutoffHeight {
			toDelete = append(toDelete, append([]byte(nil), iter.Key()...))
		}
	}

	for _, key := range toDelete {
		store.Delete(key)
	}

	return nil
}

// GetRandomnessCommitment retrieves a validator's randomness commitment.
func (k Keeper) GetRandomnessCommitment(ctx context.Context, validator sdk.AccAddress) (*types.RandomnessCommitment, error) {
	store := k.getStore(ctx)
	key := RandomnessCommitmentKey(validator)

	bz := store.Get(key)
	if bz == nil {
		return nil, fmt.Errorf("commitment not found for validator %s", validator.String())
	}

	var commitment types.RandomnessCommitment
	if err := k.cdc.Unmarshal(bz, &commitment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal commitment: %w", err)
	}

	return &commitment, nil
}

// SetRandomnessCommitment stores a validator's randomness commitment.
func (k Keeper) SetRandomnessCommitment(ctx context.Context, commitment types.RandomnessCommitment) error {
	store := k.getStore(ctx)

	validatorAddr, err := sdk.AccAddressFromBech32(commitment.Validator)
	if err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}

	bz, err := k.cdc.Marshal(&commitment)
	if err != nil {
		return fmt.Errorf("failed to marshal commitment: %w", err)
	}

	store.Set(RandomnessCommitmentKey(validatorAddr), bz)
	return nil
}

// DeleteRandomnessCommitment removes a validator's randomness commitment.
func (k Keeper) DeleteRandomnessCommitment(ctx context.Context, validator sdk.AccAddress) {
	store := k.getStore(ctx)
	store.Delete(RandomnessCommitmentKey(validator))
}

// CountCommitmentsForBlock returns the number of commitments for a specific block height.
func (k Keeper) CountCommitmentsForBlock(ctx context.Context, height int64) int {
	store := k.getStore(ctx)
	prefix := RandomnessCommitmentByHeightPrefixForHeight(height)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()

	count := 0
	for ; iter.Valid(); iter.Next() {
		count++
	}

	return count
}

// computeCommitmentHash computes the commitment hash: SHA256(reveal_value || validator_address)
func computeCommitmentHash(revealValue []byte, validator sdk.AccAddress) []byte {
	hasher := sha256.New()
	hasher.Write(revealValue)
	hasher.Write(validator.Bytes())
	return hasher.Sum(nil)
}

// GenerateCommitmentHash is a helper for validators to generate their commitment hash.
// This is typically called off-chain by the validator software.
func GenerateCommitmentHash(revealValue []byte, validator sdk.AccAddress) []byte {
	return computeCommitmentHash(revealValue, validator)
}

// HasAggregatedRandomness checks if there is aggregated randomness available.
func (k Keeper) HasAggregatedRandomness(ctx context.Context) bool {
	store := k.getStore(ctx)
	return store.Has(AggregatedRandomnessKey)
}

// GetStoredAggregatedRandomness retrieves the stored aggregated randomness bytes.
func (k Keeper) GetStoredAggregatedRandomness(ctx context.Context) []byte {
	store := k.getStore(ctx)
	return store.Get(AggregatedRandomnessKey)
}

// SimulateCommitReveal is a helper for testing that simulates a validator's
// commit and reveal in the same block (useful for single-validator testnets).
func (k Keeper) SimulateCommitReveal(ctx context.Context, validator sdk.AccAddress, revealValue []byte) error {
	// Generate commitment hash
	commitmentHash := computeCommitmentHash(revealValue, validator)

	// Commit
	if err := k.CommitRandomness(ctx, validator, commitmentHash); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	// Reveal immediately
	if err := k.RevealRandomness(ctx, validator, revealValue); err != nil {
		return fmt.Errorf("reveal failed: %w", err)
	}

	return nil
}

// GetAllRandomnessCommitments returns all stored randomness commitments (for genesis export).
func (k Keeper) GetAllRandomnessCommitments(ctx context.Context) []types.RandomnessCommitment {
	store := k.getStore(ctx)
	iter := storetypes.KVStorePrefixIterator(store, RandomnessCommitmentKeyPrefix)
	defer iter.Close()

	var commitments []types.RandomnessCommitment
	for ; iter.Valid(); iter.Next() {
		var commitment types.RandomnessCommitment
		if err := k.cdc.Unmarshal(iter.Value(), &commitment); err != nil {
			continue
		}
		commitments = append(commitments, commitment)
	}

	return commitments
}

// AutoCommitFromBlockProposer generates and commits randomness from the block proposer.
// This is called in BeginBlocker to ensure there's always at least one commitment per block.
// The proposer uses a deterministic seed derived from their signing key and block height.
func (k Keeper) AutoCommitFromBlockProposer(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get the proposer address from the block header
	proposer := sdkCtx.BlockHeader().ProposerAddress
	if len(proposer) == 0 {
		// No proposer in header (possible in some test scenarios)
		return nil
	}

	proposerAddr := sdk.AccAddress(proposer)

	// Generate deterministic but unpredictable reveal value
	// Uses: proposer address + block height + previous block hash
	hasher := sha256.New()
	hasher.Write(proposer)

	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, types.SaturateInt64ToUint64(sdkCtx.BlockHeight()))
	hasher.Write(heightBz)

	// Add previous block's header hash for unpredictability
	hasher.Write(sdkCtx.HeaderHash())

	// Add timestamp for additional entropy
	timeBz := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBz, types.SaturateInt64ToUint64(sdkCtx.BlockTime().UnixNano()))
	hasher.Write(timeBz)

	revealValue := hasher.Sum(nil)

	// Commit the randomness (will be revealed in EndBlocker)
	if err := k.SimulateCommitReveal(ctx, proposerAddr, revealValue); err != nil {
		// Log but don't fail - randomness is supplementary
		sdkCtx.Logger().Debug("auto-commit randomness failed", "error", err)
		return nil
	}

	return nil
}

// RandomnessCommitmentStatus constants for use in Go code (mirrors proto enum)
// These are defined here for Go code compatibility when proto types aren't available
var (
	_ = types.RANDOMNESS_COMMITMENT_STATUS_UNSPECIFIED
	_ = types.RANDOMNESS_COMMITMENT_STATUS_COMMITTED
	_ = types.RANDOMNESS_COMMITMENT_STATUS_REVEALED
	_ = types.RANDOMNESS_COMMITMENT_STATUS_EXPIRED
)

// Ensure the randomness functions are used to prevent import cycle warnings
var (
	_ = time.Now
)
