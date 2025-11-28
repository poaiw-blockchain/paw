# Security Directory

This directory contains security-related configuration, tools, and reports for the PAW blockchain project.

## Contents

### Configuration Files

- **.gosec.yml** - GoSec configuration for Go code security scanning
- **nancy-config.yml** - Nancy configuration for dependency vulnerability scanning

### Tools

- **crypto-check.go** - Custom cryptographic usage analyzer
  - Scans for weak crypto algorithms
  - Detects insecure random number generation
  - Identifies hardcoded secrets
  - Validates key sizes and TLS configurations

### Documentation

- **SECURITY_TESTING.md** - Comprehensive security testing guide
  - Tool descriptions and usage
  - Security testing strategy
  - How to interpret results
  - Remediation guidelines

### Reports

Security audit reports are generated in this directory:

- **gosec-report.json** - GoSec scan results (JSON format)
- **trivy-report.json** - Trivy scan results (JSON format)
- **gitleaks-report.json** - GitLeaks secret detection results
- **audit-summary-YYYYMMDD-HHMMSS.txt** - Timestamped audit summaries
- **report-latest.txt** - Latest security report
- **dependency-licenses.txt** - Dependency license report (if generated)
- **dependency-graph.png** - Visual dependency graph (if generated)

## Quick Start

### Install Security Tools

```bash
# Install all Go-based security tools
make install-security-tools

# Or manually:
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/sonatype-nexus-community/nancy@latest
```

For Trivy and GitLeaks, see installation instructions in `SECURITY_TESTING.md`.

### Run Security Audits

```bash
# Quick security check (essential tools)
make security-audit-quick

# Full comprehensive audit
make security-audit

# Check dependencies only
make check-deps

# Scan for secrets
make scan-secrets

# All security checks
make security-all
```

### Manual Tool Usage

```bash
# GoSec
gosec -conf security/.gosec.yml ./...

# Govulncheck
govulncheck ./...

# Nancy
go list -json -m all | nancy sleuth

# Trivy
trivy fs --security-checks vuln,config,secret .

# GitLeaks
gitleaks detect --verbose

# Custom crypto check
go run security/crypto-check.go
```

## CI/CD Integration

Security checks run automatically via  Actions:

- On every push to main/master/develop
- On every pull request
- Weekly scheduled scans (Sundays at midnight UTC)
- Manual workflow dispatch

See `hub/workflows/security.yml` for details.

## Security Best Practices

### For Developers

1. **Before Committing**

   ```bash
   make security-audit-quick
   ```

2. **Never commit secrets**
   - Use environment variables
   - Use secure key management systems
   - Review GitLeaks warnings

3. **Use strong cryptography**
   - Always use `crypto/rand`, never `math/rand` for security
   - Use SHA-256 or SHA-512, not MD5 or SHA1
   - Use AES-256-GCM for encryption
   - Use ECDSA or Ed25519 for signatures

4. **Keep dependencies updated**
   ```bash
   go get -u ./...
   go mod tidy
   make check-deps
   ```

### For Reviewers

Security checklist:

- [ ] No hardcoded secrets or keys
- [ ] Proper crypto usage (crypto/rand, strong algorithms)
- [ ] Input validation and sanitization
- [ ] Error handling doesn't leak sensitive info
- [ ] TLS properly configured
- [ ] Dependencies are secure and up-to-date

## False Positives

If you encounter false positives:

1. **GoSec**: Add `// #nosec GXXX` comment with justification
2. **Nancy/Govulncheck**: Document in `nancy-config.yml` why CVE doesn't apply
3. **GitLeaks**: Add to `leaksignore` (use sparingly)

## Severity Levels

- **CRITICAL**: Immediate action required, potential security breach
- **HIGH**: Significant security risk, fix urgently
- **MEDIUM**: Security concern, should be addressed soon
- **LOW**: Minor issue or best practice recommendation

## Common Issues

### Weak Cryptography

**Problem**: Use of MD5, SHA1, DES, or RC4
**Solution**: Use SHA-256/SHA-512, AES-256-GCM, ECDSA/Ed25519

### Insecure Random

**Problem**: Using `math/rand` instead of `crypto/rand`
**Solution**: Always use `crypto/rand` for security-sensitive operations

### Hardcoded Secrets

**Problem**: Credentials or keys in source code
**Solution**: Use environment variables or secure key management

### Small Key Sizes

**Problem**: RSA keys smaller than 2048 bits
**Solution**: Use at least 2048-bit RSA, or prefer ECDSA (256-bit)

### Insecure TLS

**Problem**: `InsecureSkipVerify: true` in TLS config
**Solution**: Always verify certificates, use proper cert management

## Report Security Issues

If you discover a security vulnerability:

1. **DO NOT** open a public  issue
2. Email: security@pawblockchain.io
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

See main `SECURITY.md` for our security policy.

## Additional Resources

- [OWASP Go Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Go_Security_Cheat_Sheet.html)
- [Go Security Policy](https://go.dev/security)
- [Blockchain Security Best Practices](https://consensyshub.io/smart-contract-best-practices/)
- [CWE Top 25](https://cwe.mitre.org/top25/)

## Updating Security Tools

Keep tools up to date:

```bash
# Update Go-based tools
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/sonatype-nexus-community/nancy@latest

# Update Trivy (varies by OS)
# macOS: brew upgrade trivy
# Linux: see Trivy documentation

# Update GitLeaks (varies by OS)
# macOS: brew upgrade gitleaks
# Linux: see GitLeaks documentation
```

## Contact

For security questions:

- Security Team: security@pawblockchain.io
- Security Policy: See `SECURITY.md` in project root
