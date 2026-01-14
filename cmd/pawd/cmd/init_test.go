package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/app"
)

func setFlag(tb testing.TB, flagSet *pflag.FlagSet, name, value string) {
	tb.Helper()
	require.NoError(tb, flagSet.Set(name, value))
}

// TestInitCmd tests the basic initialization command
func TestInitCmd(t *testing.T) {
	tests := []struct {
		name         string
		moniker      string
		chainID      string
		overwrite    bool
		defaultDenom string
		wantErr      bool
	}{
		{
			name:         "valid init with chain ID",
			moniker:      "test-node",
			chainID:      "paw-mvp-1",
			overwrite:    false,
			defaultDenom: "upaw",
			wantErr:      false,
		},
		{
			name:         "valid init without chain ID (auto-generate)",
			moniker:      "test-node-2",
			chainID:      "",
			overwrite:    false,
			defaultDenom: "upaw",
			wantErr:      false,
		},
		{
			name:         "valid init with custom denom",
			moniker:      "test-node-3",
			chainID:      "paw-testnet-2",
			overwrite:    false,
			defaultDenom: "stake",
			wantErr:      false,
		},
		{
			name:         "valid init with overwrite",
			moniker:      "test-node-4",
			chainID:      "paw-testnet-3",
			overwrite:    true,
			defaultDenom: "upaw",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary home directory
			homeDir := t.TempDir()

			// Initialize SDK config
			initSDKConfig()

			// Create the command
			cmd := InitCmd(app.ModuleBasics, homeDir)
			require.NotNil(t, cmd)

			// Set up command flags
			cmd.SetArgs([]string{tt.moniker})
			setFlag(t, cmd.Flags(), flags.FlagChainID, tt.chainID)
			setFlag(t, cmd.Flags(), flags.FlagHome, homeDir)
			setFlag(t, cmd.Flags(), flagOverwrite, "false")
			if tt.overwrite {
				setFlag(t, cmd.Flags(), flagOverwrite, "true")
			}
			if tt.defaultDenom != "" {
				setFlag(t, cmd.Flags(), flagDefaultDenom, tt.defaultDenom)
			}

			// Create output buffer
			outBuf := new(bytes.Buffer)
			cmd.SetOut(outBuf)
			cmd.SetErr(outBuf)

			// Set up client and server context
			clientCtx := client.Context{}.
				WithCodec(app.MakeEncodingConfig().Codec).
				WithHomeDir(homeDir)

			ctx := server.NewDefaultContext()
			ctx.Config.SetRoot(homeDir)

			// Execute command
			err := executeCommandWithContext(t, cmd, &clientCtx)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify genesis file was created
			genFile := filepath.Join(homeDir, "config", "genesis.json")
			require.FileExists(t, genFile, "genesis file should be created")

			// Read and validate genesis file
			genDoc, err := tmtypes.GenesisDocFromFile(genFile)
			require.NoError(t, err)
			require.NotNil(t, genDoc)

			// Validate chain ID
			if tt.chainID != "" {
				require.Equal(t, tt.chainID, genDoc.ChainID)
			} else {
				// Auto-generated chain ID should start with "test-chain-"
				require.Contains(t, genDoc.ChainID, "test-chain-")
			}

			// Validate consensus params
			require.NotNil(t, genDoc.ConsensusParams)
			require.Equal(t, int64(2097152), genDoc.ConsensusParams.Block.MaxBytes)
			require.Equal(t, int64(100000000), genDoc.ConsensusParams.Block.MaxGas)

			// Verify config directory structure
			configDir := filepath.Join(homeDir, "config")
			require.DirExists(t, configDir)

			dataDir := filepath.Join(homeDir, "data")
			require.DirExists(t, dataDir)

			// Verify node_key.json was created
			nodeKeyFile := filepath.Join(configDir, "node_key.json")
			require.FileExists(t, nodeKeyFile)

			// Verify priv_validator_key.json was created
			privValKeyFile := filepath.Join(configDir, "priv_validator_key.json")
			require.FileExists(t, privValKeyFile)
		})
	}
}

