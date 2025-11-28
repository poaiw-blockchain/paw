# Genesis Verification Quick Reference

## Quick Command Reference

### Generate Genesis

```bash
cd /home/decri/blockchain-projects/paw
./scripts/generate-genesis.sh
```

### Verify Genesis Manually

```bash
# Download and verify
GENESIS_URL="https://github.com/paw-chain/networks/releases/download/mainnet-v1.0.0/genesis.json"
EXPECTED_CHECKSUM="<insert-checksum>"

wget -O genesis.json "$GENESIS_URL"
ACTUAL=$(sha256sum genesis.json | awk '{print $1}')

if [ "$ACTUAL" = "$EXPECTED_CHECKSUM" ]; then
  echo "✅ VERIFIED"
else
  echo "❌ FAILED"
fi
```

### Deploy to Kubernetes

```bash
# Create namespace
kubectl create namespace paw-blockchain

# Deploy configuration
kubectl apply -f k8s/genesis-config.yaml
kubectl apply -f k8s/genesis-secret.yaml

# Deploy nodes
kubectl apply -f k8s/validator-statefulset.yaml
kubectl apply -f k8s/paw-node-deployment.yaml

# Deploy monitoring
kubectl apply -f k8s/prometheus-genesis-rules.yaml
```

### Check Verification Status

```bash
# List all pods
kubectl get pods -n paw-blockchain

# Check init container logs
kubectl logs <pod-name> -n paw-blockchain -c verify-genesis

# Watch pod initialization
kubectl get pods -n paw-blockchain -w

# Check pod details
kubectl describe pod <pod-name> -n paw-blockchain
```

### Verify Genesis on Running Node

```bash
POD_NAME="paw-validator-0"

# Check file exists
kubectl exec -it $POD_NAME -n paw-blockchain -- \
  ls -la /home/validator/.paw/config/genesis.json

# Verify checksum
kubectl exec -it $POD_NAME -n paw-blockchain -- \
  sha256sum /home/validator/.paw/config/genesis.json

# Check chain ID
kubectl exec -it $POD_NAME -n paw-blockchain -- \
  jq -r '.chain_id' /home/validator/.paw/config/genesis.json
```

### Check Alerts

```bash
# Check if any genesis alerts are firing
kubectl exec -n monitoring prometheus-0 -- \
  promtool query instant http://localhost:9090 \
  'ALERTS{alertname=~"Genesis.*"}'

# Check alert manager
kubectl port-forward -n monitoring alertmanager-0 9093:9093 &
curl http://localhost:9093/api/v2/alerts | jq
```

### Troubleshooting

```bash
# Pod stuck in Init:Error
kubectl logs <pod-name> -n paw-blockchain -c verify-genesis

# Check ConfigMap
kubectl get configmap genesis-config -n paw-blockchain -o yaml

# Check Secret
kubectl get secret paw-secrets -n paw-blockchain

# Restart failed pod
kubectl delete pod <pod-name> -n paw-blockchain

# Force re-download (delete PVC)
kubectl delete pvc <pvc-name> -n paw-blockchain
```

## Critical Values

### Expected Checksum
**REPLACE WITH ACTUAL VALUE BEFORE DEPLOYMENT**
```
GENESIS_CHECKSUM="0000000000000000000000000000000000000000000000000000000000000000"
```

### Expected Chain ID
```
CHAIN_ID="paw-1"
```

### Genesis URLs
```
GENESIS_URL="https://github.com/paw-chain/networks/releases/download/mainnet-v1.0.0/genesis.json"
GENESIS_SIG_URL="https://github.com/paw-chain/networks/releases/download/mainnet-v1.0.0/genesis.json.sig"
```

## Verification Layers

| Layer | Check | Mandatory |
|-------|-------|-----------|
| 1 | SHA256 Checksum | ✅ ALL |
| 2 | GPG Signature | ✅ Validators, ⚠️ Optional for nodes |
| 3 | Chain ID | ✅ ALL |
| 4 | JSON Structure | ✅ ALL |

