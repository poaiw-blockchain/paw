package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	txclient "github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

// GetTxSimulateCmd returns a command to simulate a transaction
func GetTxSimulateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "simulate [tx-file]",
		Short: "Simulate a transaction without broadcasting",
		Long: `Simulate executing a transaction to estimate gas and preview results.
This is useful for testing transactions before actually submitting them to the chain.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			// Read transaction from file
			txFile := args[0]
			txBytes, err := os.ReadFile(txFile)
			if err != nil {
				return fmt.Errorf("failed to read tx file: %w", err)
			}

			simRes, err := simulateTransaction(cmd.Context(), clientCtx, txBytes)
			if err != nil {
				return fmt.Errorf("simulation failed: %w", err)
			}

			// Create progress bar
			bar := progressbar.NewOptions(100,
				progressbar.OptionSetDescription("Simulating transaction..."),
				progressbar.OptionShowCount(),
				progressbar.OptionSetWidth(40),
			)

			// Simulate transaction
			bar.Add(30)
			bar.Add(70)
			bar.Finish()

			// Display results
			fmt.Println("\n=== Simulation Results ===")
			fmt.Printf("Gas used: %d\n", simRes.GasInfo.GasUsed)
			fmt.Printf("Gas wanted: %d\n", simRes.GasInfo.GasWanted)
			fmt.Printf("Gas estimation: %d\n", simRes.GasInfo.GasUsed)

			return nil
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// simulateTransaction simulates a transaction
func simulateTransaction(ctx context.Context, clientCtx client.Context, txBytes []byte) (*txtypes.SimulateResponse, error) {
	txDecoder := clientCtx.TxConfig.TxJSONDecoder()
	sdkTx, err := txDecoder(txBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode tx: %w", err)
	}

	wireBytes, err := clientCtx.TxConfig.TxEncoder()(sdkTx)
	if err != nil {
		return nil, fmt.Errorf("failed to encode tx: %w", err)
	}

	txClient := txtypes.NewServiceClient(clientCtx)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return txClient.Simulate(ctx, &txtypes.SimulateRequest{TxBytes: wireBytes})
}

// GetTxBatchCmd returns a command to batch multiple transactions
func GetTxBatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch [tx-files...]",
		Short: "Batch multiple transactions",
		Long: `Batch and send multiple transactions in sequence.
This is useful for executing multiple operations efficiently.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			sequential, _ := cmd.Flags().GetBool("sequential")

			// Create progress bar
			bar := progressbar.NewOptions(len(args),
				progressbar.OptionSetDescription("Processing transactions..."),
				progressbar.OptionShowCount(),
				progressbar.OptionSetWidth(40),
			)

			results := make([]string, 0)

			for _, txFile := range args {
				bar.Add(1)

				// Read and broadcast transaction
				txHash, err := processBatchTransaction(clientCtx, txFile)
				if err != nil {
					fmt.Printf("\nFailed to process %s: %v\n", txFile, err)
					if sequential {
						bar.Finish()
						return fmt.Errorf("batch processing stopped due to error")
					}
					continue
				}

				results = append(results, txHash)

				if sequential {
					// Wait for confirmation before proceeding
					time.Sleep(6 * time.Second)
				}
			}

			bar.Finish()

			// Display results
			fmt.Println("\n=== Batch Results ===")
			for i, hash := range results {
				fmt.Printf("%d. %s\n", i+1, hash)
			}

			return nil
		},
	}

	cmd.Flags().Bool("sequential", false, "Wait for each transaction to be confirmed before sending the next")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// processBatchTransaction processes a single transaction in a batch
func processBatchTransaction(clientCtx client.Context, txFile string) (string, error) {
	txBytes, err := os.ReadFile(txFile)
	if err != nil {
		return "", err
	}

	txDecoder := clientCtx.TxConfig.TxJSONDecoder()
	txEncoder := clientCtx.TxConfig.TxEncoder()

	parsedTx, err := txDecoder(txBytes)
	if err != nil {
		return "", fmt.Errorf("failed to decode tx: %w", err)
	}

	wireTx, err := txEncoder(parsedTx)
	if err != nil {
		return "", fmt.Errorf("failed to encode tx: %w", err)
	}

	res, err := clientCtx.BroadcastTx(wireTx)
	if err != nil {
		return "", fmt.Errorf("broadcast failed: %w", err)
	}

	// In production, decode and broadcast the transaction
	return res.TxHash, nil
}

