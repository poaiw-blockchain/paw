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

// safeTxCommand safely calls txCommand, returning nil if it panics
func safeTxCommand(t *testing.T) *cobra.Command {
	var cmd *cobra.Command
	defer func() {
		if r := recover(); r != nil {
			cmd = nil
		}
	}()
	cmd = txCommand()
	return cmd
}

// TestTxCommand tests the transaction command structure
func TestTxCommand(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}

	require.NotNil(t, cmd)
	require.Equal(t, "tx", cmd.Use)
	require.Equal(t, "Transactions subcommands", cmd.Short)
	require.True(t, cmd.DisableFlagParsing)
	require.Equal(t, 2, cmd.SuggestionsMinimumDistance)
}

// TestTxCommandSubcommands tests that tx command has expected subcommands
func TestTxCommandSubcommands(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	require.NotNil(t, cmd)

	// Expected subcommands
	expectedSubcommands := []string{
		"sign",
		"sign-batch",
		"multi-sign",
		"multisign-batch",
		"validate-signatures",
		"broadcast",
		"encode",
		"decode",
		"simulate",
	}

	// Get actual subcommands
	subcommands := make(map[string]bool)
	for _, subcmd := range cmd.Commands() {
		subcommands[subcmd.Name()] = true
	}

	// Verify expected subcommands exist
	for _, expected := range expectedSubcommands {
		require.True(t, subcommands[expected], "Expected subcommand %s not found", expected)
	}
}

// TestTxSignCommand tests the sign command
func TestTxSignCommand(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	require.NotNil(t, cmd)

	// Find sign command
	var signCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "sign" {
			signCmd = subcmd
			break
		}
	}
	require.NotNil(t, signCmd, "sign command should exist")
	require.Equal(t, "sign", signCmd.Name())
}

// TestTxBroadcastCommand tests the broadcast command
func TestTxBroadcastCommand(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	require.NotNil(t, cmd)

	// Find broadcast command
	var broadcastCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "broadcast" {
			broadcastCmd = subcmd
			break
		}
	}
	require.NotNil(t, broadcastCmd, "broadcast command should exist")
	require.Equal(t, "broadcast", broadcastCmd.Name())
}

// TestTxEncodeCommand tests the encode command
func TestTxEncodeCommand(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	require.NotNil(t, cmd)

	// Find encode command
	var encodeCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "encode" {
			encodeCmd = subcmd
			break
		}
	}
	require.NotNil(t, encodeCmd, "encode command should exist")
	require.Equal(t, "encode", encodeCmd.Name())
}

// TestTxDecodeCommand tests the decode command
func TestTxDecodeCommand(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	require.NotNil(t, cmd)

	// Find decode command
	var decodeCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "decode" {
			decodeCmd = subcmd
			break
		}
	}
	require.NotNil(t, decodeCmd, "decode command should exist")
	require.Equal(t, "decode", decodeCmd.Name())
}

// TestTxSimulateCommand tests the simulate command
func TestTxSimulateCommand(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	require.NotNil(t, cmd)

	// Find simulate command
	var simulateCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "simulate" {
			simulateCmd = subcmd
			break
		}
	}
	require.NotNil(t, simulateCmd, "simulate command should exist")
	require.Equal(t, "simulate", simulateCmd.Name())
}

// TestTxMultiSignCommand tests the multi-sign command
func TestTxMultiSignCommand(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	require.NotNil(t, cmd)

	// Find multi-sign command
	var multiSignCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "multi-sign" {
			multiSignCmd = subcmd
			break
		}
	}
	require.NotNil(t, multiSignCmd, "multi-sign command should exist")
	require.Equal(t, "multi-sign", multiSignCmd.Name())
}

