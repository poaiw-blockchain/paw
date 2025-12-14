## Network Controls - Circuit Breakers & Emergency Pause

Real-time network management controls with circuit breakers and emergency pause capabilities for PAW blockchain modules.

## Overview

The Network Controls system provides a comprehensive solution for managing blockchain module operations in real-time, with the ability to pause/resume operations, override critical parameters, and respond to emergencies.

### Key Features

- **Circuit Breakers**: Pause and resume module operations (DEX, Oracle, Compute)
- **Emergency Controls**: Rapid response to security incidents or anomalies
- **Auto-Resume**: Automatic recovery with configurable time limits
- **State Management**: Complete audit trail of all circuit breaker state changes
- **Prometheus Metrics**: Real-time monitoring of circuit breaker states
- **REST API**: Full HTTP API for control operations
- **SDK Integration**: Deep integration with Cosmos SDK modules

## Architecture

```
control-center/network-controls/
├── circuit/           # Circuit breaker state management
│   ├── state.go      # State definitions and registry
│   └── manager.go    # Manager with auto-resume logic
├── api/              # HTTP API handlers
│   └── handlers.go   # REST endpoints
├── integration/      # SDK module integration
│   ├── dex.go       # DEX module integration
│   ├── oracle.go    # Oracle module integration
│   └── compute.go   # Compute module integration
├── tests/           # Comprehensive test suite
└── server.go        # Main server component
```

## Circuit Breaker States

### Status Values

- **Closed**: Normal operation (default)
- **Open**: Circuit broken, operations blocked
- **Half-Open**: Testing if system recovered

### State Transitions

```
Closed --[pause]--> Open --[resume]--> Closed
         ^                    |
         |                    v
         +-- Half-Open <------+
```

## API Reference

### DEX Module Controls

#### Pause All DEX Operations
```bash
POST /api/v1/controls/dex/pause
{
  "actor": "admin",
  "reason": "Emergency maintenance",
  "auto_resume_mins": 60
}
```

#### Resume DEX Operations
```bash
POST /api/v1/controls/dex/resume
{
  "actor": "admin",
  "reason": "Maintenance complete"
}
```

#### Pause Specific Pool
```bash
POST /api/v1/controls/dex/pool/{poolID}/pause
{
  "actor": "admin",
  "reason": "Liquidity anomaly detected"
}
```

### Oracle Module Controls

#### Pause Oracle
```bash
POST /api/v1/controls/oracle/pause
{
  "actor": "admin",
  "reason": "Price feed validation issues"
}
```

#### Override Price
```bash
POST /api/v1/controls/oracle/override-price
{
  "actor": "admin",
  "pair": "BTC/USD",
  "price": "50000000000",
  "duration": 3600,
  "reason": "Emergency price freeze"
}
```

### Compute Module Controls

#### Pause Compute
```bash
POST /api/v1/controls/compute/pause
{
  "actor": "admin",
  "reason": "Provider misbehavior detected"
}
```

#### Cancel Job
```bash
POST /api/v1/controls/compute/job/{jobID}/cancel
{
  "actor": "admin",
  "job_id": "req-12345",
  "reason": "Invalid computation detected"
}
```

#### Override Provider Reputation
```bash
POST /api/v1/controls/compute/provider/{providerID}/pause
{
  "actor": "admin",
  "reason": "Provider under investigation"
}
```

### Status & History

#### Get All Circuit Breaker States
```bash
GET /api/v1/controls/status
```

Response:
```json
{
  "circuit_breakers": {
    "dex": {
      "module": "dex",
      "status": "closed",
      "transition_history": [...]
    },
    "oracle": {
      "module": "oracle",
      "status": "open",
      "paused_at": "2025-12-14T12:00:00Z",
      "paused_by": "admin",
      "reason": "Price feed issues",
      "auto_resume_at": "2025-12-14T13:00:00Z"
    }
  },
  "timestamp": "2025-12-14T12:30:00Z"
}
```

#### Get Module History
```bash
GET /api/v1/controls/history?module=dex
```

### Emergency Controls

