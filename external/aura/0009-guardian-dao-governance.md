# RFC-0009: Guardian DAO Governance Specification

- **Author(s):** Governance Team
- **Status:** Draft
- **Created:** 2025-11-12
- **Target Release:** Community Testnet

## Summary

Define the on-chain governance system for the AURA network, enabling decentralized control over protocol upgrades, treasury management, network parameters, and emergency actions through a proposal-based voting mechanism with VRF-selected delegate committees, timelocks, and multi-sig guardian oversight.

## Motivation & Goals

- Enable decentralized, transparent control over network evolution without relying on centralized authority.
- Protect minority stakeholders through veto mechanisms and supermajority requirements.
- Prevent governance attacks (vote buying, cartel formation) via VRF-based committee selection.
- Provide rapid response capability for emergency situations via guardian multi-sig.
- Balance security (timelocks, validation) with agility (delegate sampling, automated execution).

## Detailed Design

### Proposal Types

The governance module supports five distinct proposal categories:

1. **ParameterChange**: Modify network parameters (fee rates, emission schedule, staking requirements, slashing percentages, etc.)
2. **SoftwareUpgrade**: Coordinate hard forks and protocol upgrades with version specification and block height activation
3. **TreasurySpend**: Allocate funds from the network treasury to specified addresses for development, grants, or operations
4. **TextProposal**: Non-binding signaling proposals for community sentiment and directional guidance
5. **EmergencyAction**: Critical interventions (pause protocol, emergency slashing, circuit breakers)

Each proposal type has specific validation rules, timelock periods, and execution handlers defined in the governance module.

### Proposal Lifecycle

#### 1. Submission Phase

- **Initial Deposit**: 10,000 PAW required to submit a proposal (anti-spam mechanism)
- **Proposal Content**: Structured data including type, title, description, parameters, and optional code hash
- **Submitter**: Any address with sufficient PAW balance
- **Validation**: Automatic checks for well-formed proposals, valid parameter ranges, and schema compliance

#### 2. Deposit Period (7 days)

- **Minimum Deposit**: 100,000 PAW total must be reached for proposal to enter voting
- **Multi-contributor**: Any address can contribute to the deposit pool
- **Refund Conditions**: Deposits refunded if proposal passes or fails normally
- **Slash Conditions**: Deposits slashed (burned) if proposal is vetoed or identified as spam
- **Cancellation**: Submitter can cancel during deposit period with 50% deposit penalty

#### 3. Voting Period (14 days)

- **Eligible Voters**: All addresses with staked PAW at snapshot block (proposal entry into voting)
- **Vote Options**: Yes, No, NoWithVeto, Abstain
- **Vote Weight**: Proportional to staked PAW amount
- **Vote Changes**: Voters can change their vote anytime during the voting period (last vote counts)
- **Delegation**: Votes automatically delegated to validators unless explicitly overridden

#### 4. Tallying & Execution

- **Tally Time**: Immediately after voting period ends
- **Outcome Determination**: Based on quorum, threshold, and veto rules (see Voting Rules)
- **Execution**: Automatic via governance module handlers if proposal passes
- **Timelock**: Mandatory delay before execution based on proposal type (see Timelock section)
- **Failure Handling**: Failed proposals archived with reason; deposits handled per outcome

### Voting Rules

All proposals must satisfy the following conditions to pass:

#### Quorum Requirement

- **Minimum Participation**: 40% of total staked PAW must vote (abstain counts toward quorum)
- **Calculation**: `(total_votes / total_staked_paw) >= 0.40`
- **Rationale**: Ensures broad stakeholder engagement; prevents low-turnout manipulation

#### Approval Threshold

- **Supermajority**: 66.7% (2/3) of non-abstaining votes must be Yes
- **Calculation**: `(yes_votes / (yes_votes + no_votes + veto_votes)) >= 0.667`
- **Rationale**: High bar for protocol changes ensures consensus and stability

#### Veto Protection

- **Veto Threshold**: If 33.3% (1/3) of votes are NoWithVeto, proposal fails and deposits are slashed
- **Calculation**: `(veto_votes / total_votes) >= 0.333`
- **Rationale**: Protects minority from harmful proposals; deters malicious governance attacks
- **Slash Mechanism**: Vetoed proposal deposits burned to discourage spam/attacks

