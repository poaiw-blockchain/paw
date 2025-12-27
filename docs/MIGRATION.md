# PAW Migration Guide

Breaking changes and migration steps for upgrading between versions.

## v1 to v2 Migration

### 1. Store Key Namespace Changes (DATA-1 to DATA-5)

**What changed:** All module store keys now use a 2-byte namespace prefix.

**Why:** Prevents key collisions between modules and enables efficient store iteration.

**Before:**
```go
PoolKeyPrefix = []byte{0x01}
```

**After:**
```go
// DEX module uses 0x02 namespace
PoolKeyPrefix = []byte{0x02, 0x01}
```

**Migration:**
- Run the v2 upgrade handler which automatically migrates keys
- The handler rebuilds indexes and validates state consistency

---

### 2. Circuit Breaker Duration Parameterized (CODE-3)

**What changed:** Circuit breaker duration moved from hardcoded constant to chain params.

**Why:** Allows governance to adjust without chain upgrade.

**Before:**
```go
const CircuitBreakerDuration = 3600 // hardcoded 1 hour
```

**After:**
```go
params.CircuitBreakerDurationSeconds // governance-controlled
```

**Migration:**
- Default value: `3600` (1 hour)
- Existing chains: params auto-initialized during upgrade
- Custom duration: submit governance proposal to change

---

### 3. IBC Nonce Key Moved to Shared Package (CODE-5)

**What changed:** `GetIBCPacketNonceKey` function relocated to `x/shared/ibc`.

**Why:** Multiple modules (dex, oracle, compute) need IBC replay protection.

**Before:**
```go
import "github.com/paw-chain/paw/x/dex/types"
key := types.GetIBCPacketNonceKey(channelID, sender)
```

**After:**
```go
import sharedibc "github.com/paw-chain/paw/x/shared/ibc"
key := sharedibc.GetIBCPacketNonceKey(prefix, channelID, sender)
```

**Migration:**
- Update imports to use shared package
- Pass module-specific prefix as first argument
- Each module retains its own key prefix (e.g., `[]byte{0x02, 0x16}` for DEX)

---

### 4. HTTPS Required for Provider Endpoints (SEC-8)

**What changed:** Compute module rejects HTTP endpoints (except localhost).

**Why:** Prevents credential leakage and MITM attacks.

**Before:**
```go
// Accepted any HTTP/HTTPS endpoint
endpoint := "http://provider.example.com:8080"
```

**After:**
```go
// Production: HTTPS only
endpoint := "https://provider.example.com:8080"
// Development: HTTP allowed for localhost only
endpoint := "http://localhost:8080"  // OK
endpoint := "http://127.0.0.1:8080"  // OK
```

**Migration:**
- Update all provider endpoint configs to use HTTPS
- Obtain TLS certificates for production endpoints
- Development/testing can continue using localhost with HTTP

---

## Running the Migration

```bash
# Upgrade via cosmovisor (recommended)
cosmovisor upgrade v2

# Manual upgrade
pawd upgrade v2 --home ~/.paw
```

The upgrade handler performs:
1. Key namespace migration for all modules
2. Pool index rebuild and validation
3. Circuit breaker state initialization
4. Params update with new fields
5. Liquidity position validation

## Verification

After upgrade, verify state integrity:

```bash
# Check DEX params
pawd query dex params

# Verify pool indexes
pawd query dex pools

# Check circuit breaker status
pawd query dex circuit-breaker-status --pool-id 1
```
