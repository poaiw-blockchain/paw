# PowerShell version of dev-setup.sh for Windows
# PAW Blockchain Development Setup

$ErrorActionPreference = "Stop"

Write-Host "========================================" -ForegroundColor Blue
Write-Host "  PAW Blockchain Development Setup" -ForegroundColor Blue
Write-Host "========================================" -ForegroundColor Blue
Write-Host ""

# Function to check if command exists
function Test-CommandExists {
    param($command)
    $null = Get-Command $command -ErrorAction SilentlyContinue
    return $?
}

# Check Go installation
Write-Host "`nChecking Go installation..." -ForegroundColor Yellow
if (Test-CommandExists go) {
    $goVersion = go version
    Write-Host "✓ Go is installed: $goVersion" -ForegroundColor Green

    # Check minimum Go version (1.21)
    $versionMatch = $goVersion -match 'go(\d+\.\d+)'
    if ($versionMatch) {
        $version = [version]$matches[1]
        if ($version -ge [version]"1.21") {
            Write-Host "✓ Go version is sufficient" -ForegroundColor Green
        } else {
            Write-Host "✗ Go version 1.21 or higher is required" -ForegroundColor Red
            exit 1
        }
    }
} else {
    Write-Host "✗ Go is not installed" -ForegroundColor Red
    Write-Host "Please install Go 1.21+ from https://golang.org/dl/" -ForegroundColor Yellow
    exit 1
}

# Install Go dependencies
Write-Host "`nInstalling Go dependencies..." -ForegroundColor Yellow
go mod download
go mod verify
Write-Host "✓ Go dependencies installed" -ForegroundColor Green

# Install development tools
Write-Host "`nInstalling development tools..." -ForegroundColor Yellow

# Install golangci-lint
if (-not (Test-CommandExists golangci-lint)) {
    Write-Host "Installing golangci-lint..." -ForegroundColor Yellow
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
} else {
    Write-Host "✓ golangci-lint is already installed" -ForegroundColor Green
}

# Install goimports
if (-not (Test-CommandExists goimports)) {
    Write-Host "Installing goimports..." -ForegroundColor Yellow
    go install golang.org/x/tools/cmd/goimports@latest
} else {
    Write-Host "✓ goimports is already installed" -ForegroundColor Green
}

# Install misspell
if (-not (Test-CommandExists misspell)) {
    Write-Host "Installing misspell..." -ForegroundColor Yellow
    go install github.com/client9/misspell/cmd/misspell@latest
} else {
    Write-Host "✓ misspell is already installed" -ForegroundColor Green
}

# Install buf (for protobuf)
if (-not (Test-CommandExists buf)) {
    Write-Host "Installing buf..." -ForegroundColor Yellow
    Write-Host "Please install buf manually from: https://docs.buf.build/installation" -ForegroundColor Yellow
} else {
    Write-Host "✓ buf is already installed" -ForegroundColor Green
}

# Install GoReleaser
if (-not (Test-CommandExists goreleaser)) {
    Write-Host "Installing goreleaser..." -ForegroundColor Yellow
    go install github.com/goreleaser/goreleaser@latest
} else {
    Write-Host "✓ goreleaser is already installed" -ForegroundColor Green
}

# Check Python installation
Write-Host "`nChecking Python installation..." -ForegroundColor Yellow
if (Test-CommandExists python) {
    $pythonVersion = python --version
    Write-Host "✓ Python is installed: $pythonVersion" -ForegroundColor Green

    # Install Python dependencies if requirements.txt exists
    if (Test-Path "wallet\requirements.txt") {
        Write-Host "Installing Python dependencies..." -ForegroundColor Yellow
        python -m pip install --user -r wallet\requirements.txt
        Write-Host "✓ Python dependencies installed" -ForegroundColor Green
    }
} else {
    Write-Host "! Python not found (optional for wallet scripts)" -ForegroundColor Yellow
}

