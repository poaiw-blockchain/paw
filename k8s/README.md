# PAW Blockchain - Kubernetes Deployment

Complete guide for deploying PAW blockchain on Kubernetes.

## Quick Start

```bash
# Deploy with automated script
./scripts/deploy/deploy-k8s.sh

# Or deploy manually following this guide
```

## Architecture

The Kubernetes deployment consists of:

- **PAW Nodes**: StatefulSet with 3+ replicas for blockchain consensus
- **PAW API**: Deployment with 3+ replicas for high availability
- **Monitoring Stack**: Prometheus, Grafana, Loki, AlertManager
- **Storage**: Persistent volumes for blockchain data
- **Networking**: LoadBalancer/Ingress for external access
- **Autoscaling**: HPA for automatic scaling based on load

## Prerequisites

- Kubernetes cluster 1.28+
- kubectl configured
- At least 3 worker nodes
- Storage class with SSD support
- LoadBalancer support or Ingress controller
- cert-manager (for TLS)

## Directory Structure

```
k8s/
├── namespace.yaml                  # Namespace and quotas
├── configmap.yaml                  # Configuration
├── secrets.yaml                    # Secrets (template)
├── persistent-volumes.yaml         # Storage configuration
├── paw-node-deployment.yaml        # Node deployment
├── paw-api-deployment.yaml         # API deployment
├── all-services.yaml               # All services
├── ingress.yaml                    # Ingress configuration
├── hpa.yaml                        # Horizontal pod autoscaling
├── network-policy.yaml             # Network security
└── monitoring-deployment.yaml      # Monitoring stack
```

## Deployment Steps

### 1. Create Namespace

```bash
kubectl apply -f namespace.yaml
```

This creates:

- Namespace: `paw-blockchain`
- Resource quotas
- Limit ranges

### 2. Configure Secrets

```bash
# Generate secrets
JWT_SECRET=$(openssl rand -base64 32)
GRAFANA_PASSWORD=$(openssl rand -base64 16)

# Create secrets
kubectl create secret generic paw-secrets \
  --from-literal=JWT_SECRET="$JWT_SECRET" \
  --namespace=paw-blockchain

kubectl create secret generic monitoring-secrets \
  --from-literal=GRAFANA_ADMIN_PASSWORD="$GRAFANA_PASSWORD" \
  --namespace=paw-blockchain

# For TLS (if using cert-manager)
kubectl create secret tls paw-tls-cert \
  --cert=path/to/tls.crt \
  --key=path/to/tls.key \
  --namespace=paw-blockchain
```

### 3. Apply ConfigMaps

```bash
kubectl apply -f configmap.yaml

# Create monitoring configs
kubectl create configmap prometheus-config \
  --from-file=../monitoring/prometheus.yml \
  -n paw-blockchain

kubectl create configmap grafana-datasources \
  --from-file=../monitoring/grafana-datasources.yml \
  -n paw-blockchain

kubectl create configmap alertmanager-config \
  --from-file=../monitoring/alertmanager.yml \
  -n paw-blockchain

kubectl create configmap loki-config \
  --from-file=../monitoring/loki-config.yaml \
  -n paw-blockchain
```

### 4. Configure Storage

```bash
kubectl apply -f persistent-volumes.yaml

# Verify PVCs are bound
kubectl get pvc -n paw-blockchain -w
```

### 5. Deploy Monitoring

```bash
kubectl apply -f monitoring-deployment.yaml

# Wait for pods to be ready
kubectl wait --for=condition=ready pod \
  -l component=monitoring \
  -n paw-blockchain \
  --timeout=300s
```

### 6. Deploy Blockchain

```bash
# Deploy nodes
kubectl apply -f paw-node-deployment.yaml

# Deploy API
kubectl apply -f paw-api-deployment.yaml

# Create services
kubectl apply -f all-services.yaml
```

### 7. Configure Scaling and Network

