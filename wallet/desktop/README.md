# PAW Desktop Wallet

A secure, cross-platform desktop wallet for the PAW blockchain, built with Electron and React.

## Features

- **Multi-Platform Support**: Windows, macOS, and Linux
- **Secure Key Storage**: Encrypted mnemonic storage using electron-store
- **Full Wallet Management**: Create, import, and backup wallets
- **Token Operations**: Send and receive PAW tokens
- **Transaction History**: View complete transaction history
- **Address Book**: Save frequently used addresses
- **Auto-Updates**: Automatic application updates
- **Modern UI**: Clean, dark-themed interface
- **Network Configuration**: Configurable API endpoints

## Screenshots

### Wallet Overview
View your balance and wallet information at a glance.

### Send Tokens
Simple and secure token transfer interface with transaction preview.

### Receive Tokens
Display your address with QR code for easy sharing.

### Transaction History
Track all your transactions with detailed information.

## Installation

### Prerequisites

- Node.js 18 or higher
- npm 7 or higher
- PAW blockchain node (for live network)

### Development Setup

1. **Clone the repository**
   ```bash
   cd paw/wallet/desktop
   ```

2. **Install dependencies**
   ```bash
   npm install
   ```

3. **Start development server**
   ```bash
   npm run dev
   ```

   This will start both the React development server and Electron application.

### Production Build

Build for your platform:

```bash
# Build for current platform
npm run build

# Build for Windows
npm run build:win

# Build for macOS
npm run build:mac

# Build for Linux
npm run build:linux
```

Built applications will be in the `dist/` directory.

## Usage

### First Launch

1. **Create New Wallet**
   - Click "Create New Wallet"
   - Enter a strong password (min 8 characters)
   - Write down your 24-word mnemonic phrase
   - Confirm you've saved your mnemonic
   - Your wallet is ready!

2. **Import Existing Wallet**
   - Click "Import Wallet"
   - Enter your 24-word mnemonic phrase
   - Set a password
   - Your wallet is imported!

### Sending Tokens

1. Navigate to **Send** tab
2. Enter recipient address (must start with `paw`)
3. Enter amount in PAW
4. Add optional memo
5. Enter your password
6. Preview and confirm transaction

### Receiving Tokens

1. Navigate to **Receive** tab
2. Copy your address or scan QR code
3. Share with sender

### Managing Addresses

1. Navigate to **Address Book**
2. Click "Add Address"
3. Enter name, address, and optional note
4. Save for quick access

### Settings

- **Network Settings**: Configure API and WebSocket endpoints
- **Backup**: View and backup your mnemonic phrase
- **Reset Wallet**: Remove current wallet (requires confirmation)

## Security

### Best Practices

- **Mnemonic Phrase**: Write down your 24-word phrase and store it securely offline
- **Password**: Use a strong, unique password for your wallet
- **Backup**: Keep multiple copies of your mnemonic in different secure locations
- **Verification**: Always verify recipient addresses before sending
- **Updates**: Keep your wallet updated to the latest version

### Security Features

- **Encrypted Storage**: Mnemonic encrypted with password-based encryption
- **Context Isolation**: Electron security best practices enforced
- **No Remote Code**: All code runs locally, no external scripts
- **Secure Communication**: HTTPS/WSS for network communication
- **Sandboxing**: Renderer process runs in sandbox mode

## Configuration

### Network Endpoints

Default endpoints (can be changed in Settings):

- **API Endpoint**: `http://localhost:1317` (Cosmos REST)
- **WebSocket**: `ws://localhost:26657` (Tendermint RPC)

For testnet or mainnet, update these in Settings.

### Custom Configuration

Edit endpoints in the Settings panel or use environment variables:

```bash
# Development
VITE_API_URL=http://localhost:1317
VITE_WS_URL=ws://localhost:26657

# Production
VITE_API_URL=https://api.paw.network
VITE_WS_URL=wss://rpc.paw.network
```

## Testing

### Run All Tests

```bash
npm test
```

### Test Suites

```bash
# Unit tests
npm run test:unit

# Integration tests
npm run test:integration

# E2E tests
npm run test:e2e

# Watch mode
npm run test:watch
```

### Test Coverage

Generate coverage report:

```bash
npm test -- --coverage
```

Coverage report will be in `coverage/` directory.

## Development

### Project Structure

```
wallet/desktop/
├── src/
│   ├── components/          # React components
│   │   ├── Setup.jsx       # Wallet creation/import
│   │   ├── Wallet.jsx      # Main wallet view
│   │   ├── Send.jsx        # Send tokens
│   │   ├── Receive.jsx     # Receive tokens
│   │   ├── History.jsx     # Transaction history
│   │   ├── AddressBook.jsx # Address management
│   │   └── Settings.jsx    # Settings panel
│   ├── services/           # Core services
│   │   ├── api.js         # PAW API client
│   │   └── keystore.js    # Wallet management
│   ├── App.jsx            # Main app component
│   ├── index.jsx          # React entry point
│   └── index.css          # Global styles
├── test/                   # Test files
│   ├── main.test.js       # Main process tests
│   ├── api.test.js        # API tests
│   ├── keystore.test.js   # Keystore tests
│   ├── components.test.js # Component tests
│   ├── integration/       # Integration tests
│   └── e2e/              # End-to-end tests
├── main.js                # Electron main process
├── preload.js            # Preload script
├── package.json          # Dependencies
└── README.md            # This file
```