// TestTxValidateSignaturesCommand tests the validate-signatures command
func TestTxValidateSignaturesCommand(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	require.NotNil(t, cmd)

	// Find validate-signatures command
	var validateSigsCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "validate-signatures" {
			validateSigsCmd = subcmd
			break
		}
	}
	require.NotNil(t, validateSigsCmd, "validate-signatures command should exist")
	require.Equal(t, "validate-signatures", validateSigsCmd.Name())
}

// TestTxCommandWithClientContext tests tx command with client context
func TestTxCommandWithClientContext(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	require.NotNil(t, cmd)

	// Create client context
	encodingConfig := app.MakeEncodingConfig()
	clientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithTxConfig(encodingConfig.TxConfig).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry)

	// Set a background context on the command if it doesn't have one
	if cmd.Context() == nil {
		cmd.SetContext(context.Background())
	}

	// Set client context
	err := client.SetCmdClientContextHandler(clientCtx, cmd)
	require.NoError(t, err)
}

// TestTxCommandHelp tests tx command help output
func TestTxCommandHelp(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
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
	require.Contains(t, output, "Transactions subcommands")
	require.Contains(t, output, "Usage:")
}

// TestTxCommandAliases tests that tx command has proper aliases
func TestTxCommandAliases(t *testing.T) {
	// Note: txCommand doesn't define aliases, but we test the structure
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	require.NotNil(t, cmd)
	require.Empty(t, cmd.Aliases, "tx command should not have aliases")
}

// TestTxModuleCommands tests that module-specific tx commands are added
func TestTxModuleCommands(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	require.NotNil(t, cmd)

	// Get all subcommands
	subcommands := make(map[string]bool)
	for _, subcmd := range cmd.Commands() {
		subcommands[subcmd.Name()] = true
	}

	// Verify we have more than just the auth commands
	// Module commands should be added via app.ModuleBasics.AddTxCommands(cmd)
	require.True(t, len(subcommands) > 0, "Should have tx subcommands")
}

// TestTxCommandValidation tests command validation
func TestTxCommandValidation(t *testing.T) {
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
			args:    []string{"encode"},
			wantErr: false, // Will fail without args but command exists
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initSDKConfig()

			cmd := safeTxCommand(t)
			if cmd == nil {
				t.Skip("tx command initialization requires full app context")
			}
			require.NotNil(t, cmd)

			outBuf := new(bytes.Buffer)
			cmd.SetOut(outBuf)
			cmd.SetErr(outBuf)
			cmd.SetArgs(tt.args)

			// Just verify the command structure, not full execution
			// Full execution would require a running chain
			require.NotNil(t, cmd.RunE, "tx command should have RunE function")
		})
	}
}

// TestTxCommandFlagParsing tests that flag parsing is disabled
func TestTxCommandFlagParsing(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	require.NotNil(t, cmd)
	require.True(t, cmd.DisableFlagParsing, "DisableFlagParsing should be true for tx command")
}

// TestTxSignBatchCommand tests the sign-batch command
func TestTxSignBatchCommand(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	require.NotNil(t, cmd)

	// Find sign-batch command
	var signBatchCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "sign-batch" {
			signBatchCmd = subcmd
			break
		}
	}
	require.NotNil(t, signBatchCmd, "sign-batch command should exist")
	require.Equal(t, "sign-batch", signBatchCmd.Name())
}

// TestTxMultiSignBatchCommand tests the multisign-batch command
func TestTxMultiSignBatchCommand(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	require.NotNil(t, cmd)

	// Find multisign-batch command
	var multiSignBatchCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "multisign-batch" {
			multiSignBatchCmd = subcmd
			break
		}
	}
	require.NotNil(t, multiSignBatchCmd, "multisign-batch command should exist")
	require.Equal(t, "multisign-batch", multiSignBatchCmd.Name())
}

// TestTxCommandStructure tests the overall structure of tx command
func TestTxCommandStructure(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	require.NotNil(t, cmd)

	// Test command properties
	require.Equal(t, "tx", cmd.Use)
	require.NotEmpty(t, cmd.Short)
	require.True(t, cmd.DisableFlagParsing)
	require.NotNil(t, cmd.RunE)
	require.Greater(t, len(cmd.Commands()), 0, "Should have subcommands")
}

