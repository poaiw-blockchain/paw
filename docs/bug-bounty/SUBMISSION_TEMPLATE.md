# Vulnerability Report Submission Template

## Instructions

Use this template when reporting security vulnerabilities to the PAW blockchain bug bounty program. Complete all required sections and as many optional sections as possible. High-quality, detailed reports receive priority handling and may qualify for bonus rewards.

**Submission Methods**:

- **Preferred**: GitHub Security Advisory (private)
- **Alternative**: Encrypted email to security@paw-blockchain.org (use our PGP key)

---

## Report Header

### Report Information

- **Report Title**: [Brief, descriptive title]
- **Researcher Name/Handle**: [Your name or pseudonym]
- **Contact Email**: [Your email for communication]
- **Report Date**: [YYYY-MM-DD]
- **PAW Version Affected**: [Version number or git commit hash]
- **Your Severity Assessment**: [Critical/High/Medium/Low]

### Optional Information

- **Twitter/GitHub Handle**: [For recognition in Hall of Fame]
- **Website/Blog**: [If you want attribution]
- **Attribution Preference**: [Full / Pseudonymous / Anonymous]
- **Disclosure Timeline**: [Any constraints on your end]

---

## Executive Summary

**Required**: Provide a brief (2-4 sentence) summary of the vulnerability.

```
Example:
A signature verification bypass in the x/bank module allows an attacker to
create valid-looking transactions without possessing the corresponding private
key. This vulnerability affects all versions since v1.0.0 and could lead to
unauthorized fund transfers. The attack can be executed remotely without any
privileges or user interaction.
```

[Your summary here]

---

## Vulnerability Details

### 1. Vulnerability Type

**Required**: Select the primary category (check one):

- [ ] Consensus/Protocol Vulnerability
- [ ] Cryptographic Vulnerability
- [ ] Authentication/Authorization Bypass
- [ ] Input Validation/Injection
- [ ] Business Logic Error
- [ ] Denial of Service
- [ ] Information Disclosure
- [ ] Privilege Escalation
- [ ] Race Condition/Concurrency Issue
- [ ] Memory Safety (buffer overflow, use-after-free, etc.)
- [ ] Configuration/Deployment Issue
- [ ] Other: [Please specify]

### 2. Affected Components

**Required**: List all affected components, modules, or files:

```
Example:
- Module: x/bank/keeper/send.go (lines 123-145)
- API Endpoint: /cosmos/bank/v1beta1/send
- Dependencies: github.com/cosmos/cosmos-sdk v0.50.1
```

- Component 1: [Path/Module/Function]
- Component 2: [Path/Module/Function]
- Additional components: [List all]

### 3. Affected Versions

**Required**: Specify which versions are vulnerable:

- **First Vulnerable Version**: [e.g., v1.0.0 or commit hash]
- **Last Vulnerable Version**: [e.g., v1.2.3 or "latest"]
- **Tested On**: [Specific version you tested]

### 4. Root Cause Analysis

**Required**: Explain the underlying cause of the vulnerability:

```
Example:
The SendCoins function in x/bank/keeper/send.go fails to verify the signature
against the public key when the memo field exceeds 256 characters. This occurs
because the signature verification is skipped when processing large memos due
to a misplaced early return statement at line 135.
```

[Your analysis here]

### 5. Technical Description

**Required**: Provide detailed technical explanation of the vulnerability:

[Detailed technical description including code snippets, data structures, protocol flows, etc.]

---

## Exploitation Details

### 6. Prerequisites

**Required**: What conditions must be met for exploitation?

- [ ] No prerequisites (attack works in default configuration)
- [ ] Specific configuration: [Describe]
- [ ] Network conditions: [Describe]
- [ ] Timing requirements: [Describe]
- [ ] Other prerequisites: [Describe]

### 7. Attack Complexity

**Required**: How difficult is this vulnerability to exploit?

