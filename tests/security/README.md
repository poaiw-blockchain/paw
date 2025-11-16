# PAW Blockchain Security Test Suite

Comprehensive security testing suite for the PAW blockchain.

## Overview

This directory contains security-focused tests designed to identify vulnerabilities, validate security controls, and ensure the PAW blockchain is resistant to common attack vectors.

## Test Files

### Authentication & Authorization (`auth_test.go`)

Tests for access control, permission management, and authentication mechanisms.

**Key Test Cases:**

- Authentication bypass attempts
- Authorization escalation
- Module account access control
- Rate limiting
- Session management
- Reentrancy protection

**Run:**

```bash
go test -run TestAuthSecurityTestSuite -v
```

### Input Validation & Injection (`injection_test.go`)

Tests for input sanitization and injection attack prevention.

**Attack Vectors Covered:**

- SQL/NoSQL Injection
- Command Injection
- Cross-Site Scripting (XSS)
- Path Traversal
- Buffer Overflow
- Integer Overflow
- Format String Injection
- Unicode Normalization
- ReDoS (Regular Expression DoS)

**Run:**

```bash
go test -run TestInjectionSecurityTestSuite -v
```

### Cryptography (`crypto_test.go`)

Tests for cryptographic security and key management.

**Validation Areas:**

- Entropy and randomness quality
- Key generation (secp256k1)
- Signature verification
- BIP39 mnemonic generation
- BIP32/44 HD key derivation
- Hash collision resistance
- Timing attack resistance

**Run:**

```bash
go test -run TestCryptoSecurityTestSuite -v
```

### Adversarial Scenarios (`adversarial_test.go`)

Simulations of real-world attack scenarios and game theory exploits.

**Attack Simulations:**

- Double-spending
- Front-running & MEV
- Sandwich attacks
- Flash loan attacks
- Oracle manipulation
- Sybil attacks
- Replay attacks
- Eclipse attacks
- Censorship resistance

**Run:**

```bash
go test -run TestAdversarialTestSuite -v
```

## Quick Start

### Run All Security Tests

```bash
# Standard run
go test ./tests/security/... -v

# With race detection
go test ./tests/security/... -race -v

# With coverage
go test ./tests/security/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Run Specific Test Suite

```bash
# Authentication tests only
go test ./tests/security/ -run TestAuthSecurityTestSuite -v

# Injection tests only
go test ./tests/security/ -run TestInjectionSecurityTestSuite -v

# Cryptography tests only
go test ./tests/security/ -run TestCryptoSecurityTestSuite -v

# Adversarial tests only
go test ./tests/security/ -run TestAdversarialTestSuite -v
```

### Run Individual Test

```bash
# Test double-spending prevention
go test ./tests/security/ -run TestDoubleSpending_Prevention -v

# Test SQL injection protection
go test ./tests/security/ -run TestSQLInjection_TokenName -v

# Test BIP39 compliance
go test ./tests/security/ -run TestBIP39_MnemonicGeneration -v
```

## Test Statistics

| Test Suite     | Test Cases | Coverage Area                  |
| -------------- | ---------- | ------------------------------ |
| Authentication | 15+        | Access control, permissions    |
| Injection      | 20+        | Input validation, sanitization |
| Cryptography   | 12+        | Key management, signatures     |
| Adversarial    | 15+        | Attack simulations             |
| **Total**      | **60+**    | **Comprehensive security**     |

## Integration with CI/CD

These tests are automatically run in the GitHub Actions workflow:

- **On Push**: To master/main/develop branches
- **On Pull Request**: All PRs must pass security tests
- **Scheduled**: Daily at 2 AM UTC
- **Manual**: Via workflow_dispatch

See `.github/workflows/security.yml` for configuration.

## Test Requirements

### Go Version

- Go 1.23 or higher

### Dependencies

```bash
go mod download
```

### Environment Setup

No special environment variables required. Tests use in-memory databases and mock contexts.

## Writing New Security Tests

### Test Structure

```go
package security_test

import (
    "testing"
    "github.com/stretchr/testify/suite"
    "github.com/paw-chain/paw/app"
)

type MySecurityTestSuite struct {
    suite.Suite
    app *app.App
    ctx sdk.Context
}

func (suite *MySecurityTestSuite) SetupTest() {
    suite.app, suite.ctx = keepertest.SetupTestApp(suite.T())
}

func TestMySecurityTestSuite(t *testing.T) {
    suite.Run(t, new(MySecurityTestSuite))
}

func (suite *MySecurityTestSuite) TestMySecurityCheck() {
    // Test implementation
    suite.Require().NotNil(suite.app)
}
```

### Best Practices

1. **Use Testify Suites** - For setup/teardown and shared state
2. **Test Negative Cases** - Ensure attacks are prevented
3. **Descriptive Names** - Clear test names describing what's tested
4. **Comprehensive Coverage** - Test edge cases and boundaries
5. **Document Assumptions** - Comment expected behavior
6. **Isolate Tests** - Each test should be independent
7. **Fast Execution** - Keep tests under 30s each

### Adding New Tests

1. Identify security concern
2. Write failing test demonstrating vulnerability
3. Implement fix in code
4. Verify test passes
5. Add to appropriate test suite file
6. Update this README

## Common Issues

### Test Failures

If tests fail, check:

1. **Dependencies** - Run `go mod download`
2. **Go Version** - Ensure Go 1.23+
3. **Module Cache** - Clear with `go clean -modcache`
4. **Race Conditions** - Run with `-race` flag
5. **Test Isolation** - Ensure tests don't share mutable state

### Performance

If tests run slow:

1. **Parallel Execution** - Use `t.Parallel()` where safe
2. **Table-Driven Tests** - Reduce boilerplate
3. **Mock Heavy Operations** - Mock expensive operations
4. **Profile Tests** - Use `go test -cpuprofile`

## Security Test Checklist

Before deployment, ensure:

- [ ] All security tests pass
- [ ] No race conditions detected
- [ ] Coverage > 80% for security-critical code
- [ ] All injection vectors tested
- [ ] Cryptographic compliance validated
- [ ] Adversarial scenarios simulated
- [ ] Fuzzing completed without crashes
- [ ] No TODO/FIXME in security code
- [ ] Code review completed
- [ ] Security scan passed

## Related Documentation

- [Full Security Testing Guide](../../docs/SECURITY_TESTING.md)
- [Fuzzing Documentation](./fuzzing/README.md)
- [Security Scan Script](../../scripts/security-scan.sh)
- [CI/CD Workflow](../../.github/workflows/security.yml)

## Contributing

When contributing security tests:

1. Follow existing test patterns
2. Ensure tests are deterministic
3. Add comprehensive documentation
4. Update this README
5. Request security team review

## Support

For security testing questions:

- Open issue with `security-testing` label
- Email: security@paw-chain.io
- Review existing tests for examples

## License

Same license as PAW blockchain project.

---

**Maintained by**: PAW Security Team
**Last Updated**: 2024-11-14
**Version**: 1.0.0
