# PAW Blockchain Dashboards Guide

**Complete guide to deploying and using the PAW blockchain operational dashboards**

## Overview

The PAW blockchain provides three production-ready web dashboards for managing different aspects of the network:

1. **Staking Dashboard** - For delegators and validators to manage staking operations
2. **Validator Dashboard** - Real-time monitoring and management for validators
3. **Governance Portal** - Community governance and proposal voting interface

All dashboards are containerized, configured via environment variables, and designed for both development and production use.

## Quick Start

### Deploy All Dashboards

```bash
cd /home/hudson/blockchain-projects/paw
./scripts/deploy-dashboards.sh
```

### Access Dashboards

- **Staking Dashboard**: http://localhost:11100
- **Validator Dashboard**: http://localhost:11110
- **Governance Portal**: http://localhost:11120

### Verify Health

```bash
./scripts/verify-dashboards.sh
```

### Stop Dashboards

```bash
./scripts/stop-dashboards.sh
```

## Dashboard Features

### 1. Staking Dashboard (Port 11100)

**Purpose**: Comprehensive staking tools for validators and delegators

#### Core Features

- **Validator Discovery & Management**
  - Complete validator list with sorting and filtering
  - Real-time validator status and metrics
  - Commission rates and voting power tracking
  - Risk score calculation and indicators
  - Uptime monitoring

- **Staking Calculator**
  - APY/APR estimation
  - Simple and compound interest calculations
  - Multiple time period support (daily, weekly, monthly, yearly)
  - Real-time reward projections

- **Delegation Management**
  - Delegate to validators
  - Undelegate with unbonding period tracking
  - Redelegate between validators
  - Real-time balance validation
  - Transaction fee estimation

- **Rewards Management**
  - View pending rewards by validator
  - Claim rewards (single or all)
  - Auto-compound functionality
  - Reward history tracking

- **Portfolio View**
  - Complete portfolio overview
  - Total value calculation
  - Active delegations tracking
  - Unbonding delegations monitoring
  - Historical activity log

#### Usage

1. **Connect Wallet**: Click "Connect Wallet" and approve Keplr connection
2. **View Validators**: Browse and filter validators by various metrics
3. **Calculate Returns**: Use the calculator to estimate staking rewards
4. **Delegate Tokens**: Select a validator and specify amount to delegate
5. **Manage Rewards**: Claim or auto-compound your staking rewards
6. **Monitor Portfolio**: Track all active and unbonding delegations

#### Test Coverage

- Unit Tests: 85%+
- Integration Tests: Comprehensive
- E2E Tests: Complete user workflows
- Performance Tests: Load time and caching optimizations

### 2. Validator Dashboard (Port 11110)

**Purpose**: Real-time monitoring and management for blockchain validators

#### Core Features

- **Overview Panel**
  - Validator status and health
  - Total staked tokens
  - Commission rate
  - Total rewards earned
  - Number of delegators
  - Current uptime percentage
  - Recent activity feed

- **Delegation Management**
  - Complete list of all delegators
  - Delegation amounts and shares
  - Pending rewards per delegator
  - Search and filter functionality
  - Sort by amount, date, or rewards
  - Export capabilities

- **Rewards Tracking**
  - Historical rewards visualization
  - Interactive charts (line, bar, area)
  - Multiple timeframes (7d, 30d, 90d, 1y, all)
  - Total distributed rewards
  - Pending rewards
  - Commission earned
  - Trend analysis

- **Performance Metrics**
  - Voting power percentage
  - Block proposal statistics
  - Miss rate tracking
  - Historical performance charts
  - Comparative analytics

- **Uptime Monitoring**
  - Real-time uptime percentage
  - Block signing visualization
  - 24-hour uptime timeline
  - Uptime alerts and warnings
  - Signing window status
  - Time to slash threshold

- **Signing Statistics**
  - Total blocks signed
  - Total blocks missed
  - Signing rate percentage
  - Signing history visualization
  - Pattern analysis

- **Slash Events**
  - Complete slash event history
  - Event details and reasons
  - Amount slashed
  - Block height information

#### Usage

1. **Add Validator**: Enter validator address (pawvaloper...)
2. **Monitor Status**: Real-time updates via WebSocket connection
3. **Track Performance**: View uptime, signing stats, and performance metrics
4. **Manage Delegations**: Monitor all delegators and their stakes
5. **Analyze Rewards**: Visualize reward distribution and trends
6. **Set Alerts**: Configure email alerts for critical events

#### Real-Time Updates

