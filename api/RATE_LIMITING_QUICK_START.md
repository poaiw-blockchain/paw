# Rate Limiting Quick Start Guide

## 5-Minute Setup

### 1. Basic Usage (Default Configuration)

The rate limiter is enabled by default with sensible defaults:

```go
import "github.com/paw-chain/paw/api"

server, err := api.NewServer(clientCtx, api.DefaultConfig())
if err != nil {
    log.Fatal(err)
}

server.Start()
```

That's it! Your API now has:

- IP-based rate limiting (100 req/sec)
- Per-endpoint limits for sensitive operations
- Account-tier based limits
- Adaptive rate limiting
- Audit logging

### 2. Custom Configuration

Create `config/rate_limits.yaml`:

```yaml
enabled: true
default_rps: 50
default_burst: 100

endpoint_limits:
  /api/auth/login:
    rps: 5
    burst: 10
    enabled: true

account_tiers:
  free:
    requests_per_minute: 20
    requests_per_hour: 1000
```

Load and use:

```go
config := api.DefaultConfig()
// config.RateLimitConfig = loadFromYAML("config/rate_limits.yaml")
server, _ := api.NewServer(clientCtx, config)
```

### 3. Monitoring

```bash
# Check rate limiter stats
curl http://localhost:5000/rate-limit/stats

# View audit logs
tail -f ./logs/audit/audit_*.log
```

## Common Configurations

### Strict Security (Banking/Finance)

```yaml
endpoint_limits:
  /api/auth/login:
    rps: 3
    burst: 5
  /api/wallet/send:
    rps: 2
    burst: 3

ip_config:
  auto_block_threshold: 50
  block_duration: 2h
```

### High Throughput (Public API)

```yaml
default_rps: 200
default_burst: 500

endpoint_limits:
  /api/market/stats:
    rps: 500
    burst: 1000
```

### Development (Relaxed)

```yaml
enabled: true
default_rps: 1000
default_burst: 2000

adaptive:
  enabled: false

ip_config:
  enabled: false
```

## Testing Rate Limits

### Test with curl

```bash
# Hit endpoint repeatedly to trigger rate limit
for i in {1..100}; do
  curl -i http://localhost:5000/api/market/stats
done

# Check headers
curl -i http://localhost:5000/api/market/stats | grep "X-RateLimit"
```

### Test with code

```go
func TestRateLimit() {
    for i := 0; i < 100; i++ {
        resp, _ := http.Get("http://localhost:5000/api/market/stats")
        fmt.Printf("Status: %d, Remaining: %s\n",
            resp.StatusCode,
            resp.Header.Get("X-RateLimit-Remaining"))
    }
}
```

## Client-Side Best Practices

### Check Rate Limit Headers

```javascript
async function apiCall() {
  const response = await fetch('/api/endpoint');

  const limit = response.headers.get('X-RateLimit-Limit');
  const remaining = response.headers.get('X-RateLimit-Remaining');
  const reset = response.headers.get('X-RateLimit-Reset');

  console.log(
    `Rate Limit: ${remaining}/${limit}, resets at ${new Date(reset * 1000)}`
  );

  if (response.status === 429) {
    const retryAfter = response.headers.get('Retry-After');
    console.log(`Rate limited! Retry after ${retryAfter} seconds`);
    // Implement exponential backoff
  }

  return response.json();
}
```

### Implement Retry Logic

```javascript
async function apiCallWithRetry(url, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    const response = await fetch(url);

    if (response.status === 429) {
      const retryAfter = parseInt(response.headers.get('Retry-After') || '1');
      await new Promise(resolve => setTimeout(resolve, retryAfter * 1000));
      continue;
    }

    return response.json();
  }

  throw new Error('Max retries exceeded');
}
```

## Troubleshooting

### "Rate limit exceeded" errors

**Check current limits:**

```bash
curl http://localhost:5000/rate-limit/stats
```

**Solutions:**

1. Upgrade account tier
2. Add IP to whitelist
3. Implement request batching
4. Use caching on client side

### Too many auto-blocks

**Check audit logs:**

```bash
grep "auto_blocked" ./logs/audit/audit_*.log
```

**Solutions:**

1. Increase `auto_block_threshold`
2. Review endpoint-specific limits
3. Check for misbehaving clients

### Performance issues

**Run benchmarks:**

```bash
go test -bench=. ./api/... -run "^$"
```

**Solutions:**

1. Decrease `cleanup_interval`
2. Reduce number of tracked limiters
3. Optimize endpoint limits

## Environment Variables

Override configuration via environment:

```bash
export PAW_RATE_LIMIT_ENABLED=true
export PAW_RATE_LIMIT_DEFAULT_RPS=100
export PAW_AUDIT_LOG_DIR=./logs/audit
```

## Account Tiers

Assign tier to users:

```go
// In your authentication handler
claims := &AuthClaims{
    UserID:   user.ID,
    Username: user.Username,
    Tier:     user.Tier, // "free", "premium", "enterprise", "vip"
}
```

The middleware will automatically apply tier-specific limits.

## Whitelist/Blacklist Management

### Add to whitelist programmatically:

```go
config.RateLimitConfig.IPConfig.WhitelistIPs = append(
    config.RateLimitConfig.IPConfig.WhitelistIPs,
    "203.0.113.10",
)
```

### Add to blacklist:

```go
limiter.ipBlacklist.Store("198.51.100.100", time.Now().Add(24*time.Hour))
```

## Monitoring Dashboard

Create a simple monitoring dashboard:

```javascript
// Fetch stats every 5 seconds
setInterval(async () => {
  const stats = await fetch('/rate-limit/stats').then(r => r.json());
  console.log('Active limiters:', stats);

  // Update dashboard UI
  document.getElementById('ip-limiters').textContent = stats.ip_limiters;
  document.getElementById('account-limiters').textContent =
    stats.account_limiters;
  document.getElementById('blacklisted-ips').textContent =
    stats.blacklisted_ips;
}, 5000);
```

## Production Checklist

- [ ] Review and adjust default rate limits
- [ ] Configure endpoint-specific limits
- [ ] Set up account tiers
- [ ] Enable adaptive rate limiting
- [ ] Configure IP whitelist for trusted sources
- [ ] Set up audit log monitoring
- [ ] Test rate limits in staging
- [ ] Monitor performance in production
- [ ] Set up alerts for high rate limit violations
- [ ] Document tier limits for customers

## Support

- Full documentation: `api/RATE_LIMITING.md`
- Implementation details: `IMPLEMENTATION_SUMMARY.md`
- Configuration examples: `config/rate_limits.yaml`

## Next Steps

1. Review `config/rate_limits.yaml` for all configuration options
2. Read `api/RATE_LIMITING.md` for detailed documentation
3. Run tests: `go test ./api/... -run "Test.*Rate"`
4. Monitor stats endpoint: `/rate-limit/stats`
5. Review audit logs for security events
