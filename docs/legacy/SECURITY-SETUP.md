# PAW Blockchain Security Monitoring - Setup Guide

Quick start guide to enable and configure continuous security monitoring for PAW Blockchain.

## Quick Start (5 minutes)

### 1. Install Pre-commit Hooks

```bash
# Install pre-commit
pip install pre-commit

# Install the hooks
pre-commit install
pre-commit install --hook-type commit-msg

# Verify installation
pre-commit run --all-files
```

### 2. Install Go Security Tools

```bash
# GoSec
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Nancy
go install github.com/sonatype-nexus-community/nancy@latest

# govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest

# golangci-lint (if not already installed)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### 3. Install Python Security Tools

```bash
# Create/activate virtual environment
python -m venv .venv
source .venv/bin/activate  # On Windows: .venv\Scripts\activate

# Install security tools
pip install bandit safety pip-audit semgrep pyyaml requests
```

### 4. Run Security Scans

```bash
# Python security scan
python scripts/security_monitor.py

# Go security scan
go build -o security-monitor scripts/security_monitor.go
./security-monitor

# Check reports
cat security-report.md
cat security-report.json
```

## Full Setup (30 minutes)

### Step 1: Configure Environment Variables

Create `.env` file in project root:

```bash
# GitHub
export GITHUB_TOKEN=<your_github_token>

# Slack (optional)
export SECURITY_SLACK_WEBHOOK=<webhook_url>

# Email (optional)
export SECURITY_SMTP_SERVER=smtp.gmail.com
export SECURITY_SMTP_PORT=587
export SECURITY_EMAIL_FROM=security@example.com
export SECURITY_EMAIL_PASSWORD=<app_password>

# PagerDuty (optional)
export PAGERDUTY_INTEGRATION_KEY=<integration_key>
```

Load environment variables:
```bash
source .env  # Linux/macOS
set -a && source .env && set +a  # Bash
```

### Step 2: Configure Security Settings

Edit `.security/config.yml` to customize:
- Alert recipients
- Severity thresholds
- Scan schedules
- Exclusions
- Tools configuration

### Step 3: Enable GitHub Actions

The workflow `.github/workflows/security-monitoring.yml` will run automatically on:
- Every push to main/develop
- Every pull request
- Daily at 2 AM UTC
- Weekly at 3 AM UTC

No configuration needed - it works out of the box!

### Step 4: Configure Alerts (Optional)

#### Slack Alerts

1. Create Slack webhook:
   - Go to your Slack workspace
   - Settings → Manage apps
   - Create Incoming Webhook for #security-alerts
   - Copy webhook URL

2. Set environment variable:
   ```bash
   export SECURITY_SLACK_WEBHOOK=<webhook_url>
   ```

#### Email Alerts

1. Set SMTP credentials:
   ```bash
   export SECURITY_SMTP_SERVER=smtp.gmail.com
   export SECURITY_SMTP_PORT=587
   export SECURITY_EMAIL_FROM=your-email@gmail.com
   export SECURITY_EMAIL_PASSWORD=<app_password>
   ```

2. Note: Use Gmail app password, not account password

#### GitHub Issues

Automatically enabled with GITHUB_TOKEN.

#### PagerDuty Alerts

1. Get integration key from PagerDuty
2. Set environment variable:
   ```bash
   export PAGERDUTY_INTEGRATION_KEY=<key>
   ```

### Step 5: Suppress Known Vulnerabilities (Optional)

For false positives or accepted risks:

1. Add CVE suppression in `.security/dependency-check-suppressions.xml`:

```xml
<suppression>
  <notes>False positive - function not called in our code</notes>
  <cve>CVE-2024-XXXXX</cve>
  <until>2025-12-31</until>
</suppression>
```

2. Edit `.security/.gosec.yml` to disable specific rules:

```yaml
rules:
  G101:
    enabled: false  # Too many test false positives
```

### Step 6: Configure Branch Protection (Optional)

In GitHub repository settings:

1. Go to Settings → Branches → main
2. Add Status Check Requirements:
   - `gosec`
   - `bandit`
   - `codeql`
   - `trivy`
   - `dependency-check`

3. Enable:
   - ✓ Require status checks to pass before merging
   - ✓ Require branches to be up to date
   - ✓ Dismiss stale PR approvals

## Running Scans

### Manual Scans

```bash
# Run all Python security checks
python scripts/security_monitor.py

# Run all Go security checks
go run scripts/security_monitor.go

# Run individual tools
gosec -conf .security/.gosec.yml -fmt json ./...
bandit -r src/ -f json
safety check --json
```

### Pre-commit Scans

Automatically run on every commit:

```bash
# Bypass (not recommended)
git commit --no-verify
```

### GitHub Actions

Automatically run on:
- Push to main/develop
- Pull request
- Schedule (daily 2 AM UTC)
- Manual dispatch

### Processing Alerts

```bash
# After running security scans
python scripts/security_alerts.py
```

## Viewing Results

### Local Reports

After running scans, view:

```bash
# Summary
cat security-report.md

