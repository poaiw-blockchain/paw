// UptimeMonitor Component - Displays uptime visualization and monitoring

class UptimeMonitor {
    constructor(uptimeData) {
        this.data = uptimeData || {};
        this.blocks = this.data.blocks || [];
        this.uptimePercentage = this.data.uptimePercentage || 0;
        this.missedBlocks = this.data.missedBlocks || 0;
        this.totalBlocks = this.data.totalBlocks || 0;
    }

    render() {
        return `
            <div class="uptime-monitor-container">
                <div class="uptime-header">
                    <div class="uptime-overview">
                        <div class="uptime-percentage">
                            <span class="percentage-value ${this.getUptimeClass()}">${this.uptimePercentage.toFixed(2)}%</span>
                            <span class="percentage-label">Uptime</span>
                        </div>
                        <div class="uptime-stats">
                            <div class="stat-item">
                                <span class="stat-value">${this.totalBlocks.toLocaleString()}</span>
                                <span class="stat-label">Total Blocks</span>
                            </div>
                            <div class="stat-item">
                                <span class="stat-value success">${(this.totalBlocks - this.missedBlocks).toLocaleString()}</span>
                                <span class="stat-label">Signed</span>
                            </div>
                            <div class="stat-item">
                                <span class="stat-value danger">${this.missedBlocks.toLocaleString()}</span>
                                <span class="stat-label">Missed</span>
                            </div>
                        </div>
                    </div>

                    <div class="uptime-indicators">
                        ${this.renderUptimeIndicators()}
                    </div>
                </div>

                <div class="uptime-visualization">
                    <h4>Block Signing History (Last ${this.blocks.length} blocks)</h4>
                    <div class="block-grid">
                        ${this.renderBlockGrid()}
                    </div>
                    <div class="grid-legend">
                        <div class="legend-item">
                            <span class="block-sample signed"></span>
                            <span>Signed</span>
                        </div>
                        <div class="legend-item">
                            <span class="block-sample missed"></span>
                            <span>Missed</span>
                        </div>
                        <div class="legend-item">
                            <span class="block-sample proposed"></span>
                            <span>Proposed</span>
                        </div>
                    </div>
                </div>

                <div class="uptime-timeline">
                    <h4>24-Hour Uptime Timeline</h4>
                    ${this.renderUptimeTimeline()}
                </div>

                <div class="uptime-metrics">
                    <h4>Uptime Metrics</h4>
                    <div class="metrics-grid">
                        ${this.renderUptimeMetrics()}
                    </div>
                </div>
            </div>

            <style>
                .uptime-monitor-container {
                    display: flex;
                    flex-direction: column;
                    gap: 2rem;
                }

                .uptime-header {
                    display: grid;
                    grid-template-columns: 1fr 1fr;
                    gap: 2rem;
                }

                .uptime-overview {
                    background-color: var(--card-bg);
                    padding: 1.5rem;
                    border-radius: 0.75rem;
                    border: 1px solid var(--border-color);
                }

                .uptime-percentage {
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    margin-bottom: 1.5rem;
                }

                .percentage-value {
                    font-size: 3rem;
                    font-weight: 700;
                    line-height: 1;
                }

                .percentage-value.excellent {
                    color: var(--success-color);
                }

                .percentage-value.good {
                    color: #10b981;
                }

                .percentage-value.warning {
                    color: var(--warning-color);
                }

                .percentage-value.critical {
                    color: var(--danger-color);
                }

                .percentage-label {
                    font-size: 0.875rem;
                    color: var(--text-secondary);
                    text-transform: uppercase;
                    margin-top: 0.5rem;
                }

                .uptime-stats {
                    display: grid;
                    grid-template-columns: repeat(3, 1fr);
                    gap: 1rem;
                }

                .stat-item {
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    padding: 1rem;
                    background-color: var(--light-bg);
                    border-radius: 0.5rem;
                }

                .stat-value {
                    font-size: 1.5rem;
                    font-weight: 600;
                    color: var(--text-primary);
                }

                .stat-value.success {
                    color: var(--success-color);
                }

                .stat-value.danger {
                    color: var(--danger-color);
                }

                .stat-label {
                    font-size: 0.75rem;
                    color: var(--text-secondary);
                    text-transform: uppercase;
                    margin-top: 0.25rem;
                }

                .uptime-indicators {
                    background-color: var(--card-bg);
                    padding: 1.5rem;
                    border-radius: 0.75rem;
                    border: 1px solid var(--border-color);
                }

                .indicator {
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    padding: 0.75rem 0;
                    border-bottom: 1px solid var(--border-color);
                }

                .indicator:last-child {
                    border-bottom: none;
                }

                .indicator-label {
                    display: flex;
                    align-items: center;
                    gap: 0.5rem;
                    font-size: 0.875rem;
                    color: var(--text-secondary);
                }

                .indicator-value {
                    font-weight: 600;
                    color: var(--text-primary);
                }

                .status-icon {
                    width: 8px;
                    height: 8px;
                    border-radius: 50%;
                }

                .status-icon.active {
                    background-color: var(--success-color);
                    box-shadow: 0 0 8px var(--success-color);
                }

                .status-icon.inactive {
                    background-color: var(--danger-color);
                }

                .uptime-visualization,
                .uptime-timeline,
                .uptime-metrics {
                    background-color: var(--card-bg);
                    padding: 1.5rem;
                    border-radius: 0.75rem;
                    border: 1px solid var(--border-color);
                }

                .uptime-visualization h4,
                .uptime-timeline h4,
                .uptime-metrics h4 {
                    margin-bottom: 1rem;
                    font-size: 1.125rem;
                    color: var(--text-primary);
                }

                .block-grid {
                    display: grid;
                    grid-template-columns: repeat(auto-fill, minmax(12px, 1fr));
                    gap: 4px;
                    margin-bottom: 1rem;
                }

                .block-cell {
                    width: 12px;
                    height: 12px;
                    border-radius: 2px;
                    cursor: pointer;
                    transition: transform 0.2s;
                }

                .block-cell:hover {
                    transform: scale(1.5);
                }

                .block-cell.signed {
                    background-color: var(--success-color);
                }

                .block-cell.missed {
                    background-color: var(--danger-color);
                }

                .block-cell.proposed {
                    background-color: var(--primary-color);
                }

                .grid-legend {
                    display: flex;
                    justify-content: center;
                    gap: 2rem;
                    margin-top: 1rem;
                }

                .legend-item {
                    display: flex;
                    align-items: center;
                    gap: 0.5rem;
                    font-size: 0.875rem;
                    color: var(--text-secondary);
                }

                .block-sample {
                    width: 12px;
                    height: 12px;
                    border-radius: 2px;
                }

                .block-sample.signed {
                    background-color: var(--success-color);
                }

                .block-sample.missed {
                    background-color: var(--danger-color);
                }

                .block-sample.proposed {
                    background-color: var(--primary-color);
                }

                .timeline-chart {
                    height: 100px;
                    position: relative;
                    margin: 1rem 0;
                }

                .timeline-bar {
                    position: absolute;
                    bottom: 0;
                    background-color: var(--success-color);
                    border-radius: 2px;
                    transition: all 0.3s;
                }

                .timeline-bar.low {
                    background-color: var(--danger-color);
                }

                .timeline-labels {
                    display: flex;
                    justify-content: space-between;
                    font-size: 0.75rem;
                    color: var(--text-secondary);
                }

                .metrics-grid {
                    display: grid;
                    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
                    gap: 1rem;
                }

                .metric-card {
                    padding: 1rem;
                    background-color: var(--light-bg);
                    border-radius: 0.5rem;
                    border-left: 4px solid var(--primary-color);
                }

                .metric-card.warning {
                    border-left-color: var(--warning-color);
                }

                .metric-card.critical {
                    border-left-color: var(--danger-color);
                }

                .metric-title {
                    font-size: 0.875rem;
                    color: var(--text-secondary);
                    margin-bottom: 0.5rem;
                }

                .metric-value {
                    font-size: 1.25rem;
                    font-weight: 600;
                    color: var(--text-primary);
                }

                @media (max-width: 768px) {
                    .uptime-header {
                        grid-template-columns: 1fr;
                    }

                    .block-grid {
                        grid-template-columns: repeat(auto-fill, minmax(10px, 1fr));
                    }

                    .block-cell {
                        width: 10px;
                        height: 10px;
                    }
                }
            </style>
        `;
    }

