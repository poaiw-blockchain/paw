package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/go-bip39"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/app"
	"github.com/paw-chain/paw/cmd/pawd/cmd"
	computetypes "github.com/paw-chain/paw/x/compute/types"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// CLIIntegrationTestSuite provides a comprehensive test suite for all CLI commands
type CLIIntegrationTestSuite struct {
	suite.Suite
	homeDir      string
	chainID      string
	keyring      keyring.Keyring
	clientCtx    client.Context
	testAccounts []testAccount
}

type testAccount struct {
	name     string
	address  sdk.AccAddress
	valAddr  sdk.ValAddress
	mnemonic string
}

// SetupSuite runs once before all tests
func (s *CLIIntegrationTestSuite) SetupSuite() {
	s.chainID = "paw-test-chain"

	// Create temporary home directory
	var err error
	s.homeDir = s.T().TempDir()
	require.NoError(s.T(), err)

	// Initialize SDK config (only if not already sealed)
	cfg := sdk.GetConfig()
	defer func() {
		if r := recover(); r != nil {
			// Config was already sealed, which is fine for tests
		}
	}()
	cfg.SetBech32PrefixForAccount(app.Bech32PrefixAccAddr, app.Bech32PrefixAccPub)
	cfg.SetBech32PrefixForValidator(app.Bech32PrefixValAddr, app.Bech32PrefixValPub)
	cfg.SetBech32PrefixForConsensusNode(app.Bech32PrefixConsAddr, app.Bech32PrefixConsPub)

	// Create keyring in test directory
	kr, err := keyring.New(sdk.KeyringServiceName(), keyring.BackendTest, s.homeDir, nil, app.MakeEncodingConfig().Codec)
	require.NoError(s.T(), err)
	s.keyring = kr

	// Create test accounts
	s.testAccounts = make([]testAccount, 3)
	for i := 0; i < 3; i++ {
		acct := s.createTestAccount(fmt.Sprintf("testkey%d", i+1))
		s.testAccounts[i] = acct
	}

	// Setup client context
	encodingConfig := app.MakeEncodingConfig()
	s.clientCtx = client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithKeyring(s.keyring).
		WithHomeDir(s.homeDir).
		WithChainID(s.chainID).
		WithViper("").
		WithOffline(true)
}

// TearDownSuite runs once after all tests
func (s *CLIIntegrationTestSuite) TearDownSuite() {
	// Cleanup is automatic via TempDir
}

// createTestAccount creates a test account with mnemonic
func (s *CLIIntegrationTestSuite) createTestAccount(name string) testAccount {
	// Generate mnemonic
	entropy, err := bip39.NewEntropy(256)
	require.NoError(s.T(), err)
	mnemonic, err := bip39.NewMnemonic(entropy)
	require.NoError(s.T(), err)

	// Derive key
	hdPath := hd.CreateHDPath(sdk.CoinType, 0, 0).String()
	info, err := s.keyring.NewAccount(name, mnemonic, "", hdPath, hd.Secp256k1)
	require.NoError(s.T(), err)

	addr, err := info.GetAddress()
	require.NoError(s.T(), err)

	valAddr := sdk.ValAddress(addr)

	return testAccount{
		name:     name,
		address:  addr,
		valAddr:  valAddr,
		mnemonic: mnemonic,
	}
}

// setFlag is a helper to set command flags
func setFlag(t *testing.T, flagSet *pflag.FlagSet, name, value string) {
	t.Helper()
	require.NoError(t, flagSet.Set(name, value))
}

func execCmd(cmd *cobra.Command) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	return cmd.ExecuteContext(ctx)
}

func (s *CLIIntegrationTestSuite) attachCmdContexts(cmd *cobra.Command, homeDir string, chainID string) {
	s.T().Helper()

	clientCtx := s.clientCtx.WithHomeDir(homeDir).WithKeyring(s.keyring)
	if clientCtx.Codec == nil || clientCtx.InterfaceRegistry == nil || clientCtx.TxConfig == nil {
		enc := app.MakeEncodingConfig()
		clientCtx = clientCtx.
			WithCodec(enc.Codec).
			WithInterfaceRegistry(enc.InterfaceRegistry).
			WithTxConfig(enc.TxConfig).
			WithLegacyAmino(enc.Amino)
	}
	if chainID != "" {
		clientCtx = clientCtx.WithChainID(chainID)
	}

	require.NoError(s.T(), client.SetCmdClientContext(cmd, clientCtx))
	serverCtx := server.NewDefaultContext()
	serverCtx.Config.SetRoot(homeDir)
	require.NoError(s.T(), server.SetCmdServerContext(cmd, serverCtx))
}

func (s *CLIIntegrationTestSuite) ensureTestAccounts() {
	if s.keyring == nil {
		kr, err := keyring.New(sdk.KeyringServiceName(), keyring.BackendTest, s.homeDir, nil, app.MakeEncodingConfig().Codec)
		require.NoError(s.T(), err)
		s.keyring = kr
	}

	if len(s.testAccounts) > 0 {
		return
	}

	s.testAccounts = make([]testAccount, 3)
	for i := 0; i < 3; i++ {
		acct := s.createTestAccount(fmt.Sprintf("testkey%d", i+1))
		s.testAccounts[i] = acct
	}
}

