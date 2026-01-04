---
layout: home

hero:
  name: PAW Blockchain
  text: A Lean Layer-1 Blockchain
  tagline: Built-in DEX, secure compute aggregation, and mobile-ready wallets for rapid adoption
  image:
    src: /logo.svg
    alt: PAW Blockchain Logo
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: Developer Guide
      link: /developer/quick-start
    - theme: alt
      link: <REPO_URL>

features:
  - icon: 🚀
    title: Manageable Architecture
    details: Simplified 4-second BFT-DPoS consensus with compact validator set for rapid deployment and early adoption.

  - icon: 💱
    title: Native DEX
    details: Built-in decentralized exchange with atomic swap primitives and AMM pools for day-one liquidity.

  - icon: 🔒
    title: Secure Compute Plane
    details: TEE-protected API key aggregation for verified task execution and secure computation.

  - icon: 📱
    title: Multi-Device Wallets
    details: Desktop, mobile, and web wallets with QR code support, biometric authentication, and seamless UX.

  - icon: 📉
    title: Deflationary Economics
    details: Annual emission halving with fee-based burn mechanisms for sustainable long-term value.

  - icon: 🎁
    title: Early Adoption Rewards
    details: Boosted incentives for validators and node operators during the first 180 days.

  - icon: 🧩
    title: Modular Growth Path
    details: Governance-enabled hooks for zkML, sharding, and cross-chain bridges as the network evolves.

  - icon: 🗳️
    title: Open Governance
    details: Delegated voting with Guardian DAO oversight for decentralized decision-making.

  - icon: ⚡
    title: High Performance
    details: 1000+ transactions per second with sub-second finality and minimal latency.

  - icon: 🛡️
    title: Security First
    details: Multiple security audits, comprehensive testing, and best-practice cryptography throughout.

  - icon: 🌐
    title: Developer Friendly
    details: Complete SDKs for JavaScript, Python, and Go with extensive documentation and examples.

  - icon: 📊
    title: Real-time Analytics
    details: Comprehensive dashboards for staking, governance, and validator operations.
---

## Quick Start

Get up and running with PAW in under 5 minutes:

```bash
# Clone the repository
 clone <REPO_URL>
cd paw

# Install dependencies
go mod download
npm install

# Start local node
./scripts/bootstrap-node.sh
./build/pawd start
```

## What's New

### Version 1.0 - Production Ready 🎉

- ✅ Native DEX with full AMM functionality
- ✅ Comprehensive wallet suite (desktop, mobile, web)
- ✅ Validator and staking dashboards
- ✅ Governance portal with proposal management
- ✅ Testnet faucet with rate limiting
- ✅ Complete developer SDKs
- ✅ 92% test coverage with rigorous security testing

[View Changelog](/changelog) | [Read Release Notes](<REPO_URL>/releases)

## Community

Join our growing community of developers, validators, and blockchain enthusiasts:

- 💬 [Discord Server](https://discord.gg/DBHTc2QV) - Chat with the community
- 🐦 [Twitter](https://twitter.com/pawblockchain) - Latest updates and announcements
- 📝 [Blog](https://blog.pawblockchain.io) - Tutorials and deep dives

## Documentation Sections

<div class="doc-sections">
  <div class="section">
    <h3>👤 User Guide</h3>
    <p>Everything you need to use PAW Blockchain</p>
    <ul>
      <li><a href="/guide/getting-started">Getting Started</a></li>
      <li><a href="/guide/wallets">Creating a Wallet</a></li>
      <li><a href="/guide/dex">Using the DEX</a></li>
      <li><a href="/guide/staking">Staking Guide</a></li>
      <li><a href="/guide/governance">Governance</a></li>
    </ul>
  </div>

  <div class="section">
    <h3>💻 Developer Guide</h3>
    <p>Build applications on PAW</p>
    <ul>
      <li><a href="/developer/quick-start">Quick Start</a></li>
      <li><a href="/developer/javascript-sdk">JavaScript SDK</a></li>
      <li><a href="/developer/python-sdk">Python SDK</a></li>
      <li><a href="/developer/api">API Reference</a></li>
      <li><a href="/developer/smart-contracts">Smart Contracts</a></li>
    </ul>
  </div>

  <div class="section">
    <h3>🖥️ Validator Guide</h3>
    <p>Run and maintain validator nodes</p>
    <ul>
      <li><a href="/validator/setup">Setup Guide</a></li>
      <li><a href="/validator/operations">Operations</a></li>
      <li><a href="/validator/security">Security</a></li>
      <li><a href="/validator/monitoring">Monitoring</a></li>
      <li><a href="/validator/troubleshooting">Troubleshooting</a></li>
    </ul>
  </div>

  <div class="section">
    <h3>📚 Reference</h3>
    <p>Technical specifications and details</p>
    <ul>
      <li><a href="/reference/architecture">Architecture</a></li>
      <li><a href="/reference/tokenomics">Tokenomics</a></li>
      <li><a href="/reference/network-specs">Network Specs</a></li>
      <li><a href="/faq">FAQ</a></li>
      <li><a href="/glossary">Glossary</a></li>
    </ul>
  </div>
</div>

<style>
.doc-sections {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 2rem;
  margin: 3rem 0;
}

.section {
  padding: 1.5rem;
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  transition: all 0.3s;
}

.section:hover {
  border-color: var(--vp-c-brand);
  box-shadow: 0 4px 12px rgba(62, 175, 124, 0.1);
}

.section h3 {
  margin-top: 0;
  color: var(--vp-c-brand);
}

.section ul {
  padding-left: 1.2rem;
  margin: 1rem 0 0 0;
}

.section li {
  margin: 0.5rem 0;
}
</style>
