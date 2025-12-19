# PAW Blockchain Bug Bounty Program

## Program Overview

The PAW Blockchain Bug Bounty Program is designed to encourage security researchers and the broader community to help identify and responsibly disclose security vulnerabilities in the PAW ecosystem. We believe that working with skilled security researchers is crucial to maintaining the security and integrity of our blockchain platform.

### Program Goals

- **Strengthen Security**: Identify and remediate vulnerabilities before they can be exploited
- **Community Engagement**: Build a collaborative relationship with the security research community
- **Transparency**: Demonstrate our commitment to security through open communication
- **Continuous Improvement**: Use findings to enhance our security practices and protocols

### Important Notice

This bug bounty program is complementary to our standard security reporting process. All submissions are carefully reviewed and validated by our security team. We reserve the right to modify program terms and reward amounts at any time.

### Operations & Contacts

Looking for the internal workflow, contact matrix, or launch checklist? See [`docs/guides/security/BUG_BOUNTY_RUNBOOK.md`](./guides/security/BUG_BOUNTY_RUNBOOK.md) for the day-to-day runbook that coordinates intake, triage, payouts, and communications.

## Scope

### In-Scope Assets

The following components are eligible for bounty rewards:

#### 1. Core Blockchain Protocol

- **Consensus mechanism** - Tendermint/CometBFT integration and validator logic
- **Transaction processing** - Transaction validation, signature verification, and state transitions
- **Block production** - Block creation, validation, and propagation
- **P2P networking** - Node communication, peer discovery, and message handling
- **State machine** - Application state management and transitions
- **Genesis initialization** - Genesis file processing and chain initialization

#### 2. Custom Modules (x/\*)

- **DEX Module (x/dex)**
  - Liquidity pool creation and management
  - Atomic swaps and trading logic
  - TWAP (Time-Weighted Average Price) calculations
  - Flash loan mechanisms
  - Circuit breaker functionality
  - DEX invariants and safety checks

- **Oracle Module (x/oracle)**
  - Price feed aggregation
  - Validator reporting mechanism
  - Slashing logic for malicious/incorrect reports
  - Data validation and verification

- **Compute Module (x/compute)**
  - API computation aggregation
  - Task submission and verification
  - Reward distribution

#### 3. Cryptographic Components

- **Key generation** - BIP39/BIP44 mnemonic and key derivation
- **Signature schemes** - ECDSA, Ed25519 implementations
- **Encryption** - Fernet encryption for wallet storage
- **Hash functions** - Cryptographic hash usage and validation
- **Random number generation** - VRF and other randomness sources

#### 4. API & RPC Interfaces

- **REST API** (api/\*)
  - Authentication and authorization
  - Input validation and sanitization
  - Rate limiting and DoS protection
  - WebSocket connections

- **gRPC/Protobuf services**
  - Query interfaces
  - Transaction broadcasting
  - State queries

#### 5. Wallet & Key Management

- **CLI wallet (cmd/pawcli)**
- **Key storage** - Keyring and encrypted storage
- **Transaction signing** - Signature generation and verification
- **Recovery mechanisms** - Mnemonic backup and recovery

#### 6. Smart Contract Platform (if applicable)

- **Contract execution environment**
- **Gas metering and resource limits**
- **Contract-to-contract interactions**
- **Permission systems**

### Out-of-Scope

The following are **NOT** eligible for bounty rewards:

#### Excluded Assets

- Third-party integrations and services
- Block explorers not officially maintained by PAW
- Community-run infrastructure (validators, RPC nodes)
- Mobile/web wallets not officially maintained by PAW
- Documentation websites and static content
- Social media accounts
- Email systems and corporate IT infrastructure

#### Excluded Vulnerability Types

- **Already Known Issues**: Vulnerabilities already reported or documented in our issue tracker
- **Denial of Service**: Generic DoS/DDoS attacks without protocol-level impact
- **Social Engineering**: Phishing, social engineering of team members or users
- **Physical Attacks**: Physical access to servers or hardware
- **Best Practices**: Issues that don't directly lead to a vulnerability (e.g., missing DNSSEC, SSL/TLS best practices)
- **Theoretical Vulnerabilities**: Issues requiring unrealistic preconditions or without demonstrated impact
- **Spam/Test Content**: Ability to create spam transactions (expected blockchain behavior)
- **Information Disclosure**: Public information or non-sensitive data exposure
- **Self-XSS**: Attacks requiring the victim to paste code into browser console
- **Rate Limiting**: Missing rate limiting on non-critical endpoints
- **Open Ports**: Services that are intentionally public-facing
- **Software Version Disclosure**: Version information in headers/responses
- **Clickjacking**: On pages without sensitive actions
- **CSV Injection**: In exported data
- **Homograph/IDN Attacks**: Domain name spoofing
- **Economic Attacks**: MEV (Miner Extractable Value), front-running, or other economic considerations unless they result in a direct protocol vulnerability

