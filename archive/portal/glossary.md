# Glossary

Definitions of key terms used in PAW Blockchain.

## A

**Address**
A unique identifier (starting with "paw") that represents an account on PAW Blockchain. Example: `paw1xxxxxxxxxxxxxxxxxxxxxxxxxx`

**AMM (Automated Market Maker)**
A decentralized exchange protocol that uses liquidity pools and mathematical formulas to price assets, rather than traditional order books.

**APR (Annual Percentage Rate)**
The yearly rate of return on staked tokens, not accounting for compounding.

**APY (Annual Percentage Yield)**
The yearly rate of return on staked tokens, including compounding effects.

## B

**BFT (Byzantine Fault Tolerance)**
The ability of a distributed system to reach consensus even when some nodes are unreliable or malicious.

**Block**
A collection of transactions bundled together and added to the blockchain every 4 seconds on PAW.

**Block Height**
The sequential number of a block in the blockchain. Genesis block = height 0.

**Block Proposer**
The validator selected to create the next block in the blockchain.

## C

**Chain ID**
Unique identifier for a blockchain network. PAW mainnet: `paw-mainnet-1`, testnet: `paw-testnet-1`

**Commission**
The percentage fee validators charge delegators for staking services. Typical: 5-10%.

**Consensus**
The process by which validators agree on the next block to add to the blockchain.

**Cosmos SDK**
The modular framework used to build PAW Blockchain, enabling custom modules and IBC compatibility.

## D

**Delegator**
A token holder who stakes PAW tokens with a validator to earn rewards.

**DEX (Decentralized Exchange)**
A peer-to-peer marketplace where users can trade cryptocurrencies without intermediaries. PAW has a built-in DEX.

**Double Signing**
When a validator signs two different blocks at the same height. Results in 5% slash penalty.

**DPoS (Delegated Proof of Stake)**
A consensus mechanism where token holders vote for validators to secure the network.

## F

**Fees**
The cost to process transactions on PAW, paid in PAW tokens. Typical fee: ~0.001 PAW.

**Finality**
The point at which a transaction cannot be reversed. On PAW: 1 block (4 seconds).

## G

**Gas**
A measure of computational effort required to execute a transaction. Higher complexity = more gas.

**Genesis**
The first block of a blockchain (block height 0) containing initial state and parameters.

**Governance**
The process by which PAW token holders vote on network changes and proposals.

## I

**IBC (Inter-Blockchain Communication)**
A protocol enabling different blockchains to transfer assets and data between each other.

**Impermanent Loss**
Temporary loss of funds when providing liquidity to an AMM pool, compared to simply holding the tokens.

## J

**Jailed**
Status of a validator that has been penalized for misbehavior and temporarily removed from the active set.

## L

**Liquidity Pool**
A collection of tokens locked in a smart contract, used to facilitate trading on a DEX.

**LP Tokens (Liquidity Provider Tokens)**
Tokens received when providing liquidity to a pool, representing your share of the pool.

## M

**Mainnet**
The primary, production blockchain network where real transactions occur (vs. testnet).

**Mnemonic**
A 24-word phrase used to recover a wallet. Also called "seed phrase" or "recovery phrase".

**Mempool**
The waiting area for unconfirmed transactions before they're included in a block.

**Module**
A self-contained piece of functionality in Cosmos SDK. PAW has modules for bank, staking, DEX, etc.

## N

**Node**
A computer running PAW blockchain software, maintaining a copy of the blockchain.

**NoWithVeto**
A strong rejection vote used in governance for spam or malicious proposals.

## O

**Oracle**
A service that provides external data (like prices) to the blockchain. PAW has a decentralized oracle module.

## P

**Proposal**
A formal suggestion for changing network parameters, submitted through governance.

**Pruning**
The process of deleting old blockchain data to save disk space.

## Q

**Quorum**
The minimum participation rate required for a governance vote to be valid. PAW: 40%.

## R

**Redelegate**
Moving staked tokens from one validator to another without unbonding period.

**RPC (Remote Procedure Call)**
An interface for interacting with the blockchain programmatically.

## S

**Seed Node**
A node that helps new nodes find peers when joining the network.

**Sentry Node**
A node that sits between validators and the public internet, protecting validators from DDoS attacks.

**Slashing**
Penalty imposed on validators (and their delegators) for misbehavior like double-signing or downtime.

**Slippage**
The difference between expected and actual transaction price, common in DEX trades.

**Staking**
Locking tokens to support network security and earn rewards.

**State Sync**
A fast synchronization method that downloads a recent state snapshot instead of all historical blocks.

## T

**TEE (Trusted Execution Environment)**
Secure area of a processor that protects code and data. Used in PAW's compute module.

**Tendermint**
The BFT consensus engine that powers PAW Blockchain.

**Testnet**
A test blockchain network for development and testing, using test tokens with no real value.

**Transaction (TX)**
An operation that changes the blockchain state, like sending tokens or voting.

**TWAP (Time-Weighted Average Price)**
An average price calculated over a specific time period, used to prevent price manipulation.

## U

**Unbonding**
The 21-day process of unstaking tokens, during which they earn no rewards.

**Uptime**
The percentage of time a validator has been online and signing blocks.

## V

**Validator**
A node operator responsible for proposing and validating blocks in exchange for rewards.

**Voting Period**
The 14-day window during which stakers can vote on a governance proposal.

**Voting Power**
A validator's influence in consensus, proportional to their total stake (self + delegations).

## W

**Wallet**
Software that stores private keys and allows users to manage their PAW tokens.

**WebSocket**
A protocol for real-time, bi-directional communication, used for live blockchain data.

## Common Abbreviations

| Abbreviation | Full Term |
|--------------|-----------|
| AMM | Automated Market Maker |
| APR | Annual Percentage Rate |
| APY | Annual Percentage Yield |
| BFT | Byzantine Fault Tolerance |
| CEX | Centralized Exchange |
| CLI | Command Line Interface |
| DAO | Decentralized Autonomous Organization |
| DEX | Decentralized Exchange |
| DPoS | Delegated Proof of Stake |
| IBC | Inter-Blockchain Communication |
| LP | Liquidity Provider |
| RPC | Remote Procedure Call |
| SDK | Software Development Kit |
| TEE | Trusted Execution Environment |
| TX | Transaction |
| TWAP | Time-Weighted Average Price |

---

**Still confused?** Check the [FAQ](/faq) or ask in [Discord](https://discord.gg/pawblockchain)