#### Vote Weight Distribution

- **Yes**: Support the proposal
- **No**: Reject the proposal (deposits refunded)
- **NoWithVeto**: Reject and punish (deposits slashed if veto threshold met)
- **Abstain**: Count toward quorum but not threshold calculation

### VRF Committee Selection

To prevent cartel formation and ensure fair representation across the staking pool:

#### Committee Composition

- **Size**: 100 delegates sampled each epoch
- **Source Pool**: All active stakers (validators + delegators)
- **Selection Probability**: Proportional to staked PAW amount
- **Refresh Rate**: Every epoch (7 days on mainnet)

#### VRF Mechanism

- **Seed Source**: Previous block hash from the last block of the preceding epoch
- **Algorithm**: Verifiable Random Function (VRF) using ed25519-vrf or similar
- **Verification**: VRF proof published on-chain for public verification
- **Determinism**: Given seed + staker set, committee selection is deterministic and verifiable

#### Committee Powers

- **Fast-Track Voting**: Committee can approve emergency proposals with 75% majority
- **Proposal Review**: Optional pre-screening for technical proposals (advisory only)
- **Dispute Resolution**: Committee members serve as arbitrators in slashing appeals
- **Rotation**: No member can serve consecutive epochs (enforced by selection algorithm)

#### Anti-Gaming Measures

- **Snapshot Block**: Stake weights frozen at epoch boundary (prevents last-minute stake manipulation)
- **Minimum Stake**: 1,000 PAW minimum to be eligible for committee selection
- **Sybil Resistance**: Selection probability caps at 5% per address (prevents single-entity dominance)

### Delegation System

#### Delegation Mechanics

- **Automatic Delegation**: Delegators automatically inherit validator voting choices unless overridden
- **Override Rights**: Delegators can vote directly on any proposal, overriding validator's vote for their stake
- **Partial Delegation**: Future enhancement to delegate voting power to multiple addresses by percentage
- **Revocation**: Delegators can change delegation at any time (takes effect next voting period)

#### Validator Responsibilities

- **Vote Obligation**: Validators expected to vote on all proposals (participation tracked)
- **Transparency**: Validators publish voting rationale via off-chain governance forums
- **Commission**: Validators earn 10% of delegation rewards as compensation for governance participation
- **Penalties**: Validators with <50% governance participation rate receive reduced staking rewards

#### Delegation Rewards

- **Reward Source**: 5% of treasury inflation allocated to governance participation
- **Distribution**: Proportional to voting activity and proposal outcomes
- **Calculation**: `reward = (votes_cast / total_eligible_votes) * epoch_governance_allocation`
- **Boost**: +20% reward for voting on passed proposals (incentivizes alignment)

### Timelock Periods

Mandatory delays between proposal passage and execution:

| Proposal Type   | Timelock Duration | Rationale                                                                            |
| --------------- | ----------------- | ------------------------------------------------------------------------------------ |
| ParameterChange | 48 hours          | Allow community to prepare for parameter adjustments; verify intended effects        |
| SoftwareUpgrade | 7 days            | Coordination time for validators/node operators to upgrade software; test migrations |
| TreasurySpend   | 72 hours          | Financial oversight period; allow scrutiny of fund allocations                       |
| TextProposal    | Immediate         | Non-binding; no execution risk                                                       |
| EmergencyAction | Immediate\*       | Critical security situations require rapid response                                  |

**EmergencyAction Execution**: Bypasses standard timelock but requires guardian multi-sig approval (see below).

#### Timelock Cancellation

- **Guardian Override**: 5-of-9 guardians can cancel any pending proposal during timelock
- **Community Veto**: If 75% of original Yes voters switch to NoWithVeto during timelock, execution cancelled
- **Use Case**: Discovery of critical vulnerabilities or unintended consequences after passage

### Guardian Multi-Sig

Emergency governance backstop for critical situations:

#### Composition

- **Members**: 9 guardians selected during genesis (initial bootstrap period)
- **Threshold**: 5-of-9 signatures required for action
- **Term**: 12 months with staggered rotation (3 guardians rotate every 4 months)
- **Selection**: Future guardians elected via standard governance proposal (ParameterChange)

