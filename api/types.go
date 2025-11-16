package api

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ==================== Authentication Types ====================

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
	Recover  bool   `json:"recover,omitempty"`  // If true, recover from mnemonic
	Mnemonic string `json:"mnemonic,omitempty"` // BIP39 mnemonic for recovery
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in"` // Seconds until access token expires
	Username     string `json:"username"`
	UserID       string `json:"user_id"`
	Address      string `json:"address,omitempty"`
	Mnemonic     string `json:"mnemonic,omitempty"` // Only included on new wallet creation
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshTokenResponse represents a token refresh response
type RefreshTokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"` // Seconds until access token expires
}

// User represents a user in the system
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Address      string    `json:"address"`
	CreatedAt    time.Time `json:"created_at"`
}

// ==================== Trading Types ====================

// CreateOrderRequest represents an order creation request
type CreateOrderRequest struct {
	OrderType string  `json:"order_type" binding:"required,oneof=buy sell"`
	Price     float64 `json:"price" binding:"required,gt=0"`
	Amount    float64 `json:"amount" binding:"required,gt=0"`
}

// CreateOrderResponse represents an order creation response
type CreateOrderResponse struct {
	OrderID   string  `json:"order_id"`
	OrderType string  `json:"order_type"`
	Price     float64 `json:"price"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	Timestamp int64   `json:"timestamp"`
}

// Order represents a DEX order
type Order struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username,omitempty"`
	Type      string    `json:"order_type"` // "buy" or "sell"
	Price     float64   `json:"price"`
	Amount    float64   `json:"amount"`
	Filled    float64   `json:"filled"`
	Status    string    `json:"status"` // "open", "filled", "cancelled", "partial"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// OrderBook represents the order book structure
type OrderBook struct {
	Bids   []OrderBookEntry `json:"bids"` // Buy orders
	Asks   []OrderBookEntry `json:"asks"` // Sell orders
	Spread float64          `json:"spread"`
}

// OrderBookEntry represents a single order book entry
type OrderBookEntry struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
	Total  float64 `json:"total"`
	Count  int     `json:"count,omitempty"`
}

// Trade represents an executed trade
type Trade struct {
	ID        string    `json:"id"`
	BuyerID   string    `json:"buyer_id,omitempty"`
	SellerID  string    `json:"seller_id,omitempty"`
	OrderType string    `json:"order_type"` // Direction of taker order
	Price     float64   `json:"price"`
	Amount    float64   `json:"amount"`
	Total     float64   `json:"total"`
	Timestamp time.Time `json:"timestamp"`
}

