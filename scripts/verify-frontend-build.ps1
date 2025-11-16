# Frontend Build Verification Script (PowerShell)
# This script verifies that the frontend build pipeline is working correctly

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir

# Counters
$TestsPassed = 0
$TestsFailed = 0

# Function to print colored output
function Print-Success {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor Green
    $script:TestsPassed++
}

function Print-Error {
    param([string]$Message)
    Write-Host "✗ $Message" -ForegroundColor Red
    $script:TestsFailed++
}

function Print-Info {
    param([string]$Message)
    Write-Host "ℹ $Message" -ForegroundColor Yellow
}

function Print-Header {
    param([string]$Message)
    Write-Host ""
    Write-Host "================================================" -ForegroundColor Cyan
    Write-Host $Message -ForegroundColor Cyan
    Write-Host "================================================" -ForegroundColor Cyan
}

# Check prerequisites
Print-Header "Checking Prerequisites"

try {
    $NodeVersion = node --version
    Print-Success "Node.js installed: $NodeVersion"
} catch {
    Print-Error "Node.js not found. Please install Node.js >= 18.0.0"
    exit 1
}

try {
    $NpmVersion = npm --version
    Print-Success "npm installed: $NpmVersion"
} catch {
    Print-Error "npm not found. Please install npm >= 9.0.0"
    exit 1
}

# Verify Exchange Frontend
Print-Header "Verifying Exchange Frontend"

$ExchangeDir = Join-Path $ProjectRoot "external\crypto\exchange-frontend"

if (Test-Path $ExchangeDir) {
    Print-Success "Exchange frontend directory exists"
} else {
    Print-Error "Exchange frontend directory not found"
    exit 1
}

Set-Location $ExchangeDir

# Check package.json
if (Test-Path "package.json") {
    Print-Success "package.json exists"
} else {
    Print-Error "package.json not found"
}

# Check configuration files
$ConfigFiles = @("vite.config.js", ".env.example", ".eslintrc.json", ".prettierrc.json")
foreach ($file in $ConfigFiles) {
    if (Test-Path $file) {
        Print-Success "$file exists"
    } else {
        Print-Error "$file not found"
    }
}

# Check security files
$SecurityFiles = @("config.js", "security.js", "api-client.js", "websocket-client.js")
foreach ($file in $SecurityFiles) {
    if (Test-Path $file) {
        Print-Success "$file exists"
    } else {
        Print-Error "$file not found"
    }
}

# Check HTML files
if (Test-Path "index-secure.html") {
    Print-Success "index-secure.html exists"
} else {
    Print-Error "index-secure.html not found"
}

# Check for CSP headers in HTML
$HtmlContent = Get-Content "index-secure.html" -Raw
if ($HtmlContent -match "Content-Security-Policy") {
    Print-Success "CSP headers found in index-secure.html"
} else {
    Print-Error "CSP headers not found in index-secure.html"
}

# Install dependencies
Print-Info "Installing dependencies..."
try {
    npm install --silent | Out-Null
    Print-Success "Dependencies installed successfully"
} catch {
    Print-Error "Failed to install dependencies"
}

# Run linting
Print-Info "Running ESLint..."
try {
    npm run lint | Out-Null
    Print-Success "Linting passed"
} catch {
    Print-Error "Linting failed"
}

# Run build
Print-Info "Building for production..."
try {
    npm run build | Out-Null
    Print-Success "Build completed successfully"
} catch {
    Print-Error "Build failed"
    exit 1
}

# Check build output
if (Test-Path "dist") {
    Print-Success "Build output directory (dist/) created"

    # Check for key files
    if (Get-ChildItem -Path "dist" -Filter "*.html" -ErrorAction SilentlyContinue) {
        Print-Success "HTML files present in dist/"
    } else {
        Print-Error "No HTML files in dist/"
    }

    if (Get-ChildItem -Path "dist\assets" -Filter "*.js" -ErrorAction SilentlyContinue) {
        Print-Success "JavaScript files present in dist/assets/"
    } else {
        Print-Error "No JavaScript files in dist/assets/"
    }

    # Check for compressed files
    if (Get-ChildItem -Path "dist" -Filter "*.gz" -Recurse -ErrorAction SilentlyContinue) {
        Print-Success "Gzip compressed files present"
    } else {
        Print-Info "No gzip compressed files (optional)"
    }

    # Check build size
    $DistSize = (Get-ChildItem -Path "dist" -Recurse | Measure-Object -Property Length -Sum).Sum / 1MB
    Print-Info "Build size: $([math]::Round($DistSize, 2)) MB"

} else {
    Print-Error "Build output directory not created"
}

