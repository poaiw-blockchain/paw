# PAW Testnet Faucet

A production-ready testnet faucet for the PAW blockchain with rate limiting, captcha protection, and comprehensive monitoring.

## Features

- **Modern Web UI**: Clean, responsive interface built with vanilla JavaScript
- **Rate Limiting**: Per-IP and per-address rate limiting using Redis
- **Captcha Protection**: Turnstile integration to prevent abuse
- **Database Tracking**: PostgreSQL database for request history and statistics
- **Real-time Statistics**: Live faucet balance and distribution metrics
- **Docker Support**: Complete Docker Compose setup for easy deployment
- **Health Monitoring**: Built-in health checks and status endpoints
- **Comprehensive Tests**: Unit, integration, and E2E tests

## Architecture

### Frontend
- Pure HTML/CSS/JavaScript (no framework dependencies)
- Responsive design with dark theme
- Real-time updates
- Client-side validation

### Backend
- Go 1.23.1
- Gin web framework
- PostgreSQL for data persistence
- Redis for rate limiting
- Turnstile for bot protection

## Prerequisites

- Docker and Docker Compose
- Go 1.23.1+ (for local development)
- PostgreSQL 15+ (for local development)
- Redis 7+ (for local development)
- Access to a PAW testnet node

## Quick Start

### Using Docker Compose (Recommended)

1. Clone the repository:
```bash
 clone https://github.com/paw-chain/paw
cd paw/faucet
```

2. Copy the environment file and configure it:
```bash
cp .env.example .env
# Edit .env with your configuration
```

3. Start the services:
```bash
docker-compose up -d
```

4. Access the faucet:
- Web UI: http://localhost:8080
- API: http://localhost:8080/api/v1

### Local Development

1. Install dependencies:
```bash
cd backend
go mod download
```

2. Start PostgreSQL and Redis:
```bash
docker-compose up -d postgres redis
```

3. Set up environment variables:
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. Run the server:
```bash
cd backend
go run main.go
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `ENVIRONMENT` | Environment (development/production) | `development` |
| `NODE_RPC` | PAW node RPC endpoint | `http://localhost:26657` |
| `CHAIN_ID` | Chain ID | `paw-testnet-1` |
| `FAUCET_MNEMONIC` | Faucet wallet mnemonic | Required |
| `FAUCET_ADDRESS` | Faucet wallet address | Required |
| `AMOUNT_PER_REQUEST` | Amount to send per request (in micro-units) | `100000000` |
| `RATE_LIMIT_PER_IP` | Max requests per IP per window | `10` |
| `RATE_LIMIT_PER_ADDRESS` | Max requests per address per window | `1` |
| `RATE_LIMIT_WINDOW_HOURS` | Rate limit window in hours | `24` |
| `TURNSTILE_SECRET` | Turnstile secret key | Required in production |

### Frontend Configuration

Update the Turnstile site key in `frontend/index.html`:
```html
<div class="cf-turnstile" data-sitekey="YOUR_TURNSTILE_SITE_KEY"></div>
```

## API Documentation

### Endpoints

#### Health Check
```
GET /api/v1/health
```
Returns the health status of the faucet service.

**Response:**
```json
{
  "status": "healthy",
  "network": "paw-testnet-1",
  "height": "12345"
}
```

#### Get Faucet Info
```
GET /api/v1/faucet/info
```
Returns faucet configuration and statistics.

**Response:**
```json
{
  "amount_per_request": 100000000,
  "denom": "upaw",
  "balance": 1000000000000,
  "total_distributed": 50000000000,
  "unique_recipients": 125,
  "requests_last_24h": 45,
  "chain_id": "paw-testnet-1"
}
```

#### Request Tokens
```
POST /api/v1/faucet/request
```
Request tokens from the faucet.

**Request Body:**
```json
{
  "address": "paw1...",
  "captcha_token": "turnstile-token"
}
```