// executeCommand executes a command and returns the output
func (s *CLIIntegrationTestSuite) executeCommand(cmd *cobra.Command, args ...string) (string, error) {
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)
	cmd.SetArgs(args)

	s.attachCmdContexts(cmd, s.homeDir, s.chainID)

	// Add required flags
	if cmd.Flags().Lookup(flags.FlagHome) != nil {
		setFlag(s.T(), cmd.Flags(), flags.FlagHome, s.homeDir)
	}
	if cmd.Flags().Lookup(flags.FlagChainID) != nil {
		setFlag(s.T(), cmd.Flags(), flags.FlagChainID, s.chainID)
	}

	execCtx := cmd.Context()
	if execCtx == nil {
		execCtx = context.Background()
	}
	err := cmd.ExecuteContext(execCtx)

	if err != nil {
		return errBuf.String(), err
	}

	return outBuf.String(), nil
}

// ============================================================================
// Init Command Tests
// ============================================================================

func (s *CLIIntegrationTestSuite) TestInitCmd() {
	tests := []struct {
		name         string
		moniker      string
		chainID      string
		overwrite    bool
		defaultDenom string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "valid init with chain ID",
			moniker:      "test-node",
			chainID:      "paw-testnet-1",
			overwrite:    false,
			defaultDenom: "upaw",
			wantErr:      false,
		},
		{
			name:         "valid init without chain ID (auto-generate)",
			moniker:      "test-node-auto",
			chainID:      "",
			overwrite:    false,
			defaultDenom: "upaw",
			wantErr:      false,
		},
		{
			name:         "valid init with custom denom",
			moniker:      "test-node-custom",
			chainID:      "paw-testnet-2",
			overwrite:    false,
			defaultDenom: "stake",
			wantErr:      false,
		},
		{
			name:         "valid init with overwrite",
			moniker:      "test-node-overwrite",
			chainID:      "paw-testnet-3",
			overwrite:    true,
			defaultDenom: "upaw",
			wantErr:      false,
		},
		{
			name:         "empty moniker should fail",
			moniker:      "",
			chainID:      "paw-testnet-4",
			overwrite:    false,
			defaultDenom: "upaw",
			wantErr:      true,
			errContains:  "moniker",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Create unique temp directory for each test
			testHomeDir := s.T().TempDir()

			cmdInit := cmd.InitCmd(app.ModuleBasics, testHomeDir)
			require.NotNil(s.T(), cmdInit)

			// Set up flags
			cmdInit.SetArgs([]string{tt.moniker})
			if tt.chainID != "" {
				setFlag(s.T(), cmdInit.Flags(), flags.FlagChainID, tt.chainID)
			}
			setFlag(s.T(), cmdInit.Flags(), flags.FlagHome, testHomeDir)
			if tt.overwrite {
				setFlag(s.T(), cmdInit.Flags(), "overwrite", "true")
			}
			if tt.defaultDenom != "" {
				setFlag(s.T(), cmdInit.Flags(), "default-denom", tt.defaultDenom)
			}

			outBuf := new(bytes.Buffer)
			cmdInit.SetOut(outBuf)
			cmdInit.SetErr(outBuf)

			s.attachCmdContexts(cmdInit, testHomeDir, tt.chainID)

			err := execCmd(cmdInit)

			if tt.wantErr {
				require.Error(s.T(), err)
				if tt.errContains != "" {
					require.Contains(s.T(), err.Error(), tt.errContains)
				}
			} else {
				require.NoError(s.T(), err)

				// Verify genesis file was created
				genesisFile := filepath.Join(testHomeDir, "config", "genesis.json")
				require.FileExists(s.T(), genesisFile)

				// Verify node key was created
				nodeKeyFile := filepath.Join(testHomeDir, "config", "node_key.json")
				require.FileExists(s.T(), nodeKeyFile)

				// Verify priv_validator_key was created
				privValKeyFile := filepath.Join(testHomeDir, "config", "priv_validator_key.json")
				require.FileExists(s.T(), privValKeyFile)

				// Parse and validate genesis
				genesisData, err := os.ReadFile(genesisFile)
				require.NoError(s.T(), err)

				genDoc, err := tmtypes.GenesisDocFromJSON(genesisData)
				require.NoError(s.T(), err)

				if tt.chainID != "" {
					require.Equal(s.T(), tt.chainID, genDoc.ChainID)
				} else {
					require.NotEmpty(s.T(), genDoc.ChainID)
				}
			}
		})
	}
}

