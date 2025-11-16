# Security Testing Guide

## Overview

This document describes the security testing strategy and tools used for the PAW blockchain project. Security is paramount for blockchain applications, and we employ multiple layers of automated and manual security testing.

## Security Testing Strategy

### 1. Static Application Security Testing (SAST)

- **GoSec**: Go code security scanner
- **CodeQL**: Advanced semantic code analysis
- **Custom Crypto Checker**: Blockchain-specific cryptographic analysis

### 2. Dependency Security

- **Govulncheck**: Official Go vulnerability database scanner
- **Nancy**: Sonatype vulnerability scanner
- **Dependency Review**: GitHub's dependency analysis

### 3. Secret Detection

- **GitLeaks**: Secret and credential scanner
- **Custom patterns**: Blockchain-specific secret detection

### 4. Container & Infrastructure Security

- **Trivy**: Multi-purpose security scanner
- **Configuration auditing**: Docker, Kubernetes, etc.

## Security Tools

### GoSec - Go Security Scanner

**Purpose**: Identifies common security issues in Go code

**Installation**:

```bash
go install github.com/securego/gosec/v2/cmd/gosec@latest
```

**Usage**:

```bash
# Run with project config
gosec -conf security/.gosec.yml ./...

# Generate JSON report
gosec -conf security/.gosec.yml -fmt json -out gosec-report.json ./...

# Generate SARIF for GitHub
gosec -conf security/.gosec.yml -fmt sarif -out gosec.sarif ./...
```

**Configuration**: `security/.gosec.yml`

**What it checks**:

- Weak cryptography (MD5, SHA1, DES, RC4)
- Hardcoded credentials
- SQL injection
- Command injection
- File path traversal
- Insecure random number generation
- TLS configuration issues
- Unsafe Go constructs

**Blockchain-specific checks**:

- Crypto/rand vs math/rand usage
- Minimum RSA key sizes (2048 bits)
- Secure hash functions (SHA-256, SHA-512)
- ECDSA/Ed25519 usage

### Nancy - Dependency Vulnerability Scanner

**Purpose**: Checks Go dependencies against Sonatype vulnerability database

**Installation**:

```bash
# Linux/macOS
curl -L -o nancy https://github.com/sonatype-nexus-community/nancy/releases/latest/download/nancy-$(uname -s | tr '[:upper:]' '[:lower:]')-amd64
chmod +x nancy
sudo mv nancy /usr/local/bin/

# Windows
# Download from GitHub releases
```

**Usage**:

```bash
go list -json -m all | nancy sleuth
```

**Configuration**: `security/nancy-config.yml`

### Govulncheck - Official Go Vulnerability Scanner

**Purpose**: Scans for known vulnerabilities using Go's official vulnerability database

**Installation**:

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
```

**Usage**:

```bash
# Scan entire project
govulncheck ./...

# Scan specific package
govulncheck ./wallet/...

# JSON output
govulncheck -json ./... > vuln-report.json
```

**Advantages**:

- Official Go team tool
- Direct integration with Go vulnerability database
- Low false positive rate
- Call stack analysis (only reports vulnerabilities in used code)

### Trivy - Multi-Purpose Security Scanner

**Purpose**: Scans for vulnerabilities in dependencies, containers, and configuration files

**Installation**:

```bash
# Linux
wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | sudo apt-key add -
echo "deb https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -sc) main" | sudo tee -a /etc/apt/sources.list.d/trivy.list
sudo apt-get update
sudo apt-get install trivy

# macOS
brew install trivy

# Windows
# Download from GitHub releases
```

**Usage**:

```bash
# Scan filesystem
trivy fs .

# Scan with specific severity
trivy fs --severity HIGH,CRITICAL .

# Scan for secrets and config issues
trivy fs --security-checks vuln,config,secret .

# Generate JSON report
trivy fs --format json --output trivy-report.json .

# Scan Docker image
trivy image paw-blockchain:latest
```

**What it scans**:

- Go module vulnerabilities
- OS package vulnerabilities (in containers)
- Configuration issues (Docker, Kubernetes, Terraform)
- Hardcoded secrets
- IaC security issues

### GitLeaks - Secret Detection

**Purpose**: Detects hardcoded secrets, API keys, and credentials

**Installation**:

```bash
# macOS
brew install gitleaks

# Linux
wget https://github.com/gitleaks/gitleaks/releases/latest/download/gitleaks-linux-amd64
chmod +x gitleaks-linux-amd64
sudo mv gitleaks-linux-amd64 /usr/local/bin/gitleaks

# Windows
# Download from GitHub releases
```

**Usage**:

```bash
# Scan current state
gitleaks detect --verbose

# Scan with report
gitleaks detect --report-path gitleaks-report.json

# Scan entire git history
gitleaks detect --verbose --log-opts="--all"

# Scan specific commit range
gitleaks detect --log-opts="main..feature-branch"
```

**What it detects**:

- AWS keys
- Private keys
- API tokens
- Database credentials
- Generic secrets (regex patterns)
- Blockchain private keys

### Custom Crypto Checker

**Purpose**: Blockchain-specific cryptographic usage analysis

**Location**: `security/crypto-check.go`

**Usage**:

```bash
go run security/crypto-check.go
```

**What it checks**:

- Weak crypto algorithm imports (MD5, SHA1, DES, RC4)
- Insecure random number generation (math/rand)
- Hardcoded secrets and keys
- Small key sizes (< 2048 bits for RSA)
- TLS InsecureSkipVerify usage
- Potential hex-encoded keys in source

**Output**: Lists issues by severity (HIGH, MEDIUM, LOW)

## Running Security Audits

### Quick Audit (Essential Checks)

```bash
# Run core security checks
make security-audit-quick

