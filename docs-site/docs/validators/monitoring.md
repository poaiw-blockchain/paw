# Validator Monitoring

Monitor your PAW validator node to ensure uptime and optimal performance.

## Why Monitor?

Proper monitoring helps you:
- Detect issues before they cause downtime
- Avoid slashing for missed blocks
- Optimize node performance
- Track validator metrics and earnings
- Receive alerts for critical events

## Monitoring Stack

Recommended monitoring stack:
- **Prometheus**: Metrics collection and storage
- **Grafana**: Visualization and dashboards
- **Alertmanager**: Alert notifications
- **Node Exporter**: System metrics
- **Cosmos Exporter**: Validator-specific metrics

## Quick Setup

### 1. Install Prometheus

```bash
# Create user
sudo useradd --no-create-home --shell /bin/false prometheus

# Download Prometheus
cd /tmp
wget https://github.com/prometheus/prometheus/releases/download/v2.45.0/prometheus-2.45.0.linux-amd64.tar.gz
tar xvf prometheus-2.45.0.linux-amd64.tar.gz
cd prometheus-2.45.0.linux-amd64

# Install binaries
sudo cp prometheus /usr/local/bin/
sudo cp promtool /usr/local/bin/
sudo chown prometheus:prometheus /usr/local/bin/prometheus
sudo chown prometheus:prometheus /usr/local/bin/promtool

# Create directories
sudo mkdir -p /etc/prometheus /var/lib/prometheus
sudo chown prometheus:prometheus /etc/prometheus /var/lib/prometheus

# Copy configuration
sudo cp -r consoles /etc/prometheus
sudo cp -r console_libraries /etc/prometheus
sudo chown -R prometheus:prometheus /etc/prometheus/consoles /etc/prometheus/console_libraries
```

### 2. Configure Prometheus

Create `/etc/prometheus/prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  # PAW node metrics
  - job_name: 'paw-node'
    static_configs:
      - targets: ['localhost:26660']
        labels:
          instance: 'validator-1'

  # System metrics
  - job_name: 'node-exporter'
    static_configs:
      - targets: ['localhost:9100']

  # Cosmos-specific metrics
  - job_name: 'cosmos-exporter'
    static_configs:
      - targets: ['localhost:9300']
```

### 3. Create Prometheus Service

```bash
sudo tee /etc/systemd/system/prometheus.service > /dev/null <<EOF
[Unit]
Description=Prometheus
Wants=network-online.target
After=network-online.target

[Service]
User=prometheus
Group=prometheus
Type=simple
ExecStart=/usr/local/bin/prometheus \
  --config.file /etc/prometheus/prometheus.yml \
  --storage.tsdb.path /var/lib/prometheus/ \
  --web.console.templates=/etc/prometheus/consoles \
  --web.console.libraries=/etc/prometheus/console_libraries

[Install]
WantedBy=multi-user.target
EOF

# Start Prometheus
sudo systemctl daemon-reload
sudo systemctl enable prometheus
sudo systemctl start prometheus
```

Verify: `http://localhost:9090`

### 4. Install Node Exporter

```bash
# Download Node Exporter
cd /tmp
wget https://github.com/prometheus/node_exporter/releases/download/v1.6.1/node_exporter-1.6.1.linux-amd64.tar.gz
tar xvf node_exporter-1.6.1.linux-amd64.tar.gz
cd node_exporter-1.6.1.linux-amd64

# Install binary
sudo cp node_exporter /usr/local/bin/
sudo chown prometheus:prometheus /usr/local/bin/node_exporter

# Create service
sudo tee /etc/systemd/system/node-exporter.service > /dev/null <<EOF
[Unit]
Description=Node Exporter
Wants=network-online.target
After=network-online.target

[Service]
User=prometheus
Group=prometheus
Type=simple
ExecStart=/usr/local/bin/node_exporter

[Install]
WantedBy=multi-user.target
EOF

# Start service
sudo systemctl daemon-reload
sudo systemctl enable node-exporter
sudo systemctl start node-exporter
```

### 5. Install Cosmos Exporter

```bash
# Install Go if not already installed
go install github.com/solarlabsteam/cosmos-exporter@latest

# Create service
sudo tee /etc/systemd/system/cosmos-exporter.service > /dev/null <<EOF
[Unit]
Description=Cosmos Exporter
After=network-online.target

[Service]
User=$USER
ExecStart=$HOME/go/bin/cosmos-exporter \
  --bech-prefix paw \
  --denom upaw \
  --rpc-address http://localhost:26657 \
  --grpc-address localhost:9090 \
  --listen-address :9300
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF

# Start service
sudo systemctl daemon-reload
sudo systemctl enable cosmos-exporter
sudo systemctl start cosmos-exporter
```

