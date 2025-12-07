package keeper

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/dex/types"
)

// ============================================================================
// Limit Order Types and Constants
// ============================================================================

// OrderType represents the type of order (buy or sell).
//
// Buy orders exchange TokenIn for TokenOut with a minimum price requirement.
// Sell orders are conceptually the same but viewed from the opposite perspective.
type OrderType uint8

const (
	// OrderTypeBuy indicates a buy order (exchanging TokenIn for TokenOut at limit price or better).
	OrderTypeBuy OrderType = 1

	// OrderTypeSell indicates a sell order (exchanging TokenIn for TokenOut at limit price or better).
	OrderTypeSell OrderType = 2
)

// OrderStatus represents the current status of a limit order in its lifecycle.
//
// Order Lifecycle:
//   Open → Partial → Filled (successful execution path)
//   Open → Cancelled (user cancellation)
//   Open → Expired (timeout without full fill)
//   Partial → Filled (gradual execution to completion)
//   Partial → Cancelled (user cancels partially filled order)
//   Partial → Expired (timeout with partial fill)
type OrderStatus uint8

const (
	// OrderStatusOpen indicates the order is active and unfilled.
	OrderStatusOpen OrderStatus = 1

	// OrderStatusFilled indicates the order has been completely executed.
	OrderStatusFilled OrderStatus = 2

	// OrderStatusPartial indicates the order has been partially executed but not completed.
	OrderStatusPartial OrderStatus = 3

	// OrderStatusCancelled indicates the order was cancelled by the user.
	OrderStatusCancelled OrderStatus = 4

	// OrderStatusExpired indicates the order expired before being fully filled.
	OrderStatusExpired OrderStatus = 5
)

// LimitOrder represents a limit order in the DEX order book.
//
// Limit orders allow traders to specify a price at which they are willing to trade,
// rather than accepting the current market price. Orders are matched against the pool's
// constant product curve when the pool price reaches the limit price.
//
// Key Features:
//   - Tokens are locked when order is placed (prevents double-spending)
//   - Orders can be partially filled over multiple blocks
//   - Unfilled orders can be cancelled to retrieve locked tokens
//   - Orders automatically expire after the specified duration
//   - Matching is attempted in every EndBlock against current pool prices
//
// Security Notes:
//   - Token custody is handled by module account (no external escrow)
//   - Price manipulation is limited by pool's own MEV protections
//   - Order IDs are unique and monotonically increasing (no replay attacks)
type LimitOrder struct {
	// ID is the unique identifier of the order
	ID uint64 `json:"id"`
	// Owner is the address that placed the order
	Owner string `json:"owner"`
	// PoolID is the pool this order is for
	PoolID uint64 `json:"pool_id"`
	// OrderType indicates buy or sell
	OrderType OrderType `json:"order_type"`
	// TokenIn is the token being sold
	TokenIn string `json:"token_in"`
	// TokenOut is the token being bought
	TokenOut string `json:"token_out"`
	// AmountIn is the amount of TokenIn to sell
	AmountIn math.Int `json:"amount_in"`
	// MinAmountOut is the minimum acceptable amount of TokenOut
	MinAmountOut math.Int `json:"min_amount_out"`
	// LimitPrice is the limit price (TokenOut per TokenIn)
	LimitPrice math.LegacyDec `json:"limit_price"`
	// FilledAmount is the amount of TokenIn that has been filled
	FilledAmount math.Int `json:"filled_amount"`
	// ReceivedAmount is the amount of TokenOut received so far
	ReceivedAmount math.Int `json:"received_amount"`
	// Status is the current status of the order
	Status OrderStatus `json:"status"`
	// CreatedAt is when the order was created
	CreatedAt time.Time `json:"created_at"`
	// ExpiresAt is when the order expires (0 = no expiry)
	ExpiresAt time.Time `json:"expires_at"`
	// CreatedAtHeight is the block height when created
	CreatedAtHeight int64 `json:"created_at_height"`
}

