package cli

import (
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/math"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
)

// GetLimitOrderTxCmd returns limit order transaction commands
func GetLimitOrderTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "limit-order",
		Short: "Limit order transaction subcommands",
		Long:  `Create, cancel, and manage limit orders on the DEX.`,
	}

	cmd.AddCommand(
		CmdPlaceLimitOrder(),
		CmdCancelLimitOrder(),
		CmdCancelAllLimitOrders(),
	)

	return cmd
}

// CmdPlaceLimitOrder creates a new limit order
func CmdPlaceLimitOrder() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "place [pool-id] [token-in] [amount-in] [token-out] [min-price]",
		Short: "Place a limit order",
		Long: `Place a limit order that will execute when the price reaches the specified minimum.

The order will remain in the order book until:
- It is filled completely
- It is cancelled by the owner
- It expires (if expiration is set)

Price is expressed as: min_price = min_amount_out / amount_in

Examples:
  # Buy USDT with PAW at minimum rate of 2.0 USDT per PAW
  $ pawd tx dex limit-order place 1 upaw 1000000 uusdt 2.0 --from mykey

  # Sell 1000 ATOM for PAW at minimum 500 PAW per ATOM
  $ pawd tx dex limit-order place 2 uatom 1000000000 upaw 500.0 --from mykey --expiration 86400`,
		Args: cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			poolID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid pool ID: %w", err)
			}

			tokenIn := args[1]
			tokenOut := args[3]

			if tokenIn == tokenOut {
				return fmt.Errorf("token-in and token-out must be different")
			}

			amountIn, ok := math.NewIntFromString(args[2])
			if !ok {
				return fmt.Errorf("invalid amount-in: %s", args[2])
			}

			if amountIn.IsZero() || amountIn.IsNegative() {
				return fmt.Errorf("amount-in must be positive")
			}

			// Parse min price (float)
			minPrice, err := strconv.ParseFloat(args[4], 64)
			if err != nil {
				return fmt.Errorf("invalid min-price: %w", err)
			}

			if minPrice <= 0 {
				return fmt.Errorf("min-price must be positive")
			}

			// Calculate min amount out from price
			// minAmountOut = amountIn * minPrice
			minAmountOutFloat := float64(amountIn.Int64()) * minPrice
			minAmountOut := math.NewInt(int64(minAmountOutFloat))

			// Get expiration from flag (optional)
			expirationSeconds, _ := cmd.Flags().GetInt64("expiration")
			var expiration int64
			if expirationSeconds > 0 {
				expiration = time.Now().Unix() + expirationSeconds
			}

			// Display order details
			cmd.Println("=== Limit Order ===")
			cmd.Printf("Pool: %d\n", poolID)
			cmd.Printf("Sell: %s %s\n", amountIn.String(), tokenIn)
			cmd.Printf("Buy: minimum %s %s\n", minAmountOut.String(), tokenOut)
			cmd.Printf("Limit Price: %.8f %s per %s\n", minPrice, tokenOut, tokenIn)
			if expiration > 0 {
				cmd.Printf("Expires: %s (in %d seconds)\n", time.Unix(expiration, 0).Format(time.RFC3339), expirationSeconds)
			} else {
				cmd.Println("Expires: Never (good-til-cancelled)")
			}

			// NOTE: MsgPlaceLimitOrder needs to be defined in proto files first
			// msg := &types.MsgPlaceLimitOrder{
			// 	Owner:        clientCtx.GetFromAddress().String(),
			// 	PoolId:       poolID,
			// 	TokenIn:      tokenIn,
			// 	TokenOut:     tokenOut,
			// 	AmountIn:     amountIn,
			// 	MinAmountOut: minAmountOut,
			// 	Expiration:   expiration,
			// }
			// if err := msg.ValidateBasic(); err != nil {
			// 	return err
			// }
			// return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)

			return fmt.Errorf("limit order transactions require MsgPlaceLimitOrder to be defined in proto files")
		},
	}

	cmd.Flags().Int64("expiration", 0, "Order expiration in seconds from now (0 = never expires)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdCancelLimitOrder cancels an existing limit order
func CmdCancelLimitOrder() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel [order-id]",
		Short: "Cancel a limit order",
		Long: `Cancel a limit order that you previously placed.

Only the order owner can cancel their orders.
Cancelled orders return the remaining funds to the owner.

Example:
  $ pawd tx dex limit-order cancel 123 --from mykey`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			_, err = strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid order ID: %w", err)
			}

			// NOTE: MsgCancelLimitOrder needs to be defined in proto files first
			// msg := &types.MsgCancelLimitOrder{
			// 	Owner:   clientCtx.GetFromAddress().String(),
			// 	OrderId: orderID,
			// }
			// if err := msg.ValidateBasic(); err != nil {
			// 	return err
			// }
			// return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)

			return fmt.Errorf("limit order cancellation requires MsgCancelLimitOrder to be defined in proto files")
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdCancelAllLimitOrders cancels all limit orders for the sender
func CmdCancelAllLimitOrders() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-all",
		Short: "Cancel all your limit orders",
		Long: `Cancel all limit orders placed by your address.

Optionally filter by pool ID to cancel only orders in a specific pool.

Examples:
  $ pawd tx dex limit-order cancel-all --from mykey
  $ pawd tx dex limit-order cancel-all --pool-id 1 --from mykey`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			poolID, _ := cmd.Flags().GetUint64("pool-id")

			// NOTE: MsgCancelAllLimitOrders needs to be defined in proto files first
			// msg := &types.MsgCancelAllLimitOrders{
			// 	Owner:  clientCtx.GetFromAddress().String(),
			// 	PoolId: poolID, // 0 means all pools
			// }
			// if err := msg.ValidateBasic(); err != nil {
			// 	return err
			// }
			// return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)

			_ = poolID // Mark as used
			return fmt.Errorf("cancel-all requires MsgCancelAllLimitOrders to be defined in proto files")
		},
	}

	cmd.Flags().Uint64("pool-id", 0, "Cancel only orders in this pool (0 = all pools)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
