# PAW DEX Explorer - Quick Start Guide

## 1. Apply Database Migration

```bash
cd /home/decri/blockchain-projects/paw/explorer

# Connect to your database
psql -h localhost -U explorer -d paw_explorer \
  -f database/migrations/006_dex_enhancements.sql

# Verify tables were created
psql -h localhost -U explorer -d paw_explorer -c "\dt dex_*"
```

Expected output:
```
                      List of relations
 Schema |         Name              | Type  |  Owner
--------+---------------------------+-------+----------
 public | dex_analytics_cache       | table | explorer
 public | dex_liquidity             | table | explorer
 public | dex_pool_price_history    | table | explorer
 public | dex_pool_statistics       | table | explorer
 public | dex_pools                 | table | explorer
 public | dex_trades                | table | explorer
 public | dex_user_positions        | table | explorer
```

## 2. Rebuild Indexer

```bash
cd /home/decri/blockchain-projects/paw/explorer/indexer

# Build the indexer
go build -o paw-indexer ./cmd/main.go

# Run (assuming config is already set up)
./paw-indexer --config config/config.yaml
```

## 3. Test API Endpoints

```bash
# Test health
curl http://localhost:8080/health

# Get pool price history (24 hours)
curl "http://localhost:8080/api/v1/dex/pools/1/price-history?period=24h&interval=1h" | jq

# Get pool statistics
curl "http://localhost:8080/api/v1/dex/pools/1/statistics?period=24h" | jq

# Get DEX summary
curl "http://localhost:8080/api/v1/dex/analytics/summary" | jq

# Simulate a swap
curl -X POST http://localhost:8080/api/v1/dex/simulate-swap \
  -H "Content-Type: application/json" \
  -d '{
    "pool_id": "1",
    "token_in": "upaw",
    "amount_in": "1000000",
    "token_out": "uusdc"
  }' | jq
```

## 4. Start Frontend

```bash
cd /home/decri/blockchain-projects/paw/explorer/frontend

# Install dependencies (if not already)
npm install

# Development mode
npm run dev

# Open browser
open http://localhost:3000/dex
open http://localhost:3000/dex/pool/1
```

## 5. Populate Test Data (Optional)

If you need test data for development:

```sql
-- Connect to database
psql -h localhost -U explorer -d paw_explorer

-- Insert sample price history
INSERT INTO dex_pool_price_history (
  pool_id, timestamp, block_height,
  open, high, low, close, volume,
  liquidity_a, liquidity_b,
  price_a_to_b, price_b_to_a
) VALUES
  ('1', NOW() - INTERVAL '1 hour', 1000, '1.0', '1.05', '0.98', '1.02', '50000', '1000000', '1000000', '1.02', '0.98'),
  ('1', NOW() - INTERVAL '2 hours', 999, '0.98', '1.01', '0.96', '1.0', '45000', '1000000', '1000000', '1.0', '1.0'),
  ('1', NOW() - INTERVAL '3 hours', 998, '1.01', '1.03', '0.99', '0.98', '52000', '1000000', '1000000', '0.98', '1.02');

-- Insert sample statistics
INSERT INTO dex_pool_statistics (
  pool_id, period, period_start, period_end,
  volume_usd, trade_count, avg_price,
  fees_usd, apr, unique_traders
) VALUES
  ('1', '24h', NOW() - INTERVAL '24 hours', NOW(),
   '125000', 450, '1.0',
   '375', '15.5', 125);

-- Verify
SELECT * FROM dex_pool_price_history WHERE pool_id = '1';
SELECT * FROM dex_pool_statistics WHERE pool_id = '1';
```

## 6. Verify Everything Works

### Check Database
```bash
psql -h localhost -U explorer -d paw_explorer -c \
  "SELECT COUNT(*) FROM dex_pool_price_history;"

psql -h localhost -U explorer -d paw_explorer -c \
  "SELECT COUNT(*) FROM dex_pool_statistics;"
```

### Check API
```bash
# All endpoints should return 200 OK
curl -w "\n%{http_code}\n" "http://localhost:8080/api/v1/dex/pools/1/price-history?period=24h"
curl -w "\n%{http_code}\n" "http://localhost:8080/api/v1/dex/analytics/summary"
```

### Check Frontend
Open browser to:
- http://localhost:3000/dex - Main DEX page
- http://localhost:3000/dex/pool/1 - Pool detail page

You should see:
- Statistics cards with data
- Recent trades table
- Price history chart placeholder
- Pool information

## 7. Production Deployment

### Docker Compose
```bash
cd /home/decri/blockchain-projects/paw/explorer

# Build and start all services
docker-compose up -d

# Watch logs
docker-compose logs -f indexer frontend
```

### Kubernetes
```bash
# Apply migration to production database first
kubectl exec -it postgres-0 -n paw -- \
  psql -U explorer -d paw_explorer_prod \
  -f /migrations/006_dex_enhancements.sql

# Update deployments
kubectl apply -f k8s/indexer-deployment.yaml
kubectl apply -f k8s/frontend-deployment.yaml

# Verify
kubectl get pods -n paw
kubectl logs -f deployment/paw-indexer -n paw
```

## Troubleshooting

### Database Migration Fails
```bash
# Check if tables already exist
psql -h localhost -U explorer -d paw_explorer -c "\dt dex_*"

# If tables exist, drop and re-run migration
psql -h localhost -U explorer -d paw_explorer -c \
  "DROP TABLE IF EXISTS dex_pool_price_history CASCADE;"
```

### API Returns Empty Data
```bash
# Check if indexer is running
curl http://localhost:8080/health

# Check database connection
psql -h localhost -U explorer -d paw_explorer -c "SELECT 1;"

# Check if pools exist
psql -h localhost -U explorer -d paw_explorer -c \
  "SELECT COUNT(*) FROM dex_pools;"
```

### Frontend Shows Loading Forever
```bash
# Check API is accessible from frontend
curl http://localhost:8080/api/v1/dex/pools

# Check CORS headers
curl -v -H "Origin: http://localhost:3000" \
  http://localhost:8080/api/v1/dex/pools

# Check frontend environment variables
cat frontend/.env.local
# Should have: NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
```

### Build Errors
```bash
# Go build errors
cd indexer
go mod tidy
go build -v ./...

# Frontend build errors
cd frontend
rm -rf .next node_modules
npm install
npm run build
```

## Next Steps

1. **Integrate Chart Library**
   - Install Recharts: `npm install recharts`
   - Replace chart placeholder in pool page
   - Add interactive tooltips

2. **Implement Event Indexer**
   - Create `indexer/internal/indexer/dex_processor.go`
   - Process swap/liquidity events from chain
   - Populate price_history and statistics tables

3. **Create User Portfolio Page**
   - New route: `/account/[address]/dex`
   - Show all LP positions
   - Display P&L and impermanent loss

4. **Add Tests**
   - Database query tests
   - API endpoint tests
   - Frontend component tests

## Support

- Documentation: See `DEX_IMPLEMENTATION_REPORT.md`
- Issues: Check `COMPREHENSIVE_DEX_EXPLORER_ANALYSIS.md`
- Questions: Review `README.md` for full API reference

---

**Ready to go!** The infrastructure is in place for production-grade DEX analytics.
