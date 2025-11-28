// ===========================
// Configuration
// ===========================
const CONFIG = {
    API_BASE_URL: window.location.hostname === 'localhost'
        ? 'http://localhost:8080/api/v1'
        : '/api/v1',
    REFRESH_INTERVAL: 30000, // 30 seconds
    MOCK_DATA: true, // Enable mock data for development
};

// ===========================
// State Management
// ===========================
const state = {
    status: null,
    components: [],
    incidents: [],
    metrics: {
        tps: [],
        blockTime: [],
        peers: [],
        responseTime: []
    },
    charts: {},
    refreshInterval: null
};

// ===========================
// API Functions
// ===========================
const API = {
    async fetchStatus() {
        if (CONFIG.MOCK_DATA) {
            return this.getMockStatus();
        }
        try {
            const response = await fetch(`${CONFIG.API_BASE_URL}/status`);
            return await response.json();
        } catch (error) {
            console.error('Failed to fetch status:', error);
            return this.getMockStatus();
        }
    },

    async fetchIncidents() {
        if (CONFIG.MOCK_DATA) {
            return this.getMockIncidents();
        }
        try {
            const response = await fetch(`${CONFIG.API_BASE_URL}/incidents`);
            return await response.json();
        } catch (error) {
            console.error('Failed to fetch incidents:', error);
            return this.getMockIncidents();
        }
    },

    async fetchMetrics() {
        if (CONFIG.MOCK_DATA) {
            return this.getMockMetrics();
        }
        try {
            const response = await fetch(`${CONFIG.API_BASE_URL}/metrics`);
            return await response.json();
        } catch (error) {
            console.error('Failed to fetch metrics:', error);
            return this.getMockMetrics();
        }
    },

    async subscribe(email, preferences) {
        try {
            const response = await fetch(`${CONFIG.API_BASE_URL}/subscribe`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email, preferences })
            });
            return await response.json();
        } catch (error) {
            console.error('Failed to subscribe:', error);
            throw error;
        }
    },

    // Mock Data Generators
    getMockStatus() {
        return {
            overall_status: 'operational',
            message: 'All systems operational',
            components: [
                {
                    name: 'Blockchain',
                    status: 'operational',
                    description: 'Core blockchain network',
                    uptime: '99.99%',
                    response_time: '45ms'
                },
                {
                    name: 'API',
                    status: 'operational',
                    description: 'REST and GraphQL API endpoints',
                    uptime: '99.95%',
                    response_time: '120ms'
                },
                {
                    name: 'WebSocket',
                    status: 'operational',
                    description: 'Real-time data streaming',
                    uptime: '99.98%',
                    response_time: '35ms'
                },
                {
                    name: 'Explorer',
                    status: 'operational',
                    description: 'Block explorer interface',
                    uptime: '99.92%',
                    response_time: '180ms'
                },
                {
                    name: 'Faucet',
                    status: 'operational',
                    description: 'Testnet token distribution',
                    uptime: '99.85%',
                    response_time: '250ms'
                }
            ],
            updated_at: new Date().toISOString()
        };
    },

    getMockIncidents() {
        return {
            active: [],
            history: [
                {
                    id: 1,
                    title: 'Scheduled Maintenance - Database Upgrade',
                    severity: 'minor',
                    status: 'resolved',
                    started_at: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
                    resolved_at: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000 + 2 * 60 * 60 * 1000).toISOString(),
                    description: 'Planned database upgrade to improve performance.',
                    updates: [
                        {
                            timestamp: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
                            message: 'Maintenance window started. Expected duration: 2 hours.'
                        },
                        {
                            timestamp: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000 + 1 * 60 * 60 * 1000).toISOString(),
                            message: 'Database upgrade in progress. All services operational.'
                        },
                        {
                            timestamp: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000 + 2 * 60 * 60 * 1000).toISOString(),
                            message: 'Maintenance completed successfully. All systems normal.'
                        }
                    ]
                },
                {
                    id: 2,
                    title: 'API Rate Limiting Issues',
                    severity: 'major',
                    status: 'resolved',
                    started_at: new Date(Date.now() - 14 * 24 * 60 * 60 * 1000).toISOString(),
                    resolved_at: new Date(Date.now() - 14 * 24 * 60 * 60 * 1000 + 45 * 60 * 1000).toISOString(),
                    description: 'Some users experienced rate limiting errors on API endpoints.',
                    updates: [
                        {
                            timestamp: new Date(Date.now() - 14 * 24 * 60 * 60 * 1000).toISOString(),
                            message: 'Investigating reports of API rate limiting issues.'
                        },
                        {
                            timestamp: new Date(Date.now() - 14 * 24 * 60 * 60 * 1000 + 30 * 60 * 1000).toISOString(),
                            message: 'Issue identified. Deploying fix.'
                        },
                        {
                            timestamp: new Date(Date.now() - 14 * 24 * 60 * 60 * 1000 + 45 * 60 * 1000).toISOString(),
                            message: 'Fix deployed. Monitoring for stability.'
                        }
                    ]
                }
            ]
        };
    },

    getMockMetrics() {
        const now = Date.now();
        const generateData = (baseValue, variance) => {
            return Array.from({ length: 20 }, (_, i) => ({
                timestamp: new Date(now - (19 - i) * 60000),
                value: baseValue + (Math.random() - 0.5) * variance
            }));
        };

        return {
            tps: generateData(150, 50),
            block_time: generateData(6.5, 1),
            peers: generateData(42, 5),
            response_time: generateData(120, 30),
            network_stats: {
                block_height: 1234567,
                total_validators: 150,
                active_validators: 125,
                hash_rate: '1.2 TH/s'
            },
            uptime_data: Array.from({ length: 30 }, (_, i) => ({
                date: new Date(now - (29 - i) * 24 * 60 * 60 * 1000),
                status: Math.random() > 0.02 ? 'operational' : 'degraded'
            }))
        };
    }
};