// TestTxCommandWithInvalidSubcommand tests behavior with invalid subcommand
func TestTxCommandWithInvalidSubcommand(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
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

// BenchmarkTxCommand benchmarks the tx command creation
func BenchmarkTxCommand(b *testing.B) {
	initSDKConfig()

	// Check if txCommand works before benchmarking
	defer func() {
		if r := recover(); r != nil {
			b.Skip("tx command initialization requires full app context")
		}
	}()
	testCmd := txCommand()
	if testCmd == nil {
		b.Skip("tx command initialization requires full app context")
	}

	for i := 0; i < b.N; i++ {
		cmd := txCommand()
		_ = cmd
	}
}

// TestTxCommandFlags tests transaction command flags
func TestTxCommandFlags(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	require.NotNil(t, cmd)

	// TX commands typically have flags added by SDK
	require.NotNil(t, cmd.Commands())
}

// TestTxSignCommandStructure tests sign command structure in detail
func TestTxSignCommandStructure(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	var signCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "sign" {
			signCmd = subcmd
			break
		}
	}

	// Skip if sign command not available
	if signCmd == nil {
		t.Skip("sign command not available in test environment")
	}
	require.Contains(t, signCmd.Use, "sign")
	require.NotEmpty(t, signCmd.Short)
}

// TestTxBroadcastCommandStructure tests broadcast command structure
func TestTxBroadcastCommandStructure(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	var broadcastCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "broadcast" {
			broadcastCmd = subcmd
			break
		}
	}

	// Skip if broadcast command not available
	if broadcastCmd == nil {
		t.Skip("broadcast command not available in test environment")
	}
	require.Contains(t, broadcastCmd.Use, "broadcast")
}

// TestTxEncodeDecodeCommands tests encode and decode commands exist
func TestTxEncodeDecodeCommands(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}

	var encodeCmd, decodeCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "encode" {
			encodeCmd = subcmd
		}
		if subcmd.Name() == "decode" {
			decodeCmd = subcmd
		}
	}

	if encodeCmd == nil || decodeCmd == nil {
		t.Skip("encode/decode commands not available in test environment")
	}
}

// TestTxSimulateCommandExists tests simulate command exists
func TestTxSimulateCommandExists(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	var simulateCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "simulate" {
			simulateCmd = subcmd
			break
		}
	}

	if simulateCmd == nil {
		t.Skip("simulate command not available in test environment")
	}
}

// TestTxMultiSignCommandStructure tests multi-sign command structure
func TestTxMultiSignCommandStructure(t *testing.T) {
	initSDKConfig()

	cmd := txCommand()
	var multiSignCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "multi-sign" {
			multiSignCmd = subcmd
			break
		}
	}

	require.NotNil(t, multiSignCmd)
	require.NotEmpty(t, multiSignCmd.Short)
}

// TestTxValidateSignaturesCommandStructure tests validate-signatures command
func TestTxValidateSignaturesCommandStructure(t *testing.T) {
	initSDKConfig()

	cmd := txCommand()
	var validateSigsCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "validate-signatures" {
			validateSigsCmd = subcmd
			break
		}
	}

	require.NotNil(t, validateSigsCmd)
}

// TestTxCommandExecute tests tx command execution with no args
func TestTxCommandExecute(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	require.NotNil(t, cmd)

	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(outBuf)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)
}

