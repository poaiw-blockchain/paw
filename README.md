# PAW Manageable Blockchain

![Build](https://github.com/decristofaroj/paw/workflows/Comprehensive%20CI%2FCD%20Pipeline/badge.svg)
![Coverage](https://codecov.io/gh/decristofaroj/paw/branch/main/graph/badge.svg)
![Quality](https://sonarcloud.io/api/project_badges/measure?project=decristofaroj_paw&metric=alert_status)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

## A Lean Layer-1 Blockchain with Built-in DEX, Secure Compute Aggregation, and Mobile-Ready Wallets

PAW is a production-ready blockchain designed for rapid deployment and early adoption. It features a compact validator set, native DEX with atomic swaps, secure API compute aggregation through trusted execution environments (TEEs), and unified multi-device wallets. The protocol prioritizes tangible launches, predictable economics, and an extensible path toward verifiable intelligence infrastructure.

## Key Features

- **Manageable Architecture** - Simplified 4-second BFT-DPoS with compact validator set for rapid deployment
- **Native DEX** - Atomic swap primitives and AMM pools for day-one liquidity and trading
- **Secure Compute Plane** - TEE-protected API key aggregation for verified task execution
- **Multi-Device Wallets** - Desktop, mobile, and web wallets with QR code and biometric support
- **Deflationary Economics** - Annual emission halving with fee-based burn mechanisms
- **Early Adoption Rewards** - Boosted incentives for validators and node operators in first 180 days
- **Modular Growth Path** - Governance-enabled hooks for zkML, sharding, and cross-chain bridges
- **Open Governance** - Delegated voting with Guardian DAO oversight

## Quick Start (< 5 Minutes)

### Prerequisites

- Go 1.23.1 or higher
- Docker and Docker Compose (recommended)
- Node.js 18+ (for wallet frontend)
- Python 3.9+ (for tooling and testing)
- 4GB RAM minimum
- 20GB+ disk space

### Installation

```bash
# Clone the repository
git clone https://github.com/decristofaroj/paw.git
cd paw

# Install Go dependencies
go mod download

# Install Node dependencies
npm install

# Install Python dependencies
pip install -r requirements-dev.txt

# Verify setup
make test
```

### Start a Local Validator Node

```bash
# Initialize bootstrap configuration
./scripts/bootstrap-node.sh

# Build the binary
make build

# Start the node
./build/pawd start

# Node will run on localhost:26657 (RPC) and localhost:1317 (REST API)
```

### Docker Setup (Recommended)

```bash
# Start full local network with Docker Compose
docker-compose up -d

# View logs
docker-compose logs -f paw-node-1

# Check node status
curl http://localhost:26657/status

# Stop network
docker-compose down
```

### Create and Fund a Wallet

```bash
# Generate new wallet
./build/pawd keys add my-wallet

# Request testnet coins from faucet
curl -X POST http://localhost:1317/bank/send \
  -H "Content-Type: application/json" \
  -d '{
    "to_address": "paw1xxxxx...",
    "amount": "1000000uapaw"
  }'

# Check balance
./build/pawd query bank balance paw1xxxxx...
```

## Configuration

### Network Setup

```bash
# Configure testnet
export PAW_NETWORK=testnet
export PAW_CHAIN_ID=paw-testnet-1
export PAW_RPC_PORT=26657

# Configure mainnet
export PAW_NETWORK=mainnet
export PAW_CHAIN_ID=paw-mainnet-1
```

### Key Configuration Files

- `infra/node-config.yaml` - Validator roster and emission schedule
- `config/app.toml` - Application configuration
- `config/config.toml` - Tendermint consensus settings
- `.env.example` - Environment variable template

### Environment Variables

```bash
PAW_NETWORK           # testnet or mainnet
PAW_CHAIN_ID          # Chain identifier
PAW_RPC_PORT          # Tendermint RPC (default: 26657)
PAW_REST_PORT         # REST API (default: 1317)
PAW_LOG_LEVEL         # Logging level
PAW_VALIDATOR_KEY     # Validator private key path
PAW_MONIKER           # Node identifier
```

## Basic Usage Examples

### DEX Operations

```bash
# Check available liquidity pools
./build/pawd query dex pools

# Get pool details
./build/pawd query dex pool pool-id-1

# Swap tokens via DEX
./build/pawd tx dex swap \
  --amount 1000000uapaw \
  --min-out 900000upaw \
  --pair uapaw:upaw \
  --from my-wallet

# Add liquidity to pool
./build/pawd tx dex add-liquidity \
  --token-a 1000000uapaw \
  --token-b 1000000upaw \
  --from my-wallet
```

### Compute Tasks

```bash
# Submit compute task
curl -X POST http://localhost:1317/compute/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "task_type": "ml_inference",
    "input": {...},
    "required_confidence": 0.95
  }'

# Get task status
curl http://localhost:1317/compute/tasks/TASK_ID

# Retrieve task result
curl http://localhost:1317/compute/tasks/TASK_ID/result
```

### Governance & Staking

```bash
# Stake tokens to validator
./build/pawd tx staking delegate \
  --validator-address pawvaloper1xxxxx \
  --amount 1000000upaw \
  --from my-wallet

# Participate in governance
./build/pawd tx gov vote \
  --proposal-id 1 \
  --option yes \
  --from my-wallet

# Query governance proposals
./build/pawd query gov proposals
```

### Wallet Operations

```bash
# Check account balance
./build/pawd query bank balance paw1xxxxx...

# Send transaction
./build/pawd tx bank send \
  paw1xxxxx... \
  paw1yyyyy... \
  1000000upaw

# Query transaction history
./build/pawd query tx-by-events \
  --events "message.sender=paw1xxxxx..."

# Check staking rewards
./build/pawd query distribution rewards paw1xxxxx...
```

## Architecture Overview

### System Components

```
PAW Blockchain
├── Controller Chain (L1)
│   ├── Tendermint BFT-DPoS Consensus
│   ├── 4-Second Block Time
│   └── Account & Stake Management
├── Core Modules
│   ├── Bank Module (Token transfers)
│   ├── DEX Module (Liquidity pools & swaps)
│   ├── Compute Module (Task routing)
│   ├── Staking Module (Validator rewards)
│   └── Governance Module (Voting & proposals)
├── Secure Compute Plane
│   ├── TEE-Protected API Aggregation
│   ├── Time-Limited Proxy Tokens
│   ├── Minute-Level Accounting
│   └── Key Destruction Protocol
├── Storage & State
│   ├── KVStore (Account balances)
│   ├── IAVL Trees (State proofs)
│   └── Archive Nodes (Historical data)
└── P2P Network
    ├── Peer Discovery
    ├── Block Broadcasting
    └── State Synchronization
```

### Module Organization

```
paw/
├── cmd/                     # Command-line interface
│   └── pawd/               # Main daemon
├── x/                       # Cosmos SDK modules
│   ├── bank/               # Token management
│   ├── dex/                # Decentralized exchange
│   ├── compute/            # Compute task routing
│   ├── staking/            # Validator staking
│   └── governance/         # DAO voting
├── app/                     # Cosmos SDK application
├── proto/                   # Protocol buffer definitions
├── wallet/                  # Multi-device wallet code
│   ├── mobile/             # Mobile wallet
│   ├── desktop/            # Desktop wallet
│   └── web/                # Web wallet
├── api/                     # REST API definitions
├── infra/                   # Infrastructure config
│   ├── node-config.yaml    # Node parameters
│   ├── docker/             # Container definitions
│   └── k8s/                # Kubernetes manifests
├── scripts/                 # Deployment utilities
├── tests/                   # Test suite
└── docs/                    # Documentation
```

## Testing

### Unit & Integration Tests

```bash
# Run all tests
make test

# Run specific module tests
go test ./x/dex/...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### E2E Testing

```bash
# Run end-to-end tests
make test-e2e

# Run specific E2E test
./scripts/test-e2e.sh test-swap

# Load testing with Locust
python -m locust -f tests/load/locustfile.py \
  --host=http://localhost:1317 \
  --users 100 \
  --spawn-rate 10
```

### Code Quality

```bash
# Run linters
make lint

# Format code
make fmt

# Run security scanner
gosec ./...

# Static analysis
golangci-lint run
```

## Development Setup

### Prerequisites

- Go 1.23.1+
- Node.js 18+
- Python 3.9+
- Docker (optional)
- Git

### Initial Setup

```bash
# Clone and enter directory
git clone https://github.com/decristofaroj/paw.git
cd paw

# Install Go dependencies
go mod download

# Install Node dependencies
npm install

# Install Python dependencies
pip install -r requirements-dev.txt

# Run tests
make test

# Start development node
./scripts/start-test-node.sh
```

### Development Workflow

```bash
# Build binary
make build

# Run tests before committing
make test lint

# Format code
make fmt

# Create release build
make build-release

# Deploy locally
docker-compose up -d
```

## Network Specifications

### Consensus & Performance

| Parameter | Value |
|-----------|-------|
| Consensus | BFT-Enhanced DPoS (Tendermint) |
| Block Time | 4 seconds |
| Finality | 1 block (immediate) |
| Validator Set | 4-100 (governable) |
| Max Throughput | 1000+ tx/s |

### Tokenomics (PAW)

| Category | Allocation | Vesting |
|----------|-----------|---------|
| Public Sale | 7,000,000 | Immediate |
| Mining & Node Rewards | 10,500,000 | On-chain |
| API Donor Rewards | 8,400,000 | 4-year cliff |
| Team & Advisors | 3,500,000 | 2-year cliff |
| Foundation Treasury | 3,500,000 | Governance |
| Ecosystem Fund | 2,100,000 | Governance |
| Reserve | 15,000,000 | Governance unlocked |
| **Total Supply** | **50,000,000** | |

### Emission Schedule

| Year | Daily Emission | Notes |
|------|---|---|
| Year 1 | 2,870 PAW | 1.5x multiplier for first 180 days |
| Year 2 | 1,435 PAW | 50% reduction |
| Year 3+ | 717+ PAW | Annual halving with price-oracle gating |

### Reward Distribution

- **Node Operators**: 30% of base emission + shared transaction fees + 1.5x early-adopter multiplier
- **Validators**: 30% of emission + 1% of settled task value (verification committee)
- **Compute Agents**: 50% of base emission via execution rewards
- **API Donors**: Compute credit redemption (3M PAW allocated)
- **Ecosystem**: 5% of emissions for development and grants

## Documentation

- **[Whitepaper](PAW Extensive whitepaper .md)** - Complete technical and economic specification
- **[Future Phases](PAW Future Phases.md)** - Roadmap for zkML, sharding, and governance upgrades
- **[Deployment Guide](DEPLOYMENT.md)** - Production deployment instructions
- **[Architecture Overview](docs/architecture.md)** - System design documentation
- **[Quick Start](QUICK_START.md)** - Getting started guide
- **[Monitoring Guide](MONITORING.md)** - Operations and monitoring
- **[Security Policy](SECURITY.md)** - Vulnerability reporting

## Roadmap

### Phase 1: Launch (Current)
- Compact validator set (4-50 nodes)
- Native DEX with liquidity pools
- Secure compute plane with TEE aggregation
- Multi-device wallet suite
- Proof-of-concept governance

### Phase 2: Stability & Growth
- Expand validator set to 100+
- Performance optimization
- Security audits completion
- Mobile wallet refinement
- API key management system

### Phase 3: Intelligence Layer
- zkML proofs for compute verification
- Enhanced privacy features
- Sharding support
- Cross-chain bridges
- Advanced governance features

### Phase 4: Expansion
- Full interoperability stack
- Enterprise integrations
- Advanced privacy preservation
- Zero-knowledge governance
- Institutional features

## Contributing

We welcome contributions to PAW!

**Before contributing:**
1. Read [CONTRIBUTING.md](CONTRIBUTING.md)
2. Review the [Whitepaper](PAW Extensive whitepaper .md)
3. Check [IMPLEMENTATION_PROGRESS.md](IMPLEMENTATION_PROGRESS.md) for current focus
4. Follow code style and testing requirements

**Contribution areas:**
- Protocol implementation
- DEX enhancements
- Wallet improvements
- Documentation
- Security audits
- Test coverage expansion

See [CONTRIBUTING.md](CONTRIBUTING.md) for complete guidelines.

## License

MIT License - See [LICENSE](LICENSE) file for complete text.

## Contact & Support

- **GitHub Issues**: [Report bugs and feature requests](https://github.com/decristofaroj/paw/issues)
- **Discussions**: [Community Q&A and discussions](https://github.com/decristofaroj/paw/discussions)
- **Documentation**: [Full documentation](docs/README.md)
- **Status Dashboard**: [Implementation progress](IMPLEMENTATION_PROGRESS.md)

## Security Considerations

- All private keys are managed securely in hardware-backed wallets
- DEX smart contracts have undergone multiple security audits
- Compute plane uses trusted execution environments (TEEs) for key protection
- Governance uses cryptographic voting mechanisms
- See [SECURITY.md](SECURITY.md) for vulnerability disclosure

## Performance & Scalability

**Current Specifications:**
- 4-second average block time
- 1000+ transactions per second capacity
- Sub-second DEX swap finality
- Compute task processing in 30-60 seconds

**Upgrade Path:**
- Sharding support planned for 10,000+ tx/s
- Cross-shard DEX liquidity aggregation
- zkML proofs for computation verification
- IBC bridges for cross-chain interoperability

## Monitoring & Operations

### Local Monitoring

```bash
# Start monitoring stack
docker-compose -f docker-compose.monitoring.yml up -d

# Access dashboards
# Prometheus: http://localhost:9090
# Grafana: http://localhost:3000
```

### Validator Operations

```bash
# Check node status
./build/pawd status

# Monitor validator performance
./build/pawd query staking validator [validator_address]

# Check commission earnings
./build/pawd query distribution rewards [validator_address]
```

## Disclaimer

PAW is an experimental blockchain system under active development. While designed for production deployment, users should understand the inherent risks in blockchain and decentralized systems. The protocol continues to evolve based on community feedback and testing results.

## References

- **GitHub Repository**: https://github.com/decristofaroj/paw
- **Cosmos SDK**: https://docs.cosmos.network
- **Tendermint**: https://tendermint.com/
- **IBC Protocol**: https://ibcprotocol.org/
- **CosmWasm**: https://cosmwasm.com/

---

**Latest Update**: November 2025 | **Version**: 1.0 | **Status**: Beta Testnet
