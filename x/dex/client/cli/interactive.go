package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/paw-chain/paw/x/dex/types"
)

// GetInteractiveCmd returns the interactive DEX trading command
func GetInteractiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "interactive",
		Short: "Interactive DEX trading terminal",
		Long: `Launch an interactive terminal for DEX trading with a user-friendly interface.

Features:
- Menu-driven navigation
- Real-time pool information
- Guided swap execution
- Portfolio management
- Visual price displays

Example:
  $ pawd dex interactive --from mykey`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			terminal := NewInteractiveTerminal(clientCtx, cmd)
			return terminal.Run()
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// InteractiveTerminal provides an interactive trading interface
type InteractiveTerminal struct {
	clientCtx   client.Context
	cmd         *cobra.Command
	queryClient types.QueryClient
	reader      *bufio.Reader
	userAddress string
}

// NewInteractiveTerminal creates a new interactive terminal
func NewInteractiveTerminal(clientCtx client.Context, cmd *cobra.Command) *InteractiveTerminal {
	return &InteractiveTerminal{
		clientCtx:   clientCtx,
		cmd:         cmd,
		queryClient: types.NewQueryClient(clientCtx),
		reader:      bufio.NewReader(os.Stdin),
		userAddress: clientCtx.GetFromAddress().String(),
	}
}

// Run starts the interactive terminal
func (t *InteractiveTerminal) Run() error {
	t.printWelcome()

	for {
		t.printMainMenu()
		choice := t.readInput("Select option")

		switch choice {
		case "1":
			if err := t.viewPools(); err != nil {
				t.printError(err)
			}
		case "2":
			if err := t.executeSwap(); err != nil {
				t.printError(err)
			}
		case "3":
			if err := t.addLiquidity(); err != nil {
				t.printError(err)
			}
		case "4":
			if err := t.removeLiquidity(); err != nil {
				t.printError(err)
			}
		case "5":
			if err := t.viewPortfolio(); err != nil {
				t.printError(err)
			}
		case "6":
			if err := t.priceCalculator(); err != nil {
				t.printError(err)
			}
		case "7":
			if err := t.viewOrderBook(); err != nil {
				t.printError(err)
			}
		case "8":
			t.printGoodbye()
			return nil
		default:
			fmt.Println("❌ Invalid option. Please try again.")
		}

		fmt.Println("\nPress Enter to continue...")
		t.reader.ReadString('\n')
	}
}

func (t *InteractiveTerminal) printWelcome() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("        PAW DEX Interactive Trading Terminal")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Connected as: %s\n", t.userAddress)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()
}

func (t *InteractiveTerminal) printMainMenu() {
	fmt.Println("\n╔═══════════════════════════════════════════════════════╗")
	fmt.Println("║                    MAIN MENU                          ║")
	fmt.Println("╠═══════════════════════════════════════════════════════╣")
	fmt.Println("║  1. View All Pools                                    ║")
	fmt.Println("║  2. Execute Swap                                      ║")
	fmt.Println("║  3. Add Liquidity                                     ║")
	fmt.Println("║  4. Remove Liquidity                                  ║")
	fmt.Println("║  5. View My Portfolio                                 ║")
	fmt.Println("║  6. Price Calculator                                  ║")
	fmt.Println("║  7. View Order Book                                   ║")
	fmt.Println("║  8. Exit                                              ║")
	fmt.Println("╚═══════════════════════════════════════════════════════╝")
	fmt.Println()
}

func (t *InteractiveTerminal) printGoodbye() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Thank you for using PAW DEX!")
	fmt.Println("Happy trading! 🐾")
	fmt.Println(strings.Repeat("=", 60))
}

func (t *InteractiveTerminal) viewPools() error {
	fmt.Println("\n╔═══════════════════════════════════════════════════════╗")
	fmt.Println("║                   LIQUIDITY POOLS                     ║")
	fmt.Println("╚═══════════════════════════════════════════════════════╝")

	poolsRes, err := t.queryClient.Pools(context.Background(), &types.QueryPoolsRequest{})
	if err != nil {
		return fmt.Errorf("failed to fetch pools: %w", err)
	}

	if len(poolsRes.Pools) == 0 {
		fmt.Println("\n❌ No pools available.")
		return nil
	}

	fmt.Println()
	fmt.Printf("%-6s %-20s %-20s %-20s\n", "ID", "Pair", "Reserve A", "Reserve B")
	fmt.Println(strings.Repeat("-", 70))

	for _, pool := range poolsRes.Pools {
		pairStr := fmt.Sprintf("%s/%s", pool.TokenA, pool.TokenB)
		fmt.Printf("%-6d %-20s %-20s %-20s\n",
			pool.Id,
			pairStr,
			formatTokenAmount(pool.ReserveA, pool.TokenA),
			formatTokenAmount(pool.ReserveB, pool.TokenB),
		)
	}

	fmt.Printf("\nTotal Pools: %d\n", len(poolsRes.Pools))

	// Option to view pool details
	viewDetails := t.readInput("\nView details for pool ID (or press Enter to skip)")
	if viewDetails != "" {
		poolID, err := strconv.ParseUint(viewDetails, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid pool ID: %w", err)
		}
		return t.viewPoolDetails(poolID)
	}

	return nil
}