# Or manually:
gosec -conf security/.gosec.yml ./...
govulncheck ./...
gitleaks detect
```

### Full Audit (Comprehensive)

```bash
# Run full security audit
make security-audit

# Or use the script:
./scripts/security-audit.sh

# On Windows:
.\scripts\security-audit.ps1
```

The full audit runs:

1. GoSec (SAST)
2. Nancy (dependency vulnerabilities)
3. Govulncheck (Go vulnerabilities)
4. Trivy (multi-purpose scanner)
5. GitLeaks (secret detection)
6. Go mod verify (dependency integrity)
7. Custom crypto checker
8. Manual pattern checks (weak crypto, hardcoded secrets, TLS issues)

### Dependency-Only Check

```bash
# Check dependencies for vulnerabilities
make check-deps

# Or:
./scripts/check-deps.sh
```

### CI/CD Pipeline

Security checks run automatically on:

- Every push to main/master/develop
- Every pull request
- Weekly scheduled scans (Sundays at midnight UTC)
- Manual workflow dispatch

See `.github/workflows/security.yml` for details.

## Interpreting Results

### Severity Levels

- **CRITICAL**: Immediate action required, potential security breach
- **HIGH**: Significant security risk, fix urgently
- **MEDIUM**: Security concern, should be addressed soon
- **LOW**: Minor issue or best practice recommendation

### Common Issues and Remediation

#### Weak Cryptography

**Issue**: Use of MD5, SHA1, DES, or RC4

**Remediation**:

- Use SHA-256 or SHA-512 for hashing
- Use AES-256-GCM for symmetric encryption
- Use ECDSA or Ed25519 for signatures

#### Insecure Random Numbers

**Issue**: Using `math/rand` instead of `crypto/rand`

**Remediation**:

```go
// Bad
import "math/rand"
key := make([]byte, 32)
rand.Read(key)

// Good
import "crypto/rand"
key := make([]byte, 32)
rand.Read(key)
```

#### Hardcoded Secrets

**Issue**: Credentials or keys in source code

**Remediation**:

- Use environment variables
- Use secure key management (HashiCorp Vault, AWS KMS)
- Use the wallet's secure storage

#### Small Key Sizes

**Issue**: RSA keys smaller than 2048 bits

**Remediation**:

- Use at least 2048-bit RSA keys
- Prefer ECDSA (256-bit) or Ed25519 for better security/performance

#### Insecure TLS

**Issue**: `InsecureSkipVerify: true` in TLS config

**Remediation**:

- Always verify TLS certificates
- Use proper certificate management
- Only skip verification in test environments (with clear documentation)

## False Positives

### Handling False Positives

**GoSec**:

```go
// Add nosec comment with justification
// #nosec G404 - Using math/rand for non-security test data generation
rand.Intn(100)
```

**Nancy/Govulncheck**:

- Add to exclude list in `security/nancy-config.yml`
- Document why the vulnerability doesn't apply
- Consider if mitigation is possible

**GitLeaks**:

- Update `.gitleaksignore` (use sparingly)
- Ensure it's truly a false positive

## Best Practices

### Development

1. Run security checks before committing:

```bash
make security-check
```

2. Keep dependencies updated:

```bash
go get -u ./...
go mod tidy
```

3. Review security reports in PRs

4. Never commit secrets (use environment variables)

5. Use secure coding practices:
   - Always use `crypto/rand`
   - Use strong crypto algorithms
   - Validate all inputs
   - Use prepared statements for databases

### Code Review

Security checklist for reviewers:

- [ ] No hardcoded secrets or keys
- [ ] Proper crypto usage (crypto/rand, SHA-256+, AES-256-GCM)
- [ ] Input validation and sanitization
- [ ] Error handling doesn't leak sensitive info
- [ ] TLS properly configured
- [ ] Dependencies are secure and up-to-date
- [ ] Proper access control
- [ ] No SQL injection vulnerabilities

## Reporting Security Issues

If you discover a security vulnerability:

1. **DO NOT** open a public GitHub issue
2. Email security@pawblockchain.io with details
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

See `SECURITY.md` for our security policy and responsible disclosure process.

## Regular Security Tasks

### Weekly

- Review automated security scan results
- Update dependencies with security patches
- Check for new CVEs affecting dependencies

### Monthly

- Full security audit
- Review and update security configurations
- Penetration testing (if applicable)
- Security training/awareness

### Quarterly

- External security audit
- Threat modeling review
- Incident response plan testing
- Crypto algorithm review

## Additional Resources

- [OWASP Go Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Go_Security_Cheat_Sheet.html)
- [Go Security Policy](https://go.dev/security)
- [Blockchain Security Best Practices](https://consensys.github.io/smart-contract-best-practices/)
- [CIS Docker Benchmark](https://www.cisecurity.org/benchmark/docker)

## Tool Versions

Keep security tools updated:

```bash
# Update all security tools
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/sonatype-nexus-community/nancy@latest
```

Check versions:

```bash
gosec -version
govulncheck -version
trivy --version
gitleaks version
```

## Contact

For security questions or concerns:

- Email: security@pawblockchain.io
- Security Team: See `SECURITY.md`
