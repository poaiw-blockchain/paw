# Explorer Indexer Test Suite

## Overview

Comprehensive test coverage for the PAW blockchain explorer indexer, covering all components from database operations to WebSocket broadcasting.

## Test Files Created

### 1. Database Queries Tests (`internal/database/queries_test.go`)

**Coverage Areas:**
- Block CRUD operations (Insert, Update, Query, Pagination)
- Transaction CRUD operations
- DEX price history queries
- User position management
- Pool statistics aggregation
- Analytics caching
- Complex joins and aggregations
- Transaction handling (commit/rollback)
- Connection pool behavior
- Query timeouts

**Key Tests:**
- `TestInsertBlock` - Block insertion
- `TestInsertBlockConflict` - ON CONFLICT handling
- `TestGetBlocks_Pagination` - Pagination logic
- `TestInsertTransaction` - Transaction insertion
- `TestGetPoolPriceHistory` - OHLCV data retrieval
- `TestUpsertUserPosition` - User position upsert logic
- `TestGetUserDEXAnalytics` - Complex analytics aggregation
- `TestTransactionCommit/Rollback` - Transaction handling
- `TestConnectionPoolBehavior` - Concurrent connections
- `TestQueryTimeout` - Context timeout handling

**Benchmarks:**
- `BenchmarkInsertBlock` - Block insertion performance
- `BenchmarkGetBlockByHeight` - Query performance
- `BenchmarkGetPoolPriceHistory` - Complex query performance

### 2. WebSocket Hub Tests (`internal/websocket/hub/hub_test.go`)

**Coverage Areas:**
- Hub lifecycle (creation, start, stop)
- Client registration/unregistration
- Message broadcasting to multiple clients
- Subscription filtering
- Concurrent client handling
- Graceful shutdown
- Ping/Pong mechanism
- Backpressure handling

**Key Tests:**
- `TestNewHub` - Hub initialization
- `TestClientRegistration` - Client connect/register
- `TestClientUnregistration` - Client disconnect cleanup
- `TestMultipleClientRegistration` - Multiple concurrent clients
- `TestBroadcastBlock` - Block message broadcasting
- `TestBroadcastTransaction` - Transaction broadcasting
- `TestBroadcastDEXSwap` - DEX swap broadcasting
- `TestSubscriptionFiltering` - Message type filtering
- `TestUnsubscribe` - Unsubscribe handling
- `TestConcurrentBroadcast` - Concurrent message broadcast
- `TestConcurrentClientConnections` - 100 concurrent connections
- `TestClientDisconnectionCleanup` - Memory cleanup verification
- `TestPingPong` - Heartbeat mechanism

**Benchmarks:**
- `BenchmarkHubWith100Clients` - 100 client performance
- `BenchmarkHubWith1000Clients` - 1000 client performance
- `BenchmarkHubWith10000Clients` - 10000 client scalability
- `BenchmarkClientRegistration` - Registration overhead

## Test Files Needed (To Be Implemented)

### 3. Indexer Integration Tests (`internal/indexer/indexer_integration_test.go`)

**Planned Coverage:**
- Full block indexing workflow
- Transaction indexing with events
- Historical block indexing
- Real-time indexing
- Reorg handling (if supported)
- Indexer recovery after crash
- Parallel block processing
- Failed block retry logic

**Planned Tests:**
- `TestIndexHistoricalBlocks` - Historical indexing from height N to M
- `TestIndexRealtime` - Real-time block indexing
- `TestIndexerRecovery` - Resume from last indexed height
- `TestParallelIndexing` - Concurrent block fetching
- `TestFailedBlockRetry` - Retry mechanism
- `TestIndexingMetrics` - Prometheus metrics

**Planned Benchmarks:**
- `BenchmarkIndexingThroughput` - Blocks per second

### 4. API Handlers Tests (`internal/api/handlers_test.go`)

**Planned Coverage:**
All 46+ API endpoints:

#### Block Endpoints (4)
- GET `/api/v1/blocks` - List blocks with pagination
- GET `/api/v1/blocks/latest` - Latest N blocks
- GET `/api/v1/blocks/:height` - Get specific block
- GET `/api/v1/blocks/:height/transactions` - Block transactions

