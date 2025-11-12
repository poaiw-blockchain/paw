# Security Tools - Quick Reference Card

## Installation

```bash
# Install all Go-based security tools
make install-security-tools

# Or install individually
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/sonatype-nexus-community/nancy@latest

# Install Trivy (macOS)
brew install trivy

# Install GitLeaks (macOS)
brew install gitleaks
```

## Quick Commands

### Before Committing
```bash
make security-audit-quick
```

### Full Security Audit
```bash
make security-audit
```

### Check Dependencies
```bash
make check-deps
```

### Scan for Secrets
```bash
make scan-secrets
```

### All Security Checks
```bash
make security-all
```

## Individual Tools

### GoSec
```bash
gosec -conf security/.gosec.yml ./...
```

### Govulncheck
```bash
govulncheck ./...
```

### Nancy
```bash
go list -json -m all | nancy sleuth
```

### Trivy
```bash
trivy fs --security-checks vuln,config,secret .
```

### GitLeaks
```bash
gitleaks detect --verbose
```

### Crypto Check
```bash
go run security/crypto-check.go
```

## Report Locations

All reports are saved in `security/` directory:
- `gosec-report.json` - GoSec findings
- `trivy-report.json` - Trivy findings
- `gitleaks-report.json` - GitLeaks findings
- `audit-summary-*.txt` - Timestamped summaries

## Common Issues

### Weak Crypto
```go
// Bad
import "crypto/md5"
import "math/rand"

// Good
import "crypto/sha256"
import "crypto/rand"
```

### Hardcoded Secrets
```go
// Bad
password := "my_secret_password"

// Good
password := os.Getenv("PASSWORD")
```

### Insecure TLS
```go
// Bad
InsecureSkipVerify: true

// Good
InsecureSkipVerify: false
```

## Severity Levels

- 🔴 **CRITICAL** - Fix immediately
- 🟠 **HIGH** - Fix urgently
- 🟡 **MEDIUM** - Address soon
- 🟢 **LOW** - Best practice

## CI/CD

Security checks run automatically on:
- Every push to main/master/develop
- Every pull request
- Weekly (Sundays at midnight UTC)

View results: GitHub Security tab

## Documentation

- Full guide: `security/SECURITY_TESTING.md`
- Tool summary: `SECURITY_TOOLS_SUMMARY.md`
- Security policy: `SECURITY.md`

## Support

Security questions: security@pawblockchain.io
