# PAW API Server - Quick Start Guide

## Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose (optional)
- PAW blockchain node running (optional for basic testing)

## Quick Start (5 minutes)

### 1. Clone and Navigate

```bash
cd /path/to/paw/api
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Run the Server

```bash
go run cmd/main.go
```

The server will start on `http://localhost:5000`

### 4. Test the API

#### Health Check

```bash
curl http://localhost:5000/health
```

#### Register a User

```bash
curl -X POST http://localhost:5000/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "trader1",
    "password": "password123"
  }'
```

#### Login

```bash
curl -X POST http://localhost:5000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "trader1",
    "password": "password123"
  }'
```

Save the token from the response!

#### Get Order Book

```bash
curl http://localhost:5000/api/orders/book
```

#### Create an Order (requires token)

```bash
TOKEN="your-jwt-token-here"

curl -X POST http://localhost:5000/api/orders/create \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "order_type": "buy",
    "price": 10.50,
    "amount": 100
  }'
```

#### Get Wallet Balance (requires token)

```bash
curl http://localhost:5000/api/wallet/balance \
  -H "Authorization: Bearer $TOKEN"
```

## Docker Quick Start

### 1. Build and Run

```bash
docker-compose up -d
```

### 2. Check Logs

```bash
docker-compose logs -f api
```

### 3. Stop

```bash
docker-compose down
```

## Using the Makefile

```bash
# Build the binary
make build

# Run in development mode
make run

# Run tests
make test

# Build Docker image
make docker-build

# Start all services
make docker-up

# View logs
make docker-logs

# Stop all services
make docker-down

# Check API health
make health-check
```

## Environment Variables

Create a `.env` file (copy from `.env.example`):

```bash
cp .env.example .env
```

Edit `.env`:

```env
API_PORT=5000
JWT_SECRET=your-secure-secret-here
CHAIN_ID=paw-1
NODE_URI=tcp://localhost:26657
```

## Testing with Frontend

1. Start the API server (port 5000)
2. Open `external/crypto/exchange-frontend/index.html` in a browser
3. The frontend will automatically connect to `http://localhost:5000/api`

## WebSocket Testing

Using `wscat`:

```bash
npm install -g wscat

# Connect
wscat -c ws://localhost:5000/ws

# Subscribe to price updates
> {"type":"subscribe","channel":"price"}

# Subscribe to order book
> {"type":"subscribe","channel":"orderbook"}

# Subscribe to trades
> {"type":"subscribe","channel":"trades"}
```

## Common Commands

### Get Market Price

```bash
curl http://localhost:5000/api/market/price
```

### Get Market Stats

```bash
curl http://localhost:5000/api/market/stats
```

### Get All Pools

```bash
curl http://localhost:5000/api/pools
```

### Get Light Client Checkpoint

```bash
curl http://localhost:5000/api/light-client/checkpoint
```

### Prepare Atomic Swap

```bash
curl -X POST http://localhost:5000/api/atomic-swap/prepare \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "counterparty_address": "paw1abc...",
    "send_amount": "1000000",
    "send_denom": "paw",
    "receive_amount": "500000",
    "receive_denom": "usdc",
    "timelock_duration": 3600
  }'
```

## Troubleshooting

### Port Already in Use

```bash
# Change the port in .env or use environment variable
API_PORT=5001 go run cmd/main.go
```

### CORS Issues

Add your origin to `.env`:

```env
CORS_ORIGINS=http://localhost:3000,http://localhost:8080
```

### Connection to Blockchain Node Fails

The API will work in mock mode for testing even without a blockchain node. To connect to a real node:

```env
NODE_URI=tcp://your-node-ip:26657
```

## Next Steps

1. Read the full [README.md](README.md) for complete API documentation
2. Check [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md) for architecture details
3. Review the code in `handlers_*.go` for implementation details
4. Run tests: `go test -v ./...`
5. Integrate with your blockchain node
6. Deploy to production using Docker

## Support

- API Documentation: [README.md](README.md)
- Implementation Details: [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md)
- Frontend Integration: `external/crypto/exchange-frontend/`
- Blockchain Docs: `docs/`

## Production Checklist

Before deploying to production:

- [ ] Change JWT_SECRET to a strong random value
- [ ] Configure proper CORS origins
- [ ] Set up HTTPS/TLS (use nginx or Caddy)
- [ ] Configure rate limiting based on your needs
- [ ] Set up database for persistent storage
- [ ] Configure logging and monitoring
- [ ] Set up backups
- [ ] Review security settings
- [ ] Load test the API
- [ ] Set up CI/CD pipeline

Happy coding!
