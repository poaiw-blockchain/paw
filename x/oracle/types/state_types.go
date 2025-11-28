package types

import (
	"cosmossdk.io/math"
)

// Structs are defined in oracle.pb.go

// SlashingInfo tracks validator slashing information
type SlashingInfo struct {
	Validator     string `json:"validator"`
	MissCount     uint64 `json:"miss_count"`
	SlashedAmount math.LegacyDec `json:"slashed_amount"`
	SlashedHeight int64  `json:"slashed_height,omitempty"`
	JailedUntil   int64  `json:"jailed_until,omitempty"`
}

// TWAPDataPoint represents a single TWAP data point
type TWAPDataPoint struct {
	Price      math.LegacyDec `json:"price"`
	Timestamp  int64   `json:"timestamp"`
	Volume     math.LegacyDec `json:"volume,omitempty"`
	BlockHeight int64  `json:"block_height"`
}

// AggregateExchangeRatePrevote represents the aggregate vote hash submitted by a validator
type AggregateExchangeRatePrevote struct {
	Hash        string `json:"hash"`
	Voter       string `json:"voter"`
	SubmitBlock uint64 `json:"submit_block"`
}

// AggregateExchangeRateVote represents the aggregate vote submitted by a validator
type AggregateExchangeRateVote struct {
	ExchangeRates []math.LegacyDec `json:"exchange_rates"`
	Voter         string           `json:"voter"`
}

// Key prefixes for store
var (
	PriceKeyPrefix        = []byte{0x01}
	PrevoteKeyPrefix      = []byte{0x02}
	VoteKeyPrefix         = []byte{0x03}
	DelegateKeyPrefix     = []byte{0x04}
	MissCounterKeyPrefix  = []byte{0x05}
	SlashingKeyPrefix     = []byte{0x06}
	TWAPKeyPrefix         = []byte{0x07}
)
