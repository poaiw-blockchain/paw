"""Tests for wallet functionality."""

import pytest
from paw import PawWallet


def test_generate_mnemonic():
    """Test mnemonic generation."""
    mnemonic = PawWallet.generate_mnemonic()
    words = mnemonic.split()
    assert len(words) == 24
    assert PawWallet.validate_mnemonic(mnemonic)


def test_validate_mnemonic():
    """Test mnemonic validation."""
    valid = PawWallet.generate_mnemonic()
    assert PawWallet.validate_mnemonic(valid)

    assert not PawWallet.validate_mnemonic("invalid mnemonic")
    assert not PawWallet.validate_mnemonic("")


def test_from_mnemonic():
    """Test wallet creation from mnemonic."""
    mnemonic = PawWallet.generate_mnemonic()
    wallet = PawWallet("paw")
    wallet.from_mnemonic(mnemonic)

    assert wallet.address.startswith("paw")
    assert len(wallet.public_key) > 0


def test_invalid_mnemonic():
    """Test wallet creation with invalid mnemonic."""
    wallet = PawWallet("paw")
    with pytest.raises(ValueError):
        wallet.from_mnemonic("invalid mnemonic phrase")


def test_sign_message():
    """Test message signing."""
    mnemonic = PawWallet.generate_mnemonic()
    wallet = PawWallet("paw")
    wallet.from_mnemonic(mnemonic)

    message = b"test message"
    signature = wallet.sign(message)
    assert len(signature) > 0


def test_export_mnemonic():
    """Test mnemonic export."""
    mnemonic = PawWallet.generate_mnemonic()
    wallet = PawWallet("paw")
    wallet.from_mnemonic(mnemonic)

    exported = wallet.export_mnemonic()
    assert exported == mnemonic
