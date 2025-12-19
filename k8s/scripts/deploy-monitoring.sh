#!/bin/bash
# deploy-monitoring.sh - Deploy comprehensive monitoring stack for PAW
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
K8S_DIR="$PROJECT_ROOT/k8s"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

add_helm_repos() {
    log_info "Adding Helm repositories..."

    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts 2>/dev/null || true
    helm repo add grafana https://grafana.github.io/helm-charts 2>/dev/null || true
    helm repo update

    log_success "Helm repositories updated"
}

deploy_prometheus_stack() {
    log_info "Deploying Prometheus Stack (Prometheus, Grafana, AlertManager)..."

    # Create monitoring namespace if it doesn't exist
    kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -

    # Deploy kube-prometheus-stack
    helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
        --namespace monitoring \
        --set prometheus.prometheusSpec.retention=15d \
        --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.storageClassName=local-path \
        --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage=50Gi \
        --set prometheus.service.type=NodePort \
        --set prometheus.service.nodePort=31009 \
        --set grafana.service.type=NodePort \
        --set grafana.service.nodePort=31030 \
        --set grafana.adminPassword=paw-admin \
        --set alertmanager.service.type=NodePort \
        --set alertmanager.service.nodePort=31093 \
        --set alertmanager.alertmanagerSpec.storage.volumeClaimTemplate.spec.storageClassName=local-path \
        --set alertmanager.alertmanagerSpec.storage.volumeClaimTemplate.spec.resources.requests.storage=10Gi \
        --set "prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false" \
        --set "prometheus.prometheusSpec.podMonitorSelectorNilUsesHelmValues=false" \
        --wait --timeout 10m

    log_success "Prometheus Stack deployed"
}

deploy_loki() {
    log_info "Deploying Loki (Log Aggregation)..."

    helm upgrade --install loki grafana/loki-stack \
        --namespace monitoring \
        --set loki.persistence.enabled=true \
        --set loki.persistence.storageClassName=local-path \
        --set loki.persistence.size=50Gi \
        --set promtail.enabled=true \
        --set grafana.enabled=false \
        --wait --timeout 5m

    log_success "Loki deployed"
}

configure_grafana_datasources() {
    log_info "Configuring Grafana datasources..."

    # Create Loki datasource
    kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-loki-datasource
  namespace: monitoring
  labels:
    grafana_datasource: "1"
data:
  loki-datasource.yaml: |
    apiVersion: 1
    datasources:
      - name: Loki
        type: loki
        access: proxy
        url: http://loki:3100
        isDefault: false
        editable: true
EOF

    log_success "Grafana datasources configured"
}

