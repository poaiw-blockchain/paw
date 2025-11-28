#!/bin/bash

# PAW Blockchain - Kubernetes Deployment Script
# This script deploys the PAW blockchain to a Kubernetes cluster

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
K8S_DIR="$PROJECT_ROOT/k8s"
NAMESPACE="paw-blockchain"

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

check_prerequisites() {
    log_step "Checking prerequisites..."

    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Please install kubectl first."
        exit 1
    fi
    log_info "kubectl version: $(kubectl version --client --short 2>/dev/null || kubectl version --client)"

    # Check cluster connection
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi
    log_info "Connected to cluster: $(kubectl config current-context)"

    # Check if helm is available (optional but recommended)
    if command -v helm &> /dev/null; then
        log_info "Helm version: $(helm version --short)"
    else
        log_warn "Helm is not installed. Some features may not be available."
    fi
}

create_namespace() {
    log_step "Creating namespace..."

    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_warn "Namespace $NAMESPACE already exists. Skipping creation."
    else
        kubectl apply -f "$K8S_DIR/namespace.yaml"
        log_info "Namespace created: $NAMESPACE"
    fi
}

create_secrets() {
    log_step "Creating secrets..."

    # Check if secrets already exist
    if kubectl get secret paw-secrets -n "$NAMESPACE" &> /dev/null; then
        log_warn "Secret 'paw-secrets' already exists. Skipping creation."
        log_info "To update secrets, delete existing secret first:"
        log_info "  kubectl delete secret paw-secrets -n $NAMESPACE"
        return
    fi

    # Generate JWT secret
    JWT_SECRET=$(openssl rand -base64 32)

    # Generate Grafana password
    GRAFANA_PASSWORD=$(openssl rand -base64 16)

    # Create secret
    kubectl create secret generic paw-secrets \
        --from-literal=JWT_SECRET="$JWT_SECRET" \
        --namespace="$NAMESPACE"

    kubectl create secret generic monitoring-secrets \
        --from-literal=GRAFANA_ADMIN_PASSWORD="$GRAFANA_PASSWORD" \
        --from-literal=ALERTMANAGER_WEBHOOK_URL="" \
        --from-literal=SMTP_PASSWORD="" \
        --namespace="$NAMESPACE"

    log_info "Secrets created successfully"
    log_warn "IMPORTANT: Save these credentials!"
    echo ""
    echo "JWT_SECRET: $JWT_SECRET"
    echo "GRAFANA_PASSWORD: $GRAFANA_PASSWORD"
    echo ""
    echo "Save these credentials in a secure location!"

    # Save to file (in secure location)
    CREDENTIALS_FILE="$HOME/.paw-k8s-credentials-$(date +%Y%m%d_%H%M%S).txt"
    cat > "$CREDENTIALS_FILE" << EOF
PAW Blockchain Kubernetes Credentials
Generated: $(date)
Cluster: $(kubectl config current-context)

JWT_SECRET=$JWT_SECRET
GRAFANA_PASSWORD=$GRAFANA_PASSWORD

WARNING: Keep this file secure and delete after saving credentials elsewhere!
EOF
    chmod 600 "$CREDENTIALS_FILE"
    log_info "Credentials saved to: $CREDENTIALS_FILE"
}

apply_configmaps() {
    log_step "Applying ConfigMaps..."

    kubectl apply -f "$K8S_DIR/configmap.yaml"
    log_info "ConfigMaps applied"
}

apply_storage() {
    log_step "Configuring storage..."

    kubectl apply -f "$K8S_DIR/persistent-volumes.yaml"
    log_info "Storage configured"

    # Wait for PVCs to be bound
    log_info "Waiting for PersistentVolumeClaims to be bound..."
    kubectl wait --for=jsonpath='{.status.phase}'=Bound \
        pvc/paw-node-data -n "$NAMESPACE" \
        --timeout=300s || log_warn "PVC binding timeout - check manually"
}

deploy_monitoring() {
    log_step "Deploying monitoring stack..."

    # Apply monitoring configmaps (create if they don't exist)
    if [ -d "$PROJECT_ROOT/monitoring" ]; then
        log_info "Creating monitoring configurations..."

        # Create Prometheus config
        kubectl create configmap prometheus-config \
            --from-file="$PROJECT_ROOT/monitoring/prometheus.yml" \
            -n "$NAMESPACE" \
            --dry-run=client -o yaml | kubectl apply -f -

        # Create Grafana datasources
        kubectl create configmap grafana-datasources \
            --from-file="$PROJECT_ROOT/monitoring/grafana-datasources.yml" \
            -n "$NAMESPACE" \
            --dry-run=client -o yaml | kubectl apply -f -

        # Create Grafana dashboards
        kubectl create configmap grafana-dashboards \
            --from-file="$PROJECT_ROOT/monitoring/grafana-dashboards.json" \
            -n "$NAMESPACE" \
            --dry-run=client -o yaml | kubectl apply -f -

        # Create AlertManager config
        kubectl create configmap alertmanager-config \
            --from-file="$PROJECT_ROOT/monitoring/alertmanager.yml" \
            -n "$NAMESPACE" \
            --dry-run=client -o yaml | kubectl apply -f -

        # Create Loki config
        kubectl create configmap loki-config \
            --from-file="$PROJECT_ROOT/monitoring/loki-config.yaml" \
            -n "$NAMESPACE" \
            --dry-run=client -o yaml | kubectl apply -f -
    fi

    kubectl apply -f "$K8S_DIR/monitoring-deployment.yaml"
    log_info "Monitoring stack deployed"
}

