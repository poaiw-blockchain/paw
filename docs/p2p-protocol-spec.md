# PAW P2P Network Protocol Specification

## Overview

This document describes the complete P2P networking protocol implementation for the PAW blockchain, including all network communication, peer discovery, and connection management protocols.

---

## 1. Network Protocol Design

### 1.1 Message Format

All messages use the following wire format:

```
[4 bytes: total length][2 bytes: msgType length][msgType string][payload data]
```

**Structure:**
- **Total Length** (4 bytes, big-endian): Length of msgType length + msgType + payload
- **MsgType Length** (2 bytes, big-endian): Length of the message type string
- **MsgType** (variable): Message type identifier (e.g., "block", "tx", "ping")
- **Payload** (variable): Message-specific data

**Example:**
```
Message: type="block", data=[100 bytes]
Wire format: [0x00 0x00 0x00 0x69][0x00 0x05]["block"][100 bytes]
Total: 4 + 2 + 5 + 100 = 111 bytes
```

### 1.2 Handshake Protocol

**Format:**
```
[1 byte: protocol version][32 bytes: chain ID][32 bytes: node ID]
Total: 65 bytes
```

**Flow:**
1. Initiator sends handshake
2. Responder validates protocol version and chain ID
3. Responder sends handshake response
4. Initiator validates response
5. Connection established

**Protocol Version:** `0x01` (current)

**Validation Rules:**
- Protocol version must match exactly
- Chain ID must match (prevents cross-chain connections)
- Node ID is informational (for peer identification)

---

## 2. Peer Discovery Protocol

### 2.1 Seed Node Crawling

**Request Format:**
```
[1 byte: message type = 0x01 (peer request)]
```

**Response Format:**
```
[2 bytes: peer count]
[For each peer:]
  [1 byte: ID length]
  [ID bytes]
  [2 bytes: port]
  [1 byte: IP length]
  [IP bytes]
```

**Limits:**
- Maximum peer count per response: 1000
- Maximum ID length: 128 bytes
- Maximum IP length: 64 bytes

**Example Response:**
```
[0x00 0x03] // 3 peers
// Peer 1
[0x10]["node-abc-123..."][0x68 0x30]["0x08"]["1.2.3.4"]
// Peer 2
[0x10]["node-def-456..."][0x68 0x30]["0x08]["5.6.7.8"]
// Peer 3
[0x10]["node-ghi-789..."][0x68 0x30]["0x08]["9.10.11.12"]
```

### 2.2 Default Seed Nodes

**Production Seeds:**
```
DNS-based (Primary):
- seed1.paw.network:26656
- seed2.paw.network:26656
- seed3.paw.network:26656

IP-based Fallbacks:
- 35.184.142.101:26656 (US East)
- 34.89.156.233:26656  (Europe West)
- 35.201.123.45:26656  (Asia Pacific)

Regional Seeds:
- seed-us-west.paw.network:26656
- seed-eu-central.paw.network:26656
- seed-asia-east.paw.network:26656
```

---

## 3. Connection Management

### 3.1 Connection Timeouts

| Operation | Timeout | Description |
|-----------|---------|-------------|
| TCP Dial | 5 seconds | Time to establish TCP connection |
| Handshake | 10 seconds | Time to complete protocol handshake |
| Read | 30 seconds | Idle timeout for reading messages |
| Write | 5 seconds | Timeout for sending messages |
| Keep-alive | 30 seconds | Interval between keep-alive checks |

### 3.2 Connection Pooling

**Limits:**
- Maximum peers: 100 (configurable)
- Maximum inbound: 50
- Maximum outbound: 50
- Minimum outbound: 10 (maintained automatically)

**Connection States:**
- `Connecting`: TCP dial in progress
- `Handshaking`: Protocol handshake in progress
- `Connected`: Active connection with message flow
- `Disconnecting`: Graceful shutdown in progress
- `Disconnected`: Connection closed

### 3.3 Reconnection Logic

**Exponential Backoff:**
```
backoff = min(2^attempts * 1 second, 10 minutes)
```

**Example Backoff Sequence:**
- Attempt 1: 2 seconds
- Attempt 2: 4 seconds
- Attempt 3: 8 seconds
- Attempt 4: 16 seconds
- Attempt 5: 32 seconds
- Attempt 6+: 10 minutes (max)

**Persistent Peers:**
- Always attempt reconnection
- No maximum attempt limit
- Never removed from peer list

---

## 4. Error Handling

### 4.1 Network Errors

**Detected Errors:**
- Connection reset by peer
- Broken pipe
- Connection refused
- Timeout
- Connection closed

**Response Actions:**
```
1. Log error with context
2. Close connection cleanly
3. Remove peer from active list
4. Update peer reputation (if enabled)
5. Schedule reconnection (for persistent peers)
```

### 4.2 Protocol Errors

**Validation Failures:**
- Protocol version mismatch → Reject connection
- Chain ID mismatch → Reject connection
- Invalid message format → Drop message, log warning
- Message too large (>10 MB) → Drop message, disconnect peer

---

## 5. Implementation Details

### 5.1 Thread Safety

**Mechanisms:**
- `sync.RWMutex` for peer map access
- `sync.Mutex` for per-connection writes (WriteMu)
- Atomic operations for statistics
- Channel-based communication for dial queue

**Critical Sections:**
```go
// Reading peer list
pm.mu.RLock()
defer pm.mu.RUnlock()

// Modifying peer list
pm.mu.Lock()
defer pm.mu.Unlock()

// Writing to connection
conn.WriteMu.Lock()
defer conn.WriteMu.Unlock()
```

