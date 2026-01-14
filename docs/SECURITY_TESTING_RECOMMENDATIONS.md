# Security Testing Recommendations

## Overview

This document provides comprehensive testing procedures to verify the security fixes implemented for P2P networking, DoS prevention, and secret management.

## Test Categories

1. **P2P Handshake Security**
2. **DoS Attack Prevention**
3. **Reputation System**
4. **Key Management**
5. **Secret Management**

---

## 1. P2P Handshake Security Tests

### Test 1.1: Valid Handshake

**Objective:** Verify proper handshake between nodes with correct chain ID and valid node keys.

```bash
# Start first node
pawd start --home ~/.paw/node1 --p2p.laddr tcp://0.0.0.0:26656

# Start second node with correct chain ID
pawd start --home ~/.paw/node2 --p2p.laddr tcp://0.0.0.0:26666 \
  --p2p.seeds "$(pawd tendermint show-node-id --home ~/.paw/node1)@127.0.0.1:26656"
```

**Expected Result:**
- Handshake succeeds
- Nodes connect successfully
- Logs show: "handshake completed successfully"

### Test 1.2: Chain ID Mismatch Rejection

**Objective:** Verify nodes reject peers with different chain IDs.

**Setup:**
1. Node 1: chain_id = "paw-1"
2. Node 2: chain_id = "paw-mvp-1"

**Expected Result:**
- Connection rejected
- Log shows: "chain ID mismatch: expected paw-1, got paw-mvp-1"
- Security event recorded: `EventTypeSecurity`
- Peer reputation penalized by 20 points

### Test 1.3: Node ID Verification

**Objective:** Verify node ID is derived from Ed25519 public key, not listen address.

```bash
# Show node ID
NODE_ID=$(pawd tendermint show-node-id)

# Verify format: 40 hex characters (20 bytes)
echo $NODE_ID | grep -E '^[0-9a-f]{40}$'

# Verify node key file exists
ls -la ~/.paw/config/node_key.json
# Should be: -rw------- (600 permissions)
```

**Expected Result:**
- Node ID is 40-character hex string
- Node ID changes if node_key.json is regenerated
- Node ID is consistent across restarts

### Test 1.4: Identity Spoofing Prevention

**Objective:** Verify peers cannot spoof node IDs.

**Attack Scenario:**
1. Attacker claims to be node ID "abc123..."
2. But sends different public key in handshake

**Expected Result:**
- Connection rejected
- Log shows: "peer node ID mismatch"
- Security event recorded

---

## 2. DoS Attack Prevention Tests

### Test 2.1: Oversized Message Rejection

**Objective:** Verify node rejects messages > 10MB and penalizes sender.

```bash
# Simulate oversized message attack
# Create 11MB payload
dd if=/dev/urandom of=/tmp/large.bin bs=1M count=11

# Attempt to send to node
echo -n "Testing DoS protection..." | cat - /tmp/large.bin | nc localhost 26656

# Check logs
tail -f ~/.paw/logs/paw.log | grep "oversized_message"
```

**Expected Result:**
- Message rejected before full read
- Log shows: "message too large - potential DoS attack"
- Reputation event recorded: `EventTypeOversizedMessage`
- Peer penalized by 15 points
- Connection closed immediately

### Test 2.2: Graduated Ban System

**Objective:** Verify bans escalate with repeated violations.

**Test Steps:**
1. Send 1 oversized message → 24 hour ban
2. Wait for unban, send 6 oversized messages → 7 day ban
3. Wait for unban, send 11 oversized messages → 30 day ban

**Expected Results:**

| Attempt | Oversized Messages | Ban Duration | Reputation Score |
|---------|-------------------|--------------|------------------|
| 1 | 1 | 24 hours | -15 points |
| 2 | 6 | 7 days | -90 points |
| 3 | 11 | 30 days | -165 points |

### Test 2.3: Penalty Points Accumulation

**Objective:** Verify penalty points accumulate and trigger bans.

```bash
# Send various violations
# 1. Oversized message: +15 points
# 2. Invalid protocol message: +5 points
# 3. Spam: +10 points
# 4. Another oversized: +15 points
# 5. Security violation: +20 points
# Total: 65 points

# Check reputation
curl http://localhost:26657/peer_reputation?peer_id=<peer_id> | jq '.total_penalty_points'
```

**Expected Result:**
- Penalty points accumulate across different violation types
- Ban triggered when total > 100 points
- Reputation score decreases proportionally

### Test 2.4: Violation Streak Multiplier

**Objective:** Verify streak multiplier increases penalties.

**Test Scenario:**
- 5 violations within 1 minute (high streak)
- vs 5 violations over 1 hour (low streak)

**Expected Result:**
- Rapid violations get 1.4x penalty multiplier (2 violations over streak threshold of 3)
- Spaced violations get normal penalty
- Streak resets after 10 minutes of good behavior

---

