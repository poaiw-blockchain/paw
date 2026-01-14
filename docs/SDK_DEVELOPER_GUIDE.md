# PAW SDK Developer Guide

Build applications that interact with the PAW blockchain.

## Quick Start

### Prerequisites
- PAW node running (`pawd start`)
- gRPC: `localhost:9090`, REST: `localhost:1317`

---

## TypeScript / JavaScript

### Installation
```bash
npm install @cosmjs/stargate @cosmjs/proto-signing @cosmjs/tendermint-rpc
```

### Connect to PAW
```typescript
import { SigningStargateClient, StargateClient } from "@cosmjs/stargate";
import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";

// Query-only client
const client = await StargateClient.connect("http://localhost:26657");
const balance = await client.getBalance("paw1...", "upaw");

// Signing client (for transactions)
const wallet = await DirectSecp256k1HdWallet.fromMnemonic(
  "your mnemonic words here",
  { prefix: "paw" }
);
const [account] = await wallet.getAccounts();

const signingClient = await SigningStargateClient.connectWithSigner(
  "http://localhost:26657",
  wallet
);
```

### DEX Operations
```typescript
// Query pool
const response = await fetch("http://localhost:1317/paw/dex/v1/pools/1");
const { pool } = await response.json();

// Swap tokens
const msg = {
  typeUrl: "/paw.dex.v1.MsgSwap",
  value: {
    sender: account.address,
    poolId: "1",
    tokenIn: "upaw",
    tokenOut: "uatom",
    amountIn: "1000000",
    minAmountOut: "900000",
  },
};

const result = await signingClient.signAndBroadcast(
  account.address,
  [msg],
  "auto"
);
console.log("Tx hash:", result.transactionHash);
```

### Subscribe to Events
```typescript
import { Tendermint34Client } from "@cosmjs/tendermint-rpc";

const tm = await Tendermint34Client.connect("ws://localhost:26657/websocket");

// Subscribe to swap events
const subscription = tm.subscribeNewBlock();
subscription.addListener({
  next: (event) => {
    const swapEvents = event.events.filter(
      (e) => e.type === "dex_swap_executed"
    );
    swapEvents.forEach(console.log);
  },
});
```

---

## Python

### Installation
```bash
pip install cosmospy-protobuf httpx
```

### Connect and Query
```python
import httpx
from cosmospy import Transaction

# REST API queries
def get_pool(pool_id: int) -> dict:
    resp = httpx.get(f"http://localhost:1317/paw/dex/v1/pools/{pool_id}")
    return resp.json()["pool"]

def get_balance(address: str, denom: str) -> str:
    resp = httpx.get(
        f"http://localhost:1317/cosmos/bank/v1beta1/balances/{address}"
    )
    balances = resp.json()["balances"]
    for b in balances:
        if b["denom"] == denom:
            return b["amount"]
    return "0"

# Example
pool = get_pool(1)
print(f"Pool reserves: {pool['reserve_a']} / {pool['reserve_b']}")
```

### Sign and Broadcast
```python
from cosmospy import Transaction, generate_wallet

# Generate wallet (or use existing mnemonic)
wallet = generate_wallet(prefix="paw")
# Or: from_mnemonic("your words...", prefix="paw")

# Create swap transaction
tx = Transaction(
    privkey=bytes.fromhex(wallet["private_key"]),
    account_num=0,
    sequence=0,
    chain_id="paw-mvp-1",
    gas=200000,
)

tx.add_msg(
    msg_type="paw.dex.v1.MsgSwap",
    sender=wallet["address"],
    pool_id=1,
    token_in="upaw",
    token_out="uatom",
    amount_in="1000000",
    min_amount_out="900000",
)

# Broadcast
resp = httpx.post(
    "http://localhost:1317/cosmos/tx/v1beta1/txs",
    json={"tx_bytes": tx.get_tx_bytes(), "mode": "BROADCAST_MODE_SYNC"}
)
print(resp.json())
```

---

## Go

### Installation
```bash
go get github.com/paw-chain/paw
go get github.com/cosmos/cosmos-sdk
```

### Client Setup
```go
package main

import (
    "context"
    "fmt"

    "google.golang.org/grpc"
    dextypes "github.com/paw-chain/paw/x/dex/types"
)

func main() {
    // Connect to gRPC
    conn, err := grpc.Dial("localhost:9090", grpc.WithInsecure())
    if err != nil {
        panic(err)
    }
    defer conn.Close()

    // Query pool
    client := dextypes.NewQueryClient(conn)
    resp, err := client.Pool(context.Background(), &dextypes.QueryPoolRequest{
        PoolId: 1,
    })
    if err != nil {
        panic(err)
    }

    fmt.Printf("Pool: %+v\n", resp.Pool)
}
```

### Transaction Signing
```go
package main

import (
    "github.com/cosmos/cosmos-sdk/client"
    "github.com/cosmos/cosmos-sdk/crypto/keyring"
    dextypes "github.com/paw-chain/paw/x/dex/types"
    "cosmossdk.io/math"
)

func swap(ctx client.Context, poolID uint64, amountIn math.Int) error {
    msg := &dextypes.MsgSwap{
        Sender:       ctx.FromAddress.String(),
        PoolId:       poolID,
        TokenIn:      "upaw",
        TokenOut:     "uatom",
        AmountIn:     amountIn,
        MinAmountOut: amountIn.MulRaw(90).QuoRaw(100), // 10% slippage
    }

    return tx.GenerateOrBroadcastTxCLI(ctx, cmd.Flags(), msg)
}
```

---

## Common Patterns

### Error Handling
```typescript
try {
  const result = await signingClient.signAndBroadcast(...);
  if (result.code !== 0) {
    throw new Error(`TX failed: ${result.rawLog}`);
  }
} catch (e) {
  if (e.message.includes("insufficient funds")) {
    // Handle balance error
  } else if (e.message.includes("slippage")) {
    // Handle slippage error
  }
}
```

### Waiting for Finality
```typescript
// Wait for transaction inclusion
async function waitForTx(txHash: string, timeout = 30000) {
  const start = Date.now();
  while (Date.now() - start < timeout) {
    try {
      const tx = await client.getTx(txHash);
      if (tx) return tx;
    } catch {}
    await new Promise(r => setTimeout(r, 1000));
  }
  throw new Error("Transaction not found");
}
```

### Pagination
```typescript
// Fetch all pools with pagination
async function getAllPools() {
  const pools = [];
  let nextKey = null;

  do {
    const params = nextKey ? `?pagination.key=${nextKey}` : "";
    const resp = await fetch(`http://localhost:1317/paw/dex/v1/pools${params}`);
    const data = await resp.json();
    pools.push(...data.pools);
    nextKey = data.pagination?.next_key;
  } while (nextKey);

  return pools;
}
```

---

## Testing

### Local Testnet
```bash
# Start local testnet
pawd init testnode --chain-id paw-local-1
pawd keys add alice
pawd genesis add-genesis-account alice 1000000000upaw
pawd genesis gentx alice 1000000upaw --chain-id paw-local-1
pawd genesis collect-gentxs
pawd start
```

### Faucet
```bash
# Request testnet tokens (if faucet available)
curl -X POST http://faucet.paw.network/request \
  -d '{"address": "paw1..."}'
```

---

## Resources

- [API Reference](api/API_REFERENCE.md)
- [Architecture Decisions](architecture/README.md)
- [Proto Files](../proto/paw/)
- [Example Apps](../examples/) (coming soon)
