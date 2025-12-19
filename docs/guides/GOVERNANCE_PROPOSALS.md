# Governance Proposals Guide

Complete guide for submitting and managing governance proposals on the PAW blockchain.

## Overview

PAW uses on-chain governance for:
- Parameter changes across all modules
- Software upgrades
- IBC channel authorization
- Emergency actions (circuit breakers)
- Community pool spending
- Text proposals

## Quick Reference

| Proposal Type | Min Deposit | Voting Period | Quorum | Threshold |
|---------------|-------------|---------------|--------|-----------|
| Parameter Change | 1000 PAW | 3 days | 33.4% | 50% |
| Software Upgrade | 5000 PAW | 5 days | 33.4% | 66.7% |
| IBC Authorization | 1000 PAW | 3 days | 33.4% | 50% |
| Emergency Action | 10000 PAW | 1 day | 50% | 75% |
| Community Spend | 5000 PAW | 7 days | 33.4% | 50% |
| Text Proposal | 100 PAW | 3 days | 33.4% | 50% |

---

## Table of Contents

1. [Parameter Change Proposals](#1-parameter-change-proposals)
2. [Software Upgrade Proposals](#2-software-upgrade-proposals)
3. [IBC Channel Authorization](#3-ibc-channel-authorization)
4. [Emergency Actions](#4-emergency-actions)
5. [Proposal Submission Process](#5-proposal-submission-process)
6. [Voting Guide](#6-voting-guide)
7. [Proposal Lifecycle](#7-proposal-lifecycle)

---

## 1. Parameter Change Proposals

### 1.1 DEX Module Parameters

**Available Parameters**:

```json
{
  "swap_fee": "0.003",                    // 0.3% per swap
  "lp_fee": "0.0025",                     // 0.25% to LPs
  "protocol_fee": "0.0005",               // 0.05% to protocol
  "min_liquidity": "1000",                // Minimum initial liquidity
  "max_slippage_percent": "0.05",         // 5% maximum slippage
  "max_pool_drain_percent": "0.30",       // 30% max per swap
  "flash_loan_protection_blocks": "10",   // Flash loan detection window
  "authorized_channels": [],              // IBC channels
  "pool_creation_gas": "1000",            // Gas for pool creation
  "swap_validation_gas": "1500",          // Gas for swap validation
  "liquidity_gas": "1200"                 // Gas for liquidity ops
}
```

**Constraints**:

| Parameter | Type | Min | Max | Recommended |
|-----------|------|-----|-----|-------------|
| `swap_fee` | Dec | 0.001 | 0.01 | 0.003 (0.3%) |
| `lp_fee` | Dec | 0.0 | swap_fee | 0.0025 (83% of swap fee) |
| `protocol_fee` | Dec | 0.0 | swap_fee | 0.0005 (17% of swap fee) |
| `min_liquidity` | Int | 100 | 1000000 | 1000 |
| `max_slippage_percent` | Dec | 0.01 | 0.50 | 0.05 (5%) |
| `max_pool_drain_percent` | Dec | 0.10 | 0.50 | 0.30 (30%) |
| `flash_loan_protection_blocks` | Int | 1 | 100 | 10 |
| `max_pool_drain_percent` | Dec | 0.10 | 0.50 | 0.30 |
| `pool_creation_cooldown` | Int | 10 | 10000 | 100 |
| `max_pools_per_address` | Int | 1 | 100 | 10 |
| `min_pool_creation_deposit` | Int | 10_000_000 | 1_000_000_000_000 | 100_000_000 |

**Example Proposal** - Increase swap fee to 0.5%:

```json
{
  "title": "Increase DEX Swap Fee to 0.5%",
  "description": "Increase swap fee from 0.3% to 0.5% to improve protocol revenue and compensate LPs for higher volatility.\n\nRationale:\n- Current 0.3% fee is below market average (Uniswap V2: 0.3%, V3: 0.05-1%)\n- Higher fee will increase LP yields, attracting more liquidity\n- Additional revenue will fund development and security audits\n\nImpact Analysis:\n- Daily swap volume: ~$1M\n- Current daily fees: $3,000\n- Projected daily fees: $5,000 (+$2,000)\n- LP revenue increase: ~66%\n\nVote YES to increase fees, NO to keep current rate.",
  "changes": [
    {
      "subspace": "dex",
      "key": "SwapFee",
      "value": "\"0.005\""
    }
  ],
  "deposit": "1000000000upaw"
}
```

### 1.1.1 Timelock Workflow

- All DEX parameter changes must schedule via `MsgScheduleDexParamChange`.
- Minimum timelock: 10,000 blocks (approx 17h); proposals failing to respect this are rejected.
- CLI helper:
  ```bash
  pawd tx dex schedule-param-change param.json --apply-height 1234567 ...
  ```
- View pending changes: `pawd query dex pending-params`.

**CLI Command**:
```bash
pawd tx gov submit-proposal param-change dex-swap-fee.json \
  --from proposer \
  --chain-id paw-1 \
  --gas auto \
  --gas-adjustment 1.5
```

---

### 1.2 Oracle Module Parameters

**Available Parameters**:

```json
{
  "vote_period": "30",                          // Blocks between aggregations
  "vote_threshold": "0.67",                     // 67% consensus required
  "slash_fraction": "0.01",                     // 1% stake slashed
  "slash_window": "10000",                      // Blocks for slash window
  "min_valid_per_window": "100",                // Min valid votes per window
  "twap_lookback_window": "1000",               // Blocks for TWAP
  "authorized_channels": [],                    // IBC channels
  "allowed_regions": ["global", "na", "eu", "apac", "latam", "africa"],
  "min_geographic_regions": "1",                // Min regions represented
  "min_voting_power_for_consensus": "0.10",     // 10% min voting power
  "max_validators_per_ip": "3",                 // Max validators per IP
  "max_validators_per_asn": "5"                 // Max validators per ASN
}
```

**Constraints**:

| Parameter | Type | Min | Max | Recommended |
|-----------|------|-----|-----|-------------|
| `vote_period` | Int | 1 | 3600 | 30 blocks (~2.5 min) |
| `vote_threshold` | Dec | 0.50 | 1.00 | 0.67 (Byzantine threshold) |
| `slash_fraction` | Dec | 0.00 | 1.00 | 0.01 (1%) |
| `slash_window` | Int | 100 | 100000 | 10000 blocks (~21 hours) |
| `min_valid_per_window` | Int | 1 | slash_window | 100 |
| `twap_lookback_window` | Int | 10 | 10000 | 1000 blocks (~2 hours) |
| `min_voting_power_for_consensus` | Dec | 0.01 | 0.50 | 0.10 (10%) |
| `max_validators_per_ip` | Int | 1 | 10 | 3 |
| `max_validators_per_asn` | Int | 1 | 20 | 5 |

**Example Proposal** - Increase vote threshold for higher security:

```json
{
  "title": "Increase Oracle Vote Threshold to 75%",
  "description": "Increase oracle consensus threshold from 67% to 75% for enhanced security.\n\nRationale:\n- Current 67% provides Byzantine fault tolerance (BFT)\n- 75% provides additional safety margin\n- Reduces risk of price manipulation by colluding validators\n- Industry standard for high-value oracles (Chainlink, Band Protocol)\n\nRisks:\n- Slightly higher chance of aggregation failure if validators offline\n- Requires more validators to participate\n\nMitigation:\n- Monitor validator uptime before implementation\n- Grace period: 7 days after approval\n- Revert option if < 90% aggregation success rate\n\nVote YES for enhanced security, NO to maintain current threshold.",
  "changes": [
    {
      "subspace": "oracle",
      "key": "VoteThreshold",
      "value": "\"0.75\""
    }
  ],
  "deposit": "1000000000upaw"
}
```

---

### 1.3 Compute Module Parameters

**Available Parameters**:

```json
{
  "min_provider_stake": "1000000",              // 1 PAW minimum stake
  "verification_timeout_seconds": "300",        // 5 minutes
  "max_request_timeout_seconds": "3600",        // 1 hour max
  "reputation_slash_percentage": "10",          // 10% reputation slash
  "stake_slash_percentage": "1",                // 1% stake slash
  "min_reputation_score": "50",                 // Minimum 50/100 score
  "escrow_release_delay_seconds": "3600",       // 1 hour delay
  "authorized_channels": [],                    // IBC channels
  "nonce_retention_blocks": "17280"             // ~24 hours
}
```

**Governance Parameters** (dispute resolution):

```json
{
  "dispute_deposit": "1000000",                 // 1 PAW to dispute
  "evidence_period_seconds": "86400",           // 24 hours evidence
  "voting_period_seconds": "86400",             // 24 hours voting
  "quorum_percentage": "0.334",                 // 33.4% quorum
  "consensus_threshold": "0.5",                 // 50% to pass
  "slash_percentage": "0.1",                    // 10% slash on fraud
  "appeal_deposit_percentage": "0.05",          // 5% for appeal
  "max_evidence_size": "10485760"               // 10 MB max
}
```

**Constraints**:

| Parameter | Type | Min | Max | Recommended |
|-----------|------|-----|-----|-------------|
| `min_provider_stake` | Int | 100000 | 1000000000 | 1000000 (1 PAW) |
| `verification_timeout_seconds` | Int | 60 | 3600 | 300 (5 min) |
| `max_request_timeout_seconds` | Int | 300 | 86400 | 3600 (1 hour) |
| `reputation_slash_percentage` | Int | 1 | 50 | 10 (10%) |
| `stake_slash_percentage` | Int | 1 | 100 | 1 (1%) |
| `min_reputation_score` | Int | 0 | 100 | 50 |
| `escrow_release_delay_seconds` | Int | 0 | 86400 | 3600 (1 hour) |
| `dispute_deposit` | Int | 100000 | 100000000 | 1000000 |
| `quorum_percentage` | Dec | 0.20 | 0.75 | 0.334 (33.4%) |
| `consensus_threshold` | Dec | 0.50 | 1.00 | 0.5 (50%) |

**Example Proposal** - Reduce provider barrier to entry:

```json
{
  "title": "Reduce Minimum Compute Provider Stake to 0.5 PAW",
  "description": "Lower minimum stake requirement from 1 PAW to 0.5 PAW to increase provider diversity.\n\nRationale:\n- Current 1 PAW (~$50 at launch) is barrier for small providers\n- More providers = better decentralization and availability\n- Other networks (Akash, Golem) have lower minimums\n\nRisks:\n- Potential increase in low-quality providers\n- Higher spam/abuse risk\n\nMitigation:\n- Maintain reputation system (min 50/100 score)\n- Slash percentage remains 1% (now 0.005 PAW)\n- Monitor provider quality metrics post-implementation\n\nImpact:\n- Expected 2-3x increase in provider count\n- Geographic diversity improvement\n- Lower compute costs due to competition\n\nVote YES to reduce barrier, NO to maintain current requirement.",
  "changes": [
    {
      "subspace": "compute",
      "key": "MinProviderStake",
      "value": "\"500000\""
    }
  ],
  "deposit": "1000000000upaw"
}
```

---

## 2. Software Upgrade Proposals

### 2.1 Standard Upgrade

**Template**:

```json
{
  "title": "PAW v2.0.0 Upgrade - Advanced DEX Features",
  "description": "Upgrade PAW blockchain to v2.0.0, introducing concentrated liquidity and multi-hop routing.\n\n## New Features\n\n### 1. Concentrated Liquidity (Uniswap V3 style)\n- Liquidity providers can specify price ranges\n- Up to 4000x capital efficiency\n- Custom fee tiers (0.05%, 0.3%, 1%)\n\n### 2. Multi-Hop Routing\n- Optimal path finding across multiple pools\n- Gas-efficient batch swaps\n- Price impact minimization\n\n### 3. Oracle Enhancements\n- Multi-method TWAP (5 algorithms)\n- Enhanced outlier detection\n- Cross-chain price aggregation\n\n## Testing\n\n- Testnet running since: 2024-01-15\n- Total test transactions: 1,000,000+\n- Critical bugs found: 0\n- External audit: Trail of Bits (completed 2024-02-01)\n\n## Upgrade Plan\n\n- **Proposal Date**: 2024-02-15\n- **Voting Period**: 5 days\n- **Upgrade Height**: 1,000,000 (estimated 2024-02-25 14:00 UTC)\n- **Downtime**: ~30 minutes expected\n\n## Validator Instructions\n\n```bash\n# Stop node at upgrade height\nsudo systemctl stop pawd\n\n# Backup data\ntar -czf paw-pre-v2-backup.tar.gz ~/.paw/data\n\n# Install v2.0.0\nwget https://github.com/paw-chain/paw/releases/download/v2.0.0/pawd\nchmod +x pawd\nsudo mv pawd /usr/local/bin/\n\n# Verify version\npawd version\n# Should show: 2.0.0\n\n# Restart with auto-upgrade\ncosmovisor run start\n```\n\n## Rollback Plan\n\nIf critical issues found within 24 hours:\n1. Emergency governance proposal to halt network\n2. Rollback to v1.x using backup\n3. Investigate issues on testnet\n4. Re-propose with fixes\n\n## References\n\n- GitHub Release: https://github.com/paw-chain/paw/releases/tag/v2.0.0\n- Audit Report: https://github.com/paw-chain/audits/blob/main/trail-of-bits-v2.pdf\n- Migration Guide: https://docs.paw.network/upgrades/v2-migration\n- Testnet Explorer: https://testnet.explorer.paw.network\n\nVote YES to upgrade, NO to delay.",
  "plan": {
    "name": "v2.0.0",
    "height": "1000000",
    "info": "https://github.com/paw-chain/paw/releases/tag/v2.0.0"
  },
  "deposit": "5000000000upaw"
}
```

**CLI Command**:
```bash
pawd tx gov submit-proposal software-upgrade v2.0.0 \
  --title "PAW v2.0.0 Upgrade" \
  --description "$(cat upgrade-description.txt)" \
  --upgrade-height 1000000 \
  --upgrade-info "https://github.com/paw-chain/paw/releases/tag/v2.0.0" \
  --deposit 5000000000upaw \
  --from proposer \
  --chain-id paw-1
```

---

### 2.2 Emergency Upgrade (Critical Bug)

**Fast-Track Process**:

```json
{
  "title": "EMERGENCY: Critical Security Patch v1.0.1",
  "description": "⚠️ CRITICAL SECURITY FIX - DO NOT DELAY ⚠️\n\n## Vulnerability\n\nA critical vulnerability was discovered in the DEX module that allows:\n- Unauthorized pool drainage via reentrancy\n- Potential loss of user funds\n\n## Severity\n\n- **CVSS Score**: 9.8 (Critical)\n- **Exploitability**: High\n- **Impact**: Complete loss of DEX liquidity\n- **Detection**: Not yet exploited in the wild\n\n## Fix\n\nVersion 1.0.1 implements:\n- Reentrancy guard on all DEX functions\n- Mutex locks on pool state changes\n- Enhanced input validation\n\n## Immediate Actions\n\n1. **DO NOT use DEX** until upgrade complete\n2. Circuit breaker activated automatically\n3. All pools locked to prevent exploitation\n\n## Upgrade Timeline\n\n- **Proposal**: Immediate (2024-02-15 18:00 UTC)\n- **Voting**: 1 day (emergency)\n- **Upgrade Height**: 500,100 (2024-02-16 20:00 UTC)\n- **Expected Downtime**: 15 minutes\n\n## Validator Actions Required\n\n```bash\n# Immediate: Pull latest code\ngit pull origin main\ngit checkout v1.0.1\n\n# Build\nmake install\n\n# Verify\npawd version  # Should show 1.0.1\n\n# Restart with cosmovisor for auto-upgrade\nsudo systemctl restart cosmovisor\n```\n\n## Disclosure\n\n- **Reported by**: Anonymous security researcher\n- **Bounty**: 50,000 PAW paid\n- **Public disclosure**: After 90% validator upgrade\n\n## References\n\n- Security Advisory: https://github.com/paw-chain/paw/security/advisories/GHSA-xxxx\n- Patch Diff: https://github.com/paw-chain/paw/compare/v1.0.0...v1.0.1\n\n**Vote YES immediately to protect user funds.**",
  "plan": {
    "name": "v1.0.1-emergency",
    "height": "500100",
    "info": "https://github.com/paw-chain/paw/releases/tag/v1.0.1"
  },
  "deposit": "10000000000upaw"
}
```

**Emergency Parameters**:
- Voting period: 1 day (vs standard 5 days)
- Deposit: 10,000 PAW (vs standard 5,000 PAW)
- Quorum: 50% (vs standard 33.4%)
- Threshold: 75% (vs standard 66.7%)

---

## 3. IBC Channel Authorization

### 3.1 Authorizing New Channel

All modules use the same IBC authorization pattern.

**Example** - Authorize Osmosis DEX channel:

```json
{
  "title": "Authorize IBC Channel: PAW <-> Osmosis DEX",
  "description": "Authorize IBC channel for cross-chain DEX operations with Osmosis.\n\n## Channel Details\n\n- **Counterparty Chain**: Osmosis (osmosis-1)\n- **Port**: dex\n- **Channel ID**: channel-42\n- **Counterparty Channel**: channel-123\n- **Connection**: connection-15\n- **State**: OPEN\n\n## Verification\n\n```bash\n# Query channel state\npawd query ibc channel end dex channel-42\n\n# Verify counterparty\npawd query ibc channel client-state dex channel-42\n```\n\n## Use Cases\n\n1. **Cross-Chain Swaps**: Trade PAW tokens for OSMO\n2. **Liquidity Migration**: Move liquidity between chains\n3. **Arbitrage**: Price discovery across DEXs\n\n## Security Considerations\n\n- Channel handshake completed successfully\n- Relayer: Hermes (v1.13.2) running on trusted infrastructure\n- Rate limiting: 1000 packets/hour\n- Timeout: 10 minutes per packet\n\n## Testing\n\n- Testnet channel active since: 2024-01-01\n- Total test transfers: 5,000+\n- Failed transfers: 0%\n- Average latency: 45 seconds\n\n## Risks\n\n- Counterparty chain compromise\n- Relayer failure (mitigated by multiple relayers)\n- IBC protocol vulnerabilities\n\n## Monitoring\n\n- Relayer uptime: 99.9% SLA\n- Alert on packet timeout\n- Daily balance reconciliation\n\nVote YES to authorize, NO to reject.",
  "changes": [
    {
      "subspace": "dex",
      "key": "AuthorizedChannels",
      "value": "[{\"port_id\":\"dex\",\"channel_id\":\"channel-42\"}]"
    }
  ],
  "deposit": "1000000000upaw"
}
```

**CLI Command**:
```bash
pawd tx gov submit-proposal param-change authorize-osmosis-channel.json \
  --from proposer \
  --chain-id paw-1
```

---

### 3.2 Multi-Module IBC Authorization

**Example** - Authorize channel for DEX, Oracle, and Compute:

```json
{
  "title": "Authorize Multi-Module IBC Channel: PAW <-> Cosmos Hub",
  "description": "Authorize IBC channel for DEX, Oracle, and Compute modules with Cosmos Hub.\n\n## Modules Affected\n\n1. **DEX Module**: Cross-chain token swaps\n2. **Oracle Module**: Price feed aggregation from Cosmos Hub validators\n3. **Compute Module**: Distributed computation requests\n\n## Channel Details\n\n- **Port IDs**: dex, oracle, compute\n- **Channel ID**: channel-0 (all modules)\n- **Counterparty**: Cosmos Hub (cosmoshub-4)\n\n## Authorization\n\nThis proposal authorizes all three modules simultaneously to avoid multiple governance rounds.",
  "changes": [
    {
      "subspace": "dex",
      "key": "AuthorizedChannels",
      "value": "[{\"port_id\":\"dex\",\"channel_id\":\"channel-0\"}]"
    },
    {
      "subspace": "oracle",
      "key": "AuthorizedChannels",
      "value": "[{\"port_id\":\"oracle\",\"channel_id\":\"channel-0\"}]"
    },
    {
      "subspace": "compute",
      "key": "AuthorizedChannels",
      "value": "[{\"port_id\":\"compute\",\"channel_id\":\"channel-0\"}]"
    }
  ],
  "deposit": "1000000000upaw"
}
```

---

## 4. Emergency Actions

### 4.1 Circuit Breaker Activation

**Manual Override** (normally automatic):

```bash
# DEX circuit breaker (via governance)
pawd tx gov submit-proposal param-change dex-circuit-breaker.json

# Or direct message (requires authority)
pawd tx dex activate-circuit-breaker \
  --reason "Unusual swap volume spike detected" \
  --from authority
```

**Proposal Template**:

```json
{
  "title": "EMERGENCY: Activate DEX Circuit Breaker",
  "description": "Immediately pause all DEX operations due to suspected exploit.\n\n## Trigger Event\n\n- Time: 2024-02-15 14:32:15 UTC\n- Event: Unusual swap pattern detected\n- Pool ID: 5 (PAW/USDC)\n- Volume: $10M in 5 minutes (normal: $100K/hour)\n- Price Impact: 45% (threshold: 5%)\n\n## Analysis\n\n- Pattern consistent with flash loan attack\n- Multiple large swaps from new addresses\n- Price manipulation suspected\n\n## Actions Taken\n\n1. Automatic circuit breaker triggered\n2. All DEX operations paused\n3. Forensic analysis initiated\n\n## Recovery Plan\n\n1. Investigate transaction patterns (24 hours)\n2. Identify exploit vector\n3. Deploy patch if needed\n4. Resume operations after validation\n\nVote YES to maintain pause, NO to resume immediately (not recommended).",
  "changes": [
    {
      "subspace": "dex",
      "key": "CircuitBreakerActive",
      "value": "true"
    }
  ],
  "deposit": "10000000000upaw"
}
```

---

### 4.2 Oracle Emergency Halt

**When to Use**:
- Sybil attack detected
- >50% validators compromised
- Data poisoning confirmed

**Proposal**:

```json
{
  "title": "EMERGENCY: Halt Oracle Price Feeds",
  "description": "Suspend oracle aggregation due to suspected Sybil attack.\n\n## Incident\n\n- 20 new validators appeared simultaneously\n- All submitting identical prices\n- All from same ASN (AS12345)\n- Voting power: 15% (above min threshold)\n\n## Risk\n\nIf these validators are controlled by single entity:\n- Price manipulation possible\n- DEX arbitrage exploitation\n- User fund loss\n\n## Immediate Actions\n\n1. Halt oracle aggregation\n2. Freeze DEX oracle integration\n3. Investigate validator identities\n4. Implement geographic diversity checks\n\n## Resolution\n\n- Manual price feeds from trusted validators\n- Resume after validator verification\n- Update params: `max_validators_per_asn: 2`\n\nVote YES to halt (protect users), NO to continue (risky).",
  "changes": [
    {
      "subspace": "oracle",
      "key": "CircuitBreakerActive",
      "value": "true"
    }
  ],
  "deposit": "10000000000upaw"
}
```

---

## 5. Proposal Submission Process

### 5.1 Preparation Checklist

- [ ] Clear title (< 80 characters)
- [ ] Detailed description with rationale
- [ ] Impact analysis (technical, economic, social)
- [ ] Testing evidence (testnet, audit, etc.)
- [ ] Risk assessment and mitigation
- [ ] Rollback plan (for upgrades)
- [ ] Community discussion (forum, Discord)
- [ ] Sufficient deposit (see table above)

---

### 5.2 Submission Steps

**Step 1: Create proposal JSON**

```bash
cat > proposal.json <<EOF
{
  "title": "Your Proposal Title",
  "description": "Detailed description...",
  "changes": [...],
  "deposit": "1000000000upaw"
}
EOF
```

**Step 2: Validate JSON**

```bash
jq . proposal.json  # Verify valid JSON
pawd tx gov submit-proposal param-change proposal.json --dry-run
```

**Step 3: Submit**

```bash
pawd tx gov submit-proposal param-change proposal.json \
  --from proposer \
  --chain-id paw-1 \
  --gas auto \
  --gas-adjustment 1.5 \
  --fees 1000upaw
```

**Step 4: Note proposal ID**

```bash
# From transaction output
"proposal_id": "42"
```

**Step 5: Announce**

Share on:
- Commonwealth forum
- Discord #governance channel
- Twitter/X
- Telegram

---

## 6. Voting Guide

### 6.1 How to Vote

**Query active proposals**:

```bash
pawd query gov proposals --status voting_period
```

**Get proposal details**:

```bash
pawd query gov proposal 42
```

**Vote**:

```bash
pawd tx gov vote 42 yes \
  --from validator \
  --chain-id paw-1

# Options: yes, no, no_with_veto, abstain
```

**Delegate vote**:

```bash
# Your vote weight = your delegated stake
# If you delegate to validator, they vote on your behalf
# You can override by voting yourself
```

---

### 6.2 Vote Options Explained

| Option | Meaning | When to Use |
|--------|---------|-------------|
| **Yes** | Support proposal | You agree with the change |
| **No** | Oppose proposal | You disagree but proposal is valid |
| **No with Veto** | Reject AND burn deposit | Proposal is spam or malicious |
| **Abstain** | No opinion | You want to contribute to quorum without taking a stance |

**Veto Power**:
- If >33.3% vote "No with Veto", proposal is rejected and deposit burned
- Use sparingly for spam/malicious proposals

---

### 6.3 Voting Strategies

**For Validators**:

```bash
# 1. Read full proposal
pawd query gov proposal 42 --output json | jq .content

# 2. Test changes on testnet
pawd tx gov vote 42 yes --from testnet-validator --chain-id paw-testnet-1

# 3. Discuss with community
# Discord, Twitter, Commonwealth

# 4. Vote on mainnet
pawd tx gov vote 42 yes --from validator --chain-id paw-1

# 5. Announce vote with reasoning
echo "Voted YES on #42 because [reasoning]" | tweet
```

**For Delegators**:

```bash
# Check your validator's vote
pawd query gov vote 42 <validator-address>

# Override if you disagree
pawd tx gov vote 42 no --from delegator
```

---

## 7. Proposal Lifecycle

### 7.1 States

```
[Draft] → [Deposit Period] → [Voting Period] → [Passed/Rejected/Failed]
```

**Deposit Period** (2 weeks):
- Proposal submitted
- Collect minimum deposit (1000 PAW for param changes)
- If deposit reached → voting period
- If not reached in 2 weeks → proposal deleted

**Voting Period** (varies by type):
- Validators and delegators vote
- Quorum: 33.4% of bonded stake must vote
- Threshold: 50% of votes must be "Yes" (excluding abstain)
- Veto: If >33.3% vote "No with Veto", proposal rejected

**Execution**:
- If passed: changes applied automatically
- If rejected: no changes, deposit returned
- If vetoed: deposit burned

---

### 7.2 Monitoring Proposal

**Query status**:

```bash
# Proposal details
pawd query gov proposal 42

# Vote tally
pawd query gov tally 42

# Your vote
pawd query gov vote 42 <your-address>
```

**Example Tally**:

```json
{
  "yes": "50000000000",      // 50,000 PAW
  "abstain": "10000000000",  // 10,000 PAW
  "no": "20000000000",       // 20,000 PAW
  "no_with_veto": "5000000000"  // 5,000 PAW
}
```

**Calculation**:

```
Total Votes = 85,000 PAW
Bonded Stake = 100,000 PAW
Quorum = 85,000 / 100,000 = 85% ✓ (> 33.4%)

Yes = 50,000 / (50,000 + 20,000 + 5,000) = 66.7% ✓ (> 50%)
Veto = 5,000 / 85,000 = 5.9% ✓ (< 33.3%)

Result: PASSED
```

---

## Best Practices

### For Proposers

1. **Start with discussion**: Forum post 1 week before proposal
2. **Provide evidence**: Testnet results, audits, simulations
3. **Be transparent**: Disclose potential conflicts of interest
4. **Engage community**: Answer questions, address concerns
5. **Plan timing**: Avoid holiday periods
6. **Budget deposit**: Have 2x minimum in case of revisions

### For Voters

1. **Do your research**: Read full proposal and discussion
2. **Test if possible**: Run changes on testnet node
3. **Consider impact**: Technical, economic, social effects
4. **Vote consistently**: Your voting record is public
5. **Explain reasoning**: Help community understand your position
6. **Monitor execution**: Verify changes applied correctly

### For Validators

1. **Vote on everything**: Don't abstain unless truly uncertain
2. **Communicate votes**: Tweet/announce voting decision
3. **Lead by example**: Thorough analysis shows due diligence
4. **Upgrade promptly**: For software upgrades, update within 24h
5. **Monitor effects**: Track metrics post-implementation

---

## Common Scenarios

### Scenario 1: Fee Change Proposal

**Problem**: DEX fees too low, LPs leaving for higher yields

**Solution**:
```json
{
  "title": "Increase DEX LP Fee to 0.28%",
  "changes": [{
    "subspace": "dex",
    "key": "LpFee",
    "value": "\"0.0028\""
  }]
}
```

**Timeline**: 3 days voting, instant execution

---

### Scenario 2: Add New IBC Channel

**Problem**: Want to enable cross-chain swaps with new DEX

**Solution**:
```json
{
  "title": "Authorize Astroport Channel",
  "changes": [{
    "subspace": "dex",
    "key": "AuthorizedChannels",
    "value": "[{\"port_id\":\"dex\",\"channel_id\":\"channel-5\"}]"
  }]
}
```

**Prerequisites**:
- IBC channel handshake complete
- Relayer running
- Testnet validation

---

### Scenario 3: Emergency Security Patch

**Problem**: Critical vulnerability discovered

**Solution**:
```json
{
  "title": "EMERGENCY: Security Patch v1.0.1",
  "plan": {
    "name": "v1.0.1",
    "height": "<current_height + 1000>",
    "info": "https://github.com/..."
  }
}
```

**Timeline**: 1 day voting (emergency), 2 hours to upgrade

---

## Related Documentation

- [Parameter Reference](../PARAMETER_REFERENCE.md)
- [Error Codes](../api/guides/ERROR_CODES_REFERENCE.md)
- [Cross-Module Integration](../implementation/CROSS_MODULE_INTEGRATION.md)

---

## Support

**Need help with proposals?**

- Discord: #governance channel
- Commonwealth: https://commonwealth.im/paw
- GitHub Discussions: https://github.com/paw-chain/paw/discussions
- Email: governance@paw.network
