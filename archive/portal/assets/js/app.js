/**
 * PAW Documentation Portal - Main Application
 * Handles routing, content loading, search, theming, and interactivity
 */

// Configuration
const CONFIG = {
    version: '1.0.0',
    contentPath: 'content/',
    searchIndexPath: 'search-index.json',
    defaultPage: 'getting-started',
    theme: localStorage.getItem('paw-theme') || 'dark'
};

// State Management
const state = {
    currentPage: null,
    searchIndex: null,
    config: null,
    history: []
};

// Initialize Application
document.addEventListener('DOMContentLoaded', () => {
    initializeApp();
});

async function initializeApp() {
    try {
        // Load configuration
        await loadConfig();

        // Load search index
        await loadSearchIndex();

        // Setup theme
        initializeTheme();

        // Setup event listeners
        setupEventListeners();

        // Setup routing
        setupRouting();

        // Load initial page
        const hash = window.location.hash.slice(1) || CONFIG.defaultPage;
        await loadPage(hash);

        console.log('PAW Documentation Portal initialized successfully');
    } catch (error) {
        console.error('Failed to initialize application:', error);
        showError('Failed to load documentation. Please refresh the page.');
    }
}

// Configuration Loading
async function loadConfig() {
    try {
        const response = await fetch('config.json');
        state.config = await response.json();
        CONFIG.contentPath = state.config.contentPath || CONFIG.contentPath;
    } catch (error) {
        console.warn('Config file not found, using defaults');
        state.config = getDefaultConfig();
    }
}

function getDefaultConfig() {
    return {
        siteName: 'PAW Blockchain Documentation',
        version: '1.0.0',
        contentPath: 'content/',
        languages: ['en'],
        defaultLanguage: 'en'
    };
}

// Search Index Loading
async function loadSearchIndex() {
    try {
        const response = await fetch(CONFIG.searchIndexPath);
        state.searchIndex = await response.json();
    } catch (error) {
        console.warn('Search index not found, generating from content');
        state.searchIndex = await generateSearchIndex();
    }
}

async function generateSearchIndex() {
    const pages = [
        'getting-started', 'user-guide', 'developer-guide',
        'api-reference', 'tutorials', 'faq'
    ];

    const index = [];

    for (const page of pages) {
        try {
            const content = await fetchContent(page);
            const text = stripMarkdown(content);
            const words = text.split(/\s+/).slice(0, 500);

            index.push({
                id: page,
                title: formatTitle(page),
                content: words.join(' '),
                url: `#${page}`
            });
        } catch (error) {
            console.warn(`Failed to index page: ${page}`);
        }
    }

    return index;
}

// Theme Management
function initializeTheme() {
    document.body.classList.toggle('dark-theme', CONFIG.theme === 'dark');
    document.body.classList.toggle('light-theme', CONFIG.theme === 'light');
    updateThemeIcon();
}

function toggleTheme() {
    CONFIG.theme = CONFIG.theme === 'dark' ? 'light' : 'dark';
    localStorage.setItem('paw-theme', CONFIG.theme);
    initializeTheme();
}

function updateThemeIcon() {
    const sunIcon = document.querySelector('.sun-icon');
    const moonIcon = document.querySelector('.moon-icon');

    if (CONFIG.theme === 'dark') {
        sunIcon.style.display = 'none';
        moonIcon.style.display = 'block';
    } else {
        sunIcon.style.display = 'block';
        moonIcon.style.display = 'none';
    }
}

// Event Listeners
function setupEventListeners() {
    // Theme toggle
    document.getElementById('themeToggle')?.addEventListener('click', toggleTheme);

    // Search
    document.getElementById('searchBtn')?.addEventListener('click', openSearch);
    document.getElementById('closeSearch')?.addEventListener('click', closeSearch);
    document.getElementById('searchInput')?.addEventListener('input', handleSearch);

    // Mobile menu
    document.getElementById('mobileMenuToggle')?.addEventListener('click', toggleMobileMenu);

    // Version selector
    document.getElementById('versionSelector')?.addEventListener('click', showVersionSelector);

    // Click outside to close search
    document.getElementById('searchModal')?.addEventListener('click', (e) => {
        if (e.target.id === 'searchModal') closeSearch();
    });

    // Keyboard shortcuts
    document.addEventListener('keydown', handleKeyboardShortcuts);

    // Navigation links
    document.querySelectorAll('a[href^="#"]').forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            const page = link.getAttribute('href').slice(1);
            navigateTo(page);
        });
    });

    // Scroll handling for TOC
    window.addEventListener('scroll', updateActiveHeading);
}