func (s *CLIIntegrationTestSuite) TestCollectGenTxsCmd() {
	// Initialize a node first
	testHomeDir := s.T().TempDir()
	cmdInit := cmd.InitCmd(app.ModuleBasics, testHomeDir)
	cmdInit.SetArgs([]string{"validator1"})
	setFlag(s.T(), cmdInit.Flags(), flags.FlagChainID, "paw-test")
	setFlag(s.T(), cmdInit.Flags(), flags.FlagHome, testHomeDir)

	outBuf := new(bytes.Buffer)
	cmdInit.SetOut(outBuf)
	cmdInit.SetErr(outBuf)

	s.attachCmdContexts(cmdInit, testHomeDir, "paw-test")

	err := execCmd(cmdInit)
	require.NoError(s.T(), err)

	// Test collect-gentxs command
	cmdCollect := cmd.CollectGenTxsCmd(app.ModuleBasics, testHomeDir, banktypes.GenesisBalancesIterator{}, nil)
	cmdCollect.SetArgs([]string{})
	setFlag(s.T(), cmdCollect.Flags(), flags.FlagHome, testHomeDir)

	outBuf.Reset()
	cmdCollect.SetOut(outBuf)
	cmdCollect.SetErr(outBuf)

	s.attachCmdContexts(cmdCollect, testHomeDir, "paw-test")

	// This will succeed even with no gentxs
	err = execCmd(cmdCollect)
	require.NoError(s.T(), err)
}

// ============================================================================
// Keys Command Tests
// ============================================================================

func (s *CLIIntegrationTestSuite) TestKeysAddCmd() {
	tests := []struct {
		name        string
		keyName     string
		mnemonic    string
		recover     bool
		wantErr     bool
		errContains string
	}{
		{
			name:     "add new key",
			keyName:  "newkey1",
			mnemonic: "",
			recover:  false,
			wantErr:  false,
		},
		{
			name:     "recover key from mnemonic",
			keyName:  "recovered1",
			mnemonic: "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon art",
			recover:  true,
			wantErr:  false,
		},
		{
			name:        "empty key name should fail",
			keyName:     "",
			mnemonic:    "",
			recover:     false,
			wantErr:     true,
			errContains: "argument",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			testHomeDir := s.T().TempDir()
			kr, err := keyring.New(sdk.KeyringServiceName(), keyring.BackendTest, testHomeDir, nil, app.MakeEncodingConfig().Codec)
			require.NoError(s.T(), err)

			cmdKeys := cmd.KeysCmd()
			require.NotNil(s.T(), cmdKeys)

			// Find the "add" subcommand
			var addCmd *cobra.Command
			for _, c := range cmdKeys.Commands() {
				if c.Name() == "add" {
					addCmd = c
					break
				}
			}
			require.NotNil(s.T(), addCmd)

			// Setup client context with keyring
			clientCtx := s.clientCtx.WithKeyring(kr).WithHomeDir(testHomeDir)
			require.NoError(s.T(), client.SetCmdClientContext(cmdKeys, clientCtx))
			require.NoError(s.T(), client.SetCmdClientContext(addCmd, clientCtx))

			args := []string{"add", tt.keyName, "--home", testHomeDir, "--keyring-backend", keyring.BackendTest}

			if tt.recover {
				args = append(args, "--recover")
				// Simulate stdin for mnemonic input
				addCmd.SetIn(strings.NewReader(tt.mnemonic + "\n"))
			}

			outBuf := new(bytes.Buffer)
			cmdKeys.SetArgs(args)
			cmdKeys.SetOut(outBuf)
			cmdKeys.SetErr(outBuf)
			addCmd.SetOut(outBuf)
			addCmd.SetErr(outBuf)

			err = execCmd(cmdKeys)

			if tt.wantErr {
				require.Error(s.T(), err)
				if tt.errContains != "" {
					require.Contains(s.T(), err.Error(), tt.errContains)
				}
			} else {
				require.NoError(s.T(), err)

				// Verify key exists
				info, err := kr.Key(tt.keyName)
				require.NoError(s.T(), err)
				require.NotNil(s.T(), info)
				require.Equal(s.T(), tt.keyName, info.Name)
			}
		})
	}
}

func (s *CLIIntegrationTestSuite) TestKeysListCmd() {
	testHomeDir := s.T().TempDir()
	kr, err := keyring.New(sdk.KeyringServiceName(), keyring.BackendTest, testHomeDir, nil, app.MakeEncodingConfig().Codec)
	require.NoError(s.T(), err)

	// Add some test keys
	for i := 1; i <= 3; i++ {
		entropy, err := bip39.NewEntropy(256)
		require.NoError(s.T(), err)
		mnemonic, err := bip39.NewMnemonic(entropy)
		require.NoError(s.T(), err)

		hdPath := hd.CreateHDPath(sdk.CoinType, 0, 0).String()
		_, err = kr.NewAccount(fmt.Sprintf("testkey%d", i), mnemonic, "", hdPath, hd.Secp256k1)
		require.NoError(s.T(), err)
	}

	cmdKeys := cmd.KeysCmd()
	var listCmd *cobra.Command
	for _, c := range cmdKeys.Commands() {
		if c.Name() == "list" {
			listCmd = c
			break
		}
	}
	require.NotNil(s.T(), listCmd)

	clientCtx := s.clientCtx.WithKeyring(kr).WithHomeDir(testHomeDir)
	require.NoError(s.T(), client.SetCmdClientContext(cmdKeys, clientCtx))
	require.NoError(s.T(), client.SetCmdClientContext(listCmd, clientCtx))

	cmdKeys.SetArgs([]string{"list", "--home", testHomeDir, "--keyring-backend", keyring.BackendTest})

	outBuf := new(bytes.Buffer)
	cmdKeys.SetOut(outBuf)
	cmdKeys.SetErr(outBuf)
	listCmd.SetOut(outBuf)
	listCmd.SetErr(outBuf)

	err = execCmd(cmdKeys)
	require.NoError(s.T(), err)

	output := outBuf.String()
	require.Contains(s.T(), output, "testkey1")
	require.Contains(s.T(), output, "testkey2")
	require.Contains(s.T(), output, "testkey3")
}

