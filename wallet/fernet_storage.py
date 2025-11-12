"""Fernet-based wallet storage helpers for PAW."""

from __future__ import annotations

import base64
import os
from dataclasses import dataclass
from typing import Dict, Any

from cryptography.fernet import Fernet
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC


def _derive_fernet_key(password: str, salt: bytes) -> bytes:
    """Derive a Fernet-compatible key from `password` and `salt`."""
    kdf = PBKDF2HMAC(
        algorithm=hashes.SHA256(),
        length=32,
        salt=salt,
        iterations=600_000,
    )
    return base64.urlsafe_b64encode(kdf.derive(password.encode()))


@dataclass
class FernetPayload:
    data: str
    salt: str


def encrypt_wallet(state: Dict[str, Any], password: str) -> FernetPayload:
    """Return Fernet token + salt for the serialized wallet `state`."""
    salt = os.urandom(16)
    key = _derive_fernet_key(password, salt)
    token = Fernet(key).encrypt(base64.urlsafe_b64encode(state['serialized'].encode()))
    return FernetPayload(data=token.decode(), salt=base64.urlsafe_b64encode(salt).decode())


def decrypt_wallet(payload: FernetPayload, password: str) -> Dict[str, Any]:
    """Decrypt a Fernet payload and return the wallet state map."""
    salt = base64.urlsafe_b64decode(payload.salt)
    key = _derive_fernet_key(password, salt)
    decrypted = Fernet(key).decrypt(payload.data.encode())
    serialized = base64.urlsafe_b64decode(decrypted).decode()
    return {'serialized': serialized}
