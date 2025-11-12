# PAW Manageable Blockchain

This document distills the original PoAIW vision into a manageable, buildable initial release centered on the **PAW** token, rapid early adoption, and an extensible path toward verifiable intelligence infrastructure.

## Abstract

PAW launches as a compact layer 1 focused on useful AI work with a predictable, deflationary supply. The protocol initially delivers a sharply scoped controller chain and a paired compute plane that routes verified tasks through secure API key aggregation. Every decision, from the reduced token distribution to the incentive collar, privileges tangible launches, early rewards, and an upgrade path that the community can unlock through governance.

## Core Principles

- **Manageable Release**: Start with a compact validator set, simplified DPoS commitments, and a single compute shard so the network can be deployed, audited, and stabilized within months.
- **Early Adoption Incentives**: Validators, node operators, and API donors earn boosted rewards through the first six months, then automatically taper as coin value rises.
- **Deflationary Discipline**: Emissions halve annually; fees and burn mechanics grow with real-world demand so PAW appreciates as utility scales.
- **Modular Growth**: Roadmap milestones intentionally pause at deliverables that can be extended into zkML, sharding, and cross-chain bridges once the baseline proves stable.

## Baseline Architecture

1. **Controller Chain (L1)** - 4-second blocks, BFT-enhanced DPoS, and a deterministic epoch length for validator rotations. Only essential state lives here: accounts, stakes, governance actions, and task summaries.
2. **Secure Compute Plane** - API keys flow through TEEs to a managed pool of minutes; compute agents pull tasks via time-limited proxy tokens and report proofs back to be settled on-chain.
3. **API Key/Minutes Pool** - Donors unlock compute capacity without exposing credentials. The minimum viable pool uses tiered thresholds, immediate cryptographic key destruction, and optional bridging to foundational compute credits.
4. **Improvement Hooks** - Each major release surfaces configuration flags for future shards, zkML proofs, and third-party compute APIs so governance can ramp them on once usage matures.

## Participant Roles & Incentives

### Node Operators (Core Validators)

- **Mission**: Run validator nodes, archive state, and participate in quick finality rounds.
- **Increased Reward**: Harvests 30% of the base emission pool plus shared transaction fees; this is a deliberate uplift from the original 10% share to offset the reduced supply.
- **Early-Adopter Multiplier**: Nodes active within the first 180 days receive a 1.5x multiplier on their base reward, which decays linearly over the next year and stops once PAW crosses a governance-defined value threshold.
- **Uptime Bonus**: Maintain >=99% uptime for 60 consecutive days to unlock an additional 2.5% of daily emissions from the Node Operator Reserve (a capped 1.5M PAW fund governed separately).

### Validators (Verification Committees)

- Validators earn 30% of emission supply and an extra 1% of settled task value for committee work. The verification reward shrinks by 10% each year, mirroring growing task fees and coin value.
- Delegation commission remains at 10%, but commissions only apply to rewards above the early-adoption multiplier, so delegators share the initial premium.

### API Key/Minutes Pool Donors

- **Reduction**: This pool now receives just 3M PAW, a 75% cut from the originally listed 12M, and fits inside the 8.4M total API donor allocation after the general 30% shrink.
- **Vesting**: 4-year vesting with a 12-month cliff. Early exit or misuse forfeits the remaining allocation to the treasury.
- **Mechanics**: TEE-administered aggregation, minute-level accounting, and proof-of-exhaustion attestations continue to protect donor credentials while allowing PAW holders to redeem compute credits.
- **Incentive Shift**: The reduced pool favors only high-confidence enterprise donors; smaller commitments feed the general compute contractor rewards (part of the remaining API donor allocation) that align with GPU-hour contributions.

### Compute Agents

- Receive 50% of base emission (after validator/node slices) via execution rewards tied directly to task fees and proof quality.
- Early tasks enjoy a 2x multiplier for the first quarter, encouraging rapid onboarding of high-performance GPUs and API clients.
- A dynamic queue targets sub-minute latency for standard tasks but allows proofs to be snapped into optimistic verification when load spikes.

### Community Stewards

- Treasury, ecosystem, and governance funds share the remaining 5% of emissions, supporting education, audits, and grants to extend the manageable base into future feature sets.

## Tokenomics

- **Token Name**: PAW
- **Symbol**: `PAW`
- **Total Supply**: 50,000,000 PAW (allocations trimmed 30% from their prior amounts while keeping the global cap unchanged; the residual 15M PAW is held in the Reserve for future adoption boosts or burns).
- **Decimals**: 18

| Category | Adjusted Allocation | Purpose | Notes |
|----------|---------------------|---------|-------|
| Public Sale | 7,000,000 | Initial liquidity and decentralization | 70% of the prior 10M allocation |
| Mining & Node Rewards | 10,500,000 | Incentivize validators and node operators | 70% of the prior 15M allocation |
| API Donor Rewards | 8,400,000 | API key minutes, GPU-hour donors, and compute partners | 70% of the prior 12M allocation |
| Team & Advisors | 3,500,000 | Core team and ecosystem advisors | 70% of the prior 5M allocation |
| Foundation Treasury | 3,500,000 | Development, audits, partnerships | 70% of the prior 5M allocation |
| Ecosystem Fund | 2,100,000 | Grants, tooling, community programs | 70% of the prior 3M allocation |
| Reserve for Future Adoption | 15,000,000 | Strategic reserve for future launches, burns, or incentives | The 30% supply trimmed from each category is parked here until governance unlocks it |