func (s *CLIIntegrationTestSuite) TestKeysShowCmd() {
	testHomeDir := s.T().TempDir()
	kr, err := keyring.New(sdk.KeyringServiceName(), keyring.BackendTest, testHomeDir, nil, app.MakeEncodingConfig().Codec)
	require.NoError(s.T(), err)

	// Add a test key
	entropy, err := bip39.NewEntropy(256)
	require.NoError(s.T(), err)
	mnemonic, err := bip39.NewMnemonic(entropy)
	require.NoError(s.T(), err)

	hdPath := hd.CreateHDPath(sdk.CoinType, 0, 0).String()
	info, err := kr.NewAccount("showkey", mnemonic, "", hdPath, hd.Secp256k1)
	require.NoError(s.T(), err)

	cmdKeys := cmd.KeysCmd()
	var showCmd *cobra.Command
	for _, c := range cmdKeys.Commands() {
		if c.Name() == "show" {
			showCmd = c
			break
		}
	}
	require.NotNil(s.T(), showCmd)

	clientCtx := s.clientCtx.WithKeyring(kr).WithHomeDir(testHomeDir)
	require.NoError(s.T(), client.SetCmdClientContext(cmdKeys, clientCtx))
	require.NoError(s.T(), client.SetCmdClientContext(showCmd, clientCtx))

	cmdKeys.SetArgs([]string{"show", "showkey", "--home", testHomeDir, "--keyring-backend", keyring.BackendTest})

	outBuf := new(bytes.Buffer)
	cmdKeys.SetOut(outBuf)
	cmdKeys.SetErr(outBuf)
	showCmd.SetOut(outBuf)
	showCmd.SetErr(outBuf)

	err = execCmd(cmdKeys)
	require.NoError(s.T(), err)

	output := outBuf.String()
	addr, err := info.GetAddress()
	require.NoError(s.T(), err)
	require.Contains(s.T(), output, addr.String())
}

func (s *CLIIntegrationTestSuite) TestKeysDeleteCmd() {
	testHomeDir := s.T().TempDir()
	kr, err := keyring.New(sdk.KeyringServiceName(), keyring.BackendTest, testHomeDir, nil, app.MakeEncodingConfig().Codec)
	require.NoError(s.T(), err)

	// Add a test key
	entropy, err := bip39.NewEntropy(256)
	require.NoError(s.T(), err)
	mnemonic, err := bip39.NewMnemonic(entropy)
	require.NoError(s.T(), err)

	hdPath := hd.CreateHDPath(sdk.CoinType, 0, 0).String()
	_, err = kr.NewAccount("deletekey", mnemonic, "", hdPath, hd.Secp256k1)
	require.NoError(s.T(), err)

	// Verify key exists
	_, err = kr.Key("deletekey")
	require.NoError(s.T(), err)

	cmdKeys := cmd.KeysCmd()
	var deleteCmd *cobra.Command
	for _, c := range cmdKeys.Commands() {
		if c.Name() == "delete" {
			deleteCmd = c
			break
		}
	}
	require.NotNil(s.T(), deleteCmd)

	clientCtx := s.clientCtx.WithKeyring(kr).WithHomeDir(testHomeDir)
	require.NoError(s.T(), client.SetCmdClientContext(cmdKeys, clientCtx))
	require.NoError(s.T(), client.SetCmdClientContext(deleteCmd, clientCtx))

	cmdKeys.SetArgs([]string{"delete", "deletekey", "--home", testHomeDir, "--keyring-backend", keyring.BackendTest, "--yes"})

	outBuf := new(bytes.Buffer)
	cmdKeys.SetOut(outBuf)
	cmdKeys.SetErr(outBuf)
	deleteCmd.SetOut(outBuf)
	deleteCmd.SetErr(outBuf)

	err = execCmd(cmdKeys)
	require.NoError(s.T(), err)

	// Verify key was deleted
	_, err = kr.Key("deletekey")
	require.Error(s.T(), err)
}

// ============================================================================
// Query Command Tests (Module-specific)
// ============================================================================

