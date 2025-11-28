package main

import (
	"context"
	"fmt"
	"log"

	pawclient "github.com/paw-chain/paw/sdk/go/client"
	"github.com/paw-chain/paw/sdk/go/helpers"
)

func main() {
	// 1. Generate mnemonic
	mnemonic, err := helpers.GenerateMnemonic()
	if err != nil {
		log.Fatal("Failed to generate mnemonic:", err)
	}
	fmt.Println("Generated mnemonic:", mnemonic)

	// 2. Create client
	config := pawclient.Config{
		RPCEndpoint:  "http://localhost:26657",
		GRPCEndpoint: "localhost:9090",
		ChainID:      "paw-testnet-1",
	}

	client, err := pawclient.NewClient(config)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer client.Close()

	fmt.Println("Connected to chain:", client.GetChainID())

	// 3. Import wallet
	addr, err := client.ImportWalletFromMnemonic("my-wallet", mnemonic, "")
	if err != nil {
		log.Fatal("Failed to import wallet:", err)
	}

	fmt.Println("Wallet address:", addr.String())

	// 4. Check balance
	ctx := context.Background()
	balance, err := client.GetBalance(ctx, addr.String(), "upaw")
	if err != nil {
		log.Fatal("Failed to get balance:", err)
	}

	if balance != nil {
		fmt.Println("Balance:", helpers.FormatCoin(*balance, 6))
	} else {
		fmt.Println("No balance found")
	}

	// 5. Get all balances
	allBalances, err := client.GetAllBalances(ctx, addr.String())
	if err != nil {
		log.Fatal("Failed to get all balances:", err)
	}

	fmt.Printf("All balances (%d tokens):\n", len(allBalances))
	for _, coin := range allBalances {
		fmt.Printf("  - %s\n", helpers.FormatCoin(coin, 6))
	}
}
