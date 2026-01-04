#!/usr/bin/env python3
"""
PAW Blockchain Explorer - Flask Application

A production-ready blockchain explorer with RPC integration,
real-time updates, and comprehensive blockchain data visualization.
"""

import csv
import io
import os
import logging
import time
import math
from collections import defaultdict
from datetime import datetime, timedelta
from functools import wraps
from typing import Dict, List, Optional, Any

import requests
from flask import Flask, render_template, jsonify, request, Response, make_response
from flask_caching import Cache
from flask_cors import CORS
from flasgger import Swagger, swag_from
from werkzeug.exceptions import HTTPException
import prometheus_client
from prometheus_client import Counter, Histogram, Gauge

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Initialize Flask app
app = Flask(__name__)

# Configuration
app.config.update(
    SECRET_KEY=os.getenv('FLASK_SECRET_KEY', 'dev-secret-key-change-in-production'),
    INDEXER_API_URL=os.getenv('INDEXER_API_URL', 'http://paw-indexer:8080'),
    RPC_URL=os.getenv('RPC_URL', 'http://paw-node:26657'),
    GRPC_URL=os.getenv('GRPC_URL', 'paw-node:9090'),
    CACHE_TYPE='simple',
    CACHE_DEFAULT_TIMEOUT=300,
    REQUEST_TIMEOUT=30,
    MAX_ITEMS_PER_PAGE=100,
    DEFAULT_ITEMS_PER_PAGE=20,
)

# Initialize extensions
cache = Cache(app)
CORS(app)

# Swagger configuration
swagger_config = {
    "headers": [],
    "specs": [
        {
            "endpoint": "apispec_1",
            "route": "/api/docs/apispec_1.json",
            "rule_filter": lambda rule: rule.endpoint.startswith('api_'),
            "model_filter": lambda tag: True,
        }
    ],
    "static_url_path": "/flasgger_static",
    "swagger_ui": True,
    "specs_route": "/api/docs"
}

swagger_template = {
    "swagger": "2.0",
    "info": {
        "title": "PAW Blockchain Explorer API",
        "description": "REST API for the PAW Blockchain Explorer. Provides access to blocks, transactions, accounts, validators, DEX data, and more.",
        "version": "1.0.0",
        "contact": {
            "name": "PAW Development Team"
        },
        "license": {
            "name": "MIT"
        }
    },
    "basePath": "/api/v1",
    "schemes": ["http", "https"],
    "tags": [
        {"name": "Blocks", "description": "Block-related endpoints"},
        {"name": "Transactions", "description": "Transaction-related endpoints"},
        {"name": "Accounts", "description": "Account-related endpoints"},
        {"name": "Rich List", "description": "Top token holders"},
        {"name": "Export", "description": "Data export endpoints"},
        {"name": "Statistics", "description": "Network statistics"},
        {"name": "Search", "description": "Search functionality"},
    ]
}

swagger = Swagger(app, config=swagger_config, template=swagger_template)


def safe_float(value: Any) -> float:
    """Convert a value into a float, gracefully handling invalid input."""
    try:
        return float(value)
    except (TypeError, ValueError):
        return 0.0


def format_pair(token_a: Optional[str], token_b: Optional[str]) -> str:
    """Render a token pair string."""
    token_a = (token_a or '?').upper()
    token_b = (token_b or '?').upper()
    return f"{token_a}/{token_b}"


def estimate_execution_price(reserve_in: float, reserve_out: float, trade_amount: float) -> float:
    """Estimate constant-product execution price for a swap."""
    if trade_amount <= 0 or reserve_in <= 0 or reserve_out <= 0:
        return 0.0
    k = reserve_in * reserve_out
    new_reserve_in = reserve_in + trade_amount
    if new_reserve_in <= 0:
        return 0.0
    new_reserve_out = k / new_reserve_in
    amount_out = reserve_out - new_reserve_out
    if amount_out <= 0:
        return 0.0
    return amount_out / trade_amount


def build_limit_orders(reserve_a: float, reserve_b: float,
                       token_a: str, token_b: str) -> List[Dict[str, Any]]:
    """Construct a pseudo order book to visualize pool depth."""
    spot_price = reserve_b / reserve_a if reserve_a > 0 else 0
    levels = [0.1, 0.25, 0.5, 1, 2, 5]
    book = []

    for level in levels:
        trade_size = reserve_a * (level / 100)
        execution_price = estimate_execution_price(reserve_a, reserve_b, trade_size)
        slippage = ((execution_price / spot_price) - 1) * 100 if spot_price else 0
        book.append({
            'level': level,
            'trade_size': trade_size,
            'execution_price': execution_price,
            'slippage': slippage,
            'direction': f"Sell {token_a.upper()}",
            'pair': format_pair(token_a, token_b),
        })
    return book


def enrich_pool_metrics(pool: Dict[str, Any]) -> Dict[str, Any]:
    """Attach derived metrics to a pool dictionary."""
    reserve_a = safe_float(pool.get('reserve_a'))
    reserve_b = safe_float(pool.get('reserve_b'))
    tvl = safe_float(pool.get('tvl') or pool.get('tvl_usd'))
    volume = safe_float(pool.get('volume_24h'))
    apr = safe_float(pool.get('apr'))
    swap_fee = safe_float(pool.get('swap_fee'))
    protocol_fee = safe_float(pool.get('protocol_fee'))
    price = reserve_b / reserve_a if reserve_a > 0 else 0

    enriched = dict(pool)
    enriched.update({
        'pair': format_pair(pool.get('token_a'), pool.get('token_b')),
        'tvl_value': tvl,
        'volume_value': volume,
        'apr_value': apr,
        'swap_fee_value': swap_fee,
        'protocol_fee_value': protocol_fee,
        'price': price,
        'depth_score': min(reserve_a, reserve_b),
        'stability_ratio': volume / tvl if tvl > 0 else 0,
        'limit_orders': build_limit_orders(
            reserve_a,
            reserve_b,
            pool.get('token_a', 'token_a'),
            pool.get('token_b', 'token_b'),
        ),
    })
    return enriched


def summarize_pools(pools: List[Dict[str, Any]]) -> Dict[str, Any]:
    """Produce aggregated metrics for the analytics dashboard."""
    if not pools:
        return {
            'total_tvl': 0.0,
            'total_volume': 0.0,
            'avg_apr': 0.0,
            'avg_fee': 0.0,
            'stability_index': 0.0,
            'pool_count': 0,
            'top_pools': [],
            'volume_leaders': [],
            'limit_showcase': [],
            'liquidity_alerts': [],
        }

    total_tvl = sum(p['tvl_value'] for p in pools)
    total_volume = sum(p['volume_value'] for p in pools)
    apr_samples = [p['apr_value'] for p in pools if p['apr_value']]
    avg_apr = sum(apr_samples) / max(len(apr_samples), 1)
    avg_fee = sum(p['swap_fee_value'] for p in pools) / len(pools)
    stability_index = sum(p['stability_ratio'] for p in pools) / len(pools)

    ranked_by_tvl = sorted(pools, key=lambda p: p['tvl_value'], reverse=True)
    ranked_by_volume = sorted(pools, key=lambda p: p['volume_value'], reverse=True)
    ranked_by_depth = sorted(pools, key=lambda p: p['depth_score'], reverse=True)
    max_depth = ranked_by_depth[0]['depth_score'] if ranked_by_depth else 0
    depth_threshold = max(max_depth * 0.08, 1_000)
    liquidity_alerts = [p for p in pools if p['depth_score'] < depth_threshold][:5]

    return {
        'total_tvl': total_tvl,
        'total_volume': total_volume,
        'avg_apr': avg_apr,
        'avg_fee': avg_fee,
        'stability_index': stability_index,
        'pool_count': len(pools),
        'top_pools': ranked_by_tvl[:8],
        'volume_leaders': ranked_by_volume[:5],
        'limit_showcase': ranked_by_depth[:3],
        'liquidity_alerts': liquidity_alerts,
    }

