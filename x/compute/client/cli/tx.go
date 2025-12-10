package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// GetTxCmd returns the transaction commands for the compute module
func GetTxCmd() *cobra.Command {
	computeTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Compute transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	computeTxCmd.AddCommand(
		CmdRegisterProvider(),
		CmdUpdateProvider(),
		CmdDeactivateProvider(),
		CmdSubmitRequest(),
		CmdCancelRequest(),
		CmdSubmitResult(),
		CmdCreateDispute(),
		CmdVoteOnDispute(),
		CmdSubmitEvidence(),
		CmdAppealSlashing(),
		CmdVoteOnAppeal(),
		CmdResolveDispute(),
		CmdResolveAppeal(),
		CmdUpdateGovernanceParams(),
	)

	return computeTxCmd
}

// CmdRegisterProvider returns a CLI command handler for registering as a compute provider
func CmdRegisterProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-provider",
		Short: "Register as a compute provider",
		Long: `Register as a compute provider with specified resources and pricing.

Example:
  $ pawd tx compute register-provider \
    --moniker "MyProvider" \
    --endpoint "https://compute.github.com" \
    --cpu-cores 16 \
    --memory-mb 32768 \
    --disk-mb 512000 \
    --gpu-units 2 \
    --timeout-seconds 7200 \
    --cpu-price 100 \
    --memory-price 50 \
    --gpu-price 1000 \
    --storage-price 10 \
    --amount 1000000upaw \
    --from mykey`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Get flags
			moniker, err := cmd.Flags().GetString(FlagMoniker)
			if err != nil {
				return err
			}
			if moniker == "" {
				return fmt.Errorf("moniker cannot be empty")
			}

			endpoint, err := cmd.Flags().GetString(FlagEndpoint)
			if err != nil {
				return err
			}
			if endpoint == "" {
				return fmt.Errorf("endpoint cannot be empty")
			}

			cpuCores, err := cmd.Flags().GetUint32(FlagCpuCores)
			if err != nil {
				return err
			}

			memoryMb, err := cmd.Flags().GetUint64(FlagMemoryMb)
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

			timeoutSeconds, err := cmd.Flags().GetUint64(FlagTimeoutSeconds)
			if err != nil {
				return err
			}

			cpuPriceStr, err := cmd.Flags().GetString(FlagCpuPrice)
			if err != nil {
				return err
			}
			cpuPrice, err := math.LegacyNewDecFromStr(cpuPriceStr)
			if err != nil {
				return fmt.Errorf("invalid cpu price: %w", err)
			}

			memoryPriceStr, err := cmd.Flags().GetString(FlagMemoryPrice)
			if err != nil {
				return err
			}
			memoryPrice, err := math.LegacyNewDecFromStr(memoryPriceStr)
			if err != nil {
				return fmt.Errorf("invalid memory price: %w", err)
			}

			gpuPriceStr, err := cmd.Flags().GetString(FlagGpuPrice)
			if err != nil {
				return err
			}
			gpuPrice, err := math.LegacyNewDecFromStr(gpuPriceStr)
			if err != nil {
				return fmt.Errorf("invalid gpu price: %w", err)
			}

			storagePriceStr, err := cmd.Flags().GetString(FlagStoragePrice)
			if err != nil {
				return err
			}
			storagePrice, err := math.LegacyNewDecFromStr(storagePriceStr)
			if err != nil {
				return fmt.Errorf("invalid storage price: %w", err)
			}

			stakeStr, err := cmd.Flags().GetString("amount")
			if err != nil {
				return err
			}
			stake, ok := math.NewIntFromString(stakeStr)
			if !ok {
				return fmt.Errorf("invalid stake amount: %s", stakeStr)
			}

			specs := types.ComputeSpec{
				CpuCores:       uint64(cpuCores),
				MemoryMb:       memoryMb,
				StorageGb:      diskMb,
				GpuCount:       gpuUnits,
				TimeoutSeconds: timeoutSeconds,
			}

			pricing := types.Pricing{
				CpuPricePerMcoreHour:  cpuPrice,
				MemoryPricePerMbHour:  memoryPrice,
				GpuPricePerHour:       gpuPrice,
				StoragePricePerGbHour: storagePrice,
			}

			msg := &types.MsgRegisterProvider{
				Provider:       clientCtx.GetFromAddress().String(),
				Moniker:        moniker,
				Endpoint:       endpoint,
				AvailableSpecs: specs,
				Pricing:        pricing,
				Stake:          stake,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagMoniker, "", "Human-readable provider name")
	cmd.Flags().String(FlagEndpoint, "", "Provider service endpoint URL")
	cmd.Flags().Uint32(FlagCpuCores, 1, "Number of CPU cores available")
	cmd.Flags().Uint64(FlagMemoryMb, 1024, "Memory available in MB")
	cmd.Flags().Uint64(FlagDiskMb, 10240, "Disk space available in MB")
	cmd.Flags().Uint32(FlagGpuUnits, 0, "Number of GPU units available")
	cmd.Flags().Uint64(FlagTimeoutSeconds, 3600, "Maximum job timeout in seconds")
	cmd.Flags().String(FlagCpuPrice, "0.001", "Price per CPU mcore-hour")
	cmd.Flags().String(FlagMemoryPrice, "0.0005", "Price per MB-hour")
	cmd.Flags().String(FlagGpuPrice, "0.1", "Price per GPU-hour")
	cmd.Flags().String(FlagStoragePrice, "0.0001", "Price per GB-hour")
	cmd.Flags().String("amount", "1000000", "Stake amount (in base denomination)")

	if err := cmd.MarkFlagRequired(FlagMoniker); err != nil {
		return nil
	}
	if err := cmd.MarkFlagRequired(FlagEndpoint); err != nil {
		return nil
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdUpdateProvider returns a CLI command handler for updating provider information
func CmdUpdateProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-provider",
		Short: "Update provider information",
		Long: `Update existing provider's information (moniker, endpoint, specs, or pricing).

Example:
  $ pawd tx compute update-provider \
    --moniker "UpdatedProvider" \
    --endpoint "https://new-endpoint.github.com" \
    --cpu-cores 32 \
    --from mykey`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUpdateProvider{
				Provider: clientCtx.GetFromAddress().String(),
			}

			// Optional moniker
			if cmd.Flags().Changed(FlagMoniker) {
				moniker, err := cmd.Flags().GetString(FlagMoniker)
				if err != nil {
					return err
				}
				msg.Moniker = moniker
			}

			// Optional endpoint
			if cmd.Flags().Changed(FlagEndpoint) {
				endpoint, err := cmd.Flags().GetString(FlagEndpoint)
				if err != nil {
					return err
				}
				msg.Endpoint = endpoint
			}

			// Optional specs
			if cmd.Flags().Changed(FlagCpuCores) || cmd.Flags().Changed(FlagMemoryMb) ||
				cmd.Flags().Changed(FlagDiskMb) || cmd.Flags().Changed(FlagGpuUnits) ||
				cmd.Flags().Changed(FlagTimeoutSeconds) {

				cpuCores, _ := cmd.Flags().GetUint32(FlagCpuCores)
				memoryMb, _ := cmd.Flags().GetUint64(FlagMemoryMb)
				diskMb, _ := cmd.Flags().GetUint64(FlagDiskMb)
				gpuUnits, _ := cmd.Flags().GetUint32(FlagGpuUnits)
				timeoutSeconds, _ := cmd.Flags().GetUint64(FlagTimeoutSeconds)

				specs := types.ComputeSpec{
					CpuCores:       uint64(cpuCores),
					MemoryMb:       memoryMb,
					StorageGb:      diskMb,
					GpuCount:       gpuUnits,
					TimeoutSeconds: timeoutSeconds,
				}
				msg.AvailableSpecs = &specs
			}

			// Optional pricing
			if cmd.Flags().Changed(FlagCpuPrice) || cmd.Flags().Changed(FlagMemoryPrice) ||
				cmd.Flags().Changed(FlagGpuPrice) || cmd.Flags().Changed(FlagStoragePrice) {

				cpuPriceStr, _ := cmd.Flags().GetString(FlagCpuPrice)
				memoryPriceStr, _ := cmd.Flags().GetString(FlagMemoryPrice)
				gpuPriceStr, _ := cmd.Flags().GetString(FlagGpuPrice)
				storagePriceStr, _ := cmd.Flags().GetString(FlagStoragePrice)

				cpuPrice, _ := math.LegacyNewDecFromStr(cpuPriceStr)
				memoryPrice, _ := math.LegacyNewDecFromStr(memoryPriceStr)
				gpuPrice, _ := math.LegacyNewDecFromStr(gpuPriceStr)
				storagePrice, _ := math.LegacyNewDecFromStr(storagePriceStr)

				pricing := types.Pricing{
					CpuPricePerMcoreHour:  cpuPrice,
					MemoryPricePerMbHour:  memoryPrice,
					GpuPricePerHour:       gpuPrice,
					StoragePricePerGbHour: storagePrice,
				}
				msg.Pricing = &pricing
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagMoniker, "", "Updated provider name")
	cmd.Flags().String(FlagEndpoint, "", "Updated service endpoint URL")
	cmd.Flags().Uint32(FlagCpuCores, 0, "Updated CPU cores")
	cmd.Flags().Uint64(FlagMemoryMb, 0, "Updated memory in MB")
	cmd.Flags().Uint64(FlagDiskMb, 0, "Updated disk space in MB")
	cmd.Flags().Uint32(FlagGpuUnits, 0, "Updated GPU units")
	cmd.Flags().Uint64(FlagTimeoutSeconds, 0, "Updated timeout in seconds")
	cmd.Flags().String(FlagCpuPrice, "", "Updated CPU price")
	cmd.Flags().String(FlagMemoryPrice, "", "Updated memory price")
	cmd.Flags().String(FlagGpuPrice, "", "Updated GPU price")
	cmd.Flags().String(FlagStoragePrice, "", "Updated storage price")

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdDeactivateProvider returns a CLI command handler for deactivating a provider
func CmdDeactivateProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deactivate-provider",
		Short: "Deactivate your compute provider",
		Long: `Deactivate your compute provider (stops accepting new jobs).

Example:
  $ pawd tx compute deactivate-provider --from mykey`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgDeactivateProvider{
				Provider: clientCtx.GetFromAddress().String(),
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdSubmitRequest returns a CLI command handler for submitting a compute request
func CmdSubmitRequest() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-request",
		Short: "Submit a compute request",
		Long: `Submit a new compute request to be executed by a provider.

Example:
  $ pawd tx compute submit-request \
    --container-image "ubuntu:22.04" \
    --command "python,script.py" \
    --env-vars "KEY1=value1,KEY2=value2" \
    --cpu-cores 4 \
    --memory-mb 8192 \
    --timeout-seconds 3600 \
    --max-payment 1000000 \
    --from mykey`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			containerImage, err := cmd.Flags().GetString(FlagContainerImage)
			if err != nil {
				return err
			}
			if containerImage == "" {
				return fmt.Errorf("container image cannot be empty")
			}

			commandStr, err := cmd.Flags().GetString(FlagCommand)
			if err != nil {
				return err
			}
			var command []string
			if commandStr != "" {
				command = strings.Split(commandStr, ",")
			}

			envVarsStr, err := cmd.Flags().GetString(FlagEnvVars)
			if err != nil {
				return err
			}
			envVars := make(map[string]string)
			if envVarsStr != "" {
				pairs := strings.Split(envVarsStr, ",")
				for _, pair := range pairs {
					kv := strings.SplitN(pair, "=", 2)
					if len(kv) == 2 {
						envVars[kv[0]] = kv[1]
					}
				}
			}

			cpuCores, err := cmd.Flags().GetUint32(FlagCpuCores)
			if err != nil {
				return err
			}

			memoryMb, err := cmd.Flags().GetUint64(FlagMemoryMb)
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

			timeoutSeconds, err := cmd.Flags().GetUint64(FlagTimeoutSeconds)
			if err != nil {
				return err
			}

			maxPaymentStr, err := cmd.Flags().GetString(FlagMaxPayment)
			if err != nil {
				return err
			}
			maxPayment, ok := math.NewIntFromString(maxPaymentStr)
			if !ok {
				return fmt.Errorf("invalid max payment: %s", maxPaymentStr)
			}

			preferredProvider, err := cmd.Flags().GetString(FlagPreferredProvider)
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

			msg := &types.MsgSubmitRequest{
				Requester:         clientCtx.GetFromAddress().String(),
				Specs:             specs,
				ContainerImage:    containerImage,
				Command:           command,
				EnvVars:           envVars,
				MaxPayment:        maxPayment,
				PreferredProvider: preferredProvider,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagContainerImage, "", "Docker container image to execute")
	cmd.Flags().String(FlagCommand, "", "Command to run (comma-separated)")
	cmd.Flags().String(FlagEnvVars, "", "Environment variables (KEY=value,KEY2=value2)")
	cmd.Flags().Uint32(FlagCpuCores, 1, "Number of CPU cores required")
	cmd.Flags().Uint64(FlagMemoryMb, 1024, "Memory required in MB")
	cmd.Flags().Uint64(FlagDiskMb, 10240, "Disk space required in MB")
	cmd.Flags().Uint32(FlagGpuUnits, 0, "Number of GPU units required")
	cmd.Flags().Uint64(FlagTimeoutSeconds, 3600, "Job timeout in seconds")
	cmd.Flags().String(FlagMaxPayment, "", "Maximum payment willing to pay")
	cmd.Flags().String(FlagPreferredProvider, "", "Preferred provider address (optional)")

	if err := cmd.MarkFlagRequired(FlagContainerImage); err != nil {
		return nil
	}
	if err := cmd.MarkFlagRequired(FlagMaxPayment); err != nil {
		return nil
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdCancelRequest returns a CLI command handler for canceling a compute request
func CmdCancelRequest() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-request [request-id]",
		Short: "Cancel a pending compute request",
		Long: `Cancel a pending compute request (only pending requests can be cancelled).

Example:
  $ pawd tx compute cancel-request 123 --from mykey`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			requestID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid request ID: %w", err)
			}

			msg := &types.MsgCancelRequest{
				Requester: clientCtx.GetFromAddress().String(),
				RequestId: requestID,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdSubmitResult returns a CLI command handler for submitting a compute result
func CmdSubmitResult() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-result [request-id]",
		Short: "Submit the result for a compute request",
		Long: `Submit the execution result for a compute request (providers only).

Example:
  $ pawd tx compute submit-result 123 \
    --output-hash "abc123def456..." \
    --output-url "https://storage.github.com/result.tar.gz" \
    --exit-code 0 \
    --logs-url "https://storage.github.com/logs.txt" \
    --verification-proof proof.bin \
    --from mykey`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			requestID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid request ID: %w", err)
			}

			outputHash, err := cmd.Flags().GetString(FlagOutputHash)
			if err != nil {
				return err
			}
			if outputHash == "" {
				return fmt.Errorf("output hash cannot be empty")
			}

			outputURL, err := cmd.Flags().GetString(FlagOutputURL)
			if err != nil {
				return err
			}
			if outputURL == "" {
				return fmt.Errorf("output URL cannot be empty")
			}

			exitCode, err := cmd.Flags().GetInt32(FlagExitCode)
			if err != nil {
				return err
			}

			logsURL, err := cmd.Flags().GetString(FlagLogsURL)
			if err != nil {
				return err
			}

			var verificationProof []byte
			if cmd.Flags().Changed(FlagVerificationProof) {
				proofFile, err := cmd.Flags().GetString(FlagVerificationProof)
				if err != nil {
					return err
				}
				verificationProof, err = os.ReadFile(proofFile)
				if err != nil {
					return fmt.Errorf("failed to read verification proof file: %w", err)
				}
			}

			msg := &types.MsgSubmitResult{
				Provider:          clientCtx.GetFromAddress().String(),
				RequestId:         requestID,
				OutputHash:        outputHash,
				OutputUrl:         outputURL,
				ExitCode:          exitCode,
				LogsUrl:           logsURL,
				VerificationProof: verificationProof,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagOutputHash, "", "Hash of the computation output")
	cmd.Flags().String(FlagOutputURL, "", "URL where output can be retrieved")
	cmd.Flags().Int32(FlagExitCode, 0, "Exit code of the computation")
	cmd.Flags().String(FlagLogsURL, "", "URL where execution logs can be retrieved")
	cmd.Flags().String(FlagVerificationProof, "", "Path to verification proof file")

	if err := cmd.MarkFlagRequired(FlagOutputHash); err != nil {
		return nil
	}
	if err := cmd.MarkFlagRequired(FlagOutputURL); err != nil {
		return nil
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdCreateDispute returns a CLI command handler for creating a dispute
func CmdCreateDispute() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-dispute [request-id]",
		Short: "Create a dispute against a provider",
		Long: `Create a new dispute against a provider for a specific request.

Example:
  $ pawd tx compute create-dispute 123 \
    --reason "Incorrect computation result" \
    --evidence evidence.json \
    --deposit-amount 1000000 \
    --from mykey`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			requestID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid request ID: %w", err)
			}

			reason, err := cmd.Flags().GetString(FlagReason)
			if err != nil {
				return err
			}
			if reason == "" {
				return fmt.Errorf("reason cannot be empty")
			}

			var evidence []byte
			if cmd.Flags().Changed(FlagEvidence) {
				evidenceFile, err := cmd.Flags().GetString(FlagEvidence)
				if err != nil {
					return err
				}
				evidence, err = os.ReadFile(evidenceFile)
				if err != nil {
					return fmt.Errorf("failed to read evidence file: %w", err)
				}
			}

			depositAmountStr, err := cmd.Flags().GetString(FlagDepositAmount)
			if err != nil {
				return err
			}
			if depositAmountStr == "" {
				return fmt.Errorf("deposit amount is required")
			}

			depositAmount, ok := math.NewIntFromString(depositAmountStr)
			if !ok {
				return fmt.Errorf("invalid deposit amount: %s", depositAmountStr)
			}

			msg := &types.MsgCreateDispute{
				Requester:     clientCtx.GetFromAddress().String(),
				RequestId:     requestID,
				Reason:        reason,
				Evidence:      evidence,
				DepositAmount: depositAmount,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagReason, "", "Reason for the dispute")
	cmd.Flags().String(FlagEvidence, "", "Path to evidence file")
	cmd.Flags().String(FlagDepositAmount, "", "Deposit amount for dispute")

	if err := cmd.MarkFlagRequired(FlagReason); err != nil {
		return nil
	}
	if err := cmd.MarkFlagRequired(FlagDepositAmount); err != nil {
		return nil
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdVoteOnDispute returns a CLI command handler for voting on a dispute
func CmdVoteOnDispute() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vote-dispute [dispute-id]",
		Short: "Vote on a dispute (validators only)",
		Long: `Vote on a dispute as a validator.

Valid vote options: provider_fault, requester_fault, insufficient_evidence, no_fault

Example:
  $ pawd tx compute vote-dispute 1 \
    --vote provider_fault \
    --justification "Evidence supports requester's claim" \
    --from validator-key`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			disputeID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid dispute ID: %w", err)
			}

			voteStr, err := cmd.Flags().GetString(FlagVote)
			if err != nil {
				return err
			}
			vote, err := parseDisputeVote(voteStr)
			if err != nil {
				return err
			}

			justification, err := cmd.Flags().GetString(FlagJustification)
			if err != nil {
				return err
			}

			valAddr := sdk.ValAddress(clientCtx.GetFromAddress())

			msg := &types.MsgVoteOnDispute{
				Validator:     valAddr.String(),
				DisputeId:     disputeID,
				Vote:          vote,
				Justification: justification,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagVote, "", "Vote option (provider_fault, requester_fault, insufficient_evidence, no_fault)")
	cmd.Flags().String(FlagJustification, "", "Justification for the vote")

	if err := cmd.MarkFlagRequired(FlagVote); err != nil {
		return nil
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdSubmitEvidence returns a CLI command handler for submitting evidence
func CmdSubmitEvidence() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-evidence [dispute-id]",
		Short: "Submit evidence for a dispute",
		Long: `Submit additional evidence for an ongoing dispute.

Example:
  $ pawd tx compute submit-evidence 1 \
    --evidence evidence.json \
    --reason "Additional proof of computation error" \
    --from mykey`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			disputeID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid dispute ID: %w", err)
			}

			evidenceFile, err := cmd.Flags().GetString(FlagEvidence)
			if err != nil {
				return err
			}
			if evidenceFile == "" {
				return fmt.Errorf("evidence file cannot be empty")
			}

			evidence, err := os.ReadFile(evidenceFile)
			if err != nil {
				return fmt.Errorf("failed to read evidence file: %w", err)
			}

			reason, err := cmd.Flags().GetString(FlagReason)
			if err != nil {
				return err
			}

			msg := &types.MsgSubmitEvidence{
				Submitter:    clientCtx.GetFromAddress().String(),
				DisputeId:    disputeID,
				EvidenceType: "json",
				Data:         evidence,
				Description:  reason,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagEvidence, "", "Path to evidence file")
	cmd.Flags().String(FlagReason, "", "Description of the evidence")

	if err := cmd.MarkFlagRequired(FlagEvidence); err != nil {
		return nil
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdAppealSlashing returns a CLI command handler for appealing a slash
func CmdAppealSlashing() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "appeal-slashing [slash-id]",
		Short: "Appeal a slashing event",
		Long: `Appeal a slashing event as a provider.

Example:
  $ pawd tx compute appeal-slashing 1 \
    --justification "Slash was unjustified due to network issues" \
    --deposit-amount 1000000 \
    --from provider-key`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			slashID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid slash ID: %w", err)
			}

			justification, err := cmd.Flags().GetString(FlagJustification)
			if err != nil {
				return err
			}
			if justification == "" {
				return fmt.Errorf("justification cannot be empty")
			}

			depositAmountStr, err := cmd.Flags().GetString(FlagDepositAmount)
			if err != nil {
				return err
			}
			if depositAmountStr == "" {
				return fmt.Errorf("deposit amount is required")
			}

			depositAmount, ok := math.NewIntFromString(depositAmountStr)
			if !ok {
				return fmt.Errorf("invalid deposit amount: %s", depositAmountStr)
			}

			msg := &types.MsgAppealSlashing{
				Provider:      clientCtx.GetFromAddress().String(),
				SlashId:       slashID,
				Justification: justification,
				DepositAmount: depositAmount,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagJustification, "", "Justification for the appeal")
	cmd.Flags().String(FlagDepositAmount, "", "Deposit amount for appeal")

	if err := cmd.MarkFlagRequired(FlagJustification); err != nil {
		return nil
	}
	if err := cmd.MarkFlagRequired(FlagDepositAmount); err != nil {
		return nil
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdVoteOnAppeal returns a CLI command handler for voting on an appeal
func CmdVoteOnAppeal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vote-appeal [appeal-id]",
		Short: "Vote on an appeal (validators only)",
		Long: `Vote on an appeal as a validator.

Example:
  $ pawd tx compute vote-appeal 1 \
    --vote approve \
    --justification "Provider's argument is valid" \
    --from validator-key`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			appealID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid appeal ID: %w", err)
			}

			voteStr, err := cmd.Flags().GetString(FlagVote)
			if err != nil {
				return err
			}
			if voteStr == "" {
				return fmt.Errorf("vote option is required")
			}
			var approve bool
			switch strings.ToLower(voteStr) {
			case "approve", "yes":
				approve = true
			case "reject", "no", "deny":
				approve = false
			default:
				return fmt.Errorf("invalid vote option: %s (valid: approve, reject)", voteStr)
			}

			justification, err := cmd.Flags().GetString(FlagJustification)
			if err != nil {
				return err
			}

			valAddr := sdk.ValAddress(clientCtx.GetFromAddress())

			msg := &types.MsgVoteOnAppeal{
				Validator:     valAddr.String(),
				AppealId:      appealID,
				Approve:       approve,
				Justification: justification,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagVote, "", "Vote option (approve or reject)")
	cmd.Flags().String(FlagJustification, "", "Justification for the vote")

	if err := cmd.MarkFlagRequired(FlagVote); err != nil {
		return nil
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdResolveDispute finalizes a dispute and triggers settlement.
func CmdResolveDispute() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve-dispute [dispute-id]",
		Short: "Resolve a dispute (governance/authority only)",
		Long: `Resolve a dispute and apply the resulting slashing/refund actions.

Example:
  $ pawd tx compute resolve-dispute 7 --from gov-key`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			disputeID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid dispute ID: %w", err)
			}

			msg := &types.MsgResolveDispute{
				Authority: clientCtx.GetFromAddress().String(),
				DisputeId: disputeID,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdResolveAppeal finalizes an appeal result.
func CmdResolveAppeal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resolve-appeal [appeal-id]",
		Short: "Resolve an appeal (governance/authority only)",
		Long: `Resolve an appeal and apply the referenced slash refund if approved.

Example:
  $ pawd tx compute resolve-appeal 3 --from gov-key`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			appealID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid appeal ID: %w", err)
			}

			msg := &types.MsgResolveAppeal{
				Authority: clientCtx.GetFromAddress().String(),
				AppealId:  appealID,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdUpdateGovernanceParams updates dispute/appeal governance settings.
func CmdUpdateGovernanceParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-governance-params",
		Short: "Update dispute/appeal governance parameters (authority only)",
		Long: `Update compute governance parameters controlling dispute deposits, voting periods, and thresholds.

Example:
  $ pawd tx compute update-governance-params \
      --dispute-deposit 2000000 \
      --quorum 0.4 \
      --from gov-key`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			params := types.DefaultGovernanceParams()
			if res, err := queryClient.GovernanceParams(context.Background(), &types.QueryGovernanceParamsRequest{}); err == nil {
				params = res.Params
			}

			if cmd.Flags().Changed(FlagDisputeDeposit) {
				value, err := cmd.Flags().GetString(FlagDisputeDeposit)
				if err != nil {
					return err
				}
				amount, ok := math.NewIntFromString(value)
				if !ok {
					return fmt.Errorf("invalid dispute deposit: %s", value)
				}
				params.DisputeDeposit = amount
			}

			if cmd.Flags().Changed(FlagEvidencePeriod) {
				value, err := cmd.Flags().GetUint64(FlagEvidencePeriod)
				if err != nil {
					return err
				}
				params.EvidencePeriodSeconds = value
			}

			if cmd.Flags().Changed(FlagVotingPeriod) {
				value, err := cmd.Flags().GetUint64(FlagVotingPeriod)
				if err != nil {
					return err
				}
				params.VotingPeriodSeconds = value
			}

			if cmd.Flags().Changed(FlagQuorumPercentage) {
				value, err := cmd.Flags().GetString(FlagQuorumPercentage)
				if err != nil {
					return err
				}
				dec, err := math.LegacyNewDecFromStr(value)
				if err != nil {
					return fmt.Errorf("invalid quorum value: %w", err)
				}
				params.QuorumPercentage = dec
			}

			if cmd.Flags().Changed(FlagConsensusThreshold) {
				value, err := cmd.Flags().GetString(FlagConsensusThreshold)
				if err != nil {
					return err
				}
				dec, err := math.LegacyNewDecFromStr(value)
				if err != nil {
					return fmt.Errorf("invalid consensus threshold: %w", err)
				}
				params.ConsensusThreshold = dec
			}

			if cmd.Flags().Changed(FlagSlashPercentage) {
				value, err := cmd.Flags().GetString(FlagSlashPercentage)
				if err != nil {
					return err
				}
				dec, err := math.LegacyNewDecFromStr(value)
				if err != nil {
					return fmt.Errorf("invalid slash percentage: %w", err)
				}
				params.SlashPercentage = dec
			}

			if cmd.Flags().Changed(FlagAppealDepositPercentage) {
				value, err := cmd.Flags().GetString(FlagAppealDepositPercentage)
				if err != nil {
					return err
				}
				dec, err := math.LegacyNewDecFromStr(value)
				if err != nil {
					return fmt.Errorf("invalid appeal deposit percentage: %w", err)
				}
				params.AppealDepositPercentage = dec
			}

			if cmd.Flags().Changed(FlagMaxEvidenceSize) {
				value, err := cmd.Flags().GetUint64(FlagMaxEvidenceSize)
				if err != nil {
					return err
				}
				params.MaxEvidenceSize = value
			}

			msg := &types.MsgUpdateGovernanceParams{
				Authority: clientCtx.GetFromAddress().String(),
				Params:    params,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagDisputeDeposit, "", "Dispute deposit amount (int, upaw)")
	cmd.Flags().Uint64(FlagEvidencePeriod, 0, "Evidence period in seconds")
	cmd.Flags().Uint64(FlagVotingPeriod, 0, "Voting period in seconds")
	cmd.Flags().String(FlagQuorumPercentage, "", "Quorum percentage (e.g. 0.334)")
	cmd.Flags().String(FlagConsensusThreshold, "", "Consensus threshold (e.g. 0.5)")
	cmd.Flags().String(FlagSlashPercentage, "", "Slash percentage to apply to provider stake")
	cmd.Flags().String(FlagAppealDepositPercentage, "", "Appeal deposit percentage (fraction of slash amount)")
	cmd.Flags().Uint64(FlagMaxEvidenceSize, 0, "Maximum evidence size in bytes")

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// Helper functions

func parseDisputeVote(vote string) (types.DisputeVoteOption, error) {
	switch strings.ToLower(vote) {
	case "provider_fault":
		return types.DISPUTE_VOTE_PROVIDER_FAULT, nil
	case "requester_fault":
		return types.DISPUTE_VOTE_REQUESTER_FAULT, nil
	case "insufficient_evidence":
		return types.DISPUTE_VOTE_INSUFFICIENT_EVIDENCE, nil
	case "no_fault":
		return types.DISPUTE_VOTE_NO_FAULT, nil
	default:
		return 0, fmt.Errorf("invalid vote option: %s (valid: provider_fault, requester_fault, insufficient_evidence, no_fault)", vote)
	}
}
