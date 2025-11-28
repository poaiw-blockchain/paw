"""Main client for PAW blockchain."""

import httpx
from typing import Optional
from .types import ChainConfig
from .wallet import PawWallet
from .tx import TxBuilder
from .modules.bank import BankModule
from .modules.dex import DexModule
from .modules.staking import StakingModule
from .modules.governance import GovernanceModule


class PawClient:
    """Main client for interacting with PAW blockchain."""

    def __init__(self, config: ChainConfig):
        """Initialize the PAW client."""
        self.config = config
        self._wallet: Optional[PawWallet] = None
        self._http_client = httpx.AsyncClient(timeout=30.0)
        self._tx_builder: Optional[TxBuilder] = None

        # Initialize modules
        self.bank = BankModule(self)
        self.dex = DexModule(self)
        self.staking = StakingModule(self)
        self.governance = GovernanceModule(self)

    async def connect_wallet(self, wallet: PawWallet) -> None:
        """Connect a wallet for signing transactions."""
        self._wallet = wallet
        self._tx_builder = TxBuilder(self, wallet)

    @property
    def wallet(self) -> PawWallet:
        """Get connected wallet."""
        if self._wallet is None:
            raise RuntimeError("Wallet not connected")
        return self._wallet

    @property
    def tx_builder(self) -> TxBuilder:
        """Get transaction builder."""
        if self._tx_builder is None:
            raise RuntimeError("Transaction builder not available. Connect wallet first.")
        return self._tx_builder

    @property
    def rest_endpoint(self) -> str:
        """Get REST API endpoint."""
        if self.config.rest_endpoint:
            return self.config.rest_endpoint
        return self.config.rpc_endpoint.replace(":26657", ":1317")

    async def get(self, path: str) -> dict:
        """Make GET request to REST API."""
        url = f"{self.rest_endpoint}{path}"
        response = await self._http_client.get(url)
        response.raise_for_status()
        return response.json()

    async def post(self, path: str, data: dict) -> dict:
        """Make POST request to REST API."""
        url = f"{self.rest_endpoint}{path}"
        response = await self._http_client.post(url, json=data)
        response.raise_for_status()
        return response.json()

    async def get_height(self) -> int:
        """Get current block height."""
        data = await self.get("/cosmos/base/tendermint/v1beta1/blocks/latest")
        return int(data["block"]["header"]["height"])

    async def get_chain_id(self) -> str:
        """Get chain ID."""
        data = await self.get("/cosmos/base/tendermint/v1beta1/node_info")
        return data["default_node_info"]["network"]

    def is_connected(self) -> bool:
        """Check if wallet is connected."""
        return self._wallet is not None

    async def close(self) -> None:
        """Close the HTTP client."""
        await self._http_client.aclose()

    async def __aenter__(self):
        """Async context manager entry."""
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit."""
        await self.close()
