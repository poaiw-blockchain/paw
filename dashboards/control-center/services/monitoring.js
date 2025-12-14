/**
 * Monitoring Service
 * Handles real-time monitoring and metrics collection
 */

import CONFIG from '../config.js';
import blockchainService from './blockchain.js';

class MonitoringService {
    constructor() {
        this.intervals = {};
        this.callbacks = {};
        this.metrics = {
            cpu: 0,
            memory: 0,
            disk: 0
        };
    }

    /**
     * Start monitoring
     */
    startMonitoring() {
        this.startBlockMonitoring();
        this.startMetricsMonitoring();
        this.startEventsMonitoring();
    }

    /**
     * Stop all monitoring
     */
    stopMonitoring() {
        Object.values(this.intervals).forEach(interval => clearInterval(interval));
        this.intervals = {};
    }

    /**
     * Register callback for specific event type
     */
    on(event, callback) {
        if (!this.callbacks[event]) {
            this.callbacks[event] = [];
        }
        this.callbacks[event].push(callback);
    }

    /**
     * Emit event to all registered callbacks
     */
    emit(event, data) {
        if (this.callbacks[event]) {
            this.callbacks[event].forEach(callback => callback(data));
        }
    }

    /**
     * Start monitoring blocks
     */
    startBlockMonitoring() {
        if (this.intervals.blocks) {
            clearInterval(this.intervals.blocks);
        }

        const updateBlocks = async () => {
            try {
                const latestBlock = await blockchainService.getLatestBlock();
                if (latestBlock) {
                    this.emit('blockUpdate', latestBlock);
                }
            } catch (error) {
                console.error('Error monitoring blocks:', error);
            }
        };

        // Initial update
        updateBlocks();

        // Set interval
        this.intervals.blocks = setInterval(
            updateBlocks,
            CONFIG.updateIntervals.blockUpdates
        );
    }

    /**
     * Start monitoring system metrics
     */
    startMetricsMonitoring() {
        if (this.intervals.metrics) {
            clearInterval(this.intervals.metrics);
        }

        const updateMetrics = () => {
            // Simulate metrics (in production, these would come from actual system monitoring)
            this.metrics = {
                cpu: Math.random() * 100,
                memory: Math.random() * 100,
                disk: Math.random() * 100
            };

            this.emit('metricsUpdate', this.metrics);
        };

        // Initial update
        updateMetrics();

        // Set interval
        this.intervals.metrics = setInterval(
            updateMetrics,
            CONFIG.updateIntervals.metricsUpdates
        );
    }

    /**
     * Start monitoring events
     */
    startEventsMonitoring() {
        if (this.intervals.events) {
            clearInterval(this.intervals.events);
        }

        const updateEvents = async () => {
            try {
                const transactions = await blockchainService.getRecentTransactions(5);
                if (transactions && transactions.length > 0) {
                    transactions.forEach(tx => {
                        this.emit('newEvent', {
                            type: 'transaction',
                            message: `New ${tx.type} transaction: ${tx.hash}`,
                            timestamp: new Date().toISOString()
                        });
                    });
                }
            } catch (error) {
                console.error('Error monitoring events:', error);
            }
        };

        // Initial update
        updateEvents();

        // Set interval
        this.intervals.events = setInterval(
            updateEvents,
            CONFIG.updateIntervals.eventsUpdates
        );
    }

    /**
     * Calculate TPS (Transactions Per Second)
     */
    async calculateTPS() {
        try {
            const recentBlocks = await blockchainService.getRecentBlocks(10);
            if (recentBlocks.length < 2) return 0;

            const totalTxs = recentBlocks.reduce((sum, block) => sum + block.txCount, 0);
            const timeSpan = recentBlocks.length * 6; // Assuming 6 second block time
            return (totalTxs / timeSpan).toFixed(2);
        } catch (error) {
            console.error('Error calculating TPS:', error);
            return 0;
        }
    }

    /**
     * Get peer count
     */
    async getPeerCount() {
        try {
            // In production, this would query the actual node
            // For now, return a simulated value
            return Math.floor(Math.random() * 50) + 10;
        } catch (error) {
            console.error('Error getting peer count:', error);
            return 0;
        }
    }

    /**
     * Get consensus status
     */
    async getConsensusStatus() {
        try {
            const isConnected = blockchainService.isConnected;
            return isConnected ? 'Active' : 'Inactive';
        } catch (error) {
            console.error('Error getting consensus status:', error);
            return 'Unknown';
        }
    }

    /**
     * Get network health
     */
    async getNetworkHealth() {
        try {
            const isConnected = blockchainService.isConnected;
            const latestBlock = await blockchainService.getLatestBlock();

            if (!isConnected || !latestBlock) {
                return { status: 'error', message: 'Network disconnected' };
            }

            // Check if latest block is recent (within last minute)
            const blockTime = new Date(latestBlock.time);
            const now = new Date();
            const timeDiff = (now - blockTime) / 1000; // seconds

            if (timeDiff > 60) {
                return { status: 'warning', message: 'Block production delayed' };
            }

            return { status: 'success', message: 'Network healthy' };
        } catch (error) {
            console.error('Error checking network health:', error);
            return { status: 'error', message: 'Health check failed' };
        }
    }

    /**
     * Get current metrics
     */
    getMetrics() {
        return this.metrics;
    }

    /**
     * Add log entry
     */
    addLog(level, message) {
        const log = {
            level,
            message,
            timestamp: new Date().toISOString()
        };

        this.emit('newLog', log);
        return log;
    }

    /**
     * Monitor transaction
     */
    async monitorTransaction(txHash) {
        const checkTx = async () => {
            try {
                const response = await fetch(
                    `${blockchainService.config.restUrl}/cosmos/tx/v1beta1/txs/${txHash}`
                );
                const data = await response.json();

                if (data.tx_response) {
                    this.emit('transactionConfirmed', {
                        hash: txHash,
                        success: data.tx_response.code === 0,
                        height: data.tx_response.height
                    });
                    return true;
                }
                return false;
            } catch (error) {
                return false;
            }
        };

        // Poll for transaction confirmation
        const maxAttempts = 20;
        let attempts = 0;

        const pollInterval = setInterval(async () => {
            attempts++;
            const confirmed = await checkTx();

            if (confirmed || attempts >= maxAttempts) {
                clearInterval(pollInterval);
                if (!confirmed) {
                    this.emit('transactionTimeout', { hash: txHash });
                }
            }
        }, 3000);
    }
}

export default new MonitoringService();
