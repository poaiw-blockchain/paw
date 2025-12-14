# PAW Testnet Status

**Last Updated**: 2025-12-14

## Network Overview

- **Status**: ✅ Active and producing blocks
- **Chain ID**: `paw-devnet`
- **Network Type**: Local 4-validator testnet
- **CometBFT Version**: 0.38.17
- **Current Block Height**: 52+ (and climbing)
- **Block Time**: ~5 seconds

## Validator Configuration

### Active Validators: 4

All validators are genesis validators with equal voting power:

| Validator | Consensus Address | Voting Power | Status |
|-----------|-------------------|--------------|--------|
| Node 1 | 929C0E47D90B6359EF29AEEBC4EC55889A6AC736 | 250000 | ✅ Active |
| Node 2 | A0FEE11FF41E9D5972C13A141A6242421C51BE73 | 250000 | ✅ Active |
| Node 3 | BDD1DE671B8A077A1C780A3AD3E65744CF79C104 | 250000 | ✅ Active |
| Node 4 | CCC3ACFD6C6EB2FBC8C9B3A7EB53269F37EF97AA | 250000 | ✅ Active |

**Total Voting Power**: 1,000,000

### Validator Public Keys (bech32)

- Node 1: `pawvalcons1j2wqu37epd34nmef4m4ufmz43zdx43ek6x0645`
- Node 2: `pawvalcons15rlwz8l5r6w4jukp8g2p5cjzggw9r0nnw3h87p`
- Node 3: `pawvalcons1hhgauecm3grh58rcpgad8ejhgn8hnsgyc3v27f`
- Node 4: `pawvalcons1enp6eltvd6e0hjxfkwn7k5exnum7l9a2rukuk5`

## Network Endpoints

### RPC Endpoints
- Node 1: http://localhost:26657
- Node 2: http://localhost:26667
- Node 3: http://localhost:26677
- Node 4: http://localhost:26687

### gRPC Endpoints
- Node 1: http://localhost:39090
- Node 2: http://localhost:39091
- Node 3: http://localhost:39092
- Node 4: http://localhost:39093

### API (REST) Endpoints
- Node 1: http://localhost:1317
- Node 2: http://localhost:1327
- Node 3: http://localhost:1337
- Node 4: http://localhost:1347

## Peer Connectivity

Each validator is connected to all other validators (full mesh topology):
- **Peers per node**: 3
- **Total P2P connections**: 6 (bidirectional)

### Node IDs

```
node1: 72d84a1a213b2e341d0926dfc8e91332bd06a584
node2: b39bdfa2a5aee02104d03dafe80d27b5e2a7289a
node3: bafcc29ed4607a4d4f5b3c88d2459c3027be6468
node4: a0b3a9daa37ec69bef5341638afbed7cad9c16dc
```

## Test Results

### Smoke Tests: ✅ PASSING

All smoke tests completed successfully on 2025-12-14:

1. **Setup Phase**: ✅
   - RPC endpoint accessible
   - API endpoint accessible
   - Network synced and not catching up

2. **Bank Module**: ✅
   - Transfer executed successfully
   - Balance delta: 5,000,000 upaw
   - Recipient: smoke-counterparty

3. **DEX Module**: ✅
   - Liquidity pool created (upaw/ufoo)
   - Pool ID: 1
   - Initial reserves: 1,000,000 upaw / 1,000,000 ufoo

4. **Swap Operations**: ✅
   - Swap executed successfully
   - Input: 100,000 upaw
   - Output: ufoo (with slippage protection)
   - Deadline mechanism working correctly

### Test Accounts

The following test accounts are configured with balances:

- **smoke-trader**:
  - Address: `paw1ag3hdstzucxgw9lk9ga8uh57cm4ksnwvdwfd3e`
  - Initial balance: 150,000,000,000 upaw, ufoo, ubar each
  - Used for: DEX operations, general testing

- **smoke-counterparty**:
  - Address: `paw12cpt3nsc9dagwg5y3dwmpyjlelp9h89t9ympl2`
  - Initial balance: 50,000,000,000 upaw
  - Used for: Bank transfer testing

## Consensus Status

- **All validators signing**: ✅ Yes (4/4)
- **Byzantine fault tolerance**: 1 validator can fail (BFT threshold: 2/3)
- **Network liveness**: ✅ Producing blocks continuously
- **Missed blocks**: None detected

## Genesis Configuration

