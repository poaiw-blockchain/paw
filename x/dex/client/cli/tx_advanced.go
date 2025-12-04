package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/math"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// GetAdvancedTxCmd returns advanced transaction commands for the dex module
func GetAdvancedTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "advanced",
		Short: "Advanced DEX transaction commands",
		Long: `Advanced transaction commands for power users including batch operations,
interactive modes, and enhanced slippage protection.`,
	}

	cmd.AddCommand(
		CmdSwapWithSlippage(),
		CmdBatchSwap(),
		CmdQuickSwap(),
		CmdAddLiquidityBalanced(),
		CmdZapIn(),
	)

	return cmd
}

// CmdSwapWithSlippage returns a CLI command with automatic slippage calculation
func CmdSwapWithSlippage() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "swap-with-slippage [pool-id] [token-in] [amount-in] [token-out] [slippage-percent]",
		Short: "Execute a swap with automatic slippage protection",
		Long: `Execute a token swap with automatic slippage tolerance calculation.

The command will:
1. Simulate the swap to get expected output
2. Calculate minimum output based on slippage tolerance
3. Execute the swap with slippage protection

Slippage is specified as a percentage (e.g., 0.5 for 0.5%, 1.0 for 1%).

Examples:
  $ pawd tx dex advanced swap-with-slippage 1 upaw 1000000 uusdt 0.5 --from mykey
  $ pawd tx dex advanced swap-with-slippage 1 upaw 1000000 uusdt 1.0 --from mykey --deadline 300`,
		Args: cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
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

			slippagePercent, err := strconv.ParseFloat(args[4], 64)
			if err != nil {
				return fmt.Errorf("invalid slippage percentage: %w", err)
			}

			if slippagePercent < 0 || slippagePercent > 100 {
				return fmt.Errorf("slippage must be between 0 and 100 percent")
			}

			// Get deadline from flag or calculate default (5 minutes)
			deadlineSeconds, _ := cmd.Flags().GetInt64("deadline")
			if deadlineSeconds == 0 {
				deadlineSeconds = 300 // Default 5 minutes
			}
			deadline := time.Now().Unix() + deadlineSeconds

			// Simulate the swap to get expected output
			queryClient := types.NewQueryClient(clientCtx)
			poolRes, err := queryClient.Pool(cmd.Context(), &types.QueryPoolRequest{PoolId: poolID})
			if err != nil {
				return fmt.Errorf("failed to fetch pool %d: %w", poolID, err)
			}

			simRes, err := queryClient.SimulateSwap(cmd.Context(), &types.QuerySimulateSwapRequest{
				PoolId:   poolID,
				TokenIn:  tokenIn,
				TokenOut: tokenOut,
				AmountIn: amountIn,
			})
			if err != nil {
				return fmt.Errorf("failed to simulate swap: %w", err)
			}

			expectedOutput := simRes.AmountOut

			// Calculate minimum output with slippage tolerance
			minAmountOut, err := computeMinAmountOut(expectedOutput, slippagePercent)
			if err != nil {
				return err
			}

			// Display swap details
			cmd.Println("=== Swap Details ===")
			cmd.Printf("Pool ID: %d\n", poolID)
			cmd.Printf("Input: %s %s\n", amountIn.String(), tokenIn)
			cmd.Printf("Expected Output: %s %s\n", expectedOutput.String(), tokenOut)
			cmd.Printf("Slippage Tolerance: %.2f%%\n", slippagePercent)
			cmd.Printf("Minimum Output: %s %s\n", minAmountOut.String(), tokenOut)
			cmd.Printf("Deadline: %d seconds from now\n", deadlineSeconds)

			// Calculate price impact
			priceImpact := calculatePriceImpact(simRes, poolRes.Pool, tokenIn, amountIn)
			cmd.Printf("Price Impact: %.4f%%\n", priceImpact)

			if priceImpact > 5.0 {
				cmd.Println("⚠️  WARNING: High price impact detected!")
			}

			// Build and broadcast transaction
			msg := &types.MsgSwap{
				Trader:       clientCtx.GetFromAddress().String(),
				PoolId:       poolID,
				TokenIn:      tokenIn,
				TokenOut:     tokenOut,
				AmountIn:     amountIn,
				MinAmountOut: minAmountOut,
				Deadline:     deadline,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().Int64("deadline", 300, "Transaction deadline in seconds from now")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdBatchSwap returns a CLI command for batch swap operations
func CmdBatchSwap() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch-swap [swaps-json]",
		Short: "Execute multiple swaps in a single transaction",
		Long: `Execute multiple token swaps in a single transaction for gas efficiency.

The swaps-json parameter should be a JSON array of swap specifications.

Example JSON format:
[
  {"pool_id": 1, "token_in": "upaw", "amount_in": "1000000", "token_out": "uusdt", "slippage": 0.5},
  {"pool_id": 2, "token_in": "uusdt", "amount_in": "500000", "token_out": "uatom", "slippage": 1.0}
]

Example:
  $ pawd tx dex advanced batch-swap '[{"pool_id":1,"token_in":"upaw","amount_in":"1000000","token_out":"uusdt","slippage":0.5}]' --from mykey`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Parse batch swaps
			swaps, err := parseBatchSwaps(args[0])
			if err != nil {
				return fmt.Errorf("failed to parse swaps: %w", err)
			}

			if len(swaps) == 0 {
				return fmt.Errorf("no swaps specified")
			}

			cmd.Printf("Executing %d swaps in batch...\n", len(swaps))

			// Get deadline
			deadlineSeconds, _ := cmd.Flags().GetInt64("deadline")
			if deadlineSeconds == 0 {
				deadlineSeconds = 300
			}
			deadline := time.Now().Unix() + deadlineSeconds

			// Simulate and prepare all swaps
			queryClient := types.NewQueryClient(clientCtx)
			var msgs []types.MsgSwap

			for i, swap := range swaps {
				cmd.Printf("\n[Swap %d/%d]\n", i+1, len(swaps))

				// Simulate
				simRes, err := queryClient.SimulateSwap(cmd.Context(), &types.QuerySimulateSwapRequest{
					PoolId:   swap.PoolID,
					TokenIn:  swap.TokenIn,
					TokenOut: swap.TokenOut,
					AmountIn: swap.AmountIn,
				})
				if err != nil {
					return fmt.Errorf("swap %d simulation failed: %w", i+1, err)
				}

				// Calculate min output
				minAmountOut, err := computeMinAmountOut(simRes.AmountOut, swap.Slippage)
				if err != nil {
					return fmt.Errorf("swap %d: %w", i+1, err)
				}

				cmd.Printf("Pool: %d, In: %s %s, Expected Out: %s %s\n",
					swap.PoolID, swap.AmountIn.String(), swap.TokenIn,
					simRes.AmountOut.String(), swap.TokenOut)

				msg := types.MsgSwap{
					Trader:       clientCtx.GetFromAddress().String(),
					PoolId:       swap.PoolID,
					TokenIn:      swap.TokenIn,
					TokenOut:     swap.TokenOut,
					AmountIn:     swap.AmountIn,
					MinAmountOut: minAmountOut,
					Deadline:     deadline,
				}

				msgs = append(msgs, msg)
			}

			// Convert to sdk.Msg slice
			sdkMsgs := make([]sdk.Msg, len(msgs))
			for i := range msgs {
				sdkMsgs[i] = &msgs[i]
			}

			// Broadcast batch transaction
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), sdkMsgs...)
		},
	}

	cmd.Flags().Int64("deadline", 300, "Transaction deadline in seconds from now")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdQuickSwap returns a simplified swap command with smart defaults