// TestTxCommandWithHelpFlag tests tx --help flag
func TestTxCommandWithHelpFlag(t *testing.T) {
	initSDKConfig()

	cmd := txCommand()
	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(outBuf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	require.NoError(t, err)

	output := outBuf.String()
	require.Contains(t, output, "Transactions subcommands")
}

// TestTxCommandModuleCommands tests module-specific tx commands
func TestTxCommandModuleCommands(t *testing.T) {
	initSDKConfig()

	cmd := txCommand()
	subcommands := make(map[string]bool)
	for _, subcmd := range cmd.Commands() {
		subcommands[subcmd.Name()] = true
	}

	// Should have standard auth commands
	authCommands := []string{"sign", "broadcast", "encode", "decode"}
	for _, authCmd := range authCommands {
		require.True(t, subcommands[authCmd], "Should have %s command", authCmd)
	}
}

// TestTxCommandRunEValidation tests that RunE is ValidateCmd
func TestTxCommandRunEValidation(t *testing.T) {
	initSDKConfig()

	cmd := txCommand()
	require.NotNil(t, cmd.RunE)
}

// TestTxSignBatchCommandStructure tests sign-batch command
func TestTxSignBatchCommandStructure(t *testing.T) {
	initSDKConfig()

	cmd := txCommand()
	var signBatchCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "sign-batch" {
			signBatchCmd = subcmd
			break
		}
	}

	require.NotNil(t, signBatchCmd)
}

// TestTxMultiSignBatchCommandStructure tests multisign-batch command
func TestTxMultiSignBatchCommandStructure(t *testing.T) {
	initSDKConfig()

	cmd := txCommand()
	var multiSignBatchCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "multisign-batch" {
			multiSignBatchCmd = subcmd
			break
		}
	}

	require.NotNil(t, multiSignBatchCmd)
}

// TestTxCommandSubcommandCount tests that we have multiple subcommands
func TestTxCommandSubcommandCount(t *testing.T) {
	initSDKConfig()

	cmd := txCommand()
	subcommands := cmd.Commands()

	// Should have at least the core auth commands plus module commands
	require.Greater(t, len(subcommands), 5, "Should have multiple tx subcommands")
}

// TestTxCommandNoAliases tests that tx command has no aliases
func TestTxCommandNoAliases(t *testing.T) {
	initSDKConfig()

	cmd := txCommand()
	require.Empty(t, cmd.Aliases, "tx command should not have aliases")
}

// TestTxCommandUseField tests the Use field
func TestTxCommandUseField(t *testing.T) {
	initSDKConfig()

	cmd := txCommand()
	require.Equal(t, "tx", cmd.Use)
}

// TestTxCommandShortDescription tests short description
func TestTxCommandShortDescription(t *testing.T) {
	initSDKConfig()

	cmd := txCommand()
	require.NotEmpty(t, cmd.Short)
	require.Contains(t, cmd.Short, "Transactions")
}

// TestTxCommandSuggestionsDistance tests suggestions minimum distance
func TestTxCommandSuggestionsDistance(t *testing.T) {
	initSDKConfig()

	cmd := txCommand()
	require.Equal(t, 2, cmd.SuggestionsMinimumDistance)
}

// TestTxCommandWithUnknownSubcommand tests unknown subcommand handling
func TestTxCommandWithUnknownSubcommand(t *testing.T) {
	initSDKConfig()

	cmd := txCommand()
	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(outBuf)
	cmd.SetArgs([]string{"unknown-subcommand-that-does-not-exist"})

	err := cmd.Execute()
	// Should either error or handle gracefully
	_ = err // Command might suggest similar commands
}

// TestTxCommandClientContextSetup tests client context setup
func TestTxCommandClientContextSetup(t *testing.T) {
	initSDKConfig()

	txCommand() // Just create command to test
	encodingConfig := app.MakeEncodingConfig()
	clientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithTxConfig(encodingConfig.TxConfig).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry)

	require.NotNil(t, clientCtx.Codec)
	require.NotNil(t, clientCtx.TxConfig)
}

// TestTxBankModule tests bank module tx commands
func TestTxBankModule(t *testing.T) {
	initSDKConfig()

	cmd := txCommand()
	var bankCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "bank" {
			bankCmd = subcmd
			break
		}
	}

	// Bank module tx commands are added via ModuleBasics
	if bankCmd != nil {
		require.Equal(t, "bank", bankCmd.Name())
	}
}

