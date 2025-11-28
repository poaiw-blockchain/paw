#!/bin/bash
# PAW Blockchain - Validator Deployment Script
# Deploys validator nodes to Kubernetes cluster with security hardening

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
K8S_DIR="$PROJECT_ROOT/k8s"

# Default values
NAMESPACE="${NAMESPACE:-paw-blockchain}"
REPLICAS="${REPLICAS:-3}"
CHAIN_ID="${CHAIN_ID:-paw-1}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
STORAGE_SIZE="${STORAGE_SIZE:-1Ti}"
VALIDATOR_KEYS_PATH="${VALIDATOR_KEYS_PATH:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Please install kubectl first."
        exit 1
    fi

    # Check if connected to cluster
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Not connected to a Kubernetes cluster. Please configure kubectl."
        exit 1
    fi

    # Check if validator keys path is provided
    if [ -z "$VALIDATOR_KEYS_PATH" ]; then
        log_error "Validator keys path is required. Use --keys-path option."
        exit 1
    fi

    # Check if validator keys exist
    if [ ! -f "$VALIDATOR_KEYS_PATH/priv_validator_key.json" ]; then
        log_error "priv_validator_key.json not found at $VALIDATOR_KEYS_PATH"
        exit 1
    fi

    if [ ! -f "$VALIDATOR_KEYS_PATH/node_key.json" ]; then
        log_error "node_key.json not found at $VALIDATOR_KEYS_PATH"
        exit 1
    fi

    log_info "Prerequisites check passed"
}

# Create namespace if it doesn't exist
create_namespace() {
    log_info "Creating namespace $NAMESPACE..."

    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_info "Namespace $NAMESPACE already exists"
    else
        kubectl apply -f "$K8S_DIR/namespace.yaml"
        log_info "Namespace $NAMESPACE created"
    fi
}

# Deploy validator keys as secrets
deploy_secrets() {
    log_info "Deploying validator keys as secrets..."

    # Check if secret already exists
    if kubectl get secret paw-validator-keys -n "$NAMESPACE" &> /dev/null; then
        log_warn "Secret paw-validator-keys already exists"
        read -p "Do you want to update it? (y/n) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Skipping secret update"
            return
        fi
        kubectl delete secret paw-validator-keys -n "$NAMESPACE"
    fi

    # Create secret from validator keys
    kubectl create secret generic paw-validator-keys \
        --from-file=priv_validator_key.json="$VALIDATOR_KEYS_PATH/priv_validator_key.json" \
        --from-file=node_key.json="$VALIDATOR_KEYS_PATH/node_key.json" \
        --namespace="$NAMESPACE"

    log_info "Validator keys deployed as secret"
    log_warn "IMPORTANT: Backup your validator keys in a secure location!"
}

# Deploy storage resources
deploy_storage() {
    log_info "Deploying storage resources..."

    kubectl apply -f "$K8S_DIR/storage.yaml"
    log_info "Storage resources deployed"
}

# Deploy configuration
deploy_configuration() {
    log_info "Deploying configuration..."

    kubectl apply -f "$K8S_DIR/configmaps.yaml"
    log_info "Configuration deployed"
}

# Deploy services
deploy_services() {
    log_info "Deploying validator services..."

    kubectl apply -f "$K8S_DIR/services.yaml"
    log_info "Services deployed"
}

# Deploy validator statefulset
deploy_validator() {
    log_info "Deploying PAW validators..."

    # Update statefulset with custom values
    sed -e "s|replicas: 3|replicas: $REPLICAS|g" \
        -e "s|image: paw-chain/paw-validator:latest|image: paw-chain/paw-validator:$IMAGE_TAG|g" \
        -e "s|storage: 1Ti|storage: $STORAGE_SIZE|g" \
        "$K8S_DIR/validator-statefulset.yaml" | kubectl apply -f -

    log_info "Validator StatefulSet created"
}

# Wait for validators to be ready
wait_for_validators() {
    log_info "Waiting for validators to be ready..."

    if kubectl rollout status statefulset/paw-validator -n "$NAMESPACE" --timeout=15m; then
        log_info "Validators are ready"
    else
        log_error "Validators failed to become ready"
        log_error "Check logs with: kubectl logs -n $NAMESPACE -l component=validator"
        exit 1
    fi
}

# Verify validator setup
verify_validators() {
    log_info "Verifying validator setup..."

    # Check if validators are running
    RUNNING_PODS=$(kubectl get pods -n "$NAMESPACE" -l component=validator --field-selector=status.phase=Running --no-headers | wc -l)

    if [ "$RUNNING_PODS" -eq "$REPLICAS" ]; then
        log_info "All $REPLICAS validators are running"
    else
        log_warn "Only $RUNNING_PODS out of $REPLICAS validators are running"
    fi

    # Check validator status
    log_info "Checking validator status..."
    for i in $(seq 0 $((REPLICAS-1))); do
        POD_NAME="paw-validator-$i"
        log_info "Checking $POD_NAME..."

        if kubectl exec -n "$NAMESPACE" "$POD_NAME" -- pawcli status --node tcp://localhost:26657 &> /dev/null; then
            log_info "$POD_NAME is healthy"
        else
            log_warn "$POD_NAME is not responding to status checks"
        fi
    done
}

