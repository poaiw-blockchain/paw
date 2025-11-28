# PAW Blockchain - Pre-commit Hooks Installation Script (PowerShell)
#
# This script installs  hooks for the PAW blockchain project on Windows.
# It supports both the Python-based pre-commit framework and Husky for Node.js.
#
# Usage:
#   .\scripts\install-hooks.ps1 [-Method pre-commit|husky|both]
#
# Example:
#   .\scripts\install-hooks.ps1
#   .\scripts\install-hooks.ps1 -Method husky

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet('pre-commit', 'husky', 'both')]
    [string]$Method = 'pre-commit'
)

# Configuration
$ProjectRoot = Split-Path -Parent $PSScriptRoot
$HooksDir = Join-Path $ProjectRoot "\hooks"

# Colors
$Colors = @{
    Red = "Red"
    Green = "Green"
    Yellow = "Yellow"
    Blue = "Cyan"
}

function Write-Header {
    Write-Host ""
    Write-Host "╔════════════════════════════════════════════════════════╗" -ForegroundColor $Colors.Blue
    Write-Host "║   PAW Blockchain -  Hooks Installation Script      ║" -ForegroundColor $Colors.Blue
    Write-Host "╚════════════════════════════════════════════════════════╝" -ForegroundColor $Colors.Blue
    Write-Host ""
}

function Write-Info {
    param([string]$Message)
    Write-Host "ℹ " -ForegroundColor $Colors.Blue -NoNewline
    Write-Host $Message
}

function Write-Success {
    param([string]$Message)
    Write-Host "✓ " -ForegroundColor $Colors.Green -NoNewline
    Write-Host $Message
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠ " -ForegroundColor $Colors.Yellow -NoNewline
    Write-Host $Message
}

function Write-Error {
    param([string]$Message)
    Write-Host "✗ " -ForegroundColor $Colors.Red -NoNewline
    Write-Host $Message
}

function Test-GitRepository {
    if (-not (Test-Path (Join-Path $ProjectRoot ""))) {
        Write-Error "Not a  repository. Please run this script from the PAW project root."
        exit 1
    }
    Write-Success " repository detected"
}

function Test-Python {
    try {
        $pythonVersion = & python --version 2>&1
        Write-Success "Python found: $pythonVersion"
        return $true
    } catch {
        Write-Warning "Python not found"
        return $false
    }
}

function Test-Node {
    try {
        $nodeVersion = & node --version 2>&1
        Write-Success "Node.js found: $nodeVersion"
        return $true
    } catch {
        Write-Warning "Node.js not found"
        return $false
    }
}

function Test-Go {
    try {
        $goVersion = & go version 2>&1
        Write-Success "Go found: $goVersion"
        return $true
    } catch {
        Write-Warning "Go not found"
        return $false
    }
}

function Install-PreCommit {
    Write-Info "Installing pre-commit framework..."

    try {
        $precommitVersion = & pre-commit --version 2>&1
        Write-Success "pre-commit already installed: $precommitVersion"
    } catch {
        Write-Info "Installing pre-commit via pip..."
        try {
            & pip install --user pre-commit
            Write-Success "pre-commit installed successfully"
        } catch {
            Write-Error "Failed to install pre-commit. Please install Python and pip first."
            exit 1
        }
    }

    # Install the  hook scripts
    Write-Info "Installing pre-commit hooks..."
    Push-Location $ProjectRoot
    & pre-commit install
    & pre-commit install --hook-type commit-msg
    Pop-Location
    Write-Success "pre-commit hooks installed"

    # Initialize secrets baseline if it doesn't exist
    $secretsBaseline = Join-Path $ProjectRoot ".secrets.baseline"
    if (-not (Test-Path $secretsBaseline)) {
        Write-Info "Creating secrets baseline..."
        try {
            & detect-secrets scan > $secretsBaseline 2>$null
        } catch {
            New-Item -Path $secretsBaseline -ItemType File -Force | Out-Null
        }
        Write-Success "Secrets baseline created"
    }
}