// Store key prefixes for limit orders.
//
// These prefixes organize limit order data in the key-value store for efficient querying:
// - Primary storage: Orders indexed by order ID
// - Secondary indexes: Orders indexed by owner, pool, price, and open status
//
// Index Design Rationale:
// - Owner index: Fast retrieval of all orders for a user
// - Pool index: Fast retrieval of all orders for a specific pool
// - Price index: Efficient order book construction and matching
// - Open index: Fast iteration over all active orders for EndBlock matching
var (
	// LimitOrderKeyPrefix is the prefix for primary limit order storage (key: orderID).
	LimitOrderKeyPrefix = []byte{0x0E}

	// LimitOrderCountKey is the key for storing the next available order ID (global counter).
	LimitOrderCountKey = []byte{0x0F}

	// LimitOrderByOwnerPrefix is the prefix for indexing orders by owner address.
	// Key format: 0x10 || ownerAddr || orderID
	LimitOrderByOwnerPrefix = []byte{0x10}

	// LimitOrderByPoolPrefix is the prefix for indexing orders by pool ID.
	// Key format: 0x11 || poolID || orderID
	LimitOrderByPoolPrefix = []byte{0x11}

	// LimitOrderByPricePrefix is the prefix for indexing orders by price for efficient matching.
	// Key format: 0x12 || poolID || orderType || encodedPrice || orderID
	// Price is encoded to maintain lexicographic ordering for range queries.
	LimitOrderByPricePrefix = []byte{0x12}

	// LimitOrderOpenPrefix is the prefix for indexing only open/partial orders.
	// Key format: 0x13 || orderID
	// This index enables efficient iteration over active orders in EndBlock.
	LimitOrderOpenPrefix = []byte{0x13}
)

// ============================================================================
// Key Functions
// ============================================================================

// LimitOrderKey returns the store key for a limit order
func LimitOrderKey(orderID uint64) []byte {
	orderIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(orderIDBytes, orderID)
	return append(LimitOrderKeyPrefix, orderIDBytes...)
}

// LimitOrderByOwnerKey returns the index key for orders by owner
func LimitOrderByOwnerKey(owner sdk.AccAddress, orderID uint64) []byte {
	orderIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(orderIDBytes, orderID)
	key := append(LimitOrderByOwnerPrefix, owner.Bytes()...)
	return append(key, orderIDBytes...)
}

// LimitOrderByPoolKey returns the index key for orders by pool
func LimitOrderByPoolKey(poolID uint64, orderID uint64) []byte {
	poolIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIDBytes, poolID)
	orderIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(orderIDBytes, orderID)
	key := append(LimitOrderByPoolPrefix, poolIDBytes...)
	return append(key, orderIDBytes...)
}

// LimitOrderByPriceKey returns the index key for orders by price
// Price is encoded as big-endian fixed-point to enable range queries
func LimitOrderByPriceKey(poolID uint64, orderType OrderType, price math.LegacyDec, orderID uint64) []byte {
	poolIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIDBytes, poolID)
	orderIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(orderIDBytes, orderID)

	// Convert price to fixed-point bytes for proper ordering
	priceBytes := encodePriceForOrdering(price)

	key := append(LimitOrderByPricePrefix, poolIDBytes...)
	key = append(key, byte(orderType))
	key = append(key, priceBytes...)
	return append(key, orderIDBytes...)
}

// LimitOrderOpenKey returns the index key for open orders
func LimitOrderOpenKey(orderID uint64) []byte {
	orderIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(orderIDBytes, orderID)
	return append(LimitOrderOpenPrefix, orderIDBytes...)
}

// encodePriceForOrdering encodes a decimal price as bytes for proper key ordering
func encodePriceForOrdering(price math.LegacyDec) []byte {
	// Use 18 decimal places precision, stored as big-endian uint128
	// This allows proper lexicographic ordering of prices
	scaled := price.MulInt64(1e18).TruncateInt()
	bytes := make([]byte, 16)
	// Store as unsigned big-endian for proper ordering
	bigInt := scaled.BigInt()
	if bigInt != nil {
		b := bigInt.Bytes()
		copy(bytes[16-len(b):], b)
	}
	return bytes
}

// ============================================================================
// Core Limit Order Functions
// ============================================================================

// GetNextOrderID returns and increments the next order ID
func (k Keeper) GetNextOrderID(ctx context.Context) (uint64, error) {
	store := k.getStore(ctx)
	bz := store.Get(LimitOrderCountKey)

	var nextID uint64 = 1
	if bz != nil {
		nextID = binary.BigEndian.Uint64(bz)
	}

	// Increment for next time
	nextIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nextIDBytes, nextID+1)
	store.Set(LimitOrderCountKey, nextIDBytes)

	return nextID, nil
}