    renderUptimeIndicators() {
        const indicators = [
            {
                label: 'Current Status',
                value: this.uptimePercentage >= 95 ? 'Active' : 'At Risk',
                active: this.uptimePercentage >= 95
            },
            {
                label: 'Last 100 Blocks',
                value: `${this.calculateRecentUptime(100)}%`,
                active: this.calculateRecentUptime(100) >= 95
            },
            {
                label: 'Last 1000 Blocks',
                value: `${this.calculateRecentUptime(1000)}%`,
                active: this.calculateRecentUptime(1000) >= 95
            },
            {
                label: 'Signing Window',
                value: this.getSigningWindowStatus(),
                active: this.missedBlocks < 50
            }
        ];

        return indicators.map(indicator => `
            <div class="indicator">
                <span class="indicator-label">
                    <span class="status-icon ${indicator.active ? 'active' : 'inactive'}"></span>
                    ${indicator.label}
                </span>
                <span class="indicator-value">${indicator.value}</span>
            </div>
        `).join('');
    }

    renderBlockGrid() {
        if (!this.blocks || this.blocks.length === 0) {
            return '<div class="empty-state"><p>No block data available</p></div>';
        }

        return this.blocks.map((block, index) => {
            const className = block.proposed ? 'proposed' : (block.signed ? 'signed' : 'missed');
            const title = `Block ${block.height}: ${block.proposed ? 'Proposed' : (block.signed ? 'Signed' : 'Missed')}`;

            return `<div class="block-cell ${className}" title="${title}" data-height="${block.height}"></div>`;
        }).join('');
    }