#### Powers

- **Emergency Execution**: Execute EmergencyAction proposals immediately
- **Timelock Cancellation**: Cancel pending proposals during timelock period
- **Circuit Breakers**: Pause protocol modules (staking, governance, treasury) in crisis
- **Slashing Override**: Reverse or modify slashing events in case of bugs/exploits
- **Parameter Freezes**: Temporarily lock critical parameters during security incidents

#### Limitations

- **No Fund Withdrawal**: Cannot directly withdraw from treasury without proposal
- **No Parameter Changes**: Cannot modify parameters outside emergency context
- **Transparency**: All guardian actions logged on-chain with full audit trail
- **Sunset Clause**: Guardian powers automatically expire after 24 months (mainnet maturity)

#### Guardian Selection Criteria

- **Technical Expertise**: Core protocol understanding and blockchain security experience
- **Reputation**: Established community trust and aligned incentives
- **Geographic Diversity**: Distributed across time zones for 24/7 coverage
- **Independence**: No more than 2 guardians from single organization
- **Operational Security**: Hardware signing modules and key custody procedures

### Proposal Validation

#### Automated Checks

- **Schema Validation**: Proposal structure matches type-specific schema
- **Parameter Bounds**: ParameterChange values within acceptable ranges (e.g., fee rates 0-100%)
- **Code Verification**: SoftwareUpgrade proposals include verifiable build artifacts and checksums
- **Treasury Balance**: TreasurySpend amounts ≤ available treasury balance
- **Signature Verification**: All proposal transactions properly signed and authorized

#### Community Review

- **Minimum Discussion**: 48-hour forum discussion period before deposit period starts
- **Technical Review**: Optional committee review for complex technical proposals
- **Impact Analysis**: Automated simulation of parameter changes on testnet state
- **Conflict Detection**: Checks for conflicting proposals already in voting/execution

#### Spam Prevention

- **Deposit Slashing**: Vetoed proposals lose deposits (burned, not redistributed)
- **Rate Limiting**: Maximum 3 proposals per address per epoch
- **Duplicate Detection**: Similar proposals within 30 days flagged for review
- **Quality Threshold**: Proposals require minimum 100-word description

## Smart Contracts & Chain Modules

### Governance Module Architecture

The governance system is **built into the AURA chain** as a native module (not a separate smart contract), integrated with the Cosmos SDK governance framework:

#### Module Structure

```
x/governance/
├── keeper/          # State management and business logic
├── types/           # Proposal types, messages, and parameters
├── handler.go       # Message routing and validation
├── abci.go          # EndBlocker for tallying and execution
└── genesis.go       # Initial governance parameters
```

#### State Schema

- **Proposals**: `proposals/{proposal_id} -> Proposal`
- **Deposits**: `deposits/{proposal_id}/{depositor} -> Deposit`
- **Votes**: `votes/{proposal_id}/{voter} -> Vote`
- **Tally Results**: `tally/{proposal_id} -> TallyResult`
- **Parameters**: `params -> GovParams`

#### Message Types

- `MsgSubmitProposal`: Submit new proposal with initial deposit
- `MsgDeposit`: Add to proposal deposit pool
- `MsgVote`: Cast vote on active proposal
- `MsgVoteWeighted`: Future enhancement for split voting
- `MsgCancelProposal`: Submitter cancellation during deposit period

#### Query Endpoints

- `QueryProposal`: Get proposal details by ID
- `QueryProposals`: List proposals with filters (status, voter, depositor)
- `QueryVote`: Get specific vote details
- `QueryVotes`: List all votes for proposal
- `QueryParams`: Get governance parameters
- `QueryTally`: Get real-time tally results

#### EndBlocker Logic

1. Process proposals reaching end of deposit period (archive or promote to voting)
2. Process proposals reaching end of voting period (tally and execute/reject)
3. Execute passed proposals after timelock expiration
4. Update delegation rewards for governance participation
5. Emit events for governance state changes

### Integration with Other Modules

#### Treasury Module