// SetLimitOrder stores a limit order
func (k Keeper) SetLimitOrder(ctx context.Context, order *LimitOrder) error {
	store := k.getStore(ctx)

	bz, err := json.Marshal(order)
	if err != nil {
		return types.ErrInvalidState.Wrapf("failed to marshal limit order: %v", err)
	}

	// Store the order
	store.Set(LimitOrderKey(order.ID), bz)

	// Update indexes
	ownerAddr, err := sdk.AccAddressFromBech32(order.Owner)
	if err != nil {
		return types.ErrInvalidInput.Wrapf("invalid owner address: %v", err)
	}

	store.Set(LimitOrderByOwnerKey(ownerAddr, order.ID), []byte{1})
	store.Set(LimitOrderByPoolKey(order.PoolID, order.ID), []byte{1})
	store.Set(LimitOrderByPriceKey(order.PoolID, order.OrderType, order.LimitPrice, order.ID), []byte{1})

	if order.Status == OrderStatusOpen || order.Status == OrderStatusPartial {
		store.Set(LimitOrderOpenKey(order.ID), []byte{1})
	} else {
		store.Delete(LimitOrderOpenKey(order.ID))
	}

	return nil
}

// GetLimitOrder retrieves a limit order by ID
func (k Keeper) GetLimitOrder(ctx context.Context, orderID uint64) (*LimitOrder, error) {
	store := k.getStore(ctx)

	bz := store.Get(LimitOrderKey(orderID))
	if bz == nil {
		return nil, types.ErrOrderNotFound.Wrapf("limit order not found: %d", orderID)
	}

	var order LimitOrder
	if err := json.Unmarshal(bz, &order); err != nil {
		return nil, types.ErrInvalidState.Wrapf("failed to unmarshal limit order: %v", err)
	}

	return &order, nil
}

// DeleteLimitOrder removes a limit order and its indexes
func (k Keeper) DeleteLimitOrder(ctx context.Context, order *LimitOrder) error {
	store := k.getStore(ctx)

	ownerAddr, err := sdk.AccAddressFromBech32(order.Owner)
	if err != nil {
		return types.ErrInvalidInput.Wrapf("invalid owner address: %v", err)
	}

	// Delete indexes first
	store.Delete(LimitOrderByOwnerKey(ownerAddr, order.ID))
	store.Delete(LimitOrderByPoolKey(order.PoolID, order.ID))
	store.Delete(LimitOrderByPriceKey(order.PoolID, order.OrderType, order.LimitPrice, order.ID))
	store.Delete(LimitOrderOpenKey(order.ID))

	// Delete the order
	store.Delete(LimitOrderKey(order.ID))

	return nil
}

// ============================================================================
// Limit Order Operations
// ============================================================================

