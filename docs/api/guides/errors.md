# Error Codes Reference

## HTTP Status Codes

### Success Codes (2xx)
- **200 OK**: Request successful
- **201 Created**: Resource created successfully
- **202 Accepted**: Transaction accepted for processing

### Client Error Codes (4xx)
- **400 Bad Request**: Invalid request parameters
- **401 Unauthorized**: Authentication required
- **403 Forbidden**: Insufficient permissions
- **404 Not Found**: Resource not found
- **429 Too Many Requests**: Rate limit exceeded

### Server Error Codes (5xx)
- **500 Internal Server Error**: Server error occurred
- **502 Bad Gateway**: Gateway error
- **503 Service Unavailable**: Service temporarily unavailable
- **504 Gateway Timeout**: Request timeout

## Error Response Format

```json
{
  "code": 400,
  "message": "Invalid request parameters",
  "details": "Field 'amount_in' must be a positive integer"
}
```

## Common Errors

### DEX Module Errors

```json
// Insufficient liquidity
{
  "code": 400,
  "message": "Insufficient liquidity in pool",
  "details": "Pool 1 does not have enough reserves for this swap"
}

// Slippage exceeded
{
  "code": 400,
  "message": "Slippage tolerance exceeded",
  "details": "Expected output: 450000, actual: 440000"
}

// Pool not found
{
  "code": 404,
  "message": "Pool not found",
  "details": "Pool with ID 999 does not exist"
}
```

### Oracle Module Errors

```json
// Asset not found
{
  "code": 404,
  "message": "Price feed not found",
  "details": "No price feed exists for asset XYZ/USD"
}

// Stale price data
{
  "code": 400,
  "message": "Price data is stale",
  "details": "Last update was more than 5 minutes ago"
}
```

### Bank Module Errors

```json
// Insufficient balance
{
  "code": 400,
  "message": "Insufficient balance",
  "details": "Account has 500000uapaw, tried to send 1000000uapaw"
}
```

## Error Handling Examples

### JavaScript

```javascript
try {
  const response = await fetch('/paw/dex/v1/pools/1');
  if (!response.ok) {
    const error = await response.json();
    throw new Error(`${error.code}: ${error.message}`);
  }
  const data = await response.json();
} catch (error) {
  console.error('API Error:', error.message);
}
```

### Python

```python
try:
    response = requests.get(f"{API_URL}/paw/dex/v1/pools/1")
    response.raise_for_status()
    data = response.json()
except requests.exceptions.HTTPError as e:
    error = e.response.json()
    print(f"Error {error['code']}: {error['message']}")
```

### Go

```go
resp, err := http.Get(baseURL + "/paw/dex/v1/pools/1")
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
    var apiError struct {
        Code    int    `json:"code"`
        Message string `json:"message"`
        Details string `json:"details"`
    }
    json.NewDecoder(resp.Body).Decode(&apiError)
    log.Fatalf("API Error %d: %s", apiError.Code, apiError.Message)
}
```

## Retry Strategies

### Idempotent Requests (Safe to Retry)
- GET requests
- Price feed queries
- Balance checks

### Non-Idempotent Requests (Avoid Retry)
- Token swaps
- Pool creation
- Staking operations

## See Also

- [Authentication Guide](./authentication.md)
- [Rate Limiting](./rate-limiting.md)
- [WebSocket Guide](./websockets.md)