// TradeHistoryResponse represents trade history response
type TradeHistoryResponse struct {
	Trades     []Trade `json:"trades"`
	TotalCount int     `json:"total_count"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
}

// ==================== Wallet Types ====================

// BalanceResponse represents wallet balance response
type BalanceResponse struct {
	Address     string  `json:"address"`
	AIXNBalance float64 `json:"aixn_balance"`
	USDBalance  float64 `json:"usd_balance,omitempty"` // Optional fiat equivalent
	PAWBalance  string  `json:"paw_balance"`           // Native token balance
}

// SendTokensRequest represents a token transfer request
type SendTokensRequest struct {
	ToAddress string `json:"to_address" binding:"required"`
	Amount    string `json:"amount" binding:"required"`
	Denom     string `json:"denom" binding:"required"`
	Memo      string `json:"memo,omitempty"`
}

// SendTokensResponse represents a token transfer response
type SendTokensResponse struct {
	TxHash    string `json:"tx_hash"`
	Height    int64  `json:"height"`
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// Transaction represents a blockchain transaction
type Transaction struct {
	Hash      string    `json:"hash"`
	Height    int64     `json:"height"`
	Type      string    `json:"type"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    string    `json:"amount"`
	Denom     string    `json:"denom"`
	Fee       string    `json:"fee"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Memo      string    `json:"memo,omitempty"`
}

// TransactionHistoryResponse represents transaction history
type TransactionHistoryResponse struct {
	Transactions []Transaction `json:"transactions"`
	TotalCount   int           `json:"total_count"`
	Page         int           `json:"page"`
	PageSize     int           `json:"page_size"`
}

// ==================== Light Client Types ====================

// HeaderResponse represents a block header response
type HeaderResponse struct {
	Height    int64     `json:"height"`
	Hash      string    `json:"hash"`
	Time      time.Time `json:"time"`
	ChainID   string    `json:"chain_id"`
	Proposer  string    `json:"proposer"`
	LastHash  string    `json:"last_hash"`
	DataHash  string    `json:"data_hash"`
	Signature string    `json:"signature,omitempty"`
}

// CheckpointResponse represents a checkpoint for light client
type CheckpointResponse struct {
	Height           int64     `json:"height"`
	Hash             string    `json:"hash"`
	ValidatorSetHash string    `json:"validator_set_hash"`
	Timestamp        time.Time `json:"timestamp"`
	TrustedHeight    int64     `json:"trusted_height"`
}

// TxProofResponse represents a transaction proof
type TxProofResponse struct {
	TxHash    string   `json:"tx_hash"`
	Height    int64    `json:"height"`
	Index     int      `json:"index"`
	Proof     []string `json:"proof"` // Merkle proof
	Data      string   `json:"data"`  // Transaction data
	BlockHash string   `json:"block_hash"`
	Verified  bool     `json:"verified"`
}

// VerifyProofRequest represents a proof verification request
type VerifyProofRequest struct {
	TxHash    string   `json:"tx_hash" binding:"required"`
	Height    int64    `json:"height" binding:"required"`
	Proof     []string `json:"proof" binding:"required"`
	BlockHash string   `json:"block_hash" binding:"required"`
}

// ==================== Atomic Swap Types ====================

// PrepareSwapRequest represents a swap preparation request
type PrepareSwapRequest struct {
	CounterpartyAddress string `json:"counterparty_address" binding:"required"`
	SendAmount          string `json:"send_amount" binding:"required"`
	SendDenom           string `json:"send_denom" binding:"required"`
	ReceiveAmount       string `json:"receive_amount" binding:"required"`
	ReceiveDenom        string `json:"receive_denom" binding:"required"`
	TimeLockDuration    int64  `json:"timelock_duration"` // Seconds
	HashLock            string `json:"hash_lock,omitempty"`
}

// PrepareSwapResponse represents a swap preparation response
type PrepareSwapResponse struct {
	SwapID          string    `json:"swap_id"`
	HashLock        string    `json:"hash_lock"`
	Secret          string    `json:"secret,omitempty"`
	TimeLock        int64     `json:"time_lock"`
	Status          string    `json:"status"`
	ExpiresAt       time.Time `json:"expires_at"`
	ContractAddress string    `json:"contract_address,omitempty"`
}

// CommitSwapRequest represents a swap commitment request
type CommitSwapRequest struct {
	SwapID string `json:"swap_id" binding:"required"`
	Secret string `json:"secret,omitempty"`
}

// CommitSwapResponse represents a swap commitment response
type CommitSwapResponse struct {
	SwapID    string `json:"swap_id"`
	TxHash    string `json:"tx_hash"`
	Status    string `json:"status"`
	Completed bool   `json:"completed"`
	Message   string `json:"message,omitempty"`
}

// AtomicSwap represents an atomic swap
type AtomicSwap struct {
	ID                    string    `json:"id"`
	Initiator             string    `json:"initiator"`
	Counterparty          string    `json:"counterparty"`
	SendAmount            string    `json:"send_amount"`
	SendDenom             string    `json:"send_denom"`
	ReceiveAmount         string    `json:"receive_amount"`
	ReceiveDenom          string    `json:"receive_denom"`
	HashLock              string    `json:"hash_lock"`
	Secret                string    `json:"secret,omitempty"`
	TimeLock              int64     `json:"time_lock"`
	Status                string    `json:"status"` // "pending", "committed", "refunded", "expired"
	InitiatorCommitted    bool      `json:"initiator_committed"`
	CounterpartyCommitted bool      `json:"counterparty_committed"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
	ExpiresAt             time.Time `json:"expires_at"`
}

