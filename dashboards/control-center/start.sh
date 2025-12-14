#!/bin/bash
# PAW Control Center - Quick Start Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}===================================${NC}"
echo -e "${BLUE}  PAW Control Center Quick Start  ${NC}"
echo -e "${BLUE}===================================${NC}"
echo

# Check if .env exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}Warning: .env file not found${NC}"
    echo "Creating .env from template..."
    cp .env.example .env
    echo -e "${GREEN}Created .env file${NC}"
    echo -e "${YELLOW}IMPORTANT: Edit .env and set secure passwords before production use!${NC}"
    echo
fi

# Check Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}Error: docker-compose is not installed${NC}"
    exit 1
fi

# Parse arguments
MODE="${1:-minimal}"

case "$MODE" in
    minimal)
        echo -e "${BLUE}Starting minimal setup (Dashboard + Auth + Redis)${NC}"
        docker-compose up -d control-center auth-service redis
        ;;
    full)
        echo -e "${BLUE}Starting full stack (with blockchain node and explorer)${NC}"
        docker-compose --profile with-node --profile with-explorer up -d
        ;;
    stop)
        echo -e "${BLUE}Stopping all services${NC}"
        docker-compose down
        exit 0
        ;;
    status)
        echo -e "${BLUE}Service Status:${NC}"
        docker-compose ps
        exit 0
        ;;
    logs)
        SERVICE="${2:-control-center}"
        echo -e "${BLUE}Showing logs for: $SERVICE${NC}"
        docker-compose logs -f "$SERVICE"
        exit 0
        ;;
    *)
        echo -e "${RED}Unknown mode: $MODE${NC}"
        echo
        echo "Usage: $0 [mode]"
        echo
        echo "Modes:"
        echo "  minimal  - Start dashboard, auth service, and Redis (default)"
        echo "  full     - Start full stack with blockchain node and explorer"
        echo "  stop     - Stop all services"
        echo "  status   - Show service status"
        echo "  logs [service] - Show logs for a service"
        echo
        exit 1
        ;;
esac

# Wait for services to start
echo
echo -e "${BLUE}Waiting for services to be ready...${NC}"
sleep 5

# Check health
echo
echo -e "${BLUE}Checking service health:${NC}"

# Dashboard
if curl -s http://localhost:11200/health.html > /dev/null 2>&1; then
    echo -e "  Dashboard:    ${GREEN}✓ Healthy${NC}"
else
    echo -e "  Dashboard:    ${RED}✗ Not responding${NC}"
fi

# Auth service
if curl -s http://localhost:11201/health > /dev/null 2>&1; then
    echo -e "  Auth Service: ${GREEN}✓ Healthy${NC}"
else
    echo -e "  Auth Service: ${RED}✗ Not responding${NC}"
fi

echo
echo -e "${GREEN}===================================${NC}"
echo -e "${GREEN}  Services Started Successfully!  ${NC}"
echo -e "${GREEN}===================================${NC}"
echo
echo -e "${BLUE}Access Points:${NC}"
echo "  Dashboard:    http://localhost:11200"
echo "  Auth API:     http://localhost:11201"
echo "  Health Check: http://localhost:11200/health.html"
echo
echo -e "${BLUE}Default Login:${NC}"
echo "  Username: admin"
echo "  Password: admin123"
echo "  ${YELLOW}CHANGE PASSWORD IN PRODUCTION!${NC}"
echo
echo -e "${BLUE}Useful Commands:${NC}"
echo "  View logs:    ./start.sh logs [service-name]"
echo "  Check status: ./start.sh status"
echo "  Stop all:     ./start.sh stop"
echo