# Prometheus metrics
REQUEST_COUNT = Counter(
    'flask_explorer_requests_total',
    'Total request count',
    ['method', 'endpoint', 'status']
)
REQUEST_LATENCY = Histogram(
    'flask_explorer_request_duration_seconds',
    'Request latency',
    ['endpoint']
)
ACTIVE_REQUESTS = Gauge(
    'flask_explorer_active_requests',
    'Number of active requests'
)
RPC_ERRORS = Counter(
    'flask_explorer_rpc_errors_total',
    'Total RPC errors',
    ['endpoint']
)
CACHE_HITS = Counter(
    'flask_explorer_cache_hits_total',
    'Total cache hits',
    ['endpoint']
)


@app.template_filter('format_currency')
def format_currency(value: Any) -> str:
    """Render USD currency."""
    return f"${safe_float(value):,.2f}"


@app.template_filter('format_number')
def format_number(value: Any, decimals: int = 2) -> str:
    """Render a number with separators."""
    return f"{safe_float(value):,.{decimals}f}"


@app.template_filter('format_percent')
def format_percent(value: Any, decimals: int = 2) -> str:
    """Render percent helper."""
    return f"{safe_float(value):.{decimals}f}%"


app.jinja_env.globals['format_pair'] = format_pair


# Decorators
def track_metrics(f):
    """Decorator to track request metrics."""
    @wraps(f)
    def decorated_function(*args, **kwargs):
        ACTIVE_REQUESTS.inc()
        start_time = time.time()

        try:
            response = f(*args, **kwargs)
            status = 200 if not isinstance(response, tuple) else response[1]
            REQUEST_COUNT.labels(
                method=request.method,
                endpoint=request.endpoint,
                status=status
            ).inc()
            return response
        finally:
            duration = time.time() - start_time
            REQUEST_LATENCY.labels(endpoint=request.endpoint).observe(duration)
            ACTIVE_REQUESTS.dec()

    return decorated_function


