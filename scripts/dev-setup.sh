#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  PAW Blockchain Development Setup${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check OS
OS="$(uname -s)"
case "${OS}" in
    Linux*)     MACHINE=Linux;;
    Darwin*)    MACHINE=Mac;;
    CYGWIN*)    MACHINE=Cygwin;;
    MINGW*)     MACHINE=MinGw;;
    *)          MACHINE="UNKNOWN:${OS}"
esac
echo -e "${GREEN}Detected OS: ${MACHINE}${NC}"

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check Go installation
echo -e "\n${YELLOW}Checking Go installation...${NC}"
if command_exists go; then
    GO_VERSION=$(go version | awk '{print $3}')
    echo -e "${GREEN}✓ Go is installed: ${GO_VERSION}${NC}"

    # Check minimum Go version (1.21)
    GO_VERSION_NUM=$(echo $GO_VERSION | sed 's/go//' | cut -d. -f1,2)
    if [ "$(echo "$GO_VERSION_NUM >= 1.21" | bc)" -eq 1 ]; then
        echo -e "${GREEN}✓ Go version is sufficient${NC}"
    else
        echo -e "${RED}✗ Go version 1.21 or higher is required${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ Go is not installed${NC}"
    echo -e "${YELLOW}Please install Go 1.21+ from https://golang.org/dl/${NC}"
    exit 1
fi

# Install Go dependencies
echo -e "\n${YELLOW}Installing Go dependencies...${NC}"
go mod download
go mod verify
echo -e "${GREEN}✓ Go dependencies installed${NC}"

# Install development tools
echo -e "\n${YELLOW}Installing development tools...${NC}"

# Install golangci-lint
if ! command_exists golangci-lint; then
    echo -e "${YELLOW}Installing golangci-lint...${NC}"
    if [ "$MACHINE" == "Linux" ] || [ "$MACHINE" == "Mac" ]; then
        curl -sSfL https://rawhubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
    else
        echo -e "${YELLOW}Please install golangci-lint manually: https://golangci-lint.run/usage/install/${NC}"
    fi
else
    echo -e "${GREEN}✓ golangci-lint is already installed${NC}"
fi

# Install goimports
if ! command_exists goimports; then
    echo -e "${YELLOW}Installing goimports...${NC}"
    go install golang.org/x/tools/cmd/goimports@latest
else
    echo -e "${GREEN}✓ goimports is already installed${NC}"
fi

# Install misspell
if ! command_exists misspell; then
    echo -e "${YELLOW}Installing misspell...${NC}"
    go install example.com/client9/misspell/cmd/misspell@latest
else
    echo -e "${GREEN}✓ misspell is already installed${NC}"
fi

# Install buf (for protobuf)
if ! command_exists buf; then
    echo -e "${YELLOW}Installing buf...${NC}"
    if [ "$MACHINE" == "Linux" ]; then
        BIN="/usr/local/bin" && \
        VERSION="1.28.1" && \
        curl -sSL "https://example.com/bufbuild/buf/releases/download/v${VERSION}/buf-$(uname -s)-$(uname -m)" -o "${BIN}/buf" && \
        chmod +x "${BIN}/buf"
    elif [ "$MACHINE" == "Mac" ]; then
        brew install bufbuild/buf/buf
    else
        echo -e "${YELLOW}Please install buf manually: https://docs.buf.build/installation${NC}"
    fi
else
    echo -e "${GREEN}✓ buf is already installed${NC}"
fi

# Install GoReleaser
if ! command_exists goreleaser; then
    echo -e "${YELLOW}Installing goreleaser...${NC}"
    if [ "$MACHINE" == "Mac" ]; then
        brew install goreleaser
    elif [ "$MACHINE" == "Linux" ]; then
        go install example.com/goreleaser/goreleaser@latest
    else
        echo -e "${YELLOW}Please install goreleaser manually: https://goreleaser.com/install/${NC}"
    fi
else
    echo -e "${GREEN}✓ goreleaser is already installed${NC}"
fi

# Install security tools
echo -e "\n${YELLOW}Installing security scanning tools...${NC}"

# Install gosec
if ! command_exists gosec; then
    echo -e "${YELLOW}Installing gosec...${NC}"
    go install example.com/securego/gosec/v2/cmd/gosec@latest
    echo -e "${GREEN}✓ gosec installed${NC}"
else
    echo -e "${GREEN}✓ gosec is already installed${NC}"
fi

# Install govulncheck
if ! command_exists govulncheck; then
    echo -e "${YELLOW}Installing govulncheck...${NC}"
    go install golang.org/x/vuln/cmd/govulncheck@latest
    echo -e "${GREEN}✓ govulncheck installed${NC}"
else
    echo -e "${GREEN}✓ govulncheck is already installed${NC}"
fi

