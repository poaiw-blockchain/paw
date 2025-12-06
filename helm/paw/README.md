# PAW Blockchain Helm Chart

Production-ready Helm chart for deploying PAW blockchain validator nodes on Kubernetes.

## Overview

This Helm chart deploys a PAW blockchain node (based on Cosmos SDK) with:
- Persistent storage for blockchain data
- Configurable resource limits
- Health probes for reliability
- Prometheus metrics support
- Secure defaults and best practices

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- PersistentVolume provisioner support in the cluster (if persistence is enabled)
- (Optional) Prometheus Operator for ServiceMonitor

## Installation

### Quick Start

```bash
# Add the repository (if published)
helm repo add paw https://charts.paw.example.com
helm repo update

# Install with default values
helm install my-paw-node paw/paw

# Install with custom values
helm install my-paw-node paw/paw -f custom-values.yaml
```

### Local Installation

```bash
# From the chart directory
helm install my-paw-node ./helm/paw

# With custom values
helm install my-paw-node ./helm/paw \
  --set node.chainId=paw-mainnet-1 \
  --set node.moniker=my-validator \
  --set persistence.size=500Gi
```

### Installation in Specific Namespace

```bash
# Create namespace
kubectl create namespace paw

# Install in namespace
helm install my-paw-node ./helm/paw -n paw
```

## Configuration

### Key Configuration Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Container image repository | `ghcr.io/decristofaroj/paw` |
| `image.tag` | Container image tag | `""` (uses appVersion) |
| `node.chainId` | Blockchain chain ID | `paw-testnet-1` |
| `node.moniker` | Validator node moniker | `paw-validator` |
| `node.minGasPrices` | Minimum gas prices | `0.025upaw` |
| `node.logLevel` | Log level (debug/info/warn/error) | `info` |
| `persistence.enabled` | Enable persistent storage | `true` |
| `persistence.size` | Size of persistent volume | `100Gi` |
| `persistence.storageClassName` | Storage class name | `""` (default) |
| `resources.limits.cpu` | CPU limit | `2000m` |
| `resources.limits.memory` | Memory limit | `4Gi` |
| `resources.requests.cpu` | CPU request | `1000m` |
| `resources.requests.memory` | Memory request | `2Gi` |

### Service Ports

| Port | Description | Default |
|------|-------------|---------|
| `service.ports.p2p.port` | P2P networking port | `26656` |
| `service.ports.rpc.port` | CometBFT RPC port | `26657` |
| `service.ports.api.port` | REST API port | `1317` |
| `service.ports.grpc.port` | gRPC server port | `9090` |
| `service.ports.metrics.port` | Prometheus metrics port | `26660` |

### Example Configurations

#### Production Validator

```yaml
# production-values.yaml
replicaCount: 1

image:
  repository: ghcr.io/decristofaroj/paw
  tag: "v1.0.0"
  pullPolicy: IfNotPresent

node:
  chainId: "paw-mainnet-1"
  moniker: "production-validator-01"
  minGasPrices: "0.025upaw"
  logLevel: "info"
  logFormat: "json"
  prometheusMetrics: true

persistence:
  enabled: true
  size: 500Gi
  storageClassName: "fast-ssd"

resources:
  limits:
    cpu: 4000m
    memory: 8Gi
  requests:
    cpu: 2000m
    memory: 4Gi

podDisruptionBudget:
  enabled: true
  minAvailable: 1

serviceMonitor:
  enabled: true
  interval: 30s
```

#### Development/Testing

```yaml
# dev-values.yaml
replicaCount: 1

image:
  repository: ghcr.io/decristofaroj/paw
  tag: "latest"
  pullPolicy: Always

node:
  chainId: "paw-testnet-1"
  moniker: "dev-node"
  minGasPrices: "0.001upaw"
  logLevel: "debug"

persistence:
  enabled: true
  size: 50Gi
  storageClassName: "standard"

resources:
  limits:
    cpu: 1000m
    memory: 2Gi
  requests:
    cpu: 500m
    memory: 1Gi
```

## Operations

### Upgrading

```bash
# Upgrade with new values
helm upgrade my-paw-node paw/paw -f updated-values.yaml

# Upgrade to new chart version
helm upgrade my-paw-node paw/paw --version 0.2.0
```

### Rollback

```bash
# Rollback to previous release
helm rollback my-paw-node

# Rollback to specific revision
helm rollback my-paw-node 2
```

### Uninstalling

```bash
# Uninstall the release
helm uninstall my-paw-node

# Uninstall and delete PVC
helm uninstall my-paw-node
kubectl delete pvc my-paw-node-paw-data
```

