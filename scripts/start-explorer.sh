#!/bin/bash
# Start PAW Block Explorer
#
# This script starts the Flask-based block explorer for PAW blockchain.
# The explorer connects to the local RPC endpoint at localhost:26657

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
FLASK_APP_DIR="$PROJECT_DIR/flask-app"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}=================================${NC}"
echo -e "${BLUE}PAW Block Explorer Startup${NC}"
echo -e "${BLUE}=================================${NC}"
echo ""

# Check if RPC is accessible
echo -e "${YELLOW}Checking RPC connection...${NC}"
if ! curl -s http://localhost:26657/status > /dev/null 2>&1; then
    echo -e "${YELLOW}Warning: Cannot connect to RPC at localhost:26657${NC}"
    echo -e "${YELLOW}Make sure the PAW node is running first${NC}"
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
else
    BLOCK_HEIGHT=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')
    echo -e "${GREEN}RPC connected! Current block height: $BLOCK_HEIGHT${NC}"
fi

# Navigate to flask-app directory
cd "$FLASK_APP_DIR"

# Check if running in Docker or standalone
if [ "$1" == "docker" ]; then
    echo -e "${BLUE}Starting explorer in Docker...${NC}"
    cd "$PROJECT_DIR/docker"
    docker-compose up -d explorer
    echo -e "${GREEN}Explorer started in Docker${NC}"
    echo -e "${GREEN}Access at: http://localhost:11080${NC}"
else
    echo -e "${BLUE}Starting explorer in standalone mode...${NC}"

    # Set environment variables
    export RPC_URL="${RPC_URL:-http://localhost:26657}"
    export CHAIN_ID="${CHAIN_ID:-paw-testnet-1}"
    export FLASK_ENV="${FLASK_ENV:-development}"

    # Check if requirements are installed
    if ! python3 -c "import flask" 2>/dev/null; then
        echo -e "${YELLOW}Installing Python dependencies...${NC}"
        pip3 install -r requirements.txt
    fi

    echo -e "${GREEN}Starting Flask server...${NC}"
    echo -e "${GREEN}RPC URL: $RPC_URL${NC}"
    echo -e "${GREEN}Chain ID: $CHAIN_ID${NC}"
    echo ""
    echo -e "${GREEN}Explorer will be available at:${NC}"
    echo -e "${GREEN}  http://localhost:11080${NC}"
    echo ""
    echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
    echo ""

    # Run Flask app
    python3 app.py --port 11080 2>&1 | sed "s/5000/11080/g"
fi
