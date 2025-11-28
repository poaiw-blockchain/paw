"""Staking module for delegation operations."""

from typing import List, Optional
from ..types import (
    Validator,
    ValidatorDescription,
    ValidatorCommission,
    DelegateParams,
    UndelegateParams,
    RedelegateParams,
    Coin,
    TxResult,
    GasOptions
)


class StakingModule:
    """Staking module for delegation operations."""

    def __init__(self, client):
        """Initialize staking module."""
        self.client = client

    async def delegate(
        self,
        delegator: str,
        params: DelegateParams,
        options: Optional[GasOptions] = None
    ) -> TxResult:
        """Delegate tokens to a validator."""
        message = {
            "@type": "/cosmos.staking.v1beta1.MsgDelegate",
            "delegator_address": delegator,
            "validator_address": params.validator_address,
            "amount": {"denom": params.denom, "amount": params.amount}
        }

        return await self.client.tx_builder.sign_and_broadcast([message], options)

    async def undelegate(
        self,
        delegator: str,
        params: UndelegateParams,
        options: Optional[GasOptions] = None
    ) -> TxResult:
        """Undelegate tokens from a validator."""
        message = {
            "@type": "/cosmos.staking.v1beta1.MsgUndelegate",
            "delegator_address": delegator,
            "validator_address": params.validator_address,
            "amount": {"denom": params.denom, "amount": params.amount}
        }

        return await self.client.tx_builder.sign_and_broadcast([message], options)

    async def redelegate(
        self,
        delegator: str,
        params: RedelegateParams,
        options: Optional[GasOptions] = None
    ) -> TxResult:
        """Redelegate tokens from one validator to another."""
        message = {
            "@type": "/cosmos.staking.v1beta1.MsgBeginRedelegate",
            "delegator_address": delegator,
            "validator_src_address": params.src_validator_address,
            "validator_dst_address": params.dst_validator_address,
            "amount": {"denom": params.denom, "amount": params.amount}
        }

        return await self.client.tx_builder.sign_and_broadcast([message], options)

    async def withdraw_rewards(
        self,
        delegator: str,
        validator_address: str,
        options: Optional[GasOptions] = None
    ) -> TxResult:
        """Withdraw delegation rewards from a validator."""
        message = {
            "@type": "/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward",
            "delegator_address": delegator,
            "validator_address": validator_address
        }

        return await self.client.tx_builder.sign_and_broadcast([message], options)

    async def get_validators(self) -> List[Validator]:
        """Get all validators."""
        try:
            data = await self.client.get("/cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED")
            validators = data.get("validators", [])
            return [self._parse_validator(v) for v in validators]
        except Exception:
            return []

    async def get_delegations(self, delegator: str) -> List[dict]:
        """Get delegations for a delegator."""
        try:
            data = await self.client.get(f"/cosmos/staking/v1beta1/delegations/{delegator}")
            return data.get("delegation_responses", [])
        except Exception:
            return []

    async def get_rewards(self, delegator: str) -> List[Coin]:
        """Get rewards for a delegator."""
        try:
            data = await self.client.get(f"/cosmos/distribution/v1beta1/delegators/{delegator}/rewards")
            total = data.get("total", [])
            return [Coin(denom=c["denom"], amount=c["amount"]) for c in total]
        except Exception:
            return []

    def _parse_validator(self, data: dict) -> Validator:
        """Parse validator data."""
        desc_data = data.get("description", {})
        comm_data = data.get("commission", {}).get("commission_rates", {})

        return Validator(
            operator_address=data.get("operator_address", ""),
            consensus_pubkey=data.get("consensus_pubkey", ""),
            jailed=data.get("jailed", False),
            status=data.get("status", 0),
            tokens=data.get("tokens", "0"),
            delegator_shares=data.get("delegator_shares", "0"),
            description=ValidatorDescription(
                moniker=desc_data.get("moniker", ""),
                identity=desc_data.get("identity", ""),
                website=desc_data.get("website", ""),
                security_contact=desc_data.get("security_contact", ""),
                details=desc_data.get("details", "")
            ),
            commission=ValidatorCommission(
                rate=comm_data.get("rate", "0"),
                max_rate=comm_data.get("max_rate", "0"),
                max_change_rate=comm_data.get("max_change_rate", "0")
            )
        )
