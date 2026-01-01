# PAW Blockchain Bug Bounty Program

## Program Overview

The PAW Bug Bounty Program invites security researchers and community members to help identify vulnerabilities in the PAW blockchain. As a community-driven project with limited resources, we offer rewards exclusively in PAW tokens from our development fund.

### Program Philosophy

PAW is built by the community, for the community. Our bug bounty program reflects this ethos:

- **Token-Only Rewards**: All bounties paid in PAW tokens
- **Vesting for Large Rewards**: Critical findings include a 6-month vesting period
- **Recognition First**: Hall of Fame, contributor badges, and community acknowledgment
- **Collaborative Spirit**: We work with researchers, not against them

## Scope

### In-Scope Assets

#### Core Blockchain
- Consensus mechanism (Tendermint/CometBFT integration)
- Transaction processing and signature verification
- Block production and validation
- P2P networking and peer discovery
- State machine and genesis initialization

#### Custom Modules (x/*)
- **DEX Module**: Liquidity pools, swaps, TWAP, circuit breakers
- **Oracle Module**: Price feeds, validator reporting, slashing logic
- **Compute Module**: API computation, task verification, rewards

#### Cryptographic Components
- Key generation (BIP39/BIP44)
- Signature schemes (ECDSA, Ed25519)
- Hash functions and random number generation

#### APIs & Interfaces
- REST API (authentication, input validation, rate limiting)
- gRPC/Protobuf services
- CLI wallet (cmd/pawcli)

### Out of Scope

- Third-party integrations and dependencies
- Community-run infrastructure (validators, RPC nodes)
- Social engineering and phishing attacks
- Physical attacks or hardware access
- Generic DoS/DDoS without protocol impact
- Theoretical issues without proof of concept
- Already known or documented issues

## Reward Structure

All rewards are paid in **PAW tokens** from the project development fund.

### Severity Tiers

| Severity | PAW Tokens | Vesting |
|----------|------------|---------|
| Critical | 50,000 - 100,000 PAW | 6-month linear vest |
| High | 15,000 - 50,000 PAW | 3-month linear vest |
| Medium | 5,000 - 15,000 PAW | None |
| Low | 1,000 - 5,000 PAW | None |

### Critical Severity

**Direct loss of funds, consensus failure, or network compromise**

Examples:
- Double-spend vulnerabilities bypassing consensus
- Private key extraction or recovery
- Remote code execution on validator nodes
- Consensus-breaking bugs causing chain halt
- Unlimited token minting or burning
- Theft from liquidity pools or user accounts

### High Severity

**Significant security impact with realistic exploitation**

Examples:
- Authentication bypass for privileged functions
- Privilege escalation in governance or validators
- Oracle manipulation affecting price data
- DoS attacks halting specific modules
- Cryptographic weaknesses in verification
- Incorrect slashing of honest validators

### Medium Severity

**Limited impact requiring specific conditions**

Examples:
- Non-critical information disclosure
- Limited DoS affecting single node
- Fee calculation or rounding errors
- TWAP manipulation under specific conditions
- Memory leaks in specific scenarios

### Low Severity

**Minimal impact, security hardening**

Examples:
- Best practice violations without direct vulnerability
- Verbose error messages (no sensitive data)
- Defense-in-depth improvements
- Non-exploitable edge cases
- Configuration hardening suggestions

### Reward Modifiers

| Modifier | Adjustment |
|----------|------------|
| High-quality report with PoC | +25% |
| Actionable fix suggestion | +15% |
| Multiple chained vulnerabilities | +50% |
| First critical finding | +25% |
| Incomplete report | -25% to -50% |
| Public disclosure before fix | Disqualified |

## Vesting Terms

To protect PAW token economics and align incentives:

- **Critical rewards**: 6-month linear vesting (tokens released monthly)
- **High rewards**: 3-month linear vesting
- **Medium/Low rewards**: No vesting, immediate transfer