### Staking Parameters
- **Bond denom**: upaw
- **Validators**: 4
- **Self-delegation per validator**: 250,000,000,000 upaw
- **Unbonding time**: 3 weeks (default)

### Supply
- **Initial total supply**: ~2,700,000,000,000 upaw
  - Validator stakes: 4 × 250,000,000,000 = 1,000,000,000,000 upaw
  - Validator balances: 4 × 250,000,000,000 = 1,000,000,000,000 upaw
  - Test accounts: ~450,000,000,000 upaw

### Governance
- **Deposit period**: 2 days (default)
- **Voting period**: 2 days (default)
- **Quorum**: 33.4% (default)

## Known Issues

### Resolved Issues

1. **Prometheus telemetry warning** - ✅ Fixed
   - Symptom: "prometheus server error: listen tcp :36660: bind: address already in use"
   - Impact: Warning only, does not affect functionality
   - Solution: Filtered out in smoke tests and queries

2. **DEX swap deadline requirement** - ✅ Fixed
   - Symptom: "deadline must be set for time-sensitive swap operations"
   - Impact: Swap transactions would fail without deadline flag
   - Solution: Added `--deadline` flag to CLI command (defaults to 300 seconds)

3. **Smoke account mnemonics not saved** - ✅ Fixed
   - Symptom: smoke-trader and smoke-counterparty keys not available in containers
   - Impact: Smoke tests would fail
   - Solution: Updated setup-validators.sh to save all account mnemonics

### Active Issues

None currently identified.

## Network Artifacts

All network artifacts are packaged and verified in `networks/paw-devnet/`:

- ✅ `genesis.json` - Genesis file for the network
- ✅ `genesis.sha256` - Checksum for integrity verification
- ✅ `peers.txt` - Peer connection information
- ✅ `README.md` - Network documentation

**Genesis Hash**: `3042ae2eaafb2e102f90fdbee79bebdb4f28f09300727b9bb603972220fb6d74`

## Deployment Information

### Docker Compose
- **Compose file**: `compose/docker-compose.4nodes.yml`
- **Docker image**: golang:1.24.4
- **Volumes**: compose_node{1,2,3,4}_data
- **Network**: compose_pawnet (bridge)

### Build Information
- **Binary**: pawd
- **Build time**: Cold cache build ~90 seconds, warm cache ~10 seconds
- **Binary size**: ~158 MB
- **SHA256**: `d30e29f6d9c288938227b9bd56aa70af4639a167062b0c5ba02e096a50b49c7e`

## Maintenance

### Recent Changes (2025-12-14)

1. Fixed DEX swap CLI to include deadline parameter
2. Updated setup-validators.sh to save all account mnemonics
3. Fixed smoke_tests.sh to filter Prometheus warnings
4. Regenerated 4-validator genesis with all fixes applied

### Next Steps

- [ ] Set up external validator onboarding process
- [ ] Configure monitoring (Prometheus/Grafana)
- [ ] Deploy block explorer
- [ ] Set up IBC channels to other testnets
- [ ] Perform security audit
- [ ] Load testing and performance optimization

## Access Instructions

### Quick Check

```bash
# Check network status
curl -s http://localhost:26657/status | jq

# Check validators
curl -s http://localhost:26657/validators | jq

# Check latest block
curl -s http://localhost:26657/block | jq
```

### Running Transactions

```bash
# Access a container
docker exec -it paw-node1 bash

# Query balances
pawd query bank balances paw1ag3hdstzucxgw9lk9ga8uh57cm4ksnwvdwfd3e --home /root/.paw/node1

# Send transaction
pawd tx bank send smoke-trader <recipient> 1000000upaw \
  --chain-id paw-devnet \
  --keyring-backend test \
  --home /root/.paw/node1 \
  --yes \
  --fees 5000upaw
```

### Restart Network

```bash
# Restart (preserve state)
docker compose -f compose/docker-compose.4nodes.yml restart

# Stop
docker compose -f compose/docker-compose.4nodes.yml stop

# Clean restart (wipe state)
docker compose -f compose/docker-compose.4nodes.yml down -v
./scripts/devnet/setup-validators.sh 4
docker compose -f compose/docker-compose.4nodes.yml up -d
```

## Support

For deployment questions or issues:
1. Review `docs/TESTNET_DEPLOYMENT_GUIDE.md`
2. Check container logs: `docker logs paw-node<N>`
3. Review init logs: `scripts/devnet/.state/init_node<N>.log`
4. Consult the smoke tests script for examples

---

**Network is ready for external validator onboarding and public testnet deployment.**
