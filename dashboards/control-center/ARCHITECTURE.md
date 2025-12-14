# PAW Testing Dashboard - Architecture Overview

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Browser (Client)                          │
│                                                                   │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                     index.html (UI)                        │  │
│  │  ┌─────────────┬──────────────────────┬────────────────┐  │  │
│  │  │   Header    │   Network Selector   │  Theme Toggle  │  │  │
│  │  └─────────────┴──────────────────────┴────────────────┘  │  │
│  │  ┌─────────────┬──────────────────────┬────────────────┐  │  │
│  │  │   Left      │   Main Content       │     Right      │  │  │
│  │  │  Sidebar    │   Area (Tabs)        │    Sidebar     │  │  │
│  │  │             │                      │                │  │  │
│  │  │ Quick       │ ┌──────────────────┐ │   Live Logs   │  │  │
│  │  │ Actions     │ │ Recent Blocks    │ │   & Events    │  │  │
│  │  │             │ ├──────────────────┤ │                │  │  │
│  │  │ Testing     │ │ Transactions     │ │   System      │  │  │
│  │  │ Tools       │ ├──────────────────┤ │   Metrics     │  │  │
│  │  │             │ │ Validators       │ │                │  │  │
│  │  │ Test        │ ├──────────────────┤ │                │  │  │
│  │  │ Scenarios   │ │ Proposals        │ │                │  │  │
│  │  │             │ ├──────────────────┤ │                │  │  │
│  │  │             │ │ Liquidity Pools  │ │                │  │  │
│  │  └─────────────┴─┴──────────────────┘─┴────────────────┘  │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                   │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                      app.js (Main)                         │  │
│  │  ┌─────────────────────────────────────────────────────┐  │  │
│  │  │  Event Coordination & State Management              │  │  │
│  │  └─────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                   │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                   Components Layer                         │  │
│  │  ┌──────────────┬──────────────┬──────────────┬─────────┐ │  │
│  │  │ Network      │ Quick        │ Log          │ Metrics │ │  │
│  │  │ Selector     │ Actions      │ Viewer       │ Display │ │  │
│  │  └──────────────┴──────────────┴──────────────┴─────────┘ │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                   │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                    Services Layer                          │  │
│  │  ┌──────────────┬──────────────┬──────────────────────┐   │  │
│  │  │ Blockchain   │ Monitoring   │ Testing              │   │  │
│  │  │ Service      │ Service      │ Service              │   │  │
│  │  └──────────────┴──────────────┴──────────────────────┘   │  │
│  └───────────────────────────────────────────────────────────┘  │
│                            ↓                                     │
└────────────────────────────┼─────────────────────────────────────┘
                             ↓
                  ┌──────────┴──────────┐
                  │   Network Layer     │
                  │                     │
                  │  HTTP/REST API      │
                  └──────────┬──────────┘
                             ↓
        ┌────────────────────┼────────────────────┐
        ↓                    ↓                    ↓
   ┌─────────┐         ┌─────────┐         ┌─────────┐
   │  Local  │         │ Public  │         │ Mainnet │
   │ Testnet │         │ Testnet │         │(Read-Only)│
   └─────────┘         └─────────┘         └─────────┘