The validator dashboard uses WebSocket connections for real-time updates:
- New blocks every 6 seconds
- Validator status changes
- Signing events
- Delegation updates

### 3. Governance Portal (Port 11120)

**Purpose**: Community participation in on-chain governance

#### Core Features

- **Proposal Management**
  - View all proposals with advanced filtering
  - Detailed proposal information
  - Timeline visualization
  - Support for multiple proposal types:
    - Text Proposals (signaling)
    - Parameter Change Proposals
    - Software Upgrade Proposals
    - Community Pool Spend Proposals
  - Real-time proposal status tracking

- **Voting Interface**
  - Four voting options:
    - **Yes** - Support the proposal
    - **No** - Oppose the proposal
    - **Abstain** - Contribute to quorum without position
    - **No With Veto** - Strongly oppose and burn deposits
  - Vote tallying with interactive charts
  - Voting power calculation
  - Vote history tracking

- **Proposal Creation**
  - Multi-type proposal support
  - Form validation and preview
  - Deposit management (minimum 10,000 PAW)
  - Guided workflow

- **Analytics Dashboard**
  - Proposal success rate visualization
  - Voting trends over time
  - Participation rate tracking
  - Top voter rankings

- **Governance Parameters**
  - Display of current governance parameters
  - Deposit requirements (10,000 PAW minimum)
  - Voting periods (14 days default)
  - Quorum and threshold values (33.4% quorum, 50% threshold)

#### Usage

1. **View Proposals**: Browse active and historical proposals
2. **Filter & Search**: Find specific proposals by status or type
3. **Vote**: Connect wallet and cast your vote
4. **Create Proposal**: Submit new proposals with initial deposit
5. **Add Deposits**: Help proposals reach deposit threshold
6. **Track Analytics**: Monitor governance participation and trends

#### Proposal Lifecycle

1. **Deposit Period** (14 days)
   - Proposal needs 10,000 PAW to enter voting
   - Community can add deposits
   - Refunded if proposal enters voting

2. **Voting Period** (14 days)
   - All token holders can vote
   - Weighted by staked tokens
   - Can change vote until period ends

3. **Tallying**
   - Quorum: 33.4% participation required
   - Threshold: 50% Yes votes needed
   - Veto: 33.4% NoWithVeto fails proposal

## Configuration

### Environment Variables

All dashboards are configured via environment variables injected at runtime:

```yaml
# Staking & Validator Dashboards
PAW_API_URL: http://localhost:1317        # Cosmos SDK REST API
PAW_RPC_URL: http://localhost:26657       # Tendermint RPC
PAW_WS_URL: ws://localhost:26657/websocket # WebSocket for real-time

# Governance Portal
PAW_API_URL: http://localhost:1317        # Cosmos SDK REST API
PAW_RPC_URL: http://localhost:26657       # Tendermint RPC
PAW_MOCK_MODE: false                      # Disable mock data for production
```

### Customizing Endpoints

Edit `compose/docker-compose.dashboards.yml`:

```yaml
services:
  staking-dashboard:
    environment:
      - PAW_API_URL=https://api.paw.network:1317
      - PAW_RPC_URL=https://rpc.paw.network:26657
      - PAW_WS_URL=wss://rpc.paw.network:26657/websocket
```

### Port Configuration

Default ports (all in 11000-11999 range to avoid conflicts):

- Staking Dashboard: 11100
- Validator Dashboard: 11110
- Governance Portal: 11120

To change ports, edit `compose/docker-compose.dashboards.yml`:

```yaml
services:
  staking-dashboard:
    ports:
      - "8080:3000"  # Map to different external port
```

## Architecture

### Container Architecture

```
┌─────────────────────────────────────────┐
│         Docker Network                  │
│         (paw-dashboards)                │
│                                          │
│  ┌──────────────────────────────────┐  │
│  │   Staking Dashboard              │  │
│  │   nginx:alpine + static files    │  │
│  │   Port: 11100 → 3000             │  │
│  └──────────────────────────────────┘  │
│                                          │
│  ┌──────────────────────────────────┐  │
│  │   Validator Dashboard            │  │
│  │   nginx:alpine + static files    │  │
│  │   Port: 11110 → 3000             │  │
│  └──────────────────────────────────┘  │
│                                          │
│  ┌──────────────────────────────────┐  │
│  │   Governance Portal              │  │
│  │   nginx:alpine + static files    │  │
│  │   Port: 11120 → 3000             │  │
│  └──────────────────────────────────┘  │
│                                          │
└─────────────────────────────────────────┘
            │
            │ API Calls
            ▼
┌─────────────────────────────────────────┐
│      PAW Blockchain Node                │
│      REST API: 1317                     │
│      RPC: 26657                         │
│      WebSocket: 26657/websocket         │
└─────────────────────────────────────────┘
```