func (t *InteractiveTerminal) viewPoolDetails(poolID uint64) error {
	poolRes, err := t.queryClient.Pool(context.Background(), &types.QueryPoolRequest{
		PoolId: poolID,
	})
	if err != nil {
		return err
	}

	pool := poolRes.Pool

	fmt.Println("\n╔═══════════════════════════════════════════════════════╗")
	fmt.Printf("║              POOL #%d DETAILS                          ║\n", poolID)
	fmt.Println("╚═══════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Pair:        %s / %s\n", pool.TokenA, pool.TokenB)
	fmt.Printf("Reserve A:   %s %s\n", pool.ReserveA.String(), pool.TokenA)
	fmt.Printf("Reserve B:   %s %s\n", pool.ReserveB.String(), pool.TokenB)
	fmt.Printf("Total Shares: %s\n", pool.TotalShares.String())
	fmt.Printf("Creator:     %s\n", pool.Creator)
	fmt.Println()

	// Calculate and display price
	priceAtoB := float64(pool.ReserveB.Int64()) / float64(pool.ReserveA.Int64())
	priceBtoA := 1.0 / priceAtoB
	fmt.Println("Current Price:")
	fmt.Printf("  1 %s = %.8f %s\n", pool.TokenA, priceAtoB, pool.TokenB)
	fmt.Printf("  1 %s = %.8f %s\n", pool.TokenB, priceBtoA, pool.TokenA)

	return nil
}