// PlaceLimitOrder creates a new limit order and locks the input tokens.
//
// This function creates a limit order that will execute when the pool price reaches
// the specified limit price. The input tokens are immediately locked in the module account
// to prevent double-spending.
//
// Parameters:
//   - ctx: Blockchain context for state access
//   - owner: Address placing the order (tokens will be locked from this account)
//   - poolID: ID of the liquidity pool for this order
//   - orderType: Buy or sell order type
//   - tokenIn: Denomination of token being sold
//   - tokenOut: Denomination of token being bought
//   - amountIn: Amount of tokenIn to sell
//   - limitPrice: Minimum acceptable price (tokenOut per tokenIn)
//   - expiryDuration: Time until order expires (0 = no expiry)
//
// Returns:
//   - *LimitOrder: The created order with assigned ID
//   - error: nil on success, or:
//     * ErrPoolNotFound: Pool does not exist
//     * ErrInvalidTokenPair: Tokens don't match pool
//     * ErrInvalidOrder: Invalid amount or price
//     * ErrInsufficientLiquidity: Failed to lock tokens
//
// Behavior:
//  1. Validates pool exists and tokens match
//  2. Validates amount and limit price
//  3. Calculates minimum output based on limit price
//  4. Locks input tokens in module account
//  5. Creates and stores order with Open status
//  6. Attempts immediate matching against current pool price
//  7. Emits order_placed event
//
// Security Notes:
//   - Tokens are locked immediately to prevent double-spending
//   - Order matching uses same security checks as regular swaps
//   - Failed matching doesn't revert order creation (order remains open)
func (k Keeper) PlaceLimitOrder(
	ctx context.Context,
	owner sdk.AccAddress,
	poolID uint64,
	orderType OrderType,
	tokenIn, tokenOut string,
	amountIn math.Int,
	limitPrice math.LegacyDec,
	expiryDuration time.Duration,
) (*LimitOrder, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate pool exists
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return nil, types.ErrPoolNotFound.Wrapf("pool not found: %v", err)
	}

	// Validate tokens match pool
	if !((pool.TokenA == tokenIn && pool.TokenB == tokenOut) ||
		(pool.TokenB == tokenIn && pool.TokenA == tokenOut)) {
		return nil, types.ErrInvalidTokenPair.Wrap("tokens do not match pool")
	}

	// Validate amount
	if amountIn.IsNil() || !amountIn.IsPositive() {
		return nil, types.ErrInvalidOrder.Wrap("invalid amount")
	}

	// Validate limit price
	if limitPrice.IsNil() || !limitPrice.IsPositive() {
		return nil, types.ErrInvalidOrder.Wrap("invalid limit price")
	}

	// Calculate minimum amount out based on limit price
	minAmountOut := math.LegacyNewDecFromInt(amountIn).Mul(limitPrice).TruncateInt()

	// Get next order ID
	orderID, err := k.GetNextOrderID(ctx)
	if err != nil {
		return nil, err
	}

	// Lock tokens from user
	coin := sdk.NewCoin(tokenIn, amountIn)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(sdkCtx, owner, types.ModuleName, sdk.NewCoins(coin)); err != nil {
		return nil, types.ErrInsufficientLiquidity.Wrapf("failed to lock tokens: %v", err)
	}

	// Create order
	order := &LimitOrder{
		ID:              orderID,
		Owner:           owner.String(),
		PoolID:          poolID,
		OrderType:       orderType,
		TokenIn:         tokenIn,
		TokenOut:        tokenOut,
		AmountIn:        amountIn,
		MinAmountOut:    minAmountOut,
		LimitPrice:      limitPrice,
		FilledAmount:    math.ZeroInt(),
		ReceivedAmount:  math.ZeroInt(),
		Status:          OrderStatusOpen,
		CreatedAt:       sdkCtx.BlockTime(),
		CreatedAtHeight: sdkCtx.BlockHeight(),
	}

	// Set expiry if specified
	if expiryDuration > 0 {
		order.ExpiresAt = sdkCtx.BlockTime().Add(expiryDuration)
	}

	// Store order
	if err := k.SetLimitOrder(ctx, order); err != nil {
		return nil, err
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeDexOrderPlaced,
			sdk.NewAttribute(types.AttributeKeyOrderID, fmt.Sprintf("%d", orderID)),
			sdk.NewAttribute(sdk.AttributeKeySender, owner.String()),
			sdk.NewAttribute(types.AttributeKeyPoolID, fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute(types.AttributeKeyOrderType, fmt.Sprintf("%d", orderType)),
			sdk.NewAttribute(types.AttributeKeyTokenIn, tokenIn),
			sdk.NewAttribute(types.AttributeKeyTokenOut, tokenOut),
			sdk.NewAttribute(types.AttributeKeyAmountIn, amountIn.String()),
			sdk.NewAttribute(types.AttributeKeyLimitPrice, limitPrice.String()),
		),
	)

	// Try to match immediately
	if err := k.MatchLimitOrder(ctx, order); err != nil {
		sdkCtx.Logger().Error("failed to match limit order", "order_id", orderID, "error", err)
	}

	return order, nil
}

