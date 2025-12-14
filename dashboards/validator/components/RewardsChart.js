// RewardsChart Component - Visualizes rewards data over time

class RewardsChart {
    constructor(rewardsData) {
        this.data = rewardsData || [];
        this.chartType = 'line'; // line, bar, area
        this.timeframe = '30d'; // 7d, 30d, 90d, 1y, all
    }

    render(container) {
        if (!container) return;

        const processedData = this.processData();

        container.innerHTML = `
            <div class="rewards-chart-wrapper">
                <div class="chart-controls">
                    <div class="chart-type-selector">
                        <button class="chart-type-btn ${this.chartType === 'line' ? 'active' : ''}" data-type="line">
                            <i class="fas fa-chart-line"></i> Line
                        </button>
                        <button class="chart-type-btn ${this.chartType === 'bar' ? 'active' : ''}" data-type="bar">
                            <i class="fas fa-chart-bar"></i> Bar
                        </button>
                        <button class="chart-type-btn ${this.chartType === 'area' ? 'active' : ''}" data-type="area">
                            <i class="fas fa-chart-area"></i> Area
                        </button>
                    </div>
                    <div class="timeframe-selector">
                        <button class="timeframe-btn ${this.timeframe === '7d' ? 'active' : ''}" data-timeframe="7d">7D</button>
                        <button class="timeframe-btn ${this.timeframe === '30d' ? 'active' : ''}" data-timeframe="30d">30D</button>
                        <button class="timeframe-btn ${this.timeframe === '90d' ? 'active' : ''}" data-timeframe="90d">90D</button>
                        <button class="timeframe-btn ${this.timeframe === '1y' ? 'active' : ''}" data-timeframe="1y">1Y</button>
                        <button class="timeframe-btn ${this.timeframe === 'all' ? 'active' : ''}" data-timeframe="all">All</button>
                    </div>
                </div>

                <div class="chart-stats">
                    <div class="stat">
                        <span class="label">Total Rewards</span>
                        <span class="value">${this.formatAmount(processedData.total)}</span>
                    </div>
                    <div class="stat">
                        <span class="label">Average Daily</span>
                        <span class="value">${this.formatAmount(processedData.averageDaily)}</span>
                    </div>
                    <div class="stat">
                        <span class="label">Highest Day</span>
                        <span class="value">${this.formatAmount(processedData.highest)}</span>
                    </div>
                    <div class="stat">
                        <span class="label">Trend</span>
                        <span class="value ${processedData.trend > 0 ? 'positive' : 'negative'}">
                            <i class="fas fa-arrow-${processedData.trend > 0 ? 'up' : 'down'}"></i>
                            ${Math.abs(processedData.trend).toFixed(2)}%
                        </span>
                    </div>
                </div>

                <div class="chart-canvas" id="rewardsChartCanvas">
                    ${this.renderChart(processedData.dataPoints)}
                </div>

                <div class="chart-legend">
                    <div class="legend-item">
                        <span class="legend-color" style="background-color: #3b82f6;"></span>
                        <span class="legend-label">Rewards Earned</span>
                    </div>
                    <div class="legend-item">
                        <span class="legend-color" style="background-color: #10b981;"></span>
                        <span class="legend-label">Commission</span>
                    </div>
                </div>
            </div>

            <style>
                .rewards-chart-wrapper {
                    width: 100%;
                }

                .chart-controls {
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    margin-bottom: 1.5rem;
                    flex-wrap: wrap;
                    gap: 1rem;
                }

                .chart-type-selector,
                .timeframe-selector {
                    display: flex;
                    gap: 0.5rem;
                }

                .chart-type-btn,
                .timeframe-btn {
                    padding: 0.5rem 1rem;
                    border: 1px solid var(--border-color);
                    border-radius: 0.375rem;
                    background-color: white;
                    color: var(--text-secondary);
                    cursor: pointer;
                    transition: all 0.2s;
                    font-size: 0.875rem;
                    display: flex;
                    align-items: center;
                    gap: 0.5rem;
                }

                .chart-type-btn:hover,
                .timeframe-btn:hover {
                    background-color: var(--light-bg);
                    border-color: var(--primary-color);
                }

                .chart-type-btn.active,
                .timeframe-btn.active {
                    background-color: var(--primary-color);
                    color: white;
                    border-color: var(--primary-color);
                }

                .chart-stats {
                    display: grid;
                    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
                    gap: 1rem;
                    margin-bottom: 1.5rem;
                    padding: 1rem;
                    background-color: var(--light-bg);
                    border-radius: 0.5rem;
                }

                .chart-stats .stat {
                    display: flex;
                    flex-direction: column;
                    gap: 0.25rem;
                }

                .chart-stats .label {
                    font-size: 0.75rem;
                    color: var(--text-secondary);
                    text-transform: uppercase;
                }

                .chart-stats .value {
                    font-size: 1.125rem;
                    font-weight: 600;
                    color: var(--text-primary);
                }

                .chart-stats .value.positive {
                    color: var(--success-color);
                }

                .chart-stats .value.negative {
                    color: var(--danger-color);
                }

                .chart-canvas {
                    width: 100%;
                    height: 400px;
                    margin-bottom: 1rem;
                    position: relative;
                }

                .chart-svg {
                    width: 100%;
                    height: 100%;
                }

                .chart-grid-line {
                    stroke: var(--border-color);
                    stroke-width: 1;
                    stroke-dasharray: 3, 3;
                }

                .chart-axis-label {
                    font-size: 12px;
                    fill: var(--text-secondary);
                }

                .chart-data-line {
                    fill: none;
                    stroke: #3b82f6;
                    stroke-width: 2;
                    stroke-linecap: round;
                    stroke-linejoin: round;
                }

                .chart-data-area {
                    fill: rgba(59, 130, 246, 0.1);
                }

                .chart-data-point {
                    fill: #3b82f6;
                    cursor: pointer;
                }

                .chart-data-point:hover {
                    fill: #2563eb;
                    r: 6;
                }

                .chart-tooltip {
                    position: absolute;
                    background-color: var(--dark-bg);
                    color: white;
                    padding: 0.75rem;
                    border-radius: 0.375rem;
                    font-size: 0.875rem;
                    pointer-events: none;
                    opacity: 0;
                    transition: opacity 0.2s;
                    z-index: 10;
                }

                .chart-tooltip.visible {
                    opacity: 1;
                }

                .chart-legend {
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

                .legend-color {
                    width: 16px;
                    height: 16px;
                    border-radius: 0.25rem;
                }

                @media (max-width: 768px) {
                    .chart-controls {
                        flex-direction: column;
                        align-items: stretch;
                    }

                    .chart-type-selector,
                    .timeframe-selector {
                        justify-content: center;
                    }

                    .chart-canvas {
                        height: 300px;
                    }
                }
            </style>
        `;

        this.attachEventListeners(container);
    }