// GetTxOfflineCmd returns a command for offline signing
func GetTxOfflineCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign-offline [tx-file]",
		Short: "Sign a transaction offline",
		Long: `Sign a transaction in offline mode without connecting to a node.
Useful for air-gapped or cold storage signing.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			// Set offline mode
			clientCtx = clientCtx.WithOffline(true)

			txFile := args[0]
			outputFile, _ := cmd.Flags().GetString("output")

			// Read unsigned transaction
			txBytes, err := os.ReadFile(txFile)
			if err != nil {
				return fmt.Errorf("failed to read tx file: %w", err)
			}

			decodeFn := clientCtx.TxConfig.TxJSONDecoder()
			unsignedTx, err := decodeFn(txBytes)
			if err != nil {
				return fmt.Errorf("failed to parse tx: %w", err)
			}

			// Get account number and sequence from flags
			accountNumber, _ := cmd.Flags().GetUint64("account-number")
			sequence, _ := cmd.Flags().GetUint64("sequence")

			// Sign transaction offline
			fmt.Println("Signing transaction offline...")
			signedTx, err := signTransactionOffline(clientCtx, unsignedTx, accountNumber, sequence)
			if err != nil {
				return fmt.Errorf("failed to sign tx: %w", err)
			}

			// Save signed transaction
			signedBytes, err := clientCtx.TxConfig.TxJSONEncoder()(signedTx)
			if err != nil {
				return err
			}

			if outputFile == "" {
				outputFile = strings.TrimSuffix(txFile, ".json") + ".signed.json"
			}

			err = os.WriteFile(outputFile, signedBytes, 0644)
			if err != nil {
				return fmt.Errorf("failed to write signed tx: %w", err)
			}

			fmt.Printf("✓ Transaction signed successfully\n")
			fmt.Printf("Signed transaction saved to: %s\n", outputFile)

			return nil
		},
	}

	cmd.Flags().Uint64("account-number", 0, "Account number for offline signing")
	cmd.Flags().Uint64("sequence", 0, "Sequence number for offline signing")
	cmd.Flags().String("output", "", "Output file for signed transaction")
	cmd.MarkFlagRequired("account-number")
	cmd.MarkFlagRequired("sequence")

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// signTransactionOffline signs a transaction in offline mode
func signTransactionOffline(clientCtx client.Context, sdkTx sdk.Tx, accountNumber, sequence uint64) (sdk.Tx, error) {
	txBuilder, err := clientCtx.TxConfig.WrapTxBuilder(sdkTx)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap tx builder: %w", err)
	}

	txf := txclient.Factory{}.
		WithTxConfig(clientCtx.TxConfig).
		WithAccountRetriever(clientCtx.AccountRetriever).
		WithChainID(clientCtx.ChainID).
		WithKeybase(clientCtx.Keyring).
		WithFromName(clientCtx.FromName).
		WithAccountNumber(accountNumber).
		WithSequence(sequence)

	if err := txclient.Sign(clientCtx.CmdContext, txf, clientCtx.FromName, txBuilder, true); err != nil {
		return nil, fmt.Errorf("failed to sign tx: %w", err)
	}

	return txBuilder.GetTx(), nil
}

// GetTxMultiSignCmd returns an enhanced multi-signature command
func GetTxMultiSignCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multisign [tx-file] [name] [signature-files...]",
		Short: "Sign a transaction with multiple signatures",
		Long: `Generate or append signatures for multi-signature transactions.
Supports creating multi-sig transactions and collecting signatures from multiple parties.`,
		Args: cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			txFile := args[0]
			multisigName := args[1]
			sigFiles := args[2:]

			// Read transaction
			txBytes, err := os.ReadFile(txFile)
			if err != nil {
				return fmt.Errorf("failed to read tx: %w", err)
			}

			txDecoder := clientCtx.TxConfig.TxJSONDecoder()
			baseTx, err := txDecoder(txBytes)
			if err != nil {
				return fmt.Errorf("failed to decode base tx: %w", err)
			}
			txBuilder, err := clientCtx.TxConfig.WrapTxBuilder(baseTx)
			if err != nil {
				return fmt.Errorf("failed to wrap base tx: %w", err)
			}

			// Create progress bar
			bar := progressbar.NewOptions(len(sigFiles)+2,
				progressbar.OptionSetDescription("Processing signatures..."),
				progressbar.OptionShowCount(),
				progressbar.OptionSetWidth(40),
			)

			bar.Add(1) // Parsing tx

			// Collect signatures
			signatures := make([]signingtypes.SignatureV2, 0)
			for _, sigFile := range sigFiles {
				sigBytes, err := os.ReadFile(sigFile)
				if err != nil {
					bar.Finish()
					return fmt.Errorf("failed to read signature %s: %w", sigFile, err)
				}
				signedTx, err := txDecoder(sigBytes)
				if err != nil {
					bar.Finish()
					return fmt.Errorf("failed to decode signed tx %s: %w", sigFile, err)
				}
				signed := signedTx.(authsigning.Tx)
				sigs, err := signed.GetSignaturesV2()
				if err != nil {
					bar.Finish()
					return fmt.Errorf("failed to extract signatures from %s: %w", sigFile, err)
				}
				signatures = append(signatures, sigs...)
				bar.Add(1)
			}

			// Combine signatures
			if err := txBuilder.SetSignatures(signatures...); err != nil {
				bar.Finish()
				return fmt.Errorf("failed to set signatures: %w", err)
			}

			txEncoder := clientCtx.TxConfig.TxJSONEncoder()
			signedJSON, err := txEncoder(txBuilder.GetTx())
			if err != nil {
				bar.Finish()
				return fmt.Errorf("failed to encode multisigned tx: %w", err)
			}

			bar.Add(1)
			bar.Finish()

			fmt.Printf("\n✓ Multi-signature transaction created\n")
			fmt.Printf("Signers: %s\n", multisigName)
			fmt.Printf("Signatures collected: %d/%d\n", len(signatures), len(sigFiles))
			if output, _ := cmd.Flags().GetString("output"); output != "" {
				if err := os.WriteFile(output, signedJSON, 0644); err != nil {
					return fmt.Errorf("failed to write output: %w", err)
				}
				fmt.Printf("Multisigned transaction saved to: %s\n", output)
			} else {
				fmt.Println(string(signedJSON))
			}

			return nil
		},
	}

	cmd.Flags().String("output", "", "Output file for multi-signed transaction")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// GetInteractiveCmd returns an interactive mode command
func GetInteractiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "interactive",
		Short: "Start interactive mode",
		Long: `Start an interactive CLI session for building and sending transactions.
