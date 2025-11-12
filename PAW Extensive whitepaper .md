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

## Fee Economics & Gas Mechanism

### Gas Model Overview

PAW employs a computational cost model where each operation consumes a specific number of gas units. Users pay for transactions by multiplying the gas consumed by their chosen gas price (denominated in PAW):

- **Per-operation gas units**: Each operation has a fixed computational cost measured in gas units.
- **Gas price in PAW**: Market-driven pricing that fluctuates based on network demand.
- **Total fee calculation**: `total_fee = gas_used × gas_price`
- **Example**: A simple transfer consuming 21,000 gas at 0.001 PAW/gas costs 0.021 PAW.

This metered approach ensures that computational resources are fairly allocated while preventing network spam through economic disincentives.

### Dynamic Fee Mechanism (EIP-1559 Style)

PAW implements a dynamic fee adjustment mechanism inspired by Ethereum's EIP-1559, automatically balancing network capacity with user demand:

- **Base fee**: Algorithmically adjusted based on block fullness, targeting optimal network utilization.
- **Target block utilization**: 50% capacity (50M gas per block) ensures consistent performance headroom.
- **Base fee adjustment**: Increases by 12.5% when blocks exceed 50% fullness, decreases by 12.5% when blocks fall below 50%.
- **Priority tip**: User-specified additional payment to incentivize faster inclusion during congestion.
- **Total fee structure**: `total_fee = (base_fee + priority_tip) × gas_used`
- **Maximum fee per block**: 100M gas units (200% of target) prevents catastrophic congestion.

This mechanism creates predictable fee markets while maintaining network responsiveness during demand spikes.

### Fee Distribution

Transaction fees are strategically allocated to balance deflationary pressure, validator incentives, and ecosystem development:

- **50% burned**: Permanently removed from circulation, creating deflationary pressure that increases as network usage grows.
- **30% to block validator**: Direct compensation for consensus participation and block production.
- **20% to treasury**: Funds ecosystem development, audits, partnerships, and community programs.
- **Governance adjustability**: Distribution ratios can be modified through Guardian DAO proposals to respond to changing network conditions.

This tri-partite split ensures that transaction activity simultaneously reduces supply, rewards security providers, and funds long-term sustainability.

### Gas Cost Table

Sample operations and their associated gas costs provide transparency for developers and users:

| Operation | Gas Cost | Notes |
|-----------|----------|-------|
| Simple transfer | 21,000 gas | Base cost for transferring PAW between accounts |
| Contract deployment | 53,000 gas + 200 per byte | Initial deployment plus bytecode storage costs |
| Contract call | 21,000 gas + contract logic | Base invocation plus execution-specific costs |
| DEX swap | ~150,000 gas | Typical automated market maker swap operation |
| Stake/unstake | 75,000 gas | Validator delegation or withdrawal operations |
| Governance vote | 50,000 gas | On-chain proposal voting and signature verification |

These costs are calibrated to discourage frivolous transactions while keeping legitimate usage affordable.

### Fee Optimizer Integration

PAW integrates AI-assisted fee optimization through `external/crypto/ai/fee_optimizer.py`, providing intelligent recommendations for users:

- **Current mempool congestion**: Real-time analysis of pending transaction queue depth and gas price distribution.
- **Recent fee history**: Statistical modeling of successful transaction fees across recent blocks.
- **Transaction priority levels**: Three-tier system (high/normal/low) matching user urgency to economic cost.
- **Expected confirmation time**: Probabilistic estimates linking fee levels to inclusion likelihood.
- **Example recommendation**: Normal priority → 0.002 PAW/gas with 95% confidence of 8-second confirmation (two blocks).

The optimizer continuously learns from network patterns, helping users avoid overpaying during normal periods while ensuring fast inclusion during congestion.

### Economic Impact

The fee mechanism's deflationary effects scale with network adoption, creating powerful long-term tokenomics:

- **Burn rate at capacity**: 50% of 100M gas/block × 0.001 PAW/gas × 10,800 blocks/day = 540 PAW burned daily at full utilization.
- **Year 1 offset**: At the Year 1 emission rate of 2,870 PAW/day, full-capacity burn offsets approximately 19% of new issuance.
- **Deflationary threshold**: As network usage grows beyond ~5,400 PAW/day in fees, burn exceeds emissions, creating net deflationary conditions.
- **Value appreciation**: Reduced circulating supply combined with increasing utility demand creates upward price pressure, further incentivizing early adoption.