deploy_application() {
    log_step "Deploying PAW blockchain..."

    # Deploy nodes
    kubectl apply -f "$K8S_DIR/paw-node-deployment.yaml"
    log_info "PAW nodes deployment created"

    # Deploy API
    kubectl apply -f "$K8S_DIR/paw-api-deployment.yaml"
    log_info "PAW API deployment created"

    # Create services
    kubectl apply -f "$K8S_DIR/all-services.yaml"
    log_info "Services created"

    # Configure HPA
    kubectl apply -f "$K8S_DIR/hpa.yaml"
    log_info "Horizontal Pod Autoscaling configured"

    # Apply network policies
    kubectl apply -f "$K8S_DIR/network-policy.yaml"
    log_info "Network policies applied"

    # Apply ingress
    if [ -f "$K8S_DIR/ingress.yaml" ]; then
        kubectl apply -f "$K8S_DIR/ingress.yaml"
        log_info "Ingress configured"
    fi
}

wait_for_pods() {
    log_step "Waiting for pods to be ready..."

    log_info "Waiting for PAW nodes..."
    kubectl wait --for=condition=ready pod \
        -l app=paw-node \
        -n "$NAMESPACE" \
        --timeout=600s || log_warn "Timeout waiting for nodes - check status manually"

    log_info "Waiting for PAW API..."
    kubectl wait --for=condition=ready pod \
        -l app=paw-api \
        -n "$NAMESPACE" \
        --timeout=300s || log_warn "Timeout waiting for API - check status manually"
}

display_status() {
    log_step "Deployment Status"

    echo ""
    echo "========================================"
    echo "PAW Blockchain - Kubernetes Deployment"
    echo "========================================"
    echo ""
    echo "Namespace: $NAMESPACE"
    echo "Cluster:   $(kubectl config current-context)"
    echo ""

    # Get pod status
    log_info "Pod Status:"
    kubectl get pods -n "$NAMESPACE" -o wide

    echo ""
    log_info "Service Status:"
    kubectl get svc -n "$NAMESPACE"

    echo ""
    log_info "Ingress Status:"
    kubectl get ingress -n "$NAMESPACE" 2>/dev/null || echo "No ingress configured"

    echo ""
    echo "Useful Commands:"
    echo "  View logs:          kubectl logs -f -l app=paw-node -n $NAMESPACE"
    echo "  Check pod status:   kubectl get pods -n $NAMESPACE"
    echo "  Describe pod:       kubectl describe pod <pod-name> -n $NAMESPACE"
    echo "  Execute in pod:     kubectl exec -it <pod-name> -n $NAMESPACE -- /bin/sh"
    echo "  Port forward API:   kubectl port-forward svc/paw-api 5000:5000 -n $NAMESPACE"
    echo "  Port forward RPC:   kubectl port-forward svc/paw-node 26657:26657 -n $NAMESPACE"
    echo "  Scale deployment:   kubectl scale deployment/paw-node --replicas=5 -n $NAMESPACE"
    echo "  Delete deployment:  kubectl delete all --all -n $NAMESPACE"
    echo ""
    echo "Monitoring:"
    echo "  Grafana:            kubectl port-forward svc/grafana 3000:3000 -n $NAMESPACE"
    echo "  Prometheus:         kubectl port-forward svc/prometheus 9090:9090 -n $NAMESPACE"
    echo ""
    echo "========================================"
}

rollback() {
    log_error "Deployment failed! Rolling back..."
    kubectl delete namespace "$NAMESPACE" --ignore-not-found=true
    exit 1
}

# Main execution
main() {
    echo "========================================"
    echo "PAW Blockchain - Kubernetes Deployment"
    echo "========================================"
    echo ""

    # Parse command line arguments
    SKIP_MONITORING=false
    DRY_RUN=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-monitoring)
                SKIP_MONITORING=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --skip-monitoring    Skip monitoring stack deployment"
                echo "  --dry-run            Show what would be deployed without applying"
                echo "  --namespace NAME     Use custom namespace (default: paw-blockchain)"
                echo "  --help               Show this help message"
                echo ""
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done

    if [ "$DRY_RUN" = true ]; then
        log_info "DRY RUN MODE - No changes will be applied"
        export KUBECTL_FLAGS="--dry-run=client"
    fi

    # Set error trap
    trap rollback ERR

    check_prerequisites
    create_namespace
    create_secrets
    apply_configmaps
    apply_storage

    if [ "$SKIP_MONITORING" = false ]; then
        deploy_monitoring
    fi

    deploy_application
    wait_for_pods
    display_status

    log_info "Deployment completed successfully!"
}

# Run main function
main "$@"
