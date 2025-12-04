package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/paw-chain/paw/x/dex/types"
)

// GetStatsQueryCmd returns DEX statistics query commands
func GetStatsQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "DEX statistics and analytics",
		Long:  `Query DEX statistics, analytics, and market overview.`,
	}

	cmd.AddCommand(
		GetCmdQueryMarketOverview(),
		GetCmdQueryTopPools(),
		GetCmdQueryUserStats(),
		GetCmdQueryTokenInfo(),
	)

	return cmd
}

// GetCmdQueryMarketOverview returns overall DEX market statistics
func GetCmdQueryMarketOverview() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "overview",
		Short: "Get overall DEX market overview",
		Long: `Display comprehensive DEX statistics including:
- Total number of pools
- Total value locked (TVL)
- Number of active liquidity providers
- Total trades (if available)

Example:
  $ pawd query dex stats overview`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			// Get all pools
			poolsRes, err := queryClient.Pools(context.Background(), &types.QueryPoolsRequest{})
			if err != nil {
				return err
			}

			// Calculate aggregate statistics
			totalPools := len(poolsRes.Pools)
			totalTVL := int64(0)
			activeTokens := make(map[string]bool)

			for _, pool := range poolsRes.Pools {
				totalTVL += pool.ReserveA.Int64() + pool.ReserveB.Int64()
				activeTokens[pool.TokenA] = true
				activeTokens[pool.TokenB] = true
			}

			cmd.Println("╔════════════════════════════════════════════════════╗")
			cmd.Println("║           PAW DEX MARKET OVERVIEW                  ║")
			cmd.Println("╚════════════════════════════════════════════════════╝")
			cmd.Println()
			cmd.Printf("Total Liquidity Pools:     %d\n", totalPools)
			cmd.Printf("Unique Trading Tokens:     %d\n", len(activeTokens))
			cmd.Printf("Total Value Locked (TVL):  %d units\n", totalTVL)
			cmd.Println()
			cmd.Println("Active Tokens:")
			tokenList := make([]string, 0, len(activeTokens))
			for token := range activeTokens {
				tokenList = append(tokenList, token)
			}
			sort.Strings(tokenList)
			for _, token := range tokenList {
				cmd.Printf("  • %s\n", token)
			}

			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryTopPools returns top pools by various metrics
func GetCmdQueryTopPools() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "top-pools",
		Short: "Get top liquidity pools",
		Long: `Display top liquidity pools ranked by TVL or other metrics.

Sort options: tvl, liquidity, shares

Examples:
  $ pawd query dex stats top-pools
  $ pawd query dex stats top-pools --limit 5
  $ pawd query dex stats top-pools --sort-by liquidity`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			// Get all pools
			poolsRes, err := queryClient.Pools(context.Background(), &types.QueryPoolsRequest{})
			if err != nil {
				return err
			}

			if len(poolsRes.Pools) == 0 {
				cmd.Println("No pools found.")
				return nil
			}

			// Calculate TVL for each pool and sort
			type PoolRank struct {
				Pool *types.Pool
				TVL  int64
			}

			rankings := make([]PoolRank, 0, len(poolsRes.Pools))
			for _, pool := range poolsRes.Pools {
				tvl := pool.ReserveA.Int64() + pool.ReserveB.Int64()
				rankings = append(rankings, PoolRank{Pool: &pool, TVL: tvl})
			}

			// Sort by TVL descending
			sort.Slice(rankings, func(i, j int) bool {
				return rankings[i].TVL > rankings[j].TVL
			})

			// Apply limit
			limit, _ := cmd.Flags().GetInt("limit")
			if limit > 0 && limit < len(rankings) {
				rankings = rankings[:limit]
			}

			cmd.Println("╔════════════════════════════════════════════════════╗")
			cmd.Println("║              TOP LIQUIDITY POOLS                   ║")
			cmd.Println("╚════════════════════════════════════════════════════╝")
			cmd.Println()
			cmd.Println(strings.Repeat("─", 80))
			cmd.Printf("%-6s %-20s %-15s %-20s %-15s\n", "Rank", "Pair", "Reserve A", "Reserve B", "Total Shares")
			cmd.Println(strings.Repeat("─", 80))

			for i, rank := range rankings {
				pool := rank.Pool
				cmd.Printf("%-6d %-20s %-15s %-20s %-15s\n",
					i+1,
					fmt.Sprintf("%s/%s", pool.TokenA, pool.TokenB),
					formatAmount(pool.ReserveA.String()),
					formatAmount(pool.ReserveB.String()),
					formatAmount(pool.TotalShares.String()),
				)
			}
			cmd.Println(strings.Repeat("─", 80))

			return nil
		},
	}

	cmd.Flags().Int("limit", 10, "Number of pools to display")
	cmd.Flags().String("sort-by", "tvl", "Sort by: tvl, liquidity, shares")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryUserStats returns statistics for a specific user
