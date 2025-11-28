"""DEX module for trading operations."""

from typing import List, Optional
from ..types import (
    Pool,
    PoolParams,
    SwapParams,
    AddLiquidityParams,
    RemoveLiquidityParams,
    TxResult,
    GasOptions
)


class DexModule:
    """DEX module for decentralized exchange operations."""

    def __init__(self, client):
        """Initialize DEX module."""
        self.client = client

    async def create_pool(
        self,
        creator: str,
        params: PoolParams,
        options: Optional[GasOptions] = None
    ) -> TxResult:
        """Create a new liquidity pool."""
        message = {
            "@type": "/paw.dex.v1.MsgCreatePool",
            "creator": creator,
            "token_a": params.token_a,
            "token_b": params.token_b,
            "amount_a": params.amount_a,
            "amount_b": params.amount_b
        }

        return await self.client.tx_builder.sign_and_broadcast([message], options)

    async def swap(
        self,
        sender: str,
        params: SwapParams,
        options: Optional[GasOptions] = None
    ) -> TxResult:
        """Swap tokens in a pool."""
        message = {
            "@type": "/paw.dex.v1.MsgSwap",
            "sender": sender,
            "pool_id": params.pool_id,
            "token_in": params.token_in,
            "amount_in": params.amount_in,
            "min_amount_out": params.min_amount_out,
            "recipient": params.recipient or sender
        }

        return await self.client.tx_builder.sign_and_broadcast([message], options)

    async def add_liquidity(
        self,
        sender: str,
        params: AddLiquidityParams,
        options: Optional[GasOptions] = None
    ) -> TxResult:
        """Add liquidity to a pool."""
        message = {
            "@type": "/paw.dex.v1.MsgAddLiquidity",
            "sender": sender,
            "pool_id": params.pool_id,
            "amount_a": params.amount_a,
            "amount_b": params.amount_b,
            "min_shares": params.min_shares
        }

        return await self.client.tx_builder.sign_and_broadcast([message], options)

    async def remove_liquidity(
        self,
        sender: str,
        params: RemoveLiquidityParams,
        options: Optional[GasOptions] = None
    ) -> TxResult:
        """Remove liquidity from a pool."""
        message = {
            "@type": "/paw.dex.v1.MsgRemoveLiquidity",
            "sender": sender,
            "pool_id": params.pool_id,
            "shares": params.shares,
            "min_amount_a": params.min_amount_a,
            "min_amount_b": params.min_amount_b
        }

        return await self.client.tx_builder.sign_and_broadcast([message], options)

    async def get_pool(self, pool_id: str) -> Optional[Pool]:
        """Get pool by ID."""
        try:
            data = await self.client.get(f"/paw/dex/v1/pools/{pool_id}")
            pool_data = data.get("pool")
            if pool_data:
                return Pool(**pool_data)
            return None
        except Exception:
            return None

    async def get_all_pools(self) -> List[Pool]:
        """Get all pools."""
        try:
            data = await self.client.get("/paw/dex/v1/pools")
            pools = data.get("pools", [])
            return [Pool(**p) for p in pools]
        except Exception:
            return []

    def calculate_swap_output(
        self,
        amount_in: str,
        reserve_in: str,
        reserve_out: str,
        swap_fee: str = "0.003"
    ) -> str:
        """Calculate swap output amount."""
        amount_in_int = int(amount_in)
        reserve_in_int = int(reserve_in)
        reserve_out_int = int(reserve_out)
        fee_int = int(float(swap_fee) * 10000)

        amount_in_with_fee = (amount_in_int * (10000 - fee_int)) // 10000
        numerator = amount_in_with_fee * reserve_out_int
        denominator = reserve_in_int + amount_in_with_fee

        return str(numerator // denominator)

    def calculate_price_impact(
        self,
        amount_in: str,
        reserve_in: str,
        reserve_out: str
    ) -> float:
        """Calculate price impact percentage."""
        amount_out = self.calculate_swap_output(amount_in, reserve_in, reserve_out, "0")
        price_before = float(reserve_out) / float(reserve_in)
        price_after = float(amount_out) / float(amount_in)

        return abs((price_after - price_before) / price_before) * 100
