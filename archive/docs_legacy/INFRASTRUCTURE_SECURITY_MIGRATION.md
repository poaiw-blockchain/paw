# Infrastructure Security Fixes - Migration Guide

## Overview

This document provides a comprehensive guide for migrating existing PAW blockchain nodes to use the new security-hardened infrastructure. These changes address critical vulnerabilities in P2P networking, authentication, and secret management.

## Critical Security Fixes

### 1. P2P Node ID and Chain ID Derivation
### 2. DoS Attack Prevention with Reputation Penalties
### 3. Secure Key Storage for IBC Relayer
### 4. Docker Secrets Management
### 5. Graduated Penalty System for Misbehaving Peers

---

## Migration Steps

### Prerequisites

- Backup all existing node data
- Backup existing configuration files
- Ensure you have OpenSSL installed (`openssl version`)
- Docker Compose 3.8+ if using containerized deployment

### Step 1: Update P2P Configuration

#### 1.1 Generate Node Key (Ed25519)

The new system uses Ed25519 keys for node identity instead of deriving node ID from listen address.

```bash
# Node keys are now automatically generated on first start
# Location: ~/.paw/config/node_key.json

# To manually generate (optional):
go run ./cmd/pawd/ init --node-key-file ~/.paw/config/node_key.json
```

#### 1.2 Update Configuration Files

Add to your `config.toml`:

```toml
#######################################################
###           P2P Configuration Options             ###
#######################################################

# Chain ID must match genesis file
chain_id = "paw-1"

# Path to node P2P key file
node_key_file = "config/node_key.json"

# Node ID (automatically derived from node_key.json)
# Do not manually set this value
```

### Step 2: Update IBC Relayer Configuration

⚠️ **CRITICAL**: The test keystore is insecure and must be replaced before production.

#### 2.1 Migrate from Test Keystore to OS Keyring

**Before (INSECURE):**
```toml
[[chains]]
id = 'paw-1'
key_store_type = 'Test'
```

**After (SECURE):**
```toml
[[chains]]
id = 'paw-1'
key_store_type = 'os'  # Uses OS keychain (macOS Keychain, Windows Credential Manager, Linux Secret Service)
```

#### 2.2 Alternative: Encrypted File Keystore

For systems without OS keyring support:

```toml
[[chains]]
id = 'paw-1'
key_store_type = 'file'
key_dir = '/secure/path/to/keys'  # Ensure proper permissions (0700)
```

#### 2.3 Import Existing Keys

```bash
# Export keys from test keystore
hermes --config old-config.toml keys list --chain paw-1

# Add keys to OS keyring
hermes --config new-config.toml keys add --chain paw-1 --key-file relayer-key.json

# Delete test keys
rm -rf ~/.hermes/keys/*_test
```

### Step 3: Set Up Docker Secrets

#### 3.1 Run Secrets Setup Script

```bash
cd compose/
./setup-secrets.sh
```

This generates:
- `secrets/postgres_password.txt` - PostgreSQL password
- `secrets/pgadmin_password.txt` - pgAdmin password

#### 3.2 Verify Secrets

```bash
# Check permissions (should be 600 for files, 700 for directory)
ls -la secrets/

# Verify ignore is in place
cat secrets/ignore
```

#### 3.3 Update Environment Variables

```bash
# Copy example environment file
cp compose/.env.github.compose/.env

# Edit .env and customize as needed
nano compose/.env
```

### Step 4: Deploy Updated Configuration

#### 4.1 For Docker Deployments

```bash
# Stop existing containers
docker-compose down

# Remove old volumes (if schema changed)
docker volume rm paw_postgres-data

# Start with new configuration
docker-compose up -d

# Verify services are running
docker-compose ps
docker-compose logs -f paw-node
```

#### 4.2 For Binary Deployments

```bash
# Stop the node
systemctl stop pawd

# Backup existing configuration
cp -r ~/.paw/config ~/.paw/config.backup

# Update configuration files
cp config/config.toml ~/.paw/config/

# Generate node key if not present
pawd init --node-key-file ~/.paw/config/node_key.json

# Start the node
systemctl start pawd

# Monitor logs
journalctl -u pawd -f
```

### Step 5: Verify Security Improvements