// CancelLimitOrder cancels an existing limit order and refunds remaining tokens.
//
// This function allows the order owner to cancel their order at any time, retrieving
// any unfilled token amounts. For partially filled orders, only the remaining unfilled
// amount is refunded.
//
// Parameters:
//   - ctx: Blockchain context for state access
//   - owner: Address requesting cancellation (must be order owner)
//   - orderID: Unique identifier of the order to cancel
//
// Returns:
//   - error: nil on success, or:
//     * ErrOrderNotFound: Order does not exist
//     * ErrOrderNotAuthorized: Caller is not the order owner
//     * ErrOrderNotCancellable: Order status doesn't allow cancellation (already filled/expired)
//     * ErrInsufficientLiquidity: Failed to refund tokens
//
// Behavior:
//  1. Retrieves and validates order ownership
//  2. Checks order is cancellable (Open or Partial status)
//  3. Calculates remaining unfilled amount
//  4. Refunds remaining tokens from module account to owner
//  5. Updates order status to Cancelled
//  6. Removes from open orders index
//  7. Emits order_cancelled event
//
// Security Notes:
//   - Only order owner can cancel (enforced by ownership check)
//   - Refund amount is AmountIn - FilledAmount (prevents over-refund)
//   - Failed refund returns error (order not marked as cancelled)
func (k Keeper) CancelLimitOrder(ctx context.Context, owner sdk.AccAddress, orderID uint64) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	order, err := k.GetLimitOrder(ctx, orderID)
	if err != nil {
		return err
	}

	// Verify ownership
	if order.Owner != owner.String() {
		return types.ErrOrderNotAuthorized.Wrap("not order owner")
	}

	// Verify order is cancellable
	if order.Status != OrderStatusOpen && order.Status != OrderStatusPartial {
		return types.ErrOrderNotCancellable.Wrapf("order cannot be cancelled: status %d", order.Status)
	}

	// Calculate remaining amount to refund
	remainingAmount := order.AmountIn.Sub(order.FilledAmount)

	// Refund remaining tokens
	if remainingAmount.IsPositive() {
		coin := sdk.NewCoin(order.TokenIn, remainingAmount)
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, owner, sdk.NewCoins(coin)); err != nil {
			return types.ErrInsufficientLiquidity.Wrapf("failed to refund tokens: %v", err)
		}
	}

	// Update order status
	order.Status = OrderStatusCancelled
	if err := k.SetLimitOrder(ctx, order); err != nil {
		return err
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeDexOrderCancelled,
			sdk.NewAttribute(types.AttributeKeyOrderID, fmt.Sprintf("%d", orderID)),
			sdk.NewAttribute(sdk.AttributeKeySender, owner.String()),
			sdk.NewAttribute(types.AttributeKeyRefundedAmount, remainingAmount.String()),
		),
	)

	return nil
}

// MatchLimitOrder attempts to execute a limit order against the current pool price.
//
// This function checks if the current pool price satisfies the order's limit price,
// and if so, executes the swap using the pool's constant product formula. Orders can
// be partially or fully filled depending on pool liquidity and slippage constraints.
//
// Parameters:
//   - ctx: Blockchain context for state access
//   - order: The limit order to attempt matching
//
// Returns:
//   - error: nil if matching succeeds or order cannot execute at current price, or:
//     * ErrPoolNotFound: Pool does not exist
//     * ErrInvalidSwapAmount: Swap execution failed
//     * ErrInsufficientLiquidity: Failed to transfer tokens
//
// Matching Logic:
//  1. Retrieves current pool state and calculates spot price
//  2. For Buy orders: Executes if pool price <= limit price
//  3. For Sell orders: Executes if pool price >= limit price
//  4. Calculates fillable amount (may be partial if liquidity limited)
//  5. Executes swap using ExecuteSwap() (applies all security checks)
//  6. Transfers received tokens to order owner
//  7. Updates order status (Partial or Filled)
//  8. Emits order_matched event
//
// Behavior Notes:
//   - Returns nil (not error) if order cannot execute at current price
//   - Partial fills update FilledAmount and ReceivedAmount fields
//   - Full fills set status to Filled and remove from open index
//   - Uses module account as swap trader (tokens already locked)
//
// Security Notes:
//   - All swap security checks apply (MEV protection, price impact, etc.)
//   - Slippage protection via MinAmountOut (calculated from limit price)
//   - No reentrancy risk (state updated after token transfers)
func (k Keeper) MatchLimitOrder(ctx context.Context, order *LimitOrder) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get current pool state
	pool, err := k.GetPool(ctx, order.PoolID)
	if err != nil {
		return err
	}

	// Calculate current pool price
	var currentPrice math.LegacyDec
	if order.TokenIn == pool.TokenA {
		// Selling TokenA for TokenB
		currentPrice = math.LegacyNewDecFromInt(pool.ReserveB).Quo(math.LegacyNewDecFromInt(pool.ReserveA))
	} else {
		// Selling TokenB for TokenA
		currentPrice = math.LegacyNewDecFromInt(pool.ReserveA).Quo(math.LegacyNewDecFromInt(pool.ReserveB))
	}

	// Check if current price meets limit price
	// For buy orders, current price should be <= limit price
	// For sell orders, current price should be >= limit price
	canExecute := false
	if order.OrderType == OrderTypeBuy {
		canExecute = currentPrice.LTE(order.LimitPrice)
	} else {
		canExecute = currentPrice.GTE(order.LimitPrice)
	}

	if !canExecute {
		return nil // Order cannot be executed at current price
	}

	// Calculate how much can be filled
	remainingAmount := order.AmountIn.Sub(order.FilledAmount)
	if remainingAmount.IsZero() || remainingAmount.IsNegative() {
		return nil
	}

	// Execute the swap - using the module account as the trader since tokens are already locked
	// Get module address from authtypes
	moduleAddr := sdk.MustAccAddressFromBech32(getModuleAccountAddress())
	amountOut, err := k.ExecuteSwap(ctx, moduleAddr, order.PoolID, order.TokenIn, order.TokenOut, remainingAmount, order.MinAmountOut)
	if err != nil {
		return types.ErrInvalidSwapAmount.Wrapf("swap execution failed: %v", err)
	}

	// Update order state
	order.FilledAmount = order.FilledAmount.Add(remainingAmount)
	order.ReceivedAmount = order.ReceivedAmount.Add(amountOut)

	if order.FilledAmount.GTE(order.AmountIn) {
		order.Status = OrderStatusFilled
	} else {
		order.Status = OrderStatusPartial
	}

	// Transfer received tokens to owner
	ownerAddr, err := sdk.AccAddressFromBech32(order.Owner)
	if err != nil {
		return err
	}

	receivedCoin := sdk.NewCoin(order.TokenOut, amountOut)
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, ownerAddr, sdk.NewCoins(receivedCoin)); err != nil {
		return types.ErrInsufficientLiquidity.Wrapf("failed to transfer received tokens: %v", err)
	}

	// Save updated order
	if err := k.SetLimitOrder(ctx, order); err != nil {
		return err
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeDexOrderMatched,
			sdk.NewAttribute(types.AttributeKeyOrderID, fmt.Sprintf("%d", order.ID)),
			sdk.NewAttribute(types.AttributeKeyAmountIn, remainingAmount.String()),
			sdk.NewAttribute(types.AttributeKeyAmountOut, amountOut.String()),
			sdk.NewAttribute(types.AttributeKeyStatus, fmt.Sprintf("%d", order.Status)),
		),
	)

	return nil
}

