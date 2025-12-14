# PAW Validator Dashboard

A comprehensive, real-time dashboard for monitoring and managing validators on the PAW blockchain.

## Features

### Core Functionality
- **Real-time Validator Monitoring** - Live updates via WebSocket connection
- **Multi-Validator Support** - Monitor multiple validators from one dashboard
- **Comprehensive Statistics** - Uptime, rewards, delegations, and performance metrics
- **Responsive Design** - Works on desktop, tablet, and mobile devices

### Dashboard Sections

#### 1. Overview
- Validator status and health
- Total staked tokens
- Commission rate
- Total rewards earned
- Number of delegators
- Current uptime percentage
- Recent activity feed

#### 2. Delegation Management
- Complete list of all delegators
- Delegation amounts and shares
- Pending rewards per delegator
- Search and filter functionality
- Sort by amount, date, or rewards
- Export capabilities

#### 3. Rewards Tracking
- Historical rewards visualization
- Interactive charts (line, bar, area)
- Multiple timeframes (7d, 30d, 90d, 1y, all)
- Total distributed rewards
- Pending rewards
- Commission earned
- Trend analysis

#### 4. Performance Metrics
- Voting power percentage
- Block proposal statistics
- Miss rate tracking
- Historical performance charts
- Comparative analytics

#### 5. Uptime Monitoring
- Real-time uptime percentage
- Block signing visualization
- 24-hour uptime timeline
- Uptime alerts and warnings
- Signing window status
- Time to slash threshold

#### 6. Signing Statistics
- Total blocks signed
- Total blocks missed
- Signing rate percentage
- Signing history visualization
- Pattern analysis

#### 7. Slash Events
- Complete slash event history
- Event details and reasons
- Amount slashed
- Block height information
- Empty state for no events

#### 8. Settings
- Commission rate updates
- Validator information editing
- Alert configuration
- Email notification setup
- Custom thresholds

## Installation

### Prerequisites
- Node.js 16+ (for development)
- Docker & Docker Compose (for containerized deployment)
- Access to PAW blockchain endpoints (LCD and WebSocket)

### Local Development

1. **Clone the repository**
```bash
cd dashboards/validator
```

2. **Install dependencies**
```bash
npm install
```

3. **Start development server**
```bash
npm run dev
```

4. **Open in browser**
```
http://localhost:8080
```

### Docker Deployment

1. **Start the dashboard**
```bash
docker-compose up -d
```

2. **View logs**
```bash
docker-compose logs -f validator-dashboard
```

3. **Stop the dashboard**
```bash
docker-compose down
```

## Configuration

### API Endpoints

Edit `services/validatorAPI.js` to configure blockchain endpoints:

```javascript
static baseURL = 'http://localhost:1317'; // LCD endpoint
```

Edit `services/websocket.js` to configure WebSocket endpoint:

```javascript
this.wsURL = 'ws://localhost:26657/websocket'; // Tendermint WebSocket
```

### Environment Variables

Create a `.env` file in the dashboard directory:

```env
PAW_LCD_ENDPOINT=http://localhost:1317
PAW_WS_ENDPOINT=ws://localhost:26657/websocket
PAW_CHAIN_ID=paw-1
```

## Usage

### Adding a Validator

1. Click the "Add Validator" button in the header
2. Enter the validator address (e.g., `pawvaloper1...`)
3. Optionally provide a display name
4. Click "Add"

The validator will be saved in local storage and automatically loaded on subsequent visits.

### Monitoring Multiple Validators

Use the validator dropdown in the header to switch between validators. All validators are saved locally and persist across sessions.

### Setting Up Alerts

1. Navigate to Settings
2. Scroll to "Alert Settings"
3. Enable desired alert types:
   - Email alerts
   - Uptime alerts
   - Slashing alerts
4. Configure email address
5. Save settings

### Updating Commission

**Note:** Commission updates require transaction signing and are not available directly in the web interface. Use the PAW CLI:

```bash
pawd tx staking edit-validator --commission-rate 0.05 --from validator
```

## Testing

### Unit Tests
```bash
npm run test:unit
```

### Integration Tests
```bash
npm run test:integration
```

### E2E Tests
```bash
npm run test:e2e
```

### All Tests with Coverage
```bash
npm test
```

### Watch Mode (Development)
```bash
npm run test:watch
```

## Architecture

### Components

- **ValidatorCard** - Displays detailed validator information
- **DelegationList** - Shows and manages delegations
- **RewardsChart** - Visualizes rewards over time
- **UptimeMonitor** - Real-time uptime tracking

### Services

- **validatorAPI.js** - Handles all blockchain API calls
- **websocket.js** - Manages WebSocket connections for real-time updates

### File Structure

```
dashboards/validator/
├── index.html              # Main dashboard page
├── app.js                  # Dashboard application logic
├── assets/
│   └── css/
│       └── styles.css      # Dashboard styles
├── components/
│   ├── ValidatorCard.js    # Validator info component
│   ├── DelegationList.js   # Delegations component
│   ├── RewardsChart.js     # Rewards visualization
│   └── UptimeMonitor.js    # Uptime monitoring
├── services/
│   ├── validatorAPI.js     # API service
│   └── websocket.js        # WebSocket service
├── tests/
│   ├── unit/               # Unit tests
│   ├── integration/        # Integration tests
│   └── e2e/                # End-to-end tests
├── docker-compose.yml      # Docker configuration
├── nginx.conf              # Nginx configuration
├── package.json            # Node.js dependencies
└── README.md              # This file
```

## API Reference

### ValidatorAPI

#### `getValidatorInfo(validatorAddress)`
Fetches comprehensive validator information.

#### `getDelegations(validatorAddress)`
Retrieves all delegations for a validator.

#### `getRewards(validatorAddress)`
Gets reward information and history.

#### `getPerformance(validatorAddress)`
Fetches performance metrics.

#### `getUptime(validatorAddress)`
Retrieves uptime data and signing history.

#### `getSigningStats(validatorAddress)`
Gets signing statistics.

#### `getSlashEvents(validatorAddress)`
Retrieves slash event history.

### WebSocket Events

- `connected` - WebSocket connection established
- `disconnected` - WebSocket connection lost
- `newBlock` - New block added to chain
- `validatorUpdate` - Validator data updated
- `validatorSetUpdate` - Validator set changed
- `validatorTransaction` - Validator-related transaction

## Browser Support

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## Performance

- **Initial Load:** < 2s
- **Dashboard Refresh:** < 500ms
- **Real-time Updates:** 6s block time
- **Supports:** 1000+ delegations per validator

## Security

- XSS protection via HTML escaping
- CORS handling via proxy
- Secure WebSocket connections (wss://)
- No private key handling in browser
- Read-only dashboard (no transaction signing)

## Troubleshooting

### WebSocket Connection Failed
- Check that the blockchain node is running
- Verify WebSocket endpoint in configuration
- Check firewall rules

### API Requests Timeout
- Verify LCD endpoint is accessible
- Check network connectivity
- Increase timeout in `validatorAPI.js`

### No Data Displayed
- Ensure validator address is correct
- Check that validator exists on chain
- Verify API endpoints are responding

### Docker Container Won't Start
```bash
# Check logs
docker-compose logs validator-dashboard

# Restart containers
docker-compose restart

# Rebuild if needed
docker-compose up -d --build
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run test suite
6. Submit a pull request

## License

Apache 2.0 - See LICENSE file for details

## Support

- Documentation: https://docs.paw.network
- Discord: https://discord.gg/paw
-  Issues: https://github.com/paw-network/paw/issues

## Roadmap

- [ ] Mobile app version
- [ ] Advanced analytics
- [ ] Automated alerts via webhooks
- [ ] Multi-language support
- [ ] Historical data export
- [ ] Validator comparison tools
- [ ] Governance integration
- [ ] Staking calculator
