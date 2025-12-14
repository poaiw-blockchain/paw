/**
 * PAW Control Center - Analytics Service
 * Integrates with explorer analytics API endpoints
 */

import CONFIG from '../config.js';
import monitoringService from './monitoring.js';

class AnalyticsService {
    constructor() {
        this.baseUrl = null;
        this.cache = new Map();
        this.cacheDuration = 30000; // 30 seconds
    }

    /**
     * Initialize analytics service with current network
     */
    init() {
        const network = localStorage.getItem('selectedNetwork') || 'local';
        this.baseUrl = CONFIG.networks[network]?.analyticsUrl;

        if (!this.baseUrl) {
            console.warn('Analytics URL not configured for network:', network);
        }
    }

    /**
     * Update base URL when network changes
     */
    updateNetwork(network) {
        this.baseUrl = CONFIG.networks[network]?.analyticsUrl;
        this.cache.clear(); // Clear cache on network change
    }

    /**
     * Generic fetch with caching
     */
    async fetchWithCache(endpoint, cacheKey) {
        // Check cache
        const cached = this.cache.get(cacheKey);
        if (cached && Date.now() - cached.timestamp < this.cacheDuration) {
            return cached.data;
        }

        // Fetch from API
        try {
            const response = await fetch(`${this.baseUrl}${endpoint}`);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const data = await response.json();

            // Update cache
            this.cache.set(cacheKey, {
                data,
                timestamp: Date.now()
            });

            return data;
        } catch (error) {
            monitoringService.addLog('error', `Analytics API error: ${error.message}`);
            throw error;
        }
    }

    /**
     * Get network health metrics
     */
    async getNetworkHealth() {
        if (!this.baseUrl) {
            return null;
        }

        return this.fetchWithCache(
            CONFIG.analytics.networkHealth,
            'network-health'
        );
    }

    /**
     * Get transaction volume data
     * @param {string} period - Time period (1h, 24h, 7d, 30d)
     */
    async getTransactionVolume(period = '24h') {
        if (!this.baseUrl) {
            return null;
        }

        return this.fetchWithCache(
            `${CONFIG.analytics.transactionVolume}?period=${period}`,
            `transaction-volume-${period}`
        );
    }

    /**
     * Get DEX analytics
     */
    async getDEXAnalytics() {
        if (!this.baseUrl) {
            return null;
        }

        return this.fetchWithCache(
            CONFIG.analytics.dexAnalytics,
            'dex-analytics'
        );
    }

    /**
     * Get address growth data
     * @param {string} period - Time period (7d, 30d, 90d)
     */
    async getAddressGrowth(period = '30d') {
        if (!this.baseUrl) {
            return null;
        }

        return this.fetchWithCache(
            `${CONFIG.analytics.addressGrowth}?period=${period}`,
            `address-growth-${period}`
        );
    }

    /**
     * Get gas analytics
     * @param {string} period - Time period (24h, 7d, 30d)
     */
    async getGasAnalytics(period = '24h') {
        if (!this.baseUrl) {
            return null;
        }

        return this.fetchWithCache(
            `${CONFIG.analytics.gasAnalytics}?period=${period}`,
            `gas-analytics-${period}`
        );
    }

    /**
     * Get validator performance
     * @param {string} period - Time period (24h, 7d, 30d)
     */
    async getValidatorPerformance(period = '24h') {
        if (!this.baseUrl) {
            return null;
        }

        return this.fetchWithCache(
            `${CONFIG.analytics.validatorPerformance}?period=${period}`,
            `validator-performance-${period}`
        );
    }

    /**
     * Get comprehensive dashboard data
     * Fetches all analytics in parallel
     */
    async getDashboardData() {
        if (!this.baseUrl) {
            monitoringService.addLog('warning', 'Analytics not available for current network');
            return null;
        }

        try {
            const [
                networkHealth,
                transactionVolume,
                dexAnalytics,
                addressGrowth,
                gasAnalytics,
                validatorPerformance
            ] = await Promise.allSettled([
                this.getNetworkHealth(),
                this.getTransactionVolume('24h'),
                this.getDEXAnalytics(),
                this.getAddressGrowth('30d'),
                this.getGasAnalytics('24h'),
                this.getValidatorPerformance('24h')
            ]);

            return {
                networkHealth: networkHealth.status === 'fulfilled' ? networkHealth.value : null,
                transactionVolume: transactionVolume.status === 'fulfilled' ? transactionVolume.value : null,
                dexAnalytics: dexAnalytics.status === 'fulfilled' ? dexAnalytics.value : null,
                addressGrowth: addressGrowth.status === 'fulfilled' ? addressGrowth.value : null,
                gasAnalytics: gasAnalytics.status === 'fulfilled' ? gasAnalytics.value : null,
                validatorPerformance: validatorPerformance.status === 'fulfilled' ? validatorPerformance.value : null
            };
        } catch (error) {
            monitoringService.addLog('error', `Failed to fetch dashboard data: ${error.message}`);
            return null;
        }
    }

    /**
     * Clear analytics cache
     */
    clearCache() {
        this.cache.clear();
        monitoringService.addLog('info', 'Analytics cache cleared');
    }

    /**
     * Check if analytics is available for current network
     */
    isAvailable() {
        return this.baseUrl !== null && this.baseUrl !== undefined;
    }
}

// Export singleton instance
const analyticsService = new AnalyticsService();
export default analyticsService;