```

## Component Architecture

### 1. UI Layer (index.html + styles.css)

**Purpose**: User interface and visual presentation

**Components**:
- Header with network selector
- Left sidebar (Quick Actions, Testing Tools, Scenarios)
- Main content area (Tabs: Blocks, TXs, Validators, Proposals, Pools)
- Right sidebar (Logs, Events, Metrics)
- Modal system (overlays for forms)
- Theme system (light/dark)

**Technologies**:
- HTML5 semantic elements
- CSS3 Grid & Flexbox
- CSS Variables for theming
- Responsive design (mobile-first)

---

### 2. Application Layer (app.js)

**Purpose**: Coordinate all components and manage application state

**Responsibilities**:
- Initialize all components
- Manage tab navigation
- Handle theme switching
- Coordinate data loading
- Manage auto-refresh intervals
- Event delegation

**Key Functions**:
- `init()` - Initialize application
- `loadTabData(tab)` - Load data for specific tab
- `loadBlocks()` - Fetch and display blocks
- `loadTransactions()` - Fetch and display transactions
- `loadValidators()` - Fetch and display validators
- `loadProposals()` - Fetch and display proposals
- `loadLiquidityPools()` - Fetch and display pools

---

### 3. Components Layer

#### NetworkSelector (components/NetworkSelector.js)

**Purpose**: Handle network selection and status

**Key Features**:
- Network switching (local/testnet/mainnet)
- Connection status monitoring
- Visual status indicator
- Mainnet warning dialog

**Public Methods**:
- `init()` - Initialize component
- `handleNetworkChange(event)` - Handle network selection
- `checkNetworkStatus()` - Check connection status
- `updateStatus(status, text)` - Update UI status

---

#### QuickActions (components/QuickActions.js)

**Purpose**: Handle all quick action buttons and modals

**Quick Actions**:
1. Send Transaction
2. Create Wallet
3. Delegate Tokens
4. Submit Proposal
5. Swap Tokens
6. Query Balance

**Testing Tools**:
1. Transaction Simulator
2. Bulk Wallet Generator
3. Load Testing
4. Stress Testing
5. Faucet Integration

**Test Scenarios**:
1. Transaction Flow
2. Staking Flow
3. Governance Flow
4. DEX Trading Flow

**Public Methods**:
- `init()` - Initialize all button listeners
- `showSendTransactionModal()` - Display send TX form
- `showCreateWalletModal()` - Display wallet creation
- `runTransactionFlow()` - Execute transaction test scenario
- (+ 20 more methods)

---

#### LogViewer (components/LogViewer.js)

**Purpose**: Display and manage live logs

**Key Features**:
- Color-coded log entries (info/success/warning/error)
- Auto-scroll to latest
- Export logs as JSON
- Clear logs functionality
- Entry limit (100)

**Public Methods**:
- `init()` - Initialize component
- `addLog(log)` - Add new log entry
- `renderLogs()` - Update log display
- `clearLogs()` - Clear all logs
- `exportLogs()` - Export to JSON file

---

#### MetricsDisplay (components/MetricsDisplay.js)

**Purpose**: Display real-time network and system metrics

**Metrics Displayed**:
- Block height
- TPS (Transactions Per Second)
- Peer count
- Consensus status
- CPU usage
- Memory usage
- Disk I/O

**Public Methods**:
- `init()` - Initialize component
- `updateMetrics()` - Fetch latest metrics
- `updateBlockMetrics(block)` - Update block-related metrics
- `updateSystemMetrics(metrics)` - Update system metrics

---

### 4. Services Layer

#### BlockchainService (services/blockchain.js)

**Purpose**: Interface with blockchain via RPC/REST APIs

**Key Responsibilities**:
- Network management
- Data retrieval (blocks, TXs, validators, proposals)
- Transaction building and sending
- Wallet creation
- Balance queries

**Public API** (20+ methods):

**Network Management**:
- `switchNetwork(network)` - Switch to different network
- `checkConnection()` - Test node connectivity
- `getNetworkInfo()` - Get current network details

**Data Retrieval**:
- `getLatestBlock()` - Get latest block
- `getRecentBlocks(count)` - Get recent N blocks
- `getRecentTransactions(limit)` - Get recent transactions
- `getValidators()` - Get validator list
- `getStakingInfo()` - Get staking statistics
- `getProposals()` - Get governance proposals
- `getLiquidityPools()` - Get DEX pools
- `queryBalance(address)` - Get wallet balance

**Actions**:
- `sendTransaction(txData)` - Send transaction
- `createWallet()` - Generate new wallet
- `requestFaucet(address)` - Request test tokens

**Internal**:
- `extractTxType(tx)` - Parse transaction type

---

#### MonitoringService (services/monitoring.js)

**Purpose**: Real-time monitoring and event detection

**Key Features**:
- Block production monitoring
- System metrics collection
- Event detection and notification
- Network health checking
- Transaction confirmation tracking

**Public API**:

**Lifecycle**:
- `startMonitoring()` - Start all monitoring
- `stopMonitoring()` - Stop all monitoring

**Event System**:
- `on(event, callback)` - Register event listener
- `emit(event, data)` - Emit event to listeners

**Monitoring Functions**:
- `startBlockMonitoring()` - Monitor new blocks
- `startMetricsMonitoring()` - Monitor system metrics
- `startEventsMonitoring()` - Monitor blockchain events
- `calculateTPS()` - Calculate transactions per second
- `getPeerCount()` - Get connected peer count
- `getConsensusStatus()` - Get consensus state
- `getNetworkHealth()` - Check overall network health
- `monitorTransaction(txHash)` - Track TX confirmation

**Logging**:
- `addLog(level, message)` - Add log entry

---

#### TestingService (services/testing.js)

**Purpose**: Testing utilities and automated test scenarios

**Key Features**:
- Bulk wallet generation
- Pre-built test scenarios
- Transaction simulation
- Load testing
- Result tracking and export

**Public API**:

**Wallet Management**:
- `generateBulkWallets(count)` - Create multiple wallets
- `getTestWallets()` - Get generated wallets

**Test Scenarios**:
- `runTransactionFlowTest()` - Test transaction flow
- `runStakingFlowTest()` - Test staking flow
- `runGovernanceFlowTest()` - Test governance flow
- `runDEXFlowTest()` - Test DEX trading flow

**Utilities**:
- `simulateTransaction(txData)` - Validate transaction
- `runLoadTest(config)` - Execute load test

**Results**:
- `getTestResults()` - Get all test results
- `clearTestResults()` - Clear results
- `exportResults(format)` - Export as JSON/CSV

---

## Data Flow

### Example: Sending a Transaction

```
1. User clicks "Send Transaction" button
   ↓