This design ensures that PAW becomes increasingly scarce as the network succeeds, rewarding early validators and holders while maintaining economic security.

### Comparison to Other Chains

PAW positions itself in the accessibility gap between ultra-cheap chains and premium secure networks:

| Network | Typical Transaction Cost | Positioning |
|---------|-------------------------|-------------|
| Ethereum | $2-50 per transaction | High security, high cost barrier to entry |
| Solana | $0.00025 per transaction | Very low cost, optimistic security model |
| **PAW** | **$0.01-0.10 per transaction** | **Accessible pricing with enhanced security** |

PAW delivers approximately 10x cost savings versus Ethereum while maintaining premium security guarantees compared to ultra-low-cost chains. This balance makes PAW ideal for AI computation tasks that require both economic efficiency and verifiable execution.

## DEX, Wallets, and Multi-Device UX

- **Built-in DEX Layer**: Launch the chain with a native automated market maker and atomic swap primitives. Liquidity pools pair PAW with partner assets, and every on-chain swap uses the same secure compute proofs to avoid MEV exploits.
- **Multi-Wallet Ecosystem**: Support desktop/browser wallets (extension + native client) plus smartphone wallets that share the same seed phrase, LP staking positions, and swap history through encrypted cloud sync. Each wallet includes multi-token dashboards and QR-based linking.
- **GUI Strategy**: Deliver a responsive React/Vue interface for desktop and browser use cases, plus lightweight Progressive Web App shells for mobile that tap device secure storage. Native builds expose QR onboarding, push approval flows, and WalletConnect/QR/NFC confirmations.
- **Mobile Onboarding**: Focus on QR-code-driven account creation, biometric unlocks, and in-app tutorials. Provide one-tap staking/swapping, camera-based QR trustless connect, and wallet recovery via secret phrases or social account recovery manager.
- **Atomic Swaps & Liquidity Pools**: Implement cross-chain atomic swap contracts from day one so users can swap PAW with wrapped assets without relying on custodial bridges. Pool incentives reward early liquidity providers with bonus emissions that taper as pool depth grows.
- **Fernet-Encrypted Storage**: All wallet secrets leverage Fernet-authenticated encryption derived from on-device passwords. Fernet replaces the legacy XOR obfuscation with AES-128 in CBC mode plus HMAC-SHA256, ensuring both confidentiality and integrity of private keys, drafts, and WalletConnect session tokens.

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

## Oracle Design & Price Feeds

The PAW protocol relies on accurate, tamper-resistant price data to adjust emissions dynamically as market value rises. A robust oracle infrastructure ensures that reward curves respond to real-world adoption without introducing centralization risks or manipulation vectors.

### Oracle Architecture

PAW deploys a decentralized oracle network that eliminates single points of failure while maintaining rapid update cycles:

- **Multi-Source Aggregation**: Price data flows from multiple independent sources, aggregated on-chain to resist outliers and manipulation attempts.
- **Median Price Calculation**: The protocol computes a weighted median rather than a simple average, making the system resilient to extreme values from compromised or lagging data sources.
- **Update Frequency**: Price feeds refresh every 100 blocks (approximately 7 minutes given the 4-second block time), balancing responsiveness with cost efficiency.
- **Decentralized Submission**: No single entity controls price submissions; the network distributes oracle responsibilities across the validator set and, in later phases, integrates with proven third-party oracle networks.

### Oracle Solution: Hybrid Approach

PAW adopts a phased oracle strategy that prioritizes rapid deployment in Phase 1 while planning for institutional-grade infrastructure in Phase 2:

**Phase 1: Validator-Operated Price Oracles**

The initial launch leverages the existing validator set to bootstrap price discovery without external dependencies:

- Each validator submits price data aggregated from at least three distinct sources (exchanges, aggregators, or DEXs).
- On-chain aggregation uses a weighted median algorithm, with weights proportional to each validator's stake, ensuring that well-capitalized validators carry proportional influence.
- Price updates require submissions from at least 2/3 of active validators to be considered valid, mirroring the BFT finality threshold.
- This approach minimizes external dependencies during the critical launch phase and allows the network to stabilize before introducing third-party oracle integrations.

