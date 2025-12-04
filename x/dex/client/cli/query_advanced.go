package cli

import (
	"context"
	"fmt"
	stdmath "math"
	"sort"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/math"
	"github.com/spf13/cobra"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/paw-chain/paw/x/dex/types"
)

// GetAdvancedQueryCmd returns advanced query commands for the dex module
func GetAdvancedQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "advanced",
		Short: "Advanced DEX query commands",
		Long:  `Advanced query commands for detailed DEX analytics and trading information.`,
	}

	cmd.AddCommand(
		GetCmdQueryPoolStats(),
		GetCmdQueryAllPoolStats(),
		GetCmdQueryPrice(),
		GetCmdQueryPortfolio(),
		GetCmdQueryArbitrage(),
		GetCmdQueryRoute(),
		GetCmdQueryVolume(),
		GetCmdQueryLPPosition(),
		GetCmdQueryPriceHistory(),
		GetCmdQueryFeeAPY(),
	)

	return cmd
}

// GetCmdQueryPoolStats returns comprehensive pool statistics
func GetCmdQueryPoolStats() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pool-stats [pool-id]",
		Short: "Get comprehensive statistics for a liquidity pool",
		Long: `Query detailed statistics for a liquidity pool including:
- Total value locked (TVL)
- 24h volume
- Current price
- Fee APY
- Liquidity depth
- Number of LPs

Example:
  $ pawd query dex advanced pool-stats 1
  $ pawd query dex advanced pool-stats 1 --output json`,
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

			// Get pool info
			poolRes, err := queryClient.Pool(context.Background(), &types.QueryPoolRequest{
				PoolId: poolID,
			})
			if err != nil {
				return err
			}

			pool := poolRes.Pool

			// Calculate pool statistics
			stats := calculatePoolStats(&pool)

			// Format output based on output flag
			outputFormat, _ := cmd.Flags().GetString("output")
			if outputFormat == "json" {
				return clientCtx.PrintString(formatStatsJSON(stats))
			}

			// Table format
			cmd.Println("=== Pool Statistics ===")
			cmd.Printf("Pool ID: %d\n", pool.Id)
			cmd.Printf("Pair: %s/%s\n", pool.TokenA, pool.TokenB)
			cmd.Println("\n--- Reserves ---")
			cmd.Printf("%s: %s\n", pool.TokenA, pool.ReserveA.String())
			cmd.Printf("%s: %s\n", pool.TokenB, pool.ReserveB.String())
			cmd.Println("\n--- Price ---")
			cmd.Printf("1 %s = %.8f %s\n", pool.TokenA, stats.PriceAtoB, pool.TokenB)
			cmd.Printf("1 %s = %.8f %s\n", pool.TokenB, stats.PriceBtoA, pool.TokenA)
			cmd.Println("\n--- Liquidity ---")
			cmd.Printf("Total Shares: %s\n", pool.TotalShares.String())
			cmd.Printf("K (Constant Product): %s\n", stats.K)
			cmd.Println("\n--- Economics ---")
			cmd.Printf("Estimated 24h Volume: %s\n", stats.Volume24h)
			cmd.Printf("Estimated Fee APY: %.2f%%\n", stats.FeeAPY)
			cmd.Printf("Number of LPs: %d\n", stats.NumLPs)

			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryAllPoolStats returns stats for all pools
func GetCmdQueryAllPoolStats() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all-pool-stats",
		Short: "Get statistics for all liquidity pools",
		Long: `Query statistics for all pools with sorting and filtering options.

Sort options: tvl, volume, apy
Filter by minimum TVL or volume

Examples:
  $ pawd query dex advanced all-pool-stats
  $ pawd query dex advanced all-pool-stats --sort-by apy --limit 10
  $ pawd query dex advanced all-pool-stats --min-tvl 1000000`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			// Get all pools
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			poolsRes, err := queryClient.Pools(context.Background(), &types.QueryPoolsRequest{
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			// Calculate stats for each pool
			var allStats []PoolStats
			for _, pool := range poolsRes.Pools {
				stats := calculatePoolStats(&pool)
				stats.PoolID = pool.Id
				stats.TokenA = pool.TokenA
				stats.TokenB = pool.TokenB
				allStats = append(allStats, stats)
			}

			// Apply sorting
			sortBy, _ := cmd.Flags().GetString("sort-by")
			allStats = sortPoolStats(allStats, sortBy)

			// Apply filters
			minTVL, _ := cmd.Flags().GetInt64("min-tvl")
			if minTVL > 0 {
				allStats = filterByTVL(allStats, minTVL)
			}

			// Display results
			cmd.Println("=== All Pool Statistics ===")
			cmd.Println(strings.Repeat("=", 80))
			cmd.Printf("%-6s %-20s %-15s %-12s %-10s\n", "Pool", "Pair", "TVL", "24h Volume", "Fee APY")
			cmd.Println(strings.Repeat("-", 80))

			for _, stats := range allStats {
				cmd.Printf("%-6d %-20s %-15d %-12s %-10.2f%%\n",
					stats.PoolID,
					fmt.Sprintf("%s/%s", stats.TokenA, stats.TokenB),
					stats.TVL,
					stats.Volume24h,
					stats.FeeAPY,
				)
			}

			cmd.Printf("\nTotal Pools: %d\n", len(allStats))

			return nil
		},
	}

	cmd.Flags().String("sort-by", "tvl", "Sort pools by: tvl, volume, apy")
	cmd.Flags().Int64("min-tvl", 0, "Filter pools with minimum TVL")
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "pools")
	return cmd
}

