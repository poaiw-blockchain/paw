# Go Development

Build applications with Go on PAW Blockchain.

## Installation

```bash
go get <MODULE_PATH>
go get github.com/cosmos/cosmos-sdk
```

## Quick Example

```go
package main

import (
    "<MODULE_PATH>/app"
    sdk "github.com/cosmos/cosmos-sdk/types"
)

func main() {
    // Initialize app
    app := app.NewPAWApp()

    // Create transaction
    msg := banktypes.NewMsgSend(
        fromAddr,
        toAddr,
        sdk.NewCoins(sdk.NewCoin("upaw", sdk.NewInt(1000000))),
    )

    // Sign and broadcast
    tx := builder.Sign(msg)
    result := app.BroadcastTx(tx)
}
```

## Resources

- [Go Module Docs](https://pkg.go.dev/<MODULE_PATH>)
- [Cosmos SDK Docs](https://docs.cosmos.network)

---

**Previous:** [Python SDK](/developer/python-sdk) | **Next:** [Smart Contracts](/developer/smart-contracts) →
