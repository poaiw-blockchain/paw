# PAW Blockchain - GCP 3-Node Testnet Setup Guide
**AI Agent Reference - Last Updated: 2025-11-25**

## CURRENT STATE
- **Status**: VMs created but NOT configured with PAW blockchain yet
- **GCP Project**: `paw-mvp-1` (update for your environment)
- **Account**: `jeff@moproinsure.com`
- **Region/Zone**: `us-central1-a`
- **Nodes**: 3x e2-medium VMs running Ubuntu 22.04

## INFRASTRUCTURE

### Node Details
| Node | Internal IP | External IP | Status |
|------|-------------|-------------|--------|
| paw-testnode-1 | 10.128.0.5 | 34.29.163.145 | RUNNING |
| paw-testnode-2 | 10.128.0.3 | 108.59.86.86 | RUNNING |
| paw-testnode-3 | 10.128.0.4 | 35.184.167.38 | RUNNING |

### Cost Management
- **Running**: ~$0.10/hour ($2.40/day for 3 nodes)
- **Stopped**: ~$0.40/month (disk only)
- **CRITICAL**: Stop nodes when not testing!

## QUICK COMMANDS

### VM Management
```bash
# Check status
./scripts/devnet/gcp-manage.sh status

# Start all nodes
./scripts/devnet/gcp-manage.sh start

# Stop all nodes (SAVES MONEY)
./scripts/devnet/gcp-manage.sh stop

# SSH to a node
./scripts/devnet/gcp-manage.sh ssh 1   # or 2, 3

# Show logs
./scripts/devnet/gcp-manage.sh logs 1

# Show IPs
./scripts/devnet/gcp-manage.sh ips

# Cost estimate
./scripts/devnet/gcp-manage.sh cost
```

### GCloud Direct Commands
```bash
# List instances
gcloud compute instances list --project=paw-mvp-1

# SSH to node
gcloud compute ssh paw-testnode-1 --zone=us-central1-a --project=paw-mvp-1

# Stop specific node
gcloud compute instances stop paw-testnode-1 --zone=us-central1-a --project=paw-mvp-1

# Start specific node
gcloud compute instances start paw-testnode-1 --zone=us-central1-a --project=paw-mvp-1
```

## DEPLOYMENT STATUS

### ✅ Completed
1. GCP VMs provisioned
2. gcloud CLI authenticated
3. VM management script created (`scripts/devnet/gcp-manage.sh`)

### ❌ NOT Done Yet
1. Go installation on VMs
2. PAW binary build/deployment
3. Blockchain initialization
4. Genesis configuration
5. Node connectivity setup
6. Blockchain startup

## NEXT STEPS

### To Complete Setup
1. **Create deployment script** - Install Go, build pawd, configure nodes
2. **Initialize node1** - Create genesis, accounts, validator
3. **Initialize node2 & node3** - Copy genesis, configure peers
4. **Start blockchain** - Launch pawd on all nodes
5. **Verify sync** - Check nodes are connected and syncing

### Deployment Script Location
- Will be: `scripts/devnet/gcp-deploy.sh`
- Based on: `scripts/devnet/init_node.sh` (local devnet)

## BLOCKCHAIN CONFIG

### Chain Details
- **Chain ID**: `paw-devnet` (or `paw-mvp-1`)
- **Keyring**: `test`
- **Home Dir**: `/root/.paw/<node_name>`

### Ports
| Service | Port | Purpose |
|---------|------|---------|
| P2P | 26656 | Tendermint P2P |
| RPC | 26657 | Tendermint RPC |
| API | 1317 | Cosmos REST API |
| gRPC | 9090 | gRPC endpoint |

### Accounts to Create (on node1)
- `validator` - 200000000000upaw
- `smoke-trader` - 150000000000upaw,ufoo,ubar
- `smoke-counterparty` - 50000000000upaw

## KEY FILES

### Management Scripts
- `scripts/devnet/gcp-manage.sh` - Start/stop/status VMs
- `scripts/devnet/init_node.sh` - Local devnet initialization (reference)

### Docker/Build
- `Dockerfile` - Multi-stage build (Go + Node.js)
- `docker-compose.devnet.yml` - Local 2-node devnet
- `Makefile` - Build commands

### Configuration
- Chain home: `/root/.paw/<node_name>/` on each VM
- Genesis: `/root/.paw/node1/config/genesis.json` (node1 creates, others copy)
- Config: `/root/.paw/<node_name>/config/config.toml`
- App config: `/root/.paw/<node_name>/config/app.toml`

## TESTING APPROACH

Purpose: Find holes in code through thorough multi-node testing

### Test Categories
1. **Node sync** - Verify nodes connect and sync
2. **Transactions** - Test DEX, Oracle, Compute modules
3. **Validator operations** - Staking, slashing
4. **Network resilience** - Node failures, restarts
5. **Performance** - Load testing, stress testing
6. **Security** - Attack vectors, exploit attempts

### Test Scripts Available
- `scripts/devnet/smoke.sh` - Basic smoke tests
- `scripts/devnet/wait_and_smoke.sh` - Wait for chain + smoke test
- `scripts/devnet/smoke_checklist.sh` - Test checklist

## NETWORK ARCHITECTURE

```
paw-testnode-1 (Genesis/Validator)
    ↓ peer
paw-testnode-2 (Full Node)
    ↓ peer
paw-testnode-3 (Full Node)
```

### Connectivity
- Node1: Genesis node, creates chain state
- Node2 & Node3: Connect to Node1 as persistent peer
- All nodes: Use external IPs for inter-node communication

## TROUBLESHOOTING

### VMs not accessible
```bash
gcloud compute instances list --project=paw-mvp-1
gcloud compute ssh paw-testnode-1 --zone=us-central1-a
```

### Blockchain not starting
```bash
# Check logs on VM
journalctl -u pawd -n 100
tail -f /root/.paw/node1/pawd.log
```

### Nodes not syncing
```bash
# Check peer connections
curl http://localhost:26657/net_info

# Check node status
curl http://localhost:26657/status
```

## IMPORTANT NOTES

1. **Always stop VMs when not testing** - Use `./scripts/devnet/gcp-manage.sh stop`
2. **Node1 is special** - Creates genesis, others follow
3. **External IPs** - May change if VMs restarted (reserve static IPs if needed)
4. **Security** - Currently open ports, add firewall rules for production
5. **Backups** - No automated backups yet, create snapshots before risky operations

## REFERENCES

- GCP Console: https://console.cloud.google.com/compute/instances?project=paw-mvp-1
- Cosmos SDK Docs: https://docs.cosmos.network
- Local devnet: `docker-compose -f docker-compose.devnet.yml up`
