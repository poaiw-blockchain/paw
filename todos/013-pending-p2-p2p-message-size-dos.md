# P2P Message Size DoS Vulnerability

---
status: pending
priority: p2
issue_id: "013"
tags: [security, p2p, dos, network]
dependencies: []
---

## Problem Statement

No message size limit before deserialization in `p2p/node.go:382-436`. Attackers can send extremely large messages to cause memory exhaustion.

**Why it matters:** Node crashes from OOM, network partition possible.

## Findings

### Source: security-sentinel agent

**Location:** `/home/decri/blockchain-projects/paw/p2p/node.go:382-436`

```go
// Line 382-396: Reads unlimited size from network
totalLen := uint32(2 + len(msgTypeBytes) + len(data))

// Create message buffer
buf := make([]byte, 4+2+len(msgTypeBytes)+len(data))
// No size validation before allocation!
```

**Attack Scenario:**
1. Attacker connects to node
2. Sends message with 10GB size header
3. Node attempts to allocate 10GB buffer
4. Node crashes with OOM
5. Repeat for multiple nodes = network partition

## Proposed Solutions

### Option A: Size Limit Before Allocation (Recommended)
**Pros:** Simple, effective
**Cons:** None
**Effort:** Small
**Risk:** Low

```go
const MaxP2PMessageSize = 10 * 1024 * 1024 // 10MB

func (n *Node) readMessage(conn net.Conn) ([]byte, error) {
    // Read size header first (4 bytes)
    sizeBuf := make([]byte, 4)
    if _, err := io.ReadFull(conn, sizeBuf); err != nil {
        return nil, err
    }

    size := binary.BigEndian.Uint32(sizeBuf)

    // VALIDATE SIZE BEFORE ALLOCATION
    if size > MaxP2PMessageSize {
        n.banPeer(conn.RemoteAddr(), "oversized message")
        return nil, ErrMessageTooLarge
    }

    // Now safe to allocate
    buf := make([]byte, size)
    // ...
}
```

### Option B: Streaming with Chunked Validation
**Pros:** Handles legitimate large messages
**Cons:** More complex
**Effort:** Medium
**Risk:** Low

## Recommended Action

**Implement Option A** - add size validation before buffer allocation.

## Technical Details

**Affected Files:**
- `p2p/node.go`

**Database Changes:** None

## Acceptance Criteria

- [ ] Add MAX_MESSAGE_SIZE constant (10MB)
- [ ] Validate size BEFORE buffer allocation
- [ ] Disconnect and ban peers sending oversized messages
- [ ] Test: send 100MB message, verify rejection with minimal memory use
- [ ] Log oversized message attempts for monitoring

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-05 | Created | Identified by security-sentinel agent |

## Resources

- libp2p message size handling
- CometBFT P2P layer implementation
