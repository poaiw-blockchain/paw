# PAW Security Testing - Quick Reference Card

## Running Tests

### All Security Tests

```bash
go test ./tests/security/... -v
```

### Specific Test Suites

```bash
# Authentication & Authorization
go test ./tests/security/ -run TestAuthSecurityTestSuite -v

# Input Validation & Injection
go test ./tests/security/ -run TestInjectionSecurityTestSuite -v

# Cryptography
go test ./tests/security/ -run TestCryptoSecurityTestSuite -v

# Adversarial Attacks
go test ./tests/security/ -run TestAdversarialTestSuite -v
```

### With Coverage

```bash
go test ./tests/security/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### With Race Detection

```bash
go test ./tests/security/... -race -v
```

## Security Scanning

### Quick Scan

```bash
./scripts/security-scan.sh --quick
```

### Full Scan with Report

```bash
./scripts/security-scan.sh --full --report
```

### CI Mode

```bash
./scripts/security-scan.sh --ci
```

## Fuzzing

### Run Fuzz Test

```bash
go test -fuzz=FuzzCreatePoolMsg -fuzztime=30s ./tests/security/fuzzing/
```

### Run All Fuzzing

```bash
go test -fuzz=. -fuzztime=1m ./tests/security/fuzzing/
```

## Individual Security Tools

### GoSec

```bash
gosec ./...
```

### Staticcheck

```bash
staticcheck ./...
```

### govulncheck

```bash
govulncheck ./...
```

### Gitleaks

```bash
gitleaks detect --source=. --no-git
```

## CI/CD

### Trigger Security Workflow

```bash
# Push to main branch
git push origin main

# Or manually trigger
gh workflow run security.yml
```

### View Results

- GitHub Actions → Security Testing workflow
- Security tab for SARIF results
- Download artifacts for detailed reports

## File Locations

| Item          | Location                         |
| ------------- | -------------------------------- |
| Test Suite    | `tests/security/`                |
| Fuzzing       | `tests/security/fuzzing/`        |
| Scan Script   | `scripts/security-scan.sh`       |
| CI Workflow   | `.github/workflows/security.yml` |
| Documentation | `docs/SECURITY_TESTING.md`       |
| Quick Start   | `tests/security/README.md`       |

## Test Statistics

- **Total Tests**: 60+
- **Test Files**: 4
- **Lines of Code**: ~2,000+
- **Coverage Areas**: Auth, Injection, Crypto, Adversarial

## Common Commands

```bash
# Run specific test
go test ./tests/security/ -run TestDoubleSpending_Prevention -v

# Run with timeout
go test ./tests/security/... -timeout 15m -v

# Run parallel
go test ./tests/security/... -parallel 4 -v

# Generate coverage badge
go test ./tests/security/... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
```

## Quick Checks Before Commit

```bash
# 1. Run security tests
go test ./tests/security/... -v

# 2. Check for secrets
gitleaks detect --source=. --no-git

# 3. Run quick security scan
./scripts/security-scan.sh --quick

# 4. Check coverage
go test ./tests/security/... -cover
```

## Emergency Response

### Found Security Issue?

1. DO NOT commit or push
2. Email: security@paw-chain.io
3. Include: description, reproduction steps, impact
4. Wait for response before disclosure

### Test Failure in CI?

1. Check Actions tab for details
2. Download artifacts for full reports
3. Fix critical/high issues before merge
4. Document medium/low issues

## Help

- Full Guide: `docs/SECURITY_TESTING.md`
- Test README: `tests/security/README.md`
- Fuzzing Guide: `tests/security/fuzzing/README.md`
- Support: security@paw-chain.io

---

Version: 1.0.0 | Updated: 2024-11-14
