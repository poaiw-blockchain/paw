# PAW P2P Security and Peer Reputation System

## Overview

The PAW blockchain implements a comprehensive peer reputation system to protect against malicious peers and network attacks. This system monitors peer behavior, assigns reputation scores, and automatically bans misbehaving nodes.

## Table of Contents

1. [Architecture](#architecture)
2. [Peer Scoring Algorithm](#peer-scoring-algorithm)
3. [Ban Mechanisms](#ban-mechanisms)
4. [Security Features](#security-features)
5. [Configuration](#configuration)
6. [Integration Guide](#integration-guide)
7. [API Reference](#api-reference)
8. [Monitoring](#monitoring)
9. [Best Practices](#best-practices)

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────┐
│                    P2P Reputation System                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │   Manager    │  │    Scorer    │  │   Storage    │    │
│  │              │  │              │  │              │    │
│  │ - Peer State │  │ - Calculate  │  │ - Persist    │    │
│  │ - Decisions  │  │   Scores     │  │   Data       │    │
│  │ - Banning    │  │ - Weights    │  │ - Snapshots  │    │
│  └──────────────┘  └──────────────┘  └──────────────┘    │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │   Monitor    │  │   Metrics    │  │ HTTP Handler │    │
│  │              │  │              │  │              │    │
│  │ - Health     │  │ - Events     │  │ - API        │    │
│  │ - Alerts     │  │ - Prometheus │  │ - Dashboard  │    │
│  │ - Checks     │  │ - History    │  │ - Control    │    │
│  └──────────────┘  └──────────────┘  └──────────────┘    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
                 ┌────────────────┐
                 │   CometBFT     │
                 │   P2P Layer    │
                 └────────────────┘
```

### Data Flow

```
Peer Event → Manager.RecordEvent() → Scorer.ApplyEvent()
                                    → Update Metrics
                                    → Check Ban Rules
                                    → Storage.Save()
                                    → Monitor Alerts
```

## Peer Scoring Algorithm

### Score Calculation

Peer reputation scores range from **0-100** based on weighted factors:

```go
Score = (Uptime × 0.25) +
        (MessageValidity × 0.30) +
        (Latency × 0.20) +
        (BlockPropagation × 0.15) -
        (Violations × 0.10)
```

### Scoring Factors

#### 1. Uptime Score (Weight: 0.25)

Measures peer availability and connection stability:

- **Uptime Ratio**: Total connected time / total known time
- **Connection Stability**: Penalizes frequent disconnections
- **Bonus**: +10% for peers with >1 hour average session time

**Calculation**:

```
UptimeScore = (UptimeRatio × 0.7 + StabilityScore × 0.3) × 100
```

#### 2. Message Validity Score (Weight: 0.30)

Tracks the ratio of valid to invalid messages:

- **Valid Message Ratio**: ValidMessages / TotalMessages
- **Threshold**: 95% for excellent score
- **Non-linear**: Heavy penalty below 80%

**Scoring Bands**:

- 95-100%: 80-100 points
- 80-95%: 40-80 points
- <80%: 0-40 points

#### 3. Latency Score (Weight: 0.20)

Based on average response latency:

- **Excellent**: <500ms → 80-100 points
- **Good**: 500ms-2s → 40-80 points
- **Poor**: >2s → 0-40 points (exponential decay)

#### 4. Block Propagation Score (Weight: 0.15)

Measures block propagation speed:

- **Fast Blocks**: Propagated within 1 second
- **Average Time**: Weighted by all propagations
- **Bonus**: +10% if >80% of blocks are "fast"

**Scoring**:

- <1s: 90-100 points
- 1-5s: 60-90 points
- 5-30s: 30-60 points
- > 30s: 0-30 points

#### 5. Violation Penalties (Weight: 0.10)

Deductions for protocol violations:

| Violation Type     | Penalty              |
| ------------------ | -------------------- |
| Protocol Violation | -5.0                 |
| Malformed Message  | -2.0                 |
| Spam Attempt       | -10.0                |
| Invalid Block      | -20.0                |
| Double Signing     | -100.0 (instant ban) |

### Score Decay

Scores decay over time for inactive peers:

```
Decay = ScoreDecayFactor ^ (TimeSinceLastSeen / DecayPeriod)
```

Default: 5% decay per 24 hours (factor = 0.95)

### Trust Levels

Based on final score:

| Score Range | Trust Level | Description                    |
| ----------- | ----------- | ------------------------------ |
| 0-20        | Untrusted   | High risk, likely to be banned |
| 20-50       | Low         | Poor reputation                |
| 50-75       | Medium      | Average peer                   |
| 75-100      | High        | Excellent reputation           |
| Whitelisted | Whitelisted | Never banned                   |

## Ban Mechanisms

### Automatic Banning

Peers are automatically banned when:

1. **Severe Violations** (Permanent Ban):
   - Double signing attempt detected
   - 3+ invalid block proposals
   - Persistent protocol violations with score <20

2. **Poor Behavior** (Temporary Ban):
   - Reputation score drops below 20
   - 5+ spam attempts
   - <50% message validity ratio (with 100+ messages)

### Ban Types

#### Temporary Ban

- **Duration**: Exponential backoff (1h, 2h, 4h, 8h, ... up to 7 days)
- **Escalation**: After 3 temporary bans → permanent ban
- **Recovery**: Ban expires, peer can reconnect with neutral score

#### Permanent Ban

- **Duration**: Indefinite
- **Triggers**: Severe violations (double signing, invalid blocks)
- **Recovery**: Manual unban only (via API or governance)

### Whitelist

Whitelisted peers:

- Never automatically banned
- Always assigned "Whitelisted" trust level
- Typically your own validators or trusted partners

**Configuration**:

```toml
[reputation.whitelist]
peers = [
    "16Uiu2HAm...",  # Your validator
    "16Uiu2HAm...",  # Trusted partner
]
```

## Security Features

### 1. Sybil Attack Resistance

Limits per network segment prevent single entity dominance:

```toml
[reputation.manager]
max_peers_per_subnet = 10      # Max from same /24
max_peers_per_country = 50     # Max from same country
max_peers_per_asn = 15         # Max from same AS
```

**Protection**:

- Subnet concentration monitoring
- Geographic distribution requirements
- Automatic rejection of excess peers

### 2. Eclipse Attack Prevention

Ensures geographic diversity:

```toml
[reputation.security]
require_geo_diversity = true
min_different_countries = 3
max_percent_from_country = 0.40  # Max 40% from one country
```

**Monitoring**:

- Alerts on geographic imbalance
- Diverse peer selection for critical operations
- Connection diversity requirements

### 3. Connection Limits

```toml
[reputation.security]
max_inbound_connections = 50
max_outbound_connections = 50
max_new_peers_per_hour = 100
max_peers_from_new_subnets = 20
```

### 4. Rate Limiting

Per-peer message rate limits:

```toml
[reputation.security]
enable_rate_limiting = true
max_messages_per_second = 100
max_blocks_per_second = 10
rate_limit_window_duration = "10s"
```

**Enforcement**:

- Sliding window rate tracking
- Automatic spam detection
- Temporary bans for violations

## Configuration

### File Location

Default: `~/.paw/config/p2p_security.toml`

### Key Settings

```toml
[reputation]
enabled = true

[reputation.scoring]
# Weights (must be balanced)
uptime_weight = 0.25
message_validity_weight = 0.30
latency_weight = 0.20
block_propagation_weight = 0.15
violation_penalty = 0.10

# Thresholds
min_valid_message_ratio = 0.95
max_latency_for_good_score = "500ms"
fast_block_threshold = "1s"

# Starting score for new peers
new_peer_start_score = 50.0

[reputation.manager]
enable_auto_ban = true
temp_ban_duration = "24h"
max_temp_bans = 3

# Maintenance
snapshot_interval = "1h"
cleanup_age = "720h"  # 30 days
```

### Environment Variables

Override config via environment:

```bash
PAW_REPUTATION_ENABLED=true
PAW_REPUTATION_AUTO_BAN=true
PAW_REPUTATION_DATA_DIR=/custom/path
```

## Integration Guide

### 1. Initialize Reputation System

```go
package main

import (
    "github.com/paw-chain/paw/p2p/reputation"
    "cosmossdk.io/log"
)

func initReputationSystem(homeDir string, logger log.Logger) (*reputation.Manager, error) {
    // Load configuration
    config := reputation.DefaultConfig(homeDir)
    configPath := filepath.Join(homeDir, "config", "p2p_security.toml")

    if cfg, err := reputation.LoadConfig(configPath); err == nil {
        config = *cfg
    }

    // Create storage
    storageConfig := reputation.DefaultFileStorageConfig(homeDir)
    storage, err := reputation.NewFileStorage(storageConfig, logger)
    if err != nil {
        return nil, err
    }

    // Create manager
    managerConfig := config.Manager.ToManagerConfig(
        config.Scoring.ToScoringConfig(),
        config.Scoring.ToScoreWeights(),
    )

    manager, err := reputation.NewManager(storage, managerConfig, logger)
    if err != nil {
        return nil, err
    }

    return manager, nil
}
```

### 2. Record Peer Events

```go
// Connection event
manager.RecordEvent(reputation.PeerEvent{
    PeerID:    reputation.PeerID("16Uiu2HAm..."),
    EventType: reputation.EventTypeConnected,
    Timestamp: time.Now(),
})

// Valid message received
manager.RecordEvent(reputation.PeerEvent{
    PeerID:    reputation.PeerID("16Uiu2HAm..."),
    EventType: reputation.EventTypeValidMessage,
    Timestamp: time.Now(),
    Data: reputation.EventData{
        MessageSize: 1024,
    },
})

// Block propagated
manager.RecordEvent(reputation.PeerEvent{
    PeerID:    reputation.PeerID("16Uiu2HAm..."),
    EventType: reputation.EventTypeBlockPropagated,
    Timestamp: time.Now(),
    Data: reputation.EventData{
        Latency:     500 * time.Millisecond,
        BlockHeight: 12345,
    },
})

// Protocol violation
manager.RecordEvent(reputation.PeerEvent{
    PeerID:    reputation.PeerID("16Uiu2HAm..."),
    EventType: reputation.EventTypeProtocolViolation,
    Timestamp: time.Now(),
    Data: reputation.EventData{
        ViolationType: "invalid_signature",
        Details:       "Message signature verification failed",
    },
})
```

### 3. Peer Selection

```go
// Check if peer should be accepted
shouldAccept, reason := manager.ShouldAcceptPeer(
    reputation.PeerID("16Uiu2HAm..."),
    "192.168.1.100",
)
if !shouldAccept {
    logger.Info("rejecting peer", "reason", reason)
    return
}

// Get top peers for block requests
topPeers := manager.GetTopPeers(10, 70.0) // 10 peers with score >= 70

// Get geographically diverse peers
diversePeers := manager.GetDiversePeers(20, 50.0) // 20 diverse peers with score >= 50
```

### 4. Manual Management

```go
// Ban a peer
manager.BanPeer(
    reputation.PeerID("16Uiu2HAm..."),
    24 * time.Hour,  // duration (0 = permanent)
    "Suspicious activity detected",
)

// Unban a peer
manager.UnbanPeer(reputation.PeerID("16Uiu2HAm..."))

// Whitelist a trusted peer
manager.AddToWhitelist(reputation.PeerID("16Uiu2HAm..."))
```

### 5. Monitoring Setup

```go
// Create monitor
monitorConfig := reputation.DefaultMonitorConfig()
monitor := reputation.NewMonitor(manager, metrics, monitorConfig, logger)

// Check health
health := monitor.GetHealth()
if !health.Healthy {
    logger.Error("reputation system unhealthy", "issues", health.Issues)
}

// Get alerts
alerts := monitor.GetAlerts(
    time.Now().Add(-24 * time.Hour),  // since
    nil,  // all types
    nil,  // all severities
)
```

### 6. HTTP API Setup

```go
// Create HTTP handlers
handlers := reputation.NewHTTPHandlers(manager, monitor, metrics)

// Register routes
mux := http.NewServeMux()
handlers.RegisterRoutes(mux)

// Start server
go http.ListenAndServe(":8080", mux)
```

## API Reference

### REST Endpoints

#### Get All Peers

```http
GET /api/p2p/reputation/peers
```

**Response**:

```json
{
  "peers": [
    {
      "peer_id": "16Uiu2HAm...",
      "address": "192.168.1.100",
      "score": 85.5,
      "trust_level": "high",
      "last_seen": "2025-01-15T12:00:00Z",
      "metrics": { ... }
    }
  ],
  "count": 42
}
```

#### Get Peer Details

```http
GET /api/p2p/reputation/peer/{peer_id}
```

#### Get Top Peers

```http
GET /api/p2p/reputation/top?n=10&min_score=50
```

#### Get Diverse Peers

```http
GET /api/p2p/reputation/diverse?n=20&min_score=50
```

#### Get Statistics

```http
GET /api/p2p/reputation/stats
```

**Response**:

```json
{
  "total_peers": 100,
  "banned_peers": 5,
  "whitelisted_peers": 3,
  "avg_score": 72.3,
  "score_distribution": {
    "0-20": 5,
    "20-40": 10,
    "40-60": 25,
    "60-80": 35,
    "80-100": 25
  },
  "trust_distribution": {
    "high": 30,
    "medium": 45,
    "low": 20,
    "untrusted": 5
  }
}
```

#### Health Check

```http
GET /api/p2p/reputation/health
```

**Response**:

```json
{
  "healthy": true,
  "last_check": "2025-01-15T12:00:00Z",
  "issues": [],
  "total_peers": 100,
  "banned_peers": 5,
  "avg_score": 72.3,
  "storage_healthy": true
}
```

#### Get Alerts

```http
GET /api/p2p/reputation/alerts?since=2025-01-15T00:00:00Z&type=high_ban_rate&severity=warning
```

#### Get Metrics

```http
GET /api/p2p/reputation/metrics
```

#### Prometheus Metrics

```http
GET /api/p2p/reputation/metrics/prometheus
```

**Response**:

```
# HELP paw_p2p_reputation_events_total Total number of reputation events
# TYPE paw_p2p_reputation_events_total counter
paw_p2p_reputation_events_total{type="valid_message"} 15234
paw_p2p_reputation_events_total{type="invalid_message"} 42
...
```

#### Ban Peer

```http
POST /api/p2p/reputation/ban
Content-Type: application/json

{
  "peer_id": "16Uiu2HAm...",
  "duration": "24h",
  "reason": "Suspicious behavior"
}
```

#### Unban Peer

```http
POST /api/p2p/reputation/unban
Content-Type: application/json

{
  "peer_id": "16Uiu2HAm..."
}
```

## Monitoring

### Grafana Dashboard

Import the provided Grafana dashboard for visualization:

**Key Metrics**:

- Peer count over time
- Average reputation score
- Ban rate
- Event rates (messages, blocks, violations)
- Geographic distribution
- Subnet concentration

### Prometheus Integration

Add to `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'paw_reputation'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/api/p2p/reputation/metrics/prometheus'
    scrape_interval: 30s
```

### Alert Rules

Example Prometheus alert rules:

```yaml
groups:
  - name: paw_reputation
    rules:
      - alert: HighBanRate
        expr: rate(paw_p2p_reputation_bans_total[1h]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: 'High peer ban rate detected'

      - alert: LowAverageScore
        expr: avg(paw_p2p_reputation_score) < 60
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: 'Average peer reputation score is low'

      - alert: SubnetConcentration
        expr: max(paw_p2p_reputation_subnet_peers) / paw_p2p_reputation_peers > 0.3
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: 'High peer concentration in single subnet'
```

## Best Practices

### 1. Configuration

- **Start Conservative**: Begin with stricter limits, relax as network grows
- **Whitelist Validators**: Add known validators to whitelist
- **Tune Weights**: Adjust scoring weights based on your priorities
- **Enable Auto-Ban**: Automated protection is recommended for production

### 2. Monitoring

- **Daily Reviews**: Check reputation dashboard daily
- **Alert Setup**: Configure alerts for anomalies
- **Log Analysis**: Review ban logs weekly
- **Score Trends**: Monitor average score trends

### 3. Maintenance

- **Regular Backups**: Backup reputation data weekly
- **Cleanup**: Let automatic cleanup remove old data
- **Updates**: Keep scoring thresholds updated based on network behavior
- **Audits**: Periodically audit whitelisted and banned peers

### 4. Security

- **Whitelist Carefully**: Only whitelist fully trusted peers
- **Review Bans**: Investigate patterns in banned peers
- **Geographic Diversity**: Ensure connections span multiple regions
- **Rate Limiting**: Keep enabled to prevent DoS
- **Subnet Limits**: Prevent concentration attacks

### 5. Performance

- **Cache Tuning**: Adjust cache size based on peer count
- **Flush Interval**: Balance between performance and data safety
- **Snapshot Frequency**: Reduce if causing performance issues
- **Storage**: Use SSD for reputation data directory

### 6. Troubleshooting

**High False Positive Ban Rate**:

- Increase `new_peer_start_score`
- Reduce `violation_penalty_score`
- Review `min_valid_message_ratio` threshold

**Low Peer Count**:

- Reduce `max_peers_per_subnet`
- Disable `require_geo_diversity` temporarily
- Lower `new_peer_score_threshold`

**Performance Issues**:

- Increase `flush_interval`
- Reduce `snapshot_interval`
- Disable `enable_geo_lookup` if not needed

### 7. Integration Checklist

- [ ] Configure `p2p_security.toml`
- [ ] Initialize reputation system in node startup
- [ ] Record peer events in P2P handlers
- [ ] Use peer selection for critical operations
- [ ] Set up monitoring and alerts
- [ ] Configure Prometheus scraping
- [ ] Import Grafana dashboard
- [ ] Whitelist known validators
- [ ] Test ban/unban functionality
- [ ] Document custom configurations

## Security Considerations

### Threat Model

The reputation system protects against:

1. **Sybil Attacks**: Multiple identities from same source
2. **Eclipse Attacks**: Surrounding node with malicious peers
3. **Spam/DoS**: Message flooding
4. **Invalid Data**: Malformed messages, invalid blocks
5. **Double Signing**: Consensus attacks
6. **Network Manipulation**: Geographic concentration

### Limitations

- **Geographic Data**: Requires external service for full geo-blocking
- **CometBFT Integration**: Advisory only, cannot directly control CometBFT peer connections
- **Initial Trust**: New peers start with neutral score (50/100)
- **Storage**: Reputation data can be deleted; backups recommended

### Recommendations

1. **Defense in Depth**: Use alongside other security measures
2. **Regular Updates**: Keep scoring algorithm tuned
3. **Manual Review**: Don't rely solely on automation
4. **Incident Response**: Have plan for mass attacks
5. **Governance**: Consider governance-based ban appeals

## Future Enhancements

Planned improvements:

- [ ] Machine learning-based anomaly detection
- [ ] Integration with slashing module
- [ ] Automated geographic IP lookups
- [ ] Peer reputation sharing between validators
- [ ] Historical reputation analysis
- [ ] Advanced Sybil detection algorithms
- [ ] Economic incentives for good behavior

## Support

For issues or questions:

- GitHub Issues: https://github.com/paw-chain/paw/issues
- Discord: https://discord.gg/paw-chain
- Documentation: https://docs.paw-chain.com

## License

Copyright © 2025 PAW Chain

Licensed under the Apache License, Version 2.0.