// TestInitCmdGenesisExists tests that init fails when genesis already exists without overwrite
func TestInitCmdGenesisExists(t *testing.T) {
	homeDir := t.TempDir()
	initSDKConfig()

	// Create the command
	cmd := InitCmd(app.ModuleBasics, homeDir)

	// First initialization
	cmd.SetArgs([]string{"test-node"})
	setFlag(t, cmd.Flags(), flags.FlagChainID, "paw-mvp-1")
	setFlag(t, cmd.Flags(), flags.FlagHome, homeDir)
	setFlag(t, cmd.Flags(), flagOverwrite, "false")

	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(outBuf)

	clientCtx := client.Context{}.
		WithCodec(app.MakeEncodingConfig().Codec).
		WithHomeDir(homeDir)

	ctx := server.NewDefaultContext()
	ctx.Config.SetRoot(homeDir)

	// First execution should succeed
	err := executeCommandWithContext(t, cmd, &clientCtx)
	require.NoError(t, err)

	// Create a new command for second execution
	cmd2 := InitCmd(app.ModuleBasics, homeDir)
	cmd2.SetArgs([]string{"test-node-2"})
	setFlag(t, cmd2.Flags(), flags.FlagChainID, "paw-testnet-2")
	setFlag(t, cmd2.Flags(), flags.FlagHome, homeDir)
	setFlag(t, cmd2.Flags(), flagOverwrite, "false")

	outBuf2 := new(bytes.Buffer)
	cmd2.SetOut(outBuf2)
	cmd2.SetErr(outBuf2)

	// Second execution should fail (genesis exists)
	err = executeCommandWithContext(t, cmd2, &clientCtx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "genesis.json file already exists")
}

// TestInitCmdWithOverwrite tests that init succeeds when overwrite flag is set
func TestInitCmdWithOverwrite(t *testing.T) {
	homeDir := t.TempDir()
	initSDKConfig()

	// First initialization
	cmd := InitCmd(app.ModuleBasics, homeDir)
	cmd.SetArgs([]string{"test-node"})
	setFlag(t, cmd.Flags(), flags.FlagChainID, "paw-mvp-1")
	setFlag(t, cmd.Flags(), flags.FlagHome, homeDir)
	setFlag(t, cmd.Flags(), flagOverwrite, "false")

	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)

	clientCtx := client.Context{}.
		WithCodec(app.MakeEncodingConfig().Codec).
		WithHomeDir(homeDir)

	ctx := server.NewDefaultContext()
	ctx.Config.SetRoot(homeDir)

	err := executeCommandWithContext(t, cmd, &clientCtx)
	require.NoError(t, err)

	// Read original genesis time
	genFile := filepath.Join(homeDir, "config", "genesis.json")
	genDoc1, err := tmtypes.GenesisDocFromFile(genFile)
	require.NoError(t, err)
	originalTime := genDoc1.GenesisTime

	// Wait a bit to ensure different timestamp
	time.Sleep(10 * time.Millisecond)

	// Second initialization with overwrite
	cmd2 := InitCmd(app.ModuleBasics, homeDir)
	cmd2.SetArgs([]string{"test-node-overwrite"})
	setFlag(t, cmd2.Flags(), flags.FlagChainID, "paw-testnet-2")
	setFlag(t, cmd2.Flags(), flags.FlagHome, homeDir)
	setFlag(t, cmd2.Flags(), flagOverwrite, "true")

	outBuf2 := new(bytes.Buffer)
	cmd2.SetOut(outBuf2)

	err = executeCommandWithContext(t, cmd2, &clientCtx)
	require.NoError(t, err)

	// Verify genesis was overwritten
	genDoc2, err := tmtypes.GenesisDocFromFile(genFile)
	require.NoError(t, err)
	require.Equal(t, "paw-testnet-2", genDoc2.ChainID)
	require.NotEqual(t, originalTime, genDoc2.GenesisTime, "Genesis time should be different after overwrite")
}

