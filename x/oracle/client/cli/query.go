package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/paw-chain/paw/x/oracle/types"
)

// GetQueryCmd returns the cli query commands for the oracle module
func GetQueryCmd() *cobra.Command {
	oracleQueryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the oracle module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	oracleQueryCmd.AddCommand(
		GetCmdQueryParams(),
		GetCmdQueryPrice(),
		GetCmdQueryPrices(),
		GetCmdQueryValidator(),
		GetCmdQueryValidators(),
		GetCmdQueryValidatorPrice(),
	)

	return oracleQueryCmd
}

// GetCmdQueryParams returns the command to query module parameters
func GetCmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the current oracle module parameters",
		Long: `Query the current parameters of the oracle module including vote period and thresholds.

Example:
  $ pawd query oracle params`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Params(context.Background(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryPrice returns the command to query a price for an asset
func GetCmdQueryPrice() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "price [asset]",
		Short: "Query the current price for an asset",
		Long: `Query the current aggregated price for a specific asset.

Example:
  $ pawd query oracle price BTC
  $ pawd query oracle price ETH`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Price(context.Background(), &types.QueryPriceRequest{
				Asset: args[0],
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryPrices returns the command to query all prices
func GetCmdQueryPrices() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prices",
		Short: "Query all current prices",
		Long: `Query all current aggregated prices for all assets with pagination support.

Example:
  $ pawd query oracle prices
  $ pawd query oracle prices --limit 10`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Prices(context.Background(), &types.QueryPricesRequest{
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "prices")
	return cmd
}

// GetCmdQueryValidator returns the command to query oracle validator information
func GetCmdQueryValidator() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator [validator-address]",
		Short: "Query oracle validator information",
		Long: `Query information about a validator's oracle participation including miss counter.

Example:
  $ pawd query oracle validator pawvaloper1abcdef...`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Validator(context.Background(), &types.QueryValidatorRequest{
				ValidatorAddr: args[0],
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryValidators returns the command to query all oracle validators
func GetCmdQueryValidators() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validators",
		Short: "Query all oracle validators",
		Long: `Query information about all validators participating in the oracle with pagination support.

Example:
  $ pawd query oracle validators
  $ pawd query oracle validators --limit 20`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Validators(context.Background(), &types.QueryValidatorsRequest{
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "validators")
	return cmd
}

// GetCmdQueryValidatorPrice returns the command to query a validator's submitted price
func GetCmdQueryValidatorPrice() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator-price [validator-address] [asset]",
		Short: "Query a validator's submitted price for an asset",
		Long: `Query the price submitted by a specific validator for a specific asset.

Example:
  $ pawd query oracle validator-price pawvaloper1abcdef... BTC`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.ValidatorPrice(context.Background(), &types.QueryValidatorPriceRequest{
				ValidatorAddr: args[0],
				Asset:         args[1],
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
