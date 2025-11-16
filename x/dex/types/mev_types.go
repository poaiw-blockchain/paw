package types

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
)

// MEVProtectionConfig defines the configuration for MEV protection mechanisms
type MEVProtectionConfig struct {
	// Enable timestamp-based transaction ordering
	EnableTimestampOrdering bool

	// Enable sandwich attack detection
	EnableSandwichDetection bool

	// Enable price impact limits
	EnablePriceImpactLimits bool

	// Maximum price impact allowed per transaction (as percentage, e.g., 0.05 = 5%)
	MaxPriceImpact math.LegacyDec

	// Time window for detecting sandwich attacks (in seconds)
	SandwichDetectionWindow int64

	// Minimum value ratio for sandwich detection (attacker buy/victim ratio)
	SandwichMinRatio math.LegacyDec

	// Maximum transaction reordering time window (in seconds)
	// Transactions can only be reordered within this window based on timestamps
	MaxReorderingWindow int64

	// Enable commit-reveal scheme (future implementation)
	EnableCommitReveal bool

	// Commit-reveal timeout (in blocks)
	CommitRevealTimeout uint64

	// Enable mempool encryption (future implementation)
	EnableMempoolEncryption bool

	// Threshold for transaction batching
	BatchingThreshold uint64

	// Enable transaction batching for atomic execution
	EnableBatching bool
}

// DefaultMEVProtectionConfig returns default MEV protection configuration
func DefaultMEVProtectionConfig() MEVProtectionConfig {
	return MEVProtectionConfig{
		EnableTimestampOrdering: true,
		EnableSandwichDetection: true,
		EnablePriceImpactLimits: true,
		MaxPriceImpact:          math.LegacyNewDecWithPrec(5, 2), // 5%
		SandwichDetectionWindow: 60,                              // 60 seconds
		SandwichMinRatio:        math.LegacyNewDecWithPrec(2, 0), // 2.0x ratio
		MaxReorderingWindow:     30,                              // 30 seconds
		EnableCommitReveal:      false,                           // Future feature
		CommitRevealTimeout:     10,                              // 10 blocks
		EnableMempoolEncryption: false,                           // Future feature
		BatchingThreshold:       5,                               // Batch every 5 txs
		EnableBatching:          false,                           // Future feature
	}
}

// Validate validates the MEV protection configuration
func (c MEVProtectionConfig) Validate() error {
	if c.MaxPriceImpact.IsNegative() {
		return ErrInvalidMEVConfig.Wrap("max price impact cannot be negative")
	}
	if c.MaxPriceImpact.GT(math.LegacyOneDec()) {
		return ErrInvalidMEVConfig.Wrap("max price impact cannot exceed 100%")
	}
	if c.SandwichDetectionWindow <= 0 {
		return ErrInvalidMEVConfig.Wrap("sandwich detection window must be positive")
	}
	if c.SandwichMinRatio.LTE(math.LegacyOneDec()) {
		return ErrInvalidMEVConfig.Wrap("sandwich min ratio must be greater than 1.0")
	}
	if c.MaxReorderingWindow <= 0 {
		return ErrInvalidMEVConfig.Wrap("max reordering window must be positive")
	}
	if c.CommitRevealTimeout == 0 {
		return ErrInvalidMEVConfig.Wrap("commit reveal timeout must be positive")
	}
	if c.BatchingThreshold == 0 {
		return ErrInvalidMEVConfig.Wrap("batching threshold must be positive")
	}
	return nil
}

// TransactionRecord stores transaction information for MEV detection
type TransactionRecord struct {
	// Transaction hash
	TxHash string

	// Address of the trader
	Trader string

	// Pool ID
	PoolID uint64

	// Token in
	TokenIn string

	// Token out
	TokenOut string

	// Amount in
	AmountIn math.Int

	// Amount out
	AmountOut math.Int

	// Timestamp (Unix timestamp in seconds)
	Timestamp int64

	// Block height
	BlockHeight int64

	// Transaction index in block
	TxIndex int64

	// Price impact (as decimal)
	PriceImpact math.LegacyDec
}

// SandwichPattern represents a detected sandwich attack pattern
type SandwichPattern struct {
	// Victim transaction
	VictimTx TransactionRecord

	// Front-running transaction (attacker buy)
	FrontRunTx *TransactionRecord

	// Back-running transaction (attacker sell)
	BackRunTx *TransactionRecord

	// Detection timestamp
	DetectedAt int64

	// Pattern confidence score (0.0 to 1.0)
	ConfidenceScore math.LegacyDec

	// Whether this pattern was blocked
	Blocked bool

	// Reason for detection
	Reason string
}

// MEVDetectionResult contains the result of MEV detection checks
type MEVDetectionResult struct {
	// Whether MEV attack was detected
	Detected bool

	// Type of MEV attack (sandwich, front-running, etc.)
	AttackType string

	// Confidence score (0.0 to 1.0)
	Confidence math.LegacyDec

	// Whether the transaction should be blocked
	ShouldBlock bool

	// Detailed reason
	Reason string

	// Related transactions (if any)
	RelatedTxHashes []string

	// Suggested action
	SuggestedAction string
}

// TransactionBatch represents a batch of transactions for atomic execution
type TransactionBatch struct {
	// Batch ID
	BatchID string

	// Transactions in this batch
	Transactions []TransactionRecord

	// Batch status
	Status BatchStatus

	// Created at timestamp
	CreatedAt int64

	// Execute at block height
	ExecuteAtBlock int64
}

// BatchStatus represents the status of a transaction batch
type BatchStatus string

