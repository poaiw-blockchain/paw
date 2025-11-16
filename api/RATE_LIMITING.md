# Advanced Rate Limiting Documentation

## Overview

The PAW API server implements a sophisticated multi-layered rate limiting system that provides fine-grained control over API access. The system includes per-endpoint limits, account-tier based limits, IP-based restrictions, adaptive rate limiting based on user behavior, and comprehensive audit logging.

## Features

### 1. Per-Endpoint Rate Limiting

Different API endpoints can have different rate limits based on their resource intensity and security requirements.

**Example Configuration:**

```yaml
endpoint_limits:
  /api/auth/login:
    path: '/api/auth/login'
    method: 'POST'
    rps: 5
    burst: 10
    enabled: true
    custom_message: 'Login rate limit exceeded.'
```

**Key Features:**

- Method-specific limits (GET, POST, etc.)
- Custom error messages per endpoint
- Burst capacity for handling traffic spikes
- Can be enabled/disabled individually

### 2. Account-Level Rate Limiting

Authenticated users are assigned to tiers (free, premium, enterprise, VIP) with different limits.

**Example Configuration:**

```yaml
account_tiers:
  premium:
    requests_per_minute: 100
    requests_per_hour: 10000
    requests_per_day: 100000
    burst_size: 200
    concurrent_requests: 50
    priority: 5
```

**Tier System:**

- **Free**: 20 req/min, 1,000 req/hour, 10,000 req/day
- **Premium**: 100 req/min, 10,000 req/hour, 100,000 req/day
- **Enterprise**: 1,000 req/min, 100,000 req/hour, 1,000,000 req/day
- **VIP**: 5,000 req/min, 500,000 req/hour, 5,000,000 req/day

### 3. Adaptive Rate Limiting

The system automatically adjusts rate limits based on user behavior patterns.

**Trust Building:**

- Users with consistent successful requests gain trust levels (up to 5)
- Each trust level increases rate limit by 20% (max 2x at trust level 5)
- Trust threshold: 100 consecutive successful requests to level up

**Suspicion System:**

- Users with failed requests accumulate suspicion levels (up to 5)
- Each suspicion level decreases rate limit by 10% (down to 50%)
- Suspicion threshold: 10 consecutive failures to level up
- High suspicion levels trigger security audit logs

**Auto-Recovery:**

- Suspicion levels decay after 24 hours of good behavior
- Trust/suspicion scores decay gradually (configurable interval)

### 4. Burst Protection

Token bucket algorithm with configurable refill rate:

- **Burst Size**: Maximum tokens available for bursts
- **Sustained Rate**: Continuous request rate (tokens per second)
- **Example**: 100 burst, 50/sec sustained allows 100 requests instantly, then 50/sec

### 5. IP-Based Rate Limiting

**Whitelisting:**

```yaml
whitelist_ips:
  - '127.0.0.1'
whitelist_cidrs:
  - '127.0.0.0/8'
  - '::1/128'
```

**Blacklisting:**

```yaml
blacklist_ips:
  - '198.51.100.100'
blacklist_cidrs:
  - '10.0.0.0/8'
```

**Auto-Blocking:**

- Automatically blocks IPs after threshold violations (default: 100)
- Configurable block duration (default: 1 hour)
- Violations reset after successful requests

### 6. Rate Limit Headers

Standard HTTP headers are returned with every response:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 75
X-RateLimit-Reset: 1640000000
Retry-After: 30
```

**Header Descriptions:**

- `X-RateLimit-Limit`: Maximum requests allowed in the current window
- `X-RateLimit-Remaining`: Requests remaining in the current window
- `X-RateLimit-Reset`: Unix timestamp when the limit resets
- `Retry-After`: Seconds to wait before retrying (on 429 responses)

## Configuration

### File Location

Place your configuration file at: `config/rate_limits.yaml`

### Example Configuration

See `config/rate_limits.yaml` for a complete example with all options documented.

### Environment Variables

You can override configuration via environment variables:

```bash
export PAW_RATE_LIMIT_ENABLED=true
export PAW_RATE_LIMIT_DEFAULT_RPS=50
export PAW_RATE_LIMIT_DEFAULT_BURST=100
export PAW_AUDIT_LOG_DIR=./logs/audit
```

### Programmatic Configuration

```go
config := &api.Config{
    RateLimitConfig: &api.RateLimitConfig{
        Enabled:      true,
        DefaultRPS:   50,
        DefaultBurst: 100,
        EndpointLimits: map[string]*api.EndpointLimit{
            "/api/auth/login": {
                Path:    "/api/auth/login",
                Method:  "POST",
                RPS:     5,
                Burst:   10,
                Enabled: true,
            },
        },
    },
}