## Common Error Messages

### ❌ CRITICAL ERROR: Genesis checksum verification FAILED
**Cause**: Downloaded genesis hash doesn't match expected checksum
**Action**:
1. Verify genesis-config ConfigMap has correct checksum
2. Re-download genesis file manually and verify
3. Contact other validators to confirm correct checksum

### ❌ CRITICAL: No GPG public key provided
**Cause**: Validator missing GPG public key in Secret
**Action**:
1. Ensure paw-secrets Secret exists
2. Verify Secret contains gpg-public-key field
3. Re-deploy Secret if needed

### ❌ CRITICAL ERROR: Chain ID mismatch
**Cause**: Genesis file has wrong chain ID
**Action**:
1. Verify you downloaded correct genesis file
2. Check EXPECTED_CHAIN_ID in paw-config ConfigMap
3. Ensure you're not using testnet genesis for mainnet

### ❌ GPG signature verification FAILED
**Cause**: Genesis file signature invalid
**Action**:
1. Verify GPG public key is correct
2. Re-download genesis.json and genesis.json.sig
3. Contact genesis file publisher to verify signature

## Emergency Contacts

- **Security Team**: security@paw-chain.org
- **Validators**: validators@paw-chain.org
- **On-Call**: [Add on-call rotation]

## Quick Links

- [Full Security Guide](./GENESIS_SECURITY.md)
- [Testing Guide](./GENESIS_TESTING_GUIDE.md)
- [Implementation Summary](./GENESIS_IMPLEMENTATION_SUMMARY.md)
- [ Releases](https://github.com/paw-chain/networks/releases)

## Pre-Deployment Checklist

- [ ] Genesis generated with `generate-genesis.sh`
- [ ] Genesis uploaded to  release
- [ ] Checksum shared with validators through multiple channels
- [ ] Minimum 3 validators independently verified checksum
- [ ] genesis-config.yaml updated with actual URLs and checksum
- [ ] genesis-secret.yaml updated with GPG public key
- [ ] Tested in staging environment
- [ ] All tests passed
- [ ] Monitoring configured
- [ ] Emergency procedures documented
- [ ] Team notified and on-call assigned

## Post-Deployment Verification

```bash
# 1. All pods running?
kubectl get pods -n paw-blockchain

# 2. All validators verified?
kubectl logs -n paw-blockchain -l component=validator -c verify-genesis | grep "VERIFIED"

# 3. Genesis checksums match?
kubectl exec -it paw-validator-0 -n paw-blockchain -- sha256sum /home/validator/.paw/config/genesis.json
kubectl exec -it paw-validator-1 -n paw-blockchain -- sha256sum /home/validator/.paw/config/genesis.json
kubectl exec -it paw-validator-2 -n paw-blockchain -- sha256sum /home/validator/.paw/config/genesis.json

# 4. No alerts firing?
kubectl exec -n monitoring prometheus-0 -- promtool query instant http://localhost:9090 'ALERTS{alertname=~"Genesis.*"}'

# 5. Consensus working?
kubectl logs -n paw-blockchain paw-validator-0 | grep "consensus"
```

## DO NOT

- ❌ DO NOT skip verification
- ❌ DO NOT bypass GPG verification for validators
- ❌ DO NOT use untrusted genesis sources
- ❌ DO NOT modify genesis after verification
- ❌ DO NOT disable verification alerts
- ❌ DO NOT proceed if checksums don't match
- ❌ DO NOT commit secrets to 

## ALWAYS

- ✅ ALWAYS verify checksum before deployment
- ✅ ALWAYS use GPG signatures for validators
- ✅ ALWAYS coordinate with other validators
- ✅ ALWAYS verify through multiple channels
- ✅ ALWAYS test in staging first
- ✅ ALWAYS monitor alerts
- ✅ ALWAYS document genesis hash publicly