    renderChart(dataPoints) {
        if (!dataPoints || dataPoints.length === 0) {
            return '<div class="empty-state"><p>No rewards data available</p></div>';
        }

        const width = 100;
        const height = 100;
        const padding = 10;

        const maxValue = Math.max(...dataPoints.map(d => d.value));
        const minValue = Math.min(...dataPoints.map(d => d.value));
        const range = maxValue - minValue || 1;

        const points = dataPoints.map((d, i) => {
            const x = padding + (i / (dataPoints.length - 1)) * (width - 2 * padding);
            const y = height - padding - ((d.value - minValue) / range) * (height - 2 * padding);
            return { x, y, data: d };
        });

        let chartContent = '';

        if (this.chartType === 'area' || this.chartType === 'line') {
            const pathData = points.map((p, i) =>
                `${i === 0 ? 'M' : 'L'} ${p.x},${p.y}`
            ).join(' ');

            if (this.chartType === 'area') {
                const areaPath = `${pathData} L ${points[points.length - 1].x},${height - padding} L ${padding},${height - padding} Z`;
                chartContent += `<path d="${areaPath}" class="chart-data-area" />`;
            }

            chartContent += `<path d="${pathData}" class="chart-data-line" />`;
        } else if (this.chartType === 'bar') {
            const barWidth = (width - 2 * padding) / dataPoints.length * 0.8;
            chartContent += points.map((p, i) => {
                const barHeight = height - padding - p.y;
                return `<rect x="${p.x - barWidth / 2}" y="${p.y}" width="${barWidth}" height="${barHeight}" fill="#3b82f6" opacity="0.8" />`;
            }).join('');
        }

        // Add data points
        chartContent += points.map(p =>
            `<circle cx="${p.x}" cy="${p.y}" r="4" class="chart-data-point" data-date="${p.data.date}" data-value="${p.data.value}" />`
        ).join('');

        // Add grid lines
        const gridLines = Array.from({ length: 5 }, (_, i) => {
            const y = padding + (i / 4) * (height - 2 * padding);
            return `<line x1="${padding}" y1="${y}" x2="${width - padding}" y2="${y}" class="chart-grid-line" />`;
        }).join('');

        return `
            <svg viewBox="0 0 ${width} ${height}" class="chart-svg">
                ${gridLines}
                ${chartContent}
            </svg>
            <div class="chart-tooltip" id="chartTooltip"></div>
        `;
    }

