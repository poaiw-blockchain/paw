#!/usr/bin/env bash
# PAW Blockchain - GCP 3-Node Deployment Script
# Deploys and configures PAW blockchain on GCP test nodes

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Configuration (overridable via env vars)
PROJECT_ID="${PROJECT_ID:-aixn-node-1}"
ZONE="${ZONE:-us-central1-a}"
CHAIN_ID="${CHAIN_ID:-paw-testnet-1}"
KEYRING_BACKEND="${KEYRING_BACKEND:-test}"
NETWORK_DIR="${NETWORK_DIR:-${PROJECT_ROOT}/networks/${CHAIN_ID}}"
PUBLISH_ARTIFACTS="${PUBLISH_ARTIFACTS:-1}"
VERIFY_SCRIPT="${PROJECT_ROOT}/scripts/devnet/verify-network-artifacts.sh"

if [[ -n "${NODES_SPEC:-}" ]]; then
    IFS=',' read -r -a NODES <<< "${NODES_SPEC}"
else
    NODES=(
        "xai-testnode-1:34.29.163.145"
        "xai-testnode-2:108.59.86.86"
        "xai-testnode-3:35.184.167.38"
    )
fi

declare -A NODE_IDS
declare -A NODE_IPS
for node_info in "${NODES[@]}"; do
    IFS=':' read -r node_name node_ip <<< "$node_info"
    NODE_IPS["$node_name"]="$node_ip"
done

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
        parse_addr() { awk '/^[[:space:]]*address:/{print \$2; exit}' \"\$1\"; }

        echo '[INFO] Initializing chain...'
        pawd init node1 --chain-id $CHAIN_ID --home \$HOME_DIR

        echo '[INFO] Creating accounts...'
        pawd keys add validator --keyring-backend $KEYRING_BACKEND --home \$HOME_DIR > /tmp/validator_key.txt
        pawd keys add smoke-trader --keyring-backend $KEYRING_BACKEND --home \$HOME_DIR > /tmp/trader_key.txt
        pawd keys add smoke-counterparty --keyring-backend $KEYRING_BACKEND --home \$HOME_DIR > /tmp/counterparty_key.txt
        VALIDATOR_ADDR=\$(parse_addr /tmp/validator_key.txt)
        TRADER_ADDR=\$(parse_addr /tmp/trader_key.txt)
        COUNTERPARTY_ADDR=\$(parse_addr /tmp/counterparty_key.txt)

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
    gcloud compute scp "$node_name:/tmp/validator_key.txt" "$TMP_DIR/validator_key.txt" --zone="$ZONE" --project="$PROJECT_ID"
    gcloud compute scp "$node_name:/tmp/trader_key.txt" "$TMP_DIR/trader_key.txt" --zone="$ZONE" --project="$PROJECT_ID"
    gcloud compute scp "$node_name:/tmp/counterparty_key.txt" "$TMP_DIR/counterparty_key.txt" --zone="$ZONE" --project="$PROJECT_ID"

    log_info "Node1 ID: $(cat $TMP_DIR/node1.id)"
    NODE_IDS["$node_name"]="$(cat "$TMP_DIR/node1.id")"
}

# Initialize node2 or node3
init_follower_node() {
    local node_name=$1
    local node_num=$2
    local node1_ip="${NODE_IPS["xai-testnode-1"]}"
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
    gcloud compute scp "$node_name:/tmp/node${node_num}.id" "$TMP_DIR/${node_name}.id" --zone="$ZONE" --project="$PROJECT_ID"
    NODE_IDS["$node_name"]="$(cat "$TMP_DIR/${node_name}.id")"
}

publish_artifacts() {
    if [[ "${PUBLISH_ARTIFACTS}" != "1" ]]; then
        log_warn "Skipping local artifact sync (PUBLISH_ARTIFACTS=0)"
        return
    fi

    if [[ ! -f "$TMP_DIR/genesis.json" ]]; then
        log_warn "Genesis file not found in $TMP_DIR (skipping artifact publish)"
        return
    fi

    log_step "Syncing artifacts into ${NETWORK_DIR}"
    mkdir -p "${NETWORK_DIR}"
    cp "$TMP_DIR/genesis.json" "${NETWORK_DIR}/genesis.json"
    (cd "${NETWORK_DIR}" && sha256sum genesis.json > genesis.sha256)

    local peers_list=()
    local seeds_entry=""
    for node_info in "${NODES[@]}"; do
        IFS=':' read -r node_name node_ip <<< "$node_info"
        local node_id="${NODE_IDS["$node_name"]:-}"
        if [[ -n "$node_id" && -n "$node_ip" ]]; then
            peers_list+=("${node_id}@${node_ip}:26656")
            if [[ -z "$seeds_entry" ]]; then
                seeds_entry="${node_id}@${node_ip}:26656"
            fi
        fi
    done
    local peers_joined=""
    if [[ ${#peers_list[@]} -gt 0 ]]; then
        peers_joined=$(IFS=','; echo "${peers_list[*]}")
    fi

    cat > "${NETWORK_DIR}/peers.txt" <<EOF
# Generated by scripts/devnet/gcp-deploy.sh on $(date -u +%Y-%m-%dT%H:%M:%SZ). Update with production endpoints before publishing.
seeds=${seeds_entry}
persistent_peers=${peers_joined}
EOF

    log_info "Artifacts staged under ${NETWORK_DIR}"
    if [[ -x "${VERIFY_SCRIPT}" ]]; then
        (cd "${PROJECT_ROOT}" && "${VERIFY_SCRIPT}" "${CHAIN_ID}") || log_warn "Artifact verification reported issues"
    else
        log_warn "Verification script not found at ${VERIFY_SCRIPT}"
    fi
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
    publish_artifacts

    log_info "Node Endpoints:"
    for node_info in "${NODES[@]}"; do
        IFS=':' read -r node_name node_ip <<< "$node_info"
        echo "  ${node_name} RPC: http://${node_ip}:26657"
    done
    echo ""
    log_info "Check status:"
    echo "  curl http://34.29.163.145:26657/status"
    echo ""
    log_info "View logs:"
    echo "  ./scripts/devnet/gcp-manage.sh logs 1"
    echo ""
    log_info "Account keys saved to:"
    echo "  $TMP_DIR/validator_key.txt"
    echo "  $TMP_DIR/trader_key.txt"
    echo "  $TMP_DIR/counterparty_key.txt"
    echo ""
    log_warn "Remember to stop nodes when done testing!"
    echo "  ./scripts/devnet/gcp-manage.sh stop"
}

main "$@"