func CmdQuickSwap() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quick-swap [token-in] [amount-in] [token-out]",
		Short: "Quick swap with automatic pool selection and default slippage",
		Long: `Execute a quick swap with smart defaults:
- Automatically finds the best pool for the token pair
- Uses 1% default slippage tolerance
- 5 minute default deadline

Perfect for fast trades when you trust the defaults.

Examples:
  $ pawd tx dex advanced quick-swap upaw 1000000 uusdt --from mykey
  $ pawd tx dex advanced quick-swap uatom 500000 upaw --from mykey`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			tokenIn := args[0]
			tokenOut := args[2]

			if tokenIn == tokenOut {
				return fmt.Errorf("token-in and token-out must be different")
			}

			amountIn, ok := math.NewIntFromString(args[1])
			if !ok {
				return fmt.Errorf("invalid amount-in: %s", args[1])
			}

			// Find pool by tokens
			queryClient := types.NewQueryClient(clientCtx)
			poolRes, err := queryClient.PoolByTokens(cmd.Context(), &types.QueryPoolByTokensRequest{
				TokenA: tokenIn,
				TokenB: tokenOut,
			})
			if err != nil {
				return fmt.Errorf("no pool found for %s/%s: %w", tokenIn, tokenOut, err)
			}

			poolID := poolRes.Pool.Id

			// Use 1% default slippage
			slippagePercent := 1.0
			deadline := time.Now().Unix() + 300 // 5 minutes

			// Simulate swap
			simRes, err := queryClient.SimulateSwap(cmd.Context(), &types.QuerySimulateSwapRequest{
				PoolId:   poolID,
				TokenIn:  tokenIn,
				TokenOut: tokenOut,
				AmountIn: amountIn,
			})
			if err != nil {
				return fmt.Errorf("swap simulation failed: %w", err)
			}

			// Calculate min output
			minAmountOut, err := computeMinAmountOut(simRes.AmountOut, slippagePercent)
			if err != nil {
				return err
			}

			cmd.Println("=== Quick Swap ===")
			cmd.Printf("Pool: %d\n", poolID)
			cmd.Printf("Input: %s %s\n", amountIn.String(), tokenIn)
			cmd.Printf("Expected Output: ~%s %s\n", simRes.AmountOut.String(), tokenOut)
			cmd.Printf("Slippage: %.1f%% (default)\n", slippagePercent)

			msg := &types.MsgSwap{
				Trader:       clientCtx.GetFromAddress().String(),
				PoolId:       poolID,
				TokenIn:      tokenIn,
				TokenOut:     tokenOut,
				AmountIn:     amountIn,
				MinAmountOut: minAmountOut,
				Deadline:     deadline,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdAddLiquidityBalanced returns a command for adding balanced liquidity
func CmdAddLiquidityBalanced() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-liquidity-balanced [pool-id] [total-value-token-a]",
		Short: "Add liquidity with automatic ratio balancing",
		Long: `Add liquidity to a pool with automatic ratio calculation.

Specify the total value in token A, and the command will calculate the correct
amount of token B to maintain the pool ratio.

Example:
  $ pawd tx dex advanced add-liquidity-balanced 1 1000000 --from mykey`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			poolID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid pool ID: %w", err)
			}

			amountA, ok := math.NewIntFromString(args[1])
			if !ok {
				return fmt.Errorf("invalid amount: %s", args[1])
			}

			// Get pool info
			queryClient := types.NewQueryClient(clientCtx)
			poolRes, err := queryClient.Pool(cmd.Context(), &types.QueryPoolRequest{
				PoolId: poolID,
			})
			if err != nil {
				return err
			}

			pool := poolRes.Pool

			// Calculate proportional amount B
			// amountB = (amountA * reserveB) / reserveA
			amountB := amountA.Mul(pool.ReserveB).Quo(pool.ReserveA)

			cmd.Println("=== Balanced Liquidity Addition ===")
			cmd.Printf("Pool: %d (%s/%s)\n", poolID, pool.TokenA, pool.TokenB)
			cmd.Printf("Amount %s: %s\n", pool.TokenA, amountA.String())
			cmd.Printf("Amount %s: %s (calculated)\n", pool.TokenB, amountB.String())
			cmd.Printf("Current Ratio: 1 %s = %.6f %s\n",
				pool.TokenA,
				float64(pool.ReserveB.Int64())/float64(pool.ReserveA.Int64()),
				pool.TokenB)

			msg := &types.MsgAddLiquidity{
				Provider: clientCtx.GetFromAddress().String(),
				PoolId:   poolID,
				AmountA:  amountA,
				AmountB:  amountB,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdZapIn returns a command for single-sided liquidity provision
func CmdZapIn() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zap-in [pool-id] [token] [amount]",
		Short: "Add liquidity with a single token (zap in)",
		Long: `Add liquidity to a pool using only one token.

The command will:
1. Swap half of the input token for the other token in the pool
2. Add liquidity with both tokens in the correct ratio

This is a convenience feature for users who only have one token.

Example:
	$ pawd tx dex advanced zap-in 1 upaw 2000000 --from mykey`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			poolID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid pool ID: %w", err)
			}

			tokenIn := args[1]
			totalAmount, ok := math.NewIntFromString(args[2])
			if !ok {
				return fmt.Errorf("invalid amount: %s", args[2])
			}
			if totalAmount.LTE(math.NewInt(1)) {
				return fmt.Errorf("amount must be greater than 1 to zap in")
			}

			swapAmount := totalAmount.QuoRaw(2)
			if swapAmount.IsZero() {
				return fmt.Errorf("swap amount resolved to zero")
			}
			remainingTokenIn := totalAmount.Sub(swapAmount)

			slippagePercent, err := cmd.Flags().GetFloat64("slippage")
			if err != nil {
				return fmt.Errorf("invalid slippage flag: %w", err)
			}
			if slippagePercent < 0 || slippagePercent > 100 {
				return fmt.Errorf("slippage must be between 0 and 100 percent")
			}

			deadlineSeconds, _ := cmd.Flags().GetInt64("deadline")
			if deadlineSeconds == 0 {
				deadlineSeconds = 300
			}
			deadline := time.Now().Unix() + deadlineSeconds

			queryClient := types.NewQueryClient(clientCtx)
			poolRes, err := queryClient.Pool(cmd.Context(), &types.QueryPoolRequest{PoolId: poolID})
			if err != nil {
				return fmt.Errorf("failed to fetch pool %d: %w", poolID, err)
			}
			pool := poolRes.Pool

			var tokenOut string
			switch tokenIn {
			case pool.TokenA:
				tokenOut = pool.TokenB
			case pool.TokenB:
				tokenOut = pool.TokenA
			default:
				return fmt.Errorf("token %s is not present in pool %d", tokenIn, poolID)
			}

			simRes, err := queryClient.SimulateSwap(cmd.Context(), &types.QuerySimulateSwapRequest{
				PoolId:   poolID,
				TokenIn:  tokenIn,
				TokenOut: tokenOut,
				AmountIn: swapAmount,
			})
			if err != nil {
				return fmt.Errorf("swap simulation failed: %w", err)
			}

			minAmountOut, err := computeMinAmountOut(simRes.AmountOut, slippagePercent)
			if err != nil {
				return err
			}
			if minAmountOut.IsZero() {
				return fmt.Errorf("slippage or input size results in zero minimum output")
			}

			cmd.Println("=== Zap In Plan ===")
			cmd.Printf("Pool: %d (%s/%s)\n", poolID, pool.TokenA, pool.TokenB)
			cmd.Printf("Total Input: %s %s\n", totalAmount.String(), tokenIn)
			cmd.Printf("Swap Phase: %s %s → ~%s %s\n", swapAmount.String(), tokenIn, simRes.AmountOut.String(), tokenOut)
			cmd.Printf("Add Liquidity Phase: %s %s + %s %s (minimum)\n",
				remainingTokenIn.String(), tokenIn, minAmountOut.String(), tokenOut)

			swapMsg := &types.MsgSwap{
				Trader:       clientCtx.GetFromAddress().String(),
				PoolId:       poolID,
				TokenIn:      tokenIn,
				TokenOut:     tokenOut,
				AmountIn:     swapAmount,
				MinAmountOut: minAmountOut,
				Deadline:     deadline,
			}
			if err := swapMsg.ValidateBasic(); err != nil {
				return err
			}

			var amountA, amountB math.Int
			if tokenIn == pool.TokenA {
				amountA = remainingTokenIn
				amountB = minAmountOut
			} else {
				amountA = minAmountOut
				amountB = remainingTokenIn
			}

			addMsg := &types.MsgAddLiquidity{
				Provider: clientCtx.GetFromAddress().String(),
				PoolId:   poolID,
				AmountA:  amountA,
				AmountB:  amountB,
			}
			if err := addMsg.ValidateBasic(); err != nil {
				return err
			}

			cmd.Println("Broadcasting zap-in transaction (swap + add-liquidity)...")
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), swapMsg, addMsg)
		},
	}

	cmd.Flags().Float64("slippage", 1.0, "Slippage tolerance percent for the swap leg")
	cmd.Flags().Int64("deadline", 300, "Transaction deadline in seconds from now")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// Helper functions

