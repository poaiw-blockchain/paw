#!/usr/bin/env python3
"""
PAW Blockchain - Connect to Network Example

This example demonstrates how to connect to the PAW blockchain network
and retrieve basic network information.

Usage:
    python connect.py

Environment Variables:
    PAW_RPC_ENDPOINT - RPC endpoint URL (default: http://localhost:26657)
"""

import os
import sys
import asyncio
from datetime import datetime
from typing import Dict, Any
from dotenv import load_dotenv
import requests

# Load environment variables
load_dotenv()

# Configuration
RPC_ENDPOINT = os.getenv('PAW_RPC_ENDPOINT', 'http://localhost:26657')


class PAWClient:
    """Simple PAW blockchain client"""

    def __init__(self, rpc_endpoint: str):
        self.rpc_endpoint = rpc_endpoint.rstrip('/')

    def _rpc_call(self, method: str, params: list = None) -> Dict[str, Any]:
        """Make JSON-RPC call to the node"""
        payload = {
            'jsonrpc': '2.0',
            'id': 1,
            'method': method,
            'params': params or []
        }

        response = requests.post(self.rpc_endpoint, json=payload, timeout=10)
        response.raise_for_status()
        result = response.json()

        if 'error' in result:
            raise Exception(f"RPC Error: {result['error']}")

        return result.get('result', {})

    def get_status(self) -> Dict[str, Any]:
        """Get node status"""
        return self._rpc_call('status')

    def get_block(self, height: int = None) -> Dict[str, Any]:
        """Get block at specific height (latest if None)"""
        params = {'height': str(height)} if height else None
        return self._rpc_call('block', params)

    def get_blockchain_info(self, min_height: int, max_height: int) -> Dict[str, Any]:
        """Get blockchain information"""
        return self._rpc_call('blockchain', [str(min_height), str(max_height)])


def connect_to_network() -> Dict[str, Any]:
    """
    Connect to PAW network and display network information

    Returns:
        Dict with connection results
    """
    print('Connecting to PAW Network...')
    print(f'RPC Endpoint: {RPC_ENDPOINT}\n')

    try:
        # Create client
        client = PAWClient(RPC_ENDPOINT)

        # Get node status
        status = client.get_status()
        print('✓ Successfully connected to PAW network\n')

        # Extract information
        node_info = status.get('node_info', {})
        sync_info = status.get('sync_info', {})

        chain_id = node_info.get('network', 'unknown')
        height = int(sync_info.get('latest_block_height', 0))
        catching_up = sync_info.get('catching_up', False)

        print(f'Chain ID: {chain_id}')
        print(f'Current Block Height: {height}')
        print(f'Node Version: {node_info.get("version", "unknown")}')
        print(f'Syncing: {"Yes" if catching_up else "No"}')

        # Get latest block info
        block = client.get_block()
        block_data = block.get('block', {})
        header = block_data.get('header', {})

        print('\nLatest Block Info:')
        print(f'  Block Hash: {block.get("block_id", {}).get("hash", "N/A")}')
        print(f'  Time: {header.get("time", "N/A")}')
        print(f'  Num Transactions: {len(block_data.get("data", {}).get("txs", []))}')

        # Calculate average block time
        if height > 5:
            prev_block = client.get_block(height - 5)
            prev_time = datetime.fromisoformat(
                prev_block['block']['header']['time'].replace('Z', '+00:00')
            )
            curr_time = datetime.fromisoformat(
                header['time'].replace('Z', '+00:00')
            )
            time_diff = (curr_time - prev_time).total_seconds()
            avg_block_time = time_diff / 5
            print(f'  Average Block Time: {avg_block_time:.2f}s')

        print('\n✓ Network information retrieved successfully')

        return {
            'success': True,
            'chain_id': chain_id,
            'height': height,
            'block_hash': block.get('block_id', {}).get('hash', '')
        }

    except requests.exceptions.ConnectionError:
        print('✗ Connection refused')
        print('\nTroubleshooting:')
        print('  1. Check if the RPC endpoint is correct')
        print('  2. Ensure the node is running')
        print('  3. Check firewall settings')
        return {
            'success': False,
            'error': 'Connection refused'
        }

    except Exception as e:
        print(f'✗ Error connecting to network: {str(e)}')
        return {
            'success': False,
            'error': str(e)
        }


def main():
    """Main entry point"""
    result = connect_to_network()
    sys.exit(0 if result['success'] else 1)


if __name__ == '__main__':
    main()