## Severity Classification

We use a severity classification system based on **Impact** and **Likelihood** to determine bounty rewards.

### Critical Severity

**Criteria**: Direct loss of funds, complete protocol compromise, or network-wide consensus failure

**Examples**:

- Double-spend vulnerabilities that bypass consensus
- Private key extraction or recovery
- Remote code execution on validator nodes
- Consensus-breaking bugs causing chain halt or fork
- Unlimited token minting or burning
- Bypass of DEX circuit breaker during critical conditions
- Theft of funds from liquidity pools or user accounts
- Flash loan attacks that drain protocol reserves

**Reward**: $25,000 - $100,000 USD (paid in stablecoins or tokens)

**Response Time**: Acknowledged within 12 hours, patch within 7 days

### High Severity

**Criteria**: Significant security impact with realistic exploitation scenario

**Examples**:

- Authentication bypass allowing unauthorized access to privileged functions
- Privilege escalation in governance or validator systems
- Oracle manipulation leading to incorrect price data
- DoS attacks that can halt specific modules or validators
- Cryptographic weaknesses in signature verification
- Partial state corruption or data integrity issues
- Race conditions in transaction processing
- Incorrect slashing logic that penalizes honest validators
- MEV extraction that violates protocol fairness guarantees
- Flash loan exploits causing temporary fund lock or loss

**Reward**: $10,000 - $25,000 USD

**Response Time**: Acknowledged within 24 hours, patch within 14 days

### Medium Severity

**Criteria**: Limited security impact requiring specific conditions or limited attack scope

**Examples**:

- Information disclosure of sensitive but non-critical data
- Limited DoS affecting single node without network impact
- Incorrect fee calculations or rounding errors
- TWAP manipulation under specific market conditions
- Validator performance degradation attacks
- Memory leaks or resource exhaustion in specific scenarios
- Input validation issues without direct exploitation path
- Logic errors in reward distribution with limited impact
- Incorrect error handling exposing internal state
- API rate limiting bypass on non-critical endpoints

**Reward**: $2,500 - $10,000 USD

**Response Time**: Acknowledged within 48 hours, patch within 30 days

### Low Severity

**Criteria**: Minimal security impact, security hardening opportunities

**Examples**:

- Best practice violations without direct vulnerability
- Missing security headers on informational endpoints
- Verbose error messages without sensitive data
- Code quality issues affecting maintainability
- Defense-in-depth improvements
- Documentation of security assumptions
- Non-exploitable edge cases
- Minor cryptographic improvements
- Configuration hardening suggestions
- Non-critical dependency updates

**Reward**: $500 - $2,500 USD

**Response Time**: Acknowledged within 72 hours, addressed in next regular release

### Severity Modifiers

Severity may be adjusted based on:

- **Attack Complexity**: Higher complexity reduces severity
- **Privileges Required**: Higher privileges required reduces severity
- **User Interaction**: Required user interaction reduces severity
- **Scope**: Broader scope increases severity
- **Confidentiality/Integrity/Availability Impact**: Higher impact increases severity
- **Proof of Concept Quality**: Well-documented PoC may increase reward within tier
- **Fix Complexity**: More complex fixes may justify higher reward

## Reward Structure

### Payment Details

