# PAW Blockchain Bug Bounty Program Documentation

Welcome to the PAW blockchain bug bounty program documentation. This directory contains all the resources you need to participate in our security research program.

## Quick Links

### For Security Researchers

- **[Bug Bounty Program](../BUG_BOUNTY.md)** - Complete program details, scope, and rewards
- **[Submission Template](SUBMISSION_TEMPLATE.md)** - Template for vulnerability reports
- **[Severity Matrix](SEVERITY_MATRIX.md)** - How we assess vulnerability severity
- **[Security Policy](../../SECURITY.md)** - How to report vulnerabilities

### For PAW Team

- **[Triage Process](TRIAGE_PROCESS.md)** - Internal process for handling reports
- **[Validation Script](../../scripts/bug-bounty/validate-submission.sh)** - Automated submission checker

## Program Overview

The PAW blockchain bug bounty program rewards security researchers who help identify and responsibly disclose security vulnerabilities. We offer rewards from $500 to $100,000 USD based on severity and impact.

### Reward Tiers

| Severity | Reward Range       | Response Time |
| -------- | ------------------ | ------------- |
| Critical | $25,000 - $100,000 | 12 hours      |
| High     | $10,000 - $25,000  | 24 hours      |
| Medium   | $2,500 - $10,000   | 48 hours      |
| Low      | $500 - $2,500      | 72 hours      |

### In-Scope Assets

- Core blockchain protocol (consensus, transaction processing, state machine)
- Custom modules (DEX, Oracle, Compute)
- Cryptographic components (key generation, signatures, encryption)
- API & RPC interfaces
- Wallet & key management
- Smart contract platform

### Out of Scope

- Third-party integrations
- Social engineering
- Physical attacks
- Denial of service without protocol impact
- Best practices without direct vulnerability

## How to Submit

### Step 1: Prepare Your Report

Use our [Submission Template](SUBMISSION_TEMPLATE.md) to prepare a comprehensive report including:

- Clear description of the vulnerability
- Step-by-step reproduction instructions
- Proof of concept code or demonstration
- Impact assessment
- Affected versions

### Step 2: Validate Your Submission (Optional)

Before submitting, you can validate your report using our automated checker:

```bash
cd scripts/bug-bounty
./validate-submission.sh your-report.md
```

This will check for completeness and provide a quality score.

### Step 3: Submit Securely

**Preferred Method**:  Security Advisory

1. Go to <REPO_URL>/security/advisories
2. Click "Report a vulnerability"
3. Fill in the form using your prepared report

**Alternative Method**: Encrypted Email

1. Download our PGP key from https://paw-blockchain.org/security.asc
2. Encrypt your report
3. Email to: security@paw-blockchain.org
4. Subject: `[SECURITY] Brief Description - [Severity]`

### Step 4: Track Progress

You will receive:

- Acknowledgment within 12-72 hours with tracking ID
- Initial assessment within 3-7 days
- Regular updates throughout the process
- Reward payment within 30 days of fix deployment

## Severity Assessment

We use CVSS 3.1 scoring combined with impact analysis. See our [Severity Matrix](SEVERITY_MATRIX.md) for detailed guidance.

### Quick Severity Guide

**Critical**: Direct fund loss, consensus failure, complete system compromise

- Example: Double-spend vulnerability, private key extraction

**High**: Significant impact with realistic exploitation

- Example: Authentication bypass, oracle manipulation, network-wide DoS

**Medium**: Limited impact or specific conditions required

- Example: Single-node DoS, information disclosure, minor logic errors

**Low**: Minimal security impact, best practices

- Example: Missing security headers, configuration hardening

## Best Practices for Researchers

### Do

- Test on testnet or local instances
- Report privately and responsibly
- Provide detailed reproduction steps
- Give us time to fix before disclosure
- Ask questions if anything is unclear
- Follow coordinated disclosure timeline

### Don't

- Test on production mainnet
- Access or exfiltrate user data
- Disrupt service for other users
- Publicly disclose before coordination
- Demand payment or threaten disclosure
- Exploit vulnerabilities beyond demonstration

## Safe Harbor

PAW provides legal safe harbor for good faith security research. If you:

- Follow the bug bounty program rules
- Avoid harm to users and the network
- Report responsibly and privately
- Work with us through disclosure

Then we will:

- Not pursue legal action
- Not contact law enforcement
- Not take action against your accounts
- Publicly credit you (if desired)

## Responsible Disclosure

We follow a coordinated disclosure process:

1. **Report** privately to security team
2. **Acknowledgment** within 12-72 hours
3. **Investigation** and validation (7-14 days)
4. **Fix development** (7-90 days based on severity)
5. **Deployment** with validator coordination
6. **Public disclosure** 7-30 days after fix
7. **Reward payment** within 30 days

