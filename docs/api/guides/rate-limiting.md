# Rate Limiting Guide

## Rate Limits

### Public Endpoints
- **Limit**: 100 requests per minute
- **Burst**: 120 requests
- **No authentication required**

### Authenticated Endpoints
- **Limit**: 1000 requests per minute
- **Burst**: 1200 requests
- **Requires API key**

## Rate Limit Headers

Response headers include rate limit information:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640000000
```

## Handling Rate Limits

### Exponential Backoff

```javascript
async function fetchWithBackoff(url, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    const response = await fetch(url);

    if (response.status === 429) {
      const retryAfter = response.headers.get('Retry-After') || (2 ** i);
      await new Promise(resolve => setTimeout(resolve, retryAfter * 1000));
      continue;
    }

    return response;
  }
  throw new Error('Max retries exceeded');
}
```

### Request Queuing

```javascript
class RateLimiter {
  constructor(requestsPerMinute) {
    this.limit = requestsPerMinute;
    this.queue = [];
    this.processing = false;
  }

  async request(fn) {
    return new Promise((resolve, reject) => {
      this.queue.push({ fn, resolve, reject });
      this.processQueue();
    });
  }

  async processQueue() {
    if (this.processing || this.queue.length === 0) return;

    this.processing = true;
    const { fn, resolve, reject } = this.queue.shift();

    try {
      const result = await fn();
      resolve(result);
    } catch (error) {
      reject(error);
    }

    setTimeout(() => {
      this.processing = false;
      this.processQueue();
    }, 60000 / this.limit);
  }
}

const limiter = new RateLimiter(100);
const result = await limiter.request(() => fetch('/api/endpoint'));
```

## Best Practices

1. **Cache responses** when possible
2. **Use batch endpoints** for multiple queries
3. **Implement exponential backoff**
4. **Monitor rate limit headers**
5. **Request API key** for higher limits

## See Also

- [Error Codes](./errors.md)
- [Authentication](./authentication.md)
