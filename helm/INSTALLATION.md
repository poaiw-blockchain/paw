# PAW Blockchain Helm Chart Installation Guide

Complete guide for deploying PAW blockchain nodes on Kubernetes using Helm.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [Installation Methods](#installation-methods)
4. [Configuration](#configuration)
5. [Post-Installation](#post-installation)
6. [Troubleshooting](#troubleshooting)

## Prerequisites

### Required Tools

- **Kubernetes Cluster**: v1.19 or higher
  - Local: Minikube, kind, k3s, Docker Desktop
  - Cloud: GKE, EKS, AKS, DigitalOcean Kubernetes

- **Helm**: v3.0 or higher
  ```bash
  # Install Helm
  curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

  # Verify installation
  helm version
  ```

- **kubectl**: Compatible with your cluster version
  ```bash
  # Verify kubectl is configured
  kubectl cluster-info
  kubectl get nodes
  ```

### Optional Tools

- **Helmfile**: For managing multiple releases
- **yamllint**: For validating YAML files
- **kubectx/kubens**: For managing contexts and namespaces

### Cluster Requirements

| Resource | Minimum | Recommended (Production) |
|----------|---------|-------------------------|
| Nodes | 1 | 3+ |
| CPU per node | 2 cores | 4+ cores |
| Memory per node | 4 GB | 8+ GB |
| Storage | 100 GB | 500+ GB SSD |
| Storage Class | Dynamic provisioning | Fast SSD (NVMe) |

## Quick Start

### 1. Clone the Repository

```bash
git clone https://github.com/decristofaroj/paw.git
cd paw/helm
```

### 2. Create Namespace

```bash
kubectl create namespace paw
```

### 3. Install with Default Values

```bash
# Install the chart
helm install my-paw-node ./paw -n paw

# Watch the deployment
kubectl get pods -n paw -w
```

### 4. Verify Installation

```bash
# Check pod status
kubectl get pods -n paw

# Check service
kubectl get svc -n paw

# View logs
kubectl logs -f deployment/my-paw-node-paw -n paw
```

## Installation Methods

### Method 1: Default Installation

Best for: Testing, development, getting started quickly.

```bash
helm install my-paw-node ./paw -n paw
```

### Method 2: Development Installation

Best for: Development environments, testing features.

```bash
helm install my-paw-node ./paw \
  -f ./paw/values-dev.yaml \
  -n paw
```

### Method 3: Production Installation

Best for: Production validators, mainnet deployment.

```bash
helm install my-paw-node ./paw \
  -f ./paw/values-production.yaml \
  -n paw
```

### Method 4: Custom Configuration

Best for: Specific requirements, custom setups.

1. Copy the default values:
   ```bash
   cp paw/values.yaml my-custom-values.yaml
   ```

2. Edit the file with your settings:
   ```bash
   vim my-custom-values.yaml
   ```

3. Install with custom values:
   ```bash
   helm install my-paw-node ./paw \
     -f my-custom-values.yaml \
     -n paw
   ```

### Method 5: Command-line Overrides

Best for: Quick parameter changes, testing different configurations.

```bash
helm install my-paw-node ./paw \
  --set node.chainId=paw-mainnet-1 \
  --set node.moniker=my-validator \
  --set persistence.size=500Gi \
  --set resources.limits.cpu=4000m \
  --set resources.limits.memory=8Gi \
  -n paw
```

## Configuration

### Essential Configuration Parameters

#### Chain Configuration

```yaml
node:
  chainId: "paw-mainnet-1"        # Network to join
  moniker: "my-validator"         # Your node name
  minGasPrices: "0.025upaw"       # Minimum gas prices
  logLevel: "info"                # Log level (debug|info|warn|error)
  logFormat: "json"               # Log format (json|plain)
```

#### Storage Configuration

```yaml
persistence:
  enabled: true
  size: 500Gi                     # Blockchain data storage
  storageClassName: "fast-ssd"    # Storage class (optional)
  accessMode: ReadWriteOnce
```

#### Resource Configuration

```yaml
resources:
  limits:
    cpu: 2000m                    # Maximum CPU
    memory: 4Gi                   # Maximum memory
  requests:
    cpu: 1000m                    # Requested CPU
    memory: 2Gi                   # Requested memory
```

### Configuration Examples

#### Example 1: Testnet Validator

```yaml
# testnet-validator.yaml
node:
  chainId: "paw-testnet-1"
  moniker: "testnet-validator-01"
  minGasPrices: "0.001upaw"

persistence:
  size: 100Gi
  storageClassName: "standard"

resources:
  limits:
    cpu: 2000m
    memory: 4Gi
  requests:
    cpu: 1000m
    memory: 2Gi
```

Deploy:
```bash
helm install testnet-val ./paw -f testnet-validator.yaml -n paw
```

#### Example 2: Mainnet Validator

```yaml
# mainnet-validator.yaml
node:
  chainId: "paw-mainnet-1"
  moniker: "mainnet-validator-01"
  minGasPrices: "0.025upaw"
  logLevel: "info"
  logFormat: "json"

persistence:
  size: 500Gi
  storageClassName: "fast-ssd"

resources:
  limits:
    cpu: 4000m
    memory: 8Gi
  requests:
    cpu: 2000m
    memory: 4Gi

nodeSelector:
  workload-type: validator

podDisruptionBudget:
  enabled: true
  minAvailable: 1
```

Deploy:
```bash
helm install mainnet-val ./paw -f mainnet-validator.yaml -n paw
```

#### Example 3: Sentry Node

```yaml
# sentry-node.yaml
replicaCount: 3  # Multiple sentry nodes

node:
  chainId: "paw-mainnet-1"
  moniker: "sentry-node"
  minGasPrices: "0.025upaw"

service:
  type: LoadBalancer  # Expose to internet

persistence:
  size: 500Gi

resources:
  limits:
    cpu: 2000m
    memory: 4Gi

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

Deploy:
```bash
helm install sentry-nodes ./paw -f sentry-node.yaml -n paw
```

## Post-Installation

### 1. Verify Pod is Running

```bash
# Check pod status
kubectl get pods -n paw

# Expected output:
# NAME                              READY   STATUS    RESTARTS   AGE
# my-paw-node-paw-xxxxxxxxxx-xxxxx  1/1     Running   0          5m
```

### 2. Check Logs

```bash
# View real-time logs
kubectl logs -f deployment/my-paw-node-paw -n paw

# View last 100 lines
kubectl logs --tail=100 deployment/my-paw-node-paw -n paw
```

### 3. Access the Node

#### Port Forwarding (Development)

```bash
# Forward RPC port
kubectl port-forward svc/my-paw-node-paw 26657:26657 -n paw

# In another terminal, test RPC
curl http://localhost:26657/status
```

#### Load Balancer (Production)

```bash
# Get external IP
kubectl get svc my-paw-node-paw -n paw

# Access via external IP
curl http://<EXTERNAL-IP>:26657/status
```

### 4. Execute Commands in Pod

```bash
# Get a shell in the pod
kubectl exec -it deployment/my-paw-node-paw -n paw -- /bin/sh

# Check node status
kubectl exec -it deployment/my-paw-node-paw -n paw -- pawd status

# Query blockchain state
kubectl exec -it deployment/my-paw-node-paw -n paw -- pawd query bank balances <address>
```

### 5. Monitor Resources

```bash
# Check resource usage
kubectl top pod -n paw

# Check PVC
kubectl get pvc -n paw

# Check events
kubectl get events -n paw --sort-by=.metadata.creationTimestamp
```

## Upgrading

### Upgrade Chart

```bash
# Upgrade with new values
helm upgrade my-paw-node ./paw -f updated-values.yaml -n paw

# Upgrade to new chart version
helm upgrade my-paw-node ./paw --version 0.2.0 -n paw

# View upgrade history
helm history my-paw-node -n paw
```

### Rollback

```bash
# Rollback to previous release
helm rollback my-paw-node -n paw

# Rollback to specific revision
helm rollback my-paw-node 2 -n paw
```

## Uninstalling

### Remove Release

```bash
# Uninstall the release (keeps PVC)
helm uninstall my-paw-node -n paw

# Delete PVC (WARNING: This deletes blockchain data!)
kubectl delete pvc my-paw-node-paw-data -n paw

# Delete namespace
kubectl delete namespace paw
```

## Troubleshooting

### Pod Won't Start

```bash
# Check pod events
kubectl describe pod <pod-name> -n paw

# Common issues:
# 1. ImagePullBackOff - Check image repository and tag
# 2. Pending - Check PVC status and storage class
# 3. CrashLoopBackOff - Check logs for errors
```

### PVC Issues

```bash
# Check PVC status
kubectl get pvc -n paw

# If Pending, check storage class
kubectl get storageclass

# Check PV
kubectl get pv
```

### Network Issues

```bash
# Check service
kubectl get svc my-paw-node-paw -n paw

# Check endpoints
kubectl get endpoints my-paw-node-paw -n paw

# Test connectivity from another pod
kubectl run -it --rm debug --image=busybox --restart=Never -n paw -- sh
# Inside the pod:
# wget -O- http://my-paw-node-paw:26657/status
```

### Performance Issues

```bash
# Check resource usage
kubectl top pod -n paw
kubectl top node

# If resources are maxed out, increase limits:
helm upgrade my-paw-node ./paw \
  --set resources.limits.cpu=4000m \
  --set resources.limits.memory=8Gi \
  -n paw
```

### Logs Not Showing

```bash
# Check if pod is running
kubectl get pods -n paw

# Check pod events
kubectl describe pod <pod-name> -n paw

# View previous pod logs (if crashed)
kubectl logs <pod-name> -n paw --previous
```

## Advanced Topics

### State Sync

To speed up initial sync, configure state sync in an init container:

```yaml
initContainers:
  - name: state-sync-config
    image: busybox:latest
    command:
      - sh
      - -c
      - |
        cat > /root/.paw/config/config.toml << EOF
        [statesync]
        enable = true
        rpc_servers = "https://rpc1.paw.example.com:26657,https://rpc2.paw.example.com:26657"
        trust_height = 1000000
        trust_hash = "ABC123..."
        EOF
    volumeMounts:
      - name: data
        mountPath: /root/.paw
```

### Monitoring with Prometheus

If using Prometheus Operator:

```yaml
serviceMonitor:
  enabled: true
  interval: 30s
  labels:
    release: prometheus  # Must match Prometheus Operator release
```

### Backup and Restore

```bash
# Backup blockchain data
kubectl exec deployment/my-paw-node-paw -n paw -- tar czf - /root/.paw > paw-backup.tar.gz

# Restore (copy to pod, then extract)
kubectl cp paw-backup.tar.gz paw/my-paw-node-paw-xxx:/tmp/
kubectl exec deployment/my-paw-node-paw -n paw -- tar xzf /tmp/paw-backup.tar.gz -C /
```

## Support

For issues or questions:
- GitHub Issues: https://github.com/decristofaroj/paw/issues
- Documentation: See the main repository README
- Chart README: See helm/paw/README.md
