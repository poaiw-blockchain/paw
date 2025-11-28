# PowerShell version of clean.sh for Windows
# PAW Blockchain Cleanup

$ErrorActionPreference = "Continue"

Write-Host "========================================" -ForegroundColor Blue
Write-Host "  PAW Blockchain Cleanup" -ForegroundColor Blue
Write-Host "========================================" -ForegroundColor Blue
Write-Host ""

# Function to confirm action
function Confirm-Action {
    param([string]$message)
    $response = Read-Host "$message [y/N]"
    return $response -match '^[Yy]$'
}

# Clean build artifacts
Write-Host "Cleaning build artifacts..." -ForegroundColor Yellow
if (Test-Path "build") {
    Remove-Item -Recurse -Force "build\*" -ErrorAction SilentlyContinue
    Write-Host "✓ Removed build directory contents" -ForegroundColor Green
} else {
    Write-Host "! Build directory does not exist" -ForegroundColor Yellow
}

# Clean binaries
$goPath = go env GOPATH
if (Test-Path "$goPath\bin") {
    if (Confirm-Action "Remove installed binaries (pawd.exe, pawcli.exe)?") {
        Remove-Item -Force "$goPath\bin\pawd.exe" -ErrorAction SilentlyContinue
        Remove-Item -Force "$goPath\bin\pawcli.exe" -ErrorAction SilentlyContinue
        Write-Host "✓ Removed installed binaries" -ForegroundColor Green
    }
}

# Clean test cache
Write-Host "`nCleaning test cache..." -ForegroundColor Yellow
go clean -testcache
Write-Host "✓ Test cache cleared" -ForegroundColor Green

# Clean build cache
Write-Host "Cleaning build cache..." -ForegroundColor Yellow
go clean -cache
Write-Host "✓ Build cache cleared" -ForegroundColor Green

# Clean module cache (optional)
if (Confirm-Action "Clean Go module cache? (This will require re-downloading dependencies)") {
    go clean -modcache
    Write-Host "✓ Module cache cleared" -ForegroundColor Green
}

# Clean coverage files
Write-Host "`nCleaning coverage files..." -ForegroundColor Yellow
Remove-Item -Force "coverage.txt", "coverage.html", "*.coverprofile" -ErrorAction SilentlyContinue
Write-Host "✓ Coverage files removed" -ForegroundColor Green

# Clean proto-generated files
if (Confirm-Action "Clean proto-generated files? (Requires regeneration)") {
    Write-Host "Cleaning proto-generated files..." -ForegroundColor Yellow
    Get-ChildItem -Recurse -Filter "*.pb.go" | Remove-Item -Force
    Get-ChildItem -Recurse -Filter "*.pb.gw.go" | Remove-Item -Force
    Write-Host "✓ Proto-generated files removed" -ForegroundColor Green
}

# Clean node data
$pawHome = "$env:USERPROFILE\.paw"
if (Test-Path $pawHome) {
    if (Confirm-Action "Clean blockchain data (~/.paw)?") {
        Write-Host "Cleaning blockchain data..." -ForegroundColor Yellow
        Remove-Item -Recurse -Force $pawHome
        Write-Host "✓ Blockchain data removed" -ForegroundColor Green
    }
}

# Clean local testnet data
if (Test-Path "data") {
    if (Confirm-Action "Clean local testnet data (.\data)?") {
        Write-Host "Cleaning testnet data..." -ForegroundColor Yellow
        Remove-Item -Recurse -Force "data\*" -ErrorAction SilentlyContinue
        Write-Host "✓ Testnet data removed" -ForegroundColor Green
    }
}

# Clean node_modules
if (Test-Path "node_modules") {
    if (Confirm-Action "Clean node_modules?") {
        Write-Host "Cleaning node_modules..." -ForegroundColor Yellow
        Remove-Item -Recurse -Force "node_modules"
        Write-Host "✓ node_modules removed" -ForegroundColor Green
    }
}

# Clean Python cache
Write-Host "`nCleaning Python cache..." -ForegroundColor Yellow
Get-ChildItem -Recurse -Directory -Filter "__pycache__" | Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
Get-ChildItem -Recurse -Filter "*.pyc" | Remove-Item -Force -ErrorAction SilentlyContinue
Get-ChildItem -Recurse -Filter "*.pyo" | Remove-Item -Force -ErrorAction SilentlyContinue
Write-Host "✓ Python cache cleaned" -ForegroundColor Green

# Clean log files
if (Test-Path "logs") {
    if (Confirm-Action "Clean log files?") {
        Write-Host "Cleaning log files..." -ForegroundColor Yellow
        Remove-Item -Recurse -Force "logs\*" -ErrorAction SilentlyContinue
        Write-Host "✓ Log files removed" -ForegroundColor Green
    }
}

# Clean Docker volumes (optional)
if (Get-Command docker -ErrorAction SilentlyContinue) {
    if (Confirm-Action "Clean Docker volumes?") {
        Write-Host "Cleaning Docker volumes..." -ForegroundColor Yellow
        docker-compose -f docker-compose.dev.yml down -v 2>$null
        Write-Host "✓ Docker volumes removed" -ForegroundColor Green
    }
}

# Clean temporary files
Write-Host "`nCleaning temporary files..." -ForegroundColor Yellow
Get-ChildItem -Recurse -Filter "*.tmp" | Remove-Item -Force -ErrorAction SilentlyContinue
Get-ChildItem -Recurse -Filter "*.log" | Remove-Item -Force -ErrorAction SilentlyContinue
Get-ChildItem -Recurse -Filter ".DS_Store" | Remove-Item -Force -ErrorAction SilentlyContinue
Write-Host "✓ Temporary files removed" -ForegroundColor Green

# Clean pre-commit cache
if (Test-Path ".pre-commit-cache") {
    Remove-Item -Recurse -Force ".pre-commit-cache"
    Write-Host "✓ Pre-commit cache removed" -ForegroundColor Green
}

# Clean GoReleaser dist
if (Test-Path "dist") {
    if (Confirm-Action "Clean GoReleaser dist directory?") {
        Remove-Item -Recurse -Force "dist"
        Write-Host "✓ GoReleaser dist removed" -ForegroundColor Green
    }
}

Write-Host "`n========================================" -ForegroundColor Blue
Write-Host "  Cleanup Complete!" -ForegroundColor Blue
Write-Host "========================================" -ForegroundColor Blue
Write-Host "Your workspace has been cleaned." -ForegroundColor Green
Write-Host ""
Write-Host "To restore dependencies and rebuild:" -ForegroundColor Yellow
Write-Host "  go mod download" -ForegroundColor Green
Write-Host "  make build" -ForegroundColor Green
Write-Host ""
