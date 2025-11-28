"""Bank module for token operations."""

from typing import List, Optional
from ..types import Coin, SendParams, TxResult, GasOptions


class BankModule:
    """Bank module for managing tokens."""

    def __init__(self, client):
        """Initialize bank module."""
        self.client = client

    async def get_balance(self, address: str, denom: str) -> Optional[Coin]:
        """Get balance for a specific denom."""
        try:
            data = await self.client.get(f"/cosmos/bank/v1beta1/balances/{address}/{denom}")
            balance = data.get("balance")
            if balance:
                return Coin(denom=balance["denom"], amount=balance["amount"])
            return None
        except Exception:
            return None

    async def get_all_balances(self, address: str) -> List[Coin]:
        """Get all balances for an address."""
        try:
            data = await self.client.get(f"/cosmos/bank/v1beta1/balances/{address}")
            balances = data.get("balances", [])
            return [Coin(denom=b["denom"], amount=b["amount"]) for b in balances]
        except Exception:
            return []

    async def send(
        self,
        sender: str,
        params: SendParams,
        options: Optional[GasOptions] = None
    ) -> TxResult:
        """Send tokens to another address."""
        message = {
            "@type": "/cosmos.bank.v1beta1.MsgSend",
            "from_address": sender,
            "to_address": params.recipient,
            "amount": [{"denom": params.denom, "amount": params.amount}]
        }

        if options is None:
            options = GasOptions(memo=params.memo)

        return await self.client.tx_builder.sign_and_broadcast([message], options)

    def format_balance(self, balance: Coin, decimals: int = 6) -> str:
        """Format balance for display."""
        amount = int(balance.amount) / (10 ** decimals)
        denom_upper = balance.denom.replace("u", "").upper()
        return f"{amount:.{decimals}f} {denom_upper}"