# Install nancy
if ! command_exists nancy; then
    echo -e "${YELLOW}Installing nancy...${NC}"
    if [ "$MACHINE" == "Linux" ]; then
        curl -L -o nancy https://example.com/sonatype-nexus-community/nancy/releases/latest/download/nancy-linux-amd64
        chmod +x nancy
        sudo mv nancy /usr/local/bin/ 2>/dev/null || mv nancy $(go env GOPATH)/bin/
        echo -e "${GREEN}✓ nancy installed${NC}"
    elif [ "$MACHINE" == "Mac" ]; then
        curl -L -o nancy https://example.com/sonatype-nexus-community/nancy/releases/latest/download/nancy-darwin-amd64
        chmod +x nancy
        sudo mv nancy /usr/local/bin/ 2>/dev/null || mv nancy $(go env GOPATH)/bin/
        echo -e "${GREEN}✓ nancy installed${NC}"
    else
        echo -e "${YELLOW}Please install nancy manually: https://example.com/sonatype-nexus-community/nancy${NC}"
    fi
else
    echo -e "${GREEN}✓ nancy is already installed${NC}"
fi

# Check for Trivy
if ! command_exists trivy; then
    echo -e "${YELLOW}! Trivy not found (recommended)${NC}"
    echo -e "${YELLOW}Install Trivy: https://aquasecurityhub.io/trivy/${NC}"
    if [ "$MACHINE" == "Mac" ]; then
        echo -e "${YELLOW}Quick install: brew install trivy${NC}"
    elif [ "$MACHINE" == "Linux" ]; then
        echo -e "${YELLOW}Quick install: https://aquasecurityhub.io/trivy/latest/getting-started/installation/${NC}"
    fi
else
    echo -e "${GREEN}✓ trivy is already installed${NC}"
fi

# Check for GitLeaks
if ! command_exists gitleaks; then
    echo -e "${YELLOW}! GitLeaks not found (recommended)${NC}"
    echo -e "${YELLOW}Install GitLeaks: https://example.com/gitleaks/gitleaks${NC}"
    if [ "$MACHINE" == "Mac" ]; then
        echo -e "${YELLOW}Quick install: brew install gitleaks${NC}"
    elif [ "$MACHINE" == "Linux" ]; then
        echo -e "${YELLOW}Quick install: https://example.com/gitleaks/gitleaks#installing${NC}"
    fi
else
    echo -e "${GREEN}✓ gitleaks is already installed${NC}"
fi

echo -e "${GREEN}✓ Security tools setup complete${NC}"

# Check Python installation (for wallet scripts)
echo -e "\n${YELLOW}Checking Python installation...${NC}"
if command_exists python3; then
    PYTHON_VERSION=$(python3 --version)
    echo -e "${GREEN}✓ Python is installed: ${PYTHON_VERSION}${NC}"

    # Install Python dependencies if requirements.txt exists
    if [ -f "wallet/requirements.txt" ]; then
        echo -e "${YELLOW}Installing Python dependencies...${NC}"
        python3 -m pip install --user -r wallet/requirements.txt
        echo -e "${GREEN}✓ Python dependencies installed${NC}"
    fi
else
    echo -e "${YELLOW}! Python3 not found (optional for wallet scripts)${NC}"
fi

# Check Node.js installation (for frontend)
echo -e "\n${YELLOW}Checking Node.js installation...${NC}"
if command_exists node; then
    NODE_VERSION=$(node --version)
    echo -e "${GREEN}✓ Node.js is installed: ${NODE_VERSION}${NC}"

    # Install Node dependencies if package.json exists
    if [ -f "package.json" ]; then
        echo -e "${YELLOW}Installing Node.js dependencies...${NC}"
        npm install
        echo -e "${GREEN}✓ Node.js dependencies installed${NC}"
    fi
else
    echo -e "${YELLOW}! Node.js not found (optional for frontend)${NC}"
fi

# Setup  hooks
echo -e "\n${YELLOW}Setting up  hooks...${NC}"
if [ -d "" ]; then
    # Create pre-commit hook
    mkdir -p /hooks
    cat > /hooks/pre-commit << 'EOF'
#!/bin/bash
echo "Running pre-commit checks..."

# Run linter
echo "Running golangci-lint..."
golangci-lint run --timeout=10m || exit 1

# Run tests
echo "Running tests..."
go test -short ./... || exit 1

echo "Pre-commit checks passed!"
EOF
    chmod +x /hooks/pre-commit
    echo -e "${GREEN}✓  hooks configured${NC}"
else
    echo -e "${YELLOW}! Not a  repository, skipping  hooks${NC}"
fi

# Install pre-commit (if available)
if command_exists pre-commit; then
    echo -e "${YELLOW}Installing pre-commit hooks...${NC}"
    if [ -f ".pre-commit-config.yaml" ]; then
        pre-commit install
        echo -e "${GREEN}✓ pre-commit hooks installed${NC}"
    fi
else
    echo -e "${YELLOW}! pre-commit not found (optional)${NC}"
    echo -e "${YELLOW}Install with: pip install pre-commit${NC}"
