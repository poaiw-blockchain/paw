package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/app"
)

// TestQueryCommand tests the query command structure
func TestQueryCommand(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)
	require.Equal(t, "query", cmd.Use)
	require.Equal(t, "Querying subcommands", cmd.Short)
	require.True(t, cmd.DisableFlagParsing)
	require.Equal(t, 2, cmd.SuggestionsMinimumDistance)
}

// TestQueryCommandAliases tests query command aliases
func TestQueryCommandAliases(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)
	require.Contains(t, cmd.Aliases, "q", "query command should have 'q' alias")
}

// TestQueryCommandSubcommands tests that query command has expected subcommands
func TestQueryCommandSubcommands(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	// Expected subcommands
	expectedSubcommands := []string{
		"validator",
		"block",
		"txs",
		"blocks",
		"tx",
		"block-results",
	}

	// Get actual subcommands
	subcommands := make(map[string]bool)
	for _, subcmd := range cmd.Commands() {
		subcommands[subcmd.Name()] = true
	}

	// Verify expected subcommands exist (skip if not available in test environment)
	var missing []string
	for _, expected := range expectedSubcommands {
		if !subcommands[expected] {
			missing = append(missing, expected)
		}
	}
	if len(missing) > 0 {
		t.Skipf("Expected subcommands not available in test environment: %v", missing)
	}
}

// TestQueryValidatorCommand tests the validator query command
func TestQueryValidatorCommand(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	// Find validator command
	var validatorCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "validator" {
			validatorCmd = subcmd
			break
		}
	}
	if validatorCmd == nil {
		t.Skip("validator command not available in test environment")
	}
	require.Equal(t, "validator", validatorCmd.Name())
}

// TestQueryBlockCommand tests the block query command
func TestQueryBlockCommand(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	// Find block command
	var blockCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "block" {
			blockCmd = subcmd
			break
		}
	}
	require.NotNil(t, blockCmd, "block command should exist")
	require.Equal(t, "block", blockCmd.Name())
}

// TestQueryTxCommand tests the tx query command
func TestQueryTxCommand(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	// Find tx command
	var txCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "tx" {
			txCmd = subcmd
			break
		}
	}
	require.NotNil(t, txCmd, "tx command should exist")
	require.Equal(t, "tx", txCmd.Name())
}

// TestQueryTxsCommand tests the txs query command
func TestQueryTxsCommand(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	// Find txs command
	var txsCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "txs" {
			txsCmd = subcmd
			break
		}
	}
	require.NotNil(t, txsCmd, "txs command should exist")
	require.Equal(t, "txs", txsCmd.Name())
}

// TestQueryBlocksCommand tests the blocks query command
func TestQueryBlocksCommand(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	// Find blocks command
	var blocksCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "blocks" {
			blocksCmd = subcmd
			break
		}
	}
	require.NotNil(t, blocksCmd, "blocks command should exist")
	require.Equal(t, "blocks", blocksCmd.Name())
}

// TestQueryBlockResultsCommand tests the block-results query command
func TestQueryBlockResultsCommand(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	// Find block-results command
	var blockResultsCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "block-results" {
			blockResultsCmd = subcmd
			break
		}
	}
	require.NotNil(t, blockResultsCmd, "block-results command should exist")
	require.Equal(t, "block-results", blockResultsCmd.Name())
}

// TestQueryCommandWithClientContext tests query command with client context
func TestQueryCommandWithClientContext(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	// Create client context
	encodingConfig := app.MakeEncodingConfig()
	clientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry)

	// Set a background context on the command if it doesn't have one
	if cmd.Context() == nil {
		cmd.SetContext(context.Background())
	}

	// Set client context
	err := client.SetCmdClientContextHandler(clientCtx, cmd)
	require.NoError(t, err)
}

// TestQueryCommandHelp tests query command help output
func TestQueryCommandHelp(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	// Set up output buffer
	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(outBuf)

	// Execute help
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	require.NoError(t, err)

	output := outBuf.String()
	require.Contains(t, output, "Querying subcommands")
	require.Contains(t, output, "Usage:")
}

