package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	tmtypes "github.com/cometbft/cometbft/types"
)

const (
	flagCommissionRate          = "commission-rate"
	flagCommissionMaxRate       = "commission-max-rate"
	flagCommissionMaxChangeRate = "commission-max-change-rate"
	flagMinSelfDelegation       = "min-self-delegation"
	flagPubKey                  = "pubkey"
	flagMoniker                 = "moniker"
	flagIdentity                = "identity"
	flagWebsite                 = "website"
	flagSecurityContact         = "security-contact"
	flagDetails                 = "details"
)

// GenTxCmd builds the application's gentx command
func GenTxCmd(mbm module.BasicManager, txEncCfg client.TxEncodingConfig, genBalIterator genutiltypes.GenesisBalancesIterator, defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gentx [key_name] [amount]",
		Short: "Generate a genesis tx carrying a self delegation",
		Long: `Generate a genesis transaction that creates a validator with a self-delegation,
that is signed by the key in the Keyring referenced by a given name.

Example:
  pawd gentx validator-1 10000000000upaw \
    --chain-id paw-testnet \
    --moniker "PAW Validator 1" \
    --commission-rate 0.10 \
    --commission-max-rate 0.20 \
    --commission-max-change-rate 0.01 \
    --min-self-delegation 1000000
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cdc := clientCtx.Codec

			config := serverCtx.Config
			config.SetRoot(clientCtx.HomeDir)

			if err := canonicalizeGenesisFile(config.GenesisFile()); err != nil {
				return fmt.Errorf("failed to canonicalize genesis before gentx: %w", err)
			}
			if err := forceInitialHeightString(config.GenesisFile()); err != nil {
				return fmt.Errorf("failed to enforce initial_height string encoding: %w", err)
			}

			// Initialize node validator files
			// In SDK v0.50, InitializeNodeValidatorFiles signature changed
			nodeID, valPubKey, err := genutil.InitializeNodeValidatorFiles(config)
			if err != nil {
				return fmt.Errorf("failed to initialize node validator files: %w", err)
			}

			// Read genesis file
			genDoc, err := tmtypes.GenesisDocFromFile(config.GenesisFile())
			if err != nil {
				return fmt.Errorf("failed to read genesis doc from file: %w", err)
			}

			var genesisState map[string]json.RawMessage
			if err = json.Unmarshal(genDoc.AppState, &genesisState); err != nil {
				return fmt.Errorf("failed to unmarshal genesis state: %w", err)
			}

			if err = mbm.ValidateGenesis(cdc, txEncCfg, genesisState); err != nil {
				return fmt.Errorf("failed to validate genesis state: %w", err)
			}

			// Get validator key
			keyName := args[0]
			key, err := clientCtx.Keyring.Key(keyName)
			if err != nil {
				return fmt.Errorf("failed to get key %s: %w", keyName, err)
			}

			addr, err := key.GetAddress()
			if err != nil {
				return fmt.Errorf("failed to get address: %w", err)
			}

			// Parse amount
			amount, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return fmt.Errorf("failed to parse amount: %w", err)
			}

			// Get commission rates
			rateStr, _ := cmd.Flags().GetString(flagCommissionRate)
			maxRateStr, _ := cmd.Flags().GetString(flagCommissionMaxRate)
			maxChangeRateStr, _ := cmd.Flags().GetString(flagCommissionMaxChangeRate)

			commissionRates, err := buildCommissionRates(rateStr, maxRateStr, maxChangeRateStr)
			if err != nil {
				return err
			}

			// Get min self delegation
			minSelfDelegationStr, err := cmd.Flags().GetString(flagMinSelfDelegation)
			if err != nil {
				return err
			}
			minSelfDelegation, ok := math.NewIntFromString(minSelfDelegationStr)
			if !ok {
				return fmt.Errorf("invalid min self delegation: %s", minSelfDelegationStr)
			}

			// Build validator description
			moniker, _ := cmd.Flags().GetString(flagMoniker)
			if moniker == "" {
				moniker = config.Moniker
			}

			identity, _ := cmd.Flags().GetString(flagIdentity)
			website, _ := cmd.Flags().GetString(flagWebsite)
			security, _ := cmd.Flags().GetString(flagSecurityContact)
			details, _ := cmd.Flags().GetString(flagDetails)

			description := stakingtypes.NewDescription(
				moniker,
				identity,
				website,
				security,
				details,
			)

			// Create MsgCreateValidator
			valAddr := sdk.ValAddress(addr)
			msg, err := stakingtypes.NewMsgCreateValidator(
				valAddr.String(),
				valPubKey,
				amount,
				description,
				commissionRates,
				minSelfDelegation,
			)
			if err != nil {
				return fmt.Errorf("failed to create MsgCreateValidator: %w", err)
			}

			// Ensure delegator address is populated (constructor defaults to validator address only).
			//lint:ignore SA1019 DelegatorAddress remains for MsgCreateValidator compatibility in upstream SDK.
			msg.DelegatorAddress = addr.String()

			// ValidateBasic was removed in SDK v0.50 - validation happens in message server

			// Build and sign transaction
			txBuilder := clientCtx.TxConfig.NewTxBuilder()
			if err := txBuilder.SetMsgs(msg); err != nil {
				return err
			}

			txFactory := tx.Factory{}
			txFactory = txFactory.
				WithChainID(genDoc.ChainID).
				WithMemo("").
				WithKeybase(clientCtx.Keyring).
				WithTxConfig(clientCtx.TxConfig)

			if err = tx.Sign(context.Background(), txFactory, keyName, txBuilder, true); err != nil {
				return err
			}

			txBz, err := clientCtx.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
			if err != nil {
				return fmt.Errorf("failed to encode tx: %w", err)
			}

			// Write gentx to file
			gentxDir := filepath.Join(config.RootDir, "config", "gentx")
			if err := os.MkdirAll(gentxDir, 0o750); err != nil {
				return fmt.Errorf("failed to create gentx dir: %w", err)
			}

			gentxFile := filepath.Join(gentxDir, fmt.Sprintf("gentx-%s.json", nodeID))
			if err := os.WriteFile(gentxFile, txBz, 0o600); err != nil {
				return fmt.Errorf("failed to write gentx file: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Genesis transaction written to %s\n", gentxFile)
			fmt.Fprintf(cmd.OutOrStdout(), "\nValidator details:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  Address: %s\n", valAddr)
			fmt.Fprintf(cmd.OutOrStdout(), "  Moniker: %s\n", moniker)
			fmt.Fprintf(cmd.OutOrStdout(), "  Self-delegation: %s\n", amount)
			fmt.Fprintf(cmd.OutOrStdout(), "  Commission rate: %s\n", rateStr)

			return nil
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	cmd.Flags().String(flagCommissionRate, "0.10", "The initial commission rate percentage")
	cmd.Flags().String(flagCommissionMaxRate, "0.20", "The maximum commission rate percentage")
	cmd.Flags().String(flagCommissionMaxChangeRate, "0.01", "The maximum commission change rate percentage (per day)")
	cmd.Flags().String(flagMinSelfDelegation, "1000000", "The minimum self delegation required on the validator")
	cmd.Flags().String(flagMoniker, "", "The validator's name")
	cmd.Flags().String(flagIdentity, "", "The optional identity signature (ex. UPort or Keybase)")
	cmd.Flags().String(flagWebsite, "", "The validator's (optional) website")
	cmd.Flags().String(flagSecurityContact, "", "The validator's (optional) security contact")
	cmd.Flags().String(flagDetails, "", "The validator's (optional) details")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// buildCommissionRates builds commission rates from string values
func buildCommissionRates(rateStr, maxRateStr, maxChangeRateStr string) (stakingtypes.CommissionRates, error) {
	rate, err := math.LegacyNewDecFromStr(rateStr)
	if err != nil {
		return stakingtypes.CommissionRates{}, fmt.Errorf("invalid commission rate: %w", err)
	}

	maxRate, err := math.LegacyNewDecFromStr(maxRateStr)
	if err != nil {
		return stakingtypes.CommissionRates{}, fmt.Errorf("invalid max commission rate: %w", err)
	}

	maxChangeRate, err := math.LegacyNewDecFromStr(maxChangeRateStr)
	if err != nil {
		return stakingtypes.CommissionRates{}, fmt.Errorf("invalid max change rate: %w", err)
	}

	return stakingtypes.NewCommissionRates(rate, maxRate, maxChangeRate), nil
}
