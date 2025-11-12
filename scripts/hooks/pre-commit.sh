#!/bin/bash
# PAW Blockchain - Custom Pre-Commit Hook
#
# This script performs comprehensive checks before allowing a commit.
# It runs formatting, linting, and quick tests on changed files only.

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
STAGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' | grep -v '\.pb\.go$' || true)
STAGED_JS_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.js$' || true)
STAGED_PY_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.py$' || true)

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  PAW Blockchain - Pre-Commit Checks${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

ERRORS=0

# Function to print status
print_check() {
    echo -e "\n${BLUE}▶${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
    ERRORS=$((ERRORS + 1))
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Check Go files
if [ -n "$STAGED_GO_FILES" ]; then
    print_check "Checking Go files..."

    # Check formatting
    print_check "Running gofmt..."
    for file in $STAGED_GO_FILES; do
        if [ -f "$file" ]; then
            gofmt -l "$file" | grep -q . && {
                print_error "File not formatted: $file"
                echo "  Run: gofmt -w $file"
            } || print_success "Format OK: $file"
        fi
    done

    # Check imports
    if command -v goimports &> /dev/null; then
        print_check "Running goimports..."
        for file in $STAGED_GO_FILES; do
            if [ -f "$file" ]; then
                goimports -l -local github.com/paw "$file" | grep -q . && {
                    print_error "Imports not organized: $file"
                    echo "  Run: goimports -w -local github.com/paw $file"
                } || print_success "Imports OK: $file"
            fi
        done
    fi

    # Run go vet
    print_check "Running go vet..."
    for file in $STAGED_GO_FILES; do
        if [ -f "$file" ]; then
            go vet "$file" 2>&1 | grep -q "^" && {
                print_error "go vet failed: $file"
            } || print_success "Vet OK: $file"
        fi
    done

    # Run golangci-lint on changed files
    if command -v golangci-lint &> /dev/null; then
        print_check "Running golangci-lint..."
        cd "$PROJECT_ROOT"
        golangci-lint run --new-from-rev=HEAD --config=.golangci.yml $STAGED_GO_FILES || {
            print_error "golangci-lint found issues"
        }
    else
        print_warning "golangci-lint not installed, skipping"
    fi

    # Run quick tests for packages with changed files
    print_check "Running quick tests..."
    PACKAGES=$(echo "$STAGED_GO_FILES" | xargs -I {} dirname {} | sort -u | xargs -I {} echo "./{}...")
    for pkg in $PACKAGES; do
        if go test -short -race "$pkg" &> /dev/null; then
            print_success "Tests passed: $pkg"
        else
            print_error "Tests failed: $pkg"
        fi
    done
fi

# Check JavaScript files
if [ -n "$STAGED_JS_FILES" ]; then
    print_check "Checking JavaScript files..."

    if command -v eslint &> /dev/null; then
        print_check "Running ESLint..."
        for file in $STAGED_JS_FILES; do
            if [ -f "$file" ]; then
                eslint "$file" --fix && {
                    print_success "ESLint OK: $file"
                    git add "$file"  # Re-stage if fixed
                } || print_error "ESLint failed: $file"
            fi
        done
    else
        print_warning "ESLint not installed, skipping"
    fi

    if command -v prettier &> /dev/null; then
        print_check "Running Prettier..."
        for file in $STAGED_JS_FILES; do
            if [ -f "$file" ]; then
                prettier --write "$file" && {
                    print_success "Prettier OK: $file"
                    git add "$file"  # Re-stage if formatted
                } || print_error "Prettier failed: $file"
            fi
        done
    fi
fi

# Check Python files
if [ -n "$STAGED_PY_FILES" ]; then
    print_check "Checking Python files..."

    if command -v black &> /dev/null; then
        print_check "Running Black..."
        for file in $STAGED_PY_FILES; do
            if [ -f "$file" ]; then
                black --line-length=100 --check "$file" || {
                    black --line-length=100 "$file"
                    git add "$file"  # Re-stage if formatted
                    print_success "Black formatted: $file"
                }
            fi
        done
    else
        print_warning "Black not installed, skipping"
    fi

    if command -v pylint &> /dev/null; then
        print_check "Running Pylint..."
        for file in $STAGED_PY_FILES; do
            if [ -f "$file" ]; then
                pylint --max-line-length=100 "$file" || print_warning "Pylint warnings: $file"
            fi
        done
    fi
fi

# Check for sensitive data
print_check "Checking for sensitive data..."
SENSITIVE_PATTERNS=(
    "-----BEGIN RSA PRIVATE KEY-----"
    "-----BEGIN PRIVATE KEY-----"
    "password\s*=\s*['\"][^'\"]{8,}"
    "api[_-]?key\s*=\s*['\"][^'\"]+"
    "secret\s*=\s*['\"][^'\"]+"
    "aws_access_key_id"
    "aws_secret_access_key"
)

for pattern in "${SENSITIVE_PATTERNS[@]}"; do
    if git diff --cached | grep -iE "$pattern" > /dev/null; then
        print_error "Potential sensitive data detected: $pattern"
        echo "  Please review your changes and remove any credentials"
    fi
done

# Check for debug statements
print_check "Checking for debug statements..."
DEBUG_PATTERNS=(
    "fmt\.Println"
    "console\.log"
    "debugger"
    "print\("
)

for pattern in "${DEBUG_PATTERNS[@]}"; do
    MATCHES=$(git diff --cached | grep -E "^\+.*$pattern" | grep -v "^+++\|^---" || true)
    if [ -n "$MATCHES" ]; then
        print_warning "Debug statement detected: $pattern"
        echo "$MATCHES"
    fi
done

# Summary
echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
if [ $ERRORS -eq 0 ]; then
    echo -e "${GREEN}✓ All pre-commit checks passed!${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    exit 0
else
    echo -e "${RED}✗ Pre-commit checks failed with $ERRORS error(s)${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "\n${YELLOW}Tip:${NC} Fix the errors and try again, or use ${YELLOW}--no-verify${NC} to skip (not recommended)"
    exit 1
fi