2. QuickActions.showSendTransactionModal()
   - Displays modal with form
   - "Use Test Data" button available
   ↓
3. User fills form and clicks "Send Transaction"
   ↓
4. QuickActions.sendTransaction()
   - Validates input
   - Checks network (not mainnet)
   ↓
5. LogViewer.addLog('info', 'Sending transaction...')
   ↓
6. BlockchainService.sendTransaction(txData)
   - Builds transaction
   - Sends to node
   ↓
7. MonitoringService.monitorTransaction(txHash)
   - Polls for confirmation
   ↓
8. On confirmation:
   - MonitoringService emits 'transactionConfirmed'
   - LogViewer.addLog('success', 'Transaction confirmed')
   - MetricsDisplay updates
```

### Example: Real-Time Block Updates

```
1. App starts → MonitoringService.startMonitoring()
   ↓
2. MonitoringService.startBlockMonitoring()
   - Sets interval (every 3 seconds)
   ↓
3. Every 3 seconds:
   BlockchainService.getLatestBlock()
   ↓
4. MonitoringService emits 'blockUpdate' event
   ↓
5. MetricsDisplay receives event
   - Updates block height
   - Updates TPS
   ↓
6. If "Blocks" tab is active:
   - App.loadBlocks() refreshes table
```

---

## Configuration System

### config.js Structure

```javascript
CONFIG = {
    networks: {
        local: { /* endpoints, features */ },
        testnet: { /* endpoints, features */ },
        mainnet: { /* endpoints, features */ }
    },
    updateIntervals: {
        blockUpdates: 3000,
        metricsUpdates: 5000,
        // ...
    },
    testData: {
        wallets: [ /* test wallets */ ],
        transactions: { /* test values */ }
    },
    ui: {
        logsMaxEntries: 100,
        theme: 'light',
        // ...
    },
    api: {
        blocks: '/cosmos/base/tendermint/v1beta1/blocks/latest',
        // ...
    }
}
```

---

## State Management

**Current Approach**: Component-based state with event system

**State Location**:
- `app.js` - Global app state (current tab, refresh intervals)
- `NetworkSelector` - Current network, connection status
- `LogViewer` - Log entries array
- `MetricsDisplay` - Current metrics values
- `BlockchainService` - Network config, connection state
- `MonitoringService` - Monitoring intervals, callbacks
- `TestingService` - Test wallets, test results

**State Persistence**:
- Theme preference → localStorage
- No other state persisted (intentional for testing)

---

## Security Architecture

### Client-Side Security

1. **Input Validation**
   - All form inputs validated
   - Address format checking
   - Amount range validation

2. **XSS Prevention**
   - HTML escaping in LogViewer
   - No `innerHTML` with user input
   - Sanitized modal content

3. **Wallet Security**
   - Client-side generation only
   - No private key storage
   - Clear warnings about backups

4. **Network Protection**
   - Mainnet read-only enforcement
   - Confirmation dialogs for switching
   - Transaction validation

### Server-Side Security (nginx)

1. **Headers**
   - CORS configuration
   - X-Frame-Options
   - X-Content-Type-Options
   - X-XSS-Protection
   - Referrer-Policy

2. **Performance**
   - Gzip compression
   - Static asset caching
   - Request optimization

---

## Deployment Architecture

### Docker Deployment

```
┌─────────────────────────────────────────┐
│         Docker Host                      │
│                                          │
│  ┌────────────────────────────────────┐ │
│  │  Dashboard Container (Nginx)       │ │
│  │  Port: 8080                        │ │
│  │  Serves: /usr/share/nginx/html     │ │
│  └────────────────────────────────────┘ │
│                                          │
│  ┌────────────────────────────────────┐ │
│  │  PAW Node Container (Optional)     │ │
│  │  Ports: 26657, 1317, 9090          │ │
│  └────────────────────────────────────┘ │
│                                          │
│  ┌────────────────────────────────────┐ │
│  │  Faucet Container (Optional)       │ │
│  │  Port: 8000                        │ │
│  └────────────────────────────────────┘ │
│                                          │
│         paw-testing-network             │
└─────────────────────────────────────────┘
```

---

## Performance Considerations

### Optimizations

1. **Lazy Loading**: Components initialize only when needed
2. **Interval Management**: Clear intervals when switching tabs
3. **Data Limits**: Max entries for logs (100) and events (50)
4. **Debouncing**: Network status checks on intervals
5. **Caching**: Static assets cached by nginx

### Bottlenecks

1. **API Calls**: Rate limited by blockchain node
2. **Polling**: Using intervals instead of WebSocket
3. **DOM Updates**: Frequent table re-renders

### Future Optimizations

1. WebSocket for real-time updates
2. Virtual scrolling for large lists
3. Service workers for caching
4. Indexed DB for historical data

---

## Error Handling

### Error Flow

```
Error Occurs
   ↓
