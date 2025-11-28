#!/usr/bin/env python3
"""
PAW Blockchain - Create Wallet Example

This example demonstrates how to create a new wallet with a mnemonic phrase.

Usage:
    python create_wallet.py              # Create new wallet
    python create_wallet.py import <mnemonic>  # Import existing wallet

Security Warning:
    Never share your mnemonic phrase or private key with anyone.
    Store them securely offline.
"""

import sys
import hashlib
from typing import Dict, Any, Optional
from mnemonic import Mnemonic
from ecdsa import SigningKey, SECP256k1
from ecdsa.util import sigencode_string_canonize
import bech32

# Wallet configuration
WALLET_PREFIX = 'paw'
HD_PATH = "m/44'/118'/0'/0/0"
MNEMONIC_STRENGTH = 256  # 24 words


class PAWWallet:
    """PAW blockchain wallet"""

    def __init__(self, mnemonic_phrase: str):
        self.mnemonic = mnemonic_phrase
        self.private_key = None
        self.public_key = None
        self.address = None
        self._derive_keys()

    def _derive_keys(self):
        """Derive private and public keys from mnemonic"""
        # Generate seed from mnemonic
        mnemo = Mnemonic("english")
        seed = mnemo.to_seed(self.mnemonic)

        # For simplicity, use first 32 bytes as private key
        # In production, implement full BIP32/BIP44 derivation
        private_key_bytes = seed[:32]

        # Create signing key
        self.private_key = SigningKey.from_string(
            private_key_bytes,
            curve=SECP256k1
        )

        # Get public key
        self.public_key = self.private_key.get_verifying_key()

        # Generate address
        self.address = self._generate_address()

    def _generate_address(self) -> str:
        """Generate bech32 address from public key"""
        # Get compressed public key
        pubkey_bytes = self.public_key.to_string('compressed')

        # Hash public key (SHA256 then RIPEMD160)
        sha256_hash = hashlib.sha256(pubkey_bytes).digest()
        ripemd160_hash = hashlib.new('ripemd160', sha256_hash).digest()

        # Convert to bech32
        converted = bech32.convertbits(ripemd160_hash, 8, 5)
        address = bech32.bech32_encode(WALLET_PREFIX, converted)

        return address

    def get_public_key_hex(self) -> str:
        """Get public key as hex string"""
        return self.public_key.to_string('compressed').hex()


def generate_mnemonic(strength: int = MNEMONIC_STRENGTH) -> str:
    """
    Generate a new mnemonic phrase

    Args:
        strength: Entropy strength (128 = 12 words, 256 = 24 words)

    Returns:
        Mnemonic phrase
    """
    mnemo = Mnemonic("english")
    return mnemo.generate(strength=strength)


def create_wallet(mnemonic: Optional[str] = None) -> Dict[str, Any]:
    """
    Create a new wallet from a mnemonic

    Args:
        mnemonic: Optional mnemonic phrase (generates new if not provided)

    Returns:
        Dict with wallet information
    """
    print('Creating New PAW Wallet...\n')

    try:
        # Generate mnemonic if not provided
        if not mnemonic:
            mnemonic = generate_mnemonic()
            print('✓ Generated new mnemonic phrase')
        else:
            # Validate provided mnemonic
            mnemo = Mnemonic("english")
            if not mnemo.check(mnemonic):
                raise ValueError('Invalid mnemonic phrase')
            print('✓ Using provided mnemonic phrase')

        # Create wallet
        wallet = PAWWallet(mnemonic)

        # Display wallet information
        print('\n' + '=' * 80)
        print('WALLET CREATED SUCCESSFULLY')
        print('=' * 80)
        print('\n⚠️  SECURITY WARNING: Keep this information secure and private!\n')

        print('Mnemonic Phrase (24 words):')
        print('-' * 80)
        print(mnemonic)
        print('-' * 80)

        print('\nWallet Details:')
        print(f'  Address:    {wallet.address}')
        print(f'  Public Key: {wallet.get_public_key_hex()}')
        print(f'  HD Path:    {HD_PATH}')
        print(f'  Prefix:     {WALLET_PREFIX}')

        print('\nNext Steps:')
        print('  1. Save your mnemonic phrase in a secure location')
        print('  2. Never share your mnemonic with anyone')
        print('  3. Fund your wallet with PAW tokens')
        print('  4. Use your address to receive tokens')

        return {
            'success': True,
            'mnemonic': mnemonic,
            'address': wallet.address,
            'public_key': wallet.get_public_key_hex()
        }

    except Exception as e:
        print(f'✗ Error creating wallet: {str(e)}')
        return {
            'success': False,
            'error': str(e)
        }


def import_wallet(mnemonic: str) -> Dict[str, Any]:
    """
    Import existing wallet from mnemonic

    Args:
        mnemonic: Mnemonic phrase

    Returns:
        Dict with wallet information
    """
    print('Importing Existing Wallet...\n')

    try:
        # Validate mnemonic
        mnemo = Mnemonic("english")
        if not mnemo.check(mnemonic):
            raise ValueError('Invalid mnemonic phrase. Please check and try again.')

        # Create wallet
        wallet = PAWWallet(mnemonic)

        print('✓ Wallet imported successfully')
        print(f'\nAddress: {wallet.address}\n')

        return {
            'success': True,
            'address': wallet.address,
            'public_key': wallet.get_public_key_hex()
        }

    except Exception as e:
        print(f'✗ Error importing wallet: {str(e)}')
        return {
            'success': False,
            'error': str(e)
        }


def main():
    """Main entry point"""
    args = sys.argv[1:]

    if len(args) > 0 and args[0] == 'import':
        if len(args) < 2:
            print('Usage: python create_wallet.py import "<mnemonic>"')
            sys.exit(1)
        # Import wallet
        mnemonic = ' '.join(args[1:])
        result = import_wallet(mnemonic)
    else:
        # Create new wallet
        result = create_wallet()

    sys.exit(0 if result['success'] else 1)


if __name__ == '__main__':
    main()