func (s *CLIIntegrationTestSuite) TestDEXQueryCommands() {
	// Note: These tests verify command structure and flag parsing
	// Actual query execution requires a running node

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "query pools help",
			args:    []string{"query", "dex", "pools", "--help"},
			wantErr: false,
		},
		{
			name:    "query pool help",
			args:    []string{"query", "dex", "pool", "--help"},
			wantErr: false,
		},
		{
			name:    "query simulate-swap help",
			args:    []string{"query", "dex", "simulate-swap", "--help"},
			wantErr: false,
		},
		{
			name:    "query pool-by-tokens help",
			args:    []string{"query", "dex", "pool-by-tokens", "--help"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			rootCmd := cmd.NewRootCmd(true)
			rootCmd.SetArgs(append(tt.args, "--home", s.homeDir))

			outBuf := new(bytes.Buffer)
			rootCmd.SetOut(outBuf)
			rootCmd.SetErr(outBuf)

			err := execCmd(rootCmd)

			if tt.wantErr {
				require.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
				require.NotEmpty(s.T(), outBuf.String())
			}
		})
	}
}

func (s *CLIIntegrationTestSuite) TestComputeQueryCommands() {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "query providers help",
			args:    []string{"query", "compute", "providers", "--help"},
			wantErr: false,
		},
		{
			name:    "query provider help",
			args:    []string{"query", "compute", "provider", "--help"},
			wantErr: false,
		},
		{
			name:    "query requests help",
			args:    []string{"query", "compute", "requests", "--help"},
			wantErr: false,
		},
		{
			name:    "query request help",
			args:    []string{"query", "compute", "request", "--help"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			rootCmd := cmd.NewRootCmd(true)
			rootCmd.SetArgs(tt.args)

			outBuf := new(bytes.Buffer)
			rootCmd.SetOut(outBuf)
			rootCmd.SetErr(outBuf)

			err := execCmd(rootCmd)

			if tt.wantErr {
				require.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
				require.NotEmpty(s.T(), outBuf.String())
			}
		})
	}
}

func (s *CLIIntegrationTestSuite) TestOracleQueryCommands() {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "query price help",
			args:    []string{"query", "oracle", "price", "--help"},
			wantErr: false,
		},
		{
			name:    "query prices help",
			args:    []string{"query", "oracle", "prices", "--help"},
			wantErr: false,
		},
		{
			name:    "query validators help",
			args:    []string{"query", "oracle", "validators", "--help"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			rootCmd := cmd.NewRootCmd(true)
			rootCmd.SetArgs(tt.args)

			outBuf := new(bytes.Buffer)
			rootCmd.SetOut(outBuf)
			rootCmd.SetErr(outBuf)

			err := execCmd(rootCmd)

			if tt.wantErr {
				require.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
				require.NotEmpty(s.T(), outBuf.String())
			}
		})
	}
}

// ============================================================================
// Transaction Command Structure Tests
// ============================================================================

func (s *CLIIntegrationTestSuite) TestDEXTxCommands() {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "tx dex create-pool help",
			args:    []string{"tx", "dex", "create-pool", "--help"},
			wantErr: false,
		},
		{
			name:    "tx dex add-liquidity help",
			args:    []string{"tx", "dex", "add-liquidity", "--help"},
			wantErr: false,
		},
		{
			name:    "tx dex remove-liquidity help",
			args:    []string{"tx", "dex", "remove-liquidity", "--help"},
			wantErr: false,
		},
		{
			name:    "tx dex swap help",
			args:    []string{"tx", "dex", "swap", "--help"},
			wantErr: false,
		},
		{
			name:    "tx dex advanced help",
			args:    []string{"tx", "dex", "advanced", "--help"},
			wantErr: false,
		},
		{
			name:    "tx dex advanced swap-with-slippage help",
			args:    []string{"tx", "dex", "advanced", "swap-with-slippage", "--help"},
			wantErr: false,
		},
		{
			name:    "tx dex advanced batch-swap help",
			args:    []string{"tx", "dex", "advanced", "batch-swap", "--help"},
			wantErr: false,
		},
		{
			name:    "tx dex advanced quick-swap help",
			args:    []string{"tx", "dex", "advanced", "quick-swap", "--help"},
			wantErr: false,
		},
		{
			name:    "tx dex advanced zap-in help",
			args:    []string{"tx", "dex", "advanced", "zap-in", "--help"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			rootCmd := cmd.NewRootCmd(true)
			rootCmd.SetArgs(tt.args)

			outBuf := new(bytes.Buffer)
			rootCmd.SetOut(outBuf)
			rootCmd.SetErr(outBuf)

			err := execCmd(rootCmd)

			if tt.wantErr {
				require.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
				require.NotEmpty(s.T(), outBuf.String())
			}
		})
	}
}

