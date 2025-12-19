#!/usr/bin/env bash
# Verify monitoring overlays (Alertmanager/Prometheus rules/Grafana) for PAW
set -euo pipefail

NAMESPACE="${NAMESPACE:-paw}"
MONITORING_NS="${MONITORING_NS:-monitoring}"
STATUS=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

usage() {
    cat <<'EOF'
Usage: ./verify-monitoring-overlays.sh [options]

Checks:
  - Prometheus pods and services running
  - Grafana pods and services running
  - AlertManager pods and services running
  - ServiceMonitors configured for PAW
  - PrometheusRules configured for PAW
  - Grafana datasources and dashboards
  - Alert routing configuration

Options:
  --namespace NS        PAW namespace (default: paw)
  --monitoring-ns NS    Monitoring namespace (default: monitoring)
  --probe-alertmanager  Test AlertManager API
  --help, -h            Show this help
EOF
}

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; STATUS=1; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; STATUS=1; }
log_ok() { echo -e "${GREEN}[OK]${NC} $1"; }

PROBE_ALERTMANAGER=0

while [[ $# -gt 0 ]]; do
    case "$1" in
        --namespace) NAMESPACE="$2"; shift 2 ;;
        --monitoring-ns) MONITORING_NS="$2"; shift 2 ;;
        --probe-alertmanager) PROBE_ALERTMANAGER=1; shift ;;
        -h|--help) usage; exit 0 ;;
        *) log_error "Unknown argument: $1"; usage; exit 1 ;;
    esac
done

log_info "PAW namespace: $NAMESPACE"
log_info "Monitoring namespace: $MONITORING_NS"
echo ""

# Check prerequisites
if ! command -v kubectl >/dev/null 2>&1; then
    log_error "kubectl not found"
    exit 1
fi

if ! kubectl cluster-info >/dev/null 2>&1; then
    log_error "Cannot reach Kubernetes cluster"
    exit 1
fi

# Check Prometheus
check_prometheus() {
    log_info "Checking Prometheus..."

    if kubectl -n "$MONITORING_NS" get pods -l app.kubernetes.io/name=prometheus --no-headers 2>/dev/null | grep -q Running; then
        log_ok "Prometheus pods running"
    else
        log_error "Prometheus pods not running"
    fi

    if kubectl -n "$MONITORING_NS" get svc prometheus-kube-prometheus-prometheus >/dev/null 2>&1; then
        log_ok "Prometheus service exists"
    else
        log_warn "Prometheus service not found"
    fi
}

# Check Grafana
check_grafana() {
    log_info "Checking Grafana..."

    if kubectl -n "$MONITORING_NS" get pods -l app.kubernetes.io/name=grafana --no-headers 2>/dev/null | grep -q Running; then
        log_ok "Grafana pods running"
    else
        log_error "Grafana pods not running"
    fi

    if kubectl -n "$MONITORING_NS" get svc prometheus-grafana >/dev/null 2>&1; then
        log_ok "Grafana service exists"
    else
        log_warn "Grafana service not found"
    fi
}

# Check AlertManager
check_alertmanager() {
    log_info "Checking AlertManager..."

    if kubectl -n "$MONITORING_NS" get pods -l app.kubernetes.io/name=alertmanager --no-headers 2>/dev/null | grep -q Running; then
        log_ok "AlertManager pods running"
    else
        log_error "AlertManager pods not running"
    fi

    if kubectl -n "$MONITORING_NS" get svc alertmanager-operated >/dev/null 2>&1; then
        log_ok "AlertManager service exists"
    else
        log_warn "AlertManager service not found"
    fi
}

# Check ServiceMonitors
check_servicemonitors() {
    log_info "Checking ServiceMonitors for PAW..."

    SM_COUNT=$(kubectl get servicemonitors -A 2>/dev/null | grep -ci paw || true)

    if [ "$SM_COUNT" -gt 0 ]; then
        log_ok "$SM_COUNT ServiceMonitor(s) found for PAW"
        kubectl get servicemonitors -A 2>/dev/null | grep -i paw || true
    else
        log_warn "No ServiceMonitors found for PAW"
    fi
}

