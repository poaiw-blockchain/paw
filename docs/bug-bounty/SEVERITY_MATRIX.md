# Bug Bounty Severity Assessment Matrix

## Overview

This document provides detailed guidance on assessing the severity of security vulnerabilities in the PAW blockchain. We use a combination of CVSS 3.1 scoring and impact-based assessment to determine severity levels and corresponding rewards.

## Severity Framework

### Assessment Dimensions

Security vulnerabilities are evaluated across two primary dimensions:

1. **Impact**: The potential damage if the vulnerability is exploited
2. **Likelihood**: The probability and ease of exploitation

### Impact Categories

- **Critical Impact**: Loss of funds, consensus failure, complete system compromise
- **High Impact**: Significant service disruption, privilege escalation, major data breach
- **Medium Impact**: Limited service disruption, partial data exposure, minor fund impact
- **Low Impact**: Minimal security impact, best practice violations

### Likelihood Factors

- **Attack Complexity**: How difficult is the attack to execute?
- **Privileges Required**: What level of access is needed?
- **User Interaction**: Does the attack require user action?
- **Attack Vector**: Can it be exploited remotely or requires local access?

## Impact vs. Likelihood Matrix

| Impact / Likelihood | High Likelihood | Medium Likelihood | Low Likelihood |
| ------------------- | --------------- | ----------------- | -------------- |
| **Critical Impact** | CRITICAL        | CRITICAL          | HIGH           |
| **High Impact**     | CRITICAL        | HIGH              | HIGH           |
| **Medium Impact**   | HIGH            | MEDIUM            | MEDIUM         |
| **Low Impact**      | MEDIUM          | LOW               | LOW            |

### Likelihood Scoring

**High Likelihood**:

- Low attack complexity (easily exploitable)
- No privileges required or low privileges
- No user interaction required
- Network accessible (remote exploitation)

**Medium Likelihood**:

- Medium attack complexity
- Some privileges required
- Minimal user interaction
- Adjacent network access required

**Low Likelihood**:

- High attack complexity
- High privileges required
- Significant user interaction required
- Local access required
- Specific environmental conditions

## CVSS 3.1 Scoring Guide

We use CVSS 3.1 as a standardized framework for severity assessment.

### CVSS Score Ranges

| Severity | CVSS Score | Reward Range       |
| -------- | ---------- | ------------------ |
| Critical | 9.0 - 10.0 | $25,000 - $100,000 |
| High     | 7.0 - 8.9  | $10,000 - $25,000  |
| Medium   | 4.0 - 6.9  | $2,500 - $10,000   |
| Low      | 0.1 - 3.9  | $500 - $2,500      |

### CVSS Metric Groups

#### Base Metrics (Primary Factors)

**Attack Vector (AV)**:

- Network (N): Remotely exploitable via network - Most severe
- Adjacent (A): Requires local network access
- Local (L): Requires local system access
- Physical (P): Requires physical access - Least severe

**Attack Complexity (AC)**:

- Low (L): No special conditions, easily repeatable - Most severe
- High (H): Requires specific conditions or timing - Less severe

**Privileges Required (PR)**:

- None (N): No authentication needed - Most severe
- Low (L): Basic user privileges
- High (H): Administrator/validator privileges - Least severe

**User Interaction (UI)**:

- None (N): No user action required - Most severe
- Required (R): Requires user to perform action - Less severe

**Scope (S)**:

- Changed (C): Affects resources beyond vulnerable component - Most severe
- Unchanged (U): Limited to vulnerable component - Less severe

#### Impact Metrics

**Confidentiality Impact (C)**:

- High (H): Complete information disclosure
- Low (L): Some information disclosed
- None (N): No information disclosure

**Integrity Impact (I)**:

- High (H): Complete integrity compromise
- Low (L): Limited integrity impact
- None (N): No integrity impact

**Availability Impact (A)**:

- High (H): Complete service denial
- Low (L): Degraded performance
- None (N): No availability impact