#### Emergency Halt (All Modules)
```bash
POST /api/v1/controls/emergency/halt
{
  "actor": "admin",
  "reason": "Critical security incident",
  "modules": ["dex", "oracle", "compute"],
  "signature": "0x..."
}
```

#### Resume All Modules
```bash
POST /api/v1/controls/emergency/resume-all
{
  "actor": "admin",
  "reason": "Incident resolved"
}
```

## SDK Module Integration

### DEX Module

Circuit breaker checks are automatically performed in all message handlers:

```go
// In x/dex/keeper/msg_server.go
func (k msgServer) Swap(ctx context.Context, msg *types.MsgSwap) (*types.MsgSwapResponse, error) {
    // Check circuit breaker
    if err := k.CheckCircuitBreaker(ctx); err != nil {
        return nil, err
    }

    // Check pool-specific circuit breaker
    if err := k.CheckPoolCircuitBreaker(ctx, msg.PoolId); err != nil {
        return nil, err
    }

    // Proceed with swap
    // ...
}
```

### Oracle Module

Price override support with automatic expiration:

```go
// In x/oracle/keeper/abci.go
func (k Keeper) GetPrice(ctx context.Context, pair string) (*big.Int, bool) {
    // Check for active override
    if overridePrice, hasOverride := k.GetPriceOverride(ctx, pair); hasOverride {
        return overridePrice, true
    }

    // Normal price retrieval
    // ...
}
```

### Compute Module

Job cancellation and provider circuit breakers:

```go
// In x/compute/keeper/abci.go
func (k Keeper) ProcessJobs(ctx sdk.Context) {
    for _, job := range k.GetActiveJobs(ctx) {
        // Check if job was cancelled
        if k.IsJobCancelled(ctx, job.RequestId) {
            k.RefundEscrow(ctx, job)
            continue
        }

        // Check provider circuit breaker
        if err := k.CheckProviderCircuitBreaker(ctx, job.Provider); err != nil {
            continue
        }

        // Process job
        // ...
    }
}
```

## Prometheus Metrics

### Circuit Breaker Status
```
circuit_breaker_status{module="dex",submodule=""} 0  # 0=closed, 1=open, 2=half-open
circuit_breaker_status{module="oracle",submodule=""} 1
circuit_breaker_status{module="compute",submodule=""} 0
```

### State Transitions
```
circuit_breaker_transitions_total{module="dex",submodule="",from="closed",to="open"} 5
circuit_breaker_transitions_total{module="dex",submodule="",from="open",to="closed"} 4
```

### Auto-Resumes
```
circuit_breaker_auto_resumes_total 12
```

## Safety Features

### Multi-Signature Requirement

Critical operations (emergency halt) require multi-signature verification:

```go
func (h *Handler) handleEmergencyHalt(w http.ResponseWriter, r *http.Request) {
    var req EmergencyHaltRequest
    // ... parse request

    // Verify multi-signature
    if req.Signature == "" {
        h.writeError(w, http.StatusUnauthorized, "Emergency halt requires signature")
        return
    }

    // ... execute halt
}
```

### Auto-Resume Time Limits

All pauses can have automatic resume timers:

```go
autoResume := 60 * time.Minute
manager.PauseModule("dex", "", "admin", "maintenance", &autoResume)
```

### State Snapshots

Complete audit trail of all state changes:

```go
state, _ := manager.GetState("dex", "")
for _, transition := range state.TransitionHistory {
    log.Printf("Transition: %s -> %s by %s at %v",
        transition.From, transition.To, transition.Actor, transition.Timestamp)
}
```

## Usage Examples

### Scenario 1: Pause DEX for Maintenance

```bash
# Pause DEX with 1-hour auto-resume
curl -X POST http://localhost:11050/api/v1/controls/dex/pause \
  -H "Content-Type: application/json" \
  -d '{
    "actor": "admin",
    "reason": "Pool upgrade maintenance",
    "auto_resume_mins": 60
  }'

# Check status
curl http://localhost:11050/api/v1/controls/status

# Resume early if maintenance completes
curl -X POST http://localhost:11050/api/v1/controls/dex/resume \
  -H "Content-Type: application/json" \
  -d '{
    "actor": "admin",
    "reason": "Maintenance complete"
  }'
```

