#!/bin/bash
# Enable Prometheus metrics in all PAW nodes

set -euo pipefail

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running in Docker context
if [ -d "/root/.paw" ]; then
    PAW_HOME="/root/.paw"
    log_info "Running in Docker context"
else
    PAW_HOME="$HOME/.paw"
    log_info "Running on host"
fi

# Enable metrics for a single node
enable_metrics_for_node() {
    local node_dir="$1"
    local node_name=$(basename "$node_dir")

    log_info "Enabling metrics for $node_name..."

    local config_file="$node_dir/config/config.toml"
    local app_file="$node_dir/config/app.toml"

    if [ ! -f "$config_file" ]; then
        log_error "Config file not found: $config_file"
        return 1
    fi

    if [ ! -f "$app_file" ]; then
        log_error "App config file not found: $app_file"
        return 1
    fi

    # Enable CometBFT Prometheus metrics
    log_info "  Enabling CometBFT metrics..."
    sed -i 's/^prometheus = false/prometheus = true/' "$config_file"
    sed -i 's/^prometheus_listen_addr = ".*"/prometheus_listen_addr = ":26660"/' "$config_file"

    # Enable app telemetry
    log_info "  Enabling app telemetry..."
    sed -i '/^\[telemetry\]/,/^prometheus-retention-time/ {
        s/^enabled = false/enabled = true/
        s/^enable-hostname = false/enable-hostname = true/
        s/^enable-hostname-label = false/enable-hostname-label = true/
        s/^enable-service-label = false/enable-service-label = true/
        s/^prometheus-retention-time = 0/prometheus-retention-time = 600/
    }' "$app_file"

    # Set service name
    sed -i '/^\[telemetry\]/,/^enabled/ {
        s/^service-name = ""/service-name = "paw"/
    }' "$app_file"

    log_success "  Metrics enabled for $node_name"
}

# Main execution
main() {
    log_info "PAW Node Metrics Enabler"
    log_info "========================"
    echo ""

    # Find all node directories
    if [ -d "$PAW_HOME" ]; then
        local nodes_found=0

        # Check for single node setup
        if [ -f "$PAW_HOME/config/config.toml" ]; then
            enable_metrics_for_node "$PAW_HOME"
            nodes_found=$((nodes_found + 1))
        fi

        # Check for multi-node setup (node1, node2, etc.)
        for node_dir in "$PAW_HOME"/node*; do
            if [ -d "$node_dir" ] && [ -f "$node_dir/config/config.toml" ]; then
                enable_metrics_for_node "$node_dir"
                nodes_found=$((nodes_found + 1))
            fi
        done

        if [ $nodes_found -eq 0 ]; then
            log_error "No PAW nodes found in $PAW_HOME"
            exit 1
        fi

        echo ""
        log_success "Metrics enabled for $nodes_found node(s)"
        echo ""
        log_warn "IMPORTANT: You must restart the nodes for changes to take effect:"
        echo "  Docker: docker-compose restart"
        echo "  Systemd: sudo systemctl restart pawd"
        echo "  Manual: pkill pawd && pawd start"

    else
        log_error "PAW home directory not found: $PAW_HOME"
        log_info "Initialize a node first: pawd init <moniker> --chain-id paw-mvp-1"
        exit 1
    fi
}

main "$@"
