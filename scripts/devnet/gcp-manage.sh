#!/usr/bin/env bash
# PAW Blockchain - GCP Test Node Management Script
# Easily start, stop, and check status of test nodes to manage costs

set -euo pipefail

PROJECT_ID="${PROJECT_ID:-paw-mvp-1}"
ZONE="us-central1-a"
NODES=("paw-testnode-1" "paw-testnode-2" "paw-testnode-3")

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Cost info
COST_PER_HOUR_E2_MEDIUM="0.0335"  # Approximate USD per hour per e2-medium instance

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_cost() {
    echo -e "${BLUE}[COST]${NC} $1"
}

# Start all test nodes
start_nodes() {
    log_info "Starting all test nodes..."

    for node in "${NODES[@]}"; do
        log_info "Starting $node..."
        gcloud compute instances start "$node" --zone="$ZONE" --project="$PROJECT_ID" 2>&1 || {
            log_warn "$node might already be running"
        }
    done

    log_info "Waiting for nodes to boot (10 seconds)..."
    sleep 10

    show_status
    log_cost "Estimated cost: \$$(echo "${#NODES[@]} * $COST_PER_HOUR_E2_MEDIUM" | bc) per hour while running"
}

# Stop all test nodes
stop_nodes() {
    log_info "Stopping all test nodes to save costs..."

    for node in "${NODES[@]}"; do
        log_info "Stopping $node..."
        gcloud compute instances stop "$node" --zone="$ZONE" --project="$PROJECT_ID" 2>&1 || {
            log_warn "$node might already be stopped"
        }
    done

    log_info "All nodes stopped"
    log_cost "Nodes are now stopped - only paying for disk storage (~\$0.04/GB/month)"
    show_status
}

# Show status of all nodes
show_status() {
    log_info "Current status of test nodes:"
    echo ""
    gcloud compute instances list --filter="name:paw-testnode-*" --project="$PROJECT_ID" --format="table(name,zone,machineType,status,networkInterfaces[0].accessConfigs[0].natIP:label=EXTERNAL_IP)"
    echo ""

    # Count running nodes
    running_count=$(gcloud compute instances list --filter="name:paw-testnode-* AND status:RUNNING" --project="$PROJECT_ID" --format="value(name)" | wc -l)

    if [ "$running_count" -gt 0 ]; then
        log_cost "💰 $running_count nodes running - Estimated cost: \$$(echo "$running_count * $COST_PER_HOUR_E2_MEDIUM" | bc)/hour"
        log_info "💡 Run '$0 stop' when done testing to save costs"
    else
        log_cost "✅ All nodes stopped - minimal costs (disk storage only)"
    fi
}

# SSH into a specific node
ssh_node() {
    local node_name="$1"
    log_info "Connecting to $node_name..."
    gcloud compute ssh "$node_name" --zone="$ZONE" --project="$PROJECT_ID"
}

# Show node logs
show_logs() {
    local node_name="${1:-paw-testnode-1}"
    log_info "Fetching logs from $node_name..."
    gcloud compute ssh "$node_name" --zone="$ZONE" --project="$PROJECT_ID" --command="tail -n 100 /root/.paw/*/pawd.log 2>/dev/null || journalctl -u pawd -n 100"
}

# Get IP addresses
show_ips() {
    log_info "Node IP addresses:"
    echo ""
    for node in "${NODES[@]}"; do
        ip=$(gcloud compute instances describe "$node" --zone="$ZONE" --project="$PROJECT_ID" --format="value(networkInterfaces[0].accessConfigs[0].natIP)" 2>/dev/null || echo "STOPPED")
        echo "  $node: $ip"
    done
    echo ""
}

# Show usage
usage() {
    cat << EOF
${GREEN}PAW Blockchain - GCP Test Node Management${NC}

${YELLOW}Usage:${NC}
    $0 <command> [options]

${YELLOW}Commands:${NC}
    start           Start all test nodes
    stop            Stop all test nodes (saves money!)
    status          Show status of all nodes
    ips             Show IP addresses of all nodes
    ssh <node>      SSH into a specific node (1, 2, or 3)
    logs <node>     Show logs from a node (default: node 1)
    cost            Show cost estimate

${YELLOW}Examples:${NC}
    $0 start                    # Start all nodes
    $0 status                   # Check status
    $0 ssh paw-testnode-1       # SSH to node 1
    $0 ssh 1                    # SSH to node 1 (shorthand)
    $0 logs 2                   # Show logs from node 2
    $0 stop                     # Stop all nodes to save costs

${BLUE}Cost Information:${NC}
    - Running: ~\$$(echo "3 * $COST_PER_HOUR_E2_MEDIUM" | bc)/hour for 3 x e2-medium instances
    - Stopped: Only disk storage (~\$0.40/month total)
    - ${RED}Always stop nodes when not testing!${NC}

EOF
}

# Show cost estimate
show_cost() {
    log_cost "GCP Cost Estimates for 3 Test Nodes:"
    echo ""
    echo "  Running (3 x e2-medium):"
    echo "    - Per hour:  \$$(echo "3 * $COST_PER_HOUR_E2_MEDIUM" | bc)"
    echo "    - Per day:   \$$(echo "3 * $COST_PER_HOUR_E2_MEDIUM * 24" | bc)"
    echo "    - Per month: \$$(echo "3 * $COST_PER_HOUR_E2_MEDIUM * 24 * 30" | bc)"
    echo ""
    echo "  Stopped:"
    echo "    - Disk storage only: ~\$0.40/month"
    echo ""
    log_warn "💡 Best Practice: Run 'stop' command when not actively testing"
}

# Main command handler
case "${1:-}" in
    start)
        start_nodes
        ;;
    stop)
        stop_nodes
        ;;
    status)
        show_status
        ;;
    ips)
        show_ips
        ;;
    ssh)
        node_arg="${2:-1}"
        if [[ "$node_arg" =~ ^[1-3]$ ]]; then
            ssh_node "paw-testnode-${node_arg}"
        else
            ssh_node "$node_arg"
        fi
        ;;
    logs)
        node_arg="${2:-1}"
        if [[ "$node_arg" =~ ^[1-3]$ ]]; then
            show_logs "paw-testnode-${node_arg}"
        else
            show_logs "$node_arg"
        fi
        ;;
    cost)
        show_cost
        ;;
    -h|--help|help)
        usage
        ;;
    *)
        log_error "Unknown command: ${1:-}"
        echo ""
        usage
        exit 1
        ;;
esac
