# Phase C – Multi-Node Testnet Playbook

This guide documents the operational steps required to finish the Production Roadmap Phase C tasks: promoting additional validators, wiring persistent peers, checking consensus health, and validating slashing/jailing behavior. All steps run locally with the existing Docker Compose devnet stack.

## Prerequisites
- Docker + Docker Compose Plugin
- `go` toolchain (only needed if `pawd` binary is not prebuilt)
- `curl`, `jq`, and `bash`
- Repository root mounted at `/paw` inside the containers (default `compose` layout)

> **Home directories:** Each node stores state under `/root/.paw/<node>` inside its container. No state is stored inside the repository. Mnemonics captured by the helper script are written to `scripts/devnet/.state/`.

## 1. Start the 4-Node Devnet Stack

```bash
docker compose -f compose/docker-compose.devnet.yml up -d --remove-orphans
```

The compose file launches `node1` (validator) plus `node2`–`node4` (full nodes) sharing the repo volume and exposing RPC/gRPC/REST ports on the host. `scripts/devnet/init_node.sh` automatically sets deterministic port maps, `minimum-gas-prices`, and `persistent_peers` pointing every follower at `node1`.

Verify RPC readiness:

```bash
curl -s http://localhost:26657/status | jq '.result.sync_info'
```

## 2. Promote Additional Validators

Run the helper to convert `node2`–`node4` into bonded validators:

```bash
./scripts/devnet/setup-multivalidators.sh
```

What the script does:
1. Ensures the devnet stack is running (starts/stops automatically unless `PAW_KEEP_DEVNET=1` is set)
2. Creates a `validator` key on each node (mnemonics stored under `scripts/devnet/.state/node*_validator_key.txt`)
3. Funds every validator address from `node1`’s stake account
4. Broadcasts `tx staking create-validator` from each node and waits for `BOND_STATUS_BONDED`
5. Prints the validator set table so you can confirm voting power distribution

> **Idempotent:** Re-running the script skips nodes that are already bonded.

## 3. Validate Persistent Peers and Consensus

Check that all four nodes are connected and tracking the same height:

```bash
# Tendermint peer topology
curl -s http://localhost:26657/net_info | jq '.result.n_peers'

# Per-node status (replace NODE with node1…node4)
for NODE in node1 node2 node3 node4; do
  docker exec -i paw-${NODE} pawd status | jq '{node: "'${NODE}'", latest_block_height: .SyncInfo.latest_block_height}'
done
```

The heights should stay within 1–2 blocks of each other during normal operation. Peers are pinned through the `persistent_peers` lines written by `scripts/devnet/init_node.sh`. Update those values there if you need to test custom peer topologies or seed nodes.

## 4. Run Application Smoke Tests

Use the existing smoke suite to exercise bank + DEX flows across the multi-validator network:

```bash
./scripts/devnet/smoke_tests.sh
```

The suite keeps the compose stack up, sends funds between the prefunded accounts, creates a liquidity pool, performs swaps, and prints a summary of balances/pool counts.

## 5. Slashing & Jailing Drill

The devnet configuration keeps Cosmos SDK defaults (`signed_blocks_window = 100`, `min_signed_per_window = 0.5`). Use the following procedure to demonstrate downtime jailing and unjailing:

1. **Capture the validator identity**
   ```bash
   TARGET=node3
   CONS_ADDR=$(docker exec -i paw-${TARGET} pawd tendermint show-validator | jq -r '.key')
   pawd_exec() { docker exec -i paw-${1} pawd --home /root/.paw/${1} "$@"; }
   pawd_exec node1 query slashing signing-info "${CONS_ADDR}"
   ```
2. **Stop the validator container**
   ```bash
   docker compose -f compose/docker-compose.devnet.yml stop ${TARGET}
   ```
3. **Wait ~60 seconds (≈120 blocks)** to exceed the miss threshold:
   ```bash
   watch -n 5 'curl -s http://localhost:26657/status | jq ".result.sync_info.latest_block_height"'
   ```
4. **Check signing info** – `missed_blocks_counter` should exceed the threshold and `jailed_until` will be populated:
   ```bash
   pawd_exec node1 query slashing signing-info "${CONS_ADDR}"
   ```
5. **Restart the validator and unjail**
   ```bash
   docker compose -f compose/docker-compose.devnet.yml start ${TARGET}
   docker exec -i paw-${TARGET} pawd --home /root/.paw/${TARGET} tx slashing unjail \
     --from validator \
     --chain-id paw-devnet \
     --keyring-backend test \
     --gas auto --gas-adjustment 1.2 \
     --fees 5000upaw \
     --broadcast-mode block \
     --yes
   ```
6. **Verify recovery** – `jailed` should be false and the validator rejoins the active set once enough blocks are signed.

This drill confirms that:
- Slashing module records downtime infractions
- Validators transition to `jailed` state after exceeding miss thresholds
- Operators can unjail via the standard transaction flow once the validator node is back online

## 6. Troubleshooting Checklist
- Use `docker logs -f paw-nodeX` for per-node logs
- `scripts/devnet/.state/init_node*.log` captures bootstrap output
- If the helper script tears down the stack but you want it to persist, set `PAW_KEEP_DEVNET=1`
- Reset everything with `docker compose -f compose/docker-compose.devnet.yml down -v`

Completing the steps above satisfies the Production Roadmap Phase C deliverables: multi-validator configuration, persistent peer plumbing, consensus verification, and validation of slashing/jailing mechanics.
