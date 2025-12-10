package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	cmtos "github.com/cometbft/cometbft/libs/os"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	"github.com/spf13/cobra"

	"github.com/paw-chain/paw/app"
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
  pawd init paw-controller --chain-id paw-testnet-1 --home ~/.paw
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
				defaultDenom = app.BondDenom
			}

			// Create default genesis state
			appState, err := json.MarshalIndent(mbm.DefaultGenesis(cdc), "", " ")
			if err != nil {
				return fmt.Errorf("failed to marshal default genesis state: %w", err)
			}

			// Create genesis doc
			genDoc := &tmtypes.GenesisDoc{
				ChainID:         chainID,
				GenesisTime:     time.Now().UTC(),
				ConsensusParams: tmtypes.DefaultConsensusParams(),
				AppState:        appState,
			}

			// Update consensus params with PAW-specific values
			const (
				maxBlockBytes        int64 = 2_097_152   // 2 MB
				maxBlockGas          int64 = 100_000_000 // 100M gas
				evidenceMaxAgeBlocks       = 500_000     // ~23 days @ 4s block time
				evidenceMaxBytes     int64 = 1_048_576   // 1 MB
			)

			genDoc.ConsensusParams.Block.MaxBytes = maxBlockBytes
			genDoc.ConsensusParams.Block.MaxGas = maxBlockGas
			// TimeIotaMs was removed in CometBFT - block time is controlled by consensus
			genDoc.ConsensusParams.Evidence.MaxAgeNumBlocks = evidenceMaxAgeBlocks
			genDoc.ConsensusParams.Evidence.MaxAgeDuration = 21 * 24 * time.Hour
			genDoc.ConsensusParams.Evidence.MaxBytes = evidenceMaxBytes // 1 MB

			if err = genDoc.ValidateAndComplete(); err != nil {
				return fmt.Errorf("failed to validate genesis doc: %w", err)
			}

			// Canonicalize and save genesis using CometBFT JSON (ints as strings, no null app_hash)
			genDoc.AppHash = cmtbytes.HexBytes{} // avoid null

			type canonicalGenesis struct {
				GenesisTime     time.Time                  `json:"genesis_time"`
				ChainID         string                     `json:"chain_id"`
				InitialHeight   string                     `json:"initial_height"`
				ConsensusParams *tmtypes.ConsensusParams   `json:"consensus_params,omitempty"`
				Validators      []tmtypes.GenesisValidator `json:"validators,omitempty"`
				AppHash         string                     `json:"app_hash"`
				AppState        json.RawMessage            `json:"app_state,omitempty"`
			}

			canon := canonicalGenesis{
				GenesisTime:     genDoc.GenesisTime,
				ChainID:         genDoc.ChainID,
				InitialHeight:   fmt.Sprintf("%d", genDoc.InitialHeight),
				ConsensusParams: genDoc.ConsensusParams,
				Validators:      genDoc.Validators,
				AppHash:         genDoc.AppHash.String(),
				AppState:        genDoc.AppState,
			}

			genDocBytes, err := json.Marshal(canon)
			if err != nil {
				return fmt.Errorf("failed to marshal canonical genesis doc: %w", err)
			}
			intermediate, err := decodeJSONWithNumbers(genDocBytes)
			if err != nil {
				return fmt.Errorf("failed to canonicalize genesis structure: %w", err)
			}
			normalized := normalizeNumbersToStrings(intermediate).(map[string]interface{})
			normalized["initial_height"] = fmt.Sprintf("%d", genDoc.InitialHeight)
			pretty, err := json.MarshalIndent(normalized, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal canonical genesis doc: %w", err)
			}
			if _, err := tmtypes.GenesisDocFromJSON(pretty); err != nil {
				return fmt.Errorf("canonical genesis marshal validation failed: %w", err)
			}
			if err := cmtos.WriteFile(genFile, pretty, 0o644); err != nil {
				return fmt.Errorf("failed to save genesis file: %w", err)
			}
			if err := canonicalizeGenesisFile(genFile); err != nil {
				return fmt.Errorf("failed to canonicalize final genesis: %w", err)
			}
			if err := forceInitialHeightString(genFile); err != nil {
				return fmt.Errorf("failed to enforce initial_height string encoding: %w", err)
			}

			// Create config directory structure
			configDir := filepath.Dir(genFile)
			dataDir := filepath.Join(clientCtx.HomeDir, "data")
			if err := os.MkdirAll(dataDir, 0o750); err != nil {
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
	cmd.Flags().String(flagDefaultDenom, app.BondDenom, "default denomination for the chain")
	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "node's home directory")

	return cmd
}

// normalizeNumbersToStrings walks a decoded JSON structure and turns all numeric values into
// decimal strings (to satisfy CometBFT's Amino-compatible JSON decoding expectations).
func normalizeNumbersToStrings(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, vv := range val {
			// avoid null app_hash
			if k == "app_hash" && vv == nil {
				out[k] = ""
				continue
			}
			out[k] = normalizeNumbersToStrings(vv)
		}
		return out
	case []interface{}:
		for i, vv := range val {
			val[i] = normalizeNumbersToStrings(vv)
		}
		return val
	case json.Number:
		return val.String()
	case float64:
		return fmt.Sprintf("%.0f", val)
	default:
		return val
	}
}

func decodeJSONWithNumbers(bz []byte) (interface{}, error) {
	dec := json.NewDecoder(bytes.NewReader(bz))
	dec.UseNumber()
	var v interface{}
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// canonicalizeGenesisFile rewrites the genesis file ensuring all int64-like fields are encoded as strings,
// app_hash is non-null, and the result passes CometBFT genesis validation.
func canonicalizeGenesisFile(path string) error {
	bz, err := os.ReadFile(path) // #nosec G304 - path originates from operator-controlled init arguments
	if err != nil {
		return fmt.Errorf("failed to read genesis file for canonicalization: %w", err)
	}

	raw, err := decodeJSONWithNumbers(bz)
	if err != nil {
		return fmt.Errorf("failed to decode genesis for canonicalization: %w", err)
	}

	canonical := normalizeNumbersToStrings(raw)
	if m, ok := canonical.(map[string]interface{}); ok {
		if v, exists := m["initial_height"]; exists {
			m["initial_height"] = fmt.Sprintf("%v", v)
		}
		canonical = m
	}
	pretty, err := json.MarshalIndent(canonical, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal canonical genesis: %w", err)
	}
	prettyStr := strings.ReplaceAll(string(pretty), `"initial_height": 1`, `"initial_height": "1"`)
	pretty = []byte(prettyStr)

	if _, err := tmtypes.GenesisDocFromJSON(pretty); err != nil {
		return fmt.Errorf("canonical genesis validation failed: %w", err)
	}

	if err := cmtos.WriteFile(path, pretty, 0o644); err != nil {
		return fmt.Errorf("failed to write canonical genesis: %w", err)
	}

	return nil
}

// forceInitialHeightString is a hardening pass to ensure initial_height is encoded as a string.
func forceInitialHeightString(path string) error {
	bz, err := os.ReadFile(path) // #nosec G304 - path is operator-provided during init flow
	if err != nil {
		return err
	}
	updated := strings.ReplaceAll(string(bz), `"initial_height": 1`, `"initial_height": "1"`)
	if _, err := tmtypes.GenesisDocFromJSON([]byte(updated)); err != nil {
		return fmt.Errorf("post-rewrite genesis validation failed: %w", err)
	}
	return cmtos.WriteFile(path, []byte(updated), 0o644)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