// ===========================
// UI Rendering Functions
// ===========================
const UI = {
    updateStatusBanner(status) {
        const banner = document.getElementById('statusBanner');
        const statusText = document.querySelector('.status-text');
        const statusMessage = document.getElementById('statusMessage');

        banner.className = `status-banner ${status.overall_status}`;

        const statusLabels = {
            operational: 'All Systems Operational',
            degraded: 'Degraded Performance',
            down: 'Service Disruption'
        };

        statusText.textContent = statusLabels[status.overall_status] || 'Unknown Status';
        statusMessage.textContent = status.message;
    },

    renderComponents(components) {
        const grid = document.getElementById('componentsGrid');
        grid.innerHTML = components.map(component => `
            <div class="component-card">
                <div class="component-header">
                    <h3 class="component-name">${this.escapeHtml(component.name)}</h3>
                    <span class="component-status ${component.status}">
                        <span class="status-icon">●</span>
                        ${this.capitalize(component.status)}
                    </span>
                </div>
                <p class="component-description">${this.escapeHtml(component.description)}</p>
                <div class="component-metrics">
                    <div class="component-metric">
                        <span class="metric-label">Uptime</span>
                        <span class="metric-value">${component.uptime}</span>
                    </div>
                    <div class="component-metric">
                        <span class="metric-label">Response Time</span>
                        <span class="metric-value">${component.response_time}</span>
                    </div>
                </div>
            </div>
        `).join('');
    },

    renderIncidents(incidents) {
        const activeContainer = document.getElementById('activeIncidents');

        if (incidents.active && incidents.active.length > 0) {
            activeContainer.innerHTML = incidents.active.map(incident => `
                <div class="incident-card ${incident.severity}">
                    <div class="incident-header">
                        <div>
                            <h3 class="incident-title">${this.escapeHtml(incident.title)}</h3>
                            <span class="incident-severity ${incident.severity}">
                                ${incident.severity}
                            </span>
                        </div>
                    </div>
                    <div class="incident-time">
                        Started: ${this.formatDate(incident.started_at)}
                    </div>
                    <p class="incident-description">${this.escapeHtml(incident.description)}</p>
                    ${incident.updates && incident.updates.length > 0 ? `
                        <div class="incident-updates">
                            ${incident.updates.map(update => `
                                <div class="incident-update">
                                    <div class="update-time">${this.formatDate(update.timestamp)}</div>
                                    <div class="update-message">${this.escapeHtml(update.message)}</div>
                                </div>
                            `).join('')}
                        </div>
                    ` : ''}
                </div>
            `).join('');
        } else {
            activeContainer.innerHTML = `
                <div class="no-incidents">
                    <span class="icon">✓</span>
                    <p>No active incidents. All systems operational.</p>
                </div>
            `;
        }

        // Render incident history
        const timeline = document.getElementById('incidentTimeline');
        if (incidents.history && incidents.history.length > 0) {
            timeline.innerHTML = incidents.history.map(incident => `
                <div class="timeline-item">
                    <div class="timeline-date">
                        ${this.formatDate(incident.started_at)}
                    </div>
                    <div class="timeline-content incident-card ${incident.severity}">
                        <div class="incident-header">
                            <div>
                                <h3 class="incident-title">${this.escapeHtml(incident.title)}</h3>
                                <span class="incident-severity ${incident.severity}">
                                    ${incident.severity}
                                </span>
                            </div>
                        </div>
                        <p class="incident-description">${this.escapeHtml(incident.description)}</p>
                        <div class="incident-time">
                            Resolved: ${this.formatDate(incident.resolved_at)}
                            (Duration: ${this.calculateDuration(incident.started_at, incident.resolved_at)})
                        </div>
                    </div>
                </div>
            `).join('');
        } else {
            timeline.innerHTML = '<p style="color: var(--text-muted);">No recent incidents to display.</p>';
        }
    },

    renderMetrics(metrics) {
        // Update metric values
        const tpsValue = metrics.tps[metrics.tps.length - 1]?.value || 0;
        const blockTimeValue = metrics.block_time[metrics.block_time.length - 1]?.value || 0;
        const peersValue = metrics.peers[metrics.peers.length - 1]?.value || 0;
        const responseTimeValue = metrics.response_time[metrics.response_time.length - 1]?.value || 0;

        document.getElementById('tpsValue').textContent = Math.round(tpsValue) + ' TPS';
        document.getElementById('blockTimeValue').textContent = blockTimeValue.toFixed(2) + 's';
        document.getElementById('peersValue').textContent = Math.round(peersValue);
        document.getElementById('responseTimeValue').textContent = Math.round(responseTimeValue) + 'ms';

        // Calculate changes
        this.updateMetricChange('tps', metrics.tps);
        this.updateMetricChange('blockTime', metrics.block_time);
        this.updateMetricChange('peers', metrics.peers);
        this.updateMetricChange('responseTime', metrics.response_time);

        // Update charts
        this.updateChart('tpsChart', metrics.tps, 'TPS', '#6366f1');
        this.updateChart('blockTimeChart', metrics.block_time, 'Block Time (s)', '#10b981');
        this.updateChart('peersChart', metrics.peers, 'Peers', '#f59e0b');
        this.updateChart('responseTimeChart', metrics.response_time, 'Response Time (ms)', '#ef4444');

        // Update network stats
        if (metrics.network_stats) {
            document.getElementById('blockHeight').textContent = metrics.network_stats.block_height.toLocaleString();
            document.getElementById('totalValidators').textContent = metrics.network_stats.total_validators;
            document.getElementById('activeValidators').textContent = metrics.network_stats.active_validators;
            document.getElementById('hashRate').textContent = metrics.network_stats.hash_rate;
        }

        // Update uptime calendar
        if (metrics.uptime_data) {
            this.renderUptimeCalendar(metrics.uptime_data);
        }
    },

    updateMetricChange(metricId, data) {
        if (data.length < 2) return;

        const current = data[data.length - 1].value;
        const previous = data[data.length - 2].value;
        const change = ((current - previous) / previous * 100).toFixed(1);
        const element = document.getElementById(`${metricId}Change`);

        if (Math.abs(change) < 0.1) {
            element.textContent = 'No change';
            element.className = 'metric-change neutral';
        } else if (change > 0) {
            element.textContent = `↑ ${change}% from last update`;
            element.className = 'metric-change positive';
        } else {
            element.textContent = `↓ ${Math.abs(change)}% from last update`;
            element.className = 'metric-change negative';
        }
    },

    updateChart(canvasId, data, label, color) {
        const canvas = document.getElementById(canvasId);
        const ctx = canvas.getContext('2d');

        // Destroy existing chart if it exists
        if (state.charts[canvasId]) {
            state.charts[canvasId].destroy();
        }

        state.charts[canvasId] = new Chart(ctx, {
            type: 'line',
            data: {
                labels: data.map(d => ''),
                datasets: [{
                    label: label,
                    data: data.map(d => d.value),
                    borderColor: color,
                    backgroundColor: color + '20',
                    borderWidth: 2,
                    fill: true,
                    tension: 0.4,
                    pointRadius: 0
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: { display: false },
                    tooltip: { enabled: false }
                },
                scales: {
                    x: { display: false },
                    y: { display: false }
                }
            }
        });
    },

    renderUptimeCalendar(uptimeData) {
        const calendar = document.getElementById('uptimeCalendar');
        const uptimePercentage = document.getElementById('uptimePercentage');

        const operational = uptimeData.filter(d => d.status === 'operational').length;
        const uptime = (operational / uptimeData.length * 100).toFixed(2);

        uptimePercentage.textContent = uptime + '%';

        calendar.innerHTML = uptimeData.map(day => `
            <div class="uptime-day ${day.status}"
                 title="${this.formatDate(day.date)}: ${this.capitalize(day.status)}">
            </div>
        `).join('');
    },

    renderDependenciesGraph() {
        const svg = document.getElementById('dependenciesSvg');
        const width = svg.clientWidth;
        const height = 400;

        const dependencies = [
            { from: 'Users', to: 'API', x1: 100, y1: 100, x2: 300, y2: 100 },
            { from: 'Users', to: 'WebSocket', x1: 100, y1: 100, x2: 300, y2: 200 },
            { from: 'API', to: 'Blockchain', x1: 300, y1: 100, x2: 500, y2: 150 },
            { from: 'WebSocket', to: 'Blockchain', x1: 300, y1: 200, x2: 500, y2: 150 },
            { from: 'Explorer', to: 'API', x1: 300, y1: 300, x2: 300, y2: 100 },
            { from: 'Faucet', to: 'Blockchain', x1: 500, y1: 300, x2: 500, y2: 150 }
        ];

        const nodes = [
            { name: 'Users', x: 100, y: 100 },
            { name: 'API', x: 300, y: 100 },
            { name: 'WebSocket', x: 300, y: 200 },
            { name: 'Blockchain', x: 500, y: 150 },
            { name: 'Explorer', x: 300, y: 300 },
            { name: 'Faucet', x: 500, y: 300 }
        ];

        let svgContent = '';

        // Draw edges
        dependencies.forEach(dep => {
            svgContent += `<line x1="${dep.x1}" y1="${dep.y1}" x2="${dep.x2}" y2="${dep.y2}"
                stroke="#334155" stroke-width="2" marker-end="url(#arrowhead)" />`;
        });

        // Draw nodes
        nodes.forEach(node => {
            svgContent += `
                <circle cx="${node.x}" cy="${node.y}" r="30" fill="#1e293b" stroke="#6366f1" stroke-width="2" />
                <text x="${node.x}" y="${node.y + 5}" text-anchor="middle" fill="#f1f5f9" font-size="12">${node.name}</text>
            `;
        });

        svg.innerHTML = `
            <defs>
                <marker id="arrowhead" markerWidth="10" markerHeight="10" refX="9" refY="3" orient="auto">
                    <polygon points="0 0, 10 3, 0 6" fill="#334155" />
                </marker>
            </defs>
            ${svgContent}
        `;
    },

    // Utility functions
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    },

    capitalize(str) {
        return str.charAt(0).toUpperCase() + str.slice(1);
    },

    formatDate(dateString) {
        const date = new Date(dateString);
        const now = new Date();
        const diff = now - date;
        const minutes = Math.floor(diff / 60000);
        const hours = Math.floor(diff / 3600000);
        const days = Math.floor(diff / 86400000);

        if (minutes < 60) {
            return `${minutes} minute${minutes !== 1 ? 's' : ''} ago`;
        } else if (hours < 24) {
            return `${hours} hour${hours !== 1 ? 's' : ''} ago`;
        } else if (days < 30) {
            return `${days} day${days !== 1 ? 's' : ''} ago`;
        } else {
            return date.toLocaleDateString('en-US', {
                year: 'numeric',
                month: 'short',
                day: 'numeric',
                hour: '2-digit',
                minute: '2-digit'
            });
        }
    },

    calculateDuration(start, end) {
        const diff = new Date(end) - new Date(start);
        const hours = Math.floor(diff / 3600000);
        const minutes = Math.floor((diff % 3600000) / 60000);
        return `${hours}h ${minutes}m`;
    },

    showNotification(message, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.textContent = message;
        document.body.appendChild(notification);

        setTimeout(() => {
            notification.remove();
        }, 5000);
    }
};

