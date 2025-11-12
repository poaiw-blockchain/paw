# RFC-0006: Mobile Wallet Light Client

- **Author(s):** Wallet Team
- **Status:** Draft
- **Created:** 2025-11-11
- **Target Release:** Community Testnet

## Summary

Design the AURA mobile wallet as a non-custodial light client using Tendermint light-client protocol, secure enclave key storage, biometric gating, WalletConnect 2.0 bridge, and QR proof-of-possession workflows.

## Motivation & Goals

- Deliver sub-3-second verification UX with zero PII leakage.
- Ensure private keys never leave secure enclave / StrongBox.
- Provide simple onboarding + recovery for non-technical users.

## Detailed Design

- **Architecture:** React Native app (baseline) with native modules for secure enclave access; optional Flutter parity later.
- **Key Management:** BIP-39 seed encrypted, stored in device keystore; biometric confirmation required for sensitive actions. Social recovery + hardware backup described.
- **Light Client:** Periodically fetches trusted headers, verifies Merkle proofs for VC status, revocations, and governance commitments.
- **WalletConnect Flow:** Initiate session via QR, show human-readable consent dialogues, confirm via biometrics.
- **Credential Vault:** Stores encrypted VC payloads and associated proof parameters; purge on demand.
- **Recovery:** Multi-step flow combining mnemonic, biometric re-binding, and optional guardian approvals.

## Session Flow

Reference diagrams: `docs/architecture/flows/verifier-proof-flow.puml` (proof request) and `docs/architecture/flows/ir-completion.puml` (assistant handoff).

1. **Warm Sync:** App fetches latest trusted header from light-client peers on launch.
2. **Proof Request:** Verifier emits WalletConnect QR with VC type + nonce.
3. **User Approval:** Wallet scans QR, shows metadata, enforces biometric confirmation.
4. **Proof Assembly:** Wallet pulls cached VC + Merkle path from status registry via light client.
5. **Submission:** Proof sent through WalletConnect; response verified locally.
6. **Receipt Storage:** Encrypted receipt saved for audit (optional, default 24h retention).

## State Machine

| State | Description | Transitions |
| ----- | ----------- | ----------- |
| `FreshInstall` | App just installed; no seed yet. | `FreshInstall`, `SeedGen`, `SeedImport` |
| `SeedGen` | User creating new seed; not yet backed up. | `SeedBackedUp`, `Aborted` |
| `SeedImport` | Import mnemonic or hardware link. | `SeedBackedUp`, `Aborted` |
| `SeedBackedUp` | Backup confirmed; wallet locked until biometric binding. | `Ready` |
| `Ready` | Normal operating state; headers synced, keys bound. | `SessionPending`, `Recovery`, `Compromised` |
| `SessionPending` | WalletConnect session open awaiting biometrics. | `SessionApproved`, `SessionCancelled` |
| `SessionApproved` | Proof executed, awaiting verifier response. | `Ready` |
| `SessionCancelled` | User/retried; session cleared. | `Ready` |
| `Recovery` | Device lost / reinstall; requires guardian approval if enabled. | `Ready`, `Compromised` |
| `Compromised` | Keys suspected leaked; wallet nukes local state and prompts re-setup. | `FreshInstall` |

Guards / triggers:
- `SeedGen -> SeedBackedUp`: backup checklist (words written + camera confirmation) must succeed within 15 minutes.
- `Ready -> SessionPending`: requires active network + latest header within drift tolerance (default 5 minutes).
- `SessionPending -> SessionApproved`: biometric + secure enclave signature succeed; proof built and transmitted.
- `Recovery -> Ready`: guardian quorum (default 2-of-3) plus mnemonic verification.
- `Compromised` entered either via kill switch from server or local tamper detection.

## Parameters

- `header_drift_tolerance` (default 5 min) before forcing re-sync.
- `session_timeout` (default 120s) for WalletConnect approvals.
- `receipt_retention_hours` (default 24) for local proof receipts.
- `biometric_retries` (3) before fallback to PIN.
- Feature flags: `social_recovery_enabled`, `hardware_backup_enabled`.

## Interfaces

- gRPC/REST to `status registry` light client for VC proofs (`QueryProof`, `QueryRevocationRoot`).
- Push channel for governance announcements (optional WSS feed).
- WalletConnect 2.0 relay configuration stored in `config.json` inside the app bundle.

## Security / Privacy Considerations

- Threat model covering lost/stolen device, malicious verifier requests, MITM in WalletConnect, and side-channel attacks on proof generation.
- Prevent screenshot logging of sensitive views.

## Validation Plan

- Unit/UI tests, light-client verification tests, device-lab testing across iOS/Android.
- External security review before mainnet release.
- Automated integration tests mocking WalletConnect sessions + biometric prompts.
- Chaos testing for network partitions (header drift) and guardian-recovery flow.

## Open Questions

- Should we support desktop/browser extensions later?
- Level of optional analytics/telemetry (default off?).