func (s *CLIIntegrationTestSuite) TestComputeTxCommands() {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "tx compute register-provider help",
			args:    []string{"tx", "compute", "register-provider", "--help"},
			wantErr: false,
		},
		{
			name:    "tx compute update-provider help",
			args:    []string{"tx", "compute", "update-provider", "--help"},
			wantErr: false,
		},
		{
			name:    "tx compute deactivate-provider help",
			args:    []string{"tx", "compute", "deactivate-provider", "--help"},
			wantErr: false,
		},
		{
			name:    "tx compute submit-request help",
			args:    []string{"tx", "compute", "submit-request", "--help"},
			wantErr: false,
		},
		{
			name:    "tx compute cancel-request help",
			args:    []string{"tx", "compute", "cancel-request", "--help"},
			wantErr: false,
		},
		{
			name:    "tx compute submit-result help",
			args:    []string{"tx", "compute", "submit-result", "--help"},
			wantErr: false,
		},
		{
			name:    "tx compute create-dispute help",
			args:    []string{"tx", "compute", "create-dispute", "--help"},
			wantErr: false,
		},
		{
			name:    "tx compute vote-dispute help",
			args:    []string{"tx", "compute", "vote-dispute", "--help"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			rootCmd := cmd.NewRootCmd(true)
			rootCmd.SetArgs(tt.args)

			outBuf := new(bytes.Buffer)
			rootCmd.SetOut(outBuf)
			rootCmd.SetErr(outBuf)

			err := execCmd(rootCmd)

			if tt.wantErr {
				require.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
				require.NotEmpty(s.T(), outBuf.String())
			}
		})
	}
}

func (s *CLIIntegrationTestSuite) TestOracleTxCommands() {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "tx oracle submit-price help",
			args:    []string{"tx", "oracle", "submit-price", "--help"},
			wantErr: false,
		},
		{
			name:    "tx oracle delegate-feeder help",
			args:    []string{"tx", "oracle", "delegate-feeder", "--help"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			rootCmd := cmd.NewRootCmd(true)
			rootCmd.SetArgs(tt.args)

			outBuf := new(bytes.Buffer)
			rootCmd.SetOut(outBuf)
			rootCmd.SetErr(outBuf)

			err := execCmd(rootCmd)

			if tt.wantErr {
				require.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
				require.NotEmpty(s.T(), outBuf.String())
			}
		})
	}
}

// ============================================================================
// Message Validation Tests (Offline)
// ============================================================================

func (s *CLIIntegrationTestSuite) TestDEXMessageValidation() {
	s.ensureTestAccounts()

	tests := []struct {
		name    string
		msg     sdk.Msg
		wantErr bool
	}{
		{
			name: "valid create pool message",
			msg: &dextypes.MsgCreatePool{
				Creator: s.testAccounts[0].address.String(),
				TokenA:  "upaw",
				TokenB:  "uusdt",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(1000000),
			},
			wantErr: false,
		},
		{
			name: "invalid create pool - same tokens",
			msg: &dextypes.MsgCreatePool{
				Creator: s.testAccounts[0].address.String(),
				TokenA:  "upaw",
				TokenB:  "upaw",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(1000000),
			},
			wantErr: true,
		},
		{
			name: "invalid create pool - zero amount",
			msg: &dextypes.MsgCreatePool{
				Creator: s.testAccounts[0].address.String(),
				TokenA:  "upaw",
				TokenB:  "uusdt",
				AmountA: math.NewInt(0),
				AmountB: math.NewInt(1000000),
			},
			wantErr: true,
		},
		{
			name: "valid swap message",
			msg: &dextypes.MsgSwap{
				Trader:       s.testAccounts[0].address.String(),
				PoolId:       1,
				TokenIn:      "upaw",
				TokenOut:     "uusdt",
				AmountIn:     math.NewInt(1000),
				MinAmountOut: math.NewInt(900),
				Deadline:     time.Now().Unix() + 300,
			},
			wantErr: false,
		},
		{
			name: "invalid swap - same token in/out",
			msg: &dextypes.MsgSwap{
				Trader:       s.testAccounts[0].address.String(),
				PoolId:       1,
				TokenIn:      "upaw",
				TokenOut:     "upaw",
				AmountIn:     math.NewInt(1000),
				MinAmountOut: math.NewInt(900),
				Deadline:     time.Now().Unix() + 300,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Call ValidateBasic if the message implements it
			if validator, ok := tt.msg.(interface{ ValidateBasic() error }); ok {
				err := validator.ValidateBasic()
				if tt.wantErr {
					require.Error(s.T(), err)
				} else {
					require.NoError(s.T(), err)
				}
			} else {
				// If ValidateBasic doesn't exist, skip validation
				s.T().Skip("ValidateBasic not implemented")
			}
		})
	}
}

