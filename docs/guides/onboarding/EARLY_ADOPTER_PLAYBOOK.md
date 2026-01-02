# Early Adopter Playbook (Testnet Launch)

Actions and links for new users, operators, and contributors to get productive immediately on `paw-testnet-1`.

## 1) Get Funded & Connect
- Faucet: see `docs/TESTNET_QUICK_REFERENCE.md` (or `./scripts/faucet.sh --check <rpc-endpoint> <addr> 1000000upaw` using the published RPC endpoints).
- Status/endpoints: tracked in `docs/TESTNET_QUICK_REFERENCE.md` and `networks/paw-testnet-1/STATUS.md`.
- Wallets: desktop + browser extension + mobile (`wallet/mobile/ONBOARDING_GUIDE.md`).
- Light RPC endpoints (wallet-friendly): use a local light profile per `docs/guides/onboarding/LIGHT_CLIENT_PROFILE.md` or the published RPCs in `networks/paw-testnet-1/paw-testnet-1-manifest.json`.

## 2) Delegate & Participate in Gov
- Delegation quick path:
  ```bash
  pawd tx staking delegate <validator> 2500000upaw --from <key> --chain-id paw-testnet-1 --fees 5000upaw
  ```
- Monitor your rewards: `pawd q distribution rewards <delegator>`.
- Governance: list proposals `pawd q gov proposals`, vote `pawd tx gov vote <id> yes --from <key> --chain-id paw-testnet-1 --fees 5000upaw`.

## 3) Operators: Join & Stay Healthy
- Bootstrap via `docs/guides/onboarding/NODE_ONBOARDING.md` (full/light) or `validator-onboarding/QUICKSTART_PACK.md` (systemd/Compose).
- Health checks: `scripts/onboarding/validator-healthcheck.sh --rpc http://localhost:26657`.
- Metrics/Grafana: enable with `scripts/enable-node-metrics.sh`; dashboards live under `monitoring/grafana/dashboards/`.

## 4) Bug Bounty & Feedback
- Report critical security issues: security@paw.network.
- File reproducible bugs (wallet/core/explorer): GitHub issues with logs + `pawd version` + block height.
- Chaos/edge-case feedback: run `scripts/run-load-test.sh paw smoke` or `scripts/devnet/testnet-scenarios.sh` (see `NETWORK_CHAOS_TESTING.md`) and share findings.

## 5) Support Channels
- Discord: `#validator-tech` (operators), `#wallet-help` (end users).
- Forum: https://forum.paw.network for proposals/RCAs.
- Incident updates: monitor `networks/paw-testnet-1/STATUS.md` and repository releases.

## 6) Next Steps After Testnet
- Rotate to mainnet by swapping chain ID + artifact URLs (same onboarding scripts).
- Keep light RPCs online for wallets; archive nodes should publish snapshots every 1000 blocks to feed state-sync peers.
- Track roadmap completion in `ROADMAP_PRODUCTION.md`; add new tasks when issues are found.
