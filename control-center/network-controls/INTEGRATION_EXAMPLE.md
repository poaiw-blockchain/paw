# Network Controls Integration Example

## App Integration (app/app.go)

```go
package app

import (
    // ... other imports
    networkcontrols "github.com/paw-chain/paw/control-center/network-controls"
)

// Add to App struct
type App struct {
    // ... existing fields

    // Network Controls
    NetworkControlsServer *networkcontrols.Server
}

// In NewPAWApp function
func NewPAWApp(...) *App {
    // ... existing initialization

    // Initialize Network Controls Server
    networkControlsCfg := networkcontrols.Config{
        ListenAddr: ":11050", // PAW control center port
        EnableCORS: true,
    }

    app.NetworkControlsServer = networkcontrols.NewServer(
        networkControlsCfg,
        app.DEXKeeper,
        app.OracleKeeper,
        app.ComputeKeeper,
        func() sdk.Context {
            // Provide current context
            return app.BaseApp.NewContext(false, tmproto.Header{})
        },
    )

    // Register circuit breakers
    app.NetworkControlsServer.GetManager().RegisterCircuitBreaker("dex", "")
    app.NetworkControlsServer.GetManager().RegisterCircuitBreaker("oracle", "")
    app.NetworkControlsServer.GetManager().RegisterCircuitBreaker("compute", "")

    // Set up auto-resume callback
    app.NetworkControlsServer.GetManager().SetAutoResumeCallback(
        func(module, subModule string) error {
            app.Logger().Info("Auto-resuming circuit breaker",
                "module", module,
                "submodule", subModule)
            return nil
        },
    )

    return app
}

// Add to StartNetworkControlsServer method
func (app *App) StartNetworkControlsServer() error {
    app.Logger().Info("Starting network controls server", "port", 11050)
    return app.NetworkControlsServer.Start()
}

// Add to StopNetworkControlsServer method
func (app *App) StopNetworkControlsServer() error {
    app.Logger().Info("Stopping network controls server")
    return app.NetworkControlsServer.Stop()
}
```

## CMD Integration (cmd/pawd/main.go)

```go
package main

import (
    // ... other imports
    "os"
    "os/signal"
    "syscall"
)

func main() {
    // ... existing code

    // Start network controls server
    if err := app.StartNetworkControlsServer(); err != nil {
        panic(fmt.Errorf("failed to start network controls server: %w", err))
    }

    // Set up signal handling for graceful shutdown
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigCh
        app.Logger().Info("Shutting down network controls server")
        app.StopNetworkControlsServer()
    }()

    // ... rest of main
}
```

## Message Handler Integration

### DEX Swap Handler (x/dex/keeper/msg_server.go)

```go
func (k msgServer) Swap(ctx context.Context, msg *types.MsgSwap) (*types.MsgSwapResponse, error) {
    // CRITICAL: Check circuit breaker FIRST
    if err := k.CheckCircuitBreaker(ctx); err != nil {
        return nil, sdkerrors.Wrap(types.ErrCircuitBreakerTriggered, err.Error())
    }

    // Check pool-specific circuit breaker
    if err := k.CheckPoolCircuitBreaker(ctx, msg.PoolId); err != nil {
        return nil, sdkerrors.Wrapf(types.ErrCircuitBreakerTriggered,
            "pool %d: %s", msg.PoolId, err.Error())
    }

    // Proceed with swap
    result, err := k.executeSwap(ctx, msg)
    if err != nil {
        return nil, err
    }

    return &types.MsgSwapResponse{
        AmountOut: result.AmountOut,
    }, nil
}
```

### Oracle Price Submission (x/oracle/keeper/msg_server.go)

```go
func (k msgServer) SubmitPrice(ctx context.Context, msg *types.MsgSubmitPrice) (*types.MsgSubmitPriceResponse, error) {
    // Check circuit breaker
    if err := k.CheckCircuitBreaker(ctx); err != nil {
        return nil, sdkerrors.Wrap(types.ErrCircuitBreakerTriggered, err.Error())
    }

    // Check feed-specific circuit breaker
    if err := k.CheckFeedCircuitBreaker(ctx, msg.Pair); err != nil {
        return nil, sdkerrors.Wrapf(types.ErrCircuitBreakerTriggered,
            "feed %s: %s", msg.Pair, err.Error())
    }

    // Check if slashing is disabled
    if k.IsSlashingDisabled(ctx) {
        // Log but don't slash during maintenance
        k.Logger(ctx).Warn("Slashing disabled - skipping validation",
            "validator", msg.Validator,
            "pair", msg.Pair)
    }

    // Proceed with price submission
    err := k.submitPrice(ctx, msg)
    if err != nil {
        return nil, err
    }

    return &types.MsgSubmitPriceResponse{}, nil
}
```

### Compute Request Handler (x/compute/keeper/msg_server.go)

```go
func (k msgServer) SubmitRequest(ctx context.Context, msg *types.MsgSubmitRequest) (*types.MsgSubmitRequestResponse, error) {
    // Check global circuit breaker
    if err := k.CheckCircuitBreaker(ctx); err != nil {
        return nil, sdkerrors.Wrap(types.ErrCircuitBreakerTriggered, err.Error())
    }

    // Proceed with request
    requestID, err := k.createRequest(ctx, msg)
    if err != nil {
        return nil, err
    }

    return &types.MsgSubmitRequestResponse{
        RequestId: requestID,
    }, nil
}

func (k msgServer) SubmitResult(ctx context.Context, msg *types.MsgSubmitResult) (*types.MsgSubmitResultResponse, error) {
    // Check provider circuit breaker
    provider := msg.Provider
    if err := k.CheckProviderCircuitBreaker(ctx, provider); err != nil {
        return nil, sdkerrors.Wrapf(types.ErrCircuitBreakerTriggered,
            "provider %s: %s", provider, err.Error())
    }

    // Check if job was cancelled
    if k.IsJobCancelled(ctx, msg.RequestId) {
        return nil, sdkerrors.Wrapf(types.ErrJobCancelled,
            "job %s was cancelled", msg.RequestId)
    }

    // Proceed with result submission
    err := k.submitResult(ctx, msg)
    if err != nil {
        return nil, err
    }

    return &types.MsgSubmitResultResponse{}, nil
}
```

