package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/api"
)

func main() {
	// Configure SDK
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("paw", "pawpub")
	config.SetBech32PrefixForValidator("pawvaloper", "pawvaloperpub")
	config.SetBech32PrefixForConsensusNode("pawvalcons", "pawvalconspub")
	config.Seal()

	// Create codec
	interfaceRegistry := types.NewInterfaceRegistry()
	marshaler := codec.NewProtoCodec(interfaceRegistry)

	// Create client context
	clientCtx := client.Context{}.
		WithCodec(marshaler).
		WithInterfaceRegistry(interfaceRegistry).
		WithTxConfig(nil). // Will be set from app
		WithLegacyAmino(codec.NewLegacyAmino()).
		WithInput(os.Stdin).
		WithOutput(os.Stdout).
		WithAccountRetriever(nil). // Will be set from app
		WithBroadcastMode(flags.BroadcastSync).
		WithHomeDir(os.ExpandEnv("$HOME/.paw")).
		WithChainID("paw-1").
		WithNodeURI("tcp://localhost:26657")

	// Create server config
	serverConfig := &api.Config{
		Host:         getEnv("API_HOST", "0.0.0.0"),
		Port:         getEnv("API_PORT", "5000"),
		ChainID:      getEnv("CHAIN_ID", "paw-1"),
		NodeURI:      getEnv("NODE_URI", "tcp://localhost:26657"),
		JWTSecret:    []byte(getEnv("JWT_SECRET", "change-me-in-production")),
		CORSOrigins:  []string{"http://localhost:3000", "http://localhost:8080"},
		RateLimitRPS: 100,
	}

	// Create and start server
	server, err := api.NewServer(clientCtx, serverConfig)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	fmt.Println("╔═══════════════════════════════════════════════════╗")
	fmt.Println("║         PAW Blockchain API Server                ║")
	fmt.Println("╚═══════════════════════════════════════════════════╝")
	fmt.Printf("\nServer Configuration:\n")
	fmt.Printf("  - Host: %s\n", serverConfig.Host)
	fmt.Printf("  - Port: %s\n", serverConfig.Port)
	fmt.Printf("  - Chain ID: %s\n", serverConfig.ChainID)
	fmt.Printf("  - Node URI: %s\n", serverConfig.NodeURI)
	fmt.Printf("\nAPI Endpoints:\n")
	fmt.Printf("  - Health: http://%s:%s/health\n", serverConfig.Host, serverConfig.Port)
	fmt.Printf("  - Auth: http://%s:%s/api/auth/*\n", serverConfig.Host, serverConfig.Port)
	fmt.Printf("  - Trading: http://%s:%s/api/orders/*\n", serverConfig.Host, serverConfig.Port)
	fmt.Printf("  - Wallet: http://%s:%s/api/wallet/*\n", serverConfig.Host, serverConfig.Port)
	fmt.Printf("  - WebSocket: ws://%s:%s/ws\n", serverConfig.Host, serverConfig.Port)
	fmt.Printf("\nPress Ctrl+C to stop the server\n\n")

	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
