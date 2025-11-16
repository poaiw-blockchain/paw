# P2P Networking Issues - Quick Reference Checklist

## Critical Issues Summary

### CRITICAL PATH ISSUES (Must fix in order to get network working)

#### Phase 1: Network Foundation (5-7 days)

- [ ] **Peer Discovery Implementation**
  - [ ] Bootstrap node handling
  - [ ] DHT implementation or libp2p integration
  - [ ] Peer address book
  - [ ] Files: `p2p/discovery/` directory (5-6 files, 2,500-3,500 lines)
- [ ] **Protocol Definition & Handlers**
  - [ ] Create `proto/paw/p2p/v1/` directory with .proto files
  - [ ] Define core P2P message types
  - [ ] Implement message handler interface
  - [ ] Files: Protocol buffers + `p2p/protocol/` (4-5 files, 2,000-3,000 lines)

- [ ] **Basic Connection Establishment**
  - [ ] TCP connection handling
  - [ ] Peer handshake protocol
  - [ ] Connection state management
  - [ ] Files: `p2p/peer/` directory (3-4 files, 1,500-2,000 lines)

#### Phase 2: Core Network (5-7 days)

- [ ] **Gossip & Block Propagation**
  - [ ] Gossip protocol
  - [ ] Block relay mechanism
  - [ ] Transaction propagation
  - [ ] Message deduplication
  - [ ] Files: `p2p/gossip/` directory (4-5 files, 2,000-3,000 lines)

- [ ] **TLS Encryption & Security**
  - [ ] TLS server setup
  - [ ] Certificate management
  - [ ] Connection encryption
  - [ ] Peer authentication
  - [ ] Files: `p2p/security/` directory (3-4 files, 1,500-2,000 lines)

- [ ] **Rate Limiting Enforcement**
  - [ ] Implement rate limiter (config exists but not used)
  - [ ] Connection limit enforcement
  - [ ] Message rate limiting
  - [ ] Files: Updates to `p2p/security/` (500-800 lines)

#### Phase 3: Integration (3-5 days)

- [ ] **Application Integration**
  - [ ] Initialize reputation system in `app/app.go`
  - [ ] Register peer event callbacks
  - [ ] Initialize P2P server
  - [ ] Files: Changes to `app/app.go` (50-100 lines)

- [ ] **HTTP API Integration**
  - [ ] Register reputation HTTP handlers in `api/server.go`
  - [ ] Add routes for P2P endpoints
  - [ ] Files: Changes to `api/server.go` (20-30 lines)

- [ ] **CLI Integration**
  - [ ] Create `cmd/pawd/cmd/reputation.go`
  - [ ] Register CLI commands with cobra
  - [ ] Files: `cmd/pawd/cmd/reputation.go` (200-300 lines)

#### Phase 4: Support Components (5-7 days)

- [ ] **Stream Multiplexing**
  - [ ] Implement yamux or mplex
  - [ ] Multi-channel support
  - [ ] Files: `p2p/multiplexing/` (1,500-2,000 lines)

- [ ] **Peer Connection Manager**
  - [ ] Track active connections
  - [ ] Enforce connection limits
  - [ ] Manage peer lifecycle
  - [ ] Files: Updates to `p2p/peer/` (800-1,200 lines)

- [ ] **Message Router**
  - [ ] Route messages to handlers
  - [ ] Handler registration
  - [ ] Error handling
  - [ ] Files: Updates to `p2p/protocol/` (500-800 lines)

- [ ] **Geographic/ASN Validation**
  - [ ] Implement geo lookup (currently configured but not used)
  - [ ] ASN-based limits
  - [ ] Diversity enforcement
  - [ ] Files: `p2p/reputation/geolookup.go` (500-1,000 lines)

- [ ] **Network Metrics**
  - [ ] Bandwidth monitoring
  - [ ] Connection latency tracking
  - [ ] Throughput metrics
  - [ ] Files: `p2p/metrics/` (800-1,200 lines)

#### Phase 5: Testing & Hardening (7-10 days)

- [ ] **Reputation System Tests**
  - [ ] Unit tests for all 10 reputation files
  - [ ] Tests for scorer, manager, storage
  - [ ] Files: `p2p/reputation/*_test.go` (800-1,200 lines)

- [ ] **Peer Discovery Tests**
  - [ ] Bootstrap mechanism tests
  - [ ] DHT tests
  - [ ] Address book tests
  - [ ] Files: `p2p/discovery/*_test.go` (800-1,200 lines)