### Key Technologies

- **Electron**: Cross-platform desktop framework
- **React**: UI library
- **Vite**: Build tool and dev server
- **CosmJS**: Cosmos SDK JavaScript library
- **electron-store**: Secure encrypted storage
- **electron-updater**: Auto-update functionality
- **Jest**: Testing framework

### Adding Features

1. Create component in `src/components/`
2. Add route in `src/App.jsx`
3. Write tests in `test/`
4. Update this README

### Code Style

```bash
# Lint code
npm run lint

# Auto-fix issues
npm run lint -- --fix
```

## Building and Distribution

### Building Installers

The build system uses `electron-builder` to create installers:

**Windows**:
- NSIS installer (.exe)
- Portable executable

**macOS**:
- DMG disk image
- ZIP archive

**Linux**:
- AppImage (universal)
- DEB package (Debian/Ubuntu)
- RPM package (Fedora/RHEL)

### Code Signing

For production releases, configure code signing:

**macOS**:
```json
{
  "mac": {
    "identity": "Developer ID Application: Your Name",
    "hardenedRuntime": true
  }
}
```

**Windows**:
```json
{
  "win": {
    "certificateFile": "cert.pfx",
    "certificatePassword": "password"
  }
}
```

### Publishing Updates

Configure  releases in `package.json`:

```json
{
  "publish": {
    "owner": "pawchain",
    "repo": "paw"
  }
}
```

Then publish:

```bash
npm run build
# Upload to  releases
```

## Troubleshooting

### Common Issues

**Issue**: "Cannot connect to PAW node"
- **Solution**: Verify your PAW node is running and accessible
- Check API endpoint in Settings
- Ensure firewall allows connection

**Issue**: "Invalid mnemonic phrase"
- **Solution**: Verify all 24 words are correct and in order
- Check for typos and extra spaces
- Ensure using BIP39 wordlist

**Issue**: "Transaction failed"
- **Solution**: Check account has sufficient balance
- Verify recipient address is valid
- Ensure node is synced

**Issue**: "Password incorrect"
- **Solution**: Re-enter your password carefully
- Try resetting wallet if password is lost (requires mnemonic)

### Debug Mode

Enable debug mode:

```bash
# Start with DevTools
npm run dev

# Check logs
# macOS: ~/Library/Application Support/PAW Wallet/logs
# Windows: %APPDATA%/PAW Wallet/logs
# Linux: ~/.config/PAW Wallet/logs
```

### Reset Wallet

If you need to start fresh:

1. Settings → Reset Wallet
2. Or manually delete:
   - macOS: `~/Library/Application Support/paw-wallet-config.json`
   - Windows: `%APPDATA%/paw-wallet-config.json`
   - Linux: `~/.config/paw-wallet-config.json`

## API Reference

### Keystore Service

```javascript
import { KeystoreService } from './services/keystore';

const keystore = new KeystoreService();

// Generate mnemonic
const mnemonic = await keystore.generateMnemonic();

// Create wallet
const wallet = await keystore.createWallet(mnemonic, password);

// Unlock wallet
const unlocked = await keystore.unlockWallet(password);
```

### API Service

```javascript
import { ApiService } from './services/api';

const api = new ApiService();

// Get balance
const balance = await api.getBalance(address);

// Send tokens
const result = await api.sendTokens(from, to, amount, denom, memo, privateKey);

// Get transactions
const txs = await api.getTransactions(address);
```

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

### Development Workflow

1. Fork the repository
2. Create feature branch: ` checkout -b feature/my-feature`
3. Make changes and add tests
4. Run tests: `npm test`
5. Commit: ` commit -m 'Add my feature'`
6. Push: ` push origin feature/my-feature`
7. Create Pull Request

## Roadmap

### Version 1.1 (Q1 2025)
- [ ] Hardware wallet support (Ledger, Trezor)
- [ ] Multi-account management
- [ ] Staking interface
- [ ] Governance voting

### Version 1.2 (Q2 2025)
- [ ] DEX trading interface
- [ ] Token swap functionality
- [ ] Portfolio analytics
- [ ] Price charts

### Version 2.0 (Q3 2025)
- [ ] Multi-signature support
- [ ] NFT management
- [ ] DApp integration
- [ ] Advanced security features

## Support


## License

MIT License - see [LICENSE](../../LICENSE) for details

## Acknowledgments

- Built with [Electron](https://www.electronjs.org/)
- UI powered by [React](https://react.dev/)
- Cosmos integration via [CosmJS](https://github.com/cosmos/cosmjs)
- Icons from [Heroicons](https://heroicons.com/)

---

**Made with ❤️ by the PAW Blockchain Team**

**Version**: 1.0.0
**Last Updated**: 2025-11-19
