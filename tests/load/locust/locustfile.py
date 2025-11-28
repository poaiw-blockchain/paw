"""
PAW Blockchain Load Testing with Locust

This module provides comprehensive load testing scenarios for the PAW blockchain,
including REST API queries, transaction submissions, and DEX operations.
"""

from locust import HttpUser, TaskSet, task, between, events
import json
import random
import time
from datetime import datetime

# Test configuration
TEST_ADDRESSES = [
    'paw1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq0d8t4q',
    'paw1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqg3vxq7',
    'paw1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqa29wr0',
    'paw1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqcjqxn9',
]

TOKEN_PAIRS = [
    {'tokenA': 'upaw', 'tokenB': 'uatom'},
    {'tokenA': 'upaw', 'tokenB': 'uosmo'},
    {'tokenA': 'uatom', 'tokenB': 'uosmo'},
]

# Custom event handlers for detailed reporting
@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    print(f"Load test starting at {datetime.now().isoformat()}")
    print(f"Target host: {environment.host}")

@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    print(f"Load test completed at {datetime.now().isoformat()}")
    print(f"Total requests: {environment.stats.total.num_requests}")
    print(f"Total failures: {environment.stats.total.num_failures}")


class BlockchainTasks(TaskSet):
    """Basic blockchain query tasks"""

    @task(10)
    def query_balance(self):
        """Query account balance - most frequent operation"""
        address = random.choice(TEST_ADDRESSES)
        with self.client.get(
            f"/cosmos/bank/v1beta1/balances/{address}",
            catch_response=True,
            name="/cosmos/bank/v1beta1/balances/[address]"
        ) as response:
            if response.status_code == 200:
                try:
                    data = response.json()
                    if 'balances' in data:
                        response.success()
                    else:
                        response.failure("Missing 'balances' in response")
                except json.JSONDecodeError:
                    response.failure("Invalid JSON response")
            else:
                response.failure(f"Got status code {response.status_code}")

    @task(5)
    def query_account(self):
        """Query account information"""
        address = random.choice(TEST_ADDRESSES)
        with self.client.get(
            f"/cosmos/auth/v1beta1/accounts/{address}",
            catch_response=True,
            name="/cosmos/auth/v1beta1/accounts/[address]"
        ) as response:
            if response.status_code in [200, 404]:  # 404 is OK if account doesn't exist
                response.success()
            else:
                response.failure(f"Unexpected status code {response.status_code}")

    @task(3)
    def query_validators(self):
        """Query validator set"""
        with self.client.get(
            "/cosmos/staking/v1beta1/validators",
            catch_response=True
        ) as response:
            if response.status_code == 200:
                try:
                    data = response.json()
                    if 'validators' in data:
                        response.success()
                    else:
                        response.failure("Missing 'validators' in response")
                except json.JSONDecodeError:
                    response.failure("Invalid JSON response")

    @task(2)
    def query_node_info(self):
        """Query node information"""
        self.client.get("/cosmos/base/tendermint/v1beta1/node_info")

    @task(1)
    def submit_transaction(self):
        """Submit a bank send transaction"""
        from_addr = random.choice(TEST_ADDRESSES)
        to_addr = random.choice([a for a in TEST_ADDRESSES if a != from_addr])

        tx_payload = {
            "tx": {
                "body": {
                    "messages": [{
                        "@type": "/cosmos.bank.v1beta1.MsgSend",
                        "from_address": from_addr,
                        "to_address": to_addr,
                        "amount": [{
                            "denom": "upaw",
                            "amount": str(random.randint(100, 10000))
                        }]
                    }],
                    "memo": f"locust-test-{int(time.time())}",
                    "timeout_height": "0",
                },
                "auth_info": {
                    "signer_infos": [],
                    "fee": {
                        "amount": [{"denom": "upaw", "amount": "5000"}],
                        "gas_limit": "200000",
                    }
                },
                "signatures": []
            },
            "mode": "BROADCAST_MODE_ASYNC"
        }

        with self.client.post(
            "/cosmos/tx/v1beta1/txs",
            json=tx_payload,
            catch_response=True
        ) as response:
            # Accept both 200 (success) and 400 (invalid signature expected)
            if response.status_code in [200, 400]:
                response.success()
            else:
                response.failure(f"Unexpected status code {response.status_code}")