### Accessing the Node

```bash
# Port-forward RPC endpoint
kubectl port-forward svc/my-paw-node-paw 26657:26657

# Port-forward REST API
kubectl port-forward svc/my-paw-node-paw 1317:1317

# Execute commands in the pod
kubectl exec -it deployment/my-paw-node-paw -- pawd status

# View logs
kubectl logs -f deployment/my-paw-node-paw
```

### Monitoring

```bash
# Check pod status
kubectl get pods -l app.kubernetes.io/name=paw

# Check resource usage
kubectl top pod -l app.kubernetes.io/name=paw

# View events
kubectl get events --sort-by=.metadata.creationTimestamp

# Check Prometheus metrics (if enabled)
kubectl port-forward svc/my-paw-node-paw 26660:26660
curl http://localhost:26660/metrics
```

## Advanced Configuration

### Using Init Containers

To download genesis file or perform initialization:

```yaml
initContainers:
  - name: genesis-downloader
    image: curlimages/curl:latest
    command:
      - sh
      - -c
      - |
        curl -o /root/.paw/config/genesis.json \
          https://raw.githubusercontent.com/decristofaroj/paw/main/genesis/paw-mainnet-1.json
    volumeMounts:
      - name: data
        mountPath: /root/.paw
```

### State Sync Configuration

Add state sync configuration via environment variables:

```yaml
env:
  - name: STATESYNC_RPC_SERVERS
    value: "https://rpc1.paw.example.com:26657,https://rpc2.paw.example.com:26657"
  - name: STATESYNC_TRUST_HEIGHT
    value: "1000000"
  - name: STATESYNC_TRUST_HASH
    value: "ABC123..."
```

### Node Affinity

Run nodes on specific hardware:

```yaml
nodeSelector:
  node-type: validator
  disk-type: nvme

affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
            - key: app.kubernetes.io/name
              operator: In
              values:
                - paw
        topologyKey: kubernetes.io/hostname
```

### Tolerations

Allow scheduling on tainted nodes:

```yaml
tolerations:
  - key: "validator"
    operator: "Equal"
    value: "true"
    effect: "NoSchedule"
```

## Security Considerations

### Default Security Settings

This chart implements security best practices:
- Non-root user (UID 1000)
- Read-only root filesystem capability
- Dropped all capabilities
- No privilege escalation
- Security context enforcement

### Recommendations

1. **Secrets Management**: Use Kubernetes Secrets or external secret managers for sensitive data
2. **Network Policies**: Implement NetworkPolicies to restrict pod communication
3. **RBAC**: Configure appropriate ServiceAccount permissions
4. **Image Security**: Use signed images and scan for vulnerabilities
5. **Resource Limits**: Always set resource limits to prevent resource exhaustion

### Example Network Policy

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: paw-network-policy
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: paw
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 26656
        - protocol: TCP
          port: 26657
        - protocol: TCP
          port: 1317
        - protocol: TCP
          port: 9090
  egress:
    - to:
        - namespaceSelector: {}
    - to:
        - podSelector: {}
      ports:
        - protocol: TCP
          port: 53
        - protocol: UDP
          port: 53
```

## Troubleshooting

### Pod Not Starting

```bash
# Check pod status
kubectl describe pod <pod-name>

# Check logs
kubectl logs <pod-name>

# Check events
kubectl get events --field-selector involvedObject.name=<pod-name>
```

### Storage Issues

```bash
# Check PVC status
kubectl get pvc

# Check PV
kubectl get pv

# Describe PVC for events
kubectl describe pvc my-paw-node-paw-data
```

### Performance Issues

```bash
# Check resource usage
kubectl top pod <pod-name>

# Check node resources
kubectl top node

# Increase resources in values.yaml
resources:
  limits:
    cpu: 4000m
    memory: 8Gi
```

### Common Issues

| Issue | Solution |
|-------|----------|
| Pod in CrashLoopBackOff | Check logs for errors, verify genesis file exists |
| PVC stuck in Pending | Verify StorageClass exists and is provisioned |
| Cannot connect to RPC | Verify Service is created, check network policies |
| High memory usage | Increase memory limits, check for memory leaks |
| Slow sync | Increase CPU/Memory, consider state sync |

## Contributing

Contributions are welcome! Please:
1. Test changes locally with `helm lint` and `helm template`
2. Update documentation for new features
3. Follow Kubernetes and Helm best practices

## License

This chart is released under the same license as the PAW blockchain project.

## Support

- GitHub Issues: https://github.com/decristofaroj/paw/issues
- Documentation: See the main repository README