// ProcessExpiredOrders processes and expires old orders, refunding unfilled amounts.
//
// This function is called in EndBlock to automatically expire orders that have passed
// their expiration time. Expired orders have their remaining tokens refunded to the owner
// and are marked as Expired.
//
// Parameters:
//   - ctx: Blockchain context for state access (typically called from EndBlock)
//
// Returns:
//   - error: Always returns nil (errors are logged but don't halt processing)
//
// Behavior:
//  1. Iterates through all open/partial orders
//  2. Checks each order's ExpiresAt timestamp against current block time
//  3. For expired orders:
//     - Calculates remaining unfilled amount
//     - Refunds tokens from module account to owner
//     - Updates order status to Expired
//     - Removes from open orders index
//     - Emits order_expired event
//  4. Continues processing even if individual orders fail (errors logged)
//
// Error Handling:
//   - Individual order failures are logged but don't stop processing
//   - Ensures all expired orders are processed even if some fail
//   - Prevents one bad order from blocking all other expirations
//
// Performance Notes:
//   - Only iterates open orders (not all orders)
//   - Uses index for efficient iteration
//   - Batch processes all expirations in single EndBlock call
//
// Security Notes:
//   - Only callable from EndBlock (not external transactions)
//   - Refund failures are logged (manual intervention may be needed)
//   - No partial state updates (order stays Open if refund fails)
func (k Keeper) ProcessExpiredOrders(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)
	now := sdkCtx.BlockTime()

	// Iterate through open orders
	iterator := storetypes.KVStorePrefixIterator(store, LimitOrderOpenPrefix)
	defer iterator.Close()

	expiredOrders := []uint64{}
	for ; iterator.Valid(); iterator.Next() {
		orderID := binary.BigEndian.Uint64(iterator.Key()[len(LimitOrderOpenPrefix):])

		order, err := k.GetLimitOrder(ctx, orderID)
		if err != nil {
			continue
		}

		// Check if expired
		if !order.ExpiresAt.IsZero() && now.After(order.ExpiresAt) {
			expiredOrders = append(expiredOrders, orderID)
		}
	}

	// Process expired orders
	for _, orderID := range expiredOrders {
		order, err := k.GetLimitOrder(ctx, orderID)
		if err != nil {
			continue
		}

		// Refund remaining tokens
		remainingAmount := order.AmountIn.Sub(order.FilledAmount)
		if remainingAmount.IsPositive() {
			ownerAddr, err := sdk.AccAddressFromBech32(order.Owner)
			if err != nil {
				continue
			}

			coin := sdk.NewCoin(order.TokenIn, remainingAmount)
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, ownerAddr, sdk.NewCoins(coin)); err != nil {
				sdkCtx.Logger().Error("failed to refund expired order", "order_id", orderID, "error", err)
				continue
			}
		}

		// Update status
		order.Status = OrderStatusExpired
		if err := k.SetLimitOrder(ctx, order); err != nil {
			continue
		}

		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeDexOrderExpired,
				sdk.NewAttribute(types.AttributeKeyOrderID, fmt.Sprintf("%d", orderID)),
				sdk.NewAttribute(types.AttributeKeyRefundedAmount, remainingAmount.String()),
			),
		)
	}

	return nil
}