### Disclosure Timeline

- **Standard**: 90 days from report to public disclosure
- **Critical**: Expedited timeline (7-14 days to fix)
- **Extensions**: Available if needed with justification

## Contact Information

### Security Team

- **Email**: security@paw-blockchain.org
- **Bug Bounty**: bugbounty@paw-blockchain.org
- **PGP Key**: https://paw-blockchain.org/security.asc
- ****: <REPO_URL>/security/advisories

### Response Times

| Severity | Acknowledgment | Initial Assessment | Patch Target |
| -------- | -------------- | ------------------ | ------------ |
| Critical | 12 hours       | 24 hours           | 7 days       |
| High     | 24 hours       | 3 days             | 14 days      |
| Medium   | 48 hours       | 5 days             | 30 days      |
| Low      | 72 hours       | 7 days             | Next release |

## Frequently Asked Questions

### Can I test on mainnet?

No. All testing should be done on testnet or local instances. Exploiting vulnerabilities on mainnet may result in disqualification.

### What if I find a vulnerability in a dependency?

Report it to us if it affects PAW. We will coordinate with the upstream project. You may receive a reward if the impact is significant for PAW.

### How are rewards calculated?

Based on severity, impact, exploitability, and report quality. See our [Severity Matrix](SEVERITY_MATRIX.md) for details.

### Can I appeal a severity assessment?

Yes. Provide additional justification and evidence within 14 days of the assessment.

### What payment methods do you accept?

We pay in USDC, USDT, or PAW tokens (your choice) on Ethereum mainnet or PAW mainnet.

### Do I need KYC?

KYC is required for rewards over $5,000 USD.

### Can I remain anonymous?

Yes. You can choose to remain anonymous and still receive the reward. We offer three attribution levels:

- Full attribution (name, links)
- Pseudonymous (handle only)
- Anonymous (no public credit)

### What if the vulnerability is already being exploited?

Contact us immediately at security@paw-blockchain.org with subject "URGENT - ACTIVE EXPLOITATION". We will expedite response and may increase the reward.

## Resources

### CVSS Scoring

- **Calculator**: https://www.first.org/cvss/calculator/3.1
- **Specification**: https://www.first.org/cvss/v3.1/specification-document
- **User Guide**: https://www.first.org/cvss/user-guide

### Vulnerability Research

- **Cosmos Hub Bug Bounty**: https://hackerone.com/cosmos
- **Ethereum Bug Bounty**: https://ethereum.org/en/bug-bounty/
- **Immunefi**: https://immunefi.com/explore/
- **OWASP**: https://owasp.org/www-project-top-ten/

### Blockchain Security

- **Smart Contract Weakness Classification**: https://swcregistry.io
- **Consensys Best Practices**: https://consensyshub.io/smart-contract-best-practices/
- **Trail of Bits Resources**: https://github.com/crytic

## Hall of Fame

We maintain a Hall of Fame to recognize security researchers who have contributed to PAW security. Top contributors receive:

- Public recognition
- Links to their website/social media
- Invitation to private researcher community
- Early access to program updates
- Recommendation letters (upon request)

### Recognition Tiers

- **Critical Contributors**: Discovered Critical severity vulnerabilities
- **Security Champions**: Discovered 3+ High severity vulnerabilities
- **Top Researchers**: Highest total bounty earnings annually
- **Community Heroes**: Most valuable overall contributions

Current Hall of Fame members will be listed in the main [Bug Bounty Program](../BUG_BOUNTY.md) document.

## Program Statistics

- **Total Rewards Paid**: $0 (launching soon)
- **Vulnerabilities Fixed**: 0
- **Active Researchers**: TBD
- **Average Response Time**: TBD
- **Average Time to Fix**: TBD

Statistics updated monthly at: https://paw-blockchain.org/bug-bounty/stats

## Updates and Changes

This program may be updated from time to time. Major changes will be announced 30 days in advance. Pending submissions are evaluated under the terms in effect at submission time.

### Changelog

**Version 1.0 - November 14, 2025**

- Initial bug bounty program launch
- Comprehensive documentation
- Automated validation tools
- Clear severity matrix and rewards

## Contributing to These Docs

If you find errors or have suggestions for improving this documentation:

1. Open an issue: <REPO_URL>/issues
2. Submit a PR with fixes
3. Email suggestions to: bugbounty@paw-blockchain.org

---

**Thank you for helping keep PAW secure!**

**Last Updated**: November 14, 2025
**Program Version**: 1.0
**Next Review**: February 14, 2026
