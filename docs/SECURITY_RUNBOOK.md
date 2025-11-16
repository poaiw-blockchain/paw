# PAW Blockchain Security Runbook

**Version:** 1.0
**Last Updated:** 2025-11-14
**Document Owner:** Security Operations Team

## Table of Contents

1. [Overview](#overview)
2. [Common Security Operations](#common-security-operations)
3. [Emergency Pause Procedures](#emergency-pause-procedures)
4. [Upgrade Procedures](#upgrade-procedures)
5. [Key Rotation Procedures](#key-rotation-procedures)
6. [Monitoring and Alerting](#monitoring-and-alerting)
7. [Security Incident Handling](#security-incident-handling)
8. [Access Control Management](#access-control-management)

---

## Overview

### Purpose

This Security Runbook provides step-by-step procedures for common security operations on the PAW blockchain. It serves as a quick reference for security engineers, validators, and operations staff during both routine operations and security incidents.

### When to Use This Runbook

- **Routine Operations:** Regular security tasks (key rotation, access reviews)
- **Emergency Response:** Quick reference during security incidents
- **Upgrades:** Secure deployment of network upgrades
- **Audits:** Security review procedures
- **Training:** Onboarding new security personnel

### Security Principles

1. **Least Privilege:** Grant minimum necessary permissions
2. **Defense in Depth:** Multiple layers of security controls
3. **Zero Trust:** Verify every access request
4. **Separation of Duties:** No single person has complete control
5. **Audit Everything:** Maintain comprehensive logs
6. **Fail Secure:** Systems fail to safe state

---

## Common Security Operations

### 1. Security Health Check

**Frequency:** Daily
**Duration:** 15 minutes
**Purpose:** Verify overall security posture

#### Daily Security Checklist

```bash
#!/bin/bash
# daily-security-check.sh

REPORT_FILE="/var/log/paw/security-check-$(date +%Y%m%d).log"

echo "=== PAW Security Health Check ===" | tee $REPORT_FILE
echo "Date: $(date -u)" | tee -a $REPORT_FILE
echo | tee -a $REPORT_FILE

# 1. Check for unauthorized access attempts
echo "1. Checking access logs..." | tee -a $REPORT_FILE
FAILED_SSH=$(grep "Failed password" /var/log/auth.log | wc -l)
echo "   Failed SSH attempts (24h): $FAILED_SSH" | tee -a $REPORT_FILE
if [ $FAILED_SSH -gt 100 ]; then
    echo "   ⚠ WARNING: High number of failed login attempts" | tee -a $REPORT_FILE
fi

# 2. Verify validator key permissions
echo "2. Checking validator key permissions..." | tee -a $REPORT_FILE
KEY_PERMS=$(stat -c %a ~/.paw/config/priv_validator_key.json)
if [ "$KEY_PERMS" = "600" ]; then
    echo "   ✓ Key permissions correct (600)" | tee -a $REPORT_FILE
else
    echo "   ✗ CRITICAL: Key permissions incorrect ($KEY_PERMS)" | tee -a $REPORT_FILE
fi

# 3. Check for unexpected processes
echo "3. Checking running processes..." | tee -a $REPORT_FILE
PAWD_COUNT=$(pgrep -c pawd)
echo "   Pawd processes: $PAWD_COUNT" | tee -a $REPORT_FILE
if [ $PAWD_COUNT -ne 1 ]; then
    echo "   ⚠ WARNING: Expected 1 pawd process, found $PAWD_COUNT" | tee -a $REPORT_FILE
fi

# 4. Verify firewall rules
echo "4. Checking firewall..." | tee -a $REPORT_FILE
if systemctl is-active --quiet ufw; then
    echo "   ✓ UFW active" | tee -a $REPORT_FILE
    ufw status | grep -q "26656.*ALLOW"
    if [ $? -eq 0 ]; then
        echo "   ✓ P2P port (26656) configured" | tee -a $REPORT_FILE
    fi
else
    echo "   ✗ WARNING: UFW not active" | tee -a $REPORT_FILE
fi

# 5. Check SSL certificate expiry
echo "5. Checking SSL certificates..." | tee -a $REPORT_FILE
if [ -f /etc/ssl/certs/paw-api.crt ]; then
    EXPIRY=$(openssl x509 -enddate -noout -in /etc/ssl/certs/paw-api.crt | cut -d= -f2)
    EXPIRY_EPOCH=$(date -d "$EXPIRY" +%s)
    NOW_EPOCH=$(date +%s)
    DAYS_LEFT=$(( (EXPIRY_EPOCH - NOW_EPOCH) / 86400 ))
    echo "   SSL expires in: $DAYS_LEFT days" | tee -a $REPORT_FILE
    if [ $DAYS_LEFT -lt 30 ]; then
        echo "   ⚠ WARNING: SSL certificate expires soon" | tee -a $REPORT_FILE
    fi
fi

# 6. Check for security updates
echo "6. Checking for security updates..." | tee -a $REPORT_FILE
UPDATES=$(apt list --upgradable 2>/dev/null | grep -i security | wc -l)
echo "   Security updates available: $UPDATES" | tee -a $REPORT_FILE
if [ $UPDATES -gt 0 ]; then
    echo "   ⚠ Security updates pending" | tee -a $REPORT_FILE
fi

# 7. Verify backup integrity
echo "7. Checking backups..." | tee -a $REPORT_FILE
LATEST_BACKUP=$(ls -t /mnt/backups/paw/full_*.tar.gz 2>/dev/null | head -1)
if [ -n "$LATEST_BACKUP" ]; then
    BACKUP_AGE=$(( ($(date +%s) - $(stat -c %Y "$LATEST_BACKUP")) / 86400 ))
    echo "   Latest backup: $BACKUP_AGE days old" | tee -a $REPORT_FILE
    if [ $BACKUP_AGE -gt 7 ]; then
        echo "   ⚠ WARNING: Backup older than 7 days" | tee -a $REPORT_FILE
    fi
else
    echo "   ✗ CRITICAL: No backups found" | tee -a $REPORT_FILE
fi

# 8. Check validator signing status
echo "8. Checking validator status..." | tee -a $REPORT_FILE
VALIDATOR_ADDR=$(pawd tendermint show-address)
SIGNING_INFO=$(pawd query slashing signing-info $VALIDATOR_ADDR --output json 2>/dev/null)
if [ $? -eq 0 ]; then
    MISSED_BLOCKS=$(echo $SIGNING_INFO | jq -r '.missed_blocks_counter')
    JAILED=$(echo $SIGNING_INFO | jq -r '.jailed')
    echo "   Missed blocks: $MISSED_BLOCKS" | tee -a $REPORT_FILE
    echo "   Jailed: $JAILED" | tee -a $REPORT_FILE
    if [ "$JAILED" = "true" ]; then
        echo "   ✗ CRITICAL: Validator is jailed" | tee -a $REPORT_FILE
    fi
fi

echo | tee -a $REPORT_FILE
echo "Security health check complete" | tee -a $REPORT_FILE

# Send report to monitoring
# curl -X POST https://monitoring.paw.network/api/security-report \
#     -H "Authorization: Bearer $MONITORING_TOKEN" \
#     -d @$REPORT_FILE
```

### 2. Security Audit Scan

**Frequency:** Weekly
**Duration:** 30-60 minutes
**Purpose:** Comprehensive security assessment

```bash
#!/bin/bash
# security-audit.sh

AUDIT_DIR="/var/log/paw/security-audits"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_DIR="$AUDIT_DIR/audit_$TIMESTAMP"

mkdir -p "$REPORT_DIR"

echo "Starting security audit: $TIMESTAMP"

# 1. Run GoSec (Go code security scanner)
echo "Running GoSec..."
gosec -conf security/.gosec.yml -fmt json -out "$REPORT_DIR/gosec.json" ./...

# 2. Run vulnerability check
echo "Running govulncheck..."
govulncheck -json ./... > "$REPORT_DIR/govulncheck.json"

# 3. Dependency audit
echo "Checking dependencies..."
go list -json -m all | nancy sleuth > "$REPORT_DIR/nancy.txt"

# 4. Secret scanning
echo "Scanning for secrets..."
gitleaks detect --no-git --verbose --report-path "$REPORT_DIR/gitleaks.json" .

# 5. Docker image scanning (if applicable)
if command -v trivy &> /dev/null; then
    echo "Scanning Docker images..."
    trivy image --format json --output "$REPORT_DIR/trivy.json" paw:latest
fi

# 6. Network port scan
echo "Scanning open ports..."
nmap -sV localhost -oN "$REPORT_DIR/nmap.txt"

# 7. System hardening check
echo "Checking system hardening..."
lynis audit system --quick --quiet --report-file "$REPORT_DIR/lynis.txt"

# 8. Generate summary
echo "Generating summary report..."
cat > "$REPORT_DIR/summary.txt" <<EOF
PAW Security Audit Summary
Date: $(date -u)
Audit ID: audit_$TIMESTAMP

=== GoSec Results ===
$(jq -r '.Stats.files, .Stats.lines, .Stats.found' "$REPORT_DIR/gosec.json" 2>/dev/null || echo "No issues found")

=== Vulnerability Scan ===
$(jq -r '.Vulns[].Symbol' "$REPORT_DIR/govulncheck.json" 2>/dev/null || echo "No vulnerabilities")

=== Dependency Issues ===
$(grep -c "HIGH\|CRITICAL" "$REPORT_DIR/nancy.txt" || echo "0") high/critical issues

=== Secrets Found ===
$(jq -r '.[] | .Description' "$REPORT_DIR/gitleaks.json" 2>/dev/null || echo "None")

=== Next Actions ===
1. Review all findings in: $REPORT_DIR
2. Prioritize critical/high severity issues
3. Create remediation tickets
4. Update this summary with resolution status

EOF

cat "$REPORT_DIR/summary.txt"

echo
echo "Audit complete. Results in: $REPORT_DIR"
echo "Summary: $REPORT_DIR/summary.txt"
```

### 3. Access Review

**Frequency:** Monthly
**Duration:** 1-2 hours
**Purpose:** Review and validate access permissions

```bash
#!/bin/bash
# access-review.sh

echo "=== PAW Access Review ==="
echo "Date: $(date -u)"
echo

# 1. Review system users
echo "1. System Users with Shell Access:"
echo "======================================"
awk -F: '$7 !~ /nologin|false/ {print $1, $7}' /etc/passwd

# 2. Review SSH authorized keys
echo
echo "2. SSH Authorized Keys:"
echo "======================="
for user in $(awk -F: '$7 !~ /nologin|false/ {print $1}' /etc/passwd); do
    AUTHORIZED_KEYS="/home/$user/.ssh/authorized_keys"
    if [ -f "$AUTHORIZED_KEYS" ]; then
        echo "User: $user"
        while IFS= read -r key; do
            # Extract key comment (usually contains user/host info)
            COMMENT=$(echo "$key" | awk '{print $NF}')
            echo "  - $COMMENT"
        done < "$AUTHORIZED_KEYS"
    fi
done

# 3. Review sudo access
echo
echo "3. Sudo Access:"
echo "==============="
grep -v '^#' /etc/sudoers.d/* 2>/dev/null || echo "None"

# 4. Review validator key access
echo
echo "4. Validator Key File Permissions:"
echo "==================================="
ls -l ~/.paw/config/priv_validator_key.json 2>/dev/null || echo "Not found"
ls -l ~/.paw/data/priv_validator_state.json 2>/dev/null || echo "Not found"

# 5. Review API keys (if applicable)
echo
echo "5. Active API Keys:"
echo "==================="
# List API keys from database or config
# pawd query api keys --output json | jq -r '.keys[] | "\(.name): \(.created_at)"'

# 6. Review firewall rules
echo
echo "6. Firewall Rules:"
echo "=================="
ufw status numbered

# 7. Review docker access (if applicable)
echo
echo "7. Docker Access:"
echo "================="
getent group docker | cut -d: -f4

echo
echo "=== Action Items ==="
echo "1. Verify all SSH keys are still needed"
echo "2. Remove access for departed team members"
echo "3. Rotate any shared credentials"
echo "4. Update access control documentation"
echo "5. Archive this review: /var/log/paw/access-reviews/review_$(date +%Y%m%d).txt"
```

---

## Emergency Pause Procedures

### When to Pause

Emergency pause should be executed when:

- Active exploit detected
- Critical vulnerability discovered
- Smart contract bug identified
- Oracle manipulation occurring
- At direction of incident commander

### DEX Module Pause

**Authority:** Governance or emergency multisig
**Impact:** Halts all DEX operations (swaps, liquidity operations)
**Duration:** Until fix deployed

#### Pause Procedure

```bash
#!/bin/bash
# emergency-pause-dex.sh

set -e

echo "=== EMERGENCY DEX PAUSE PROCEDURE ==="
echo "WARNING: This will halt all DEX operations"
echo

read -p "Confirm DEX pause (type 'PAUSE' to confirm): " CONFIRM
if [ "$CONFIRM" != "PAUSE" ]; then
    echo "Aborted"
    exit 1
fi

# Record pause decision
cat >> /var/log/paw/emergency-actions.log <<EOF
=== DEX PAUSE ===
Timestamp: $(date -u)
Executed by: $(whoami)
Reason: [MUST BE FILLED IN]
Incident ID: [MUST BE FILLED IN]
EOF

echo "Enter reason for pause:"
read -r REASON
echo "Reason: $REASON" >> /var/log/paw/emergency-actions.log

echo "Enter incident ID:"
read -r INCIDENT_ID
echo "Incident ID: $INCIDENT_ID" >> /var/log/paw/emergency-actions.log

# Get current block height for reference
CURRENT_HEIGHT=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')
echo "Current block height: $CURRENT_HEIGHT" >> /var/log/paw/emergency-actions.log

# Execute pause via governance
echo "Executing pause..."

# Method 1: Via governance (if time permits)
# cat > pause-proposal.json <<EOF
# {
#   "title": "Emergency DEX Pause",
#   "description": "Emergency pause due to: $REASON. Incident: $INCIDENT_ID",
#   "changes": [{
#     "subspace": "dex",
#     "key": "Paused",
#     "value": true
#   }]
# }
# EOF

# pawd tx gov submit-proposal param-change pause-proposal.json \
#     --from governance \
#     --gas auto \
#     --gas-adjustment 1.5 \
#     --yes

# Method 2: Direct pause (requires governance authority)
pawd tx dex pause-module \
    --from governance \
    --gas auto \
    --gas-adjustment 1.5 \
    --yes

if [ $? -eq 0 ]; then
    echo "✓ DEX pause executed successfully" | tee -a /var/log/paw/emergency-actions.log

    # Verify pause status
    sleep 10
    PAUSED=$(pawd query dex params --output json | jq -r '.paused')
    if [ "$PAUSED" = "true" ]; then
        echo "✓ Pause verified active" | tee -a /var/log/paw/emergency-actions.log
    else
        echo "✗ ERROR: Pause not confirmed" | tee -a /var/log/paw/emergency-actions.log
        exit 1
    fi

    # Notify stakeholders
    echo "Sending notifications..."

    # Status page update
    curl -X POST https://status.paw.network/api/incidents \
        -H "Authorization: Bearer $STATUS_PAGE_TOKEN" \
        -d "{
            \"name\": \"DEX Operations Paused\",
            \"status\": \"investigating\",
            \"message\": \"DEX operations have been paused due to a security concern. Investigation in progress.\",
            \"component_id\": \"dex\"
        }"

    # Slack notification
    curl -X POST $SLACK_WEBHOOK_URL \
        -H 'Content-Type: application/json' \
        -d "{\"text\":\"🚨 EMERGENCY: DEX paused at block $CURRENT_HEIGHT. Reason: $REASON. Incident: $INCIDENT_ID\"}"

    # Email notification
    echo "DEX paused at block $CURRENT_HEIGHT. Reason: $REASON. Incident ID: $INCIDENT_ID" | \
        mail -s "EMERGENCY: PAW DEX Paused" validators@paw.network

    echo
    echo "=== PAUSE COMPLETE ==="
    echo "Status: DEX operations halted"
    echo "Next steps:"
    echo "1. Investigate root cause"
    echo "2. Develop and test fix"
    echo "3. Prepare unpause procedure"
    echo "4. Communicate timeline to users"
else
    echo "✗ FAILED to pause DEX" | tee -a /var/log/paw/emergency-actions.log
    exit 1
fi
```

#### Unpause Procedure

```bash
#!/bin/bash
# unpause-dex.sh

set -e

echo "=== DEX UNPAUSE PROCEDURE ==="
echo

# Pre-flight checks
echo "Pre-flight checks:"
echo "=================="

# 1. Verify fix is deployed
read -p "Is the fix deployed and tested? (yes/no): " FIX_DEPLOYED
if [ "$FIX_DEPLOYED" != "yes" ]; then
    echo "Aborting: Fix must be deployed first"
    exit 1
fi

# 2. Verify security team approval
read -p "Has security team approved unpause? (yes/no): " SECURITY_APPROVED
if [ "$SECURITY_APPROVED" != "yes" ]; then
    echo "Aborting: Requires security team approval"
    exit 1
fi

# 3. Verify incident closed
read -p "Is incident resolved and closed? (yes/no): " INCIDENT_CLOSED
if [ "$INCIDENT_CLOSED" != "yes" ]; then
    echo "Aborting: Incident must be resolved first"
    exit 1
fi

echo
echo "All checks passed. Proceeding with unpause..."

# Record unpause
cat >> /var/log/paw/emergency-actions.log <<EOF
=== DEX UNPAUSE ===
Timestamp: $(date -u)
Executed by: $(whoami)
Fix deployed: Yes
Security approval: Yes
Incident resolved: Yes
EOF

# Execute unpause
pawd tx dex unpause-module \
    --from governance \
    --gas auto \
    --gas-adjustment 1.5 \
    --yes

if [ $? -eq 0 ]; then
    echo "✓ DEX unpause executed"

    # Verify unpause
    sleep 10
    PAUSED=$(pawd query dex params --output json | jq -r '.paused')
    if [ "$PAUSED" = "false" ]; then
        echo "✓ DEX operations resumed"

        # Update status page
        curl -X PATCH https://status.paw.network/api/incidents/$INCIDENT_ID \
            -H "Authorization: Bearer $STATUS_PAGE_TOKEN" \
            -d '{"status": "resolved", "message": "DEX operations have been resumed. All systems operational."}'

        # Notify stakeholders
        curl -X POST $SLACK_WEBHOOK_URL \
            -H 'Content-Type: application/json' \
            -d '{"text":"✓ DEX operations resumed. All systems operational."}'

        echo
        echo "=== UNPAUSE COMPLETE ==="
        echo "Status: DEX operations active"
    else
        echo "✗ ERROR: Unpause not confirmed"
        exit 1
    fi
else
    echo "✗ FAILED to unpause DEX"
    exit 1
fi
```

### Specific Pool Pause

For isolated issues affecting single pool:

```bash
# Pause specific pool
pawd tx dex pause-pool <POOL_ID> \
    --from governance \
    --gas auto \
    --yes

# Verify
pawd query dex pool <POOL_ID> --output json | jq '.pool.paused'

# Unpause when ready
pawd tx dex unpause-pool <POOL_ID> \
    --from governance \
    --gas auto \
    --yes
```

### Oracle Feed Pause

```bash
# Pause oracle updates for specific asset
pawd tx oracle pause-feed <ASSET_ID> \
    --from governance \
    --gas auto \
    --yes

# Resume oracle feed
pawd tx oracle resume-feed <ASSET_ID> \
    --from governance \
    --gas auto \
    --yes
```

---

## Upgrade Procedures

### Network Upgrade Process

**Types of upgrades:**

1. **Consensus-breaking:** Requires coordinated upgrade
2. **Non-consensus:** Can be deployed gradually
3. **Security patches:** May require emergency deployment

### Planning Phase

**Timeline: 2-4 weeks before upgrade**

#### Pre-Upgrade Checklist

```markdown
## Network Upgrade Checklist

### Planning (T-4 weeks)

- [ ] Upgrade version decided and tested
- [ ] Release notes published
- [ ] Breaking changes documented
- [ ] Migration guide created
- [ ] Testnet upgrade completed successfully
- [ ] Governance proposal drafted
- [ ] Validator communication sent

### Testing (T-3 weeks)

- [ ] Testnet upgrade successful
- [ ] Integration tests passing
- [ ] Security audit completed (if needed)
- [ ] Performance benchmarks acceptable
- [ ] Rollback procedure tested

### Coordination (T-2 weeks)

- [ ] Governance proposal submitted
- [ ] Validators acknowledged upgrade
- [ ] Upgrade block height determined
- [ ] Documentation published
- [ ] Support channels prepared

### Preparation (T-1 week)

- [ ] Binary releases published
- [ ] Checksum verification documented
- [ ] Upgrade scripts prepared
- [ ] Monitoring enhanced
- [ ] Incident response team on standby

### Execution (T-0)

- [ ] Validators ready
- [ ] Communication channels active
- [ ] Monitoring active
- [ ] Support available
```

### Governance Proposal Submission

```bash
#!/bin/bash
# submit-upgrade-proposal.sh

UPGRADE_NAME="v2.0.0"
UPGRADE_HEIGHT="1000000"  # Coordinate with validators
UPGRADE_INFO="https://github.com/paw-chain/paw/releases/tag/v2.0.0"

cat > upgrade-proposal.json <<EOF
{
    "title": "Upgrade to PAW v2.0.0",
    "description": "This proposal initiates an upgrade to PAW v2.0.0 at block height $UPGRADE_HEIGHT.\n\nKey changes:\n- Feature A\n- Feature B\n- Security fix C\n\nValidator guide: https://docs.paw.network/upgrades/v2.0.0\nRelease notes: $UPGRADE_INFO\n\nVoting period: 7 days\nUpgrade height: $UPGRADE_HEIGHT (estimated 2025-12-01 15:00 UTC)",
    "plan": {
        "name": "$UPGRADE_NAME",
        "height": "$UPGRADE_HEIGHT",
        "info": "$UPGRADE_INFO"
    },
    "deposit": "10000000upaw"
}
EOF

# Submit proposal
pawd tx gov submit-proposal software-upgrade $UPGRADE_NAME \
    --title "Upgrade to PAW v2.0.0" \
    --description "$(cat upgrade-proposal.json | jq -r '.description')" \
    --upgrade-height $UPGRADE_HEIGHT \
    --upgrade-info "$UPGRADE_INFO" \
    --deposit 10000000upaw \
    --from governance \
    --gas auto \
    --yes

# Get proposal ID
PROPOSAL_ID=$(pawd query gov proposals --status voting_period --output json | \
    jq -r '.proposals[-1].proposal_id')

echo "Proposal submitted: #$PROPOSAL_ID"
echo "Vote: pawd tx gov vote $PROPOSAL_ID yes --from validator --gas auto --yes"
```

### Validator Upgrade Procedure

**Execution: At upgrade height**

```bash
#!/bin/bash
# validator-upgrade.sh

set -e

UPGRADE_NAME="v2.0.0"
BINARY_URL="https://github.com/paw-chain/paw/releases/download/v2.0.0/pawd-linux-amd64"
EXPECTED_CHECKSUM="abc123def456..."  # SHA256 checksum

echo "=== PAW Validator Upgrade: $UPGRADE_NAME ==="

# 1. Download new binary
echo "1. Downloading new binary..."
wget -O /tmp/pawd-new "$BINARY_URL"

# 2. Verify checksum
echo "2. Verifying checksum..."
ACTUAL_CHECKSUM=$(sha256sum /tmp/pawd-new | cut -d' ' -f1)
if [ "$ACTUAL_CHECKSUM" != "$EXPECTED_CHECKSUM" ]; then
    echo "✗ ERROR: Checksum mismatch!"
    echo "Expected: $EXPECTED_CHECKSUM"
    echo "Actual: $ACTUAL_CHECKSUM"
    exit 1
fi
echo "✓ Checksum verified"

# 3. Backup current binary
echo "3. Backing up current binary..."
cp /usr/local/bin/pawd /usr/local/bin/pawd.backup.$(date +%s)

# 4. Install new binary
echo "4. Installing new binary..."
chmod +x /tmp/pawd-new
sudo mv /tmp/pawd-new /usr/local/bin/pawd

# 5. Verify installation
echo "5. Verifying installation..."
NEW_VERSION=$(pawd version)
echo "Installed version: $NEW_VERSION"

# 6. Wait for upgrade height (if using Cosmovisor, it auto-upgrades)
echo "6. Waiting for upgrade height..."
# Manual upgrade: restart service at exact upgrade height
# Cosmovisor: handles automatically

# If manual:
# CURRENT_HEIGHT=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')
# while [ $CURRENT_HEIGHT -lt $UPGRADE_HEIGHT ]; do
#     echo "Current: $CURRENT_HEIGHT, Upgrade: $UPGRADE_HEIGHT"
#     sleep 5
#     CURRENT_HEIGHT=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')
# done
#
# echo "Upgrade height reached. Restarting..."
# systemctl restart pawd

echo "✓ Upgrade preparation complete"
echo "Monitoring startup..."
journalctl -u pawd -f
```

### Using Cosmovisor (Recommended)

**Setup Cosmovisor for automated upgrades:**

```bash
#!/bin/bash
# setup-cosmovisor.sh

# Install Cosmovisor
go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@latest

# Setup directory structure
export DAEMON_HOME=$HOME/.paw
export DAEMON_NAME=pawd

mkdir -p $DAEMON_HOME/cosmovisor/genesis/bin
mkdir -p $DAEMON_HOME/cosmovisor/upgrades

# Copy current binary to genesis
cp $(which pawd) $DAEMON_HOME/cosmovisor/genesis/bin/

# Create systemd service
sudo tee /etc/systemd/system/cosmovisor.service > /dev/null <<EOF
[Unit]
Description=Cosmovisor daemon
After=network-online.target

[Service]
User=paw
ExecStart=$(which cosmovisor) run start
Restart=always
RestartSec=3
LimitNOFILE=65535
Environment="DAEMON_HOME=$DAEMON_HOME"
Environment="DAEMON_NAME=pawd"
Environment="DAEMON_ALLOW_DOWNLOAD_BINARIES=false"
Environment="DAEMON_RESTART_AFTER_UPGRADE=true"
Environment="UNSAFE_SKIP_BACKUP=false"

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable cosmovisor
sudo systemctl restart cosmovisor
```

**Prepare upgrade binary:**

```bash
# For upgrade named "v2.0.0"
mkdir -p ~/.paw/cosmovisor/upgrades/v2.0.0/bin

# Download and verify binary
wget -O ~/.paw/cosmovisor/upgrades/v2.0.0/bin/pawd \
    https://github.com/paw-chain/paw/releases/download/v2.0.0/pawd-linux-amd64

# Verify checksum
sha256sum ~/.paw/cosmovisor/upgrades/v2.0.0/bin/pawd

# Make executable
chmod +x ~/.paw/cosmovisor/upgrades/v2.0.0/bin/pawd

# Cosmovisor will automatically switch to this binary at upgrade height
```

### Post-Upgrade Validation

```bash
#!/bin/bash
# post-upgrade-validation.sh

echo "=== Post-Upgrade Validation ==="

# 1. Check node version
echo "1. Verifying version..."
VERSION=$(pawd version)
echo "Current version: $VERSION"

# 2. Check consensus
echo "2. Checking consensus..."
STATUS=$(curl -s http://localhost:26657/status)
CATCHING_UP=$(echo $STATUS | jq -r '.result.sync_info.catching_up')
LATEST_HEIGHT=$(echo $STATUS | jq -r '.result.sync_info.latest_block_height')

if [ "$CATCHING_UP" = "false" ]; then
    echo "✓ Node in consensus at height $LATEST_HEIGHT"
else
    echo "⚠ Node still catching up"
fi

# 3. Check validator signing
echo "3. Checking validator signing..."
sleep 30  # Wait for some blocks
SIGNED=$(journalctl -u pawd --since "30 seconds ago" | grep -c "signed" || true)
if [ $SIGNED -gt 0 ]; then
    echo "✓ Validator signing blocks"
else
    echo "✗ Validator not signing"
fi

# 4. Test transaction
echo "4. Testing transaction..."
pawd tx bank send test1 test2 1upaw --gas auto --yes --keyring-backend test
if [ $? -eq 0 ]; then
    echo "✓ Transactions working"
else
    echo "✗ Transaction failed"
fi

# 5. Check module versions
echo "5. Checking module versions..."
pawd query upgrade applied v2.0.0
if [ $? -eq 0 ]; then
    echo "✓ Upgrade applied successfully"
else
    echo "✗ Upgrade not confirmed"
fi

echo
echo "Validation complete"
```

---

## Key Rotation Procedures

### Validator Key Rotation

**Frequency:** Annually or after suspected compromise
**Downtime:** None (with proper procedure)
**Risk Level:** HIGH - requires careful execution

#### Pre-Rotation Checklist

- [ ] New validator keys generated in secure environment
- [ ] Old keys backed up
- [ ] Rotation procedure reviewed
- [ ] Team member available for monitoring
- [ ] Rollback plan prepared

#### Rotation Procedure

**Method 1: Seamless Rotation (Recommended)**

```bash
#!/bin/bash
# rotate-validator-key-seamless.sh
# This method uses a brief overlap to prevent downtime

set -e

echo "=== Validator Key Rotation ==="
echo "WARNING: This operation is sensitive. Ensure you understand the process."
echo

read -p "Continue with key rotation? (yes/no): " CONFIRM
if [ "$CONFIRM" != "yes" ]; then
    echo "Aborted"
    exit 1
fi

NODE_HOME="$HOME/.paw"
BACKUP_DIR="/secure/backups/validator-keys/rotation_$(date +%Y%m%d_%H%M%S)"
NEW_KEY_DIR="/secure/new-keys"

# 1. Backup current keys
echo "1. Backing up current keys..."
mkdir -p "$BACKUP_DIR"
cp "$NODE_HOME/config/priv_validator_key.json" "$BACKUP_DIR/"
cp "$NODE_HOME/data/priv_validator_state.json" "$BACKUP_DIR/"
echo "✓ Keys backed up to: $BACKUP_DIR"

# 2. Generate new validator key
echo "2. Generating new validator key..."
mkdir -p "$NEW_KEY_DIR"

# Generate on air-gapped machine or HSM
pawd init temp --home /tmp/newkey
cp /tmp/newkey/config/priv_validator_key.json "$NEW_KEY_DIR/"
shred -vfz -n 10 /tmp/newkey/config/priv_validator_key.json
rm -rf /tmp/newkey

# Reset state for new key
cat > "$NEW_KEY_DIR/priv_validator_state.json" <<EOF
{
  "height": "0",
  "round": 0,
  "step": 0
}
EOF

echo "New validator address: $(jq -r '.address' $NEW_KEY_DIR/priv_validator_key.json)"

# 3. Stop node briefly
echo "3. Stopping node for key rotation..."
sudo systemctl stop pawd

# 4. Replace keys
echo "4. Installing new keys..."
cp "$NEW_KEY_DIR/priv_validator_key.json" "$NODE_HOME/config/"
cp "$NEW_KEY_DIR/priv_validator_state.json" "$NODE_HOME/data/"
chmod 600 "$NODE_HOME/config/priv_validator_key.json"
chmod 600 "$NODE_HOME/data/priv_validator_state.json"

# 5. Start node
echo "5. Restarting node..."
sudo systemctl start pawd

# 6. Monitor
echo "6. Monitoring startup..."
sleep 10

# Check if signing
SIGNED=$(journalctl -u pawd --since "10 seconds ago" | grep -c "signed" || true)
if [ $SIGNED -gt 0 ]; then
    echo "✓ Validator signing with new key"
else
    echo "⚠ Validator not yet signing (may take a few blocks)"
fi

# 7. Update validator info on-chain
echo "7. Updating validator info..."
NEW_PUBKEY=$(pawd tendermint show-validator)

# Submit edit-validator transaction
pawd tx staking edit-validator \
    --new-moniker "$(pawd query staking validator $(pawd keys show validator --bech val -a) -o json | jq -r '.description.moniker')" \
    --from validator \
    --gas auto \
    --yes

echo
echo "=== Key Rotation Complete ==="
echo "Old key backed up to: $BACKUP_DIR"
echo "New validator key active"
echo "Monitor for next 24 hours to ensure no issues"
echo
echo "IMPORTANT: Securely backup new key:"
echo "  $NODE_HOME/config/priv_validator_key.json"
```

**Method 2: Using Tendermint KMS (Production Recommended)**

```bash
# Install Tendermint KMS with HSM support
# See: https://github.com/iqlusioninc/tmkms

# Configuration for YubiHSM2
cat > /etc/tmkms/tmkms.toml <<EOF
[[chain]]
id = "paw-mainnet-1"
key_format = { type = "bech32", account_key_prefix = "pawpub", consensus_key_prefix = "pawvalconspub" }

[[validator]]
addr = "tcp://localhost:26658"
chain_id = "paw-mainnet-1"
reconnect = true
secret_key = "/etc/tmkms/secrets/kms-identity.key"

[[providers.yubihsm]]
adapter = { type = "usb" }
auth = { key = 1, password_file = "/etc/tmkms/password" }
keys = [{ chain_ids = ["paw-mainnet-1"], key = 1 }]
EOF

# Import existing key to HSM
tmkms yubihsm keys import -i 1 /path/to/priv_validator_key.json

# Start KMS
systemctl start tmkms
```

### SSH Key Rotation

**Frequency:** Quarterly
**Purpose:** Reduce exposure of compromised keys

```bash
#!/bin/bash
# rotate-ssh-keys.sh

echo "=== SSH Key Rotation ==="

# 1. Generate new key pair
ssh-keygen -t ed25519 -C "paw-validator-$(date +%Y%m%d)" -f ~/.ssh/paw-new

# 2. Add new public key to authorized_keys
cat ~/.ssh/paw-new.pub >> ~/.ssh/authorized_keys

# 3. Test new key from another session
echo "Test new key from another terminal:"
echo "  ssh -i ~/.ssh/paw-new user@validator-host"
read -p "New key working? (yes/no): " WORKING

if [ "$WORKING" = "yes" ]; then
    # 4. Remove old keys
    echo "Removing old keys from authorized_keys..."
    # Manually edit ~/.ssh/authorized_keys to remove old keys

    # 5. Backup old private key
    mv ~/.ssh/paw-old ~/.ssh/backup/paw-old-$(date +%s) 2>/dev/null || true

    echo "✓ SSH key rotation complete"
    echo "New private key: ~/.ssh/paw-new"
    echo "Backup this key securely"
else
    echo "Rolling back..."
    # Remove new key from authorized_keys
    grep -v "$(cat ~/.ssh/paw-new.pub)" ~/.ssh/authorized_keys > ~/.ssh/authorized_keys.tmp
    mv ~/.ssh/authorized_keys.tmp ~/.ssh/authorized_keys
    echo "Rotation aborted"
fi
```

### API Key Rotation

```bash
#!/bin/bash
# rotate-api-keys.sh

echo "=== API Key Rotation ==="

# 1. Generate new API key
NEW_API_KEY=$(openssl rand -hex 32)
echo "New API key: $NEW_API_KEY"

# 2. Update in environment/config
echo "Update API key in:"
echo "  - /etc/paw/api.conf"
echo "  - Environment variables"
echo "  - Secret management system"

# 3. Deploy new key
# Update config files and restart services

# 4. Deprecation period
echo "Old key will be valid for 7 days for graceful transition"

# 5. After 7 days, revoke old key
# Schedule: at now + 7 days <<EOF
# /usr/local/bin/revoke-api-key.sh OLD_KEY_ID
# EOF
```

---

## Monitoring and Alerting

### Critical Security Alerts

#### 1. Validator Double-Sign Detection

```yaml
# prometheus-alerts.yml

groups:
  - name: validator_security
    interval: 10s
    rules:
      - alert: ValidatorDoubleSigning
        expr: increase(tendermint_consensus_validator_double_signed[5m]) > 0
        for: 1m
        labels:
          severity: critical
          category: security
        annotations:
          summary: 'Validator double-signing detected'
          description: 'Validator {{ $labels.validator_address }} has double-signed. IMMEDIATE ACTION REQUIRED.'
          action: '1. Investigate validator compromise 2. Halt validator immediately 3. Initiate incident response'
```

#### 2. Unauthorized Access Attempts

```yaml
- alert: UnauthorizedAccessAttempts
  expr: rate(ssh_failed_login[5m]) > 10
  for: 5m
  labels:
    severity: high
    category: security
  annotations:
    summary: 'High rate of failed SSH attempts'
    description: '{{ $value }} failed SSH attempts per second from {{ $labels.source_ip }}'
    action: 'Review auth logs, consider IP blocking'
```

#### 3. Suspicious Transaction Patterns

```yaml
- alert: SuspiciousLargeTransaction
  expr: paw_tx_amount_upaw > 1000000000000 # 1M PAW
  labels:
    severity: medium
    category: security
  annotations:
    summary: 'Large transaction detected'
    description: 'Transaction of {{ $value }} upaw from {{ $labels.from_address }}'
    action: 'Review transaction for legitimacy'
```

#### 4. Key File Permission Changes

```bash
#!/bin/bash
# monitor-key-permissions.sh
# Run via cron every 5 minutes

KEY_FILE="$HOME/.paw/config/priv_validator_key.json"
EXPECTED_PERMS="600"

CURRENT_PERMS=$(stat -c %a "$KEY_FILE")

if [ "$CURRENT_PERMS" != "$EXPECTED_PERMS" ]; then
    # Alert
    curl -X POST $SLACK_WEBHOOK \
        -d "{\"text\":\"🚨 SECURITY: Validator key permissions changed to $CURRENT_PERMS (expected $EXPECTED_PERMS)\"}"

    # Auto-remediate
    chmod 600 "$KEY_FILE"
    echo "$(date): Key permissions corrected from $CURRENT_PERMS to $EXPECTED_PERMS" >> /var/log/paw/security.log
fi
```

### Security Monitoring Dashboard

**Grafana Dashboard JSON** (excerpt):

```json
{
  "dashboard": {
    "title": "PAW Security Monitoring",
    "panels": [
      {
        "title": "Failed Authentication Attempts",
        "targets": [
          {
            "expr": "rate(ssh_failed_login[5m])"
          }
        ]
      },
      {
        "title": "Validator Missed Blocks",
        "targets": [
          {
            "expr": "tendermint_consensus_validator_missed_blocks"
          }
        ]
      },
      {
        "title": "Unusual Transaction Patterns",
        "targets": [
          {
            "expr": "rate(paw_tx_total[5m]) > 100"
          }
        ]
      },
      {
        "title": "Open Network Connections",
        "targets": [
          {
            "expr": "node_network_connections_total"
          }
        ]
      }
    ]
  }
}
```

### Log Monitoring

**Loki query examples:**

```logql
# Failed authentication attempts
{job="auth"} |= "Failed password"

# Validator errors
{job="paw-node"} |= "error" |= "validator"

# Suspicious activity
{job="paw-node"} |~ "panic|fatal|double.*sign"

# Large transactions
{job="paw-node"} |= "execute transaction" | json | amount > 1000000000
```

---

## Security Incident Handling

### Quick Reference: Incident Response

**See INCIDENT_RESPONSE_PLAN.md for complete procedures**

#### Immediate Actions for Common Incidents

**1. Suspected Key Compromise**

```bash
# Immediately isolate validator
systemctl stop pawd

# Secure current keys
cp ~/.paw/config/priv_validator_key.json /secure/forensics/

# Alert team
curl -X POST $SLACK_WEBHOOK -d '{"text":"🚨 CRITICAL: Validator key compromise suspected"}'

# Follow key rotation procedure
# See "Key Rotation Procedures" section above
```

**2. Active Exploit**

```bash
# Pause affected module
./emergency-pause-dex.sh

# Alert team
./security-alert.sh "Active exploit detected in DEX"

# Begin investigation
# See INCIDENT_RESPONSE_PLAN.md "Smart Contract Exploit"
```

**3. DDoS Attack**

```bash
# Enable aggressive rate limiting
ufw limit 26657/tcp
ufw limit 26656/tcp

# Enable CDN "Under Attack" mode
# (via Cloudflare dashboard or API)

# Monitor for escalation
watch -n 5 'netstat -an | grep SYN_RECV | wc -l'
```

---

## Access Control Management

### Adding New Team Member

```bash
#!/bin/bash
# add-team-member.sh

USERNAME="$1"
SSH_PUBLIC_KEY="$2"
ROLE="$3"  # validator, developer, auditor

if [ -z "$USERNAME" ] || [ -z "$SSH_PUBLIC_KEY" ]; then
    echo "Usage: $0 <username> <ssh-public-key> <role>"
    exit 1
fi

echo "Adding team member: $USERNAME (Role: $ROLE)"

# 1. Create system user
sudo useradd -m -s /bin/bash "$USERNAME"

# 2. Add SSH key
sudo mkdir -p /home/$USERNAME/.ssh
echo "$SSH_PUBLIC_KEY" | sudo tee -a /home/$USERNAME/.ssh/authorized_keys
sudo chmod 700 /home/$USERNAME/.ssh
sudo chmod 600 /home/$USERNAME/.ssh/authorized_keys
sudo chown -R $USERNAME:$USERNAME /home/$USERNAME/.ssh

# 3. Grant appropriate permissions based on role
case $ROLE in
    validator)
        # Add to paw group for validator operations
        sudo usermod -a -G paw $USERNAME
        ;;
    developer)
        # Read-only access
        sudo usermod -a -G paw-read $USERNAME
        ;;
    auditor)
        # Read-only with log access
        sudo usermod -a -G paw-read,adm $USERNAME
        ;;
esac

# 4. Log access grant
echo "$(date -u): Added user $USERNAME (role: $ROLE)" >> /var/log/paw/access-changes.log

# 5. Notify team
curl -X POST $SLACK_WEBHOOK \
    -d "{\"text\":\"New team member added: $USERNAME (Role: $ROLE)\"}"

echo "✓ User $USERNAME added successfully"
echo "Next steps:"
echo "1. User should change password: passwd"
echo "2. Review permissions: id $USERNAME"
echo "3. Update team documentation"
```

### Removing Team Member

```bash
#!/bin/bash
# remove-team-member.sh

USERNAME="$1"
REASON="$2"

echo "Removing access for: $USERNAME"
echo "Reason: $REASON"

# 1. Disable account
sudo usermod -L $USERNAME

# 2. Remove SSH keys
sudo rm /home/$USERNAME/.ssh/authorized_keys

# 3. Kill active sessions
sudo pkill -u $USERNAME

# 4. Revoke sudo access (if any)
sudo deluser $USERNAME sudo

# 5. Log removal
echo "$(date -u): Removed user $USERNAME. Reason: $REASON" >> /var/log/paw/access-changes.log

# 6. Security review
echo "Security review required:"
echo "- [ ] Rotate shared credentials"
echo "- [ ] Review recent activities by user"
echo "- [ ] Check for backdoors or persistence"
echo "- [ ] Update documentation"

# 7. Notify team
curl -X POST $SLACK_WEBHOOK \
    -d "{\"text\":\"Access removed for: $USERNAME. Reason: $REASON\"}"
```

---

## Appendices

### Appendix A: Security Tools Quick Reference

```bash
# Network scanning
nmap -sV localhost
netstat -tulpn

# Process monitoring
ps aux | grep pawd
top -u paw

# File integrity
sha256sum ~/.paw/config/priv_validator_key.json
find ~/.paw -type f -mtime -1  # Files modified in last 24h

# Log analysis
journalctl -u pawd --since "1 hour ago"
grep -i "failed\|error\|panic" /var/log/paw/*.log

# Access review
last -n 50
w
who

# File permissions
find ~/.paw -type f -perm /o+rwx  # World-readable files (bad)
find /etc -type f -name "*.conf" -perm /o+w  # World-writable configs (bad)
```

### Appendix B: Emergency Contact List

```markdown
## Emergency Contacts

### On-Call Rotation

- Current on-call: See PagerDuty schedule
- PagerDuty: security@paw.pagerduty.com

### Security Team

- Security Lead: security-lead@paw.network
- Incident Commander: ic@paw.network

### External Resources

- Security Audit Firm: [Name] - [Contact]
- Legal Counsel: legal@paw.network
- Law Enforcement Contact: [Details for criminal activity]

### Validator Coordination

- Validators mailing list: validators@paw.network
- Validators Telegram: @PAWValidators
- Emergency broadcast: emergency@paw.network
```

### Appendix C: Compliance Checklist

```markdown
## Security Compliance Checklist

### Daily

- [ ] Security health check completed
- [ ] Monitoring alerts reviewed
- [ ] Backup verification (automated)

### Weekly

- [ ] Security audit scan completed
- [ ] Vulnerability scan reviewed
- [ ] Patch status reviewed

### Monthly

- [ ] Access review completed
- [ ] Security training completed
- [ ] Incident response drill (if scheduled)

### Quarterly

- [ ] Key rotation completed
- [ ] Disaster recovery test
- [ ] Security documentation updated
- [ ] Compliance audit

### Annually

- [ ] Full security audit
- [ ] Penetration testing
- [ ] Validator key rotation
- [ ] Security policy review
```

---

## Document Control

**Version History:**

| Version | Date       | Author        | Changes         |
| ------- | ---------- | ------------- | --------------- |
| 1.0     | 2025-11-14 | Security Team | Initial release |

**Review Schedule:** Quarterly or after security incidents

**Next Review Date:** 2026-02-14

**Approval:**

- CISO: ********\_******** Date: ****\_****
- Security Lead: ********\_******** Date: ****\_****

**Related Documents:**

- INCIDENT_RESPONSE_PLAN.md
- DISASTER_RECOVERY.md
- SECURITY_TESTING.md
- security/README.md

---

**Remember:** Security is everyone's responsibility. When in doubt, ask. Better to pause and verify than rush and compromise security.
