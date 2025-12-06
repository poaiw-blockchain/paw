# IBC Packet Replay Attack Vulnerability

---
status: pending
priority: p1
issue_id: "003"
tags: [security, ibc, dex, oracle, compute, critical]
dependencies: []
---

## Problem Statement

IBC packet nonce validation in `x/dex/ibc_module.go:244-252` checks for nonce uniqueness but lacks:
1. Timestamp-based expiration for old packets
2. Monotonic sequence enforcement
3. Protection against nonce reuse after state pruning

**Why it matters:** Attackers could replay old valid packets to execute duplicate cross-chain swaps, oracle updates, or compute job results.

## Findings

### Source: security-sentinel & architecture-strategist agents

**Location:** `/home/decri/blockchain-projects/paw/x/dex/ibc_module.go:244-252`

```go
packetNonce := im.packetNonce(packetData)
if packetNonce == 0 {
    return channeltypes.NewErrorAcknowledgement(errorsmod.Wrap(types.ErrInvalidPacket, "packet nonce missing"))
}

sender := im.packetSender(packet, packetData)
if err := im.keeper.ValidateIncomingPacketNonce(ctx, packet.SourceChannel, sender, packetNonce); err != nil {
    return channeltypes.NewErrorAcknowledgement(err)
}
```

**Attack Scenario:**
1. User executes valid cross-chain swap with nonce=100
2. State pruning removes old nonce records (e.g., after 1000 blocks)
3. Attacker replays packet with nonce=100
4. Validation passes because nonce no longer stored
5. Duplicate swap executes - double the tokens

**Additional Risk - Missing Nonce Storage Key:**
Architecture-strategist found no visible key prefix for nonce storage in `x/dex/types/keys.go`, suggesting nonce tracking may not be properly persisted.

## Proposed Solutions

### Option A: Timestamp + Monotonic Nonce (Recommended)
**Pros:** Comprehensive protection, prevents old packet replay
**Cons:** Slightly more complex
**Effort:** Medium
**Risk:** Low

```go
// In x/dex/types/keys.go
PacketNonceKeyPrefix = []byte{0x0C}

func GetPacketNonceKey(channelID, sender string) []byte {
    key := append(PacketNonceKeyPrefix, []byte(channelID)...)
    key = append(key, '/')
    key = append(key, []byte(sender)...)
    return key
}

// In keeper
func (k Keeper) ValidateIncomingPacketNonce(ctx sdk.Context, channelID, sender string, nonce uint64, timestamp int64) error {
    store := ctx.KVStore(k.storeKey)
    key := types.GetPacketNonceKey(channelID, sender)

    // Check timestamp not too old (24 hour max)
    blockTime := ctx.BlockTime().Unix()
    if blockTime - timestamp > 86400 {
        return types.ErrPacketExpired
    }

    // Check nonce is strictly increasing
    bz := store.Get(key)
    if bz != nil {
        lastNonce := binary.BigEndian.Uint64(bz)
        if nonce <= lastNonce {
            return types.ErrInvalidNonce.Wrapf("nonce %d <= last %d", nonce, lastNonce)
        }
    }

    // Store new nonce
    nonceBz := make([]byte, 8)
    binary.BigEndian.PutUint64(nonceBz, nonce)
    store.Set(key, nonceBz)

    return nil
}
```

### Option B: Packet Hash with Expiration
**Pros:** No nonce coordination needed
**Cons:** Higher storage requirements
**Effort:** Medium
**Risk:** Medium

### Option C: Sliding Window Nonce Validation
**Pros:** Bounded storage
**Cons:** Complex implementation
**Effort:** Large
**Risk:** Medium

## Recommended Action

**Implement Option A** with:
1. Add nonce storage key prefix
2. Implement monotonic nonce validation
3. Add timestamp-based expiration (24 hours)
4. Apply to ALL three IBC modules (DEX, Oracle, Compute)

## Technical Details

**Affected Files:**
- `x/dex/types/keys.go`
- `x/dex/keeper/keeper.go`
- `x/dex/ibc_module.go`
- `x/oracle/types/keys.go`
- `x/oracle/keeper/keeper.go`
- `x/oracle/ibc_module.go`
- `x/compute/types/keys.go`
- `x/compute/keeper/keeper.go`
- `x/compute/ibc_module.go`

**Database Changes:** Add nonce storage key prefix to each module

## Acceptance Criteria

- [ ] Nonce key prefix added to all three modules
- [ ] Monotonic nonce validation implemented
- [ ] Timestamp expiration (24h) added
- [ ] Nonces exported in genesis for chain restart preservation
- [ ] Test: replay same packet, verify rejection
- [ ] Test: replay expired packet, verify rejection
- [ ] Test: out-of-order nonce, verify rejection

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-05 | Created | Identified by security-sentinel and architecture-strategist agents |

## Resources

- IBC Specification: Packet handling requirements
- Similar issue in ROADMAP_PRODUCTION.md: CRITICAL-1 (marked completed, but implementation may be incomplete)
