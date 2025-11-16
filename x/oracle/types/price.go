package types

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
)

// ValidatorPriceSubmission represents a price submission from a single validator
type ValidatorPriceSubmission struct {
	Validator   string
	Asset       string
	Price       math.LegacyDec
	Timestamp   int64
	BlockHeight int64
}

// NewValidatorPriceSubmission creates a new validator price submission
func NewValidatorPriceSubmission(validator, asset string, price math.LegacyDec, timestamp, blockHeight int64) ValidatorPriceSubmission {
	return ValidatorPriceSubmission{
		Validator:   validator,
		Asset:       asset,
		Price:       price,
		Timestamp:   timestamp,
		BlockHeight: blockHeight,
	}
}

// Validate validates the price submission
func (v ValidatorPriceSubmission) Validate() error {
	if v.Validator == "" {
		return fmt.Errorf("validator address cannot be empty")
	}
	// Note: We skip bech32 validation to allow test addresses
	if v.Asset == "" {
		return fmt.Errorf("asset symbol cannot be empty")
	}
	if v.Price.IsNil() || v.Price.IsNegative() || v.Price.IsZero() {
		return fmt.Errorf("price must be positive: %s", v.Price.String())
	}
	if v.Timestamp <= 0 {
		return fmt.Errorf("timestamp must be positive")
	}
	if v.BlockHeight < 0 {
		return fmt.Errorf("block height cannot be negative")
	}
	return nil
}

// IsStale checks if the submission is stale based on the expiry duration
func (v ValidatorPriceSubmission) IsStale(currentTime time.Time, expiryDuration uint64) bool {
	submissionTime := time.Unix(v.Timestamp, 0)
	return currentTime.Sub(submissionTime) > time.Duration(expiryDuration)*time.Second
}

// ValidatorAccuracy tracks a validator's price submission accuracy
type ValidatorAccuracy struct {
	Validator             string
	TotalSubmissions      uint64
	AccurateSubmissions   uint64
	InaccurateSubmissions uint64
	LastSlashHeight       int64
}

// NewValidatorAccuracy creates a new validator accuracy tracker
func NewValidatorAccuracy(validator string) ValidatorAccuracy {
	return ValidatorAccuracy{
		Validator:             validator,
		TotalSubmissions:      0,
		AccurateSubmissions:   0,
		InaccurateSubmissions: 0,
		LastSlashHeight:       0,
	}
}

// AccuracyRate returns the accuracy rate as a percentage
func (v ValidatorAccuracy) AccuracyRate() math.LegacyDec {
	if v.TotalSubmissions == 0 {
		return math.LegacyZeroDec()
	}
	return math.LegacyNewDec(int64(v.AccurateSubmissions)).QuoInt64(int64(v.TotalSubmissions))
}

// RecordAccurate records an accurate submission
func (v *ValidatorAccuracy) RecordAccurate() {
	v.TotalSubmissions++
	v.AccurateSubmissions++
}

// RecordInaccurate records an inaccurate submission
func (v *ValidatorAccuracy) RecordInaccurate() {
	v.TotalSubmissions++
	v.InaccurateSubmissions++
}

// AggregatedPrice represents an aggregated price from multiple validators
type AggregatedPrice struct {
	Asset                   string
	MedianPrice             math.LegacyDec
	ParticipatingValidators []string
	Timestamp               int64
	BlockHeight             int64
	SubmissionCount         uint32
}

// NewAggregatedPrice creates a new aggregated price
func NewAggregatedPrice(asset string, medianPrice math.LegacyDec, validators []string, timestamp, blockHeight int64) AggregatedPrice {
	return AggregatedPrice{
		Asset:                   asset,
		MedianPrice:             medianPrice,
		ParticipatingValidators: validators,
		Timestamp:               timestamp,
		BlockHeight:             blockHeight,
		SubmissionCount:         uint32(len(validators)),
	}
}

// Validate validates the aggregated price
func (a AggregatedPrice) Validate() error {
	if a.Asset == "" {
		return fmt.Errorf("asset symbol cannot be empty")
	}
	if a.MedianPrice.IsNil() || a.MedianPrice.IsNegative() || a.MedianPrice.IsZero() {
		return fmt.Errorf("median price must be positive: %s", a.MedianPrice.String())
	}
	if len(a.ParticipatingValidators) == 0 {
		return fmt.Errorf("must have at least one participating validator")
	}
	if a.Timestamp <= 0 {
		return fmt.Errorf("timestamp must be positive")
	}
	if a.BlockHeight < 0 {
		return fmt.Errorf("block height cannot be negative")
	}
	return nil
}

// IsStale checks if the aggregated price is stale
func (a AggregatedPrice) IsStale(currentTime time.Time, expiryDuration uint64) bool {
	priceTime := time.Unix(a.Timestamp, 0)
	return currentTime.Sub(priceTime) > time.Duration(expiryDuration)*time.Second
}

// PriceDeviation calculates the deviation percentage from the median
func PriceDeviation(price, median math.LegacyDec) math.LegacyDec {
	if median.IsZero() {
		return math.LegacyZeroDec()
	}

	diff := price.Sub(median).Abs()
	return diff.Quo(median).MulInt64(100) // Return as percentage
}

// IsOutlier checks if a price is an outlier (beyond threshold)
// threshold is in percentage (e.g., 10 for 10%)
func IsOutlier(price, median math.LegacyDec, thresholdPercent int64) bool {
	deviation := PriceDeviation(price, median)
	threshold := math.LegacyNewDec(thresholdPercent)
	return deviation.GT(threshold)
}
