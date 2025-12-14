/**
 * Metrics Display Component
 * Displays network metrics and system health
 */

import blockchainService from '../services/blockchain.js';
import monitoringService from '../services/monitoring.js';

class MetricsDisplay {
    constructor() {
        this.metrics = {
            blockHeight: '0',
            tps: '0',
            peerCount: '0',
            consensus: 'Unknown'
        };
    }

    /**
     * Initialize the component
     */
    init() {
        // Listen to monitoring events
        monitoringService.on('blockUpdate', (block) => this.updateBlockMetrics(block));
        monitoringService.on('metricsUpdate', (metrics) => this.updateSystemMetrics(metrics));

        // Initial update
        this.updateMetrics();

        // Periodic updates
        setInterval(() => this.updateMetrics(), 5000);
    }

    /**
     * Update all metrics
     */
    async updateMetrics() {
        try {
            // Update block metrics
            const latestBlock = await blockchainService.getLatestBlock();
            if (latestBlock) {
                this.updateBlockMetrics(latestBlock);
            }

            // Update TPS
            const tps = await monitoringService.calculateTPS();
            this.updateElement('tps', tps);

            // Update peer count
            const peerCount = await monitoringService.getPeerCount();
            this.updateElement('peer-count', peerCount);

            // Update consensus status
            const consensus = await monitoringService.getConsensusStatus();
            this.updateElement('consensus', consensus);
        } catch (error) {
            console.error('Error updating metrics:', error);
        }
    }

    /**
     * Update block-related metrics
     */
    updateBlockMetrics(block) {
        if (block && block.height) {
            this.updateElement('block-height', block.height);
            this.metrics.blockHeight = block.height;
        }
    }

    /**
     * Update system metrics (CPU, Memory, Disk)
     */
    updateSystemMetrics(metrics) {
        if (!metrics) return;

        // Update CPU
        const cpuFill = document.getElementById('cpu-usage');
        const cpuPercent = document.getElementById('cpu-percent');
        if (cpuFill && cpuPercent) {
            const cpu = Math.round(metrics.cpu);
            cpuFill.style.width = cpu + '%';
            cpuPercent.textContent = cpu + '%';
        }

        // Update Memory
        const memoryFill = document.getElementById('memory-usage');
        const memoryPercent = document.getElementById('memory-percent');
        if (memoryFill && memoryPercent) {
            const memory = Math.round(metrics.memory);
            memoryFill.style.width = memory + '%';
            memoryPercent.textContent = memory + '%';
        }

        // Update Disk
        const diskFill = document.getElementById('disk-usage');
        const diskPercent = document.getElementById('disk-percent');
        if (diskFill && diskPercent) {
            const disk = Math.round(metrics.disk);
            diskFill.style.width = disk + '%';
            diskPercent.textContent = disk + '%';
        }
    }

    /**
     * Update element text content
     */
    updateElement(id, value) {
        const element = document.getElementById(id);
        if (element) {
            element.textContent = value;
        }
    }

    /**
     * Format large numbers
     */
    formatNumber(num) {
        if (num >= 1000000) {
            return (num / 1000000).toFixed(2) + 'M';
        } else if (num >= 1000) {
            return (num / 1000).toFixed(2) + 'K';
        }
        return num.toString();
    }
}

export default new MetricsDisplay();