**Phase 2: Integration with Chainlink Price Feeds**

Once the mainnet matures and demonstrates sustained transaction volume, the Guardian DAO can activate Chainlink integration:

- Chainlink's decentralized oracle network provides battle-tested infrastructure with a proven track record across major DeFi protocols.
- Multiple independent node operators fetch data from diverse sources and submit cryptographic proofs of data authenticity.
- PAW can leverage Chainlink's existing price feeds for major trading pairs, reducing the burden on validators and increasing data reliability.
- The transition to Phase 2 is governance-gated, requiring a Guardian DAO vote with 2/3 approval and a 7-day timelock to ensure community consensus.

### Data Sources

Price data aggregates from a diversified mix of centralized exchanges, decentralized protocols, and aggregator services:

**Centralized Exchanges:**
- Binance (BTC/USDT, ETH/USDT pairs as proxies for market sentiment)
- Coinbase Pro (regulated USD pairs for price anchoring)
- Kraken (institutional volume indicators)
- OKX (Asian market price discovery)

**Decentralized Exchanges:**
- Uniswap (Ethereum-based liquidity depth)
- PancakeSwap (BSC-based trading activity)

**Price Aggregators:**
- CoinGecko (multi-exchange composite pricing)
- CoinMarketCap (global volume-weighted averages)

The protocol requires a minimum of five distinct sources for each price update. Outliers exceeding two standard deviations from the median are automatically discarded to prevent wash trading, flash loan attacks, or exchange-specific anomalies from corrupting the feed.

### Oracle Security Model

PAW's oracle design anticipates and mitigates common attack vectors:

**Attack Vectors:**
- **Price Manipulation**: Flash loans, wash trading, or coordinated market manipulation attempts that temporarily distort exchange prices.
- **Oracle Node Compromise**: Validators submitting fraudulent data to manipulate emissions or exploit price-dependent mechanisms.
- **Network Partition Attacks**: Censorship or network splits that prevent sufficient validators from submitting price data.

**Mitigations:**
- **Time-Weighted Average Price (TWAP)**: Price submissions use a 10-minute TWAP rather than spot prices, smoothing out short-term volatility and rendering flash loan attacks ineffective.
- **Volume-Weighted Median**: Sources with higher trading volume carry greater weight in median calculations, prioritizing liquid markets over thin order books.
- **Anomaly Detection**: Price submissions deviating more than 20% from the previous consensus trigger automatic rejection, preventing sudden shocks from bad data.
- **Validator Slashing**: Validators who submit price data that deviates significantly from the final consensus face slashing penalties of 1% of their staked PAW, creating strong economic disincentives for manipulation.
- **Emergency Circuit Breaker**: The Guardian DAO maintains emergency pause authority to freeze oracle updates in extreme market conditions or detected attacks, protecting the protocol until manual intervention resolves the issue.

### Oracle Incentives

Validators earn tangible rewards for accurate, timely price submissions:

- **Base Reward**: Validators receive 0.5% of the treasury allocation annually, distributed proportionally to the number of accurate price submissions.
- **Accuracy Measurement**: Each submission is scored against the final consensus median; submissions within 1% of consensus earn full rewards, while those within 1-5% earn reduced rewards.
- **Penalties**: Validators whose submissions deviate more than 5% from consensus three times within a 30-day period enter a 24-hour jail period, during which they forfeit oracle rewards and cannot participate in price submissions.
- **Long-Term Alignment**: Oracle rewards vest over 90 days, ensuring that validators remain incentivized to maintain data quality over extended periods rather than gaming short-term payouts.

### Use Cases

PAW's oracle infrastructure serves multiple protocol functions beyond the core emission adjustment mechanism:

**Reward Curve Adjustment:**
The primary use case referenced in the whitepaper's emission schedule (line 74) ties oracle price data directly to emissions. As PAW's market value rises by 25% or more, the halving schedule accelerates, reducing daily emissions and tightening supply. This market-responsive mechanism ensures that early participants capture higher rewards while later entrants benefit from a more valuable, scarcer token.