### CVSS Calculator

Use the official CVSS 3.1 calculator: https://www.first.org/cvss/calculator/3.1

## Severity Definitions with Examples

### Critical Severity (CVSS 9.0-10.0)

#### Criteria

- **Direct loss of funds** from protocol or users
- **Consensus failure** causing network halt or permanent fork
- **Complete system compromise** with remote code execution
- **Unlimited token minting/burning** bypassing protocol limits
- **Private key extraction** from validators or users

#### Typical CVSS Vector

`CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H` (Score: 10.0)

#### Examples

1. **Double-Spend via Consensus Bypass**
   - **Description**: Vulnerability allowing transaction to be included in multiple blocks
   - **Impact**: Direct fund theft, consensus failure
   - **CVSS Vector**: `AV:N/AC:L/PR:N/UI:N/S:C/C:N/I:H/A:H`
   - **Score**: 10.0
   - **Reward**: $75,000 - $100,000

2. **Private Key Recovery from Validator Signatures**
   - **Description**: Cryptographic flaw allowing private key extraction from signatures
   - **Impact**: Complete validator compromise, fund theft
   - **CVSS Vector**: `AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H`
   - **Score**: 10.0
   - **Reward**: $80,000 - $100,000

3. **Unlimited Token Minting via Module Bypass**
   - **Description**: Authentication bypass in bank module allowing arbitrary minting
   - **Impact**: Complete economic collapse, total value loss
   - **CVSS Vector**: `AV:N/AC:L/PR:N/UI:N/S:C/C:N/I:H/A:H`
   - **Score**: 10.0
   - **Reward**: $75,000 - $100,000

4. **DEX Liquidity Pool Drainage via Flash Loan**
   - **Description**: Flash loan exploit allowing complete pool drainage
   - **Impact**: Total loss of liquidity pool funds
   - **CVSS Vector**: `AV:N/AC:L/PR:N/UI:N/S:C/C:N/I:H/A:H`
   - **Score**: 9.8
   - **Reward**: $50,000 - $75,000

5. **Consensus Breaking State Transition**
   - **Description**: Invalid state transition accepted by consensus
   - **Impact**: Permanent chain fork, network split
   - **CVSS Vector**: `AV:N/AC:L/PR:N/UI:N/S:C/C:N/I:H/A:H`
   - **Score**: 9.5
   - **Reward**: $60,000 - $80,000

6. **Remote Code Execution on Validator Nodes**
   - **Description**: RPC vulnerability allowing arbitrary code execution
   - **Impact**: Complete node compromise, network takeover
   - **CVSS Vector**: `AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H`
   - **Score**: 10.0
   - **Reward**: $75,000 - $100,000

### High Severity (CVSS 7.0-8.9)

#### Criteria

- **Significant fund loss** (limited scope or conditions)
- **Authentication bypass** for privileged operations
- **Network-wide DoS** affecting all nodes
- **Oracle manipulation** enabling profitable attacks
- **Incorrect slashing** penalizing honest validators
- **Privilege escalation** in governance or consensus

#### Typical CVSS Vector

`CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H` (Score: 8.8)

#### Examples

1. **Oracle Price Feed Manipulation**
   - **Description**: Ability to submit false price data affecting DEX
   - **Impact**: Profitable arbitrage, limited fund loss
   - **CVSS Vector**: `AV:N/AC:H/PR:L/UI:N/S:C/C:N/I:H/A:N`
   - **Score**: 7.7
   - **Reward**: $15,000 - $20,000

2. **Validator Privilege Escalation**
   - **Description**: Non-validator can perform validator-only operations
   - **Impact**: Unauthorized governance actions, protocol changes
   - **CVSS Vector**: `AV:N/AC:L/PR:L/UI:N/S:C/C:L/I:H/A:L`
   - **Score**: 8.5
   - **Reward**: $18,000 - $23,000

