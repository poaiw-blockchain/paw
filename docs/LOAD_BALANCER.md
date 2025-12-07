# PAW Blockchain Load Balancer Configuration

**Version:** 1.0.0
**Last Updated:** 2025-12-07
**Audience:** DevOps Engineers, SREs, Infrastructure Architects

---

## Table of Contents

1. [Overview](#overview)
2. [RPC Load Balancing Requirements](#rpc-load-balancing-requirements)
3. [Nginx Configuration](#nginx-configuration)
4. [HAProxy Configuration](#haproxy-configuration)
5. [Cloud Load Balancers](#cloud-load-balancers)
6. [Geo-Distributed Deployment](#geo-distributed-deployment)
7. [Health Checks](#health-checks)
8. [SSL/TLS Termination](#ssltls-termination)
9. [Rate Limiting and DDoS Protection](#rate-limiting-and-ddos-protection)
10. [Monitoring and Metrics](#monitoring-and-metrics)
11. [Troubleshooting](#troubleshooting)

---

## Overview

PAW blockchain nodes expose multiple endpoints (RPC, API, gRPC) that may require load balancing for high availability, horizontal scaling, and geographic distribution. This guide covers production-grade load balancer configurations for various scenarios.

### Why Load Balance PAW Nodes?

**Benefits:**
- **High Availability:** Automatic failover if node becomes unavailable
- **Horizontal Scaling:** Distribute load across multiple nodes
- **Geographic Distribution:** Route users to nearest node (reduced latency)
- **DDoS Protection:** Rate limiting and connection pooling
- **SSL Termination:** Centralized certificate management

**Challenges:**
- **State Synchronization:** All nodes must have consistent blockchain state
- **WebSocket Support:** Some clients require WebSocket connections
- **Sticky Sessions:** Certain operations require session affinity
- **Health Monitoring:** Detect lagging or unhealthy nodes

---

## RPC Load Balancing Requirements

### Critical Requirements

1. **Sticky Sessions (Session Affinity)**
   - **Why:** Some RPC methods require multiple sequential requests (e.g., transaction broadcasting + query)
   - **Implementation:** IP hash or cookie-based affinity
   - **Duration:** At least 5 minutes (typical transaction finalization time)

2. **WebSocket Support**
   - **Why:** Event subscription endpoints require persistent WebSocket connections
   - **Endpoints:** `/websocket`, `/subscribe`
   - **Configuration:** Upgrade HTTP to WebSocket, long connection timeouts

3. **Health Checks**
   - **Method:** Query `/status` endpoint
   - **Validation:** Check `sync_info.catching_up == false`
   - **Frequency:** Every 10-30 seconds
   - **Failure Threshold:** 3 consecutive failures

4. **Connection Pooling**
   - **Why:** Blockchain RPC can handle limited concurrent connections
   - **Recommendation:** 100-500 connections per backend node
   - **Timeout:** 60-120 seconds idle timeout

### Endpoints to Load Balance

| Endpoint | Port | Protocol | Sticky Sessions | WebSocket | Purpose |
|----------|------|----------|-----------------|-----------|---------|
| RPC | 26657 | HTTP/WS | Yes | Yes | CometBFT RPC |
| REST API | 1317 | HTTP | No | No | Cosmos SDK REST |
| gRPC | 9090 | HTTP/2 | No | No | gRPC queries |
| gRPC-Web | 9091 | HTTP | No | No | Browser gRPC |
| Metrics | 26660 | HTTP | No | No | Prometheus (internal only) |

**Security Note:** Only RPC (26657) and REST API (1317) should be publicly exposed. gRPC endpoints are typically for backend services only.

---

## Nginx Configuration

Nginx is a high-performance web server and reverse proxy ideal for HTTP/HTTPS load balancing.

### Basic RPC Load Balancing

**File:** `/etc/nginx/sites-available/paw-rpc`

```nginx
# Upstream pool of PAW RPC nodes
upstream paw_rpc_backend {
    # IP hash ensures sticky sessions
    ip_hash;

    # Backend nodes (adjust IPs/ports to your deployment)
    server 10.0.1.10:26657 max_fails=3 fail_timeout=30s;
    server 10.0.1.11:26657 max_fails=3 fail_timeout=30s;
    server 10.0.1.12:26657 max_fails=3 fail_timeout=30s;

    # Optional: weights for unequal capacity
    # server 10.0.1.10:26657 weight=2 max_fails=3 fail_timeout=30s;

    # Connection limits
    keepalive 64;
    keepalive_timeout 60s;
}

# Map for WebSocket upgrade
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      close;
}

# HTTP server (redirect to HTTPS)
server {
    listen 80;
    listen [::]:80;
    server_name rpc.paw-chain.io;

    # ACME challenge for Let's Encrypt
    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    # Redirect all other requests to HTTPS
    location / {
        return 301 https://$server_name$request_uri;
    }
}

# HTTPS server
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name rpc.paw-chain.io;

    # SSL configuration
    ssl_certificate /etc/letsencrypt/live/rpc.paw-chain.io/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/rpc.paw-chain.io/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options DENY always;
    add_header X-Content-Type-Options nosniff always;
    add_header X-XSS-Protection "1; mode=block" always;

    # CORS headers (adjust origin as needed)
    add_header Access-Control-Allow-Origin "https://app.paw-chain.io" always;
    add_header Access-Control-Allow-Methods "GET, POST, OPTIONS" always;
    add_header Access-Control-Allow-Headers "Authorization, Content-Type" always;
    add_header Access-Control-Max-Age 86400 always;

    # Handle OPTIONS preflight
    if ($request_method = 'OPTIONS') {
        return 204;
    }

    # Logging
    access_log /var/log/nginx/paw-rpc-access.log;
    error_log /var/log/nginx/paw-rpc-error.log warn;

    # Client body size (for large transactions)
    client_max_body_size 10M;

    # Timeouts
    proxy_connect_timeout 60s;
    proxy_send_timeout 120s;
    proxy_read_timeout 120s;

    # WebSocket support
    location /websocket {
        proxy_pass http://paw_rpc_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Long timeout for WebSocket
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }

    # Standard RPC requests
    location / {
        proxy_pass http://paw_rpc_backend;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Connection header for keep-alive
        proxy_set_header Connection "";

        # Enable caching for idempotent requests
        proxy_cache_methods GET HEAD;
        proxy_cache_key "$scheme$request_method$host$request_uri";
        proxy_cache_valid 200 1s;  # Very short cache for blockchain data
    }

    # Health check endpoint (internal use only)
    location /health {
        access_log off;
        proxy_pass http://paw_rpc_backend/status;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
    }
}
```

### REST API Load Balancing

**File:** `/etc/nginx/sites-available/paw-api`

```nginx
upstream paw_api_backend {
    # Least connections (no sticky sessions needed)
    least_conn;

    server 10.0.1.10:1317 max_fails=3 fail_timeout=30s;
    server 10.0.1.11:1317 max_fails=3 fail_timeout=30s;
    server 10.0.1.12:1317 max_fails=3 fail_timeout=30s;

    keepalive 128;
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name api.paw-chain.io;

    ssl_certificate /etc/letsencrypt/live/api.paw-chain.io/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.paw-chain.io/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;

    access_log /var/log/nginx/paw-api-access.log;
    error_log /var/log/nginx/paw-api-error.log;

    # Rate limiting (1000 requests per minute per IP)
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=1000r/m;
    limit_req zone=api_limit burst=50 nodelay;

    location / {
        proxy_pass http://paw_api_backend;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header Connection "";

        # Caching for GET requests
        proxy_cache_methods GET HEAD;
        proxy_cache_key "$scheme$request_method$host$request_uri$args";
        proxy_cache_valid 200 2s;
        add_header X-Cache-Status $upstream_cache_status;
    }

    # Swagger UI
    location /swagger/ {
        proxy_pass http://paw_api_backend;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
    }
}
```

### Enable Configuration

```bash
# Symlink to sites-enabled
sudo ln -s /etc/nginx/sites-available/paw-rpc /etc/nginx/sites-enabled/
sudo ln -s /etc/nginx/sites-available/paw-api /etc/nginx/sites-enabled/

# Test configuration
sudo nginx -t

# Reload Nginx
sudo systemctl reload nginx
```

---

## HAProxy Configuration

HAProxy is a robust, high-performance TCP/HTTP load balancer ideal for complex routing scenarios.

### Full Configuration

**File:** `/etc/haproxy/haproxy.cfg`

```haproxy
#---------------------------------------------------------------------
# Global settings
#---------------------------------------------------------------------
global
    log /dev/log local0
    log /dev/log local1 notice
    chroot /var/lib/haproxy
    stats socket /run/haproxy/admin.sock mode 660 level admin
    stats timeout 30s
    user haproxy
    group haproxy
    daemon

    # Default SSL material locations
    ca-base /etc/ssl/certs
    crt-base /etc/ssl/private

    # TLS configuration
    ssl-default-bind-ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384
    ssl-default-bind-ciphersuites TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256
    ssl-default-bind-options ssl-min-ver TLSv1.2 no-tls-tickets

    # Performance tuning
    tune.ssl.default-dh-param 2048
    maxconn 10000

#---------------------------------------------------------------------
# Defaults
#---------------------------------------------------------------------
defaults
    log global
    mode http
    option httplog
    option dontlognull
    option http-server-close
    option forwardfor except 127.0.0.0/8
    option redispatch
    retries 3
    timeout connect 5s
    timeout client 120s
    timeout server 120s
    timeout http-request 10s
    timeout http-keep-alive 30s
    errorfile 400 /etc/haproxy/errors/400.http
    errorfile 403 /etc/haproxy/errors/403.http
    errorfile 408 /etc/haproxy/errors/408.http
    errorfile 500 /etc/haproxy/errors/500.http
    errorfile 502 /etc/haproxy/errors/502.http
    errorfile 503 /etc/haproxy/errors/503.http
    errorfile 504 /etc/haproxy/errors/504.http

#---------------------------------------------------------------------
# Statistics page
#---------------------------------------------------------------------
listen stats
    bind *:8404
    stats enable
    stats uri /stats
    stats refresh 30s
    stats show-legends
    stats show-node
    stats auth admin:CHANGE_ME_PASSWORD

#---------------------------------------------------------------------
# PAW RPC Frontend (HTTPS)
#---------------------------------------------------------------------
frontend paw_rpc_frontend
    bind *:443 ssl crt /etc/haproxy/certs/rpc.paw-chain.io.pem
    bind *:80
    mode http
    option httplog

    # Redirect HTTP to HTTPS
    redirect scheme https code 301 if !{ ssl_fc }

    # ACLs for routing
    acl is_websocket hdr(Upgrade) -i websocket
    acl is_health path /health

    # Request tracking
    unique-id-format %{+X}o\ %ci:%cp_%fi:%fp_%Ts_%rt:%pid
    unique-id-header X-Request-ID

    # Rate limiting (stick table based)
    stick-table type ip size 100k expire 30s store http_req_rate(10s)
    http-request track-sc0 src
    http-request deny deny_status 429 if { sc_http_req_rate(0) gt 100 }

    # Use backends
    use_backend paw_rpc_websocket if is_websocket
    use_backend paw_rpc_backend if !is_health
    use_backend paw_health if is_health

    default_backend paw_rpc_backend

#---------------------------------------------------------------------
# PAW RPC Backend (Sticky Sessions)
#---------------------------------------------------------------------
backend paw_rpc_backend
    mode http
    balance source  # IP-based sticky sessions
    option httpchk GET /status
    http-check expect status 200

    # Backend servers
    server paw-node-1 10.0.1.10:26657 check inter 10s fall 3 rise 2 maxconn 500
    server paw-node-2 10.0.1.11:26657 check inter 10s fall 3 rise 2 maxconn 500
    server paw-node-3 10.0.1.12:26657 check inter 10s fall 3 rise 2 maxconn 500

    # Advanced health check (verify not catching up)
    http-check send meth GET uri /status ver HTTP/1.1 hdr Host rpc.paw-chain.io
    http-check expect string "catching_up\":false"

    # Response headers
    http-response set-header X-Backend-Server %s

#---------------------------------------------------------------------
# PAW WebSocket Backend
#---------------------------------------------------------------------
backend paw_rpc_websocket
    mode http
    balance source
    option httpchk GET /status
    http-check expect status 200

    # WebSocket timeout (1 hour)
    timeout server 3600s
    timeout tunnel 3600s

    server paw-node-1 10.0.1.10:26657 check inter 10s maxconn 200
    server paw-node-2 10.0.1.11:26657 check inter 10s maxconn 200
    server paw-node-3 10.0.1.12:26657 check inter 10s maxconn 200

#---------------------------------------------------------------------
# Health Check Backend (Internal)
#---------------------------------------------------------------------
backend paw_health
    mode http
    balance roundrobin
    server paw-node-1 10.0.1.10:26657 check
    server paw-node-2 10.0.1.11:26657 check
    server paw-node-3 10.0.1.12:26657 check

#---------------------------------------------------------------------
# PAW API Frontend
#---------------------------------------------------------------------
frontend paw_api_frontend
    bind *:1317 ssl crt /etc/haproxy/certs/api.paw-chain.io.pem
    mode http
    option httplog

    default_backend paw_api_backend

#---------------------------------------------------------------------
# PAW API Backend (No Sticky Sessions)
#---------------------------------------------------------------------
backend paw_api_backend
    mode http
    balance leastconn  # Send to server with fewest connections
    option httpchk GET /cosmos/base/tendermint/v1beta1/node_info
    http-check expect status 200

    server paw-node-1 10.0.1.10:1317 check inter 10s maxconn 1000
    server paw-node-2 10.0.1.11:1317 check inter 10s maxconn 1000
    server paw-node-3 10.0.1.12:1317 check inter 10s maxconn 1000

    # Caching (requires haproxy-cache module)
    # http-response cache-store caching-rules
```

### SSL Certificate Preparation

HAProxy requires combined certificate + key:

```bash
# Combine certificate and private key
sudo cat /etc/letsencrypt/live/rpc.paw-chain.io/fullchain.pem \
         /etc/letsencrypt/live/rpc.paw-chain.io/privkey.pem \
    > /etc/haproxy/certs/rpc.paw-chain.io.pem

sudo chmod 600 /etc/haproxy/certs/rpc.paw-chain.io.pem
```

### Enable and Test

```bash
# Test configuration
sudo haproxy -c -f /etc/haproxy/haproxy.cfg

# Restart HAProxy
sudo systemctl restart haproxy

# Check stats page
curl http://localhost:8404/stats
```

---

## Cloud Load Balancers

### AWS Application Load Balancer (ALB)

**CloudFormation Template:**

```yaml
AWSTemplateFormatVersion: '2010-09-09'
Description: PAW Blockchain Load Balancer

Parameters:
  VpcId:
    Type: AWS::EC2::VPC::Id
    Description: VPC for load balancer
  SubnetIds:
    Type: List<AWS::EC2::Subnet::Id>
    Description: Subnets for load balancer (multi-AZ)
  CertificateArn:
    Type: String
    Description: ACM certificate ARN for HTTPS

Resources:
  # Security Group
  LoadBalancerSecurityGroup:
    Type: AWS::EC2::SecurityGroup
    Properties:
      GroupDescription: PAW RPC Load Balancer
      VpcId: !Ref VpcId
      SecurityGroupIngress:
        - IpProtocol: tcp
          FromPort: 443
          ToPort: 443
          CidrIp: 0.0.0.0/0
        - IpProtocol: tcp
          FromPort: 80
          ToPort: 80
          CidrIp: 0.0.0.0/0

  # Application Load Balancer
  LoadBalancer:
    Type: AWS::ElasticLoadBalancingV2::LoadBalancer
    Properties:
      Name: paw-rpc-alb
      Type: application
      Scheme: internet-facing
      IpAddressType: dualstack
      Subnets: !Ref SubnetIds
      SecurityGroups:
        - !Ref LoadBalancerSecurityGroup
      Tags:
        - Key: Name
          Value: paw-rpc-alb

  # Target Group
  TargetGroup:
    Type: AWS::ElasticLoadBalancingV2::TargetGroup
    Properties:
      Name: paw-rpc-targets
      Port: 26657
      Protocol: HTTP
      VpcId: !Ref VpcId
      HealthCheckEnabled: true
      HealthCheckProtocol: HTTP
      HealthCheckPath: /status
      HealthCheckIntervalSeconds: 30
      HealthCheckTimeoutSeconds: 10
      HealthyThresholdCount: 2
      UnhealthyThresholdCount: 3
      Matcher:
        HttpCode: 200
      TargetGroupAttributes:
        - Key: stickiness.enabled
          Value: true
        - Key: stickiness.type
          Value: source_ip
        - Key: stickiness.source_ip.duration_seconds
          Value: 300
        - Key: deregistration_delay.timeout_seconds
          Value: 30

  # HTTPS Listener
  HttpsListener:
    Type: AWS::ElasticLoadBalancingV2::Listener
    Properties:
      LoadBalancerArn: !Ref LoadBalancer
      Port: 443
      Protocol: HTTPS
      Certificates:
        - CertificateArn: !Ref CertificateArn
      DefaultActions:
        - Type: forward
          TargetGroupArn: !Ref TargetGroup

  # HTTP Listener (redirect to HTTPS)
  HttpListener:
    Type: AWS::ElasticLoadBalancingV2::Listener
    Properties:
      LoadBalancerArn: !Ref LoadBalancer
      Port: 80
      Protocol: HTTP
      DefaultActions:
        - Type: redirect
          RedirectConfig:
            Protocol: HTTPS
            Port: 443
            StatusCode: HTTP_301

Outputs:
  LoadBalancerDNS:
    Description: DNS name of load balancer
    Value: !GetAtt LoadBalancer.DNSName
  TargetGroupArn:
    Description: ARN of target group
    Value: !Ref TargetGroup
```

**Register Targets:**

```bash
# Register EC2 instances
aws elbv2 register-targets \
  --target-group-arn arn:aws:elasticloadbalancing:... \
  --targets Id=i-1234567890abcdef0 Id=i-0987654321fedcba0
```

---

### Google Cloud Load Balancer

**Terraform Configuration:**

```hcl
# Backend service
resource "google_compute_backend_service" "paw_rpc" {
  name                  = "paw-rpc-backend"
  protocol              = "HTTP"
  port_name             = "rpc"
  timeout_sec           = 120
  enable_cdn            = false
  load_balancing_scheme = "EXTERNAL"

  backend {
    group          = google_compute_instance_group.paw_nodes.self_link
    balancing_mode = "UTILIZATION"
    capacity_scaler = 1.0
  }

  health_checks = [google_compute_health_check.paw_rpc.self_link]

  session_affinity = "CLIENT_IP"
  affinity_cookie_ttl_sec = 300
}

# Health check
resource "google_compute_health_check" "paw_rpc" {
  name               = "paw-rpc-health-check"
  check_interval_sec = 10
  timeout_sec        = 5
  healthy_threshold  = 2
  unhealthy_threshold = 3

  http_health_check {
    port         = 26657
    request_path = "/status"
  }
}

# URL map
resource "google_compute_url_map" "paw_rpc" {
  name            = "paw-rpc-url-map"
  default_service = google_compute_backend_service.paw_rpc.self_link
}

# HTTPS proxy
resource "google_compute_target_https_proxy" "paw_rpc" {
  name             = "paw-rpc-https-proxy"
  url_map          = google_compute_url_map.paw_rpc.self_link
  ssl_certificates = [google_compute_managed_ssl_certificate.paw_rpc.self_link]
}

# SSL certificate
resource "google_compute_managed_ssl_certificate" "paw_rpc" {
  name = "paw-rpc-ssl-cert"

  managed {
    domains = ["rpc.paw-chain.io"]
  }
}

# Forwarding rule
resource "google_compute_global_forwarding_rule" "paw_rpc_https" {
  name       = "paw-rpc-https"
  target     = google_compute_target_https_proxy.paw_rpc.self_link
  port_range = "443"
  ip_address = google_compute_global_address.paw_rpc.address
}

# Static IP
resource "google_compute_global_address" "paw_rpc" {
  name = "paw-rpc-ip"
}
```

---

## Geo-Distributed Deployment

For global user base, deploy nodes in multiple regions and route users to the nearest node.

### DNS-Based Geographic Load Balancing

**AWS Route 53 Configuration:**

```json
{
  "Comment": "Geo-distributed PAW RPC endpoints",
  "Changes": [
    {
      "Action": "CREATE",
      "ResourceRecordSet": {
        "Name": "rpc.paw-chain.io",
        "Type": "A",
        "SetIdentifier": "US-East",
        "GeoLocation": {
          "ContinentCode": "NA"
        },
        "AliasTarget": {
          "HostedZoneId": "Z35SXDOTRQ7X7K",
          "DNSName": "us-east-alb-123456.elb.amazonaws.com",
          "EvaluateTargetHealth": true
        }
      }
    },
    {
      "Action": "CREATE",
      "ResourceRecordSet": {
        "Name": "rpc.paw-chain.io",
        "Type": "A",
        "SetIdentifier": "EU-West",
        "GeoLocation": {
          "ContinentCode": "EU"
        },
        "AliasTarget": {
          "HostedZoneId": "Z32O12XQLNTSW2",
          "DNSName": "eu-west-alb-789012.elb.amazonaws.com",
          "EvaluateTargetHealth": true
        }
      }
    },
    {
      "Action": "CREATE",
      "ResourceRecordSet": {
        "Name": "rpc.paw-chain.io",
        "Type": "A",
        "SetIdentifier": "Asia-Pacific",
        "GeoLocation": {
          "ContinentCode": "AS"
        },
        "AliasTarget": {
          "HostedZoneId": "Z1WI8VXHPB1R38",
          "DNSName": "ap-southeast-alb-345678.elb.amazonaws.com",
          "EvaluateTargetHealth": true
        }
      }
    }
  ]
}
```

### Multi-Region Architecture

```
┌────────────────────────────────────────────────────────────────┐
│ Global DNS Load Balancing (GeoDNS)                            │
│ Route 53 / Cloud DNS / Cloudflare                             │
└────┬────────────────────┬────────────────────┬────────────────┘
     │                    │                    │
     │                    │                    │
┌────▼─────────┐   ┌──────▼──────────┐   ┌────▼─────────────┐
│ US-East-1    │   │ EU-West-1       │   │ AP-Southeast-1   │
│              │   │                 │   │                  │
│ ┌──────────┐ │   │ ┌─────────────┐ │   │ ┌──────────────┐ │
│ │   ALB    │ │   │ │     ALB     │ │   │ │     ALB      │ │
│ └────┬─────┘ │   │ └──────┬──────┘ │   │ └──────┬───────┘ │
│      │       │   │        │        │   │        │         │
│ ┌────▼─────┐ │   │ ┌──────▼──────┐ │   │ ┌──────▼───────┐ │
│ │ Node 1   │ │   │ │   Node 4    │ │   │ │   Node 7     │ │
│ │ Node 2   │ │   │ │   Node 5    │ │   │ │   Node 8     │ │
│ │ Node 3   │ │   │ │   Node 6    │ │   │ │   Node 9     │ │
│ └──────────┘ │   │ └─────────────┘ │   │ └──────────────┘ │
└──────────────┘   └─────────────────┘   └──────────────────┘
```

### Latency-Based Routing

**Cloudflare Load Balancer Configuration:**

```yaml
load_balancers:
  - name: paw-rpc-global
    default_pools:
      - us-east-pool
      - eu-west-pool
      - ap-southeast-pool
    fallback_pool: us-east-pool
    steering_policy: geo
    proxied: true
    session_affinity: ip_cookie
    session_affinity_ttl: 300

pools:
  - name: us-east-pool
    origins:
      - name: us-east-1
        address: us-east-alb.elb.amazonaws.com
        enabled: true
    monitor: paw-health-check
    notification_email: ops@paw-chain.io

  - name: eu-west-pool
    origins:
      - name: eu-west-1
        address: eu-west-alb.elb.amazonaws.com
        enabled: true
    monitor: paw-health-check

  - name: ap-southeast-pool
    origins:
      - name: ap-southeast-1
        address: ap-southeast-alb.elb.amazonaws.com
        enabled: true
    monitor: paw-health-check

monitors:
  - name: paw-health-check
    type: https
    method: GET
    path: /status
    interval: 60
    timeout: 5
    retries: 2
    expected_codes: "200"
    follow_redirects: false
    allow_insecure: false
```

---

## Health Checks

### Advanced Health Check Script

**File:** `/usr/local/bin/paw-health-check.sh`

```bash
#!/bin/bash
set -euo pipefail

# PAW Blockchain Health Check Script
# Returns HTTP 200 if node is healthy, 503 otherwise

RPC_URL="${PAW_RPC_URL:-http://localhost:26657}"
MAX_BLOCK_AGE_SECONDS="${MAX_BLOCK_AGE:-60}"
MIN_PEER_COUNT="${MIN_PEERS:-5}"

# Query node status
STATUS=$(curl -sf "${RPC_URL}/status" || echo '{}')

# Check if catching up
CATCHING_UP=$(echo "$STATUS" | jq -r '.result.sync_info.catching_up // true')
if [ "$CATCHING_UP" = "true" ]; then
    echo "UNHEALTHY: Node is catching up"
    exit 1
fi

# Check block age
LATEST_BLOCK_TIME=$(echo "$STATUS" | jq -r '.result.sync_info.latest_block_time // ""')
if [ -n "$LATEST_BLOCK_TIME" ]; then
    BLOCK_TIMESTAMP=$(date -d "$LATEST_BLOCK_TIME" +%s 2>/dev/null || echo 0)
    NOW=$(date +%s)
    BLOCK_AGE=$((NOW - BLOCK_TIMESTAMP))

    if [ "$BLOCK_AGE" -gt "$MAX_BLOCK_AGE_SECONDS" ]; then
        echo "UNHEALTHY: Latest block is ${BLOCK_AGE}s old (max: ${MAX_BLOCK_AGE_SECONDS}s)"
        exit 1
    fi
fi

# Check peer count
PEER_COUNT=$(curl -sf "${RPC_URL}/net_info" | jq -r '.result.n_peers // 0')
if [ "$PEER_COUNT" -lt "$MIN_PEER_COUNT" ]; then
    echo "UNHEALTHY: Only $PEER_COUNT peers (min: $MIN_PEER_COUNT)"
    exit 1
fi

# Check validator signing (if validator)
if [ -f "/var/run/paw/validator.flag" ]; then
    VALIDATOR_ADDR=$(pawd tendermint show-address)
    SIGNING_INFO=$(pawd query slashing signing-info "$VALIDATOR_ADDR" -o json 2>/dev/null || echo '{}')
    JAILED=$(echo "$SIGNING_INFO" | jq -r '.val_signing_info.jailed_until // "1970-01-01T00:00:00Z"')

    if [ "$JAILED" != "1970-01-01T00:00:00Z" ]; then
        echo "UNHEALTHY: Validator is jailed until $JAILED"
        exit 1
    fi
fi

echo "HEALTHY: Node is synced, block age ${BLOCK_AGE}s, $PEER_COUNT peers"
exit 0
```

**Make executable:**

```bash
sudo chmod +x /usr/local/bin/paw-health-check.sh
```

**Use in load balancer:**

```nginx
# Nginx health check
location /health {
    access_log off;
    proxy_pass http://127.0.0.1:8080/health;
}
```

**Wrapper HTTP server for health check:**

```python
#!/usr/bin/env python3
from http.server import BaseHTTPRequestHandler, HTTPServer
import subprocess
import logging

class HealthCheckHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            try:
                result = subprocess.run(
                    ['/usr/local/bin/paw-health-check.sh'],
                    capture_output=True,
                    text=True,
                    timeout=5
                )
                if result.returncode == 0:
                    self.send_response(200)
                    self.send_header('Content-Type', 'text/plain')
                    self.end_headers()
                    self.wfile.write(result.stdout.encode())
                else:
                    self.send_response(503)
                    self.send_header('Content-Type', 'text/plain')
                    self.end_headers()
                    self.wfile.write(result.stdout.encode())
            except Exception as e:
                self.send_response(500)
                self.send_header('Content-Type', 'text/plain')
                self.end_headers()
                self.wfile.write(f"Error: {e}".encode())
        else:
            self.send_response(404)
            self.end_headers()

if __name__ == '__main__':
    server = HTTPServer(('127.0.0.1', 8080), HealthCheckHandler)
    logging.info('Health check server listening on :8080')
    server.serve_forever()
```

---

## SSL/TLS Termination

### Automated Certificate Management (Let's Encrypt)

**Certbot with Nginx:**

```bash
# Install certbot
sudo apt-get install certbot python3-certbot-nginx

# Obtain certificate
sudo certbot --nginx -d rpc.paw-chain.io -d api.paw-chain.io

# Auto-renewal (crontab)
sudo crontab -e
# Add: 0 3 * * * certbot renew --quiet --deploy-hook "systemctl reload nginx"
```

**Manual Certificate Renewal:**

```bash
# Renew all certificates
sudo certbot renew

# Reload Nginx
sudo systemctl reload nginx
```

---

## Rate Limiting and DDoS Protection

### Nginx Rate Limiting

```nginx
# Define rate limit zones
http {
    # Limit by IP address (100 req/sec)
    limit_req_zone $binary_remote_addr zone=perip:10m rate=100r/s;

    # Limit by server (global 10000 req/sec)
    limit_req_zone $server_name zone=perserver:10m rate=10000r/s;

    # Connection limit (10 concurrent connections per IP)
    limit_conn_zone $binary_remote_addr zone=addr:10m;
}

server {
    location / {
        limit_req zone=perip burst=50 nodelay;
        limit_req zone=perserver burst=500;
        limit_conn addr 10;

        # ... proxy configuration ...
    }
}
```

### Cloudflare DDoS Protection

Configure in Cloudflare dashboard:
- **Security → WAF:** Enable "I'm Under Attack" mode for DDoS
- **Security → Rate Limiting:** 100 requests per minute per IP
- **Security → Bot Fight Mode:** Enable to block bots
- **Firewall Rules:** Block by country, ASN, threat score

---

## Monitoring and Metrics

### Nginx Metrics (VTS Module)

**Install nginx-module-vts:**

```bash
# Add to nginx.conf
http {
    vhost_traffic_status_zone;
    vhost_traffic_status_filter_by_host on;
}

server {
    location /status {
        vhost_traffic_status_display;
        vhost_traffic_status_display_format prometheus;
        allow 127.0.0.1;
        deny all;
    }
}
```

**Prometheus scrape config:**

```yaml
scrape_configs:
  - job_name: 'nginx'
    static_configs:
      - targets: ['nginx:8080']
    metrics_path: '/status/format/prometheus'
```

### HAProxy Metrics

**Enable Prometheus exporter:**

```bash
# Install haproxy_exporter
docker run -d --name haproxy-exporter \
  -p 9101:9101 \
  --network host \
  quay.io/prometheus/haproxy-exporter:latest \
  --haproxy.scrape-uri="http://localhost:8404/stats;csv"
```

**Grafana Dashboard:** Use dashboard ID 2428

---

## Troubleshooting

### Sticky Sessions Not Working

**Symptom:** Clients routed to different backends between requests

**Solutions:**

1. **Nginx:** Verify `ip_hash` is set
   ```nginx
   upstream backend {
       ip_hash;
       server ...;
   }
   ```

2. **HAProxy:** Check `balance source`
   ```haproxy
   backend paw_rpc_backend
       balance source
   ```

3. **ALB:** Enable stickiness in target group attributes

### WebSocket Connections Failing

**Symptom:** WebSocket upgrade fails with 400/502

**Solutions:**

1. **Nginx:** Ensure upgrade headers are set
   ```nginx
   proxy_set_header Upgrade $http_upgrade;
   proxy_set_header Connection $connection_upgrade;
   ```

2. **HAProxy:** Use `tunnel` timeout
   ```haproxy
   timeout tunnel 3600s
   ```

3. **Check logs:**
   ```bash
   tail -f /var/log/nginx/error.log
   journalctl -u haproxy -f
   ```

### Health Check Failures

**Symptom:** Backends marked unhealthy despite being up

**Solutions:**

1. **Test health endpoint manually:**
   ```bash
   curl -v http://node-ip:26657/status
   ```

2. **Check response format:**
   ```bash
   curl -s http://node-ip:26657/status | jq '.result.sync_info.catching_up'
   ```

3. **Adjust health check threshold:**
   - Increase interval (less aggressive)
   - Decrease healthy/unhealthy thresholds

---

## References

- [Nginx Load Balancing](https://docs.nginx.com/nginx/admin-guide/load-balancer/http-load-balancer/)
- [HAProxy Documentation](https://www.haproxy.org/documentation.html)
- [AWS Application Load Balancer](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/)
- [Google Cloud Load Balancing](https://cloud.google.com/load-balancing/docs)
- [CometBFT RPC Documentation](https://docs.cometbft.com/v0.38/core/rpc)

---

**For additional support, see:**
- [DEPLOYMENT_QUICKSTART.md](/docs/DEPLOYMENT_QUICKSTART.md)
- [METRICS.md](/docs/METRICS.md)
- [ENVIRONMENT_VARIABLES.md](/docs/ENVIRONMENT_VARIABLES.md)
- [TROUBLESHOOTING.md](/docs/TROUBLESHOOTING.md)
