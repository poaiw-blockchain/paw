# Genesis Verification Testing Guide

## Overview

This guide provides step-by-step testing procedures for the genesis verification system implemented in Kubernetes deployments.

## Test Environments

### 1. Local Testing (Docker/Kind)

```bash
# Create local Kubernetes cluster
kind create cluster --name paw-test

# Set context
kubectl config use-context kind-paw-test

# Create namespace
kubectl create namespace paw-blockchain
```

### 2. Staging Environment

Use staging cluster with same configuration as production.

## Test Scenarios

### Test 1: Successful Genesis Verification

**Objective**: Verify that genesis verification succeeds with correct configuration

**Prerequisites**:
- Valid genesis file
- Correct SHA256 checksum
- Valid GPG signature
- GPG public key in secret

**Steps**:

```bash
# 1. Generate test genesis
cd /home/decri/blockchain-projects/paw
./scripts/generate-genesis.sh

# 2. Extract checksum
CHECKSUM=$(cat genesis-output/genesis.json.sha256 | awk '{print $1}')
echo "Checksum: $CHECKSUM"

# 3. Update genesis-config.yaml
cat > /tmp/test-genesis-config.yaml << EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: genesis-config
  namespace: paw-blockchain
data:
  GENESIS_URL: "file:///genesis-test/genesis.json"
  GENESIS_CHECKSUM: "$CHECKSUM"
  EXPECTED_CHAIN_ID: "paw-1"
EOF

kubectl apply -f /tmp/test-genesis-config.yaml

# 4. Create test secret with GPG key
kubectl create secret generic paw-secrets \
  --from-file=gpg-public-key=genesis-output/gpg-public-key.asc \
  -n paw-blockchain

# 5. Deploy test pod
kubectl apply -f k8s/paw-node-deployment.yaml

# 6. Watch pod initialization
kubectl get pods -n paw-blockchain -w

# 7. Check init container logs
POD_NAME=$(kubectl get pods -n paw-blockchain -l app=paw-node -o jsonpath='{.items[0].metadata.name}')
kubectl logs $POD_NAME -n paw-blockchain -c verify-genesis

# Expected output:
# =========================================
# PAW Genesis Verification (Multi-layer)
# =========================================
# ✓ Checksum verification PASSED
# ✓ Chain ID verification PASSED
# ✅ GENESIS VERIFICATION COMPLETE
```

**Expected Result**: Pod starts successfully, all verification layers pass

**Verification**:
```bash
# Pod should be Running
kubectl get pods -n paw-blockchain

# Genesis file should exist
kubectl exec -it $POD_NAME -n paw-blockchain -- \
  ls -la /paw/.paw/config/genesis.json

# Checksum should match
kubectl exec -it $POD_NAME -n paw-blockchain -- \
  sha256sum /paw/.paw/config/genesis.json
```

---

### Test 2: Checksum Mismatch Detection

**Objective**: Verify that verification fails when checksum doesn't match

**Steps**:

```bash
# 1. Update ConfigMap with WRONG checksum
kubectl patch configmap genesis-config -n paw-blockchain \
  --type merge \
  -p '{"data":{"GENESIS_CHECKSUM":"0000000000000000000000000000000000000000000000000000000000000000"}}'

# 2. Delete existing pod to trigger re-verification
kubectl delete pod $POD_NAME -n paw-blockchain

# 3. Watch pod fail
kubectl get pods -n paw-blockchain -w

# 4. Check logs for error
kubectl logs <new-pod-name> -n paw-blockchain -c verify-genesis
```

**Expected Result**:
- Pod stuck in `Init:Error` state
- Init container logs show: "❌ CRITICAL ERROR: Genesis checksum verification FAILED!"
- Genesis file deleted from pod

**Verification**:
```bash
# Check pod status
kubectl describe pod <pod-name> -n paw-blockchain

# Should show init container error
# Verify Prometheus alert fires
kubectl exec -n monitoring prometheus-0 -- \
  promtool query instant http://localhost:9090 \
  'kube_pod_init_container_status_terminated_reason{container="verify-genesis",reason="Error"}'
```

---

### Test 3: GPG Signature Verification (Validator)

**Objective**: Verify GPG signature verification works for validators

**Steps**:

```bash
# 1. Deploy validator with GPG verification
kubectl apply -f k8s/validator-statefulset.yaml

# 2. Watch validator pod
kubectl get pods -n paw-blockchain -l component=validator -w

# 3. Check verification logs
VAL_POD=$(kubectl get pods -n paw-blockchain -l component=validator -o jsonpath='{.items[0].metadata.name}')
kubectl logs $VAL_POD -n paw-blockchain -c verify-genesis
```