class RPCClient:
    """Client for interacting with blockchain RPC and indexer API."""

    def __init__(self):
        self.indexer_url = app.config['INDEXER_API_URL']
        self.rpc_url = app.config['RPC_URL']
        self.rest_url = os.getenv('REST_URL', 'http://paw-node:1317')
        self.timeout = app.config['REQUEST_TIMEOUT']
        self.session = requests.Session()
        self.session.headers.update({'Content-Type': 'application/json'})

    def _make_request(self, url: str, method: str = 'GET',
                     params: Optional[Dict] = None,
                     json_data: Optional[Dict] = None) -> Optional[Dict]:
        """Make HTTP request with error handling."""
        try:
            response = self.session.request(
                method=method,
                url=url,
                params=params,
                json=json_data,
                timeout=self.timeout
            )
            response.raise_for_status()
            return response.json()
        except requests.RequestException as e:
            logger.error(f"Request failed: {url} - {str(e)}")
            RPC_ERRORS.labels(endpoint=url).inc()
            return None

    def get_latest_blocks(self, limit: int = 20) -> Optional[Dict]:
        """Get latest blocks from RPC."""
        # Get latest block height first
        status = self._make_request(f"{self.rpc_url}/status")
        if not status:
            return None

        latest_height = int(status.get('result', {}).get('sync_info', {}).get('latest_block_height', 0))
        if latest_height == 0:
            return {'blocks': [], 'total': 0}

        blocks = []
        for height in range(latest_height, max(0, latest_height - limit), -1):
            block_data = self._make_request(f"{self.rpc_url}/block", params={'height': str(height)})
            if block_data and 'result' in block_data:
                block = block_data['result'].get('block', {})
                header = block.get('header', {})
                blocks.append({
                    'height': int(header.get('height', 0)),
                    'hash': block_data['result'].get('block_id', {}).get('hash', ''),
                    'time': header.get('time', ''),
                    'proposer': header.get('proposer_address', ''),
                    'num_txs': len(block.get('data', {}).get('txs', []) or []),
                })

        return {'blocks': blocks, 'total': latest_height}

    def get_block(self, height: int) -> Optional[Dict]:
        """Get block by height from RPC."""
        block_data = self._make_request(f"{self.rpc_url}/block", params={'height': str(height)})
        if not block_data or 'result' not in block_data:
            return None

        block = block_data['result'].get('block', {})
        header = block.get('header', {})
        return {
            'height': int(header.get('height', 0)),
            'hash': block_data['result'].get('block_id', {}).get('hash', ''),
            'time': header.get('time', ''),
            'proposer': header.get('proposer_address', ''),
            'num_txs': len(block.get('data', {}).get('txs', []) or []),
            'txs': block.get('data', {}).get('txs', []),
        }

    def get_block_transactions(self, height: int) -> Optional[List[Dict]]:
        """Get transactions for a block from RPC."""
        result = self._make_request(f"{self.rpc_url}/tx_search",
                                    params={'query': f'"tx.height={height}"', 'prove': 'false'})
        if not result or 'result' not in result:
            return []
        return result['result'].get('txs', [])

    def get_latest_transactions(self, limit: int = 20) -> Optional[Dict]:
        """Get latest transactions from RPC."""
        result = self._make_request(f"{self.rpc_url}/tx_search",
                                    params={
                                        'query': '"tx.height>0"',
                                        'prove': 'false',
                                        'page': '1',
                                        'per_page': str(limit),
                                        'order_by': '"desc"'
                                    })
        if not result or 'result' not in result:
            return {'transactions': [], 'total': 0}

        txs = result['result'].get('txs', [])
        transactions = []
        for tx in txs:
            tx_result = tx.get('tx_result', {})
            transactions.append({
                'hash': tx.get('hash', ''),
                'height': int(tx.get('height', 0)),
                'code': tx_result.get('code', 0),
                'gas_used': tx_result.get('gas_used', '0'),
                'gas_wanted': tx_result.get('gas_wanted', '0'),
            })

        return {
            'transactions': transactions,
            'total': int(result['result'].get('total_count', 0))
        }

    def get_transaction(self, tx_hash: str) -> Optional[Dict]:
        """Get transaction by hash from RPC."""
        result = self._make_request(f"{self.rpc_url}/tx", params={'hash': f'0x{tx_hash}'})
        if not result or 'result' not in result:
            return None

        tx = result['result']
        tx_result = tx.get('tx_result', {})
        return {
            'hash': tx.get('hash', ''),
            'height': int(tx.get('height', 0)),
            'code': tx_result.get('code', 0),
            'gas_used': tx_result.get('gas_used', '0'),
            'gas_wanted': tx_result.get('gas_wanted', '0'),
            'log': tx_result.get('log', ''),
            'tx': tx.get('tx', ''),
        }

    def get_account(self, address: str) -> Optional[Dict]:
        """Get account information from REST API."""
        # Get account info
        auth_url = f"{self.rest_url}/cosmos/auth/v1beta1/accounts/{address}"
        account_data = self._make_request(auth_url)

        # Get balance
        balance_url = f"{self.rest_url}/cosmos/bank/v1beta1/balances/{address}"
        balance_data = self._make_request(balance_url)

        if not account_data and not balance_data:
            return None

        account = account_data.get('account', {}) if account_data else {}
        balances = balance_data.get('balances', []) if balance_data else []

        return {
            'account': account,
            'address': address,
            'balances': balances,
            'account_number': account.get('account_number', ''),
            'sequence': account.get('sequence', ''),
        }

    def get_account_transactions(self, address: str, limit: int = 20) -> Optional[List[Dict]]:
        """Get transactions for an account from RPC."""
        # Search for transactions involving this address
        result = self._make_request(f"{self.rpc_url}/tx_search",
                                    params={
                                        'query': f'"message.sender=\'{address}\'"',
                                        'prove': 'false',
                                        'page': '1',
                                        'per_page': str(limit),
                                        'order_by': '"desc"'
                                    })
        if not result or 'result' not in result:
            return []

        return result['result'].get('txs', [])

    def get_validators(self) -> Optional[List[Dict]]:
        """Get all validators."""
        url = f"{self.indexer_url}/api/v1/validators"
        return self._make_request(url)

    def get_validator(self, address: str) -> Optional[Dict]:
        """Get validator by address."""
        url = f"{self.indexer_url}/api/v1/validators/{address}"
        return self._make_request(url)

    def get_dex_pools(self) -> Optional[List[Dict]]:
        """Get all DEX pools."""
        url = f"{self.indexer_url}/api/v1/dex/pools"
        return self._make_request(url)

    def get_dex_pool(self, pool_id: str) -> Optional[Dict]:
        """Get DEX pool by ID."""
        url = f"{self.indexer_url}/api/v1/dex/pools/{pool_id}"
        return self._make_request(url)

    def get_pool_trades(self, pool_id: str, limit: int = 20) -> Optional[List[Dict]]:
        """Get trades for a pool."""
        url = f"{self.indexer_url}/api/v1/dex/pools/{pool_id}/trades"
        return self._make_request(url, params={'limit': limit})

    def get_dex_analytics_summary(self) -> Optional[Dict]:
        """Get aggregated DEX analytics summary."""
        url = f"{self.indexer_url}/api/v1/dex/analytics/summary"
        return self._make_request(url)

    def get_top_trading_pairs(self, limit: int = 10, period: str = '24h') -> Optional[Dict]:
        """Get top trading pairs."""
        url = f"{self.indexer_url}/api/v1/dex/analytics/top-pairs"
        return self._make_request(url, params={'limit': limit, 'period': period})

    def get_latest_dex_trades(self, limit: int = 20) -> Optional[Dict]:
        """Get latest DEX trades."""
        url = f"{self.indexer_url}/api/v1/dex/trades/latest"
        return self._make_request(url, params={'limit': limit})

    def get_oracle_prices(self) -> Optional[Dict]:
        """Get oracle price data."""
        url = f"{self.indexer_url}/api/v1/oracle/prices"
        return self._make_request(url)

    def get_compute_jobs(self, limit: int = 20) -> Optional[List[Dict]]:
        """Get compute jobs."""
        url = f"{self.indexer_url}/api/v1/compute/jobs"
        return self._make_request(url, params={'limit': limit})

    def get_network_stats(self) -> Optional[Dict]:
        """Get network statistics from RPC and REST."""
        # Get status from RPC
        status = self._make_request(f"{self.rpc_url}/status")
        if not status:
            return None

        sync_info = status.get('result', {}).get('sync_info', {})
        node_info = status.get('result', {}).get('node_info', {})

        # Get validators
        validators_data = self.get_validators_rest()
        validator_count = len(validators_data.get('validators', [])) if validators_data else 0

        # Get staking pool
        pool_data = self.get_staking_pool()
        bonded_tokens = pool_data.get('pool', {}).get('bonded_tokens', '0') if pool_data else '0'

        # Get total supply
        supply_url = f"{self.rest_url}/cosmos/bank/v1beta1/supply/upaw"
        supply_data = self._make_request(supply_url)
        total_supply = supply_data.get('amount', {}).get('amount', '0') if supply_data else '0'

        return {
            'latest_block_height': int(sync_info.get('latest_block_height', 0)),
            'latest_block_time': sync_info.get('latest_block_time', ''),
            'catching_up': sync_info.get('catching_up', False),
            'chain_id': node_info.get('network', ''),
            'moniker': node_info.get('moniker', ''),
            'validator_count': validator_count,
            'bonded_tokens': bonded_tokens,
            'total_supply': total_supply,
        }

    def search(self, query: str) -> Optional[Dict]:
        """Search for blocks, transactions, or accounts."""
        query = query.strip()

        # If it looks like a block height (all digits)
        if query.isdigit():
            block = self.get_block(int(query))
            if block:
                return {'type': 'block', 'result': block}

        # If it looks like a tx hash (64 hex chars)
        if len(query) == 64 and all(c in '0123456789abcdefABCDEF' for c in query):
            tx = self.get_transaction(query)
            if tx:
                return {'type': 'transaction', 'result': tx}

        # If it looks like an address (starts with paw1)
        if query.startswith('paw1') or query.startswith('pawvaloper1'):
            account = self.get_account(query)
            if account:
                return {'type': 'account', 'result': account}

        return {'type': 'not_found', 'result': None, 'query': query}

    def get_rpc_status(self) -> Optional[Dict]:
        """Get RPC node status."""
        url = f"{self.rpc_url}/status"
        return self._make_request(url)

    def get_rpc_health(self) -> Optional[Dict]:
        """Get RPC health status."""
        url = f"{self.rpc_url}/health"
        return self._make_request(url)

    # Governance endpoints (Cosmos SDK REST API)
    def get_proposals(self, status: Optional[str] = None) -> Optional[Dict]:
        """Get all governance proposals."""
        url = f"{self.rest_url}/cosmos/gov/v1beta1/proposals"
        params = {}
        if status:
            # Status values: PROPOSAL_STATUS_VOTING_PERIOD, PROPOSAL_STATUS_PASSED, etc.
            params['proposal_status'] = status
        return self._make_request(url, params=params if params else None)

    def get_proposal(self, proposal_id: int) -> Optional[Dict]:
        """Get proposal by ID."""
        url = f"{self.rest_url}/cosmos/gov/v1beta1/proposals/{proposal_id}"
        return self._make_request(url)

    def get_proposal_votes(self, proposal_id: int, pagination_key: Optional[str] = None) -> Optional[Dict]:
        """Get votes for a proposal."""
        url = f"{self.rest_url}/cosmos/gov/v1beta1/proposals/{proposal_id}/votes"
        params = {}
        if pagination_key:
            params['pagination.key'] = pagination_key
        return self._make_request(url, params=params if params else None)

    def get_proposal_tally(self, proposal_id: int) -> Optional[Dict]:
        """Get tally result for a proposal."""
        url = f"{self.rest_url}/cosmos/gov/v1beta1/proposals/{proposal_id}/tally"
        return self._make_request(url)

    def get_gov_params(self, params_type: str = 'voting') -> Optional[Dict]:
        """Get governance parameters (voting, tallying, deposit)."""
        url = f"{self.rest_url}/cosmos/gov/v1beta1/params/{params_type}"
        return self._make_request(url)

    # Staking endpoints (Cosmos SDK REST API)
    def get_staking_pool(self) -> Optional[Dict]:
        """Get staking pool info."""
        url = f"{self.rest_url}/cosmos/staking/v1beta1/pool"
        return self._make_request(url)

    def get_staking_params(self) -> Optional[Dict]:
        """Get staking parameters."""
        url = f"{self.rest_url}/cosmos/staking/v1beta1/params"
        return self._make_request(url)

    def get_delegations(self, delegator_address: str) -> Optional[Dict]:
        """Get delegations for an address."""
        url = f"{self.rest_url}/cosmos/staking/v1beta1/delegations/{delegator_address}"
        return self._make_request(url)

    def get_unbonding_delegations(self, delegator_address: str) -> Optional[Dict]:
        """Get unbonding delegations for an address."""
        url = f"{self.rest_url}/cosmos/staking/v1beta1/delegators/{delegator_address}/unbonding_delegations"
        return self._make_request(url)

    def get_delegation_rewards(self, delegator_address: str) -> Optional[Dict]:
        """Get pending rewards for a delegator."""
        url = f"{self.rest_url}/cosmos/distribution/v1beta1/delegators/{delegator_address}/rewards"
        return self._make_request(url)

    def get_validator_delegations(self, validator_address: str) -> Optional[Dict]:
        """Get delegations to a validator."""
        url = f"{self.rest_url}/cosmos/staking/v1beta1/validators/{validator_address}/delegations"
        return self._make_request(url)

    # Validators endpoints (Cosmos SDK REST API)
    def get_validators_rest(self, status: Optional[str] = None) -> Optional[Dict]:
        """Get all validators from REST API."""
        url = f"{self.rest_url}/cosmos/staking/v1beta1/validators"
        params = {}
        if status:
            # BOND_STATUS_BONDED, BOND_STATUS_UNBONDED, BOND_STATUS_UNBONDING
            params['status'] = status
        return self._make_request(url, params=params if params else None)

    def get_validator_rest(self, validator_address: str) -> Optional[Dict]:
        """Get validator details from REST API."""
        url = f"{self.rest_url}/cosmos/staking/v1beta1/validators/{validator_address}"
        return self._make_request(url)

    def get_validator_commission(self, validator_address: str) -> Optional[Dict]:
        """Get validator commission."""
        url = f"{self.rest_url}/cosmos/distribution/v1beta1/validators/{validator_address}/commission"
        return self._make_request(url)

    def get_slashing_params(self) -> Optional[Dict]:
        """Get slashing parameters."""
        url = f"{self.rest_url}/cosmos/slashing/v1beta1/params"
        return self._make_request(url)

    def get_validator_signing_info(self, cons_address: str) -> Optional[Dict]:
        """Get validator signing info for uptime."""
        url = f"{self.rest_url}/cosmos/slashing/v1beta1/signing_infos/{cons_address}"
        return self._make_request(url)

    def get_all_balances(self, limit: int = 1000) -> Optional[List[Dict]]:
        """Get all account balances from indexer for rich list."""
        url = f"{self.indexer_url}/api/v1/accounts/balances"
        return self._make_request(url, params={'limit': limit, 'sort': 'balance_desc'})

    def get_total_supply(self, denom: str = 'upaw') -> Optional[Dict]:
        """Get total supply of a token."""
        url = f"{self.rest_url}/cosmos/bank/v1beta1/supply/{denom}"
        return self._make_request(url)

    def get_account_transactions_all(self, address: str, limit: int = 1000) -> Optional[List[Dict]]:
        """Get all transactions for an account (for export)."""
        # Use RPC tx_search to find transactions
        result = self._make_request(f"{self.rpc_url}/tx_search",
                                    params={
                                        'query': f'"message.sender=\'{address}\'"',
                                        'prove': 'false',
                                        'page': '1',
                                        'per_page': str(min(limit, 100)),
                                        'order_by': '"desc"'
                                    })
        if not result or 'result' not in result:
            return []

        txs = result['result'].get('txs', [])
        transactions = []
        for tx in txs:
            tx_result = tx.get('tx_result', {})
            transactions.append({
                'hash': tx.get('hash', ''),
                'height': tx.get('height', ''),
                'timestamp': '',  # RPC doesn't provide timestamp directly
                'type': 'tx',
                'sender': address,
                'receiver': '',
                'messages': [],
                'fee_amount': '',
                'fee_denom': '',
                'status': 'success' if tx_result.get('code', 0) == 0 else 'failed',
                'block_height': tx.get('height', ''),
                'memo': '',
            })

        return transactions


