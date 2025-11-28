"""Governance module for proposal operations."""

from typing import List, Optional
from datetime import datetime
from dateutil import parser
from ..types import (
    Proposal,
    TallyResult,
    VoteParams,
    DepositParams,
    VoteOption,
    Coin,
    TxResult,
    GasOptions
)


class GovernanceModule:
    """Governance module for proposal operations."""

    def __init__(self, client):
        """Initialize governance module."""
        self.client = client

    async def submit_text_proposal(
        self,
        proposer: str,
        title: str,
        description: str,
        initial_deposit: str,
        denom: str = "upaw",
        options: Optional[GasOptions] = None
    ) -> TxResult:
        """Submit a text proposal."""
        message = {
            "@type": "/cosmos.gov.v1beta1.MsgSubmitProposal",
            "content": {
                "@type": "/cosmos.gov.v1beta1.TextProposal",
                "title": title,
                "description": description
            },
            "initial_deposit": [{"denom": denom, "amount": initial_deposit}],
            "proposer": proposer
        }

        return await self.client.tx_builder.sign_and_broadcast([message], options)

    async def vote(
        self,
        voter: str,
        params: VoteParams,
        options: Optional[GasOptions] = None
    ) -> TxResult:
        """Vote on a proposal."""
        message = {
            "@type": "/cosmos.gov.v1beta1.MsgVote",
            "proposal_id": params.proposal_id,
            "voter": voter,
            "option": params.option,
            "metadata": params.metadata
        }

        return await self.client.tx_builder.sign_and_broadcast([message], options)

    async def deposit(
        self,
        depositor: str,
        params: DepositParams,
        options: Optional[GasOptions] = None
    ) -> TxResult:
        """Deposit to a proposal."""
        message = {
            "@type": "/cosmos.gov.v1beta1.MsgDeposit",
            "proposal_id": params.proposal_id,
            "depositor": depositor,
            "amount": [{"denom": params.denom, "amount": params.amount}]
        }

        return await self.client.tx_builder.sign_and_broadcast([message], options)

    async def get_proposals(self, status: Optional[int] = None) -> List[Proposal]:
        """Get all proposals."""
        try:
            url = "/cosmos/gov/v1beta1/proposals"
            if status is not None:
                url += f"?proposal_status={status}"

            data = await self.client.get(url)
            proposals = data.get("proposals", [])
            return [self._parse_proposal(p) for p in proposals]
        except Exception:
            return []

    async def get_proposal(self, proposal_id: str) -> Optional[Proposal]:
        """Get proposal by ID."""
        try:
            data = await self.client.get(f"/cosmos/gov/v1beta1/proposals/{proposal_id}")
            proposal_data = data.get("proposal")
            if proposal_data:
                return self._parse_proposal(proposal_data)
            return None
        except Exception:
            return None

    async def get_tally(self, proposal_id: str) -> Optional[TallyResult]:
        """Get tally for a proposal."""
        try:
            data = await self.client.get(f"/cosmos/gov/v1beta1/proposals/{proposal_id}/tally")
            tally_data = data.get("tally")
            if tally_data:
                return TallyResult(**tally_data)
            return None
        except Exception:
            return None

    def _parse_proposal(self, data: dict) -> Proposal:
        """Parse proposal data."""
        tally_data = data.get("final_tally_result", {})
        deposits = data.get("total_deposit", [])

        return Proposal(
            proposal_id=data.get("proposal_id", ""),
            status=data.get("status", 0),
            final_tally_result=TallyResult(
                yes=tally_data.get("yes", "0"),
                abstain=tally_data.get("abstain", "0"),
                no=tally_data.get("no", "0"),
                no_with_veto=tally_data.get("no_with_veto", "0")
            ),
            submit_time=parser.parse(data.get("submit_time", "")),
            deposit_end_time=parser.parse(data.get("deposit_end_time", "")),
            total_deposit=[Coin(denom=d["denom"], amount=d["amount"]) for d in deposits],
            voting_start_time=parser.parse(data.get("voting_start_time", "")),
            voting_end_time=parser.parse(data.get("voting_end_time", ""))
        )

    @staticmethod
    def get_vote_option_name(option: VoteOption) -> str:
        """Get vote option name."""
        return {
            VoteOption.YES: "Yes",
            VoteOption.NO: "No",
            VoteOption.ABSTAIN: "Abstain",
            VoteOption.NO_WITH_VETO: "No with Veto",
            VoteOption.UNSPECIFIED: "Unspecified"
        }.get(option, "Unknown")
