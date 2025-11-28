# Go API Examples

Complete Go examples for interacting with the PAW Blockchain API.

## Installation

```bash
go get github.com/cosmos/cosmos-sdk
```

## Basic Client

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type PAWClient struct {
    BaseURL    string
    HTTPClient *http.Client
}

func NewPAWClient(baseURL string) *PAWClient {
    return &PAWClient{
        BaseURL: baseURL,
        HTTPClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (c *PAWClient) Get(endpoint string, result interface{}) error {
    resp, err := c.HTTPClient.Get(c.BaseURL + endpoint)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
    }

    return json.NewDecoder(resp.Body).Decode(result)
}

func (c *PAWClient) Post(endpoint string, data interface{}, result interface{}) error {
    jsonData, err := json.Marshal(data)
    if err != nil {
        return err
    }

    resp, err := c.HTTPClient.Post(
        c.BaseURL+endpoint,
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
    }

    return json.NewDecoder(resp.Body).Decode(result)
}

func main() {
    client := NewPAWClient("http://localhost:1317")
}
```

## Type Definitions

```go
type Pool struct {
    ID          uint64 `json:"id"`
    TokenA      string `json:"token_a"`
    TokenB      string `json:"token_b"`
    ReserveA    string `json:"reserve_a"`
    ReserveB    string `json:"reserve_b"`
    TotalShares string `json:"total_shares"`
    FeeRate     string `json:"fee_rate"`
    CreatedAt   string `json:"created_at"`
}

type PriceFeed struct {
    Asset            string `json:"asset"`
    Price            string `json:"price"`
    Source           string `json:"source"`
    UpdatedAt        string `json:"updated_at"`
    ValidatorsVoted  int    `json:"validators_voted"`
    TotalValidators  int    `json:"total_validators"`
}

type Coin struct {
    Denom  string `json:"denom"`
    Amount string `json:"amount"`
}

type Validator struct {
    OperatorAddress string `json:"operator_address"`
    Jailed          bool   `json:"jailed"`
    Status          string `json:"status"`
    Tokens          string `json:"tokens"`
    DelegatorShares string `json:"delegator_shares"`
}
```

## DEX Module

```go
func (c *PAWClient) ListPools() ([]Pool, error) {
    var response struct {
        Pools []Pool `json:"pools"`
    }
    err := c.Get("/paw/dex/v1/pools", &response)
    return response.Pools, err
}

func (c *PAWClient) GetPool(poolID uint64) (*Pool, error) {
    var response struct {
        Pool Pool `json:"pool"`
    }
    endpoint := fmt.Sprintf("/paw/dex/v1/pools/%d", poolID)
    err := c.Get(endpoint, &response)
    return &response.Pool, err
}

func (c *PAWClient) EstimateSwap(poolID uint64, tokenIn string, amountIn string) (map[string]interface{}, error) {
    data := map[string]interface{}{
        "pool_id":   poolID,
        "token_in":  tokenIn,
        "amount_in": amountIn,
    }
    var result map[string]interface{}
    err := c.Post("/paw/dex/v1/estimate_swap", data, &result)
    return result, err
}

// Example usage
func ExampleDEX() {
    client := NewPAWClient("http://localhost:1317")

    pools, err := client.ListPools()
    if err != nil {
        log.Fatal(err)
    }

    for _, pool := range pools {
        fmt.Printf("Pool %d: %s/%s\n", pool.ID, pool.TokenA, pool.TokenB)
    }
}
```

## Oracle Module

```go
func (c *PAWClient) ListPriceFeeds() ([]PriceFeed, error) {
    var response struct {
        PriceFeeds []PriceFeed `json:"price_feeds"`
    }
    err := c.Get("/paw/oracle/v1/prices", &response)
    return response.PriceFeeds, err
}

func (c *PAWClient) GetPrice(asset string) (*PriceFeed, error) {
    var response struct {
        PriceFeed PriceFeed `json:"price_feed"`
    }
    endpoint := fmt.Sprintf("/paw/oracle/v1/prices/%s", url.QueryEscape(asset))
    err := c.Get(endpoint, &response)
    return &response.PriceFeed, err
}

// Monitor prices
func MonitorPrices(client *PAWClient, assets []string, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for range ticker.C {
        for _, asset := range assets {
            feed, err := client.GetPrice(asset)
            if err != nil {
                log.Printf("Error fetching %s: %v", asset, err)
                continue
            }
            fmt.Printf("%s: $%s\n", asset, feed.Price)
        }
    }
}
```

## Bank Module

```go
func (c *PAWClient) GetBalance(address string) ([]Coin, error) {
    var response struct {
        Balances []Coin `json:"balances"`
    }
    endpoint := fmt.Sprintf("/cosmos/bank/v1beta1/balances/%s", address)
    err := c.Get(endpoint, &response)
    return response.Balances, err
}

func (c *PAWClient) SendTokens(from, to string, amount []Coin) error {
    data := map[string]interface{}{
        "from_address": from,
        "to_address":   to,
        "amount":       amount,
    }
    var result map[string]interface{}
    return c.Post("/cosmos/bank/v1beta1/send", data, &result)
}
```

## Staking Module

```go
func (c *PAWClient) ListValidators(status string) ([]Validator, error) {
    var response struct {
        Validators []Validator `json:"validators"`
    }
    endpoint := "/cosmos/staking/v1beta1/validators"
    if status != "" {
        endpoint += "?status=" + status
    }
    err := c.Get(endpoint, &response)
    return response.Validators, err
}

func (c *PAWClient) GetValidator(address string) (*Validator, error) {
    var response struct {
        Validator Validator `json:"validator"`
    }
    endpoint := fmt.Sprintf("/cosmos/staking/v1beta1/validators/%s", address)
    err := c.Get(endpoint, &response)
    return &response.Validator, err
}
```

## Advanced Examples

### Concurrent Requests

```go
func (c *PAWClient) GetMultiplePools(poolIDs []uint64) ([]*Pool, error) {
    type result struct {
        pool *Pool
        err  error
    }

    results := make(chan result, len(poolIDs))

    for _, id := range poolIDs {
        go func(poolID uint64) {
            pool, err := c.GetPool(poolID)
            results <- result{pool, err}
        }(id)
    }

    pools := make([]*Pool, 0, len(poolIDs))
    for i := 0; i < len(poolIDs); i++ {
        r := <-results
        if r.err != nil {
            return nil, r.err
        }
        pools = append(pools, r.pool)
    }

    return pools, nil
}
```

### Retry Logic

```go
func RetryRequest(fn func() error, maxAttempts int, delay time.Duration) error {
    for attempt := 0; attempt < maxAttempts; attempt++ {
        err := fn()
        if err == nil {
            return nil
        }
        if attempt < maxAttempts-1 {
            log.Printf("Retry %d/%d after %v...", attempt+1, maxAttempts, delay)
            time.Sleep(delay)
        }
    }
    return fmt.Errorf("max retries exceeded")
}
```

## See Also

- [JavaScript Examples](./javascript.md)
- [Python Examples](./python.md)
- [cURL Examples](./curl.md)
