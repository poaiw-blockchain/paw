//go:build paw_examples
// +build paw_examples

package main

/*
PAW Compute Provider - Basic Implementation

This example demonstrates a complete compute provider implementation including:
- Provider registration
- Job polling and processing
- Result submission
- Error handling and recovery

Usage:
    go run provider.go [flags]

Environment Variables:
    PAW_RPC_ENDPOINT - RPC endpoint URL (default: http://localhost:26657)
    PAW_GRPC_ENDPOINT - gRPC endpoint URL (default: localhost:9090)
    PAW_PROVIDER_MNEMONIC - Provider wallet mnemonic (required)
    PAW_PROVIDER_ENDPOINT - Provider compute service URL (default: http://localhost:8080)
*/

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	computetypes "github.com/paw-chain/paw/x/compute/types"
)

// ProviderConfig holds provider configuration
type ProviderConfig struct {
	RPCEndpoint  string
	GRPCEndpoint string
	ChainID      string
	Mnemonic     string
	ProviderAddr string
	Moniker      string
	Endpoint     string
	Stake        math.Int
	PollInterval time.Duration
}

// ComputeProvider implements a compute resource provider
type ComputeProvider struct {
	config      ProviderConfig
	clientCtx   client.Context
	queryClient computetypes.QueryClient
	grpcConn    *grpc.ClientConn
}

// NewComputeProvider creates a new compute provider instance
func NewComputeProvider(cfg ProviderConfig) (*ComputeProvider, error) {
	// Create codec
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	sdk.RegisterInterfaces(interfaceRegistry)
	computetypes.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// Create keyring and import mnemonic
	kr := keyring.NewInMemory(cdc)

	// Derive key from mnemonic
	derivedPriv, err := hd.Secp256k1.Derive()(cfg.Mnemonic, "", sdk.GetConfig().GetFullBIP44Path())
	if err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	privKey := hd.Secp256k1.Generate()(derivedPriv)

	// Import key into keyring
	err = kr.ImportPrivKey("provider", string(privKey.Bytes()), hd.Secp256k1Type)
	if err != nil {
		return nil, fmt.Errorf("failed to import key: %w", err)
	}

	// Create gRPC connection
	grpcConn, err := grpc.Dial(
		cfg.GRPCEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	// Create client context
	clientCtx := client.Context{}.
		WithCodec(cdc).
		WithInterfaceRegistry(interfaceRegistry).
		WithTxConfig(authtx.NewTxConfig(cdc, authtx.DefaultSignModes)).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithBroadcastMode("sync").
		WithChainID(cfg.ChainID).
		WithKeyring(kr).
		WithGRPCClient(grpcConn)

	// Create query client
	queryClient := computetypes.NewQueryClient(grpcConn)

	return &ComputeProvider{
		config:      cfg,
		clientCtx:   clientCtx,
		queryClient: queryClient,
		grpcConn:    grpcConn,
	}, nil
}

// Close cleans up provider resources
func (cp *ComputeProvider) Close() error {
	if cp.grpcConn != nil {
		return cp.grpcConn.Close()
	}
	return nil
}

// Register registers the provider on-chain
func (cp *ComputeProvider) Register() error {
	fmt.Println("Registering compute provider...")

	// Create registration message
	msg := &computetypes.MsgRegisterProvider{
		Provider: cp.config.ProviderAddr,
		Moniker:  cp.config.Moniker,
		Endpoint: cp.config.Endpoint,
		AvailableSpecs: computetypes.ComputeSpec{
			CpuCores:       4000, // 4 cores (in millicores)
			MemoryMb:       8192, // 8GB RAM
			GpuCount:       0,    // No GPU
			StorageGb:      100,  // 100GB storage
			TimeoutSeconds: 3600, // 1 hour max
		},
		Pricing: computetypes.Pricing{
			CpuPricePerMcoreHour:  math.NewInt(100), // 100 tokens per mcore-hour
			MemoryPricePerMbHour:  math.NewInt(10),  // 10 tokens per MB-hour
			GpuPricePerHour:       math.NewInt(0),   // No GPU pricing
			StoragePricePerGbHour: math.NewInt(5),   // 5 tokens per GB-hour
		},
		Stake: cp.config.Stake,
	}

	// Broadcast transaction
	if err := cp.broadcastTx(msg); err != nil {
		return fmt.Errorf("failed to register provider: %w", err)
	}

	fmt.Printf("✓ Provider registered successfully: %s\n", cp.config.ProviderAddr)
	return nil
}

// PollJobs continuously polls for new compute jobs
func (cp *ComputeProvider) PollJobs(ctx context.Context) error {
	fmt.Printf("Starting job polling (interval: %v)...\n", cp.config.PollInterval)

	ticker := time.NewTicker(cp.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Job polling stopped")
			return nil
		case <-ticker.C:
			if err := cp.checkForJobs(ctx); err != nil {
				fmt.Printf("⚠ Error checking for jobs: %v\n", err)
			}
		}
	}
}

// checkForJobs queries for pending jobs assigned to this provider
func (cp *ComputeProvider) checkForJobs(ctx context.Context) error {
	// Query for requests assigned to this provider
	resp, err := cp.queryClient.ProviderRequests(ctx, &computetypes.QueryProviderRequestsRequest{
		Provider: cp.config.ProviderAddr,
	})
	if err != nil {
		return fmt.Errorf("failed to query requests: %w", err)
	}

	// Process each pending request
	for _, req := range resp.Requests {
		if req.Status == computetypes.REQUEST_STATUS_ASSIGNED {
			fmt.Printf("📦 Processing job %d...\n", req.Id)
			if err := cp.processJob(ctx, req); err != nil {
				fmt.Printf("✗ Failed to process job %d: %v\n", req.Id, err)
			}
		}
	}

	return nil
}