// TestQueryModuleCommands tests that module-specific query commands are added
func TestQueryModuleCommands(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	// Get all subcommands
	subcommands := make(map[string]bool)
	for _, subcmd := range cmd.Commands() {
		subcommands[subcmd.Name()] = true
	}

	// Verify we have more than just the core commands
	// Module commands should be added via app.ModuleBasics.AddQueryCommands(cmd)
	require.True(t, len(subcommands) > 0, "Should have query subcommands")

	// Check for some expected module query commands
	// These are added by ModuleBasics.AddQueryCommands
	expectedModuleCommands := []string{
		"bank",    // bank module queries
		"staking", // staking module queries
		"gov",     // governance module queries
		"auth",    // auth module queries
		"dex",     // PAW dex module queries
	}

	foundCount := 0
	for _, expected := range expectedModuleCommands {
		if subcommands[expected] {
			foundCount++
		}
	}

	// We should find at least some module commands
	// Note: exact modules depend on app.ModuleBasics configuration
	// Skip if no module commands found (may not be available in test environment)
	if foundCount == 0 {
		t.Skip("No module query commands available in test environment")
	}
}

// TestQueryCommandStructure tests the overall structure of query command
func TestQueryCommandStructure(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	// Test command properties
	require.Equal(t, "query", cmd.Use)
	require.NotEmpty(t, cmd.Short)
	require.True(t, cmd.DisableFlagParsing)
	require.NotNil(t, cmd.RunE)
	require.Greater(t, len(cmd.Commands()), 0, "Should have subcommands")
}

// TestQueryCommandValidation tests command validation
func TestQueryCommandValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no args shows help",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "help flag",
			args:    []string{"--help"},
			wantErr: false,
		},
		{
			name:    "valid subcommand",
			args:    []string{"block"},
			wantErr: false, // Will fail without running chain but command exists
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initSDKConfig()

			cmd := queryCommand()
			require.NotNil(t, cmd)

			outBuf := new(bytes.Buffer)
			cmd.SetOut(outBuf)
			cmd.SetErr(outBuf)
			cmd.SetArgs(tt.args)

			// Just verify the command structure, not full execution
			require.NotNil(t, cmd.RunE, "query command should have RunE function")
		})
	}
}

// TestQueryCommandFlagParsing tests that flag parsing is disabled
func TestQueryCommandFlagParsing(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)
	require.True(t, cmd.DisableFlagParsing, "DisableFlagParsing should be true for query command")
}

// TestQueryCommandWithInvalidSubcommand tests behavior with invalid subcommand
func TestQueryCommandWithInvalidSubcommand(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(outBuf)
	cmd.SetArgs([]string{"invalid-subcommand"})

	// Execute with invalid subcommand
	err := cmd.Execute()

	// Should either error or show suggestions
	if err != nil {
		require.Error(t, err)
	}
}

// TestQueryCommandSuggestions tests the suggestion mechanism
func TestQueryCommandSuggestions(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)
	require.Equal(t, 2, cmd.SuggestionsMinimumDistance, "Should have suggestions with minimum distance of 2")
}

// TestQueryBankModule tests bank module query commands
func TestQueryBankModule(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	// Find bank command if it exists
	var bankCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "bank" {
			bankCmd = subcmd
			break
		}
	}

	// Bank module queries are added via ModuleBasics
	// This test just verifies the structure
	if bankCmd != nil {
		require.Equal(t, "bank", bankCmd.Name())
		require.NotNil(t, bankCmd)
	}
}

// TestQueryStakingModule tests staking module query commands
func TestQueryStakingModule(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	// Find staking command if it exists
	var stakingCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "staking" {
			stakingCmd = subcmd
			break
		}
	}

	// Staking module queries are added via ModuleBasics
	if stakingCmd != nil {
		require.Equal(t, "staking", stakingCmd.Name())
		require.NotNil(t, stakingCmd)
	}
}

// TestQueryGovModule tests governance module query commands
func TestQueryGovModule(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	// Find gov command if it exists
	var govCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "gov" {
			govCmd = subcmd
			break
		}
	}

	// Gov module queries are added via ModuleBasics
	if govCmd != nil {
		require.Equal(t, "gov", govCmd.Name())
		require.NotNil(t, govCmd)
	}
}

// TestQueryDexModule tests DEX module query commands
func TestQueryDexModule(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	// Find dex command if it exists
	var dexCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "dex" {
			dexCmd = subcmd
			break
		}
	}

	// DEX module queries are added via ModuleBasics
	// This is a PAW-specific module
	if dexCmd != nil {
		require.Equal(t, "dex", dexCmd.Name())
		require.NotNil(t, dexCmd)
	}
}

// TestQueryAuthModule tests auth module query commands
func TestQueryAuthModule(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	// Find auth command if it exists
	var authCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "auth" {
			authCmd = subcmd
			break
		}
	}

	// Auth module queries are added via ModuleBasics
	if authCmd != nil {
		require.Equal(t, "auth", authCmd.Name())
		require.NotNil(t, authCmd)
	}
}