class DEXTasks(TaskSet):
    """DEX-specific operations"""

    @task(10)
    def query_all_pools(self):
        """Query all DEX pools"""
        with self.client.get(
            "/paw/dex/v1/pools",
            catch_response=True
        ) as response:
            if response.status_code == 200:
                try:
                    data = response.json()
                    if 'pools' in data:
                        response.success()
                    else:
                        response.failure("Missing 'pools' in response")
                except json.JSONDecodeError:
                    response.failure("Invalid JSON response")

    @task(5)
    def query_specific_pool(self):
        """Query a specific pool by ID"""
        pool_id = random.randint(1, 10)
        with self.client.get(
            f"/paw/dex/v1/pools/{pool_id}",
            catch_response=True,
            name="/paw/dex/v1/pools/[id]"
        ) as response:
            # Pool might not exist, so 404 is acceptable
            if response.status_code in [200, 404]:
                response.success()

    @task(3)
    def query_pool_params(self):
        """Query DEX module parameters"""
        self.client.get("/paw/dex/v1/params")

    @task(2)
    def simulate_swap(self):
        """Simulate a token swap"""
        pair = random.choice(TOKEN_PAIRS)
        sender = random.choice(TEST_ADDRESSES)

        swap_payload = {
            "tx": {
                "body": {
                    "messages": [{
                        "@type": "/paw.dex.v1.MsgSwapExactAmountIn",
                        "sender": sender,
                        "routes": [{
                            "pool_id": "1",
                            "token_out_denom": pair['tokenB'],
                        }],
                        "token_in": {
                            "denom": pair['tokenA'],
                            "amount": str(random.randint(1000, 100000))
                        },
                        "token_out_min_amount": "1"
                    }],
                    "memo": f"dex-locust-test-{int(time.time())}",
                },
                "auth_info": {
                    "fee": {
                        "amount": [{"denom": "upaw", "amount": "10000"}],
                        "gas_limit": "300000",
                    }
                },
                "signatures": []
            },
            "mode": "BROADCAST_MODE_ASYNC"
        }

        with self.client.post(
            "/cosmos/tx/v1beta1/txs",
            json=swap_payload,
            catch_response=True,
            name="DEX Swap Transaction"
        ) as response:
            if response.status_code in [200, 400]:
                response.success()


class PAWUser(HttpUser):
    """
    Simulates a general PAW blockchain user
    Performs mixed operations: queries and transactions
    """
    wait_time = between(1, 3)  # Wait 1-3 seconds between tasks
    tasks = [BlockchainTasks]


class DEXTrader(HttpUser):
    """
    Simulates a DEX trader
    Focused on DEX operations
    """
    wait_time = between(2, 5)  # DEX operations might be less frequent
    tasks = [DEXTasks]


class HeavyUser(HttpUser):
    """
    Simulates a heavy user with mixed workload
    80% queries, 20% DEX operations
    """
    wait_time = between(1, 2)

    tasks = {
        BlockchainTasks: 8,
        DEXTasks: 2
    }


# Custom load shape for advanced scenarios
from locust import LoadTestShape

class StepLoadShape(LoadTestShape):
    """
    A load test shape that increases load in steps
    """
    step_time = 60  # seconds
    step_load = 20  # users per step
    spawn_rate = 5
    time_limit = 600  # 10 minutes total

    def tick(self):
        run_time = self.get_run_time()

        if run_time > self.time_limit:
            return None

        current_step = run_time // self.step_time
        return (current_step * self.step_load, self.spawn_rate)


class WaveLoadShape(LoadTestShape):
    """
    A load test shape that creates waves of traffic
    Simulates realistic usage patterns with peaks and valleys
    """
    def tick(self):
        run_time = self.get_run_time()

        if run_time > 600:  # 10 minutes
            return None

        # Create wave pattern using sine function
        import math
        user_count = int(50 + 50 * math.sin(run_time / 60))  # Oscillate between 0-100 users

        return (user_count, 10)