- [ ] Low: Anyone can exploit with minimal effort
- [ ] Medium: Requires some technical knowledge or specific conditions
- [ ] High: Requires significant expertise or rare conditions

**Justification**: [Explain your assessment]

### 8. Step-by-Step Reproduction

**Required**: Provide detailed steps to reproduce the vulnerability:

```
Example:

1. Set up a local PAW node using the following command:
   ./pawd start --home ./testnet

2. Create a test account:
   ./pawcli keys add attacker

3. Craft a malicious transaction with oversized memo:
   ./pawcli tx bank send [victim] [attacker] 1000upaw \
     --memo "$(python -c 'print("A"*300)')" \
     --from attacker

4. Observe that the transaction is accepted without valid signature

5. Verify funds were transferred:
   ./pawcli query bank balances [attacker]
```

[Your reproduction steps here]

### 9. Proof of Concept

**Required**: Provide working code, scripts, or detailed commands:

```python
# Example PoC script
import requests

def exploit_vulnerability():
    # PoC code here
    pass

if __name__ == "__main__":
    exploit_vulnerability()
```

[Your PoC here - can be code, curl commands, screenshots, video link, etc.]

### 10. Attack Scenario

**Required**: Describe a realistic attack scenario:

```
Example:
An attacker monitors the mempool for pending transactions. When they observe
a high-value transfer, they quickly submit their own transaction using the
vulnerability to redirect the funds to their own account. The malicious
transaction is processed before the legitimate one due to higher gas fees,
resulting in theft of the victim's funds.
```

[Your attack scenario here]

---

## Impact Assessment

### 11. Impact Description

**Required**: What can an attacker achieve?

**Direct Impact**:

- Confidentiality: [High/Medium/Low/None]
- Integrity: [High/Medium/Low/None]
- Availability: [High/Medium/Low/None]

**Detailed Impact**:
[Describe the specific impact on the system, users, validators, etc.]

### 12. Scope of Impact

**Required**: Who or what is affected?

- [ ] All network participants
- [ ] Validators only
- [ ] Token holders
- [ ] DEX users
- [ ] Specific module users
- [ ] Node operators
- [ ] Other: [Specify]

**Estimated Affected Population**: [Percentage or number if known]

### 13. Financial Impact

**Optional but helpful**: Estimate potential financial losses:

- **Worst Case Scenario**: [Dollar amount or % of TVL]
- **Likely Case**: [More realistic estimate]
- **Assumptions**: [Explain your calculations]

### 14. Exploitability Assessment

**Required**: Is this actively exploited or easily exploitable?

- [ ] Actively being exploited in the wild
- [ ] Easily exploitable by anyone
- [ ] Exploitable with moderate effort
- [ ] Difficult to exploit (theoretical)
- [ ] No known exploitation method

**Evidence**: [Any evidence of exploitation or monitoring data]

---

## CVSS Scoring

### 15. CVSS 3.1 Assessment

**Optional but recommended**: Calculate CVSS score using https://www.first.org/cvss/calculator/3.1

**CVSS Vector String**: [e.g., CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H]

**CVSS Base Score**: [0.0-10.0]

**Metric Breakdown**:

- Attack Vector (AV): [Network/Adjacent/Local/Physical]
- Attack Complexity (AC): [Low/High]
- Privileges Required (PR): [None/Low/High]
- User Interaction (UI): [None/Required]
- Scope (S): [Unchanged/Changed]
- Confidentiality (C): [None/Low/High]
- Integrity (I): [None/Low/High]
- Availability (A): [None/Low/High]

---

## Evidence and Supporting Materials

### 16. Screenshots/Videos

**Optional**: Include visual evidence if applicable:

- Screenshot 1: [Description and link/attachment]
- Video demonstration: [Link to unlisted YouTube/Vimeo]
- Logs: [Relevant log excerpts]

### 17. Additional Evidence

**Optional**: Any additional supporting materials:

- Transaction hashes (testnet only)
- Network captures
- Core dumps
- Debug logs
- Related issues or CVEs

