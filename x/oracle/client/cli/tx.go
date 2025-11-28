package cli

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

// GetTxCmd returns the transaction commands for the oracle module
func GetTxCmd() *cobra.Command {
	oracleTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Oracle transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	oracleTxCmd.AddCommand(
		CmdSubmitPrice(),
		CmdDelegateFeedConsent(),
	)

	return oracleTxCmd
}

// CmdSubmitPrice returns a CLI command handler for submitting a price
func CmdSubmitPrice() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-price [validator] [asset] [price]",
		Short: "Submit a price for an asset",
		Long: `Submit a price feed update for a specific asset as a validator or delegated feeder.

The price should be submitted as a decimal value (e.g., 50000.5 for BTC).
IMPORTANT: Prices are internally stored as integers with 18 decimal places to avoid
floating point issues. The CLI will convert your decimal input appropriately.

The validator argument should be the validator's operator address (pawvaloper...).
If you are a delegated feeder, you must use the validator address that delegated to you.

Examples:
  Validator submitting directly:
  $ pawd tx oracle submit-price pawvaloper1abcdef... BTC 50000.5 --from validator-key

  Delegated feeder submitting:
  $ pawd tx oracle submit-price pawvaloper1xyz... ETH 3000.25 --from feeder-key

  Multiple decimal places:
  $ pawd tx oracle submit-price pawvaloper1abc... ATOM 12.345678 --from validator-key`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Validate validator address
			validatorAddr := args[0]
			if _, err := sdk.ValAddressFromBech32(validatorAddr); err != nil {
				return fmt.Errorf("invalid validator address %s: %w", validatorAddr, err)
			}

			// Parse asset
			asset := args[1]
			if asset == "" {
				return fmt.Errorf("asset cannot be empty")
			}

			// Parse price as decimal
			priceStr := args[2]
			price, err := math.LegacyNewDecFromStr(priceStr)
			if err != nil {
				return fmt.Errorf("invalid price %s: %w (must be a decimal number)", priceStr, err)
			}

			if price.IsNil() || price.LTE(math.LegacyZeroDec()) {
				return fmt.Errorf("price must be positive, got: %s", priceStr)
			}

			// Feeder is the signer
			feederAddr := clientCtx.GetFromAddress().String()

			msg := types.NewMsgSubmitPrice(
				validatorAddr,
				feederAddr,
				asset,
				price,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdDelegateFeedConsent returns a CLI command handler for delegating price feed consent
func CmdDelegateFeedConsent() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegate-feeder [delegate-address]",
		Short: "Delegate price feed submission to another address",
		Long: `Delegate the right to submit price feeds to another address (validators only).

This allows you to use a separate hot wallet for price submissions while keeping
your validator key secure.

To revoke delegation, delegate to your own validator's account address.

Examples:
  Delegate to a feeder:
  $ pawd tx oracle delegate-feeder paw1feederaddr... --from validator-key

  Revoke delegation (delegate to self):
  $ pawd tx oracle delegate-feeder paw1validatoraddr... --from validator-key`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Validate delegate address
			delegateAddr := args[0]
			if _, err := sdk.AccAddressFromBech32(delegateAddr); err != nil {
				return fmt.Errorf("invalid delegate address %s: %w", delegateAddr, err)
			}

			// The signer must be the validator (as account address)
			// We need to convert it to validator address
			validatorAddr := sdk.ValAddress(clientCtx.GetFromAddress())

			msg := types.NewMsgDelegateFeedConsent(
				validatorAddr.String(),
				delegateAddr,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