### Scenario 2: Emergency Price Override

```bash
# Oracle price feed is compromised, set emergency override
curl -X POST http://localhost:11050/api/v1/controls/oracle/override-price \
  -H "Content-Type: application/json" \
  -d '{
    "actor": "admin",
    "pair": "ETH/USD",
    "price": "3000000000",
    "duration": 7200,
    "reason": "Price feed compromise detected"
  }'

# Clear override when feeds are restored
curl -X POST http://localhost:11050/api/v1/controls/oracle/resume \
  -H "Content-Type: application/json" \
  -d '{
    "actor": "admin",
    "reason": "Price feeds restored and verified"
  }'
```

### Scenario 3: Cancel Malicious Compute Job

```bash
# Cancel specific job
curl -X POST http://localhost:11050/api/v1/controls/compute/job/req-12345/cancel \
  -H "Content-Type: application/json" \
  -d '{
    "actor": "admin",
    "job_id": "req-12345",
    "reason": "Malicious computation detected"
  }'

# Pause entire provider
curl -X POST http://localhost:11050/api/v1/controls/compute/provider/pawaddr123/pause \
  -H "Content-Type: application/json" \
  -d '{
    "actor": "admin",
    "reason": "Provider under investigation"
  }'
```

## Configuration

### Server Configuration

```go
cfg := networkcontrols.Config{
    ListenAddr: ":11050",  // Control center port for PAW
    EnableCORS: true,
}

server := networkcontrols.NewServer(
    cfg,
    app.DEXKeeper,
    app.OracleKeeper,
    app.ComputeKeeper,
    func() sdk.Context { return app.GetContext() },
)

if err := server.Start(); err != nil {
    panic(err)
}
```

### Module Integration

In your Cosmos SDK app initialization:

```go
// Register circuit breakers during app initialization
app.NetworkControlsServer.GetManager().RegisterCircuitBreaker("dex", "")
app.NetworkControlsServer.GetManager().RegisterCircuitBreaker("oracle", "")
app.NetworkControlsServer.GetManager().RegisterCircuitBreaker("compute", "")

// Set up auto-resume callback
app.NetworkControlsServer.GetManager().SetAutoResumeCallback(
    func(module, subModule string) error {
        log.Printf("Auto-resuming %s:%s", module, subModule)
        return nil
    },
)
```

## Testing

Run the comprehensive test suite:

```bash
cd control-center/network-controls/tests
go test -v -race ./...
```

## Security Considerations

1. **Access Control**: All control operations should require authentication and authorization
2. **Audit Logging**: All state changes are logged with actor, reason, and timestamp
3. **Rate Limiting**: API endpoints should be rate-limited to prevent abuse
4. **Multi-Signature**: Critical operations (emergency halt) require multi-sig
5. **Auto-Resume**: Always set auto-resume timers to prevent indefinite pauses
6. **Monitoring**: Set up alerts for circuit breaker state changes

## Troubleshooting

### Circuit Breaker Won't Close

Check the current state and transition history:

```bash
curl http://localhost:11050/api/v1/controls/status/dex
```

Verify auto-resume timer hasn't expired yet:

```json
{
  "auto_resume_at": "2025-12-14T13:00:00Z"
}
```

### Operations Still Blocked After Resume

Check if there's a submodule-specific circuit breaker:

```bash
curl http://localhost:11050/api/v1/controls/status
```

Verify SDK module state is synced (auto-sync runs every 5 seconds).

### Auto-Resume Not Working

Check manager health:

```bash
curl http://localhost:11050/api/v1/controls/health
```

Verify auto-resume callback is set and functioning.

## Future Enhancements

- [ ] Web UI for circuit breaker management
- [ ] Governance proposals for circuit breaker configuration
- [ ] Automated circuit breaker triggers based on metrics
- [ ] Integration with alerting systems (PagerDuty, etc.)
- [ ] Circuit breaker testing mode (dry-run)
- [ ] Historical analytics and reporting
- [ ] Role-based access control (RBAC)

## License

Copyright © 2025 PAW Chain. All rights reserved.