---

## Remediation Suggestions

### 18. Proposed Fix

**Optional but appreciated**: Suggest how to fix the vulnerability:

```
Example:

The fix should include:
1. Move signature verification before memo processing
2. Add explicit signature validation regardless of memo size
3. Add unit tests for edge cases with large memos

Suggested code change:
// Before processing memo, validate signature
if !VerifySignature(tx.Signature, tx.SignBytes()) {
    return sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "invalid signature")
}
```

[Your suggestions here]

### 19. Mitigation/Workarounds

**Optional**: Temporary mitigations while fix is developed:

[Describe any workarounds or temporary mitigations]

### 20. Testing Recommendations

**Optional**: Suggestions for testing the fix:

[Recommended test cases, regression tests, etc.]

---

## Additional Context

### 21. Discovery Timeline

**Optional**: When and how did you discover this?

- **Discovery Date**: [YYYY-MM-DD]
- **Discovery Method**: [Code audit, fuzzing, incident response, etc.]
- **Time Invested**: [Approximate hours spent researching]

### 22. Related Vulnerabilities

**Optional**: Are there related issues?

- Similar vulnerabilities in other modules
- Variants of this vulnerability
- Links to similar CVEs or security advisories

### 23. References

**Optional**: Relevant references, research, or documentation:

- Research papers
- Blog posts
- Similar vulnerabilities
- Relevant documentation

### 24. Additional Notes

**Optional**: Any other information you think is relevant:

[Additional notes, observations, or comments]

---

## Researcher Declaration

### 25. Testing Declaration

**Required**: Confirm responsible testing practices:

I declare that:

- [ ] I have only tested on testnet or local instances
- [ ] I have not accessed or exfiltrated user data
- [ ] I have not disrupted the production network
- [ ] I have not disclosed this vulnerability publicly
- [ ] I have followed the bug bounty program terms
- [ ] I understand the coordinated disclosure policy

### 26. Originality Declaration

**Required**: Confirm this is your original work:

- [ ] This is my original research
- [ ] This has not been reported elsewhere
- [ ] I am the first to discover this vulnerability (to my knowledge)

### 27. Disclosure Consent

**Required**: Publication and attribution preferences:

I consent to:

- [ ] PAW publishing a security advisory after the fix
- [ ] Being credited in the security advisory
- [ ] Being listed in the Hall of Fame

Attribution preference:

- [ ] Full name and links: [Provide details]
- [ ] Pseudonym only: [Specify handle]
- [ ] Anonymous (no public attribution)

---

## Submission Checklist

Before submitting, verify you have:

- [ ] Completed all required sections
- [ ] Provided clear reproduction steps
- [ ] Included working proof of concept
- [ ] Assessed severity and impact
- [ ] Tested responsibly (not on mainnet)
- [ ] Not disclosed publicly
- [ ] Provided contact information
- [ ] Encrypted sensitive information (if via email)

---

## For PAW Security Team Use Only

**Do not fill out this section - for internal use**

- **Tracking ID**: [Assigned by security team]
- **Received Date**: [YYYY-MM-DD HH:MM UTC]
- **Assigned To**: [Security team member]
- **Triage Date**: [YYYY-MM-DD]
- **Validated**: [Yes/No]
- **Confirmed Severity**: [Critical/High/Medium/Low]
- **Approved Reward**: [Amount]
- **Fix Status**: [Planned/In Progress/Deployed]
- **Advisory ID**: [CVE or internal ID]

---

## Contact Information

If you have questions about this template or the submission process:

- **Email**: bugbounty@paw-blockchain.org
- **PGP Key**: https://paw-blockchain.org/security.asc
- **Documentation**: https://github.com/[PAW-ORG]/paw/docs/BUG_BOUNTY.md

---

**Thank you for helping make PAW more secure!**

**Template Version**: 1.0
**Last Updated**: November 14, 2025
