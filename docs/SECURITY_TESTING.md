# PAW Blockchain Security Testing Guide

Comprehensive guide for security testing the PAW blockchain implementation.

## Table of Contents

1. [Overview](#overview)
2. [Security Test Suite](#security-test-suite)
3. [Running Security Tests](#running-security-tests)
4. [Test Categories](#test-categories)
5. [Fuzzing](#fuzzing)
6. [Automated Security Scanning](#automated-security-scanning)
7. [CI/CD Integration](#cicd-integration)
8. [Vulnerability Reporting](#vulnerability-reporting)
9. [Best Practices](#best-practices)
10. [Compliance](#compliance)

## Overview

The PAW blockchain security testing suite provides comprehensive coverage for:

- **Authentication & Authorization** - Access control and permission escalation
- **Input Validation** - Injection attacks and malicious inputs
- **Cryptography** - Key generation, signatures, and entropy
- **Adversarial Scenarios** - Attack simulations and game theory
- **Fuzzing** - Automated input generation for edge cases
- **Dependency Security** - Third-party vulnerability scanning

### Security Goals

1. **Prevent Unauthorized Access** - Strong authentication and authorization
2. **Input Sanitization** - Reject malicious or malformed inputs
3. **Cryptographic Security** - Industry-standard cryptographic primitives
4. **Attack Resistance** - Resilience against known attack vectors
5. **Data Integrity** - Tamper-proof transaction and state management
6. **Availability** - DoS/DDoS attack resistance

## Security Test Suite

### Test Structure

```
tests/security/
├── auth_test.go          # Authentication & authorization tests
├── injection_test.go     # Input validation & injection tests
├── crypto_test.go        # Cryptographic security tests
├── adversarial_test.go   # Attack simulation tests
└── fuzzing/              # Fuzzing tests
    └── README.md         # Fuzzing documentation
```

### Test Coverage

| Test File           | Coverage Area                                                  | Test Count | Priority |
| ------------------- | -------------------------------------------------------------- | ---------- | -------- |
| auth_test.go        | Authentication bypass, authorization escalation, rate limiting | 15+        | Critical |
| injection_test.go   | SQL/NoSQL injection, XSS, command injection, buffer overflow   | 20+        | Critical |
| crypto_test.go      | Key generation, signatures, entropy, BIP39/32 compliance       | 12+        | Critical |
| adversarial_test.go | Double-spending, front-running, Sybil attacks, replay attacks  | 15+        | High     |

## Running Security Tests

### Quick Start

```bash
# Run all security tests
go test ./tests/security/... -v

# Run with race detection
go test ./tests/security/... -race -v

# Run with coverage
go test ./tests/security/... -cover -coverprofile=security-coverage.out

# View coverage report
go tool cover -html=security-coverage.out
```

### Individual Test Suites

```bash
# Authentication tests
go test ./tests/security/ -run TestAuthSecurityTestSuite -v

# Injection tests
go test ./tests/security/ -run TestInjectionSecurityTestSuite -v

# Cryptography tests
go test ./tests/security/ -run TestCryptoSecurityTestSuite -v

# Adversarial tests
go test ./tests/security/ -run TestAdversarialTestSuite -v
```

### Specific Test Cases

```bash
# Test authentication bypass
go test ./tests/security/ -run TestAuthenticationBypass_InvalidSignature -v

# Test SQL injection
go test ./tests/security/ -run TestSQLInjection_TokenName -v

# Test double-spending
go test ./tests/security/ -run TestDoubleSpending_Prevention -v

# Test BIP39 compliance
go test ./tests/security/ -run TestBIP39_MnemonicGeneration -v
```

## Test Categories

### 1. Authentication & Authorization Tests

**File:** `tests/security/auth_test.go`

**Test Cases:**

- Authentication bypass with invalid signatures
- Authorization escalation via module account access
- Cross-module access control
- Permission escalation in pool creation
- Rate limiting and transaction spam
- Session management with stale contexts
- Oracle submission access control
- Compute provider registration authorization
- Timestamp manipulation attacks
- Reentrancy protection in DEX swaps
- Governance parameter bypass attempts
- Resource exhaustion DoS attacks

**Example:**

```go
func (suite *AuthSecurityTestSuite) TestAuthenticationBypass_InvalidSignature() {
    // Test that invalid signatures are rejected
}
```

### 2. Input Validation & Injection Tests

**File:** `tests/security/injection_test.go`

**Attack Vectors Tested:**

- SQL Injection (token names, asset pairs)
- Command Injection (compute endpoints)
- Path Traversal (endpoint URLs)
- Cross-Site Scripting (asset names)
- Buffer Overflow (long inputs)
- Integer Overflow (amount fields)
- Null Byte Injection
- Format String Injection
- Unicode Normalization Attacks
- Regular Expression DoS (ReDoS)
- LDAP Injection
- XML/XXE Injection
- NoSQL Injection
- Special Character Validation

**Example:**

```go
func (suite *InjectionSecurityTestSuite) TestSQLInjection_TokenName() {
    sqlPayloads := []string{
        "'; DROP TABLE pools; --",
        "1' OR '1'='1",
        // ...
    }
    // Test each payload is rejected
}
```

### 3. Cryptographic Security Tests

**File:** `tests/security/crypto_test.go`

**Test Cases:**

- Entropy and randomness quality (1000+ samples)
- Secp256k1 key generation uniqueness
- Signature creation and verification
- Message tampering detection
- BIP39 mnemonic generation (multiple entropy sizes)
- Invalid mnemonic rejection
- BIP32/44 hierarchical key derivation
- Hash collision resistance (SHA3-256)
- Weak key detection
- Timing attack resistance
- Private key recovery prevention
- Nonce reuse prevention
- Address generation consistency
- Cryptographic standard compliance

**Example:**

```go
func (suite *CryptoSecurityTestSuite) TestEntropy_Randomness() {
    // Generate 1000 random samples
    // Verify no duplicates
    // Check bit distribution (45-55% for each bit)
}
```

### 4. Adversarial Attack Tests

**File:** `tests/security/adversarial_test.go`

**Attack Scenarios:**

- Double-spending attempts
- Front-running simulation
- Sandwich attacks (MEV)
- Flash loan attacks
- Oracle price manipulation
- Sybil attacks
- Time bandit attacks
- Long-range attacks
- Griefing and resource exhaustion
- Replay attacks
- Eclipse attacks
- Selfish mining
- Transaction censorship

**Example:**

```go
func (suite *AdversarialTestSuite) TestDoubleSpending_Prevention() {
    // Attempt to spend same funds twice
    // Verify balance tracking prevents double-spend
}
```

## Fuzzing

### Overview

Fuzzing uses automated input generation to discover edge cases, crashes, and vulnerabilities.

**Location:** `tests/security/fuzzing/`

### Running Fuzzing Tests

```bash
# Run a specific fuzz test
go test -fuzz=FuzzCreatePoolMsg -fuzztime=30s ./tests/security/fuzzing/

# Run with longer duration
go test -fuzz=FuzzCreatePoolMsg -fuzztime=5m ./tests/security/fuzzing/

# Run all fuzz tests
go test -fuzz=. -fuzztime=1m ./tests/security/fuzzing/
```

### Fuzz Test Targets

1. **DEX Module**
   - `MsgCreatePool` - Pool creation parameters
   - `MsgSwap` - Swap amounts and slippage
   - `MsgAddLiquidity` - Liquidity ratios
   - `MsgRemoveLiquidity` - Withdrawal amounts

2. **Oracle Module**
   - `MsgSubmitPrice` - Price values and precision
   - `MsgRegisterOracle` - Oracle addresses

3. **Compute Module**
   - `MsgRegisterProvider` - Endpoint validation
   - `MsgSubmitJob` - Job parameters

4. **Protobuf Messages**
   - Message unmarshaling
   - Field validation
   - Nested structures

### Analyzing Fuzz Results

When a crash is found:

```bash
# Crashes are saved to testdata/fuzz/FuzzTestName/
ls testdata/fuzz/FuzzCreatePoolMsg/

# Reproduce the crash
go test -run=FuzzCreatePoolMsg/crash-hash

# Minimize the crashing input
go test -fuzz=FuzzCreatePoolMsg -run=FuzzCreatePoolMsg/crash-hash
```

See `tests/security/fuzzing/README.md` for detailed fuzzing documentation.

## Automated Security Scanning

### Security Scan Script

**Location:** `scripts/security-scan.sh`

### Usage

```bash
# Quick scan (critical issues only)
./scripts/security-scan.sh --quick

# Full comprehensive scan
./scripts/security-scan.sh --full

# Generate HTML report
./scripts/security-scan.sh --full --report

# CI mode (exit with error if issues found)
./scripts/security-scan.sh --ci
```

### Security Tools Used

| Tool            | Purpose                          | Severity                    |
| --------------- | -------------------------------- | --------------------------- |
| **GoSec**       | Security vulnerability scanner   | Critical, High, Medium, Low |
| **Staticcheck** | Static analysis                  | Medium                      |
| **govulncheck** | Go vulnerability database        | Critical, High              |
| **Nancy**       | Dependency vulnerability scanner | High, Medium                |
| **Gitleaks**    | Secret detection                 | Critical                    |
| **go vet**      | Go standard checks               | Medium                      |
| **ineffassign** | Ineffectual assignments          | Low                         |
| **errcheck**    | Unchecked errors                 | Medium                      |

### Report Outputs

Reports are saved to `security-reports/` directory:

```
security-reports/
├── gosec_TIMESTAMP.json
├── staticcheck_TIMESTAMP.txt
├── govulncheck_TIMESTAMP.txt
├── nancy_TIMESTAMP.txt
├── gitleaks_TIMESTAMP.json
├── security_report_TIMESTAMP.html
└── ...
```

## CI/CD Integration

### GitHub Actions Workflow

**Location:** `.github/workflows/security.yml`

### Workflow Jobs

1. **security-tests** - Run security test suite
2. **gosec** - Security vulnerability scanning
3. **govulncheck** - Go vulnerability database check
4. **staticcheck** - Static analysis
5. **nancy** - Dependency scanning
6. **gitleaks** - Secret scanning
7. **semgrep** - Semantic code analysis
8. **codeql** - Advanced security analysis
9. **trivy** - Container/filesystem scanning
10. **fuzzing** - Automated fuzzing (scheduled)

### Trigger Events

- **Push** to master/main/develop branches
- **Pull Request** to master/main/develop
- **Schedule** - Daily at 2 AM UTC
- **Manual** - workflow_dispatch

### Viewing Results

1. Navigate to **Actions** tab in GitHub
2. Select **Security Testing** workflow
3. View individual job results
4. Download artifacts for detailed reports
5. Check **Security** tab for SARIF uploads

## Vulnerability Reporting

### Responsible Disclosure

If you discover a security vulnerability in PAW blockchain:

1. **DO NOT** open a public GitHub issue
2. Email security details to: security@paw-chain.io
3. Include:
   - Vulnerability description
   - Steps to reproduce
   - Impact assessment
   - Suggested fixes (optional)
4. Allow 90 days for patching before public disclosure
5. Coordinate disclosure timeline with maintainers

### Security Advisory Process

1. **Report Received** - Acknowledge within 48 hours
2. **Validation** - Confirm vulnerability within 7 days
3. **Patching** - Develop and test fix
4. **Disclosure** - Coordinate public disclosure
5. **Recognition** - Credit reporter in security advisory

## Best Practices

### Development

1. **Write Security Tests First** - TDD for security-critical code
2. **Code Review** - All security code requires 2+ reviewers
3. **Input Validation** - Validate all external inputs
4. **Least Privilege** - Grant minimum required permissions
5. **Defense in Depth** - Multiple layers of security
6. **Fail Securely** - Default deny on errors
7. **Keep Dependencies Updated** - Regular dependency updates
8. **Use Safe APIs** - Avoid unsafe packages and functions

### Testing

1. **Run Security Tests Locally** - Before every commit
2. **Check Coverage** - Maintain >80% security test coverage
3. **Fuzz Regularly** - Weekly fuzzing sessions
4. **Review Scan Results** - Address all critical/high findings
5. **Test Edge Cases** - Boundary conditions and error paths
6. **Simulate Attacks** - Regular adversarial testing
7. **Monitor Dependencies** - Check for CVEs weekly

### Deployment

1. **Security Scan Before Release** - Full scan on release candidates
2. **Penetration Testing** - Third-party pentests before mainnet
3. **Bug Bounty** - Run continuous bug bounty program
4. **Incident Response Plan** - Documented response procedures
5. **Security Monitoring** - Real-time threat detection
6. **Regular Audits** - Annual security audits

## Compliance

### Standards Compliance

The security test suite validates compliance with:

1. **BIP39** - Mnemonic code for generating deterministic keys
2. **BIP32** - Hierarchical deterministic wallets
3. **BIP44** - Multi-account hierarchy for deterministic wallets
4. **Cosmos SDK Security Best Practices**
5. **OWASP Top 10** - Web application security risks
6. **CWE Top 25** - Most dangerous software weaknesses

### Compliance Tests

```bash
# Test BIP39 compliance
go test ./tests/security/ -run TestBIP39 -v

# Test BIP32/44 compliance
go test ./tests/security/ -run TestBIP32_HDKeyDerivation -v

# Test cryptographic standards
go test ./tests/security/ -run TestCryptographicPrimitives_Standards -v
```

### Audit Checklist

- [ ] All security tests passing
- [ ] No critical or high vulnerabilities
- [ ] Fuzzing completed with no crashes
- [ ] Dependency scan shows no known CVEs
- [ ] No hardcoded secrets or credentials
- [ ] Cryptographic primitives use industry standards
- [ ] Input validation on all external inputs
- [ ] Access control properly enforced
- [ ] Rate limiting implemented
- [ ] Error messages don't leak sensitive info
- [ ] Logging captures security events
- [ ] Incident response plan documented

## Appendix

### Common Vulnerabilities

1. **Integer Overflow/Underflow** - Arithmetic on amounts
2. **Reentrancy** - State changes before external calls
3. **Access Control** - Insufficient permission checks
4. **Input Validation** - Unvalidated user input
5. **Cryptographic Weaknesses** - Weak or custom crypto
6. **DoS** - Resource exhaustion
7. **Front-Running** - Transaction ordering attacks
8. **Oracle Manipulation** - Price feed attacks

### Security Resources

- [Cosmos SDK Security](https://docs.cosmos.network/main/core/security)
- [OWASP Testing Guide](https://owasp.org/www-project-web-security-testing-guide/)
- [CWE Top 25](https://cwe.mitre.org/top25/)
- [Go Security Resources](https://go.dev/security/)
- [Blockchain Security Best Practices](https://consensys.github.io/smart-contract-best-practices/)

### Contact

- **Security Email**: security@paw-chain.io
- **Bug Bounty**: https://hackerone.com/paw-chain
- **Documentation**: https://docs.paw-chain.io/security
- **GitHub Issues**: https://github.com/paw-chain/paw/issues (non-security only)

---

**Last Updated**: 2024-11-14
**Version**: 1.0.0
**Maintainer**: PAW Security Team