### File Structure

```
dashboards/
├── staking/                    # Staking dashboard
│   ├── Dockerfile             # Container definition
│   ├── index.html             # Main HTML
│   ├── app.js                 # Application logic
│   ├── package.json           # Dependencies
│   ├── components/            # UI components
│   ├── services/              # API services
│   ├── styles/                # CSS styles
│   ├── utils/                 # Utilities
│   └── tests/                 # Test suites
│
├── validator/                 # Validator dashboard
│   ├── Dockerfile
│   ├── index.html
│   ├── app.js
│   ├── package.json
│   ├── components/
│   ├── services/
│   ├── assets/
│   └── tests/
│
└── governance/                # Governance portal
    ├── Dockerfile
    ├── index.html
    ├── app.js
    ├── components/
    ├── services/
    ├── assets/
    └── tests/
```

## Development

### Local Development (Without Docker)

Each dashboard can be run locally for development:

```bash
# Staking Dashboard
cd dashboards/staking
python -m http.server 8080

# Validator Dashboard
cd dashboards/validator
npm start  # or: http-server -p 8080

# Governance Portal
cd dashboards/governance
python -m http.server 8080
```

**Note**: Update API URLs in the service files for local development:
- `services/stakingAPI.js`
- `services/validatorAPI.js`
- `services/governanceAPI.js`

### Running Tests

#### Staking Dashboard
```bash
cd dashboards/staking
npm install
npm test                 # Run all tests
npm run test:coverage    # With coverage report
npm run test:watch       # Watch mode
```

#### Validator Dashboard
```bash
cd dashboards/validator
npm install
npm test                 # All tests with coverage
npm run test:unit        # Unit tests only
npm run test:integration # Integration tests
npm run test:e2e         # End-to-end tests
```

#### Governance Portal
```bash
cd dashboards/governance
# Tests run in browser via tests/test-runner.html
open tests/test-runner.html
```

### Building Images Manually

```bash
# Build specific dashboard
docker build -t paw-staking:latest dashboards/staking/

# Build all dashboards
docker compose -f compose/docker-compose.dashboards.yml build
```

## Troubleshooting

### Dashboards Not Accessible

**Symptoms**: HTTP 502/503 or connection refused

**Solutions**:
1. Check container status:
   ```bash
   docker compose -f compose/docker-compose.dashboards.yml ps
   ```

2. Check logs:
   ```bash
   docker compose -f compose/docker-compose.dashboards.yml logs staking-dashboard
   ```

3. Verify health:
   ```bash
   ./scripts/verify-dashboards.sh
   ```

4. Restart services:
   ```bash
   ./scripts/stop-dashboards.sh
   ./scripts/deploy-dashboards.sh
   ```

### API Connection Errors

**Symptoms**: "Disconnected" status or "Failed to fetch" errors

**Solutions**:
1. Verify blockchain node is running:
   ```bash
   curl http://localhost:1317/cosmos/base/tendermint/v1beta1/node_info
   curl http://localhost:26657/status
   ```

2. Check environment variables:
   ```bash
   docker exec paw-staking-dashboard cat /usr/share/nginx/html/config.js
   ```

3. Update endpoints in `docker-compose.dashboards.yml`

4. Rebuild and redeploy:
   ```bash
   ./scripts/deploy-dashboards.sh
   ```

### WebSocket Connection Failed (Validator Dashboard)

**Symptoms**: No real-time updates, connection errors in console

**Solutions**:
1. Verify WebSocket endpoint:
   ```bash
   wscat -c ws://localhost:26657/websocket
   ```

2. Check firewall/proxy settings

3. Ensure `PAW_WS_URL` is correctly set:
   ```yaml
   - PAW_WS_URL=ws://localhost:26657/websocket  # Development
   - PAW_WS_URL=wss://rpc.paw.network:26657/websocket  # Production (SSL)
   ```

### Wallet Connection Issues

**Symptoms**: Keplr wallet won't connect

**Solutions**:
1. Install Keplr wallet extension
2. Unlock wallet
3. Add PAW chain to Keplr (if not already added)
4. Clear browser cache and cookies
5. Try different browser

### Container Health Checks Failing

**Symptoms**: Container shows "unhealthy" status

**Solutions**:
1. Check nginx configuration:
   ```bash
   docker exec paw-staking-dashboard nginx -t
   ```