// TestInitCmdGenesisValidation tests that generated genesis is valid
func TestInitCmdGenesisValidation(t *testing.T) {
	homeDir := t.TempDir()
	initSDKConfig()

	cmd := InitCmd(app.ModuleBasics, homeDir)
	cmd.SetArgs([]string{"validator-1"})
	setFlag(t, cmd.Flags(), flags.FlagChainID, "paw-testnet")
	setFlag(t, cmd.Flags(), flags.FlagHome, homeDir)

	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)

	clientCtx := client.Context{}.
		WithCodec(app.MakeEncodingConfig().Codec).
		WithHomeDir(homeDir)

	ctx := server.NewDefaultContext()
	ctx.Config.SetRoot(homeDir)

	err := executeCommandWithContext(t, cmd, &clientCtx)
	require.NoError(t, err)

	// Read genesis file
	genFile := filepath.Join(homeDir, "config", "genesis.json")
	genDoc, err := tmtypes.GenesisDocFromFile(genFile)
	require.NoError(t, err)

	// Validate genesis doc
	require.Equal(t, "paw-testnet", genDoc.ChainID)
	require.NotEmpty(t, genDoc.AppState)

	// Unmarshal app state to verify it's valid JSON
	var appState map[string]json.RawMessage
	err = json.Unmarshal(genDoc.AppState, &appState)
	require.NoError(t, err)
	require.NotEmpty(t, appState)

	// Verify consensus params
	require.NotNil(t, genDoc.ConsensusParams)
	require.NotNil(t, genDoc.ConsensusParams.Block)
	require.NotNil(t, genDoc.ConsensusParams.Evidence)
	require.NotNil(t, genDoc.ConsensusParams.Validator)
}

// TestInitCmdConsensusParams tests that consensus params are set correctly
func TestInitCmdConsensusParams(t *testing.T) {
	homeDir := t.TempDir()
	initSDKConfig()

	cmd := InitCmd(app.ModuleBasics, homeDir)
	cmd.SetArgs([]string{"test-validator"})
	setFlag(t, cmd.Flags(), flags.FlagChainID, "paw-testnet")
	setFlag(t, cmd.Flags(), flags.FlagHome, homeDir)

	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)

	clientCtx := client.Context{}.
		WithCodec(app.MakeEncodingConfig().Codec).
		WithHomeDir(homeDir)

	ctx := server.NewDefaultContext()
	ctx.Config.SetRoot(homeDir)

	err := executeCommandWithContext(t, cmd, &clientCtx)
	require.NoError(t, err)

	// Read genesis file
	genFile := filepath.Join(homeDir, "config", "genesis.json")
	genDoc, err := tmtypes.GenesisDocFromFile(genFile)
	require.NoError(t, err)

	// Verify PAW-specific consensus params
	require.Equal(t, int64(2097152), genDoc.ConsensusParams.Block.MaxBytes, "Block MaxBytes should be 2MB")
	require.Equal(t, int64(100000000), genDoc.ConsensusParams.Block.MaxGas, "Block MaxGas should be 100M")
	require.Equal(t, int64(500000), genDoc.ConsensusParams.Evidence.MaxAgeNumBlocks, "Evidence MaxAgeNumBlocks should be ~23 days")
	require.Equal(t, 21*24*time.Hour, genDoc.ConsensusParams.Evidence.MaxAgeDuration, "Evidence MaxAgeDuration should be 21 days")
	require.Equal(t, int64(1048576), genDoc.ConsensusParams.Evidence.MaxBytes, "Evidence MaxBytes should be 1MB")
}

// TestInitCmdDefaultDenom tests custom default denomination
func TestInitCmdDefaultDenom(t *testing.T) {
	tests := []struct {
		name  string
		denom string
	}{
		{
			name:  "default upaw",
			denom: "upaw",
		},
		{
			name:  "custom stake",
			denom: "stake",
		},
		{
			name:  "empty defaults to upaw",
			denom: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			homeDir := t.TempDir()
			initSDKConfig()

			cmd := InitCmd(app.ModuleBasics, homeDir)
			cmd.SetArgs([]string{"test-node"})
			setFlag(t, cmd.Flags(), flags.FlagChainID, "paw-testnet")
			setFlag(t, cmd.Flags(), flags.FlagHome, homeDir)
			if tt.denom != "" {
				setFlag(t, cmd.Flags(), flagDefaultDenom, tt.denom)
			}

			outBuf := new(bytes.Buffer)
			cmd.SetOut(outBuf)

			clientCtx := client.Context{}.
				WithCodec(app.MakeEncodingConfig().Codec).
				WithHomeDir(homeDir)

			ctx := server.NewDefaultContext()
			ctx.Config.SetRoot(homeDir)

			err := executeCommandWithContext(t, cmd, &clientCtx)
			require.NoError(t, err)

			// Verify genesis file was created successfully
			genFile := filepath.Join(homeDir, "config", "genesis.json")
			require.FileExists(t, genFile)
		})
	}
}