// GetCmdQueryPrice returns current price for a token pair
func GetCmdQueryPrice() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "price [token-in] [token-out]",
		Short: "Get current price for a token pair",
		Long: `Query the current exchange rate between two tokens.

The price is calculated from the pool reserves.

Examples:
  $ pawd query dex advanced price upaw uusdt
  $ pawd query dex advanced price uatom upaw`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			tokenIn := args[0]
			tokenOut := args[1]

			if tokenIn == tokenOut {
				return fmt.Errorf("tokens must be different")
			}

			queryClient := types.NewQueryClient(clientCtx)

			// Find pool
			poolRes, err := queryClient.PoolByTokens(context.Background(), &types.QueryPoolByTokensRequest{
				TokenA: tokenIn,
				TokenB: tokenOut,
			})
			if err != nil {
				return fmt.Errorf("no pool found for %s/%s: %w", tokenIn, tokenOut, err)
			}

			pool := poolRes.Pool

			// Calculate price
			var price float64
			if tokenIn == pool.TokenA {
				price = float64(pool.ReserveB.Int64()) / float64(pool.ReserveA.Int64())
			} else {
				price = float64(pool.ReserveA.Int64()) / float64(pool.ReserveB.Int64())
			}

			cmd.Println("=== Price Query ===")
			cmd.Printf("Pool ID: %d\n", pool.Id)
			cmd.Printf("1 %s = %.8f %s\n", tokenIn, price, tokenOut)
			cmd.Printf("1 %s = %.8f %s\n", tokenOut, 1/price, tokenIn)
			cmd.Println("\nReserves:")
			cmd.Printf("  %s: %s\n", pool.TokenA, pool.ReserveA.String())
			cmd.Printf("  %s: %s\n", pool.TokenB, pool.ReserveB.String())

			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryPortfolio returns portfolio information for an address
func GetCmdQueryPortfolio() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "portfolio [address]",
		Short: "Get DEX portfolio for an address",
		Long: `Query all liquidity positions and their current value for an address.

Shows:
- All LP positions
- Token amounts
- Share of pool
- Current value

Example:
  $ pawd query dex advanced portfolio paw1abcdef...`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			address := args[0]
			queryClient := types.NewQueryClient(clientCtx)

			// Get all pools
			poolsRes, err := queryClient.Pools(context.Background(), &types.QueryPoolsRequest{})
			if err != nil {
				return err
			}

			cmd.Println("=== DEX Portfolio ===")
			cmd.Printf("Address: %s\n\n", address)

			totalPositions := 0
			hasPositions := false

			for _, pool := range poolsRes.Pools {
				// Query liquidity for this pool
				liqRes, err := queryClient.Liquidity(context.Background(), &types.QueryLiquidityRequest{
					PoolId:   pool.Id,
					Provider: address,
				})
				if err != nil || liqRes.Shares.IsZero() {
					continue
				}

				hasPositions = true
				totalPositions++

				// Calculate share percentage
				sharePercent := float64(liqRes.Shares.Int64()) / float64(pool.TotalShares.Int64()) * 100

				// Calculate token amounts
				tokenAAmount := pool.ReserveA.Mul(liqRes.Shares).Quo(pool.TotalShares)
				tokenBAmount := pool.ReserveB.Mul(liqRes.Shares).Quo(pool.TotalShares)

				cmd.Printf("Pool %d: %s/%s\n", pool.Id, pool.TokenA, pool.TokenB)
				cmd.Printf("  Shares: %s (%.4f%% of pool)\n", liqRes.Shares.String(), sharePercent)
				cmd.Printf("  Tokens: %s %s + %s %s\n",
					tokenAAmount.String(), pool.TokenA,
					tokenBAmount.String(), pool.TokenB)
				cmd.Println()
			}

			if !hasPositions {
				cmd.Println("No liquidity positions found.")
			} else {
				cmd.Printf("Total Positions: %d\n", totalPositions)
			}

			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryArbitrage returns arbitrage opportunities