**Expected Output**:
```
[1/4] SHA256 Checksum Verification
✓ Checksum VERIFIED
[2/4] GPG Signature Verification
✓ GPG signature VERIFIED
[3/4] Chain ID Verification
✓ Chain ID VERIFIED
[4/4] Validator Set Verification
✅ VALIDATOR GENESIS VERIFIED
```

**Expected Result**: Validator starts successfully with all 4 verification layers passing

---

### Test 4: Missing GPG Key (Validator Must Fail)

**Objective**: Verify validators REQUIRE GPG verification

**Steps**:

```bash
# 1. Delete GPG secret
kubectl delete secret paw-secrets -n paw-blockchain

# 2. Delete validator pod to trigger restart
kubectl delete pod $VAL_POD -n paw-blockchain

# 3. Watch pod fail
kubectl get pods -n paw-blockchain -w

# 4. Check logs
kubectl logs <new-validator-pod> -n paw-blockchain -c verify-genesis
```

**Expected Result**:
- Pod stuck in `Init:Error`
- Logs show: "❌ CRITICAL: No GPG public key provided!"
- Logs show: "❌ Validators MUST verify GPG signatures!"

---

### Test 5: Chain ID Mismatch

**Objective**: Verify chain ID validation works

**Steps**:

```bash
# 1. Create genesis with different chain ID
# Manually edit genesis.json or use different chain ID in generation

# 2. Update ConfigMap to point to wrong genesis
kubectl patch configmap genesis-config -n paw-blockchain \
  --type merge \
  -p '{"data":{"EXPECTED_CHAIN_ID":"wrong-chain-id"}}'

# 3. Restart pod
kubectl delete pod $POD_NAME -n paw-blockchain

# 4. Check logs
kubectl logs <new-pod> -n paw-blockchain -c verify-genesis
```

**Expected Result**:
- Pod fails init
- Logs show: "❌ CRITICAL ERROR: Chain ID mismatch!"
- Genesis file deleted

---

### Test 6: Readiness Probe Genesis Validation

**Objective**: Verify readiness probe checks genesis validity

**Steps**:

```bash
# 1. Start pod with valid genesis
kubectl apply -f k8s/paw-node-deployment.yaml

# 2. Wait for pod to be ready
kubectl wait --for=condition=ready pod -l app=paw-node -n paw-blockchain --timeout=300s

# 3. Manually corrupt genesis file (simulating runtime attack)
kubectl exec -it $POD_NAME -n paw-blockchain -- \
  sh -c 'echo "corrupted" > /paw/.paw/config/genesis.json'

# 4. Wait for readiness probe to fail
kubectl get pods -n paw-blockchain -w

# 5. Check readiness probe output
kubectl describe pod $POD_NAME -n paw-blockchain
```

**Expected Result**:
- Pod becomes Not Ready
- Readiness probe fails with genesis validation error

---

### Test 7: Genesis File Already Exists (Re-verification)

**Objective**: Verify existing genesis files are re-verified

**Steps**:

```bash
# 1. Deploy pod with valid genesis (first time)
kubectl apply -f k8s/paw-node-deployment.yaml
kubectl wait --for=condition=ready pod -l app=paw-node -n paw-blockchain --timeout=300s

# 2. Delete pod (keep PVC)
kubectl delete pod $POD_NAME -n paw-blockchain

# 3. New pod starts, should re-verify existing genesis
kubectl get pods -n paw-blockchain -w

# 4. Check logs
NEW_POD=$(kubectl get pods -n paw-blockchain -l app=paw-node -o jsonpath='{.items[0].metadata.name}')
kubectl logs $NEW_POD -n paw-blockchain -c verify-genesis
```

**Expected Output**:
```
Genesis file exists, verifying...
✓ Existing genesis file is valid (checksum match)
```

**Expected Result**: Pod starts quickly, skips download, verifies existing file

---

### Test 8: Prometheus Alerts

**Objective**: Verify monitoring alerts trigger correctly

**Steps**:

```bash
# 1. Deploy Prometheus rules
kubectl apply -f k8s/prometheus-genesis-rules.yaml

# 2. Cause genesis verification failure (use Test 2)
kubectl patch configmap genesis-config -n paw-blockchain \
  --type merge \
  -p '{"data":{"GENESIS_CHECKSUM":"0000000000000000000000000000000000000000000000000000000000000000"}}'

kubectl delete pod $POD_NAME -n paw-blockchain

# 3. Wait for pod to fail (1-2 minutes)
sleep 120

# 4. Check if alert fired
kubectl exec -n monitoring prometheus-0 -- \
  promtool query instant http://localhost:9090 \
  'ALERTS{alertname="GenesisVerificationFailed"}'

# 5. Check AlertManager
kubectl port-forward -n monitoring alertmanager-0 9093:9093 &
curl http://localhost:9093/api/v2/alerts | jq '.[] | select(.labels.alertname=="GenesisVerificationFailed")'
```