// ===========================
// Event Handlers
// ===========================
const EventHandlers = {
    setupEventListeners() {
        // Subscribe button
        document.getElementById('subscribeBtn').addEventListener('click', () => {
            document.getElementById('subscribeModal').classList.add('active');
        });

        // Close modal
        document.getElementById('closeModal').addEventListener('click', () => {
            document.getElementById('subscribeModal').classList.remove('active');
        });

        document.getElementById('cancelSubscribe').addEventListener('click', () => {
            document.getElementById('subscribeModal').classList.remove('active');
        });

        // Subscribe form
        document.getElementById('subscribeForm').addEventListener('submit', async (e) => {
            e.preventDefault();

            const email = document.getElementById('subscriberEmail').value;
            const preferences = {
                incidents: document.getElementById('subscribeIncidents').checked,
                maintenance: document.getElementById('subscribeMaintenance').checked
            };

            try {
                await API.subscribe(email, preferences);
                UI.showNotification('Successfully subscribed to status updates!', 'success');
                document.getElementById('subscribeModal').classList.remove('active');
                document.getElementById('subscribeForm').reset();
            } catch (error) {
                UI.showNotification('Failed to subscribe. Please try again.', 'error');
            }
        });

        // Close modal on outside click
        document.getElementById('subscribeModal').addEventListener('click', (e) => {
            if (e.target.id === 'subscribeModal') {
                document.getElementById('subscribeModal').classList.remove('active');
            }
        });
    }
};

// ===========================
// Data Refresh
// ===========================
async function refreshData() {
    try {
        const [status, incidents, metrics] = await Promise.all([
            API.fetchStatus(),
            API.fetchIncidents(),
            API.fetchMetrics()
        ]);

        state.status = status;
        state.incidents = incidents;
        state.metrics = metrics;

        UI.updateStatusBanner(status);
        UI.renderComponents(status.components);
        UI.renderIncidents(incidents);
        UI.renderMetrics(metrics);

        document.getElementById('lastUpdated').textContent = new Date().toLocaleString();
    } catch (error) {
        console.error('Failed to refresh data:', error);
        UI.showNotification('Failed to refresh status data', 'error');
    }
}

// ===========================
// Initialization
// ===========================
async function init() {
    EventHandlers.setupEventListeners();
    UI.renderDependenciesGraph();

    await refreshData();

    // Set up auto-refresh
    state.refreshInterval = setInterval(refreshData, CONFIG.REFRESH_INTERVAL);
}

// Start the application when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}