func GetCmdQueryArbitrage() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "arbitrage",
		Short: "Detect arbitrage opportunities across pools",
		Long: `Scan all pools for arbitrage opportunities.

An arbitrage opportunity exists when you can:
1. Swap A→B in one pool
2. Swap B→A in another route
3. End up with more A than you started

Example:
  $ pawd query dex advanced arbitrage
  $ pawd query dex advanced arbitrage --min-profit 0.5`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			minProfit, _ := cmd.Flags().GetFloat64("min-profit")
			queryClient := types.NewQueryClient(clientCtx)

			// Get all pools
			poolsRes, err := queryClient.Pools(context.Background(), &types.QueryPoolsRequest{})
			if err != nil {
				return err
			}

			cmd.Println("=== Arbitrage Scanner ===")
			cmd.Printf("Scanning %d pools...\n\n", len(poolsRes.Pools))

			opportunities := findArbitrageOpportunities(poolsRes.Pools, minProfit)

			if len(opportunities) == 0 {
				cmd.Println("No arbitrage opportunities found above minimum profit threshold.")
				return nil
			}

			cmd.Printf("Found %d opportunities:\n\n", len(opportunities))

			for i, opp := range opportunities {
				cmd.Printf("[%d] Profit: %.2f%%\n", i+1, opp.ProfitPercent)
				cmd.Printf("    Route: %s\n", opp.Route)
				cmd.Printf("    Estimated Profit: %s\n", opp.EstimatedProfit)
				cmd.Println()
			}

			return nil
		},
	}

	cmd.Flags().Float64("min-profit", 0.5, "Minimum profit percentage to display")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryRoute returns optimal swap routing
func GetCmdQueryRoute() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route [token-in] [token-out] [amount]",
		Short: "Find optimal swap route between tokens",
		Long: `Find the best route to swap between two tokens, including multi-hop routes.

The command will:
- Find all possible routes between tokens
- Calculate expected output for each route
- Recommend the best route

Examples:
  $ pawd query dex advanced route upaw uatom 1000000
  $ pawd query dex advanced route uusdt upaw 5000000`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			tokenIn := args[0]
			tokenOut := args[1]
			amount, ok := math.NewIntFromString(args[2])
			if !ok {
				return fmt.Errorf("invalid amount: %s", args[2])
			}

			queryClient := types.NewQueryClient(clientCtx)

			// Get all pools
			poolsRes, err := queryClient.Pools(context.Background(), &types.QueryPoolsRequest{})
			if err != nil {
				return err
			}

			cmd.Println("=== Swap Route Finder ===")
			cmd.Printf("Finding routes for %s %s → %s\n\n", amount.String(), tokenIn, tokenOut)

			routes := findSwapRoutes(poolsRes.Pools, tokenIn, tokenOut, amount, queryClient, clientCtx)

			if len(routes) == 0 {
				cmd.Println("No routes found between these tokens.")
				return nil
			}

			cmd.Printf("Found %d route(s):\n\n", len(routes))

			for i, route := range routes {
				cmd.Printf("[%d] %s\n", i+1, route.Description)
				cmd.Printf("    Expected Output: %s %s\n", route.OutputAmount.String(), tokenOut)
				cmd.Printf("    Price Impact: %.4f%%\n", route.PriceImpact)
				if i == 0 {
					cmd.Printf("    ⭐ BEST ROUTE\n")
				}
				cmd.Println()
			}

			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryVolume returns trading volume statistics