- [ ] **Protocol Handler Tests**
  - [ ] Message encoding/decoding tests
  - [ ] Handler registration tests
  - [ ] Message routing tests
  - [ ] Files: `p2p/protocol/*_test.go` (800-1,200 lines)

- [ ] **Gossip Tests**
  - [ ] Broadcast tests
  - [ ] Deduplication tests
  - [ ] Propagation tests
  - [ ] Files: `p2p/gossip/*_test.go` (600-1,000 lines)

- [ ] **Integration Tests**
  - [ ] Multi-node network tests
  - [ ] Message delivery tests
  - [ ] Block propagation tests
  - [ ] Files: `tests/p2p_integration_test.go` (800-1,200 lines)

- [ ] **E2E Network Tests**
  - [ ] Full network startup
  - [ ] Network sync tests
  - [ ] Peer discovery tests
  - [ ] Files: `tests/e2e/p2p_e2e_test.go` (800-1,200 lines)

---

## All 16 Critical Issues Checklist

### Issue C1: Peer Discovery

- [ ] Bootstrap peer handling
- [ ] DHT implementation
- [ ] Peer address book
- [ ] DNS seed support
- [ ] Address validation
- **Location:** Missing `p2p/discovery/`
- **Estimated Effort:** 3-5 days
- **Lines:** 2,500-3,500

### Issue C2: Protocol Handlers

- [ ] Proto file definitions
- [ ] Handler interface
- [ ] Handler registration
- [ ] Message routing
- [ ] Codec/serialization
- **Location:** Missing `p2p/protocol/` and `proto/paw/p2p/v1/`
- **Estimated Effort:** 3-4 days
- **Lines:** 2,000-3,000

### Issue C3: Gossip/Broadcast

- [ ] Gossip algorithm
- [ ] Block relay
- [ ] Transaction propagation
- [ ] Deduplication
- [ ] Message validation
- **Location:** Missing `p2p/gossip/`
- **Estimated Effort:** 2-3 days
- **Lines:** 2,000-3,000

### Issue C4: Connection Management

- [ ] TCP connection setup
- [ ] Peer handshake
- [ ] Stream multiplexing
- [ ] Connection lifecycle
- [ ] Keep-alive logic
- **Location:** Missing `p2p/peer/`
- **Estimated Effort:** 2-3 days
- **Lines:** 2,500-3,500

### Issue C5: TLS Encryption

- [ ] TLS server setup
- [ ] Certificate loading
- [ ] Connection encryption
- [ ] Peer authentication
- [ ] Session management
- **Location:** Missing `p2p/security/`
- **Estimated Effort:** 2-3 days
- **Lines:** 1,500-2,000

### Issue C6: App Integration (Reputation)

- [ ] Initialize reputation manager in `app.go`
- [ ] Register peer event handlers
- [ ] Wire up P2P events
- **Location:** `app/app.go`
- **Estimated Effort:** 1-2 days
- **Lines:** 50-100

### Issue C7: HTTP Routes Registration

- [ ] Register reputation HTTP handlers
- [ ] Add routes to mux
- [ ] Enable REST API endpoints
- **Location:** `api/server.go`
- **Estimated Effort:** 1 day
- **Lines:** 20-30

### Issue C8: Protocol Buffer Files

- [ ] Define core P2P messages
- [ ] Define consensus messages
- [ ] Define sync messages
- [ ] Define gossip messages
- [ ] Define query messages
- **Location:** `proto/paw/p2p/v1/`
- **Estimated Effort:** 1-2 days
- **Lines:** 300-500

### Issue C9: Network Tests

- [ ] Reputation system tests
- [ ] Discovery tests
- [ ] Protocol handler tests
- [ ] Gossip tests
- [ ] Integration tests
- [ ] E2E tests
- **Location:** All `*_test.go` files
- **Estimated Effort:** 3-5 days
- **Lines:** 3,000-5,000
- **Current Coverage:** 0%

### Issue C10: Empty Certs Directory

- [ ] Generate TLS server certificate
- [ ] Generate client certificate
- [ ] Generate CA certificate
- [ ] Store private keys securely
- **Location:** `certs/` (empty)
- **Estimated Effort:** 1 day
- **Scripts Exist:** Yes (generate-tls-certs.sh/.ps1)

### Issue C11: Rate Limiting Enforcement