**Response:**
```json
{
  "tx_hash": "ABCD1234...",
  "recipient": "paw1...",
  "amount": 100000000,
  "message": "Tokens sent successfully"
}
```

#### Get Recent Transactions
```
GET /api/v1/faucet/recent
```
Returns recent faucet transactions.

**Response:**
```json
{
  "transactions": [
    {
      "recipient": "paw1...",
      "amount": 100000000,
      "tx_hash": "ABCD1234...",
      "timestamp": "2025-11-19T12:00:00Z"
    }
  ]
}
```

#### Get Statistics
```
GET /api/v1/faucet/stats
```
Returns detailed faucet statistics.

## Testing

### Run All Tests
```bash
cd backend
go test ./... -v
```

### Run Specific Test Suite
```bash
# Unit tests
go test ./pkg/... -v

# Integration tests
go test ./tests/integration/... -v

# E2E tests
go test ./tests/e2e/... -v
```

### Run Tests with Coverage
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Database Schema

### faucet_requests Table
```sql
CREATE TABLE faucet_requests (
    id SERIAL PRIMARY KEY,
    recipient VARCHAR(255) NOT NULL,
    amount BIGINT NOT NULL,
    tx_hash VARCHAR(255),
    ip_address VARCHAR(45) NOT NULL,
    status VARCHAR(20) NOT NULL,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE
);
```

## Rate Limiting

The faucet implements two-tier rate limiting:

1. **IP-based**: Limits requests from the same IP address
2. **Address-based**: Limits requests to the same wallet address

Rate limits are enforced using Redis with sliding window algorithm.

## Security Features

- Turnstile verification (production mode)
- Rate limiting (IP and address-based)
- Request validation and sanitization
- Database transaction logging
- Secure configuration management
- CORS protection
- Health check endpoints

## Monitoring

### Health Checks
- API health endpoint: `/api/v1/health`
- Docker health checks for all services
- Database connection monitoring
- Redis connection monitoring
- Blockchain node status monitoring

### Logging
Structured JSON logging with different log levels:
- `debug`: Detailed debugging information
- `info`: General information
- `warn`: Warning messages
- `error`: Error messages

## Troubleshooting

### Common Issues

1. **Cannot connect to database**
   - Ensure PostgreSQL is running
   - Check DATABASE_URL configuration
   - Verify network connectivity

2. **Cannot connect to Redis**
   - Ensure Redis is running
   - Check REDIS_URL configuration
   - Verify network connectivity

3. **Transactions failing**
   - Check faucet account balance
   - Verify NODE_RPC endpoint is accessible
   - Check faucet mnemonic/address configuration

4. **Captcha not working**
   - Verify Turnstile site key in frontend
   - Check TURNSTILE_SECRET in backend
   - Ensure production mode is set

## Production Deployment

### Using Docker Compose

1. Configure production environment:
```bash
cp .env.example .env
# Update all values for production
```

2. Generate SSL certificates (if using nginx):
```bash
mkdir -p ssl
# Add your SSL certificates
```

3. Start with production profile:
```bash
docker-compose --profile production up -d
```

### Security Checklist

- [ ] Update database password
- [ ] Configure Turnstile keys
- [ ] Set up SSL/TLS certificates
- [ ] Configure firewall rules
- [ ] Set up monitoring and alerts
- [ ] Configure backup strategy
- [ ] Review and update rate limits
- [ ] Set LOG_LEVEL to "info" or "warn"

## Maintenance

### Database Backup
```bash
docker-compose exec postgres pg_dump -U faucet faucet > backup.sql
```

### Database Restore
```bash
cat backup.sql | docker-compose exec -T postgres psql -U faucet faucet
```

### View Logs
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f faucet-backend
```

## Contributing

Please read [CONTRIBUTING.md](../CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.

## Support

- Documentation: https://docs.paw-chain.com
- Discord: https://discord.gg/DBHTc2QV
-  Issues: https://github.com/paw-chain/paw/issues
