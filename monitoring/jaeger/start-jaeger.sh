#!/bin/bash
# Quick start script for Jaeger distributed tracing

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Starting Jaeger distributed tracing for PAW blockchain..."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "Error: Docker is not running. Please start Docker first."
    exit 1
fi

# Check if Jaeger is already running
if docker ps | grep -q paw-jaeger; then
    echo "Jaeger is already running."
    echo "UI: http://localhost:16686"
    echo "OTLP endpoint: http://localhost:4318"
    exit 0
fi

# Start Jaeger
echo "Starting Jaeger container..."
docker-compose up -d

# Wait for Jaeger to be ready
echo "Waiting for Jaeger to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:16686/ > /dev/null 2>&1; then
        echo "Jaeger is ready!"
        break
    fi
    echo -n "."
    sleep 1
done

if ! curl -s http://localhost:16686/ > /dev/null 2>&1; then
    echo "Error: Jaeger failed to start within 30 seconds."
    echo "Check logs with: docker logs paw-jaeger"
    exit 1
fi

echo ""
echo "======================================"
echo "Jaeger is running successfully!"
echo "======================================"
echo ""
echo "Jaeger UI:        http://localhost:16686"
echo "OTLP HTTP:        http://localhost:4318"
echo "OTLP gRPC:        http://localhost:4317"
echo "Health Check:     http://localhost:14269"
echo ""
echo "To configure PAW node, add to app.toml:"
echo ""
echo "[telemetry]"
echo "enabled = true"
echo "jaeger-endpoint = \"http://localhost:4318\""
echo "sample-rate = 1.0"
echo ""
echo "To stop Jaeger:"
echo "  docker-compose down"
echo ""
echo "To view logs:"
echo "  docker logs -f paw-jaeger"
echo ""