# Initialize RPC client
rpc_client = RPCClient()


# Error handlers
@app.errorhandler(404)
def not_found(e):
    """Handle 404 errors."""
    if request.path.startswith('/api/'):
        return jsonify({'error': 'Not found', 'status': 404}), 404
    return render_template('404.html'), 404


@app.errorhandler(500)
def internal_error(e):
    """Handle 500 errors."""
    logger.error(f"Internal error: {str(e)}")
    if request.path.startswith('/api/'):
        return jsonify({'error': 'Internal server error', 'status': 500}), 500
    return render_template('500.html'), 500


@app.errorhandler(Exception)
def handle_exception(e):
    """Handle all other exceptions."""
    if isinstance(e, HTTPException):
        return e

    logger.exception("Unhandled exception")
    if request.path.startswith('/api/'):
        return jsonify({'error': 'An unexpected error occurred', 'status': 500}), 500
    return render_template('500.html'), 500


# Health check endpoints
@app.route('/health')
@track_metrics
def health():
    """Basic health check."""
    return jsonify({
        'status': 'healthy',
        'timestamp': datetime.utcnow().isoformat(),
        'version': '1.0.0'
    })


@app.route('/health/ready')
@track_metrics
def health_ready():
    """Readiness check."""
    # Check if indexer is accessible
    try:
        response = requests.get(
            f"{app.config['INDEXER_API_URL']}/health",
            timeout=5
        )
        indexer_healthy = response.status_code == 200
    except Exception:
        indexer_healthy = False

    # Check if RPC is accessible
    try:
        response = requests.get(
            f"{app.config['RPC_URL']}/health",
            timeout=5
        )
        rpc_healthy = response.status_code == 200
    except Exception:
        rpc_healthy = False

    ready = indexer_healthy and rpc_healthy

    return jsonify({
        'ready': ready,
        'checks': {
            'indexer': indexer_healthy,
            'rpc': rpc_healthy
        }
    }), 200 if ready else 503


