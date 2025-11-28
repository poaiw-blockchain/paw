# Comprehensive Security Audit Script for PAW Blockchain (PowerShell)
# This script runs multiple security scanning tools on Windows

param(
    [switch]$SkipGoSec = $false,
    [switch]$SkipNancy = $false,
    [switch]$SkipGovulncheck = $false,
    [switch]$SkipTrivy = $false,
    [switch]$SkipGitLeaks = $false,
    [switch]$GenerateReportsOnly = $false
)

$ErrorActionPreference = "Continue"

# Colors for output
function Write-Header {
    param([string]$Message)
    Write-Host "`n==== $Message ====`n" -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠ $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "✗ $Message" -ForegroundColor Red
}

# Get script location
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
Set-Location $ProjectRoot

Write-Host "================================================" -ForegroundColor Cyan
Write-Host "PAW Blockchain - Comprehensive Security Audit" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""

# Track overall status
$AuditFailed = $false

# Create security directory if it doesn't exist
if (-not (Test-Path "security")) {
    New-Item -ItemType Directory -Path "security" | Out-Null
}

# Helper function to check if command exists
function Test-Command {
    param([string]$Command)
    try {
        Get-Command $Command -ErrorAction Stop | Out-Null
        return $true
    }
    catch {
        return $false
    }
}

# 1. GoSec - Go Security Scanner
if (-not $SkipGoSec) {
    Write-Header "Running GoSec (Go Security Scanner)"
    if (Test-Command "gosec") {
        try {
            gosec -conf security\.gosec.yml -fmt=json -out=security\gosec-report.json .\...
            if ($LASTEXITCODE -eq 0) {
                Write-Success "GoSec scan completed - No critical issues found"
                gosec -conf security\.gosec.yml .\...
            }
            else {
                Write-Error "GoSec found security issues"
                $AuditFailed = $true
                gosec -conf security\.gosec.yml .\...
            }
        }
        catch {
            Write-Error "GoSec execution failed: $_"
            $AuditFailed = $true
        }
    }
    else {
        Write-Warning "GoSec not installed. Run: go install example.com/securego/gosec/v2/cmd/gosec@latest"
    }
}

# 2. Nancy - Dependency Vulnerability Scanner
if (-not $SkipNancy) {
    Write-Header "Running Nancy (Dependency Vulnerability Scanner)"
    if (Test-Command "nancy") {
        try {
            go list -json -m all | nancy sleuth
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Nancy scan completed - No known vulnerabilities"
            }
            else {
                Write-Error "Nancy found vulnerable dependencies"
                $AuditFailed = $true
            }
        }
        catch {
            Write-Error "Nancy execution failed: $_"
            $AuditFailed = $true
        }
    }
    else {
        Write-Warning "Nancy not installed. Run: go install example.com/sonatype-nexus-community/nancy@latest"
    }
}

# 3. Govulncheck - Official Go Vulnerability Scanner
if (-not $SkipGovulncheck) {
    Write-Header "Running Govulncheck (Official Go Vulnerability Scanner)"
    if (Test-Command "govulncheck") {
        try {
            govulncheck .\...
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Govulncheck completed - No vulnerabilities found"
            }
            else {
                Write-Error "Govulncheck found vulnerabilities"
                $AuditFailed = $true
            }
        }
        catch {
            Write-Error "Govulncheck execution failed: $_"
            $AuditFailed = $true
        }
    }
    else {
        Write-Warning "Govulncheck not installed. Run: go install golang.org/x/vuln/cmd/govulncheck@latest"
    }
}

# 4. Trivy - Container and Filesystem Security Scanner
if (-not $SkipTrivy) {
    Write-Header "Running Trivy (Container and Filesystem Security)"
    if (Test-Command "trivy") {
        try {
            trivy fs --security-checks vuln,config --severity HIGH,CRITICAL --format table .
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Trivy filesystem scan completed"
            }
            else {
                Write-Error "Trivy found security issues"
                $AuditFailed = $true
            }

            # Generate JSON report
            trivy fs --security-checks vuln,config --format json --output security\trivy-report.json .
        }
        catch {
            Write-Error "Trivy execution failed: $_"
            $AuditFailed = $true
        }
    }
    else {
        Write-Warning "Trivy not installed. Visit: https://aquasecurityhub.io/trivy/"
    }
}

# 5. GitLeaks - Secret Scanner
if (-not $SkipGitLeaks) {
    Write-Header "Running GitLeaks (Secret Detection)"
    if (Test-Command "gitleaks") {
        try {
            gitleaks detect --verbose --report-path=security\gitleaks-report.json
            if ($LASTEXITCODE -eq 0) {
                Write-Success "GitLeaks scan completed - No secrets detected"
            }
            else {
                Write-Error "GitLeaks found potential secrets"
                $AuditFailed = $true
            }
        }
        catch {
            Write-Error "GitLeaks execution failed: $_"
            $AuditFailed = $true
        }
    }
    else {
        Write-Warning "GitLeaks not installed. Visit: https://example.com/gitleaks/gitleaks"
    }
}

# 6. Go Mod Verify - Dependency Integrity
Write-Header "Verifying Go Module Dependencies"
try {
    go mod verify
    if ($LASTEXITCODE -eq 0) {
        Write-Success "All dependencies verified successfully"
    }
    else {
        Write-Error "Dependency verification failed"
        $AuditFailed = $true
    }
}
catch {
    Write-Error "Go mod verify failed: $_"
    $AuditFailed = $true
}

