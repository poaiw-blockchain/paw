# Security Policy

## Overview

Security is a top priority for the PAW blockchain project. We appreciate the community's efforts in responsibly disclosing security vulnerabilities and helping us maintain a secure platform.

## Supported Versions

We release security updates for the following versions:

| Version | Supported | End of Support |
| ------- | --------- | -------------- |
| 1.x.x   | ✓         | TBD            |
| < 1.0   | ✗         | Unsupported    |

**Note**: We recommend always running the latest stable version to ensure you have the most recent security patches.

## Reporting a Vulnerability

**CRITICAL: DO NOT create public GitHub issues for security vulnerabilities.**

If you discover a security vulnerability in PAW, please report it privately using one of the following methods:

### Preferred Method: GitHub Security Advisories

1. Go to the [Security Advisories](https://github.com/OWNER/paw/security/advisories) page
2. Click "Report a vulnerability"
3. Fill out the vulnerability report form with as much detail as possible
4. Use our [Submission Template](docs/bug-bounty/SUBMISSION_TEMPLATE.md) for guidance

### Alternative Method: Encrypted Email

Send an email to: **security@paw-blockchain.org**

For sensitive reports, use our PGP key for encryption:

```
-----BEGIN PGP PUBLIC KEY BLOCK-----

[PGP PUBLIC KEY WILL BE INSERTED HERE]

Key ID: 0x[KEY_ID]
Fingerprint: [FINGERPRINT]
Download: https://paw-blockchain.org/security.asc
-----END PGP PUBLIC KEY BLOCK-----
```

**Email Subject Format**: `[SECURITY] Brief Description - [Severity]`

Example: `[SECURITY] Double-spend via consensus bypass - CRITICAL`

### Required Information

Include the following information in your report:

- **Type of vulnerability** (e.g., consensus bug, cryptographic flaw, DoS)
- **Affected components** (e.g., transaction validation, P2P networking, DEX module)
- **Affected versions** (specific version numbers or commit hashes)
- **Description** of the vulnerability with technical details
- **Steps to reproduce** (detailed, step-by-step instructions)
- **Proof of concept** code, configuration, or demonstration
- **Potential impact** assessment (what an attacker could achieve)
- **Suggested mitigation** (if you have recommendations)
- **Your contact information** for follow-up questions
- **Disclosure timeline preference** (if you have public disclosure plans)

See our [Submission Template](docs/bug-bounty/SUBMISSION_TEMPLATE.md) for a detailed format.

## What to Expect

### Response Timeline

1. **Acknowledgment**: We will acknowledge receipt of your report within **12-72 hours** depending on severity
2. **Initial Assessment**: We will provide an initial assessment within **3-7 business days**
3. **Investigation**: Our security team will investigate and validate the report
4. **Resolution**: We will work on a fix and coordinate disclosure timeline with you
5. **Credit**: We will credit you in the security advisory (unless you prefer to remain anonymous)

### Severity-Based Response Times

| Severity | Acknowledgment | Initial Assessment | Patch Target | Public Disclosure |
| -------- | -------------- | ------------------ | ------------ | ----------------- |
| Critical | 12 hours       | 24 hours           | 7 days       | 14 days post-fix  |
| High     | 24 hours       | 3 days             | 14 days      | 30 days post-fix  |
| Medium   | 48 hours       | 5 days             | 30 days      | 60 days post-fix  |
| Low      | 72 hours       | 7 days             | Next release | 90 days post-fix  |

**Note**: These are target timelines. Complex vulnerabilities may require more time. We will keep you informed of any delays.

## Coordinated Disclosure Policy

We follow **coordinated disclosure** (also known as responsible disclosure):

### Standard Disclosure Process

1. **Private Disclosure**: Report sent privately to security team
2. **Acknowledgment**: Security team confirms receipt and assigns tracking ID
3. **Investigation**: Security team investigates and validates the vulnerability
4. **Patch Development**: Fix is developed and tested internally
5. **Advance Notice**: Validators and major node operators notified 48-72 hours before public release
6. **Public Release**: Security patch released with coordination
7. **Security Advisory**: Published after deployment with CVE (if applicable)
8. **Full Disclosure**: Technical details published after users have had time to update

### Timeline Expectations

#### Standard Timeline (90 days)

- **Day 0**: Vulnerability reported privately
- **Day 1-7**: Acknowledgment, triage, and initial validation
- **Day 7-60**: Investigation, patch development, and testing
- **Day 60-90**: Coordinated deployment and user notification
- **Day 90**: Public disclosure (if fix is deployed)
- **Post-disclosure**: Full technical details and researcher recognition

#### Critical Vulnerability Timeline (Expedited)

- **Hour 0**: Report received
- **Hour 12**: Acknowledgment and emergency response initiated
- **Day 1**: Validation and impact assessment complete
- **Day 3-7**: Emergency patch developed and tested
- **Day 7**: Coordinated emergency deployment
- **Day 14**: Public security advisory with technical details

#### Extended Timeline

If 90 days is insufficient due to complexity:

- We will request an extension with justification
- Regular progress updates provided to researcher
- Maximum extension: 180 days (exceptional circumstances only)
- Researcher cooperation appreciated but not required

### Embargo Period

We request a **90-day embargo period** to allow time for:

- Thorough investigation and validation
- Development and testing of patches
- Coordination with downstream projects and partners
- Safe deployment across the network
- Validator and node operator notification

If you plan to publish your findings or present at a conference, please coordinate with us to ensure:

- Fixes are deployed before public disclosure
- Users have time to update
- Coordinated messaging
- Proper credit and recognition

### Early Disclosure

If you need to disclose earlier than our timeline:

1. **Notify us immediately** of your timeline constraints
2. **Provide justification** (e.g., conference deadline, active exploitation)
3. **Coordinate messaging** to ensure consistent communication
4. **Allow minimum 7 days** for emergency response (critical issues)

We will work with you to accommodate reasonable disclosure timelines.

## Security Vulnerability Severity

We use a severity classification system based on CVSS 3.1 and impact assessment.

### Critical Severity

**Criteria**: Direct loss of funds, complete protocol compromise, or network-wide consensus failure

**Impact**:

- Complete system compromise
- Unauthorized fund transfers or theft
- Consensus failure or chain halt
- Remote code execution on nodes

**Examples**:

- Double-spend vulnerabilities that bypass consensus
- Private key extraction or recovery
- Consensus-breaking bugs causing permanent fork
- Unlimited token minting or protocol reserve drainage
- Critical DEX vulnerabilities causing fund loss

**Response**: Immediate emergency action, patch within 7 days

**CVSS Score**: 9.0 - 10.0

### High Severity

**Criteria**: Significant security impact with realistic exploitation scenario

**Impact**:

- Significant but limited fund loss
- Service disruption affecting network
- Privilege escalation
- Authentication bypass

**Examples**:

- Privilege escalation in governance or validator systems
- Authentication bypass allowing unauthorized privileged operations
- Oracle manipulation enabling profitable attacks
- Network-wide DoS attacks
- Cryptographic weaknesses in signature verification
- Incorrect slashing logic that penalizes honest validators

**Response**: Urgent patch within 14 days

**CVSS Score**: 7.0 - 8.9

### Medium Severity

**Criteria**: Limited security impact requiring specific conditions or limited attack scope

**Impact**:

- Minor fund loss or lock
- Limited service disruption
- Information disclosure of sensitive data
- Limited privilege escalation

**Examples**:

- Information disclosure of sensitive but non-critical data
- Limited DoS affecting single node without network impact
- Incorrect fee calculations with limited impact
- TWAP manipulation under specific conditions
- Memory leaks or resource exhaustion (specific scenarios)
- Logic errors in reward distribution with minor impact

**Response**: Patch within 30 days

**CVSS Score**: 4.0 - 6.9

### Low Severity

**Criteria**: Minimal security impact, security hardening opportunities

**Impact**:

- No direct fund loss or service impact
- Informational findings
- Best practice violations
- Defense-in-depth opportunities

**Examples**:

- Best practice violations without direct vulnerability
- Missing security headers on informational endpoints
- Verbose error messages without sensitive data
- Code quality issues affecting security maintainability
- Non-exploitable edge cases
- Configuration hardening suggestions

**Response**: Addressed in next regular release

**CVSS Score**: 0.1 - 3.9

See our [Severity Assessment Matrix](docs/bug-bounty/SEVERITY_MATRIX.md) for detailed scoring guidance.

## Bug Bounty Program

We operate a comprehensive bug bounty program to reward security researchers who help improve PAW's security.

### Reward Structure

| Severity | Reward Range           |
| -------- | ---------------------- |
| Critical | $25,000 - $100,000 USD |
| High     | $10,000 - $25,000 USD  |
| Medium   | $2,500 - $10,000 USD   |
| Low      | $500 - $2,500 USD      |

**Payment Options**: USDC, USDT, or PAW tokens (researcher's choice)

**KYC Requirements**: Required for rewards over $5,000 USD

### Program Details

For complete bug bounty program details, including:

- Detailed scope and exclusions
- Submission requirements and process
- Reward modifiers and bonuses
- Legal safe harbor provisions
- Hall of fame recognition

See our [Bug Bounty Program](docs/BUG_BOUNTY.md) document.

### Quick Links

- [Bug Bounty Program](docs/BUG_BOUNTY.md) - Complete program details
- [Submission Template](docs/bug-bounty/SUBMISSION_TEMPLATE.md) - Report format
- [Severity Matrix](docs/bug-bounty/SEVERITY_MATRIX.md) - Detailed scoring guidance
- [Triage Process](docs/bug-bounty/TRIAGE_PROCESS.md) - How we handle reports

## Security Best Practices

### For Node Operators

#### System Security

1. **Keep Software Updated**
   - Run the latest stable version
   - Subscribe to security announcements
   - Apply security patches within 48 hours of release
   - Monitor for security advisories

2. **Network Security**
   - Use firewall rules to restrict access to RPC ports
   - Enable TLS for all RPC connections
   - Isolate validator nodes from public internet
   - Use VPN or private networks for validator communication
   - Implement DDoS protection
   - Use separate sentry nodes for validators

3. **System Hardening**
   - Disable unnecessary services
   - Use security-enhanced Linux (SELinux/AppArmor)
   - Enable automatic security updates for OS
   - Implement intrusion detection systems
   - Regular security audits and penetration testing

#### Key Management

4. **Private Key Protection**
   - Store validator keys in HSM (Hardware Security Module)
   - Use Tendermint KMS for remote signing
   - Never store unencrypted keys on disk
   - Implement key rotation policies
   - Use different keys for different purposes
   - Maintain secure offline backups

5. **Access Control**
   - Use SSH key-based authentication only (disable password auth)
   - Implement multi-factor authentication
   - Use strong, unique passwords for all accounts
   - Implement principle of least privilege
   - Regularly audit access logs
   - Rotate credentials quarterly
   - Use separate accounts for different roles

#### Operational Security

6. **Monitoring and Alerting**
   - Monitor node health and performance metrics
   - Set up alerts for unusual activity
   - Track validator signing performance
   - Monitor for missed blocks
   - Review logs daily
   - Track resource usage and set thresholds
   - Monitor network connectivity

7. **Backup and Recovery**
   - Maintain regular backups of node data
   - Test recovery procedures monthly
   - Store backups securely offline
   - Document recovery processes
   - Maintain redundant infrastructure
   - Have disaster recovery plan

8. **Incident Response**
   - Have incident response plan documented
   - Know how to contact security team
   - Maintain emergency contacts list
   - Practice incident response scenarios
   - Document all security incidents

### For Developers

#### Secure Coding

1. **Input Validation**
   - Validate all inputs at trust boundaries
   - Use allowlists over denylists
   - Sanitize data before use
   - Implement proper type checking
   - Validate ranges and formats

2. **Error Handling**
   - Never expose sensitive data in errors
   - Log errors securely
   - Implement proper exception handling
   - Fail securely (fail closed, not open)
   - Don't reveal system internals

3. **Cryptography**
   - Use established cryptographic libraries only
   - Never implement custom cryptography
   - Use appropriate key lengths (256-bit minimum for symmetric)
   - Implement proper random number generation
   - Follow current cryptographic best practices
   - Use authenticated encryption (e.g., AES-GCM)

4. **Authentication and Authorization**
   - Implement defense in depth
   - Use principle of least privilege
   - Validate permissions on every operation
   - Implement proper session management
   - Use secure password hashing (Argon2, scrypt)

#### Development Process

5. **Code Review**
   - All security-critical code requires review
   - Use automated security scanning tools
   - Look for common vulnerability patterns
   - Review cryptographic implementations carefully
   - Check for race conditions and concurrency issues

6. **Testing**
   - Write security-focused unit tests
   - Perform fuzzing on critical components
   - Conduct regular security audits
   - Test edge cases and error conditions
   - Run integration and E2E security tests
   - Perform load and stress testing

7. **Dependencies**
   - Keep dependencies updated
   - Review dependency security advisories
   - Use dependency scanning tools (Dependabot, Snyk)
   - Minimize dependency count
   - Pin dependency versions
   - Audit critical dependencies

8. **Deployment**
   - Use CI/CD security scanning
   - Implement security gates in pipeline
   - Use infrastructure as code
   - Maintain separate environments
   - Implement blue-green deployments

### For Users

#### Wallet Security

1. **Key Management**
   - Use hardware wallets for large amounts
   - Keep wallet software updated
   - Backup recovery phrases securely (offline, metal)
   - Never share private keys or recovery phrases
   - Use different wallets for different purposes
   - Test recovery process with small amounts

2. **Transaction Safety**
   - Verify addresses before sending transactions
   - Double-check recipient addresses (compare multiple characters)
   - Verify transaction details before signing
   - Start with small test transactions
   - Be cautious with smart contracts
   - Verify contract addresses from official sources

#### Security Hygiene

3. **Phishing Protection**
   - Verify URLs and domains carefully
   - Don't click suspicious links
   - Don't download software from unofficial sources
   - Be wary of social engineering attempts
   - Don't trust unsolicited messages
   - Verify official communication channels

4. **Device Security**
   - Keep operating system updated
   - Use antivirus/antimalware software
   - Don't use public WiFi for transactions
   - Use VPN for sensitive operations
   - Enable full disk encryption
   - Lock devices when not in use

5. **Privacy**
   - Don't share wallet addresses publicly
   - Use different addresses for different purposes
   - Be aware of on-chain privacy limitations
   - Don't discuss holdings publicly
   - Be cautious on social media

## Security Audit History

We conduct regular security audits and publish results:

### Completed Audits

| Date    | Auditor  | Scope                 | Status   | Report                                                           |
| ------- | -------- | --------------------- | -------- | ---------------------------------------------------------------- |
| 2025-11 | Internal | Full codebase review  | Complete | [SECURITY_AUDIT_SUMMARY.txt](SECURITY_AUDIT_SUMMARY.txt)         |
| 2025-11 | Internal | Transaction security  | Complete | [TRANSACTION_SECURITY_AUDIT.md](TRANSACTION_SECURITY_AUDIT.md)   |
| 2025-11 | Internal | Wallet key management | Complete | [WALLET_KEY_MANAGEMENT_AUDIT.md](WALLET_KEY_MANAGEMENT_AUDIT.md) |
| 2025-11 | Internal | Network security      | Complete | [NETWORK_SECURITY_AUDIT.md](NETWORK_SECURITY_AUDIT.md)           |

### Planned Audits

| Quarter | Auditor        | Scope         | Status  |
| ------- | -------------- | ------------- | ------- |
| Q1 2026 | External (TBD) | Core protocol | Planned |
| Q2 2026 | External (TBD) | DEX module    | Planned |
| Q3 2026 | External (TBD) | Cryptography  | Planned |

## Security Contacts

### Primary Contacts

- **Security Team Email**: security@paw-blockchain.org
- **Bug Bounty Email**: bugbounty@paw-blockchain.org
- **PGP Key**: https://paw-blockchain.org/security.asc
- **GitHub Security Advisories**: https://github.com/[PAW-ORG]/paw/security/advisories

### PGP Key Information

```
Key ID: 0x[KEY_ID]
Fingerprint: [FULL_FINGERPRINT]
Download: https://paw-blockchain.org/security.asc
Keyserver: keys.openpgp.org
```

### Emergency Contact

For critical vulnerabilities requiring immediate attention:

- **Email**: security@paw-blockchain.org (Subject: URGENT - CRITICAL VULNERABILITY)
- **Expected Response**: Within 12 hours
- **24/7 Monitoring**: Yes (for critical severity)

## Known Security Considerations

### Blockchain-Specific Risks

#### Consensus Attacks

1. **51% Attack**
   - Risk: Attacker controlling majority of stake could reorganize chain
   - Mitigation: High validator count, stake distribution, social consensus
   - Monitoring: Track validator stake concentration

2. **Long Range Attack**
   - Risk: Attacker with old keys could create alternative history
   - Mitigation: Weak subjectivity, checkpointing
   - Monitoring: Track chain finality

3. **Eclipse Attack**
   - Risk: Node isolation from honest peers
   - Mitigation: Diverse peer connections, persistent peer lists
   - Monitoring: Peer diversity metrics

#### Network Attacks

4. **DDoS Attacks**
   - Risk: Network flooding, resource exhaustion
   - Mitigation: Rate limiting, peer reputation, sentry nodes
   - Monitoring: Network traffic analysis

5. **Sybil Attacks**
   - Risk: Adversary creating many identities
   - Mitigation: Proof of stake, reputation systems
   - Monitoring: Peer behavior analysis

### Cryptographic Assumptions

PAW's security relies on the following cryptographic assumptions:

#### Primitives

1. **SHA-256**
   - Assumption: Collision resistance and preimage resistance
   - Use: Block hashing, Merkle trees
   - Risk: Collision attacks (theoretical)

2. **ECDSA (secp256k1)**
   - Assumption: Discrete logarithm problem hardness
   - Use: Transaction signatures
   - Risk: Signature malleability (mitigated)

3. **Ed25519**
   - Assumption: Curve25519 DLP hardness
   - Use: Consensus signatures
   - Risk: Implementation vulnerabilities

4. **AES-256-GCM**
   - Assumption: AES security, GCM authentication
   - Use: Encrypted storage
   - Risk: Implementation side-channels

### Quantum Computing Risk

Current cryptographic algorithms may be vulnerable to quantum computers in the future.

#### Timeline Assessment

- **Near-term (1-5 years)**: Low risk
- **Medium-term (5-10 years)**: Moderate risk
- **Long-term (10+ years)**: High risk

#### Monitoring

We actively monitor:

- Quantum computing developments
- Post-quantum cryptography (PQC) research
- NIST PQC standardization process
- Industry migration strategies

#### Preparation

- Research PQC algorithms (Kyber, Dilithium, SPHINCS+)
- Plan migration strategy
- Design upgrade path
- Test quantum-resistant implementations

## Security Updates and Announcements

### Subscription Channels

Stay informed about security updates:

1. **GitHub Watch**
   - Watch the repository: https://github.com/[PAW-ORG]/paw
   - Enable notifications for security advisories
   - Star the repository for updates

2. **Mailing Lists**
   - Security announcements: security-announce@paw-blockchain.org
   - Node operator updates: validators@paw-blockchain.org
   - Subscribe: https://paw-blockchain.org/subscribe

3. **Social Media**
   - Twitter/X: @PAWBlockchain
   - Discord: https://discord.gg/paw (security-announcements channel)
   - Telegram: @PAWAnnouncements

4. **RSS/Atom Feeds**
   - Security advisories: https://paw-blockchain.org/security/feed.xml
   - Blog: https://blog.paw-blockchain.org/feed.xml

### Advisory Format

Security advisories include:

- CVE identifier (if assigned)
- Severity rating
- Affected versions
- Impact description
- Remediation steps
- Credit to researcher(s)
- Timeline of events

## Compliance and Standards

PAW follows security best practices from:

### Industry Standards

- **OWASP**: Web application security, API security
- **CWE**: Common Weakness Enumeration
- **NIST**: Cryptographic standards (FIPS 140-2, SP 800-series)
- **ISO 27001**: Information security management
- **SOC 2**: Security, availability, confidentiality

### Blockchain-Specific Standards

- **DASP**: Decentralized Application Security Project
- **Smart Contract Best Practices**: Consensys, Trail of Bits
- **Cosmos SDK Security**: Standard practices
- **Tendermint Security**: Consensus security guidelines

### Vulnerability Databases

- **CVE**: Common Vulnerabilities and Exposures
- **NVD**: National Vulnerability Database
- **GitHub Security Advisories**: Advisory database

## Security Researcher Recognition

We recognize and appreciate security researchers who have helped improve PAW:

### Hall of Fame

_Security researchers will be listed here upon their consent after successful disclosure_

### Recognition Levels

- **Critical Contributors**: Discovered Critical severity vulnerabilities
- **Security Champions**: Discovered 3+ High severity vulnerabilities
- **Top Researchers**: Highest total bounty earnings
- **Community Heroes**: Most valuable overall contributions

### Benefits

- Public recognition in this document
- Mention in security advisories
- Featured in blog posts
- Recommendation letters available upon request
- Invitation to private security researcher community
- Early access to bug bounty program updates

### Privacy Options

You may choose:

- **Full Attribution**: Name, website, social media links
- **Partial Attribution**: Handle/username only
- **Anonymous**: No public attribution (still receive reward)

## Legal Safe Harbor

### Authorization and Protection

PAW provides legal safe harbor for security research conducted in good faith.

#### Safe Harbor Conditions

If you:

- Make a good faith effort to comply with this policy
- Avoid privacy violations, data destruction, and service interruption
- Only interact with accounts you own or have explicit permission to test
- Don't exploit vulnerabilities beyond demonstrating the issue
- Report vulnerabilities promptly and privately
- Don't publicly disclose issues before coordinated disclosure

Then we will:

- Not pursue legal action against you
- Not refer you to law enforcement
- Not take administrative action against your accounts
- Work with you to understand and remediate the issue
- Publicly acknowledge your contribution (if desired)

#### Authorization Scope

This security policy provides authorization to:

- Test PAW software within defined scope
- Attempt to identify security vulnerabilities
- Bypass security controls for research purposes
- Report findings privately to our security team

#### Limitations

Safe harbor does **NOT** apply if you:

- Violate any law or regulation
- Access, modify, or delete data belonging to others
- Intentionally harm users or the network
- Demand payment or other compensation (extortion)
- Disclose vulnerabilities publicly before coordination
- Test on production mainnet without permission

### Disclaimer

This safe harbor:

- Is limited to actions authorized by this policy
- Does not waive rights of third parties
- Is subject to applicable laws
- May be revoked for policy violations

## Additional Resources

### Documentation

- [Bug Bounty Program](docs/BUG_BOUNTY.md) - Complete bounty program
- [Security Runbook](docs/SECURITY_RUNBOOK.md) - Operational security procedures
- [Incident Response Plan](docs/INCIDENT_RESPONSE_PLAN.md) - Incident handling procedures
- [Disaster Recovery](docs/DISASTER_RECOVERY.md) - Recovery procedures
- [Security Audit Summary](SECURITY_AUDIT_SUMMARY.txt) - Latest audit findings

### External Resources

- [Cosmos SDK Security](https://docs.cosmos.network/main/building-apps/app-go-v2#security)
- [Tendermint Security](https://github.com/tendermint/tendermint/blob/master/SECURITY.md)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CWE Top 25](https://cwe.mitre.org/top25/)

### Community

- [Contributing Guide](CONTRIBUTING.md) - How to contribute
- [Code of Conduct](CODE_OF_CONDUCT.md) - Community standards
- [Discord](https://discord.gg/paw) - Community chat
- [Forum](https://forum.paw-blockchain.org) - Discussion forum

---

## Program Updates

### Changelog

**Version 2.0 - November 14, 2025**

- Expanded bug bounty program with detailed structure
- Added PGP key information for encrypted reports
- Enhanced severity classification with CVSS scoring
- Added coordinated disclosure timeline details
- Expanded security best practices
- Added quantum computing risk assessment
- Enhanced legal safe harbor provisions

**Version 1.0 - November 12, 2025**

- Initial security policy

---

**Thank you for helping keep PAW and our community secure!**

**Last Updated**: November 14, 2025
**Policy Version**: 2.0
**Next Review**: February 14, 2026
