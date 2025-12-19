# Validator Quickstart Pack (paw-testnet-1)

Ready-to-run assets for bringing up a public testnet validator with either systemd or Docker Compose, plus health checks and sizing guidance for light validators.

## What's Included
- `validator-onboarding/quickstart-pack/pawd.service` – systemd unit template.
- `validator-onboarding/quickstart-pack/docker-compose.validator.yml` – single-validator Compose stack (RPC/REST/gRPC/metrics exposed).
- `scripts/onboarding/node-onboard.sh` – one-line config/bootstrapper (full or light profile).
- `scripts/onboarding/validator-healthcheck.sh` – readiness probe (RPC + voting power).

## Systemd Path
```bash
sudo useradd -m -s /bin/bash paw || true
sudo mkdir -p /var/lib/paw && sudo chown -R paw:paw /var/lib/paw
sudo cp validator-onboarding/quickstart-pack/pawd.service /etc/systemd/system/pawd.service
sudo systemctl daemon-reload
curl -sL https://raw.githubusercontent.com/decristofaroj/paw/main/scripts/onboarding/node-onboard.sh \
  | sudo -u paw bash -s -- --mode full --chain-id paw-testnet-1 --home /var/lib/paw
sudo systemctl enable --now pawd
```
- Update `/var/lib/paw/config/config.toml` peers/seeds from `networks/paw-testnet-1/peers.txt` if they rotate.

## Docker Compose Path
```bash
cd validator-onboarding/quickstart-pack
docker compose -f docker-compose.validator.yml up -d --build
```
- Uses the repo `Dockerfile`, binds `~/.paw` into the container, and exposes 26656/26657/9090/1317/26660.
- Override chain ID or mount a prepared home by passing envs: `CHAIN_ID=paw-mainnet-1 docker compose ...`.

## Join Checklist
- [ ] Fetch genesis + checksum + peers from `networks/paw-testnet-1/` (verify SHA256).
- [ ] Set `minimum-gas-prices = "0.025upaw"`; open P2P port 26656 to trusted sentries.
- [ ] Enable metrics (`scripts/enable-node-metrics.sh`) and restart.
- [ ] Run `scripts/onboarding/validator-healthcheck.sh --rpc http://localhost:26657 --home /var/lib/paw`.
- [ ] Register validator (after faucet funding) via `scripts/register-validator.sh` or manual `pawd tx staking create-validator`.
- [ ] Add monitoring alerts for slashing/peers/height lag.

## Light-Validator Sizing (state sync + pruning)
- **Testnet**: 2 vCPU, 8 GB RAM, 150 GB NVMe, `pruning = "custom"`, `keep-recent = 1000`, `snapshot-interval = 0`, state sync trust window 168h.
- **Mainnet draft**: 4 vCPU, 16 GB RAM, 500 GB NVMe; raise `pruning-keep-recent` to 5000 once stable.
- Pair each light validator with a nearby full node for RPC; run `validator-healthcheck.sh` after every trust parameter rotation.