function Install-Husky {
    Write-Info "Installing Node.js dependencies..."

    $packageJson = Join-Path $ProjectRoot "package.json"
    if (-not (Test-Path $packageJson)) {
        Write-Error "package.json not found"
        return $false
    }

    Push-Location $ProjectRoot

    try {
        & npm install
        Write-Success "npm dependencies installed"
    } catch {
        Write-Error "Failed to install npm dependencies"
        Pop-Location
        return $false
    }

    # Install Husky hooks
    Write-Info "Setting up Husky..."
    & npx husky install

    # Create Husky hooks directory
    $huskyDir = Join-Path $ProjectRoot ".husky"
    New-Item -Path $huskyDir -ItemType Directory -Force | Out-Null

    # Create pre-commit hook
    $preCommitHook = Join-Path $huskyDir "pre-commit"
    @'
#!/bin/sh
. "$(dirname "$0")/_/husky.sh"

# Run pre-commit checks
echo "Running pre-commit checks..."

# Run Go formatting check
if  diff --cached --name-only | grep -q '\.go$'; then
    echo "Checking Go formatting..."
    go fmt ./...
    go vet ./...
fi

# Run JavaScript linting
if  diff --cached --name-only | grep -q '\.js$'; then
    echo "Linting JavaScript files..."
    npm run lint:js
fi

# Run Python formatting
if  diff --cached --name-only | grep -q '\.py$'; then
    echo "Formatting Python files..."
    if command -v black &> /dev/null; then
        black --line-length=100 .
    fi
fi

echo "✓ Pre-commit checks passed"
'@ | Out-File -FilePath $preCommitHook -Encoding utf8 -NoNewline

    # Create commit-msg hook
    $commitMsgHook = Join-Path $huskyDir "commit-msg"
    @'
#!/bin/sh
. "$(dirname "$0")/_/husky.sh"

# Validate commit message format
npx --no-install commitlint --edit "$1"
'@ | Out-File -FilePath $commitMsgHook -Encoding utf8 -NoNewline

    # Create pre-push hook
    $prePushHook = Join-Path $huskyDir "pre-push"
    @'
#!/bin/sh
. "$(dirname "$0")/_/husky.sh"

echo "Running pre-push checks..."

# Run tests
if command -v go &> /dev/null; then
    echo "Running Go tests..."
    go test -short -race ./...
fi

echo "✓ Pre-push checks passed"
'@ | Out-File -FilePath $prePushHook -Encoding utf8 -NoNewline

    Pop-Location
    Write-Success "Husky hooks installed"
    return $true
}

function Install-CustomHooks {
    Write-Info "Installing custom  hooks..."
    # Custom hooks are referenced by pre-commit config
    Write-Success "Custom hooks configured"
}

# Main installation flow
function Main {
    Write-Header
    Write-Info "Starting installation..."
    Write-Host ""

    # Check prerequisites
    Test-GitRepository

    $hasPython = Test-Python
    $hasNode = Test-Node
    $hasGo = Test-Go

    Write-Host ""

    # Install based on method
    switch ($Method) {
        'pre-commit' {
            if ($hasPython) {
                Install-PreCommit
            } else {
                Write-Error "Python is required for pre-commit framework"
                exit 1
            }
        }
        'husky' {
            if ($hasNode) {
                Install-Husky
            } else {
                Write-Error "Node.js is required for Husky"
                exit 1
            }
        }
        'both' {
            $installed = $false

            if ($hasPython) {
                Install-PreCommit
                $installed = $true
            }

            if ($hasNode) {
                Install-Husky
                $installed = $true
            }

            if (-not $installed) {
                Write-Error "Neither Python nor Node.js is installed. Please install at least one."
                exit 1
            }
        }
    }

    Install-CustomHooks

    Write-Host ""
    Write-Host "╔════════════════════════════════════════════════════════╗" -ForegroundColor $Colors.Green
    Write-Host "║           Installation completed successfully!         ║" -ForegroundColor $Colors.Green
    Write-Host "╚════════════════════════════════════════════════════════╝" -ForegroundColor $Colors.Green
    Write-Host ""
    Write-Info "Next steps:"
    Write-Host "  1. Test the hooks:  commit --allow-empty -m 'test: verify hooks'"
    Write-Host "  2. Run all hooks manually: pre-commit run --all-files"
    Write-Host "  3. Update hooks: pre-commit autoupdate"
    Write-Host ""
    Write-Info "To bypass hooks (use sparingly):"
    Write-Host "   commit --no-verify -m 'message'"
    Write-Host ""
}

Main