// Routing
function setupRouting() {
    window.addEventListener('hashchange', () => {
        const page = window.location.hash.slice(1);
        if (page && page !== state.currentPage) {
            loadPage(page);
        }
    });
}

function navigateTo(page) {
    window.location.hash = page;
    closeMobileMenu();
}

// Page Loading
async function loadPage(pageName) {
    try {
        showLoading();

        const content = await fetchContent(pageName);
        const html = marked.parse(content);

        renderContent(html, pageName);
        updateNavigation(pageName);
        generateTableOfContents();
        renderBreadcrumbs(pageName);

        state.currentPage = pageName;
        state.history.push(pageName);

        // Highlight code blocks
        document.querySelectorAll('pre code').forEach((block) => {
            hljs.highlightElement(block);
        });

        // Scroll to top
        window.scrollTo(0, 0);

        hideLoading();
    } catch (error) {
        console.error('Failed to load page:', error);
        showError(`Failed to load ${formatTitle(pageName)}`);
        hideLoading();
    }
}

async function fetchContent(pageName) {
    try {
        const response = await fetch(`${CONFIG.contentPath}${pageName}.md`);
        if (!response.ok) throw new Error('Content not found');
        return await response.text();
    } catch (error) {
        // Fallback to default content
        return getDefaultContent(pageName);
    }
}