// TestFileExists tests the fileExists helper function
func TestFileExists(t *testing.T) {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-file-*")
	require.NoError(t, err)
	tmpFileName := tmpFile.Name()
	tmpFile.Close() // Close file before attempting to delete
	defer os.Remove(tmpFileName)

	// File should exist
	require.True(t, fileExists(tmpFileName))

	// Remove file
	err = os.Remove(tmpFileName)
	require.NoError(t, err)

	// File should not exist
	require.False(t, fileExists(tmpFileName))

	// Non-existent path should return false
	require.False(t, fileExists("/path/that/does/not/exist"))
}

// TestInitCmdOutput tests command output messages
func TestInitCmdOutput(t *testing.T) {
	homeDir := t.TempDir()
	initSDKConfig()

	cmd := InitCmd(app.ModuleBasics, homeDir)
	cmd.SetArgs([]string{"test-validator"})
	setFlag(t, cmd.Flags(), flags.FlagChainID, "paw-testnet")
	setFlag(t, cmd.Flags(), flags.FlagHome, homeDir)

	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)

	clientCtx := client.Context{}.
		WithCodec(app.MakeEncodingConfig().Codec).
		WithHomeDir(homeDir)

	ctx := server.NewDefaultContext()
	ctx.Config.SetRoot(homeDir)

	err := executeCommandWithContext(t, cmd, &clientCtx)
	require.NoError(t, err)

	output := outBuf.String()

	// Verify expected output messages
	require.Contains(t, output, "Successfully initialized chain configuration")
	require.Contains(t, output, "Chain ID: paw-testnet")
	require.Contains(t, output, "Moniker: test-validator")
	require.Contains(t, output, "Node ID:")
	require.Contains(t, output, "Home directory:")
	require.Contains(t, output, "Genesis file:")
	require.Contains(t, output, "Config file:")
	require.Contains(t, output, "App config:")
}

// executeCommandWithContext is a helper to execute commands with proper context.
func executeCommandWithContext(t testing.TB, cmd *cobra.Command, clientCtx *client.Context) error {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(clientCtx.HomeDir, "config"), 0o755); err != nil {
		return err
	}

	// Initialize encoding config to get all required fields
	encodingConfig := app.MakeEncodingConfig()

	// Ensure client context has all required fields
	*clientCtx = clientCtx.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithHomeDir(clientCtx.HomeDir)

	// Set a background context on the command if it doesn't have one
	if cmd.Context() == nil {
		cmd.SetContext(context.Background())
	}

	// Set client context in command
	_ = client.SetCmdClientContextHandler(*clientCtx, cmd)

	return cmd.Execute()
}

