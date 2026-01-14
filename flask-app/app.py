"""
PAW Blockchain Explorer - Flask Application

A lightweight block explorer for PAW testnet that uses Cosmos RPC endpoints.
Displays blocks, transactions, validators, and network status.
"""

from flask import Flask, render_template, jsonify, request
import requests
from datetime import datetime
import os

app = Flask(__name__)

# Configuration
RPC_URL = os.getenv('RPC_URL', 'http://localhost:26657')
CHAIN_ID = os.getenv('CHAIN_ID', 'paw-mvp-1')

class RPCClient:
    """Client for Cosmos RPC endpoints"""

    def __init__(self, url):
        self.url = url.rstrip('/')

    def _get(self, endpoint, params=None):
        """Make GET request to RPC endpoint"""
        try:
            response = requests.get(f"{self.url}{endpoint}", params=params, timeout=5)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            app.logger.error(f"RPC request failed: {e}")
            return None

    def get_status(self):
        """Get node status"""
        return self._get('/status')

    def get_block(self, height=None):
        """Get block by height (latest if None)"""
        params = {'height': height} if height else None
        return self._get('/block', params=params)

    def get_block_results(self, height):
        """Get block results (tx results, begin/end block events)"""
        return self._get('/block_results', params={'height': height})

    def get_validators(self, height=None):
        """Get validators at height"""
        params = {'height': height} if height else None
        return self._get('/validators', params=params)

    def get_tx(self, hash_value):
        """Get transaction by hash"""
        return self._get('/tx', params={'hash': f"0x{hash_value}"})

    def search_tx(self, query, page=1, per_page=30):
        """Search transactions"""
        return self._get('/tx_search', params={
            'query': query,
            'page': page,
            'per_page': per_page,
            'order_by': 'desc'
        })

    def get_blockchain(self, min_height, max_height):
        """Get block headers between min and max height"""
        return self._get('/blockchain', params={
            'minHeight': min_height,
            'maxHeight': max_height
        })

rpc = RPCClient(RPC_URL)

# Helper functions
def format_timestamp(timestamp_str):
    """Format ISO timestamp to readable string"""
    try:
        dt = datetime.fromisoformat(timestamp_str.replace('Z', '+00:00'))
        return dt.strftime('%Y-%m-%d %H:%M:%S UTC')
    except:
        return timestamp_str

def format_hash(hash_str, length=16):
    """Truncate hash for display"""
    if not hash_str:
        return 'N/A'
    return f"{hash_str[:length]}...{hash_str[-8:]}" if len(hash_str) > length else hash_str

def parse_tx_data(tx_result):
    """Parse transaction result data"""
    if not tx_result:
        return None

    tx_data = {
        'hash': tx_result.get('hash', 'N/A'),
        'height': tx_result.get('height', 'N/A'),
        'index': tx_result.get('index', 0),
        'code': tx_result.get('tx_result', {}).get('code', 0),
        'gas_wanted': tx_result.get('tx_result', {}).get('gas_wanted', 0),
        'gas_used': tx_result.get('tx_result', {}).get('gas_used', 0),
        'events': tx_result.get('tx_result', {}).get('events', []),
        'log': tx_result.get('tx_result', {}).get('log', ''),
    }

    # Success if code == 0
    tx_data['success'] = tx_data['code'] == 0

    return tx_data

# Routes
@app.route('/')
def index():
    """Homepage - show recent blocks and network stats"""
    status = rpc.get_status()
    if not status:
        return render_template('error.html', error="Unable to connect to RPC node"), 500

    result = status.get('result', {})
    sync_info = result.get('sync_info', {})
    node_info = result.get('node_info', {})

    latest_height = int(sync_info.get('latest_block_height', 0))

    # Get recent blocks
    min_height = max(1, latest_height - 19)
    blocks_data = rpc.get_blockchain(min_height, latest_height)

    recent_blocks = []
    if blocks_data and 'result' in blocks_data:
        block_metas = blocks_data['result'].get('block_metas', [])
        for meta in block_metas:
            header = meta.get('header', {})
            recent_blocks.append({
                'height': header.get('height', 'N/A'),
                'time': format_timestamp(header.get('time', '')),
                'proposer': format_hash(header.get('proposer_address', ''), 12),
                'num_txs': meta.get('num_txs', 0),
                'hash': format_hash(meta.get('block_id', {}).get('hash', ''))
            })

    # Sort by height descending
    recent_blocks.sort(key=lambda x: int(x['height']) if x['height'] != 'N/A' else 0, reverse=True)

    return render_template('index.html',
                         chain_id=CHAIN_ID,
                         latest_height=latest_height,
                         latest_time=format_timestamp(sync_info.get('latest_block_time', '')),
                         node_version=node_info.get('version', 'Unknown'),
                         catching_up=sync_info.get('catching_up', False),
                         recent_blocks=recent_blocks[:20])