- [ ] Implement rate limiter (config exists: p2p_security.toml)
- [ ] Per-peer message limit
- [ ] Per-peer block limit
- [ ] Connection limit enforcement
- **Location:** `p2p/security/rate_limiter.go` (missing)
- **Estimated Effort:** 1-2 days
- **Lines:** 500-800

### Issue C12: Stream Multiplexing

- [ ] Implement yamux or mplex
- [ ] Multi-channel support
- [ ] Stream lifecycle management
- [ ] Flow control
- **Location:** `p2p/multiplexing/`
- **Estimated Effort:** 2 days
- **Lines:** 1,500-2,000

### Issue C13: Peer Connection Manager

- [ ] Track active peer connections
- [ ] Enforce inbound/outbound quotas
- [ ] Manage connection lifecycle
- [ ] Ban peer enforcement
- **Location:** `p2p/peer/manager.go` (missing)
- **Estimated Effort:** 1-2 days
- **Lines:** 800-1,200

### Issue C14: Message Router Implementation

- [ ] Create message router
- [ ] Handler lookup and dispatch
- [ ] Error handling
- [ ] Message validation pipeline
- **Location:** `p2p/protocol/router.go` (missing)
- **Estimated Effort:** 2 days
- **Lines:** 500-800

### Issue C15: Geographic Lookup Implementation

- [ ] Geo lookup service integration
- [ ] ASN lookup
- [ ] Country enforcement
- [ ] Diversity validation
- **Location:** `p2p/reputation/geolookup.go` (missing)
- **Config Exists:** Yes (p2p_security.toml)
- **Estimated Effort:** 1 day
- **Lines:** 500-1,000

### Issue C16: Network Metrics

- [ ] Bandwidth metrics
- [ ] Latency tracking
- [ ] Throughput metrics
- [ ] Peer churn monitoring
- **Location:** `p2p/metrics/`
- **Estimated Effort:** 1-2 days
- **Lines:** 800-1,200

---

## Integration Points Checklist

### Application Integration

- [ ] **File:** `app/app.go`
  - [ ] Import reputation package
  - [ ] Initialize reputation manager
  - [ ] Store reference in app struct
  - [ ] Register peer event callbacks
  - [ ] Add shutdown hook
  - **Lines:** ~50-100

### API Server Integration

- [ ] **File:** `api/server.go`
  - [ ] Import reputation handlers
  - [ ] Create HTTP handlers
  - [ ] Register routes with mux
  - [ ] Test endpoint accessibility
  - **Lines:** ~20-30

### CLI Integration

- [ ] **File:** `cmd/pawd/cmd/reputation.go` (NEW FILE)
  - [ ] Create reputation command group
  - [ ] Add subcommands: list, show, stats, ban, unban, whitelist, export
  - [ ] Wire into root command
  - [ ] Test all commands
  - **Lines:** ~200-300

### Main Server Integration

- [ ] **File:** `cmd/pawd/cmd/start.go` or equivalent
  - [ ] Initialize P2P server
  - [ ] Start peer discovery
  - [ ] Start connection manager
  - [ ] Register cleanup on shutdown

### Network Event Callbacks

- [ ] Peer connected event handler
- [ ] Peer disconnected event handler
- [ ] Message received event handler
- [ ] Block received event handler
- [ ] Protocol violation event handler

---

## Configuration Checklist

### Existing Configuration Files

- [x] `p2p/config/p2p_security.toml` (Exists, mostly complete)
  - [x] Reputation settings
  - [x] Storage settings
  - [x] Scoring weights
  - [x] Ban settings
  - [x] Security settings
  - [ ] Implementation code (MISSING)

### Configuration Implementation Tasks

- [ ] Load config on startup
- [ ] Validate configuration values
- [ ] Support environment variable overrides
- [ ] Dynamic config reloading
- [ ] Config migration support

---

## Files to Create or Modify

### New Directories to Create

```
p2p/
├── discovery/          (NEW) - Peer discovery
├── protocol/           (NEW) - Protocol handlers
├── gossip/             (NEW) - Gossip/broadcast
├── peer/               (NEW) - Connection management
├── security/           (NEW) - TLS/encryption
├── multiplexing/       (NEW) - Stream multiplexing
├── metrics/            (NEW) - Network metrics
└── proto/              (NEW) - Proto files location

proto/paw/p2p/v1/      (NEW) - Protocol buffer definitions
```

### New Files to Create

