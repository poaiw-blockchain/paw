#!/usr/bin/env bash
# PAW Blockchain - GCP 3-Node Deployment Script
# Deploys and configures PAW blockchain on GCP test nodes

set -euo pipefail

# Configuration
PROJECT_ID="aixn-node-1"
ZONE="us-central1-a"
CHAIN_ID="paw-testnet-gcp"
KEYRING_BACKEND="test"

NODES=(
    "xai-testnode-1:34.29.163.145"
    "xai-testnode-2:108.59.86.86"
    "xai-testnode-3:35.184.167.38"
)

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${BLUE}[STEP]${NC} $1"; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TMP_DIR="/tmp/paw-deploy-$$"

cleanup() {
    log_info "Cleaning up temporary files..."
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

mkdir -p "$TMP_DIR"

# Build PAW binary locally
build_binary() {
    log_step "Building PAW binary locally..."
    cd "$PROJECT_ROOT"

    if ! go version &>/dev/null; then
        log_error "Go not installed locally. Install Go 1.23+"
        exit 1
    fi

    log_info "Running go build..."
    CGO_ENABLED=1 go build -o "$TMP_DIR/pawd" ./cmd/pawd/main.go

    if [ ! -f "$TMP_DIR/pawd" ]; then
        log_error "Build failed - pawd binary not created"
        exit 1
    fi

    log_info "Binary built successfully: $(du -h $TMP_DIR/pawd | cut -f1)"
}

# Install dependencies on a node
install_dependencies() {
    local node_name=$1
    local node_ip=$2

    log_step "Installing dependencies on $node_name..."

    gcloud compute ssh "$node_name" --zone="$ZONE" --project="$PROJECT_ID" --command="
        set -euo pipefail
        echo '[INFO] Updating system packages...'
        sudo apt-get update -qq

        echo '[INFO] Installing required packages...'
        sudo apt-get install -y -qq curl wget jq bc net-tools

        echo '[INFO] Creating directories...'
        sudo mkdir -p /opt/paw/bin
        sudo mkdir -p /root/.paw

        echo '[INFO] Dependencies installed successfully'
    "
}

# Copy binary to node
copy_binary() {
    local node_name=$1

    log_step "Copying PAW binary to $node_name..."
    gcloud compute scp "$TMP_DIR/pawd" "$node_name:/tmp/pawd" --zone="$ZONE" --project="$PROJECT_ID"

    gcloud compute ssh "$node_name" --zone="$ZONE" --project="$PROJECT_ID" --command="
        sudo mv /tmp/pawd /opt/paw/bin/pawd
        sudo chmod +x /opt/paw/bin/pawd
        sudo ln -sf /opt/paw/bin/pawd /usr/local/bin/pawd
        pawd version
    "
}

# Initialize node1 (genesis node)
init_node1() {
    local node_name="xai-testnode-1"

    log_step "Initializing $node_name as genesis node..."

    gcloud compute ssh "$node_name" --zone="$ZONE" --project="$PROJECT_ID" --command="
        set -euo pipefail
        export HOME_DIR=/root/.paw/node1

        echo '[INFO] Initializing chain...'
        pawd init node1 --chain-id $CHAIN_ID --home \$HOME_DIR

        echo '[INFO] Creating accounts...'
        pawd keys add validator --keyring-backend $KEYRING_BACKEND --home \$HOME_DIR --output json > /tmp/validator_key.json
        pawd keys add smoke-trader --keyring-backend $KEYRING_BACKEND --home \$HOME_DIR --output json > /tmp/trader_key.json
        pawd keys add smoke-counterparty --keyring-backend $KEYRING_BACKEND --home \$HOME_DIR --output json > /tmp/counterparty_key.json

        echo '[INFO] Adding genesis accounts...'
        pawd genesis add-genesis-account validator 200000000000upaw --keyring-backend $KEYRING_BACKEND --home \$HOME_DIR
        pawd genesis add-genesis-account smoke-trader 150000000000upaw,150000000000ufoo,150000000000ubar --keyring-backend $KEYRING_BACKEND --home \$HOME_DIR
        pawd genesis add-genesis-account smoke-counterparty 50000000000upaw --keyring-backend $KEYRING_BACKEND --home \$HOME_DIR

        echo '[INFO] Creating genesis transaction...'
        pawd genesis gentx validator 100000000000upaw \
            --chain-id $CHAIN_ID \
            --moniker node1 \
            --commission-rate 0.10 \
            --commission-max-rate 0.20 \
            --commission-max-change-rate 0.01 \
            --min-self-delegation 1 \
            --keyring-backend $KEYRING_BACKEND \
            --home \$HOME_DIR

        echo '[INFO] Collecting genesis transactions...'
        pawd genesis collect-gentxs --home \$HOME_DIR

        echo '[INFO] Validating genesis...'
        pawd genesis validate-genesis --home \$HOME_DIR

        echo '[INFO] Getting node ID...'
        pawd tendermint show-node-id --home \$HOME_DIR > /tmp/node1.id

        echo '[INFO] Configuring node...'
        sed -i 's/^minimum-gas-prices = \"\"/minimum-gas-prices = \"0.025upaw\"/' \$HOME_DIR/config/app.toml
        sed -i 's/^enable = false/enable = true/' \$HOME_DIR/config/app.toml
        sed -i 's|address = \"tcp://localhost:1317\"|address = \"tcp://0.0.0.0:1317\"|' \$HOME_DIR/config/app.toml
        sed -i 's|laddr = \"tcp://127.0.0.1:26657\"|laddr = \"tcp://0.0.0.0:26657\"|' \$HOME_DIR/config/config.toml
        sed -i 's/addr_book_strict = true/addr_book_strict = false/' \$HOME_DIR/config/config.toml

        echo '[INFO] Node1 initialization complete'
    "

    # Download genesis and node ID
    log_info "Downloading genesis and node ID from node1..."
    gcloud compute scp "$node_name:/root/.paw/node1/config/genesis.json" "$TMP_DIR/genesis.json" --zone="$ZONE" --project="$PROJECT_ID"
    gcloud compute scp "$node_name:/tmp/node1.id" "$TMP_DIR/node1.id" --zone="$ZONE" --project="$PROJECT_ID"
    gcloud compute scp "$node_name:/tmp/validator_key.json" "$TMP_DIR/validator_key.json" --zone="$ZONE" --project="$PROJECT_ID"
    gcloud compute scp "$node_name:/tmp/trader_key.json" "$TMP_DIR/trader_key.json" --zone="$ZONE" --project="$PROJECT_ID"
    gcloud compute scp "$node_name:/tmp/counterparty_key.json" "$TMP_DIR/counterparty_key.json" --zone="$ZONE" --project="$PROJECT_ID"

    log_info "Node1 ID: $(cat $TMP_DIR/node1.id)"
}

# Initialize node2 or node3
init_follower_node() {
    local node_name=$1
    local node_num=$2
    local node1_ip="34.29.163.145"
    local node1_id=$(cat "$TMP_DIR/node1.id")

    log_step "Initializing $node_name as follower node..."

    # Upload genesis
    gcloud compute scp "$TMP_DIR/genesis.json" "$node_name:/tmp/genesis.json" --zone="$ZONE" --project="$PROJECT_ID"

    gcloud compute ssh "$node_name" --zone="$ZONE" --project="$PROJECT_ID" --command="
        set -euo pipefail
        export HOME_DIR=/root/.paw/node${node_num}

        echo '[INFO] Initializing chain...'
        pawd init node${node_num} --chain-id $CHAIN_ID --home \$HOME_DIR

        echo '[INFO] Copying genesis...'
        cp /tmp/genesis.json \$HOME_DIR/config/genesis.json

        echo '[INFO] Configuring node...'
        sed -i 's/^minimum-gas-prices = \"\"/minimum-gas-prices = \"0.025upaw\"/' \$HOME_DIR/config/app.toml
        sed -i 's/^enable = false/enable = true/' \$HOME_DIR/config/app.toml
        sed -i 's|address = \"tcp://localhost:1317\"|address = \"tcp://0.0.0.0:1317\"|' \$HOME_DIR/config/app.toml
        sed -i 's|laddr = \"tcp://127.0.0.1:26657\"|laddr = \"tcp://0.0.0.0:26657\"|' \$HOME_DIR/config/config.toml
        sed -i 's/addr_book_strict = true/addr_book_strict = false/' \$HOME_DIR/config/config.toml

        echo '[INFO] Setting persistent peer...'
        sed -i 's/persistent_peers = \"\"/persistent_peers = \"${node1_id}@${node1_ip}:26656\"/' \$HOME_DIR/config/config.toml

        echo '[INFO] Node${node_num} initialization complete'
    "
}

# Create systemd service on node
create_service() {
    local node_name=$1
    local node_num=$2

    log_step "Creating systemd service on $node_name..."

    gcloud compute ssh "$node_name" --zone="$ZONE" --project="$PROJECT_ID" --command="
        set -euo pipefail

        echo '[INFO] Creating systemd service...'
        sudo tee /etc/systemd/system/pawd.service > /dev/null <<EOF
[Unit]
Description=PAW Blockchain Node
After=network-online.target

[Service]
User=root
ExecStart=/usr/local/bin/pawd start --home /root/.paw/node${node_num}
Restart=always
RestartSec=3
LimitNOFILE=4096
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

        echo '[INFO] Enabling service...'
        sudo systemctl daemon-reload
        sudo systemctl enable pawd

        echo '[INFO] Service created successfully'
    "
}

# Start node
start_node() {
    local node_name=$1

    log_step "Starting blockchain on $node_name..."

    gcloud compute ssh "$node_name" --zone="$ZONE" --project="$PROJECT_ID" --command="
        sudo systemctl start pawd
        sleep 3
        sudo systemctl status pawd --no-pager || true
    "
}

# Check node status
check_node_status() {
    local node_name=$1
    local node_ip=$2

    log_step "Checking status of $node_name..."

    sleep 5

    if gcloud compute ssh "$node_name" --zone="$ZONE" --project="$PROJECT_ID" --command="curl -s http://localhost:26657/status" 2>/dev/null; then
        log_info "✅ $node_name is running and responding"
    else
        log_warn "⚠️  $node_name may not be fully started yet"
    fi
}

# Main deployment process
main() {
    log_info "PAW Blockchain - GCP 3-Node Deployment"
    log_info "========================================"
    echo ""

    log_info "Chain ID: $CHAIN_ID"
    log_info "Nodes: 3"
    echo ""

    read -p "Start deployment? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Deployment cancelled"
        exit 0
    fi

    # Step 1: Build binary
    build_binary

    # Step 2: Install dependencies and copy binary to all nodes
    for node_info in "${NODES[@]}"; do
        IFS=':' read -r node_name node_ip <<< "$node_info"
        install_dependencies "$node_name" "$node_ip"
        copy_binary "$node_name"
    done

    # Step 3: Initialize node1 (genesis)
    init_node1

    # Step 4: Initialize node2 and node3
    init_follower_node "xai-testnode-2" 2
    init_follower_node "xai-testnode-3" 3

    # Step 5: Create systemd services
    create_service "xai-testnode-1" 1
    create_service "xai-testnode-2" 2
    create_service "xai-testnode-3" 3

    # Step 6: Start all nodes
    start_node "xai-testnode-1"
    sleep 5  # Give node1 time to start
    start_node "xai-testnode-2"
    start_node "xai-testnode-3"

    # Step 7: Check status
    log_info "Waiting for nodes to start (15 seconds)..."
    sleep 15

    for node_info in "${NODES[@]}"; do
        IFS=':' read -r node_name node_ip <<< "$node_info"
        check_node_status "$node_name" "$node_ip"
    done

    echo ""
    log_info "========================================="
    log_info "Deployment Complete!"
    log_info "========================================="
    echo ""
    log_info "Node Endpoints:"
    echo "  Node1 RPC: http://34.29.163.145:26657"
    echo "  Node2 RPC: http://108.59.86.86:26657"
    echo "  Node3 RPC: http://35.184.167.38:26657"
    echo ""
    log_info "Check status:"
    echo "  curl http://34.29.163.145:26657/status"
    echo ""
    log_info "View logs:"
    echo "  ./scripts/devnet/gcp-manage.sh logs 1"
    echo ""
    log_info "Account keys saved to:"
    echo "  $TMP_DIR/validator_key.json"
    echo "  $TMP_DIR/trader_key.json"
    echo "  $TMP_DIR/counterparty_key.json"
    echo ""
    log_warn "Remember to stop nodes when done testing!"
    echo "  ./scripts/devnet/gcp-manage.sh stop"
}

main "$@"