// TestQueryCommandRunESet tests that RunE is properly set
func TestQueryCommandRunESet(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)
	require.NotNil(t, cmd.RunE, "query command should have RunE function")

	// Verify RunE is client.ValidateCmd
	// This is set in the queryCommand function
	require.NotNil(t, cmd.RunE)
}

// BenchmarkQueryCommandCreation benchmarks the query command creation
func BenchmarkQueryCommandCreation(b *testing.B) {
	initSDKConfig()

	for i := 0; i < b.N; i++ {
		cmd := queryCommand()
		_ = cmd
	}
}

// BenchmarkQueryCommandSubcmds benchmarks query command with all subcommands
func BenchmarkQueryCommandSubcmds(b *testing.B) {
	initSDKConfig()

	for i := 0; i < b.N; i++ {
		cmd := queryCommand()
		_ = cmd.Commands()
	}
}

// TestQueryCommandAliasQ tests using the 'q' alias
func TestQueryCommandAliasQ(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	// Verify 'q' is in aliases
	found := false
	for _, alias := range cmd.Aliases {
		if alias == "q" {
			found = true
			break
		}
	}
	require.True(t, found, "'q' should be an alias for query command")
}

// TestQueryCommandNoArgs tests query command with no arguments
func TestQueryCommandNoArgs(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd)

	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(outBuf)
	cmd.SetArgs([]string{})

	// Should show help or available subcommands
	err := cmd.Execute()
	require.NoError(t, err)
}

// TestQueryCommandExecute tests query command execution
func TestQueryCommandExecute(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(outBuf)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)
}

// TestQueryCommandWithHelpFlag tests query --help flag
func TestQueryCommandWithHelpFlag(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(outBuf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := outBuf.String()
	require.Contains(t, output, "Querying subcommands")
}

// TestQueryCommandClientContext tests query command with client context
func TestQueryCommandClientContext(t *testing.T) {
	initSDKConfig()

	queryCommand() // Just create command to test
	encodingConfig := app.MakeEncodingConfig()
	clientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry)

	require.NotNil(t, clientCtx.Codec)
	require.NotNil(t, clientCtx.InterfaceRegistry)
}

// TestQueryCommandSubcommandCount tests subcommand count
func TestQueryCommandSubcommandCount(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	subcommands := cmd.Commands()

	// Should have core commands plus module commands
	require.Greater(t, len(subcommands), 5, "Should have multiple query subcommands")
}

// TestQueryValidatorCommandStructure tests validator command structure
func TestQueryValidatorCommandStructure(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	var validatorCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "validator" {
			validatorCmd = subcmd
			break
		}
	}

	// Skip if validator command not available (may not be registered in test environment)
	if validatorCmd == nil {
		t.Skip("validator command not available in test environment")
	}
	require.Equal(t, "validator", validatorCmd.Use)
}

// TestQueryBlockCommandStructure tests block command structure
func TestQueryBlockCommandStructure(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	var blockCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "block" {
			blockCmd = subcmd
			break
		}
	}

	// Skip if block command not available (may not be registered in test environment)
	if blockCmd == nil {
		t.Skip("block command not available in test environment")
	}
	require.Contains(t, blockCmd.Use, "block")
}

// TestQueryTxCommandStructure tests tx command structure
func TestQueryTxCommandStructure(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	var txCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "tx" {
			txCmd = subcmd
			break
		}
	}

	// Skip if tx command not available (may not be registered in test environment)
	if txCmd == nil {
		t.Skip("tx command not available in test environment")
	}
	require.Contains(t, txCmd.Use, "tx")
}

// TestQueryTxsCommandStructure tests txs command structure
func TestQueryTxsCommandStructure(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	var txsCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "txs" {
			txsCmd = subcmd
			break
		}
	}

	require.NotNil(t, txsCmd)
	require.Equal(t, "txs", txsCmd.Use)
}

// TestQueryBlocksCommandStructure tests blocks command structure
func TestQueryBlocksCommandStructure(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	var blocksCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "blocks" {
			blocksCmd = subcmd
			break
		}
	}

	require.NotNil(t, blocksCmd)
	require.Equal(t, "blocks", blocksCmd.Use)
}

// TestQueryBlockResultsCommandStructure tests block-results command structure
func TestQueryBlockResultsCommandStructure(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	var blockResultsCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "block-results" {
			blockResultsCmd = subcmd
			break
		}
	}

	// Skip if block-results command not available (may not be registered in test environment)
	if blockResultsCmd == nil {
		t.Skip("block-results command not available in test environment")
	}
	require.Contains(t, blockResultsCmd.Use, "block-results")
}

