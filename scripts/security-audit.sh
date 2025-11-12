#!/bin/bash
# Comprehensive Security Audit Script for PAW Blockchain
# This script runs multiple security scanning tools

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo -e "${BLUE}================================================${NC}"
echo -e "${BLUE}PAW Blockchain - Comprehensive Security Audit${NC}"
echo -e "${BLUE}================================================${NC}"
echo ""

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to print section headers
print_header() {
    echo ""
    echo -e "${BLUE}==== $1 ====${NC}"
    echo ""
}

# Function to print success
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Function to print warning
print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# Function to print error
print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Track overall status
AUDIT_FAILED=0

# Change to project root
cd "$PROJECT_ROOT"

# 1. GoSec - Go Security Scanner
print_header "Running GoSec (Go Security Scanner)"
if command_exists gosec; then
    if gosec -conf security/.gosec.yml -fmt=json -out=security/gosec-report.json ./...; then
        print_success "GoSec scan completed - No critical issues found"
        gosec -conf security/.gosec.yml ./...
    else
        print_error "GoSec found security issues"
        AUDIT_FAILED=1
        gosec -conf security/.gosec.yml ./...
    fi
else
    print_warning "GoSec not installed. Run: go install github.com/securego/gosec/v2/cmd/gosec@latest"
fi

# 2. Nancy - Dependency Vulnerability Scanner
print_header "Running Nancy (Dependency Vulnerability Scanner)"
if command_exists nancy; then
    if go list -json -m all | nancy sleuth; then
        print_success "Nancy scan completed - No known vulnerabilities"
    else
        print_error "Nancy found vulnerable dependencies"
        AUDIT_FAILED=1
    fi
else
    print_warning "Nancy not installed. Run: go install github.com/sonatype-nexus-community/nancy@latest"
fi

# 3. Govulncheck - Official Go Vulnerability Scanner
print_header "Running Govulncheck (Official Go Vulnerability Scanner)"
if command_exists govulncheck; then
    if govulncheck ./...; then
        print_success "Govulncheck completed - No vulnerabilities found"
    else
        print_error "Govulncheck found vulnerabilities"
        AUDIT_FAILED=1
    fi
else
    print_warning "Govulncheck not installed. Run: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi

# 4. Trivy - Container and Filesystem Security Scanner
print_header "Running Trivy (Container and Filesystem Security)"
if command_exists trivy; then
    # Scan filesystem
    if trivy fs --security-checks vuln,config --severity HIGH,CRITICAL --format table .; then
        print_success "Trivy filesystem scan completed"
    else
        print_error "Trivy found security issues"
        AUDIT_FAILED=1
    fi

    # Generate JSON report
    trivy fs --security-checks vuln,config --format json --output security/trivy-report.json .
else
    print_warning "Trivy not installed. Visit: https://aquasecurity.github.io/trivy/"
fi

# 5. GitLeaks - Secret Scanner
print_header "Running GitLeaks (Secret Detection)"
if command_exists gitleaks; then
    if gitleaks detect --verbose --report-path=security/gitleaks-report.json; then
        print_success "GitLeaks scan completed - No secrets detected"
    else
        print_error "GitLeaks found potential secrets"
        AUDIT_FAILED=1
    fi
else
    print_warning "GitLeaks not installed. Visit: https://github.com/gitleaks/gitleaks"
fi

# 6. Go Mod Verify - Dependency Integrity
print_header "Verifying Go Module Dependencies"
if go mod verify; then
    print_success "All dependencies verified successfully"
else
    print_error "Dependency verification failed"
    AUDIT_FAILED=1
fi

# 7. Custom Crypto Analysis
print_header "Analyzing Cryptographic Usage"
if [ -f "$PROJECT_ROOT/security/crypto-check.go" ]; then
    if go run "$PROJECT_ROOT/security/crypto-check.go"; then
        print_success "Crypto analysis completed"
    else
        print_warning "Crypto analysis found potential issues"
    fi
else
    print_warning "Crypto check tool not found at security/crypto-check.go"
fi

# 8. Check for Weak Crypto Imports
print_header "Checking for Weak Cryptographic Imports"
WEAK_CRYPTO_FOUND=0

if grep -r "crypto/md5" --include="*.go" --exclude-dir=vendor .; then
    print_error "Found MD5 usage (weak crypto)"
    WEAK_CRYPTO_FOUND=1