- **Currency**: Payments made in USDC, USDT, or PAW tokens (researcher's choice)
- **Network**: Ethereum mainnet or PAW mainnet
- **Processing Time**: Within 30 days of fix verification and deployment
- **Tax Responsibility**: Recipients responsible for tax obligations in their jurisdiction
- **KYC Requirements**: Required for rewards over $5,000 USD

### Reward Bonuses

Additional rewards may be granted for:

- **High-Quality Reports**: Clear, detailed reports with PoC (+10-25%)
- **Responsible Disclosure**: Following disclosure timeline (+10%)
- **Fix Suggestions**: Actionable remediation recommendations (+5-15%)
- **Multiple Vulnerabilities**: Chained vulnerabilities increasing impact (+25-50%)
- **Critical Timing**: Vulnerabilities found before mainnet launch (2x multiplier)

### Reward Reductions

Rewards may be reduced or denied for:

- **Public Disclosure**: Premature public disclosure before fix (-100%)
- **Incomplete Reports**: Missing critical details or reproduction steps (-25-50%)
- **Duplicates**: Already reported issues (first report receives reward)
- **Bad Faith**: Intentional harm or extortion attempts (-100%, possible legal action)
- **Scope Violations**: Testing on live mainnet without permission (-50-100%)

## Submission Process

### 1. Preparation

Before submitting:

- **Verify Scope**: Ensure vulnerability is in-scope
- **Check Duplicates**: Review existing reports and public issues
- **Prepare Evidence**: Document reproduction steps and create PoC
- **Assess Impact**: Evaluate severity and potential damage
- **DO NOT**: Test on mainnet or cause actual harm

### 2. Submission Methods

#### Preferred:  Security Advisory (Private)

1. Navigate to <REPO_URL>/security/advisories
2. Click "Report a vulnerability"
3. Fill out the report using our template
4. Submit and await response

#### Alternative: Encrypted Email

1. Download our PGP key: https://paw-blockchain.org/security.asc
2. Encrypt your report using PGP
3. Send to: security@paw-blockchain.org
4. Use subject: "Bug Bounty: [Brief Description]"

### 3. Report Requirements

Your submission must include:

#### Required Information

- **Summary**: Brief description of the vulnerability
- **Severity Assessment**: Your severity estimate with justification
- **Affected Components**: Specific modules, functions, or endpoints
- **Vulnerability Type**: Category (e.g., authentication bypass, cryptographic flaw)
- **Impact**: What an attacker could achieve
- **Prerequisites**: Conditions required for exploitation
- **Reproduction Steps**: Detailed step-by-step instructions
- **Proof of Concept**: Code, screenshots, or video demonstration
- **Environment**: Version, configuration, and testing environment details
- **Remediation Suggestions**: Optional but appreciated
- **Your Contact**: Email for communication

#### Optional but Helpful

- **CVSS Score**: Your calculated CVSS v3.1 score
- **Timeline**: When you discovered it, how long you've known
- **Related Issues**: Links to similar vulnerabilities or research
- **Attack Scenarios**: Real-world exploitation scenarios

### 4. What Happens Next

#### Phase 1: Acknowledgment (12-72 hours)

- Confirmation of receipt
- Ticket/tracking number assignment
- Initial point of contact identified

#### Phase 2: Triage (3-7 days)

- Security team reviews submission
- Severity validation and assessment
- Reproduction attempt
- Impact analysis
- Preliminary reward estimate communicated

#### Phase 3: Validation (7-14 days)

- Detailed investigation
- Code analysis and scope determination
- Development of fix strategy
- Coordination with development team
- Progress updates to researcher

#### Phase 4: Remediation (varies by severity)

- Patch development
- Internal testing
- Security review of fix
- Preparation for deployment
- Researcher notification before deployment

#### Phase 5: Deployment (varies)

- Coordinated deployment across network
- Monitoring for issues
- Confirmation of fix effectiveness
- Request for researcher verification

#### Phase 6: Disclosure & Reward (7-30 days post-fix)

- Public security advisory published
- CVE assignment (if applicable)
- Researcher credited (unless anonymous preference)
- Reward payment processing
- Hall of Fame recognition

## Response Time Commitments

### Our Promises

| Severity | Acknowledgment | Initial Assessment | Patch Target | Public Disclosure |
| -------- | -------------- | ------------------ | ------------ | ----------------- |
| Critical | 12 hours       | 24 hours           | 7 days       | 14 days post-fix  |
| High     | 24 hours       | 3 days             | 14 days      | 30 days post-fix  |
| Medium   | 48 hours       | 5 days             | 30 days      | 60 days post-fix  |
| Low      | 72 hours       | 7 days             | Next release | 90 days post-fix  |

### Communication

- **Regular Updates**: Status updates at least every 7 days during investigation
- **Transparency**: Clear communication about timeline or delays
- **Accessibility**: Dedicated security team contact available
- **Coordination**: Work with you on disclosure timeline if needed

## Responsible Disclosure Policy

### Our Expectations

We expect security researchers to:

1. **Private Disclosure**: Report vulnerabilities privately first
2. **No Harm**: Avoid actions that could harm users or the network
3. **Reasonable Testing**: Test on testnet or private instances when possible
4. **Data Protection**: Don't access, modify, or exfiltrate user data
5. **Service Continuity**: Don't disrupt service for other users
6. **Cooperation**: Work with us through the disclosure process
7. **Patience**: Allow reasonable time for investigation and fix
8. **Confidentiality**: Keep details confidential until public disclosure

### Prohibited Activities

- **Mainnet Exploitation**: Actively exploiting on production network
- **Data Theft**: Accessing, downloading, or exfiltrating user data
- **Service Disruption**: DoS attacks, resource exhaustion, or network disruption
- **Spam/Noise**: Creating excessive transactions or spam
- **Social Engineering**: Phishing or manipulating users/staff
- **Extortion**: Demanding payment or threatening disclosure
- **Automated Scanning**: Aggressive scanning causing load issues

### Coordinated Disclosure Timeline

#### Standard Timeline

1. **Day 0**: Vulnerability reported privately
2. **Day 1-3**: Acknowledgment and initial triage
3. **Day 7-14**: Validation and severity confirmation
4. **Day 14-90**: Patch development and testing
5. **Day 90**: Coordinated public disclosure (if not yet fixed)
6. **Post-fix +7-30 days**: Public security advisory

#### Expedited Timeline (Critical Severity)

1. **Hour 0**: Report received
2. **Hour 12**: Acknowledgment
3. **Hour 24**: Validation complete
4. **Day 3-7**: Emergency patch developed
5. **Day 7**: Deployment and public advisory
6. **Day 14**: Full technical disclosure

#### Extended Timeline

- If 90 days is insufficient, we will request extension
- Extensions granted based on fix complexity and coordination needs
- Regular updates provided to researcher during extension
- Maximum extension: 180 days (requires strong justification)

## Legal Safe Harbor

### Safe Harbor Protection

PAW provides legal safe harbor for security research conducted in good faith. If you:

- Make a good faith effort to comply with this policy
- Avoid privacy violations, data destruction, and service disruption
- Only interact with accounts you own or have explicit permission to test
- Don't exploit vulnerabilities beyond demonstrating the issue
- Report vulnerabilities promptly and privately
- Don't publicly disclose issues before coordination

Then we will:

- Not pursue legal action against you
- Not refer you to law enforcement
- Not take administrative action against your accounts
- Work with you to understand and remediate the issue
- Publicly acknowledge your contribution (unless you prefer anonymity)

### Safe Harbor Limitations

Safe harbor does **NOT** apply if you:

- Violate any law or regulation
- Access, modify, or delete data belonging to others
- Intentionally harm users or the network
- Demand payment or other compensation (extortion)
- Disclose vulnerabilities publicly before coordination
- Test on production systems when alternatives exist

### Authorization

This bug bounty program provides authorization to:

- Test PAW software and systems within defined scope
- Attempt to identify and demonstrate security vulnerabilities
- Bypass security controls for research purposes within guidelines
- Report findings privately to our security team

This authorization is:

- Limited to the scope defined in this program
- Revocable at any time if terms are violated
- Not a defense against third-party complaints
- Subject to all applicable laws

## Hall of Fame

We recognize and thank security researchers who have helped improve PAW security:

### 2025 Contributors

_Contributors will be listed here upon successful disclosure and their consent_

### Recognition Tiers

- **Critical Contributors**: Discovered Critical severity vulnerabilities
- **Security Champions**: Discovered 3+ High severity vulnerabilities
- **Top Researchers**: Highest total bounty earnings annually
- **Community Heroes**: Most valuable overall contributions

### Public Recognition

With your consent, we will:

- List your name/handle in this document
- Mention you in security advisories
- Link to your website/social media
- Feature you in blog posts or announcements
- Provide recommendation letter upon request

You may choose to remain:

- **Anonymous**: No public attribution
- **Pseudonymous**: Handle/username only
- **Attributed**: Full name and links

## Program Rules

### Eligibility

- Open to anyone globally (subject to sanctions screening)
- Must be 18+ years old or have parental consent
- PAW employees and immediate family are ineligible
- Current contractors/consultants are ineligible
- Multiple submissions allowed from same researcher

### Duplicate Reports

- First valid report receives full reward
- Duplicate reports receive no reward
- Timestamp of submission determines priority
- Substantially different vulnerabilities in same component are not duplicates
- Same vulnerability in different components may qualify for partial reward

### Disclosure

- Researcher agrees to coordinated disclosure
- Public disclosure only after fix is deployed
- Security advisory will credit researcher (unless anonymous)
- Researcher may publish own write-up after disclosure period
- Marketing or media requests should coordinate with PAW team

### Payment

- Rewards paid after fix is verified and deployed
- Payment in USDC, USDT, or PAW tokens (researcher choice)
- KYC required for rewards over $5,000
- Subject to sanctions screening
- Tax documentation may be required
- Payment processing: 15-30 days after fix deployment

### Modifications

- PAW reserves right to modify program terms
- Changes will be announced 30 days in advance
- Pending submissions evaluated under terms at time of submission
- Reward amounts subject to change based on program budget

### Final Authority

- PAW security team has final authority on:
  - Severity assessment
  - Reward amounts
  - Eligibility decisions
  - Scope determinations
- Decisions are final and binding
- Good faith appeals will be considered

## Contact Information

### Security Team

- **Email**: security@paw-blockchain.org
- **PGP Key**: https://paw-blockchain.org/security.asc
- ****: <REPO_URL>/security/advisories
- **Response Time**: 12-72 hours depending on severity

### Bug Bounty Specific

- **Email**: bugbounty@paw-blockchain.org
- **PGP Key**: Same as security team
- **Telegram**: @PAWBugBounty (for general questions only, not for reporting)
- **Discord**: #bug-bounty channel (for general questions only)

### Program Management

- **Status Page**: https://paw-blockchain.org/bug-bounty/status
- **Statistics**: https://paw-blockchain.org/bug-bounty/stats
- **Blog**: https://blog.paw-blockchain.org/security

## Additional Resources

- [Vulnerability Disclosure Policy](../SECURITY.md)
- [Severity Assessment Matrix](bug-bounty/SEVERITY_MATRIX.md)
- [Submission Template](bug-bounty/SUBMISSION_TEMPLATE.md)
- [Triage Process](bug-bounty/TRIAGE_PROCESS.md)
- [Security Runbook](SECURITY_RUNBOOK.md)
- [Contributing Guide](../CONTRIBUTING.md)

## Frequently Asked Questions

### Q: Can I test on mainnet?

**A**: No. All testing should be conducted on testnet, local instances, or in controlled environments. Exploiting vulnerabilities on mainnet may result in disqualification and legal action.

### Q: What if I find a vulnerability in a dependency?

**A**: Report it to us if it affects PAW. We will coordinate with the upstream project. You may be eligible for a reduced reward if the vulnerability has significant PAW-specific impact.

### Q: How are reward amounts determined?

**A**: Based on severity, impact, quality of report, and complexity of fix. Our security team uses CVSS scoring and internal assessment guidelines.

### Q: Can I receive partial credit for a duplicate?

**A**: Generally no, unless your report provides substantial new information or a different attack vector.

### Q: What if I disagree with the severity assessment?

**A**: You may appeal with additional justification. Final decisions rest with the security team.

### Q: Are automated scanner results eligible?

**A**: Only if you've validated the finding, assessed impact, and provided context. Raw scanner output is not sufficient.

### Q: Can I submit the same bug to multiple programs?

**A**: Yes, if the vulnerability affects multiple projects. Disclose to each project independently.

### Q: What happens if the vulnerability is already being exploited?

**A**: Contact us immediately via security@paw-blockchain.org. We will expedite response and may increase reward for critical findings.

### Q: How do I receive payment?

**A**: After KYC (if required), you'll provide a wallet address. Payment processed via smart contract or direct transfer.

### Q: Is there a maximum number of submissions per researcher?

**A**: No limit, but quality over quantity. Spam submissions may result in disqualification.

---

## Program Statistics

**Total Rewards Paid**: $0 (Program launching soon)
**Vulnerabilities Fixed**: 0
**Average Response Time**: TBD
**Average Time to Fix**: TBD
**Top Researcher**: TBD

---

**Last Updated**: November 14, 2025
**Program Version**: 1.0
**Program Status**: Active

Thank you for helping keep PAW secure!