**DEX Price Discovery:**
The built-in DEX layer (referenced on line 177) uses oracle feeds to bootstrap liquidity pool pricing, preventing front-running and ensuring that initial LP positions reflect fair market value.

**Collateral Ratios:**
Future lending protocols or synthetic asset systems can leverage oracle feeds to set collateral requirements, liquidation thresholds, and interest rates based on real-time PAW valuations.

**Stablecoin Pegs:**
If the ecosystem introduces PAW-backed stablecoins or wrapped assets, oracle feeds provide the reference prices necessary to maintain pegs and execute redemptions.

### Price Feed Contract Interface

The oracle system exposes a standardized smart contract interface for on-chain queries and submissions:

```
// Query Operations
query get_latest_price() -> {
    price: Decimal,        // Latest consensus price in USDT
    timestamp: u64,        // Block height of last update
    confidence: u8         // Confidence score (0-100) based on validator participation
}

query get_historical_price(timestamp: u64) -> Price {
    price: Decimal,
    timestamp: u64,
    sources: Vec<String>   // List of sources used in this update
}

// Submission Operations (Validator-only)
msg submit_price {
    sources: Vec<PriceData>,  // Array of source prices with identifiers
    signature: Signature      // Validator's cryptographic signature
}
```

This interface allows smart contracts to query current prices, historical data, and confidence scores, enabling sophisticated DeFi applications to build on top of PAW's oracle infrastructure.

### Failure Handling

The protocol includes multiple fallback mechanisms to maintain operation during oracle outages or network disruptions:

**Insufficient Validator Participation:**
If fewer than 2/3 of validators submit price data for a given update window, the protocol reuses the last valid consensus price, marking it as stale. Stale prices remain valid for up to 24 hours, ensuring that short-term disruptions do not halt critical protocol functions.

**Extended Staleness:**
If price data remains stale for more than 24 hours, the protocol automatically freezes reward adjustments, preventing emissions from operating on outdated market information. An on-chain alert notifies the Guardian DAO, which can investigate the root cause and, if necessary, manually intervene.

**Emergency Fallback:**
In extreme situations such as prolonged network partitions, coordinated attacks, or global exchange outages, the Guardian DAO retains emergency override authority to manually set a reference price. This fallback requires a 2/3 supermajority vote and includes a mandatory 7-day timelock before execution, ensuring that emergency powers cannot be abused for short-term manipulation.

**Graceful Degradation:**
Rather than halting all protocol operations during oracle failures, PAW prioritizes core functionality: block production, transaction processing, and staking continue normally, while only price-dependent features (emission adjustments, DEX pool initializations) pause until valid data resumes.

### Oracle Governance

The Guardian DAO exercises comprehensive control over oracle parameters and policies:

**DAO Authority:**
- Addition or removal of trusted data sources (exchanges, aggregators, oracle providers)
- Adjustment of update frequency (100-block default, tunable from 50 to 500 blocks)
- Configuration of deviation thresholds (anomaly detection, slashing parameters)
- Emergency pause authority to freeze oracle updates during crises
- Transition triggers for Phase 2 Chainlink integration

**Proposal Requirements:**
All oracle governance proposals require 2/3 approval from Guardian DAO members with a mandatory 7-day timelock before execution. This delay provides the community time to review changes, identify potential issues, and, in extreme cases, coordinate emergency responses if a malicious proposal passes.

**Transparency Guarantees:**
Every oracle update, validator submission, and governance action is logged on-chain with full provenance. Community members can audit historical price data, validate submission accuracy, and track slashing events through block explorers and dedicated oracle monitoring dashboards.

## Economic Security Analysis

Economic security measures the cost an adversary must incur to attack the network. PAW uses staking, slashing, and validator rewards to align economic incentives and ensure that attacking the network is prohibitively expensive relative to the potential gains.

### Byzantine Fault Tolerance (BFT) Security

PAW uses Tendermint BFT consensus, which requires 2/3+ honest validators for correctness and safety. The attack thresholds are:

- **Halt attack**: Control >1/3 of stake to prevent finality and halt the chain
- **Invalid block attack**: Control >2/3 of stake to create and finalize invalid blocks

At genesis (50M PAW total supply, assuming 30% staked = 15M PAW):
- **Halt attack cost**: 5M PAW (~$500K at $0.10/PAW)
- **Invalid block attack cost**: 10M PAW (~$1M at $0.10/PAW)