function getDefaultContent(pageName) {
    const defaultContents = {
        'getting-started': `# Getting Started with PAW Blockchain

Welcome to PAW Blockchain! This guide will help you get started with PAW.

## What is PAW?

PAW is a high-performance blockchain platform designed for decentralized finance (DeFi), featuring:

- **High Throughput**: Process thousands of transactions per second
- **Low Fees**: Minimal transaction costs
- **Built-in DEX**: Native decentralized exchange
- **Staking**: Earn rewards by securing the network
- **Governance**: Community-driven decision making

## Quick Start

### 1. Install a Wallet

Choose from our available wallets:

- **Browser Extension**: For quick access while browsing
- **Desktop Wallet**: Full-featured application for Windows, macOS, and Linux
- **Mobile Wallet**: iOS and Android apps

### 2. Get Some PAW Tokens

- Use the [testnet faucet](#faucet) for testing
- Purchase from supported exchanges
- Receive from another user

### 3. Make Your First Transaction

\`\`\`bash
# Using the CLI
pawd tx bank send <from> <to> <amount>upaw --chain-id paw-1
\`\`\`

## Next Steps

- [Set up your wallet](#wallet-setup)
- [Learn about staking](#staking-guide)
- [Explore the DEX](#using-dex)
- [Join governance](#governance)

## Need Help?

- Check our [FAQ](#faq)
- Join our [Discord community](https://discord.gg/DBHTc2QV)
- Read the [User Guide](#user-guide)`,

        'user-guide': `# User Guide

Complete guide for PAW users.

## Wallet Setup

### Browser Extension

1. Install from Chrome Web Store or Firefox Add-ons
2. Create new wallet or import existing
3. Securely back up your recovery phrase

### Desktop Wallet

Download for your platform:
- Windows: \`.exe\` installer
- macOS: \`.dmg\` disk image
- Linux: \`.AppImage\`, \`.deb\`, or \`.rpm\`

### Mobile Wallet

Available on:
- iOS: App Store
- Android: Google Play Store

## Sending and Receiving Tokens

### Receiving

1. Open your wallet
2. Copy your address or scan QR code
3. Share with sender

### Sending

1. Click "Send"
2. Enter recipient address
3. Enter amount
4. Review transaction
5. Confirm and sign

## Using the DEX

### Swapping Tokens

\`\`\`javascript
// Example swap
const swap = await client.dex.swap({
  poolId: "1",
  tokenIn: "100000upaw",
  tokenOutMinAmount: "95000uusdc",
  slippage: 0.5
});
\`\`\`

### Adding Liquidity

Provide liquidity to earn fees:

1. Select token pair
2. Enter amounts
3. Preview pool share
4. Add liquidity
5. Receive LP tokens

## Security Best Practices

- ✅ Never share your private key or recovery phrase
- ✅ Use strong passwords
- ✅ Enable biometric authentication
- ✅ Verify addresses before sending
- ✅ Keep software updated
- ❌ Don't fall for phishing scams
- ❌ Don't use public WiFi for transactions`,

        'developer-guide': `# Developer Guide

Build on PAW Blockchain.

## Development Environment Setup

### Prerequisites

- Node.js 18+
- Go 1.21+
- 

### Install PAW CLI

\`\`\`bash
# Clone repository
 clone https://example.com/paw-chain/paw
cd paw

# Build
make install

# Verify
pawd version
\`\`\`

## SDK Usage

### JavaScript/TypeScript

\`\`\`bash
npm install @paw-chain/sdk
\`\`\`

\`\`\`javascript
import { PAWClient, Wallet } from '@paw-chain/sdk';

// Connect to network
const client = new PAWClient({
  rpcEndpoint: 'https://rpc.pawchain.io',
  chainId: 'paw-1'
});

// Create wallet
const wallet = await Wallet.generate();

// Send transaction
const result = await client.bank.send({
  from: wallet.address,
  to: 'paw1...',
  amount: '1000000upaw'
});
\`\`\`

### Python

\`\`\`bash
pip install paw-sdk
\`\`\`

\`\`\`python
from paw import PAWClient, Wallet

# Connect
client = PAWClient(
    rpc_endpoint='https://rpc.pawchain.io',
    chain_id='paw-1'
)

# Create wallet
wallet = Wallet.generate()

# Send transaction
result = await client.bank.send(
    from_address=wallet.address,
    to_address='paw1...',
    amount='1000000upaw'
)
\`\`\`

### Go

\`\`\`go
import (
    "example.com/paw-chain/paw/sdk/client"
    "example.com/paw-chain/paw/sdk/wallet"
)

// Create client
client := client.NewPAWClient("https://rpc.pawchain.io")

// Create wallet
w, _ := wallet.Generate()

// Send transaction
result, _ := client.Bank.Send(ctx, &bank.SendRequest{
    From: w.Address,
    To: "paw1...",
    Amount: "1000000upaw",
})
\`\`\`

## Testing

\`\`\`bash
# Run unit tests
make test

# Run integration tests
make test-integration

# Check coverage
make test-coverage
\`\`\`

## API Integration

See [API Reference](#api-reference) for complete API documentation.`,

        'api-reference': `# API Reference

Complete API documentation for PAW Blockchain.

## Base URL

\`\`\`
Mainnet: https://api.pawchain.io
Testnet: https://testnet-api.pawchain.io
Local: http://localhost:1317
\`\`\`

## Authentication

Most API endpoints are public. For transaction broadcasting, you need to sign with your private key.

## Bank Module

### Get Balance

\`\`\`http
GET /cosmos/bank/v1beta1/balances/{address}
\`\`\`

**Response:**
\`\`\`json
{
  "balances": [
    {
      "denom": "upaw",
      "amount": "1000000"
    }
  ]
}
\`\`\`

### Send Tokens

\`\`\`http
POST /cosmos/tx/v1beta1/txs
\`\`\`

**Request:**
\`\`\`json
{
  "tx": {
    "body": {
      "messages": [{
        "@type": "/cosmos.bank.v1beta1.MsgSend",
        "from_address": "paw1...",
        "to_address": "paw1...",
        "amount": [{"denom": "upaw", "amount": "1000000"}]
      }]
    }
  }
}
\`\`\`

## DEX Module

### Get Pools

\`\`\`http
GET /paw/dex/v1/pools
\`\`\`

### Swap Tokens

\`\`\`http
POST /paw/dex/v1/swap
\`\`\`

## Staking Module

### Get Validators

\`\`\`http
GET /cosmos/staking/v1beta1/validators
\`\`\`

### Delegate

\`\`\`http
POST /cosmos/staking/v1beta1/delegate
\`\`\`

## WebSocket API

\`\`\`javascript
const ws = new WebSocket('wss://ws.pawchain.io');

ws.on('message', (data) => {
  console.log('New block:', data);
});
\`\`\`

## Rate Limits

- 100 requests per minute per IP
- 1000 requests per hour per IP

## Error Codes

| Code | Description |
|------|-------------|
| 400 | Bad Request |
| 404 | Not Found |
| 429 | Rate Limited |
| 500 | Internal Error |`,

        'tutorials': `# Tutorials

Step-by-step guides for common tasks.

## Tutorial 1: Creating Your First Wallet

**Duration:** 5 minutes

### Step 1: Install Wallet

Download the browser extension or desktop wallet.

### Step 2: Create New Wallet

Click "Create New Wallet" and follow the prompts.

### Step 3: Backup Recovery Phrase

Write down your 24-word recovery phrase and store it safely.

### Step 4: Verify Backup

Re-enter your recovery phrase to verify.

### Video Tutorial

[Watch on YouTube](#)

## Tutorial 2: Swapping Tokens on DEX

**Duration:** 10 minutes

### Step 1: Connect Wallet

Open the DEX interface and connect your wallet.

### Step 2: Select Tokens

Choose the tokens you want to swap.

### Step 3: Enter Amount

Enter the amount to swap.

### Step 4: Review and Confirm

Review the exchange rate and fees, then confirm.

## Tutorial 3: Staking PAW Tokens

**Duration:** 15 minutes

### Step 1: Choose a Validator

Browse the list of validators and select one.

### Step 2: Delegate Tokens

Enter the amount to stake.

### Step 3: Confirm Transaction

Review and sign the delegation transaction.

### Step 4: Track Rewards

Monitor your staking rewards in the dashboard.

## Tutorial 4: Building a dApp

**Duration:** 60 minutes

Complete guide to building your first decentralized application on PAW.

### Prerequisites
- Node.js installed
- Basic JavaScript knowledge
- PAW wallet

### Steps
1. Set up development environment
2. Install SDK
3. Create project
4. Implement smart contract logic
5. Build frontend
6. Test on testnet
7. Deploy to mainnet

[View Full Tutorial](#)

## Video Tutorials

### Getting Started Series
- Introduction to PAW (5:00)
- Setting Up Your Wallet (8:00)
- Making Your First Transaction (6:00)

### Advanced Series
- Building Smart Contracts (45:00)
- DEX Integration (30:00)
- Validator Operations (60:00)`,

        'faq': `# Frequently Asked Questions

## General Questions

### What is PAW Blockchain?

PAW is a high-performance blockchain platform designed for DeFi applications with built-in DEX, staking, and governance features.

### How do I get started?

Follow our [Getting Started Guide](#getting-started) to create a wallet and make your first transaction.

### Is PAW open source?

Yes! All our code is open source and available on [](https://example.com/paw-chain).

## Wallet Questions

### How do I create a wallet?

You can create a wallet using our browser extension, desktop app, or mobile app. See [Wallet Setup](#wallet-setup).

### What if I lose my recovery phrase?

Your recovery phrase is the only way to restore your wallet. Store it securely and never share it with anyone.

### Can I have multiple wallets?

Yes, you can create as many wallets as you need.

## Transaction Questions

### How long do transactions take?

Transactions typically confirm in 6-7 seconds (one block time).

### What are the fees?

Transaction fees are minimal, typically less than $0.01 USD.

### Can I cancel a transaction?

Once broadcast, transactions cannot be cancelled. Always verify before confirming.

## Staking Questions

### How do I stake PAW tokens?

See our [Staking Guide](#staking-guide) for detailed instructions.

### What is the staking APY?

APY varies based on total staked amount and validator commission. Current rates are displayed in the staking dashboard.

### How long is the unbonding period?

21 days. During this time, your tokens cannot be used or transferred.

## DEX Questions

### How does the DEX work?

PAW's built-in DEX uses an automated market maker (AMM) model with liquidity pools.

### What is slippage?

Slippage is the difference between expected and actual price due to market movement during the transaction.

### How do I provide liquidity?

See [Adding Liquidity](#using-dex) in the user guide.

## Developer Questions

### What programming languages are supported?

- JavaScript/TypeScript
- Python
- Go

### Where can I find the API documentation?

See our [API Reference](#api-reference).

### Is there a testnet?

Yes, testnet is available at testnet-api.pawchain.io

### How do I get testnet tokens?

Use our [testnet faucet](#faucet).

## Governance Questions

### How do I vote on proposals?

Any token holder can vote on active proposals through the governance interface.

### What types of proposals are there?

- Text proposals
- Parameter changes
- Software upgrades
- Community spend

### How long is the voting period?

Typically 7 days, but this can vary by proposal type.

## Security Questions

### Is PAW secure?

Yes, PAW uses industry-standard cryptography and has undergone security audits.

### What should I do if I suspect fraud?

Contact our security team immediately at security@pawchain.io

### How can I report a bug?

See our [Bug Bounty Program](https://docs.pawchain.io/bug-bounty) for details on responsible disclosure.

## Support

### How do I get help?

- Check this FAQ
- Read the documentation
- Join our [Discord](https://discord.gg/DBHTc2QV)
- Visit our [forum](https://forum.pawchain.io)

### Is there customer support?

Community support is available on Discord and Telegram. For critical issues, email support@pawchain.io`
    };

    return defaultContents[pageName] || `# ${formatTitle(pageName)}\n\nContent coming soon...`;
}