# Check Node.js installation
Write-Host "`nChecking Node.js installation..." -ForegroundColor Yellow
if (Test-CommandExists node) {
    $nodeVersion = node --version
    Write-Host "✓ Node.js is installed: $nodeVersion" -ForegroundColor Green

    # Install Node dependencies if package.json exists
    if (Test-Path "package.json") {
        Write-Host "Installing Node.js dependencies..." -ForegroundColor Yellow
        npm install
        Write-Host "✓ Node.js dependencies installed" -ForegroundColor Green
    }
} else {
    Write-Host "! Node.js not found (optional for frontend)" -ForegroundColor Yellow
}

# Setup Git hooks
Write-Host "`nSetting up Git hooks..." -ForegroundColor Yellow
if (Test-Path ".git") {
    $hookPath = ".git\hooks"
    if (-not (Test-Path $hookPath)) {
        New-Item -ItemType Directory -Path $hookPath | Out-Null
    }

    $preCommitHook = @'
#!/bin/bash
echo "Running pre-commit checks..."

# Run linter
echo "Running golangci-lint..."
golangci-lint run --timeout=10m || exit 1

# Run tests
echo "Running tests..."
go test -short ./... || exit 1

echo "Pre-commit checks passed!"
'@

    Set-Content -Path "$hookPath\pre-commit" -Value $preCommitHook
    Write-Host "✓ Git hooks configured" -ForegroundColor Green
} else {
    Write-Host "! Not a git repository, skipping Git hooks" -ForegroundColor Yellow
}

# Check Docker installation
Write-Host "`nChecking Docker installation..." -ForegroundColor Yellow
if (Test-CommandExists docker) {
    $dockerVersion = docker --version
    Write-Host "✓ Docker is installed: $dockerVersion" -ForegroundColor Green

    if (Test-CommandExists docker-compose) {
        $composeVersion = docker-compose --version
        Write-Host "✓ Docker Compose is installed: $composeVersion" -ForegroundColor Green
    } else {
        Write-Host "! Docker Compose not found" -ForegroundColor Yellow
    }
} else {
    Write-Host "! Docker not found (optional for containerized development)" -ForegroundColor Yellow
}

# Run initial tests
Write-Host "`nRunning initial tests..." -ForegroundColor Yellow
$testResult = go test -short ./... 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ All tests passed" -ForegroundColor Green
} else {
    Write-Host "✗ Some tests failed" -ForegroundColor Red
    Write-Host "This is normal for a new setup. Fix issues and run 'make test' again." -ForegroundColor Yellow
}

# Create necessary directories
Write-Host "`nCreating necessary directories..." -ForegroundColor Yellow
@("build", "logs", "data") | ForEach-Object {
    if (-not (Test-Path $_)) {
        New-Item -ItemType Directory -Path $_ | Out-Null
    }
}
Write-Host "✓ Directories created" -ForegroundColor Green

# Summary
Write-Host "`n========================================" -ForegroundColor Blue
Write-Host "  Setup Complete!" -ForegroundColor Blue
Write-Host "========================================" -ForegroundColor Blue
Write-Host "Your PAW development environment is ready!" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. Build the project: " -NoNewline -ForegroundColor Yellow
Write-Host "make build" -ForegroundColor Green
Write-Host "  2. Run tests: " -NoNewline -ForegroundColor Yellow
Write-Host "make test" -ForegroundColor Green
Write-Host "  3. Start local testnet: " -NoNewline -ForegroundColor Yellow
Write-Host "make localnet-start" -ForegroundColor Green
Write-Host "  4. Run linter: " -NoNewline -ForegroundColor Yellow
Write-Host "make lint" -ForegroundColor Green
Write-Host "  5. Format code: " -NoNewline -ForegroundColor Yellow
Write-Host "make format" -ForegroundColor Green
Write-Host ""
Write-Host "For Docker development:" -ForegroundColor Yellow
Write-Host "  docker-compose -f docker-compose.dev.yml up" -ForegroundColor Green
Write-Host ""