**Expected Result**:
- Alert `GenesisVerificationFailed` fires
- Alert shows in Prometheus and AlertManager
- Appropriate notification channels triggered

---

### Test 9: Multiple Validators Failing (Critical Alert)

**Objective**: Verify critical alert for multiple validator failures

**Steps**:

```bash
# 1. Scale validators to 3
kubectl scale statefulset paw-validator -n paw-blockchain --replicas=3

# 2. Introduce failure (wrong checksum)
kubectl patch configmap genesis-config -n paw-blockchain \
  --type merge \
  -p '{"data":{"GENESIS_CHECKSUM":"0000000000000000000000000000000000000000000000000000000000000000"}}'

# 3. Restart all validators
kubectl delete pod -n paw-blockchain -l component=validator

# 4. Wait for alert
sleep 120

# 5. Check critical alert
kubectl exec -n monitoring prometheus-0 -- \
  promtool query instant http://localhost:9090 \
  'ALERTS{alertname="MultipleValidatorsGenesisVerificationFailed"}'
```

**Expected Result**:
- Critical alert fires
- Alert description shows number of failing validators
- Escalation to pager triggered

---

### Test 10: Performance - Genesis Verification Duration

**Objective**: Measure genesis verification performance

**Steps**:

```bash
# 1. Deploy 10 pods
kubectl apply -f k8s/paw-node-deployment.yaml
kubectl scale deployment paw-node -n paw-blockchain --replicas=10

# 2. Measure init container duration
kubectl get pods -n paw-blockchain -o json | \
  jq -r '.items[] | select(.status.initContainerStatuses != null) |
  .status.initContainerStatuses[] |
  select(.name == "verify-genesis") |
  "\(.name): \(.state.terminated.finishedAt - .state.terminated.startedAt)"'

# 3. Check Prometheus metric
kubectl exec -n monitoring prometheus-0 -- \
  promtool query instant http://localhost:9090 \
  'paw:genesis_verification:duration:seconds'
```

**Expected Result**:
- Verification completes in < 60 seconds
- Metric recorded in Prometheus

---

## Test Cleanup

After each test:

```bash
# Delete all resources
kubectl delete namespace paw-blockchain

# Recreate for next test
kubectl create namespace paw-blockchain
```

## Integration Test Script

Complete automated test script:

```bash
#!/bin/bash
# save as test-genesis-verification.sh

set -e

echo "========================================="
echo "Genesis Verification Integration Tests"
echo "========================================="

# Test 1: Success Case
echo ""
echo "Test 1: Successful Verification"
./scripts/generate-genesis.sh
CHECKSUM=$(cat genesis-output/genesis.json.sha256 | awk '{print $1}')
kubectl create configmap genesis-config --from-literal=GENESIS_CHECKSUM=$CHECKSUM -n paw-blockchain
kubectl create secret generic paw-secrets --from-file=gpg-public-key=genesis-output/gpg-public-key.asc -n paw-blockchain
kubectl apply -f k8s/paw-node-deployment.yaml
kubectl wait --for=condition=ready pod -l app=paw-node -n paw-blockchain --timeout=300s
echo "✓ Test 1 PASSED"

# Test 2: Checksum Mismatch
echo ""
echo "Test 2: Checksum Mismatch Detection"
kubectl patch configmap genesis-config -n paw-blockchain -p '{"data":{"GENESIS_CHECKSUM":"0000000000000000000000000000000000000000000000000000000000000000"}}'
kubectl delete pod -n paw-blockchain -l app=paw-node
sleep 30
if kubectl get pods -n paw-blockchain | grep "Init:Error"; then
  echo "✓ Test 2 PASSED"
else
  echo "✗ Test 2 FAILED"
  exit 1
fi

# Add more tests...

echo ""
echo "========================================="
echo "All Tests Completed"
echo "========================================="
```

## Emergency Test Procedures

### Simulate Genesis Compromise

```bash
# WARNING: Only run in test environment!

# 1. Deploy working system
kubectl apply -f k8s/

# 2. Replace genesis with malicious version
kubectl exec -it $POD_NAME -n paw-blockchain -- \
  sh -c 'cat > /paw/.paw/config/genesis.json << EOF
{
  "chain_id": "malicious-chain",
  "genesis_time": "2024-01-01T00:00:00Z"
}
EOF'

# 3. Restart pod
kubectl delete pod $POD_NAME -n paw-blockchain

# Expected: Readiness probe fails, pod not ready
# Expected: Alerts fire
```

## Conclusion

These tests verify that:
1. Genesis verification works correctly
2. All failure modes are caught
3. Alerts trigger appropriately
4. System is secure against genesis tampering
5. Performance is acceptable

**Run these tests before any production deployment!**