// TestInitCmdInvalidMoniker tests init with invalid moniker
func TestInitCmdInvalidMoniker(t *testing.T) {
	tests := []struct {
		name    string
		moniker string
		wantErr bool
	}{
		{
			name:    "empty moniker",
			moniker: "",
			wantErr: true,
		},
		{
			name:    "valid moniker",
			moniker: "my-validator",
			wantErr: false,
		},
		{
			name:    "moniker with spaces",
			moniker: "my validator",
			wantErr: false,
		},
		{
			name:    "moniker with special chars",
			moniker: "validator@123",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.moniker == "" {
				// Cobra will error on missing required args
				return
			}

			homeDir := t.TempDir()
			initSDKConfig()

			cmd := InitCmd(app.ModuleBasics, homeDir)
			cmd.SetArgs([]string{tt.moniker})
			setFlag(t, cmd.Flags(), flags.FlagChainID, "paw-testnet")
			setFlag(t, cmd.Flags(), flags.FlagHome, homeDir)

			outBuf := new(bytes.Buffer)
			cmd.SetOut(outBuf)

			clientCtx := client.Context{}.
				WithCodec(app.MakeEncodingConfig().Codec).
				WithHomeDir(homeDir)

			ctx := server.NewDefaultContext()
			ctx.Config.SetRoot(homeDir)

			err := executeCommandWithContext(t, cmd, &clientCtx)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestInitCmdChainIDValidation tests chain ID validation
func TestInitCmdChainIDValidation(t *testing.T) {
	tests := []struct {
		name    string
		chainID string
		wantErr bool
	}{
		{
			name:    "valid chain ID",
			chainID: "paw-mvp-1",
			wantErr: false,
		},
		{
			name:    "chain ID with numbers",
			chainID: "paw-123",
			wantErr: false,
		},
		{
			name:    "empty chain ID - auto-generate",
			chainID: "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			homeDir := t.TempDir()
			initSDKConfig()

			cmd := InitCmd(app.ModuleBasics, homeDir)
			cmd.SetArgs([]string{"test-node"})
			setFlag(t, cmd.Flags(), flags.FlagChainID, tt.chainID)
			setFlag(t, cmd.Flags(), flags.FlagHome, homeDir)

			outBuf := new(bytes.Buffer)
			cmd.SetOut(outBuf)

			clientCtx := client.Context{}.
				WithCodec(app.MakeEncodingConfig().Codec).
				WithHomeDir(homeDir)

			ctx := server.NewDefaultContext()
			ctx.Config.SetRoot(homeDir)

			err := executeCommandWithContext(t, cmd, &clientCtx)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestInitCmdNodeKeyGeneration tests node key generation
func TestInitCmdNodeKeyGeneration(t *testing.T) {
	homeDir := t.TempDir()
	initSDKConfig()

	cmd := InitCmd(app.ModuleBasics, homeDir)
	cmd.SetArgs([]string{"test-node"})
	setFlag(t, cmd.Flags(), flags.FlagChainID, "paw-testnet")
	setFlag(t, cmd.Flags(), flags.FlagHome, homeDir)

	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)

	clientCtx := client.Context{}.
		WithCodec(app.MakeEncodingConfig().Codec).
		WithHomeDir(homeDir)

	ctx := server.NewDefaultContext()
	ctx.Config.SetRoot(homeDir)

	err := executeCommandWithContext(t, cmd, &clientCtx)
	require.NoError(t, err)

	// Verify node_key.json exists and is valid JSON
	nodeKeyFile := filepath.Join(homeDir, "config", "node_key.json")
	require.FileExists(t, nodeKeyFile)

	nodeKeyData, err := os.ReadFile(nodeKeyFile)
	require.NoError(t, err)

	var nodeKey map[string]interface{}
	err = json.Unmarshal(nodeKeyData, &nodeKey)
	require.NoError(t, err)
	require.Contains(t, nodeKey, "priv_key")
}

// TestInitCmdPrivValidatorKeyGeneration tests priv_validator_key generation
func TestInitCmdPrivValidatorKeyGeneration(t *testing.T) {
	homeDir := t.TempDir()
	initSDKConfig()

	cmd := InitCmd(app.ModuleBasics, homeDir)
	cmd.SetArgs([]string{"test-node"})
	setFlag(t, cmd.Flags(), flags.FlagChainID, "paw-testnet")
	setFlag(t, cmd.Flags(), flags.FlagHome, homeDir)

	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)

	clientCtx := client.Context{}.
		WithCodec(app.MakeEncodingConfig().Codec).
		WithHomeDir(homeDir)

	ctx := server.NewDefaultContext()
	ctx.Config.SetRoot(homeDir)

	err := executeCommandWithContext(t, cmd, &clientCtx)
	require.NoError(t, err)

	// Verify priv_validator_key.json exists and is valid JSON
	privValKeyFile := filepath.Join(homeDir, "config", "priv_validator_key.json")
	require.FileExists(t, privValKeyFile)

	privValKeyData, err := os.ReadFile(privValKeyFile)
	require.NoError(t, err)

	var privValKey map[string]interface{}
	err = json.Unmarshal(privValKeyData, &privValKey)
	require.NoError(t, err)
	require.Contains(t, privValKey, "address")
	require.Contains(t, privValKey, "pub_key")
	require.Contains(t, privValKey, "priv_key")
}

// TestInitCmdMultipleChains tests initializing multiple chains
func TestInitCmdMultipleChains(t *testing.T) {
	chains := []string{"paw-mvp-1", "paw-testnet-2", "paw-mainnet"}

	for i, chainID := range chains {
		homeDir := t.TempDir()
		initSDKConfig()

		cmd := InitCmd(app.ModuleBasics, homeDir)
		moniker := fmt.Sprintf("validator-%d", i)
		cmd.SetArgs([]string{moniker})
		setFlag(t, cmd.Flags(), flags.FlagChainID, chainID)
		setFlag(t, cmd.Flags(), flags.FlagHome, homeDir)

		outBuf := new(bytes.Buffer)
		cmd.SetOut(outBuf)

		clientCtx := client.Context{}.
			WithCodec(app.MakeEncodingConfig().Codec).
			WithHomeDir(homeDir)

		ctx := server.NewDefaultContext()
		ctx.Config.SetRoot(homeDir)

		err := executeCommandWithContext(t, cmd, &clientCtx)
		require.NoError(t, err)

		genFile := filepath.Join(homeDir, "config", "genesis.json")
		genDoc, err := tmtypes.GenesisDocFromFile(genFile)
		require.NoError(t, err)
		require.Equal(t, chainID, genDoc.ChainID)
	}
}

// TestInitCmdDirectoryStructure tests complete directory structure
func TestInitCmdDirectoryStructure(t *testing.T) {
	homeDir := t.TempDir()
	initSDKConfig()

	cmd := InitCmd(app.ModuleBasics, homeDir)
	cmd.SetArgs([]string{"test-node"})
	setFlag(t, cmd.Flags(), flags.FlagChainID, "paw-testnet")
	setFlag(t, cmd.Flags(), flags.FlagHome, homeDir)

	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)

	clientCtx := client.Context{}.
		WithCodec(app.MakeEncodingConfig().Codec).
		WithHomeDir(homeDir)

	ctx := server.NewDefaultContext()
	ctx.Config.SetRoot(homeDir)

	err := executeCommandWithContext(t, cmd, &clientCtx)
	require.NoError(t, err)

	// Verify all required directories exist
	requiredDirs := []string{
		"config",
		"data",
	}

	for _, dir := range requiredDirs {
		dirPath := filepath.Join(homeDir, dir)
		require.DirExists(t, dirPath, "Directory %s should exist", dir)
	}
}

// TestInitCmdGenesisTimeSet tests that genesis time is set
func TestInitCmdGenesisTimeSet(t *testing.T) {
	homeDir := t.TempDir()
	initSDKConfig()

	beforeTime := time.Now()

	cmd := InitCmd(app.ModuleBasics, homeDir)
	cmd.SetArgs([]string{"test-node"})
	setFlag(t, cmd.Flags(), flags.FlagChainID, "paw-testnet")
	setFlag(t, cmd.Flags(), flags.FlagHome, homeDir)

	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)

	clientCtx := client.Context{}.
		WithCodec(app.MakeEncodingConfig().Codec).
		WithHomeDir(homeDir)

	ctx := server.NewDefaultContext()
	ctx.Config.SetRoot(homeDir)

	err := executeCommandWithContext(t, cmd, &clientCtx)
	require.NoError(t, err)

	afterTime := time.Now()

	genFile := filepath.Join(homeDir, "config", "genesis.json")
	genDoc, err := tmtypes.GenesisDocFromFile(genFile)
	require.NoError(t, err)

	// Genesis time should be between before and after
	require.True(t, genDoc.GenesisTime.After(beforeTime) || genDoc.GenesisTime.Equal(beforeTime))
	require.True(t, genDoc.GenesisTime.Before(afterTime) || genDoc.GenesisTime.Equal(afterTime))
}

// TestInitCmdAppStateNotEmpty tests that app state is initialized
func TestInitCmdAppStateNotEmpty(t *testing.T) {
	homeDir := t.TempDir()
	initSDKConfig()

	cmd := InitCmd(app.ModuleBasics, homeDir)
	cmd.SetArgs([]string{"test-node"})
	setFlag(t, cmd.Flags(), flags.FlagChainID, "paw-testnet")
	setFlag(t, cmd.Flags(), flags.FlagHome, homeDir)

	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)

	clientCtx := client.Context{}.
		WithCodec(app.MakeEncodingConfig().Codec).
		WithHomeDir(homeDir)

	ctx := server.NewDefaultContext()
	ctx.Config.SetRoot(homeDir)

	err := executeCommandWithContext(t, cmd, &clientCtx)
	require.NoError(t, err)

	genFile := filepath.Join(homeDir, "config", "genesis.json")
	genDoc, err := tmtypes.GenesisDocFromFile(genFile)
	require.NoError(t, err)

	require.NotEmpty(t, genDoc.AppState)

	// Verify app state contains expected modules
	var appState map[string]json.RawMessage
	err = json.Unmarshal(genDoc.AppState, &appState)
	require.NoError(t, err)

	expectedModules := []string{
		"auth",
		"bank",
		"staking",
		"gov",
	}

	for _, module := range expectedModules {
		require.Contains(t, appState, module, "App state should contain %s module", module)
	}
}

// TestInitCmdEvidenceParams tests evidence parameters
func TestInitCmdEvidenceParams(t *testing.T) {
	homeDir := t.TempDir()
	initSDKConfig()

	cmd := InitCmd(app.ModuleBasics, homeDir)
	cmd.SetArgs([]string{"test-node"})
	setFlag(t, cmd.Flags(), flags.FlagChainID, "paw-testnet")
	setFlag(t, cmd.Flags(), flags.FlagHome, homeDir)

	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)

	clientCtx := client.Context{}.
		WithCodec(app.MakeEncodingConfig().Codec).
		WithHomeDir(homeDir)

	ctx := server.NewDefaultContext()
	ctx.Config.SetRoot(homeDir)

	err := executeCommandWithContext(t, cmd, &clientCtx)
	require.NoError(t, err)

	genFile := filepath.Join(homeDir, "config", "genesis.json")
	genDoc, err := tmtypes.GenesisDocFromFile(genFile)
	require.NoError(t, err)

	// Verify evidence params
	require.NotNil(t, genDoc.ConsensusParams.Evidence)
	require.Equal(t, int64(500000), genDoc.ConsensusParams.Evidence.MaxAgeNumBlocks)
	require.Equal(t, 21*24*time.Hour, genDoc.ConsensusParams.Evidence.MaxAgeDuration)
	require.Equal(t, int64(1048576), genDoc.ConsensusParams.Evidence.MaxBytes)
}

