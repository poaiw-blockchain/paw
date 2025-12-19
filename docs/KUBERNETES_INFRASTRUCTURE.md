# PAW Blockchain - Kubernetes Infrastructure

Production-grade Kubernetes infrastructure for the PAW blockchain with comprehensive local testing capabilities.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          PAW Kubernetes Cluster                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐              │
│  │   Validator-0   │  │   Validator-1   │  │   Validator-2   │              │
│  │   (StatefulSet) │  │   (StatefulSet) │  │   (StatefulSet) │              │
│  │   500Gi PVC     │  │   500Gi PVC     │  │   500Gi PVC     │              │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘              │
│           │                    │                    │                        │
│           └────────────────────┼────────────────────┘                        │
│                                │                                             │
│                    ┌───────────▼───────────┐                                │
│                    │   Headless Service    │                                │
│                    │   (Pod Discovery)     │                                │
│                    └───────────────────────┘                                │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │                        Services Layer                                    ││
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐        ││
│  │  │   P2P   │  │   RPC   │  │   API   │  │  gRPC   │  │ Metrics │        ││
│  │  │ :26656  │  │ :26657  │  │  :1317  │  │  :9090  │  │ :26660  │        ││
│  │  │NodePort │  │ClusterIP│  │ClusterIP│  │ClusterIP│  │ClusterIP│        ││
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘  └─────────┘        ││
│  └─────────────────────────────────────────────────────────────────────────┘│
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │                        Security Layer                                    ││
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐  ││
│  │  │Network Policy│  │    RBAC      │  │   Vault      │  │  Linkerd    │  ││
│  │  │ (Deny All +  │  │(Least Priv)  │  │ (Secrets)    │  │  (mTLS)     │  ││
│  │  │  Allow List) │  │              │  │              │  │             │  ││
│  │  └──────────────┘  └──────────────┘  └──────────────┘  └─────────────┘  ││
│  └─────────────────────────────────────────────────────────────────────────┘│
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │                      Observability Layer                                 ││
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐         ││
│  │  │ Prometheus │  │  Grafana   │  │   Loki     │  │AlertManager│         ││
│  │  │  :9090     │  │   :3000    │  │   :3100    │  │   :9093    │         ││
│  │  └────────────┘  └────────────┘  └────────────┘  └────────────┘         ││
│  └─────────────────────────────────────────────────────────────────────────┘│
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Port Allocation

PAW uses the 11000-11999 port range per `PORT_ALLOCATION.md`:

| Service | Internal Port | NodePort | Purpose |
|---------|--------------|----------|---------|
| P2P | 26656 | 31656 | CometBFT P2P |
| RPC | 26657 | 31657 | Tendermint RPC |
| REST API | 1317 | 31317 | Cosmos REST API |
| gRPC | 9090 | 31090 | gRPC API |
| gRPC-Web | 9091 | 31091 | gRPC-Web Gateway |
| Metrics | 26660 | - | Prometheus scrape |
| Health | 36661 | - | Health checks |
| Grafana | 3000 | 31030 | Monitoring UI |
| Prometheus | 9090 | 31009 | Metrics storage |
| AlertManager | 9093 | 31093 | Alert routing |
| Loki | 3100 | - | Log aggregation |

## Quick Start - Local Testing

### Prerequisites

```bash
# Required tools
docker --version        # >= 24.0
kubectl version         # >= 1.28
helm version           # >= 3.12
kind version           # >= 0.20

# Optional but recommended
k9s version            # Terminal UI
linkerd version        # Service mesh
vault version          # Secrets management
```

### Option 1: Kind Cluster (Recommended for Local Development)

```bash
# Create cluster with local registry
cd ~/blockchain-projects/paw
./scripts/k8s/setup-kind-cluster.sh

# Deploy PAW infrastructure
./scripts/k8s/deploy-local.sh

# Verify deployment
./scripts/k8s/verify-deployment.sh
```

