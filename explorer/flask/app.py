#!/usr/bin/env python3
"""
PAW Blockchain Explorer - Flask Application

A production-ready blockchain explorer with RPC integration,
real-time updates, and comprehensive blockchain data visualization.
"""

import os
import logging
import time
import math
from collections import defaultdict
from datetime import datetime, timedelta
from functools import wraps
from typing import Dict, List, Optional, Any

import requests
from flask import Flask, render_template, jsonify, request, Response
from flask_caching import Cache
from flask_cors import CORS
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

    def get_latest_blocks(self, limit: int = 20) -> Optional[List[Dict]]:
        """Get latest blocks from indexer."""
        url = f"{self.indexer_url}/api/v1/blocks/latest"
        return self._make_request(url, params={'limit': limit})

    def get_block(self, height: int) -> Optional[Dict]:
        """Get block by height."""
        url = f"{self.indexer_url}/api/v1/blocks/{height}"
        return self._make_request(url)

    def get_block_transactions(self, height: int) -> Optional[List[Dict]]:
        """Get transactions for a block."""
        url = f"{self.indexer_url}/api/v1/blocks/{height}/transactions"
        return self._make_request(url)

    def get_latest_transactions(self, limit: int = 20) -> Optional[List[Dict]]:
        """Get latest transactions."""
        url = f"{self.indexer_url}/api/v1/transactions/latest"
        return self._make_request(url, params={'limit': limit})

    def get_transaction(self, tx_hash: str) -> Optional[Dict]:
        """Get transaction by hash."""
        url = f"{self.indexer_url}/api/v1/transactions/{tx_hash}"
        return self._make_request(url)

    def get_account(self, address: str) -> Optional[Dict]:
        """Get account information."""
        url = f"{self.indexer_url}/api/v1/accounts/{address}"
        return self._make_request(url)

    def get_account_transactions(self, address: str, limit: int = 20) -> Optional[List[Dict]]:
        """Get transactions for an account."""
        url = f"{self.indexer_url}/api/v1/accounts/{address}/transactions"
        return self._make_request(url, params={'limit': limit})

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
        """Get network statistics."""
        url = f"{self.indexer_url}/api/v1/stats"
        return self._make_request(url)

    def search(self, query: str) -> Optional[Dict]:
        """Search for blocks, transactions, or accounts."""
        url = f"{self.indexer_url}/api/v1/search"
        return self._make_request(url, params={'q': query})

    def get_rpc_status(self) -> Optional[Dict]:
        """Get RPC node status."""
        url = f"{self.rpc_url}/status"
        return self._make_request(url)

    def get_rpc_health(self) -> Optional[Dict]:
        """Get RPC health status."""
        url = f"{self.rpc_url}/health"
        return self._make_request(url)


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
