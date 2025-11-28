"""Wallet management for PAW SDK."""

import hashlib
import hmac
from typing import Optional, Tuple
from mnemonic import Mnemonic
from ecdsa import SigningKey, SECP256k1
from ecdsa.util import sigencode_string_canonize
import bech32


class PawWallet:
    """PAW wallet with BIP39 support."""

    def __init__(self, prefix: str = "paw"):
        """Initialize wallet with address prefix."""
        self.prefix = prefix
        self._private_key: Optional[bytes] = None
        self._public_key: Optional[bytes] = None
        self._address: Optional[str] = None
        self._mnemonic: Optional[str] = None

    @staticmethod
    def generate_mnemonic(strength: int = 256) -> str:
        """Generate a new BIP39 mnemonic (24 words for 256 bits)."""
        mnemo = Mnemonic("english")
        return mnemo.generate(strength=strength)

    @staticmethod
    def validate_mnemonic(mnemonic: str) -> bool:
        """Validate a BIP39 mnemonic."""
        mnemo = Mnemonic("english")
        return mnemo.check(mnemonic)

    def from_mnemonic(self, mnemonic: str, hd_path: str = "m/44'/118'/0'/0/0") -> None:
        """Initialize wallet from mnemonic."""
        if not self.validate_mnemonic(mnemonic):
            raise ValueError("Invalid mnemonic phrase")

        self._mnemonic = mnemonic

        # Generate seed from mnemonic
        mnemo = Mnemonic("english")
        seed = mnemo.to_seed(mnemonic)

        # Derive private key from seed and HD path
        private_key = self._derive_private_key(seed, hd_path)

        # Generate public key and address
        self._private_key = private_key
        self._public_key = self._get_public_key(private_key)
        self._address = self._get_address(self._public_key)

    def _derive_private_key(self, seed: bytes, hd_path: str) -> bytes:
        """Derive private key from seed using BIP32 derivation."""
        # Simplified BIP32 derivation (for production, use a proper BIP32 library)
        # This is a basic implementation for demonstration
        master_key = hmac.new(b"Bitcoin seed", seed, hashlib.sha512).digest()
        private_key = master_key[:32]

        # In production, implement full BIP32 path derivation
        # For now, return the master private key
        return private_key

    def _get_public_key(self, private_key: bytes) -> bytes:
        """Get public key from private key."""
        sk = SigningKey.from_string(private_key, curve=SECP256k1)
        vk = sk.get_verifying_key()
        return vk.to_string("compressed")

    def _get_address(self, public_key: bytes) -> str:
        """Get bech32 address from public key."""
        # Hash the public key
        sha256_hash = hashlib.sha256(public_key).digest()
        ripemd160_hash = hashlib.new("ripemd160", sha256_hash).digest()

        # Convert to bech32
        five_bit_r = bech32.convertbits(ripemd160_hash, 8, 5)
        if five_bit_r is None:
            raise ValueError("Failed to convert address bits")

        address = bech32.bech32_encode(self.prefix, five_bit_r)
        return address

    @property
    def address(self) -> str:
        """Get wallet address."""
        if self._address is None:
            raise RuntimeError("Wallet not initialized")
        return self._address

    @property
    def public_key(self) -> bytes:
        """Get public key."""
        if self._public_key is None:
            raise RuntimeError("Wallet not initialized")
        return self._public_key

    def sign(self, message: bytes) -> bytes:
        """Sign a message with the private key."""
        if self._private_key is None:
            raise RuntimeError("Wallet not initialized")

        sk = SigningKey.from_string(self._private_key, curve=SECP256k1)
        return sk.sign_digest(
            hashlib.sha256(message).digest(),
            sigencode=sigencode_string_canonize
        )

    def export_mnemonic(self) -> str:
        """Export mnemonic (use with caution!)."""
        if self._mnemonic is None:
            raise RuntimeError("Wallet not initialized from mnemonic")
        return self._mnemonic