const (
	BatchStatusPending   BatchStatus = "pending"
	BatchStatusExecuting BatchStatus = "executing"
	BatchStatusExecuted  BatchStatus = "executed"
	BatchStatusFailed    BatchStatus = "failed"
)

// CommitRevealEntry represents a commit-reveal transaction entry
type CommitRevealEntry struct {
	// Commitment hash
	CommitHash string

	// Trader address
	Trader string

	// Committed at block height
	CommittedAt int64

	// Reveal timeout block height
	RevealDeadline int64

	// Whether revealed
	Revealed bool

	// Actual transaction data (after reveal)
	Transaction *TransactionRecord

	// Status
	Status CommitRevealStatus
}

// CommitRevealStatus represents the status of a commit-reveal entry
type CommitRevealStatus string

const (
	CommitRevealStatusCommitted CommitRevealStatus = "committed"
	CommitRevealStatusRevealed  CommitRevealStatus = "revealed"
	CommitRevealStatusExpired   CommitRevealStatus = "expired"
)

// PriceImpactCheck represents the result of a price impact check
type PriceImpactCheck struct {
	// Calculated price impact
	PriceImpact math.LegacyDec

	// Whether impact exceeds limit
	ExceedsLimit bool

	// Maximum allowed impact
	MaxAllowed math.LegacyDec

	// Pool ID
	PoolID uint64

	// Reserve state before swap
	ReserveInBefore  math.Int
	ReserveOutBefore math.Int

	// Reserve state after swap
	ReserveInAfter  math.Int
	ReserveOutAfter math.Int
}

// CalculatePriceImpact calculates the price impact of a swap
func CalculatePriceImpact(reserveIn, reserveOut, amountIn, amountOut math.Int) math.LegacyDec {
	// Price impact = |1 - (output_amount / expected_output_without_slippage)|
	// Expected output without slippage = amountIn * (reserveOut / reserveIn)

	if reserveIn.IsZero() || reserveOut.IsZero() {
		return math.LegacyZeroDec()
	}

	// Calculate expected output (without AMM curve impact)
	// expectedOut = amountIn * reserveOut / reserveIn
	expectedOut := amountIn.ToLegacyDec().Mul(reserveOut.ToLegacyDec()).Quo(reserveIn.ToLegacyDec())

	// Actual output
	actualOut := amountOut.ToLegacyDec()

	// Price impact = (expectedOut - actualOut) / expectedOut
	if expectedOut.IsZero() {
		return math.LegacyZeroDec()
	}

	impact := expectedOut.Sub(actualOut).Quo(expectedOut)

	// Return absolute value
	if impact.IsNegative() {
		return impact.Neg()
	}
	return impact
}

// TransactionOrdering represents the ordering information for a transaction
type TransactionOrdering struct {
	// Transaction hash
	TxHash string

	// Original timestamp from user
	Timestamp time.Time

	// Block height where transaction is included
	BlockHeight int64

	// Proposed ordering index
	OrderIndex int64

	// Whether timestamp was validated
	TimestampValidated bool

	// Ordering method used (timestamp, fee, random)
	OrderingMethod string
}

// MEVProtectionMetrics tracks metrics for MEV protection
type MEVProtectionMetrics struct {
	// Total transactions processed
	TotalTransactions uint64

	// Total MEV attacks detected
	TotalMEVDetected uint64

	// Total MEV attacks blocked
	TotalMEVBlocked uint64

	// Sandwich attacks detected
	SandwichAttacksDetected uint64

	// Front-running detected
	FrontRunningDetected uint64

	// Back-running detected
	BackRunningDetected uint64

	// Price impact violations
	PriceImpactViolations uint64

	// Timestamp ordering enforced
	TimestampOrderingEnforced uint64

	// Last updated
	LastUpdated int64
}

// NewTransactionRecord creates a new transaction record
func NewTransactionRecord(
	txHash string,
	trader string,
	poolID uint64,
	tokenIn, tokenOut string,
	amountIn, amountOut math.Int,
	timestamp int64,
	blockHeight int64,
	txIndex int64,
	priceImpact math.LegacyDec,
) TransactionRecord {
	return TransactionRecord{
		TxHash:      txHash,
		Trader:      trader,
		PoolID:      poolID,
		TokenIn:     tokenIn,
		TokenOut:    tokenOut,
		AmountIn:    amountIn,
		AmountOut:   amountOut,
		Timestamp:   timestamp,
		BlockHeight: blockHeight,
		TxIndex:     txIndex,
		PriceImpact: priceImpact,
	}
}

// IsSameDirection checks if two transactions trade in the same direction
func (tr TransactionRecord) IsSameDirection(other TransactionRecord) bool {
	return tr.TokenIn == other.TokenIn && tr.TokenOut == other.TokenOut
}

// IsOppositeDirection checks if two transactions trade in opposite directions
func (tr TransactionRecord) IsOppositeDirection(other TransactionRecord) bool {
	return tr.TokenIn == other.TokenOut && tr.TokenOut == other.TokenIn
}

// GetKey returns a unique key for this transaction record
func (tr TransactionRecord) GetKey() []byte {
	return []byte(tr.TxHash)
}

// String returns a string representation of the MEV detection result
func (mdr MEVDetectionResult) String() string {
	return fmt.Sprintf(
		"MEV Detection: detected=%t, type=%s, confidence=%.4f, block=%t, reason=%s",
		mdr.Detected,
		mdr.AttackType,
		mdr.Confidence,
		mdr.ShouldBlock,
		mdr.Reason,
	)
}