// SwapStatusResponse represents swap status
type SwapStatusResponse struct {
	Swap      *AtomicSwap `json:"swap"`
	CanCommit bool        `json:"can_commit"`
	CanRefund bool        `json:"can_refund"`
	Message   string      `json:"message,omitempty"`
}

// ==================== Pool/Liquidity Types ====================

// Pool represents a liquidity pool
type Pool struct {
	ID              string  `json:"id"`
	TokenA          string  `json:"token_a"`
	TokenB          string  `json:"token_b"`
	ReserveA        string  `json:"reserve_a"`
	ReserveB        string  `json:"reserve_b"`
	LiquidityShares string  `json:"liquidity_shares"`
	SwapFee         float64 `json:"swap_fee"`
	PoolType        string  `json:"pool_type"` // "amm", "orderbook"
}

// AddLiquidityRequest represents add liquidity request
type AddLiquidityRequest struct {
	PoolID   string  `json:"pool_id" binding:"required"`
	AmountA  string  `json:"amount_a" binding:"required"`
	AmountB  string  `json:"amount_b" binding:"required"`
	Slippage float64 `json:"slippage"`
}

// RemoveLiquidityRequest represents remove liquidity request
type RemoveLiquidityRequest struct {
	PoolID     string `json:"pool_id" binding:"required"`
	Shares     string `json:"shares" binding:"required"`
	MinAmountA string `json:"min_amount_a"`
	MinAmountB string `json:"min_amount_b"`
}

// ==================== Market Data Types ====================

// PriceResponse represents current price data
type PriceResponse struct {
	CurrentPrice     float64   `json:"current_price"`
	Change24h        float64   `json:"change_24h"`
	ChangePercent24h float64   `json:"change_percent_24h"`
	High24h          float64   `json:"high_24h"`
	Low24h           float64   `json:"low_24h"`
	Volume24h        float64   `json:"volume_24h"`
	LastUpdated      time.Time `json:"last_updated"`
}

// MarketStats represents comprehensive market statistics
type MarketStats struct {
	Price                 float64   `json:"price"`
	Volume24h             float64   `json:"volume_24h"`
	VolumeChange24h       float64   `json:"volume_change_24h"`
	High24h               float64   `json:"high_24h"`
	Low24h                float64   `json:"low_24h"`
	PriceChange24h        float64   `json:"price_change_24h"`
	PriceChangePercent24h float64   `json:"price_change_percent_24h"`
	MarketCap             float64   `json:"market_cap,omitempty"`
	TotalLiquidity        float64   `json:"total_liquidity"`
	TotalTrades           int64     `json:"total_trades"`
	LastUpdated           time.Time `json:"last_updated"`
}

// ==================== WebSocket Types ====================

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string      `json:"type"`
	Channel string      `json:"channel,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// WSSubscribeMessage represents a subscription message
type WSSubscribeMessage struct {
	Type    string `json:"type"`    // "subscribe" or "unsubscribe"
	Channel string `json:"channel"` // "price", "orderbook", "trades"
}

// ==================== Common Response Types ====================

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page     int `form:"page" json:"page"`
	PageSize int `form:"page_size" json:"page_size"`
	Offset   int `json:"offset"`
}

// DefaultPagination returns default pagination parameters
func DefaultPagination() PaginationParams {
	return PaginationParams{
		Page:     1,
		PageSize: 20,
		Offset:   0,
	}
}

// ==================== Helper Types ====================

// CoinsFromString converts string amount to sdk.Coins
func CoinsFromString(amount, denom string) (sdk.Coins, error) {
	coin, err := sdk.ParseCoinNormalized(amount + denom)
	if err != nil {
		return nil, err
	}
	return sdk.NewCoins(coin), nil
}