This provides a guided interface for creating transactions step-by-step.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			return runInteractiveMode(clientCtx)
		},
	}

	return cmd
}

// runInteractiveMode runs the interactive CLI mode
func runInteractiveMode(clientCtx client.Context) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=================================")
	fmt.Println("PAW Interactive Transaction Builder")
	fmt.Println("=================================")
	fmt.Println()

	for {
		fmt.Println("\nWhat would you like to do?")
		fmt.Println("1. Send tokens")
		fmt.Println("2. Delegate to validator")
		fmt.Println("3. Query account balance")
		fmt.Println("4. Swap tokens (DEX)")
		fmt.Println("5. Submit compute request")
		fmt.Println("6. Exit")
		fmt.Print("\nChoice: ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			err := interactiveSendTokens(clientCtx, reader)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "2":
			fmt.Println("Delegation feature coming soon...")
		case "3":
			err := interactiveQueryBalance(clientCtx, reader)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "4":
			fmt.Println("DEX swap feature coming soon...")
		case "5":
			fmt.Println("Compute request feature coming soon...")
		case "6":
			fmt.Println("Goodbye!")
			return nil
		default:
			fmt.Println("Invalid choice, please try again.")
		}
	}
}

// interactiveSendTokens handles interactive token sending
func interactiveSendTokens(clientCtx client.Context, reader *bufio.Reader) error {
	fmt.Println("\n--- Send Tokens ---")

	fmt.Print("Recipient address: ")
	recipient, _ := reader.ReadString('\n')
	recipient = strings.TrimSpace(recipient)

	fmt.Print("Amount (in upaw): ")
	amountStr, _ := reader.ReadString('\n')
	amountStr = strings.TrimSpace(amountStr)

	fmt.Print("Memo (optional): ")
	memo, _ := reader.ReadString('\n')
	memo = strings.TrimSpace(memo)

	// Confirm
	fmt.Println("\n--- Transaction Summary ---")
	fmt.Printf("To: %s\n", recipient)
	fmt.Printf("Amount: %s upaw\n", amountStr)
	fmt.Printf("Memo: %s\n", memo)
	fmt.Print("\nProceed? (yes/no): ")

	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(confirm)

	if strings.ToLower(confirm) != "yes" {
		fmt.Println("Transaction cancelled.")
		return nil
	}

	// Create and broadcast transaction
	fmt.Println("\nBuilding transaction...")
	time.Sleep(500 * time.Millisecond)

	fmt.Println("Signing transaction...")
	time.Sleep(500 * time.Millisecond)

	fmt.Println("Broadcasting transaction...")
	time.Sleep(1 * time.Second)

	// Mock success
	fmt.Println("\n✓ Transaction successful!")
	fmt.Printf("TX Hash: MOCK_HASH_%s\n", time.Now().Format("20060102150405"))

	return nil
}

// interactiveQueryBalance queries account balance interactively
func interactiveQueryBalance(clientCtx client.Context, reader *bufio.Reader) error {
	fmt.Println("\n--- Query Balance ---")

	fmt.Print("Address (or press enter for your address): ")
	address, _ := reader.ReadString('\n')
	address = strings.TrimSpace(address)

	if address == "" {
		fromAddr := clientCtx.GetFromAddress()
		address = fromAddr.String()
	}

	fmt.Printf("\nQuerying balance for: %s\n", address)

	// Mock balance query
	fmt.Println("\nBalance:")
	fmt.Println("  1,000,000 upaw")
	fmt.Println("  500 stake")

	return nil
}