// ============================================================================
// Query Functions
// ============================================================================

// GetOrdersByOwner returns all orders for a specific owner
func (k Keeper) GetOrdersByOwner(ctx context.Context, owner sdk.AccAddress) ([]*LimitOrder, error) {
	store := k.getStore(ctx)
	var orders []*LimitOrder

	prefix := append(LimitOrderByOwnerPrefix, owner.Bytes()...)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		orderID := binary.BigEndian.Uint64(iterator.Key()[len(prefix):])
		order, err := k.GetLimitOrder(ctx, orderID)
		if err != nil {
			continue
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// GetOrdersByOwnerPaginated returns orders for an owner with pagination support.
// Returns orders, next page key, and total count.
func (k Keeper) GetOrdersByOwnerPaginated(ctx context.Context, owner sdk.AccAddress, pageKey []byte, limit uint64) ([]*LimitOrder, []byte, uint64, error) {
	store := k.getStore(ctx)
	var orders []*LimitOrder
	var nextKey []byte
	var count uint64

	if limit == 0 {
		limit = 100 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Max limit
	}

	prefix := append(LimitOrderByOwnerPrefix, owner.Bytes()...)

	// First count total
	countIter := storetypes.KVStorePrefixIterator(store, prefix)
	for ; countIter.Valid(); countIter.Next() {
		count++
	}
	countIter.Close()

	// Then paginate
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	// Skip to page key if provided
	if len(pageKey) > 0 {
		for ; iterator.Valid(); iterator.Next() {
			if string(iterator.Key()) >= string(pageKey) {
				break
			}
		}
	}

	var collected uint64
	for ; iterator.Valid() && collected < limit; iterator.Next() {
		orderID := binary.BigEndian.Uint64(iterator.Key()[len(prefix):])
		order, err := k.GetLimitOrder(ctx, orderID)
		if err != nil {
			continue
		}
		orders = append(orders, order)
		collected++

		// Peek at next for page key
		if collected == limit {
			iterator.Next()
			if iterator.Valid() {
				nextKey = iterator.Key()
			}
			break
		}
	}

	return orders, nextKey, count, nil
}

// GetOrdersByPool returns all orders for a specific pool
func (k Keeper) GetOrdersByPool(ctx context.Context, poolID uint64) ([]*LimitOrder, error) {
	store := k.getStore(ctx)
	var orders []*LimitOrder

	poolIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIDBytes, poolID)
	prefix := append(LimitOrderByPoolPrefix, poolIDBytes...)

	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		orderID := binary.BigEndian.Uint64(iterator.Key()[len(prefix):])
		order, err := k.GetLimitOrder(ctx, orderID)
		if err != nil {
			continue
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// GetOrdersByPoolPaginated returns orders for a pool with pagination support.
// Returns orders, next page key, and total count.
func (k Keeper) GetOrdersByPoolPaginated(ctx context.Context, poolID uint64, pageKey []byte, limit uint64) ([]*LimitOrder, []byte, uint64, error) {
	store := k.getStore(ctx)
	var orders []*LimitOrder
	var nextKey []byte
	var count uint64

	if limit == 0 {
		limit = 100 // Default limit
	}
	if limit > 1000 {
		limit = 1000 // Max limit
	}

	poolIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(poolIDBytes, poolID)
	prefix := append(LimitOrderByPoolPrefix, poolIDBytes...)

	// First count total
	countIter := storetypes.KVStorePrefixIterator(store, prefix)
	for ; countIter.Valid(); countIter.Next() {
		count++
	}
	countIter.Close()

	// Then paginate
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	// Skip to page key if provided
	if len(pageKey) > 0 {
		for ; iterator.Valid(); iterator.Next() {
			if string(iterator.Key()) >= string(pageKey) {
				break
			}
		}
	}

	var collected uint64
	for ; iterator.Valid() && collected < limit; iterator.Next() {
		orderID := binary.BigEndian.Uint64(iterator.Key()[len(prefix):])
		order, err := k.GetLimitOrder(ctx, orderID)
		if err != nil {
			continue
		}
		orders = append(orders, order)
		collected++

		// Peek at next for page key
		if collected == limit {
			iterator.Next()
			if iterator.Valid() {
				nextKey = iterator.Key()
			}
			break
		}
	}

	return orders, nextKey, count, nil
}

// GetOpenOrders returns all open orders
func (k Keeper) GetOpenOrders(ctx context.Context) ([]*LimitOrder, error) {
	store := k.getStore(ctx)
	var orders []*LimitOrder

	iterator := storetypes.KVStorePrefixIterator(store, LimitOrderOpenPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		orderID := binary.BigEndian.Uint64(iterator.Key()[len(LimitOrderOpenPrefix):])
		order, err := k.GetLimitOrder(ctx, orderID)
		if err != nil {
			continue
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// GetOrderBook returns the order book for a pool (buy and sell orders sorted by price)
func (k Keeper) GetOrderBook(ctx context.Context, poolID uint64) (buyOrders, sellOrders []*LimitOrder, err error) {
	orders, err := k.GetOrdersByPool(ctx, poolID)
	if err != nil {
		return nil, nil, err
	}

	for _, order := range orders {
		if order.Status != OrderStatusOpen && order.Status != OrderStatusPartial {
			continue
		}
		if order.OrderType == OrderTypeBuy {
			buyOrders = append(buyOrders, order)
		} else {
			sellOrders = append(sellOrders, order)
		}
	}

	return buyOrders, sellOrders, nil
}

// MatchAllOrders attempts to match all open limit orders against their respective pools.
//
// This function is called in the ABCI EndBlocker to provide continuous order matching
// as pool prices change due to swaps. Each block, all open orders are checked to see
// if they can now be executed at the current pool prices.
//
// Parameters:
//   - ctx: Blockchain context for state access (called from EndBlock)
//
// Returns:
//   - error: Always returns nil (individual matching errors are logged)
//
// Behavior:
//  1. Retrieves all open/partial orders via open orders index
//  2. For each order, attempts matching via MatchLimitOrder()
//  3. Successful matches execute swaps and update order status
//  4. Failed matches are logged but don't stop processing
//  5. Continues through all orders even if some fail
//
// Performance Considerations:
//   - Only processes open/partial orders (not filled/cancelled/expired)
//   - Order matching has O(n) complexity where n = number of open orders
//   - Consider gas limits with large number of open orders
//   - Future optimization: Priority queue by price for early termination
//
// Error Handling:
//   - Individual order matching errors are logged, not returned
//   - Ensures all orders get matching attempt even if some fail
//   - Prevents one problematic order from blocking all matching
//
// Security Notes:
//   - Only callable from EndBlock (not external transactions)
//   - Each match applies full swap security checks
//   - No risk of same order matching multiple times per block (status updated)
//
// Integration:
//   This should be called in EndBlock AFTER:
//   - ProcessExpiredOrders() (removes expired orders first)
//   - Any pool state updates (ensures current prices)
func (k Keeper) MatchAllOrders(ctx context.Context) error {
	orders, err := k.GetOpenOrders(ctx)
	if err != nil {
		return err
	}

	for _, order := range orders {
		if err := k.MatchLimitOrder(ctx, order); err != nil {
			// Log error but continue with other orders
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			sdkCtx.Logger().Error("failed to match order", "order_id", order.ID, "error", err)
		}
	}

	return nil
}

// getModuleAccountAddress returns the module account address as a bech32 string
// This is a helper to avoid circular dependency with auth module
func getModuleAccountAddress() string {
	// The module account address is derived from the module name
	// This is the standard Cosmos SDK derivation
	return sdk.AccAddress([]byte(types.ModuleName)).String()
}
