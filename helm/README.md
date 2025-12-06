# PAW Blockchain Helm Charts

Production-ready Kubernetes deployment for PAW blockchain nodes.

## Quick Start

```bash
# Install with default values
helm install my-paw-node ./paw -n paw

# Install for production
helm install my-paw-node ./paw -f ./paw/values-production.yaml -n paw

# Install for development
helm install my-paw-node ./paw -f ./paw/values-dev.yaml -n paw
```

## Documentation

- **[Installation Guide](INSTALLATION.md)** - Complete installation and deployment guide
- **[Quick Reference](QUICKREF.md)** - Common commands and operations
- **[Chart README](paw/README.md)** - Detailed chart documentation

## Chart Structure

```
helm/paw/
├── Chart.yaml                    # Chart metadata
├── values.yaml                   # Default configuration values
├── values-production.yaml        # Production configuration example
├── values-dev.yaml              # Development configuration example
├── templates/
│   ├── _helpers.tpl             # Template helper functions
│   ├── deployment.yaml          # Validator deployment
│   ├── service.yaml             # Service definition
│   ├── pvc.yaml                 # Persistent volume claim
│   ├── serviceaccount.yaml      # Service account
│   ├── configmap.yaml           # Configuration map
│   ├── servicemonitor.yaml      # Prometheus ServiceMonitor
│   ├── poddisruptionbudget.yaml # Pod disruption budget
│   └── NOTES.txt                # Post-install notes
└── README.md                     # Chart documentation
```

## Features

- **Production-Ready**: Implements security best practices and resource management
- **Flexible Configuration**: Extensive values.yaml with sensible defaults
- **High Availability**: Support for pod disruption budgets and anti-affinity
- **Monitoring**: Built-in Prometheus metrics and ServiceMonitor support
- **Persistent Storage**: Configurable persistent volumes for blockchain data
- **Security**: Non-root containers, dropped capabilities, security contexts
- **Health Checks**: Liveness, readiness, and startup probes
- **Resource Management**: Configurable CPU and memory limits

## Requirements

- Kubernetes 1.19+
- Helm 3.0+
- PersistentVolume provisioner (if persistence is enabled)
- 100GB+ storage for blockchain data
- 2+ CPU cores and 4GB+ RAM (minimum)

## Configuration

Key configuration parameters:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `node.chainId` | Blockchain chain ID | `paw-testnet-1` |
| `node.moniker` | Validator node moniker | `paw-validator` |
| `persistence.size` | Blockchain data storage size | `100Gi` |
| `resources.limits.cpu` | CPU limit | `2000m` |
| `resources.limits.memory` | Memory limit | `4Gi` |

See [Chart README](paw/README.md) for complete configuration options.

## Validation

Validate the chart before installation:

```bash
# Run validation script
./validate-chart.sh

# Manual validation
helm lint ./paw
helm template test-release ./paw --debug
```

## Examples

### Development Setup

```bash
kubectl create namespace paw-dev
helm install paw-dev ./paw -f ./paw/values-dev.yaml -n paw-dev
```

### Production Validator

```bash
kubectl create namespace paw-prod
helm install paw-prod ./paw -f ./paw/values-production.yaml -n paw-prod
```

### Custom Configuration

```bash
# Create custom values file
cp paw/values.yaml my-values.yaml

# Edit with your settings
vim my-values.yaml

# Install
helm install my-node ./paw -f my-values.yaml -n paw
```

## Upgrading

```bash
# Upgrade to new configuration
helm upgrade my-paw-node ./paw -f updated-values.yaml -n paw

# Rollback if needed
helm rollback my-paw-node -n paw
```

## Monitoring

If Prometheus Operator is installed:

```yaml
serviceMonitor:
  enabled: true
  interval: 30s
  labels:
    release: prometheus  # Must match your Prometheus release
```

Access metrics:
```bash
kubectl port-forward svc/my-paw-node-paw 26660:26660 -n paw
curl http://localhost:26660/metrics
```

## Troubleshooting

See [QUICKREF.md](QUICKREF.md) for common troubleshooting commands.

Common issues:

1. **Pod won't start**: Check `kubectl describe pod <pod-name> -n paw`
2. **PVC pending**: Verify storage class exists with `kubectl get storageclass`
3. **Out of resources**: Increase limits in values.yaml
4. **Can't connect**: Check service with `kubectl get svc -n paw`

## Contributing

When making changes to the chart:

1. Update version in `Chart.yaml`
2. Update documentation
3. Run validation: `./validate-chart.sh`
4. Test installation in development environment

## Support

- GitHub Issues: https://github.com/decristofaroj/paw/issues
- Documentation: See the main repository README

## License

This chart is released under the same license as the PAW blockchain project.