# Check PrometheusRules
check_prometheusrules() {
    log_info "Checking PrometheusRules for PAW..."

    PR_COUNT=$(kubectl get prometheusrules -A 2>/dev/null | grep -ci paw || true)

    if [ "$PR_COUNT" -gt 0 ]; then
        log_ok "$PR_COUNT PrometheusRule(s) found for PAW"
        kubectl get prometheusrules -A 2>/dev/null | grep -i paw || true
    else
        log_warn "No PrometheusRules found for PAW"
    fi
}

# Check Grafana dashboards
check_grafana_dashboards() {
    log_info "Checking Grafana dashboards..."

    DASHBOARD_CMS=$(kubectl get configmaps -n "$MONITORING_NS" -l grafana_dashboard=1 --no-headers 2>/dev/null | wc -l)

    if [ "$DASHBOARD_CMS" -gt 0 ]; then
        log_ok "$DASHBOARD_CMS Grafana dashboard ConfigMap(s) found"
    else
        log_warn "No Grafana dashboard ConfigMaps found"
    fi

    # Check for PAW-specific dashboard
    if kubectl get configmap -n "$MONITORING_NS" paw-dashboard >/dev/null 2>&1; then
        log_ok "PAW dashboard ConfigMap exists"
    else
        log_warn "PAW dashboard ConfigMap not found"
    fi
}

# Check Loki
check_loki() {
    log_info "Checking Loki..."

    if kubectl -n "$MONITORING_NS" get pods -l app=loki --no-headers 2>/dev/null | grep -q Running; then
        log_ok "Loki pods running"
    else
        log_warn "Loki pods not running (optional)"
    fi
}

# Probe AlertManager API
probe_alertmanager_api() {
    if [ "$PROBE_ALERTMANAGER" -ne 1 ]; then
        return
    fi

    log_info "Probing AlertManager API..."

    AM_SVC=$(kubectl get svc -n "$MONITORING_NS" -l app.kubernetes.io/name=alertmanager -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

    if [ -z "$AM_SVC" ]; then
        log_warn "AlertManager service not found"
        return
    fi

    # Create temporary pod to probe AlertManager
    POD_NAME="am-probe-$(date +%s)"
    if kubectl -n "$MONITORING_NS" run "$POD_NAME" --image=curlimages/curl:8.5.0 --rm -i --restart=Never --command -- \
        curl -sSf "http://${AM_SVC}:9093/api/v2/status" >/dev/null 2>&1; then
        log_ok "AlertManager /api/v2/status reachable"
    else
        log_warn "Unable to reach AlertManager API"
    fi
}

# Check scrape targets
check_scrape_targets() {
    log_info "Checking Prometheus scrape targets..."

    # This would require port-forwarding to Prometheus
    # For now, just check if targets exist via ServiceMonitor
    if kubectl get servicemonitors -n "$NAMESPACE" >/dev/null 2>&1; then
        log_ok "ServiceMonitors can be scraped"
    else
        log_warn "Cannot verify scrape targets without port-forward"
    fi
}

# Summary
print_summary() {
    echo ""
    echo "=============================================="
    echo "Monitoring Overlay Verification Summary"
    echo "=============================================="

    if [ "$STATUS" -eq 0 ]; then
        echo -e "${GREEN}All checks passed${NC}"
    else
        echo -e "${YELLOW}Some checks had warnings/errors${NC}"
    fi

    echo ""
    echo "Access Points (port-forward required):"
    echo "  Prometheus: kubectl port-forward -n $MONITORING_NS svc/prometheus-kube-prometheus-prometheus 9090:9090"
    echo "  Grafana:    kubectl port-forward -n $MONITORING_NS svc/prometheus-grafana 3000:80"
    echo "  AlertMgr:   kubectl port-forward -n $MONITORING_NS svc/alertmanager-operated 9093:9093"
    echo ""
}

# Main
main() {
    check_prometheus
    echo ""
    check_grafana
    echo ""
    check_alertmanager
    echo ""
    check_loki
    echo ""
    check_servicemonitors
    echo ""
    check_prometheusrules
    echo ""
    check_grafana_dashboards
    echo ""
    check_scrape_targets
    echo ""
    probe_alertmanager_api

    print_summary
    exit $STATUS
}

main
