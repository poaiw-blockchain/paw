package types

// Event types for the DEX module
// All event types use lowercase with underscore separator (module_action format)
const (
	// Swap events
	EventTypeDexSwap           = "dex_swap"
	EventTypeDexSwapFailed     = "dex_swap_failed"
	EventTypeDexLimitOrder     = "dex_limit_order"
	EventTypeDexOrderFilled    = "dex_order_filled"
	EventTypeDexOrderCancelled = "dex_limit_order_cancelled"
	EventTypeDexOrderPlaced    = "dex_limit_order_placed"
	EventTypeDexOrderMatched   = "dex_limit_order_matched"
	EventTypeDexOrderExpired   = "dex_limit_order_expired"
	EventTypeDexOrdersPruned   = "dex_limit_orders_pruned"

	// Liquidity events
	EventTypeDexAddLiquidity    = "dex_add_liquidity"
	EventTypeDexRemoveLiquidity = "dex_remove_liquidity"
	EventTypeDexLiquidityLocked = "dex_liquidity_locked"

	// Pool events
	EventTypeDexPoolCreated = "dex_pool_created"
	EventTypeDexPoolUpdated = "dex_pool_updated"

	// Security events
	EventTypeDexLargeSwap               = "dex_large_swap"
	EventTypeDexLargeLiquidityAddition  = "dex_large_liquidity_addition"
	EventTypeDexSlippageExceeded        = "dex_slippage_exceeded"
	EventTypeDexPotentialSandwichAttack = "dex_potential_sandwich_attack"

	// Cross-chain events
	EventTypeDexCrossChainSwap        = "dex_cross_chain_swap"
	EventTypeDexCrossChainSwapTimeout = "dex_cross_chain_swap_timeout"
	EventTypeDexCrossChainSwapFailed  = "dex_cross_chain_swap_failed"

	// Fee events
	EventTypeDexFeeCollected = "dex_fee_collected"
	EventTypeDexFeeUpdated   = "dex_fee_updated"
)

// Event attribute keys for the DEX module
// All attribute keys use lowercase with underscore separator
const (
	// Common attributes
	AttributeKeySender    = "sender"
	AttributeKeyTrader    = "trader"
	AttributeKeyProvider  = "provider"
	AttributeKeyRecipient = "recipient"
	AttributeKeyPoolID    = "pool_id"

	// Token attributes
	AttributeKeyTokenIn  = "token_in"
	AttributeKeyTokenOut = "token_out"
	AttributeKeyTokenA   = "token_a"
	AttributeKeyTokenB   = "token_b"
	AttributeKeyDenom    = "denom"

	// Amount attributes
	AttributeKeyAmount         = "amount"
	AttributeKeyAmountIn       = "amount_in"
	AttributeKeyAmountOut      = "amount_out"
	AttributeKeyMinAmountOut   = "min_amount_out"
	AttributeKeyAmountA        = "amount_a"
	AttributeKeyAmountB        = "amount_b"
	AttributeKeyRefundedAmount = "refunded_amount"

	// Liquidity attributes
	AttributeKeyShares       = "shares"
	AttributeKeyLockedShares = "locked_shares"
	AttributeKeyTotalShares  = "total_shares"

	// Reserve attributes
	AttributeKeyReserveA = "reserve_a"
	AttributeKeyReserveB = "reserve_b"

	// Fee attributes
	AttributeKeyFee         = "fee"
	AttributeKeyFeeAmount   = "fee_amount"
	AttributeKeyLpFee       = "lp_fee"
	AttributeKeyProtocolFee = "protocol_fee"

	// Price attributes
	AttributeKeyPrice       = "price"
	AttributeKeyPriceImpact = "price_impact"
	AttributeKeySlippage    = "slippage"

	// Order attributes
	AttributeKeyOrderID     = "order_id"
	AttributeKeyOrderType   = "order_type"
	AttributeKeyLimitPrice  = "limit_price"
	AttributeKeyOrderStatus = "order_status"

	// Security attributes
	AttributeKeyPercentage     = "percentage"
	AttributeKeySwapPercentage = "swap_percentage"
	AttributeKeyBlocksApart    = "blocks_apart"
	AttributeKeyMaxAllowed     = "max_allowed"
	AttributeKeyBlockHeight    = "block_height"
	AttributeKeyTimestamp      = "timestamp"

	// Cross-chain attributes
	AttributeKeySwapID      = "swap_id"
	AttributeKeyJobID       = "job_id"
	AttributeKeyTargetChain = "target_chain"
	AttributeKeyError       = "error"
	AttributeKeyUserAddress = "user_address"

	// Status attributes
	AttributeKeyStatus = "status"
	AttributeKeyReason = "reason"
	AttributeKeyActor  = "actor"
)

// Circuit breaker event types
const (
	EventTypeCircuitBreakerOpen  = "dex_circuit_breaker_open"
	EventTypeCircuitBreakerClose = "dex_circuit_breaker_close"
)
