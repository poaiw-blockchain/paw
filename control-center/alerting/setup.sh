#!/bin/bash

set -e

echo "========================================"
echo "PAW Alert Manager - Setup Script"
echo "========================================"
echo

# Check prerequisites
check_prerequisites() {
    echo "Checking prerequisites..."

    if ! command -v docker &> /dev/null; then
        echo "ERROR: Docker is not installed"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null; then
        echo "ERROR: Docker Compose is not installed"
        exit 1
    fi

    echo "✓ Docker and Docker Compose found"
}

# Generate JWT secret
generate_jwt_secret() {
    echo
    echo "Generating JWT secret..."
    JWT_SECRET=$(openssl rand -base64 32)
    echo "✓ JWT secret generated"
}

# Create configuration
create_config() {
    echo
    echo "Creating configuration..."

    if [ ! -f config.yaml ]; then
        cp config.example.yaml config.yaml
        echo "✓ Created config.yaml from example"
    else
        echo "! config.yaml already exists, skipping"
    fi
}

# Create .env file
create_env_file() {
    echo
    echo "Creating .env file..."

    cat > .env <<EOF
# Database
DATABASE_URL=postgres://postgres:postgres@postgres:5432/paw_control_center?sslmode=disable
REDIS_URL=redis://redis:6379/0

# Security
JWT_SECRET=${JWT_SECRET}

# Integration URLs
PROMETHEUS_URL=http://prometheus:9090
EXPLORER_URL=http://localhost:11080
ADMIN_API_URL=http://localhost:11201

# Environment
ENVIRONMENT=development

# Email Configuration (optional - configure if needed)
# SMTP_HOST=smtp.gmail.com
# SMTP_PORT=587
# SMTP_USERNAME=alerts@example.com
# SMTP_PASSWORD=your-password
# SMTP_FROM_ADDRESS=alerts@example.com

# SMS Configuration (optional - configure if needed)
# TWILIO_ACCOUNT_SID=
# TWILIO_AUTH_TOKEN=
# TWILIO_FROM_NUMBER=

# Slack Configuration (optional - configure if needed)
# SLACK_WEBHOOK_URL=
# SLACK_BOT_TOKEN=

# Discord Configuration (optional - configure if needed)
# DISCORD_WEBHOOK_URL=
EOF

    echo "✓ Created .env file"
}

# Create Docker network
create_network() {
    echo
    echo "Creating Docker network..."

    if ! docker network ls | grep -q paw-network; then
        docker network create paw-network
        echo "✓ Created paw-network"
    else
        echo "! paw-network already exists, skipping"
    fi
}

# Start services
start_services() {
    echo
    echo "Starting services..."

    docker-compose up -d

    echo "✓ Services started"
    echo
    echo "Waiting for services to be ready..."
    sleep 10
}

# Check health
check_health() {
    echo
    echo "Checking service health..."

    MAX_RETRIES=30
    RETRY=0

    while [ $RETRY -lt $MAX_RETRIES ]; do
        if curl -f http://localhost:11210/health &> /dev/null; then
            echo "✓ Alert Manager is healthy"
            return 0
        fi

        echo "Waiting for Alert Manager to be ready... ($((RETRY+1))/$MAX_RETRIES)"
        sleep 2
        RETRY=$((RETRY+1))
    done

    echo "ERROR: Alert Manager did not become healthy"
    echo "Check logs with: docker-compose logs alert-manager"
    return 1
}

# Display summary
display_summary() {
    echo
    echo "========================================"
    echo "Setup Complete!"
    echo "========================================"
    echo
    echo "Alert Manager API: http://localhost:11210"
    echo "Health Check:      http://localhost:11210/health"
    echo
    echo "Next steps:"
    echo "1. View logs:      docker-compose logs -f alert-manager"
    echo "2. Create rules:   See examples/example-rules.json"
    echo "3. Create channels: See examples/example-channels.json"
    echo "4. Read docs:      cat README.md"
    echo
    echo "Useful commands:"
    echo "  docker-compose ps              # View running services"
    echo "  docker-compose logs            # View logs"
    echo "  docker-compose down            # Stop services"
    echo "  docker-compose restart         # Restart services"
    echo
    echo "JWT Secret (save this): ${JWT_SECRET}"
    echo
}

# Main execution
main() {
    check_prerequisites
    generate_jwt_secret
    create_config
    create_env_file
    create_network
    start_services

    if check_health; then
        display_summary
    else
        echo
        echo "Setup completed with errors. Please check the logs."
        exit 1
    fi
}

# Run main
main