```bash
# Apply HPA
kubectl apply -f hpa.yaml

# Apply network policies
kubectl apply -f network-policy.yaml

# Configure ingress
kubectl apply -f ingress.yaml
```

### 8. Verify Deployment

```bash
# Check all pods
kubectl get pods -n paw-blockchain

# Check services
kubectl get svc -n paw-blockchain

# Check HPA status
kubectl get hpa -n paw-blockchain

# View logs
kubectl logs -f -l app=paw-node -n paw-blockchain
```

## Accessing Services

### Port Forwarding (Development)

```bash
# PAW Node RPC
kubectl port-forward svc/paw-node 26657:26657 -n paw-blockchain

# PAW API
kubectl port-forward svc/paw-api 5000:5000 -n paw-blockchain

# Prometheus
kubectl port-forward svc/prometheus 9090:9090 -n paw-blockchain

# Grafana
kubectl port-forward svc/grafana 3000:3000 -n paw-blockchain
```

### Production Access (via Ingress)

Configure DNS to point to your ingress:

- api.paw-chain.io -> API
- rpc.paw-chain.io -> RPC
- grpc.paw-chain.io -> gRPC
- grafana.internal.paw-chain.io -> Grafana
- prometheus.internal.paw-chain.io -> Prometheus

## Scaling

### Manual Scaling

```bash
# Scale nodes
kubectl scale deployment/paw-node --replicas=5 -n paw-blockchain

# Scale API
kubectl scale deployment/paw-api --replicas=10 -n paw-blockchain
```

### Automatic Scaling (HPA)

HPA is configured to scale based on:

- CPU utilization (70%)
- Memory utilization (80%)
- Custom metrics (requests per second)

View HPA status:

```bash
kubectl get hpa -n paw-blockchain
kubectl describe hpa paw-api-hpa -n paw-blockchain
```

## Resource Management

### View Resource Usage

```bash
# Current resource usage
kubectl top pods -n paw-blockchain
kubectl top nodes

# Resource quotas
kubectl describe resourcequota -n paw-blockchain
```

### Adjust Resource Limits

Edit deployment files:

```yaml
resources:
  requests:
    cpu: '2'
    memory: '4Gi'
  limits:
    cpu: '4'
    memory: '8Gi'
```

Apply changes:

```bash
kubectl apply -f paw-node-deployment.yaml
```

## Monitoring

### Accessing Monitoring Tools

```bash
# Grafana
kubectl port-forward svc/grafana 3000:3000 -n paw-blockchain
# Open: http://localhost:3000
# Default credentials: admin / <from secret>

# Prometheus
kubectl port-forward svc/prometheus 9090:9090 -n paw-blockchain
# Open: http://localhost:9090

# AlertManager
kubectl port-forward svc/alertmanager 9093:9093 -n paw-blockchain
# Open: http://localhost:9093
```

### Key Metrics

Monitor these metrics in Grafana:

- Block height
- Block time
- Validator count
- Peer count
- Transaction throughput
- API request rate
- Error rates
- Resource utilization

## Backup and Recovery

### Backup

```bash
# Automated backup
../scripts/deploy/backup.sh k8s

# Manual backup of specific pod
POD=$(kubectl get pods -n paw-blockchain -l app=paw-node -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n paw-blockchain $POD -- tar czf /tmp/backup.tar.gz -C /paw/.paw data config
kubectl cp paw-blockchain/$POD:/tmp/backup.tar.gz ./backup.tar.gz
```

### Restore

```bash
# Automated restore
../scripts/deploy/restore.sh k8s <backup-file>

# Manual restore
POD=$(kubectl get pods -n paw-blockchain -l app=paw-node -o jsonpath='{.items[0].metadata.name}')
kubectl cp ./backup.tar.gz paw-blockchain/$POD:/tmp/backup.tar.gz
kubectl exec -n paw-blockchain $POD -- tar xzf /tmp/backup.tar.gz -C /paw/.paw
kubectl delete pod $POD -n paw-blockchain  # Restart pod
```

