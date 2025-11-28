#!/bin/bash
# PAW Blockchain - Pre-commit Hooks Installation Script
#
# This script installs  hooks for the PAW blockchain project.
# It supports both the Python-based pre-commit framework and Husky for Node.js.
#
# Usage:
#   ./scripts/install-hooks.sh [--method=pre-commit|husky|both]
#
# Windows (PowerShell):
#   bash scripts/install-hooks.sh
#   OR use: scripts/install-hooks.ps1
#
# Linux/Mac:
#   bash scripts/install-hooks.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOOKS_DIR="${PROJECT_ROOT}//hooks"
METHOD="${1:-pre-commit}"

echo -e "${BLUE}╔════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   PAW Blockchain -  Hooks Installation Script      ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════╝${NC}"
echo ""

# Function to print colored messages
print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
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

# Check if we're in a  repository
check_git_repo() {
    if [ ! -d "${PROJECT_ROOT}/" ]; then
        print_error "Not a  repository. Please run this script from the PAW project root."
        exit 1
    fi
    print_success " repository detected"
}

# Check if Python is installed
check_python() {
    if command -v python3 &> /dev/null; then
        PYTHON_VERSION=$(python3 --version | cut -d' ' -f2)
        print_success "Python 3 found: ${PYTHON_VERSION}"
        return 0
    else
        print_warning "Python 3 not found"
        return 1
    fi
}

# Check if Node.js is installed
check_node() {
    if command -v node &> /dev/null; then
        NODE_VERSION=$(node --version)
        print_success "Node.js found: ${NODE_VERSION}"
        return 0
    else
        print_warning "Node.js not found"
        return 1
    fi
}

# Check if Go is installed
check_go() {
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | cut -d' ' -f3)
        print_success "Go found: ${GO_VERSION}"
        return 0
    else
        print_warning "Go not found"
        return 1
    fi
}

# Install pre-commit framework
install_precommit() {
    print_info "Installing pre-commit framework..."

    if command -v pre-commit &> /dev/null; then
        print_success "pre-commit already installed: $(pre-commit --version)"
    else
        print_info "Installing pre-commit via pip..."
        if command -v pip3 &> /dev/null; then
            pip3 install --user pre-commit
            print_success "pre-commit installed successfully"
        else
            print_error "pip3 not found. Please install Python and pip first."
            exit 1
        fi
    fi

    # Install the  hook scripts
    print_info "Installing pre-commit hooks..."
    cd "${PROJECT_ROOT}"
    pre-commit install
    pre-commit install --hook-type commit-msg
    print_success "pre-commit hooks installed"

    # Initialize secrets baseline if it doesn't exist
    if [ ! -f "${PROJECT_ROOT}/.secrets.baseline" ]; then
        print_info "Creating secrets baseline..."
        detect-secrets scan > .secrets.baseline 2>/dev/null || touch .secrets.baseline
        print_success "Secrets baseline created"
    fi
}

# Install Node.js dependencies and Husky
install_husky() {
    print_info "Installing Node.js dependencies..."

    if [ ! -f "${PROJECT_ROOT}/package.json" ]; then
        print_error "package.json not found"
        return 1
    fi

    cd "${PROJECT_ROOT}"

    # Install npm dependencies
    if command -v npm &> /dev/null; then
        npm install
        print_success "npm dependencies installed"
    else
        print_error "npm not found. Please install Node.js first."
        return 1
    fi

    # Install Husky hooks
    print_info "Setting up Husky..."
    npx husky install

    # Create Husky hooks directory
    mkdir -p .husky

    # Create pre-commit hook
    cat > .husky/pre-commit << 'EOF'
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
EOF
    chmod +x .husky/pre-commit

    # Create commit-msg hook
    cat > .husky/commit-msg << 'EOF'
#!/bin/sh
. "$(dirname "$0")/_/husky.sh"

# Validate commit message format
npx --no-install commitlint --edit "$1"
EOF
    chmod +x .husky/commit-msg

    # Create pre-push hook
    cat > .husky/pre-push << 'EOF'
#!/bin/sh
. "$(dirname "$0")/_/husky.sh"

echo "Running pre-push checks..."

# Run tests
if command -v go &> /dev/null; then
    echo "Running Go tests..."
    go test -short -race ./...
fi

echo "✓ Pre-push checks passed"
EOF
    chmod +x .husky/pre-push

    print_success "Husky hooks installed"
}

# Install custom  hooks
install_custom_hooks() {
    print_info "Installing custom  hooks..."

    # Make hook scripts executable
    chmod +x "${PROJECT_ROOT}/scripts/hooks/"*.sh 2>/dev/null || true

    print_success "Custom hooks configured"
}

# Main installation flow
main() {
    echo ""
    print_info "Starting installation..."
    echo ""

    # Check prerequisites
    check_git_repo

    HAS_PYTHON=false
    HAS_NODE=false
    HAS_GO=false

    check_python && HAS_PYTHON=true
    check_node && HAS_NODE=true
    check_go && HAS_GO=true

    echo ""

    # Parse method argument
    case "$METHOD" in
        --method=pre-commit|pre-commit)
            if [ "$HAS_PYTHON" = true ]; then
                install_precommit
            else
                print_error "Python is required for pre-commit framework"
                exit 1
            fi
            ;;
        --method=husky|husky)
            if [ "$HAS_NODE" = true ]; then
                install_husky
            else
                print_error "Node.js is required for Husky"
                exit 1
            fi
            ;;
        --method=both|both|*)
            INSTALLED=false

            if [ "$HAS_PYTHON" = true ]; then
                install_precommit
                INSTALLED=true
            fi

            if [ "$HAS_NODE" = true ]; then
                install_husky
                INSTALLED=true
            fi

            if [ "$INSTALLED" = false ]; then
                print_error "Neither Python nor Node.js is installed. Please install at least one."
                exit 1
            fi
            ;;
    esac

    install_custom_hooks

    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║           Installation completed successfully!         ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════╝${NC}"
    echo ""
    print_info "Next steps:"
    echo "  1. Test the hooks:  commit --allow-empty -m \"test: verify hooks\""
    echo "  2. Run all hooks manually: pre-commit run --all-files"
    echo "  3. Update hooks: pre-commit autoupdate"
    echo ""
    print_info "To bypass hooks (use sparingly):"
    echo "   commit --no-verify -m \"message\""
    echo ""
}

main
