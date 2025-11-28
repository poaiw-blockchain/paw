# Validator Monitoring

Set up comprehensive monitoring for your PAW validator.

## Prometheus Setup

```bash
# Install Prometheus
wget https://github.com/prometheus/prometheus/releases/download/v2.45.0/prometheus-2.45.0.linux-amd64.tar.gz
tar xvfz prometheus-*.tar.gz
cd prometheus-*

# Configure
cat > prometheus.yml << EOF
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'paw-validator'
    static_configs:
      - targets: ['localhost:26660']
EOF

# Start Prometheus
./prometheus --config.file=prometheus.yml
```

## Grafana Dashboards

```bash
# Install Grafana
sudo apt install -y software-properties-common
sudo add-apt-repository "deb https://packages.grafana.com/oss/deb stable main"
wget -q -O - https://packages.grafana.com/gpg.key | sudo apt-key add -
sudo apt update
sudo apt install grafana

# Start Grafana
sudo systemctl start grafana-server
sudo systemctl enable grafana-server

# Access at http://localhost:3000
# Default login: admin/admin
```

## Key Metrics

- Block height
- Validator uptime
- Missed blocks
- Commission earned
- Delegations
- Memory usage
- Disk I/O

## Alerts

Set up alerts for:
- Node offline >5 minutes
- Missed blocks >10
- Disk space <20%
- High memory usage
- Sync status issues

---

**Previous:** [Security](/validator/security) | **Next:** [Troubleshooting](/validator/troubleshooting) →
