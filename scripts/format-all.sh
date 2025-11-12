#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  PAW Blockchain Code Formatter${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Count files formatted
GO_COUNT=0
JS_COUNT=0
PY_COUNT=0
PROTO_COUNT=0

# Format Go files
echo -e "${YELLOW}Formatting Go files...${NC}"
if command_exists gofmt; then
    # Find and format all Go files
    while IFS= read -r -d '' file; do
        gofmt -w -s "$file"
        ((GO_COUNT++))
    done < <(find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/docs/statik/statik.go" -print0)
    echo -e "${GREEN}✓ Formatted $GO_COUNT Go files with gofmt${NC}"
else
    echo -e "${RED}✗ gofmt not found${NC}"
fi

# Fix common misspellings
if command_exists misspell; then
    echo -e "${YELLOW}Fixing misspellings in Go files...${NC}"
    find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/docs/statik/statik.go" | xargs misspell -w
    echo -e "${GREEN}✓ Misspellings fixed${NC}"
else
    echo -e "${YELLOW}! misspell not found, skipping${NC}"
fi

# Fix imports
if command_exists goimports; then
    echo -e "${YELLOW}Fixing imports in Go files...${NC}"
    find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/docs/statik/statik.go" | xargs goimports -w -local github.com/paw-chain/paw
    echo -e "${GREEN}✓ Imports fixed${NC}"
else
    echo -e "${YELLOW}! goimports not found, skipping${NC}"
fi

# Format JavaScript/TypeScript files
if command_exists prettier; then
    echo -e "\n${YELLOW}Formatting JavaScript/TypeScript files...${NC}"
    if find . -name "*.js" -o -name "*.ts" -o -name "*.jsx" -o -name "*.tsx" | grep -q .; then
        prettier --write "**/*.{js,ts,jsx,tsx,json}" 2>/dev/null || true
        JS_COUNT=$(find . \( -name "*.js" -o -name "*.ts" -o -name "*.jsx" -o -name "*.tsx" \) -type f | wc -l)
        echo -e "${GREEN}✓ Formatted $JS_COUNT JavaScript/TypeScript files${NC}"
    else
        echo -e "${YELLOW}! No JavaScript/TypeScript files found${NC}"
    fi
else
    echo -e "${YELLOW}! prettier not found, skipping JavaScript/TypeScript formatting${NC}"
fi

# Format Python files
if command_exists black; then
    echo -e "\n${YELLOW}Formatting Python files...${NC}"
    if find . -name "*.py" | grep -q .; then
        black . --exclude="/(\.git|\.venv|venv|env|build|dist)/" 2>/dev/null || true
        PY_COUNT=$(find . -name "*.py" -type f -not -path "*/\.*" -not -path "*/venv/*" | wc -l)
        echo -e "${GREEN}✓ Formatted $PY_COUNT Python files${NC}"
    else
        echo -e "${YELLOW}! No Python files found${NC}"
    fi
elif command_exists autopep8; then
    echo -e "\n${YELLOW}Formatting Python files with autopep8...${NC}"
    if find . -name "*.py" | grep -q .; then
        find . -name "*.py" -type f -not -path "*/\.*" -not -path "*/venv/*" | xargs autopep8 --in-place --aggressive --aggressive
        PY_COUNT=$(find . -name "*.py" -type f -not -path "*/\.*" -not -path "*/venv/*" | wc -l)
        echo -e "${GREEN}✓ Formatted $PY_COUNT Python files${NC}"
    fi
else
    echo -e "${YELLOW}! black/autopep8 not found, skipping Python formatting${NC}"
fi

# Format Protobuf files
if command_exists clang-format; then
    echo -e "\n${YELLOW}Formatting Protobuf files...${NC}"
    if find . -name "*.proto" -not -path "./third_party/*" | grep -q .; then
        find . -name '*.proto' -not -path "./third_party/*" -exec clang-format -i {} \;
        PROTO_COUNT=$(find . -name "*.proto" -not -path "./third_party/*" | wc -l)
        echo -e "${GREEN}✓ Formatted $PROTO_COUNT Protobuf files${NC}"
    else
        echo -e "${YELLOW}! No Protobuf files found${NC}"
    fi
else
    echo -e "${YELLOW}! clang-format not found, skipping Protobuf formatting${NC}"
fi

# Format YAML files
if command_exists yamllint; then
    echo -e "\n${YELLOW}Linting YAML files...${NC}"
    if find . -name "*.yml" -o -name "*.yaml" | grep -q .; then
        yamllint . || true
        echo -e "${GREEN}✓ YAML files linted${NC}"
    fi
else
    echo -e "${YELLOW}! yamllint not found, skipping YAML linting${NC}"
fi

# Format Markdown files
if command_exists prettier; then
    echo -e "\n${YELLOW}Formatting Markdown files...${NC}"
    if find . -name "*.md" | grep -q .; then
        prettier --write "**/*.md" 2>/dev/null || true
        echo -e "${GREEN}✓ Markdown files formatted${NC}"
    fi
fi

# Format shell scripts
echo -e "\n${YELLOW}Formatting shell scripts...${NC}"
if command_exists shfmt; then
    if find . -name "*.sh" | grep -q .; then
        shfmt -w -i 2 -ci -bn scripts/*.sh 2>/dev/null || true
        echo -e "${GREEN}✓ Shell scripts formatted${NC}"
    fi
else
    echo -e "${YELLOW}! shfmt not found, skipping shell script formatting${NC}"
fi

# Summary
echo -e "\n${BLUE}========================================${NC}"
echo -e "${BLUE}  Formatting Complete!${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Summary:${NC}"
echo -e "  Go files: $GO_COUNT"
echo -e "  JavaScript/TypeScript files: $JS_COUNT"
echo -e "  Python files: $PY_COUNT"
echo -e "  Protobuf files: $PROTO_COUNT"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo -e "  1. Review changes: ${GREEN}git diff${NC}"
echo -e "  2. Run linter: ${GREEN}make lint${NC}"
echo -e "  3. Run tests: ${GREEN}make test${NC}"
echo -e "  4. Commit changes: ${GREEN}git add . && git commit${NC}"
echo ""
