#!/bin/bash

###############################################################################
# PAW Blockchain Security Scanner
#
# Automated security scanning script that runs multiple security tools
# to detect vulnerabilities, code quality issues, and security weaknesses
#
# Usage: ./scripts/security-scan.sh [options]
#
# Options:
#   --quick       Run quick scan (skip slow checks)
#   --full        Run comprehensive scan (all tools)
#   --report      Generate HTML report
#   --fix         Attempt to auto-fix issues where possible
#   --ci          CI mode (exit with error if issues found)
###############################################################################

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPORT_DIR="${PROJECT_ROOT}/security-reports"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Flags
QUICK_SCAN=false
FULL_SCAN=false
GENERATE_REPORT=false
AUTO_FIX=false
CI_MODE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --quick)
      QUICK_SCAN=true
      shift
      ;;
    --full)
      FULL_SCAN=true
      shift
      ;;
    --report)
      GENERATE_REPORT=true
      shift
      ;;
    --fix)
      AUTO_FIX=true
      shift
      ;;
    --ci)
      CI_MODE=true
      shift
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Create report directory
mkdir -p "${REPORT_DIR}"

# Initialize counters
TOTAL_ISSUES=0
CRITICAL_ISSUES=0
HIGH_ISSUES=0
MEDIUM_ISSUES=0
LOW_ISSUES=0

###############################################################################
# Helper Functions
###############################################################################

