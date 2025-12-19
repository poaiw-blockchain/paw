//go:build paw_examples
// +build paw_examples

package main

/*
PAW Compute Provider - Registration Example

This example demonstrates how to register as a compute provider on the PAW blockchain.

Usage:
    go run register.go

Environment Variables:
    PAW_GRPC_ENDPOINT - gRPC endpoint (default: localhost:9090)
    PAW_CHAIN_ID - Chain ID (default: paw-testnet-1)
    PAW_PROVIDER_MNEMONIC - Provider mnemonic (required)
*/

import (
	"context"
	"fmt"
	"os"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	computetypes "github.com/paw-chain/paw/x/compute/types"
)

func main() {
	// Load configuration
	grpcEndpoint := getEnv("PAW_GRPC_ENDPOINT", "localhost:9090")
	chainID := getEnv("PAW_CHAIN_ID", "paw-testnet-1")
	mnemonic := os.Getenv("PAW_PROVIDER_MNEMONIC")

	if mnemonic == "" {
		fmt.Fprintln(os.Stderr, "✗ PAW_PROVIDER_MNEMONIC is required")
		os.Exit(1)
	}

	fmt.Println("PAW Compute Provider Registration")
	fmt.Println("==================================")
	fmt.Printf("Chain ID: %s\n", chainID)
	fmt.Printf("gRPC: %s\n\n", grpcEndpoint)

	// Setup codec
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	sdk.RegisterInterfaces(interfaceRegistry)
	computetypes.RegisterInterfaces(interfaceRegistry)
	authtypes.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// Create keyring
	kr := keyring.NewInMemory(cdc)

	// Derive key from mnemonic
	derivedPriv, err := hd.Secp256k1.Derive()(mnemonic, "", sdk.GetConfig().GetFullBIP44Path())
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Failed to derive key: %v\n", err)
		os.Exit(1)
	}

	privKey := hd.Secp256k1.Generate()(derivedPriv)
	pubKey := privKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	fmt.Printf("Provider Address: %s\n\n", addr.String())

	// Import into keyring
	err = kr.ImportPrivKey("provider", string(privKey.Bytes()), hd.Secp256k1Type)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Failed to import key: %v\n", err)
		os.Exit(1)
	}

	// Create gRPC connection
	grpcConn, err := grpc.Dial(
		grpcEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Failed to connect to gRPC: %v\n", err)
		os.Exit(1)
	}
	defer grpcConn.Close()

	// Create client context
	clientCtx := client.Context{}.
		WithCodec(cdc).
		WithInterfaceRegistry(interfaceRegistry).
		WithTxConfig(authtx.NewTxConfig(cdc, authtx.DefaultSignModes)).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithBroadcastMode("sync").
		WithChainID(chainID).
		WithKeyring(kr).
		WithGRPCClient(grpcConn)

	// Create registration message
	msg := &computetypes.MsgRegisterProvider{
		Provider: addr.String(),
		Moniker:  "Example Provider",
		Endpoint: "http://localhost:8080",
		AvailableSpecs: computetypes.ComputeSpec{
			CpuCores:       8000,  // 8 cores
			MemoryMb:       16384, // 16GB RAM
			GpuCount:       1,     // 1 GPU
			GpuType:        "NVIDIA RTX 4090",
			StorageGb:      500,   // 500GB storage
			TimeoutSeconds: 7200,  // 2 hours max
		},
		Pricing: computetypes.Pricing{
			CpuPricePerMcoreHour:  math.NewInt(150),  // 150 tokens per mcore-hour
			MemoryPricePerMbHour:  math.NewInt(15),   // 15 tokens per MB-hour
			GpuPricePerHour:       math.NewInt(5000), // 5000 tokens per GPU-hour
			StoragePricePerGbHour: math.NewInt(8),    // 8 tokens per GB-hour
		},
		Stake: math.NewInt(2000000), // 2M tokens stake
	}

	fmt.Println("Registration Details:")
	fmt.Printf("  Moniker: %s\n", msg.Moniker)
	fmt.Printf("  Endpoint: %s\n", msg.Endpoint)
	fmt.Printf("  CPU Cores: %d millicores\n", msg.AvailableSpecs.CpuCores)
	fmt.Printf("  Memory: %d MB\n", msg.AvailableSpecs.MemoryMb)
	fmt.Printf("  GPUs: %d (%s)\n", msg.AvailableSpecs.GpuCount, msg.AvailableSpecs.GpuType)
	fmt.Printf("  Storage: %d GB\n", msg.AvailableSpecs.StorageGb)
	fmt.Printf("  Stake: %s\n\n", msg.Stake.String())

	// Validate message
	if err := msg.ValidateBasic(); err != nil {
		fmt.Fprintf(os.Stderr, "✗ Invalid message: %v\n", err)
		os.Exit(1)
	}

	// Create transaction factory
	txf := tx.Factory{}.
		WithKeybase(clientCtx.Keyring).
		WithTxConfig(clientCtx.TxConfig).
		WithAccountRetriever(clientCtx.AccountRetriever).
		WithChainID(chainID).
		WithGas(300000).
		WithGasAdjustment(1.3).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)

	// Get account number and sequence
	key, err := clientCtx.Keyring.Key("provider")
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Failed to get key: %v\n", err)
		os.Exit(1)
	}

	accAddr, err := key.GetAddress()
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Failed to get address: %v\n", err)
		os.Exit(1)
	}

	accNum, accSeq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, accAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Failed to get account info: %v\n", err)
		os.Exit(1)
	}

	txf = txf.WithAccountNumber(accNum).WithSequence(accSeq)

	// Build unsigned transaction
	txBuilder, err := txf.BuildUnsignedTx(msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Failed to build tx: %v\n", err)
		os.Exit(1)
	}

	// Sign transaction
	if err := tx.Sign(txf, "provider", txBuilder, true); err != nil {
		fmt.Fprintf(os.Stderr, "✗ Failed to sign tx: %v\n", err)
		os.Exit(1)
	}

	// Encode transaction
	txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Failed to encode tx: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Broadcasting registration transaction...")

	// Broadcast transaction
	res, err := clientCtx.BroadcastTx(txBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Failed to broadcast: %v\n", err)
		os.Exit(1)
	}

	if res.Code != 0 {
		fmt.Fprintf(os.Stderr, "✗ Transaction failed (code %d): %s\n", res.Code, res.RawLog)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Provider registered successfully!\n")
	fmt.Printf("  TX Hash: %s\n", res.TxHash)
	fmt.Printf("  Height: %d\n", res.Height)
	fmt.Printf("\nVerify with:\n")
	fmt.Printf("  pawd query compute provider %s\n", addr.String())
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
