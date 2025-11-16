# BIP39 Quick Start Guide

## Installation

The BIP39 mnemonic support is built into the `pawd` binary. No additional installation needed.

## Quick Command Reference

### Create a New Wallet (24 words - recommended)

```bash
pawd keys add mywallet
```

**Output:**

```
- name: mywallet
  type: secp256k1
  address: paw1xyze3h7r4mt0sjk9tjfg45y6hk0r2zns3nvq8a
  pubkey: pawpub1addwnpepqtq...

**IMPORTANT** Write this mnemonic phrase in a safe place.
It is the only way to recover your account if you ever forget your password.

abandon ability able about above absent absorb abstract absurd abuse access accident account accuse achieve acid acoustic acquire across act action actor actress actual
```

### Create a Wallet with 12 Words

```bash
pawd keys add mywallet --mnemonic-length 12
```

### Recover a Wallet from Mnemonic

```bash
pawd keys recover mywallet
```

**Prompt:**

```
Enter your bip39 mnemonic
> abandon ability able about above absent absorb abstract absurd abuse access accident
```

### List All Wallets

```bash
pawd keys list
```

**Output:**

```
- name: mywallet
  address: paw1xyze3h7r4mt0sjk9tjfg45y6hk0r2zns3nvq8a

- name: validator
  address: paw1abc123def456ghi789jkl012mno345pqr678st
```

### Show Wallet Details

```bash
pawd keys show mywallet
```

### Backup a Wallet (Encrypted Export)

```bash
pawd keys export mywallet > mywallet_backup.txt
```

**Prompt:**

```
Enter passphrase to encrypt the exported key:
> *********
```

### Restore from Backup

```bash
pawd keys import mywallet mywallet_backup.txt
```

**Prompt:**

```
Enter passphrase to decrypt the key:
> *********
```

### Delete a Wallet

```bash
pawd keys delete mywallet
```

**Prompt:**

```
Are you sure you want to delete key 'mywallet'? [y/N]
> y
Key 'mywallet' deleted successfully.
```

## Security Best Practices

### DO:

1. Write your mnemonic on paper (not digital)
2. Store in a secure location (safe, safety deposit box)
3. Make multiple backups in different locations
4. Test recovery before using with real funds
5. Use 24-word mnemonics for maximum security

### DON'T:

1. Take screenshots of your mnemonic
2. Store in cloud storage or email
3. Share with anyone
4. Use example mnemonics from documentation
5. Skip the backup step

## Common Workflows

### Setting Up a New Validator

```bash
# 1. Create validator key
pawd keys add validator --mnemonic-length 24

# 2. Write down the mnemonic (shown once!)

# 3. Verify the key was created
pawd keys show validator

# 4. Export encrypted backup
pawd keys export validator > validator_backup.txt
```

### Recovering on a New Machine

```bash
# Option 1: From Mnemonic (most common)
pawd keys recover validator
# Enter your 24-word mnemonic

# Option 2: From Encrypted Backup
pawd keys import validator validator_backup.txt
# Enter your encryption passphrase
```

### Multiple Accounts from Same Mnemonic

```bash
# Account 0, Index 0 (default)
pawd keys add account0

# Account 0, Index 1
pawd keys add account1 --index 1

# Account 1, Index 0
pawd keys add account2 --account 1 --index 0
```

## Advanced Options

### Custom HD Derivation Path

```bash
pawd keys add customkey --account 5 --index 10
```

This creates a key at path: `m/44'/118'/5'/0/10`

### Different Keyring Backends

```bash
# OS keyring (most secure, default)
pawd keys add mykey --keyring-backend os

# File-based (encrypted file)
pawd keys add mykey --keyring-backend file

# Test (memory only, for testing)
pawd keys add mykey --keyring-backend test
```

## Troubleshooting

### "Invalid mnemonic: checksum failed"

- Check for typos in your mnemonic
- Ensure all words are from the BIP39 wordlist
- Verify the correct number of words (12 or 24)
- Make sure words are in the correct order

### "Key already exists"

```bash
# List existing keys
pawd keys list

# Delete the old key (if you're sure)
pawd keys delete mykey

# Then try again
pawd keys add mykey
```

### "Failed to generate secure entropy"

This is a system-level issue with your random number generator. Try:

- Restarting your terminal
- Checking system entropy: `cat /proc/sys/kernel/random/entropy_avail` (Linux)
- Running as a different user

## Testing Your Setup

### Verify Mnemonic Recovery Works

```bash
# 1. Create a test key
pawd keys add test-wallet

# 2. Copy the mnemonic that's displayed

# 3. Delete the key
pawd keys delete test-wallet

# 4. Recover it
pawd keys recover test-wallet
# Paste the mnemonic

# 5. Verify addresses match
pawd keys show test-wallet
```

## Example Scenarios

### Scenario 1: Complete New Setup

```bash
# Step 1: Create your main wallet
pawd keys add main-wallet

# Step 2: IMMEDIATELY write down your mnemonic on paper

# Step 3: Create encrypted backup
pawd keys export main-wallet > ~/safe-location/main-wallet.backup

# Step 4: Test recovery (to verify your backup works)
pawd keys delete main-wallet
pawd keys recover main-wallet
# Enter your mnemonic

# Step 5: Verify same address
pawd keys show main-wallet
```

### Scenario 2: Lost Computer

```bash
# On new computer, with your mnemonic:
pawd keys recover my-wallet
# Enter your 24-word mnemonic

# Verify
pawd keys show my-wallet
```

### Scenario 3: Multiple Wallets

```bash
# Personal wallet
pawd keys add personal

# Business wallet
pawd keys add business

# Validator wallet
pawd keys add validator

# List all
pawd keys list
```

## Performance Notes

Based on benchmarks:

- **Mnemonic generation**: ~2 microseconds (12 words), ~3 microseconds (24 words)
- **Mnemonic validation**: ~0.4 microseconds
- **Seed generation**: ~1 millisecond (intentionally slow for security)

These operations are nearly instantaneous from a user perspective.

## Getting Help

For more detailed information:

- Full documentation: `docs/BIP39_IMPLEMENTATION.md`
- Implementation summary: `BIP39_IMPLEMENTATION_SUMMARY.md`
- Command help: `pawd keys --help`
- Specific command: `pawd keys add --help`

## Important Reminders

1. **Your mnemonic is your wallet** - Anyone with your mnemonic has full access to your funds
2. **No recovery without mnemonic** - If you lose your mnemonic and keyring, your funds are gone forever
3. **PAW cannot help** - There is no "forgot password" or support recovery. Only your mnemonic works.
4. **Test before use** - Always verify recovery works before sending real funds
5. **Multiple backups** - Keep backups in at least 2 physical locations

## Next Steps

After creating your wallet:

1. Fund your wallet (get tokens)
2. Test sending a small transaction
3. Set up your validator (if applicable)
4. Configure automatic backups of your keyring
5. Document your HD derivation paths if using multiple accounts

## Quick Reference Card

```
CREATE:     pawd keys add <name> [--mnemonic-length 12|24]
RECOVER:    pawd keys recover <name>
LIST:       pawd keys list
SHOW:       pawd keys show <name>
EXPORT:     pawd keys export <name> > backup.txt
IMPORT:     pawd keys import <name> backup.txt
DELETE:     pawd keys delete <name>
```

---

**Last Updated**: 2025-11-14
**Version**: 1.0