Vesting begins after the vulnerability fix is deployed to mainnet. Researchers may opt out of vesting for a 50% reduced reward.

## Submission Process

### How to Submit

1. **Preferred**: GitHub Security Advisory at our repository
2. **Alternative**: Encrypted email to security@paw-chain.io
3. **PGP Key**: Available in repository at `/docs/security/PGP_KEY.asc`

### Required Information

```markdown
## Summary
Brief description of the vulnerability

## Severity Assessment
Your assessment with justification

## Affected Components
Specific modules, functions, or endpoints

## Reproduction Steps
1. Step one
2. Step two
3. ...

## Proof of Concept
Code, commands, or demonstration

## Impact
What an attacker could achieve

## Suggested Fix (Optional)
Proposed remediation

## Contact
Your name/handle and PAW address for payment
```

### Response Timeline

| Severity | Acknowledgment | Assessment | Patch Target |
|----------|----------------|------------|--------------|
| Critical | 24 hours | 48 hours | 7 days |
| High | 48 hours | 5 days | 14 days |
| Medium | 72 hours | 7 days | 30 days |
| Low | 1 week | 14 days | Next release |

## Responsible Disclosure

### Expectations

- Report vulnerabilities privately first
- Avoid actions that harm users or the network
- Test on testnet or local instances only
- Do not access, modify, or exfiltrate user data
- Allow reasonable time for investigation and fix
- Keep details confidential until coordinated disclosure

### Coordinated Disclosure Timeline

1. **Day 0**: Vulnerability reported privately
2. **Day 1-7**: Acknowledgment and validation
3. **Day 7-90**: Patch development and testing
4. **Day 90**: Coordinated public disclosure
5. **Post-fix**: Security advisory and researcher credit

## Legal Safe Harbor

PAW provides legal safe harbor for good-faith security research. We will not pursue legal action if you:

- Make good faith effort to comply with this policy
- Avoid privacy violations, data destruction, and service disruption
- Only interact with accounts you own or have permission to test
- Report vulnerabilities promptly and privately
- Do not exploit beyond demonstrating the issue

## Recognition

### Hall of Fame

Security researchers who contribute will be recognized:

| Tier | Criteria |
|------|----------|
| Guardian | 1+ Critical finding |
| Champion | 3+ High severity findings |
| Contributor | Any valid finding |

### Recognition Options

- Public attribution in Hall of Fame
- Credit in security advisories
- Contributor badge for community platforms
- Recommendation letter upon request
- Anonymous recognition if preferred

## Program Rules

### Eligibility

- Open globally (subject to legal restrictions)
- Must be 18+ or have parental consent
- PAW core team and contractors are ineligible
- First valid report receives full reward

### Payment

- Rewards paid in PAW tokens only
- Transfer to provided PAW address after fix deployment
- Subject to vesting schedule for larger rewards
- Researchers responsible for tax obligations

### Modifications

- Program terms may change with 30 days notice
- Pending submissions evaluated under terms at submission time
- Reward amounts may adjust based on development fund

## Contact

- **Security Email**: security@paw-chain.io
- **PGP Key**: `/docs/security/PGP_KEY.asc`
- **GitHub Advisories**: Repository security tab
- **Response Time**: 24-72 hours based on severity

## FAQ

**Q: Why token-only rewards?**
A: As a community-funded project, our development fund holds PAW tokens. This aligns researcher incentives with project success.

**Q: Can I sell tokens immediately?**
A: Medium/Low rewards have no restrictions. Critical/High rewards vest over time to protect token economics.

**Q: What if I find a dependency vulnerability?**
A: Report to the upstream project. If it has PAW-specific impact, inform us for potential reduced reward.

**Q: Can I test on mainnet?**
A: No. Use testnet or local instances. Mainnet exploitation disqualifies you.

---

**Last Updated**: January 1, 2026
**Program Version**: 2.0
**Program Status**: Active

*Thank you for helping keep PAW secure!*
