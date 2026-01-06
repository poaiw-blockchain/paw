# Local CI/CD Pipeline - PowerShell Script
# Replicates  Actions workflows locally for FREE
# Run this before pushing to  to save CI minutes

param(
    [switch]$Quick,      # Run only essential checks
    [switch]$SkipTests,  # Skip time-consuming tests
    [switch]$SecurityOnly # Run only security scans
)

$ErrorActionPreference = "Continue"
$failedChecks = @()
$passedChecks = @()

function Write-Header($text) {
    Write-Host "`n========================================" -ForegroundColor Cyan
    Write-Host " $text" -ForegroundColor Cyan
    Write-Host "========================================`n" -ForegroundColor Cyan
}

function Write-Success($text) {
    Write-Host "✓ $text" -ForegroundColor Green
    $script:passedChecks += $text
}

function Write-Failure($text) {
    Write-Host "✗ $text" -ForegroundColor Red
    $script:failedChecks += $text
}

Write-Host @"
╔══════════════════════════════════════════════════════════╗
║         LOCAL CI/CD PIPELINE - FREE ALTERNATIVE          ║
║              Replicates  Actions                   ║
╚══════════════════════════════════════════════════════════╝
"@ -ForegroundColor Magenta

# Check if virtual environment exists
if (!(Test-Path ".venv") -and !(Test-Path "venv")) {
    Write-Warning "No virtual environment found. Creating one..."
    python -m venv .venv
    .\.venv\Scripts\Activate.ps1
    pip install --upgrade pip
} else {
    if (Test-Path ".venv") { .\.venv\Scripts\Activate.ps1 }
    elseif (Test-Path "venv") { .\venv\Scripts\Activate.ps1 }
}

# ============================================================================
# 1. LINTING - Code Quality Checks
# ============================================================================
if (!$SecurityOnly) {
    Write-Header "LINTING - Code Quality Checks"

    # Install linting tools if needed
    Write-Host "Installing linting tools..." -ForegroundColor Yellow
    pip install -q black isort flake8 pylint mypy 2>$null

    # Black - Code formatting
    Write-Host "`nRunning Black (code formatter)..." -ForegroundColor Yellow
    black --check --diff src/ tests/ scripts/ 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) { Write-Success "Black formatting check" }
    else { Write-Failure "Black formatting check"; black --check src/ tests/ scripts/ }

    # isort - Import sorting
    Write-Host "Running isort (import sorting)..." -ForegroundColor Yellow
    isort --check-only --diff src/ tests/ scripts/ 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) { Write-Success "isort import sorting" }
    else { Write-Failure "isort import sorting"; isort --check-only src/ tests/ }

    # Flake8 - Style guide enforcement
    Write-Host "Running Flake8 (style guide)..." -ForegroundColor Yellow
    flake8 src/ tests/ --max-line-length=100 --extend-ignore=E203,W503 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) { Write-Success "Flake8 style guide" }
    else { Write-Failure "Flake8 style guide"; flake8 src/ tests/ --max-line-length=100 }

    if (!$Quick) {
        # Pylint - Code analysis
        Write-Host "Running Pylint (code analysis)..." -ForegroundColor Yellow
        pylint src/ --exit-zero --output-format=colorized | Out-Null
        if ($LASTEXITCODE -eq 0) { Write-Success "Pylint code analysis" }
        else { Write-Failure "Pylint code analysis" }

        # MyPy - Type checking
        Write-Host "Running MyPy (type checking)..." -ForegroundColor Yellow
        mypy src/ --ignore-missing-imports --no-strict-optional 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) { Write-Success "MyPy type checking" }
        else { Write-Failure "MyPy type checking" }
    }
}

# ============================================================================
# 2. SECURITY SCANNING
# ============================================================================
Write-Header "SECURITY SCANNING"

Write-Host "Installing security tools..." -ForegroundColor Yellow
pip install -q bandit safety semgrep pip-audit 2>$null

# Bandit - Security linter
Write-Host "`nRunning Bandit (security linter)..." -ForegroundColor Yellow
bandit -r src/ -f json -o bandit-report.json 2>$null
bandit -r src/ -f txt --quiet
if ($LASTEXITCODE -eq 0) { Write-Success "Bandit security scan" }
else { Write-Failure "Bandit security scan" }

# Safety - Dependency vulnerability scanner
Write-Host "Running Safety (dependency vulnerabilities)..." -ForegroundColor Yellow
safety check --json --output safety-report.json 2>$null
safety check
if ($LASTEXITCODE -eq 0) { Write-Success "Safety dependency scan" }
else { Write-Failure "Safety dependency scan" }

# pip-audit - Dependency auditing
Write-Host "Running pip-audit (dependency auditing)..." -ForegroundColor Yellow
pip-audit --format json --output pip-audit-report.json 2>$null
pip-audit
if ($LASTEXITCODE -eq 0) { Write-Success "pip-audit dependency audit" }
else { Write-Failure "pip-audit dependency audit" }

if (!$Quick) {
    # Semgrep - SAST scanning
    Write-Host "Running Semgrep (SAST analysis)..." -ForegroundColor Yellow
    semgrep --config=auto src/ --json --output=semgrep-report.json 2>$null
    semgrep --config=auto src/ --quiet
    if ($LASTEXITCODE -eq 0) { Write-Success "Semgrep SAST scan" }
    else { Write-Failure "Semgrep SAST scan" }
}

# ============================================================================
# 3. TESTING
# ============================================================================
if (!$SkipTests -and !$SecurityOnly) {
    Write-Header "TESTING"

    Write-Host "Installing test dependencies..." -ForegroundColor Yellow
    pip install -q pytest pytest-cov pytest-xdist pytest-timeout pytest-benchmark hypothesis 2>$null
}

# ============================================================================
# SUMMARY
# ============================================================================
Write-Header "CI PIPELINE SUMMARY"

Write-Host "`nPassed Checks ($($passedChecks.Count)):" -ForegroundColor Green
foreach ($check in $passedChecks) {
    Write-Host "  ✓ $check" -ForegroundColor Green
}

if ($failedChecks.Count -gt 0) {
    Write-Host "`nFailed Checks ($($failedChecks.Count)):" -ForegroundColor Red
    foreach ($check in $failedChecks) {
        Write-Host "  ✗ $check" -ForegroundColor Red
    }
    Write-Host "`n⚠ Fix failed checks before pushing to !" -ForegroundColor Yellow
    exit 1
} else {
    Write-Host "`n✓ All checks passed! Safe to push to ." -ForegroundColor Green
    Write-Host "💰 You just saved  Actions minutes!" -ForegroundColor Cyan
    exit 0
}