create_paw_dashboard() {
    log_info "Creating PAW Grafana dashboard..."

    kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: paw-dashboard
  namespace: monitoring
  labels:
    grafana_dashboard: "1"
data:
  paw-overview.json: |
    {
      "annotations": {
        "list": []
      },
      "editable": true,
      "fiscalYearStartMonth": 0,
      "graphTooltip": 0,
      "id": null,
      "links": [],
      "liveNow": false,
      "panels": [
        {
          "datasource": {
            "type": "prometheus",
            "uid": "prometheus"
          },
          "fieldConfig": {
            "defaults": {
              "color": {
                "mode": "palette-classic"
              },
              "mappings": [],
              "thresholds": {
                "mode": "absolute",
                "steps": [
                  {
                    "color": "green",
                    "value": null
                  }
                ]
              },
              "unit": "none"
            }
          },
          "gridPos": {
            "h": 6,
            "w": 6,
            "x": 0,
            "y": 0
          },
          "id": 1,
          "options": {
            "colorMode": "value",
            "graphMode": "area",
            "justifyMode": "auto",
            "orientation": "auto",
            "reduceOptions": {
              "calcs": ["lastNotNull"],
              "fields": "",
              "values": false
            },
            "textMode": "auto"
          },
          "title": "Block Height",
          "type": "stat",
          "targets": [
            {
              "expr": "cometbft_consensus_height{job=~\".*paw.*\"}",
              "refId": "A"
            }
          ]
        },
        {
          "datasource": {
            "type": "prometheus",
            "uid": "prometheus"
          },
          "fieldConfig": {
            "defaults": {
              "color": {
                "mode": "palette-classic"
              },
              "mappings": [],
              "thresholds": {
                "mode": "absolute",
                "steps": [
                  {
                    "color": "green",
                    "value": null
                  }
                ]
              },
              "unit": "none"
            }
          },
          "gridPos": {
            "h": 6,
            "w": 6,
            "x": 6,
            "y": 0
          },
          "id": 2,
          "options": {
            "colorMode": "value",
            "graphMode": "area",
            "justifyMode": "auto",
            "orientation": "auto",
            "reduceOptions": {
              "calcs": ["lastNotNull"],
              "fields": "",
              "values": false
            },
            "textMode": "auto"
          },
          "title": "Connected Peers",
          "type": "stat",
          "targets": [
            {
              "expr": "cometbft_p2p_peers{job=~\".*paw.*\"}",
              "refId": "A"
            }
          ]
        },
        {
          "datasource": {
            "type": "prometheus",
            "uid": "prometheus"
          },
          "fieldConfig": {
            "defaults": {
              "color": {
                "mode": "palette-classic"
              },
              "mappings": [],
              "thresholds": {
                "mode": "absolute",
                "steps": [
                  {
                    "color": "green",
                    "value": null
                  }
                ]
              },
              "unit": "none"
            }
          },
          "gridPos": {
            "h": 6,
            "w": 6,
            "x": 12,
            "y": 0
          },
          "id": 3,
          "options": {
            "colorMode": "value",
            "graphMode": "area",
            "justifyMode": "auto",
            "orientation": "auto",
            "reduceOptions": {
              "calcs": ["lastNotNull"],
              "fields": "",
              "values": false
            },
            "textMode": "auto"
          },
          "title": "Validators",
          "type": "stat",
          "targets": [
            {
              "expr": "cometbft_consensus_validators{job=~\".*paw.*\"}",
              "refId": "A"
            }
          ]
        },
        {
          "datasource": {
            "type": "prometheus",
            "uid": "prometheus"
          },
          "fieldConfig": {
            "defaults": {
              "color": {
                "mode": "palette-classic"
              },
              "mappings": [],
              "thresholds": {
                "mode": "absolute",
                "steps": [
                  {
                    "color": "green",
                    "value": null
                  },
                  {
                    "color": "red",
                    "value": 1
                  }
                ]
              },
              "unit": "none"
            }
          },
          "gridPos": {
            "h": 6,
            "w": 6,
            "x": 18,
            "y": 0
          },
          "id": 4,
          "options": {
            "colorMode": "value",
            "graphMode": "area",
            "justifyMode": "auto",
            "orientation": "auto",
            "reduceOptions": {
              "calcs": ["lastNotNull"],
              "fields": "",
              "values": false
            },
            "textMode": "auto"
          },
          "title": "Syncing",
          "type": "stat",
          "targets": [
            {
              "expr": "cometbft_consensus_fast_syncing{job=~\".*paw.*\"}",
              "refId": "A"
            }
          ]
        }
      ],
      "refresh": "5s",
      "schemaVersion": 38,
      "style": "dark",
      "tags": ["paw", "blockchain"],
      "templating": {
        "list": []
      },
      "time": {
        "from": "now-1h",
        "to": "now"
      },
      "timepicker": {},
      "timezone": "",
      "title": "PAW Blockchain Overview",
      "uid": "paw-overview",
      "version": 1,
      "weekStart": ""
    }
EOF

    log_success "PAW dashboard created"
}