    processData() {
        const filteredData = this.filterDataByTimeframe();

        const total = filteredData.reduce((sum, d) => sum + (d.amount || 0), 0);
        const averageDaily = filteredData.length > 0 ? total / filteredData.length : 0;
        const highest = filteredData.length > 0 ? Math.max(...filteredData.map(d => d.amount || 0)) : 0;

        // Calculate trend (simple linear regression)
        const trend = this.calculateTrend(filteredData);

        const dataPoints = filteredData.map(d => ({
            date: d.timestamp,
            value: d.amount || 0,
            commission: d.commission || 0
        }));

        return {
            total,
            averageDaily,
            highest,
            trend,
            dataPoints
        };
    }

    filterDataByTimeframe() {
        const now = new Date();
        const msPerDay = 24 * 60 * 60 * 1000;

        let cutoffDate;
        switch (this.timeframe) {
            case '7d':
                cutoffDate = new Date(now.getTime() - 7 * msPerDay);
                break;
            case '30d':
                cutoffDate = new Date(now.getTime() - 30 * msPerDay);
                break;
            case '90d':
                cutoffDate = new Date(now.getTime() - 90 * msPerDay);
                break;
            case '1y':
                cutoffDate = new Date(now.getTime() - 365 * msPerDay);
                break;
            case 'all':
            default:
                return this.data;
        }

        return this.data.filter(d => new Date(d.timestamp) >= cutoffDate);
    }

    calculateTrend(data) {
        if (data.length < 2) return 0;

        const first = data.slice(0, Math.floor(data.length / 3));
        const last = data.slice(-Math.floor(data.length / 3));

        const firstAvg = first.reduce((sum, d) => sum + (d.amount || 0), 0) / first.length;
        const lastAvg = last.reduce((sum, d) => sum + (d.amount || 0), 0) / last.length;

        return ((lastAvg - firstAvg) / (firstAvg || 1)) * 100;
    }

    attachEventListeners(container) {
        // Chart type selector
        container.querySelectorAll('.chart-type-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                this.chartType = e.currentTarget.getAttribute('data-type');
                this.render(container);
            });
        });

        // Timeframe selector
        container.querySelectorAll('.timeframe-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                this.timeframe = e.currentTarget.getAttribute('data-timeframe');
                this.render(container);
            });
        });

        // Tooltip on data points
        container.querySelectorAll('.chart-data-point').forEach(point => {
            point.addEventListener('mouseenter', (e) => {
                const tooltip = container.querySelector('#chartTooltip');
                const date = e.target.getAttribute('data-date');
                const value = parseFloat(e.target.getAttribute('data-value'));

                tooltip.innerHTML = `
                    <div><strong>${new Date(date).toLocaleDateString()}</strong></div>
                    <div>Rewards: ${this.formatAmount(value)}</div>
                `;

                const rect = e.target.getBoundingClientRect();
                tooltip.style.left = `${rect.left}px`;
                tooltip.style.top = `${rect.top - 60}px`;
                tooltip.classList.add('visible');
            });

            point.addEventListener('mouseleave', (e) => {
                const tooltip = container.querySelector('#chartTooltip');
                tooltip.classList.remove('visible');
            });
        });
    }

    formatAmount(amount) {
        if (!amount) return '0 PAW';
        const value = parseFloat(amount);
        if (value >= 1000000) {
            return `${(value / 1000000).toFixed(2)}M PAW`;
        } else if (value >= 1000) {
            return `${(value / 1000).toFixed(2)}K PAW`;
        }
        return `${value.toFixed(2)} PAW`;
    }
}

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = RewardsChart;
}
