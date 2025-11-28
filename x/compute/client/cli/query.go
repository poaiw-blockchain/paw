package cli

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/paw-chain/paw/x/compute/types"
)

// GetQueryCmd returns the cli query commands for the compute module
func GetQueryCmd() *cobra.Command {
	computeQueryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the compute module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	computeQueryCmd.AddCommand(
		GetCmdQueryParams(),
		GetCmdQueryProvider(),
		GetCmdQueryProviders(),
		GetCmdQueryActiveProviders(),
		GetCmdQueryRequest(),
		GetCmdQueryRequests(),
		GetCmdQueryRequestsByRequester(),
		GetCmdQueryRequestsByProvider(),
		GetCmdQueryRequestsByStatus(),
		GetCmdQueryResult(),
		GetCmdQueryEstimateCost(),
		// GetCmdQuerySlashRecord(),
		// GetCmdQuerySlashRecords(),
		// GetCmdQuerySlashRecordsByProvider(),
		// GetCmdQueryAppeal(),
		// GetCmdQueryAppeals(),
		// GetCmdQueryAppealsByStatus(),
		// GetCmdQueryGovernanceParams(),
	)

	return computeQueryCmd
}

// GetCmdQueryParams returns the command to query module parameters
func GetCmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the current compute module parameters",
		Long: `Query the current parameters of the compute module.

Example:
  $ pawd query compute params`,
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