type batchSwap struct {
	PoolID   uint64
	TokenIn  string
	AmountIn math.Int
	TokenOut string
	Slippage float64
}

func parseBatchSwaps(jsonStr string) ([]batchSwap, error) {
	type payload struct {
		PoolID   uint64  `json:"pool_id"`
		TokenIn  string  `json:"token_in"`
		AmountIn string  `json:"amount_in"`
		TokenOut string  `json:"token_out"`
		Slippage float64 `json:"slippage"`
	}

	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	decoder.DisallowUnknownFields()

	var raw []payload
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode batch swaps: %w", err)
	}

	if len(raw) == 0 {
		return nil, fmt.Errorf("no swaps provided")
	}

	swaps := make([]batchSwap, 0, len(raw))
	for idx, entry := range raw {
		if entry.PoolID == 0 {
			return nil, fmt.Errorf("swap %d: pool_id must be positive", idx+1)
		}
		if entry.TokenIn == "" || entry.TokenOut == "" {
			return nil, fmt.Errorf("swap %d: token_in and token_out are required", idx+1)
		}
		if entry.TokenIn == entry.TokenOut {
			return nil, fmt.Errorf("swap %d: token_in and token_out must differ", idx+1)
		}

		amountIn, ok := math.NewIntFromString(entry.AmountIn)
		if !ok || !amountIn.IsPositive() {
			return nil, fmt.Errorf("swap %d: invalid amount_in %q", idx+1, entry.AmountIn)
		}

		slippage := entry.Slippage
		if slippage == 0 {
			slippage = 1.0
		}
		if slippage < 0 || slippage > 100 {
			return nil, fmt.Errorf("swap %d: slippage must be between 0 and 100 percent", idx+1)
		}

		swaps = append(swaps, batchSwap{
			PoolID:   entry.PoolID,
			TokenIn:  entry.TokenIn,
			AmountIn: amountIn,
			TokenOut: entry.TokenOut,
			Slippage: slippage,
		})
	}

	return swaps, nil
}