# Display deployment info
display_info() {
    log_info "Validator deployment completed successfully!"
    echo ""
    log_info "Deployment Information:"
    echo "  Namespace: $NAMESPACE"
    echo "  Replicas: $REPLICAS"
    echo "  Chain ID: $CHAIN_ID"
    echo "  Image: paw-chain/paw-validator:$IMAGE_TAG"
    echo ""

    log_info "Validator Pods:"
    kubectl get pods -n "$NAMESPACE" -l component=validator -o wide

    echo ""
    log_info "Validator Services:"
    kubectl get svc -n "$NAMESPACE" -l component=validator

    echo ""
    log_info "To view validator logs:"
    echo "  kubectl logs -f -n $NAMESPACE paw-validator-0"

    echo ""
    log_info "To check validator status:"
    echo "  kubectl exec -n $NAMESPACE paw-validator-0 -- pawcli status"

    echo ""
    log_info "To create validator transaction:"
    echo "  kubectl exec -it -n $NAMESPACE paw-validator-0 -- pawd tx staking create-validator \\"
    echo "    --amount=1000000upaw \\"
    echo "    --pubkey=\$(pawd tendermint show-validator) \\"
    echo "    --moniker=\"My Validator\" \\"
    echo "    --commission-rate=\"0.10\" \\"
    echo "    --commission-max-rate=\"0.20\" \\"
    echo "    --commission-max-change-rate=\"0.01\" \\"
    echo "    --min-self-delegation=\"1\" \\"
    echo "    --from=validator \\"
    echo "    --chain-id=$CHAIN_ID"

    echo ""
    log_warn "IMPORTANT SECURITY NOTES:"
    echo "  1. Backup validator keys regularly"
    echo "  2. Monitor validator uptime and performance"
    echo "  3. Keep the validator binary updated"
    echo "  4. Use sentry nodes to protect validators"
    echo "  5. Enable monitoring and alerting"
}

# Main deployment function
main() {
    log_info "Starting PAW validator deployment..."
    log_info "Configuration:"
    echo "  Namespace: $NAMESPACE"
    echo "  Replicas: $REPLICAS"
    echo "  Chain ID: $CHAIN_ID"
    echo "  Image Tag: $IMAGE_TAG"
    echo "  Storage Size: $STORAGE_SIZE"
    echo "  Keys Path: $VALIDATOR_KEYS_PATH"
    echo ""

    log_warn "WARNING: This will deploy validator nodes with the provided keys."
    log_warn "Make sure you have backed up your validator keys securely!"
    echo ""

    read -p "Proceed with deployment? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Deployment cancelled"
        exit 0
    fi

    check_prerequisites
    create_namespace
    deploy_secrets
    deploy_storage
    deploy_configuration
    deploy_services
    deploy_validator
    wait_for_validators
    verify_validators
    display_info

    log_info "Deployment script completed successfully"
}

# Show usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Deploy PAW blockchain validator nodes to Kubernetes

OPTIONS:
    -n, --namespace     Kubernetes namespace (default: paw-blockchain)
    -r, --replicas      Number of validators (default: 3)
    -c, --chain-id      Chain ID (default: paw-1)
    -t, --image-tag     Docker image tag (default: latest)
    -s, --storage       Storage size (default: 1Ti)
    -k, --keys-path     Path to validator keys directory (required)
    -h, --help          Show this help message

REQUIRED:
    Validator keys directory must contain:
    - priv_validator_key.json
    - node_key.json

EXAMPLES:
    # Deploy with defaults
    $0 --keys-path /path/to/validator/keys

    # Deploy with custom replicas
    $0 --keys-path /path/to/keys --replicas 5

    # Deploy with specific image tag
    $0 --keys-path /path/to/keys --image-tag v1.0.0

    # Deploy to custom namespace
    $0 --keys-path /path/to/keys --namespace paw-testnet

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -r|--replicas)
            REPLICAS="$2"
            shift 2
            ;;
        -c|--chain-id)
            CHAIN_ID="$2"
            shift 2
            ;;
        -t|--image-tag)
            IMAGE_TAG="$2"
            shift 2
            ;;
        -s|--storage)
            STORAGE_SIZE="$2"
            shift 2
            ;;
        -k|--keys-path)
            VALIDATOR_KEYS_PATH="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Run main function
main