@app.route('/metrics')
def metrics():
    """Prometheus metrics endpoint."""
    return Response(
        prometheus_client.generate_latest(),
        mimetype='text/plain'
    )


# Web UI routes
@app.route('/')
@track_metrics
@cache.cached(timeout=60)
def index():
    """Home page with dashboard."""
    stats = rpc_client.get_network_stats() or {}
    latest_blocks = rpc_client.get_latest_blocks(10) or []
    latest_txs = rpc_client.get_latest_transactions(10) or []

    return render_template(
        'index.html',
        stats=stats,
        blocks=latest_blocks,
        transactions=latest_txs
    )


@app.route('/blocks')
@track_metrics
def blocks_page():
    """Blocks list page."""
    page = request.args.get('page', 1, type=int)
    limit = min(
        request.args.get('limit', app.config['DEFAULT_ITEMS_PER_PAGE'], type=int),
        app.config['MAX_ITEMS_PER_PAGE']
    )

    blocks = rpc_client.get_latest_blocks(limit) or []

    return render_template('blocks.html', blocks=blocks, page=page)


@app.route('/block/<int:height>')
@track_metrics
@cache.cached(timeout=300, query_string=True)
def block_detail(height):
    """Block detail page."""
    block = rpc_client.get_block(height)
    if not block:
        return render_template('404.html'), 404

    transactions = rpc_client.get_block_transactions(height) or []

    return render_template('block.html', block=block, transactions=transactions)


@app.route('/transactions')
@track_metrics
def transactions_page():
    """Transactions list page."""
    page = request.args.get('page', 1, type=int)
    limit = min(
        request.args.get('limit', app.config['DEFAULT_ITEMS_PER_PAGE'], type=int),
        app.config['MAX_ITEMS_PER_PAGE']
    )

    transactions = rpc_client.get_latest_transactions(limit) or []

    return render_template('transactions.html', transactions=transactions, page=page)


@app.route('/tx/<tx_hash>')
@track_metrics
@cache.cached(timeout=300, query_string=True)
def transaction_detail(tx_hash):
    """Transaction detail page."""
    transaction = rpc_client.get_transaction(tx_hash)
    if not transaction:
        return render_template('404.html'), 404

    return render_template('transaction.html', transaction=transaction)


@app.route('/account/<address>')
@track_metrics
@cache.cached(timeout=60, query_string=True)
def account_detail(address):
    """Account detail page."""
    account = rpc_client.get_account(address)
    if not account:
        return render_template('404.html'), 404

    transactions = rpc_client.get_account_transactions(address, 20) or []

    return render_template('account.html', account=account, transactions=transactions)


@app.route('/validators')
@track_metrics
@cache.cached(timeout=120)
def validators_page():
    """Validators list page."""
    validators = rpc_client.get_validators() or []

    return render_template('validators.html', validators=validators)


@app.route('/validator/<address>')
@track_metrics
@cache.cached(timeout=60, query_string=True)
def validator_detail(address):
    """Validator detail page."""
    validator = rpc_client.get_validator(address)
    if not validator:
        return render_template('404.html'), 404

    return render_template('validator.html', validator=validator)


@app.route('/dex')
@track_metrics
@cache.cached(timeout=30)
def dex_page():
    """DEX overview page."""
    pools_raw = rpc_client.get_dex_pools() or []
    pools = [enrich_pool_metrics(pool) for pool in pools_raw]
    stats = summarize_pools(pools)

    summary_resp = rpc_client.get_dex_analytics_summary() or {}
    analytics_summary = summary_resp.get('summary') if isinstance(summary_resp, dict) else summary_resp
    top_pairs_resp = rpc_client.get_top_trading_pairs(limit=8) or {}
    top_pairs = top_pairs_resp.get('top_pairs') if isinstance(top_pairs_resp, dict) else top_pairs_resp
    trades_resp = rpc_client.get_latest_dex_trades(limit=20) or {}
    recent_trades = trades_resp.get('trades') if isinstance(trades_resp, dict) else trades_resp

    return render_template(
        'dex.html',
        pools=pools,
        stats=stats,
        analytics_summary=analytics_summary or {},
        top_pairs=top_pairs or [],
        recent_trades=(recent_trades or [])[:15],
    )


@app.route('/dex/pool/<pool_id>')
@track_metrics
@cache.cached(timeout=60, query_string=True)
def pool_detail(pool_id):
    """DEX pool detail page."""
    pool = rpc_client.get_dex_pool(pool_id)
    if not pool:
        return render_template('404.html'), 404

    trades = rpc_client.get_pool_trades(pool_id, 20) or []

    return render_template('pool.html', pool=pool, trades=trades)


@app.route('/oracle')
@track_metrics
@cache.cached(timeout=30)
def oracle_page():
    """Oracle prices page."""
    prices = rpc_client.get_oracle_prices() or {}

    return render_template('oracle.html', prices=prices)


@app.route('/compute')
@track_metrics
@cache.cached(timeout=60)
def compute_page():
    """Compute jobs page."""
    jobs = rpc_client.get_compute_jobs(20) or []

    return render_template('compute.html', jobs=jobs)


@app.route('/search')
@track_metrics
def search_page():
    """Search page."""
    query = request.args.get('q', '').strip()

    if not query:
        return render_template('search.html', query='', results=None)

    results = rpc_client.search(query)

    return render_template('search.html', query=query, results=results)


# API endpoints (proxy to indexer with caching)
@app.route('/api/v1/blocks')
@track_metrics
@cache.cached(timeout=30, query_string=True)
def api_blocks():
    """Get blocks."""
    limit = min(
        request.args.get('limit', 20, type=int),
        app.config['MAX_ITEMS_PER_PAGE']
    )

    blocks = rpc_client.get_latest_blocks(limit)
    if blocks is None:
        return jsonify({'error': 'Failed to fetch blocks'}), 500

    CACHE_HITS.labels(endpoint='blocks').inc()
    return jsonify(blocks)


@app.route('/api/v1/blocks/<int:height>')
@track_metrics
@cache.cached(timeout=300, query_string=True)
def api_block(height):
    """Get block by height."""
    block = rpc_client.get_block(height)
    if block is None:
        return jsonify({'error': 'Block not found'}), 404

    CACHE_HITS.labels(endpoint='block').inc()
    return jsonify(block)


@app.route('/api/v1/transactions')
@track_metrics
@cache.cached(timeout=30, query_string=True)
def api_transactions():
    """Get transactions."""
    limit = min(
        request.args.get('limit', 20, type=int),
        app.config['MAX_ITEMS_PER_PAGE']
    )

    transactions = rpc_client.get_latest_transactions(limit)
    if transactions is None:
        return jsonify({'error': 'Failed to fetch transactions'}), 500

    CACHE_HITS.labels(endpoint='transactions').inc()
    return jsonify(transactions)