As PAW price increases, the cost to attack grows exponentially, making the network more secure over time.

### Slashing Conditions & Penalties

The protocol enforces strict slashing to penalize malicious or negligent behavior:

**Double-signing (Byzantine behavior)**:
- **Penalty**: 5% of validator stake + permanent jail (tombstone)
- **Example**: A validator with 200K PAW stake loses 10K PAW
- **Delegators**: Also slashed at the same rate to ensure aligned incentives

**Downtime (liveness failure)**:
- **Threshold**: Miss >5% of blocks in a 10,000 block window (~11 hours at 4s blocks)
- **First penalty**: 0.1% of stake + 24-hour jail
- **Repeated failures**: After 3 jails, validator loses 1% of stake and is removed from the active set

**Bad oracle data**:
- **Penalty**: 1% stake for >5% price deviation from consensus median
- **Strike system**: 3 strikes result in permanent removal from oracle committee
- **Application**: Ensures market-responsive rewards use accurate PAW price data

**Invalid compute attestation**:
- **Penalty**: 10% stake for verified fraud proofs showing incorrect compute results
- **Consequence**: Loss of compute agent registration privileges
- **Detection**: Challenge period allows full nodes to submit fraud proofs

### 51% Attack Analysis

A 51% attack (or more precisely, a >2/3 stake attack) would allow an adversary to produce invalid blocks and finalize them.

**Attack cost at launch**:
- Acquire 10M PAW (~$1M at initial valuation)
- Ongoing opportunity cost of staking rewards foregone
- Risk of total stake loss via social consensus recovery

**Detection**:
- Invalid state transitions are detected by full nodes
- Blocks violating protocol rules are rejected
- Network alerts trigger if conflicting blocks are signed

**Recovery**:
- Social consensus coordinates to fork the chain
- Attacker stake is slashed in the recovery fork
- Honest validators continue on the canonical chain

**Long-term cost**: As stake participation grows toward the target of 50%+ of total supply, the attack cost increases exponentially (from $1M at launch to potentially $50M+ at mature adoption).

### Nothing-at-Stake Mitigation

**Problem**: In pure PoS systems, validators could sign multiple forks without economic cost, enabling costless attacks on finality.

**Solution**:
- **Slashing for double-signing**: 5% stake penalty for signing conflicting blocks
- **Monitoring**: Full nodes track all validator signatures and submit fraud proofs
- **Incentive**: Fraud proof submitters earn 20% of slashed stake as a reward

This ensures validators have strong economic incentives to sign only valid blocks on the canonical chain.

### Long-Range Attack Prevention

**Problem**: An attacker could acquire old validator private keys and rewrite chain history from an early checkpoint.

**Solutions**:
- **Weak subjectivity**: Light clients checkpoint every 14 days and reject forks diverging before the checkpoint
- **Unbonding period**: 21-day unstaking period prevents attackers from exiting stake and then rewriting history
- **Social consensus**: Community validates and publishes canonical checkpoints
- **Slashing history**: Slashed validators cannot rewrite history as their keys are publicly known to be compromised

**Cost**: Attacker must acquire a supermajority of old validator keys, which is impractical after keys are destroyed or slashed validators are removed.

### Validator Cartel Formation

**Risk**: Validators could collude to extract MEV (Maximal Extractable Value), censor transactions, or manipulate governance.

**Mitigations**:
- **VRF-based committee selection**: Governance committees are selected via Verifiable Random Functions, preventing coordination
- **Delegator choice**: Delegators can switch validators with 21-day unbonding, punishing misbehaving validators
- **Slash risk**: Evidence of collusion results in 5% stake loss for all participants
- **Reputation tracking**: Public dashboard shows validator uptime, commission rates, and slash history

**Economic disincentive**: Cartels lose delegations, which reduces their share of the 30% validator emission rewards. Honest behavior is more profitable long-term.

### Sybil Attack Resistance

**Attack**: Create many validator identities to gain outsized influence.

**Defense**:
- **Minimum stake requirement**: 10,000 PAW per validator (~$1,000 at $0.10/PAW)
- **Cost**: Creating 25 validators requires 250K PAW (~$25K minimum investment)
- **Limitation**: Voting power is proportional to stake, not identity count