@app.route('/block/<int:height>')
def block_detail(height):
    """Show detailed information about a specific block"""
    block_data = rpc.get_block(height)
    if not block_data or 'result' not in block_data:
        return render_template('error.html', error=f"Block {height} not found"), 404

    block = block_data['result']['block']
    header = block['header']

    # Get block results for transaction details
    block_results = rpc.get_block_results(height)
    tx_results = []
    if block_results and 'result' in block_results:
        tx_results = block_results['result'].get('txs_results', [])

    # Parse transactions
    txs = block.get('data', {}).get('txs', [])
    transactions = []
    for idx, tx_hash in enumerate(txs):
        tx_info = {
            'index': idx,
            'hash': tx_hash[:32] if len(tx_hash) > 32 else tx_hash,
            'success': True,
            'gas_used': 0
        }
        if idx < len(tx_results):
            tx_result = tx_results[idx]
            tx_info['success'] = tx_result.get('code', 0) == 0
            tx_info['gas_used'] = tx_result.get('gas_used', 0)
        transactions.append(tx_info)

    block_info = {
        'height': header.get('height'),
        'chain_id': header.get('chain_id'),
        'time': format_timestamp(header.get('time', '')),
        'proposer': header.get('proposer_address'),
        'hash': block_data['result'].get('block_id', {}).get('hash'),
        'parent_hash': header.get('last_block_id', {}).get('hash'),
        'num_txs': len(txs),
        'transactions': transactions,
        'app_hash': header.get('app_hash'),
        'validators_hash': header.get('validators_hash'),
        'consensus_hash': header.get('consensus_hash'),
    }

    return render_template('block.html', block=block_info, chain_id=CHAIN_ID)

@app.route('/tx/<tx_hash>')
def tx_detail(tx_hash):
    """Show detailed information about a specific transaction"""
    # Remove 0x prefix if present
    tx_hash = tx_hash.replace('0x', '')

    tx_data = rpc.get_tx(tx_hash)
    if not tx_data or 'result' not in tx_data:
        return render_template('error.html', error=f"Transaction {tx_hash} not found"), 404

    tx_info = parse_tx_data(tx_data['result'])
    if not tx_info:
        return render_template('error.html', error="Error parsing transaction data"), 500

    return render_template('transaction.html', tx=tx_info, chain_id=CHAIN_ID)

@app.route('/validators')
def validators_list():
    """Show current validator set"""
    validators_data = rpc.get_validators()
    if not validators_data or 'result' not in validators_data:
        return render_template('error.html', error="Unable to fetch validators"), 500

    validators = []
    for val in validators_data['result'].get('validators', []):
        validators.append({
            'address': val.get('address'),
            'pub_key': format_hash(val.get('pub_key', {}).get('value', ''), 20),
            'voting_power': int(val.get('voting_power', 0)),
            'proposer_priority': int(val.get('proposer_priority', 0))
        })

    # Sort by voting power descending
    validators.sort(key=lambda x: x['voting_power'], reverse=True)

    total_power = sum(v['voting_power'] for v in validators)

    return render_template('validators.html',
                         validators=validators,
                         total_validators=len(validators),
                         total_power=total_power,
                         chain_id=CHAIN_ID)

@app.route('/search')
def search():
    """Search for blocks or transactions"""
    query = request.args.get('q', '').strip()

    if not query:
        return render_template('search.html', results=None, query='')

    # Try to parse as block height
    if query.isdigit():
        height = int(query)
        block_data = rpc.get_block(height)
        if block_data and 'result' in block_data:
            return render_template('search.html',
                                 results={'type': 'block', 'height': height},
                                 query=query)

    # Try as transaction hash
    if len(query) >= 32:
        tx_data = rpc.get_tx(query.replace('0x', ''))
        if tx_data and 'result' in tx_data:
            return render_template('search.html',
                                 results={'type': 'tx', 'hash': query},
                                 query=query)

    return render_template('search.html',
                         results={'type': 'none'},
                         query=query)

# API endpoints
@app.route('/api/status')
def api_status():
    """API: Get node status"""
    status = rpc.get_status()
    if not status:
        return jsonify({'error': 'Unable to connect to node'}), 500
    return jsonify(status.get('result', {}))

@app.route('/api/block/<int:height>')
def api_block(height):
    """API: Get block by height"""
    block_data = rpc.get_block(height)
    if not block_data:
        return jsonify({'error': f'Block {height} not found'}), 404
    return jsonify(block_data.get('result', {}))

@app.route('/api/validators')
def api_validators():
    """API: Get current validators"""
    validators_data = rpc.get_validators()
    if not validators_data:
        return jsonify({'error': 'Unable to fetch validators'}), 500
    return jsonify(validators_data.get('result', {}))

@app.errorhandler(404)
def not_found(e):
    return render_template('error.html', error="Page not found"), 404

@app.errorhandler(500)
def server_error(e):
    return render_template('error.html', error="Internal server error"), 500

if __name__ == '__main__':
    import sys
    port = 5000

    # Check for --port argument
    for i, arg in enumerate(sys.argv):
        if arg == '--port' and i + 1 < len(sys.argv):
            try:
                port = int(sys.argv[i + 1])
            except ValueError:
                print(f"Invalid port: {sys.argv[i + 1]}")
                sys.exit(1)

    app.run(host='0.0.0.0', port=port, debug=True)
