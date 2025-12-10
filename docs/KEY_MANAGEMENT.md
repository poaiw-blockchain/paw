# PAW Key Management (Production/Testnet)

## Default Keyring Backend
- **Backend:** `os` (native OS keyring) — expected and recommended for validators, relayers, and faucets.
- Persisted under: `${PAW_HOME}/keyring-os/` (encrypted by OS keyring).
- Configure once:
  ```bash
  export PAW_HOME=${PAW_HOME:-$HOME/.paw}
  pawd config keyring-backend os --home "$PAW_HOME"
  ```

## Create Keys
- **Validator operator key (signs txs, not blocks):**
  ```bash
  pawd keys add validator-operator --home "$PAW_HOME" --keyring-backend os
  pawd keys show validator-operator --bech val -a --home "$PAW_HOME" --keyring-backend os
  ```
- **Faucet/distribution key:**
  ```bash
  pawd keys add faucet --home "$PAW_HOME" --keyring-backend os
  pawd keys show faucet -a --home "$PAW_HOME" --keyring-backend os
  ```
- **Air-gapped export (for cold backup only):**
  ```bash
  pawd keys export validator-operator --home "$PAW_HOME" --keyring-backend os > /tmp/validator-operator.key
  gpg -c /tmp/validator-operator.key
  shred -u /tmp/validator-operator.key
  ```
  Store the encrypted backup offline; never commit or share plaintext keys.

## Restore Keys
```bash
gpg -d validator-operator.key.gpg > /tmp/validator-operator.key
pawd keys import validator-operator /tmp/validator-operator.key --home "$PAW_HOME" --keyring-backend os
shred -u /tmp/validator-operator.key
```

## Operational Guidance
- Keep `${PAW_HOME}` out of git; it contains `config/`, `data/`, and `keyring-*`.
- Use separate keys for validator ops and faucet funding; least privilege.
- Prefer sentry architecture with remote signers (e.g., TMKMS) for consensus keys; do not store validator consensus keys in the same host keyring as the operator key.
- Rotate faucet keys periodically and cap per-transaction limits at the application layer.

## Validation Checks
- List keys: `pawd keys list --home "$PAW_HOME" --keyring-backend os`
- Verify backend: `pawd config get keyring-backend --home "$PAW_HOME"`
- Ensure backups exist and are encrypted (see `docs/DISASTER_RECOVERY.md` for rotation/verification playbooks).