@app.route('/api/v1/transactions/<tx_hash>')
@track_metrics
@cache.cached(timeout=300, query_string=True)
def api_transaction(tx_hash):
    """Get transaction by hash."""
    transaction = rpc_client.get_transaction(tx_hash)
    if transaction is None:
        return jsonify({'error': 'Transaction not found'}), 404

    CACHE_HITS.labels(endpoint='transaction').inc()
    return jsonify(transaction)


@app.route('/api/v1/stats')
@track_metrics
@cache.cached(timeout=60)
def api_stats():
    """Get network statistics."""
    stats = rpc_client.get_network_stats()
    if stats is None:
        return jsonify({'error': 'Failed to fetch stats'}), 500

    CACHE_HITS.labels(endpoint='stats').inc()
    return jsonify(stats)


@app.route('/api/v1/search')
@track_metrics
def api_search():
    """Search endpoint."""
    query = request.args.get('q', '').strip()

    if not query:
        return jsonify({'error': 'Query parameter required'}), 400

    results = rpc_client.search(query)
    if results is None:
        return jsonify({'error': 'Search failed'}), 500

    return jsonify(results)


# Governance API endpoints
@app.route('/api/governance/proposals')
@track_metrics
@cache.cached(timeout=60, query_string=True)
def api_governance_proposals():
    """Get all governance proposals."""
    status = request.args.get('status')
    status_map = {
        'voting': 'PROPOSAL_STATUS_VOTING_PERIOD',
        'passed': 'PROPOSAL_STATUS_PASSED',
        'rejected': 'PROPOSAL_STATUS_REJECTED',
        'failed': 'PROPOSAL_STATUS_FAILED',
        'deposit': 'PROPOSAL_STATUS_DEPOSIT_PERIOD',
    }
    cosmos_status = status_map.get(status) if status else None

    proposals_data = rpc_client.get_proposals(cosmos_status)
    if proposals_data is None:
        return jsonify({'error': 'Failed to fetch proposals'}), 500

    proposals = proposals_data.get('proposals', [])

    # Enrich with human-readable status
    for p in proposals:
        raw_status = p.get('status', '')
        if 'VOTING' in raw_status:
            p['status_label'] = 'Voting'
        elif 'PASSED' in raw_status:
            p['status_label'] = 'Passed'
        elif 'REJECTED' in raw_status:
            p['status_label'] = 'Rejected'
        elif 'FAILED' in raw_status:
            p['status_label'] = 'Failed'
        elif 'DEPOSIT' in raw_status:
            p['status_label'] = 'Deposit'
        else:
            p['status_label'] = 'Unknown'

    CACHE_HITS.labels(endpoint='governance_proposals').inc()
    return jsonify({'proposals': proposals, 'total': len(proposals)})


@app.route('/api/governance/proposals/<int:proposal_id>')
@track_metrics
@cache.cached(timeout=60, query_string=True)
def api_governance_proposal(proposal_id):
    """Get proposal details."""
    proposal_data = rpc_client.get_proposal(proposal_id)
    if proposal_data is None:
        return jsonify({'error': 'Proposal not found'}), 404

    proposal = proposal_data.get('proposal', {})
    tally_data = rpc_client.get_proposal_tally(proposal_id)
    tally = tally_data.get('tally', {}) if tally_data else {}

    # Get governance params for context
    voting_params = rpc_client.get_gov_params('voting')
    tallying_params = rpc_client.get_gov_params('tallying')

    raw_status = proposal.get('status', '')
    if 'VOTING' in raw_status:
        proposal['status_label'] = 'Voting'
    elif 'PASSED' in raw_status:
        proposal['status_label'] = 'Passed'
    elif 'REJECTED' in raw_status:
        proposal['status_label'] = 'Rejected'
    elif 'FAILED' in raw_status:
        proposal['status_label'] = 'Failed'
    elif 'DEPOSIT' in raw_status:
        proposal['status_label'] = 'Deposit'
    else:
        proposal['status_label'] = 'Unknown'

    CACHE_HITS.labels(endpoint='governance_proposal').inc()
    return jsonify({
        'proposal': proposal,
        'tally': tally,
        'voting_params': voting_params.get('voting_params') if voting_params else None,
        'tallying_params': tallying_params.get('tally_params') if tallying_params else None,
    })


@app.route('/api/governance/proposals/<int:proposal_id>/votes')
@track_metrics
@cache.cached(timeout=30, query_string=True)
def api_governance_proposal_votes(proposal_id):
    """Get votes for a proposal."""
    pagination_key = request.args.get('pagination_key')
    votes_data = rpc_client.get_proposal_votes(proposal_id, pagination_key)
    if votes_data is None:
        return jsonify({'error': 'Failed to fetch votes'}), 500

    votes = votes_data.get('votes', [])

    # Map vote options to labels
    option_map = {
        'VOTE_OPTION_YES': 'Yes',
        'VOTE_OPTION_NO': 'No',
        'VOTE_OPTION_ABSTAIN': 'Abstain',
        'VOTE_OPTION_NO_WITH_VETO': 'NoWithVeto',
    }
    for v in votes:
        for opt in v.get('options', []):
            opt['option_label'] = option_map.get(opt.get('option'), opt.get('option'))

    pagination = votes_data.get('pagination', {})
    CACHE_HITS.labels(endpoint='governance_votes').inc()
    return jsonify({
        'votes': votes,
        'total': len(votes),
        'next_key': pagination.get('next_key'),
    })


# Staking API endpoints
@app.route('/api/staking/pool')
@track_metrics
@cache.cached(timeout=60)
def api_staking_pool():
    """Get staking pool info."""
    pool_data = rpc_client.get_staking_pool()
    if pool_data is None:
        return jsonify({'error': 'Failed to fetch staking pool'}), 500

    params_data = rpc_client.get_staking_params()
    params = params_data.get('params', {}) if params_data else {}

    pool = pool_data.get('pool', {})
    CACHE_HITS.labels(endpoint='staking_pool').inc()
    return jsonify({
        'pool': pool,
        'params': params,
    })


@app.route('/api/staking/delegations/<address>')
@track_metrics
@cache.cached(timeout=30, query_string=True)
def api_staking_delegations(address):
    """Get delegations for an address."""
    delegations_data = rpc_client.get_delegations(address)
    if delegations_data is None:
        return jsonify({'error': 'Failed to fetch delegations'}), 500

    unbonding_data = rpc_client.get_unbonding_delegations(address)
    unbonding = unbonding_data.get('unbonding_responses', []) if unbonding_data else []

    delegations = delegations_data.get('delegation_responses', [])
    CACHE_HITS.labels(endpoint='staking_delegations').inc()
    return jsonify({
        'delegations': delegations,
        'unbonding': unbonding,
        'total': len(delegations),
    })


@app.route('/api/staking/rewards/<address>')
@track_metrics
@cache.cached(timeout=30, query_string=True)
def api_staking_rewards(address):
    """Get pending rewards for an address."""
    rewards_data = rpc_client.get_delegation_rewards(address)
    if rewards_data is None:
        return jsonify({'error': 'Failed to fetch rewards'}), 500

    rewards = rewards_data.get('rewards', [])
    total = rewards_data.get('total', [])
    CACHE_HITS.labels(endpoint='staking_rewards').inc()
    return jsonify({
        'rewards': rewards,
        'total': total,
    })