- **Spending**: TreasurySpend proposals trigger `MsgSend` from treasury account
- **Funding**: Treasury receives tx fees and inflation allocation
- **Balance Checks**: Governance module queries treasury balance before approving spend

#### Staking Module

- **Vote Weight**: Governance queries staking module for delegator weights
- **Validator Set**: Committee selection pulls from active validator set
- **Slashing**: EmergencyAction proposals can trigger staking slashing events

#### Identity Module

- **DID Association**: Future enhancement to link proposals to issuer DIDs
- **Reputation**: Track governance participation in identity attestations

#### VRF Module

- **Randomness**: Committee selection uses VRF module for verifiable randomness
- **Seed Management**: VRF module maintains seed history for reproducibility

## Security Considerations

### Attack Vectors & Mitigations

#### Vote Buying

- **Risk**: Wealthy actors buying votes to pass malicious proposals
- **Mitigation**: VRF committee, veto mechanism, delegation transparency, timelock review

#### Cartel Formation

- **Risk**: Validators colluding to control governance outcomes
- **Mitigation**: VRF sampling prevents persistent power concentration; rotation enforced

#### Spam Attacks

- **Risk**: Flooding governance with frivolous proposals
- **Mitigation**: Deposit requirements, rate limiting, veto slashing, duplicate detection

#### Flash Loan Attacks

- **Risk**: Borrowing PAW to gain voting power temporarily
- **Mitigation**: Snapshot-based voting (stake must be held at snapshot block)

#### Sybil Attacks

- **Risk**: Creating many addresses to gain committee seats
- **Mitigation**: Probability caps, minimum stake requirements, stake-weighted selection

#### Timelock Front-Running

- **Risk**: Exploiting knowledge of upcoming parameter changes
- **Mitigation**: Transparent timelock periods, guardian cancellation rights, public execution queue

### Privacy Considerations

- **Vote Privacy**: Future enhancement for private voting using ZK-SNARKs (phase 2)
- **Delegation Privacy**: Delegation relationships publicly visible (required for verification)
- **Proposal Anonymity**: Not supported (accountability requires attribution)

### Audit & Monitoring

- **Governance Dashboard**: Real-time monitoring of active proposals, participation rates, treasury status
- **Alert System**: Automated alerts for unusual voting patterns, large deposits, guardian actions
- **Transparency Logs**: All governance events logged and indexed for public analysis
- **Penetration Testing**: Annual security audits of governance module and VRF implementation

## Validation Plan

### Testnet Governance Exercises

#### Phase 1: Basic Functionality (4 weeks)

- Submit and pass TextProposal
- Submit and reject proposal (fails threshold)
- Veto spam proposal (test deposit slashing)
- Exercise delegation override mechanism

#### Phase 2: Parameter Changes (4 weeks)

- Modify fee rates via ParameterChange
- Adjust staking parameters
- Update governance params (quorum, threshold)
- Verify timelock enforcement

#### Phase 3: Treasury & Upgrades (4 weeks)

- Execute TreasurySpend proposal
- Coordinate SoftwareUpgrade (testnet hard fork)
- Test upgrade rollback procedures
- Validate upgrade coordination timing

#### Phase 4: Emergency Scenarios (2 weeks)

- Guardian multi-sig emergency pause
- Timelock cancellation exercise
- Committee-driven fast-track proposal
- Recovery from governance module bug

#### Phase 5: Attack Simulations (2 weeks)

- Vote buying simulation (detect and alert)
- Spam proposal flood (rate limiting)
- Sybil attack on committee selection (verify caps)
- Flash loan voting attempt (verify snapshot protection)

### Success Criteria

- 100% of test proposals execute as expected
- Zero deposit/fund losses during testing
- VRF committee distribution within 5% of theoretical (chi-squared test)
- Guardian response time <2 hours for emergency actions
- Participation rate >50% among active testnet validators

### Monitoring Metrics

- **Proposal Success Rate**: Percentage of proposals reaching quorum and passing
- **Participation Rate**: Average voter turnout across all proposals
- **Delegation Rate**: Percentage of stake using delegation vs. direct voting
- **Committee Diversity**: Gini coefficient of committee member selection frequency
- **Execution Latency**: Time from proposal passage to execution completion
- **Guardian Activity**: Frequency and justification of guardian interventions

