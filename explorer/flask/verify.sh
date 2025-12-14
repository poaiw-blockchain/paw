#!/bin/bash
# PAW Flask Explorer - Verification Script
# Verifies all components are correctly installed and configured

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

print_check() {
    echo -e "${GREEN}✓${NC} $1"
}

print_fail() {
    echo -e "${RED}✗${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

CHECKS_PASSED=0
CHECKS_FAILED=0

check_file() {
    if [ -f "$1" ]; then
        print_check "File exists: $1"
        ((CHECKS_PASSED++))
        return 0
    else
        print_fail "File missing: $1"
        ((CHECKS_FAILED++))
        return 1
    fi
}

check_executable() {
    if [ -x "$1" ]; then
        print_check "File is executable: $1"
        ((CHECKS_PASSED++))
        return 0
    else
        print_fail "File not executable: $1"
        ((CHECKS_FAILED++))
        return 1
    fi
}

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  PAW Flask Explorer Verification      ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"

# Check core files
print_header "Core Application Files"
check_file "app.py"
check_file "requirements.txt"
check_file ".env.example"

# Check Docker files
print_header "Docker Configuration"
check_file "Dockerfile"
check_file "docker-compose.yml"
check_file ".dockerignore"

# Check nginx
print_header "Nginx Configuration"
check_file "nginx.conf"

# Check templates
print_header "Web Templates"
check_file "templates/base.html"
check_file "templates/index.html"
check_file "templates/blocks.html"
check_file "templates/search.html"
check_file "templates/404.html"
check_file "templates/500.html"

# Check documentation
print_header "Documentation"
check_file "README.md"
check_file "DEPLOYMENT.md"
check_file "IMPLEMENTATION_SUMMARY.md"

# Check automation
print_header "Automation Scripts"
check_file "Makefile"
check_file "deploy.sh"
check_executable "deploy.sh"

# Check Python syntax
print_header "Python Syntax Validation"
if command -v python3 &> /dev/null; then
    if python3 -m py_compile app.py 2>/dev/null; then
        print_check "Python syntax is valid"
        ((CHECKS_PASSED++))
    else
        print_fail "Python syntax errors in app.py"
        ((CHECKS_FAILED++))
    fi
else
    print_info "Python3 not found, skipping syntax check"
fi

# Check Docker syntax
print_header "Docker Configuration Validation"
if command -v docker-compose &> /dev/null; then
    if docker-compose config > /dev/null 2>&1; then
        print_check "Docker Compose syntax is valid"
        ((CHECKS_PASSED++))
    else
        print_fail "Docker Compose syntax errors"
        ((CHECKS_FAILED++))
    fi
else
    print_info "Docker Compose not found, skipping validation"
fi

# Check port allocation
print_header "Port Configuration"
if grep -q "11080" nginx.conf && grep -q "11080" docker-compose.yml; then
    print_check "Port 11080 configured correctly"
    ((CHECKS_PASSED++))
else
    print_fail "Port 11080 not configured"
    ((CHECKS_FAILED++))
fi

# Check network configuration
print_header "Network Configuration"
if grep -q "paw-network" docker-compose.yml; then
    print_check "Network 'paw-network' configured"
    ((CHECKS_PASSED++))
else
    print_fail "Network 'paw-network' not configured"
    ((CHECKS_FAILED++))
fi

# Check service configuration
print_header "Service Configuration"
if grep -q "INDEXER_API_URL" docker-compose.yml && grep -q "RPC_URL" docker-compose.yml; then
    print_check "Service endpoints configured"
    ((CHECKS_PASSED++))
else
    print_fail "Service endpoints not configured"
    ((CHECKS_FAILED++))
fi

# Check health checks
print_header "Health Check Configuration"
if grep -q "healthcheck" docker-compose.yml; then
    print_check "Health checks configured in docker-compose.yml"
    ((CHECKS_PASSED++))
else
    print_fail "Health checks not configured"
    ((CHECKS_FAILED++))
fi

# File size checks
print_header "File Size Verification"
APP_SIZE=$(wc -l < app.py)
if [ "$APP_SIZE" -gt 700 ]; then
    print_check "app.py has sufficient content ($APP_SIZE lines)"
    ((CHECKS_PASSED++))
else
    print_fail "app.py seems too small ($APP_SIZE lines)"
    ((CHECKS_FAILED++))
fi

NGINX_SIZE=$(wc -l < nginx.conf)
if [ "$NGINX_SIZE" -gt 300 ]; then
    print_check "nginx.conf has sufficient content ($NGINX_SIZE lines)"
    ((CHECKS_PASSED++))
else
    print_fail "nginx.conf seems too small ($NGINX_SIZE lines)"
    ((CHECKS_FAILED++))
fi

# Check directory structure
print_header "Directory Structure"
if [ -d "templates" ]; then
    print_check "templates/ directory exists"
    ((CHECKS_PASSED++))
else
    print_fail "templates/ directory missing"
    ((CHECKS_FAILED++))
fi

if [ -d "static" ]; then
    print_check "static/ directory exists"
    ((CHECKS_PASSED++))
else
    print_fail "static/ directory missing"
    ((CHECKS_FAILED++))
fi

# Summary
print_header "Verification Summary"
echo ""
echo -e "Checks passed: ${GREEN}${CHECKS_PASSED}${NC}"
echo -e "Checks failed: ${RED}${CHECKS_FAILED}${NC}"
echo ""

if [ $CHECKS_FAILED -eq 0 ]; then
    echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║  ✓ ALL CHECKS PASSED                  ║${NC}"
    echo -e "${GREEN}║  Flask Explorer is ready for deployment║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "Next steps:"
    echo -e "  1. Run ${BLUE}./deploy.sh${NC} to deploy"
    echo -e "  2. Or run ${BLUE}make up${NC} to start services"
    echo -e "  3. Access at ${BLUE}http://localhost:11080${NC}"
    echo ""
    exit 0
else
    echo -e "${RED}╔════════════════════════════════════════╗${NC}"
    echo -e "${RED}║  ✗ SOME CHECKS FAILED                  ║${NC}"
    echo -e "${RED}║  Please review the errors above        ║${NC}"
    echo -e "${RED}╚════════════════════════════════════════╝${NC}"
    echo ""
    exit 1
fi