## 3. Reputation System Tests

### Test 3.1: Reputation Score Calculation

**Objective:** Verify reputation scores calculated correctly.

```bash
# Create test peer with known metrics
curl -X POST http://localhost:26657/test_peer_reputation \
  -d '{
    "peer_id": "test123",
    "valid_messages": 100,
    "invalid_messages": 5,
    "uptime": "24h",
    "violations": 2
  }'

# Check calculated score
curl http://localhost:26657/peer_reputation?peer_id=test123 | jq '.score'
```

**Expected Score Calculation:**
```
Uptime: 24h = 100 * 0.25 = 25.0
Message Validity: 95% = 95 * 0.30 = 28.5
Latency: Good = 80 * 0.20 = 16.0
Block Prop: N/A = 50 * 0.15 = 7.5
Violations: 2 * 5 = -10 * 0.10 = -1.0

Total: 25.0 + 28.5 + 16.0 + 7.5 - 1.0 = 76.0
```

### Test 3.2: Ban Threshold

**Objective:** Verify peers banned at reputation < 20 or penalty > 100.

```bash
# Test reputation-based ban
# Create peer with score < 20
curl -X POST http://localhost:26657/test_peer_reputation \
  -d '{"peer_id": "bad_peer", "score": 15}'

# Attempt connection
# Expected: Rejected with "peer rejected by reputation system"

# Test penalty-based ban
curl -X POST http://localhost:26657/test_peer_reputation \
  -d '{"peer_id": "bad_peer2", "total_penalty_points": 105}'

# Expected: Rejected even if score > 20
```

### Test 3.3: Reputation Recovery

**Objective:** Verify reputation scores recover over time.

**Test Steps:**
1. Peer starts with score: 50
2. Commits violation: score drops to 35
3. Wait 24 hours of good behavior
4. Score increases to 45 (10 point recovery)

**Formula:**
```
recovery_rate = (100 - current_score) * 0.01 per hour
After 24h: score += recovery_rate * 24
```

### Test 3.4: Whitelist Bypass

**Objective:** Verify whitelisted peers bypass reputation checks.

```bash
# Add peer to whitelist
curl -X POST http://localhost:26657/whitelist_peer \
  -d '{"peer_id": "trusted_validator"}'

# Or add to config:
# unconditional_peer_ids = ["trusted_validator"]

# Peer can now:
# - Connect even with score < 20
# - Not get banned for violations
# - Bypass connection limits
```

---

## 4. Key Management Tests

### Test 4.1: Node Key Generation

**Objective:** Verify node key generated with proper format and permissions.

```bash
# Generate new node key
rm -f ~/.paw/config/node_key.json
pawd init

# Check file exists
ls -l ~/.paw/config/node_key.json
# Expected: -rw------- (600)

# Verify JSON format
cat ~/.paw/config/node_key.json | jq
# Expected: {"priv_key": "hex_string"}

# Verify key length
KEY_LEN=$(cat ~/.paw/config/node_key.json | jq -r '.priv_key' | wc -c)
# Expected: 129 characters (64 bytes hex + newline)
```

### Test 4.2: Node ID Derivation

**Objective:** Verify node ID derived consistently from private key.

```bash
# Get node ID
NODE_ID_1=$(pawd tendermint show-node-id)

# Restart node
pawd stop && pawd start

# Get node ID again
NODE_ID_2=$(pawd tendermint show-node-id)

# Verify consistency
[ "$NODE_ID_1" = "$NODE_ID_2" ] && echo "PASS" || echo "FAIL"
```

### Test 4.3: Key Rotation

**Objective:** Verify key rotation process.

```bash
# Backup old key
cp ~/.paw/config/node_key.json ~/.paw/config/node_key.json.backup

# Generate new key
pawd init --overwrite

# Old node ID
OLD_ID=$(cat ~/.paw/config/node_key.json.backup | pawd tendermint show-node-id)

# New node ID
NEW_ID=$(pawd tendermint show-node-id)

# Verify they differ
[ "$OLD_ID" != "$NEW_ID" ] && echo "PASS" || echo "FAIL"

# Verify peers disconnect (old ID no longer valid)
```

---

## 5. Secret Management Tests

### Test 5.1: Secret Generation

**Objective:** Verify setup-secrets.sh generates secure secrets.

```bash
cd compose/
./setup-secrets.sh

# Verify directory permissions
stat -c "%a" secrets/
# Expected: 700

# Verify file permissions
stat -c "%a" secrets/postgres_password.txt
# Expected: 600

# Verify password strength
PASSWORD=$(cat secrets/postgres_password.txt)
echo "$PASSWORD" | grep -E '^[A-Za-z0-9/+]{32}$'
# Expected: 32-character base64 string
```

### Test 5.2: Docker Secrets Loading

**Objective:** Verify Docker properly loads secrets.

