# Missing gRPC Gateway Registration

---
status: pending
priority: p2
issue_id: "023"
tags: [architecture, api, all-modules, high]
dependencies: []
---

## Problem Statement

All three custom modules have empty `RegisterGRPCGatewayRoutes` implementations, meaning REST API endpoints are not registered. Only gRPC access works.

**Why it matters:** Breaks compatibility with web frontends, explorers, and HTTP-based tooling.

## Findings

### Source: architecture-strategist agent

**Location:**
- `/home/decri/blockchain-projects/paw/x/dex/module.go:63`
- `/home/decri/blockchain-projects/paw/x/oracle/module.go:63`
- `/home/decri/blockchain-projects/paw/x/compute/module.go:63`

**Code:**
```go
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {}
```

**Impact:**
- REST API endpoints unavailable for custom modules
- Web frontends cannot query module state via HTTP
- Block explorers cannot access module data
- Forces all clients to use gRPC exclusively

## Proposed Solutions

### Option A: Implement Gateway Registration (Recommended)
**Pros:** Enables REST API, standard practice
**Cons:** None
**Effort:** Small
**Risk:** Low

```go
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
    types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx))
}
```

This should be auto-generated from protobuf files using `buf generate` with grpc-gateway plugin.

## Recommended Action

**Implement Option A** for all three modules.

## Technical Details

**Affected Files:**
- `x/dex/module.go`
- `x/oracle/module.go`
- `x/compute/module.go`

**Proto Generation:** Ensure `buf.gen.yaml` includes grpc-gateway plugin.

## Acceptance Criteria

- [ ] RegisterGRPCGatewayRoutes implemented for DEX module
- [ ] RegisterGRPCGatewayRoutes implemented for Oracle module
- [ ] RegisterGRPCGatewayRoutes implemented for Compute module
- [ ] Test: curl REST endpoints return correct data
- [ ] Document REST API endpoints

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-07 | Created | Identified by architecture-strategist agent |

## Resources

- Cosmos SDK gRPC gateway documentation
- buf.build grpc-gateway plugin