func calculatePriceImpact(simRes *types.QuerySimulateSwapResponse, pool types.Pool, tokenIn string, amountIn math.Int) float64 {
	if simRes == nil || amountIn.IsZero() {
		return 0
	}

	var reserveIn, reserveOut math.LegacyDec
	switch tokenIn {
	case pool.TokenA:
		reserveIn = math.LegacyNewDecFromInt(pool.ReserveA)
		reserveOut = math.LegacyNewDecFromInt(pool.ReserveB)
	case pool.TokenB:
		reserveIn = math.LegacyNewDecFromInt(pool.ReserveB)
		reserveOut = math.LegacyNewDecFromInt(pool.ReserveA)
	default:
		return 0
	}

	if reserveIn.IsZero() || reserveOut.IsZero() {
		return 0
	}

	spotPrice := reserveOut.Quo(reserveIn)
	if spotPrice.IsZero() {
		return 0
	}

	effectivePrice := math.LegacyNewDecFromInt(simRes.AmountOut).Quo(math.LegacyNewDecFromInt(amountIn))
	if effectivePrice.IsZero() {
		return 0
	}

	impact := spotPrice.Sub(effectivePrice).Quo(spotPrice)
	if impact.IsNegative() {
		impact = impact.Neg()
	}

	val, err := impact.Float64()
	if err != nil || val < 0 {
		return 0
	}

	return val * 100
}

func computeMinAmountOut(expected math.Int, slippagePercent float64) (math.Int, error) {
	if slippagePercent < 0 || slippagePercent > 100 {
		return math.Int{}, fmt.Errorf("slippage must be between 0 and 100 percent")
	}

	if !expected.IsPositive() {
		return math.ZeroInt(), nil
	}

	ratioStr := strconv.FormatFloat(slippagePercent/100, 'f', -1, 64)
	slippageRatio, err := math.LegacyNewDecFromStr(ratioStr)
	if err != nil {
		return math.Int{}, fmt.Errorf("invalid slippage percent: %w", err)
	}

	if slippageRatio.GTE(math.LegacyOneDec()) {
		return math.ZeroInt(), nil
	}

	factor := math.LegacyOneDec().Sub(slippageRatio)
	minDec := math.LegacyNewDecFromInt(expected).Mul(factor)
	return minDec.TruncateInt(), nil
}
