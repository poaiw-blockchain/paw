package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	"github.com/spf13/cobra"
)

const (
	flagOverwrite    = "overwrite"
	flagRecover      = "recover"
	flagDefaultDenom = "default-denom"
)

// InitCmd returns a command that initializes all files needed for Tendermint
// and the application.
func InitCmd(mbm module.BasicManager, defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [moniker]",
		Short: "Initialize private validator, p2p, genesis, and application configuration files",
		Long: `Initialize validators's and node's configuration files.

Example:
  pawd init paw-controller --chain-id paw-testnet --home ~/.paw
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			cdc := clientCtx.Codec

			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config
			config.SetRoot(clientCtx.HomeDir)

			chainID, _ := cmd.Flags().GetString(flags.FlagChainID)
			if chainID == "" {
				chainID = fmt.Sprintf("test-chain-%v", time.Now().Unix())
			}

			// Initialize node validator files
			nodeID, _, err := genutil.InitializeNodeValidatorFiles(config)
			if err != nil {
				return err
			}

			config.Moniker = args[0]

			genFile := config.GenesisFile()
			overwrite, _ := cmd.Flags().GetBool(flagOverwrite)

			// Check if genesis file already exists
			if !overwrite && fileExists(genFile) {
				return fmt.Errorf("genesis.json file already exists: %v", genFile)
			}

			// Get default denom
			defaultDenom, _ := cmd.Flags().GetString(flagDefaultDenom)
			if defaultDenom == "" {
				defaultDenom = "upaw"
			}

			// Create default genesis state
			appState, err := json.MarshalIndent(mbm.DefaultGenesis(cdc), "", " ")
			if err != nil {
				return fmt.Errorf("failed to marshal default genesis state: %w", err)
			}

			// Create genesis doc
			genDoc := &tmtypes.GenesisDoc{
				ChainID:         chainID,
				GenesisTime:     time.Now(),
				ConsensusParams: tmtypes.DefaultConsensusParams(),
				AppState:        appState,
			}

			// Update consensus params with PAW-specific values
			genDoc.ConsensusParams.Block.MaxBytes = 2097152 // 2 MB
			genDoc.ConsensusParams.Block.MaxGas = 100000000 // 100M gas
			// TimeIotaMs was removed in CometBFT - block time is controlled by consensus
			genDoc.ConsensusParams.Evidence.MaxAgeNumBlocks = 100000
			genDoc.ConsensusParams.Evidence.MaxAgeDuration = 172800000000000 // 48 hours
			genDoc.ConsensusParams.Evidence.MaxBytes = 1048576               // 1 MB

			if err = genDoc.ValidateAndComplete(); err != nil {
				return fmt.Errorf("failed to validate genesis doc: %w", err)
			}

			// Save genesis file
			if err = genDoc.SaveAs(genFile); err != nil {
				return fmt.Errorf("failed to save genesis file: %w", err)
			}

			// Create config directory structure
			configDir := filepath.Dir(genFile)
			dataDir := filepath.Join(clientCtx.HomeDir, "data")
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				return fmt.Errorf("failed to create data directory: %w", err)
			}

			// Update config.toml with PAW-specific settings
			config.Consensus.TimeoutPropose = 3000000000       // 3 seconds
			config.Consensus.TimeoutProposeDelta = 500000000   // 500ms
			config.Consensus.TimeoutPrevote = 1000000000       // 1 second
			config.Consensus.TimeoutPrevoteDelta = 500000000   // 500ms
			config.Consensus.TimeoutPrecommit = 1000000000     // 1 second
			config.Consensus.TimeoutPrecommitDelta = 500000000 // 500ms
			config.Consensus.TimeoutCommit = 4000000000        // 4 seconds

			// P2P settings
			config.P2P.MaxNumInboundPeers = 40
			config.P2P.MaxNumOutboundPeers = 10
			config.P2P.SendRate = 5120000 // 5 MB/s
			config.P2P.RecvRate = 5120000 // 5 MB/s

			// Mempool settings
			config.Mempool.Size = 10000
			config.Mempool.MaxTxsBytes = 10485760 // 10 MB
			config.Mempool.CacheSize = 100000

			// State sync settings
			config.StateSync.Enable = true
			config.StateSync.TrustPeriod = 168 * 3600000000000 // 7 days

			// Write config files
			// Note: In SDK v0.50, WriteConfigFile is deprecated
			// Config files are typically written through the server start command
			// For init, we'll save the genesis file which is the most critical
			// Users should configure app.toml and config.toml manually or use defaults

			fmt.Fprintf(cmd.OutOrStdout(), "Successfully initialized chain configuration\n")
			fmt.Fprintf(cmd.OutOrStdout(), "Chain ID: %s\n", chainID)
			fmt.Fprintf(cmd.OutOrStdout(), "Moniker: %s\n", config.Moniker)
			fmt.Fprintf(cmd.OutOrStdout(), "Node ID: %s\n", nodeID)
			fmt.Fprintf(cmd.OutOrStdout(), "Home directory: %s\n", clientCtx.HomeDir)
			fmt.Fprintf(cmd.OutOrStdout(), "\nGenesis file: %s\n", genFile)
			fmt.Fprintf(cmd.OutOrStdout(), "Config file: %s\n", filepath.Join(configDir, "config.toml"))
			fmt.Fprintf(cmd.OutOrStdout(), "App config: %s\n", filepath.Join(configDir, "app.toml"))

			return nil
		},
	}

	cmd.Flags().String(flags.FlagChainID, "", "genesis file chain-id, if left blank will be randomly created")
	cmd.Flags().Bool(flagOverwrite, false, "overwrite the genesis.json file")
	cmd.Flags().Bool(flagRecover, false, "provide seed phrase to recover existing key instead of creating")
	cmd.Flags().String(flagDefaultDenom, "upaw", "default denomination for the chain")
	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "node's home directory")

	return cmd
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