#### Transaction Endpoints (4)
- GET `/api/v1/transactions` - List transactions
- GET `/api/v1/transactions/latest` - Latest transactions
- GET `/api/v1/transactions/:hash` - Get transaction
- GET `/api/v1/transactions/:hash/events` - Transaction events

#### Account Endpoints (6)
- GET `/api/v1/accounts/:address` - Get account
- GET `/api/v1/accounts/:address/transactions` - Account transactions
- GET `/api/v1/accounts/:address/balances` - Account balances
- GET `/api/v1/accounts/:address/tokens` - Account tokens
- GET `/api/v1/accounts/:address/dex-positions` - DEX positions
- GET `/api/v1/accounts/:address/dex-history` - DEX activity history

#### Validator Endpoints (5)
- GET `/api/v1/validators` - List validators
- GET `/api/v1/validators/active` - Active validators
- GET `/api/v1/validators/:address` - Get validator
- GET `/api/v1/validators/:address/uptime` - Validator uptime
- GET `/api/v1/validators/:address/rewards` - Validator rewards

#### DEX Endpoints (14)
- GET `/api/v1/dex/pools` - List pools
- GET `/api/v1/dex/pools/:id` - Get pool
- GET `/api/v1/dex/pools/:id/trades` - Pool trades
- GET `/api/v1/dex/pools/:id/liquidity` - Pool liquidity
- GET `/api/v1/dex/pools/:id/chart` - Pool chart
- GET `/api/v1/dex/pools/:id/price-history` - OHLCV data
- GET `/api/v1/dex/pools/:id/liquidity-chart` - TVL chart
- GET `/api/v1/dex/pools/:id/volume-chart` - Volume chart
- GET `/api/v1/dex/pools/:id/fees` - Fee breakdown
- GET `/api/v1/dex/pools/:id/apr-history` - APR history
- GET `/api/v1/dex/pools/:id/depth` - Liquidity depth
- GET `/api/v1/dex/pools/:id/statistics` - Pool statistics
- GET `/api/v1/dex/trades` - All trades
- GET `/api/v1/dex/trades/latest` - Latest trades

#### Oracle Endpoints (6)
- GET `/api/v1/oracle/prices` - All oracle prices
- GET `/api/v1/oracle/prices/:asset` - Asset price
- GET `/api/v1/oracle/prices/:asset/history` - Price history
- GET `/api/v1/oracle/prices/:asset/chart` - Price chart
- GET `/api/v1/oracle/submissions` - Oracle submissions
- GET `/api/v1/oracle/slashes` - Oracle slashes

#### Compute Endpoints (6)
- GET `/api/v1/compute/requests` - List compute requests
- GET `/api/v1/compute/requests/:id` - Get request
- GET `/api/v1/compute/requests/:id/results` - Request results
- GET `/api/v1/compute/requests/:id/verifications` - Verifications
- GET `/api/v1/compute/providers` - List providers
- GET `/api/v1/compute/providers/:address` - Get provider

#### Statistics Endpoints (5)
- GET `/api/v1/stats/network` - Network statistics
- GET `/api/v1/stats/charts/transactions` - TX chart
- GET `/api/v1/stats/charts/addresses` - Address chart
- GET `/api/v1/stats/charts/volume` - Volume chart
- GET `/api/v1/stats/charts/gas` - Gas chart

#### Utility Endpoints (3)
- GET `/api/v1/search` - Universal search
- GET `/api/v1/export/transactions` - Export transactions (CSV/JSON)
- GET `/api/v1/export/trades` - Export trades (CSV/JSON)

**Test Categories:**
- Response status codes (200, 404, 500, etc.)
- Pagination validation
- Filter and sort validation
- Rate limiting
- Authentication (if implemented)
- Error handling
- Response format validation

### 5. Subscriber Tests (`internal/subscriber/subscriber_test.go`)

**Planned Coverage:**
- WebSocket connection to blockchain node
- Subscription to NewBlock events
- Event parsing
- Reconnection logic
- Backpressure handling
- Error handling

**Planned Tests:**
- `TestSubscriberConnect` - Connection establishment
- `TestSubscriberSubscribe` - Subscribe to events
- `TestSubscriberReceiveBlock` - Receive and parse block
- `TestSubscriberReconnect` - Reconnection on disconnect
- `TestSubscriberBackpressure` - Handle fast block production

