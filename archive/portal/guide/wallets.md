# Creating and Managing Wallets

Learn how to create, secure, and manage your PAW wallet across different platforms.

## Wallet Options

PAW offers multiple wallet options to suit your needs:

| Wallet Type | Platform | Best For | Security Level |
|-------------|----------|----------|----------------|
| Desktop Wallet | Windows, macOS, Linux | Daily use, trading | High |
| Mobile Wallet | iOS, Android | On-the-go transactions | High |
| Web Wallet | Browser | Quick access | Medium |
| CLI Wallet | Command line | Developers, advanced users | Highest |
| Hardware Wallet | Ledger, Trezor | Long-term storage | Highest |

## Desktop Wallet

The PAW Desktop Wallet provides a full-featured experience with advanced security.

### Installation

::: code-group

```bash [Windows]
# Download from  Releases
<REPO_URL>/releases/latest/download/PAW-Desktop-Setup.exe

# Run the installer
# Follow the installation wizard
```

```bash [macOS]
# Download the DMG file
<REPO_URL>/releases/latest/download/PAW-Desktop.dmg

# Open the DMG and drag to Applications
```

```bash [Linux]
# Download AppImage
wget <REPO_URL>/releases/latest/download/PAW-Desktop.AppImage

# Make it executable
chmod +x PAW-Desktop.AppImage

# Run the wallet
./PAW-Desktop.AppImage
```

:::

### Creating a Wallet

1. **Launch the desktop wallet**
2. **Select "Create New Wallet"**
3. **Set a strong password** (min. 12 characters)
4. **Write down your 24-word recovery phrase**
5. **Verify the recovery phrase**
6. **Complete setup**

::: danger Security Warning
Your 24-word recovery phrase is the master key to your wallet. Anyone with this phrase can access your funds. Write it down on paper and store it in a secure location. NEVER store it digitally or share it with anyone.
:::

### Features

- ✅ Send and receive PAW tokens
- ✅ Transaction history with detailed views
- ✅ Address book for frequent contacts
- ✅ DEX integration for token swaps
- ✅ Staking directly from wallet
- ✅ Governance voting
- ✅ Multi-account support
- ✅ Password-protected encryption
- ✅ Automatic updates

### Desktop Wallet Security

The desktop wallet includes multiple security features:

- **Encrypted Storage**: All keys encrypted with AES-256
- **Password Protection**: Required for all sensitive operations
- **Auto-lock**: Wallet locks after inactivity
- **Secure Key Generation**: BIP39/BIP44 compliant
- **Transaction Signing**: All transactions require password

## Mobile Wallet

Available for iOS and Android devices.

### Download