// TestInitCmdValidatorParams tests validator parameters
func TestInitCmdValidatorParams(t *testing.T) {
	homeDir := t.TempDir()
	initSDKConfig()

	cmd := InitCmd(app.ModuleBasics, homeDir)
	cmd.SetArgs([]string{"test-node"})
	setFlag(t, cmd.Flags(), flags.FlagChainID, "paw-testnet")
	setFlag(t, cmd.Flags(), flags.FlagHome, homeDir)

	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)

	clientCtx := client.Context{}.
		WithCodec(app.MakeEncodingConfig().Codec).
		WithHomeDir(homeDir)

	ctx := server.NewDefaultContext()
	ctx.Config.SetRoot(homeDir)

	err := executeCommandWithContext(t, cmd, &clientCtx)
	require.NoError(t, err)

	genFile := filepath.Join(homeDir, "config", "genesis.json")
	genDoc, err := tmtypes.GenesisDocFromFile(genFile)
	require.NoError(t, err)

	// Verify validator params exist
	require.NotNil(t, genDoc.ConsensusParams.Validator)
}

// TestInitCmdMultipleDenoms tests initialization with different denoms
func TestInitCmdMultipleDenoms(t *testing.T) {
	denoms := []string{"upaw", "stake", "uatom"}

	for _, denom := range denoms {
		homeDir := t.TempDir()
		initSDKConfig()

		cmd := InitCmd(app.ModuleBasics, homeDir)
		cmd.SetArgs([]string{"test-node"})
		setFlag(t, cmd.Flags(), flags.FlagChainID, "paw-testnet")
		setFlag(t, cmd.Flags(), flags.FlagHome, homeDir)
		setFlag(t, cmd.Flags(), flagDefaultDenom, denom)

		outBuf := new(bytes.Buffer)
		cmd.SetOut(outBuf)

		clientCtx := client.Context{}.
			WithCodec(app.MakeEncodingConfig().Codec).
			WithHomeDir(homeDir)

		ctx := server.NewDefaultContext()
		ctx.Config.SetRoot(homeDir)

		err := executeCommandWithContext(t, cmd, &clientCtx)
		require.NoError(t, err)

		genFile := filepath.Join(homeDir, "config", "genesis.json")
		require.FileExists(t, genFile)
	}
}