2. View detailed health logs:
   ```bash
   docker inspect paw-staking-dashboard | jq '.[0].State.Health'
   ```

3. Increase health check timeout in `docker-compose.dashboards.yml`

## Security Best Practices

### Production Deployment

1. **Use HTTPS**: Always serve dashboards over HTTPS in production
   ```nginx
   server {
       listen 443 ssl http2;
       ssl_certificate /path/to/cert.pem;
       ssl_certificate_key /path/to/key.pem;
   }
   ```

2. **Configure CORS**: Restrict API access to dashboard domains
   ```yaml
   environment:
     - CORS_ALLOWED_ORIGINS=https://staking.paw.network,https://governance.paw.network
   ```

3. **Rate Limiting**: Implement rate limiting on nginx
   ```nginx
   limit_req_zone $binary_remote_addr zone=dashboard:10m rate=10r/s;
   limit_req zone=dashboard burst=20;
   ```

4. **Security Headers**: Add security headers to nginx config
   ```nginx
   add_header X-Frame-Options "SAMEORIGIN" always;
   add_header X-Content-Type-Options "nosniff" always;
   add_header X-XSS-Protection "1; mode=block" always;
   add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com; style-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com;" always;
   ```

### Wallet Security

- **Never store private keys**: All transactions signed in wallet extension
- **Verify transactions**: Always review transaction details before signing
- **Use hardware wallets**: When possible, use Ledger with Keplr
- **Check URLs**: Verify dashboard URL before connecting wallet

## Monitoring

### Health Checks

Automated health checks run every 30 seconds:

```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:3000/"]
  interval: 30s
  timeout: 3s
  retries: 3
  start_period: 5s
```

### Metrics

Monitor dashboard metrics:

```bash
# Container stats
docker stats paw-staking-dashboard paw-validator-dashboard paw-governance-portal

# Resource usage
./scripts/verify-dashboards.sh
```

### Logs

Access dashboard logs:

```bash
# All dashboards
docker compose -f compose/docker-compose.dashboards.yml logs -f

# Specific dashboard
docker compose -f compose/docker-compose.dashboards.yml logs -f staking-dashboard

# Last 100 lines
docker compose -f compose/docker-compose.dashboards.yml logs --tail=100 staking-dashboard
```

## Performance Optimization

### Caching

All dashboards implement response caching:
- Staking Dashboard: 30-second cache TTL
- Validator Dashboard: Real-time with intelligent refresh
- Governance Portal: Configurable cache per endpoint

### CDN Usage

Dashboards use CDNs for external resources:
- Font Awesome icons
- Chart.js library
- CSS frameworks

For production, consider hosting these locally or using your own CDN.

### Nginx Optimizations

Nginx is configured for optimal performance:
- Gzip compression enabled
- Static file caching (1 hour)
- HTML no-cache policy
- Keep-alive connections

## Roadmap

### Planned Features

**Staking Dashboard**:
- Mobile wallet support (Keplr Mobile)
- Advanced charting with TradingView
- Dark mode theme
- Multi-language support (i18n)
- Automated delegation strategies
- Tax reporting tools

**Validator Dashboard**:
- Mobile app version
- Advanced analytics and AI predictions
- Automated alerts via webhooks
- Governance integration
- Multi-validator comparison tools
- Historical data export (CSV, JSON)

**Governance Portal**:
- Mobile app version
- Advanced proposal analytics
- Delegation-weighted voting calculator
- Discussion forum integration
- Proposal template library
- Automated proposal monitoring

## Support & Resources

### Documentation

- **Main Docs**: `/docs/README.md`
- **API Reference**: PAW Cosmos SDK REST API
- **Test Coverage**: Each dashboard includes comprehensive tests

### Getting Help

1. **Check logs**: `docker compose logs`
2. **Verify health**: `./scripts/verify-dashboards.sh`
3. **Review README**: Each dashboard has detailed README
4. **Test locally**: Run dashboards without Docker for debugging

### Contributing

Contributions welcome! Each dashboard follows:
- **Code Style**: ESLint + Prettier configurations
- **Testing**: Jest for unit/integration tests
- **Documentation**: Update README when adding features
- **Security**: Follow OWASP best practices

## Version Information

- **Dashboards Version**: 1.0.0
- **Docker Base Image**: nginx:alpine
- **Node.js Version**: 16+ (for development)
- **Browser Support**: Chrome 90+, Firefox 88+, Safari 14+, Edge 90+

## License

Part of the PAW blockchain project. See main repository LICENSE file.

---

**Last Updated**: 2025-12-14
**Status**: Production Ready
**Maintained By**: PAW Development Team