server, err := api.NewServer(clientCtx, config)
```

## Usage Examples

### Basic Usage

The rate limiter is automatically enabled when the server starts:

```go
server, err := api.NewServer(clientCtx, api.DefaultConfig())
if err != nil {
    log.Fatal(err)
}

err = server.Start()
```

### Monitoring Rate Limiter

Access real-time statistics:

```bash
curl http://localhost:5000/rate-limit/stats
```

Response:

```json
{
  "ip_limiters": 42,
  "account_limiters": 15,
  "behavior_trackers": 10,
  "blacklisted_ips": 2,
  "enabled": true
}
```

### Client-Side Handling

```javascript
async function makeRequest() {
  try {
    const response = await fetch('/api/orders/create', {
      method: 'POST',
      headers: {
        Authorization: 'Bearer YOUR_TOKEN',
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(orderData),
    });

    if (response.status === 429) {
      const retryAfter = response.headers.get('Retry-After');
      console.log(`Rate limited. Retry after ${retryAfter} seconds`);

      // Wait and retry
      await new Promise(resolve => setTimeout(resolve, retryAfter * 1000));
      return makeRequest();
    }

    return await response.json();
  } catch (error) {
    console.error('Request failed:', error);
  }
}
```

## Architecture

### Layered Approach

Rate limits are checked in the following order:

1. **IP Blacklist Check**: Block if IP is blacklisted
2. **IP Whitelist Check**: Allow if IP is whitelisted (skip further checks)
3. **Endpoint Limit Check**: Check endpoint-specific limits
4. **IP Limit Check**: Check IP-based global limits
5. **Account Limit Check**: Check authenticated user's tier limits
6. **Adaptive Adjustment**: Apply trust/suspicion multipliers

### Components

```
┌─────────────────────────────────────────────────────────┐
│                  AdvancedRateLimiter                    │
├─────────────────────────────────────────────────────────┤
│  - IP Limiters (sync.Map)                               │
│  - Account Limiters (sync.Map)                          │
│  - Behavior Trackers (sync.Map)                         │
│  - Endpoint Limiters (map)                              │
│  - Whitelist/Blacklist (CIDR matching)                  │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│           AdvancedRateLimitMiddleware                   │
├─────────────────────────────────────────────────────────┤
│  1. Check rate limit                                    │
│  2. Set response headers                                │
│  3. Track concurrent requests                           │
│  4. Record success/failure for adaptive limiting        │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│                   Audit Logger                          │
├─────────────────────────────────────────────────────────┤
│  - Log rate limit violations                            │
│  - Log suspicious activity                              │
│  - Log IP auto-blocks                                   │
└─────────────────────────────────────────────────────────┘
```

### Thread Safety

All components are thread-safe using:

- `sync.Map` for concurrent access to limiters
- `sync.RWMutex` for read/write locking
- Atomic operations where appropriate

### Memory Management

- Automatic cleanup of inactive limiters (default: every 5 minutes)
- Limiters removed after 10 minutes of inactivity
- Configurable cleanup interval

## Performance

### Benchmark Results

```
BenchmarkIPRateLimiting-12         	 3579559	 324.7 ns/op	  80 B/op	 4 allocs/op
BenchmarkAccountRateLimiting-12    	 3411501	 357.8 ns/op	  80 B/op	 4 allocs/op
BenchmarkAdaptiveTracking-12       	12994071	  95.05 ns/op	 128 B/op	 2 allocs/op
```

**Key Metrics:**

- ~325ns per IP rate limit check
- ~358ns per account rate limit check
- ~95ns per adaptive tracking update
- Minimal memory allocations (4 per check)

### Scalability

- Handles 1M+ requests per minute
- Supports 100K+ concurrent users
- Minimal memory footprint (~100 bytes per active limiter)
- O(1) lookup time for all operations

## Testing

### Run All Tests

```bash
go test ./api/... -v -run "Test.*RateLimit"
```

### Run Specific Tests

```bash
# Test IP-based limiting
go test ./api/... -v -run TestIPBasedRateLimiting

# Test endpoint-specific limiting
go test ./api/... -v -run TestEndpointSpecificRateLimiting

# Test adaptive limiting
go test ./api/... -v -run TestAdaptiveRateLimiting

# Test auto-blocking
go test ./api/... -v -run TestAutoIPBlocking
```

### Run Benchmarks

```bash
go test -bench=. -benchmem ./api/... -run "^$"
```

## Security Considerations

### Best Practices

1. **Always Enable TLS**: Rate limiting doesn't protect against packet inspection
2. **Use Strong Tier Limits**: Don't make free tier too generous
3. **Monitor Audit Logs**: Regularly review for suspicious patterns
4. **Whitelist Carefully**: Only whitelist truly trusted IPs/networks
5. **Set Appropriate Burst Sizes**: Balance UX with security

### Attack Mitigation

**Brute Force Protection:**

- Login endpoint: 5 req/sec with 10 burst
- Auto-block after 100 violations
- Suspicious activity logging at 3+ suspicion level

**DDoS Protection:**

- IP-based global limits (100 req/sec default)
- Auto-blocking for repeat offenders
- Distributed across multiple layers

**Account Enumeration:**

- Registration endpoint: 2 req/sec with 5 burst
- Same error messages for existing/non-existing accounts
- Rate limit applies before account check

## Troubleshooting

### Common Issues

**Problem**: Rate limits too strict for legitimate users

**Solution**:

- Increase tier limits in configuration
- Enable adaptive rate limiting to reward good behavior
- Add trusted IPs to whitelist

**Problem**: High false positive rate on auto-blocking

**Solution**:

- Increase `auto_block_threshold` (default: 100)
- Increase `block_duration` for gradual escalation
- Review audit logs to identify patterns

**Problem**: Performance degradation

**Solution**:

- Decrease `cleanup_interval` for more frequent cleanup
- Review and optimize endpoint-specific limits
- Consider caching frequently accessed data

### Debug Logging

Enable debug logging:

```go
config.RateLimitConfig.DebugMode = true
```

View audit logs:

```bash
tail -f ./logs/audit/audit_*.log | jq .
```

## Migration Guide

### From Basic to Advanced Rate Limiting

1. **Update Configuration**:

   ```go
   config := api.DefaultConfig()
   config.RateLimitConfig = api.DefaultRateLimitConfig()
   ```

2. **Test in Staging**: Verify limits don't impact legitimate traffic

3. **Monitor Metrics**: Watch rate limit stats endpoint

4. **Gradual Rollout**: Start with lenient limits, tighten gradually

### Backward Compatibility

The legacy `RateLimitMiddleware` is still available:

```go
// Old way (deprecated)
router.Use(RateLimitMiddleware(100))

// New way (recommended)
router.Use(AdvancedRateLimitMiddleware(rateLimiter))
```

## API Reference

### Configuration Types

See `api/rate_limiter_config.go` for complete type definitions:

- `RateLimitConfig`: Main configuration structure
- `EndpointLimit`: Per-endpoint limit configuration
- `TierLimit`: Account tier limit configuration
- `AdaptiveConfig`: Adaptive rate limiting settings
- `IPConfig`: IP-based limiting configuration

### Rate Limiter Methods

```go
type AdvancedRateLimiter interface {
    CheckLimit(c *gin.Context) (allowed bool, headers *RateLimitHeaders, err error)
    RecordSuccess(userID string)
    RecordFailure(userID string)
    DecrementConcurrent(userID string)
    GetStats() map[string]interface{}
    Close()
}
```

## Support

For questions, issues, or feature requests:

- GitHub Issues: https://github.com/paw-chain/paw/issues
- Documentation: https://docs.paw.network
- Discord: https://discord.gg/paw

## License

This rate limiting implementation is part of the PAW blockchain project and is licensed under the same terms.
