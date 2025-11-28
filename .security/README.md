# PAW Blockchain Security Monitoring System

Comprehensive continuous security monitoring for PAW Blockchain with dual-language support (Go and Python).

## Overview

This directory contains all security monitoring configurations and tools for the PAW Blockchain project. The system provides:

- **Automated Security Scanning** - 12+ security tools running continuously
- **Multi-Language Support** - Go and Python codebases
- **Real-time Alerts** - Email, Slack, GitHub, PagerDuty
- **Vulnerability Management** - Detection, tracking, and remediation
- **Compliance Reporting** - SARIF, JSON, HTML formats

## Directory Structure

```
.security/
├── README.md                           # This file
├── config.yml                          # Master security configuration
├── .gosec.yml                          # GoSec configuration
├── dependency-check-suppressions.xml  # CVE suppressions
└── reports/                            # Generated reports directory
```

## Configuration

### Main Configuration: `config.yml`

Master configuration file defining:
- Scan schedules (cron expressions)
- Severity thresholds
- Tool-specific settings
- Alert routing
- Exclusions and suppressions

**Key Sections:**

```yaml
# Scan schedules
schedules:
  daily_vulnerability_scan: "0 2 * * *"
  weekly_comprehensive: "0 3 * * 0"

# Severity levels trigger actions
severity_thresholds:
  fail_on:
    - CRITICAL
    - HIGH

# Language-specific tools
go_security:
  tools:
    gosec:
      enabled: true
    nancy:
      enabled: true

python_security:
  tools:
    bandit:
      enabled: true
    safety:
      enabled: true
```

### GoSec Configuration: `.gosec.yml`

Specific configuration for Go security scanning:
- Rule enablement/disabling
- CWE mappings
- Exclusion patterns
- Audit settings

### Dependency-Check Suppressions: `dependency-check-suppressions.xml`

CVE suppressions for known issues:
- False positives
- Mitigated vulnerabilities
- Non-applicable issues
- Accepted risks

Format:
```xml
<suppression>
  <notes>Reason for suppression</notes>
  <cve>CVE-YYYY-NNNNN</cve>
  <until>2025-12-31</until>
</suppression>
```

## Security Tools

### Go Security Stack

1. **GoSec** - Static analysis for security issues
   - CWE coverage: 20+ types
   - Reports: JSON, SARIF, text
   - Config: `.gosec.yml`

2. **Nancy** - Dependency vulnerability checking
   - Database: OSS Index
   - Frequency: Daily
   - Action: Fail build on vulnerabilities

3. **govulncheck** - Official Go vulnerability database
   - Database: golang.org/x/vuln
   - Coverage: All Go packages
   - Frequency: Daily

4. **golangci-lint** - Comprehensive linting
   - Security linters: gosec, staticcheck, gas
   - Config: `../.golangci.yml`
   - Frequency: On every commit

5. **Trivy** - Vulnerability scanner
   - Scans: FS, configs, dependencies
   - Frequency: Daily
   - Severity: MEDIUM, HIGH, CRITICAL

### Python Security Stack

1. **Bandit** - Python security issue detection
   - Tests: 50+ security checks
   - Config: Inline in security_monitor.py
   - Frequency: On every push

2. **Safety** - Dependency vulnerability checking
   - Database: PyUp.io
   - Update frequency: Multiple times daily
   - Coverage: All packages

3. **pip-audit** - Package vulnerability auditing
   - Database: PyPA Advisory Database
   - Format: JSON reports
   - Frequency: Daily

4. **Semgrep** - Static analysis and SAST
   - Rules: p/security-audit, p/owasp-top-ten
   - Languages: Python, Go, JavaScript
   - Frequency: Daily

### Universal Tools

1. **CodeQL** - Semantic code analysis
   - Languages: Go, Python, JavaScript
   - Queries: security-extended, security-and-quality
   - Frequency: On every push

2. **Trivy** - Container and dependency scanning
   - Scan types: FS, configs, secrets
   - Reports: JSON, SARIF
   - Frequency: Daily

3. **TruffleHog** - Secret and sensitive data detection
   - Entropy threshold: 4.5
   - Scope: Filesystem, git history
   - Frequency: Daily

