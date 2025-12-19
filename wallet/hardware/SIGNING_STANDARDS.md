# Hardware Signing Standards (Wallet Stack)

- **Derivation paths**: only `m/44'/118'/[0-4]'/0/0` are permitted; hardened on purpose for the first 3 segments. Any other path is rejected before touching a device.
- **Prefixes**: bech32 prefix must be `paw` for all addresses present in messages, fee payers, and memo metadata. Mixed-prefix payloads are refused.
- **Chain guardrails**: hardware signing requires an explicit `chain_id`; requests are rejected when the chain_id differs from the expected network or is missing.
- **Fee policy**: fee must contain a positive `gas` value and at least one coin; only `upaw` is allowed; negative amounts are rejected.
- **Sign modes**: amino is the hardware baseline; direct-sign requests are routed to software signers unless the transport advertises native support. The caller must request amino for Ledger/Trezor.
- **Attestation**: transports must advertise an allowed manufacturer/model; unexpected manufacturer/product values abort the session.
- **Transport coverage**: desktop → HID/WebHID; extension → WebHID/WebUSB; mobile → BLE with biometric gating; Trezor (web) uses Connect and inherits the same guards.
- **Rate limits**: signing is capped per-origin to prevent spam (default 5 requests per minute in the browser wallet).
- **User gates**: software signing requires a FIDO2/WebAuthn prompt (desktop/extension); mobile requires biometrics for pairing *and* signing.
- **Auditability**: every hardware/software signing request is logged with id/origin/mode/chain/address/outcome for replay-free auditing.