func (s *CLIIntegrationTestSuite) TestComputeMessageValidation() {
	s.ensureTestAccounts()

	tests := []struct {
		name    string
		msg     sdk.Msg
		wantErr bool
	}{
		{
			name: "valid register provider message",
			msg: &computetypes.MsgRegisterProvider{
				Provider: s.testAccounts[0].address.String(),
				Moniker:  "TestProvider",
				Endpoint: "https://provider.example.com",
				AvailableSpecs: computetypes.ComputeSpec{
					CpuCores:       8,
					MemoryMb:       16384,
					StorageGb:      1000,
					GpuCount:       2,
					TimeoutSeconds: 3600,
				},
				Pricing: computetypes.Pricing{
					CpuPricePerMcoreHour:  math.LegacyNewDecWithPrec(1, 3),
					MemoryPricePerMbHour:  math.LegacyNewDecWithPrec(5, 4),
					GpuPricePerHour:       math.LegacyNewDecWithPrec(1, 1),
					StoragePricePerGbHour: math.LegacyNewDecWithPrec(1, 4),
				},
				Stake: math.NewInt(1000000),
			},
			wantErr: false,
		},
		{
			name: "invalid register provider - empty moniker",
			msg: &computetypes.MsgRegisterProvider{
				Provider: s.testAccounts[0].address.String(),
				Moniker:  "",
				Endpoint: "https://provider.example.com",
				AvailableSpecs: computetypes.ComputeSpec{
					CpuCores: 8,
				},
				Stake: math.NewInt(1000000),
			},
			wantErr: true,
		},
		{
			name: "valid submit request message",
			msg: &computetypes.MsgSubmitRequest{
				Requester:      s.testAccounts[0].address.String(),
				ContainerImage: "ubuntu:22.04",
				Command:        []string{"python", "script.py"},
				EnvVars:        map[string]string{"KEY": "value"},
				Specs: computetypes.ComputeSpec{
					CpuCores:       4,
					MemoryMb:       8192,
					TimeoutSeconds: 1800,
				},
				MaxPayment: math.NewInt(100000),
			},
			wantErr: false,
		},
		{
			name: "invalid submit request - empty container image",
			msg: &computetypes.MsgSubmitRequest{
				Requester:      s.testAccounts[0].address.String(),
				ContainerImage: "",
				Specs: computetypes.ComputeSpec{
					CpuCores: 4,
				},
				MaxPayment: math.NewInt(100000),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Call ValidateBasic if the message implements it
			if validator, ok := tt.msg.(interface{ ValidateBasic() error }); ok {
				err := validator.ValidateBasic()
				if tt.wantErr {
					require.Error(s.T(), err)
				} else {
					require.NoError(s.T(), err)
				}
			} else {
				// If ValidateBasic doesn't exist, skip validation
				s.T().Skip("ValidateBasic not implemented")
			}
		})
	}
}

func (s *CLIIntegrationTestSuite) TestOracleMessageValidation() {
	s.ensureTestAccounts()

	tests := []struct {
		name    string
		msg     sdk.Msg
		wantErr bool
	}{
		{
			name: "valid submit price message",
			msg: oracletypes.NewMsgSubmitPrice(
				s.testAccounts[0].valAddr.String(),
				s.testAccounts[0].address.String(),
				"BTC",
				math.LegacyNewDec(50000),
			),
			wantErr: false,
		},
		{
			name: "invalid submit price - empty asset",
			msg: oracletypes.NewMsgSubmitPrice(
				s.testAccounts[0].valAddr.String(),
				s.testAccounts[0].address.String(),
				"",
				math.LegacyNewDec(50000),
			),
			wantErr: true,
		},
		{
			name: "invalid submit price - zero price",
			msg: oracletypes.NewMsgSubmitPrice(
				s.testAccounts[0].valAddr.String(),
				s.testAccounts[0].address.String(),
				"BTC",
				math.LegacyZeroDec(),
			),
			wantErr: true,
		},
		{
			name: "invalid submit price - negative price",
			msg: oracletypes.NewMsgSubmitPrice(
				s.testAccounts[0].valAddr.String(),
				s.testAccounts[0].address.String(),
				"BTC",
				math.LegacyNewDec(-1000),
			),
			wantErr: true,
		},
		{
			name: "valid delegate feeder message",
			msg: oracletypes.NewMsgDelegateFeedConsent(
				s.testAccounts[0].valAddr.String(),
				s.testAccounts[1].address.String(),
			),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Call ValidateBasic if the message implements it
			if validator, ok := tt.msg.(interface{ ValidateBasic() error }); ok {
				err := validator.ValidateBasic()
				if tt.wantErr {
					require.Error(s.T(), err)
				} else {
					require.NoError(s.T(), err)
				}
			} else {
				// If ValidateBasic doesn't exist, skip validation
				s.T().Skip("ValidateBasic not implemented")
			}
		})
	}
}

// ============================================================================
// Standard Bank/Staking Command Tests
// ============================================================================

func (s *CLIIntegrationTestSuite) TestBankTxCommands() {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "tx bank send help",
			args:    []string{"tx", "bank", "send", "--help"},
			wantErr: false,
		},
		{
			name:    "tx bank multi-send help",
			args:    []string{"tx", "bank", "multi-send", "--help"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			rootCmd := cmd.NewRootCmd(true)
			rootCmd.SetArgs(tt.args)

			outBuf := new(bytes.Buffer)
			rootCmd.SetOut(outBuf)
			rootCmd.SetErr(outBuf)

			err := execCmd(rootCmd)

			if tt.wantErr {
				require.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
				require.NotEmpty(s.T(), outBuf.String())
			}
		})
	}
}