- `p2p/discovery/discovery.go` (250-400 lines)
- `p2p/discovery/bootstrap.go` (300-500 lines)
- `p2p/discovery/dht.go` (400-600 lines)
- `p2p/protocol/handler.go` (200-300 lines)
- `p2p/protocol/processor.go` (300-500 lines)
- `p2p/protocol/router.go` (250-400 lines)
- `p2p/gossip/gossip.go` (300-500 lines)
- `p2p/gossip/broadcast.go` (250-400 lines)
- `p2p/peer/connector.go` (300-500 lines)
- `p2p/peer/stream.go` (200-300 lines)
- `p2p/security/tls.go` (300-500 lines)
- `p2p/security/auth.go` (200-300 lines)
- ... + many more test files

### Files to Modify

- `app/app.go` - Initialize P2P/reputation (50-100 lines)
- `api/server.go` - Register HTTP routes (20-30 lines)
- `cmd/pawd/cmd/root.go` - Register CLI commands (5-10 lines)
- `cmd/pawd/cmd/start.go` - Start P2P server (20-30 lines)

---

## Timeline Estimate

### Aggressive (2 full-time developers)

- Week 1: Peer discovery + protocol handlers (days 1-5)
- Week 2: Connections + gossip + TLS (days 6-10)
- Week 3: Integration + metrics (days 11-15)
- Week 4-5: Testing + hardening (days 16-25)
- **Total:** 25 days (5 weeks)

### Realistic (2 developers with other responsibilities)

- Weeks 1-2: Peer discovery + protocol handlers (10 days)
- Weeks 2-3: Connections + gossip + TLS (10 days)
- Weeks 3-4: Integration + metrics (8 days)
- Weeks 4-5: Testing + hardening (10 days)
- **Total:** 38 days (7-8 weeks with part-time)

### Conservative (1 developer)

- Weeks 1-4: Peer discovery + protocol handlers (20 days)
- Weeks 4-6: Connections + gossip + TLS (15 days)
- Weeks 6-7: Integration + metrics (8 days)
- Weeks 7-9: Testing + hardening (10 days)
- **Total:** 53 days (10-11 weeks)

---

## Dependency Graph for Checklist

```
Foundation (Do these first):
  ├─ C8: Protocol Buffer Files
  ├─ C1: Peer Discovery
  └─ C2: Protocol Handlers

Network (Depends on Foundation):
  ├─ C4: Connection Management
  ├─ C5: TLS Encryption
  └─ C10: Certificates

Transport (Depends on Network):
  ├─ C3: Gossip/Broadcast
  ├─ C11: Rate Limiting
  ├─ C12: Stream Multiplexing
  └─ C13: Peer Manager

Integration (Depends on Transport):
  ├─ C6: App Integration
  ├─ C7: HTTP Routes
  ├─ C14: Message Router
  ├─ C15: Geo Lookup
  └─ C16: Network Metrics

Testing (Last):
  └─ C9: Network Tests
```

---

## Success Criteria

### Phase 1 Complete

- [ ] Nodes can discover each other
- [ ] Nodes can bootstrap from known peers
- [ ] Address book is persistent
- [ ] Test coverage: >80% for discovery

### Phase 2 Complete

- [ ] Nodes can connect to each other
- [ ] Connections are encrypted with TLS
- [ ] Message handlers can process messages
- [ ] Protocol negotiation works
- [ ] Test coverage: >80% for protocols

### Phase 3 Complete

- [ ] Blocks propagate through network
- [ ] Transactions propagate
- [ ] Consensus can be achieved
- [ ] Network health is monitored
- [ ] Test coverage: >80% for gossip

### Phase 4 Complete

- [ ] Reputation system is active
- [ ] Bad peers are rejected
- [ ] Rate limiting prevents DoS
- [ ] HTTP API is accessible
- [ ] CLI commands work

### Phase 5 Complete

- [ ] All tests pass
- [ ] Load testing successful
- [ ] No security issues found
- [ ] Documentation complete
- [ ] Test coverage: >90% overall

---

## Rollback Plan

If critical issues are discovered during implementation:

1. **Design Issues:** Pause and redesign before continuing
2. **Security Issues:** Fix immediately before proceeding
3. **Performance Issues:** Optimize or refactor affected component
4. **Test Failures:** Debug and fix before moving to next phase

Keep each phase's work in separate branches to allow rollback if needed.

---

_This checklist is the actionable version of the audit report._  
_Use this to track progress as work is completed._