### Emission Schedule

- Year 1: 2,870 PAW/day (70% of the previously planned 4,100/day) with a built-in ramp that multiplies this emission by 1.5x for the first 180 days to stimulate validators and node operators.
- Year 2: 1,435 PAW/day (halved) with built-in price-oracle gating so that emissions only drop further when PAW's market value rises by at least 25%.
- Year 3: 717 PAW/day (halved) and beyond: the halving cadence continues toward ~65 PAW/day by Year 6 while protocol fees (burn + treasury) soak the delta.
- After Year 10: Base emission reaches zero; ongoing participation pays out through tx fees, task fees, and premium verification bonuses.

The halving schedule guarantees that rewards shrink as PAW appreciates, reinforcing a value-led incentive system rather than constant output.

## Reward Model

- **Early Adoption**: Validators, node operators, and compute agents enjoy multipliers on the first six months of launch, after which rewards step down in scheduled increments.
- **Validators**: Capture 30% of emission plus fee rebates; their increased share ensures the mainnet remains secure even with the trimmed supply.
- **Node Operators**: Benefit from the Node Operator Reserve (1.5M PAW) that pays uptime bonuses and supports hardware refresh grants.
- **API Key/Minutes Pool**: Shrunk to 3M PAW with tight vesting, emphasizing quality over quantity while preserving the security guarantee for donated keys.
- **Scaling Down**: Each yearly halving reduces on-chain rewards, and long-term stake-weighted governance can trigger additional reductions once the PAW market price crosses consensus thresholds.

## DEX, Wallets, and Multi-Device UX

- **Built-in DEX Layer**: Launch the chain with a native automated market maker and atomic swap primitives. Liquidity pools pair PAW with partner assets, and every on-chain swap uses the same secure compute proofs to avoid MEV exploits.
- **Multi-Wallet Ecosystem**: Support desktop/browser wallets (extension + native client) plus smartphone wallets that share the same seed phrase, LP staking positions, and swap history through encrypted cloud sync. Each wallet includes multi-token dashboards and QR-based linking.
- **GUI Strategy**: Deliver a responsive React/Vue interface for desktop and browser use cases, plus lightweight Progressive Web App shells for mobile that tap device secure storage. Native builds expose QR onboarding, push approval flows, and WalletConnect/QR/NFC confirmations.
- **Mobile Onboarding**: Focus on QR-code-driven account creation, biometric unlocks, and in-app tutorials. Provide one-tap staking/swapping, camera-based QR trustless connect, and wallet recovery via secret phrases or social account recovery manager.
- **Atomic Swaps & Liquidity Pools**: Implement cross-chain atomic swap contracts from day one so users can swap PAW with wrapped assets without relying on custodial bridges. Pool incentives reward early liquidity providers with bonus emissions that taper as pool depth grows.

## Governance Expansion

- **Broad Participation Voting**: Expand voting outside the initial validator set by allowing any staker (including small holders) to participate through delegated voting groups selected via VRF. Each epoch selects a representative committee sampled from the entire staked population, throttling power consolidation.
- **Universal Delegate Pools**: Enable “Delegate Pools” where retail holders can opt into pooled voting that mirrors the early validator multiplier schedule, letting them influence Guardian DAO decisions without running a node.
- **Mobile-Friendly Voting**: Voting interfaces integrate with the multi-device GUI so holders can participate through browser, desktop, or smartphone with biometric sign-off and QR challenges for high-value proposals, ensuring secure yet accessible governance.

## Roadmap & Improvement Path

1. **Q1 Launch** - Deploy the controller chain with 25 validators, launch the API key aggregation pilot, and audit the reward contracts.
2. **Q2 Adoption** - Open compute agent registration, publish SDKs, and lock the first 100,000 GPU-hours behind on-chain attestations.
3. **Q3 Governance** - Activate the Guardian DAO, fund the Node Operator Reserve, and add telemetry for reward scaling.
4. **Q4 Extension** - Gateways for future compute shards, zkML rollouts, and bridging remain configurable options; node operators can toggle them once usage metrics justify the added complexity.

Each step holds configurable knobs so that the network can stay lean; governors can vote to unlock additional shards, prove systems, or API providers only when demand warrants it.

## Governance & Security

- **Guardian DAO** governs protocol upgrades, treasury spending, and reward adjustments, ensuring modular improvements stay aligned with the community.
- **Security Posture** combines layered attestations (TEE, cryptographic destruction of keys), immutable audits, and a rapid-response plan that can slash or pause emissions within minutes if a breach occurs.
- **Market-Responsive Rewards**: Oracles feed PAW price data to reward curves so that emissions automatically shrink as value rises, keeping supply tight and making early participation more lucrative than late entry.

## Conclusion

PAW refocuses the original PoAIW ambitions into a manageable blockchain with tightened supply, elevated validator/node operator compensation, and a disciplined yet extensible incentive model. The lowered API Key/minutes rewards and the rebalanced token distribution deliver a sober initial release while leaving room for future compute, privacy, and interoperability upgrades.
