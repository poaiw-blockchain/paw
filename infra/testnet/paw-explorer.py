#!/usr/bin/env python3
"""PAW Blockchain Explorer - Standalone Backend"""

import os
import logging
import time
import requests
from flask import Flask, jsonify, request
from flask_cors import CORS
from flask_caching import Cache

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = Flask(__name__)
CORS(app)

app.config["CACHE_TYPE"] = "simple"
app.config["CACHE_DEFAULT_TIMEOUT"] = 60
cache = Cache(app)

RPC_URL = os.getenv("RPC_URL", "http://localhost:26657")
API_URL = os.getenv("API_URL", "http://localhost:1317")
CHAIN_ID = "paw-testnet-1"


def rpc_call(method, params=None):
    try:
        response = requests.get(f"{RPC_URL}/{method}", params=params, timeout=10)
        return response.json() if response.ok else None
    except Exception as e:
        logger.error(f"RPC error: {e}")
        return None


def rpc_tx_search(limit, offset):
    page = max(1, (offset // limit) + 1)
    params = {
        "query": "\"tx.height>0\"",
        "prove": "false",
        "page": str(page),
        "per_page": str(limit),
        "order_by": "\"desc\""
    }
    return rpc_call("tx_search", params)


@app.route("/health")
def health():
    status = rpc_call("status")
    return jsonify({
        "status": "healthy" if status else "unhealthy",
        "node": {"reachable": status is not None, "rpc": RPC_URL},
        "chain_id": CHAIN_ID,
        "timestamp": time.time()
    })


@app.route("/api/status")
@cache.cached(timeout=5)
def get_status():
    status = rpc_call("status")
    if not status:
        return jsonify({"error": "Node unreachable"}), 503
    info = status.get("result", {})
    return jsonify({
        "chain_id": info.get("node_info", {}).get("network"),
        "latest_height": int(info.get("sync_info", {}).get("latest_block_height", 0)),
        "latest_block_time": info.get("sync_info", {}).get("latest_block_time"),
        "catching_up": info.get("sync_info", {}).get("catching_up", False)
    })


@app.route("/api/stats")
@cache.cached(timeout=10)
def get_stats():
    status = rpc_call("status")
    if not status:
        return jsonify({"error": "Node unreachable"}), 503

    info = status.get("result", {})
    latest_height = int(info.get("sync_info", {}).get("latest_block_height", 0))
    latest_block_time = info.get("sync_info", {}).get("latest_block_time")

    validators = rpc_call("validators", {"page": "1", "per_page": "200"})
    validator_list = validators.get("result", {}).get("validators", []) if validators else []

    tx_search = rpc_tx_search(1, 0)
    total_txs = 0
    if tx_search:
        total_txs = int(tx_search.get("result", {}).get("total_count", 0))

    return jsonify({
        "latest_block": latest_height,
        "latest_block_time": latest_block_time,
        "total_txs": total_txs,
        "active_validators": len(validator_list)
    })


@app.route("/api/blocks")
@cache.cached(timeout=10)
def get_blocks():
    limit = min(int(request.args.get("limit", 20)), 100)
    status = rpc_call("status")
    if not status:
        return jsonify({"error": "Node unreachable"}), 503
    latest = int(status["result"]["sync_info"]["latest_block_height"])
    blocks = []
    for h in range(latest, max(latest - limit, 0), -1):
        block = rpc_call("block", {"height": h})
        if block and "result" in block:
            b = block["result"]["block"]
            blocks.append({
                "height": int(b["header"]["height"]),
                "hash": block["result"]["block_id"]["hash"],
                "time": b["header"]["time"],
                "txs_count": len(b.get("data", {}).get("txs", []))
            })
    return jsonify({"blocks": blocks, "latest_height": latest})


@app.route("/api/block/<height>")
@cache.cached(timeout=300)
def get_block(height):
    block = rpc_call("block", {"height": height})
    if not block or "result" not in block:
        return jsonify({"error": "Block not found"}), 404
    b = block["result"]["block"]
    return jsonify({
        "height": int(b["header"]["height"]),
        "hash": block["result"]["block_id"]["hash"],
        "time": b["header"]["time"],
        "proposer": b["header"]["proposer_address"],
        "txs_count": len(b.get("data", {}).get("txs", [])),
        "txs": b.get("data", {}).get("txs", [])
    })


@app.route("/api/transactions")
@cache.cached(timeout=10, query_string=True)
def get_transactions():
    limit = min(int(request.args.get("limit", 20)), 100)
    offset = max(int(request.args.get("offset", 0)), 0)

    tx_search = rpc_tx_search(limit, offset)
    if not tx_search or "result" not in tx_search:
        return jsonify({"error": "Failed to fetch transactions"}), 503

    txs = []
    height_times = {}
    for tx in tx_search.get("result", {}).get("txs", []):
        height = int(tx.get("height", 0))
        if height not in height_times:
            block = rpc_call("block", {"height": height})
            if block and "result" in block:
                height_times[height] = block["result"]["block"]["header"]["time"]
        txs.append({
            "hash": tx.get("hash"),
            "height": height,
            "time": height_times.get(height),
            "status": "success" if tx.get("tx_result", {}).get("code", 0) == 0 else "failed",
            "code": tx.get("tx_result", {}).get("code", 0),
            "gas_wanted": tx.get("tx_result", {}).get("gas_wanted"),
            "gas_used": tx.get("tx_result", {}).get("gas_used")
        })

    total = int(tx_search.get("result", {}).get("total_count", len(txs)))
    return jsonify({"transactions": txs, "total": total})


@app.route("/api/validators")
@cache.cached(timeout=60)
def get_validators():
    result = rpc_call("validators")
    if not result:
        return jsonify({"error": "Failed to get validators"}), 503
    return jsonify(result.get("result", {}))


@app.route("/api/tx/<txhash>")
@cache.cached(timeout=300)
def get_tx(txhash):
    result = rpc_call("tx", {"hash": f"0x{txhash}"})
    if not result or "error" in result:
        return jsonify({"error": "Transaction not found"}), 404
    return jsonify(result.get("result", {}))


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8082)