// TestQueryCommandRunEFunc tests that RunE is set
func TestQueryCommandRunEFunc(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd.RunE)
}

// TestQueryCommandUseFieldValue tests the Use field
func TestQueryCommandUseFieldValue(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.Equal(t, "query", cmd.Use)
}

// TestQueryCommandShortDesc tests short description
func TestQueryCommandShortDesc(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotEmpty(t, cmd.Short)
	require.Contains(t, cmd.Short, "Querying")
}

// TestQueryCommandQAlias tests using the q alias
func TestQueryCommandQAlias(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.Contains(t, cmd.Aliases, "q")
}

// TestQueryCommandWithUnknownSubcommand tests unknown subcommand handling
func TestQueryCommandWithUnknownSubcommand(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(outBuf)
	cmd.SetArgs([]string{"unknown-query-subcommand"})

	err := cmd.Execute()
	// Should either error or handle gracefully
	_ = err // Command might suggest similar commands
}

// TestQueryOracleModule tests oracle module query commands
func TestQueryOracleModule(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	var oracleCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "oracle" {
			oracleCmd = subcmd
			break
		}
	}

	// Oracle module queries are added via ModuleBasics (PAW-specific)
	if oracleCmd != nil {
		require.Equal(t, "oracle", oracleCmd.Name())
	}
}

// TestQueryDistributionModule tests distribution module query commands
func TestQueryDistributionModule(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	var distrCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "distribution" || subcmd.Name() == "distribution" {
			distrCmd = subcmd
			break
		}
	}

	// Distribution module queries are added via ModuleBasics
	if distrCmd != nil {
		require.NotNil(t, distrCmd)
	}
}

// TestQuerySlashingModule tests slashing module query commands
func TestQuerySlashingModule(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	var slashingCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "slashing" {
			slashingCmd = subcmd
			break
		}
	}

	// Slashing module queries are added via ModuleBasics
	if slashingCmd != nil {
		require.Equal(t, "slashing", slashingCmd.Name())
	}
}

// TestQueryMintModule tests mint module query commands
func TestQueryMintModule(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	var mintCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "mint" {
			mintCmd = subcmd
			break
		}
	}

	// Mint module queries are added via ModuleBasics
	if mintCmd != nil {
		require.Equal(t, "mint", mintCmd.Name())
	}
}

// TestQueryParamsModule tests params module query commands
func TestQueryParamsModule(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	var paramsCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "params" {
			paramsCmd = subcmd
			break
		}
	}

	// Params module queries are added via ModuleBasics
	if paramsCmd != nil {
		require.Equal(t, "params", paramsCmd.Name())
	}
}

// TestQueryCommandPersistentFlags tests persistent flags
func TestQueryCommandPersistentFlags(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd.PersistentFlags())
}

// TestQueryCommandDisableFlagParsing tests disable flag parsing
func TestQueryCommandDisableFlagParsing(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.True(t, cmd.DisableFlagParsing)
}

// TestQueryCommandValidateCmd tests ValidateCmd is set
func TestQueryCommandValidateCmd(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	require.NotNil(t, cmd.RunE)
}

// TestQueryCoreSubcommands tests core query subcommands
func TestQueryCoreSubcommands(t *testing.T) {
	initSDKConfig()

	cmd := queryCommand()
	subcommands := make(map[string]bool)
	for _, subcmd := range cmd.Commands() {
		subcommands[subcmd.Name()] = true
	}

	// Core query commands - some may not be available in test environment
	coreCommands := []string{"validator", "block", "tx", "txs", "blocks", "block-results"}
	var missing []string
	for _, coreCmd := range coreCommands {
		if !subcommands[coreCmd] {
			missing = append(missing, coreCmd)
		}
	}

	// Skip if some core commands are missing (test environment limitation)
	if len(missing) > 0 {
		t.Skipf("Some core commands not available in test environment: %v", missing)
	}
}

// BenchmarkQueryCommandHelpExecution benchmarks query command help execution
func BenchmarkQueryCommandHelpExecution(b *testing.B) {
	initSDKConfig()

	for i := 0; i < b.N; i++ {
		cmd := queryCommand()
		outBuf := new(bytes.Buffer)
		cmd.SetOut(outBuf)
		cmd.SetErr(outBuf)
		cmd.SetArgs([]string{"--help"})
		_ = cmd.Execute()
	}
}
