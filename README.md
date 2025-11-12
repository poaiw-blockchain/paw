# PAW Manageable Blockchain

> **🛠️ Development Tools Ready!** All linters, formatters, security scanners, and testing tools are installed and configured. See [TOOLS_SETUP.md](TOOLS_SETUP.md) for details.

This repo hosts the condensed PAW release: a lean layer-1 blockchain with a built-in DEX, secure API compute aggregation, and multi-device, mobile-friendly wallets. The key artifacts are:

- `PAW Extensive whitepaper .md`: the core spec covering tokenomics, DEX strategy, reward model, governance expansion, and launch roadmap.
- `PAW Future Phases.md`: a companion roadmap for later zkML, sharding, interoperability, and governance upgrades.

## Goals & Experience

- **Rapid Adoption**: Elevated early rewards for validators, node operators, and compute partners plus atomic-swap-ready liquidity pools and GUI hooks for quick onboarding.
- **DEX First**: Native AMM/atomic swap primitives, multi-token wallets, QR-friendly GUI, and mobile/desktop parity deliver liquidity-first functionality on day one.
- **Open Governance**: Delegated voting groups drawn via VRF expand participation beyond the initial validator set while the Guardian DAO governs upgrades and the adoption reserve.

## Getting Started

1. Review `PAW Extensive whitepaper .md` for launch details.
2. Reference `PAW Future Phases.md` when planning later scaling or privacy-focused releases.
3. Build GUIs and wallets that mirror the described QR / biometric flows to ensure the chain ships with consumer-ready UX.
4. See `TOOLS_SETUP.md` for all installed development tools (linters, formatters, security scanners, etc.)

## Publishing
Use the provided GitHub remote to push once the implementation is ready. The repository is configured to deploy the content above and should host any future contracts, SDKs, or UI prototypes.

## Reused Assets
- `external/aura/0003-ai-assistant-network.md` and `0006-wallet-light-client.md` document the wallet, assistant, and verifier flows we’re adopting for PAW’s compute plane and mobile GUI.
- `external/crypto/docs/*` contains onboarding, embedded-wallet, mobile bridge, light-client, and notification specs ready for PAW’s multi-device experience.
- `external/crypto/ai` brings fee optimization, fraud detection, and API-rotator helpers that can be wired into the reward/compute subsystems.
- `external/crypto/exchange-frontend` + `browser-wallet-extension` are production-grade DEX GUI and WalletConnect-style extension assets we can rebrand to make PAW a DEX from inception.
- `wallet/fernet_storage.py` demonstrates how PAW will replace the legacy XOR/weak storage layers with Fernet-authenticated encryption for any wallet payloads we ship.

## Bootstrap & Node Configuration
- `infra/node-config.yaml` encodes the validator roster, emission schedule, and reward breakdown used by our controller.
- `scripts/bootstrap-node.sh` initializes the `infra/node` directory, writes a placeholder genesis file, emits Fernet salts, and prepares `infra/node.env` that downstream binaries can consume.
- `infra/start-test-node.sh` simulates launching the controller node, printing genesis data and reminding operators how to plug in their binary.
