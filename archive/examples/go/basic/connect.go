package main

/*
PAW Blockchain - Connect to Network Example

This example demonstrates how to connect to the PAW blockchain network
and retrieve basic network information.

Usage:
    go run connect.go

Environment Variables:
    PAW_RPC_ENDPOINT - RPC endpoint URL (default: http://localhost:26657)
*/

import (
	"context"
	"fmt"
	"os"

	rpcclient "github.com/cometbft/cometbft/rpc/client/http"
)

// Config holds network configuration
type Config struct {
	RPCEndpoint string
}

// LoadConfig loads configuration from environment
func LoadConfig() Config {
	rpcEndpoint := os.Getenv("PAW_RPC_ENDPOINT")
	if rpcEndpoint == "" {
		rpcEndpoint = "http://localhost:26657"
	}

	return Config{
		RPCEndpoint: rpcEndpoint,
	}
}

// ConnectToNetwork connects to PAW network and displays information
func ConnectToNetwork(cfg Config) error {
	fmt.Println("Connecting to PAW Network...")
	fmt.Printf("RPC Endpoint: %s\n\n", cfg.RPCEndpoint)

	// Create RPC client
	client, err := rpcclient.New(cfg.RPCEndpoint, "/websocket")
	if err != nil {
		return fmt.Errorf("failed to create RPC client: %w", err)
	}
	defer client.Stop()

	ctx := context.Background()

	// Get node status
	status, err := client.Status(ctx)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	fmt.Println("✓ Successfully connected to PAW network\n")

	// Display chain information
	fmt.Printf("Chain ID: %s\n", status.NodeInfo.Network)
	fmt.Printf("Current Block Height: %d\n", status.SyncInfo.LatestBlockHeight)
	fmt.Printf("Node Version: %s\n", status.NodeInfo.Version)
	fmt.Printf("Moniker: %s\n", status.NodeInfo.Moniker)

	// Get latest block
	block, err := client.Block(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get block: %w", err)
	}

	fmt.Println("\nLatest Block Info:")
	fmt.Printf("  Block Hash: %s\n", block.BlockID.Hash)
	fmt.Printf("  Time: %s\n", block.Block.Time)
	fmt.Printf("  Num Transactions: %d\n", len(block.Block.Txs))
	fmt.Printf("  Proposer: %X\n", block.Block.ProposerAddress)

	// Calculate average block time
	height := status.SyncInfo.LatestBlockHeight
	if height > 5 {
		prevHeight := height - 5
		prevBlock, err := client.Block(ctx, &prevHeight)
		if err == nil {
			timeDiff := block.Block.Time.Sub(prevBlock.Block.Time)
			avgBlockTime := timeDiff.Seconds() / 5
			fmt.Printf("  Average Block Time: %.2fs\n", avgBlockTime)
		}
	}

	fmt.Println("\n✓ Network information retrieved successfully")

	return nil
}

func main() {
	cfg := LoadConfig()

	if err := ConnectToNetwork(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "✗ Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "\nTroubleshooting:")
		fmt.Fprintln(os.Stderr, "  1. Check if the RPC endpoint is correct")
		fmt.Fprintln(os.Stderr, "  2. Ensure the node is running")
		fmt.Fprintln(os.Stderr, "  3. Check firewall settings")
		os.Exit(1)
	}
}
