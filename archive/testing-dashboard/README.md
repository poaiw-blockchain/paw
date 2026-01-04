# PAW Testing Control Panel

A comprehensive, user-friendly testing dashboard for the PAW blockchain. This control panel allows both technical and non-technical users to interact with, monitor, and test the PAW blockchain across local testnet, public testnet, and mainnet environments.

![PAW Testing Control Panel](https://img.shields.io/badge/version-1.0.0-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

## Features

### Network Management
- **One-Click Network Switching**: Easily switch between Local Testnet, Public Testnet, and Mainnet
- **Real-Time Status Monitoring**: Live connection status with visual indicators
- **Auto-Detection**: Automatically detects local testnet when available
- **Read-Only Mainnet**: Safe monitoring mode for mainnet with transaction protection

### Quick Actions
- **Send Transactions**: Simple form to send test transactions
- **Create Wallets**: One-click wallet generation with mnemonic backup
- **Delegate Tokens**: Easy staking interface with validator selection
- **Submit Proposals**: Streamlined governance proposal submission
- **Swap Tokens**: DEX integration for token swaps
- **Query Balances**: Instant balance checks for any address

### Monitoring Dashboard
- **Live Block Updates**: Real-time block production monitoring
- **Transaction Feed**: Recent transactions with status indicators
- **Validator Status**: Active validators with voting power and commission
- **Governance Proposals**: Track active proposals and voting
- **Liquidity Pools**: DEX pool monitoring with volume data
- **System Metrics**: CPU, Memory, and Disk usage visualization

### Testing Tools
- **Transaction Simulator**: Build and test transactions before sending
- **Bulk Wallet Generator**: Create multiple test wallets at once
- **Load Testing**: Stress test the network with configurable parameters
- **Faucet Integration**: Request test tokens for development
- **Pre-Built Test Scenarios**: Automated test flows for common operations

### User Experience
- **No-Code Interface**: All actions via buttons and simple forms
- **Test Data Auto-Fill**: One-click population of test values
- **Tooltips & Help**: Contextual help on every element
- **Live Logs**: Real-time event and error logging
- **Dark/Light Theme**: Customizable appearance
- **Mobile Responsive**: Works on all devices

## Quick Start

### Prerequisites
- Modern web browser (Chrome, Firefox, Safari, Edge)
- For local testnet: Running PAW node
- For public testnet: Internet connection

### Option 1: Direct Access (Simplest)

1. **Open the Dashboard**
   ```bash
   # Simply open index.html in your browser
   open index.html
   # or double-click index.html in your file explorer
   ```

2. **Select Network**
   - Use the dropdown in the top navigation
   - Start with "Local Testnet" if you have a node running
   - Or select "Public Testnet" to connect to the public network

3. **Start Testing!**
   - Green indicator = Connected and ready
   - Use Quick Actions in the left sidebar
   - Monitor activity in real-time

### Option 2: Docker Deployment (Recommended)

1. **Start Dashboard Only**
   ```bash
   docker-compose up -d dashboard
   ```

2. **Access Dashboard**
   - Open browser to: http://localhost:8080
   - Dashboard is now running!

3. **Optional: Start with Local Node**
   ```bash
   # Includes local PAW node and faucet
   docker-compose --profile with-node --profile with-faucet up -d
   ```

### Option 3: Local Web Server

```bash
# Python 3
python -m http.server 8080

# Node.js
npx http-server -p 8080

# Then open: http://localhost:8080
```

## User Guide

### For Non-Technical Users

#### Creating Your First Wallet

1. Click **"Create Wallet"** in Quick Actions
2. Save the **mnemonic phrase** securely (write it down!)
3. Copy your wallet **address**
4. Done! You now have a wallet

#### Getting Test Tokens

1. Click **"Request Tokens"** in Testing Tools
2. Enter your wallet address
3. Click "Request Tokens"
4. Wait 10-30 seconds for tokens to arrive
5. Use "Query Balance" to check your balance

#### Sending a Transaction

1. Click **"Send Transaction"** in Quick Actions
2. Fill in:
   - **From**: Your wallet address
   - **To**: Recipient address
   - **Amount**: Amount to send (in upaw)
3. Or click **"Use Test Data"** to auto-fill
4. Click "Send Transaction"
5. Watch the logs for confirmation

#### Testing Complete Flows

1. Click any **Test Scenario** button (bottom left)
2. Watch the logs as the test runs automatically
3. See success/failure status
4. All steps are logged in real-time

### For Technical Users

#### Network Configuration

Edit `config.js` to customize network endpoints:

```javascript
export const CONFIG = {
    networks: {
        local: {
            rpcUrl: 'http://localhost:26657',
            restUrl: 'http://localhost:1317',
            // ...
        }
    }
};
```

#### Custom Test Scenarios

Add custom test scenarios in `services/testing.js`:

```javascript
async runCustomTest() {
    monitoringService.addLog('info', 'Running custom test...');
    // Your test logic here
}
```

#### API Integration

Use blockchain service directly:

```javascript
import blockchainService from './services/blockchain.js';

// Query data
const blocks = await blockchainService.getRecentBlocks(10);
const validators = await blockchainService.getValidators();

// Send transaction
const result = await blockchainService.sendTransaction({
    from: 'paw1...',
    to: 'paw1...',
    amount: '1000000'
});
```

## Dashboard Components

### Network Overview
- **Block Height**: Current blockchain height
- **TPS**: Transactions per second (calculated)
- **Peers**: Connected peer count
- **Consensus**: Network consensus status

### Tabs

#### Recent Blocks
- Latest blocks with height, hash, proposer
- Transaction count per block
- Timestamp

#### Recent Transactions
- Transaction hash and type
- Status (Success/Failed)
- Real-time updates

#### Validators
- Active/Inactive status
- Voting power
- Commission rates
- Delegation actions

#### Proposals
- Governance proposals
- Voting status and deadlines
- Vote buttons

#### Liquidity Pools
- DEX pools
- Token pairs and liquidity
- 24h volume
- Add liquidity/Swap actions

### Live Logs
- All actions logged in real-time
- Color-coded by severity (info/success/warning/error)
- Export logs as JSON
- Auto-scroll to latest

### System Metrics
- CPU usage visualization
- Memory consumption
- Disk I/O activity

## Testing Scenarios

### Transaction Flow
1. Creates a test wallet
2. Requests tokens from faucet
3. Queries wallet balance
4. Simulates transaction sending

### Staking Flow
1. Fetches active validators
2. Gets staking information
3. Simulates token delegation

### Governance Flow
1. Lists active proposals
2. Simulates proposal submission
3. Simulates voting

### DEX Trading Flow
1. Fetches liquidity pools
2. Simulates token swap
3. Simulates liquidity addition

## Configuration

### Network Endpoints

Configure in `config.js`:
- RPC URLs
- REST API URLs
- Chain IDs
- Faucet URLs
- Explorer URLs

### Update Intervals

Customize refresh rates:
```javascript
updateIntervals: {
    blockUpdates: 3000,      // 3 seconds
    metricsUpdates: 5000,    // 5 seconds
    logsUpdates: 2000,       // 2 seconds
    eventsUpdates: 3000      // 3 seconds
}
```

### UI Settings

```javascript
ui: {
    logsMaxEntries: 100,     // Max log entries
    eventsMaxEntries: 50,    // Max events
    tablePageSize: 10,       // Rows per page
    autoRefresh: true,       // Auto-refresh data
    theme: 'light'           // Default theme
}
```

## Troubleshooting

### Red Status Indicator

**Problem**: Dashboard shows "Disconnected"

**Solutions**:
1. Check if your PAW node is running
2. Verify network URLs in config.js
3. Try switching to Public Testnet
4. Check browser console for errors

### No Data Loading

**Problem**: Tables show "Loading..." indefinitely

**Solutions**:
1. Check network connection
2. Verify node is synced
3. Try refreshing the page
4. Check browser console for API errors

### Transactions Not Working

**Problem**: Transactions fail or don't send

**Solutions**:
1. Ensure you're not on Mainnet (read-only)
2. Check wallet has sufficient balance
3. Verify addresses are correct
4. Check logs for error messages

### Faucet Not Working

**Problem**: Faucet requests fail

**Solutions**:
1. Verify faucet is running (local testnet)
2. Check faucet URL in config
3. Ensure address format is correct
4. Check rate limits

## Development

### Project Structure

```
testing-dashboard/
├── index.html              # Main HTML file
├── styles.css             # All styling
├── config.js              # Configuration
├── app.js                 # Main application
├── services/              # Core services
│   ├── blockchain.js      # Blockchain API
│   ├── monitoring.js      # Real-time monitoring
│   └── testing.js         # Testing utilities
├── components/            # UI components
│   ├── NetworkSelector.js
│   ├── QuickActions.js
│   ├── LogViewer.js
│   └── MetricsDisplay.js
├── tests/                 # Test files
│   ├── dashboard.test.js
│   ├── actions.test.js
│   └── monitoring.test.js
├── docker-compose.yml     # Docker deployment
├── nginx.conf            # Nginx configuration
└── README.md             # This file
```

### Running Tests

```bash
# Install dependencies (if using test framework)
npm install

# Run tests
npm test

# Run specific test suite
npm test -- dashboard.test.js
```

### Building for Production

1. **Optimize Assets**
   ```bash
   # Minify JavaScript
   npm run build

   # Optimize images
   npm run optimize-images
   ```

2. **Deploy**
   ```bash
   # Deploy to web server
   docker-compose up -d dashboard
   ```

## Security Considerations

### Mainnet Protection
- Read-only mode enforced
- Warning dialogs for sensitive actions
- Transaction confirmation required
- No private key storage

### Local Development
- Never commit private keys
- Use test wallets only
- Rotate faucet wallet regularly
- Limit faucet amounts

### Best Practices
- Always verify addresses before sending
- Save mnemonic phrases securely offline
- Use hardware wallets for real funds
- Test on testnet first

## API Reference

### Blockchain Service

```javascript
// Network Management
await blockchainService.switchNetwork('testnet')
await blockchainService.checkConnection()

// Data Retrieval
await blockchainService.getLatestBlock()
await blockchainService.getRecentBlocks(count)
await blockchainService.getRecentTransactions(limit)
await blockchainService.getValidators()
await blockchainService.getProposals()
await blockchainService.getLiquidityPools()
await blockchainService.queryBalance(address)

// Actions
await blockchainService.sendTransaction(txData)
await blockchainService.createWallet()
await blockchainService.requestFaucet(address)
```

### Monitoring Service

```javascript
// Monitoring Control
monitoringService.startMonitoring()
monitoringService.stopMonitoring()

// Event Listeners
monitoringService.on('blockUpdate', callback)
monitoringService.on('metricsUpdate', callback)
monitoringService.on('newEvent', callback)
monitoringService.on('newLog', callback)

// Metrics
await monitoringService.calculateTPS()
await monitoringService.getPeerCount()
await monitoringService.getConsensusStatus()
await monitoringService.getNetworkHealth()

// Logging
monitoringService.addLog('info', 'Message')
```

### Testing Service

```javascript
// Test Scenarios
await testingService.runTransactionFlowTest()
await testingService.runStakingFlowTest()
await testingService.runGovernanceFlowTest()
await testingService.runDEXFlowTest()

// Utilities
await testingService.generateBulkWallets(count)
await testingService.simulateTransaction(txData)
await testingService.runLoadTest(config)

// Results
testingService.getTestResults()
testingService.exportResults('json')
```

## FAQ

### Can I use this on mainnet?

Yes, but only in **read-only mode**. You can monitor mainnet activity, but cannot send transactions. This is a safety feature to prevent accidental real-value transactions.

### Do I need to run a local node?

No! You can use the Public Testnet option to connect to hosted nodes. Local node is only needed for Local Testnet testing.

### Is my private key secure?

The dashboard generates wallets in your browser only. **Never enter real private keys or mnemonics**. This is a testing tool - use test wallets only.

### Can I customize the dashboard?

Absolutely! All code is open and documented. Edit any component or service to add custom functionality.

### How do I report bugs?

Create an issue in the PAW repository with:
- Dashboard version
- Browser and OS
- Network (local/testnet/mainnet)
- Steps to reproduce
- Error logs (from browser console)

## Support

### Documentation
- [PAW Documentation](https://docs.paw.network)
- [API Reference](https://docs.paw.network/api)
- [Developer Guide](https://docs.paw.network/developers)

### Community
- Discord: [PAW Community](https://discord.gg/DBHTc2QV)
- Forum: [PAW Forum](https://forum.paw.network)
- Twitter: [@PAWNetwork](https://twitter.com/PAWNetwork)

### Issues
-  Issues: [Report Bug](https://github.com/paw/blockchain/issues)
- Email: support@paw.network

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

### Development Workflow

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

### Code Style

- Use ESLint for JavaScript
- Follow existing patterns
- Document all functions
- Add comments for complex logic

## License

MIT License - see [LICENSE](../LICENSE) for details

## Changelog

### Version 1.0.0 (Current)
- Initial release
- Full network support (local/testnet/mainnet)
- All quick actions implemented
- Real-time monitoring
- Pre-built test scenarios
- Dark/light theme
- Mobile responsive design

### Planned Features
- Transaction history export
- Custom test scenario builder
- Validator performance analytics
- Governance voting history
- DEX trading charts
- Multi-language support

---

**Made with ❤️ for the PAW Community**

For questions or feedback, please open an issue or contact the development team.