fi

if grep -r "crypto/sha1" --include="*.go" --exclude-dir=vendor .; then
    print_error "Found SHA1 usage (weak crypto)"
    WEAK_CRYPTO_FOUND=1
fi

if grep -r "crypto/des" --include="*.go" --exclude-dir=vendor .; then
    print_error "Found DES usage (weak crypto)"
    WEAK_CRYPTO_FOUND=1
fi

if grep -r "crypto/rc4" --include="*.go" --exclude-dir=vendor .; then
    print_error "Found RC4 usage (weak crypto)"
    WEAK_CRYPTO_FOUND=1
fi

if grep -r "math/rand" --include="*.go" --exclude-dir=vendor --exclude="*_test.go" .; then
    print_warning "Found math/rand usage (should use crypto/rand for security)"
fi

if [ $WEAK_CRYPTO_FOUND -eq 0 ]; then
    print_success "No weak cryptographic imports found"
else
    AUDIT_FAILED=1
fi

# 9. Check for Hardcoded Secrets
print_header "Checking for Hardcoded Secrets/Keys"
SECRET_PATTERNS=(
    "password.*=.*['\"]"
    "secret.*=.*['\"]"
    "api_key.*=.*['\"]"
    "private_key.*=.*['\"]"
    "token.*=.*['\"]"
)

SECRETS_FOUND=0
for pattern in "${SECRET_PATTERNS[@]}"; do
    if grep -rE "$pattern" --include="*.go" --exclude-dir=vendor --exclude="*_test.go" .; then
        SECRETS_FOUND=1
    fi
done

if [ $SECRETS_FOUND -eq 0 ]; then
    print_success "No obvious hardcoded secrets found"
else
    print_warning "Potential hardcoded secrets detected - review manually"
fi

# 10. Check TLS Configuration
print_header "Checking TLS Configuration"
if grep -r "InsecureSkipVerify.*true" --include="*.go" --exclude-dir=vendor .; then
    print_error "Found InsecureSkipVerify enabled (insecure TLS)"
    AUDIT_FAILED=1
else
    print_success "No insecure TLS configurations found"
fi

# 11. Check File Permissions
print_header "Checking File Permission Settings"
if grep -rE "(0777|0666)" --include="*.go" --exclude-dir=vendor .; then
    print_warning "Found overly permissive file permissions"
else
    print_success "No overly permissive file permissions found"
fi

# 12. Generate Summary Report
print_header "Generating Summary Report"
REPORT_FILE="security/audit-summary-$(date +%Y%m%d-%H%M%S).txt"
{
    echo "PAW Blockchain Security Audit Summary"
    echo "======================================"
    echo "Date: $(date)"
    echo "Git Commit: $(git rev-parse HEAD 2>/dev/null || echo 'Not a git repository')"
    echo ""
    echo "Tools Run:"
    echo "  - GoSec: $(command_exists gosec && echo 'Yes' || echo 'No')"
    echo "  - Nancy: $(command_exists nancy && echo 'Yes' || echo 'No')"
    echo "  - Govulncheck: $(command_exists govulncheck && echo 'Yes' || echo 'No')"
    echo "  - Trivy: $(command_exists trivy && echo 'Yes' || echo 'No')"
    echo "  - GitLeaks: $(command_exists gitleaks && echo 'Yes' || echo 'No')"
    echo ""
    echo "Results:"
    echo "  - Overall Status: $([ $AUDIT_FAILED -eq 0 ] && echo 'PASSED' || echo 'FAILED')"
    echo ""
    echo "Detailed reports generated:"
    echo "  - GoSec: security/gosec-report.json"
    echo "  - Trivy: security/trivy-report.json"
    echo "  - GitLeaks: security/gitleaks-report.json"
} > "$REPORT_FILE"

print_success "Summary report saved to: $REPORT_FILE"

# Final status
echo ""
echo -e "${BLUE}================================================${NC}"
if [ $AUDIT_FAILED -eq 0 ]; then
    echo -e "${GREEN}Security Audit: PASSED${NC}"
    echo -e "${BLUE}================================================${NC}"
    exit 0
else
    echo -e "${RED}Security Audit: FAILED${NC}"
    echo -e "${RED}Please review the issues found above${NC}"
    echo -e "${BLUE}================================================${NC}"
    exit 1
fi
