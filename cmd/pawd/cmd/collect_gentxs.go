package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	tmtypes "github.com/tendermint/tendermint/types"
)

// CollectGenTxsCmd returns a command to collect genesis transactions
func CollectGenTxsCmd(mbm module.BasicManager, defaultNodeHome string, genBalIterator genutiltypes.GenesisBalancesIterator, validator genutiltypes.MessageValidator) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collect-gentxs",
		Short: "Collect genesis txs and output a genesis.json file",
		Long: `Collect genesis transactions from the configured gentx directory and
update the genesis file with the collected transactions.

Example:
  pawd collect-gentxs --home ~/.paw
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)
			clientCtx := client.GetClientContextFromCmd(cmd)

			config := serverCtx.Config
			config.SetRoot(clientCtx.HomeDir)

			// Read genesis file
			genFile := config.GenesisFile()
			genDoc, err := tmtypes.GenesisDocFromFile(genFile)
			if err != nil {
				return fmt.Errorf("failed to read genesis doc from file %s: %w", genFile, err)
			}

			// Unmarshal genesis state
			var genesisState map[string]json.RawMessage
			if err = json.Unmarshal(genDoc.AppState, &genesisState); err != nil {
				return fmt.Errorf("failed to unmarshal genesis state: %w", err)
			}

			// Read gentx files
			gentxDir := filepath.Join(config.RootDir, "config", "gentx")
			gentxFiles, err := os.ReadDir(gentxDir)
			if err != nil {
				return fmt.Errorf("failed to read gentx directory: %w", err)
			}

			if len(gentxFiles) == 0 {
				return fmt.Errorf("no gentx files found in %s", gentxDir)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Collecting genesis transactions...\n")
			fmt.Fprintf(cmd.OutOrStdout(), "Found %d gentx files\n", len(gentxFiles))

			// Collect all gentxs
			var genTxs []sdk.Tx
			for _, gentxFile := range gentxFiles {
				if !gentxFile.IsDir() && filepath.Ext(gentxFile.Name()) == ".json" {
					gentxPath := filepath.Join(gentxDir, gentxFile.Name())

					// Read gentx file
					gentxBz, err := os.ReadFile(gentxPath)
					if err != nil {
						return fmt.Errorf("failed to read gentx file %s: %w", gentxPath, err)
					}

					// Decode gentx
					tx, err := clientCtx.TxConfig.TxJSONDecoder()(gentxBz)
					if err != nil {
						return fmt.Errorf("failed to decode gentx %s: %w", gentxPath, err)
					}

					// Validate gentx
					msgs := tx.GetMsgs()
					if len(msgs) != 1 {
						return fmt.Errorf("gentx must contain exactly one message, got %d", len(msgs))
					}

					// Verify it's a MsgCreateValidator
					msgCreateVal, ok := msgs[0].(*stakingtypes.MsgCreateValidator)
					if !ok {
						return fmt.Errorf("gentx message must be MsgCreateValidator")
					}

					if err := msgCreateVal.ValidateBasic(); err != nil {
						return fmt.Errorf("invalid gentx: %w", err)
					}

					genTxs = append(genTxs, tx)
					fmt.Fprintf(cmd.OutOrStdout(), "  ✓ Collected gentx from %s\n", gentxFile.Name())
				}
			}

			// Update genesis state with collected gentxs
			genUtilGenesis := genutiltypes.GetGenesisStateFromAppState(clientCtx.Codec, genesisState)

			// Convert gentxs to JSON
			genTxsJSON := make([]json.RawMessage, len(genTxs))
			for i, tx := range genTxs {
				txBz, err := clientCtx.TxConfig.TxJSONEncoder()(tx)
				if err != nil {
					return fmt.Errorf("failed to encode gentx: %w", err)
				}
				genTxsJSON[i] = txBz
			}

			genUtilGenesis.GenTxs = genTxsJSON

			// Save updated genutil genesis state
			genesisState[genutiltypes.ModuleName] = clientCtx.Codec.MustMarshalJSON(genUtilGenesis)

			// Update staking genesis state with validators
			stakingGenesis := stakingtypes.GetGenesisStateFromAppState(clientCtx.Codec, genesisState)

			// Extract validators from gentxs
			for _, tx := range genTxs {
				msgs := tx.GetMsgs()
				msgCreateVal := msgs[0].(*stakingtypes.MsgCreateValidator)

				// Add validator to genesis state
				validator, err := stakingtypes.NewValidator(
					msgCreateVal.ValidatorAddress,
					msgCreateVal.Pubkey,
					stakingtypes.Description{
						Moniker: msgCreateVal.Description.Moniker,
					},
				)
				if err != nil {
					return fmt.Errorf("failed to create validator: %w", err)
				}

				validator.Commission = stakingtypes.Commission{
					CommissionRates: msgCreateVal.Commission,
					UpdateTime:      genDoc.GenesisTime,
				}
				validator.MinSelfDelegation = msgCreateVal.MinSelfDelegation
				validator.Tokens = msgCreateVal.Value.Amount
				validator.DelegatorShares = sdk.NewDecFromInt(msgCreateVal.Value.Amount)

				stakingGenesis.Validators = append(stakingGenesis.Validators, validator)

				// Add delegation
				delegation := stakingtypes.Delegation{
					DelegatorAddress: sdk.AccAddress(validator.GetOperator()).String(),
					ValidatorAddress: validator.GetOperator().String(),
					Shares:           validator.DelegatorShares,
				}
				stakingGenesis.Delegations = append(stakingGenesis.Delegations, delegation)
			}

			// Save updated staking genesis state
			genesisState[stakingtypes.ModuleName] = clientCtx.Codec.MustMarshalJSON(stakingGenesis)

			// Validate genesis state
			if err = mbm.ValidateGenesis(clientCtx.Codec, clientCtx.TxConfig, genesisState); err != nil {
				return fmt.Errorf("failed to validate genesis state: %w", err)
			}

			// Marshal updated genesis state
			appStateJSON, err := json.MarshalIndent(genesisState, "", " ")
			if err != nil {
				return fmt.Errorf("failed to marshal genesis state: %w", err)
			}

			genDoc.AppState = appStateJSON

			// Validate and complete genesis doc
			if err = genDoc.ValidateAndComplete(); err != nil {
				return fmt.Errorf("failed to validate genesis doc: %w", err)
			}

			// Save updated genesis file
			if err = genDoc.SaveAs(genFile); err != nil {
				return fmt.Errorf("failed to save genesis file: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "\nSuccessfully collected %d genesis transactions\n", len(genTxs))
			fmt.Fprintf(cmd.OutOrStdout(), "Genesis file updated: %s\n", genFile)
			fmt.Fprintf(cmd.OutOrStdout(), "\nValidators:\n")
			for i, val := range stakingGenesis.Validators {
				fmt.Fprintf(cmd.OutOrStdout(), "  %d. %s (%s)\n", i+1, val.Description.Moniker, val.OperatorAddress)
			}

			return nil
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
