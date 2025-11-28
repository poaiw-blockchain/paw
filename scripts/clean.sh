#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  PAW Blockchain Cleanup${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Function to confirm action
confirm() {
    read -p "$1 [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        return 0
    else
        return 1
    fi
}

# Clean build artifacts
echo -e "${YELLOW}Cleaning build artifacts...${NC}"
if [ -d "build" ]; then
    rm -rf build/*
    echo -e "${GREEN}✓ Removed build directory contents${NC}"
else
    echo -e "${YELLOW}! Build directory does not exist${NC}"
fi

# Clean binaries
if [ -d "$(go env GOPATH)/bin" ]; then
    if confirm "Remove installed binaries (pawd, pawcli)?"; then
        rm -f "$(go env GOPATH)/bin/pawd"
        rm -f "$(go env GOPATH)/bin/pawcli"
        echo -e "${GREEN}✓ Removed installed binaries${NC}"
    fi
fi

# Clean test cache
echo -e "\n${YELLOW}Cleaning test cache...${NC}"
go clean -testcache
echo -e "${GREEN}✓ Test cache cleared${NC}"

# Clean build cache
echo -e "${YELLOW}Cleaning build cache...${NC}"
go clean -cache
echo -e "${GREEN}✓ Build cache cleared${NC}"

# Clean module cache (optional)
if confirm "Clean Go module cache? (This will require re-downloading dependencies)"; then
    go clean -modcache
    echo -e "${GREEN}✓ Module cache cleared${NC}"
fi

# Clean coverage files
echo -e "\n${YELLOW}Cleaning coverage files...${NC}"
rm -f coverage.txt coverage.html *.coverprofile
echo -e "${GREEN}✓ Coverage files removed${NC}"

# Clean proto-generated files
if confirm "Clean proto-generated files? (Requires regeneration)"; then
    echo -e "${YELLOW}Cleaning proto-generated files...${NC}"
    find . -name "*.pb.go" -type f -delete
    find . -name "*.pb.gw.go" -type f -delete
    echo -e "${GREEN}✓ Proto-generated files removed${NC}"
fi

# Clean node data
if [ -d "$HOME/.paw" ]; then
    if confirm "Clean blockchain data (~/.paw)?"; then
        echo -e "${YELLOW}Cleaning blockchain data...${NC}"
        rm -rf "$HOME/.paw"
        echo -e "${GREEN}✓ Blockchain data removed${NC}"
    fi
fi

# Clean local testnet data
if [ -d "data" ]; then
    if confirm "Clean local testnet data (./data)?"; then
        echo -e "${YELLOW}Cleaning testnet data...${NC}"
        rm -rf data/*
        echo -e "${GREEN}✓ Testnet data removed${NC}"
    fi
fi

# Clean node_modules (if exists)
if [ -d "node_modules" ]; then
    if confirm "Clean node_modules?"; then
        echo -e "${YELLOW}Cleaning node_modules...${NC}"
        rm -rf node_modules
        echo -e "${GREEN}✓ node_modules removed${NC}"
    fi
fi

# Clean Python cache
echo -e "\n${YELLOW}Cleaning Python cache...${NC}"
find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true
find . -type f -name "*.pyc" -delete 2>/dev/null || true
find . -type f -name "*.pyo" -delete 2>/dev/null || true
echo -e "${GREEN}✓ Python cache cleaned${NC}"

# Clean log files
if [ -d "logs" ]; then
    if confirm "Clean log files?"; then
        echo -e "${YELLOW}Cleaning log files...${NC}"
        rm -rf logs/*
        echo -e "${GREEN}✓ Log files removed${NC}"
    fi
fi

# Clean Docker volumes (optional)
if command -v docker >/dev/null 2>&1; then
    if confirm "Clean Docker volumes?"; then
        echo -e "${YELLOW}Cleaning Docker volumes...${NC}"
        docker-compose -f docker-compose.dev.yml down -v 2>/dev/null || true
        echo -e "${GREEN}✓ Docker volumes removed${NC}"
    fi
fi

# Clean temporary files
echo -e "\n${YELLOW}Cleaning temporary files...${NC}"
find . -type f -name "*.tmp" -delete 2>/dev/null || true
find . -type f -name "*.log" -delete 2>/dev/null || true
find . -type f -name ".DS_Store" -delete 2>/dev/null || true
echo -e "${GREEN}✓ Temporary files removed${NC}"

# Clean pre-commit cache
if [ -d ".pre-commit-cache" ]; then
    rm -rf .pre-commit-cache
    echo -e "${GREEN}✓ Pre-commit cache removed${NC}"
fi

# Clean GoReleaser dist
if [ -d "dist" ]; then
    if confirm "Clean GoReleaser dist directory?"; then
        rm -rf dist
        echo -e "${GREEN}✓ GoReleaser dist removed${NC}"
    fi
fi

echo -e "\n${BLUE}========================================${NC}"
echo -e "${BLUE}  Cleanup Complete!${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Your workspace has been cleaned.${NC}"
echo ""
echo -e "${YELLOW}To restore dependencies and rebuild:${NC}"
echo -e "  ${GREEN}go mod download${NC}"
echo -e "  ${GREEN}make build${NC}"
echo ""