- **iOS**: [App Store](https://apps.apple.com/app/paw-wallet)
- **Android**: [Google Play](https://play.google.com/store/apps/details?id=io.pawblockchain.wallet)

### Mobile-Specific Features

- 📱 Biometric authentication (Face ID, Touch ID, Fingerprint)
- 📷 QR code scanning for addresses
- 📲 Push notifications for transactions
- 🔔 Price alerts
- 💳 Contact-based payments
- 🌐 Multi-language support

### Setting Up Mobile Wallet

1. **Download the app** from App Store or Google Play
2. **Open the app** and select your language
3. **Choose setup method**:
   - Create new wallet
   - Import existing wallet
   - Connect hardware wallet

4. **Enable biometric authentication** (recommended)
5. **Set up backup** to cloud (encrypted)
6. **Complete setup**

### Backing Up Mobile Wallet

::: tip Backup Options
- **iCloud Keychain** (iOS) - Encrypted backup
- **Google Drive** (Android) - Encrypted backup
- **Manual backup** - Write down recovery phrase
:::

```bash
Settings → Backup → Enable Cloud Backup
# Set a strong backup password
```

## Web Wallet

Access your wallet from any browser.

### Using Web Wallet

Visit [wallet.paw.network](https://wallet.paw.network)

**Important**: Always verify the URL is correct before entering any sensitive information.

### Features

- Quick access from any device
- No installation required
- Full send/receive functionality
- DEX integration
- Staking support

### Security Considerations

::: warning Web Wallet Security
- Only use on trusted devices
- Clear browser cache after use
- Never save credentials in browser
- Use hardware wallet for large amounts
- Enable 2FA if available
:::

## CLI Wallet (Command Line)

For developers and advanced users.

### Create Wallet

```bash
# Create new wallet
pawd keys add my-wallet

# Create with specific derivation path
pawd keys add my-wallet --account 0 --index 0

# Create with custom key algorithm
pawd keys add my-wallet --algo secp256k1
```

### Import Wallet

```bash
# Import from mnemonic
pawd keys add my-wallet --recover

# Import from private key
pawd keys import my-wallet private_key.txt
```

### Export Keys

```bash
# Export private key (use with extreme caution)
pawd keys export my-wallet

# Export to file
pawd keys export my-wallet > my-wallet-key.txt

# Protect the exported file
chmod 400 my-wallet-key.txt
```

### List Wallets

```bash
# Show all wallets
pawd keys list

# Show specific wallet details
pawd keys show my-wallet

# Show only address
pawd keys show my-wallet --address

# Show public key
pawd keys show my-wallet --pubkey
```

## Hardware Wallet Integration

### Ledger Support

PAW supports Ledger Nano S and Nano X devices.

#### Setup Instructions

1. **Install Cosmos app** on your Ledger device
2. **Connect Ledger** to computer via USB
3. **Open Cosmos app** on device

```bash
# Create wallet using Ledger
pawd keys add my-ledger --ledger

# Use wallet for transactions
pawd tx bank send my-ledger paw1xxx... 1000000upaw --ledger
```

#### Benefits

- ✅ Private keys never leave device
- ✅ Transaction signing on device
- ✅ PIN protection
- ✅ Recovery phrase backup
- ✅ Physical confirmation required

### Trezor Support

Support for Trezor Model T and Model One.

```bash
# Create wallet using Trezor
pawd keys add my-trezor --trezor

# Use for transactions
pawd tx bank send my-trezor paw1xxx... 1000000upaw --trezor
```

## Wallet Security Best Practices

### Recovery Phrase Security

::: danger Critical Security
Your 24-word recovery phrase is the master key to your wallet:

**DO:**
- ✅ Write it on paper with a pen
- ✅ Store in multiple secure locations
- ✅ Use a fireproof/waterproof safe
- ✅ Consider metal backup plates
- ✅ Verify spelling and order

**DON'T:**
- ❌ Store digitally (photos, files, cloud)
- ❌ Share with anyone, ever
- ❌ Type into any website or app
- ❌ Store in email or messaging apps
- ❌ Take screenshots
:::

### Password Best Practices

```bash
# Good password examples:
- Tr0ub4dor&3Kd8!mNp (16+ chars, mixed case, numbers, symbols)
- correct-horse-battery-staple-9876 (passphrase style)

# Bad passwords:
- password123 (too simple)
- PAW2024 (too short, predictable)
- mywalletpass (dictionary word)
```

### Multi-Signature Wallets

For enhanced security, create multi-signature wallets:

```bash
# Create 2-of-3 multisig wallet
pawd keys add multisig-wallet \
  --multisig key1,key2,key3 \
  --multisig-threshold 2

# Sign transactions
pawd tx bank send multisig-wallet paw1xxx... 1000000upaw \
  --generate-only > tx.json

pawd tx sign tx.json --from key1 > signed1.json
pawd tx sign tx.json --from key2 > signed2.json

pawd tx multisign tx.json multisig-wallet signed1.json signed2.json \
  > signed-tx.json

pawd tx broadcast signed-tx.json
```

## Managing Multiple Wallets

### Organizing Wallets

Create different wallets for different purposes:

```bash
# Personal wallet
pawd keys add personal-wallet

# Trading wallet
pawd keys add trading-wallet

# Savings wallet
pawd keys add savings-wallet

# Staking wallet
pawd keys add staking-wallet
```

### Wallet Labels and Notes

Most wallet apps allow you to add labels:

- **Main Account** - Daily transactions
- **Trading** - DEX operations
- **Cold Storage** - Long-term holdings
- **Validator** - Validator operations

## Wallet Recovery

### Recovering from Mnemonic

```bash
# Recover wallet with mnemonic phrase
pawd keys add recovered-wallet --recover

# Then enter your 24-word phrase
```

### Recovering from Private Key

```bash
# Import private key
pawd keys import recovered-wallet private_key.txt
```

### Recovering Desktop Wallet

1. **Reinstall wallet application**
2. **Select "Recover Wallet"**
3. **Enter 24-word recovery phrase**
4. **Set new password**
5. **Wait for sync to complete**

## Common Wallet Operations

### Viewing Balance

```bash
# CLI
pawd query bank balances paw1xxxxxxxxxxxxx

# REST API
curl http://localhost:1317/cosmos/bank/v1beta1/balances/paw1xxxxxxxxxxxxx
```

### Sending Tokens

```bash
# Basic send
pawd tx bank send my-wallet paw1yyy... 1000000upaw \
  --fees 500upaw

# With memo
pawd tx bank send my-wallet paw1yyy... 1000000upaw \
  --fees 500upaw \
  --memo "Payment for services"
```

### Transaction History

```bash
# Query transactions by sender
pawd query txs --events "message.sender=paw1xxx..."

# Query transactions by recipient
pawd query txs --events "transfer.recipient=paw1xxx..."

# Query specific transaction
pawd query tx <TX_HASH>
```

## Troubleshooting

### Forgot Password

::: warning
Desktop wallet passwords cannot be recovered. You must restore from your 24-word recovery phrase.
:::

### Lost Recovery Phrase

::: danger
Without your recovery phrase, funds cannot be recovered if you lose access to your wallet. There is no password reset or recovery process.
:::

### Wallet Not Syncing

```bash
# Check connection
pawd status

# Reset and resync
pawd unsafe-reset-all
# Re-download genesis and restart
```

### Transaction Failed

Common reasons:

- Insufficient balance for fees
- Incorrect recipient address format
- Node not fully synced
- Invalid transaction parameters

## Video Tutorials

### Desktop Wallet Setup

<div class="video-container">
  <iframe
    src="https://www.youtube.com/embed/DESKTOP_WALLET_VIDEO_ID"
    frameborder="0"
    allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
    allowfullscreen>
  </iframe>
</div>

### Mobile Wallet Guide

<div class="video-container">
  <iframe
    src="https://www.youtube.com/embed/MOBILE_WALLET_VIDEO_ID"
    frameborder="0"
    allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
    allowfullscreen>
  </iframe>
</div>

## Next Steps

- **[Using the DEX](/guide/dex)** - Trade tokens on the decentralized exchange
- **[Staking Guide](/guide/staking)** - Earn rewards by staking PAW
- **[Governance](/guide/governance)** - Participate in network governance

---

**Previous:** [Getting Started](/guide/getting-started) | **Next:** [Using the DEX](/guide/dex) →
