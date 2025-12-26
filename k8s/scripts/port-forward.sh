#!/bin/bash
# PAW K8s Port Forward Helper
# Exposes RPC, REST, and gRPC endpoints for cross-environment access

set -euo pipefail

NAMESPACE="${NAMESPACE:-paw-blockchain}"
RPC_LOCAL="${RPC_PORT:-26658}"
REST_LOCAL="${REST_PORT:-1318}"
GRPC_LOCAL="${GRPC_PORT:-9091}"

usage() {
    cat <<EOF
Usage: $0 [start|stop|status]

Forwards K8s ports for cross-environment access:
  RPC:  localhost:$RPC_LOCAL  -> pod:26657
  REST: localhost:$REST_LOCAL -> pod:1317
  gRPC: localhost:$GRPC_LOCAL -> pod:9090

Environment variables:
  NAMESPACE  - K8s namespace (default: paw-blockchain)
  RPC_PORT   - Local RPC port (default: 26658)
  REST_PORT  - Local REST port (default: 1318)
  GRPC_PORT  - Local gRPC port (default: 9091)

Access from other machines via Tailscale:
  RPC:  curl http://100.91.253.108:$RPC_LOCAL/status
  REST: curl http://100.91.253.108:$REST_LOCAL/cosmos/base/tendermint/v1beta1/node_info
EOF
}

start_forwards() {
    echo "Starting port forwards for namespace: $NAMESPACE"

    # Get first running validator pod
    local pod=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=validator \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)

    if [ -z "$pod" ]; then
        echo "Error: No validator pod found in namespace $NAMESPACE"
        exit 1
    fi

    echo "Using pod: $pod"

    # Start port forwards in background
    kubectl port-forward -n "$NAMESPACE" "pod/$pod" "$RPC_LOCAL:26657" --address 0.0.0.0 &
    echo $! > /tmp/paw-pf-rpc.pid

    kubectl port-forward -n "$NAMESPACE" "pod/$pod" "$REST_LOCAL:1317" --address 0.0.0.0 &
    echo $! > /tmp/paw-pf-rest.pid

    kubectl port-forward -n "$NAMESPACE" "pod/$pod" "$GRPC_LOCAL:9090" --address 0.0.0.0 &
    echo $! > /tmp/paw-pf-grpc.pid

    sleep 2

    echo ""
    echo "Port forwards active:"
    echo "  RPC:  localhost:$RPC_LOCAL  (Tailscale: 100.91.253.108:$RPC_LOCAL)"
    echo "  REST: localhost:$REST_LOCAL (Tailscale: 100.91.253.108:$REST_LOCAL)"
    echo "  gRPC: localhost:$GRPC_LOCAL (Tailscale: 100.91.253.108:$GRPC_LOCAL)"
    echo ""
    echo "Test: curl localhost:$RPC_LOCAL/status | jq .result.sync_info.latest_block_height"
}

stop_forwards() {
    echo "Stopping port forwards..."
    for pidfile in /tmp/paw-pf-*.pid; do
        if [ -f "$pidfile" ]; then
            pid=$(cat "$pidfile")
            if kill -0 "$pid" 2>/dev/null; then
                kill "$pid" 2>/dev/null || true
                echo "Stopped PID $pid"
            fi
            rm -f "$pidfile"
        fi
    done

    # Also kill any lingering kubectl port-forward processes for paw
    pkill -f "kubectl port-forward.*paw" 2>/dev/null || true
    echo "Done"
}

show_status() {
    echo "Port forward status:"
    echo ""

    local running=0
    for service in rpc rest grpc; do
        pidfile="/tmp/paw-pf-$service.pid"
        if [ -f "$pidfile" ]; then
            pid=$(cat "$pidfile")
            if kill -0 "$pid" 2>/dev/null; then
                echo "  $service: Running (PID $pid)"
                ((running++))
            else
                echo "  $service: Stopped (stale PID file)"
            fi
        else
            echo "  $service: Not running"
        fi
    done

    if [ $running -gt 0 ]; then
        echo ""
        echo "Testing RPC connectivity..."
        if curl -s "localhost:$RPC_LOCAL/status" > /dev/null 2>&1; then
            height=$(curl -s "localhost:$RPC_LOCAL/status" | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
            echo "  RPC responsive, block height: $height"
        else
            echo "  RPC not responding"
        fi
    fi
}

case "${1:-}" in
    start)
        start_forwards
        ;;
    stop)
        stop_forwards
        ;;
    status)
        show_status
        ;;
    *)
        usage
        ;;
esac
