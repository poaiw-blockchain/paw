package database

import (
	"encoding/json"
	"time"
)

// Block represents a blockchain block
type Block struct {
	Height          int64     `json:"height"`
	Hash            string    `json:"hash"`
	ProposerAddress string    `json:"proposer_address"`
	Time            time.Time `json:"time"`
	TxCount         int       `json:"tx_count"`
	GasUsed         int64     `json:"gas_used"`
	GasWanted       int64     `json:"gas_wanted"`
	EvidenceCount   int       `json:"evidence_count"`
}

// Transaction represents a blockchain transaction
type Transaction struct {
	Hash        string          `json:"hash"`
	BlockHeight int64           `json:"block_height"`
	TxIndex     int             `json:"tx_index"`
	Type        string          `json:"type"`
	Sender      string          `json:"sender"`
	Status      string          `json:"status"`
	Code        int             `json:"code"`
	GasUsed     int64           `json:"gas_used"`
	GasWanted   int64           `json:"gas_wanted"`
	FeeAmount   string          `json:"fee_amount"`
	FeeDenom    string          `json:"fee_denom"`
	Memo        string          `json:"memo"`
	RawLog      string          `json:"raw_log"`
	Time        time.Time       `json:"time"`
	Messages    json.RawMessage `json:"messages"`
	Events      json.RawMessage `json:"events"`
}

// Event represents a transaction event
type Event struct {
	TxHash      string          `json:"tx_hash"`
	BlockHeight int64           `json:"block_height"`
	EventType   string          `json:"event_type"`
	Module      string          `json:"module"`
	Attributes  json.RawMessage `json:"attributes"`
}

// DEXPool represents a DEX liquidity pool
type DEXPool struct {
	PoolID        string  `json:"pool_id"`
	TokenA        string  `json:"token_a"`
	TokenB        string  `json:"token_b"`
	ReserveA      float64 `json:"reserve_a"`
	ReserveB      float64 `json:"reserve_b"`
	LPTokenSupply float64 `json:"lp_token_supply"`
	SwapFeeRate   float64 `json:"swap_fee_rate"`
	TVL           float64 `json:"tvl"`
	CreatedHeight int64   `json:"created_height"`
}

// DEXSwap represents a DEX swap transaction
type DEXSwap struct {
	TxHash    string    `json:"tx_hash"`
	PoolID    string    `json:"pool_id"`
	Sender    string    `json:"sender"`
	TokenIn   string    `json:"token_in"`
	TokenOut  string    `json:"token_out"`
	AmountIn  float64   `json:"amount_in"`
	AmountOut float64   `json:"amount_out"`
	Price     float64   `json:"price"`
	Fee       float64   `json:"fee"`
	Time      time.Time `json:"time"`
}

// OraclePrice represents an oracle price feed
type OraclePrice struct {
	Asset       string    `json:"asset"`
	Price       float64   `json:"price"`
	Timestamp   time.Time `json:"timestamp"`
	BlockHeight int64     `json:"block_height"`
	Source      string    `json:"source"`
}

// Validator represents a blockchain validator
type Validator struct {
	Address                 string  `json:"address"`
	OperatorAddress         string  `json:"operator_address"`
	ConsensusPubkey         string  `json:"consensus_pubkey"`
	Moniker                 string  `json:"moniker"`
	CommissionRate          float64 `json:"commission_rate"`
	CommissionMaxRate       float64 `json:"commission_max_rate"`
	CommissionMaxChangeRate float64 `json:"commission_max_change_rate"`
	VotingPower             int64   `json:"voting_power"`
	Jailed                  bool    `json:"jailed"`
	Status                  string  `json:"status"`
	Tokens                  int64   `json:"tokens"`
	DelegatorShares         float64 `json:"delegator_shares"`
}

// ComputeRequest represents a compute module request
type ComputeRequest struct {
	RequestID          string    `json:"request_id"`
	Requester          string    `json:"requester"`
	Provider           string    `json:"provider"`
	Status             string    `json:"status"`
	TaskType           string    `json:"task_type"`
	PaymentAmount      float64   `json:"payment_amount"`
	PaymentDenom       string    `json:"payment_denom"`
	EscrowAmount       float64   `json:"escrow_amount"`
	ResultHash         string    `json:"result_hash"`
	VerificationStatus string    `json:"verification_status"`
	CreatedHeight      int64     `json:"created_height"`
	CompletedHeight    int64     `json:"completed_height"`
	CreatedAt          time.Time `json:"created_at"`
}

// IndexingProgress represents the current indexing progress
type IndexingProgress struct {
	LastIndexedHeight  int64      `json:"last_indexed_height"`
	TotalBlocksIndexed int64      `json:"total_blocks_indexed"`
	Status             string     `json:"status"`
	StartHeight        *int64     `json:"start_height"`
	TargetHeight       *int64     `json:"target_height"`
	StartedAt          *time.Time `json:"started_at"`
	CompletedAt        *time.Time `json:"completed_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// FailedBlock represents a block that failed to index
type FailedBlock struct {
	Height       int64      `json:"height"`
	ErrorMessage string     `json:"error_message"`
	RetryCount   int        `json:"retry_count"`
	LastRetryAt  *time.Time `json:"last_retry_at"`
}

// IndexingMetric represents an indexing performance metric
type IndexingMetric struct {
	Name            string   `json:"name"`
	Value           float64  `json:"value"`
	StartHeight     *int64   `json:"start_height"`
	EndHeight       *int64   `json:"end_height"`
	BlocksProcessed *int64   `json:"blocks_processed"`
	DurationSeconds *float64 `json:"duration_seconds"`
	BlocksPerSecond *float64 `json:"blocks_per_second"`
}

// IndexingCheckpoint represents a progress checkpoint
type IndexingCheckpoint struct {
	Height                    int64   `json:"height"`
	BlockHash                 string  `json:"block_hash"`
	BlocksSinceLastCheckpoint int     `json:"blocks_since_last_checkpoint"`
	TimeSinceLastCheckpoint   string  `json:"time_since_last_checkpoint"` // PostgreSQL INTERVAL as string
	AvgBlocksPerSecond        float64 `json:"avg_blocks_per_second"`
	Status                    string  `json:"status"`
}

// IndexingStatistics represents comprehensive indexing statistics
type IndexingStatistics struct {
	TotalBlocksIndexed      int64      `json:"total_blocks_indexed"`
	LastIndexedHeight       int64      `json:"last_indexed_height"`
	CurrentStatus           string     `json:"current_status"`
	FailedBlocksCount       int64      `json:"failed_blocks_count"`
	UnresolvedFailedBlocks  int64      `json:"unresolved_failed_blocks"`
	AvgBlocksPerSecond      *float64   `json:"avg_blocks_per_second"`
	EstimatedCompletionTime *time.Time `json:"estimated_completion_time"`
}