Sybil attacks are economically infeasible because stake is the security resource, not identities.

### Eclipse Attack on Light Clients

**Attack**: Isolate a light client from honest peers and feed it false block headers.

**Defense**:
- **Multiple connections**: Light clients connect to at least 5 full nodes by default
- **Header comparison**: Clients compare headers across peers and reject outliers
- **Alert on conflict**: Clients alert users when receiving conflicting headers from different peers

**Detection**: Full nodes publish fraud proofs for conflicting validator signatures, which light clients can verify.

### Compute Plane Attack Vectors

**TEE Compromise**:
- **Risk**: Side-channel attacks on Trusted Execution Environments (e.g., Intel SGX)
- **Mitigation**: Use AWS Nitro Enclaves (no known practical attacks), employ defense in depth
- **Fallback**: Optimistic verification with fraud proofs and 24-hour challenge period

**Proof Withholding**:
- **Risk**: Compute agent accepts a task but withholds the proof to deny service
- **Mitigation**: 6-hour timeout triggers task reassignment and 5% slash of agent stake
- **Incentive**: Agents earn rewards only upon successful proof submission

**Result Manipulation**:
- **Risk**: Agent submits incorrect compute results to benefit from payment without doing work
- **Mitigation**: Optimistic verification with fraud proofs; 10% slash if fraud is proven
- **Challenge period**: 24 hours for anyone to submit fraud proof disputing result

### Economic Sustainability

The protocol is designed for long-term economic sustainability through multiple mechanisms:

**Emission halving**:
- Rewards decrease 50% annually, reducing selling pressure on PAW
- Year 1: 2,870 PAW/day → Year 6: ~65 PAW/day
- After Year 10: Zero base emissions, all rewards from fees

**Fee burn**:
- 50% of transaction and task fees are burned
- Network becomes net deflationary at high usage levels
- Reduces circulating supply over time

**Treasury funding**:
- 20% of fees + 5% of emissions fund development
- Ensures sustainable protocol development without external fundraising
- Treasury governed by Guardian DAO

**Break-even analysis**:
- At 10M daily gas usage (~100 TPS sustained), fee revenue covers validator operating costs
- Higher usage increases validator profitability and security budget

### Cost of Attack Summary

| Attack Type | Minimum Cost | Success Probability | Detection Time | Recovery |
|-------------|--------------|---------------------|----------------|----------|
| 51% Attack (>2/3 stake) | 10M PAW ($1M) | Low (social consensus) | Immediate | Fork & slash |
| Halt (>1/3 stake) | 5M PAW ($500K) | Medium | Immediate | Social recovery |
| Double-sign | 200K PAW stake | Low (slashed) | 1 block (4s) | Auto-slash |
| Long-range | Impractical | Very low | 14 days | Checkpoints |
| Validator cartel | Reputation loss | Medium | Days-weeks | Delegator exit |
| TEE compromise | $50K+ exploit | Low (Nitro) | 24 hours | Fraud proof |
| Sybil (25 validators) | 250K PAW ($25K) | Very low | N/A | Stake-weighted voting |
| Eclipse attack | Network-level | Low | Minutes | Multi-peer validation |

### Comparison to Other Chains

PAW's security budget and attack costs scale with adoption:

- **Ethereum**: ~$20B+ attack cost (most robust PoS chain)
- **Solana**: ~$2B attack cost (based on stake distribution)
- **PAW (Year 1)**: ~$1M attack cost at genesis
- **PAW (Mature)**: $50M+ attack cost as adoption increases and stake grows

**Security budget**:
- Year 1: 30% of 2,870 PAW/day = 861 PAW/day (~$86/day at $0.10)
- As PAW price increases, security budget grows proportionally
- Target: 50%+ of supply staked to maximize attack cost

The market-responsive reward system ensures that as PAW value increases, the network becomes exponentially more secure while maintaining sustainable validator incentives.

## Conclusion

PAW refocuses the original PoAIW ambitions into a manageable blockchain with tightened supply, elevated validator/node operator compensation, and a disciplined yet extensible incentive model. The lowered API Key/minutes rewards and the rebalanced token distribution deliver a sober initial release while leaving room for future compute, privacy, and interoperability upgrades.