@app.route('/api/staking/unbonding/<address>')
@track_metrics
@cache.cached(timeout=30, query_string=True)
def api_staking_unbonding(address):
    """Get unbonding delegations for an address."""
    unbonding_data = rpc_client.get_unbonding_delegations(address)
    if unbonding_data is None:
        return jsonify({'error': 'Failed to fetch unbonding delegations'}), 500

    unbonding = unbonding_data.get('unbonding_responses', [])
    CACHE_HITS.labels(endpoint='staking_unbonding').inc()
    return jsonify({
        'unbonding_delegations': unbonding,
        'total': len(unbonding),
    })


# Validators API endpoints
@app.route('/api/validators')
@track_metrics
@cache.cached(timeout=60, query_string=True)
def api_validators_list():
    """Get all validators with sorting."""
    status = request.args.get('status')
    sort_by = request.args.get('sort', 'voting_power')
    order = request.args.get('order', 'desc')

    status_map = {
        'bonded': 'BOND_STATUS_BONDED',
        'unbonded': 'BOND_STATUS_UNBONDED',
        'unbonding': 'BOND_STATUS_UNBONDING',
    }
    cosmos_status = status_map.get(status) if status else None

    validators_data = rpc_client.get_validators_rest(cosmos_status)
    if validators_data is None:
        return jsonify({'error': 'Failed to fetch validators'}), 500

    validators = validators_data.get('validators', [])

    # Enrich validators with computed fields
    for v in validators:
        tokens = int(v.get('tokens', '0'))
        v['voting_power'] = tokens
        v['voting_power_formatted'] = tokens / 1_000_000  # Convert to display units

        commission = v.get('commission', {}).get('commission_rates', {})
        v['commission_rate'] = float(commission.get('rate', '0'))
        v['commission_max_rate'] = float(commission.get('max_rate', '0'))
        v['commission_max_change_rate'] = float(commission.get('max_change_rate', '0'))

        desc = v.get('description', {})
        v['moniker'] = desc.get('moniker', 'Unknown')
        v['identity'] = desc.get('identity', '')
        v['website'] = desc.get('website', '')
        v['details'] = desc.get('details', '')

        raw_status = v.get('status', '')
        if 'BONDED' in raw_status:
            v['status_label'] = 'Active'
        elif 'UNBONDING' in raw_status:
            v['status_label'] = 'Unbonding'
        else:
            v['status_label'] = 'Inactive'

    # Sort validators
    reverse = order == 'desc'
    if sort_by == 'voting_power':
        validators.sort(key=lambda x: x.get('voting_power', 0), reverse=reverse)
    elif sort_by == 'commission':
        validators.sort(key=lambda x: x.get('commission_rate', 0), reverse=reverse)
    elif sort_by == 'moniker':
        validators.sort(key=lambda x: x.get('moniker', '').lower(), reverse=reverse)

    # Add rank
    for i, v in enumerate(validators):
        v['rank'] = i + 1

    CACHE_HITS.labels(endpoint='validators_list').inc()
    return jsonify({
        'validators': validators,
        'total': len(validators),
    })


@app.route('/api/validators/<address>')
@track_metrics
@cache.cached(timeout=60, query_string=True)
def api_validator_detail(address):
    """Get validator details."""
    validator_data = rpc_client.get_validator_rest(address)
    if validator_data is None:
        return jsonify({'error': 'Validator not found'}), 404

    validator = validator_data.get('validator', {})

    # Get commission info
    commission_data = rpc_client.get_validator_commission(address)
    commission = commission_data.get('commission', {}) if commission_data else {}

    # Get delegations to this validator
    delegations_data = rpc_client.get_validator_delegations(address)
    delegations = delegations_data.get('delegation_responses', []) if delegations_data else []

    # Enrich validator
    tokens = int(validator.get('tokens', '0'))
    validator['voting_power'] = tokens
    validator['voting_power_formatted'] = tokens / 1_000_000

    commission_rates = validator.get('commission', {}).get('commission_rates', {})
    validator['commission_rate'] = float(commission_rates.get('rate', '0'))
    validator['commission_max_rate'] = float(commission_rates.get('max_rate', '0'))
    validator['commission_max_change_rate'] = float(commission_rates.get('max_change_rate', '0'))

    desc = validator.get('description', {})
    validator['moniker'] = desc.get('moniker', 'Unknown')
    validator['identity'] = desc.get('identity', '')
    validator['website'] = desc.get('website', '')
    validator['details'] = desc.get('details', '')
    validator['security_contact'] = desc.get('security_contact', '')

    raw_status = validator.get('status', '')
    if 'BONDED' in raw_status:
        validator['status_label'] = 'Active'
    elif 'UNBONDING' in raw_status:
        validator['status_label'] = 'Unbonding'
    else:
        validator['status_label'] = 'Inactive'

    CACHE_HITS.labels(endpoint='validator_detail').inc()
    return jsonify({
        'validator': validator,
        'commission_earned': commission,
        'delegations': delegations[:50],  # Limit to 50 delegators
        'delegator_count': len(delegations),
    })


# Rich List endpoint
@app.route('/api/v1/richlist')
@track_metrics
@cache.cached(timeout=600)  # Cache for 10 minutes
def api_richlist():
    """
    Get top token holders (Rich List)
    ---
    tags:
      - Rich List
    parameters:
      - name: limit
        in: query
        type: integer
        default: 100
        description: Number of top holders to return (max 500)
      - name: denom
        in: query
        type: string
        default: upaw
        description: Token denomination to query
    responses:
      200:
        description: Top token holders
        schema:
          type: object
          properties:
            richlist:
              type: array
              items:
                type: object
                properties:
                  rank:
                    type: integer
                  address:
                    type: string
                  balance:
                    type: string
                  percentage:
                    type: number
            total_supply:
              type: string
            total_holders:
              type: integer
            last_updated:
              type: string
      500:
        description: Failed to fetch rich list
    """
    limit = min(request.args.get('limit', 100, type=int), 500)
    denom = request.args.get('denom', 'upaw')

    # Get all balances sorted by amount
    balances_data = rpc_client.get_all_balances(limit)

    # Get total supply
    supply_data = rpc_client.get_total_supply(denom)
    total_supply = 0
    if supply_data and 'amount' in supply_data:
        try:
            total_supply = int(supply_data['amount'].get('amount', 0))
        except (TypeError, ValueError):
            total_supply = 0

    # Build rich list
    richlist = []
    if balances_data:
        accounts = balances_data if isinstance(balances_data, list) else balances_data.get('accounts', [])
        for idx, account in enumerate(accounts[:limit], 1):
            balance = 0
            if isinstance(account, dict):
                balance_value = account.get('balance') or account.get('amount', 0)
                if isinstance(balance_value, dict):
                    balance = int(balance_value.get('amount', 0))
                else:
                    try:
                        balance = int(balance_value)
                    except (TypeError, ValueError):
                        balance = 0

            percentage = (balance / total_supply * 100) if total_supply > 0 else 0

            richlist.append({
                'rank': idx,
                'address': account.get('address', ''),
                'balance': str(balance),
                'percentage': round(percentage, 4)
            })

    CACHE_HITS.labels(endpoint='richlist').inc()
    return jsonify({
        'richlist': richlist,
        'total_supply': str(total_supply),
        'total_holders': len(richlist),
        'denom': denom,
        'last_updated': datetime.utcnow().isoformat()
    })