### 6. Install Grafana

```bash
# Add Grafana repository
sudo apt-get install -y software-properties-common
sudo add-apt-repository "deb https://packages.grafana.com/oss/deb stable main"
wget -q -O - https://packages.grafana.com/gpg.key | sudo apt-key add -

# Install Grafana
sudo apt-get update
sudo apt-get install -y grafana

# Start Grafana
sudo systemctl enable grafana-server
sudo systemctl start grafana-server
```

Access Grafana at `http://localhost:3000` (default credentials: admin/admin)

### 7. Configure Grafana

1. Add Prometheus data source:
   - Go to Configuration → Data Sources
   - Click "Add data source"
   - Select "Prometheus"
   - URL: `http://localhost:9090`
   - Click "Save & Test"

2. Import dashboard:
   - Go to Dashboards → Import
   - Upload JSON from `monitoring/grafana-dashboard.json`
   - Or use dashboard ID: 11036 (Cosmos Validator Dashboard)

## Key Metrics to Monitor

### Node Health

```promql
# Node is syncing
tendermint_consensus_latest_block_height - tendermint_consensus_height > 10

# Peer connections
tendermint_p2p_peers < 5

# Memory usage
process_resident_memory_bytes / 1024 / 1024 / 1024
```

### Validator Performance

```promql
# Missed blocks (last hour)
increase(cosmos_validators_missed_blocks[1h])

# Signing percentage
cosmos_validator_rank

# Voting power
cosmos_validator_voting_power

# Commission rate
cosmos_validator_commission
```

### System Resources

```promql
# CPU usage
100 - (avg by (instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)

# Memory usage
(node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes * 100

# Disk usage
(node_filesystem_size_bytes - node_filesystem_free_bytes) / node_filesystem_size_bytes * 100

# Network traffic
irate(node_network_receive_bytes_total[5m])
irate(node_network_transmit_bytes_total[5m])
```

### Blockchain Metrics

```promql
# Block time
rate(tendermint_consensus_height[1m])

# Transaction throughput
rate(tendermint_consensus_num_txs[1m])

# Mempool size
tendermint_mempool_size
```

## Alerting

### Install Alertmanager

```bash
# Download Alertmanager
cd /tmp
wget https://github.com/prometheus/alertmanager/releases/download/v0.26.0/alertmanager-0.26.0.linux-amd64.tar.gz
tar xvf alertmanager-0.26.0.linux-amd64.tar.gz
cd alertmanager-0.26.0.linux-amd64

# Install binary
sudo cp alertmanager /usr/local/bin/
sudo cp amtool /usr/local/bin/
sudo chown prometheus:prometheus /usr/local/bin/alertmanager /usr/local/bin/amtool

# Create directory
sudo mkdir -p /etc/alertmanager
sudo chown prometheus:prometheus /etc/alertmanager
```

### Configure Alertmanager

Create `/etc/alertmanager/alertmanager.yml`:

```yaml
global:
  resolve_timeout: 5m

route:
  receiver: 'default'
  group_by: ['alertname', 'cluster']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h

receivers:
  - name: 'default'
    email_configs:
      - to: 'your-email@example.com'
        from: 'alertmanager@paw-validator.com'
        smarthost: 'smtp.gmail.com:587'
        auth_username: 'your-email@example.com'
        auth_password: 'your-app-password'
    webhook_configs:
      - url: 'https://discord.com/api/webhooks/YOUR_WEBHOOK_URL'
```

### Create Alert Rules

Create `/etc/prometheus/alerts.yml`:

```yaml
groups:
  - name: validator_alerts
    interval: 30s
    rules:
      # Node down
      - alert: NodeDown
        expr: up{job="paw-node"} == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "PAW node is down"
          description: "Node {{ $labels.instance }} has been down for more than 2 minutes"

      # Not syncing
      - alert: NodeNotSyncing
        expr: tendermint_consensus_latest_block_height - tendermint_consensus_height > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Node not syncing"
          description: "Node is {{ $value }} blocks behind"

      # Missed blocks
      - alert: MissedBlocks
        expr: increase(cosmos_validators_missed_blocks[10m]) > 5
        labels:
          severity: warning
        annotations:
          summary: "Validator missing blocks"
          description: "Validator missed {{ $value }} blocks in last 10 minutes"

      # Low peer count
      - alert: LowPeerCount
        expr: tendermint_p2p_peers < 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Low peer count"
          description: "Only {{ $value }} peers connected"

      # High memory usage
      - alert: HighMemoryUsage
        expr: (node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes * 100 > 90
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage"
          description: "Memory usage is {{ $value }}%"

      # Disk space low
      - alert: DiskSpaceLow
        expr: (node_filesystem_size_bytes - node_filesystem_free_bytes) / node_filesystem_size_bytes * 100 > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Disk space low"
          description: "Disk usage is {{ $value }}%"
```

