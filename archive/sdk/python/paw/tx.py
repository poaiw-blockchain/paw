"""Transaction building and signing."""

import json
import base64
from typing import List, Dict, Any, Optional
from .types import TxResult, GasOptions, Coin


class TxBuilder:
    """Transaction builder for PAW blockchain."""

    def __init__(self, client, wallet):
        """Initialize transaction builder."""
        self.client = client
        self.wallet = wallet

    async def sign_and_broadcast(
        self,
        messages: List[Dict[str, Any]],
        options: Optional[GasOptions] = None
    ) -> TxResult:
        """Sign and broadcast a transaction."""
        if options is None:
            options = GasOptions()

        # Get account info
        account_info = await self._get_account_info(self.wallet.address)

        # Build transaction
        tx_body = self._build_tx_body(messages, options.memo)

        # Estimate gas if not provided
        gas_limit = options.gas_limit
        if gas_limit is None:
            gas_limit = await self._estimate_gas(tx_body, account_info)

        # Build auth info
        auth_info = self._build_auth_info(gas_limit, options.gas_price or self.client.config.gas_price)

        # Sign transaction
        signed_tx = await self._sign_tx(tx_body, auth_info, account_info)

        # Broadcast transaction
        return await self._broadcast_tx(signed_tx)

    def _build_tx_body(self, messages: List[Dict[str, Any]], memo: str) -> Dict[str, Any]:
        """Build transaction body."""
        return {
            "messages": messages,
            "memo": memo,
            "timeout_height": "0",
            "extension_options": [],
            "non_critical_extension_options": []
        }

    def _build_auth_info(self, gas_limit: int, gas_price: str) -> Dict[str, Any]:
        """Build auth info."""
        # Extract denom and amount from gas price
        denom = gas_price.lstrip("0123456789.")
        price = float(gas_price.rstrip("abcdefghijklmnopqrstuvwxyz"))
        fee_amount = str(int(gas_limit * price))

        return {
            "signer_infos": [{
                "public_key": {
                    "@type": "/cosmos.crypto.secp256k1.PubKey",
                    "key": base64.b64encode(self.wallet.public_key).decode()
                },
                "mode_info": {
                    "single": {"mode": "SIGN_MODE_DIRECT"}
                },
                "sequence": "0"  # Will be filled from account info
            }],
            "fee": {
                "amount": [{"denom": denom, "amount": fee_amount}],
                "gas_limit": str(gas_limit),
                "payer": "",
                "granter": ""
            }
        }

    async def _get_account_info(self, address: str) -> Dict[str, Any]:
        """Get account information."""
        try:
            data = await self.client.get(f"/cosmos/auth/v1beta1/accounts/{address}")
            return data["account"]
        except Exception:
            # Return default account info for new accounts
            return {
                "account_number": "0",
                "sequence": "0"
            }

    async def _estimate_gas(self, tx_body: Dict[str, Any], account_info: Dict[str, Any]) -> int:
        """Estimate gas for transaction."""
        # Simplified gas estimation
        # In production, use /cosmos/tx/v1beta1/simulate endpoint
        base_gas = 200000
        per_message = 100000
        estimated = base_gas + (len(tx_body["messages"]) * per_message)
        return int(estimated * self.client.config.gas_adjustment)

    async def _sign_tx(
        self,
        tx_body: Dict[str, Any],
        auth_info: Dict[str, Any],
        account_info: Dict[str, Any]
    ) -> Dict[str, Any]:
        """Sign transaction."""
        # Update sequence
        auth_info["signer_infos"][0]["sequence"] = account_info.get("sequence", "0")

        # Create sign doc
        sign_doc = {
            "body_bytes": json.dumps(tx_body).encode(),
            "auth_info_bytes": json.dumps(auth_info).encode(),
            "chain_id": self.client.config.chain_id,
            "account_number": account_info.get("account_number", "0")
        }

        # Sign (simplified - in production, use proper protobuf encoding)
        message = json.dumps(sign_doc).encode()
        signature = self.wallet.sign(message)

        return {
            "body": tx_body,
            "auth_info": auth_info,
            "signatures": [base64.b64encode(signature).decode()]
        }

    async def _broadcast_tx(self, signed_tx: Dict[str, Any]) -> TxResult:
        """Broadcast signed transaction."""
        # Encode transaction
        tx_bytes = base64.b64encode(json.dumps(signed_tx).encode()).decode()

        # Broadcast
        data = await self.client.post("/cosmos/tx/v1beta1/txs", {
            "tx_bytes": tx_bytes,
            "mode": "BROADCAST_MODE_SYNC"
        })

        tx_response = data["tx_response"]

        return TxResult(
            transaction_hash=tx_response["txhash"],
            height=int(tx_response.get("height", 0)),
            code=int(tx_response.get("code", 0)),
            raw_log=tx_response.get("raw_log"),
            gas_used=int(tx_response.get("gas_used", 0)),
            gas_wanted=int(tx_response.get("gas_wanted", 0))
        )