# CSV Export endpoints
@app.route('/api/v1/export/transactions/<address>')
@track_metrics
def api_export_transactions(address):
    """
    Export account transaction history as CSV
    ---
    tags:
      - Export
    parameters:
      - name: address
        in: path
        type: string
        required: true
        description: Account address
      - name: format
        in: query
        type: string
        enum: [csv, json]
        default: csv
        description: Export format
      - name: limit
        in: query
        type: integer
        default: 1000
        description: Maximum transactions to export
    responses:
      200:
        description: Transaction data in requested format
      404:
        description: Account not found
      500:
        description: Failed to export transactions
    """
    export_format = request.args.get('format', 'csv').lower()
    limit = min(request.args.get('limit', 1000, type=int), 5000)

    transactions = rpc_client.get_account_transactions_all(address, limit)

    if transactions is None:
        return jsonify({'error': 'Failed to fetch transactions'}), 500

    tx_list = transactions if isinstance(transactions, list) else transactions.get('transactions', [])

    if export_format == 'json':
        return jsonify({
            'address': address,
            'transactions': tx_list,
            'count': len(tx_list),
            'exported_at': datetime.utcnow().isoformat()
        })

    # Generate CSV
    output = io.StringIO()
    writer = csv.writer(output)

    writer.writerow([
        'TxHash', 'Timestamp', 'Type', 'From', 'To', 'Amount', 'Denom', 'Fee', 'FeeDenom', 'Status', 'BlockHeight', 'Memo'
    ])

    for tx in tx_list:
        tx_hash = tx.get('hash', '')
        timestamp = tx.get('timestamp', '')
        tx_type = tx.get('type', '')
        sender = tx.get('sender', '') or tx.get('from_address', '')
        receiver = tx.get('receiver', '') or tx.get('to_address', '')

        amount = ''
        amount_denom = ''
        messages = tx.get('messages', [])
        if messages and isinstance(messages, list) and len(messages) > 0:
            msg = messages[0]
            if isinstance(msg, dict):
                amount_info = msg.get('amount', {})
                if isinstance(amount_info, dict):
                    amount = amount_info.get('amount', '')
                    amount_denom = amount_info.get('denom', '')
                elif isinstance(amount_info, list) and len(amount_info) > 0:
                    amount = amount_info[0].get('amount', '')
                    amount_denom = amount_info[0].get('denom', '')

        fee = tx.get('fee_amount', '') or tx.get('fee', '')
        fee_denom = tx.get('fee_denom', '')
        status = tx.get('status', '')
        block_height = tx.get('block_height', '')
        memo = tx.get('memo', '')

        writer.writerow([
            tx_hash, timestamp, tx_type, sender, receiver, amount, amount_denom, fee, fee_denom, status, block_height, memo
        ])

    output.seek(0)
    response = make_response(output.getvalue())
    response.headers['Content-Type'] = 'text/csv'
    response.headers['Content-Disposition'] = f'attachment; filename=paw_transactions_{address}_{datetime.utcnow().strftime("%Y%m%d")}.csv'

    return response


@app.route('/api/v1/export/account/<address>')
@track_metrics
def api_export_account(address):
    """
    Export account summary as CSV
    ---
    tags:
      - Export
    parameters:
      - name: address
        in: path
        type: string
        required: true
        description: Account address
      - name: format
        in: query
        type: string
        enum: [csv, json]
        default: csv
        description: Export format
    responses:
      200:
        description: Account summary in requested format
      404:
        description: Account not found
      500:
        description: Failed to export account
    """
    export_format = request.args.get('format', 'csv').lower()

    account = rpc_client.get_account(address)

    if account is None:
        return jsonify({'error': 'Account not found'}), 404

    account_data = account if isinstance(account, dict) and 'account' not in account else account.get('account', account)

    if export_format == 'json':
        return jsonify({
            'address': address,
            'account': account_data,
            'exported_at': datetime.utcnow().isoformat()
        })

    output = io.StringIO()
    writer = csv.writer(output)

    writer.writerow(['Field', 'Value'])
    writer.writerow(['Address', address])
    writer.writerow(['Total Transactions', account_data.get('tx_count', 0)])
    writer.writerow(['Total Received', account_data.get('total_received', '0')])
    writer.writerow(['Total Sent', account_data.get('total_sent', '0')])
    writer.writerow(['First Seen Block', account_data.get('first_seen_height', '')])
    writer.writerow(['First Seen At', account_data.get('first_seen_at', '')])
    writer.writerow(['Last Seen Block', account_data.get('last_seen_height', '')])
    writer.writerow(['Last Seen At', account_data.get('last_seen_at', '')])
    writer.writerow(['Export Date', datetime.utcnow().isoformat()])

    output.seek(0)
    response = make_response(output.getvalue())
    response.headers['Content-Type'] = 'text/csv'
    response.headers['Content-Disposition'] = f'attachment; filename=paw_account_{address}_{datetime.utcnow().strftime("%Y%m%d")}.csv'

    return response


# Template filters
@app.template_filter('timestamp')
def format_timestamp(value):
    """Format timestamp for display."""
    if not value:
        return 'N/A'

    try:
        dt = datetime.fromisoformat(value.replace('Z', '+00:00'))
        return dt.strftime('%Y-%m-%d %H:%M:%S UTC')
    except Exception:
        return value


@app.template_filter('timeago')
def timeago(value):
    """Format timestamp as relative time."""
    if not value:
        return 'N/A'

    try:
        dt = datetime.fromisoformat(value.replace('Z', '+00:00'))
        now = datetime.utcnow()
        diff = now - dt

        if diff.days > 0:
            return f"{diff.days} day{'s' if diff.days != 1 else ''} ago"
        elif diff.seconds > 3600:
            hours = diff.seconds // 3600
            return f"{hours} hour{'s' if hours != 1 else ''} ago"
        elif diff.seconds > 60:
            minutes = diff.seconds // 60
            return f"{minutes} minute{'s' if minutes != 1 else ''} ago"
        else:
            return f"{diff.seconds} second{'s' if diff.seconds != 1 else ''} ago"
    except Exception:
        return value


@app.template_filter('shorten')
def shorten_hash(value, length=8):
    """Shorten hash for display."""
    if not value or len(value) <= length * 2:
        return value
    return f"{value[:length]}...{value[-length:]}"


@app.template_filter('number')
def format_number(value):
    """Format number with thousand separators."""
    try:
        return f"{int(value):,}"
    except (ValueError, TypeError):
        return value


if __name__ == '__main__':
    # Run development server
    app.run(
        host='0.0.0.0',
        port=int(os.getenv('FLASK_PORT', 5000)),
        debug=os.getenv('FLASK_DEBUG', 'false').lower() == 'true'
    )
