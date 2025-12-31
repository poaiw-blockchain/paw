package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestFlagConstants verifies all flag constants are properly defined
func TestFlagConstants(t *testing.T) {
	t.Parallel()

	// Pool creation flags
	require.Equal(t, "token-a", FlagTokenA)
	require.Equal(t, "token-b", FlagTokenB)
	require.Equal(t, "amount-a", FlagAmountA)
	require.Equal(t, "amount-b", FlagAmountB)

	// Liquidity flags
	require.Equal(t, "shares", FlagShares)

	// Swap flags
	require.Equal(t, "token-in", FlagTokenIn)
	require.Equal(t, "token-out", FlagTokenOut)
	require.Equal(t, "amount-in", FlagAmountIn)
	require.Equal(t, "min-amount-out", FlagMinAmountOut)
	require.Equal(t, "slippage", FlagSlippage)
	require.Equal(t, "deadline", FlagDeadline)
}

// TestGetQueryCmdStructure verifies the query command tree structure
func TestGetQueryCmdStructure(t *testing.T) {
	t.Parallel()

	queryCmd := GetQueryCmd()

	require.NotNil(t, queryCmd)
	require.Equal(t, "dex", queryCmd.Use)
	// Parent commands in Cosmos SDK typically disable flag parsing to delegate to subcommands
	require.True(t, queryCmd.DisableFlagParsing)

	// Verify subcommands exist
	subcommands := queryCmd.Commands()
	require.NotEmpty(t, subcommands)

	// Expected query subcommands (match actual command names from query.go)
	expectedCommands := []string{
		"params",
		"pool",
		"pools",
		"pool-by-tokens",
		"liquidity",
		"simulate-swap",
		"limit-order",
		"limit-orders",
		"orders-by-owner",
		"orders-by-pool",
		"order-book",
	}

	// Build map of actual commands
	commandNames := make(map[string]bool)
	for _, cmd := range subcommands {
		commandNames[cmd.Use] = true
		// Also check first word of Use for commands with args
		if len(cmd.Use) > 0 {
			firstWord := cmd.Use
			for i, c := range cmd.Use {
				if c == ' ' || c == '[' {
					firstWord = cmd.Use[:i]
					break
				}
			}
			commandNames[firstWord] = true
		}
	}

	// Verify each expected command exists
	for _, expected := range expectedCommands {
		require.True(t, commandNames[expected], "expected command %q not found", expected)
	}
}

// TestGetTxCmdStructure verifies the transaction command tree structure
func TestGetTxCmdStructure(t *testing.T) {
	t.Parallel()

	txCmd := GetTxCmd()

	require.NotNil(t, txCmd)
	require.Equal(t, "dex", txCmd.Use)

	// Verify subcommands exist
	subcommands := txCmd.Commands()
	require.NotEmpty(t, subcommands)

	// Expected tx subcommands
	expectedCommands := []string{
		"create-pool",
		"add-liquidity",
		"remove-liquidity",
		"swap",
	}

	commandNames := make(map[string]bool)
	for _, cmd := range subcommands {
		firstWord := cmd.Use
		for i, c := range cmd.Use {
			if c == ' ' || c == '[' {
				firstWord = cmd.Use[:i]
				break
			}
		}
		commandNames[firstWord] = true
	}

	for _, expected := range expectedCommands {
		require.True(t, commandNames[expected], "expected command %q not found", expected)
	}
}

// TestGetCmdQueryParamsNoArgs verifies params query requires no args
func TestGetCmdQueryParamsNoArgs(t *testing.T) {
	t.Parallel()

	cmd := GetCmdQueryParams()

	require.NotNil(t, cmd)
	require.Equal(t, "params", cmd.Use)

	// Verify the Args validator
	require.NotNil(t, cmd.Args)
	err := cmd.Args(cmd, []string{})
	require.NoError(t, err)

	err = cmd.Args(cmd, []string{"extra"})
	require.Error(t, err)
}

// TestGetCmdQueryPoolArgs verifies pool query requires exactly 1 arg
func TestGetCmdQueryPoolArgs(t *testing.T) {
	t.Parallel()

	cmd := GetCmdQueryPool()

	require.NotNil(t, cmd)
	require.Contains(t, cmd.Use, "pool")

	// Verify the Args validator
	require.NotNil(t, cmd.Args)

	err := cmd.Args(cmd, []string{"1"})
	require.NoError(t, err)

	err = cmd.Args(cmd, []string{})
	require.Error(t, err)

	err = cmd.Args(cmd, []string{"1", "2"})
	require.Error(t, err)
}

// TestGetCmdQueryPoolByTokensArgs verifies pool-by-tokens query requires 2 args
func TestGetCmdQueryPoolByTokensArgs(t *testing.T) {
	t.Parallel()

	cmd := GetCmdQueryPoolByTokens()

	require.NotNil(t, cmd)
	require.Contains(t, cmd.Use, "pool-by-tokens")

	// Verify the Args validator
	require.NotNil(t, cmd.Args)

	err := cmd.Args(cmd, []string{"tokenA", "tokenB"})
	require.NoError(t, err)

	err = cmd.Args(cmd, []string{"tokenA"})
	require.Error(t, err)

	err = cmd.Args(cmd, []string{})
	require.Error(t, err)
}

// TestGetCmdQueryLiquidityArgs verifies liquidity query requires 2 args
func TestGetCmdQueryLiquidityArgs(t *testing.T) {
	t.Parallel()

	cmd := GetCmdQueryLiquidity()

	require.NotNil(t, cmd)
	require.Contains(t, cmd.Use, "liquidity")

	require.NotNil(t, cmd.Args)

	err := cmd.Args(cmd, []string{"1", "paw1address"})
	require.NoError(t, err)

	err = cmd.Args(cmd, []string{"1"})
	require.Error(t, err)
}

// TestCommandHasDescription verifies all commands have descriptions
func TestCommandHasDescription(t *testing.T) {
	t.Parallel()

	queryCmd := GetQueryCmd()
	for _, cmd := range queryCmd.Commands() {
		require.NotEmpty(t, cmd.Short, "command %s has no short description", cmd.Use)
	}

	txCmd := GetTxCmd()
	for _, cmd := range txCmd.Commands() {
		require.NotEmpty(t, cmd.Short, "command %s has no short description", cmd.Use)
	}
}

// TestCommandHasRunE verifies all commands have RunE function
func TestCommandHasRunE(t *testing.T) {
	t.Parallel()

	var checkRunE func(*cobra.Command)
	checkRunE = func(cmd *cobra.Command) {
		if len(cmd.Commands()) > 0 {
			// Parent command - check children
			for _, child := range cmd.Commands() {
				checkRunE(child)
			}
		} else {
			// Leaf command - must have RunE
			require.NotNil(t, cmd.RunE, "command %s has no RunE function", cmd.Use)
		}
	}

	checkRunE(GetQueryCmd())
	checkRunE(GetTxCmd())
}

// TestQueryCmdFlagsAdded verifies query commands have standard flags
func TestQueryCmdFlagsAdded(t *testing.T) {
	t.Parallel()

	cmd := GetCmdQueryParams()

	// Standard query flags should be present
	flags := cmd.Flags()
	require.NotNil(t, flags)

	// Common query flags from cosmos sdk
	_, err := flags.GetString("node")
	require.NoError(t, err, "node flag should be present")

	_, err = flags.GetString("output")
	require.NoError(t, err, "output flag should be present")
}