func (t *InteractiveTerminal) executeSwap() error {
	fmt.Println("\n╔═══════════════════════════════════════════════════════╗")
	fmt.Println("║                    EXECUTE SWAP                       ║")
	fmt.Println("╚═══════════════════════════════════════════════════════╝")
	fmt.Println()

	// Get pool ID
	poolIDStr := t.readInput("Enter Pool ID")
	poolID, err := strconv.ParseUint(poolIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid pool ID: %w", err)
	}

	// Get pool info
	poolRes, err := t.queryClient.Pool(context.Background(), &types.QueryPoolRequest{
		PoolId: poolID,
	})
	if err != nil {
		return err
	}

	pool := poolRes.Pool
	fmt.Printf("\nPool: %s / %s\n", pool.TokenA, pool.TokenB)

	// Get input token
	tokenIn := t.readInput(fmt.Sprintf("Token In (%s or %s)", pool.TokenA, pool.TokenB))
	if tokenIn != pool.TokenA && tokenIn != pool.TokenB {
		return fmt.Errorf("token must be either %s or %s", pool.TokenA, pool.TokenB)
	}

	// Determine output token
	var tokenOut string
	if tokenIn == pool.TokenA {
		tokenOut = pool.TokenB
	} else {
		tokenOut = pool.TokenA
	}

	// Get amount
	amountInStr := t.readInput("Amount In")
	amountIn, ok := math.NewIntFromString(amountInStr)
	if !ok {
		return fmt.Errorf("invalid amount: %s", amountInStr)
	}

	// Simulate swap
	fmt.Println("\n⏳ Simulating swap...")
	simRes, err := t.queryClient.SimulateSwap(context.Background(), &types.QuerySimulateSwapRequest{
		PoolId:   poolID,
		TokenIn:  tokenIn,
		TokenOut: tokenOut,
		AmountIn: amountIn,
	})
	if err != nil {
		return fmt.Errorf("simulation failed: %w", err)
	}

	expectedOut := simRes.AmountOut

	// Display swap details
	fmt.Println("\n" + strings.Repeat("-", 50))
	fmt.Println("SWAP DETAILS:")
	fmt.Printf("Input:           %s %s\n", amountIn.String(), tokenIn)
	fmt.Printf("Expected Output: %s %s\n", expectedOut.String(), tokenOut)

	// Calculate price
	price := float64(expectedOut.Int64()) / float64(amountIn.Int64())
	fmt.Printf("Exchange Rate:   1 %s = %.8f %s\n", tokenIn, price, tokenOut)
	fmt.Println(strings.Repeat("-", 50))

	// Get slippage tolerance
	slippageStr := t.readInput("\nSlippage tolerance (%) [default: 1.0]")
	slippage := 1.0
	if slippageStr != "" {
		slippage, err = strconv.ParseFloat(slippageStr, 64)
		if err != nil {
			return fmt.Errorf("invalid slippage: %w", err)
		}
	}

	// Calculate min output
	slippageFactor := 1.0 - (slippage / 100.0)
	minOutputFloat := float64(expectedOut.Int64()) * slippageFactor
	minAmountOut := math.NewInt(int64(minOutputFloat))

	fmt.Printf("Minimum Output:  %s %s (with %.1f%% slippage)\n", minAmountOut.String(), tokenOut, slippage)

	// Confirm
	confirm := t.readInput("\n⚠️  Execute this swap? (yes/no)")
	if strings.ToLower(confirm) != "yes" && strings.ToLower(confirm) != "y" {
		fmt.Println("❌ Swap cancelled.")
		return nil
	}

	// Execute swap
	deadline := int64(300) // 5 minutes
	msg := &types.MsgSwap{
		Trader:       t.userAddress,
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

	fmt.Println("\n⏳ Broadcasting transaction...")
	err = tx.GenerateOrBroadcastTxCLI(t.clientCtx, t.cmd.Flags(), msg)
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	fmt.Println("✅ Swap executed successfully!")
	return nil
}

func (t *InteractiveTerminal) addLiquidity() error {
	fmt.Println("\n╔═══════════════════════════════════════════════════════╗")
	fmt.Println("║                   ADD LIQUIDITY                       ║")
	fmt.Println("╚═══════════════════════════════════════════════════════╝")
	fmt.Println()

	// Get pool ID
	poolIDStr := t.readInput("Enter Pool ID")
	poolID, err := strconv.ParseUint(poolIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid pool ID: %w", err)
	}

	// Get pool info
	poolRes, err := t.queryClient.Pool(context.Background(), &types.QueryPoolRequest{
		PoolId: poolID,
	})
	if err != nil {
		return err
	}

	pool := poolRes.Pool

	fmt.Printf("\nPool: %s / %s\n", pool.TokenA, pool.TokenB)
	fmt.Printf("Current Ratio: 1 %s = %.6f %s\n",
		pool.TokenA,
		float64(pool.ReserveB.Int64())/float64(pool.ReserveA.Int64()),
		pool.TokenB)

	// Get amount A
	amountAStr := t.readInput(fmt.Sprintf("\nAmount %s", pool.TokenA))
	amountA, ok := math.NewIntFromString(amountAStr)
	if !ok {
		return fmt.Errorf("invalid amount: %s", amountAStr)
	}

	// Calculate proportional amount B
	amountB := amountA.Mul(pool.ReserveB).Quo(pool.ReserveA)

	fmt.Printf("\nProportional amount %s: %s\n", pool.TokenB, amountB.String())

	// Allow manual override
	override := t.readInput(fmt.Sprintf("Override %s amount? (leave blank to use calculated)", pool.TokenB))
	if override != "" {
		var ok bool
		amountB, ok = math.NewIntFromString(override)
		if !ok {
			return fmt.Errorf("invalid amount: %s", override)
		}
	}

	// Confirm
	fmt.Println("\n" + strings.Repeat("-", 50))
	fmt.Printf("Adding: %s %s + %s %s\n", amountA.String(), pool.TokenA, amountB.String(), pool.TokenB)
	fmt.Println(strings.Repeat("-", 50))

	confirm := t.readInput("\nConfirm? (yes/no)")
	if strings.ToLower(confirm) != "yes" && strings.ToLower(confirm) != "y" {
		fmt.Println("❌ Operation cancelled.")
		return nil
	}

	// Execute
	msg := &types.MsgAddLiquidity{
		Provider: t.userAddress,
		PoolId:   poolID,
		AmountA:  amountA,
		AmountB:  amountB,
	}

	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	fmt.Println("\n⏳ Broadcasting transaction...")
	err = tx.GenerateOrBroadcastTxCLI(t.clientCtx, t.cmd.Flags(), msg)
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	fmt.Println("✅ Liquidity added successfully!")
	return nil
}

func (t *InteractiveTerminal) removeLiquidity() error {
	fmt.Println("\n╔═══════════════════════════════════════════════════════╗")
	fmt.Println("║                  REMOVE LIQUIDITY                     ║")
	fmt.Println("╚═══════════════════════════════════════════════════════╝")
	fmt.Println()

	// Get pool ID
	poolIDStr := t.readInput("Enter Pool ID")
	poolID, err := strconv.ParseUint(poolIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid pool ID: %w", err)
	}

	// Get user's liquidity
	liqRes, err := t.queryClient.Liquidity(context.Background(), &types.QueryLiquidityRequest{
		PoolId:   poolID,
		Provider: t.userAddress,
	})
	if err != nil {
		return err
	}

	if liqRes.Shares.IsZero() {
		return fmt.Errorf("you have no liquidity in this pool")
	}

	// Get pool info
	poolRes, err := t.queryClient.Pool(context.Background(), &types.QueryPoolRequest{
		PoolId: poolID,
	})
	if err != nil {
		return err
	}

	pool := poolRes.Pool

	fmt.Printf("\nYour Shares: %s\n", liqRes.Shares.String())
	fmt.Printf("Pool: %s / %s\n", pool.TokenA, pool.TokenB)

	// Calculate current value
	tokenA := pool.ReserveA.Mul(liqRes.Shares).Quo(pool.TotalShares)
	tokenB := pool.ReserveB.Mul(liqRes.Shares).Quo(pool.TotalShares)

	fmt.Printf("\nCurrent Value:\n")
	fmt.Printf("  %s: %s\n", pool.TokenA, tokenA.String())
	fmt.Printf("  %s: %s\n", pool.TokenB, tokenB.String())

	// Get shares to remove
	sharesStr := t.readInput(fmt.Sprintf("\nShares to remove (max: %s, 0 for all)", liqRes.Shares.String()))
	var shares math.Int
	if sharesStr == "0" || sharesStr == "" {
		shares = liqRes.Shares
	} else {
		var ok bool
		shares, ok = math.NewIntFromString(sharesStr)
		if !ok {
			return fmt.Errorf("invalid shares: %s", sharesStr)
		}
		if shares.GT(liqRes.Shares) {
			return fmt.Errorf("insufficient shares")
		}
	}

	// Calculate output
	outA := pool.ReserveA.Mul(shares).Quo(pool.TotalShares)
	outB := pool.ReserveB.Mul(shares).Quo(pool.TotalShares)

	fmt.Println("\n" + strings.Repeat("-", 50))
	fmt.Printf("Removing: %s shares\n", shares.String())
	fmt.Printf("Receiving: %s %s + %s %s\n", outA.String(), pool.TokenA, outB.String(), pool.TokenB)
	fmt.Println(strings.Repeat("-", 50))

	confirm := t.readInput("\nConfirm? (yes/no)")
	if strings.ToLower(confirm) != "yes" && strings.ToLower(confirm) != "y" {
		fmt.Println("❌ Operation cancelled.")
		return nil
	}

	// Execute
	msg := &types.MsgRemoveLiquidity{
		Provider: t.userAddress,
		PoolId:   poolID,
		Shares:   shares,
	}

	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	fmt.Println("\n⏳ Broadcasting transaction...")
	err = tx.GenerateOrBroadcastTxCLI(t.clientCtx, t.cmd.Flags(), msg)
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	fmt.Println("✅ Liquidity removed successfully!")
	return nil
}

func (t *InteractiveTerminal) viewPortfolio() error {
	fmt.Println("\n╔═══════════════════════════════════════════════════════╗")
	fmt.Println("║                   MY PORTFOLIO                        ║")
	fmt.Println("╚═══════════════════════════════════════════════════════╝")
	fmt.Printf("\nAddress: %s\n\n", t.userAddress)

	// Get all pools
	poolsRes, err := t.queryClient.Pools(context.Background(), &types.QueryPoolsRequest{})
	if err != nil {
		return err
	}

	hasPositions := false
	totalPools := 0

	for _, pool := range poolsRes.Pools {
		liqRes, err := t.queryClient.Liquidity(context.Background(), &types.QueryLiquidityRequest{
			PoolId:   pool.Id,
			Provider: t.userAddress,
		})
		if err != nil || liqRes.Shares.IsZero() {
			continue
		}

		hasPositions = true
		totalPools++

		// Calculate values
		tokenA := pool.ReserveA.Mul(liqRes.Shares).Quo(pool.TotalShares)
		tokenB := pool.ReserveB.Mul(liqRes.Shares).Quo(pool.TotalShares)
		sharePercent := float64(liqRes.Shares.Int64()) / float64(pool.TotalShares.Int64()) * 100

		fmt.Printf("Pool #%d: %s/%s\n", pool.Id, pool.TokenA, pool.TokenB)
		fmt.Printf("  Shares: %s (%.4f%% of pool)\n", liqRes.Shares.String(), sharePercent)
		fmt.Printf("  Value:  %s %s + %s %s\n\n", tokenA.String(), pool.TokenA, tokenB.String(), pool.TokenB)
	}

	if !hasPositions {
		fmt.Println("❌ No liquidity positions found.")
	} else {
		fmt.Printf("Total Positions: %d\n", totalPools)
	}

	return nil
}

func (t *InteractiveTerminal) priceCalculator() error {
	fmt.Println("\n╔═══════════════════════════════════════════════════════╗")
	fmt.Println("║                  PRICE CALCULATOR                     ║")
	fmt.Println("╚═══════════════════════════════════════════════════════╝")
	fmt.Println()

	tokenIn := t.readInput("Token In")
	tokenOut := t.readInput("Token Out")
	amountStr := t.readInput("Amount")

	amount, ok := math.NewIntFromString(amountStr)
	if !ok {
		return fmt.Errorf("invalid amount: %s", amountStr)
	}

	// Find pool
	poolRes, err := t.queryClient.PoolByTokens(context.Background(), &types.QueryPoolByTokensRequest{
		TokenA: tokenIn,
		TokenB: tokenOut,
	})
	if err != nil {
		return fmt.Errorf("no pool found for %s/%s: %w", tokenIn, tokenOut, err)
	}

	// Simulate
	simRes, err := t.queryClient.SimulateSwap(context.Background(), &types.QuerySimulateSwapRequest{
		PoolId:   poolRes.Pool.Id,
		TokenIn:  tokenIn,
		TokenOut: tokenOut,
		AmountIn: amount,
	})
	if err != nil {
		return err
	}

	fmt.Println("\n" + strings.Repeat("-", 50))
	fmt.Printf("Input:  %s %s\n", amount.String(), tokenIn)
	fmt.Printf("Output: %s %s\n", simRes.AmountOut.String(), tokenOut)
	rate := float64(simRes.AmountOut.Int64()) / float64(amount.Int64())
	fmt.Printf("Rate:   1 %s = %.8f %s\n", tokenIn, rate, tokenOut)
	fmt.Println(strings.Repeat("-", 50))

	return nil
}

func (t *InteractiveTerminal) viewOrderBook() error {
	fmt.Println("\n╔═══════════════════════════════════════════════════════╗")
	fmt.Println("║                    ORDER BOOK                         ║")
	fmt.Println("╚═══════════════════════════════════════════════════════╝")
	fmt.Println()

	poolIDStr := t.readInput("Enter Pool ID")
	poolID, err := strconv.ParseUint(poolIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid pool ID: %w", err)
	}

	orderBookRes, err := t.queryClient.OrderBook(context.Background(), &types.QueryOrderBookRequest{
		PoolId: poolID,
		Limit:  20,
	})
	if err != nil {
		return err
	}

	fmt.Println("\n📊 BUY ORDERS:")
	if len(orderBookRes.BuyOrders) == 0 {
		fmt.Println("  No buy orders")
	} else {
		for _, order := range orderBookRes.BuyOrders {
			fmt.Printf("  #%d: %s @ %s\n", order.Id, order.AmountIn.String(), order.LimitPrice)
		}
	}

	fmt.Println("\n📊 SELL ORDERS:")
	if len(orderBookRes.SellOrders) == 0 {
		fmt.Println("  No sell orders")
	} else {
		for _, order := range orderBookRes.SellOrders {
			fmt.Printf("  #%d: %s @ %s\n", order.Id, order.AmountIn.String(), order.LimitPrice)
		}
	}

	return nil
}

func (t *InteractiveTerminal) readInput(prompt string) string {
	fmt.Printf("%s: ", prompt)
	input, _ := t.reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func (t *InteractiveTerminal) printError(err error) {
	fmt.Printf("\n❌ Error: %s\n", err.Error())
}

func formatTokenAmount(amount math.Int, denom string) string {
	return fmt.Sprintf("%s %s", amount.String(), denom)
}