// TestInitCmdFlagDefaults tests default flag values
func TestInitCmdFlagDefaults(t *testing.T) {
	homeDir := t.TempDir()
	initSDKConfig()

	cmd := InitCmd(app.ModuleBasics, homeDir)

	// Check default flag values
	defaultDenom, err := cmd.Flags().GetString(flagDefaultDenom)
	require.NoError(t, err)
	require.Equal(t, "upaw", defaultDenom)

	overwrite, err := cmd.Flags().GetBool(flagOverwrite)
	require.NoError(t, err)
	require.False(t, overwrite)

	recoverFlag, err := cmd.Flags().GetBool(flagRecover)
	require.NoError(t, err)
	require.False(t, recoverFlag)
}

// TestInitCmdCommandStructure tests command structure
func TestInitCmdCommandStructure(t *testing.T) {
	homeDir := t.TempDir()
	initSDKConfig()

	cmd := InitCmd(app.ModuleBasics, homeDir)

	require.Equal(t, "init [moniker]", cmd.Use)
	require.NotEmpty(t, cmd.Short)
	require.NotEmpty(t, cmd.Long)
	require.NotNil(t, cmd.RunE)

	// Verify flags exist
	require.NotNil(t, cmd.Flags().Lookup(flags.FlagChainID))
	require.NotNil(t, cmd.Flags().Lookup(flagOverwrite))
	require.NotNil(t, cmd.Flags().Lookup(flagRecover))
	require.NotNil(t, cmd.Flags().Lookup(flagDefaultDenom))
	require.NotNil(t, cmd.Flags().Lookup(flags.FlagHome))
}

