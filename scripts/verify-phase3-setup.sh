#!/usr/bin/env bash
# verify-phase3-setup.sh - Verify Phase 3 testing environment is ready
# Run this before executing Phase 3 tests to ensure all prerequisites are met

set -euo pipefail

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PASS_COUNT=0
FAIL_COUNT=0
WARN_COUNT=0

check_pass() {
    echo -e "${GREEN}✓${NC} $*"
    PASS_COUNT=$((PASS_COUNT + 1))
}

check_fail() {
    echo -e "${RED}✗${NC} $*"
    FAIL_COUNT=$((FAIL_COUNT + 1))
}

check_warn() {
    echo -e "${YELLOW}⚠${NC} $*"
    WARN_COUNT=$((WARN_COUNT + 1))
}

echo -e "${BLUE}==================================================${NC}"
echo -e "${BLUE}Phase 3 Testing Environment Verification${NC}"
echo -e "${BLUE}==================================================${NC}"
echo ""

# Check scripts exist
echo -e "${BLUE}Checking test scripts...${NC}"
for script in test-multinode.sh phase3.1-devnet-baseline.sh \
              phase3.2-consensus-liveness.sh phase3.3-network-conditions.sh \
              phase3.4-malicious-peer.sh; do
    if [[ -x "scripts/${script}" ]]; then
        check_pass "Script exists and is executable: ${script}"
    else
        check_fail "Script missing or not executable: ${script}"
    fi
done
echo ""

# Check required tools
echo -e "${BLUE}Checking required tools...${NC}"

if command -v docker &>/dev/null; then
    check_pass "docker installed ($(docker --version | cut -d' ' -f3 | tr -d ','))"
else
    check_fail "docker not found - REQUIRED"
fi

if docker compose version &>/dev/null; then
    check_pass "docker compose installed ($(docker compose version --short))"
else
    check_fail "docker compose not found - REQUIRED"
fi

if command -v jq &>/dev/null; then
    check_pass "jq installed ($(jq --version | cut -d'-' -f2))"
else
    check_fail "jq not found - REQUIRED"
fi

if command -v curl &>/dev/null; then
    check_pass "curl installed ($(curl --version | head -1 | cut -d' ' -f2))"
else
    check_fail "curl not found - REQUIRED"
fi

if command -v tc &>/dev/null; then
    check_pass "tc (traffic control) installed"
else
    check_warn "tc not found - REQUIRED for Phase 3.3 only"
fi

if command -v grpcurl &>/dev/null; then
    check_pass "grpcurl installed (optional)"
else
    check_warn "grpcurl not found - optional, some tests will be skipped"
fi
echo ""

# Check Docker is running
echo -e "${BLUE}Checking Docker service...${NC}"
if docker info &>/dev/null; then
    check_pass "Docker daemon is running"
else
    check_fail "Docker daemon is not running - start with: sudo systemctl start docker"
fi
echo ""

# Check port availability
echo -e "${BLUE}Checking port availability...${NC}"
REQUIRED_PORTS=(26657 26667 26677 26687 1317 1327 1337 1347 39090 39091 39092 39093)
for port in "${REQUIRED_PORTS[@]}"; do
    if ! sudo netstat -tlnp 2>/dev/null | grep -q ":${port} "; then
        check_pass "Port ${port} is available"
    else
        check_fail "Port ${port} is already in use"
    fi
done
echo ""

# Check Docker Compose file
echo -e "${BLUE}Checking Docker Compose configuration...${NC}"
if [[ -f "compose/docker-compose.devnet.yml" ]]; then
    check_pass "Docker Compose file exists"

    # Validate compose file
    if docker compose -f compose/docker-compose.devnet.yml config &>/dev/null; then
        check_pass "Docker Compose file is valid"
    else
        check_fail "Docker Compose file has syntax errors"
    fi
else
    check_fail "Docker Compose file not found: compose/docker-compose.devnet.yml"
fi
echo ""

# Check network simulation script
echo -e "${BLUE}Checking network simulation tools...${NC}"
NETWORK_SIM="${HOME}/blockchain-projects/scripts/network-sim.sh"
if [[ -x "$NETWORK_SIM" ]]; then
    check_pass "Network simulation script exists: ${NETWORK_SIM}"
else
    check_warn "Network simulation script not found - Phase 3.3 may fail"
fi
echo ""

# Check disk space
echo -e "${BLUE}Checking system resources...${NC}"
DISK_AVAIL=$(df -BG . | tail -1 | awk '{print $4}' | tr -d 'G')
if [[ $DISK_AVAIL -ge 10 ]]; then
    check_pass "Sufficient disk space available (${DISK_AVAIL}GB)"
else
    check_warn "Low disk space (${DISK_AVAIL}GB) - 10GB+ recommended"
fi

MEMORY_TOTAL=$(free -g | awk '/^Mem:/ {print $2}')
if [[ $MEMORY_TOTAL -ge 4 ]]; then
    check_pass "Sufficient memory available (${MEMORY_TOTAL}GB)"
else
    check_warn "Low memory (${MEMORY_TOTAL}GB) - 4GB+ recommended"
fi

CPU_CORES=$(nproc)
if [[ $CPU_CORES -ge 2 ]]; then
    check_pass "Sufficient CPU cores (${CPU_CORES})"
else
    check_warn "Low CPU count (${CPU_CORES}) - 4+ cores recommended"
fi
echo ""

# Check if pawd binary exists
echo -e "${BLUE}Checking pawd binary...${NC}"
if [[ -x "pawd" ]]; then
    check_pass "pawd binary exists and is executable"
elif command -v go &>/dev/null; then
    check_warn "pawd binary not found - will be built from source"
else
    check_fail "pawd binary not found and Go not installed"
fi
echo ""

# Check supporting files
echo -e "${BLUE}Checking supporting files...${NC}"
if [[ -f "scripts/devnet/lib.sh" ]]; then
    check_pass "Devnet library exists: scripts/devnet/lib.sh"
else
    check_warn "Devnet library not found - some features may not work"
fi

if [[ -f "scripts/devnet/init_node.sh" ]]; then
    check_pass "Node init script exists: scripts/devnet/init_node.sh"
else
    check_warn "Node init script not found - network startup may fail"
fi
echo ""

# Check report directory
echo -e "${BLUE}Checking report directory...${NC}"
if [[ -d "test-reports" ]]; then
    check_pass "Test reports directory exists"
else
    check_warn "Test reports directory will be created automatically"
fi
echo ""

# Summary
echo -e "${BLUE}==================================================${NC}"
echo -e "${BLUE}Verification Summary${NC}"
echo -e "${BLUE}==================================================${NC}"
echo -e "${GREEN}Passed:  ${PASS_COUNT}${NC}"
echo -e "${YELLOW}Warnings: ${WARN_COUNT}${NC}"
echo -e "${RED}Failed:   ${FAIL_COUNT}${NC}"
echo ""

if [[ $FAIL_COUNT -eq 0 ]]; then
    echo -e "${GREEN}✓ Environment is ready for Phase 3 testing${NC}"
    echo ""
    echo "Run tests with:"
    echo "  ./scripts/test-multinode.sh          # All tests"
    echo "  ./scripts/test-multinode.sh 3.1      # Phase 3.1 only"
    echo "  sudo ./scripts/test-multinode.sh 3.3 # Phase 3.3 (needs sudo)"
    echo ""
    exit 0
else
    echo -e "${RED}✗ Environment has critical issues that must be fixed${NC}"
    echo ""
    echo "Fix the failed checks above before running Phase 3 tests."
    echo ""
    exit 1
fi
