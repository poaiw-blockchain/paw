# PAW Blockchain - IBC Implementation Documentation

## Overview

This document describes the production-ready Inter-Blockchain Communication (IBC) implementation for the PAW blockchain. The implementation enables cross-chain interoperability with other Cosmos chains and beyond.

## Table of Contents

1. [Architecture](#architecture)
2. [Modules Implemented](#modules-implemented)
3. [Cross-Chain Features](#cross-chain-features)
4. [Supported Chains](#supported-chains)
5. [Security Measures](#security-measures)
6. [Testing](#testing)
7. [Deployment](#deployment)
8. [Usage Examples](#usage-examples)

## Architecture

### IBC Core Integration

**File**: `app/app_ibc.go` (500+ lines)

The main IBC integration file implements:
- IBC core protocol (client, connection, channel management)
- ICS-20 token transfers with fee middleware
- ICS-27 interchain accounts (controller + host)
- ICS-29 fee payment middleware for relayer incentivization
- 07-tendermint light client (for Cosmos chains)
- 08-wasm light client (for non-Tendermint chains like Ethereum)

**Key Components**:
```go
type IBCKeepers struct {
    CapabilityKeeper      *capabilitykeeper.Keeper
    IBCKeeper             *ibckeeper.Keeper
    TransferKeeper        ibctransferkeeper.Keeper
    ICAControllerKeeper   icacontrollerkeeper.Keeper
    ICAHostKeeper         icahostkeeper.Keeper
    IBCFeeKeeper          ibcfeekeeper.Keeper
    WasmClientKeeper      ibcwasmkeeper.Keeper
    // Scoped keepers for port isolation
}
```

## Modules Implemented

### 1. DEX Cross-Chain Aggregation

**File**: `x/dex/keeper/ibc_aggregation.go` (470+ lines)
**Packet Types**: `x/dex/types/ibc_packets.go` (280+ lines)

**Features**:
- Query liquidity pools on remote chains (Osmosis, Injective)
- Execute cross-chain swaps with optimal routing
- Multi-hop swaps across multiple chains
- Automatic timeout handling and refunds
- Slippage protection
- Cross-chain price aggregation

**IBC Packet Types**:
- `QueryPoolsPacketData`: Request pool information
- `ExecuteSwapPacketData`: Execute swap on remote chain
- `CrossChainSwapPacketData`: Multi-hop cross-chain swaps
- `PoolUpdatePacketData`: Broadcast pool state changes

**Example Usage**:
```go
// Query pools on Osmosis
pools, err := dexKeeper.QueryCrossChainPools(
    ctx,
    "upaw",
    "uosmo",
    []string{"osmosis-1"},
)

// Execute cross-chain swap
route := CrossChainSwapRoute{
    Steps: []SwapStep{
        {ChainID: "paw-1", PoolID: "pool-1", TokenIn: "upaw", TokenOut: "uatom"},
        {ChainID: "osmosis-1", PoolID: "pool-42", TokenIn: "uatom", TokenOut: "uosmo"},
    },
}

result, err := dexKeeper.ExecuteCrossChainSwap(ctx, sender, route, maxSlippage)
```

### 2. Oracle Cross-Chain Price Feeds

**File**: `x/oracle/keeper/ibc_prices.go` (680+ lines)
**Packet Types**: `x/oracle/types/ibc_packets.go` (200+ lines)

**Features**:
- Subscribe to price feeds from remote oracles (Band Protocol, Slinky)
- Aggregate multi-chain oracle data
- Byzantine fault tolerance (requires 2/3+ consensus)
- Reputation-based weighting
- Anomaly detection and outlier filtering
- Automatic heartbeat monitoring

**IBC Packet Types**:
- `SubscribePricesPacketData`: Subscribe to price feeds
- `QueryPricePacketData`: Query current prices
- `PriceUpdatePacketData`: Receive price updates
- `OracleHeartbeatPacketData`: Liveness monitoring

**Example Usage**:
```go
// Register remote oracle source
err := oracleKeeper.RegisterCrossChainOracleSource(
    ctx,
    "band-laozi-testnet4",
    "band",
    "connection-1",
    "channel-3",
)

// Subscribe to price feeds
err = oracleKeeper.SubscribeToCrossChainPrices(
    ctx,
    []string{"BTC/USD", "ETH/USD", "ATOM/USD"},
    []string{"band-laozi-testnet4"},
    60, // Update every 60 seconds
)

// Get aggregated price from multiple sources
price, err := oracleKeeper.AggregateCrossChainPrices(ctx, "BTC/USD")
// Returns weighted median with Byzantine fault tolerance
```

### 3. Compute Cross-Chain Jobs

**File**: `x/compute/keeper/ibc_compute.go` (850+ lines)
**Packet Types**: `x/compute/types/ibc_packets.go` (240+ lines)

**Features**:
- Discover compute providers on remote chains
- Submit compute jobs to remote chains (Akash, Flux, Render)
- Cross-chain escrow management
- Zero-knowledge proof verification of results
- Result attestation from multiple validators
- Automatic escrow release/refund

**IBC Packet Types**:
- `DiscoverProvidersPacketData`: Find remote compute providers
- `SubmitJobPacketData`: Submit computation job
- `JobResultPacketData`: Receive computation results
- `JobStatusPacketData`: Query job status
- `ReleaseEscrowPacketData`: Release payment to provider

**Example Usage**:
```go
// Discover providers on Akash
providers, err := computeKeeper.DiscoverRemoteProviders(
    ctx,
    []string{"akashnet-2"},
    []string{"gpu", "tee"},
    sdk.NewDec(10),
)

// Submit compute job
job, err := computeKeeper.SubmitCrossChainJob(
    ctx,
    "wasm",
    jobData,
    JobRequirements{
        CPUCores: 4,
        MemoryMB: 8192,
        GPURequired: true,
        TEERequired: true,
    },
    "akashnet-2",
    "provider-123",
    requester,
    sdk.NewCoin("upaw", math.NewInt(1000000)),
)

// Job results received via IBC callback
// Escrow automatically released on successful verification
```

## Cross-Chain Features

### ICS-20 Token Transfers
- Standard IBC token transfers
- Multi-hop transfers (A → B → C)
- Timeout handling and refunds
- Memo support for contract calls

### ICS-27 Interchain Accounts
- **Controller**: PAW can control accounts on other chains
- **Host**: Other chains can control accounts on PAW
- Use cases: Governance voting, staking, liquidity provision

### ICS-29 Fee Middleware
- Relayer incentivization
- Configurable fees (receive, ack, timeout)
- Automatic fee distribution

### Custom Application Protocols
- **DEX**: Cross-chain liquidity aggregation
- **Oracle**: Multi-chain price feeds
- **Compute**: Distributed computation

## Supported Chains

### Cosmos Ecosystem
- **Cosmos Hub** (cosmoshub-4): Token transfers, ICA, Oracle feeds
- **Osmosis** (osmosis-1): DEX aggregation, token transfers
- **Celestia** (celestia): Compute jobs, data availability
- **Injective** (injective-1): DEX aggregation, derivatives

### Non-Tendermint Chains (via 08-wasm)
- **Ethereum**: Via wasm light client
- **Solana**: Via wasm light client
- Future support for more chains

## Security Measures

### IBC Security
1. **Light Client Verification**: All packets verified against light clients
2. **Timeout Handling**: Automatic refunds on timeout
3. **Byzantine Fault Tolerance**: Oracle data requires 2/3+ consensus
4. **Rate Limiting**: Configurable rate limits on IBC operations
5. **Proof Verification**: All cross-chain proofs cryptographically verified

### Module-Specific Security
1. **DEX**:
   - Slippage protection
   - Front-running prevention
   - Atomic cross-chain swaps

2. **Oracle**:
   - Reputation-based weighting
   - Anomaly detection
   - Heartbeat monitoring
   - BFT consensus (2/3+ threshold)

3. **Compute**:
   - Cryptographic escrow
   - Zero-knowledge proof verification
   - Result attestation
   - Provider reputation tracking

## Testing

### Test Suite

**Location**: `tests/ibc/` (1,800+ lines)

Comprehensive test coverage including:

1. **transfer_test.go** (400+ lines)
   - Basic IBC transfers
   - Timeout handling
   - Multi-hop transfers
   - Fee payment
   - Memo support

2. **dex_cross_chain_test.go** (450+ lines)
   - Remote pool queries
   - Cross-chain swaps
   - Multi-hop routing
   - Slippage protection
   - Timeout refunds

3. **oracle_ibc_test.go** (420+ lines)
   - Price subscriptions
   - Price queries
   - Price aggregation
   - Byzantine fault tolerance
   - Heartbeat monitoring

4. **compute_ibc_test.go** (530+ lines)
   - Provider discovery
   - Job submission
   - Result verification
   - ZK proof validation
   - Escrow management
   - Timeout handling

### Running Tests

```bash
# Run all IBC tests
go test ./tests/ibc/... -v

# Run specific test suite
go test ./tests/ibc/transfer_test.go -v
go test ./tests/ibc/dex_cross_chain_test.go -v
go test ./tests/ibc/oracle_ibc_test.go -v
go test ./tests/ibc/compute_ibc_test.go -v

# Run with coverage
go test ./tests/ibc/... -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Deployment

### Relayer Configuration

**File**: `ibc/relayer-config.yaml`

The Hermes relayer configuration includes:
- PAW chain configuration
- Target chain configurations (Osmosis, Cosmos Hub, Celestia, Injective)
- Channel and port filters
- Fee payment settings
- Telemetry and monitoring

### Setting Up Relayer

```bash
# Install Hermes
cargo install ibc-relayer-cli --bin hermes

# Add relayer keys
hermes keys add --chain paw-1 --key-file paw-relayer.json
hermes keys add --chain osmosis-1 --key-file osmosis-relayer.json

# Create IBC clients
hermes create client --host-chain paw-1 --reference-chain osmosis-1
hermes create client --host-chain osmosis-1 --reference-chain paw-1

# Create connection
hermes create connection \
  --a-chain paw-1 \
  --b-chain osmosis-1

# Create channels
hermes create channel \
  --a-chain paw-1 \
  --a-connection connection-0 \
  --a-port transfer \
  --b-port transfer

hermes create channel \
  --a-chain paw-1 \
  --a-connection connection-0 \
  --a-port dex \
  --b-port dex

# Start relayer
hermes --config ibc/relayer-config.yaml start
```

## Usage Examples

### Cross-Chain Token Transfer

```bash
# Transfer PAW tokens to Osmosis
pawd tx ibc-transfer transfer \
  transfer \
  channel-0 \
  osmo1recipientaddress... \
  1000000upaw \
  --from sender \
  --chain-id paw-1
```

### Cross-Chain DEX Swap

```bash
# Execute swap on remote chain
pawd tx dex cross-chain-swap \
  --route "paw-1:pool-1:upaw:uatom,osmosis-1:pool-42:uatom:uosmo" \
  --amount-in 1000000upaw \
  --min-out 950000uosmo \
  --max-slippage 0.05 \
  --from sender \
  --chain-id paw-1
```

### Subscribe to Oracle Prices

```bash
# Subscribe to cross-chain price feeds
pawd tx oracle subscribe-prices \
  --symbols "BTC/USD,ETH/USD,ATOM/USD" \
  --sources "band-laozi-testnet4" \
  --interval 60 \
  --from subscriber \
  --chain-id paw-1

# Query aggregated price
pawd query oracle aggregated-price BTC/USD
```

### Submit Compute Job

```bash
# Submit job to remote compute network
pawd tx compute submit-job \
  --job-type wasm \
  --job-data @job.wasm \
  --target-chain akashnet-2 \
  --provider provider-123 \
  --cpu 4 \
  --memory 8192 \
  --gpu \
  --tee \
  --payment 1000000upaw \
  --from requester \
  --chain-id paw-1

# Query job status
pawd query compute job-status job-123
```

## Monitoring

### Metrics

The relayer exposes Prometheus metrics at `http://localhost:3001`:
- Packet latency
- Success/failure rates
- Fee collection
- Channel health

### Logs

Enable verbose logging:
```bash
hermes --config ibc/relayer-config.yaml --log-level debug start
```

## Troubleshooting

### Common Issues

1. **Packet Timeout**
   - Increase timeout duration
   - Check relayer is running
   - Verify network connectivity

2. **Acknowledgement Errors**
   - Check packet data validity
   - Verify channel configuration
   - Review module-specific error logs

3. **Light Client Expiry**
   - Update light client
   - Configure client refresh in relayer

### Debug Commands

```bash
# Check channel status
hermes query channel end --chain paw-1 --port transfer --channel channel-0

# Query pending packets
hermes query packet pending --chain paw-1 --port transfer --channel channel-0

# Clear pending packets
hermes clear packets --chain paw-1 --port transfer --channel channel-0
```

## Performance Optimization

### Packet Batching
- Enable `tx_confirmation` for reliability
- Configure `clear_interval` for packet clearing
- Use `max_msg_num` to batch messages

### Gas Optimization
- Set appropriate `gas_multiplier` (1.1-1.3)
- Configure `max_gas` limits
- Monitor fee consumption

## Future Enhancements

1. **ICS-31 Interchain Queries**: Query state from remote chains
2. **Packet Fee Optimization**: Dynamic fee calculation
3. **Channel Upgrades**: Support for channel version upgrades
4. **Additional Light Clients**: Support more blockchain types
5. **Advanced Routing**: Multi-path routing for swaps

## References

- [IBC Protocol](https://ibcprotocol.org/)
- [IBC-Go Documentation](https://github.com/cosmos/ibc-go)
- [Hermes Relayer](https://hermes.informal.systems/)
- [ICS Standards](https://github.com/cosmos/ibc)

## Support

For issues and questions:
- Discord: https://discord.gg/DBHTc2QV
- Documentation: https://docs.paw-chain.com/ibc