print_header() {
  echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "${BLUE}  $1${NC}"
  echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

print_success() {
  echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
  echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
  echo -e "${RED}✗${NC} $1"
}

print_info() {
  echo -e "${BLUE}ℹ${NC} $1"
}

check_tool() {
  if ! command -v "$1" &> /dev/null; then
    print_warning "$1 not found. Installing..."
    return 1
  fi
  return 0
}

###############################################################################
# Tool Installation
###############################################################################

install_tools() {
  print_header "Installing Security Tools"

  # GoSec
  if ! check_tool gosec; then
    go install example.com/securego/gosec/v2/cmd/gosec@latest
    print_success "Installed gosec"
  fi

  # Nancy (dependency vulnerability scanner)
  if ! check_tool nancy; then
    go install example.com/sonatype-nexus-community/nancy@latest
    print_success "Installed nancy"
  fi

  # Staticcheck
  if ! check_tool staticcheck; then
    go install honnef.co/go/tools/cmd/staticcheck@latest
    print_success "Installed staticcheck"
  fi

  # govulncheck
  if ! check_tool govulncheck; then
    go install golang.org/x/vuln/cmd/govulncheck@latest
    print_success "Installed govulncheck"
  fi

  # gitleaks (if not in CI)
  if ! check_tool gitleaks && [ "$CI_MODE" = false ]; then
    print_info "Gitleaks not found. Install from: https://example.com/gitleaks/gitleaks"
  fi
}

###############################################################################
# Security Scans
###############################################################################

run_gosec() {
  print_header "Running GoSec (Security Scanner)"

  local output_file="${REPORT_DIR}/gosec_${TIMESTAMP}.json"
  local severity_flag=""

  if [ "$QUICK_SCAN" = true ]; then
    severity_flag="-severity high"
  fi

  if gosec -fmt=json -out="${output_file}" ${severity_flag} ./... 2>&1 | tee "${REPORT_DIR}/gosec_${TIMESTAMP}.log"; then
    print_success "GoSec scan completed"

    # Parse results
    if [ -f "${output_file}" ]; then
      local issues=$(jq -r '.Issues | length' "${output_file}" 2>/dev/null || echo "0")
      TOTAL_ISSUES=$((TOTAL_ISSUES + issues))

      if [ "$issues" -gt 0 ]; then
        print_warning "Found ${issues} security issues"

        # Count by severity
        local high=$(jq -r '[.Issues[] | select(.severity == "HIGH")] | length' "${output_file}" 2>/dev/null || echo "0")
        local medium=$(jq -r '[.Issues[] | select(.severity == "MEDIUM")] | length' "${output_file}" 2>/dev/null || echo "0")
        local low=$(jq -r '[.Issues[] | select(.severity == "LOW")] | length' "${output_file}" 2>/dev/null || echo "0")

        HIGH_ISSUES=$((HIGH_ISSUES + high))
        MEDIUM_ISSUES=$((MEDIUM_ISSUES + medium))
        LOW_ISSUES=$((LOW_ISSUES + low))

        echo "  High: ${high}, Medium: ${medium}, Low: ${low}"
      else
        print_success "No security issues found"
      fi
    fi
  else
    print_error "GoSec scan failed"
  fi
}

run_staticcheck() {
  print_header "Running Staticcheck (Static Analysis)"

  local output_file="${REPORT_DIR}/staticcheck_${TIMESTAMP}.txt"

  if staticcheck ./... 2>&1 | tee "${output_file}"; then
    print_success "Staticcheck completed"

    local issues=$(wc -l < "${output_file}" | tr -d ' ')
    if [ "$issues" -gt 0 ]; then
      TOTAL_ISSUES=$((TOTAL_ISSUES + issues))
      MEDIUM_ISSUES=$((MEDIUM_ISSUES + issues))
      print_warning "Found ${issues} code quality issues"
    else
      print_success "No staticcheck issues found"
    fi
  else
    print_error "Staticcheck found issues"
    TOTAL_ISSUES=$((TOTAL_ISSUES + 1))
  fi
}

run_govulncheck() {
  print_header "Running govulncheck (Vulnerability Scanner)"

  local output_file="${REPORT_DIR}/govulncheck_${TIMESTAMP}.txt"

  if govulncheck ./... 2>&1 | tee "${output_file}"; then
    print_success "No known vulnerabilities found"
  else
    print_error "Vulnerabilities detected in dependencies"

    local vulns=$(grep -c "Vulnerability" "${output_file}" || echo "0")
    CRITICAL_ISSUES=$((CRITICAL_ISSUES + vulns))
    TOTAL_ISSUES=$((TOTAL_ISSUES + vulns))
  fi
}

run_nancy() {
  print_header "Running Nancy (Dependency Scanner)"

  local output_file="${REPORT_DIR}/nancy_${TIMESTAMP}.txt"

  # Generate dependency list
  go list -json -m all | nancy sleuth 2>&1 | tee "${output_file}" || true

  if grep -q "Vulnerable" "${output_file}"; then
    print_error "Vulnerable dependencies found"

    local vulns=$(grep -c "Vulnerable" "${output_file}" || echo "0")
    HIGH_ISSUES=$((HIGH_ISSUES + vulns))
    TOTAL_ISSUES=$((TOTAL_ISSUES + vulns))
  else
    print_success "No vulnerable dependencies found"
  fi
}

run_gitleaks() {
  print_header "Running Gitleaks (Secret Scanner)"

  if ! check_tool gitleaks; then
    print_warning "Gitleaks not installed, skipping secret scan"
    return
  fi

  local output_file="${REPORT_DIR}/gitleaks_${TIMESTAMP}.json"

  if gitleaks detect --source="${PROJECT_ROOT}" --report-path="${output_file}" --no-; then
    print_success "No secrets detected"
  else
    print_error "Potential secrets found"

    if [ -f "${output_file}" ]; then
      local secrets=$(jq '. | length' "${output_file}" 2>/dev/null || echo "0")
      CRITICAL_ISSUES=$((CRITICAL_ISSUES + secrets))
      TOTAL_ISSUES=$((TOTAL_ISSUES + secrets))
    fi
  fi
}

run_go_vet() {
  print_header "Running go vet (Go Tool)"

  local output_file="${REPORT_DIR}/govet_${TIMESTAMP}.txt"

  if go vet ./... 2>&1 | tee "${output_file}"; then
    print_success "No vet issues found"
  else
    print_warning "Vet found issues"

    local issues=$(wc -l < "${output_file}" | tr -d ' ')
    MEDIUM_ISSUES=$((MEDIUM_ISSUES + issues))
    TOTAL_ISSUES=$((TOTAL_ISSUES + issues))
  fi
}

run_ineffassign() {
  print_header "Running ineffassign (Ineffectual Assignments)"

  if ! check_tool ineffassign; then
    go install example.com/gordonklaus/ineffassign@latest
  fi

  local output_file="${REPORT_DIR}/ineffassign_${TIMESTAMP}.txt"

  if ineffassign ./... 2>&1 | tee "${output_file}"; then
    print_success "No ineffectual assignments found"
  else
    local issues=$(wc -l < "${output_file}" | tr -d ' ')
    if [ "$issues" -gt 0 ]; then
      LOW_ISSUES=$((LOW_ISSUES + issues))
      TOTAL_ISSUES=$((TOTAL_ISSUES + issues))
      print_warning "Found ${issues} ineffectual assignments"
    fi
  fi
}

run_errcheck() {
  print_header "Running errcheck (Unchecked Errors)"

  if ! check_tool errcheck; then
    go install example.com/kisielk/errcheck@latest
  fi

  local output_file="${REPORT_DIR}/errcheck_${TIMESTAMP}.txt"

  if errcheck ./... 2>&1 | tee "${output_file}"; then
    print_success "All errors are checked"
  else
    local issues=$(wc -l < "${output_file}" | tr -d ' ')
    MEDIUM_ISSUES=$((MEDIUM_ISSUES + issues))
    TOTAL_ISSUES=$((TOTAL_ISSUES + issues))
    print_warning "Found ${issues} unchecked errors"
  fi
}

run_custom_checks() {
  print_header "Running Custom Security Checks"

  local output_file="${REPORT_DIR}/custom_${TIMESTAMP}.txt"

  # Check for common security anti-patterns
  {
    echo "=== Checking for hardcoded credentials ==="
    grep -r -i "password\s*=\s*['\"]" --include="*.go" . || echo "None found"

    echo -e "\n=== Checking for weak cryptography ==="
    grep -r "md5\|sha1" --include="*.go" . || echo "None found"

    echo -e "\n=== Checking for SQL injection patterns ==="
    grep -r "Exec.*\+" --include="*.go" . || echo "None found"

    echo -e "\n=== Checking for command injection ==="
    grep -r "exec.Command" --include="*.go" . || echo "None found"

    echo -e "\n=== Checking for unsafe package usage ==="
    grep -r "import.*unsafe" --include="*.go" . || echo "None found"

    echo -e "\n=== Checking for TODO/FIXME security items ==="
    grep -r "TODO.*security\|FIXME.*security" --include="*.go" . || echo "None found"
  } > "${output_file}"

  print_info "Custom checks completed (see ${output_file})"
}

run_dependency_check() {
  print_header "Checking Dependency Licenses"

  if ! check_tool go-licenses; then
    go install example.com/google/go-licenses@latest
  fi

  local output_file="${REPORT_DIR}/licenses_${TIMESTAMP}.csv"

  go-licenses csv ./... 2>&1 > "${output_file}" || true
  print_info "License report generated: ${output_file}"
}

###############################################################################
# Report Generation
###############################################################################

generate_html_report() {
  print_header "Generating HTML Report"

  local report_file="${REPORT_DIR}/security_report_${TIMESTAMP}.html"

  cat > "${report_file}" <<EOF
<!DOCTYPE html>
<html>
<head>
  <title>PAW Blockchain Security Report - ${TIMESTAMP}</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
    .header { background-color: #2c3e50; color: white; padding: 20px; border-radius: 5px; }
    .summary { background-color: white; padding: 20px; margin: 20px 0; border-radius: 5px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
    .critical { color: #e74c3c; font-weight: bold; }
    .high { color: #e67e22; font-weight: bold; }
    .medium { color: #f39c12; font-weight: bold; }
    .low { color: #3498db; }
    .success { color: #27ae60; font-weight: bold; }
    .issue-count { font-size: 48px; margin: 10px 0; }
    table { width: 100%; border-collapse: collapse; margin: 20px 0; background-color: white; }
    th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
    th { background-color: #34495e; color: white; }
    .tool-section { margin: 20px 0; padding: 15px; background-color: white; border-radius: 5px; }
  </style>
</head>
<body>
  <div class="header">
    <h1>PAW Blockchain Security Report</h1>
    <p>Generated: ${TIMESTAMP}</p>
  </div>

  <div class="summary">
    <h2>Executive Summary</h2>
    <div class="issue-count">
      $([ $TOTAL_ISSUES -eq 0 ] && echo '<span class="success">✓ No Issues Found</span>' || echo "<span class=\"critical\">${TOTAL_ISSUES} Issues Found</span>")
    </div>
    <table>
      <tr>
        <th>Severity</th>
        <th>Count</th>
      </tr>
      <tr>
        <td class="critical">Critical</td>
        <td>${CRITICAL_ISSUES}</td>
      </tr>
      <tr>
        <td class="high">High</td>
        <td>${HIGH_ISSUES}</td>
      </tr>
      <tr>
        <td class="medium">Medium</td>
        <td>${MEDIUM_ISSUES}</td>
      </tr>
      <tr>
        <td class="low">Low</td>
        <td>${LOW_ISSUES}</td>
      </tr>
    </table>
  </div>

  <div class="tool-section">
    <h2>Tool Results</h2>
    <ul>
      <li><strong>GoSec:</strong> Security vulnerability scanner</li>
      <li><strong>Staticcheck:</strong> Static analysis tool</li>
      <li><strong>govulncheck:</strong> Go vulnerability database checker</li>
      <li><strong>Nancy:</strong> Dependency vulnerability scanner</li>
      <li><strong>Gitleaks:</strong> Secret scanner</li>
      <li><strong>go vet:</strong> Go standard tool</li>
    </ul>
    <p>Detailed reports available in: <code>${REPORT_DIR}</code></p>
  </div>

  <div class="summary">
    <h2>Recommendations</h2>
    <ul>
      $([ $CRITICAL_ISSUES -gt 0 ] && echo "<li class=\"critical\">Address ${CRITICAL_ISSUES} critical issues immediately</li>" || echo "")
      $([ $HIGH_ISSUES -gt 0 ] && echo "<li class=\"high\">Fix ${HIGH_ISSUES} high severity issues before deployment</li>" || echo "")
      $([ $MEDIUM_ISSUES -gt 0 ] && echo "<li class=\"medium\">Review and fix ${MEDIUM_ISSUES} medium severity issues</li>" || echo "")
      $([ $TOTAL_ISSUES -eq 0 ] && echo "<li class=\"success\">All security checks passed! ✓</li>" || echo "")
      <li>Run security scans regularly in CI/CD pipeline</li>
      <li>Keep dependencies up to date</li>
      <li>Review and address all findings before production deployment</li>
    </ul>
  </div>
</body>
</html>
EOF

  print_success "HTML report generated: ${report_file}"

  # Try to open in browser
  if command -v xdg-open &> /dev/null; then
    xdg-open "${report_file}" 2>/dev/null &
  elif command -v open &> /dev/null; then
    open "${report_file}" 2>/dev/null &
  fi
}

###############################################################################
# Main Execution
###############################################################################

main() {
  print_header "PAW Blockchain Security Scanner"
  print_info "Project: ${PROJECT_ROOT}"
  print_info "Report Directory: ${REPORT_DIR}"

  # Install required tools
  install_tools

  # Run scans based on mode
  if [ "$QUICK_SCAN" = true ]; then
    print_info "Running quick security scan..."
    run_gosec
    run_go_vet
  elif [ "$FULL_SCAN" = true ]; then
    print_info "Running comprehensive security scan..."
    run_gosec
    run_staticcheck
    run_govulncheck
    run_nancy
    run_gitleaks
    run_go_vet
    run_ineffassign
    run_errcheck
    run_custom_checks
    run_dependency_check
  else
    # Default scan
    print_info "Running standard security scan..."
    run_gosec
    run_staticcheck
    run_govulncheck
    run_go_vet
    run_gitleaks
  fi

  # Generate report if requested
  if [ "$GENERATE_REPORT" = true ]; then
    generate_html_report
  fi

  # Print summary
  print_header "Security Scan Summary"
  echo "Total Issues: ${TOTAL_ISSUES}"
  echo "  Critical: ${CRITICAL_ISSUES}"
  echo "  High: ${HIGH_ISSUES}"
  echo "  Medium: ${MEDIUM_ISSUES}"
  echo "  Low: ${LOW_ISSUES}"
  echo ""
  echo "Reports saved to: ${REPORT_DIR}"

  # Exit with error in CI mode if issues found
  if [ "$CI_MODE" = true ] && [ "$TOTAL_ISSUES" -gt 0 ]; then
    print_error "Security scan failed in CI mode"
    exit 1
  fi

  if [ "$TOTAL_ISSUES" -eq 0 ]; then
    print_success "All security checks passed! ✓"
  else
    print_warning "Security issues found - please review and fix"
  fi
}

# Run main function
main "$@"