# 7. Custom Crypto Analysis
Write-Header "Analyzing Cryptographic Usage"
if (Test-Path "security\crypto-check.go") {
    try {
        go run security\crypto-check.go
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Crypto analysis completed"
        }
        else {
            Write-Warning "Crypto analysis found potential issues"
        }
    }
    catch {
        Write-Warning "Crypto analysis failed: $_"
    }
}
else {
    Write-Warning "Crypto check tool not found at security\crypto-check.go"
}

# 8. Check for Weak Crypto Imports
Write-Header "Checking for Weak Cryptographic Imports"
$WeakCryptoFound = $false

$WeakCryptoPatterns = @(
    @{ Pattern = "crypto/md5"; Name = "MD5" },
    @{ Pattern = "crypto/sha1"; Name = "SHA1" },
    @{ Pattern = "crypto/des"; Name = "DES" },
    @{ Pattern = "crypto/rc4"; Name = "RC4" }
)

foreach ($item in $WeakCryptoPatterns) {
    $results = Get-ChildItem -Path . -Filter "*.go" -Recurse -Exclude "*_test.go" |
        Where-Object { $_.FullName -notlike "*\vendor\*" } |
        Select-String -Pattern $item.Pattern

    if ($results) {
        Write-Error "Found $($item.Name) usage (weak crypto)"
        $WeakCryptoFound = $true
    }
}

# Check for math/rand in non-test files
$mathRandResults = Get-ChildItem -Path . -Filter "*.go" -Recurse -Exclude "*_test.go" |
    Where-Object { $_.FullName -notlike "*\vendor\*" } |
    Select-String -Pattern "math/rand"

if ($mathRandResults) {
    Write-Warning "Found math/rand usage (should use crypto/rand for security)"
}

if (-not $WeakCryptoFound) {
    Write-Success "No weak cryptographic imports found"
}
else {
    $AuditFailed = $true
}

# 9. Check for Hardcoded Secrets
Write-Header "Checking for Hardcoded Secrets/Keys"
$SecretPatterns = @(
    "password.*=.*['\`"]",
    "secret.*=.*['\`"]",
    "api_key.*=.*['\`"]",
    "private_key.*=.*['\`"]",
    "token.*=.*['\`"]"
)

$SecretsFound = $false
foreach ($pattern in $SecretPatterns) {
    $results = Get-ChildItem -Path . -Filter "*.go" -Recurse -Exclude "*_test.go" |
        Where-Object { $_.FullName -notlike "*\vendor\*" } |
        Select-String -Pattern $pattern

    if ($results) {
        $SecretsFound = $true
    }
}

if (-not $SecretsFound) {
    Write-Success "No obvious hardcoded secrets found"
}
else {
    Write-Warning "Potential hardcoded secrets detected - review manually"
}

# 10. Check TLS Configuration
Write-Header "Checking TLS Configuration"
$insecureTLS = Get-ChildItem -Path . -Filter "*.go" -Recurse |
    Where-Object { $_.FullName -notlike "*\vendor\*" } |
    Select-String -Pattern "InsecureSkipVerify.*true"

if ($insecureTLS) {
    Write-Error "Found InsecureSkipVerify enabled (insecure TLS)"
    $AuditFailed = $true
}
else {
    Write-Success "No insecure TLS configurations found"
}

# 11. Check File Permissions
Write-Header "Checking File Permission Settings"
$permissivePerms = Get-ChildItem -Path . -Filter "*.go" -Recurse |
    Where-Object { $_.FullName -notlike "*\vendor\*" } |
    Select-String -Pattern "(0777|0666)"

if ($permissivePerms) {
    Write-Warning "Found overly permissive file permissions"
}
else {
    Write-Success "No overly permissive file permissions found"
}

# 12. Generate Summary Report
Write-Header "Generating Summary Report"
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$reportFile = "security\audit-summary-$timestamp.txt"

$gitCommit = try {  rev-parse HEAD 2>$null } catch { "Not a  repository" }

$report = @"
PAW Blockchain Security Audit Summary
======================================
Date: $(Get-Date)
 Commit: $gitCommit

Tools Run:
  - GoSec: $(if (Test-Command 'gosec') { 'Yes' } else { 'No' })
  - Nancy: $(if (Test-Command 'nancy') { 'Yes' } else { 'No' })
  - Govulncheck: $(if (Test-Command 'govulncheck') { 'Yes' } else { 'No' })
  - Trivy: $(if (Test-Command 'trivy') { 'Yes' } else { 'No' })
  - GitLeaks: $(if (Test-Command 'gitleaks') { 'Yes' } else { 'No' })

Results:
  - Overall Status: $(if (-not $AuditFailed) { 'PASSED' } else { 'FAILED' })

Detailed reports generated:
  - GoSec: security\gosec-report.json
  - Trivy: security\trivy-report.json
  - GitLeaks: security\gitleaks-report.json
"@

$report | Out-File -FilePath $reportFile -Encoding UTF8
Write-Success "Summary report saved to: $reportFile"

# Final status
Write-Host "`n================================================" -ForegroundColor Cyan
if (-not $AuditFailed) {
    Write-Host "Security Audit: PASSED" -ForegroundColor Green
    Write-Host "================================================" -ForegroundColor Cyan
    exit 0
}
else {
    Write-Host "Security Audit: FAILED" -ForegroundColor Red
    Write-Host "Please review the issues found above" -ForegroundColor Red
    Write-Host "================================================" -ForegroundColor Cyan
    exit 1
}
