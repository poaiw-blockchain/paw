#!/bin/bash

# Frontend Build Verification Script
# This script verifies that the frontend build pipeline is working correctly

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
TESTS_PASSED=0
TESTS_FAILED=0

# Function to print colored output
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
    ((TESTS_PASSED++))
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
    ((TESTS_FAILED++))
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

print_header() {
    echo ""
    echo "================================================"
    echo "$1"
    echo "================================================"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
print_header "Checking Prerequisites"

if command_exists node; then
    NODE_VERSION=$(node --version)
    print_success "Node.js installed: $NODE_VERSION"
else
    print_error "Node.js not found. Please install Node.js >= 18.0.0"
    exit 1
fi

if command_exists npm; then
    NPM_VERSION=$(npm --version)
    print_success "npm installed: $NPM_VERSION"
else
    print_error "npm not found. Please install npm >= 9.0.0"
    exit 1
fi

# Verify Exchange Frontend
print_header "Verifying Exchange Frontend"

EXCHANGE_DIR="$PROJECT_ROOT/external/crypto/exchange-frontend"

if [ -d "$EXCHANGE_DIR" ]; then
    print_success "Exchange frontend directory exists"
else
    print_error "Exchange frontend directory not found"
    exit 1
fi

cd "$EXCHANGE_DIR"

# Check package.json
if [ -f "package.json" ]; then
    print_success "package.json exists"
else
    print_error "package.json not found"
fi

# Check configuration files
for file in vite.config.js .env.example .eslintrc.json .prettierrc.json; do
    if [ -f "$file" ]; then
        print_success "$file exists"
    else
        print_error "$file not found"
    fi
done

# Check security files
for file in config.js security.js api-client.js websocket-client.js; do
    if [ -f "$file" ]; then
        print_success "$file exists"
    else
        print_error "$file not found"
    fi
done

# Check HTML files
for file in index-secure.html; do
    if [ -f "$file" ]; then
        print_success "$file exists"
    else
        print_error "$file not found"
    fi
done

# Check for CSP headers in HTML
if grep -q "Content-Security-Policy" index-secure.html; then
    print_success "CSP headers found in index-secure.html"
else
    print_error "CSP headers not found in index-secure.html"
fi

# Install dependencies
print_info "Installing dependencies..."
if npm install --silent; then
    print_success "Dependencies installed successfully"
else
    print_error "Failed to install dependencies"
fi

# Run linting
print_info "Running ESLint..."
if npm run lint; then
    print_success "Linting passed"
else
    print_error "Linting failed"
fi

# Run build
print_info "Building for production..."
if npm run build; then
    print_success "Build completed successfully"
else
    print_error "Build failed"
    exit 1
fi

# Check build output
if [ -d "dist" ]; then
    print_success "Build output directory (dist/) created"

    # Check for key files
    if ls dist/*.html >/dev/null 2>&1; then
        print_success "HTML files present in dist/"
    else
        print_error "No HTML files in dist/"
    fi

    if ls dist/assets/*.js >/dev/null 2>&1; then
        print_success "JavaScript files present in dist/assets/"
    else
        print_error "No JavaScript files in dist/assets/"
    fi

    # Check for compressed files
    if ls dist/**/*.gz >/dev/null 2>&1; then
        print_success "Gzip compressed files present"
    else
        print_info "No gzip compressed files (optional)"
    fi

    # Check build size
    DIST_SIZE=$(du -sh dist | cut -f1)
    print_info "Build size: $DIST_SIZE"

else
    print_error "Build output directory not created"
fi

# Verify Browser Wallet Extension
print_header "Verifying Browser Wallet Extension"

WALLET_DIR="$PROJECT_ROOT/external/crypto/browser-wallet-extension"

if [ -d "$WALLET_DIR" ]; then
    print_success "Wallet extension directory exists"
else
    print_error "Wallet extension directory not found"
    exit 1
fi

cd "$WALLET_DIR"

# Check package.json
if [ -f "package.json" ]; then
    print_success "package.json exists"
else
    print_error "package.json not found"
fi

# Check configuration files
for file in build.js manifest.json .eslintrc.json .prettierrc.json; do
    if [ -f "$file" ]; then
        print_success "$file exists"
    else
        print_error "$file not found"
    fi
done

# Check extension files
for file in popup.html popup.js background.js styles.css; do
    if [ -f "$file" ]; then
        print_success "$file exists"
    else
        print_error "$file not found"
    fi
done

# Install dependencies
print_info "Installing dependencies..."
if npm install --silent; then
    print_success "Dependencies installed successfully"
else
    print_error "Failed to install dependencies"
fi

# Run linting
print_info "Running ESLint..."
if npm run lint; then
    print_success "Linting passed"
else
    print_error "Linting failed"
fi

# Run build
print_info "Building extension..."
if npm run build; then
    print_success "Build completed successfully"
else
    print_error "Build failed"
    exit 1
fi

# Check build output
if [ -d "dist" ]; then
    print_success "Build output directory (dist/) created"

    # Check for key files
    for file in popup.html popup.js background.js manifest.json; do
        if [ -f "dist/$file" ]; then
            print_success "$file present in dist/"
        else
            print_error "$file not found in dist/"
        fi
    done

    # Check if manifest is valid JSON
    if jq empty dist/manifest.json 2>/dev/null; then
        print_success "manifest.json is valid JSON"
    else
        print_info "Could not validate manifest.json (jq not installed)"
    fi

else
    print_error "Build output directory not created"
fi

# Verify CI/CD Pipeline
print_header "Verifying CI/CD Pipeline"

cd "$PROJECT_ROOT"

if [ -f ".github/workflows/frontend-ci.yml" ]; then
    print_success "Frontend CI/CD workflow exists"

    # Check for key jobs
    if grep -q "lint-and-format:" .github/workflows/frontend-ci.yml; then
        print_success "Lint and format job configured"
    fi

    if grep -q "security-scan:" .github/workflows/frontend-ci.yml; then
        print_success "Security scan job configured"
    fi

    if grep -q "build-exchange-frontend:" .github/workflows/frontend-ci.yml; then
        print_success "Exchange frontend build job configured"
    fi

    if grep -q "build-wallet-extension:" .github/workflows/frontend-ci.yml; then
        print_success "Wallet extension build job configured"
    fi
else
    print_error "Frontend CI/CD workflow not found"
fi

# Verify Documentation
print_header "Verifying Documentation"

for file in FRONTEND_BUILD_SECURITY_SUMMARY.md FRONTEND_QUICK_START.md; do
    if [ -f "$file" ]; then
        print_success "$file exists"
    else
        print_error "$file not found"
    fi
done

# Final Summary
print_header "Verification Summary"

echo ""
echo "Tests Passed: $TESTS_PASSED"
echo "Tests Failed: $TESTS_FAILED"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    print_success "All verifications passed! Frontend build pipeline is ready."
    exit 0
else
    print_error "Some verifications failed. Please review the errors above."
    exit 1
fi