Update Prometheus config to include alerts:

```yaml
# Add to /etc/prometheus/prometheus.yml
rule_files:
  - "alerts.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['localhost:9093']
```

### Start Alertmanager

```bash
sudo tee /etc/systemd/system/alertmanager.service > /dev/null <<EOF
[Unit]
Description=Alertmanager
Wants=network-online.target
After=network-online.target

[Service]
User=prometheus
Group=prometheus
Type=simple
ExecStart=/usr/local/bin/alertmanager \
  --config.file=/etc/alertmanager/alertmanager.yml \
  --storage.path=/var/lib/alertmanager/

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable alertmanager
sudo systemctl start alertmanager
sudo systemctl restart prometheus
```

## Useful Queries

### Check Validator Status

```bash
# Via CLI
pawd query staking validator $(pawd keys show validator --bech val -a)

# Via API
curl http://localhost:1317/cosmos/staking/v1beta1/validators/$(pawd keys show validator --bech val -a)

# Prometheus query
cosmos_validator_status{moniker="your-moniker"}
```

### Monitor Signing

```bash
# Get signing info
pawd query slashing signing-info $(pawd tendermint show-validator)

# Check missed block counter
pawd query slashing signing-info $(pawd tendermint show-validator) | grep missed
```

### Track Rewards

```bash
# Query rewards
pawd query distribution validator-outstanding-rewards $(pawd keys show validator --bech val -a)

# Query commission
pawd query distribution commission $(pawd keys show validator --bech val -a)
```

## Remote Monitoring

### Expose Metrics Securely

Use a reverse proxy with authentication:

```nginx
# /etc/nginx/sites-available/metrics
server {
    listen 443 ssl;
    server_name metrics.your-validator.com;

    ssl_certificate /etc/letsencrypt/live/your-validator.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-validator.com/privkey.pem;

    location /prometheus/ {
        auth_basic "Prometheus";
        auth_basic_user_file /etc/nginx/.htpasswd;
        proxy_pass http://localhost:9090/;
    }

    location /grafana/ {
        proxy_pass http://localhost:3000/;
    }
}
```

### Monitoring as a Service

Consider using:
- **PANIC**: Cosmos validator monitoring tool
- **Tenderduty**: Telegram alerts for validators
- **Uptime Kuma**: Self-hosted monitoring

## Best Practices

1. **Set up alerts BEFORE going live** - Don't wait for issues
2. **Monitor multiple metrics** - CPU, memory, disk, network, validator performance
3. **Test alerts** - Verify you receive notifications
4. **Keep historical data** - Useful for debugging and analysis
5. **Secure your monitoring stack** - Use authentication and HTTPS
6. **Monitor from multiple locations** - Detect network issues
7. **Set up redundancy** - Multiple alerting channels (email, Discord, SMS)
8. **Regular maintenance** - Update monitoring tools and dashboards

## Troubleshooting

### Prometheus Not Scraping

Check targets: `http://localhost:9090/targets`

Verify PAW node exposes metrics:
```bash
curl http://localhost:26660/metrics
```

Enable metrics in `config.toml`:
```toml
prometheus = true
prometheus_listen_addr = ":26660"
```

### Grafana Dashboard Empty

1. Check Prometheus data source is connected
2. Verify metrics are being collected: `http://localhost:9090/graph`
3. Check time range in Grafana (top right)
4. Verify dashboard queries match your metric names

### Missing Alerts

1. Check Alertmanager is running: `sudo systemctl status alertmanager`
2. Verify alert rules: `http://localhost:9090/alerts`
3. Check Alertmanager config: `http://localhost:9093`
4. Test email/webhook configuration

## Next Steps

- [Validator Setup Guide](setup.md)
- [PANIC Monitoring](https://github.com/SimplyVC/panic)
- [Tenderduty](https://github.com/blockpane/tenderduty)
- [Cosmos Validator Best Practices](https://docs.cosmos.network/main/validators/validator-faq)
