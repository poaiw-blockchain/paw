# PAW Control Center

Production-ready dashboard for monitoring, testing, and managing the PAW blockchain. Provides comprehensive analytics, real-time monitoring, and secure access control for network operators and stakeholders.

![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)
![Production Ready](https://img.shields.io/badge/production-ready-green.svg)

## Features

### Enterprise-Grade Security
- **JWT-Based Authentication**: Secure token-based authentication with refresh tokens
- **Role-Based Access Control (RBAC)**: Fine-grained permissions (admin, operator, viewer)
- **Redis Session Management**: Distributed session storage and caching
- **Rate Limiting**: Protection against brute force and DDoS attacks
- **Security Headers**: Helmet.js for comprehensive HTTP security

### Advanced Analytics Integration
- **Network Health Monitoring**: Real-time consensus, validator, and transaction metrics
- **Transaction Volume Analytics**: Historical and real-time transaction data with charts
- **DEX Analytics**: Liquidity pools, trading volume, APR, and fee revenue
- **Address Growth Tracking**: User adoption and active address metrics
- **Gas Analytics**: Gas usage patterns and price trends
- **Validator Performance**: Uptime, block production, and performance metrics

### Operational Excellence
- **Health Endpoints**: Kubernetes-ready liveness and readiness probes
- **Structured Logging**: Winston-based logging with rotation and levels
- **Docker Deployment**: Multi-stage builds with security hardening
- **Auto-Scaling Ready**: Stateless design with external session storage
- **Monitoring Integration**: Prometheus-compatible metrics endpoints

### Multi-Network Support
- **Local Testnet**: Full control for development and testing (port 11001-11003)
- **Public Testnet**: Connect to hosted testnet infrastructure
- **Mainnet**: Read-only monitoring mode with safety protections

## Architecture

```
control-center/
├── index.html              # Main dashboard UI
├── config.js              # Network and auth configuration
├── docker-compose.yml     # Production Docker orchestration
├── health.html           # Health check page
├── auth/                 # Authentication microservice
│   ├── Dockerfile        # Node.js auth service container
│   ├── package.json      # Dependencies
│   └── src/
│       ├── server.js     # Express server with JWT & RBAC
│       ├── routes/       # Auth and health routes
│       ├── middleware/   # JWT verification and RBAC
│       └── utils/        # Logger and utilities
├── services/             # Frontend services
│   ├── blockchain.js     # Blockchain API client
│   ├── monitoring.js     # Real-time monitoring
│   ├── analytics.js      # Analytics API integration (NEW)
│   └── testing.js        # Testing utilities
└── components/           # UI components
    ├── NetworkSelector.js
    ├── QuickActions.js
    ├── LogViewer.js
    └── MetricsDisplay.js
```

## Quick Start

### Production Deployment (Docker)

1. **Clone and Configure**
   ```bash
   cd dashboards/control-center
   cp .env.example .env
   # Edit .env with your secrets
   ```

2. **Generate Secure Secrets**
   ```bash
   # Generate JWT secret
   openssl rand -base64 32

   # Generate Redis password
   openssl rand -base64 24

   # Generate Postgres password
   openssl rand -base64 24
   ```

3. **Update .env File**
   ```bash
   vim .env
   # Set JWT_SECRET, REDIS_PASSWORD, POSTGRES_PASSWORD
   ```

4. **Start Core Services** (Dashboard + Auth + Redis)
   ```bash
   docker-compose up -d control-center auth-service redis
   ```

5. **Access Dashboard**
   - Dashboard: http://localhost:11200
   - Auth API: http://localhost:11201
   - Redis: localhost:11202

### Full Stack (with Local Node and Explorer)

```bash
# Start everything including blockchain node and explorer
docker-compose --profile with-node --profile with-explorer up -d
```

**Services**:
- Control Center: http://localhost:11200
- Auth Service: http://localhost:11201
- Explorer/Analytics: http://localhost:11080
- PAW Node RPC: http://localhost:11001
- PAW Node REST: http://localhost:11002
- PostgreSQL: localhost:11203

### Development Mode

```bash
# Dashboard only (no Docker)
python -m http.server 11200

# Or with Node.js
npx http-server -p 11200
```

## Authentication

### Default Users

| Username | Password    | Role     | Permissions                                      |
|----------|-------------|----------|--------------------------------------------------|
| admin    | admin123    | admin    | All permissions including user management        |
| operator | operator123 | operator | Read, write, network control (no user management)|
| viewer   | viewer123   | viewer   | Read-only access                                 |

**IMPORTANT**: Change default passwords in production!

### Login Flow

1. **POST /api/auth/login**
   ```bash
   curl -X POST http://localhost:11201/api/auth/login \
     -H "Content-Type: application/json" \
     -d '{"username":"admin","password":"admin123"}'
   ```

2. **Response**
   ```json
   {
     "success": true,
     "accessToken": "eyJhbGciOiJIUzI1NiIs...",
     "refreshToken": "eyJhbGciOiJIUzI1NiIs...",
     "user": {
       "id": "1",
       "username": "admin",
       "role": "admin",
       "email": "admin@paw.network"
     }
   }
   ```

3. **Use Access Token**
   ```bash
   curl http://localhost:11201/api/auth/me \
     -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
   ```

### Token Refresh

```bash
curl -X POST http://localhost:11201/api/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refreshToken":"eyJhbGciOiJIUzI1NiIs..."}'
```

## RBAC Permissions

### Roles and Permissions

```javascript
{
  admin: ['read', 'write', 'delete', 'manage_users', 'network_control'],
  operator: ['read', 'write', 'network_control'],
  viewer: ['read']
}
```

### Frontend Permission Checks

```javascript
import { checkPermission } from './services/auth.js';

// Check if user has permission
if (checkPermission('network_control')) {
  // Show network control buttons
}

// Check if user has role
if (userRole === 'admin') {
  // Show admin panel
}
```

## Analytics API Integration

The dashboard automatically connects to the explorer analytics API when available.

### Available Analytics

1. **Network Health** (`/api/v1/analytics/network-health`)
   - Overall network status and score
   - Block production metrics
   - Validator health
   - Transaction metrics
   - Consensus metrics

2. **Transaction Volume** (`/api/v1/analytics/transaction-volume?period=24h`)
   - Time-series transaction data
   - TPS (current, average, peak)
   - Success rates
   - Gas usage

3. **DEX Analytics** (`/api/v1/analytics/dex-analytics`)
   - Total TVL and volume
   - Top pools by liquidity
   - Trading volume charts
   - Fee revenue

4. **Address Growth** (`/api/v1/analytics/address-growth?period=30d`)
   - Total addresses
   - New addresses (24h)
   - Active addresses (24h)
   - Growth timeline

5. **Gas Analytics** (`/api/v1/analytics/gas-analytics?period=24h`)
   - Average/median gas prices
   - Gas utilization
   - Top gas consumers

6. **Validator Performance** (`/api/v1/analytics/validator-performance?period=24h`)
   - Individual validator metrics
   - Uptime percentages
   - Block production stats

### Using Analytics in Dashboard

```javascript
import analyticsService from './services/analytics.js';

// Get network health
const health = await analyticsService.getNetworkHealth();
console.log(health.status, health.score);

// Get all dashboard data
const data = await analyticsService.getDashboardData();
```

## Configuration

### Network Endpoints

Edit `config.js` to customize network endpoints:

```javascript
export const CONFIG = {
  networks: {
    local: {
      rpcUrl: 'http://localhost:11001',
      restUrl: 'http://localhost:11002',
      analyticsUrl: 'http://localhost:11080/api/v1/analytics',
      chainId: 'paw-local-1'
    },
    testnet: {
      rpcUrl: 'https://testnet-rpc.paw.network',
      analyticsUrl: 'https://testnet-explorer.paw.network/api/v1/analytics',
      // ...
    }
  }
};
```

### Authentication Settings

```javascript
auth: {
  enabled: true,
  jwtExpiry: '24h',
  refreshTokenExpiry: '7d'
}
```

### Update Intervals

```javascript
updateIntervals: {
  blockUpdates: 3000,      // 3 seconds
  metricsUpdates: 5000,    // 5 seconds
  analyticsRefresh: 30000  // 30 seconds
}
```

## Health Checks

### Dashboard Health
```bash
curl http://localhost:11200/health.html
```

### Auth Service Health
```bash
curl http://localhost:11201/health
```

**Response**:
```json
{
  "status": "healthy",
  "service": "paw-auth-service",
  "version": "1.0.0",
  "uptime": 3600,
  "redis": "connected",
  "memory": {
    "rss": "45 MB",
    "heapUsed": "23 MB"
  }
}
```

### Kubernetes Probes

```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 3000
  initialDelaySeconds: 15
  periodSeconds: 20

readinessProbe:
  httpGet:
    path: /health/ready
    port: 3000
  initialDelaySeconds: 5
  periodSeconds: 10
```

## Monitoring and Logging

### Auth Service Logs

```bash
# View logs
docker-compose logs -f auth-service

# Log location
./auth/logs/combined.log
./auth/logs/error.log
./auth/logs/exceptions.log
```

### Nginx Access Logs

```bash
# View dashboard access logs
docker-compose logs -f control-center
```

### Log Levels

Set via environment variable:
```bash
LOG_LEVEL=debug  # debug, info, warn, error
```

## Security Best Practices

### Production Checklist

- [ ] Change all default passwords
- [ ] Generate unique JWT_SECRET (32+ characters)
- [ ] Set strong Redis and PostgreSQL passwords
- [ ] Enable HTTPS (use reverse proxy like Nginx/Traefik)
- [ ] Restrict network access (firewall rules)
- [ ] Enable audit logging
- [ ] Regular security updates
- [ ] Monitor failed login attempts
- [ ] Implement IP whitelisting for admin access

### Environment Variables

**Never commit .env files to git!**

```bash
# .env is in .gitignore
# Always use .env.example as template
```

## Troubleshooting

### Dashboard Not Loading

1. Check if container is running:
   ```bash
   docker-compose ps
   ```

2. View logs:
   ```bash
   docker-compose logs control-center
   ```

3. Verify port is available:
   ```bash
   lsof -i :11200
   ```

### Authentication Failures

1. Check auth service health:
   ```bash
   curl http://localhost:11201/health
   ```

2. Verify JWT secret matches:
   ```bash
   docker-compose exec auth-service env | grep JWT_SECRET
   ```

3. Check Redis connection:
   ```bash
   docker-compose exec redis redis-cli ping
   ```

### Analytics Not Loading

1. Verify explorer is running:
   ```bash
   curl http://localhost:11080/api/v1/health
   ```

2. Check analytics endpoint:
   ```bash
   curl http://localhost:11080/api/v1/analytics/network-health
   ```

3. Verify config.js has correct analyticsUrl

## Upgrading

### Update Dashboard

```bash
cd dashboards/control-center
git pull
docker-compose build
docker-compose up -d
```

### Database Migrations

For explorer analytics database:
```bash
docker-compose exec postgres psql -U paw -d paw_explorer -f /migrations/latest.sql
```

## API Reference

### Auth Endpoints

- `POST /api/auth/login` - User login
- `POST /api/auth/register` - Register user (admin only)
- `POST /api/auth/refresh` - Refresh access token
- `POST /api/auth/logout` - Logout user
- `GET /api/auth/verify` - Verify token
- `GET /api/auth/me` - Get current user

### Analytics Endpoints

See explorer analytics documentation for complete API reference.

## Support

- **Documentation**: /docs/control-center/
- **Issues**: GitHub Issues
- **Discord**: PAW Community Discord

## License

MIT License - see LICENSE file for details

## Changelog

### v1.0.0 (2024-12-14)
- Initial production release
- JWT authentication and RBAC
- Analytics API integration
- Docker deployment with health checks
- Redis session management
- Structured logging with Winston
- Multi-network support
- Comprehensive security hardening

---

**Made with ❤️ for the PAW Community**

For production deployment assistance, contact the PAW DevOps team.