    renderUptimeTimeline() {
        const hourlyData = this.aggregateHourlyData();

        const maxHeight = 80; // pixels
        const barWidth = 100 / hourlyData.length;

        const bars = hourlyData.map((hour, index) => {
            const height = (hour.uptime / 100) * maxHeight;
            const left = index * barWidth;
            const className = hour.uptime < 95 ? 'low' : '';

            return `
                <div class="timeline-bar ${className}"
                     style="left: ${left}%; width: ${barWidth}%; height: ${height}px"
                     title="Hour ${hour.hour}: ${hour.uptime.toFixed(2)}% uptime">
                </div>
            `;
        }).join('');

        const labels = [0, 6, 12, 18, 24].map(hour =>
            `<span>${hour}h</span>`
        ).join('');

        return `
            <div class="timeline-chart">
                ${bars}
            </div>
            <div class="timeline-labels">
                ${labels}
            </div>
        `;
    }

    renderUptimeMetrics() {
        const metrics = [
            {
                title: '7-Day Uptime',
                value: `${this.data.uptime7d || this.uptimePercentage}%`,
                critical: (this.data.uptime7d || this.uptimePercentage) < 90
            },
            {
                title: '30-Day Uptime',
                value: `${this.data.uptime30d || this.uptimePercentage}%`,
                critical: (this.data.uptime30d || this.uptimePercentage) < 95
            },
            {
                title: 'Consecutive Misses',
                value: this.data.consecutiveMisses || 0,
                critical: (this.data.consecutiveMisses || 0) > 5
            },
            {
                title: 'Longest Uptime Streak',
                value: `${this.data.longestStreak || 0} blocks`,
                critical: false
            },
            {
                title: 'Time to Slash',
                value: this.calculateTimeToSlash(),
                warning: this.missedBlocks > 30
            },
            {
                title: 'Recovery Needed',
                value: this.calculateRecoveryNeeded(),
                warning: this.uptimePercentage < 95
            }
        ];

        return metrics.map(metric => {
            const className = metric.critical ? 'critical' : (metric.warning ? 'warning' : '');
            return `
                <div class="metric-card ${className}">
                    <div class="metric-title">${metric.title}</div>
                    <div class="metric-value">${metric.value}</div>
                </div>
            `;
        }).join('');
    }

    getUptimeClass() {
        if (this.uptimePercentage >= 99) return 'excellent';
        if (this.uptimePercentage >= 95) return 'good';
        if (this.uptimePercentage >= 90) return 'warning';
        return 'critical';
    }

    calculateRecentUptime(blockCount) {
        if (!this.blocks || this.blocks.length === 0) return 0;

        const recentBlocks = this.blocks.slice(-blockCount);
        const signed = recentBlocks.filter(b => b.signed || b.proposed).length;

        return ((signed / recentBlocks.length) * 100).toFixed(2);
    }

    getSigningWindowStatus() {
        // Assuming signing window is 10000 blocks and slash threshold is 500 misses
        const windowSize = 10000;
        const slashThreshold = 500;
        const remaining = slashThreshold - this.missedBlocks;

        if (remaining > 100) return 'Safe';
        if (remaining > 50) return 'Caution';
        return 'Critical';
    }

    aggregateHourlyData() {
        // Generate 24 hours of data
        const hourlyData = [];
        for (let i = 0; i < 24; i++) {
            hourlyData.push({
                hour: i,
                uptime: 95 + Math.random() * 5, // Simulated data
                signed: 0,
                missed: 0
            });
        }

        return hourlyData;
    }

    calculateTimeToSlash() {
        const slashThreshold = 500;
        const remaining = slashThreshold - this.missedBlocks;

        if (remaining <= 0) return 'Slashed';
        if (remaining < 50) return `${remaining} blocks`;
        return 'Safe';
    }

    calculateRecoveryNeeded() {
        if (this.uptimePercentage >= 95) return 'None';

        const target = 95;
        const needed = Math.ceil((target - this.uptimePercentage) * 10);

        return `~${needed} blocks`;
    }
}

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = UptimeMonitor;
}
