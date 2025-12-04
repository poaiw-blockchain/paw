# PAW Explorer Historical Indexing - Quick Start Guide

## 5-Minute Setup

### Prerequisites
- PostgreSQL database running
- PAW blockchain node with RPC endpoint
- Go 1.21+

### Step 1: Apply Database Migration

```bash
cd /home/decri/blockchain-projects/paw

# Apply the indexing progress migration
psql -h localhost -U explorer -d paw_explorer \
  -f explorer/database/migrations/005_indexing_progress.sql
```

Expected output:
```
NOTICE: Migration 005: Indexing progress tracking tables created successfully
```

### Step 2: Configure the Indexer

```bash
cd explorer/indexer

# Copy example configuration
cp config.example.yaml config.yaml

# Edit the configuration
nano config.yaml
```

**Minimal configuration changes:**
```yaml
database:
  host: localhost        # Your database host
  user: explorer         # Your database user
  password: your_pass    # Your database password
  database: paw_explorer # Your database name

chain:
  rpc_url: http://localhost:26657  # Your blockchain RPC

indexer:
  enable_historical_indexing: true  # Enable historical indexing
  start_height: 1                   # Start from genesis (or any height)
```

### Step 3: Build and Run

```bash
# Build the indexer
go build -o paw-indexer ./cmd/main.go

# Run the indexer
./paw-indexer --config config.yaml
```

### Step 4: Monitor Progress

**Option A: API Endpoint**
```bash
# Check indexing status
curl http://localhost:8080/api/v1/indexing/status | jq

# Sample output:
{
  "status": "indexing",
  "is_active": true,
  "last_indexed_height": 1500,
  "current_chain_height": 100000,
  "progress_percent": 1.5,
  "avg_blocks_per_second": 25.3,
  "estimated_completion": "2024-01-15T18:30:00Z"
}
```

**Option B: Prometheus Metrics**
```bash
curl http://localhost:9090/metrics | grep explorer_historical
```

**Option C: Database Query**
```sql
SELECT * FROM get_indexing_statistics();
```

---

## Common Use Cases

### Use Case 1: Index Entire Chain from Genesis

```yaml
indexer:
  enable_historical_indexing: true
  start_height: 1
  historical_batch_size: 100
  parallel_fetches: 10
```

Expected time for 100,000 blocks: **4-12 hours**

### Use Case 2: Index from Specific Height

```yaml
indexer:
  enable_historical_indexing: true
  start_height: 50000  # Start from block 50000
  historical_batch_size: 100
```

### Use Case 3: Resume After Interruption

The indexer automatically resumes from the last indexed height. Just restart:

```bash
./paw-indexer --config config.yaml
```

Output:
```
INFO Resuming historical indexing from last checkpoint resuming_from=50000
```

### Use Case 4: Fast Indexing (Dedicated Hardware)

```yaml
indexer:
  enable_historical_indexing: true
  historical_batch_size: 200
  parallel_fetches: 20
  rpc_requests_per_sec: 20

database:
  max_open_conns: 100
```

Expected speed: **15,000-25,000 blocks/hour**

### Use Case 5: Conservative Indexing (Shared RPC)

```yaml
indexer:
  enable_historical_indexing: true
  historical_batch_size: 50
  parallel_fetches: 5
  rpc_requests_per_sec: 5

database:
  max_open_conns: 25
```

Expected speed: **3,000-8,000 blocks/hour**

---

## Monitoring Commands

### Check Overall Progress

```bash
# Quick status
curl http://localhost:8080/api/v1/indexing/status | jq '.progress_percent'

# Detailed statistics
curl http://localhost:8080/api/v1/indexing/statistics | jq
```

### Check Failed Blocks

```bash
curl http://localhost:8080/api/v1/indexing/failed-blocks | jq
```

### Watch Real-Time Progress

```bash
watch -n 5 'curl -s http://localhost:8080/api/v1/indexing/status | jq'
```

### Database Queries

```sql
-- Current progress
SELECT * FROM indexing_progress;

-- Failed blocks
SELECT height, error_message, retry_count
FROM failed_blocks
WHERE resolved = FALSE;

-- Recent checkpoints
SELECT * FROM indexing_checkpoints
ORDER BY created_at DESC
LIMIT 10;

-- Performance metrics
SELECT * FROM indexing_metrics
WHERE metric_name = 'batch_performance'
ORDER BY timestamp DESC
LIMIT 10;
```

---

## Troubleshooting

### Problem: Indexing is slow (< 5 blocks/sec)