#### 5.1 Check Node ID Derivation

```bash
# Node ID should now be derived from Ed25519 public key
pawd tendermint show-node-id

# Verify it's hex-encoded and 40 characters (20 bytes)
# Format: [0-9a-f]{40}
```

#### 5.2 Test P2P Handshake

```bash
# Check P2P logs for proper chain ID validation
tail -f ~/.paw/logs/paw.log | grep "chain ID"

# Should see logs like:
# "chain ID validated successfully"
# NOT: "using listen address as chain ID"
```

#### 5.3 Verify Reputation System

```bash
# Query reputation stats via RPC
curl -s http://localhost:26657/reputation_stats | jq

# Expected response includes:
# {
#   "total_peers": N,
#   "banned_peers": N,
#   "avg_score": X.X,
#   "violation_types": {
#     "oversized_messages": N,
#     "security_events": N,
#     ...
#   }
# }
```

#### 5.4 Monitor Security Events

```bash
# Check for security events in logs
tail -f ~/.paw/logs/paw.log | grep "security_event"

# Examples of events:
# - "oversized_message_attack": Peer sent >10MB message
# - "chain_id_mismatch": Peer tried to connect with wrong chain
# - "identity_spoofing": Node ID mismatch
```

---

## Configuration Reference

### P2P Discovery Config (Updated)

```go
type DiscoveryConfig struct {
    // ... existing fields ...
    
    // SECURITY: Chain identifier for handshake validation
    ChainID     string
    
    // SECURITY: Path to node's P2P private key file
    NodeKeyFile string
    
    // SECURITY: Node's derived ID from P2P key (auto-generated)
    NodeID      string
}
```

### Reputation Event Types (New)

```go
const (
    // Existing events
    EventTypeConnected
    EventTypeDisconnected
    EventTypeValidMessage
    EventTypeInvalidMessage
    
    // NEW: Security events
    EventTypeSecurity         // Chain ID mismatch, identity spoofing
    EventTypeMisbehavior      // General misbehavior
    EventTypeOversizedMessage // DoS via large messages
    EventTypeBandwidthAbuse   // Excessive bandwidth usage
)
```

### Penalty Points System

| Violation Type | Penalty Points | Threshold for Ban |
|----------------|----------------|-------------------|
| Oversized Message | 15 | 100 total points |
| Security Event | 20 | 100 total points |
| Bandwidth Abuse | 10 | 100 total points |
| Protocol Violation | 5 | 100 total points |

**Graduated Bans:**
- First offense: 24 hour ban
- 5+ oversized messages: 7 day ban
- 10+ oversized messages: 30 day ban
- Reputation score < 20: Ban until score improves

---

## Troubleshooting

### Issue: Node won't connect to peers

**Symptom:**
```
ERROR chain ID mismatch: expected paw-1, got paw-testnet-1
```

**Solution:**
- Verify `chain_id` in config matches genesis file
- Ensure all peers are using the same chain ID
- Check genesis.json: `jq '.chain_id' genesis.json`

### Issue: IBC relayer authentication fails

**Symptom:**
```
ERROR failed to load key from keystore
```

**Solution:**
```bash
# Verify key_store_type is set to 'os' or 'file'
grep key_store_type ibc/relayer-config.yaml

# Re-import keys to OS keyring
hermes keys add --chain paw-1 --key-file relayer.json
```

### Issue: Docker secrets not loaded

**Symptom:**
```
ERROR: Couldn't connect to database: authentication failed
```

**Solution:**
```bash
# Verify secrets exist
ls -la compose/secrets/

# Check permissions
chmod 700 compose/secrets
chmod 600 compose/secrets/*.txt

# Verify docker-compose.yml references secrets correctly
grep -A5 "secrets:" compose/docker-compose.yml
```

### Issue: Peer keeps getting banned

**Symptom:**
```
WARN peer banned for 24h: peer_id=abc123, reason=repeated violations
```

**Solution:**
```bash
# Check peer's reputation
curl http://localhost:26657/peer_reputation?peer_id=abc123

# If false positive, whitelist the peer:
curl -X POST http://localhost:26657/whitelist_peer \
  -d '{"peer_id": "abc123"}'

# Or add to config:
unconditional_peer_ids = ["abc123..."]
```