### 6. End-to-End Tests (`test/e2e_test.go`)

**Planned Coverage:**
- Complete flow: blockchain → indexer → database → API → WebSocket
- Real-time updates via WebSocket
- Historical data queries
- Concurrent reads and writes

**Planned Tests:**
- `TestE2E_BlockIndexingToAPI` - Block indexed and queryable via API
- `TestE2E_TransactionIndexingToAPI` - Transaction flow
- `TestE2E_DEXSwapIndexingToWS` - DEX swap real-time broadcast
- `TestE2E_ConcurrentOperations` - Concurrent indexing and queries

### 7. Test Helpers (`test/helpers.go`)

**Planned Utilities:**
- Mock blockchain data generators
- Test database setup/teardown
- Mock WebSocket clients
- Test RPC server
- Assertion helpers

## Running Tests

### Run All Tests
```bash
cd explorer/indexer
go test ./... -v
```

### Run Specific Package
```bash
go test ./internal/database -v
go test ./internal/websocket/hub -v
```

### Run With Coverage
```bash
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Run Benchmarks
```bash
go test ./internal/database -bench=. -benchmem
go test ./internal/websocket/hub -bench=. -benchmem
```

### Run Integration Tests
```bash
go test ./test -tags=integration -v
```

## Test Requirements

### Prerequisites
- PostgreSQL 13+ running on localhost:5432
- Test database: `paw_explorer_test`
- Redis (optional, for cache tests)
- PAW blockchain node (for E2E tests)

### Database Setup
```sql
CREATE DATABASE paw_explorer_test;
GRANT ALL PRIVILEGES ON DATABASE paw_explorer_test TO postgres;
```

### Environment Variables
```bash
export DB_URL="postgres://postgres:postgres@localhost:5432/paw_explorer_test?sslmode=disable"
export REDIS_URL="redis://localhost:6379/1"
export NODE_RPC_URL="http://localhost:26657"
export NODE_WS_URL="ws://localhost:26657/websocket"
```

## Coverage Goals

- **Overall Target:** >85%
- **Database Package:** >90% (critical path)
- **WebSocket Hub:** >85%
- **API Handlers:** >80%
- **Indexer:** >85%
- **Subscriber:** >75%

## Continuous Integration

### GitHub Actions Workflow
```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:13
        env:
          POSTGRES_PASSWORD: postgres
        ports:
          - 5432:5432
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: go test ./... -v -cover
```

## Current Test Statistics

### Implemented
- **Database Tests:** 25+ tests, 3 benchmarks
- **WebSocket Hub Tests:** 20+ tests, 4 benchmarks
- **Total Coverage:** ~40% (after implementing all planned tests: ~85%)

### Pending
- Indexer integration tests
- API handler tests (46+ endpoints)
- Subscriber tests
- E2E tests
- Test helpers/utilities

## Best Practices

1. **Isolation:** Each test should be independent
2. **Cleanup:** Always teardown test resources
3. **Fixtures:** Use realistic test data
4. **Assertions:** Use require for critical checks, assert for optional
5. **Naming:** Test names should describe what they test
6. **Parallelization:** Use `t.Parallel()` where safe
7. **Timeouts:** Always use timeouts for async operations
8. **Mocks:** Mock external dependencies (blockchain node)

## Troubleshooting

### Tests Fail to Connect to Database
- Ensure PostgreSQL is running
- Verify database credentials
- Check database exists: `paw_explorer_test`

### WebSocket Tests Timeout
- Increase timeout values
- Check for goroutine leaks
- Verify proper cleanup in teardown

### Benchmarks Show Poor Performance
- Run benchmarks multiple times: `-benchtime=10s`
- Check for resource contention
- Profile with `go test -bench=. -cpuprofile=cpu.out`

## Future Enhancements

1. **Fuzzing:** Add fuzz tests for parsers
2. **Property-Based Testing:** Use `gopter` for invariants
3. **Mutation Testing:** Verify test quality
4. **Performance Regression Tests:** Track performance over time
5. **Chaos Engineering:** Test resilience under failures