func (s *CLIIntegrationTestSuite) TestStakingTxCommands() {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "tx staking delegate help",
			args:    []string{"tx", "staking", "delegate", "--help"},
			wantErr: false,
		},
		{
			name:    "tx staking unbond help",
			args:    []string{"tx", "staking", "unbond", "--help"},
			wantErr: false,
		},
		{
			name:    "tx staking redelegate help",
			args:    []string{"tx", "staking", "redelegate", "--help"},
			wantErr: false,
		},
		{
			name:    "tx staking create-validator help",
			args:    []string{"tx", "staking", "create-validator", "--help"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			rootCmd := cmd.NewRootCmd(true)
			rootCmd.SetArgs(tt.args)

			outBuf := new(bytes.Buffer)
			rootCmd.SetOut(outBuf)
			rootCmd.SetErr(outBuf)

			err := execCmd(rootCmd)

			if tt.wantErr {
				require.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
				require.NotEmpty(s.T(), outBuf.String())
			}
		})
	}
}

func (s *CLIIntegrationTestSuite) TestQueryCommands() {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "query bank balances help",
			args:    []string{"query", "bank", "balances", "--help"},
			wantErr: false,
		},
		{
			name:    "query staking validators help",
			args:    []string{"query", "staking", "validators", "--help"},
			wantErr: false,
		},
		{
			name:    "query staking delegations help",
			args:    []string{"query", "staking", "delegations", "--help"},
			wantErr: false,
		},
		{
			name:    "query gov proposals help",
			args:    []string{"query", "gov", "proposals", "--help"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			rootCmd := cmd.NewRootCmd(true)
			rootCmd.SetArgs(tt.args)

			outBuf := new(bytes.Buffer)
			rootCmd.SetOut(outBuf)
			rootCmd.SetErr(outBuf)

			err := execCmd(rootCmd)

			if tt.wantErr {
				require.Error(s.T(), err)
			} else {
				require.NoError(s.T(), err)
				require.NotEmpty(s.T(), outBuf.String())
			}
		})
	}
}

// ============================================================================
// Command Structure and Help Tests
// ============================================================================

func (s *CLIIntegrationTestSuite) TestRootCommandStructure() {
	rootCmd := cmd.NewRootCmd(true)
	require.NotNil(s.T(), rootCmd)
	require.Equal(s.T(), "pawd", rootCmd.Use)

	// Verify main subcommands exist
	commands := make(map[string]bool)
	for _, c := range rootCmd.Commands() {
		commands[c.Name()] = true
	}

	// Essential commands
	require.True(s.T(), commands["init"], "init command should exist")
	require.True(s.T(), commands["start"], "start command should exist")
	require.True(s.T(), commands["query"], "query command should exist")
	require.True(s.T(), commands["tx"], "tx command should exist")
	require.True(s.T(), commands["keys"], "keys command should exist")
	require.True(s.T(), commands["status"], "status command should exist")
}

func (s *CLIIntegrationTestSuite) TestHelpCommand() {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "root help",
			args: []string{"--help"},
		},
		{
			name: "query help",
			args: []string{"query", "--help"},
		},
		{
			name: "tx help",
			args: []string{"tx", "--help"},
		},
		{
			name: "keys help",
			args: []string{"keys", "--help"},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			rootCmd := cmd.NewRootCmd(true)
			rootCmd.SetArgs(tt.args)

			outBuf := new(bytes.Buffer)
			rootCmd.SetOut(outBuf)
			rootCmd.SetErr(outBuf)

			err := execCmd(rootCmd)
			require.NoError(s.T(), err)

			output := outBuf.String()
			require.NotEmpty(s.T(), output)
			require.Contains(s.T(), output, "Usage:")
		})
	}
}

func (s *CLIIntegrationTestSuite) TestVersionCommand() {
	rootCmd := cmd.NewRootCmd(true)
	rootCmd.SetArgs([]string{"version"})

	outBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(outBuf)

	err := execCmd(rootCmd)
	require.NoError(s.T(), err)

	output := outBuf.String()
	require.NotEmpty(s.T(), output)
}

// ============================================================================
// Flag Validation Tests
// ============================================================================

func (s *CLIIntegrationTestSuite) TestCommonFlags() {
	tests := []struct {
		name     string
		args     []string
		checkOut func(t *testing.T, output string)
	}{
		{
			name: "chain-id flag",
			args: []string{"query", "bank", "balances", "paw1test", "--help"},
			checkOut: func(t *testing.T, output string) {
				require.Contains(t, output, "--chain-id")
			},
		},
		{
			name: "home flag",
			args: []string{"init", "--help"},
			checkOut: func(t *testing.T, output string) {
				require.Contains(t, output, "--home")
			},
		},
		{
			name: "keyring-backend flag",
			args: []string{"keys", "add", "--help"},
			checkOut: func(t *testing.T, output string) {
				require.Contains(t, output, "--keyring-backend")
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			rootCmd := cmd.NewRootCmd(true)
			rootCmd.SetArgs(tt.args)

			outBuf := new(bytes.Buffer)
			rootCmd.SetOut(outBuf)
			rootCmd.SetErr(outBuf)

			err := execCmd(rootCmd)
			require.NoError(s.T(), err)

			if tt.checkOut != nil {
				tt.checkOut(s.T(), outBuf.String())
			}
		})
	}
}

// ============================================================================
// Run Test Suite
// ============================================================================

func TestCLIIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CLIIntegrationTestSuite))
}
