# PAW Go SDK Helpers

Go SDK helpers and utilities for PAW blockchain development.

## Overview

This package provides helper functions and utilities that complement the core PAW blockchain codebase. While the main blockchain code is in the root `github.com/paw-chain/paw` module, this SDK provides convenient wrappers and utilities for building applications.

## Features

- **Client Helpers**: Easy-to-use client wrappers for common operations
- **Wallet Management**: Mnemonic generation and wallet import
- **Transaction Helpers**: Simplified transaction building
- **DEX Calculations**: Swap calculations, price impact, LP shares
- **Testing Utilities**: Helper functions for writing tests
- **Address Utilities**: Address validation and conversion

## Installation

```bash
go get github.com/paw-chain/paw/sdk/go
```

## Quick Start

### Create Client

```go
import (
    pawclient "github.com/paw-chain/paw/sdk/go/client"
    "github.com/paw-chain/paw/sdk/go/helpers"
)

func main() {
    config := pawclient.Config{
        RPCEndpoint:  "http://localhost:26657",
        GRPCEndpoint: "localhost:9090",
        ChainID:      "paw-testnet-1",
    }

    client, err := pawclient.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
}
```

### Generate Wallet

```go
// Generate mnemonic
mnemonic, err := helpers.GenerateMnemonic()
if err != nil {
    log.Fatal(err)
}

// Import wallet
addr, err := client.ImportWalletFromMnemonic("my-wallet", mnemonic, "")
if err != nil {
    log.Fatal(err)
}

fmt.Println("Address:", addr.String())
```

### Query Balance

```go
ctx := context.Background()

balance, err := client.GetBalance(ctx, addr.String(), "upaw")
if err != nil {
    log.Fatal(err)
}

fmt.Println("Balance:", helpers.FormatCoin(*balance, 6))
```

### Send Transaction

```go
import (
    sdk "github.com/cosmos/cosmos-sdk/types"
    banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// Create message
msg := banktypes.NewMsgSend(
    fromAddr,
    toAddr,
    sdk.NewCoins(sdk.NewCoin("upaw", sdk.NewInt(1000000))),
)

// Sign and broadcast
resp, err := client.SignAndBroadcast(ctx, "my-wallet", msg)
if err != nil {
    log.Fatal(err)
}

fmt.Println("TX Hash:", resp.TxHash)
```

## Helper Functions

### Mnemonic Management

```go
// Generate 24-word mnemonic
mnemonic, err := helpers.GenerateMnemonic()

// Validate mnemonic
valid := helpers.ValidateMnemonic(mnemonic)
```

### Coin Formatting

```go
coin := sdk.NewCoin("upaw", sdk.NewInt(1500000))

// Format with 6 decimals
formatted := helpers.FormatCoin(coin, 6)
// Output: "1.500000 upaw"
```

### DEX Calculations

```go
// Calculate swap output
amountIn := sdk.NewInt(1000000)
reserveIn := sdk.NewInt(10000000)
reserveOut := sdk.NewInt(20000000)
swapFee := sdk.NewDecWithPrec(3, 3) // 0.3%

amountOut := helpers.CalculateSwapOutput(amountIn, reserveIn, reserveOut, swapFee)

// Calculate price impact
impact := helpers.CalculatePriceImpact(amountIn, reserveIn, reserveOut)
fmt.Printf("Price impact: %.2f%%\n", impact.MustFloat64())

// Calculate LP shares
shares := helpers.CalculateShares(amountA, amountB, reserveA, reserveB, totalShares)
```

### Address Utilities

```go
// Validate address
err := helpers.ValidateAddress("paw1...", "paw")

// Convert address between prefixes
converted, err := helpers.ConvertAddress(
    "paw1...",
    "paw",
    "cosmos",
)
```

## Testing Utilities

### Setup Test Client

```go
import (
    "testing"
    sdktesting "github.com/paw-chain/paw/sdk/go/testing"
)

func TestMyFeature(t *testing.T) {
    config := sdktesting.DefaultTestConfig()
    client := sdktesting.SetupTestClient(t, config)
    defer client.Close()

    // Your test code
}
```

### Create Test Wallet

```go
func TestWallet(t *testing.T) {
    client := sdktesting.SetupTestClient(t, sdktesting.DefaultTestConfig())

    mnemonic, addr := sdktesting.CreateTestWallet(t, client, "test-wallet")

    // Use wallet in tests
}
```

### Assert Transaction Success

```go
resp, err := client.SignAndBroadcast(ctx, "wallet", msg)
require.NoError(t, err)

sdktesting.AssertTransactionSuccess(t, resp)
```

### Assert Balance

```go
minAmount := sdk.NewInt(1000000)
sdktesting.AssertBalanceGreaterThan(
    t,
    client,
    addr.String(),
    "upaw",
    minAmount,
)
```

## Examples

See the `examples/` directory for complete working examples:

- `basic_usage.go`: Wallet creation and balance queries
- More examples coming soon

Run examples:

```bash
cd examples
go run basic_usage.go
```

## Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...
```

## Project Structure

```
sdk/go/
├── client/          # Client wrappers
│   ├── client.go    # Main client implementation
│   └── encoding.go  # Encoding configuration
├── helpers/         # Helper functions
│   └── helpers.go   # Utility functions
├── testing/         # Testing utilities
│   └── testing.go   # Test helpers
├── examples/        # Example code
└── README.md
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](../../LICENSE) for details.

## Support

- Documentation: https://docs.paw.network
- : https://github.com/paw-chain/paw
- Discord: https://discord.gg/paw
