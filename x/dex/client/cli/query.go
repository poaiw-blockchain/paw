package cli

import (
	"context"
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/paw-chain/paw/x/dex/types"
)

// GetQueryCmd returns the cli query commands for the dex module
func GetQueryCmd() *cobra.Command {
	dexQueryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the dex module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	dexQueryCmd.AddCommand(
		GetCmdQueryParams(),
		GetCmdQueryPool(),
		GetCmdQueryPools(),
		GetCmdQueryPoolByTokens(),
		GetCmdQueryLiquidity(),
		GetCmdQuerySimulateSwap(),
	)

	return dexQueryCmd
}

// GetCmdQueryParams returns the command to query module parameters
func GetCmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the current dex module parameters",
		Long: `Query the current parameters of the dex module including fees and limits.

Example:
  $ pawd query dex params`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Params(context.Background(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryPool returns the command to query a pool by ID
func GetCmdQueryPool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pool [pool-id]",
		Short: "Query liquidity pool by ID",
		Long: `Query detailed information about a liquidity pool by its ID.

Example:
  $ pawd query dex pool 1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			poolID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid pool ID: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Pool(context.Background(), &types.QueryPoolRequest{
				PoolId: poolID,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryPools returns the command to query all pools
func GetCmdQueryPools() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pools",
		Short: "Query all liquidity pools",
		Long: `Query all liquidity pools with pagination support.

Example:
  $ pawd query dex pools
  $ pawd query dex pools --limit 10 --offset 20`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Pools(context.Background(), &types.QueryPoolsRequest{
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "pools")
	return cmd
}

// GetCmdQueryPoolByTokens returns the command to query a pool by token pair
func GetCmdQueryPoolByTokens() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pool-by-tokens [token-a] [token-b]",
		Short: "Query liquidity pool by token pair",
		Long: `Query a liquidity pool by its token pair. Order doesn't matter.

Example:
  $ pawd query dex pool-by-tokens upaw uusdt`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.PoolByTokens(context.Background(), &types.QueryPoolByTokensRequest{
				TokenA: args[0],
				TokenB: args[1],
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryLiquidity returns the command to query a user's liquidity
func GetCmdQueryLiquidity() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "liquidity [pool-id] [provider]",
		Short: "Query a user's liquidity in a pool",
		Long: `Query the amount of liquidity shares a user has in a specific pool.

Example:
  $ pawd query dex liquidity 1 paw1abcdef...`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			poolID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid pool ID: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Liquidity(context.Background(), &types.QueryLiquidityRequest{
				PoolId:   poolID,
				Provider: args[1],
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQuerySimulateSwap returns the command to simulate a swap
func GetCmdQuerySimulateSwap() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "simulate-swap [pool-id] [token-in] [token-out] [amount-in]",
		Short: "Simulate a token swap without executing it",
		Long: `Simulate a token swap to estimate the output amount including fees.

Example:
  $ pawd query dex simulate-swap 1 upaw uusdt 1000000`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			poolID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid pool ID: %w", err)
			}

			amountIn, ok := math.NewIntFromString(args[3])
			if !ok {
				return fmt.Errorf("invalid amount-in: %s (must be integer)", args[3])
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.SimulateSwap(context.Background(), &types.QuerySimulateSwapRequest{
				PoolId:   poolID,
				TokenIn:  args[1],
				TokenOut: args[2],
				AmountIn: amountIn,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