## Troubleshooting

### Pod Not Starting

```bash
# Describe pod
kubectl describe pod <pod-name> -n paw-blockchain

# Check logs
kubectl logs <pod-name> -n paw-blockchain

# Check events
kubectl get events -n paw-blockchain --sort-by='.lastTimestamp'
```

### PVC Not Binding

```bash
# Check PVC status
kubectl get pvc -n paw-blockchain

# Describe PVC
kubectl describe pvc paw-node-data -n paw-blockchain

# Check storage class
kubectl get storageclass
```

### Network Issues

```bash
# Test service connectivity
kubectl run -it --rm debug --image=alpine --restart=Never -n paw-blockchain -- sh
# Inside pod: wget -O- http://paw-node:26657/health

# Check network policies
kubectl get networkpolicy -n paw-blockchain

# Describe network policy
kubectl describe networkpolicy paw-node-network-policy -n paw-blockchain
```

### Services Not Routing to New Pods

If deployments were created with different app labels (e.g., `paw-node-single` / `paw-api-single`), services targeting `app=paw-node` or `app=paw-api` will not route traffic. Delete and recreate the deployments so selectors match:

```bash
kubectl delete deployment/paw-node-single deployment/paw-api-single -n paw-blockchain --ignore-not-found
./scripts/deploy/deploy-k8s.sh
```

### High Resource Usage

```bash
# Check resource usage
kubectl top pods -n paw-blockchain

# Adjust resource limits in deployment
# Then apply:
kubectl apply -f paw-node-deployment.yaml

# Force restart pods
kubectl rollout restart deployment/paw-node -n paw-blockchain
```

## Upgrading

### Rolling Update

```bash
# Update image version in deployment
kubectl set image deployment/paw-node \
  paw-node=paw-chain/paw:v2.0.0 \
  -n paw-blockchain

# Watch rollout status
kubectl rollout status deployment/paw-node -n paw-blockchain

# Rollback if needed
kubectl rollout undo deployment/paw-node -n paw-blockchain
```

### Chain Upgrade

For governance-approved chain upgrades:

```bash
# Deploy upgrade handler
# Wait for upgrade height
# Pods will automatically restart with new binary
```

## Security

### Network Policies

Network policies are configured to:

- Allow P2P traffic from anywhere
- Restrict RPC access to API pods and monitoring
- Restrict gRPC access to API pods
- Allow metrics scraping from Prometheus

### RBAC

Service accounts are configured with minimal permissions:

- paw-node: Can only read/write own data
- paw-api: Can only access node services
- prometheus: Can scrape metrics

### Secrets Management

Use external secret management in production:

- Sealed Secrets
- External Secrets Operator
- HashiCorp Vault
- Cloud provider secret managers

## Performance Tuning

### Node Optimization

```yaml
# In configmap.yaml
pruning: 'custom'
pruning-keep-recent: '100'
pruning-interval: '10'
min-retain-blocks: '0'
```

### Storage Optimization

Use fast SSD storage:

- AWS: gp3 with high IOPS
- GCP: pd-ssd
- Azure: Premium SSD

### Network Optimization

- Use LoadBalancer for P2P traffic
- Configure pod anti-affinity for distribution
- Use node affinity for SSD-equipped nodes

## Best Practices

1. **High Availability**: Deploy at least 3 node replicas
2. **Monitoring**: Set up alerts for critical metrics
3. **Backups**: Automated daily backups
4. **Security**: Use network policies and RBAC
5. **Resources**: Set appropriate requests and limits
6. **Updates**: Use rolling updates with health checks
7. **Logging**: Centralized logging with Loki
8. **Testing**: Test upgrades in staging first

## Additional Resources

- [Main Deployment Guide](../DEPLOYMENT.md)
- [Monitoring Guide](../monitoring/README.md)
- [Security Runbook](../docs/SECURITY_RUNBOOK.md)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