# Verify Browser Wallet Extension
Print-Header "Verifying Browser Wallet Extension"

$WalletDir = Join-Path $ProjectRoot "external\crypto\browser-wallet-extension"

if (Test-Path $WalletDir) {
    Print-Success "Wallet extension directory exists"
} else {
    Print-Error "Wallet extension directory not found"
    exit 1
}

Set-Location $WalletDir

# Check package.json
if (Test-Path "package.json") {
    Print-Success "package.json exists"
} else {
    Print-Error "package.json not found"
}

# Check configuration files
$ConfigFiles = @("build.js", "manifest.json", ".eslintrc.json", ".prettierrc.json")
foreach ($file in $ConfigFiles) {
    if (Test-Path $file) {
        Print-Success "$file exists"
    } else {
        Print-Error "$file not found"
    }
}

# Check extension files
$ExtensionFiles = @("popup.html", "popup.js", "background.js", "styles.css")
foreach ($file in $ExtensionFiles) {
    if (Test-Path $file) {
        Print-Success "$file exists"
    } else {
        Print-Error "$file not found"
    }
}

# Install dependencies
Print-Info "Installing dependencies..."
try {
    npm install --silent | Out-Null
    Print-Success "Dependencies installed successfully"
} catch {
    Print-Error "Failed to install dependencies"
}

# Run linting
Print-Info "Running ESLint..."
try {
    npm run lint | Out-Null
    Print-Success "Linting passed"
} catch {
    Print-Error "Linting failed"
}

# Run build
Print-Info "Building extension..."
try {
    npm run build | Out-Null
    Print-Success "Build completed successfully"
} catch {
    Print-Error "Build failed"
    exit 1
}

# Check build output
if (Test-Path "dist") {
    Print-Success "Build output directory (dist/) created"

    # Check for key files
    $ExtensionFiles = @("popup.html", "popup.js", "background.js", "manifest.json")
    foreach ($file in $ExtensionFiles) {
        if (Test-Path "dist\$file") {
            Print-Success "$file present in dist/"
        } else {
            Print-Error "$file not found in dist/"
        }
    }

    # Check if manifest is valid JSON
    try {
        $manifest = Get-Content "dist\manifest.json" -Raw | ConvertFrom-Json
        Print-Success "manifest.json is valid JSON"
    } catch {
        Print-Error "manifest.json is not valid JSON"
    }

} else {
    Print-Error "Build output directory not created"
}

# Verify CI/CD Pipeline
Print-Header "Verifying CI/CD Pipeline"

Set-Location $ProjectRoot

$WorkflowFile = ".github\workflows\frontend-ci.yml"
if (Test-Path $WorkflowFile) {
    Print-Success "Frontend CI/CD workflow exists"

    $WorkflowContent = Get-Content $WorkflowFile -Raw

    # Check for key jobs
    if ($WorkflowContent -match "lint-and-format:") {
        Print-Success "Lint and format job configured"
    }

    if ($WorkflowContent -match "security-scan:") {
        Print-Success "Security scan job configured"
    }

    if ($WorkflowContent -match "build-exchange-frontend:") {
        Print-Success "Exchange frontend build job configured"
    }

    if ($WorkflowContent -match "build-wallet-extension:") {
        Print-Success "Wallet extension build job configured"
    }
} else {
    Print-Error "Frontend CI/CD workflow not found"
}

# Verify Documentation
Print-Header "Verifying Documentation"

$DocFiles = @("FRONTEND_BUILD_SECURITY_SUMMARY.md", "FRONTEND_QUICK_START.md")
foreach ($file in $DocFiles) {
    if (Test-Path $file) {
        Print-Success "$file exists"
    } else {
        Print-Error "$file not found"
    }
}

# Final Summary
Print-Header "Verification Summary"

Write-Host ""
Write-Host "Tests Passed: $TestsPassed"
Write-Host "Tests Failed: $TestsFailed"
Write-Host ""

if ($TestsFailed -eq 0) {
    Print-Success "All verifications passed! Frontend build pipeline is ready."
    exit 0
} else {
    Print-Error "Some verifications failed. Please review the errors above."
    exit 1
}
