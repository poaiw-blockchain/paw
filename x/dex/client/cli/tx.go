package cli

import (
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/paw-chain/paw/x/dex/types"
)

// GetTxCmd returns the transaction commands for the dex module
func GetTxCmd() *cobra.Command {
	dexTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "DEX transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	dexTxCmd.AddCommand(
		CmdCreatePool(),
		CmdAddLiquidity(),
		CmdRemoveLiquidity(),
		CmdSwap(),
	)

	return dexTxCmd
}

// CmdCreatePool returns a CLI command handler for creating a liquidity pool
func CmdCreatePool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-pool [token-a] [amount-a] [token-b] [amount-b]",
		Short: "Create a new liquidity pool",
		Long: `Create a new liquidity pool with an initial deposit of both tokens.

Example:
  $ pawd tx dex create-pool upaw 1000000 uusdt 2000000 --from mykey
  $ pawd tx dex create-pool upaw 500000000 uatom 1000000 --from mykey`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			tokenA := args[0]
			tokenB := args[2]

			if tokenA == tokenB {
				return fmt.Errorf("tokens must be different")
			}

			amountA, ok := math.NewIntFromString(args[1])
			if !ok {
				return fmt.Errorf("invalid amount-a: %s (must be integer)", args[1])
			}

			amountB, ok := math.NewIntFromString(args[3])
			if !ok {
				return fmt.Errorf("invalid amount-b: %s (must be integer)", args[3])
			}

			if amountA.IsZero() || amountA.IsNegative() {
				return fmt.Errorf("amount-a must be positive")
			}

			if amountB.IsZero() || amountB.IsNegative() {
				return fmt.Errorf("amount-b must be positive")
			}

			msg := &types.MsgCreatePool{
				Creator: clientCtx.GetFromAddress().String(),
				TokenA:  tokenA,
				TokenB:  tokenB,
				AmountA: amountA,
				AmountB: amountB,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdAddLiquidity returns a CLI command handler for adding liquidity to a pool
func CmdAddLiquidity() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-liquidity [pool-id] [amount-a] [amount-b]",
		Short: "Add liquidity to an existing pool",
		Long: `Add liquidity to an existing pool by depositing both tokens proportionally.

The amounts should be proportional to the current pool ratio to avoid significant slippage.

Example:
  $ pawd tx dex add-liquidity 1 1000000 2000000 --from mykey`,
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

			amountA, ok := math.NewIntFromString(args[1])
			if !ok {
				return fmt.Errorf("invalid amount-a: %s (must be integer)", args[1])
			}

			amountB, ok := math.NewIntFromString(args[2])
			if !ok {
				return fmt.Errorf("invalid amount-b: %s (must be integer)", args[2])
			}

			if amountA.IsZero() || amountA.IsNegative() {
				return fmt.Errorf("amount-a must be positive")
			}

			if amountB.IsZero() || amountB.IsNegative() {
				return fmt.Errorf("amount-b must be positive")
			}

			msg := &types.MsgAddLiquidity{
				Provider: clientCtx.GetFromAddress().String(),
				PoolId:   poolID,
				AmountA:  amountA,
				AmountB:  amountB,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdRemoveLiquidity returns a CLI command handler for removing liquidity from a pool
func CmdRemoveLiquidity() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-liquidity [pool-id] [shares]",
		Short: "Remove liquidity from a pool",
		Long: `Remove liquidity from a pool by burning liquidity shares.

You will receive both tokens proportional to your share of the pool.

Example:
  $ pawd tx dex remove-liquidity 1 1000000 --from mykey`,
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

			shares, ok := math.NewIntFromString(args[1])
			if !ok {
				return fmt.Errorf("invalid shares: %s (must be integer)", args[1])
			}

			if shares.IsZero() || shares.IsNegative() {
				return fmt.Errorf("shares must be positive")
			}

			msg := &types.MsgRemoveLiquidity{
				Provider: clientCtx.GetFromAddress().String(),
				PoolId:   poolID,
				Shares:   shares,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdSwap returns a CLI command handler for swapping tokens
func CmdSwap() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "swap [pool-id] [token-in] [amount-in] [token-out] [min-amount-out]",
		Short: "Execute a token swap",
		Long: `Execute a token swap on a liquidity pool using the AMM algorithm.

The min-amount-out parameter protects against slippage. The transaction will fail
if the output amount is less than the specified minimum.

Use the simulate-swap query to estimate the output before swapping.

Examples:
  $ pawd tx dex swap 1 upaw 1000000 uusdt 1900000 --from mykey
  $ pawd tx dex swap 1 uusdt 2000000 upaw 950000 --from mykey

With slippage calculation (5%):
  $ pawd query dex simulate-swap 1 upaw uusdt 1000000
  # Assuming output is 2000000, calculate 5% slippage:
  # min_out = 2000000 * 0.95 = 1900000
  $ pawd tx dex swap 1 upaw 1000000 uusdt 1900000 --from mykey`,
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
				return fmt.Errorf("invalid amount-in: %s (must be integer)", args[2])
			}

			minAmountOut, ok := math.NewIntFromString(args[4])
			if !ok {
				return fmt.Errorf("invalid min-amount-out: %s (must be integer)", args[4])
			}

			if amountIn.IsZero() || amountIn.IsNegative() {
				return fmt.Errorf("amount-in must be positive")
			}

			if minAmountOut.IsNegative() {
				return fmt.Errorf("min-amount-out cannot be negative")
			}

			msg := &types.MsgSwap{
				Trader:       clientCtx.GetFromAddress().String(),
				PoolId:       poolID,
				TokenIn:      tokenIn,
				TokenOut:     tokenOut,
				AmountIn:     amountIn,
				MinAmountOut: minAmountOut,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