// GetCmdQueryProvider returns the command to query a provider by address
func GetCmdQueryProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider [address]",
		Short: "Query provider details by address",
		Long: `Query detailed information about a registered compute provider.

Example:
  $ pawd query compute provider paw1abcdef...`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Provider(context.Background(), &types.QueryProviderRequest{
				Address: args[0],
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

/*
func GetCmdQueryDispute(queryRoute string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dispute [id]",
		Short: "Query a dispute details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			res, err := queryClient.Dispute(cmd.Context(), &types.QueryDisputeRequest{
				DisputeId: id,
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
*/

// GetCmdQueryProviders returns the command to query all providers
func GetCmdQueryProviders() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "Query all registered providers",
		Long: `Query all registered compute providers with pagination support.

Example:
  $ pawd query compute providers
  $ pawd query compute providers --limit 10 --offset 20`,
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
			res, err := queryClient.Providers(context.Background(), &types.QueryProvidersRequest{
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "providers")
	return cmd
}

// GetCmdQueryActiveProviders returns the command to query active providers
func GetCmdQueryActiveProviders() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "active-providers",
		Short: "Query all active providers",
		Long: `Query all active compute providers (providers currently accepting jobs).

Example:
  $ pawd query compute active-providers`,
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
			res, err := queryClient.ActiveProviders(context.Background(), &types.QueryActiveProvidersRequest{
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "active-providers")
	return cmd
}

// GetCmdQueryRequest returns the command to query a request by ID
func GetCmdQueryRequest() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "request [request-id]",
		Short: "Query compute request by ID",
		Long: `Query detailed information about a specific compute request.

Example:
  $ pawd query compute request 1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			requestID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid request ID: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Request(context.Background(), &types.QueryRequestRequest{
				Id: requestID,
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

// GetCmdQueryRequests returns the command to query all requests
func GetCmdQueryRequests() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "requests",
		Short: "Query all compute requests",
		Long: `Query all compute requests with pagination support.

Example:
  $ pawd query compute requests
  $ pawd query compute requests --limit 50`,
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
			res, err := queryClient.Requests(context.Background(), &types.QueryRequestsRequest{
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "requests")
	return cmd
}

// GetCmdQueryRequestsByRequester returns the command to query requests by requester
func GetCmdQueryRequestsByRequester() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "requests-by-requester [address]",
		Short: "Query all requests by a specific requester",
		Long: `Query all compute requests submitted by a specific address.

Example:
  $ pawd query compute requests-by-requester paw1abcdef...`,
		Args: cobra.ExactArgs(1),
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
			res, err := queryClient.RequestsByRequester(context.Background(), &types.QueryRequestsByRequesterRequest{
				Requester:  args[0],
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "requests")
	return cmd
}

// GetCmdQueryRequestsByProvider returns the command to query requests by provider
func GetCmdQueryRequestsByProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "requests-by-provider [address]",
		Short: "Query all requests assigned to a specific provider",
		Long: `Query all compute requests assigned to a specific provider.

Example:
  $ pawd query compute requests-by-provider paw1abcdef...`,
		Args: cobra.ExactArgs(1),
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
			res, err := queryClient.RequestsByProvider(context.Background(), &types.QueryRequestsByProviderRequest{
				Provider:   args[0],
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "requests")
	return cmd
}

// GetCmdQueryRequestsByStatus returns the command to query requests by status
func GetCmdQueryRequestsByStatus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "requests-by-status [status]",
		Short: "Query all requests with a specific status",
		Long: `Query all compute requests with a specific status.

Valid status values: pending, assigned, processing, completed, failed, cancelled, disputed

Example:
  $ pawd query compute requests-by-status pending`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			status, err := parseRequestStatus(args[0])
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.RequestsByStatus(context.Background(), &types.QueryRequestsByStatusRequest{
				Status:     status,
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "requests")
	return cmd
}

// GetCmdQueryResult returns the command to query a result
func GetCmdQueryResult() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "result [request-id]",
		Short: "Query the result of a compute request",
		Long: `Query the result (output) of a specific compute request.

Example:
  $ pawd query compute result 1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			requestID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid request ID: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Result(context.Background(), &types.QueryResultRequest{
				RequestId: requestID,
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

// GetCmdQueryEstimateCost returns the command to estimate compute cost
func GetCmdQueryEstimateCost() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "estimate-cost",
		Short: "Estimate the cost of a compute request",
		Long: `Estimate the cost of running a compute job with specified resources.

Example:
  $ pawd query compute estimate-cost --cpu-cores 4 --memory-mb 8192 --timeout-seconds 3600`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			cpuCores, err := cmd.Flags().GetUint32(FlagCpuCores)
			if err != nil {
				return err
			}

			memoryMb, err := cmd.Flags().GetUint64(FlagMemoryMb)
			if err != nil {
				return err
			}

			timeoutSeconds, err := cmd.Flags().GetUint64(FlagTimeoutSeconds)
			if err != nil {
				return err
			}

			diskMb, err := cmd.Flags().GetUint64(FlagDiskMb)
			if err != nil {
				return err
			}

			gpuUnits, err := cmd.Flags().GetUint32(FlagGpuUnits)
			if err != nil {
				return err
			}

			providerAddr, err := cmd.Flags().GetString(FlagProviderAddress)
			if err != nil {
				return err
			}

			specs := types.ComputeSpec{
				CpuCores:       uint64(cpuCores),
				MemoryMb:       memoryMb,
				StorageGb:      diskMb,
				GpuCount:       gpuUnits,
				TimeoutSeconds: timeoutSeconds,
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.EstimateCost(context.Background(), &types.QueryEstimateCostRequest{
				Specs:           specs,
				ProviderAddress: providerAddr,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cmd.Flags().Uint32(FlagCpuCores, 1, "Number of CPU cores")
	cmd.Flags().Uint64(FlagMemoryMb, 1024, "Memory in MB")
	cmd.Flags().Uint64(FlagTimeoutSeconds, 3600, "Timeout in seconds")
	cmd.Flags().Uint64(FlagDiskMb, 10240, "Disk space in MB")
	cmd.Flags().Uint32(FlagGpuUnits, 0, "Number of GPU units")
	cmd.Flags().String(FlagProviderAddress, "", "Provider address (optional)")

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

/*
// GetCmdQueryDispute returns the command to query a dispute
func GetCmdQueryDispute(queryRoute string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dispute [dispute-id]",
		Short: "Query dispute by ID",
		Long: `Query detailed information about a specific dispute.

Example:
  $ pawd query compute dispute 1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			disputeID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid dispute ID: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Dispute(cmd.Context(), &types.QueryDisputeRequest{
				DisputeId: disputeID,
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
*/

/*
// GetCmdQueryDisputes returns the command to query all disputes
func GetCmdQueryDisputes() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disputes",
		Short: "Query all disputes",
		Long: `Query all disputes with pagination support.

Example:
  $ pawd query compute disputes`,
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
			res, err := queryClient.Disputes(context.Background(), &types.QueryDisputesRequest{
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "disputes")
	return cmd
}

// GetCmdQueryDisputesByRequest returns the command to query disputes by request
func GetCmdQueryDisputesByRequest() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disputes-by-request [request-id]",
		Short: "Query all disputes for a specific request",
		Long: `Query all disputes related to a specific compute request.

Example:
  $ pawd query compute disputes-by-request 1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			requestID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid request ID: %w", err)
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.DisputesByRequest(context.Background(), &types.QueryDisputesByRequestRequest{
				RequestId:  requestID,
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "disputes")
	return cmd
}

// GetCmdQueryDisputesByStatus returns the command to query disputes by status
func GetCmdQueryDisputesByStatus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disputes-by-status [status]",
		Short: "Query all disputes with a specific status",
		Long: `Query all disputes with a specific status.

Valid status values: pending, voting, resolved_favor_requester, resolved_favor_provider, cancelled

Example:
  $ pawd query compute disputes-by-status pending`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			status, err := parseDisputeStatus(args[0])
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.DisputesByStatus(context.Background(), &types.QueryDisputesByStatusRequest{
				Status:     status,
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "disputes")
	return cmd
}

// GetCmdQueryEvidence returns the command to query evidence for a dispute
func GetCmdQueryEvidence() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "evidence [dispute-id]",
		Short: "Query evidence for a dispute",
		Long: `Query all evidence submitted for a specific dispute.

Example:
  $ pawd query compute evidence 1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			disputeID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid dispute ID: %w", err)
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Evidence(context.Background(), &types.QueryEvidenceRequest{
				DisputeId:  disputeID,
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "evidence")
	return cmd
}
*/

/*
// GetCmdQuerySlashRecord returns the command to query a slash record
func GetCmdQuerySlashRecord() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slash-record [slash-id]",
		Short: "Query slash record by ID",
		Long: `Query detailed information about a specific slash record.

Example:
  $ pawd query compute slash-record 1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			slashID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid slash ID: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.SlashRecord(context.Background(), &types.QuerySlashRecordRequest{
				SlashId: slashID,
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

// GetCmdQuerySlashRecords returns the command to query all slash records
func GetCmdQuerySlashRecords() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slash-records",
		Short: "Query all slash records",
		Long: `Query all slash records with pagination support.

Example:
  $ pawd query compute slash-records`,
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
			res, err := queryClient.SlashRecords(context.Background(), &types.QuerySlashRecordsRequest{
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "slash-records")
	return cmd
}

// GetCmdQuerySlashRecordsByProvider returns the command to query slash records by provider
func GetCmdQuerySlashRecordsByProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slash-records-by-provider [address]",
		Short: "Query all slash records for a specific provider",
		Long: `Query all slash records for a specific provider.

Example:
  $ pawd query compute slash-records-by-provider paw1abcdef...`,
		Args: cobra.ExactArgs(1),
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
			res, err := queryClient.SlashRecordsByProvider(context.Background(), &types.QuerySlashRecordsByProviderRequest{
				Provider:   args[0],
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "slash-records")
	return cmd
}

// GetCmdQueryAppeal returns the command to query an appeal
func GetCmdQueryAppeal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "appeal [appeal-id]",
		Short: "Query appeal by ID",
		Long: `Query detailed information about a specific appeal.

Example:
  $ pawd query compute appeal 1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			appealID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid appeal ID: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Appeal(context.Background(), &types.QueryAppealRequest{
				AppealId: appealID,
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
*/

/*
// GetCmdQueryAppeals returns the command to query all appeals
func GetCmdQueryAppeals() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "appeals",
		Short: "Query all appeals",
		Long: `Query all appeals with pagination support.

Example:
  $ pawd query compute appeals`,
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
			res, err := queryClient.Appeals(context.Background(), &types.QueryAppealsRequest{
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "appeals")
	return cmd
}

// GetCmdQueryAppealsByStatus returns the command to query appeals by status
func GetCmdQueryAppealsByStatus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "appeals-by-status [status]",
		Short: "Query all appeals with a specific status",
		Long: `Query all appeals with a specific status.

Valid status values: pending, voting, approved, rejected, cancelled

Example:
  $ pawd query compute appeals-by-status pending`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			status, err := parseAppealStatus(args[0])
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.AppealsByStatus(context.Background(), &types.QueryAppealsByStatusRequest{
				Status:     status,
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "appeals")
	return cmd
}

// GetCmdQueryGovernanceParams returns the command to query governance parameters
func GetCmdQueryGovernanceParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "governance-params",
		Short: "Query the current compute governance parameters",
		Long: `Query the current governance parameters for the compute module.

Example:
  $ pawd query compute governance-params`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.GovernanceParams(context.Background(), &types.QueryGovernanceParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
*/

// Helper functions for parsing enums

func parseRequestStatus(status string) (types.RequestStatus, error) {
	switch status {
	case "pending":
		return types.RequestStatus_REQUEST_STATUS_PENDING, nil
	case "assigned":
		return types.RequestStatus_REQUEST_STATUS_ASSIGNED, nil
	case "processing":
		return types.RequestStatus_REQUEST_STATUS_PROCESSING, nil
	case "completed":
		return types.RequestStatus_REQUEST_STATUS_COMPLETED, nil
	case "failed":
		return types.RequestStatus_REQUEST_STATUS_FAILED, nil
	case "cancelled":
		return types.RequestStatus_REQUEST_STATUS_CANCELLED, nil

	default:
		return 0, fmt.Errorf("invalid request status: %s (valid: pending, assigned, processing, completed, failed, cancelled, disputed)", status)
	}
}

/*
func parseDisputeStatus(status string) (types.DisputeStatus, error) {
	switch status {
	case "pending":
		return types.DisputeStatus_DISPUTE_STATUS_PENDING, nil
	case "voting":
		return types.DisputeStatus_DISPUTE_STATUS_VOTING, nil
	case "resolved_favor_requester":
		return types.DisputeStatus_DISPUTE_STATUS_RESOLVED_FAVOR_REQUESTER, nil
	case "resolved_favor_provider":
		return types.DisputeStatus_DISPUTE_STATUS_RESOLVED_FAVOR_PROVIDER, nil
	case "cancelled":
		return types.DisputeStatus_DISPUTE_STATUS_CANCELLED, nil
	default:
		return 0, fmt.Errorf("invalid dispute status: %s (valid: pending, voting, resolved_favor_requester, resolved_favor_provider, cancelled)", status)
	}
}

func parseAppealStatus(status string) (types.AppealStatus, error) {
	switch status {
	case "pending":
		return types.AppealStatus_APPEAL_STATUS_PENDING, nil
	case "voting":
		return types.AppealStatus_APPEAL_STATUS_VOTING, nil
	case "approved":
		return types.AppealStatus_APPEAL_STATUS_APPROVED, nil
	case "rejected":
		return types.AppealStatus_APPEAL_STATUS_REJECTED, nil
	case "cancelled":
		return types.AppealStatus_APPEAL_STATUS_CANCELLED, nil
	default:
		return 0, fmt.Errorf("invalid appeal status: %s (valid: pending, voting, approved, rejected, cancelled)", status)
	}
}
*/
