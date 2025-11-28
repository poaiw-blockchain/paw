# P2P Reputation System

A comprehensive peer reputation and security system for the PAW blockchain P2P network.

## Features

- **Reputation Scoring**: Multi-factor scoring algorithm (0-100 scale)
- **Automatic Banning**: Temporary and permanent bans for misbehaving peers
- **Security Protections**: Sybil, Eclipse, and DoS attack resistance
- **Persistence**: File-based storage with snapshots
- **Monitoring**: Health checks, alerts, and metrics
- **HTTP API**: RESTful API for management and monitoring
- **Prometheus**: Built-in metrics export
- **CLI Tools**: Command-line interface for administration

## Quick Start

### Basic Integration

```go
import (
    "github.com/paw-chain/paw/p2p/reputation"
    "cosmossdk.io/log"
)

// Initialize reputation system
repSystem, err := reputation.NewExampleIntegration(homeDir, logger)
if err != nil {
    return err
}
defer repSystem.Shutdown(context.Background())

// Record peer events
repSystem.HandlePeerConnected(peerID, address)
repSystem.HandleMessageReceived(peerID, messageSize, valid)
repSystem.HandleBlockReceived(peerID, blockHeight, propagationTime)

// Select peers for operations
topPeers := repSystem.SelectPeersForBlockRequest(10)
diversePeers := repSystem.SelectDiversePeers(20)
```

### CLI Usage

```bash
# List all peers
pawcli reputation list

# Show peer details
pawcli reputation show <peer_id>

# Show statistics
pawcli reputation stats

# Ban a peer
pawcli reputation ban <peer_id> --duration 24h --reason "spam"

# Unban a peer
pawcli reputation unban <peer_id>

# Whitelist a peer
pawcli reputation whitelist <peer_id>

# Export data
pawcli reputation export --output reputation.json
```

### HTTP API

Start the HTTP server:

```go
handlers := reputation.NewHTTPHandlers(manager, monitor, metrics)
mux := http.NewServeMux()
handlers.RegisterRoutes(mux)
http.ListenAndServe(":8080", mux)
```

Access endpoints:

- `GET /api/p2p/reputation/peers` - List all peers
- `GET /api/p2p/reputation/peer/{id}` - Get peer details
- `GET /api/p2p/reputation/stats` - Get statistics
- `GET /api/p2p/reputation/health` - Health check
- `GET /api/p2p/reputation/metrics/prometheus` - Prometheus metrics

## Architecture

### Components

- **Manager**: Central coordinator for reputation system
- **Scorer**: Calculates reputation scores based on behavior
- **Storage**: Persists reputation data (file or memory)
- **Monitor**: Health checks and alerting
- **Metrics**: Performance and behavior tracking
- **HTTPHandlers**: REST API endpoints

### Scoring Algorithm

Score = weighted sum of:

1. **Uptime** (25%): Connection stability and availability
2. **Message Validity** (30%): Ratio of valid to total messages
3. **Latency** (20%): Response time performance
4. **Block Propagation** (15%): Block relay speed
5. **Violations** (-10%): Protocol violation penalties

### Ban Triggers

**Permanent Ban**:

- Double signing attempt
- 3+ invalid block proposals
- Persistent violations with low score

**Temporary Ban**:

- Score drops below 20
- 5+ spam attempts
- <50% message validity (100+ messages)

## Configuration

Edit `~/.paw/config/p2p_security.toml`:

```toml
[reputation]
enabled = true

[reputation.scoring]
uptime_weight = 0.25
message_validity_weight = 0.30
latency_weight = 0.20
block_propagation_weight = 0.15
violation_penalty = 0.10

min_valid_message_ratio = 0.95
max_latency_for_good_score = "500ms"
new_peer_start_score = 50.0

[reputation.manager]
enable_auto_ban = true
temp_ban_duration = "24h"
max_temp_bans = 3

max_peers_per_subnet = 10
max_peers_per_country = 50
min_geographic_diversity = 3

[reputation.security]
max_new_peers_per_hour = 100
require_geo_diversity = true
enable_rate_limiting = true
max_messages_per_second = 100

[reputation.whitelist]
peers = ["16Uiu2HAm..."]
```

## Security Features

### Sybil Attack Resistance

- Subnet-based connection limits
- ASN-based peer limits
- Geographic diversity requirements

### Eclipse Attack Prevention

- Minimum geographic diversity
- Maximum concentration per country
- Diverse peer selection for critical ops

### DoS Protection

- Per-peer rate limiting
- Automatic spam detection
- Connection limits

### Data Integrity

- Malformed message detection
- Invalid block tracking
- Double-signing detection

## Monitoring

### Health Checks

```go
health := monitor.GetHealth()
if !health.Healthy {
    log.Error("issues", health.Issues)
}
```

### Alerts

```go
alerts := monitor.GetAlerts(time.Now().Add(-24*time.Hour), nil, nil)
for _, alert := range alerts {
    log.Warn(alert.Message, "severity", alert.Severity)
}
```

### Prometheus Metrics

```
paw_p2p_reputation_events_total{type="valid_message"}
paw_p2p_reputation_bans_total{type="temporary"}
paw_p2p_reputation_peers
paw_p2p_reputation_processing_seconds
```

## Testing

### Unit Tests

```bash
cd p2p/reputation
go test -v
```

### Integration Tests

```bash
go test -v -tags=integration
```

### Load Tests

```bash
go test -v -bench=. -benchmem
```

## Performance

- **Storage**: ~1KB per peer
- **Memory**: ~100MB for 10,000 peers (with cache)
- **CPU**: <1% overhead on modern hardware
- **Disk I/O**: Batched writes every 30s

### Optimization Tips

1. **Increase cache size** for large networks (1000+ peers)
2. **Adjust flush interval** for better performance (60s)
3. **Disable geo lookup** if not needed
4. **Use memory storage** for testing environments

## Files

- `types.go` - Core data structures
- `scorer.go` - Scoring algorithm
- `storage.go` - Persistence layer
- `manager.go` - Main coordinator
- `config.go` - Configuration system
- `metrics.go` - Metrics tracking
- `monitor.go` - Health and alerting
- `http_handlers.go` - HTTP API
- `cli.go` - Command-line interface
- `example_integration.go` - Integration examples

## Documentation

- [P2P Security Guide](../../docs/P2P_SECURITY.md) - Comprehensive documentation
- [Configuration Reference](../config/p2p_security.toml) - Config file with comments

## License

Copyright © 2025 PAW Chain. Licensed under Apache License 2.0.