### Option 2: k3s Cluster (Recommended for Multi-Node Testing)

```bash
# Install k3s on control plane (bcpc)
./scripts/k8s/setup-k3s-server.sh

# Join worker nodes (optional)
./scripts/k8s/setup-k3s-agent.sh <server-ip> <token>

# Deploy PAW infrastructure
./scripts/k8s/deploy-k3s.sh
```

## Directory Structure

```
paw/k8s/
├── base/                          # Base Kustomize manifests
│   ├── kustomization.yaml
│   ├── namespace.yaml
│   ├── rbac.yaml
│   ├── storage-class.yaml
│   ├── network-policies.yaml
│   ├── pod-security-standards.yaml
│   └── resource-quotas.yaml
│
├── components/                    # Reusable components
│   ├── vault/                     # Vault + External Secrets
│   │   ├── vault-dev.yaml
│   │   ├── vault-prod.yaml
│   │   ├── external-secrets.yaml
│   │   └── cluster-secret-store.yaml
│   ├── monitoring/                # Prometheus stack
│   │   ├── prometheus.yaml
│   │   ├── grafana.yaml
│   │   ├── alertmanager.yaml
│   │   ├── loki.yaml
│   │   └── promtail.yaml
│   ├── linkerd/                   # Service mesh
│   │   ├── linkerd-install.yaml
│   │   └── linkerd-viz.yaml
│   └── cert-manager/              # TLS automation
│       ├── cert-manager.yaml
│       └── cluster-issuer.yaml
│
├── validators/                    # Validator nodes
│   ├── statefulset.yaml
│   ├── headless-service.yaml
│   ├── pdb.yaml
│   └── servicemonitor.yaml
│
├── nodes/                         # Full nodes
│   ├── deployment.yaml
│   ├── services.yaml
│   └── hpa.yaml
│
├── api/                           # API layer
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── ingress.yaml
│   └── hpa.yaml
│
├── overlays/                      # Environment-specific
│   ├── local/                     # Kind/local development
│   │   ├── kustomization.yaml
│   │   ├── patches/
│   │   └── configmaps/
│   ├── staging/                   # Pre-production
│   │   ├── kustomization.yaml
│   │   └── patches/
│   └── production/                # Production
│       ├── kustomization.yaml
│       └── patches/
│
├── scripts/                       # Automation scripts
│   ├── setup-kind-cluster.sh
│   ├── setup-k3s-server.sh
│   ├── setup-k3s-agent.sh
│   ├── deploy-local.sh
│   ├── deploy-vault.sh
│   ├── deploy-monitoring.sh
│   ├── deploy-linkerd.sh
│   ├── verify-deployment.sh
│   ├── run-chaos-tests.sh
│   ├── backup-state.sh
│   └── restore-state.sh
│
└── tests/                         # Test suites
    ├── smoke-tests.sh
    ├── integration-tests.sh
    ├── chaos-tests.sh
    ├── security-tests.sh
    └── load-tests.sh
```

## Component Details

### 1. Vault Integration

PAW uses HashiCorp Vault for secrets management with External Secrets Operator for Kubernetes integration.

**Secrets Stored in Vault:**
- `secret/paw/validators/priv_validator_key` - Validator signing keys
- `secret/paw/validators/node_key` - Node identity keys
- `secret/paw/api/jwt_secret` - API JWT signing secret
- `secret/paw/genesis/genesis_json` - Genesis file (base64)
- `secret/paw/tls/certificates` - TLS certificates

**Local Vault Setup:**
```bash
# Deploy Vault in dev mode
./scripts/k8s/deploy-vault.sh --mode=dev

# Initialize secrets
./scripts/k8s/init-vault-secrets.sh

# Verify External Secrets sync
kubectl get externalsecrets -n paw
```

### 2. Validator StatefulSet