// Content Rendering
function renderContent(html, pageName) {
    const contentDiv = document.getElementById('content');
    contentDiv.innerHTML = html;
    contentDiv.setAttribute('data-page', pageName);
}

// Navigation
function updateNavigation(pageName) {
    // Update active link in sidebar
    document.querySelectorAll('.nav-link').forEach(link => {
        link.classList.remove('active');
        if (link.getAttribute('href') === `#${pageName}`) {
            link.classList.add('active');
        }
    });
}

// Table of Contents
function generateTableOfContents() {
    const content = document.getElementById('content');
    const headings = content.querySelectorAll('h2, h3');
    const tocNav = document.getElementById('tocNav');

    if (headings.length === 0) {
        tocNav.innerHTML = '<p class="toc-empty">No headings found</p>';
        return;
    }

    let tocHtml = '<ul class="toc-list">';

    headings.forEach((heading, index) => {
        const id = `heading-${index}`;
        heading.id = id;

        const level = heading.tagName === 'H2' ? 'toc-level-2' : 'toc-level-3';
        const text = heading.textContent;

        tocHtml += `<li class="${level}"><a href="#${id}" class="toc-link">${text}</a></li>`;
    });

    tocHtml += '</ul>';
    tocNav.innerHTML = tocHtml;

    // Add click handlers
    tocNav.querySelectorAll('a').forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            const targetId = link.getAttribute('href').slice(1);
            const target = document.getElementById(targetId);
            if (target) {
                target.scrollIntoView({ behavior: 'smooth' });
            }
        });
    });
}