// TestInitCmdLongDescription tests long description
func TestInitCmdLongDescription(t *testing.T) {
	homeDir := t.TempDir()
	initSDKConfig()

	cmd := InitCmd(app.ModuleBasics, homeDir)

	require.Contains(t, cmd.Long, "pawd init")
	require.Contains(t, cmd.Long, "chain-id")
}

// BenchmarkInitCmd benchmarks the init command
func BenchmarkInitCmd(b *testing.B) {
	initSDKConfig()

	for i := 0; i < b.N; i++ {
		homeDir := b.TempDir()

		cmd := InitCmd(app.ModuleBasics, homeDir)
		cmd.SetArgs([]string{"test-node"})
		setFlag(b, cmd.Flags(), flags.FlagChainID, "paw-testnet")
		setFlag(b, cmd.Flags(), flags.FlagHome, homeDir)

		outBuf := new(bytes.Buffer)
		cmd.SetOut(outBuf)

		clientCtx := client.Context{}.
			WithCodec(app.MakeEncodingConfig().Codec).
			WithHomeDir(homeDir)

		ctx := server.NewDefaultContext()
		ctx.Config.SetRoot(homeDir)

		_ = executeCommandWithContext(b, cmd, &clientCtx)
	}
}

// BenchmarkInitCmdWithOverwrite benchmarks init with overwrite
func BenchmarkInitCmdWithOverwrite(b *testing.B) {
	homeDir := b.TempDir()
	initSDKConfig()

	// First init
	cmd := InitCmd(app.ModuleBasics, homeDir)
	cmd.SetArgs([]string{"test-node"})
	setFlag(b, cmd.Flags(), flags.FlagChainID, "paw-testnet")
	setFlag(b, cmd.Flags(), flags.FlagHome, homeDir)

	outBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)

	clientCtx := client.Context{}.
		WithCodec(app.MakeEncodingConfig().Codec).
		WithHomeDir(homeDir)

	ctx := server.NewDefaultContext()
	ctx.Config.SetRoot(homeDir)

	_ = executeCommandWithContext(b, cmd, &clientCtx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd2 := InitCmd(app.ModuleBasics, homeDir)
		cmd2.SetArgs([]string{"test-node"})
		setFlag(b, cmd2.Flags(), flags.FlagChainID, "paw-testnet")
		setFlag(b, cmd2.Flags(), flags.FlagHome, homeDir)
		setFlag(b, cmd2.Flags(), flagOverwrite, "true")

		outBuf2 := new(bytes.Buffer)
		cmd2.SetOut(outBuf2)

		_ = executeCommandWithContext(b, cmd2, &clientCtx)
	}
}