### 5.2 Message Reading Loop

**Per-Peer Goroutine:**
```go
1. Set read deadline (30 seconds)
2. Read message header (4 bytes)
3. Parse message length
4. Validate length (<10 MB)
5. Read full message
6. Update statistics
7. Parse message type and data
8. Process message (upper layers)
9. Repeat or exit on error
```

**Graceful Shutdown:**
- Context cancellation propagates to all goroutines
- Connections closed cleanly
- Wait group ensures all goroutines complete

---

## 6. Security Considerations

### 6.1 Input Validation

**All inputs validated:**
- Message length limits enforced
- String lengths bounded
- Peer counts limited
- IP addresses validated
- Ports validated (1-65535)

### 6.2 Resource Limits

**Protection Mechanisms:**
- Maximum message size: 10 MB
- Maximum peers: 100
- Dial queue size: 100
- Result queue size: 100
- Connection timeout prevents hanging

### 6.3 Reputation Integration

**Peer Scoring:**
- Failed connections decrease score
- Successful connections increase score
- Misbehavior triggers immediate ban
- Low reputation peers auto-rejected

---

## 7. Statistics and Monitoring

### 7.1 Tracked Metrics

**Per-Peer:**
- Bytes sent/received
- Messages sent/received
- Connection duration
- Last activity timestamp

**Global:**
- Total connections
- Failed dials
- Successful dials
- Rejected inbound connections
- Peer discovery stats

### 7.2 Logging

**Log Levels:**
- `DEBUG`: Message flow, dial attempts
- `INFO`: Connections, disconnections, bootstrap
- `WARN`: Protocol violations, invalid messages
- `ERROR`: Network failures, critical errors

---

## 8. Testing Recommendations

### 8.1 Unit Tests

**Coverage Areas:**
- Message serialization/deserialization
- Handshake validation
- Error handling
- Timeout behavior
- Connection pooling limits

### 8.2 Integration Tests

**Scenarios:**
- Multi-peer network setup
- Peer discovery workflow
- Connection failure recovery
- Message broadcast
- Network partition recovery

### 8.3 Load Tests

**Metrics:**
- Connections per second
- Messages per second
- Memory usage under load
- CPU usage under load
- Connection stability over time

---

## 9. Protocol Versions

### Version 1 (0x01) - Current

**Features:**
- Basic TCP connectivity
- Simple handshake protocol
- Message framing
- Peer discovery via seeds
- Connection management
- Keep-alive support

### Future Enhancements

**Planned:**
- Multiplexing (multiple streams per connection)
- Message compression
- Encryption (TLS/noise protocol)
- Advanced peer exchange (PEX)
- NAT traversal support
- IPv6 support enhancement

---

## 10. Configuration

### 10.1 Default Values

```go
MaxInboundPeers:  50
MaxOutboundPeers: 50
MaxPeers:         100
MinOutboundPeers: 10
DialTimeout:      5 * time.Second
HandshakeTimeout: 10 * time.Second
ReadTimeout:      30 * time.Second
WriteTimeout:     5 * time.Second
PingInterval:     30 * time.Second
InactivityTimeout: 5 * time.Minute
```

### 10.2 Production Tuning

**High-Traffic Nodes:**
```go
MaxPeers:         200
MaxInboundPeers:  100
MaxOutboundPeers: 100
```

**Low-Resource Nodes:**
```go
MaxPeers:         20
MaxInboundPeers:  10
MaxOutboundPeers: 10
```

---

## Appendix A: Wire Format Examples

### A.1 Handshake Example

```
Outgoing handshake:
[0x01] // protocol version
[0x70 0x61 0x77 0x2d 0x6d ... 32 bytes] // "paw-mainnet" padded
[0x6e 0x6f 0x64 0x65 0x2d ... 32 bytes] // "node-abc123" padded

Response:
[0x01] // protocol version matches
[0x70 0x61 0x77 0x2d 0x6d ... 32 bytes] // same chain ID
[0x6e 0x6f 0x64 0x65 0x2d ... 32 bytes] // peer's node ID
```

### A.2 Message Example

```
Block message:
Type: "block"
Payload: [serialized block data, 1024 bytes]

Wire format:
[0x00 0x00 0x04 0x05] // total length = 1029 (2 + 5 + 1024)
[0x00 0x05]           // type length = 5
[0x62 0x6c 0x6f 0x63 0x6b] // "block"
[... 1024 bytes of block data ...]
```

---

## Appendix B: Error Codes

| Code | Name | Description |
|------|------|-------------|
| 0x01 | ProtocolVersionMismatch | Incompatible protocol versions |
| 0x02 | ChainIDMismatch | Different blockchain networks |
| 0x03 | HandshakeTimeout | Handshake took too long |
| 0x04 | MessageTooLarge | Message exceeds size limit |
| 0x05 | InvalidMessageFormat | Malformed message |
| 0x06 | PeerLimitReached | Cannot accept more peers |
| 0x07 | ReputationRejection | Peer rejected by reputation system |
| 0x08 | NetworkError | Generic network failure |

---

## Document Version

- **Version:** 1.0
- **Date:** 2025-11-25
- **Status:** Production Ready

---

## References

- PAW P2P Implementation: `/p2p/`
- Discovery Service: `/p2p/discovery/`
- Protocol Messages: `/p2p/protocol/messages.go`
- Peer Manager: `/p2p/discovery/peer_manager.go`
