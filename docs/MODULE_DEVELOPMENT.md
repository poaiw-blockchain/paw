# PAW Module Development Guide

Guide for third-party developers building custom modules for the PAW blockchain.

## Quick Start

```bash
# 1. Clone and setup
git clone https://github.com/paw-chain/paw
cd paw && go mod download

# 2. Generate module scaffold
ignite scaffold module mymodule --dep bank,staking

# 3. Add message types
ignite scaffold message CreateWidget name:string value:uint

# 4. Generate proto types
make proto-gen
```

## Module Structure

```
x/mymodule/
├── keeper/           # State management
│   ├── keeper.go     # Core keeper
│   ├── msg_server.go # Message handlers
│   ├── query.go      # Query handlers
│   └── genesis.go    # Genesis import/export
├── types/
│   ├── keys.go       # Store key prefixes
│   ├── msgs.go       # Message validation
│   ├── params.go     # Module parameters
│   └── errors.go     # Custom errors
├── module.go         # AppModule interface
└── README.md
```

## Key Patterns

### 1. Store Key Prefixes

Use namespaced prefixes to avoid collisions:

```go
// x/mymodule/types/keys.go
const (
    ModuleName = "mymodule"
    StoreKey   = ModuleName
)

var (
    ParamsKey     = []byte{0x01}
    WidgetKey     = []byte{0x02}
    WidgetByOwner = []byte{0x03}
)

func GetWidgetKey(id uint64) []byte {
    return append(WidgetKey, sdk.Uint64ToBigEndian(id)...)
}
```

### 2. Keeper Methods

```go
// x/mymodule/keeper/keeper.go
type Keeper struct {
    cdc        codec.BinaryCodec
    storeKey   storetypes.StoreKey
    bankKeeper types.BankKeeper
}

func (k Keeper) SetWidget(ctx context.Context, widget types.Widget) error {
    store := k.getStore(ctx)
    bz, err := k.cdc.Marshal(&widget)
    if err != nil {
        return err
    }
    store.Set(types.GetWidgetKey(widget.Id), bz)
    return nil
}
```

### 3. Message Handlers

```go
// x/mymodule/keeper/msg_server.go
func (k msgServer) CreateWidget(ctx context.Context, msg *types.MsgCreateWidget) (*types.MsgCreateWidgetResponse, error) {
    // 1. Validate
    if err := msg.ValidateBasic(); err != nil {
        return nil, err
    }

    // 2. Business logic
    widget := types.Widget{
        Id:    k.GetNextWidgetId(ctx),
        Name:  msg.Name,
        Owner: msg.Creator,
    }

    // 3. Persist
    if err := k.SetWidget(ctx, widget); err != nil {
        return nil, err
    }

    // 4. Emit events
    sdkCtx := sdk.UnwrapSDKContext(ctx)
    sdkCtx.EventManager().EmitEvent(
        sdk.NewEvent("widget_created",
            sdk.NewAttribute("id", fmt.Sprintf("%d", widget.Id)),
        ),
    )

    return &types.MsgCreateWidgetResponse{Id: widget.Id}, nil
}
```

### 4. ABCI Hooks

```go
// x/mymodule/keeper/abci.go
func (k Keeper) BeginBlocker(ctx context.Context) error {
    // Called at start of each block
    return k.ProcessPendingOperations(ctx)
}

func (k Keeper) EndBlocker(ctx context.Context) error {
    // Called at end of each block
    return k.CleanupExpiredItems(ctx)
}
```

## Integration with PAW Modules

### Using DEX Module

```go
// Query pool price
pool, err := k.dexKeeper.GetPool(ctx, poolId)
price := pool.ReserveA.ToLegacyDec().Quo(pool.ReserveB.ToLegacyDec())

// Execute swap (via message)
msg := &dextypes.MsgSwap{
    Sender:       sender.String(),
    PoolId:       poolId,
    TokenIn:      sdk.NewCoin("upaw", amount),
    MinAmountOut: minOut,
}
```

### Using Oracle Module

```go
// Get latest price
price, err := k.oracleKeeper.GetPrice(ctx, "BTC/USD")
if err != nil {
    return err
}

// Get TWAP
twap, err := k.oracleKeeper.CalculateTWAP(ctx, "ETH/USD")
```

### Using Compute Module

```go
// Submit compute job
request := &computetypes.MsgSubmitRequest{
    Requester: sender.String(),
    Specs:     computetypes.ComputeSpec{CpuCores: 4, MemoryMb: 8192},
    Payload:   payload,
}
```

## Security Checklist

- [ ] Input validation on all messages
- [ ] Check sender authorization
- [ ] Use SafeMath for arithmetic
- [ ] Bound iterations (no unbounded loops)
- [ ] Proper iterator cleanup (`defer iter.Close()`)
- [ ] Gas metering for expensive operations
- [ ] No panics in keeper methods
- [ ] Deterministic state transitions

## Testing

```go
func TestCreateWidget(t *testing.T) {
    k, ctx := setupKeeper(t)

    msg := &types.MsgCreateWidget{
        Creator: "paw1abc...",
        Name:    "test",
    }

    resp, err := k.CreateWidget(ctx, msg)
    require.NoError(t, err)
    require.NotZero(t, resp.Id)

    // Verify state
    widget, err := k.GetWidget(ctx, resp.Id)
    require.NoError(t, err)
    require.Equal(t, "test", widget.Name)
}
```

## Proto Best Practices

```protobuf
// proto/mymodule/v1/types.proto
message Widget {
  uint64 id = 1;
  string name = 2;
  string owner = 3 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  string value = 4 [
    (cosmos_proto.scalar) = "cosmos.Int",
    (gogoproto.customtype) = "cosmossdk.io/math.Int",
    (gogoproto.nullable) = false
  ];
}
```

## Resources

- [Cosmos SDK Docs](https://docs.cosmos.network/)
- [PAW API Reference](./api/API_REFERENCE.md)
- [PAW Architecture](./architecture/README.md)
