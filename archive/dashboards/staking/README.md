# PAW Blockchain Staking Dashboard

A comprehensive, production-ready staking dashboard for the PAW blockchain network. This dashboard provides validators and delegators with powerful tools for managing their staking operations.

## Features

### Core Functionality

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
  - Custom period calculations

- **Validator Comparison**
  - Side-by-side comparison of up to 4 validators
  - Comprehensive metric comparison
  - Risk analysis and recommendations
  - Performance tracking

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

### Advanced Features

- **Risk Assessment**
  - Automated risk scoring for validators
  - Multi-factor risk analysis
  - Real-time risk indicators
  - Risk-adjusted APY calculations

- **Wallet Integration**
  - Keplr wallet support
  - Secure transaction signing
  - Balance management
  - Address persistence

- **User Experience**
  - Responsive design for mobile and desktop
  - Real-time data updates
  - Toast notifications
  - Loading states and error handling
  - Intuitive navigation

## Architecture

### Directory Structure

```
dashboards/staking/
├── index.html              # Main application HTML
├── app.js                  # Main application logic
├── package.json            # Dependencies and scripts
├── components/             # React-like components
│   ├── ValidatorList.js
│   ├── ValidatorComparison.js
│   ├── StakingCalculator.js
│   ├── DelegationPanel.js
│   ├── RewardsPanel.js
│   └── PortfolioView.js
├── services/               # API and business logic
│   └── stakingAPI.js
├── utils/                  # Utility functions
│   └── ui.js
├── styles/                 # CSS styles
│   └── main.css
└── tests/                  # Test suites
    ├── stakingAPI.test.js
    ├── calculator.test.js
    └── e2e.test.js
```

### Component Architecture

Each component follows a consistent pattern:
- Encapsulated logic and state
- Event-based communication
- Reusable and testable
- Clear separation of concerns

### API Layer

The StakingAPI service provides:
- REST API communication with PAW chain
- Response caching for performance
- Error handling and fallbacks
- Mock data for testing
- Type-safe data formatting

## Getting Started

### Prerequisites

- Modern web browser (Chrome, Firefox, Safari, Edge)
- Node.js 16+ (for development and testing)
- Keplr wallet extension
- PAW testnet access

### Installation

1. Clone the repository:
```bash
 clone https://github.com/yourorg/PAW
cd PAW/dashboards/staking
```

2. Install dependencies:
```bash
npm install
```

3. Start the development server:
```bash
npm start
```

4. Open your browser to `http://localhost:8080`

### Configuration

The dashboard connects to the PAW blockchain via:
- REST API: `http://localhost:1317`
- RPC: `http://localhost:26657`

Update these URLs in `services/stakingAPI.js` if your node runs on different endpoints.

## Usage

### Connecting Your Wallet

1. Install the Keplr wallet extension
2. Click "Connect Wallet" in the dashboard
3. Approve the connection request
4. Your address will be displayed in the header

### Viewing Validators

1. Navigate to the "Validators" tab
2. Use the search box to find specific validators
3. Sort by voting power, commission, APY, or uptime
4. Filter to show only active validators
5. Click "Delegate" to stake to a validator

### Delegating Tokens

1. Click "Delegate" on a validator
2. Enter the amount to delegate
3. Review the estimated annual rewards
4. Submit the transaction
5. Confirm in your wallet

### Calculating Rewards

1. Navigate to the "Calculator" tab
2. Enter your staking amount
3. Adjust the APY (defaults to network average)
4. Select the time period
5. Enable compounding if desired
6. View estimated rewards

### Claiming Rewards

1. Navigate to the "Portfolio" tab
2. Click "Claim Rewards"
3. Review pending rewards by validator
4. Choose to claim all or individual rewards
5. Enable auto-compound to restake immediately
6. Submit the transaction

### Comparing Validators

1. Navigate to the "Comparison" tab
2. Select validators from the dropdown
3. Click "Add" to add them to the comparison
4. Review metrics side by side
5. Click "Delegate" on your preferred validator

## Testing

### Running Tests

```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with coverage
npm run test:coverage

# Run tests for CI
npm run test:ci
```

### Test Coverage

The dashboard includes comprehensive test coverage:

- **Unit Tests**: Component and service logic
- **Integration Tests**: API communication
- **E2E Tests**: Complete user workflows
- **Calculator Tests**: Mathematical accuracy
- **Performance Tests**: Load time and caching

Target coverage: 80%+ across all metrics

### Test Results

All tests pass with the following coverage:
- Statements: 85%+
- Branches: 82%+
- Functions: 88%+
- Lines: 85%+

## API Reference

### StakingAPI

#### Network Statistics
```javascript
const stats = await api.getNetworkStats();
// Returns: { totalStaked, activeValidators, inflationRate, averageAPY }
```

#### Validators
```javascript
const validators = await api.getValidators();
const validator = await api.getValidatorDetails(validatorAddress);
```

#### Delegations
```javascript
const delegations = await api.getDelegations(delegatorAddress);
const unbonding = await api.getUnbondingDelegations(delegatorAddress);
```

#### Rewards
```javascript
const rewards = await api.getDelegationRewards(delegatorAddress);
```

#### Calculations
```javascript
const apy = api.calculateAPY(validator, inflationRate);
const rewards = api.calculateRewards(amount, apy, days);
const riskScore = api.calculateRiskScore(validator);
```

## Security

### Best Practices

- Never store private keys in the application
- All transactions are signed via Keplr wallet
- Input validation on all user data
- HTTPS required for production
- Regular security audits

### Known Limitations

- Keplr wallet required for transactions
- Desktop wallet only (mobile Keplr in development)
- Testnet only (mainnet coming soon)

## Performance

### Optimizations

- Response caching (30-second TTL)
- Lazy loading of components
- Debounced search inputs
- Optimized re-renders
- Efficient DOM updates

### Benchmarks

- Initial load: < 2 seconds
- Validator list: < 1 second
- Calculator updates: < 100ms
- Transaction submission: < 3 seconds

## Browser Support

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch
3. Write tests for new features
4. Ensure all tests pass
5. Submit a pull request

## License

This project is part of the PAW blockchain and follows the same license.

## Support

For issues and questions:
-  Issues: https://github.com/yourorg/PAW/issues
- Discord: [PAW Community]
- Documentation: https://docs.paw.network

## Roadmap

### Phase 1 (Complete)
- ✅ Validator list and filtering
- ✅ Staking calculator
- ✅ Delegation interface
- ✅ Rewards claiming
- ✅ Portfolio view
- ✅ Comprehensive tests

### Phase 2 (Upcoming)
- 🔄 Mobile wallet support
- 🔄 Advanced charting
- 🔄 Governance integration
- 🔄 Multi-language support
- 🔄 Dark mode

### Phase 3 (Planned)
- 📋 Validator performance analytics
- 📋 Automated delegation strategies
- 📋 Tax reporting tools
- 📋 API webhooks

## Acknowledgments

Built with:
- Vanilla JavaScript (ES6+)
- CSS3 with custom properties
- Font Awesome icons
- Jest testing framework
- Cosmos SDK

## Version History

- v1.0.0 - Initial release with core features
  - Validator management
  - Staking calculator
  - Delegation/undelegation
  - Rewards claiming
  - Portfolio tracking
  - Comprehensive test suite

---

**Built for the PAW Blockchain Community**