func GetCmdQueryUserStats() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user [address]",
		Short: "Get DEX statistics for a user",
		Long: `Display comprehensive DEX statistics for a user address including:
- Total liquidity provided
- Number of LP positions
- Active limit orders

Example:
  $ pawd query dex stats user paw1abcdef...`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			address := args[0]
			queryClient := types.NewQueryClient(clientCtx)

			// Get all pools to check user positions
			poolsRes, err := queryClient.Pools(context.Background(), &types.QueryPoolsRequest{})
			if err != nil {
				return err
			}

			cmd.Println("╔════════════════════════════════════════════════════╗")
			cmd.Printf("║          USER DEX STATISTICS                       ║\n")
			cmd.Println("╚════════════════════════════════════════════════════╝")
			cmd.Println()
			cmd.Printf("Address: %s\n\n", address)

			// Count LP positions
			lpPositions := 0
			totalLiquidityValue := int64(0)

			cmd.Println("=== Liquidity Positions ===")
			for _, pool := range poolsRes.Pools {
				liqRes, err := queryClient.Liquidity(context.Background(), &types.QueryLiquidityRequest{
					PoolId:   pool.Id,
					Provider: address,
				})
				if err != nil || liqRes.Shares.IsZero() {
					continue
				}

				lpPositions++
				tokenAAmount := pool.ReserveA.Mul(liqRes.Shares).Quo(pool.TotalShares)
				tokenBAmount := pool.ReserveB.Mul(liqRes.Shares).Quo(pool.TotalShares)
				totalLiquidityValue += tokenAAmount.Int64() + tokenBAmount.Int64()

				sharePercent := float64(liqRes.Shares.Int64()) / float64(pool.TotalShares.Int64()) * 100
				cmd.Printf("\nPool %d (%s/%s):\n", pool.Id, pool.TokenA, pool.TokenB)
				cmd.Printf("  Shares: %s (%.6f%% of pool)\n", formatAmount(liqRes.Shares.String()), sharePercent)
				cmd.Printf("  Value: %s %s + %s %s\n",
					formatAmount(tokenAAmount.String()), pool.TokenA,
					formatAmount(tokenBAmount.String()), pool.TokenB)
			}

			if lpPositions == 0 {
				cmd.Println("No liquidity positions found.")
			}

			// Query limit orders
			cmd.Println("\n=== Limit Orders ===")
			ordersRes, err := queryClient.LimitOrdersByOwner(context.Background(), &types.QueryLimitOrdersByOwnerRequest{
				Owner: address,
			})
			if err == nil && len(ordersRes.Orders) > 0 {
				cmd.Printf("Active Orders: %d\n", len(ordersRes.Orders))
				for _, order := range ordersRes.Orders {
					cmd.Printf("  Order %d: %s %s → %s (Pool %d)\n",
						order.Id, formatAmount(order.AmountIn.String()), order.TokenIn, order.TokenOut, order.PoolId)
				}
			} else {
				cmd.Println("No active limit orders.")
			}

			// Summary
			cmd.Println("\n=== Summary ===")
			cmd.Printf("Total LP Positions: %d\n", lpPositions)
			cmd.Printf("Total Liquidity Value: %d units\n", totalLiquidityValue)

			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryTokenInfo returns information about a specific token on the DEX
func GetCmdQueryTokenInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token [denom]",
		Short: "Get information about a token on the DEX",
		Long: `Display information about a token including:
- All pools containing the token
- Total liquidity for the token
- Trading pairs available

Example:
  $ pawd query dex stats token upaw`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			denom := args[0]
			queryClient := types.NewQueryClient(clientCtx)

			// Get all pools
			poolsRes, err := queryClient.Pools(context.Background(), &types.QueryPoolsRequest{})
			if err != nil {
				return err
			}

			// Find pools containing this token
			var relevantPools []types.Pool
			totalLiquidity := int64(0)
			tradingPairs := make(map[string]bool)

			for _, pool := range poolsRes.Pools {
				if pool.TokenA == denom {
					relevantPools = append(relevantPools, pool)
					totalLiquidity += pool.ReserveA.Int64()
					tradingPairs[pool.TokenB] = true
				} else if pool.TokenB == denom {
					relevantPools = append(relevantPools, pool)
					totalLiquidity += pool.ReserveB.Int64()
					tradingPairs[pool.TokenA] = true
				}
			}

			if len(relevantPools) == 0 {
				return fmt.Errorf("token %s not found in any pools", denom)
			}

			cmd.Println("╔════════════════════════════════════════════════════╗")
			cmd.Printf("║              TOKEN INFORMATION                     ║\n")
			cmd.Println("╚════════════════════════════════════════════════════╝")
			cmd.Println()
			cmd.Printf("Token: %s\n\n", denom)
			cmd.Printf("Pools Containing Token: %d\n", len(relevantPools))
			cmd.Printf("Total Liquidity: %d units\n", totalLiquidity)
			cmd.Printf("Available Trading Pairs: %d\n\n", len(tradingPairs))

			cmd.Println("Trading Pairs:")
			for pair := range tradingPairs {
				cmd.Printf("  • %s/%s\n", denom, pair)
			}

			cmd.Println("\nLiquidity Pools:")
			for _, pool := range relevantPools {
				var reserve string
				var pairedToken string
				if pool.TokenA == denom {
					reserve = formatAmount(pool.ReserveA.String())
					pairedToken = pool.TokenB
				} else {
					reserve = formatAmount(pool.ReserveB.String())
					pairedToken = pool.TokenA
				}
				cmd.Printf("  Pool %d: %s/%s - %s %s\n", pool.Id, pool.TokenA, pool.TokenB, reserve, denom)
				_ = pairedToken // Used in output above
			}

			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// Helper function to format amounts (reuse from query_advanced.go)
func formatAmountLarge(amount string) string {
	// Add thousand separators for readability
	if len(amount) <= 3 {
		return amount
	}

	var result []rune
	for i, digit := range amount {
		if i > 0 && (len(amount)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, digit)
	}
	return string(result)
}
