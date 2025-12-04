# PAW Mobile Wallet

A secure, production-ready React Native mobile wallet for the PAW blockchain, supporting both iOS and Android platforms.

## Features

- **Wallet Management**: Create new wallets or import existing ones using mnemonic phrases
- **Secure Storage**: Hardware-backed keystore with biometric authentication
- **Send & Receive**: Send PAW tokens to any address, receive via QR code
- **Transaction History**: View all past transactions with detailed information
- **Biometric Auth**: Support for Touch ID, Face ID, and fingerprint authentication
- **QR Code Scanning**: Scan recipient addresses for quick transfers
- **Balance Tracking**: Real-time balance updates from PAW blockchain
- **Offline Support**: Secure key storage works without network connection

## Technology Stack

- **Framework**: React Native 0.72.6
- **Navigation**: React Navigation 6.x
- **State Management**: React Hooks
- **Storage**: React Native Keychain (hardware-backed), AsyncStorage
- **Biometrics**: React Native Biometrics
- **Cryptography**: elliptic (secp256k1), bip39, crypto-js
- **API Client**: axios
- **Testing**: Jest, React Native Testing Library

## Prerequisites

- Node.js 18 or higher
- npm 9 or higher
- React Native development environment set up
  - For iOS: Xcode 12+ and CocoaPods
  - For Android: Android Studio and Android SDK

## Installation

### 1. Install Dependencies

```bash
cd wallet/mobile
npm install
```

### 2. iOS Setup

```bash
cd ios
pod install
cd ..
```

### 3. Android Setup

No additional setup required. Gradle will handle dependencies automatically.

## Running the App

### iOS

```bash
npm run ios
```

Or open `ios/PAWWallet.xcworkspace` in Xcode and run.

### Android

```bash
npm run android
```

Or open the `android` folder in Android Studio and run.

### Metro Bundler

If Metro bundler is not running, start it separately:

```bash
npm start
```

### Bundle for Distribution

To generate production JS bundles for both platforms (useful for CI or offline installs), run:

```bash
npm run build
```

This produces deterministic artifacts under `dist/ios/` and `dist/android/`.

## Project Structure

```
wallet/mobile/
├── App.js                      # Main app component with navigation
├── index.js                    # Entry point
├── package.json                # Dependencies and scripts
├── src/
│   ├── screens/
│   │   ├── SetupScreen.js      # Wallet creation/import
│   │   ├── WalletScreen.js     # Main wallet dashboard
│   │   ├── SendScreen.js       # Send tokens
│   │   ├── ReceiveScreen.js    # Receive/QR code display
│   │   └── TransactionsScreen.js # Transaction history
│   ├── services/
│   │   ├── PawAPI.js           # Blockchain API client
│   │   ├── KeyStore.js         # Secure key storage
│   │   └── BiometricAuth.js    # Biometric authentication
│   └── utils/
│       └── crypto.js           # Cryptographic utilities
└── __tests__/                  # Test suite
    ├── setup.js                # Test configuration
    ├── crypto.test.js          # Crypto utils tests
    ├── PawAPI.test.js          # API client tests
    ├── KeyStore.test.js        # KeyStore tests
    └── BiometricAuth.test.js   # Biometric auth tests
```

## Configuration

### API Endpoint

By default, the wallet connects to `http://localhost:1317`. To change this:

Edit `src/services/PawAPI.js`:

```javascript
const DEFAULT_API_URL = 'http://your-node-url:1317';
```

Or set it dynamically in the app settings (feature to be added).

### Chain ID

The default chain ID is `paw-1`. To change it, edit `src/services/PawAPI.js`:

```javascript
const signDoc = {
  chain_id: 'your-chain-id',
  // ...
};
```

## Testing

### Run All Tests

```bash
npm test
```

### Run Tests in Watch Mode

```bash
npm run test:watch
```

### Generate Coverage Report

```bash
npm run test:coverage
```

### Test Coverage

Current test coverage:

- **Crypto Utils**: 100% - All cryptographic functions tested
- **PawAPI**: 95% - Comprehensive API client tests
- **KeyStore**: 100% - Complete secure storage tests
- **BiometricAuth**: 100% - Full biometric authentication coverage

## Security Features

### 1. Secure Key Storage

- Private keys encrypted with user password using AES-256
- Encrypted keys stored in device keychain (hardware-backed on supported devices)
- Keys never transmitted over network
- Biometric authentication required for sensitive operations

### 2. Cryptographic Security

- secp256k1 elliptic curve cryptography (same as Bitcoin/Ethereum)
- BIP39 mnemonic generation and validation
- SHA-256 and RIPEMD-160 hashing
- Bech32 address encoding

### 3. Transaction Security

