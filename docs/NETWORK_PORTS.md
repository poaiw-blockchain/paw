# PAW Blockchain - Network Port Reference

**Version:** 1.0
**Last Updated:** 2025-12-07
**Audience:** Network Administrators, DevOps Engineers, Security Teams

---

## Table of Contents

1. [Port Overview](#port-overview)
2. [CometBFT Ports](#cometbft-ports)
3. [Application Ports](#application-ports)
4. [Monitoring Ports](#monitoring-ports)
5. [IBC Relayer Ports](#ibc-relayer-ports)
6. [Security Considerations](#security-considerations)
7. [Firewall Configuration Examples](#firewall-configuration-examples)
8. [Kubernetes Port Mappings](#kubernetes-port-mappings)
9. [Docker Port Mappings](#docker-port-mappings)
10. [Port Conflict Resolution](#port-conflict-resolution)

---

## Port Overview

### Port Matrix - All Services

| Port | Protocol | Service | Component | Public Access | Required | Notes |
|------|----------|---------|-----------|---------------|----------|-------|
| **26656** | TCP | P2P | CometBFT | Yes (Validators/Full Nodes) | ✅ Required | Peer-to-peer consensus communication |
| **26657** | TCP | RPC | CometBFT | No (Internal Only) | ✅ Required | JSON-RPC for queries and transactions |
| **26658** | TCP | ABCI | CometBFT | No (Internal Only) | ✅ Required | Application Blockchain Interface |
| **26660** | TCP | Prometheus | CometBFT | No (Monitoring Only) | ✅ Required | Metrics export for Prometheus |
| **1317** | TCP | REST API | Cosmos SDK | Optional (Public API) | Optional | HTTP REST API (LCD) |
| **9090** | TCP | gRPC | Cosmos SDK | Optional (Public API) | Optional | gRPC API for clients |
| **9091** | TCP | gRPC-Web | Cosmos SDK | Optional (Web Clients) | Optional | gRPC-Web gateway |
| **6060** | TCP | pprof | Go Runtime | No (Debug Only) | Optional | Go profiling endpoint (dev only) |

### Port Matrix - Monitoring Stack

| Port | Protocol | Service | Component | Public Access | Required | Notes |
|------|----------|---------|-----------|---------------|----------|-------|
| **9090** | TCP | Prometheus | Monitoring | No (Internal Only) | ✅ Required | Time-series metrics database |
| **9093** | TCP | AlertManager | Monitoring | No (Internal Only) | Optional | Alert routing and management |
| **3000** | TCP | Grafana | Monitoring | Yes (Dashboard UI) | ✅ Required | Metrics visualization |
| **3100** | TCP | Loki | Logging | No (Internal Only) | Optional | Log aggregation API |
| **9080** | TCP | Promtail | Logging | No (Internal Only) | Optional | Log collection agent |

### Port Matrix - External Services

| Port | Protocol | Service | Component | Public Access | Required | Notes |
|------|----------|---------|-----------|---------------|----------|-------|
| **80** | TCP | HTTP | Ingress/LB | Yes | Optional | HTTP (redirect to HTTPS) |
| **443** | TCP | HTTPS | Ingress/LB | Yes | ✅ Required | Secure API/RPC access |
| **5432** | TCP | PostgreSQL | Database | No (Internal Only) | Optional | Indexer/explorer database |
| **6379** | TCP | Redis | Cache | No (Internal Only) | Optional | Query result caching |

---

## CometBFT Ports

### 26656 - P2P Port (Tendermint P2P Protocol)

**Purpose:** Peer-to-peer communication for consensus and block propagation.

**Protocol:** TCP
**Direction:** Bidirectional (Inbound + Outbound)
**Required:** ✅ Yes (all nodes)
**Public Exposure:** Required for validators and full nodes participating in consensus

**Traffic Characteristics:**
- **Bandwidth:** Medium to High (blocks, transactions, consensus messages)
- **Connection Type:** Persistent connections to peers
- **Peer Count:** Typically 10-50 persistent peers + 20-100 dial peers
- **Data Rate:** 1-10 Mbps per peer (varies with network activity)

**Security Considerations:**
- **DDoS Protection:** Rate limiting required (max 50 inbound connections recommended)
- **Peer Discovery:** Uses P2P address book, can be seeded with trusted peers
- **Encryption:** All P2P traffic encrypted with Noise protocol (ed25519 keys)
- **Authentication:** Peer authentication via cryptographic handshake

**Firewall Rules:**
```bash
# Allow P2P from all (required for validators)
sudo ufw allow 26656/tcp comment 'PAW P2P'

# For private validators (sentry architecture), restrict to sentry nodes only
sudo ufw allow from SENTRY_NODE_IP to any port 26656 proto tcp
```

**Configuration:**
```toml
# config/config.toml
[p2p]
laddr = "tcp://0.0.0.0:26656"
persistent_peers = "node1@ip1:26656,node2@ip2:26656"
max_num_inbound_peers = 50
max_num_outbound_peers = 50
```

**Health Check:**
```bash
# Check P2P connectivity
curl -s http://localhost:26657/net_info | jq '.result.n_peers'

# List connected peers
curl -s http://localhost:26657/net_info | jq '.result.peers[].node_info.moniker'
```

---

### 26657 - RPC Port (CometBFT JSON-RPC)

**Purpose:** Query blockchain state, submit transactions, subscribe to events.

**Protocol:** HTTP/WebSocket over TCP
**Direction:** Inbound only
**Required:** ✅ Yes (always active)
**Public Exposure:** ⚠️ **INTERNAL ONLY** - Do NOT expose to public internet

**Endpoints:**
- `/health` - Node health status
- `/status` - Node sync status, chain info
- `/block` - Query blocks by height
- `/tx` - Query transactions by hash
- `/broadcast_tx_sync` - Submit transactions
- `/abci_query` - Query application state
- `/subscribe` - WebSocket event subscriptions

**Traffic Characteristics:**
- **Bandwidth:** Low to Medium (queries and tx broadcasts)
- **Connection Type:** Short-lived HTTP requests + persistent WebSocket
- **Request Rate:** Varies (100-1000 req/sec on public API nodes)

**Security Considerations:**
- **Authentication:** None by default - add nginx/authentication proxy
- **Rate Limiting:** Required for public-facing nodes (10-100 req/sec per IP)
- **CORS:** Disabled by default, enable only for trusted origins
- **Query Limits:** Configure max_open_connections to prevent DoS

**Firewall Rules:**
```bash
# INTERNAL ONLY - Do not expose publicly
# Allow from localhost only
sudo ufw allow from 127.0.0.1 to any port 26657 proto tcp

# For API nodes behind load balancer, allow from LB subnet only
sudo ufw allow from 10.0.0.0/16 to any port 26657 proto tcp
```

**Configuration:**
```toml
# config/config.toml
[rpc]
laddr = "tcp://127.0.0.1:26657"  # localhost only for validators
# laddr = "tcp://0.0.0.0:26657"  # all interfaces for API nodes (behind LB)
cors_allowed_origins = []
max_open_connections = 900
```

**Health Check:**
```bash
# Check RPC availability
curl -s http://localhost:26657/health

# Check sync status
curl -s http://localhost:26657/status | jq '.result.sync_info'
```

**Production Deployment:**
- Validators: Bind to localhost only (127.0.0.1:26657)
- API Nodes: Bind to all interfaces (0.0.0.0:26657) behind nginx/LB with:
  - Rate limiting (100 req/min per IP)
  - Authentication (API keys)
  - TLS termination
  - CORS restrictions
  - Request size limits

---

### 26658 - ABCI Port (Application Blockchain Interface)

**Purpose:** Internal communication between CometBFT consensus engine and Cosmos SDK application.

**Protocol:** Socket/gRPC over TCP
**Direction:** Loopback only
**Required:** ✅ Yes (critical for consensus)
**Public Exposure:** ❌ **Never** - Internal process communication only

**Traffic Characteristics:**
- **Bandwidth:** High (every block, every transaction)
- **Connection Type:** Persistent local socket
- **Data Rate:** 10-100 MB/sec during high load

**Security Considerations:**
- **Exposure:** Must NEVER be exposed to network
- **Binding:** Should only bind to localhost (127.0.0.1)
- **Firewall:** No firewall rules needed (localhost only)

**Configuration:**
```toml
# config/config.toml
proxy_app = "tcp://127.0.0.1:26658"
```

**Health Check:**
```bash
# Verify ABCI is listening locally only
sudo netstat -tlnp | grep 26658
# Should show: 127.0.0.1:26658 (not 0.0.0.0:26658)
```

**Troubleshooting:**
- If node fails to start, check ABCI port is not in use
- If performance degrades, check ABCI connection latency
- ABCI errors indicate application-level issues (check logs)

---

### 26660 - Prometheus Metrics Port

**Purpose:** Export CometBFT and application metrics for Prometheus monitoring.

**Protocol:** HTTP over TCP
**Direction:** Inbound only
**Required:** ✅ Yes (production monitoring required)
**Public Exposure:** ⚠️ **Monitoring subnet only** - Not public

**Metrics Exported:**
- `tendermint_consensus_height` - Current block height
- `tendermint_consensus_validators` - Total validators
- `tendermint_consensus_missing_validators` - Missing validators
- `tendermint_consensus_byzantine_validators` - Byzantine validators detected
- `tendermint_mempool_size` - Mempool transaction count
- `tendermint_p2p_peers` - Connected peer count
- Custom application metrics (DEX, Oracle, Compute modules)

**Traffic Characteristics:**
- **Bandwidth:** Very Low (<1 KB/sec)
- **Connection Type:** HTTP GET /metrics every 15-60 seconds
- **Data Size:** ~50-100 KB per scrape

**Security Considerations:**
- **Exposure:** Allow from Prometheus server IP only
- **Sensitive Data:** Metrics can reveal network topology and validator status
- **Authentication:** Add basic auth or IP whitelist

**Firewall Rules:**
```bash
# Allow from Prometheus server only
sudo ufw allow from PROMETHEUS_SERVER_IP to any port 26660 proto tcp

# In Kubernetes, use NetworkPolicy to restrict to monitoring namespace
```

**Configuration:**
```toml
# config/config.toml
[instrumentation]
prometheus = true
prometheus_listen_addr = ":26660"
namespace = "tendermint"
```

**Health Check:**
```bash
# Check metrics endpoint
curl -s http://localhost:26660/metrics | head -20

# Get current block height
curl -s http://localhost:26660/metrics | grep tendermint_consensus_height
```

**Prometheus Scrape Config:**
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'paw-validators'
    static_configs:
      - targets: ['validator1:26660', 'validator2:26660']
    scrape_interval: 15s
    scrape_timeout: 10s
```

---

## Application Ports

### 1317 - REST API Port (LCD - Light Client Daemon)

**Purpose:** HTTP REST API for querying blockchain state and submitting transactions.

**Protocol:** HTTP/HTTPS over TCP
**Direction:** Inbound only
**Required:** Optional (required for web clients)
**Public Exposure:** Optional (can be public for API nodes)

**API Endpoints:**
- `/cosmos/auth/v1beta1/accounts/{address}` - Account information
- `/cosmos/bank/v1beta1/balances/{address}` - Token balances
- `/cosmos/tx/v1beta1/txs` - Submit transactions
- `/paw/dex/v1/pools` - DEX liquidity pools
- `/paw/oracle/v1/prices` - Oracle price feeds
- `/paw/compute/v1/providers` - Compute providers

**Traffic Characteristics:**
- **Bandwidth:** Medium (varies with query complexity)
- **Connection Type:** Short-lived HTTP requests
- **Request Rate:** 10-1000 req/sec depending on deployment

**Security Considerations:**
- **Rate Limiting:** Required (100 req/min per IP recommended)
- **Authentication:** Add API keys for production
- **CORS:** Configure allowed origins
- **Input Validation:** Cosmos SDK handles, but add WAF for extra protection
- **SSL/TLS:** Terminate at load balancer

**Firewall Rules:**
```bash
# For public API nodes (behind load balancer)
# Allow from load balancer subnet only
sudo ufw allow from 10.0.0.0/16 to any port 1317 proto tcp

# Validators: Disable API entirely
# Set api.enable = false in app.toml
```

**Configuration:**
```toml
# config/app.toml
[api]
enable = true  # false for validators
swagger = true
address = "tcp://0.0.0.0:1317"
max-open-connections = 1000
rpc-read-timeout = 10
rpc-write-timeout = 10
enabled-unsafe-cors = false
```

**Health Check:**
```bash
# Check API availability
curl -s http://localhost:1317/cosmos/base/tendermint/v1beta1/node_info

# Check specific module
curl -s http://localhost:1317/paw/dex/v1/pools | jq
```

**Production Deployment:**
- Validators: Disable entirely (set `enable = false`)
- API Nodes: Enable with nginx reverse proxy:
  ```nginx
  upstream paw_api {
      least_conn;
      server api1:1317 max_fails=3 fail_timeout=30s;
      server api2:1317 max_fails=3 fail_timeout=30s;
      server api3:1317 max_fails=3 fail_timeout=30s;
  }

  server {
      listen 443 ssl http2;
      server_name api.paw.network;

      ssl_certificate /etc/letsencrypt/live/api.paw.network/fullchain.pem;
      ssl_certificate_key /etc/letsencrypt/live/api.paw.network/privkey.pem;

      location / {
          limit_req zone=api_limit burst=20 nodelay;
          proxy_pass http://paw_api;
          proxy_set_header X-Real-IP $remote_addr;
          proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      }
  }
  ```

---

### 9090 - gRPC API Port

**Purpose:** gRPC API for efficient client communication (used by wallets, SDKs).

**Protocol:** gRPC over HTTP/2 (TCP)
**Direction:** Inbound only
**Required:** Optional (required for gRPC clients)
**Public Exposure:** Optional (can be public for API nodes)

**gRPC Services:**
- `cosmos.auth.v1beta1.Query` - Account queries
- `cosmos.bank.v1beta1.Query` - Balance queries
- `cosmos.tx.v1beta1.Service` - Transaction broadcasting
- `paw.dex.v1.Query` - DEX queries
- `paw.oracle.v1.Query` - Oracle queries
- `paw.compute.v1.Query` - Compute queries

**Traffic Characteristics:**
- **Bandwidth:** Medium (more efficient than REST)
- **Connection Type:** Persistent HTTP/2 streams
- **Request Rate:** Higher throughput than REST (binary protocol)

**Security Considerations:**
- **TLS:** Mandatory for production (gRPC over TLS)
- **Authentication:** Use interceptors for API key validation
- **Rate Limiting:** Per-service rate limiting
- **Reflection:** Disable gRPC reflection in production

**Firewall Rules:**
```bash
# For public gRPC nodes (behind load balancer)
sudo ufw allow from 10.0.0.0/16 to any port 9090 proto tcp

# Validators: Disable gRPC or bind to localhost
```

**Configuration:**
```toml
# config/app.toml
[grpc]
enable = true  # false for validators
address = "0.0.0.0:9090"
max-recv-msg-size = 10485760  # 10 MB
max-send-msg-size = 2147483647  # 2 GB
```

**Health Check:**
```bash
# Check gRPC availability (requires grpcurl)
grpcurl -plaintext localhost:9090 list

# Query node info
grpcurl -plaintext localhost:9090 cosmos.base.tendermint.v1beta1.Service/GetNodeInfo
```

**Production Deployment:**
- Use nginx gRPC proxy with TLS:
  ```nginx
  upstream paw_grpc {
      server grpc1:9090;
      server grpc2:9090;
  }

  server {
      listen 443 ssl http2;
      server_name grpc.paw.network;

      ssl_certificate /etc/letsencrypt/live/grpc.paw.network/fullchain.pem;
      ssl_certificate_key /etc/letsencrypt/live/grpc.paw.network/privkey.pem;

      location / {
          grpc_pass grpc://paw_grpc;
          grpc_set_header X-Real-IP $remote_addr;
      }
  }
  ```

---

### 9091 - gRPC-Web Port

**Purpose:** gRPC-Web gateway for browser-based clients (JavaScript/TypeScript).

**Protocol:** gRPC-Web over HTTP/1.1
**Direction:** Inbound only
**Required:** Optional (required for web apps)
**Public Exposure:** Optional (can be public for API nodes)

**Configuration:**
```toml
# config/app.toml
[grpc-web]
enable = true
address = "0.0.0.0:9091"
enable-unsafe-cors = false
```

---

### 6060 - pprof Debug Port (Go Runtime Profiling)

**Purpose:** Go runtime profiling for performance debugging.

**Protocol:** HTTP over TCP
**Direction:** Inbound only
**Required:** ❌ No (development/debug only)
**Public Exposure:** ❌ **Never** - Development only

**Security Considerations:**
- **Exposure:** MUST be disabled in production
- **Information Leak:** Exposes memory, goroutines, CPU profiles
- **Binding:** Only enable on localhost during debugging

**Configuration:**
```bash
# Enable pprof in development
export PPROF_ADDR=localhost:6060
pawd start

# Disable in production (default)
unset PPROF_ADDR
```

**Usage:**
```bash
# View goroutines
go tool pprof http://localhost:6060/debug/pprof/goroutine

# CPU profile (30 seconds)
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Memory heap
go tool pprof http://localhost:6060/debug/pprof/heap
```

---

## Monitoring Ports

### 9090 - Prometheus Server (Note: Conflicts with gRPC)

**Port Conflict:** This port conflicts with Cosmos SDK gRPC. In practice:
- **Option 1:** Run Prometheus on different port (9091 or 9095)
- **Option 2:** Separate Prometheus on different node
- **Option 3:** Use non-standard gRPC port (9092)

**Configuration:**
```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'paw-validators'
    static_configs:
      - targets: ['validator1:26660', 'validator2:26660', 'validator3:26660']
```

---

### 9093 - AlertManager

**Purpose:** Alert routing, grouping, and notification.

**Protocol:** HTTP over TCP
**Public Exposure:** ❌ Internal only

**Configuration:**
```yaml
# alertmanager.yml
route:
  receiver: 'slack'
  group_by: ['alertname', 'cluster']
  group_wait: 10s
  group_interval: 5m
  repeat_interval: 3h

receivers:
  - name: 'slack'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/XXX'
        channel: '#paw-alerts'
```

---

### 3000 - Grafana Dashboard

**Purpose:** Metrics visualization and dashboarding.

**Protocol:** HTTP/HTTPS over TCP
**Public Exposure:** Yes (with authentication)

**Security Considerations:**
- **Authentication:** Mandatory (disable anonymous access)
- **TLS:** Use HTTPS with valid certificate
- **Session Security:** Enable secure cookies, CSRF protection
- **User Roles:** Use viewer/editor/admin roles appropriately

**Configuration:**
```ini
# grafana.ini
[server]
protocol = http
http_addr = 0.0.0.0
http_port = 3000
domain = grafana.paw.network
root_url = https://grafana.paw.network

[auth]
disable_login_form = false
disable_signout_menu = false

[auth.anonymous]
enabled = false
```

---

### 3100 - Loki (Log Aggregation)

**Purpose:** Log aggregation API for Grafana Loki.

**Protocol:** HTTP over TCP
**Public Exposure:** ❌ Internal only (Promtail clients)

**Configuration:**
```yaml
# loki-config.yaml
server:
  http_listen_port: 3100

ingester:
  lifecycler:
    ring:
      kvstore:
        store: inmemory
      replication_factor: 1
```

---

## IBC Relayer Ports

IBC relayers use the same RPC/gRPC ports as application clients:
- **26657** - RPC for querying IBC state
- **9090** - gRPC for IBC transactions

Relayers do NOT require separate ports, but DO require:
- Access to RPC on both source and destination chains
- Funded relayer wallet for gas fees
- Persistent connection to both chains

---

## Security Considerations

### Defense-in-Depth Strategy

```
┌─────────────────────────────────────────────────────────────────┐
│ Layer 1: Network Firewall (Cloud Provider / Hardware)          │
│ - Block all traffic except allowed ports                       │
│ - DDoS protection at network edge                              │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│ Layer 2: Host Firewall (iptables / ufw / firewalld)            │
│ - Per-port rules with source IP restrictions                   │
│ - Rate limiting (fail2ban, iptables recent module)             │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│ Layer 3: Application Security (CometBFT / Cosmos SDK)          │
│ - Connection limits (max_num_inbound_peers)                    │
│ - Request size limits (max-body-bytes)                         │
│ - Authentication (API keys, JWT tokens)                        │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│ Layer 4: Monitoring & Alerting                                 │
│ - Anomaly detection (unusual traffic patterns)                 │
│ - Connection monitoring (netstat, ss)                          │
│ - Alert on suspicious activity                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Principle of Least Privilege

**Validator Nodes:**
- ✅ Open: 26656 (P2P to internet or sentry nodes only)
- ✅ Open: 26660 (Metrics to Prometheus server only)
- ❌ Closed: 26657 (RPC - localhost only)
- ❌ Closed: 1317, 9090, 9091 (APIs - disabled)

**Sentry Nodes (if using sentry architecture):**
- ✅ Open: 26656 (P2P to internet)
- ✅ Open: 26657 (RPC to API nodes only)
- ✅ Open: 26660 (Metrics to Prometheus only)
- ❌ Closed: 1317, 9090, 9091 (APIs - disabled, use separate API nodes)

**API Nodes:**
- ✅ Open: 26656 (P2P to sentry nodes only)
- ✅ Open: 1317, 9090, 9091 (APIs to load balancer only)
- ✅ Open: 26660 (Metrics to Prometheus only)
- ❌ Closed: 26657 (RPC bound to localhost, accessed via load balancer)

**Monitoring Nodes:**
- ✅ Open: 3000 (Grafana to authenticated users via HTTPS)
- ❌ Closed: 9090, 9093, 3100 (Prometheus, AlertManager, Loki - internal only)

---

## Firewall Configuration Examples

### Ubuntu/Debian (ufw)

```bash
#!/bin/bash
# PAW Validator Node Firewall Configuration

# Reset firewall
sudo ufw --force reset

# Default policies
sudo ufw default deny incoming
sudo ufw default allow outgoing

# Allow SSH (adjust port if using non-standard)
sudo ufw allow 22/tcp comment 'SSH'

# Allow P2P for consensus
sudo ufw allow 26656/tcp comment 'PAW P2P'

# Allow Prometheus metrics from monitoring server
sudo ufw allow from PROMETHEUS_IP to any port 26660 proto tcp comment 'Prometheus Metrics'

# Enable firewall
sudo ufw --force enable

# Verify rules
sudo ufw status verbose
```

### CentOS/RHEL (firewalld)

```bash
#!/bin/bash
# PAW Validator Node Firewall Configuration

# Enable firewalld
sudo systemctl enable firewalld
sudo systemctl start firewalld

# Set default zone to drop
sudo firewall-cmd --set-default-zone=drop

# Allow SSH
sudo firewall-cmd --permanent --add-service=ssh

# Allow P2P
sudo firewall-cmd --permanent --add-port=26656/tcp

# Allow metrics from monitoring subnet
sudo firewall-cmd --permanent --add-rich-rule='rule family="ipv4" source address="PROMETHEUS_SUBNET/24" port protocol="tcp" port="26660" accept'

# Reload firewall
sudo firewall-cmd --reload

# Verify rules
sudo firewall-cmd --list-all
```

### Advanced iptables (Rate Limiting)

```bash
#!/bin/bash
# PAW Node with DDoS Protection

# Flush existing rules
iptables -F
iptables -X

# Default policies
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT ACCEPT

# Allow loopback
iptables -A INPUT -i lo -j ACCEPT

# Allow established connections
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Allow SSH (rate limited)
iptables -A INPUT -p tcp --dport 22 -m state --state NEW -m recent --set
iptables -A INPUT -p tcp --dport 22 -m state --state NEW -m recent --update --seconds 60 --hitcount 4 -j DROP
iptables -A INPUT -p tcp --dport 22 -j ACCEPT

# Allow P2P with connection limits
iptables -A INPUT -p tcp --dport 26656 -m connlimit --connlimit-above 50 --connlimit-mask 32 -j DROP
iptables -A INPUT -p tcp --dport 26656 -j ACCEPT

# Allow metrics from specific IP
iptables -A INPUT -p tcp -s PROMETHEUS_IP --dport 26660 -j ACCEPT

# Log dropped packets (optional)
iptables -A INPUT -m limit --limit 5/min -j LOG --log-prefix "iptables-dropped: " --log-level 7

# Save rules
iptables-save > /etc/iptables/rules.v4
```

---

## Kubernetes Port Mappings

### Service Definitions

```yaml
# Validator P2P Service (NodePort)
apiVersion: v1
kind: Service
metadata:
  name: paw-validator-p2p
  namespace: paw-blockchain
spec:
  type: NodePort
  selector:
    app: paw
    component: validator
  ports:
    - name: p2p
      port: 26656        # ClusterIP port
      targetPort: 26656  # Container port
      nodePort: 30656    # External port on node
      protocol: TCP
```

### Port Mapping Matrix

| Service | Type | ClusterIP Port | Container Port | NodePort/LB Port | Notes |
|---------|------|----------------|----------------|------------------|-------|
| validator-p2p | NodePort | 26656 | 26656 | 30656 | P2P consensus |
| node-rpc | LoadBalancer | 26657 | 26657 | 26657 | RPC API |
| node-grpc | LoadBalancer | 9090 | 9090 | 9090 | gRPC API |
| node-api | LoadBalancer | 1317 | 1317 | 1317 | REST API |
| metrics | ClusterIP | 26660 | 26660 | - | Prometheus metrics |

### Network Policies

```yaml
# Restrict RPC access to monitoring namespace only
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: paw-validator-network-policy
  namespace: paw-blockchain
spec:
  podSelector:
    matchLabels:
      component: validator
  policyTypes:
    - Ingress
  ingress:
    # Allow P2P from anywhere
    - from: []
      ports:
        - protocol: TCP
          port: 26656
    # Allow metrics from monitoring namespace only
    - from:
        - namespaceSelector:
            matchLabels:
              name: monitoring
      ports:
        - protocol: TCP
          port: 26660
```

---

## Docker Port Mappings

### Docker Compose Port Mapping

```yaml
# docker-compose.yml
version: '3.8'

services:
  paw-validator:
    image: paw:latest
    ports:
      # P2P (published to host)
      - "26656:26656"
      # RPC (localhost only)
      - "127.0.0.1:26657:26657"
      # Metrics (internal network only)
      - "26660:26660"
    networks:
      - paw-network
      - monitoring

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "127.0.0.1:9090:9090"
    networks:
      - monitoring

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    networks:
      - monitoring

networks:
  paw-network:
    driver: bridge
  monitoring:
    driver: bridge
```

### Docker CLI Port Mapping

```bash
# Validator node
docker run -d \
  --name paw-validator \
  -p 26656:26656 \
  -p 127.0.0.1:26657:26657 \
  -p 26660:26660 \
  -v paw-data:/root/.paw \
  paw:latest

# API node
docker run -d \
  --name paw-api \
  -p 26656:26656 \
  -p 127.0.0.1:26657:26657 \
  -p 1317:1317 \
  -p 9090:9090 \
  -p 9091:9091 \
  -v paw-data:/root/.paw \
  paw:latest
```

---

## Port Conflict Resolution

### Common Conflicts

| Port | PAW Service | Common Conflict | Resolution |
|------|-------------|-----------------|------------|
| 9090 | gRPC | Prometheus Server | Run Prometheus on 9095 or separate node |
| 1317 | REST API | Other APIs | Use 1318 or nginx reverse proxy |
| 3000 | Grafana | Development servers | Change Grafana to 3001 |
| 6060 | pprof | Other Go apps | Change to 6061 or disable |
| 26656 | P2P | Multiple chains | Use different ports per chain |

### Resolving Prometheus/gRPC Conflict

**Option 1: Change Prometheus port**
```yaml
# prometheus.yml
global:
  scrape_interval: 15s

# Use port 9095 instead of 9090
```

```bash
# Run Prometheus on alternate port
prometheus --config.file=/etc/prometheus/prometheus.yml --web.listen-address=:9095
```

**Option 2: Change gRPC port**
```toml
# config/app.toml
[grpc]
address = "0.0.0.0:9092"  # Use 9092 instead of 9090
```

**Option 3: Separate nodes (Recommended for production)**
- Run Prometheus on dedicated monitoring node
- Run gRPC on API nodes
- No conflict

### Running Multiple Chains on Same Host

```bash
# Chain 1 (PAW testnet)
pawd start --home ~/.paw-testnet \
  --p2p.laddr tcp://0.0.0.0:26656 \
  --rpc.laddr tcp://127.0.0.1:26657 \
  --grpc.address 0.0.0.0:9090 \
  --api.address tcp://0.0.0.0:1317

# Chain 2 (PAW mainnet)
pawd start --home ~/.paw-mainnet \
  --p2p.laddr tcp://0.0.0.0:27656 \
  --rpc.laddr tcp://127.0.0.1:27657 \
  --grpc.address 0.0.0.0:9092 \
  --api.address tcp://0.0.0.0:1318
```

---

## Diagnostic Commands

### Check Which Ports Are Listening

```bash
# Using netstat
sudo netstat -tlnp | grep -E "26656|26657|26660|1317|9090"

# Using ss (modern replacement for netstat)
sudo ss -tlnp | grep -E "26656|26657|26660|1317|9090"

# Using lsof
sudo lsof -i -P -n | grep LISTEN | grep -E "26656|26657|26660|1317|9090"
```

### Check Firewall Rules

```bash
# ufw
sudo ufw status numbered

# firewalld
sudo firewall-cmd --list-all

# iptables
sudo iptables -L -n -v --line-numbers
```

### Test Port Connectivity

```bash
# Test from local machine
nc -zv localhost 26657

# Test from remote machine
nc -zv validator.paw.network 26656

# Test with timeout
timeout 5 bash -c '</dev/tcp/validator.paw.network/26656' && echo "Port open" || echo "Port closed"
```

### Monitor Connection Counts

```bash
# Count connections per port
netstat -an | grep -E "26656|26657" | awk '{print $4}' | sort | uniq -c

# Monitor in real-time
watch -n 1 'netstat -an | grep -E "26656|26657" | wc -l'

# Show established connections
netstat -anp | grep ESTABLISHED | grep -E "26656|26657"
```

---

## Summary

### Production Validator Quick Reference

```bash
# Minimal secure configuration for validators
Exposed Ports:
  - 26656/tcp (P2P) → Internet (or sentry nodes only in sentry architecture)
  - 26660/tcp (Metrics) → Prometheus server IP only

Firewall Rules:
  sudo ufw allow 26656/tcp
  sudo ufw allow from PROMETHEUS_IP to any port 26660 proto tcp
  sudo ufw default deny incoming
  sudo ufw enable

Configuration:
  config/config.toml:
    [rpc]
    laddr = "tcp://127.0.0.1:26657"  # localhost only
    [p2p]
    laddr = "tcp://0.0.0.0:26656"    # all interfaces

  config/app.toml:
    [api]
    enable = false                   # disable REST API
    [grpc]
    enable = false                   # disable gRPC
```

### Production API Node Quick Reference

```bash
# API nodes behind load balancer
Exposed Ports:
  - 26656/tcp (P2P) → Sentry nodes only
  - 26657/tcp (RPC) → Load balancer subnet
  - 1317/tcp (REST API) → Load balancer subnet
  - 9090/tcp (gRPC) → Load balancer subnet
  - 26660/tcp (Metrics) → Prometheus server

Configuration:
  config/app.toml:
    [api]
    enable = true
    address = "tcp://0.0.0.0:1317"
    [grpc]
    enable = true
    address = "0.0.0.0:9090"
```

---

**For additional support:**
- See [DEPLOYMENT_QUICKSTART.md](DEPLOYMENT_QUICKSTART.md) for initial setup
- See [VALIDATOR_OPERATOR_GUIDE.md](VALIDATOR_OPERATOR_GUIDE.md) for validator-specific configuration
- See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common port-related issues