3. **Network-Wide DoS via P2P Message Flooding**
   - **Description**: Malformed P2P message crashes all receiving nodes
   - **Impact**: Complete network halt, service disruption
   - **CVSS Vector**: `AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H`
   - **Score**: 7.5
   - **Reward**: $12,000 - $18,000

4. **Cryptographic Signature Verification Bypass**
   - **Description**: Invalid signatures accepted under specific conditions
   - **Impact**: Unauthorized transactions, limited fund theft
   - **CVSS Vector**: `AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N`
   - **Score**: 7.4
   - **Reward**: $15,000 - $20,000

5. **Incorrect Slashing Logic Affecting Honest Validators**
   - **Description**: Bug causing honest validators to be slashed incorrectly
   - **Impact**: Validator fund loss, reduced security
   - **CVSS Vector**: `AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:H/A:L`
   - **Score**: 7.1
   - **Reward**: $12,000 - $16,000

6. **Flash Loan Attack with Limited Impact**
   - **Description**: Flash loan exploit profitable but limited by circuit breaker
   - **Impact**: Limited fund extraction before protection triggers
   - **CVSS Vector**: `AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N`
   - **Score**: 7.4
   - **Reward**: $14,000 - $18,000

7. **Authentication Bypass in Admin API**
   - **Description**: Bypass authentication for administrative RPC endpoints
   - **Impact**: Unauthorized node control, configuration changes
   - **CVSS Vector**: `AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H`
   - **Score**: 9.8
   - **Reward**: $20,000 - $25,000

### Medium Severity (CVSS 4.0-6.9)

#### Criteria

- **Limited fund loss** requiring specific conditions
- **Single-node DoS** without network impact
- **Information disclosure** of sensitive data
- **TWAP manipulation** under specific scenarios
- **Minor logic errors** in reward distribution
- **Memory leaks** affecting long-running nodes

#### Typical CVSS Vector

`CVSS:3.1/AV:N/AC:H/PR:L/UI:N/S:U/C:L/I:L/A:L` (Score: 4.6)

#### Examples

1. **TWAP Manipulation in Low Liquidity Pools**
   - **Description**: Price manipulation possible with low liquidity
   - **Impact**: Limited arbitrage opportunity, minor losses
   - **CVSS Vector**: `AV:N/AC:H/PR:L/UI:N/S:U/C:N/I:L/A:L`
   - **Score**: 4.2
   - **Reward**: $3,000 - $5,000

2. **Memory Leak in WebSocket Handler**
   - **Description**: Long-running WebSocket connections cause memory leak
   - **Impact**: Node crash after extended uptime
   - **CVSS Vector**: `AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:L`
   - **Score**: 5.3
   - **Reward**: $2,500 - $4,000

3. **Information Disclosure of Node Metrics**
   - **Description**: Unauthenticated access to detailed node performance data
   - **Impact**: Reveals validator behavior, aids other attacks
   - **CVSS Vector**: `AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N`
   - **Score**: 5.3
   - **Reward**: $3,000 - $5,000

4. **Incorrect Fee Calculation with Rounding Error**
   - **Description**: Fee calculation rounding favors users over protocol
   - **Impact**: Slow fund drain over many transactions
   - **CVSS Vector**: `AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:L/A:N`
   - **Score**: 4.3
   - **Reward**: $2,500 - $4,500

5. **Single-Node DoS via Malformed Query**
   - **Description**: Specially crafted query crashes single node
   - **Impact**: Individual node restart required, no network impact
   - **CVSS Vector**: `AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:L`
   - **Score**: 5.3
   - **Reward**: $3,500 - $6,000

6. **Race Condition in Concurrent Transaction Processing**
   - **Description**: Race condition under specific timing conditions
   - **Impact**: Occasional transaction processing error
   - **CVSS Vector**: `AV:N/AC:H/PR:L/UI:N/S:U/C:N/I:L/A:L`
   - **Score**: 4.2
   - **Reward**: $4,000 - $7,000