## ABCI Integration

### Oracle Price Aggregation (x/oracle/keeper/abci.go)

```go
func (k Keeper) EndBlocker(ctx sdk.Context) {
    // Check if Oracle is paused
    if k.IsCircuitBreakerOpen(ctx) {
        k.Logger(ctx).Info("Oracle circuit breaker open - skipping price aggregation")
        return
    }

    // Get all price pairs
    pairs := k.GetAllPairs(ctx)

    for _, pair := range pairs {
        // Check for price override
        if overridePrice, hasOverride := k.GetPriceOverride(ctx, pair); hasOverride {
            k.Logger(ctx).Info("Using price override",
                "pair", pair,
                "price", overridePrice)
            continue
        }

        // Check feed-specific circuit breaker
        if k.IsFeedCircuitBreakerOpen(ctx, pair) {
            k.Logger(ctx).Info("Feed circuit breaker open - skipping",
                "pair", pair)
            continue
        }

        // Normal price aggregation
        k.AggregatePrices(ctx, pair)
    }
}
```

### Compute Job Processing (x/compute/keeper/abci.go)

```go
func (k Keeper) EndBlocker(ctx sdk.Context) {
    // Check if Compute is paused
    if k.IsCircuitBreakerOpen(ctx) {
        k.Logger(ctx).Info("Compute circuit breaker open - skipping job processing")
        return
    }

    // Get all active jobs
    jobs := k.GetActiveJobs(ctx)

    for _, job := range jobs {
        // Check if job was cancelled
        if k.IsJobCancelled(ctx, job.RequestId) {
            k.Logger(ctx).Info("Job cancelled - refunding escrow",
                "job_id", job.RequestId)
            k.RefundEscrow(ctx, job)
            continue
        }

        // Check provider circuit breaker
        if k.IsProviderCircuitBreakerOpen(ctx, job.Provider) {
            k.Logger(ctx).Info("Provider circuit breaker open - skipping",
                "provider", job.Provider,
                "job_id", job.RequestId)
            continue
        }

        // Check provider reputation (with override support)
        if score, hasOverride := k.GetReputationOverride(ctx, job.Provider); hasOverride {
            k.Logger(ctx).Info("Using reputation override",
                "provider", job.Provider,
                "score", score)
        }

        // Process job normally
        k.ProcessJob(ctx, job)
    }
}
```

## Testing Integration

```go
package app_test

import (
    "testing"
    "time"

    "github.com/stretchr/testify/require"
)

func TestCircuitBreakerIntegration(t *testing.T) {
    app := SetupTestApp(t)
    ctx := app.BaseApp.NewContext(false, tmproto.Header{})

    // Start network controls server
    require.NoError(t, app.StartNetworkControlsServer())
    defer app.StopNetworkControlsServer()

    // Pause DEX
    manager := app.NetworkControlsServer.GetManager()
    err := manager.PauseModule("dex", "", "test", "testing", nil)
    require.NoError(t, err)

    // Attempt swap (should fail)
    msg := &dextypes.MsgSwap{
        Sender:   testAddr,
        PoolId:   1,
        TokenIn:  sdk.NewCoin("token1", sdk.NewInt(100)),
        MinAmountOut: sdk.NewInt(90),
    }

    _, err = app.DEXKeeper.Swap(ctx, msg)
    require.Error(t, err)
    require.Contains(t, err.Error(), "circuit breaker")

    // Resume DEX
    err = manager.ResumeModule("dex", "", "test", "done")
    require.NoError(t, err)

    // Swap should now work
    _, err = app.DEXKeeper.Swap(ctx, msg)
    require.NoError(t, err)
}
```

## Monitoring Integration

### Prometheus Metrics

```go
// In your Prometheus scrape config
scrape_configs:
  - job_name: 'paw-network-controls'
    static_configs:
      - targets: ['localhost:11050']
    metrics_path: '/metrics'
```

### Grafana Dashboard

```json
{
  "panels": [
    {
      "title": "Circuit Breaker Status",
      "targets": [
        {
          "expr": "circuit_breaker_status{module=\"dex\"}"
        }
      ]
    },
    {
      "title": "Circuit Breaker Transitions",
      "targets": [
        {
          "expr": "rate(circuit_breaker_transitions_total[5m])"
        }
      ]
    }
  ]
}
```

## CLI Commands

Add custom CLI commands for circuit breaker operations:

```go
// cmd/pawd/cmd/circuit.go
package cmd

import (
    "github.com/spf13/cobra"
)

func CircuitBreakerCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "circuit",
        Short: "Circuit breaker operations",
    }

    cmd.AddCommand(
        CircuitBreakerPauseCmd(),
        CircuitBreakerResumeCmd(),
        CircuitBreakerStatusCmd(),
    )

    return cmd
}

func CircuitBreakerPauseCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "pause [module]",
        Short: "Pause a module",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            // Implementation
            return nil
        },
    }
}
```