4. **GitLeaks** - Secret detection in git
   - Patterns: 50+ built-in
   - Scope: Full git history
   - Frequency: Daily

5. **OWASP Dependency-Check** - Dependency vulnerabilities
   - Database: NVD, CVE
   - Suppressions: `dependency-check-suppressions.xml`
   - Frequency: Weekly

## Running Scans Locally

### Python Security Monitor

```bash
# Install dependencies
pip install bandit safety pip-audit semgrep pyyaml

# Run all Python security scans
python scripts/security_monitor.py

# Outputs:
# - security-report.json
# - security-report.md
# - security-report.sarif
# - bandit-report.json
```

### Go Security Monitor

```bash
# Build the monitor
go build -o security-monitor scripts/security_monitor.go

# Run all Go security scans
./security-monitor

# Outputs:
# - security-report.json
# - security-report.md
# - security-report.sarif
# - gosec-report.json
```

### Individual Tools

```bash
# GoSec
gosec -conf .security/.gosec.yml -fmt json -out gosec-report.json ./...

# Bandit
bandit -r src/ -f json -o bandit-report.json

# Nancy
go list -json -m all | nancy sleuth

# govulncheck
govulncheck ./...

# Safety
safety check

# pip-audit
pip-audit

# Semgrep
semgrep --config=p/security-audit src/

# CodeQL (requires setup)
codeql database create my-database --language=go --source-root=.
codeql database analyze my-database codeql/go-queries --format=sarif-latest --output=results.sarif

# Trivy
trivy fs .

# TruffleHog
trufflehog filesystem .

# GitLeaks
gitleaks detect --verbose

# OWASP Dependency-Check
dependency-check --project "PAW" --scan .
```

## Alert Routing

Configured in `config.yml` under `alert_routing`:

```yaml
alert_routing:
  CRITICAL:
    channels:
      - email
      - slack
      - github_issue
      - pagerduty
  HIGH:
    channels:
      - email
      - slack
      - github_issue
  MEDIUM:
    channels:
      - slack
      - github_issue
```

## Processing Alerts

```bash
# Install dependencies
pip install pyyaml requests

# Process findings and send alerts
python scripts/security_alerts.py

# Environment variables needed:
# - GITHUB_TOKEN
# - SECURITY_SLACK_WEBHOOK (optional)
# - SECURITY_SMTP_SERVER, SECURITY_EMAIL_PASSWORD (for email)
# - PAGERDUTY_INTEGRATION_KEY (optional)
```

## GitHub Actions Integration

### Workflow: `security-monitoring.yml`

Automatically triggered by:
- Push to main/develop
- Pull requests
- Daily schedule (2 AM UTC)
- Weekly schedule (3 AM UTC)
- Manual trigger

### Jobs Executed

1. `python-security` - Python security scans
2. `go-security` - Go security scans
3. `bandit` - Detailed Bandit analysis
4. `safety-check` - Dependency safety
5. `gosec` - Detailed GoSec analysis
6. `nancy` - Dependency checking
7. `govulncheck` - Go vulnerabilities
8. `trivy` - Container/dependency scanning
9. `gitleaks` - Secret detection
10. `codeql` - Semantic analysis
11. `dependency-check` - OWASP scanning
12. `trufflehog` - Secret detection
13. `security-alerts` - Alert processing
14. `security-dashboard` - Dashboard generation
15. `security-summary` - Final report

## Pre-commit Hooks

Security checks integrated into git workflow (`.pre-commit-config.yaml`):

```bash
# Install pre-commit
pip install pre-commit
pre-commit install

# Run manually
pre-commit run --all-files

# Hooks include:
# - Bandit (Python security)
# - GoSec (Go security)
# - TruffleHog (Secret detection)
# - Semgrep (SAST)
```

## Excluding Files

### Global Exclusions

Defined in `config.yml`:
```yaml
exclusions:
  patterns:
    - "*.pb.go"
    - "*.pb.gw.go"
    - "*_test.go"
    - "*_test.py"
    - "vendor/**"
    - ".venv/**"
```

### Tool-Specific Exclusions