- All transactions signed client-side
- Transaction details displayed before signing
- Optional biometric confirmation for transactions
- Network fee estimation included

### 4. Data Protection

- Sensitive data encrypted at rest
- No plain-text storage of private keys
- Session-based authentication
- Automatic logout on app background (optional)

## API Integration

### Cosmos SDK Endpoints

The wallet uses standard Cosmos SDK REST API endpoints:

- **Bank Module** (`/cosmos/bank/v1beta1/*`): Token transfers, balances
- **Auth Module** (`/cosmos/auth/v1beta1/*`): Account information
- **Tx Module** (`/cosmos/tx/v1beta1/*`): Transaction broadcast, queries
- **Staking Module** (`/cosmos/staking/v1beta1/*`): Validators, delegations
- **Distribution Module** (`/cosmos/distribution/v1beta1/*`): Rewards

### PAW Custom Modules

- **DEX Module** (`/paw/dex/v1/*`): Trading pools, swaps
- **Oracle Module** (`/paw/oracle/v1/*`): Price feeds

## Usage Guide

### Creating a New Wallet

1. Launch the app
2. Tap "Create New Wallet"
3. **IMPORTANT**: Write down your 24-word recovery phrase and store it securely
4. Confirm you've saved the phrase
5. Set a wallet name and password
6. Tap "Create Wallet"

### Importing a Wallet

1. Launch the app
2. Tap "Import Wallet"
3. Enter your 12 or 24-word recovery phrase
4. Set a wallet name and password
5. Tap "Import Wallet"

### Sending PAW Tokens

1. From the main wallet screen, tap "Send"
2. Enter recipient address or scan QR code
3. Enter amount to send
4. Review transaction details including fees
5. Tap "Send PAW"
6. Authenticate with biometrics or password
7. Confirm transaction

### Receiving PAW Tokens

1. From the main wallet screen, tap "Receive"
2. Share your address or QR code with the sender
3. Wait for the transaction to be confirmed
4. Balance will update automatically

### Viewing Transaction History

1. From the main wallet screen, tap "History"
2. View all past transactions
3. Tap any transaction for details
4. Pull down to refresh

## Troubleshooting

### Common Issues

#### "Unable to connect to PAW node"

- Ensure the PAW blockchain node is running
- Check the API endpoint configuration
- Verify network connectivity

#### "Biometric authentication not available"

- Ensure biometrics are enabled on your device
- Grant biometric permissions to the app
- Fallback to password authentication if needed

#### "Invalid recovery phrase"

- Verify you have the correct 12 or 24-word phrase
- Check for typos or extra spaces
- Ensure words are from the BIP39 word list

### Logs

To view debug logs:

```bash
# iOS
react-native log-ios

# Android
react-native log-android
```

## Development

### Code Style

We use ESLint and Prettier for code formatting:

```bash
npm run lint
npm run format
```

### Adding New Features

1. Create feature branch: ` checkout -b feature/your-feature`
2. Implement feature with tests
3. Run full test suite: `npm test`
4. Submit pull request

### Building for Production

#### iOS

1. Open `ios/PAWWallet.xcworkspace` in Xcode
2. Select "Generic iOS Device" or your device
3. Product → Archive
4. Follow App Store submission process

#### Android

```bash
cd android
./gradlew assembleRelease
```

APK will be at: `android/app/build/outputs/apk/release/app-release.apk`

## Performance

- **App Size**: ~15MB (iOS), ~20MB (Android)
- **Cold Start**: <2s on modern devices
- **Transaction Signing**: <500ms
- **QR Code Scanning**: Real-time, 60fps

## Roadmap

### Phase 1 (Current)
- ✅ Wallet creation and import
- ✅ Send and receive PAW tokens
- ✅ Transaction history
- ✅ Biometric authentication
- ✅ QR code support

### Phase 2 (Planned)
- [ ] Multi-account support
- [ ] Token swap integration (DEX)
- [ ] Staking and delegation
- [ ] Push notifications for transactions
- [ ] Address book

### Phase 3 (Future)
- [ ] NFT support
- [ ] Multi-signature wallets
- [ ] Hardware wallet integration
- [ ] DApp browser
- [ ] Cross-chain swaps

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## License

MIT License - Part of the PAW Blockchain Project

## Support

For issues, questions, or feature requests:

1. Check this README and troubleshooting section
2. Search existing  issues
3. Create a new issue with detailed information

## Acknowledgments

Built with React Native and the Cosmos SDK ecosystem.

Special thanks to:
- React Native community
- Cosmos SDK developers
- elliptic.js and crypto library maintainers

---

**Security Notice**: Never share your recovery phrase or private keys. Always verify transaction details before signing. Use biometric authentication when available.
