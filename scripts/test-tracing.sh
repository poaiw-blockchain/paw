#!/bin/bash
#
# Test script for distributed tracing setup
# Verifies Jaeger deployment and trace collection
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=========================================="
echo "PAW Distributed Tracing Test"
echo "=========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Check Jaeger container
echo "Test 1: Checking Jaeger container status..."
if docker ps | grep -q paw-jaeger; then
    if docker ps | grep paw-jaeger | grep -q "Up"; then
        echo -e "${GREEN}✓ Jaeger container is running${NC}"
    else
        echo -e "${RED}✗ Jaeger container exists but is not running${NC}"
        docker ps -a | grep paw-jaeger
        exit 1
    fi
else
    echo -e "${RED}✗ Jaeger container not found${NC}"
    echo "Run: docker compose -f compose/docker-compose.tracing.yml up -d"
    exit 1
fi
echo ""

# Test 2: Check Jaeger health
echo "Test 2: Checking Jaeger health..."
if curl -s http://localhost:16686/ | grep -q "jaeger"; then
    echo -e "${GREEN}✓ Jaeger UI is accessible${NC}"
else
    echo -e "${RED}✗ Jaeger UI is not accessible${NC}"
    exit 1
fi
echo ""

# Test 3: Check OTLP endpoint
echo "Test 3: Checking OTLP HTTP endpoint..."
http_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:4318/v1/traces \
    -H "Content-Type: application/json" \
    -d '{"resourceSpans":[]}')

if [ "$http_code" == "200" ] || [ "$http_code" == "202" ]; then
    echo -e "${GREEN}✓ OTLP endpoint is accepting traces (HTTP $http_code)${NC}"
else
    echo -e "${RED}✗ OTLP endpoint returned HTTP $http_code${NC}"
    exit 1
fi
echo ""

# Test 4: Check config file
echo "Test 4: Checking telemetry configuration..."
if [ -f "$HOME/.paw/config/app.toml" ]; then
    if grep -q "telemetry" "$HOME/.paw/config/app.toml"; then
        echo -e "${GREEN}✓ app.toml has telemetry section${NC}"

        if grep -A 5 "\[telemetry\]" "$HOME/.paw/config/app.toml" | grep -q "enabled = true"; then
            echo -e "${GREEN}✓ Telemetry is enabled${NC}"
        else
            echo -e "${YELLOW}⚠ Telemetry exists but may not be enabled${NC}"
            echo "Check ~/.paw/config/app.toml"
        fi
    else
        echo -e "${YELLOW}⚠ app.toml exists but telemetry section not found${NC}"
        echo "Copy from config/app.toml.example"
    fi
else
    echo -e "${YELLOW}⚠ app.toml not found at ~/.paw/config/app.toml${NC}"
    echo "Node must be initialized first: pawd init <moniker>"
fi
echo ""

# Test 5: Check services in Jaeger
echo "Test 5: Checking for paw-blockchain service in Jaeger..."
services=$(curl -s http://localhost:16686/api/services | jq -r '.data[]' 2>/dev/null)

if [ -n "$services" ]; then
    if echo "$services" | grep -q "paw-blockchain"; then
        echo -e "${GREEN}✓ paw-blockchain service found in Jaeger${NC}"

        # Count traces
        trace_count=$(curl -s "http://localhost:16686/api/traces?service=paw-blockchain&limit=1000" | jq '.data | length' 2>/dev/null || echo "0")
        echo "  Found $trace_count recent traces"
    else
        echo -e "${YELLOW}⚠ paw-blockchain service not found (no traces yet)${NC}"
        echo "  Available services: $services"
        echo "  Start the node and generate some transactions to see traces"
    fi
else
    echo -e "${YELLOW}⚠ No services found in Jaeger (no traces collected yet)${NC}"
    echo "  This is expected if the node hasn't been started or no transactions processed"
fi
echo ""

# Test 6: Check sampling configuration
echo "Test 6: Checking sampling configuration..."
if [ -f "$PROJECT_DIR/compose/docker/tracing/sampling_strategies.json" ]; then
    echo -e "${GREEN}✓ Sampling strategies file exists${NC}"

    default_rate=$(jq -r '.default_strategy.param' "$PROJECT_DIR/compose/docker/tracing/sampling_strategies.json" 2>/dev/null || echo "unknown")
    echo "  Default sampling rate: $default_rate (${default_rate}00%)"
else
    echo -e "${RED}✗ Sampling strategies file not found${NC}"
    exit 1
fi
echo ""

# Summary
echo "=========================================="
echo "Summary"
echo "=========================================="
echo ""
echo "Jaeger UI: http://localhost:16686"
echo "OTLP HTTP Endpoint: http://localhost:4318"
echo "OTLP gRPC Endpoint: http://localhost:11317"
echo ""
echo "Next steps:"
echo "1. Ensure ~/.paw/config/app.toml has telemetry.enabled = true"
echo "2. Start the node: pawd start"
echo "3. Generate transactions (send, swap, etc.)"
echo "4. View traces at http://localhost:16686"
echo ""
echo -e "${GREEN}All tests passed!${NC}"
