# PAW Blockchain Incident Response Plan

**Version:** 1.0
**Last Updated:** 2025-11-14
**Document Owner:** Security & Operations Team

## Table of Contents

1. [Overview](#overview)
2. [Incident Classification](#incident-classification)
3. [Response Team Structure](#response-team-structure)
4. [Communication Protocols](#communication-protocols)
5. [Response Procedures](#response-procedures)
6. [Post-Incident Review](#post-incident-review)
7. [Contact Information](#contact-information)

---

## Overview

This Incident Response Plan (IRP) provides a structured approach to handling security incidents and operational disruptions on the PAW blockchain. The plan ensures rapid detection, containment, and resolution of incidents while maintaining transparency with stakeholders.

### Objectives

- Minimize impact and duration of security incidents
- Protect validator integrity and network consensus
- Maintain chain continuity and data integrity
- Provide clear communication to stakeholders
- Learn from incidents to improve security posture

### Scope

This plan covers all incidents affecting:

- PAW blockchain network and consensus
- Validator nodes and infrastructure
- Smart contracts and DEX operations
- Oracle price feeds
- API and user-facing services
- Private keys and cryptographic material

---

## Incident Classification

### Severity Levels

#### CRITICAL (P0)

**Response Time:** Immediate (< 15 minutes)
**Resolution Target:** < 4 hours

**Criteria:**

- Complete network halt or consensus failure
- Active exploit draining significant funds (>$100K)
- Multiple validator compromises
- Private key compromise of core infrastructure
- Data corruption affecting chain state
- Complete service outage affecting all users

**Escalation:** Immediately notify all response team members, initiate emergency procedures

#### HIGH (P1)

**Response Time:** < 30 minutes
**Resolution Target:** < 24 hours

**Criteria:**

- Single validator compromise
- Smart contract exploit (limited scope)
- Oracle manipulation affecting price feeds
- Significant DDoS attack degrading service
- Network partition affecting >30% of validators
- Major security vulnerability discovered

**Escalation:** Notify incident commander and relevant team leads

#### MEDIUM (P2)

**Response Time:** < 2 hours
**Resolution Target:** < 72 hours

**Criteria:**

- Minor DDoS attack
- Performance degradation (non-critical)
- Security vulnerability (theoretical or low impact)
- Suspicious activity detected (no confirmed exploit)
- Configuration errors affecting single node
- API rate limiting bypass

**Escalation:** Notify on-call engineer and security team

#### LOW (P3)

**Response Time:** < 8 hours
**Resolution Target:** < 1 week

**Criteria:**

- Minor bugs or issues
- Documentation errors
- Non-security configuration issues
- Performance optimization opportunities
- User-reported issues (isolated)

**Escalation:** Create ticket for standard review

---

## Response Team Structure

### Core Incident Response Team (IRT)

#### Incident Commander (IC)

**Primary:** Chief Technology Officer
**Backup:** Lead Security Engineer

**Responsibilities:**

- Overall incident coordination
- Final decision authority
- Stakeholder communication
- Resource allocation
- Post-incident review leadership

#### Security Lead

**Primary:** Lead Security Engineer
**Backup:** Senior Backend Engineer

**Responsibilities:**

- Security analysis and assessment
- Exploit investigation
- Vulnerability remediation
- Forensics coordination
- Security tool deployment

#### Validator Operations Lead

**Primary:** DevOps Lead
**Backup:** Senior Infrastructure Engineer

**Responsibilities:**

- Validator health monitoring
- Node recovery procedures
- Infrastructure scaling
- Network partition resolution
- Backup/restore operations

#### Smart Contract Lead

**Primary:** Smart Contract Developer
**Backup:** Senior Blockchain Developer

**Responsibilities:**

- Contract security analysis
- Emergency pause execution
- Contract upgrade coordination
- DEX operations monitoring
- Oracle feed validation

#### Communications Lead

**Primary:** Head of Communications
**Backup:** Community Manager

**Responsibilities:**

- Public communications
- Stakeholder updates
- Social media monitoring
- Documentation of communications
- Transparency reporting

#### Technical Writer

**Primary:** Documentation Lead
**Backup:** Senior Developer

**Responsibilities:**

- Incident documentation
- Timeline tracking
- Action item recording
- Post-mortem documentation
- Knowledge base updates

### On-Call Rotation

- **Primary On-Call:** 24/7 rotating weekly schedule
- **Secondary On-Call:** Backup coverage
- **Escalation Path:** Primary → Secondary → Incident Commander
- **Handoff Time:** Monday 9:00 AM UTC

---

## Communication Protocols

### Internal Communication

#### Slack Channels

- `#paw-incidents` - Primary incident coordination (CRITICAL/HIGH)
- `#paw-critical` - Critical alerts and escalations
- `#paw-alerts` - Monitoring alerts and warnings
- `#paw-security` - Security-specific discussions
- `#paw-validators` - Validator operations

#### Communication Bridge

For CRITICAL incidents, establish a conference bridge:

- **Primary:** Zoom incident room (persistent link)
- **Backup:** Discord voice channel
- **Phone:** Emergency conference line

#### Status Updates

- **CRITICAL:** Every 30 minutes
- **HIGH:** Every 2 hours
- **MEDIUM:** Every 8 hours
- **LOW:** Daily until resolved

### External Communication

#### Status Page

**URL:** status.paw.network

**Update Frequency:**

- CRITICAL: Every 30-60 minutes
- HIGH: Every 2-4 hours
- MEDIUM: Every 8-12 hours

**Template:**

```
[TIMESTAMP] - [STATUS: INVESTIGATING/IDENTIFIED/MONITORING/RESOLVED]

Brief description of the incident and current status.

Impact: [Description of user/service impact]
Next Update: [Expected time of next update]

Updates:
- [HH:MM UTC] - Latest action/finding
- [HH:MM UTC] - Previous update
```

#### Social Media

**Platforms:** Twitter/X (@PAWBlockchain), Discord, Telegram

**Guidelines:**

- Post initial notification within 30 minutes for CRITICAL/HIGH
- Use clear, non-technical language
- Avoid speculation or premature conclusions
- Direct users to status page for updates
- Coordinate with Communications Lead

**Template:**

```
We are investigating [brief issue description].
[Impact statement]. Updates: status.paw.network
```

#### Email Notifications

**Distribution Lists:**

- `validators@paw.network` - All validator operators
- `integrators@paw.network` - Exchange and wallet integrations
- `stakeholders@paw.network` - Major stakeholders

**Send for:** CRITICAL and HIGH incidents

---

## Response Procedures

### General Incident Response Workflow

```
┌─────────────────┐
│  Detection      │
│  & Alert        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Initial        │
│  Assessment     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Classification │
│  & Escalation   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Containment    │
│  & Investigation│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Eradication    │
│  & Recovery     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Post-Incident  │
│  Review         │
└─────────────────┘
```

### 1. Detection & Alert

**Sources:**

- Automated monitoring (Prometheus alerts)
- Validator reports
- User reports
- Security scanning tools
- External security researchers
- Social media monitoring

**Initial Actions:**

1. Acknowledge the alert
2. Verify the incident is real (not false positive)
3. Record initial detection time
4. Begin incident documentation

### 2. Initial Assessment

**Checklist:**

- [ ] What is the nature of the incident?
- [ ] What systems/services are affected?
- [ ] What is the scope of impact?
- [ ] How many users/validators are affected?
- [ ] Is there active malicious activity?
- [ ] What is the potential financial impact?
- [ ] Is there immediate danger of escalation?

**Actions:**

1. Gather initial facts (no speculation)
2. Check monitoring dashboards for anomalies
3. Review recent changes or deployments
4. Identify affected components
5. Document findings in incident channel

### 3. Classification & Escalation

**Actions:**

1. Assign severity level (P0-P3)
2. Notify appropriate team members per severity
3. Designate Incident Commander
4. Establish communication channels
5. Create incident ticket/tracking

**Escalation Matrix:**

| Severity | Notify                       | Response Time | Communication Bridge |
| -------- | ---------------------------- | ------------- | -------------------- |
| CRITICAL | All IRT members + executives | < 15 min      | Yes - immediately    |
| HIGH     | IC + relevant leads          | < 30 min      | If needed            |
| MEDIUM   | On-call + security           | < 2 hours     | No                   |
| LOW      | On-call engineer             | < 8 hours     | No                   |

### 4. Containment & Investigation

**Primary Goal:** Stop the bleeding, prevent further damage

**Actions vary by incident type (see specific procedures below)**

**General containment measures:**

- Isolate affected systems
- Implement emergency controls
- Preserve evidence for forensics
- Monitor for lateral movement
- Document all actions taken

### 5. Eradication & Recovery

**Actions:**

1. Remove threat/vulnerability
2. Apply patches or fixes
3. Verify integrity of affected systems
4. Test recovery procedures
5. Gradual service restoration
6. Monitor for recurrence

### 6. Post-Incident Review

**Actions:**

1. Schedule post-mortem within 48 hours
2. Document timeline and actions
3. Identify root causes
4. Determine preventive measures
5. Update runbooks and procedures
6. Share lessons learned

---

## Specific Incident Response Procedures

### Procedure 1: Smart Contract Exploit

**Severity:** Typically CRITICAL or HIGH

#### Detection Indicators

- Unusual transaction patterns
- Large unexpected fund movements
- Smart contract event anomalies
- User reports of fund discrepancies
- External security researcher alert

#### Immediate Response Checklist

**Phase 1: Assess (First 15 minutes)**

- [ ] Confirm exploit is occurring
- [ ] Identify affected contract(s)
- [ ] Estimate funds at risk
- [ ] Check if exploit is ongoing
- [ ] Review recent contract interactions
- [ ] Notify Incident Commander and Smart Contract Lead

**Phase 2: Contain (15-30 minutes)**

- [ ] **Execute emergency pause** on affected contract
  ```bash
  # Connect to validator node
  pawcli tx dex pause-module \
    --from governance \
    --gas auto \
    --gas-adjustment 1.5
  ```
- [ ] Halt related operations (swaps, liquidity adds)
- [ ] Monitor for continued exploitation attempts
- [ ] Document all transactions related to exploit
- [ ] Alert major exchanges to halt deposits/withdrawals

**Phase 3: Investigate (30 minutes - 4 hours)**

- [ ] Analyze exploit transaction(s)

  ```bash
  # Get transaction details
  pawcli query tx <TX_HASH>

  # Review contract state
  pawcli query dex pool <POOL_ID>
  ```

- [ ] Identify vulnerability in code
- [ ] Trace fund movement on-chain
- [ ] Estimate total loss
- [ ] Determine if other contracts vulnerable
- [ ] Check if attacker identity is traceable
- [ ] Review contract audit reports
- [ ] Engage external security firm if needed

**Phase 4: Remediate (2-24 hours)**

- [ ] Develop patch or contract upgrade
- [ ] Test fix in isolated environment
- [ ] Prepare governance proposal for upgrade
- [ ] Review fix with security team
- [ ] Prepare communication for users
- [ ] Coordinate with affected users

**Phase 5: Recover (24-72 hours)**

- [ ] Deploy fixed contract via governance
- [ ] Verify fix resolves vulnerability
- [ ] Gradually unpause operations
- [ ] Monitor closely for issues
- [ ] Process user claims if applicable
- [ ] Update documentation

#### Communication Templates

**Initial Alert (First 30 minutes):**

```
SECURITY INCIDENT - [Contract Name]

We have detected unusual activity in [contract name] and have
implemented an emergency pause as a precautionary measure.

Impact: [Swap/liquidity operations] are temporarily unavailable.
Funds: All user funds are secure. [Or: We are investigating
potential fund exposure.]

Status: Investigating
Next Update: [Time]
```

**Resolution Communication:**

```
INCIDENT RESOLVED - [Contract Name]

Summary: [Brief description of vulnerability]
Impact: [What occurred]
Resolution: [What was fixed]
User Action Required: [If any]

All systems are now operational. We appreciate your patience.

Detailed post-mortem: [Link]
```

#### Post-Incident Actions

- [ ] Complete forensic analysis
- [ ] File bug bounty report if applicable
- [ ] Update audit requirements
- [ ] Review all similar code patterns
- [ ] Conduct code review training
- [ ] Update testing procedures
- [ ] Consider white hat rescue if funds recoverable

---

### Procedure 2: Validator Compromise

**Severity:** HIGH or CRITICAL (based on number of validators)

#### Detection Indicators

- Validator double-signing detected
- Unusual validator behavior (voting patterns)
- Validator node SSH access from unknown IPs
- Unexpected validator key usage
- Alert from validator operator
- Missing blocks from validator

#### Immediate Response Checklist

**Phase 1: Assess (First 10 minutes)**

- [ ] Confirm validator compromise
- [ ] Identify affected validator(s)
- [ ] Determine if consensus at risk
- [ ] Check for double-signing
- [ ] Verify current voting power of affected validator(s)
- [ ] Calculate total compromised voting power
- [ ] Notify Incident Commander and Validator Ops Lead

**Phase 2: Contain (10-30 minutes)**

- [ ] **Isolate compromised validator immediately**

  ```bash
  # On validator node
  systemctl stop pawd
  systemctl stop cometbft

  # Block network access if needed
  iptables -A INPUT -j DROP
  iptables -A OUTPUT -j DROP
  ```

- [ ] Tombstone validator if double-signing detected
  ```bash
  # From governance
  pawcli tx slashing unjail <VALIDATOR_ADDRESS> --from governance
  ```
- [ ] Alert other validators to monitor
- [ ] Preserve validator logs and state
- [ ] Rotate SSH keys and access credentials
- [ ] Enable enhanced monitoring on remaining validators

**Phase 3: Investigate (30 minutes - 4 hours)**

- [ ] Review validator access logs

  ```bash
  # Check SSH access
  grep -i "accepted publickey\|failed password" /var/log/auth.log

  # Review pawd logs
  journalctl -u pawd --since "2 hours ago" -p err
  ```

- [ ] Identify compromise vector
- [ ] Check for unauthorized key changes
- [ ] Review recent validator transactions
- [ ] Analyze network traffic logs
- [ ] Scan for malware/backdoors
- [ ] Determine timeline of compromise
- [ ] Check if other validators affected

**Phase 4: Remediate (2-12 hours)**

- [ ] Provision new validator infrastructure
- [ ] Generate new validator keys in secure environment
  ```bash
  # Generate new validator key
  pawd init <moniker> --chain-id paw-mainnet-1
  pawd keys add validator --keyring-backend file
  ```
- [ ] Harden new validator (apply security patches)
- [ ] Implement additional security controls
- [ ] Review and update firewall rules
- [ ] Enable additional monitoring/alerting
- [ ] Document security improvements

**Phase 5: Recover (4-24 hours)**

- [ ] Migrate validator to new infrastructure
- [ ] Update validator key registration
- [ ] Restore voting power via governance if needed
- [ ] Verify validator signing correctly
- [ ] Monitor for 24-48 hours
- [ ] Update validator documentation

#### Consensus Risk Decision Tree

```
Is compromised voting power > 33%?
│
├─ YES → CRITICAL: Consensus at risk
│         └─ Coordinate emergency validator restart
│         └─ May require chain halt and recovery
│
└─ NO → Is compromised voting power > 10%?
        │
        ├─ YES → HIGH: Significant risk
        │        └─ Fast-track containment
        │        └─ Alert all validators
        │
        └─ NO → MEDIUM: Isolated incident
                 └─ Standard containment procedure
```

#### Communication Templates

**Initial Alert:**

```
VALIDATOR SECURITY INCIDENT

We have detected a potential security issue affecting validator
[name/ID]. As a precautionary measure, this validator has been
isolated from the network.

Network Impact: None - consensus operating normally
Validator Count: [X] active validators remain
Voting Power: [Y]% unaffected

Status: Investigating
Next Update: [Time]
```

#### Post-Incident Actions

- [ ] Security audit of all validators
- [ ] Review validator security requirements
- [ ] Update validator onboarding procedures
- [ ] Implement additional monitoring
- [ ] Consider HSM requirements
- [ ] Share security best practices with validators

---

### Procedure 3: Network Partition

**Severity:** HIGH or CRITICAL

#### Detection Indicators

- Multiple validators unable to reach each other
- Consensus stalled or forking
- Two or more competing chains
- Significant increase in missed blocks
- P2P peer count drops dramatically
- Geographic isolation detected

#### Immediate Response Checklist

**Phase 1: Assess (First 15 minutes)**

- [ ] Confirm network partition exists
- [ ] Identify scope of partition
- [ ] Map which validators are isolated
- [ ] Calculate voting power on each partition
- [ ] Check for multiple chain heads
- [ ] Determine root cause (network issue, attack, bug)
- [ ] Notify Incident Commander and Validator Ops Lead

**Phase 2: Contain (15-45 minutes)**

- [ ] **Halt consensus if multiple competing chains**
  ```bash
  # Emergency chain halt via validator coordination
  # Coordinate with >67% validators to stop
  systemctl stop pawd
  ```
- [ ] Establish communication with isolated validators
- [ ] Document current chain state on all partitions
- [ ] Preserve all logs and state data
- [ ] Prevent further divergence

**Phase 3: Investigate (30 minutes - 2 hours)**

- [ ] Identify partition boundaries

  ```bash
  # Check peer connectivity
  curl http://localhost:26657/net_info | jq

  # Check consensus state
  curl http://localhost:26657/consensus_state | jq
  ```

- [ ] Determine cause (network, BGP, DDoS, attack)
- [ ] Identify which partition has >67% voting power
- [ ] Check for double-signing
- [ ] Review network topology
- [ ] Analyze routing/connectivity issues

**Phase 4: Remediate (1-8 hours)**

- [ ] Resolve network connectivity issues
- [ ] Coordinate validator reconnection
- [ ] **Determine canonical chain** (>67% voting power)
- [ ] Plan state reconciliation for minority partition
- [ ] Test connectivity before restart
- [ ] Prepare rollback if needed

**Phase 5: Recover (2-12 hours)**

- [ ] Coordinate synchronized restart

  ```bash
  # On all validators simultaneously
  systemctl start pawd

  # Verify consensus
  curl http://localhost:26657/status | jq .result.sync_info
  ```

- [ ] Monitor consensus formation
- [ ] Verify all validators reconnecting
- [ ] Handle any forked state
- [ ] Monitor closely for 24-48 hours

#### Decision Matrix: Which Chain to Keep

| Criteria          | Weight   | Description                        |
| ----------------- | -------- | ---------------------------------- |
| Voting Power      | Critical | Must have >67% for valid consensus |
| Block Height      | High     | Higher height = more progress      |
| Transaction Count | Medium   | More user activity                 |
| Validator Count   | Medium   | More decentralization              |
| First to Halt     | Low      | If manual halt needed              |

**Rule:** Always choose partition with >67% voting power. If unclear, halt both and coordinate recovery.

#### Communication Templates

**Initial Alert:**

```
NETWORK PARTITION DETECTED

We have detected a network partition affecting consensus.

Impact: Block production may be slowed or halted
Status: Coordinating validator reconnection
Estimated Resolution: [Timeframe]

Your funds are safe. The chain will reconcile once
connectivity is restored.

Updates: status.paw.network
```

#### Post-Incident Actions

- [ ] Network topology review
- [ ] Implement partition detection monitoring
- [ ] Improve validator geographic distribution
- [ ] Update emergency coordination procedures
- [ ] Review consensus parameters
- [ ] Conduct network resilience testing

---

### Procedure 4: DDoS Attack

**Severity:** MEDIUM to HIGH (based on impact)

#### Detection Indicators

- Massive spike in request volume
- API endpoints timing out
- Validator nodes unreachable
- High CPU/memory on all nodes
- Legitimate traffic being dropped
- Specific service degradation

#### Immediate Response Checklist

**Phase 1: Assess (First 10 minutes)**

- [ ] Confirm DDoS attack (vs. legitimate traffic spike)
- [ ] Identify attack vectors (HTTP, P2P, RPC)
- [ ] Measure attack volume and type
- [ ] Determine affected services
- [ ] Check if consensus affected
- [ ] Identify attack patterns

**Phase 2: Contain (10-30 minutes)**

- [ ] **Enable DDoS mitigation on CDN/WAF**

  ```bash
  # If using Cloudflare
  # Enable "I'm Under Attack" mode via dashboard

  # Configure rate limiting
  iptables -A INPUT -p tcp --dport 26657 -m limit \
    --limit 25/minute --limit-burst 100 -j ACCEPT
  ```

- [ ] Implement aggressive rate limiting
  ```bash
  # API endpoint rate limits
  # In nginx.conf
  limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
  limit_req zone=api burst=20 nodelay;
  ```
- [ ] Block malicious IP ranges
- [ ] Enable SYN cookies if SYN flood
  ```bash
  sysctl -w net.ipv4.tcp_syncookies=1
  ```
- [ ] Redirect traffic through DDoS protection
- [ ] Isolate critical infrastructure

**Phase 3: Investigate (30 minutes - 4 hours)**

- [ ] Analyze attack patterns

  ```bash
  # Analyze connections
  netstat -ntu | awk '{print $5}' | cut -d: -f1 | sort | uniq -c | sort -n

  # Check for SYN flood
  netstat -an | grep SYN_RECV | wc -l
  ```

- [ ] Identify attack sources
- [ ] Determine attack motivation
- [ ] Check for application-layer attacks
- [ ] Review firewall and access logs
- [ ] Coordinate with ISP/hosting provider

**Phase 4: Remediate (1-12 hours)**

- [ ] Fine-tune mitigation rules
- [ ] Implement additional filtering
- [ ] Blackhole malicious ASNs if needed
- [ ] Scale infrastructure if needed
- [ ] Deploy additional DDoS protection
- [ ] Optimize service performance

**Phase 5: Recover (2-24 hours)**

- [ ] Gradually remove aggressive filtering
- [ ] Monitor for attack resumption
- [ ] Verify all services operational
- [ ] Review and optimize mitigations
- [ ] Document attack characteristics

#### Attack Type Response Matrix

| Attack Type       | Primary Mitigation        | Secondary Mitigation       |
| ----------------- | ------------------------- | -------------------------- |
| HTTP Flood        | WAF/CDN rate limiting     | Challenge pages            |
| SYN Flood         | SYN cookies, firewall     | Increase backlog           |
| UDP Flood         | Rate limiting, filtering  | Null route                 |
| P2P Flood         | Peer limits, blacklisting | Network segmentation       |
| Application Layer | Smart rate limiting       | CAPTCHA, proof-of-work     |
| Volumetric        | ISP/transit filtering     | Anycast, scrubbing centers |

#### Communication Templates

**Status Update:**

```
SERVICE DEGRADATION - DDoS MITIGATION ACTIVE

We are currently mitigating a DDoS attack targeting our
infrastructure. You may experience:
- Slower API response times
- Occasional timeouts
- Challenge pages on website

Blockchain consensus is unaffected. Mitigation in progress.

Updates: status.paw.network
```

#### Post-Incident Actions

- [ ] Review and update DDoS protection
- [ ] Implement permanent rate limiting improvements
- [ ] Consider additional DDoS services
- [ ] Update incident playbooks
- [ ] Share attack intelligence
- [ ] Conduct resilience testing

---

### Procedure 5: Oracle Manipulation

**Severity:** HIGH or CRITICAL

#### Detection Indicators

- Price feed deviations >10% from other sources
- Single oracle source showing anomalies
- Large trades executed at manipulated prices
- Alert from oracle monitoring system
- User reports of incorrect prices
- Flash loan attacks coordinating with price manipulation

#### Immediate Response Checklist

**Phase 1: Assess (First 15 minutes)**

- [ ] Confirm price manipulation occurred
- [ ] Identify affected price feeds
- [ ] Compare oracle price vs. market price
- [ ] Check other oracle sources
- [ ] Identify manipulation mechanism
- [ ] Estimate financial impact
- [ ] Check for ongoing exploitation
- [ ] Notify Incident Commander and Smart Contract Lead

**Phase 2: Contain (15-30 minutes)**

- [ ] **Pause affected DEX pools**
  ```bash
  # Pause specific pool
  pawcli tx dex pause-pool <POOL_ID> \
    --from governance --gas auto
  ```
- [ ] Halt oracle price updates if needed
- [ ] Alert traders and market makers
- [ ] Monitor for continued manipulation
- [ ] Document affected transactions

**Phase 3: Investigate (30 minutes - 4 hours)**

- [ ] Analyze oracle data sources

  ```bash
  # Query oracle state
  pawcli query oracle price <ASSET_ID>

  # Check oracle update history
  pawcli query oracle updates <ASSET_ID> --height <HEIGHT>
  ```

- [ ] Review oracle update transactions
- [ ] Check oracle validator participation
- [ ] Identify manipulation methodology
- [ ] Determine if external or internal attack
- [ ] Review affected trades
- [ ] Calculate total losses

**Phase 4: Remediate (2-24 hours)**

- [ ] Fix oracle vulnerabilities
- [ ] Update price deviation thresholds
  ```bash
  # Update oracle params via governance
  pawcli tx gov submit-proposal param-change proposal.json
  ```
- [ ] Implement additional price sanity checks
- [ ] Add time-weighted average price (TWAP) checks
- [ ] Enable circuit breakers
- [ ] Review oracle validator set

**Phase 5: Recover (4-72 hours)**

- [ ] Deploy oracle improvements
- [ ] Verify price feeds accurate
- [ ] Unpause affected pools gradually
- [ ] Monitor for anomalies
- [ ] Process user claims if applicable
- [ ] Update oracle documentation

#### Oracle Health Checks

**Pre-flight checks before unpausing:**

- [ ] All oracle sources reporting
- [ ] Price deviation <2% from market
- [ ] Oracle validator participation >80%
- [ ] No single source outliers
- [ ] TWAP aligns with spot price
- [ ] Circuit breakers functional
- [ ] Monitoring alerts operational

#### Communication Templates

**Initial Alert:**

```
ORACLE PRICE FEED ANOMALY DETECTED

We have detected unusual price data in oracle feeds for
[asset names]. Affected DEX pools have been paused as a
precaution.

Impact: Trading temporarily halted for [pools]
User Funds: Secure
Cause: Under investigation

Status: Investigating
Next Update: [Time]
```

#### Post-Incident Actions

- [ ] Comprehensive oracle security audit
- [ ] Implement additional oracle sources
- [ ] Add TWAP and median price checks
- [ ] Enable automated circuit breakers
- [ ] Review oracle economic incentives
- [ ] Update oracle documentation

---

### Procedure 6: Private Key Compromise

**Severity:** CRITICAL

#### Detection Indicators

- Unauthorized transactions from critical accounts
- Governance proposals from unknown sources
- Unexpected validator key activity
- Alert from key management system
- Report from team member
- Suspicious contract upgrades

#### Immediate Response Checklist

**Phase 1: Assess (First 5 minutes)**

- [ ] Confirm key compromise
- [ ] Identify which key(s) compromised
- [ ] Determine key permissions/capabilities
- [ ] Check for unauthorized transactions
- [ ] Estimate funds at risk
- [ ] **CRITICAL:** Notify Incident Commander immediately

**Phase 2: Contain (5-20 minutes)**

- [ ] **Immediately freeze affected accounts if possible**
  ```bash
  # Via governance emergency action
  pawcli tx bank send <COMPROMISED_ACCOUNT> <SAFE_ACCOUNT> \
    --amount <ALL_FUNDS> --from emergency-multisig
  ```
- [ ] Revoke key permissions
- [ ] Alert exchanges to freeze addresses
- [ ] Monitor for additional unauthorized activity
- [ ] Isolate systems that had key access
- [ ] Change all related credentials

**Phase 3: Investigate (20 minutes - 2 hours)**

- [ ] Determine how key was compromised
- [ ] Review access logs for key usage
- [ ] Check all systems with key access
- [ ] Identify timeline of compromise
- [ ] Trace any unauthorized transactions
- [ ] Assess scope of damage
- [ ] Check for other compromised keys

**Phase 4: Remediate (1-8 hours)**

- [ ] Generate new keys in secure environment
  ```bash
  # Use hardware wallet or HSM
  # Generate offline in air-gapped environment
  pawd keys add new-key --keyring-backend file
  ```
- [ ] Update key permissions/assignments
- [ ] Implement key rotation
- [ ] Enhance key security measures
- [ ] Deploy HSM if not already used
- [ ] Update access control policies

**Phase 5: Recover (2-48 hours)**

- [ ] Transfer authority to new keys
- [ ] Verify new key security
- [ ] Update all systems with new keys
- [ ] Monitor for misuse of old keys
- [ ] Complete security review
- [ ] Update key management procedures

#### Key Type Response Matrix

| Key Type       | Impact             | Immediate Action   | Recovery Time |
| -------------- | ------------------ | ------------------ | ------------- |
| Validator      | Consensus risk     | Rotate immediately | 1-4 hours     |
| Governance     | Proposal risk      | Freeze, rotate     | 2-8 hours     |
| Treasury       | Fund theft         | Transfer funds out | Immediate     |
| Admin          | System access      | Revoke, rotate     | 1-4 hours     |
| Oracle         | Price manipulation | Pause oracles      | 1-2 hours     |
| Contract Owner | Contract control   | Transfer ownership | 2-8 hours     |

#### Communication Templates

**CRITICAL - Internal Only Initially:**

```
CRITICAL: PRIVATE KEY COMPROMISE

DO NOT share publicly yet.

Key Type: [Type]
Funds at Risk: [Amount]
Status: Containment in progress

Action Required:
- [Specific actions for team members]

Updates in #paw-critical
```

**Public (After Containment):**

```
SECURITY INCIDENT RESOLVED - KEY ROTATION

We detected unauthorized access to [key type - vague].
The issue has been contained and all sensitive keys
have been rotated.

Impact: [Specific impact]
User Action: None required
Status: Resolved

We take security seriously and will publish a detailed
post-mortem within 48 hours.
```

#### Post-Incident Actions

- [ ] Full security audit
- [ ] Implement hardware security modules (HSMs)
- [ ] Review key management procedures
- [ ] Mandatory security training
- [ ] Implement key rotation schedule
- [ ] Update access control policies
- [ ] Consider multi-party computation (MPC)

---

## Post-Incident Review

### Timeline

**Schedule:** Within 48 hours of incident resolution

**Duration:** 90-120 minutes

**Required Attendees:**

- Incident Commander
- All active incident responders
- Team leads from affected areas
- Technical writer (for documentation)

**Optional Attendees:**

- Executives
- External security consultants
- Affected users (for major incidents)

### Post-Mortem Template

#### 1. Incident Summary

**One-line summary:**
[Brief description of what happened]

**Incident Details:**

- **Detection Time:** [Timestamp]
- **Start Time:** [When incident actually began]
- **Containment Time:** [When contained]
- **Resolution Time:** [When fully resolved]
- **Duration:** [Total duration]
- **Severity:** [P0-P3]

#### 2. Impact Assessment

**Services Affected:**

- [List of affected services/systems]

**User Impact:**

- **Users Affected:** [Number/percentage]
- **Financial Impact:** [If applicable]
- **Availability Impact:** [% uptime lost]

**Business Impact:**

- [Description of business consequences]

#### 3. Timeline

[Detailed timeline of events]

```
YYYY-MM-DD HH:MM UTC - Event description
YYYY-MM-DD HH:MM UTC - Action taken
YYYY-MM-DD HH:MM UTC - Next event
...
```

#### 4. Root Cause Analysis

**What Happened:**
[Detailed technical explanation]

**Why It Happened:**
[Root causes - be honest]

**5 Whys Analysis:**

1. Why did the incident occur? [Answer]
2. Why did that happen? [Answer]
3. Why did that happen? [Answer]
4. Why did that happen? [Answer]
5. Why did that happen? [Answer - root cause]

#### 5. What Went Well

- [Things that worked as expected]
- [Effective responses]
- [Good decisions made]

#### 6. What Went Wrong

- [Failures or delays]
- [Ineffective responses]
- [Communication issues]
- [Tool/process gaps]

#### 7. Action Items

| Action               | Owner  | Priority | Due Date | Status |
| -------------------- | ------ | -------- | -------- | ------ |
| [Action description] | [Name] | [P0-P3]  | [Date]   | [ ]    |

**Categories:**

- **Prevent:** Stop this from happening again
- **Detect:** Detect faster next time
- **Respond:** Respond more effectively
- **Learn:** Training or documentation needs

#### 8. Lessons Learned

- [Key takeaways]
- [Process improvements]
- [Technical improvements]

### Blameless Culture

**Rules for Post-Mortems:**

- Focus on systems and processes, not individuals
- Assume everyone acted with best intentions
- Avoid language like "human error" - examine why the system allowed error
- Encourage transparency - no punishment for honest mistakes
- Learn and improve - that's the goal

---

## Contact Information

### Emergency Contacts

#### Incident Commander

- **Primary:** [Name]
  - Email: ic@paw.network
  - Phone: [REDACTED]
  - Signal: [REDACTED]

- **Backup:** [Name]
  - Email: ic-backup@paw.network
  - Phone: [REDACTED]

#### Security Lead

- **Primary:** [Name]
  - Email: security-lead@paw.network
  - Phone: [REDACTED]
  - PGP: [Key ID]

#### Validator Operations Lead

- **Primary:** [Name]
  - Email: validator-ops@paw.network
  - Phone: [REDACTED]

#### On-Call Schedule

- **Current On-Call:** See PagerDuty
- **PagerDuty:** incidents@paw.pagerduty.com
- **Escalation Policy:** https://paw.pagerduty.com/escalation_policies

### External Contacts

#### Security Researchers

- **Email:** security@paw.network
- **PGP Key:** [Public key or keybase link]
- **Bug Bounty:** https://bugcrowd.com/paw

#### Validators

- **Mailing List:** validators@paw.network
- **Telegram:** @PAWValidators
- **Discord:** #validators

#### Exchanges & Integrators

- **Email:** integrations@paw.network
- **Status Updates:** status.paw.network

#### Legal & Compliance

- **Legal Counsel:** legal@paw.network
- **Phone:** [REDACTED]

#### Infrastructure Providers

- **Hosting Provider:** [Provider] - [Support phone]
- **CDN/DDoS Protection:** [Provider] - [Support contact]
- **Monitoring Services:** [Provider] - [Support contact]

#### Law Enforcement (for criminal activity)

- **FBI Internet Crime Complaint Center:** https://www.ic3.gov
- **Local Law Enforcement:** [Contact info]

---

## Appendices

### Appendix A: Quick Reference Commands

#### Node Management

```bash
# Check node status
curl http://localhost:26657/status | jq

# Check peer count
curl http://localhost:26657/net_info | jq '.result.n_peers'

# Check if node is catching up
curl http://localhost:26657/status | jq '.result.sync_info.catching_up'

# Stop node
systemctl stop pawd

# Start node
systemctl start pawd

# View logs
journalctl -u pawd -f
```

#### Emergency Governance

```bash
# Submit emergency proposal
pawcli tx gov submit-proposal [proposal.json] \
  --from governance --gas auto

# Vote on proposal
pawcli tx gov vote [proposal-id] yes \
  --from validator --gas auto
```

#### Account Management

```bash
# Check account balance
pawcli query bank balances [address]

# Send funds (emergency transfer)
pawcli tx bank send [from] [to] [amount] --gas auto

# Check validator info
pawcli query staking validator [validator-addr]
```

### Appendix B: Monitoring Dashboards

- **Grafana:** http://monitoring.paw.network:3000
- **Prometheus:** http://monitoring.paw.network:9090
- **Alertmanager:** http://monitoring.paw.network:9093
- **Jaeger Tracing:** http://monitoring.paw.network:16686

### Appendix C: Useful Log Locations

```
/var/log/paw/node.log              - Main node logs
/var/log/paw/tendermint.log        - Consensus logs
/var/log/paw/dex.log               - DEX module logs
/var/log/paw/api.log               - API server logs
/var/log/paw/error.log             - Error logs
~/.paw/config/config.toml          - Node configuration
~/.paw/config/app.toml             - Application configuration
~/.paw/data/                       - Chain data
```

### Appendix D: Severity Assessment Questions

**Ask these questions to determine severity:**

1. Is consensus affected or at risk? → Likely CRITICAL
2. Are user funds at risk or lost? → Likely CRITICAL/HIGH
3. Is this being actively exploited? → Elevate one level
4. How many users/validators affected? → >50% = CRITICAL/HIGH
5. Can this escalate without intervention? → Elevate one level
6. Is there a workaround for users? → May lower one level
7. Is this visible to users? → If yes, at least MEDIUM

---

## Document Control

**Version History:**

| Version | Date       | Author        | Changes         |
| ------- | ---------- | ------------- | --------------- |
| 1.0     | 2025-11-14 | Security Team | Initial release |

**Review Schedule:** Quarterly or after major incidents

**Next Review Date:** 2026-02-14

**Approval:**

- CTO: ********\_******** Date: ****\_****
- Security Lead: ********\_******** Date: ****\_****

**Related Documents:**

- DISASTER_RECOVERY.md
- SECURITY_RUNBOOK.md
- SECURITY_TESTING.md
- TECHNICAL_SPECIFICATION.md

---

**This is a living document. If you identify gaps or improvements, please submit a pull request or contact the security team.**
