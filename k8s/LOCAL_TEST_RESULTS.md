# PAW Kubernetes Local Test Results

**Date**: 2025-12-19
**Cluster**: Kind (paw-local) - 1 control-plane, 2 workers
**Namespace**: paw

## Test Summary

| Phase | Tests | Status |
|-------|-------|--------|
| Infrastructure (Phase 1) | 11 | PASS |
| Security (Phase 2) | 8 | PASS |
| Chaos/Resilience (Phase 3) | 2 | PASS |
| Monitoring (Phase 5) | 3 | PASS |

## Detailed Results

### Phase 1: Infrastructure
- Namespace exists
- Pod running (1/1)
- StatefulSet ready (1/1)
- PVCs bound
- Services have endpoints
- Network policies configured (7)
- RPC endpoint responsive
- Health endpoint responsive
- Block production active

### Phase 2: Security
- Pod Security Standards: `restricted` level enforced
- runAsNonRoot: true
- runAsUser: 1000
- allowPrivilegeEscalation: false
- capabilities.drop: ALL
- seccompProfile: RuntimeDefault
- 5 ServiceAccounts configured
- 3 RoleBindings configured
- ResourceQuota enforced
- LimitRange configured

### Phase 3: Chaos/Resilience
- Pod deletion recovery: PASS (67s recovery time)
- Consensus resumed after restart

### Phase 5: Monitoring
- Prometheus deployed
- Grafana deployed (NodePort 31030)
- AlertManager deployed
- ServiceMonitor for PAW validator created
- Metrics endpoint accessible (port 26660)

## Configuration Files Saved
- `k8s/validators/statefulset-local.yaml` - Working StatefulSet with telemetry

## Known Issues Fixed
1. Slashing module `signing_infos` empty - Added manual population in init container
2. RPC bound to localhost - Changed to 0.0.0.0
3. Telemetry disabled - Enabled in app.toml
4. Prometheus metrics disabled - Enabled in config.toml

## Access Points (via port-forward)
- RPC: localhost:26657
- API: localhost:1317
- gRPC: localhost:9090
- Metrics: localhost:26660
- Grafana: NodePort 31030