**Check 1: RPC connectivity**
```bash
curl http://localhost:26657/status
```

**Check 2: Database performance**
```sql
SELECT * FROM pg_stat_activity WHERE state = 'active';
```

**Solution:** Increase resources
```yaml
indexer:
  parallel_fetches: 20      # Increase from 10
  historical_batch_size: 200 # Increase from 100
```

### Problem: RPC timeouts

**Error:** `Failed to fetch block batch error="context deadline exceeded"`

**Solution:**
```yaml
chain:
  timeout: 60s  # Increase timeout

indexer:
  parallel_fetches: 5  # Reduce concurrent requests
```

### Problem: Database connection errors

**Error:** `Failed to begin transaction error="too many connections"`

**Solution:**
```yaml
database:
  max_open_conns: 100  # Increase connection pool
```

### Problem: Out of memory

**Error:** `runtime: out of memory`

**Solution:**
```yaml
indexer:
  historical_batch_size: 50  # Reduce batch size
  parallel_fetches: 5        # Reduce concurrency
```

---

## Performance Tuning

### Optimize for Speed (Good Hardware)

```yaml
database:
  max_open_conns: 100
  max_idle_conns: 20

chain:
  timeout: 30s
  retry_attempts: 3

indexer:
  historical_batch_size: 200
  parallel_fetches: 20
  rpc_requests_per_sec: 20
  max_retries: 3
```

**Expected:** 15,000-25,000 blocks/hour

### Optimize for Reliability (Shared Resources)

```yaml
database:
  max_open_conns: 25
  max_idle_conns: 5

chain:
  timeout: 60s
  retry_attempts: 5

indexer:
  historical_batch_size: 50
  parallel_fetches: 5
  rpc_requests_per_sec: 5
  max_retries: 5
```

**Expected:** 3,000-8,000 blocks/hour

---

## Docker Quick Start

### Docker Compose

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_DB: paw_explorer
      POSTGRES_USER: explorer
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"

  indexer:
    build: ./explorer/indexer
    depends_on:
      - postgres
    environment:
      DB_HOST: postgres
      DB_USER: explorer
      DB_PASSWORD: password
      RPC_URL: http://paw-node:26657
    ports:
      - "8080:8080"
      - "9090:9090"
```

### Run with Docker Compose

```bash
# Start services
docker-compose up -d

# Apply migrations
docker-compose exec postgres psql -U explorer -d paw_explorer \
  -f /migrations/005_indexing_progress.sql

# Check logs
docker-compose logs -f indexer

# Check status
curl http://localhost:8080/api/v1/indexing/status
```

---

## API Reference

### GET /api/v1/indexing/status

Returns current indexing status with progress percentage.

**Response:**
```json
{
  "status": "indexing",
  "is_active": true,
  "last_indexed_height": 50000,
  "current_chain_height": 100000,
  "progress_percent": 50.0,
  "total_blocks_indexed": 50000,
  "avg_blocks_per_second": 27.5,
  "estimated_completion": "2024-01-15T18:30:00Z"
}
```

### GET /api/v1/indexing/progress

Returns detailed indexing progress.

### GET /api/v1/indexing/failed-blocks

Returns list of blocks that failed to index.

### GET /api/v1/indexing/statistics

Returns comprehensive indexing statistics.

---

## Prometheus Metrics

```prometheus
# Total historical blocks indexed
explorer_historical_blocks_indexed_total

# Indexing progress (0-100%)
explorer_historical_indexing_progress_percent

# Current blocks per second
explorer_blocks_per_second

# Failed blocks count
explorer_failed_blocks_total

# Current indexer height
explorer_indexer_height

# Current chain height
explorer_chain_height
```

---

## Next Steps

After historical indexing completes:

1. **Verify data integrity**
   ```sql
   SELECT COUNT(*) FROM blocks;
   SELECT MAX(height) FROM blocks;
   ```

2. **Enable real-time indexing** (happens automatically)

3. **Set up monitoring dashboards** (Grafana + Prometheus)

4. **Configure backups** for the database

5. **Optimize database** with VACUUM and ANALYZE
   ```sql
   VACUUM ANALYZE blocks;
   VACUUM ANALYZE transactions;
   ```

---

## Support

For issues or questions:
- Check the full implementation guide: `HISTORICAL_INDEXING_IMPLEMENTATION.md`
- Review configuration examples: `config.example.yaml`
- Check database schema: `explorer/database/migrations/005_indexing_progress.sql`