# Detailed JSON
cat security-report.json

# SARIF format (for IDE)
cat security-report.sarif
```

### GitHub Security Tab

Reports automatically uploaded to:
- GitHub → Security → Code scanning alerts
- Shows SARIF results from all tools

### Dashboard

View comprehensive dashboard:

```bash
cat SECURITY-DASHBOARD.md
```

Updates automatically after GitHub Actions runs.

## Customization

### Change Scan Schedule

Edit `.security/config.yml`:

```yaml
schedules:
  daily_vulnerability_scan: "0 2 * * *"  # 2 AM UTC
  weekly_comprehensive: "0 3 * * 0"      # 3 AM Sunday
```

Cron format: `minute hour day month day-of-week`

### Exclude Files/Directories

In `.security/config.yml`:

```yaml
exclusions:
  patterns:
    - "vendor/"
    - "*_test.go"
    - "testdata/"
```

In individual tool configs:
- `.security/.gosec.yml` - GoSec exclusions
- `.pre-commit-config.yaml` - Pre-commit exclusions

### Adjust Severity Thresholds

In `.security/config.yml`:

```yaml
severity_thresholds:
  fail_on:
    - CRITICAL
    - HIGH
  create_issue_on:
    - CRITICAL
    - HIGH
    - MEDIUM
```

### Add New Tools

1. Add tool configuration to `.security/config.yml`
2. Add installation to workflow `.github/workflows/security-monitoring.yml`
3. Add scan step to workflow
4. Add artifact upload if tool generates reports

## Troubleshooting

### "Tool not found" errors

Install missing tools:

```bash
# Go tools
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install github.com/sonatype-nexus-community/nancy@latest

# Python tools
pip install bandit safety pip-audit semgrep

# System tools
brew install trufflehog  # macOS
# OR use Docker: docker run ghcr.io/trufflesecurity/trufflehog:latest
```

### Alerts not sending

Check:
1. Environment variables set: `echo $SECURITY_SLACK_WEBHOOK`
2. Webhook URL is correct
3. Network connectivity
4. Check GitHub Actions logs for errors

### High false positive rate

1. Review and update tool versions:
   ```bash
   go get -u github.com/securego/gosec/v2/cmd/gosec@latest
   pip install --upgrade bandit
   ```

2. Adjust tool configuration:
   - Edit `.security/.gosec.yml` for GoSec
   - Edit `.security/config.yml` for Python tools
   - Disable specific rules causing false positives

3. Add suppressions for known issues

### Scans too slow

1. Add exclusions for non-source directories
2. Run scans in parallel
3. Increase tool timeouts in config
4. Use caching in GitHub Actions

## Maintenance

### Daily

- Check GitHub Security tab
- Review Slack alerts
- Triage any new issues

### Weekly

- Review detailed reports
- Update tool versions
- Check dependency security

### Monthly

- Full security audit
- Review and rotate secrets
- Update suppression rules
- Team training on findings

## Security Best Practices

1. **Never commit secrets** - TruffleHog scans prevent this
2. **Review dependencies** - Check Nancy/Safety alerts
3. **Keep tools updated** - Monthly version checks
4. **Document suppressions** - Always explain why
5. **Act on findings** - Follow SLA response times
6. **Team awareness** - Share security reports
7. **Secure CI/CD** - Use branch protection rules
8. **Incident response** - Plan for security events

## Support

### Get Help

1. **Documentation**: Read `.security/README.md`
2. **Dashboard**: Check `SECURITY-DASHBOARD.md`
3. **Tool Docs**: Visit tool repositories
4. **Team**: Email security-alerts@example.com

### Report Issues

1. **Security vulnerability**: Email security@example.com
2. **Tool malfunction**: GitHub issue with #security label
3. **Configuration help**: Email security team

## Next Steps

1. ✅ Run first security scan
2. ✅ Configure alerts
3. ✅ Review baseline findings
4. ✅ Suppress false positives
5. ✅ Enable GitHub Actions
6. ✅ Set branch protection rules
7. ✅ Document procedures
8. ✅ Train team members

## Resources

- **Configuration**: `.security/config.yml`
- **Dashboard**: `SECURITY-DASHBOARD.md`
- **Documentation**: `.security/README.md`
- **Workflow**: `.github/workflows/security-monitoring.yml`
- **Pre-commit**: `.pre-commit-config.yaml`

## Checklist

- [ ] Pre-commit hooks installed
- [ ] Security tools installed
- [ ] First scan completed
- [ ] Environment variables set
- [ ] Alerts configured
- [ ] GitHub Actions running
- [ ] Branch protection enabled
- [ ] Team trained
- [ ] Documentation reviewed
- [ ] Baseline established

---

**Setup Complete!** Your PAW Blockchain is now under continuous security monitoring.

For detailed information, see `.security/README.md` and `SECURITY-DASHBOARD.md`.