Validators use StatefulSet for:
- Stable network identities (paw-validator-0, paw-validator-1, etc.)
- Persistent storage (500Gi per validator)
- Ordered deployment and scaling
- Pod disruption budgets

**Key Features:**
- Init container for genesis verification
- Sidecar for metrics export
- Anti-affinity across nodes/zones
- Automatic backup via CronJob

### 3. Network Policies

Default-deny with explicit allowlists:

```yaml
# Allowed traffic patterns:
P2P (26656):     validator ↔ validator, validator ↔ sentry
RPC (26657):     ingress → nodes, nodes → validators
API (1317):      ingress → api-deployment
gRPC (9090):     ingress → nodes
Metrics (26660): prometheus → all pods
DNS (53):        all pods → kube-dns
```

### 4. Monitoring Stack

**Prometheus:**
- Scrapes all PAW components every 15s
- Alert rules for consensus, sync, resources
- 15-day retention (configurable)

**Grafana Dashboards:**
- PAW Overview (blocks, transactions, peers)
- Validator Health (signing, jailing, delegations)
- IBC Activity (channels, packets, relayers)
- DEX Metrics (pools, swaps, liquidity)
- Infrastructure (CPU, memory, disk, network)

**Loki:**
- Centralized log aggregation
- LogQL queries for debugging
- 7-day retention

**AlertManager:**
- Slack/PagerDuty/Email integrations
- Alert grouping and deduplication
- Silence management

### 5. Linkerd Service Mesh

Provides:
- Automatic mTLS between all pods
- Traffic metrics and observability
- Retries and timeouts
- Traffic splitting for canary deployments

```bash
# Install Linkerd
./scripts/k8s/deploy-linkerd.sh

# Mesh the namespace
kubectl annotate namespace paw linkerd.io/inject=enabled

# Verify mesh
linkerd check --proxy -n paw
```

## Testing Infrastructure

### Smoke Tests

Quick validation of basic functionality:

```bash
./scripts/k8s/tests/smoke-tests.sh

# Tests:
# - Pod readiness
# - Service endpoints
# - RPC connectivity
# - Block production
# - Health endpoints
```

### Integration Tests

End-to-end functionality:

```bash
./scripts/k8s/tests/integration-tests.sh

# Tests:
# - Transaction submission
# - Block finality
# - IBC channel creation
# - DEX operations
# - Governance proposals
```

### Chaos Tests

Resilience testing with chaos engineering:

```bash
./scripts/k8s/tests/chaos-tests.sh

# Scenarios:
# - Pod failures (random validator kill)
# - Network partition (split brain)
# - High latency (200ms+ delays)
# - Resource exhaustion (memory pressure)
# - Storage failures (I/O errors)
```

### Security Tests

Security validation:

```bash
./scripts/k8s/tests/security-tests.sh

# Tests:
# - Network policy enforcement
# - RBAC restrictions
# - Pod security standards
# - Secret encryption
# - mTLS verification
# - CVE scanning
```

### Load Tests

Performance benchmarking:

```bash
./scripts/k8s/tests/load-tests.sh --profile=standard

# Profiles:
# - minimal: 10 TPS, 5 minutes
# - standard: 100 TPS, 30 minutes
# - stress: 1000 TPS, 1 hour
# - endurance: 50 TPS, 24 hours
```

## Deployment Procedures

### Initial Deployment

```bash
# 1. Create cluster
./scripts/k8s/setup-kind-cluster.sh

# 2. Deploy infrastructure components
./scripts/k8s/deploy-local.sh --components=all

# 3. Initialize validators
./scripts/k8s/init-validators.sh --count=3

# 4. Verify deployment
./scripts/k8s/verify-deployment.sh --full

# 5. Run smoke tests
./scripts/k8s/tests/smoke-tests.sh
```

### Upgrade Procedures

