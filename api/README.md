# PAW API Server

REST API server that bridges the PAW blockchain with frontend applications, providing endpoints for trading, wallet operations, light client sync, and atomic swaps.

## Features

- **Authentication**: JWT-based user authentication
- **Trading**: Order creation, order book queries, trade execution
- **Wallet**: Balance queries, token transfers, transaction history
- **Light Client**: Block headers, checkpoints, transaction proofs
- **Atomic Swaps**: Cross-chain atomic swap preparation and execution
- **DEX Pools**: Liquidity pool management and AMM swaps
- **WebSocket**: Real-time updates for prices, order book, and trades

## Architecture

```
api/
├── server.go              # Main HTTP server setup
├── routes.go              # Route registration
├── types.go               # Request/response types
├── middleware.go          # Authentication, CORS, rate limiting
├── websocket.go           # WebSocket hub and client management
├── handlers_auth.go       # User registration and login
├── handlers_trading.go    # Order creation and trading
├── handlers_wallet.go     # Wallet balance and transactions
├── handlers_lightclient.go # Light client proofs
├── handlers_swap.go       # Atomic swap operations
├── handlers_pools.go      # Liquidity pool management
├── handlers_market.go     # Market data and statistics
└── cmd/
    └── main.go            # Server entry point
```

## API Endpoints

### Authentication
- `POST /api/auth/register` - Register new user
- `POST /api/auth/login` - User login

### Trading/Orders
- `POST /api/orders/create` - Create buy/sell order (protected)
- `GET /api/orders/book` - Get order book
- `GET /api/orders/recent` - Get recent trades
- `GET /api/orders/my-orders` - Get user's orders (protected)
- `DELETE /api/orders/:order_id` - Cancel order (protected)

### Wallet
- `GET /api/wallet/balance` - Get wallet balance (protected)
- `GET /api/wallet/address` - Get wallet address (protected)
- `POST /api/wallet/send` - Send tokens (protected)
- `GET /api/wallet/transactions` - Get transaction history (protected)

### Light Client
- `GET /api/light-client/headers` - Get block headers
- `GET /api/light-client/headers/:height` - Get specific header
- `GET /api/light-client/checkpoint` - Get trusted checkpoint
- `GET /api/light-client/tx-proof/:txid` - Get transaction proof
- `POST /api/light-client/verify-proof` - Verify transaction proof

### Atomic Swaps
- `POST /api/atomic-swap/prepare` - Prepare atomic swap (protected)
- `POST /api/atomic-swap/commit` - Commit to swap (protected)
- `POST /api/atomic-swap/refund` - Refund expired swap (protected)
- `GET /api/atomic-swap/status/:swap_id` - Get swap status
- `GET /api/atomic-swap/my-swaps` - Get user's swaps (protected)

### Liquidity Pools
- `GET /api/pools` - Get all pools
- `GET /api/pools/:pool_id` - Get specific pool
- `GET /api/pools/:pool_id/liquidity` - Get pool liquidity
- `POST /api/pools/add-liquidity` - Add liquidity (protected)
- `POST /api/pools/remove-liquidity` - Remove liquidity (protected)

### Market Data
- `GET /api/market/price` - Get current price
- `GET /api/market/stats` - Get market statistics
- `GET /api/market/24h` - Get 24-hour statistics

### WebSocket
- `GET /ws` - WebSocket connection for real-time updates

## WebSocket Channels

Subscribe to channels by sending:
```json
{
  "type": "subscribe",
  "channel": "price"
}
```

Available channels:
- `price` - Real-time price updates
- `orderbook` - Order book updates
- `trades` - New trade notifications

## Configuration

Environment variables:
- `API_HOST` - Server host (default: "0.0.0.0")
- `API_PORT` - Server port (default: "5000")
- `CHAIN_ID` - Blockchain chain ID (default: "paw-1")
- `NODE_URI` - Blockchain node URI (default: "tcp://localhost:26657")
- `JWT_SECRET` - JWT signing secret (required for production)

## Running the Server

```bash
# Set environment variables
export API_PORT=5000
export JWT_SECRET=your-secret-key
export NODE_URI=tcp://localhost:26657

# Run the server
go run api/cmd/main.go
```

## Example Usage

### Register User
```bash
curl -X POST http://localhost:5000/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "trader1",
    "password": "secure123"
  }'
```

### Login
```bash
curl -X POST http://localhost:5000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "trader1",
    "password": "secure123"
  }'
```

### Create Buy Order
```bash
curl -X POST http://localhost:5000/api/orders/create \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "order_type": "buy",
    "price": 10.50,
    "amount": 100
  }'
```

### Get Order Book
```bash
curl http://localhost:5000/api/orders/book
```

### Get Wallet Balance
```bash
curl http://localhost:5000/api/wallet/balance \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## Integration with Frontend

The API is designed to work with the exchange frontend located in `external/crypto/exchange-frontend/`. The frontend expects:

1. API base URL: `http://localhost:5000/api`
2. WebSocket URL: `ws://localhost:5000`
3. JWT token in Authorization header: `Bearer <token>`

## Security Features

- **JWT Authentication**: Secure token-based authentication
- **Rate Limiting**: 100 requests per second per IP
- **CORS**: Configurable cross-origin resource sharing
- **Input Validation**: Request validation using Gin binding
- **Timeouts**: Configurable read/write timeouts

## Blockchain Integration

The API uses Cosmos SDK's `client.Context` for blockchain queries and transaction broadcasting:

- **Queries**: Balance queries, transaction lookups, block headers
- **Transactions**: Token transfers, order creation, liquidity operations
- **Light Client**: Merkle proofs, validator set verification

## Development

### Adding New Endpoints

1. Define request/response types in `types.go`
2. Create handler function in appropriate `handlers_*.go` file
3. Register route in `routes.go`
4. Add middleware if needed (authentication, etc.)

### Testing

```bash
# Run health check
curl http://localhost:5000/health

# Test WebSocket connection
wscat -c ws://localhost:5000/ws
```

## Production Considerations

1. **JWT Secret**: Use a strong, random secret
2. **CORS**: Restrict to specific origins
3. **Rate Limiting**: Adjust based on load
4. **Database**: Replace in-memory storage with persistent database
5. **Blockchain Connection**: Ensure reliable connection to blockchain node
6. **TLS/HTTPS**: Use reverse proxy (nginx, Caddy) for HTTPS
7. **Monitoring**: Add metrics and logging
8. **Error Handling**: Implement comprehensive error handling

## License

MIT License - See LICENSE file for details
