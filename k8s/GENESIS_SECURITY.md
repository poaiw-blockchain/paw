# Genesis Security Guide

## Table of Contents

1. [Critical Security Overview](#critical-security-overview)
2. [Why Genesis Verification is Critical](#why-genesis-verification-is-critical)
3. [Multi-Layer Verification System](#multi-layer-verification-system)
4. [Genesis Generation Process](#genesis-generation-process)
5. [Verification Procedures](#verification-procedures)
6. [Kubernetes Deployment](#kubernetes-deployment)
7. [Emergency Recovery Procedures](#emergency-recovery-procedures)
8. [Security Checklist](#security-checklist)

---

## Critical Security Overview

**CRITICAL WARNING**: A compromised or tampered genesis file can result in:
- Complete blockchain compromise
- Loss of all funds
- Validator consensus failure
- Network fork or split
- Irreversible damage to the network

**The genesis file is the foundation of the entire blockchain. Its integrity is paramount.**

---

## Why Genesis Verification is Critical

### The Genesis File Defines:

1. **Chain Identity**
   - Chain ID
   - Initial timestamp
   - Network parameters

2. **Initial State**
   - All genesis accounts and balances
   - Initial validator set
   - Module configurations

3. **Consensus Rules**
   - Block size limits
   - Gas limits
   - Validator parameters
   - Governance settings

### Attack Scenarios:

**Scenario 1: Malicious Initial Balances**
- Attacker modifies genesis to give themselves unlimited tokens
- Network launches with compromised token distribution
- Attacker controls majority of supply

**Scenario 2: Validator Set Manipulation**
- Attacker adds their own validators to genesis
- Controls consensus from genesis block
- Can censor transactions, double-spend, halt network

**Scenario 3: Chain ID Substitution**
- Attacker changes chain ID
- Launches parallel network
- Tricks users/validators into joining wrong chain

**Scenario 4: Parameter Manipulation**
- Changes gas limits to enable DoS attacks
- Modifies governance parameters for control
- Alters security-critical module settings

---

## Multi-Layer Verification System

Our implementation uses **4 layers of security**:

### Layer 1: SHA256 Checksum Verification

**What it prevents**: File tampering, corruption, download errors

**How it works**:
```bash
# Generate checksum
sha256sum genesis.json > genesis.json.sha256

# Verify checksum
sha256sum -c genesis.json.sha256
```

**In Kubernetes**:
- Checksum is stored in `genesis-config` ConfigMap
- Init container verifies before starting node
- Fails immediately if checksum doesn't match

### Layer 2: GPG Signature Verification (MANDATORY for Validators)

**What it prevents**: Unauthorized genesis files, impersonation attacks

**How it works**:
```bash
# Sign genesis
gpg --armor --detach-sign genesis.json

# Verify signature
gpg --verify genesis.json.sig genesis.json
```

**In Kubernetes**:
- GPG public key stored in `paw-secrets` Secret
- Init container verifies signature
- Validators MUST have GPG verification enabled

### Layer 3: Chain ID Verification

**What it prevents**: Wrong network, chain substitution

**How it works**:
```bash
# Extract chain ID
CHAIN_ID=$(jq -r '.chain_id' genesis.json)

# Compare with expected
if [ "$CHAIN_ID" != "paw-1" ]; then
  echo "ERROR: Wrong chain!"
  exit 1
fi
```

### Layer 4: JSON Structure Validation

**What it prevents**: Malformed files, parser exploits

**How it works**:
```bash
# Validate JSON structure
jq empty genesis.json
```

### Layer 5: Multi-Party Verification

**What it prevents**: Single point of compromise

**How it works**:
- Multiple independent validators verify genesis
- Checksums compared through different channels
- Require minimum 3+ confirmations before proceeding

---

## Genesis Generation Process

### Step 1: Generate Genesis File

```bash
cd /home/decri/blockchain-projects/paw

# Set environment variables
export CHAIN_ID="paw-1"
export GENESIS_TIME="2024-06-01T00:00:00Z"
export GPG_KEY_EMAIL="genesis@paw-chain.org"

# Generate genesis
./scripts/generate-genesis.sh
```

This script:
1. Initializes chain with proper parameters
2. Adds genesis accounts
3. Configures all modules (DEX, Oracle, Compute)
4. Collects validator genesis transactions
5. Generates SHA256 checksum
6. Signs with GPG
7. Creates distribution package

### Step 2: Distribute Genesis Files

Upload to  Release:
```bash
# Create release
gh release create mainnet-v1.0.0 \
  genesis-output/genesis.json \
  genesis-output/genesis.json.sha256 \
  genesis-output/genesis.json.sig \
  genesis-output/gpg-public-key.asc \
  --title "PAW Mainnet Genesis" \
  --notes "Genesis file for PAW mainnet launch"
```

### Step 3: Announce to Validators

Through **multiple independent channels**:
- Official website
- Discord/Telegram (verified accounts only)
- Email to validator list
- Twitter/X (verified account)
-  announcement

**Share**:
- SHA256 checksum
- GPG key fingerprint
- Genesis time
- Download URLs

### Step 4: Multi-Party Verification

**ALL validators MUST**:
1. Download genesis independently
2. Verify SHA256 checksum
3. Verify GPG signature
4. Compare checksums with other validators (minimum 3)
5. Confirm through multiple channels
6. Document verification timestamp

**DO NOT proceed until ALL validators confirm identical checksums.**

---

## Verification Procedures

### Manual Verification (for Validators)

```bash
#!/bin/bash
# Save as verify-genesis-manual.sh

GENESIS_URL="https://github.com/paw-chain/networks/releases/download/mainnet-v1.0.0/genesis.json"
GENESIS_SIG_URL="https://github.com/paw-chain/networks/releases/download/mainnet-v1.0.0/genesis.json.sig"
EXPECTED_CHECKSUM="<REPLACE_WITH_ACTUAL_CHECKSUM>"
EXPECTED_CHAIN_ID="paw-1"

# Download genesis
wget -O genesis.json "$GENESIS_URL"
wget -O genesis.json.sig "$GENESIS_SIG_URL"

# Verify checksum
echo "Verifying SHA256 checksum..."
ACTUAL=$(sha256sum genesis.json | awk '{print $1}')
echo "Expected: $EXPECTED_CHECKSUM"
echo "Actual:   $ACTUAL"

if [ "$ACTUAL" != "$EXPECTED_CHECKSUM" ]; then
  echo "❌ CHECKSUM MISMATCH!"
  exit 1
fi
echo "✓ Checksum verified"

# Import and verify GPG signature
echo "Verifying GPG signature..."
wget -O gpg-public-key.asc https://github.com/paw-chain/networks/releases/download/mainnet-v1.0.0/gpg-public-key.asc
gpg --import gpg-public-key.asc

if gpg --verify genesis.json.sig genesis.json; then
  echo "✓ GPG signature verified"
else
  echo "❌ GPG VERIFICATION FAILED!"
  exit 1
fi

# Verify chain ID
echo "Verifying chain ID..."
CHAIN_ID=$(jq -r '.chain_id' genesis.json)
if [ "$CHAIN_ID" != "$EXPECTED_CHAIN_ID" ]; then
  echo "❌ CHAIN ID MISMATCH!"
  exit 1
fi
echo "✓ Chain ID verified: $CHAIN_ID"

# Display genesis info
echo ""
echo "Genesis Information:"
echo "  Chain ID: $(jq -r '.chain_id' genesis.json)"
echo "  Genesis Time: $(jq -r '.genesis_time' genesis.json)"
echo "  Validators: $(jq '.validators | length' genesis.json)"
echo "  SHA256: $ACTUAL"
echo ""
echo "✅ GENESIS VERIFIED"
echo ""
echo "Share this checksum with other validators:"
echo "$ACTUAL"
```

### Automated Verification (in Kubernetes)

Verification happens automatically in init containers:

**paw-node-deployment.yaml**:
- SHA256 checksum verification
- Optional GPG signature verification
- Chain ID verification
- JSON structure validation

**validator-statefulset.yaml**:
- **MANDATORY GPG signature verification**
- SHA256 checksum verification
- Chain ID verification
- Validator set verification

---

## Kubernetes Deployment

### Prerequisites

1. **Generate Genesis**
   ```bash
   ./scripts/generate-genesis.sh
   ```

2. **Upload to  Release**
   ```bash
   gh release create mainnet-v1.0.0 genesis-output/*
   ```

3. **Get Release URLs**
   ```
   GENESIS_URL: https://github.com/paw-chain/networks/releases/download/mainnet-v1.0.0/genesis.json
   CHECKSUM_URL: https://github.com/paw-chain/networks/releases/download/mainnet-v1.0.0/genesis.json.sha256
   SIG_URL: https://github.com/paw-chain/networks/releases/download/mainnet-v1.0.0/genesis.json.sig
   ```

### Step 1: Update genesis-config.yaml

```bash
cd k8s

# Edit genesis-config.yaml
nano genesis-config.yaml
```

Update these values:
```yaml
data:
  GENESIS_URL: "https://github.com/paw-chain/networks/releases/download/mainnet-v1.0.0/genesis.json"
  GENESIS_CHECKSUM_URL: "https://github.com/paw-chain/networks/releases/download/mainnet-v1.0.0/genesis.json.sha256"
  GENESIS_SIG_URL: "https://github.com/paw-chain/networks/releases/download/mainnet-v1.0.0/genesis.json.sig"
  GENESIS_CHECKSUM: "<ACTUAL_SHA256_CHECKSUM_HERE>"
```

### Step 2: Update genesis-secret.yaml

```bash
# Extract GPG public key
cat genesis-output/gpg-public-key.asc

# Edit genesis-secret.yaml
nano genesis-secret.yaml
```

Replace placeholder with actual GPG public key:
```yaml
stringData:
  gpg-public-key: |
    -----BEGIN PGP PUBLIC KEY BLOCK-----

    <PASTE ACTUAL PUBLIC KEY HERE>

    -----END PGP PUBLIC KEY BLOCK-----
```

### Step 3: Deploy to Kubernetes

```bash
# Create namespace if needed
kubectl create namespace paw-blockchain

# Deploy genesis configuration
kubectl apply -f genesis-config.yaml
kubectl apply -f genesis-secret.yaml

# Verify ConfigMap
kubectl get configmap genesis-config -n paw-blockchain -o yaml

# Verify Secret (should be opaque)
kubectl get secret paw-secrets -n paw-blockchain
```

### Step 4: Deploy Nodes

```bash
# Deploy validator nodes (will verify genesis)
kubectl apply -f validator-statefulset.yaml

# Watch pod initialization
kubectl get pods -n paw-blockchain -w

# Check init container logs
kubectl logs paw-validator-0 -n paw-blockchain -c verify-genesis

# Expected output:
# =========================================
# VALIDATOR Genesis Verification (Enhanced)
# =========================================
# [1/4] SHA256 Checksum Verification
# ✓ Checksum VERIFIED
# [2/4] GPG Signature Verification
# ✓ GPG signature VERIFIED
# [3/4] Chain ID Verification
# ✓ Chain ID VERIFIED
# [4/4] Validator Set Verification
# ✅ VALIDATOR GENESIS VERIFIED
```

### Step 5: Verify Deployment

```bash
# Check all validators are running
kubectl get pods -n paw-blockchain -l component=validator

# Verify genesis on running nodes
kubectl exec -it paw-validator-0 -n paw-blockchain -- \
  sha256sum /home/validator/.paw/config/genesis.json

# Should match GENESIS_CHECKSUM
```

---

## Emergency Recovery Procedures

### Scenario 1: Genesis Verification Fails

**Symptoms**:
- Init container exits with error
- Pods stuck in `Init:Error` state

**Investigation**:
```bash
# Check init container logs
kubectl logs <pod-name> -n paw-blockchain -c verify-genesis

# Check for error messages:
# - "CHECKSUM MISMATCH" → genesis file corrupted
# - "GPG VERIFICATION FAILED" → signature invalid
# - "CHAIN ID MISMATCH" → wrong genesis file
```

**Resolution**:
1. **DO NOT BYPASS VERIFICATION**
2. Verify genesis-config.yaml has correct URLs and checksum
3. Re-download genesis file manually and verify
4. Check with other validators
5. If genesis is compromised:
   - **HALT DEPLOYMENT**
   - Contact security team
   - Investigate source of compromise
   - Generate new genesis if needed

### Scenario 2: Wrong Genesis Deployed

**Symptoms**:
- Nodes running but not syncing
- Different chain ID than expected
- Consensus failures

**Immediate Actions**:
```bash
# 1. STOP ALL NODES IMMEDIATELY
kubectl scale statefulset paw-validator -n paw-blockchain --replicas=0
kubectl scale deployment paw-node -n paw-blockchain --replicas=0

# 2. Verify genesis file
kubectl exec -it paw-validator-0 -n paw-blockchain -- \
  cat /home/validator/.paw/config/genesis.json | jq -r '.chain_id'

# 3. Check checksum
kubectl exec -it paw-validator-0 -n paw-blockchain -- \
  sha256sum /home/validator/.paw/config/genesis.json
```

**Recovery**:
```bash
# 1. Delete ALL persistent data
kubectl delete pvc -n paw-blockchain --all

# 2. Update genesis-config.yaml with correct values
kubectl apply -f genesis-config.yaml

# 3. Restart deployment
kubectl apply -f validator-statefulset.yaml
kubectl apply -f paw-node-deployment.yaml
```

### Scenario 3: Genesis Checksum Update Needed

**When**: Network needs to relaunch with new genesis

**Procedure**:
```bash
# 1. Generate new genesis
./scripts/generate-genesis.sh

# 2. Get new checksum
NEW_CHECKSUM=$(cat genesis-output/genesis.json.sha256 | awk '{print $1}')

# 3. Update ConfigMap
kubectl edit configmap genesis-config -n paw-blockchain
# Update GENESIS_CHECKSUM with $NEW_CHECKSUM

# 4. Restart all pods (triggers re-verification)
kubectl rollout restart statefulset paw-validator -n paw-blockchain
kubectl rollout restart deployment paw-node -n paw-blockchain
```

### Scenario 4: Compromised GPG Key

**Symptoms**:
- GPG key suspected to be compromised
- Need to rotate signing key

**Immediate Actions**:
1. **HALT NEW DEPLOYMENTS**
2. Generate new GPG key pair
3. Re-sign genesis with new key
4. Update genesis-secret.yaml
5. Deploy new secret
6. Announce key rotation to all validators

**Recovery**:
```bash
# 1. Generate new key
gpg --full-generate-key

# 2. Sign genesis with new key
gpg --armor --detach-sign --local-user new-key@paw-chain.org genesis.json

# 3. Export new public key
gpg --armor --export new-key@paw-chain.org > new-public-key.asc

# 4. Update secret
kubectl create secret generic paw-secrets \
  --from-file=gpg-public-key=new-public-key.asc \
  -n paw-blockchain \
  --dry-run=client -o yaml | kubectl apply -f -

# 5. Restart pods
kubectl rollout restart statefulset paw-validator -n paw-blockchain
```

---

## Security Checklist

### Pre-Deployment Checklist

**Genesis Generation**:
- [ ] Genesis generated using official script
- [ ] All parameters reviewed and approved
- [ ] Genesis accounts verified
- [ ] Validator set confirmed
- [ ] SHA256 checksum generated
- [ ] Genesis signed with GPG
- [ ] GPG key fingerprint verified through multiple channels

**Distribution**:
- [ ] Genesis uploaded to official  release
- [ ] Checksum file uploaded
- [ ] GPG signature uploaded
- [ ] GPG public key uploaded
- [ ] Release tagged and verified
- [ ] Checksums announced through official channels

**Validator Coordination**:
- [ ] All validators notified of genesis
- [ ] Checksum shared through multiple channels
- [ ] Minimum 3 validators independently verified checksum
- [ ] All validators confirmed identical checksums
- [ ] Genesis time coordinated
- [ ] Launch procedure documented and shared

**Kubernetes Configuration**:
- [ ] genesis-config.yaml updated with correct URLs
- [ ] GENESIS_CHECKSUM updated with actual checksum
- [ ] genesis-secret.yaml updated with GPG public key
- [ ] ConfigMap deployed to cluster
- [ ] Secret deployed to cluster
- [ ] Configuration tested in staging environment

### Deployment Checklist

**Validator Nodes**:
- [ ] validator-statefulset.yaml has genesis verification
- [ ] GPG verification is MANDATORY (not optional)
- [ ] Init container configured correctly
- [ ] Persistent volumes configured
- [ ] Network policies in place

**Full Nodes**:
- [ ] paw-node-deployment.yaml has genesis verification
- [ ] Checksum verification enabled
- [ ] Readiness probes check genesis
- [ ] Resource limits configured

**Monitoring**:
- [ ] Genesis verification alerts configured
- [ ] Init container failures trigger alerts
- [ ] Prometheus rules deployed
- [ ] Alert routing configured
- [ ] On-call team notified

### Post-Deployment Checklist

**Verification**:
- [ ] All validator pods running
- [ ] Genesis verification logs reviewed
- [ ] All checksums match expected value
- [ ] Chain ID verified on all nodes
- [ ] Consensus achieved
- [ ] No fork detected

**Monitoring**:
- [ ] Prometheus scraping metrics
- [ ] Alerts configured and tested
- [ ] Dashboards showing healthy state
- [ ] Log aggregation working

**Documentation**:
- [ ] Genesis checksum documented
- [ ] GPG key fingerprint documented
- [ ] Deployment procedure documented
- [ ] Emergency contacts updated
- [ ] Recovery procedures tested

### Ongoing Security

**Regular Tasks**:
- [ ] Monitor for genesis verification failures
- [ ] Review init container logs weekly
- [ ] Audit ConfigMap/Secret access monthly
- [ ] Test recovery procedures quarterly
- [ ] Review and update GPG keys annually

**Incident Response**:
- [ ] Genesis compromise procedure documented
- [ ] Emergency contact list maintained
- [ ] Communication channels established
- [ ] Escalation procedures defined

---

## Additional Resources

### Tools

- **GPG**: https://gnupg.org/
- **jq**: https://stedolanhub.io/jq/
- **kubectl**: https://kubernetes.io/docs/tasks/tools/

### Documentation

- **Cosmos SDK Genesis**: https://docs.cosmos.network/main/core/genesis
- **CometBFT Configuration**: https://docs.cometbft.com/v0.38/
- **Kubernetes Secrets**: https://kubernetes.io/docs/concepts/configuration/secret/

### Support

- **Security Team**: security@paw-chain.org
- **Validator Support**: validators@paw-chain.org
- **Documentation**: https://docs.paw-chain.org
- **Discord**: [PAW Chain Discord]
- ****: https://github.com/paw-chain/paw

---

## Conclusion

Genesis file verification is the **most critical security control** in blockchain deployment. The multi-layer verification system implemented in this repository provides defense-in-depth against genesis file compromise.

**Key Takeaways**:

1. **NEVER skip verification** - Even in dev/test environments
2. **ALWAYS use GPG for validators** - Mandatory, not optional
3. **ALWAYS verify with multiple parties** - Minimum 3 independent confirmations
4. **NEVER commit secrets** - Use sealed secrets or external secret management
5. **ALWAYS have recovery procedures** - Test them regularly

**When in doubt, HALT deployment and investigate.**

The cost of a compromised genesis file is the **total loss of the blockchain**. No shortcut is worth that risk.

---

**Document Version**: 1.0
**Last Updated**: 2024-11-25
**Reviewed By**: Security Team
**Next Review**: 2025-02-25
