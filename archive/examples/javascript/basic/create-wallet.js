#!/usr/bin/env node

/**
 * PAW Blockchain - Create Wallet Example
 *
 * This example demonstrates how to create a new wallet with a mnemonic phrase.
 *
 * Usage:
 *   node create-wallet.js
 *
 * Security Warning:
 *   Never share your mnemonic phrase or private key with anyone.
 *   Store them securely offline.
 */

import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';
import { stringToPath } from '@cosmjs/crypto';
import * as bip39 from 'bip39';

// Wallet configuration
const WALLET_PREFIX = 'paw';
const HD_PATH = "m/44'/118'/0'/0/0"; // Standard Cosmos HD path

/**
 * Generate a new mnemonic phrase
 * @param {number} strength - Entropy strength (128 = 12 words, 256 = 24 words)
 * @returns {string} Mnemonic phrase
 */
function generateMnemonic(strength = 256) {
  return bip39.generateMnemonic(strength);
}

/**
 * Create a new wallet from a mnemonic
 * @param {string} mnemonic - Optional mnemonic phrase (generates new if not provided)
 * @returns {Promise<Object>} Wallet information
 */
async function createWallet(mnemonic = null) {
  console.log('Creating New PAW Wallet...\n');

  try {
    // Generate mnemonic if not provided
    if (!mnemonic) {
      mnemonic = generateMnemonic();
      console.log('✓ Generated new mnemonic phrase');
    } else {
      // Validate provided mnemonic
      if (!bip39.validateMnemonic(mnemonic)) {
        throw new Error('Invalid mnemonic phrase');
      }
      console.log('✓ Using provided mnemonic phrase');
    }

    // Create wallet from mnemonic
    const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
      prefix: WALLET_PREFIX,
      hdPaths: [stringToPath(HD_PATH)]
    });

    // Get accounts from wallet
    const accounts = await wallet.getAccounts();
    const address = accounts[0].address;
    const pubkey = Buffer.from(accounts[0].pubkey).toString('hex');

    // Display wallet information
    console.log('\n' + '='.repeat(80));
    console.log('WALLET CREATED SUCCESSFULLY');
    console.log('='.repeat(80));
    console.log('\n⚠️  SECURITY WARNING: Keep this information secure and private!\n');

    console.log('Mnemonic Phrase (24 words):');
    console.log('-'.repeat(80));
    console.log(mnemonic);
    console.log('-'.repeat(80));

    console.log('\nWallet Details:');
    console.log(`  Address:    ${address}`);
    console.log(`  Public Key: ${pubkey}`);
    console.log(`  HD Path:    ${HD_PATH}`);
    console.log(`  Prefix:     ${WALLET_PREFIX}`);

    console.log('\nNext Steps:');
    console.log('  1. Save your mnemonic phrase in a secure location');
    console.log('  2. Never share your mnemonic with anyone');
    console.log('  3. Fund your wallet with PAW tokens');
    console.log('  4. Use your address to receive tokens');

    return {
      success: true,
      mnemonic,
      address,
      pubkey,
      hdPath: HD_PATH
    };

  } catch (error) {
    console.error('✗ Error creating wallet:', error.message);
    return {
      success: false,
      error: error.message
    };
  }
}

/**
 * Import existing wallet from mnemonic
 * @param {string} mnemonic - Mnemonic phrase
 * @returns {Promise<Object>} Wallet information
 */
async function importWallet(mnemonic) {
  console.log('Importing Existing Wallet...\n');

  try {
    // Validate mnemonic
    if (!bip39.validateMnemonic(mnemonic)) {
      throw new Error('Invalid mnemonic phrase. Please check and try again.');
    }

    const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
      prefix: WALLET_PREFIX,
      hdPaths: [stringToPath(HD_PATH)]
    });

    const accounts = await wallet.getAccounts();
    const address = accounts[0].address;

    console.log('✓ Wallet imported successfully');
    console.log(`\nAddress: ${address}\n`);

    return {
      success: true,
      address,
      wallet
    };

  } catch (error) {
    console.error('✗ Error importing wallet:', error.message);
    return {
      success: false,
      error: error.message
    };
  }
}

/**
 * Validate a mnemonic phrase
 * @param {string} mnemonic - Mnemonic phrase to validate
 * @returns {boolean} True if valid
 */
function validateMnemonic(mnemonic) {
  try {
    const isValid = bip39.validateMnemonic(mnemonic);
    console.log(`Mnemonic validation: ${isValid ? '✓ Valid' : '✗ Invalid'}`);
    return isValid;
  } catch (error) {
    console.error('✗ Error validating mnemonic:', error.message);
    return false;
  }
}

// Run the example if executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
  const args = process.argv.slice(2);

  if (args[0] === 'import' && args[1]) {
    // Import wallet from provided mnemonic
    importWallet(args.slice(1).join(' '))
      .then(result => {
        if (!result.success) {
          process.exit(1);
        }
      });
  } else if (args[0] === 'validate' && args[1]) {
    // Validate mnemonic
    const isValid = validateMnemonic(args.slice(1).join(' '));
    process.exit(isValid ? 0 : 1);
  } else {
    // Create new wallet
    createWallet()
      .then(result => {
        if (!result.success) {
          process.exit(1);
        }
      });
  }
}

// Export for testing
export { createWallet, importWallet, validateMnemonic, generateMnemonic };