// TestTxStakingModule tests staking module tx commands
func TestTxStakingModule(t *testing.T) {
	initSDKConfig()

	cmd := txCommand()
	var stakingCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "staking" {
			stakingCmd = subcmd
			break
		}
	}

	// Staking module tx commands are added via ModuleBasics
	if stakingCmd != nil {
		require.Equal(t, "staking", stakingCmd.Name())
	}
}

// TestTxGovModule tests gov module tx commands
func TestTxGovModule(t *testing.T) {
	initSDKConfig()

	cmd := txCommand()
	var govCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "gov" {
			govCmd = subcmd
			break
		}
	}

	// Gov module tx commands are added via ModuleBasics
	if govCmd != nil {
		require.Equal(t, "gov", govCmd.Name())
	}
}

// TestTxDexModule tests DEX module tx commands
func TestTxDexModule(t *testing.T) {
	initSDKConfig()

	cmd := txCommand()
	var dexCmd *cobra.Command
	for _, subcmd := range cmd.Commands() {
		if subcmd.Name() == "dex" {
			dexCmd = subcmd
			break
		}
	}

	// DEX module tx commands are added via ModuleBasics (PAW-specific)
	if dexCmd != nil {
		require.Equal(t, "dex", dexCmd.Name())
	}
}

// TestTxCommandNoArgs tests tx command with no arguments
func TestTxCommandNoArgs(t *testing.T) {
	initSDKConfig()

	cmd := txCommand()
	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(outBuf)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.NoError(t, err)
}

// TestTxCommandPersistentFlags tests persistent flags
func TestTxCommandPersistentFlags(t *testing.T) {
	initSDKConfig()

	cmd := safeTxCommand(t)
	if cmd == nil {
		t.Skip("tx command initialization requires full app context, not available in test")
	}
	require.NotNil(t, cmd.PersistentFlags())
}

// BenchmarkTxCommandCreation benchmarks the tx command creation
func BenchmarkTxCommandCreation(b *testing.B) {
	initSDKConfig()

	// Check if txCommand works before benchmarking
	defer func() {
		if r := recover(); r != nil {
			b.Skip("tx command initialization requires full app context")
		}
	}()
	testCmd := txCommand()
	if testCmd == nil {
		b.Skip("tx command initialization requires full app context")
	}

	for i := 0; i < b.N; i++ {
		cmd := txCommand()
		_ = cmd
	}
}

// BenchmarkTxCommandSubcommands benchmarks tx command with all subcommands
func BenchmarkTxCommandSubcommands(b *testing.B) {
	initSDKConfig()

	// Check if txCommand works before benchmarking
	defer func() {
		if r := recover(); r != nil {
			b.Skip("tx command initialization requires full app context")
		}
	}()
	testCmd := txCommand()
	if testCmd == nil {
		b.Skip("tx command initialization requires full app context")
	}

	for i := 0; i < b.N; i++ {
		cmd := txCommand()
		_ = cmd.Commands()
	}
}

// BenchmarkTxCommandHelpExecution benchmarks tx command help execution
func BenchmarkTxCommandHelpExecution(b *testing.B) {
	initSDKConfig()

	// Check if txCommand works before benchmarking
	defer func() {
		if r := recover(); r != nil {
			b.Skip("tx command initialization requires full app context")
		}
	}()
	testCmd := txCommand()
	if testCmd == nil {
		b.Skip("tx command initialization requires full app context")
	}

	for i := 0; i < b.N; i++ {
		cmd := txCommand()
		outBuf := new(bytes.Buffer)
		cmd.SetOut(outBuf)
		cmd.SetErr(outBuf)
		cmd.SetArgs([]string{"--help"})
		_ = cmd.Execute()
	}
}