7. **Insufficient Rate Limiting on Non-Critical Endpoints**
   - **Description**: Missing rate limits on query endpoints
   - **Impact**: Resource exhaustion on single node
   - **CVSS Vector**: `AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:L`
   - **Score**: 5.3
   - **Reward**: $2,500 - $4,000

### Low Severity (CVSS 0.1-3.9)

#### Criteria

- **No direct security impact**
- **Best practice violations**
- **Defense-in-depth opportunities**
- **Code quality issues**
- **Non-exploitable edge cases**
- **Configuration hardening**

#### Typical CVSS Vector

`CVSS:3.1/AV:L/AC:H/PR:H/UI:R/S:U/C:L/I:N/A:N` (Score: 1.8)

#### Examples

1. **Missing Security Headers on Informational Endpoints**
   - **Description**: HSTS, CSP headers missing on docs/status pages
   - **Impact**: Defense-in-depth opportunity
   - **CVSS Vector**: `AV:N/AC:H/PR:N/UI:R/S:U/C:N/I:L/A:N`
   - **Score**: 3.1
   - **Reward**: $500 - $1,000

2. **Verbose Error Messages in Logs**
   - **Description**: Logs contain stack traces but no sensitive data
   - **Impact**: Minimal information disclosure
   - **CVSS Vector**: `AV:L/AC:L/PR:L/UI:N/S:U/C:L/I:N/A:N`
   - **Score**: 3.3
   - **Reward**: $500 - $800

3. **Outdated Dependency Without Known Vulnerabilities**
   - **Description**: Dependency is outdated but no CVEs affect PAW
   - **Impact**: Best practice improvement
   - **CVSS Vector**: `AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:N`
   - **Score**: 3.7
   - **Reward**: $500 - $1,200

4. **Weak Random Seed in Non-Critical Feature**
   - **Description**: Weak randomness in non-security feature
   - **Impact**: No security impact
   - **CVSS Vector**: `AV:L/AC:H/PR:H/UI:R/S:U/C:L/I:N/A:N`
   - **Score**: 1.8
   - **Reward**: $500

5. **Missing Input Validation Without Exploit Path**
   - **Description**: Input not validated but no exploitation possible
   - **Impact**: Code quality issue
   - **CVSS Vector**: `AV:N/AC:H/PR:L/UI:R/S:U/C:N/I:L/A:N`
   - **Score**: 2.6
   - **Reward**: $800 - $1,500

6. **Unencrypted Internal Service Communication**
   - **Description**: Internal services communicate without TLS
   - **Impact**: Defense-in-depth (services on private network)
   - **CVSS Vector**: `AV:A/AC:H/PR:H/UI:N/S:U/C:L/I:N/A:N`
   - **Score**: 2.0
   - **Reward**: $1,000 - $1,800

7. **Potential for Timing Attack on Non-Sensitive Operation**
   - **Description**: Timing differences in non-critical code path
   - **Impact**: No exploitable information
   - **CVSS Vector**: `AV:N/AC:H/PR:L/UI:N/S:U/C:L/I:N/A:N`
   - **Score**: 3.1
   - **Reward**: $600 - $1,200

## Severity Modifiers

### Factors That Increase Severity

1. **Active Exploitation**: Evidence of active exploitation in the wild (+1 severity level)
2. **Chained Vulnerabilities**: Multiple vulnerabilities combined for greater impact (+0.5 to +1 level)
3. **No Mitigations**: Lack of compensating controls (+0.5 level)
4. **Wide Attack Surface**: Affects many components or users (+0.5 level)
5. **Critical Timing**: Found before mainnet launch (may increase reward 2x)

### Factors That Decrease Severity

1. **Mitigating Controls**: Existing protections limit impact (-0.5 level)
2. **Specific Conditions**: Requires very specific preconditions (-0.5 level)
3. **Limited Scope**: Affects only small subset of users/validators (-0.5 level)
4. **High Privileges**: Requires admin/validator access (-0.5 to -1 level)
5. **Theoretical Only**: No practical exploitation path (-1 level)