---

## Testing Recommendations

### 1. Security Testing

```bash
# Test oversized message protection
# This should trigger a ban:
echo "Testing DoS protection..." | \
  head -c 11M | \
  nc localhost 26656

# Verify peer was banned:
curl http://localhost:26657/banned_peers
```

### 2. Load Testing

```bash
# Simulate multiple peers with varying behavior
go test ./p2p/discovery/... -run TestPeerReputation -v

# Monitor metrics:
curl http://localhost:26657/metrics | grep reputation
```

### 3. Integration Testing

```bash
# Full stack test
cd tests/e2e
go test -v -run TestP2PSecurityHandshake
```

---

## Production Deployment Checklist

- [ ] Node keys generated and backed up securely
- [ ] Chain ID matches across all nodes and genesis file
- [ ] IBC relayer using OS keyring (`key_store_type = 'os'`)
- [ ] Docker secrets created with proper permissions
- [ ] `.env` file configured for environment
- [ ] Secrets directory added to `ignore`
- [ ] Test keystore credentials removed
- [ ] Firewall rules updated for P2P port (26656)
- [ ] Monitoring configured for security events
- [ ] Backup strategy includes encrypted secrets
- [ ] Secret rotation schedule defined (90 days recommended)

---

## Security Best Practices

### 1. Key Management

- **Node Keys**: Store in `~/.paw/config/node_key.json` with 600 permissions
- **IBC Keys**: Use OS keyring or encrypted file storage
- **Backup**: Encrypt all backups, store off-site
- **Rotation**: Rotate secrets every 90 days

### 2. Network Security

- **Firewall**: Restrict P2P port (26656) to known peers
- **Whitelisting**: Use `unconditional_peer_ids` for trusted validators
- **Monitoring**: Alert on security events (oversized messages, chain ID mismatch)

### 3. Reputation System

- **Thresholds**: Adjust penalty thresholds based on network conditions
- **False Positives**: Monitor for legitimate peers being banned
- **Metrics**: Track violation rates, ban rates, average reputation

---

## Support

For issues or questions:
- Discord: https://discord.gg/paw-chain
- Security: security@paw-chain.org

**For security vulnerabilities, please email security@paw-chain.org directly. Do not create public issues.**

---

## Appendix A: File Modifications

### Modified Files

1. **p2p/nodekey.go** (NEW)
   - Ed25519 key generation and management
   - Node ID derivation from public key
   - Tendermint-compatible format

2. **p2p/discovery/types.go**
   - Added `ChainID`, `NodeKeyFile`, `NodeID` fields
   - Updated `DiscoveryConfig` structure

3. **p2p/discovery/peer_manager.go**
   - Lines 544-566: Updated handshake to use real chain ID and node ID
   - Lines 594-614: Added chain ID validation
   - Lines 677-733: Enhanced DoS protection with reputation penalties

4. **p2p/reputation/types.go**
   - Added new EventTypes: `EventTypeSecurity`, `EventTypeMisbehavior`, 
     `EventTypeOversizedMessage`, `EventTypeBandwidthAbuse`
   - Added violation tracking fields to `PeerMetrics`

5. **p2p/reputation/scorer.go**
   - Enhanced violation penalty calculation
   - Added streak multiplier for repeat offenders
   - Integrated new violation types into scoring

6. **ibc/relayer-config.yaml**
   - Changed `key_store_type` from 'Test' to 'os' for all chains
   - Added security comments and warnings

7. **compose/docker-compose.yml**
   - Already using Docker secrets (no changes needed)
   - Added documentation comments

8. **compose/setup-secrets.sh** (NEW)
   - Automated secret generation script
   - Creates secure random passwords
   - Sets proper file permissions

### Configuration Changes Required

| File | Change | Required |
|------|--------|----------|
| `config.toml` | Add `chain_id` field | Yes |
| `config.toml` | Add `node_key_file` path | Yes |
| `ibc/relayer-config.yaml` | Change `key_store_type` | Yes |
| `compose/.env` | Set environment variables | Yes (if using Docker) |
| `~/.paw/config/node_key.json` | Generate node key | Auto-generated |

---

Generated: 2024-11-25
Version: 1.0