function updateActiveHeading() {
    const headings = document.querySelectorAll('#content h2, #content h3');
    let currentHeading = null;

    headings.forEach(heading => {
        const rect = heading.getBoundingClientRect();
        if (rect.top <= 100) {
            currentHeading = heading;
        }
    });

    if (currentHeading) {
        document.querySelectorAll('.toc-link').forEach(link => {
            link.classList.remove('active');
        });

        const activeLink = document.querySelector(`.toc-link[href="#${currentHeading.id}"]`);
        if (activeLink) {
            activeLink.classList.add('active');
        }
    }
}

// Breadcrumbs
function renderBreadcrumbs(pageName) {
    const breadcrumbs = document.querySelector('.breadcrumbs');
    if (breadcrumbs) breadcrumbs.remove();

    const template = document.getElementById('breadcrumbsTemplate');
    const clone = template.content.cloneNode(true);
    const list = clone.querySelector('.breadcrumb-list');

    const segments = pageName.split('-');
    let path = '';

    segments.forEach((segment, index) => {
        path += (index > 0 ? '-' : '') + segment;
        const li = document.createElement('li');

        if (index === segments.length - 1) {
            li.textContent = formatTitle(segment);
        } else {
            const a = document.createElement('a');
            a.href = `#${path}`;
            a.textContent = formatTitle(segment);
            li.appendChild(a);
        }

        list.appendChild(li);
    });

    const content = document.getElementById('content');
    content.insertBefore(clone, content.firstChild);
}