apply_prometheus_rules() {
    log_info "Applying Prometheus alerting rules..."

    kubectl apply -f - <<EOF
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: paw-alerts
  namespace: monitoring
  labels:
    release: prometheus
spec:
  groups:
    - name: paw.rules
      rules:
        - alert: PAWValidatorDown
          expr: up{job=~".*paw.*", app_kubernetes_io_component="validator"} == 0
          for: 2m
          labels:
            severity: critical
          annotations:
            summary: "PAW Validator is down"
            description: "Validator {{ \$labels.pod }} has been down for more than 2 minutes."

        - alert: PAWConsensusStalled
          expr: increase(cometbft_consensus_height{job=~".*paw.*"}[5m]) == 0
          for: 5m
          labels:
            severity: critical
          annotations:
            summary: "PAW consensus is stalled"
            description: "No new blocks produced in the last 5 minutes."

        - alert: PAWLowPeerCount
          expr: cometbft_p2p_peers{job=~".*paw.*"} < 2
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "PAW node has low peer count"
            description: "Node {{ \$labels.pod }} has only {{ \$value }} peers."

        - alert: PAWCatchingUp
          expr: cometbft_consensus_fast_syncing{job=~".*paw.*"} == 1
          for: 30m
          labels:
            severity: warning
          annotations:
            summary: "PAW node is still syncing"
            description: "Node {{ \$labels.pod }} has been syncing for over 30 minutes."

        - alert: PAWHighMemory
          expr: container_memory_usage_bytes{namespace="paw", container!=""} / container_spec_memory_limit_bytes{namespace="paw", container!=""} > 0.9
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "PAW container using high memory"
            description: "Container {{ \$labels.container }} in {{ \$labels.pod }} is using over 90% of memory limit."

        - alert: PAWDiskUsageHigh
          expr: kubelet_volume_stats_used_bytes{namespace="paw"} / kubelet_volume_stats_capacity_bytes{namespace="paw"} > 0.85
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "PAW PVC disk usage high"
            description: "PVC {{ \$labels.persistentvolumeclaim }} is using over 85% of capacity."
EOF

    log_success "Prometheus rules applied"
}

apply_service_monitor() {
    log_info "Applying ServiceMonitor for PAW..."

    kubectl apply -f "$K8S_DIR/validators/servicemonitor.yaml" 2>/dev/null || log_warn "ServiceMonitor not applied - CRD may not exist"

    log_success "ServiceMonitor applied"
}

verify_monitoring() {
    log_info "Verifying monitoring deployment..."

    echo ""
    echo "=== Monitoring Pods ==="
    kubectl get pods -n monitoring

    echo ""
    echo "=== Monitoring Services ==="
    kubectl get svc -n monitoring

    echo ""
    echo "=== PrometheusRules ==="
    kubectl get prometheusrules -n monitoring

    echo ""
    echo "=== ServiceMonitors ==="
    kubectl get servicemonitors -A
}

print_summary() {
    GRAFANA_PASS=$(kubectl get secret -n monitoring prometheus-grafana -o jsonpath="{.data.admin-password}" 2>/dev/null | base64 -d || echo "paw-admin")

    echo ""
    echo "=============================================="
    echo -e "${GREEN}Monitoring Stack Deployed${NC}"
    echo "=============================================="
    echo ""
    echo "Access Points:"
    echo "  Grafana:     http://localhost:31030"
    echo "  Prometheus:  http://localhost:31009"
    echo "  AlertManager: http://localhost:31093"
    echo ""
    echo "Grafana Credentials:"
    echo "  Username: admin"
    echo "  Password: ${GRAFANA_PASS}"
    echo ""
    echo "Dashboards:"
    echo "  - PAW Blockchain Overview"
    echo "  - Kubernetes / Compute Resources"
    echo "  - Node Exporter / Full"
    echo ""
}

main() {
    echo ""
    echo "=============================================="
    echo -e "${BLUE}PAW Monitoring Stack Deployment${NC}"
    echo "=============================================="
    echo ""

    add_helm_repos
    deploy_prometheus_stack
    deploy_loki
    configure_grafana_datasources
    create_paw_dashboard
    apply_prometheus_rules
    apply_service_monitor
    verify_monitoring
    print_summary
}

main