## Special Considerations for Blockchain Vulnerabilities

### Consensus Vulnerabilities

Vulnerabilities affecting consensus are generally elevated in severity:

- **Consensus failure**: Automatically Critical
- **Consensus degradation**: High or Critical depending on impact
- **Validator compromise**: High or Critical depending on scope

### Economic Vulnerabilities

Economic attacks evaluated based on profitability and systemic risk:

- **Profitable for attacker**: Generally High or Critical
- **Loss exceeds cost**: Severity based on net loss
- **Market manipulation**: Medium to High depending on scale
- **MEV without protocol violation**: Generally out of scope unless breaks fairness guarantees

### Cryptographic Vulnerabilities

Cryptographic issues severity based on what is compromised:

- **Private key recovery**: Critical
- **Signature forgery**: Critical
- **Hash collision**: Critical to High depending on usage
- **Weak randomness**: High to Medium depending on usage
- **Side-channel attacks**: Medium to High depending on exploitability

## Assessment Process

### Step 1: Understand the Vulnerability

- What component is affected?
- What is the root cause?
- How can it be exploited?
- What are the prerequisites?

### Step 2: Assess Impact

- What can an attacker achieve?
- What is the worst-case scenario?
- How many users/validators are affected?
- What is the financial impact?

### Step 3: Determine Likelihood

- How complex is the attack?
- What privileges are required?
- Is user interaction needed?
- What is the attack vector?

### Step 4: Calculate CVSS Score

- Use CVSS 3.1 calculator
- Evaluate each metric carefully
- Document the vector string
- Record the final score

### Step 5: Apply Modifiers

- Consider aggravating factors
- Consider mitigating factors
- Adjust severity if needed
- Document reasoning

### Step 6: Determine Reward

- Use severity matrix for range
- Consider report quality
- Consider researcher cooperation
- Apply any bonuses or reductions

## Report Quality Factors

High-quality reports receive higher rewards within the severity tier:

### Excellent Report (+20-25% in tier)

- Complete proof of concept
- Detailed exploitation steps
- Clear impact analysis
- Suggested remediation
- Well-formatted and professional

### Good Report (+10-15% in tier)

- Working proof of concept
- Clear reproduction steps
- Impact assessment
- Professional presentation

### Adequate Report (Base reward)

- Reproduction steps
- Basic impact description
- Sufficient technical detail

### Incomplete Report (-25-50% in tier)

- Missing key details
- No proof of concept
- Unclear impact
- Difficult to reproduce

## Dispute Resolution

If you disagree with the severity assessment:

1. **Review this matrix** to understand our criteria
2. **Provide additional evidence** supporting your assessment
3. **Calculate CVSS score** with justification
4. **Submit appeal** to security team within 14 days
5. **Final decision** made by security team lead

Appeals should include:

- Original severity assigned
- Your proposed severity
- Detailed justification
- Additional evidence or PoC
- CVSS calculation

## Resources

### CVSS Resources

- **CVSS 3.1 Calculator**: https://www.first.org/cvss/calculator/3.1
- **CVSS 3.1 Specification**: https://www.first.org/cvss/v3.1/specification-document
- **CVSS User Guide**: https://www.first.org/cvss/user-guide

### Vulnerability Databases

- **CVE**: https://cve.mitre.org
- **NVD**: https://nvd.nist.gov
- **GitHub Security Advisories**: https://github.com/advisories

### Blockchain Security Resources

- **Cosmos Hub Bug Bounty**: https://hackerone.com/cosmos
- **Ethereum Bug Bounty**: https://ethereum.org/en/bug-bounty/
- **Immunefi**: https://immunefi.com/explore/
- **Smart Contract Weakness Classification**: https://swcregistry.io

---

**Last Updated**: November 14, 2025
**Version**: 1.0
**Next Review**: February 14, 2026

For questions about severity assessment, contact: bugbounty@paw-blockchain.org
