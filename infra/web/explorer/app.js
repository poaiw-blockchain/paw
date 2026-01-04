const statusEl = document.getElementById('status');
const chainIdEl = document.getElementById('chainId');
const latestHeightEl = document.getElementById('latestHeight');
const latestTimeEl = document.getElementById('latestTime');
const catchingUpEl = document.getElementById('catchingUp');
const blocksEl = document.getElementById('blocks');

async function loadStatus() {
    try {
        const response = await fetch('/api/status');
        if (!response.ok) {
            throw new Error('Status fetch failed');
        }
        const data = await response.json();
        chainIdEl.textContent = data.chain_id || '-';
        latestHeightEl.textContent = data.latest_height ?? '-';
        latestTimeEl.textContent = data.latest_block_time || '-';
        catchingUpEl.textContent = data.catching_up ? 'Yes' : 'No';

        statusEl.textContent = 'Node healthy';
        statusEl.style.background = 'rgba(88, 178, 108, 0.18)';
        statusEl.style.color = '#7ee39a';
    } catch (error) {
        statusEl.textContent = 'Node unreachable';
        statusEl.style.background = 'rgba(255, 107, 107, 0.18)';
        statusEl.style.color = '#ff9e9e';
    }
}

async function loadBlocks() {
    try {
        const response = await fetch('/api/blocks?limit=8');
        if (!response.ok) {
            throw new Error('Blocks fetch failed');
        }
        const data = await response.json();
        const blocks = data.blocks || [];
        if (blocks.length === 0) {
            blocksEl.textContent = 'No blocks available yet.';
            return;
        }
        blocksEl.innerHTML = blocks.map((block) => {
            return `
                <div class="block-item">
                    <span>#${block.height}</span>
                    <span>${block.hash.slice(0, 12)}…</span>
                    <span>${new Date(block.time).toLocaleString()}</span>
                </div>
            `;
        }).join('');
    } catch (error) {
        blocksEl.textContent = 'Failed to load blocks.';
    }
}

loadStatus();
loadBlocks();

setInterval(() => {
    loadStatus();
    loadBlocks();
}, 15000);
