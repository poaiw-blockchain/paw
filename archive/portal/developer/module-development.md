# Module Development

Extend PAW by developing custom Cosmos SDK modules.

## Module Structure

```
x/mymodule/
├── keeper/        # State management
├── types/         # Message & query types
├── client/        # CLI & REST
└── module.go      # Module interface
```

## Creating a Module

```go
package mymodule

import (
    sdk "github.com/cosmos/cosmos-sdk/types"
    "github.com/cosmos/cosmos-sdk/types/module"
)

type AppModule struct {
    keeper Keeper
}

func (am AppModule) RegisterServices(cfg module.Configurator) {
    types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
    types.RegisterQueryServer(cfg.QueryServer(), am.keeper)
}
```

## Integration

Add to `app/app.go`:

```go
app.MyModuleKeeper = mymodulekeeper.NewKeeper(
    appCodec,
    keys[mymoduletypes.StoreKey],
    app.GetSubspace(mymoduletypes.ModuleName),
)
```

## Resources

- [Cosmos SDK Docs](https://docs.cosmos.network/main/building-modules/intro)
- [Module Template](<REPO_URL>/tree/master/x/template)

---

**Previous:** [Smart Contracts](/developer/smart-contracts) | **Next:** [API Reference](/developer/api) →