## Backwards Compatibility

### Migration from Genesis

- **Initial Parameters**: Genesis file includes default governance parameters
- **Bootstrap Guardians**: 9 genesis guardians appointed by foundation (published pre-launch)
- **First Proposals**: Simple TextProposals to validate system before critical changes
- **Parameter Evolution**: Governance parameters themselves modifiable via ParameterChange

### Module Upgrades

- **Versioning**: Governance module version tracked in chain state
- **Soft Forks**: Compatible changes deployed via software updates (no proposal needed)
- **Hard Forks**: Breaking changes require SoftwareUpgrade proposal + coordination
- **State Migration**: Module upgrade handler performs state schema migrations

### Future Enhancements

- **Weighted Voting**: Split vote across Yes/No/Abstain with percentage weights
- **Private Voting**: ZK-SNARK based vote privacy (maintain verifiable tallying)
- **Quadratic Voting**: Alternative voting mechanism for specific proposal types
- **Conditional Proposals**: Proposals that execute only if conditions met (e.g., treasury balance threshold)
- **On-Chain Forums**: Integrate discussion directly into governance module (currently off-chain)

## Open Questions

1. **Committee Compensation**: Should VRF committee members receive additional rewards for service?
2. **Proposal Templates**: Should we provide standardized templates for common proposal types?
3. **Cross-Chain Governance**: How to handle governance proposals affecting cross-chain integrations (IBC)?
4. **Guardian Rotation**: Should guardian rotation be automatic or require governance approval?
5. **Vote Delegation Marketplace**: Allow delegators to "sell" votes in transparent marketplace with price discovery?
6. **Proposal Bounties**: Should submitters of passed proposals receive rewards from treasury?
7. **Multi-Chain Coordination**: How to synchronize governance across AURA mainnet + testnets?
8. **Emergency Threshold Adjustment**: Should guardian threshold decrease over time as network matures?

## References

- Cosmos SDK Governance Module: https://docs.cosmos.network/main/modules/gov
- Tendermint Consensus: https://docs.tendermint.com/master/spec/
- VRF Specification: https://datatracker.ietf.org/doc/html/draft-irtf-cfrg-vrf
- AURA Whitepaper: `docs/whitepaper.md` (VRF committee mechanism)
- Governance Best Practices: https://github.com/gauntlet-labs/governance-toolkit

## Appendix: Parameter Reference

### Default Governance Parameters

```yaml
governance_params:
  # Deposit Phase
  min_deposit: '100000000000' # 100,000 PAW (18 decimals)
  deposit_period: '604800s' # 7 days

  # Voting Phase
  voting_period: '1209600s' # 14 days
  quorum: '0.40' # 40%
  threshold: '0.667' # 66.7%
  veto_threshold: '0.333' # 33.3%

  # Timelocks
  parameter_change_timelock: '172800s' # 48 hours
  software_upgrade_timelock: '604800s' # 7 days
  treasury_spend_timelock: '259200s' # 72 hours

  # Committee
  committee_size: 100
  committee_epoch: '604800s' # 7 days
  committee_min_stake: '1000000000000' # 1,000 PAW
  committee_max_probability: '0.05' # 5%

  # Delegation
  validator_commission: '0.10' # 10%
  governance_reward_allocation: '0.05' # 5% of inflation

  # Spam Prevention
  max_proposals_per_epoch: 3
  min_description_length: 100
  duplicate_detection_window: '2592000s' # 30 days

  # Guardian
  guardian_threshold: 5
  guardian_total: 9
  guardian_term: '31536000s' # 12 months
  guardian_sunset: '63072000s' # 24 months from genesis
```

### Emergency Parameter Overrides

Guardians can temporarily override the following parameters during emergencies:

- `voting_period`: Reduce to 24 hours for critical security fixes
- `quorum`: Lower to 25% if network partition prevents normal participation
- `timelock`: Reduce to 0 for immediate execution of emergency patches
- `committee_size`: Expand to 200 if needed for broader representation

All emergency overrides automatically expire after 7 days unless renewed via guardian vote.
