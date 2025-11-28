# Python API Examples

Complete Python examples for interacting with the PAW Blockchain API.

## Installation

```bash
pip install requests cosmpy
```

## Basic Client

```python
import requests
from typing import Dict, List, Optional
import time

class PAWClient:
    """PAW Blockchain API Client"""

    def __init__(self, base_url: str = "http://localhost:1317"):
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers.update({"Content-Type": "application/json"})

    def get(self, endpoint: str, params: Optional[Dict] = None) -> Dict:
        """Make GET request"""
        response = self.session.get(f"{self.base_url}{endpoint}", params=params)
        response.raise_for_status()
        return response.json()

    def post(self, endpoint: str, data: Dict) -> Dict:
        """Make POST request"""
        response = self.session.post(f"{self.base_url}{endpoint}", json=data)
        response.raise_for_status()
        return response.json()

# Initialize client
client = PAWClient()
```

## DEX Module

```python
def list_pools(limit: int = 100, offset: int = 0) -> List[Dict]:
    """List all DEX pools"""
    response = client.get("/paw/dex/v1/pools", {
        "pagination.limit": limit,
        "pagination.offset": offset
    })
    return response["pools"]

def get_pool(pool_id: int) -> Dict:
    """Get specific pool by ID"""
    response = client.get(f"/paw/dex/v1/pools/{pool_id}")
    return response["pool"]

def estimate_swap(pool_id: int, token_in: str, amount_in: int) -> Dict:
    """Estimate swap output"""
    response = client.post("/paw/dex/v1/estimate_swap", {
        "pool_id": pool_id,
        "token_in": token_in,
        "amount_in": str(amount_in)
    })
    return response

def create_pool(creator: str, token_a: str, token_b: str,
                amount_a: int, amount_b: int) -> Dict:
    """Create new liquidity pool"""
    return client.post("/paw/dex/v1/create_pool", {
        "creator": creator,
        "token_a": token_a,
        "token_b": token_b,
        "amount_a": str(amount_a),
        "amount_b": str(amount_b)
    })

# Example usage
pools = list_pools()
for pool in pools:
    print(f"Pool {pool['id']}: {pool['token_a']}/{pool['token_b']}")
```

## Oracle Module

```python
from urllib.parse import quote

def list_price_feeds() -> List[Dict]:
    """List all price feeds"""
    response = client.get("/paw/oracle/v1/prices")
    return response["price_feeds"]

def get_price(asset: str) -> Dict:
    """Get price for specific asset"""
    encoded_asset = quote(asset)
    response = client.get(f"/paw/oracle/v1/prices/{encoded_asset}")
    return response["price_feed"]

def monitor_prices(assets: List[str], interval: int = 5):
    """Monitor price updates"""
    while True:
        for asset in assets:
            try:
                feed = get_price(asset)
                print(f"{asset}: ${feed['price']}")
            except Exception as e:
                print(f"Error fetching {asset}: {e}")
        time.sleep(interval)

# Example
btc_price = get_price("BTC/USD")
print(f"BTC Price: ${btc_price['price']}")
```

## Compute Module

```python
def list_tasks(status: Optional[str] = None,
               requester: Optional[str] = None) -> List[Dict]:
    """List compute tasks"""
    params = {}
    if status:
        params["status"] = status
    if requester:
        params["requester"] = requester

    response = client.get("/paw/compute/v1/tasks", params)
    return response["tasks"]

def get_task(task_id: int) -> Dict:
    """Get task by ID"""
    response = client.get(f"/paw/compute/v1/tasks/{task_id}")
    return response["task"]

def submit_task(requester: str, task_type: str,
                task_data: Dict, fee: Dict) -> Dict:
    """Submit compute task"""
    return client.post("/paw/compute/v1/submit_task", {
        "requester": requester,
        "task_type": task_type,
        "task_data": task_data,
        "fee": fee
    })
```

## Bank Module

```python
def get_balance(address: str) -> List[Dict]:
    """Get account balances"""
    response = client.get(f"/cosmos/bank/v1beta1/balances/{address}")
    return response["balances"]

def get_balance_by_denom(address: str, denom: str) -> int:
    """Get balance for specific denomination"""
    balances = get_balance(address)
    for balance in balances:
        if balance["denom"] == denom:
            return int(balance["amount"])
    return 0

def send_tokens(from_address: str, to_address: str,
                amount: int, denom: str = "uapaw") -> Dict:
    """Send tokens"""
    return client.post("/cosmos/bank/v1beta1/send", {
        "from_address": from_address,
        "to_address": to_address,
        "amount": [{
            "denom": denom,
            "amount": str(amount)
        }]
    })
```

## Staking Module

```python
def list_validators(status: Optional[str] = None) -> List[Dict]:
    """List validators"""
    params = {"status": status} if status else {}
    response = client.get("/cosmos/staking/v1beta1/validators", params)
    return response["validators"]

def get_validator(validator_address: str) -> Dict:
    """Get validator details"""
    response = client.get(f"/cosmos/staking/v1beta1/validators/{validator_address}")
    return response["validator"]

def get_delegations(delegator_address: str) -> List[Dict]:
    """Get delegations for address"""
    response = client.get(f"/cosmos/staking/v1beta1/delegations/{delegator_address}")
    return response["delegation_responses"]

def delegate(delegator_address: str, validator_address: str,
             amount: int, denom: str = "uapaw") -> Dict:
    """Delegate tokens"""
    return client.post("/cosmos/staking/v1beta1/delegate", {
        "delegator_address": delegator_address,
        "validator_address": validator_address,
        "amount": {
            "denom": denom,
            "amount": str(amount)
        }
    })
```

## Advanced Examples

### Async Client

```python
import aiohttp
import asyncio

class AsyncPAWClient:
    def __init__(self, base_url: str = "http://localhost:1317"):
        self.base_url = base_url

    async def get(self, endpoint: str) -> Dict:
        async with aiohttp.ClientSession() as session:
            async with session.get(f"{self.base_url}{endpoint}") as response:
                return await response.json()

    async def get_multiple_pools(self, pool_ids: List[int]) -> List[Dict]:
        tasks = [self.get(f"/paw/dex/v1/pools/{pid}") for pid in pool_ids]
        results = await asyncio.gather(*tasks)
        return [r["pool"] for r in results]

# Usage
async def main():
    client = AsyncPAWClient()
    pools = await client.get_multiple_pools([1, 2, 3, 4, 5])
    print(pools)

asyncio.run(main())
```

### Retry Logic

```python
from functools import wraps
import time

def retry(max_attempts: int = 3, delay: float = 1.0):
    def decorator(func):
        @wraps(func)
        def wrapper(*args, **kwargs):
            for attempt in range(max_attempts):
                try:
                    return func(*args, **kwargs)
                except Exception as e:
                    if attempt == max_attempts - 1:
                        raise
                    print(f"Retry {attempt + 1}/{max_attempts} after {delay}s...")
                    time.sleep(delay)
        return wrapper
    return decorator

@retry(max_attempts=3, delay=1.0)
def get_pool_safe(pool_id: int) -> Dict:
    return get_pool(pool_id)
```

## See Also

- [JavaScript Examples](./javascript.md)
- [Go Examples](./go.md)
- [cURL Examples](./curl.md)