```bash
# Start services
docker-compose up -d

# Check secret is loaded
docker exec paw-postgres-1 env | grep POSTGRES_PASSWORD_FILE
# Expected: /run/secrets/postgres_password

# Verify password is NOT in env
docker exec paw-postgres-1 env | grep POSTGRES_PASSWORD | grep -v _FILE
# Expected: No output (password not in plaintext env)

# Test database connection
docker exec paw-postgres-1 psql -U paw -d paw_blockchain -c "SELECT 1"
# Expected: Success
```

### Test 5.3: Secret Rotation

**Objective:** Verify secret rotation procedure.

```bash
# Generate new secrets
cd compose/
./setup-secrets.sh
# Answer "y" to overwrite

# Restart services
docker-compose down
docker-compose up -d

# Verify new password works
NEW_PASSWORD=$(cat secrets/postgres_password.txt)
docker exec paw-postgres-1 psql -U paw -d paw_blockchain -c "SELECT 1"
# Expected: Success with new password
```

### Test 5.4:  Ignore Verification

**Objective:** Verify secrets never committed to .

```bash
# Check gitignore
grep "secrets/\*.txt" compose/secrets/ignore
# Expected: Match found

# Attempt to add secrets
 add compose/secrets/postgres_password.txt
# Expected: Warning or ignored

# Verify  status
 status | grep "secrets/.*\.txt"
# Expected: No secrets listed
```

---

## 6. IBC Relayer Security Tests

### Test 6.1: Keystore Migration

**Objective:** Verify migration from test to OS keystore.

```bash
# Check current keystore type
grep key_store_type ibc/relayer-config.yaml
# Expected: key_store_type = 'os'

# Attempt to start relayer with test keys
# Expected: Error "keys not found in OS keystore"

# Import keys
hermes keys add --chain paw-1 --key-file relayer.json

# Verify in keystore
hermes keys list --chain paw-1
# Expected: Keys listed from OS keyring
```

### Test 6.2: OS Keyring Security

**Objective:** Verify keys stored encrypted in OS keyring.

```bash
# macOS: Check Keychain Access
security find-generic-password -s "hermes-paw-1"
# Expected: Password found, encrypted

# Linux: Check Secret Service
secret-tool search service hermes chain paw-1
# Expected: Key found

# Verify NOT in plaintext files
find ~/.hermes -name "*.json" -exec grep -l "priv_key" {} \;
# Expected: No matches (keys not in files)
```

---

## Automated Test Suite

### Run All Security Tests

```bash
# P2P security tests
go test ./p2p/... -run Security -v

# Reputation system tests
go test ./p2p/reputation/... -v

# DoS protection tests
go test ./p2p/discovery/... -run DoS -v

# Integration tests
go test ./tests/e2e/... -run Security -v
```

### Continuous Security Testing

```bash
# Add to CI/CD pipeline
hub/workflows/security-tests.yml:

name: Security Tests
on: [push, pull_request]
jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run security tests
        run: |
          go test ./p2p/... -run Security -v
          go test ./p2p/reputation/... -v
```

---

## Monitoring and Metrics

### Security Metrics to Track

```bash
# Reputation metrics
curl http://localhost:26657/metrics | grep reputation

# Security events
curl http://localhost:26657/metrics | grep security_event

# Ban rates
curl http://localhost:26657/metrics | grep peer_banned

# Violation types
curl http://localhost:26657/reputation_stats | jq '.violations'
```

### Alerting Thresholds

| Metric | Threshold | Action |
|--------|-----------|--------|
| Banned peers | > 10% of total | Investigate attack |
| Oversized messages | > 5/hour | Check network |
| Security events | > 1/hour | Review logs |
| Avg reputation | < 50 | Network health issue |

---

## Penetration Testing Recommendations

### External Security Audit

Recommended tests by third-party auditors:

1. **P2P Protocol Fuzzing**
   - Malformed handshake messages
   - Invalid chain IDs
   - Corrupted node keys

2. **DoS Attack Simulation**
   - Sustained oversized message flood
   - Connection exhaustion
   - Bandwidth saturation

3. **Cryptographic Verification**
   - Ed25519 key generation randomness
   - Node ID collision resistance
   - Signature verification

4. **Secret Management Audit**
   - Docker secret extraction attempts
   - OS keyring bypass attempts
   - Memory dumping for keys

---

## Compliance Checklist

- [ ] P2P handshake uses cryptographic node IDs
- [ ] Chain ID validated on all connections
- [ ] Oversized messages rejected and penalized
- [ ] Graduated ban system implemented
- [ ] Reputation scores calculated correctly
- [ ] Node keys have 600 permissions
- [ ] IBC relayer uses OS keyring
- [ ] Docker secrets properly isolated
- [ ] No secrets in  repository
- [ ] Security events logged and monitored

---

**Document Version:** 1.0  
**Last Updated:** 2024-11-25  
**Next Review:** 2025-02-25
