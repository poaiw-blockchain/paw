# PAW Block Explorer

A lightweight, Flask-based block explorer for the PAW blockchain testnet. The explorer provides a web interface to view blocks, transactions, validators, and network status using Cosmos RPC endpoints.

## Features

- **Block Browser**: View recent blocks and navigate through blockchain history
- **Block Details**: Detailed information about each block including transactions, hashes, and metadata
- **Transaction Viewer**: View transaction details, events, and execution results
- **Validator Set**: Display current validators with voting power and proposer priority
- **Network Status**: Real-time network statistics and sync status
- **Search**: Search by block height or transaction hash
- **REST API**: JSON API endpoints for programmatic access

## Architecture

The explorer is a Python Flask application that queries the Cosmos RPC endpoint directly:

```
┌─────────────┐      HTTP/RPC       ┌──────────────┐
│   Browser   │ ◄──────────────────► │    Flask     │
│  (Port 11080)│                     │   Explorer   │
└─────────────┘                      └──────┬───────┘
                                            │
                                        RPC │
                                            │
                                      ┌─────▼──────┐
                                      │ PAW Node   │
                                      │(Port 26657)│
                                      └────────────┘
```

## Prerequisites

- PAW blockchain node running with RPC enabled on port 26657
- Python 3.11+
- Flask, requests (installed via requirements.txt)

## Installation & Setup

### Option 1: Docker (Recommended)

The explorer can be run as part of the Docker Compose stack:

```bash
cd /home/hudson/blockchain-projects/paw/docker
docker-compose up -d explorer
```

Access the explorer at: **http://localhost:11080**

### Option 2: Standalone

Run the explorer directly on the host:

```bash
cd /home/hudson/blockchain-projects/paw
./scripts/start-explorer.sh
```

Or manually:

```bash
cd /home/hudson/blockchain-projects/paw/flask-app

# Install dependencies
pip3 install -r requirements.txt

# Set environment variables (optional, defaults shown)
export RPC_URL=http://localhost:26657
export CHAIN_ID=paw-testnet-1

# Run the explorer
python3 app.py --port 11080
```

Access the explorer at: **http://localhost:11080**

## Configuration

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `RPC_URL` | Cosmos RPC endpoint URL | `http://localhost:26657` |
| `CHAIN_ID` | Blockchain chain identifier | `paw-testnet-1` |
| `FLASK_ENV` | Flask environment (development/production) | `development` |

## Usage

### Web Interface

1. **Homepage** (`/`): Shows network overview and recent 20 blocks
2. **Block Detail** (`/block/<height>`): Detailed view of a specific block
3. **Transaction Detail** (`/tx/<hash>`): View transaction details and events
4. **Validators** (`/validators`): Current validator set with voting power
5. **Search** (`/search?q=<query>`): Search by block height or tx hash

### API Endpoints

The explorer provides JSON API endpoints for programmatic access:

```bash
# Get node status
curl http://localhost:11080/api/status

# Get block by height
curl http://localhost:11080/api/block/100

# Get current validators
curl http://localhost:11080/api/validators
```

## RPC Endpoints Used

The explorer uses the following Cosmos RPC endpoints:

- `/status` - Node status and sync info
- `/block` - Get block by height
- `/block_results` - Get transaction results for a block
- `/blockchain` - Get range of block headers
- `/validators` - Get validator set
- `/tx` - Get transaction by hash
- `/tx_search` - Search transactions (not currently used)

## Troubleshooting

### Explorer cannot connect to RPC

**Symptom**: "Unable to connect to RPC node" error

**Solution**:
1. Verify the PAW node is running: `curl http://localhost:26657/status`
2. Check the RPC_URL environment variable points to the correct endpoint
3. Ensure port 26657 is not blocked by firewall

### Docker container cannot reach host RPC

**Symptom**: Docker explorer container shows connection errors

**Solution**: The docker-compose.yml uses `host.docker.internal:26657` to reach the host's RPC endpoint. This should work on Linux with the `extra_hosts` configuration. If not:

1. Find your host IP: `ip addr show | grep "inet 192"`
2. Update RPC_URL in docker-compose.yml to use the host IP directly

### Page shows "No results" or empty data

**Symptom**: Explorer loads but shows no blocks/validators

**Solution**:
1. Check the node is producing blocks: `curl http://localhost:26657/status | jq .result.sync_info.latest_block_height`
2. Verify the node is not in catch-up mode
3. Check Flask logs for RPC errors

## File Structure

```
flask-app/
├── app.py                  # Main Flask application
├── requirements.txt        # Python dependencies
├── Dockerfile             # Docker build configuration
├── static/
│   └── css/
│       └── style.css      # Stylesheet
└── templates/
    ├── base.html          # Base template
    ├── index.html         # Homepage
    ├── block.html         # Block detail page
    ├── transaction.html   # Transaction detail page
    ├── validators.html    # Validator list page
    ├── search.html        # Search results page
    └── error.html         # Error page
```

## Limitations

1. **REST API not available**: Due to an IAVL bug in the current version, Cosmos SDK REST endpoints (`/cosmos/...`) are not working. The explorer uses RPC endpoints only.

2. **Transaction decoding**: Raw transaction data is displayed but not decoded into human-readable format. Future versions could add protobuf decoding.

3. **No historical charts**: The explorer shows current state only, not historical trends. Integration with Prometheus/Grafana is recommended for time-series data.

4. **No WebSocket support**: Updates require manual page refresh. Real-time updates could be added via WebSocket subscriptions to Tendermint RPC.

## Integration with Monitoring Stack

The explorer complements the existing monitoring infrastructure:

- **Prometheus** (port 11090): Metrics collection from node
- **Grafana** (port 11030): Time-series visualization and dashboards
- **Explorer** (port 11080): Human-readable blockchain data

Together they provide complete observability:
- Grafana: Historical trends, node performance, validator metrics
- Explorer: Current blockchain state, transactions, validator set

## Security Considerations

1. **Development mode only**: The current setup runs Flask in development mode with debug=True. For production:
   - Set `FLASK_ENV=production`
   - Use a production WSGI server (gunicorn/uwsgi)
   - Add authentication if exposed publicly

2. **No input sanitization**: User input (search queries) is minimally validated. Add proper input validation for production use.

3. **Rate limiting**: No rate limiting is implemented. Consider adding rate limiting to prevent abuse.

## Future Enhancements

Potential improvements:

1. **Transaction decoding**: Decode protobuf messages to show human-readable transaction types
2. **Address lookup**: Add address pages showing transaction history
3. **WebSocket updates**: Real-time block updates without page refresh
4. **Account balances**: Query account balances via gRPC
5. **IBC tracking**: Show IBC channels, packets, and cross-chain transfers
6. **Mempool viewer**: Show pending transactions in mempool
7. **Network topology**: Visualize P2P network connections
8. **Proposal viewer**: Display governance proposals and votes

## Support

For issues or questions:
1. Check the Flask application logs
2. Verify RPC endpoint connectivity
3. Review Docker logs: `docker logs paw-explorer`
4. Test RPC manually: `curl http://localhost:26657/status`

## References

- [Cosmos SDK RPC Documentation](https://docs.cosmos.network/main/core/rpc)
- [Tendermint RPC Spec](https://docs.tendermint.com/master/rpc/)
- [Flask Documentation](https://flask.palletsprojects.com/)