```bash
# 1. Create backup
./scripts/k8s/backup-state.sh --namespace=paw

# 2. Apply upgrade
kubectl apply -k k8s/overlays/local

# 3. Monitor rollout
kubectl rollout status statefulset/paw-validator -n paw

# 4. Verify health
./scripts/k8s/verify-deployment.sh
```

### Rollback Procedures

```bash
# Automatic rollback on failure
kubectl rollout undo statefulset/paw-validator -n paw

# Restore from backup
./scripts/k8s/restore-state.sh --backup=<backup-dir>
```

## Operational Runbooks

### Validator Not Signing

```bash
# Check validator status
kubectl exec -n paw paw-validator-0 -- pawd status | jq '.SyncInfo'

# Check for jailing
kubectl exec -n paw paw-validator-0 -- pawd query staking validators

# Check logs
kubectl logs -n paw paw-validator-0 --tail=100

# Unjail if needed
kubectl exec -n paw paw-validator-0 -- pawd tx slashing unjail --from=validator
```

### Consensus Stalled

```bash
# Check peer count
kubectl exec -n paw paw-validator-0 -- pawd status | jq '.NodeInfo.n_peers'

# Check network connectivity
kubectl exec -n paw paw-validator-0 -- curl -s http://localhost:26657/net_info

# Restart validators (rolling)
kubectl rollout restart statefulset/paw-validator -n paw
```

### Storage Issues

```bash
# Check PVC status
kubectl get pvc -n paw

# Check disk usage
kubectl exec -n paw paw-validator-0 -- df -h /data

# Expand PVC (if supported)
kubectl patch pvc paw-data-paw-validator-0 -n paw -p '{"spec":{"resources":{"requests":{"storage":"1Ti"}}}}'
```

## Security Hardening Checklist

- [ ] Pod Security Standards enforced (restricted)
- [ ] Network policies applied (default deny)
- [ ] RBAC configured (least privilege)
- [ ] Secrets encrypted at rest (Vault)
- [ ] mTLS enabled (Linkerd)
- [ ] Resource limits set (all containers)
- [ ] Security contexts defined (non-root)
- [ ] Audit logging enabled
- [ ] CVE scanning automated
- [ ] Backup encryption enabled

## Resource Requirements

### Minimum (Local Development)

| Component | CPU | Memory | Storage |
|-----------|-----|--------|---------|
| Validator | 2 | 4Gi | 50Gi |
| Full Node | 1 | 2Gi | 50Gi |
| Prometheus | 0.5 | 1Gi | 10Gi |
| Grafana | 0.2 | 512Mi | 1Gi |
| Vault | 0.2 | 256Mi | 1Gi |

### Recommended (Production)

| Component | CPU | Memory | Storage |
|-----------|-----|--------|---------|
| Validator | 4 | 8Gi | 500Gi |
| Full Node | 2 | 4Gi | 500Gi |
| Prometheus | 2 | 4Gi | 100Gi |
| Grafana | 0.5 | 1Gi | 10Gi |
| Vault | 1 | 2Gi | 10Gi |

## Troubleshooting

### Common Issues

**Pods stuck in Pending:**
```bash
kubectl describe pod <pod-name> -n paw
# Check for: insufficient resources, PVC not bound, node selectors
```

**CrashLoopBackOff:**
```bash
kubectl logs <pod-name> -n paw --previous
# Check for: config errors, missing secrets, genesis issues
```

**Services not accessible:**
```bash
kubectl get endpoints -n paw
# Check for: selector mismatch, port mismatch, network policies
```

**Vault secrets not syncing:**
```bash
kubectl describe externalsecret -n paw
# Check for: Vault connectivity, policy permissions, secret paths
```

## References

- [Cosmos SDK Documentation](https://docs.cosmos.network/)
- [CometBFT Documentation](https://docs.cometbft.com/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Vault Documentation](https://developer.hashicorp.com/vault/docs)
- [Linkerd Documentation](https://linkerd.io/docs/)
- [Prometheus Documentation](https://prometheus.io/docs/)