// Search Functionality
function openSearch() {
    document.getElementById('searchModal').classList.add('active');
    document.getElementById('searchInput').focus();
}

function closeSearch() {
    document.getElementById('searchModal').classList.remove('active');
    document.getElementById('searchInput').value = '';
    document.getElementById('searchResults').innerHTML = '';
}

function handleSearch(e) {
    const query = e.target.value.toLowerCase().trim();
    const results = document.getElementById('searchResults');

    if (query.length < 2) {
        results.innerHTML = '';
        return;
    }

    const matches = searchContent(query);

    if (matches.length === 0) {
        results.innerHTML = '<div class="search-no-results">No results found</div>';
        return;
    }

    let html = '<div class="search-results-list">';
    matches.slice(0, 10).forEach(match => {
        html += `
            <a href="${match.url}" class="search-result" onclick="closeSearch()">
                <h4>${match.title}</h4>
                <p>${highlightMatch(match.excerpt, query)}</p>
            </a>
        `;
    });
    html += '</div>';

    results.innerHTML = html;
}

function searchContent(query) {
    if (!state.searchIndex) return [];

    const results = [];

    state.searchIndex.forEach(item => {
        const titleMatch = item.title.toLowerCase().includes(query);
        const contentMatch = item.content.toLowerCase().includes(query);

        if (titleMatch || contentMatch) {
            const excerptStart = Math.max(0, item.content.toLowerCase().indexOf(query) - 50);
            const excerptEnd = Math.min(item.content.length, excerptStart + 200);

            results.push({
                title: item.title,
                url: item.url,
                excerpt: item.content.substring(excerptStart, excerptEnd),
                score: titleMatch ? 2 : 1
            });
        }
    });

    return results.sort((a, b) => b.score - a.score);
}

function highlightMatch(text, query) {
    const regex = new RegExp(`(${query})`, 'gi');
    return text.replace(regex, '<mark>$1</mark>');
}

// Mobile Menu
function toggleMobileMenu() {
    document.getElementById('mobileMenu').classList.toggle('active');
}

function closeMobileMenu() {
    document.getElementById('mobileMenu').classList.remove('active');
}

// Version Selector
function showVersionSelector() {
    const versions = ['v1.0.0', 'v0.9.0', 'v0.8.0'];
    const html = versions.map(v => `<a href="#" onclick="selectVersion('${v}')">${v}</a>`).join('');

    // Create dropdown (simplified)
    alert('Version selector - Available versions: ' + versions.join(', '));
}

function selectVersion(version) {
    CONFIG.version = version;
    document.getElementById('versionSelector').textContent = version;
}

// Keyboard Shortcuts
function handleKeyboardShortcuts(e) {
    // Ctrl+K or Cmd+K for search
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
        openSearch();
    }

    // Escape to close modals
    if (e.key === 'Escape') {
        closeSearch();
        closeMobileMenu();
    }
}

// Utility Functions
function formatTitle(str) {
    return str.split('-').map(word =>
        word.charAt(0).toUpperCase() + word.slice(1)
    ).join(' ');
}

function stripMarkdown(markdown) {
    return markdown
        .replace(/#{1,6}\s/g, '')
        .replace(/\*\*([^*]+)\*\*/g, '$1')
        .replace(/\*([^*]+)\*/g, '$1')
        .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
        .replace(/`([^`]+)`/g, '$1')
        .replace(/```[\s\S]*?```/g, '');
}

function showLoading() {
    const content = document.getElementById('content');
    content.innerHTML = '<div class="loading">Loading...</div>';
}

function hideLoading() {
    const loading = document.querySelector('.loading');
    if (loading) loading.remove();
}

function showError(message) {
    const content = document.getElementById('content');
    content.innerHTML = `
        <div class="error-message">
            <h2>Error</h2>
            <p>${message}</p>
            <button onclick="location.reload()" class="btn-primary">Reload Page</button>
        </div>
    `;
}

// Export for testing
if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        formatTitle,
        stripMarkdown,
        searchContent,
        generateSearchIndex
    };
}