func GetCmdQueryVolume() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volume [pool-id]",
		Short: "Get trading volume statistics for a pool",
		Long: `Query historical trading volume for a liquidity pool.

Shows volume over different time periods: 24h, 7d, 30d

Example:
  $ pawd query dex advanced volume 1`,
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
			poolRes, err := queryClient.Pool(context.Background(), &types.QueryPoolRequest{PoolId: poolID})
			if err != nil {
				return err
			}
			pool := poolRes.Pool

			maxBlocks, _ := cmd.Flags().GetInt("max-blocks")
			if maxBlocks <= 0 {
				maxBlocks = 4000
			}

			rpcClient := clientCtx.Client
			if rpcClient == nil {
				return fmt.Errorf("rpc client is not initialized (offline mode?)")
			}

			ctx := cmd.Context()
			latestBlock, err := rpcClient.Block(ctx, nil)
			if err != nil {
				return fmt.Errorf("failed to fetch latest block: %w", err)
			}

			latestHeight := latestBlock.Block.Header.Height
			latestTime := latestBlock.Block.Header.Time
			poolIDStr := fmt.Sprintf("%d", poolID)

			buckets := []*volumeBucket{
				newVolumeBucket("24h", 24*time.Hour, pool.TokenA, pool.TokenB),
				newVolumeBucket("7d", 7*24*time.Hour, pool.TokenA, pool.TokenB),
				newVolumeBucket("30d", 30*24*time.Hour, pool.TokenA, pool.TokenB),
			}
			maxDuration := buckets[len(buckets)-1].duration

			var (
				blocksScanned  int
				earliestHeight = latestHeight
				earliestTime   = latestTime
			)

			for height := latestHeight; height > 0 && blocksScanned < maxBlocks; height-- {
				var blockResp *coretypes.ResultBlock
				if height == latestHeight {
					blockResp = latestBlock
				} else {
					blockResp, err = rpcClient.Block(ctx, &height)
					if err != nil {
						return fmt.Errorf("failed to fetch block %d: %w", height, err)
					}
				}

				blockTime := blockResp.Block.Time
				age := latestTime.Sub(blockTime)
				if age > maxDuration {
					break
				}

				results, err := rpcClient.BlockResults(ctx, &height)
				if err != nil {
					return fmt.Errorf("failed to fetch block results %d: %w", height, err)
				}

				blockHasSwaps := false

				for _, txRes := range results.TxsResults {
					if txRes == nil {
						continue
					}
					for _, ev := range txRes.Events {
						if ev.Type != "swap_executed" {
							continue
						}
						attrMap := attributesToMap(ev.Attributes)
						if attrMap["pool_id"] != poolIDStr {
							continue
						}
						if err := updateVolumeBuckets(buckets, attrMap, age); err != nil {
							return err
						}
						blockHasSwaps = true
					}
				}

				if blockHasSwaps {
					blocksScanned++
					earliestHeight = height
					earliestTime = blockTime
				} else if blocksScanned == 0 {
					// If no swaps yet, continue without counting towards block limit
					continue
				} else {
					blocksScanned++
				}
			}

			cmd.Println("=== Trading Volume ===")
			cmd.Printf("Pool: %d (%s/%s)\n", pool.Id, pool.TokenA, pool.TokenB)
			cmd.Printf("Scanned Blocks: %d (heights %d → %d)\n", blocksScanned, earliestHeight, latestHeight)
			cmd.Printf("Time Span: %s\n", latestTime.Sub(earliestTime).Round(time.Minute))
			if blocksScanned >= maxBlocks {
				cmd.Printf("⚠️  Hit max-blocks limit (%d); data may be incomplete for older windows.\n", maxBlocks)
			}

			for _, bucket := range buckets {
				cmd.Printf("\n[%s Window]\n", bucket.label)
				if bucket.trades == 0 {
					cmd.Println("  No swaps recorded.")
					continue
				}
				cmd.Printf("  Swaps: %d\n", bucket.trades)
				cmd.Printf("  %s volume: %s\n", pool.TokenA, formatAmount(bucket.tokenAVolume.String()))
				cmd.Printf("  %s volume: %s\n", pool.TokenB, formatAmount(bucket.tokenBVolume.String()))
			}

			return nil
		},
	}

	cmd.Flags().Int("max-blocks", 4000, "Maximum number of historical blocks to scan for volume data")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryLPPosition returns detailed LP position info
