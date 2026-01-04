const statusEl = document.getElementById('status');
const latestHeightEl = document.getElementById('latestHeight');
const blockTimeEl = document.getElementById('blockTime');
const tpsEl = document.getElementById('tps');
const validatorsEl = document.getElementById('validators');
const blocksEl = document.getElementById('blocks');

async function fetchJson(url) {
    const response = await fetch(url);
    if (!response.ok) {
        throw new Error(`Request failed: ${response.status}`);
    }
    return response.json();
}

function formatDuration(seconds) {
    if (!Number.isFinite(seconds)) {
        return '-';
    }
    return `${seconds.toFixed(2)}s`;
}

async function loadStats() {
    try {
        const [statusData, blocksData, validatorsData] = await Promise.all([
            fetchJson('https://testnet-rpc.poaiw.org/status'),
            fetchJson('https://testnet-rpc.poaiw.org/blockchain?minHeight=1&maxHeight=50'),
            fetchJson('https://testnet-rpc.poaiw.org/validators'),
        ]);

        const status = statusData.result || {};
        const syncInfo = status.sync_info || {};
        const latestHeight = Number(syncInfo.latest_block_height || 0);
        latestHeightEl.textContent = latestHeight || '-';

        const metas = blocksData.result?.block_metas || [];
        if (metas.length > 1) {
            const times = metas.map((meta) => new Date(meta.header.time).getTime());
            const sorted = times.sort((a, b) => a - b);
            const diffs = [];
            for (let i = 1; i < sorted.length; i += 1) {
                diffs.push((sorted[i] - sorted[i - 1]) / 1000);
            }
            const avg = diffs.reduce((sum, val) => sum + val, 0) / diffs.length;
            blockTimeEl.textContent = formatDuration(avg);

            const latestMeta = metas[0];
            const txCount = Number(latestMeta.num_txs || 0);
            tpsEl.textContent = avg > 0 ? (txCount / avg).toFixed(2) : '0.00';
        } else {
            blockTimeEl.textContent = '-';
            tpsEl.textContent = '-';
        }

        const validators = validatorsData.result?.validators || [];
        validatorsEl.textContent = validators.length.toString();

        blocksEl.innerHTML = metas.slice(0, 8).map((meta) => {
            const header = meta.header;
            return `
                <div class="block-item">
                    <span>#${header.height}</span>
                    <span>${meta.block_id.hash.slice(0, 12)}…</span>
                    <span>${new Date(header.time).toLocaleString()}</span>
                </div>
            `;
        }).join('');

        statusEl.textContent = 'Stats online';
        statusEl.style.background = 'rgba(88, 178, 108, 0.18)';
        statusEl.style.color = '#7ee39a';
    } catch (error) {
        statusEl.textContent = 'Stats unavailable';
        statusEl.style.background = 'rgba(255, 107, 107, 0.18)';
        statusEl.style.color = '#ff9e9e';
        blocksEl.textContent = 'Failed to load stats.';
    }
}

loadStats();
setInterval(loadStats, 15000);
