# PAW Helm Chart Quick Reference

Quick command reference for common operations.

## Installation

```bash
# Development
helm install paw-dev ./paw -f ./paw/values-dev.yaml -n paw

# Production
helm install paw-prod ./paw -f ./paw/values-production.yaml -n paw

# Custom
helm install my-node ./paw -f my-values.yaml -n paw

# With overrides
helm install my-node ./paw --set node.chainId=paw-mainnet-1 -n paw
```

## Management

```bash
# List releases
helm list -n paw

# Get status
helm status my-node -n paw

# Get values
helm get values my-node -n paw

# Get manifest
helm get manifest my-node -n paw

# Upgrade
helm upgrade my-node ./paw -f updated-values.yaml -n paw

# Rollback
helm rollback my-node -n paw

# Uninstall
helm uninstall my-node -n paw
```

## Kubernetes Operations

```bash
# Pods
kubectl get pods -n paw
kubectl describe pod <pod-name> -n paw
kubectl logs -f deployment/my-node-paw -n paw
kubectl exec -it deployment/my-node-paw -n paw -- /bin/sh

# Services
kubectl get svc -n paw
kubectl describe svc my-node-paw -n paw

# PVC
kubectl get pvc -n paw
kubectl describe pvc my-node-paw-data -n paw

# Events
kubectl get events -n paw --sort-by=.metadata.creationTimestamp

# Resources
kubectl top pod -n paw
kubectl top node
```

## Port Forwarding

```bash
# RPC
kubectl port-forward svc/my-node-paw 26657:26657 -n paw

# API
kubectl port-forward svc/my-node-paw 1317:1317 -n paw

# gRPC
kubectl port-forward svc/my-node-paw 9090:9090 -n paw

# Metrics
kubectl port-forward svc/my-node-paw 26660:26660 -n paw
```

## Node Commands

```bash
# Status
kubectl exec deployment/my-node-paw -n paw -- pawd status

# Version
kubectl exec deployment/my-node-paw -n paw -- pawd version

# Query balance
kubectl exec deployment/my-node-paw -n paw -- pawd query bank balances <address>

# List keys
kubectl exec deployment/my-node-paw -n paw -- pawd keys list
```

## Troubleshooting

```bash
# Pod not starting
kubectl describe pod <pod-name> -n paw
kubectl logs <pod-name> -n paw
kubectl logs <pod-name> -n paw --previous

# PVC issues
kubectl get pvc -n paw
kubectl get storageclass
kubectl get pv

# Network issues
kubectl get svc -n paw
kubectl get endpoints my-node-paw -n paw

# Resource issues
kubectl top pod -n paw
kubectl top node
```

## Common Value Overrides

```bash
# Change chain ID
--set node.chainId=paw-mainnet-1

# Change moniker
--set node.moniker=my-validator

# Increase storage
--set persistence.size=500Gi

# Change storage class
--set persistence.storageClassName=fast-ssd

# Increase resources
--set resources.limits.cpu=4000m \
--set resources.limits.memory=8Gi

# Enable service monitor
--set serviceMonitor.enabled=true

# Disable persistence (testing only!)
--set persistence.enabled=false
```

## Validation

```bash
# Lint chart
helm lint ./paw

# Dry-run install
helm install test-release ./paw --dry-run -n paw

# Template (generate manifests)
helm template test-release ./paw -n paw

# Template with values
helm template test-release ./paw -f ./paw/values-production.yaml -n paw
```

## Cleanup

```bash
# Uninstall release
helm uninstall my-node -n paw

# Delete PVC (deletes data!)
kubectl delete pvc my-node-paw-data -n paw

# Delete namespace
kubectl delete namespace paw

# Complete cleanup
helm uninstall my-node -n paw && \
kubectl delete pvc --all -n paw && \
kubectl delete namespace paw
```
