# RFC-0003: AI Assistant Network & GUI

- **Author(s):** AI/Oracle Team
- **Status:** Draft
- **Created:** 2025-11-11
- **Target Release:** Community Testnet

## Summary

Define the on-chain/off-chain architecture for user-sponsored AI assistants, including staking/bonding, locale routing, attestation formats, sponsorship vouchers, and the zero-code GUI that enables non-technical users to run assistants privately.

## Motivation & Goals

- Ensure assistants are accountable (stake + slashing) while keeping users in control of their own API keys/data.
- Provide high availability across locales/languages.
- Deliver a streamlined onboarding experience with incentive wallets and guided IR flows.

## Detailed Design

### On-Chain Module
- **State:** `AssistantRecord {address, owner_did, stake, locales[], model_hash, api_key_fingerprint, sponsorship_balance, heartbeat_ts}`.
- **Messages:** `MsgRegisterAssistant`, `MsgUpdateLocales`, `MsgReportMisbehavior`, `MsgHeartbeat`.
- **Slashing:** triggered by conflicting attestations, missed heartbeats, or fraud proofs submitted by monitors.
- **Routing:** identity module queries active assistants per locale; fallback to global pool if undersupplied.

### Off-Chain Components
- **GUI:** creates wallets, requests user API key, validates minimal hardware, guides IR capture, stores data locally, deletes on completion.
- **Attestation Signer:** packages `{ir_id, proof_hash, model_hash, timestamp}` and signs with assistant key.
- **Sponsorships:** optional encrypted vouchers from foundation/partners replenish compute credits without revealing user data.

## Security / Privacy Considerations

- Never transmit raw IR media off-device; only hashed proofs leave the machine.
- API keys stored encrypted and optionally hardware-bound; user can revoke anytime.
- Provide transparency logs of model versions and assistant updates.

## Validation Plan

- Testnet program requiring minimum assistants per locale with monitoring bots verifying heartbeat and attestation quality.
- Penetration testing on GUI + signer.

## Backwards Compatibility

- Migration path from single-assistant devnet to multi-assistant testnet (e.g., auto-register default assistant at genesis, allow upgrades via governance).

## Open Questions

- How to quantify locale demand and auto-adjust staking requirements?
- Should sponsorship vouchers be tradable NFTs? 
