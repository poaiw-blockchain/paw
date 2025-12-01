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

// OrderType represents the type of order (buy or sell)
type OrderType uint8

const (
	OrderTypeBuy  OrderType = 1
	OrderTypeSell OrderType = 2
)

// OrderStatus represents the status of a limit order
type OrderStatus uint8

const (
	OrderStatusOpen      OrderStatus = 1
	OrderStatusFilled    OrderStatus = 2
	OrderStatusPartial   OrderStatus = 3
	OrderStatusCancelled OrderStatus = 4
	OrderStatusExpired   OrderStatus = 5
)

// LimitOrder represents a limit order in the DEX
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

// Store key prefixes for limit orders
var (
	// LimitOrderKeyPrefix is the prefix for limit order storage
	LimitOrderKeyPrefix = []byte{0x0E}
	// LimitOrderCountKey is the key for the next order ID
	LimitOrderCountKey = []byte{0x0F}
	// LimitOrderByOwnerPrefix is for indexing orders by owner
	LimitOrderByOwnerPrefix = []byte{0x10}
	// LimitOrderByPoolPrefix is for indexing orders by pool
	LimitOrderByPoolPrefix = []byte{0x11}
	// LimitOrderByPricePrefix is for indexing orders by price (for matching)
	LimitOrderByPricePrefix = []byte{0x12}
	// LimitOrderOpenPrefix indexes only open orders
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
		return fmt.Errorf("failed to marshal limit order: %w", err)
	}

	// Store the order
	store.Set(LimitOrderKey(order.ID), bz)

	// Update indexes
	ownerAddr, err := sdk.AccAddressFromBech32(order.Owner)
	if err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
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
		return nil, fmt.Errorf("limit order not found: %d", orderID)
	}

	var order LimitOrder
	if err := json.Unmarshal(bz, &order); err != nil {
		return nil, fmt.Errorf("failed to unmarshal limit order: %w", err)
	}

	return &order, nil
}

// DeleteLimitOrder removes a limit order and its indexes
func (k Keeper) DeleteLimitOrder(ctx context.Context, order *LimitOrder) error {
	store := k.getStore(ctx)

	ownerAddr, err := sdk.AccAddressFromBech32(order.Owner)
	if err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
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

// PlaceLimitOrder creates a new limit order
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
		return nil, fmt.Errorf("pool not found: %w", err)
	}

	// Validate tokens match pool
	if !((pool.TokenA == tokenIn && pool.TokenB == tokenOut) ||
		(pool.TokenB == tokenIn && pool.TokenA == tokenOut)) {
		return nil, fmt.Errorf("tokens do not match pool")
	}

	// Validate amount
	if amountIn.IsNil() || !amountIn.IsPositive() {
		return nil, fmt.Errorf("invalid amount")
	}

	// Validate limit price
	if limitPrice.IsNil() || !limitPrice.IsPositive() {
		return nil, fmt.Errorf("invalid limit price")
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
		return nil, fmt.Errorf("failed to lock tokens: %w", err)
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
			"limit_order_placed",
			sdk.NewAttribute("order_id", fmt.Sprintf("%d", orderID)),
			sdk.NewAttribute("owner", owner.String()),
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute("order_type", fmt.Sprintf("%d", orderType)),
			sdk.NewAttribute("token_in", tokenIn),
			sdk.NewAttribute("token_out", tokenOut),
			sdk.NewAttribute("amount_in", amountIn.String()),
			sdk.NewAttribute("limit_price", limitPrice.String()),
		),
	)

	// Try to match immediately
	if err := k.MatchLimitOrder(ctx, order); err != nil {
		sdkCtx.Logger().Error("failed to match limit order", "order_id", orderID, "error", err)
	}

	return order, nil
}

// CancelLimitOrder cancels an existing limit order
func (k Keeper) CancelLimitOrder(ctx context.Context, owner sdk.AccAddress, orderID uint64) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	order, err := k.GetLimitOrder(ctx, orderID)
	if err != nil {
		return err
	}

	// Verify ownership
	if order.Owner != owner.String() {
		return fmt.Errorf("not order owner")
	}

	// Verify order is cancellable
	if order.Status != OrderStatusOpen && order.Status != OrderStatusPartial {
		return fmt.Errorf("order cannot be cancelled: status %d", order.Status)
	}

	// Calculate remaining amount to refund
	remainingAmount := order.AmountIn.Sub(order.FilledAmount)

	// Refund remaining tokens
	if remainingAmount.IsPositive() {
		coin := sdk.NewCoin(order.TokenIn, remainingAmount)
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, owner, sdk.NewCoins(coin)); err != nil {
			return fmt.Errorf("failed to refund tokens: %w", err)
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
			"limit_order_cancelled",
			sdk.NewAttribute("order_id", fmt.Sprintf("%d", orderID)),
			sdk.NewAttribute("owner", owner.String()),
			sdk.NewAttribute("refunded_amount", remainingAmount.String()),
		),
	)

	return nil
}

// MatchLimitOrder attempts to match a limit order against the pool
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
		return fmt.Errorf("swap execution failed: %w", err)
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
		return fmt.Errorf("failed to transfer received tokens: %w", err)
	}

	// Save updated order
	if err := k.SetLimitOrder(ctx, order); err != nil {
		return err
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"limit_order_matched",
			sdk.NewAttribute("order_id", fmt.Sprintf("%d", order.ID)),
			sdk.NewAttribute("filled_amount", remainingAmount.String()),
			sdk.NewAttribute("received_amount", amountOut.String()),
			sdk.NewAttribute("status", fmt.Sprintf("%d", order.Status)),
		),
	)

	return nil
}

// ProcessExpiredOrders processes and expires old orders
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
				"limit_order_expired",
				sdk.NewAttribute("order_id", fmt.Sprintf("%d", orderID)),
				sdk.NewAttribute("refunded_amount", remainingAmount.String()),
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

// MatchAllOrders attempts to match all open orders against their pools
// This is called in the ABCI EndBlocker
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