// processJob executes a compute job and submits the result
func (cp *ComputeProvider) processJob(ctx context.Context, req *computetypes.Request) error {
	// Simulate compute work
	// In a real implementation, this would:
	// 1. Pull the container image
	// 2. Run the container with the specified command
	// 3. Collect the output
	// 4. Upload results to IPFS or object storage

	fmt.Printf("  Container: %s\n", req.ContainerImage)
	fmt.Printf("  Command: %s\n", req.Command)
	fmt.Printf("  CPU: %d millicores, Memory: %d MB\n", req.Specs.CpuCores, req.Specs.MemoryMb)

	time.Sleep(2 * time.Second) // Simulate work

	// Generate mock output
	outputData := []byte(fmt.Sprintf("Job %d completed successfully at %s", req.Id, time.Now()))
	outputHash := hashData(outputData)

	// In production, upload to IPFS/S3 and get URL
	outputURL := fmt.Sprintf("https://storage.example.com/results/%d", req.Id)

	// Submit result
	return cp.submitResult(req.Id, outputHash, outputURL)
}

// submitResult submits a compute job result
func (cp *ComputeProvider) submitResult(requestID uint64, outputHash, outputURL string) error {
	fmt.Printf("📤 Submitting result for job %d...\n", requestID)

	msg := &computetypes.MsgSubmitResult{
		Provider:   cp.config.ProviderAddr,
		RequestId:  requestID,
		OutputHash: outputHash,
		OutputUrl:  outputURL,
	}

	if err := cp.broadcastTx(msg); err != nil {
		return fmt.Errorf("failed to submit result: %w", err)
	}

	fmt.Printf("✓ Result submitted successfully for job %d\n", requestID)
	return nil
}

// broadcastTx broadcasts a transaction to the chain
func (cp *ComputeProvider) broadcastTx(msg sdk.Msg) error {
	// Get provider account
	key, err := cp.clientCtx.Keyring.Key("provider")
	if err != nil {
		return fmt.Errorf("failed to get key: %w", err)
	}

	addr, err := key.GetAddress()
	if err != nil {
		return fmt.Errorf("failed to get address: %w", err)
	}

	// Create transaction factory
	txf := tx.Factory{}.
		WithKeybase(cp.clientCtx.Keyring).
		WithTxConfig(cp.clientCtx.TxConfig).
		WithAccountRetriever(cp.clientCtx.AccountRetriever).
		WithChainID(cp.config.ChainID).
		WithGas(200000).
		WithGasAdjustment(1.2).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)

	// Build and sign transaction
	txBuilder, err := txf.BuildUnsignedTx(msg)
	if err != nil {
		return fmt.Errorf("failed to build unsigned tx: %w", err)
	}

	if err := tx.Sign(txf, "provider", txBuilder, true); err != nil {
		return fmt.Errorf("failed to sign tx: %w", err)
	}

	// Encode transaction
	txBytes, err := cp.clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return fmt.Errorf("failed to encode tx: %w", err)
	}

	// Broadcast transaction
	res, err := cp.clientCtx.BroadcastTx(txBytes)
	if err != nil {
		return fmt.Errorf("failed to broadcast tx: %w", err)
	}

	if res.Code != 0 {
		return fmt.Errorf("tx failed with code %d: %s", res.Code, res.RawLog)
	}

	return nil
}

// hashData computes SHA-256 hash of data
func hashData(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// LoadConfig loads configuration from environment
func LoadConfig() ProviderConfig {
	rpcEndpoint := os.Getenv("PAW_RPC_ENDPOINT")
	if rpcEndpoint == "" {
		rpcEndpoint = "http://localhost:26657"
	}

	grpcEndpoint := os.Getenv("PAW_GRPC_ENDPOINT")
	if grpcEndpoint == "" {
		grpcEndpoint = "localhost:9090"
	}

	mnemonic := os.Getenv("PAW_PROVIDER_MNEMONIC")
	if mnemonic == "" {
		fmt.Fprintln(os.Stderr, "PAW_PROVIDER_MNEMONIC is required")
		os.Exit(1)
	}

	endpoint := os.Getenv("PAW_PROVIDER_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8080"
	}

	return ProviderConfig{
		RPCEndpoint:  rpcEndpoint,
		GRPCEndpoint: grpcEndpoint,
		ChainID:      "paw-mvp-1",
		Mnemonic:     mnemonic,
		ProviderAddr: "", // Will be derived from mnemonic
		Moniker:      "Example Provider",
		Endpoint:     endpoint,
		Stake:        math.NewInt(1000000), // 1M tokens
		PollInterval: 5 * time.Second,
	}
}

func main() {
	cfg := LoadConfig()

	// Create provider
	provider, err := NewComputeProvider(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Failed to create provider: %v\n", err)
		os.Exit(1)
	}
	defer provider.Close()

	// Register provider
	if err := provider.Register(); err != nil {
		fmt.Fprintf(os.Stderr, "✗ Failed to register: %v\n", err)
		os.Exit(1)
	}

	// Start job polling
	ctx := context.Background()
	if err := provider.PollJobs(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "✗ Job polling error: %v\n", err)
		os.Exit(1)
	}
}