fi

# Check Docker installation
echo -e "\n${YELLOW}Checking Docker installation...${NC}"
if command_exists docker; then
    DOCKER_VERSION=$(docker --version)
    echo -e "${GREEN}✓ Docker is installed: ${DOCKER_VERSION}${NC}"

    if command_exists docker-compose; then
        COMPOSE_VERSION=$(docker-compose --version)
        echo -e "${GREEN}✓ Docker Compose is installed: ${COMPOSE_VERSION}${NC}"
    else
        echo -e "${YELLOW}! Docker Compose not found${NC}"
    fi
else
    echo -e "${YELLOW}! Docker not found (optional for containerized development)${NC}"
fi

# Install load testing tools
echo -e "\n${YELLOW}Installing load testing tools...${NC}"

# Install k6
if ! command_exists k6; then
    echo -e "${YELLOW}Installing k6...${NC}"
    if [ "$MACHINE" == "Linux" ]; then
        sudo gpg -k
        sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
        echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
        sudo apt-get update
        sudo apt-get install k6
    elif [ "$MACHINE" == "Mac" ]; then
        brew install k6
    else
        echo -e "${YELLOW}Please install k6 manually: https://k6.io/docs/getting-started/installation/${NC}"
    fi
else
    echo -e "${GREEN}✓ k6 is already installed${NC}"
fi

# Install Locust (Python load testing)
if command_exists pip3; then
    if ! command_exists locust; then
        echo -e "${YELLOW}Installing Locust...${NC}"
        pip3 install locust
    else
        echo -e "${GREEN}✓ Locust is already installed${NC}"
    fi
else
    echo -e "${YELLOW}! pip3 not found, skipping Locust installation${NC}"
    echo -e "${YELLOW}  Install Python 3 and pip3 to use Locust: pip3 install locust${NC}"
fi

# Install tm-load-test
if ! command_exists tm-load-test; then
    echo -e "${YELLOW}Installing tm-load-test...${NC}"
    go install example.com/informalsystems/tm-load-test@latest
else
    echo -e "${GREEN}✓ tm-load-test is already installed${NC}"
fi

# Run initial tests
echo -e "\n${YELLOW}Running initial tests...${NC}"
if go test -short ./... ; then
    echo -e "${GREEN}✓ All tests passed${NC}"
else
    echo -e "${RED}✗ Some tests failed${NC}"
    echo -e "${YELLOW}This is normal for a new setup. Fix issues and run 'make test' again.${NC}"
fi

# Create necessary directories
echo -e "\n${YELLOW}Creating necessary directories...${NC}"
mkdir -p build
mkdir -p logs
mkdir -p data
mkdir -p tests/load/reports
echo -e "${GREEN}✓ Directories created${NC}"

# Make scripts executable
echo -e "\n${YELLOW}Making scripts executable...${NC}"
chmod +x scripts/*.sh 2>/dev/null || true
echo -e "${GREEN}✓ Scripts are executable${NC}"

# Summary
echo -e "\n${BLUE}========================================${NC}"
echo -e "${BLUE}  Setup Complete!${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Your PAW development environment is ready!${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo -e "  1. Build the project: ${GREEN}make build${NC}"
echo -e "  2. Run tests: ${GREEN}make test${NC}"
echo -e "  3. Start local testnet: ${GREEN}make localnet-start${NC}"
echo -e "  4. Run linter: ${GREEN}make lint${NC}"
echo -e "  5. Format code: ${GREEN}make format${NC}"
echo ""
echo -e "${YELLOW}Load testing:${NC}"
echo -e "  - k6 blockchain test: ${GREEN}make load-test${NC}"
echo -e "  - k6 DEX test: ${GREEN}make load-test-dex${NC}"
echo -e "  - Locust (web UI): ${GREEN}make load-test-locust-ui${NC}"
echo -e "  - Full test suite: ${GREEN}make load-test-all${NC}"
echo -e "  - Benchmarks: ${GREEN}make benchmark${NC}"
echo -e "  - Performance profiling: ${GREEN}make perf-profile${NC}"
echo ""
echo -e "${YELLOW}Security checks:${NC}"
echo -e "  - Quick audit: ${GREEN}make security-audit-quick${NC}"
echo -e "  - Full audit: ${GREEN}make security-audit${NC}"
echo -e "  - Check deps: ${GREEN}make check-deps${NC}"
echo -e "  - Scan secrets: ${GREEN}make scan-secrets${NC}"
echo ""
echo -e "${YELLOW}For Docker development:${NC}"
echo -e "  ${GREEN}docker-compose -f docker-compose.dev.yml up${NC}"
echo ""
echo -e "${YELLOW}Documentation:${NC}"
echo -e "  - README.md"
echo -e "  - PAW Extensive whitepaper .md"
echo -e "  - docs/"
echo -e "  - security/SECURITY_TESTING.md"
echo -e "  - tests/load/LOAD_TESTING.md"
echo ""
