# PAW Blockchain - Production Deployment Guide

Complete guide for deploying PAW blockchain to production environments.

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Infrastructure Setup](#infrastructure-setup)
4. [Deployment Methods](#deployment-methods)
5. [Configuration](#configuration)
6. [Monitoring](#monitoring)
7. [Operations](#operations)
8. [Security](#security)
9. [Troubleshooting](#troubleshooting)

## Overview

This guide covers deploying PAW blockchain nodes and validators in production using:

- **Docker** - Containerized deployment for local testing
- **Kubernetes** - Production-grade orchestration
- **Monitoring** - Prometheus, Grafana, and AlertManager
- **CI/CD** - Automated build, test, and deployment pipelines

### Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Load Balancer                         │
└──────────────┬──────────────────────────┬───────────────┘
               │                          │
    ┌──────────▼─────────┐    ┌──────────▼─────────┐
    │   Full Nodes       │    │   Validators       │
    │  (RPC/API/gRPC)    │    │  (Consensus Only)  │
    └──────────┬─────────┘    └──────────┬─────────┘
               │                          │
               └──────────┬───────────────┘
                          │
                  ┌───────▼────────┐
                  │  Monitoring    │
                  │  (Prometheus)  │
                  └────────────────┘
```

## Prerequisites

### Required Software

- **Docker** (20.10+)
- **Kubernetes** (1.28+)
- **kubectl** (1.28+)
- **Helm** (3.0+) - Optional but recommended
- **Go** (1.23.1+) - For building from source
- **AWS CLI** - If using AWS

### Recommended Resources

#### Full Node

- **CPU**: 4 cores
- **Memory**: 8 GB RAM
- **Storage**: 500 GB SSD
- **Network**: 100 Mbps

#### Validator Node

- **CPU**: 8 cores
- **Memory**: 16 GB RAM
- **Storage**: 1 TB SSD (NVMe preferred)
- **Network**: 1 Gbps

### Access Requirements

- Kubernetes cluster with admin access
- Container registry access (GitHub Container Registry)
- S3 bucket for backups (optional)
- Monitoring infrastructure (Prometheus/Grafana)

## Infrastructure Setup

### 1. Kubernetes Cluster Setup

#### Using AWS EKS

```bash
# Create EKS cluster
eksctl create cluster \
  --name paw-production \
  --region us-east-1 \
  --node-type m5.2xlarge \
  --nodes 5 \
  --nodes-min 3 \
  --nodes-max 10 \
  --with-oidc \
  --managed

# Configure kubectl
aws eks update-kubeconfig --name paw-production --region us-east-1
```

#### Using GKE

```bash
# Create GKE cluster
gcloud container clusters create paw-production \
  --region us-central1 \
  --node-locations us-central1-a,us-central1-b,us-central1-c \
  --num-nodes 2 \
  --machine-type n2-standard-8 \
  --disk-type pd-ssd \
  --disk-size 200 \
  --enable-autoscaling \
  --min-nodes 3 \
  --max-nodes 10

# Get credentials
gcloud container clusters get-credentials paw-production --region us-central1
```

### 2. Create Namespace and Resources

```bash
# Create namespace
kubectl apply -f k8s/namespace.yaml

# Create storage classes
kubectl apply -f k8s/storage.yaml

# Verify
kubectl get namespace paw-blockchain
kubectl get storageclass
```

### 3. Configure Secrets

#### Create Validator Keys Secret

```bash
# Generate validator keys (if new validator)
pawd init my-validator --chain-id paw-1 --home /tmp/validator-init
cd /tmp/validator-init/config

# Create Kubernetes secret
kubectl create secret generic paw-validator-keys \
  --from-file=priv_validator_key.json \
  --from-file=node_key.json \
  --namespace=paw-blockchain

# IMPORTANT: Backup these keys!
cp priv_validator_key.json ~/secure-backup/
cp node_key.json ~/secure-backup/
```

#### Create Genesis Secret

```bash
# Download or create genesis file
curl -o genesis.json https://genesis.paw.network/genesis.json

# Create secret
kubectl create secret generic paw-genesis \
  --from-file=genesis.json \
  --namespace=paw-blockchain
```

#### Create API Keys Secret

```bash
# Create from template
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: paw-api-keys
  namespace: paw-blockchain
type: Opaque
stringData:
  grafana-admin-password: "your-secure-password"
  alertmanager-slack-webhook: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
  backup-s3-access-key: "your-aws-access-key"
  backup-s3-secret-key: "your-aws-secret-key"
EOF
```

## Deployment Methods

### Method 1: Using Deployment Scripts (Recommended)

#### Deploy Full Nodes

```bash
# Make scripts executable
chmod +x scripts/deploy/*.sh

# Deploy 3 full nodes
./scripts/deploy/deploy-node.sh \
  --namespace paw-blockchain \
  --replicas 3 \
  --chain-id paw-1 \
  --image-tag v1.0.0

# Verify deployment
kubectl get pods -n paw-blockchain -l component=node
kubectl logs -f -n paw-blockchain paw-node-0
```

#### Deploy Validators

```bash
# Deploy validators (requires validator keys)
./scripts/deploy/deploy-validator.sh \
  --namespace paw-blockchain \
  --replicas 3 \
  --chain-id paw-1 \
  --image-tag v1.0.0 \
  --keys-path /path/to/validator/keys

# Verify validators
kubectl get pods -n paw-blockchain -l component=validator
kubectl exec -n paw-blockchain paw-validator-0 -- pawcli status
```

### Method 2: Using kubectl Directly

```bash
# Deploy configuration
kubectl apply -f k8s/configmaps.yaml

# Deploy services
kubectl apply -f k8s/services.yaml

# Deploy nodes
kubectl apply -f k8s/node-deployment.yaml

# Deploy validators
kubectl apply -f k8s/validator-statefulset.yaml

# Check status
kubectl get all -n paw-blockchain
```

### Method 3: Using Docker Compose (Development Only)

```bash
# Start local testnet
docker-compose -f docker/docker-compose.yml up -d

# View logs
docker-compose logs -f validator1

# Stop testnet
docker-compose down
```

## Configuration

### Environment-Specific Configuration

#### Testnet

```yaml
# k8s/overlays/testnet/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: paw-testnet

resources:
  - ../../base

configMapGenerator:
  - name: paw-config
    literals:
      - chain_id=paw-testnet-1
      - minimum_gas_prices=0.001upaw

replicas:
  - name: paw-node
    count: 3
  - name: paw-validator
    count: 3
```

#### Production

```yaml
# k8s/overlays/production/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: paw-blockchain

resources:
  - ../../base

configMapGenerator:
  - name: paw-config
    literals:
      - chain_id=paw-1
      - minimum_gas_prices=0.001upaw

replicas:
  - name: paw-node
    count: 5
  - name: paw-validator
    count: 5
```

### Node Configuration

See `config/node-config.toml.template` for full node configuration options.

Key settings:

```toml
[p2p]
max_num_inbound_peers = 40
max_num_outbound_peers = 10

[mempool]
size = 5000

[consensus]
timeout_commit = "4s"
```

### Validator Configuration

See `config/validator-config.toml.template` for validator-specific configuration.

Key security settings:

```toml
[rpc]
laddr = "tcp://127.0.0.1:26657"  # Restrict RPC access

[p2p]
filter_peers = true
private_peer_ids = "sentry-node-id-1,sentry-node-id-2"
```

## Monitoring

### Deploy Monitoring Stack

```bash
# Deploy Prometheus
kubectl apply -f monitoring/prometheus/

# Deploy Grafana
kubectl apply -f monitoring/grafana/

# Deploy AlertManager
kubectl apply -f monitoring/alertmanager/

# Port forward to access Grafana
kubectl port-forward -n paw-blockchain svc/grafana 3000:3000

# Access at http://localhost:3000
# Default credentials: admin / (see secret)
```

### Key Metrics to Monitor

1. **Chain Health**
   - Block height
   - Block time
   - Transaction rate
   - Missed blocks

2. **Validator Performance**
   - Signing percentage
   - Double sign protection
   - Uptime

3. **Node Performance**
   - Peer count
   - Memory usage
   - Disk usage
   - CPU usage

4. **Network Health**
   - P2P connections
   - Network latency
   - Mempool size

### Alerts

Configure alerts in `monitoring/prometheus/alerts.yml`:

- ValidatorDown (critical)
- ValidatorNotSigning (critical)
- ChainHalted (critical)
- LowPeerCount (warning)
- DiskSpaceCritical (critical)

## Operations

### Backup and Restore

#### Create Backup

```bash
# Full backup including validator keys and data
./scripts/deploy/backup-state.sh \
  --namespace paw-blockchain \
  --tag manual-backup-$(date +%Y%m%d) \
  --include-data \
  --s3-bucket paw-backups

# Quick backup (keys and config only)
./scripts/deploy/backup-state.sh \
  --namespace paw-blockchain \
  --tag quick-backup
```

#### Restore from Backup

```bash
# Restore from local backup
./scripts/deploy/restore-state.sh \
  --namespace paw-blockchain \
  --backup manual-backup-20240101

# Restore from S3
./scripts/deploy/restore-state.sh \
  --namespace paw-blockchain \
  --backup manual-backup-20240101 \
  --s3-bucket paw-backups
```

### Chain Upgrades

```bash
# Upgrade to new version
./scripts/deploy/upgrade-chain.sh \
  --namespace paw-blockchain \
  --version v1.1.0 \
  --upgrade-height 1000000

# The script will:
# 1. Create backup
# 2. Submit upgrade proposal
# 3. Update container images
# 4. Monitor upgrade progress
# 5. Verify upgrade success
```

### Scaling

#### Scale Full Nodes

```bash
# Scale to 5 nodes
kubectl scale deployment paw-node \
  --replicas=5 \
  -n paw-blockchain

# Verify
kubectl get pods -n paw-blockchain -l component=node
```

#### Scale Validators

```bash
# Scale validators (use with caution)
kubectl scale statefulset paw-validator \
  --replicas=5 \
  -n paw-blockchain

# Note: Scaling validators requires coordination and governance
```

## Security

### Best Practices

1. **Validator Security**
   - Use sentry architecture
   - Restrict RPC access
   - Enable double-sign protection
   - Regular key backups
   - Monitor for anomalies

2. **Network Security**
   - Use NetworkPolicies
   - Enable TLS for external endpoints
   - Implement rate limiting
   - DDoS protection

3. **Access Control**
   - RBAC for Kubernetes
   - Separate namespaces for environments
   - Audit logging
   - Secret rotation

4. **Key Management**
   - Hardware security modules (HSM)
   - Encrypted backups
   - Secure key distribution
   - Regular security audits

### Sentry Architecture

```
Internet
    │
    ▼
[Load Balancer]
    │
    ├──▶ [Sentry Node 1] ──┐
    ├──▶ [Sentry Node 2] ──┼──▶ [Validator] (private)
    └──▶ [Sentry Node 3] ──┘
```

Configure sentry nodes in validator config:

```toml
[p2p]
private_peer_ids = "validator-node-id"
```

## Troubleshooting

### Common Issues

#### Nodes Not Syncing

```bash
# Check peer connections
kubectl exec -n paw-blockchain paw-node-0 -- \
  curl http://localhost:26657/net_info

# Check logs
kubectl logs -n paw-blockchain paw-node-0 --tail=100

# Restart node
kubectl delete pod paw-node-0 -n paw-blockchain
```

#### Validator Not Signing

```bash
# Check validator status
kubectl exec -n paw-blockchain paw-validator-0 -- \
  pawcli status

# Check validator signing info
kubectl exec -n paw-blockchain paw-validator-0 -- \
  pawd query slashing signing-info \
  $(pawd tendermint show-validator)

# Check for double signs
kubectl logs -n paw-blockchain paw-validator-0 | grep -i "double"
```

#### High Resource Usage

```bash
# Check pod resources
kubectl top pods -n paw-blockchain

# Check node resources
kubectl top nodes

# Scale down or upgrade resources
kubectl edit deployment paw-node -n paw-blockchain
```

#### Storage Issues

```bash
# Check PVC status
kubectl get pvc -n paw-blockchain

# Check volume usage
kubectl exec -n paw-blockchain paw-node-0 -- df -h

# Expand volume (if supported)
kubectl edit pvc data-paw-node-0 -n paw-blockchain
```

### Getting Help

- **Documentation**: https://docs.paw.network
- **Discord**: https://discord.gg/paw
- **GitHub Issues**: https://github.com/paw-chain/paw/issues
- **Security**: security@paw.network

## Additional Resources

- [Configuration Guide](config/README.md)
- [Monitoring Guide](monitoring/README.md)
- [Security Best Practices](SECURITY.md)
- [API Documentation](docs/API.md)
- [Validator Guide](docs/VALIDATOR_GUIDE.md)

## License

Copyright © 2024 PAW Network. All rights reserved.