func GetCmdQueryLPPosition() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lp-position [pool-id] [address]",
		Short: "Get detailed liquidity provider position",
		Long: `Query detailed information about an LP position including:
- Current token amounts
- Impermanent loss
- Fees earned
- Position value

Example:
  $ pawd query dex advanced lp-position 1 paw1abcdef...`,
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

			address := args[1]
			queryClient := types.NewQueryClient(clientCtx)

			// Get pool
			poolRes, err := queryClient.Pool(context.Background(), &types.QueryPoolRequest{
				PoolId: poolID,
			})
			if err != nil {
				return err
			}

			pool := poolRes.Pool

			// Get liquidity
			liqRes, err := queryClient.Liquidity(context.Background(), &types.QueryLiquidityRequest{
				PoolId:   poolID,
				Provider: address,
			})
			if err != nil {
				return err
			}

			if liqRes.Shares.IsZero() {
				return fmt.Errorf("no position found")
			}

			// Calculate current token amounts
			tokenAAmount := pool.ReserveA.Mul(liqRes.Shares).Quo(pool.TotalShares)
			tokenBAmount := pool.ReserveB.Mul(liqRes.Shares).Quo(pool.TotalShares)
			sharePercent := float64(liqRes.Shares.Int64()) / float64(pool.TotalShares.Int64()) * 100

			cmd.Println("=== LP Position Details ===")
			cmd.Printf("Pool: %d (%s/%s)\n", pool.Id, pool.TokenA, pool.TokenB)
			cmd.Printf("Provider: %s\n\n", address)
			cmd.Println("--- Position ---")
			cmd.Printf("Shares: %s\n", liqRes.Shares.String())
			cmd.Printf("Pool Share: %.6f%%\n", sharePercent)
			cmd.Println("\n--- Current Value ---")
			cmd.Printf("%s: %s\n", pool.TokenA, tokenAAmount.String())
			cmd.Printf("%s: %s\n", pool.TokenB, tokenBAmount.String())
			cmd.Println("\n--- Economics ---")
			cmd.Println("Impermanent Loss: N/A (requires entry price tracking)")
			cmd.Println("Fees Earned: N/A (requires fee distribution tracking)")

			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryPriceHistory returns price history for a pool
func GetCmdQueryPriceHistory() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "price-history [pool-id]",
		Short: "Get price history for a liquidity pool",
		Long: `Query historical price data for a pool.

Would show price changes over time using TWAP (Time-Weighted Average Price).

Example:
  $ pawd query dex advanced price-history 1`,
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
			poolRes, err := queryClient.Pool(context.Background(), &types.QueryPoolRequest{PoolId: poolID})
			if err != nil {
				return err
			}
			pool := poolRes.Pool

			samples, _ := cmd.Flags().GetInt("samples")
			if samples <= 0 {
				samples = 10
			}
			step, _ := cmd.Flags().GetInt("step-blocks")
			if step <= 0 {
				step = 50
			}

			rpcClient := clientCtx.Client
			if rpcClient == nil {
				return fmt.Errorf("rpc client is not initialized (offline mode?)")
			}

			ctx := cmd.Context()
			latestBlock, err := rpcClient.Block(ctx, nil)
			if err != nil {
				return fmt.Errorf("failed to fetch latest block: %w", err)
			}
			latestHeight := latestBlock.Block.Header.Height

			points := make([]pricePoint, 0, samples)
			var lastErr error

			for i := 0; i < samples; i++ {
				height := latestHeight - int64(i*step)
				if height <= 0 {
					break
				}

				blockResp, err := rpcClient.Block(ctx, &height)
				if err != nil {
					lastErr = fmt.Errorf("failed to fetch block %d: %w", height, err)
					break
				}

				historicalCtx := clientCtx.WithHeight(height)
				historicalQuery := types.NewQueryClient(historicalCtx)
				historicalPool, err := historicalQuery.Pool(context.Background(), &types.QueryPoolRequest{PoolId: poolID})
				if err != nil {
					lastErr = fmt.Errorf("failed to query pool at height %d: %w", height, err)
					break
				}

				priceAB, priceBA := derivePoolPrices(historicalPool.Pool)
				points = append(points, pricePoint{
					Height: height,
					Time:   blockResp.Block.Time,
					AB:     priceAB,
					BA:     priceBA,
				})
			}

			if len(points) == 0 {
				if lastErr != nil {
					return lastErr
				}
				return fmt.Errorf("no historical points collected")
			}

			sort.SliceStable(points, func(i, j int) bool {
				return points[i].Height < points[j].Height
			})

			cmd.Println("=== Price History ===")
			cmd.Printf("Pool: %d (%s/%s)\n", pool.Id, pool.TokenA, pool.TokenB)
			cmd.Printf("Samples: %d, Step: %d blocks\n\n", len(points), step)
			cmd.Printf("%-12s %-24s %-20s %-20s\n", "Height", "Timestamp (UTC)", fmt.Sprintf("1 %s -> %s", pool.TokenA, pool.TokenB), fmt.Sprintf("1 %s -> %s", pool.TokenB, pool.TokenA))
			cmd.Println(strings.Repeat("-", 80))

			for _, pt := range points {
				cmd.Printf("%-12d %-24s %-20f %-20f\n",
					pt.Height,
					pt.Time.UTC().Format(time.RFC3339),
					pt.AB,
					pt.BA,
				)
			}

			if lastErr != nil {
				cmd.Printf("\n⚠️  %v\n", lastErr)
			}

			return nil
		},
	}

	cmd.Flags().Int("samples", 10, "Number of historical samples to fetch")
	cmd.Flags().Int("step-blocks", 50, "Number of blocks between samples")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryFeeAPY calculates fee APY for a pool