- **GoSec**: `.gosec.yml` - `exclude_dir`, `exclude_generated`
- **Bandit**: `security_monitor.py` - inline configuration
- **Trivy**: Ignore files specified in scans
- **TruffleHog**: Exclude patterns in scans

### Suppressing Vulnerabilities

For false positives or accepted risks:

1. **CVE Suppressions**: `dependency-check-suppressions.xml`
   ```xml
   <suppression>
     <notes>False positive - function not used</notes>
     <cve>CVE-YYYY-NNNNN</cve>
   </suppression>
   ```

2. **Tool-Specific**: See individual tool configs

## Severity Levels

| Level | Acronym | Action |
|-------|---------|--------|
| CRITICAL | C | Fail build, page on-call, create issue |
| HIGH | H | Fail build, email alert, create issue |
| MEDIUM | M | Create issue, Slack alert |
| LOW | L | Create issue |
| INFO | I | Log only |

## Reports

### Generated Report Formats

1. **JSON** - Machine-readable findings
   - File: `security-report.json`
   - Contains: All findings with details

2. **Markdown** - Human-readable report
   - File: `security-report.md`
   - Contains: Summary and detailed findings

3. **SARIF** - Security Analysis Results Format
   - File: `security-report.sarif`
   - Used by: GitHub Security tab

4. **SARIF (GitHub)** - Native GitHub import
   - Uploaded to GitHub Security
   - Visible in Security tab

### Report Location

All reports generated in project root:
- `security-report.json`
- `security-report.md`
- `security-report.sarif`
- `bandit-report.json`
- `gosec-report.json`
- etc.

## Dashboard

Comprehensive security dashboard available at `SECURITY-DASHBOARD.md`:
- Security scores
- Tool status
- Recent findings
- Trends and metrics
- Alert configuration

## Environment Variables

### Required for GitHub Actions

```bash
GITHUB_TOKEN=<github_personal_access_token>
```

### Optional for Alerts

```bash
# Slack notifications
SECURITY_SLACK_WEBHOOK=<webhook_url>

# Email alerts
SECURITY_SMTP_SERVER=smtp.gmail.com
SECURITY_SMTP_PORT=587
SECURITY_EMAIL_FROM=security@github.com
SECURITY_EMAIL_PASSWORD=<app_password>

# PagerDuty (critical alerts)
PAGERDUTY_INTEGRATION_KEY=<integration_key>
```

## Troubleshooting

### Tools Not Found

```bash
# Go tools
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install github.com/sonatype-nexus-community/nancy@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Python tools
pip install bandit safety pip-audit semgrep pyyaml requests

# System tools
brew install trufflehog  # macOS
apt-get install trufflehog  # Linux
```

### High False Positive Rate

1. Review and update tool versions
2. Adjust severity thresholds in `config.yml`
3. Add exclusions for specific patterns
4. Create CVE suppressions for known issues

### Slow Scans

1. Increase parallel execution in `.gosec.yml`
2. Exclude non-source directories
3. Run scans on fewer files during dev
4. Use caching in GitHub Actions

## Best Practices

1. **Review Regularly** - Check dashboard daily
2. **Update Tools** - Keep tools updated monthly
3. **Triage Issues** - Address findings within SLA
4. **Suppress Wisely** - Document all suppressions
5. **Team Training** - Educate team on security
6. **CI/CD Integration** - Require passing security checks
7. **Dependency Audits** - Review updates before merging
8. **Secret Rotation** - Rotate exposed secrets immediately

## Documentation

- **Security Dashboard**: `SECURITY-DASHBOARD.md`
- **Tool Docs**: See links in dashboard
- **Configuration**: See inline comments in config files
- **Scripts**: See docstrings in Python/Go files

## Support & Escalation

1. **Questions** - Email: security-alerts@github.com
2. **Critical Issues** - Page on-call engineer
3. **Bug Report** - GitHub issue with security label
4. **Vulnerability** - Email security@github.com

## License

This security monitoring system is part of the PAW Blockchain project and follows the same license.

## Version

- **System Version:** 1.0.0
- **Last Updated:** 2025-11-18
- **Maintained By:** Security Team