Try-Catch Block
   ↓
Log to Console (for debugging)
   ↓
LogViewer.addLog('error', message)
   ↓
User sees error in logs
   ↓
User can export logs for support
```

### Error Types Handled

1. **Network Errors**: Connection failures, timeouts
2. **Validation Errors**: Invalid addresses, amounts
3. **API Errors**: Failed requests, malformed responses
4. **State Errors**: Invalid state transitions

---

## Testing Architecture

### Test Structure

```
tests/
├── dashboard.test.js     # UI & navigation tests
├── actions.test.js       # Quick action tests
└── monitoring.test.js    # Monitoring service tests
```

### Test Coverage

- **Unit Tests**: Individual function testing
- **Integration Tests**: Component interaction
- **User Scenario Tests**: End-to-end workflows

### Testing Tools

- Jest for test runner
- JSDOM for browser environment
- Mock API responses

---

## Extension Points

### Easy to Extend

1. **Add New Network**:
   - Add to `config.js` networks
   - Automatically supported

2. **Add New Quick Action**:
   - Add button to `index.html`
   - Add handler to `QuickActions.js`

3. **Add New Tab**:
   - Add tab button to `index.html`
   - Add pane to tab content
   - Add loader to `app.js`

4. **Add New Service**:
   - Create in `services/`
   - Import in `app.js`
   - Initialize in `app.init()`

---

## File Organization

```
testing-dashboard/
├── index.html                    # UI entry point
├── styles.css                    # All styles
├── app.js                        # Main coordinator
├── config.js                     # Configuration
│
├── services/                     # Business logic
│   ├── blockchain.js             # Blockchain API
│   ├── monitoring.js             # Monitoring
│   └── testing.js                # Testing utils
│
├── components/                   # UI components
│   ├── NetworkSelector.js        # Network switching
│   ├── QuickActions.js           # Actions & modals
│   ├── LogViewer.js              # Log display
│   └── MetricsDisplay.js         # Metrics display
│
├── tests/                        # Test suites
│   ├── dashboard.test.js
│   ├── actions.test.js
│   └── monitoring.test.js
│
├── docker-compose.yml            # Docker setup
├── nginx.conf                    # Nginx config
├── package.json                  # NPM config
│
├── README.md                     # Documentation
├── USER_GUIDE.md                 # User manual
├── TESTING_SUMMARY.md            # Test results
├── QUICK_START.md                # Quick reference
└── ARCHITECTURE.md               # This file
```

---

## Conclusion

The PAW Testing Dashboard uses a clean, modular architecture that separates concerns effectively:

- **UI Layer**: Pure HTML/CSS for presentation
- **Application Layer**: Coordination and state management
- **Component Layer**: Reusable UI components
- **Service Layer**: Business logic and API interaction

This architecture enables:
- Easy maintenance and updates
- Clear separation of concerns
- Testability at every layer
- Simple extension for new features
- No build step required
- Minimal dependencies

The result is a production-ready, performant, and maintainable testing dashboard.