func GetCmdQueryFeeAPY() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fee-apy [pool-id]",
		Short: "Calculate fee APY for a liquidity pool",
		Long: `Calculate the current APY from trading fees for LP providers.

Based on recent trading volume and pool TVL.

Example:
  $ pawd query dex advanced fee-apy 1`,
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

			// Get pool
			poolRes, err := queryClient.Pool(context.Background(), &types.QueryPoolRequest{
				PoolId: poolID,
			})
			if err != nil {
				return err
			}

			pool := poolRes.Pool

			// Get params for fee rate
			paramsRes, err := queryClient.Params(context.Background(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			cmd.Println("=== Fee APY Calculation ===")
			cmd.Printf("Pool: %d (%s/%s)\n\n", pool.Id, pool.TokenA, pool.TokenB)
			cmd.Printf("Swap Fee: %s\n", paramsRes.Params.SwapFee)
			cmd.Println("\n⚠️  APY calculation requires:")
			cmd.Println("- Historical volume data")
			cmd.Println("- Current TVL valuation")
			cmd.Println("- Fee distribution tracking")
			cmd.Println("\nEstimated APY: N/A")

			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// Helper types and functions

type volumeBucket struct {
	label        string
	duration     time.Duration
	tokenA       string
	tokenB       string
	tokenAVolume math.Int
	tokenBVolume math.Int
	trades       int
}

func newVolumeBucket(label string, duration time.Duration, tokenA, tokenB string) *volumeBucket {
	return &volumeBucket{
		label:        label,
		duration:     duration,
		tokenA:       tokenA,
		tokenB:       tokenB,
		tokenAVolume: math.ZeroInt(),
		tokenBVolume: math.ZeroInt(),
	}
}

func (b *volumeBucket) addVolume(token string, amount math.Int) {
	if amount.IsZero() {
		return
	}

	switch token {
	case b.tokenA:
		b.tokenAVolume = b.tokenAVolume.Add(amount)
	case b.tokenB:
		b.tokenBVolume = b.tokenBVolume.Add(amount)
	}
}

type pricePoint struct {
	Height int64
	Time   time.Time
	AB     float64
	BA     float64
}

func attributesToMap(attrs []abcitypes.EventAttribute) map[string]string {
	out := make(map[string]string, len(attrs))
	for _, attr := range attrs {
		out[string(attr.Key)] = string(attr.Value)
	}
	return out
}

func updateVolumeBuckets(buckets []*volumeBucket, attrs map[string]string, age time.Duration) error {
	amountIn, ok := math.NewIntFromString(attrs["amount_in"])
	if !ok {
		return fmt.Errorf("invalid amount_in in swap_executed event")
	}
	amountOut, ok := math.NewIntFromString(attrs["amount_out"])
	if !ok {
		return fmt.Errorf("invalid amount_out in swap_executed event")
	}

	tokenIn := attrs["token_in"]
	tokenOut := attrs["token_out"]
	if tokenIn == "" || tokenOut == "" {
		return fmt.Errorf("swap_executed event missing token metadata")
	}

	for _, bucket := range buckets {
		if age > bucket.duration {
			continue
		}
		bucket.trades++
		bucket.addVolume(tokenIn, amountIn)
		bucket.addVolume(tokenOut, amountOut)
	}

	return nil
}

func derivePoolPrices(pool types.Pool) (float64, float64) {
	if pool.ReserveA.IsZero() || pool.ReserveB.IsZero() {
		return 0, 0
	}

	ab := math.LegacyNewDecFromInt(pool.ReserveB).Quo(math.LegacyNewDecFromInt(pool.ReserveA))
	ba := math.LegacyNewDecFromInt(pool.ReserveA).Quo(math.LegacyNewDecFromInt(pool.ReserveB))

	abFloat, err := ab.Float64()
	if err != nil {
		abFloat = 0
	}
	baFloat, err := ba.Float64()
	if err != nil {
		baFloat = 0
	}

	return abFloat, baFloat
}

type PoolStats struct {
	PoolID    uint64
	TokenA    string
	TokenB    string
	PriceAtoB float64
	PriceBtoA float64
	K         string
	TVL       int64
	Volume24h string
	FeeAPY    float64
	NumLPs    int
}

func calculatePoolStats(pool *types.Pool) PoolStats {
	priceAtoB := float64(pool.ReserveB.Int64()) / float64(pool.ReserveA.Int64())
	priceBtoA := 1.0 / priceAtoB
	k := pool.ReserveA.Mul(pool.ReserveB).String()

	return PoolStats{
		PriceAtoB: priceAtoB,
		PriceBtoA: priceBtoA,
		K:         k,
		TVL:       pool.ReserveA.Int64() + pool.ReserveB.Int64(), // Simplified
		Volume24h: "N/A",
		FeeAPY:    0.0,
		NumLPs:    0, // Would need to query all LP positions
	}
}

func formatStatsJSON(stats PoolStats) string {
	return fmt.Sprintf(`{
  "price_a_to_b": %.8f,
  "price_b_to_a": %.8f,
  "constant_k": "%s",
  "tvl": %d,
  "volume_24h": "%s",
  "fee_apy": %.2f,
  "num_lps": %d
}`, stats.PriceAtoB, stats.PriceBtoA, stats.K, stats.TVL, stats.Volume24h, stats.FeeAPY, stats.NumLPs)
}

func sortPoolStats(stats []PoolStats, sortBy string) []PoolStats {
	switch strings.ToLower(sortBy) {
	case "volume":
		sort.SliceStable(stats, func(i, j int) bool {
			return parseVolumeValue(stats[i].Volume24h) > parseVolumeValue(stats[j].Volume24h)
		})
	case "apy":
		sort.SliceStable(stats, func(i, j int) bool {
			return stats[i].FeeAPY > stats[j].FeeAPY
		})
	default:
		sort.SliceStable(stats, func(i, j int) bool {
			return stats[i].TVL > stats[j].TVL
		})
	}
	return stats
}

func filterByTVL(stats []PoolStats, minTVL int64) []PoolStats {
	var filtered []PoolStats
	for _, s := range stats {
		if s.TVL >= minTVL {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func formatAmount(amount string) string {
	if amount == "" {
		return "0"
	}

	neg := false
	if amount[0] == '-' {
		neg = true
		amount = amount[1:]
	}

	if len(amount) <= 3 {
		if neg {
			return "-" + amount
		}
		return amount
	}

	var parts []string
	for len(amount) > 3 {
		part := amount[len(amount)-3:]
		parts = append([]string{part}, parts...)
		amount = amount[:len(amount)-3]
	}
	parts = append([]string{amount}, parts...)

	result := strings.Join(parts, ",")
	if neg {
		result = "-" + result
	}
	return result
}

type ArbitrageOpportunity struct {
	Route           string
	ProfitPercent   float64
	EstimatedProfit string
}

func findArbitrageOpportunities(pools []types.Pool, minProfit float64) []ArbitrageOpportunity {
	var opportunities []ArbitrageOpportunity

	for i := range pools {
		for j := i + 1; j < len(pools); j++ {
			if !sameTokenPair(pools[i], pools[j]) {
				continue
			}

			priceA, _ := derivePoolPrices(pools[i])
			priceB, _ := derivePoolPrices(pools[j])
			if priceA <= 0 || priceB <= 0 {
				continue
			}

			diff := stdmath.Abs(priceA - priceB)
			base := stdmath.Min(priceA, priceB)
			if base == 0 {
				continue
			}

			profit := (diff / base) * 100
			if profit < minProfit {
				continue
			}

			opportunities = append(opportunities, ArbitrageOpportunity{
				Route:         fmt.Sprintf("Pool %d ⇄ Pool %d (%s/%s)", pools[i].Id, pools[j].Id, pools[i].TokenA, pools[i].TokenB),
				ProfitPercent: profit,
				EstimatedProfit: fmt.Sprintf("Buy low in pool %d, sell high in pool %d (spread %.2f%%)",
					pools[i].Id, pools[j].Id, profit),
			})
		}
	}

	return opportunities
}

type SwapRoute struct {
	Description  string
	OutputAmount math.Int
	PriceImpact  float64
}

func findSwapRoutes(pools []types.Pool, tokenIn, tokenOut string, amount math.Int, queryClient types.QueryClient, clientCtx client.Context) []SwapRoute {
	var routes []SwapRoute
	ctx := context.Background()

	// Direct routes
	for _, pool := range pools {
		if !poolHasPair(pool, tokenIn, tokenOut) {
			continue
		}
		amountOut, err := queryClient.SimulateSwap(ctx, &types.QuerySimulateSwapRequest{
			PoolId:   pool.Id,
			TokenIn:  tokenIn,
			TokenOut: tokenOut,
			AmountIn: amount,
		})
		if err != nil {
			continue
		}
		routes = append(routes, SwapRoute{
			Description:  fmt.Sprintf("%s → %s via pool %d", tokenIn, tokenOut, pool.Id),
			OutputAmount: amountOut.AmountOut,
			PriceImpact:  estimatePriceImpact(amount, amountOut.AmountOut, pool, tokenIn),
		})
	}

	// Two-hop routes
	for _, firstPool := range pools {
		if !poolHasToken(firstPool, tokenIn) {
			continue
		}
		intermediate := otherToken(firstPool, tokenIn)
		if intermediate == "" {
			continue
		}

		firstHop, err := queryClient.SimulateSwap(ctx, &types.QuerySimulateSwapRequest{
			PoolId:   firstPool.Id,
			TokenIn:  tokenIn,
			TokenOut: intermediate,
			AmountIn: amount,
		})
		if err != nil || firstHop.AmountOut.IsZero() {
			continue
		}

		for _, secondPool := range pools {
			if secondPool.Id == firstPool.Id {
				continue
			}
			if !poolHasPair(secondPool, intermediate, tokenOut) {
				continue
			}

			secondHop, err := queryClient.SimulateSwap(ctx, &types.QuerySimulateSwapRequest{
				PoolId:   secondPool.Id,
				TokenIn:  intermediate,
				TokenOut: tokenOut,
				AmountIn: firstHop.AmountOut,
			})
			if err != nil || secondHop.AmountOut.IsZero() {
				continue
			}

			impact := estimatePriceImpact(amount, secondHop.AmountOut, firstPool, tokenIn)
			impact += estimatePriceImpact(firstHop.AmountOut, secondHop.AmountOut, secondPool, intermediate)

			routes = append(routes, SwapRoute{
				Description:  fmt.Sprintf("%s → %s → %s via pools %d -> %d", tokenIn, intermediate, tokenOut, firstPool.Id, secondPool.Id),
				OutputAmount: secondHop.AmountOut,
				PriceImpact:  impact,
			})
		}
	}

	sort.SliceStable(routes, func(i, j int) bool {
		return routes[i].OutputAmount.GT(routes[j].OutputAmount)
	})

	if len(routes) > 5 {
		return routes[:5]
	}
	return routes
}

func parseVolumeValue(volume string) int64 {
	clean := strings.ReplaceAll(volume, ",", "")
	if clean == "" || clean == "N/A" {
		return 0
	}
	val, err := strconv.ParseInt(clean, 10, 64)
	if err != nil {
		return 0
	}
	return val
}

func sameTokenPair(a, b types.Pool) bool {
	return (a.TokenA == b.TokenA && a.TokenB == b.TokenB) ||
		(a.TokenA == b.TokenB && a.TokenB == b.TokenA)
}

func poolHasPair(pool types.Pool, tokenIn, tokenOut string) bool {
	return (pool.TokenA == tokenIn && pool.TokenB == tokenOut) ||
		(pool.TokenA == tokenOut && pool.TokenB == tokenIn)
}

func poolHasToken(pool types.Pool, token string) bool {
	return pool.TokenA == token || pool.TokenB == token
}

func otherToken(pool types.Pool, token string) string {
	if pool.TokenA == token {
		return pool.TokenB
	}
	if pool.TokenB == token {
		return pool.TokenA
	}
	return ""
}

func estimatePriceImpact(amountIn, amountOut math.Int, pool types.Pool, tokenIn string) float64 {
	if amountIn.IsZero() || amountOut.IsZero() {
		return 0
	}

	var reserveIn, reserveOut math.Int
	if pool.TokenA == tokenIn {
		reserveIn = pool.ReserveA
		reserveOut = pool.ReserveB
	} else if pool.TokenB == tokenIn {
		reserveIn = pool.ReserveB
		reserveOut = pool.ReserveA
	} else {
		return 0
	}

	if reserveIn.IsZero() || reserveOut.IsZero() {
		return 0
	}

	spotPrice := math.LegacyNewDecFromInt(reserveOut).Quo(math.LegacyNewDecFromInt(reserveIn))
	executionPrice := math.LegacyNewDecFromInt(amountOut).Quo(math.LegacyNewDecFromInt(amountIn))
	if spotPrice.IsZero() {
		return 0
	}

	diff := executionPrice.Sub(spotPrice).Abs()
	impact, err := diff.Quo(spotPrice).Float64()
	if err != nil {
		return 0
	}
	return impact * 100
}
