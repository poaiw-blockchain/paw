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
		GetCmdQueryLimitOrder(),
		GetCmdQueryLimitOrders(),
		GetCmdQueryLimitOrdersByOwner(),
		GetCmdQueryLimitOrdersByPool(),
		GetCmdQueryOrderBook(),
		GetAdvancedQueryCmd(),
		GetStatsQueryCmd(),
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

// GetCmdQueryLimitOrder returns the command to query a limit order by ID
func GetCmdQueryLimitOrder() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "limit-order [order-id]",
		Short: "Query a limit order by ID",
		Long: `Query detailed information about a limit order by its ID.

Example:
  $ pawd query dex limit-order 1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			orderID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid order ID: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.LimitOrder(context.Background(), &types.QueryLimitOrderRequest{
				OrderId: orderID,
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

// GetCmdQueryLimitOrders returns the command to query all limit orders
func GetCmdQueryLimitOrders() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "limit-orders",
		Short: "Query all limit orders",
		Long: `Query all limit orders with pagination support.

Example:
  $ pawd query dex limit-orders
  $ pawd query dex limit-orders --limit 10 --offset 20`,
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
			res, err := queryClient.LimitOrders(context.Background(), &types.QueryLimitOrdersRequest{
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "limit-orders")
	return cmd
}

// GetCmdQueryLimitOrdersByOwner returns the command to query limit orders by owner
func GetCmdQueryLimitOrdersByOwner() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orders-by-owner [owner-address]",
		Short: "Query limit orders by owner address",
		Long: `Query all limit orders placed by a specific owner with pagination support.

Example:
  $ pawd query dex orders-by-owner paw1abcdef...
  $ pawd query dex orders-by-owner paw1abcdef... --limit 10`,
		Args: cobra.ExactArgs(1),
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
			res, err := queryClient.LimitOrdersByOwner(context.Background(), &types.QueryLimitOrdersByOwnerRequest{
				Owner:      args[0],
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "orders-by-owner")
	return cmd
}

// GetCmdQueryLimitOrdersByPool returns the command to query limit orders by pool
func GetCmdQueryLimitOrdersByPool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orders-by-pool [pool-id]",
		Short: "Query limit orders by pool ID",
		Long: `Query all limit orders for a specific liquidity pool with pagination support.

Example:
  $ pawd query dex orders-by-pool 1
  $ pawd query dex orders-by-pool 1 --limit 10`,
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

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.LimitOrdersByPool(context.Background(), &types.QueryLimitOrdersByPoolRequest{
				PoolId:     poolID,
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "orders-by-pool")
	return cmd
}

// GetCmdQueryOrderBook returns the command to query the order book
func GetCmdQueryOrderBook() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "order-book [pool-id]",
		Short: "Query the order book for a pool",
		Long: `Query the order book showing buy and sell orders for a specific liquidity pool.

Example:
  $ pawd query dex order-book 1
  $ pawd query dex order-book 1 --limit 20`,
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

			limit, _ := cmd.Flags().GetUint32("limit")
			if limit == 0 {
				limit = 50 // Default limit
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.OrderBook(context.Background(), &types.QueryOrderBookRequest{
				PoolId: poolID,
				Limit:  limit,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Uint32("limit", 50, "Maximum number of orders per side to return")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
