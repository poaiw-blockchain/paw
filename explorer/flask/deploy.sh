#!/bin/bash
# PAW Flask Explorer - Quick Deployment Script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
EXPLORER_PORT=11080
REQUIRED_SERVICES=("paw-indexer" "paw-node")

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}PAW Flask Explorer - Deployment Script${NC}"
echo -e "${BLUE}========================================${NC}\n"

# Function to print colored messages
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running in correct directory
if [ ! -f "app.py" ] || [ ! -f "docker-compose.yml" ]; then
    print_error "Must run from explorer/flask directory"
    exit 1
fi

# Check Docker installation
print_info "Checking Docker installation..."
if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    print_error "Docker Compose is not installed"
    exit 1
fi

print_success "Docker and Docker Compose are installed"

# Check if port is available
print_info "Checking port availability..."
if netstat -tuln 2>/dev/null | grep -q ":${EXPLORER_PORT} "; then
    print_warning "Port ${EXPLORER_PORT} is already in use"
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
else
    print_success "Port ${EXPLORER_PORT} is available"
fi

# Check Docker network
print_info "Checking Docker network..."
if ! docker network inspect paw-network &> /dev/null; then
    print_warning "Docker network 'paw-network' does not exist"
    print_info "Creating network..."
    docker network create paw-network
    print_success "Network created"
else
    print_success "Network exists"
fi

# Check required services
print_info "Checking required services..."
for service in "${REQUIRED_SERVICES[@]}"; do
    if docker ps --format '{{.Names}}' | grep -q "^${service}$"; then
        print_success "${service} is running"
    else
        print_warning "${service} is not running"
    fi
done

# Check/create .env file
print_info "Checking environment configuration..."
if [ ! -f ".env" ]; then
    print_warning ".env file not found"
    print_info "Creating from template..."

    if [ -f ".env.example" ]; then
        cp .env.example .env

        # Generate random secret key
        if command -v openssl &> /dev/null; then
            SECRET_KEY=$(openssl rand -hex 32)
            sed -i "s/change-me-to-a-random-secret-key-in-production/${SECRET_KEY}/" .env
            print_success "Generated random FLASK_SECRET_KEY"
        else
            print_warning "openssl not found, using default secret key"
        fi

        print_success ".env file created"
    else
        print_error ".env.example not found"
        exit 1
    fi
else
    print_success ".env file exists"

    # Check if secret key is still default
    if grep -q "change-me-to-a-random-secret-key-in-production" .env; then
        print_warning "FLASK_SECRET_KEY is set to default value!"
        print_warning "This is NOT secure for production!"
    fi
fi

# Ask for deployment type
echo ""
print_info "Select deployment type:"
echo "1) Production (build and deploy)"
echo "2) Development (build with latest code)"
echo "3) Quick start (use existing images)"
read -p "Enter choice (1-3): " choice

case $choice in
    1)
        print_info "Production deployment selected"
        BUILD_ARGS="--no-cache"
        ;;
    2)
        print_info "Development deployment selected"
        BUILD_ARGS=""
        ;;
    3)
        print_info "Quick start selected (skipping build)"
        BUILD_ARGS="skip"
        ;;
    *)
        print_error "Invalid choice"
        exit 1
        ;;
esac

# Build images
if [ "$BUILD_ARGS" != "skip" ]; then
    print_info "Building Docker images..."
    if docker-compose build $BUILD_ARGS; then
        print_success "Images built successfully"
    else
        print_error "Build failed"
        exit 1
    fi
else
    print_info "Skipping build"
fi

# Stop existing containers
print_info "Stopping existing containers..."
docker-compose down 2>/dev/null || true

# Start services
print_info "Starting services..."
if docker-compose up -d; then
    print_success "Services started"
else
    print_error "Failed to start services"
    exit 1
fi

# Wait for services to be ready
print_info "Waiting for services to be ready..."
MAX_WAIT=60
WAITED=0

while [ $WAITED -lt $MAX_WAIT ]; do
    if curl -sf http://localhost:${EXPLORER_PORT}/health > /dev/null 2>&1; then
        print_success "Explorer is ready!"
        break
    fi

    echo -n "."
    sleep 2
    WAITED=$((WAITED + 2))
done

echo ""

if [ $WAITED -ge $MAX_WAIT ]; then
    print_error "Services did not become ready in time"
    print_info "Check logs with: docker-compose logs"
    exit 1
fi

# Verify deployment
print_info "Verifying deployment..."

# Health check
if curl -sf http://localhost:${EXPLORER_PORT}/health > /dev/null; then
    print_success "Health check passed"
else
    print_error "Health check failed"
fi

# Readiness check
if curl -sf http://localhost:${EXPLORER_PORT}/health/ready > /dev/null; then
    print_success "Readiness check passed"
else
    print_warning "Readiness check failed (backend services may not be ready)"
fi

# Show service status
echo ""
print_info "Service status:"
docker-compose ps

# Show URLs
echo ""
print_success "Deployment complete!"
echo ""
echo -e "${GREEN}Web UI:${NC}     http://localhost:${EXPLORER_PORT}"
echo -e "${GREEN}API:${NC}        http://localhost:${EXPLORER_PORT}/api/v1/stats"
echo -e "${GREEN}Health:${NC}     http://localhost:${EXPLORER_PORT}/health"
echo -e "${GREEN}Metrics:${NC}    http://localhost:${EXPLORER_PORT}/metrics"
echo ""
print_info "View logs with: docker-compose logs -f"
print_info "Stop services with: docker-compose down"
echo ""
