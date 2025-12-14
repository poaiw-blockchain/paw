/**
 * TallyChart Component
 * Renders voting tally visualization using Chart.js
 */

class TallyChart {
    constructor() {
        this.charts = {};
    }

    render(canvasId, tally) {
        const canvas = document.getElementById(canvasId);
        if (!canvas) {
            console.warn(`Canvas element ${canvasId} not found`);
            return;
        }

        // Destroy existing chart if it exists
        if (this.charts[canvasId]) {
            this.charts[canvasId].destroy();
        }

        const ctx = canvas.getContext('2d');

        const yes = parseInt(tally.yes || 0);
        const no = parseInt(tally.no || 0);
        const abstain = parseInt(tally.abstain || 0);
        const veto = parseInt(tally.no_with_veto || 0);
        const total = yes + no + abstain + veto;

        // If no votes yet, show empty state
        if (total === 0) {
            this.charts[canvasId] = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: ['No votes yet'],
                    datasets: [{
                        data: [1],
                        backgroundColor: ['#e0e0e0'],
                        borderWidth: 0
                    }]
                },
                options: this.getChartOptions(true)
            });
            return;
        }

        this.charts[canvasId] = new Chart(ctx, {
            type: 'doughnut',
            data: {
                labels: ['Yes', 'No', 'Abstain', 'No With Veto'],
                datasets: [{
                    data: [yes, no, abstain, veto],
                    backgroundColor: [
                        '#4CAF50', // Yes - Green
                        '#f44336', // No - Red
                        '#9E9E9E', // Abstain - Gray
                        '#FF5722'  // Veto - Deep Orange
                    ],
                    borderWidth: 2,
                    borderColor: '#ffffff'
                }]
            },
            options: this.getChartOptions(false)
        });
    }

    renderBarChart(canvasId, tally) {
        const canvas = document.getElementById(canvasId);
        if (!canvas) {
            console.warn(`Canvas element ${canvasId} not found`);
            return;
        }

        if (this.charts[canvasId]) {
            this.charts[canvasId].destroy();
        }

        const ctx = canvas.getContext('2d');

        const yes = parseInt(tally.yes || 0);
        const no = parseInt(tally.no || 0);
        const abstain = parseInt(tally.abstain || 0);
        const veto = parseInt(tally.no_with_veto || 0);

        this.charts[canvasId] = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: ['Yes', 'No', 'Abstain', 'No With Veto'],
                datasets: [{
                    label: 'Votes',
                    data: [yes, no, abstain, veto],
                    backgroundColor: [
                        '#4CAF50',
                        '#f44336',
                        '#9E9E9E',
                        '#FF5722'
                    ]
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        display: false
                    },
                    tooltip: {
                        callbacks: {
                            label: (context) => {
                                const value = context.parsed.y;
                                const formatted = this.formatVotingPower(value);
                                return `${formatted} PAW`;
                            }
                        }
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        ticks: {
                            callback: (value) => {
                                return this.formatVotingPower(value);
                            }
                        }
                    }
                }
            }
        });
    }

    renderPolarChart(canvasId, tally) {
        const canvas = document.getElementById(canvasId);
        if (!canvas) {
            console.warn(`Canvas element ${canvasId} not found`);
            return;
        }

        if (this.charts[canvasId]) {
            this.charts[canvasId].destroy();
        }

        const ctx = canvas.getContext('2d');

        const yes = parseInt(tally.yes || 0);
        const no = parseInt(tally.no || 0);
        const abstain = parseInt(tally.abstain || 0);
        const veto = parseInt(tally.no_with_veto || 0);

        this.charts[canvasId] = new Chart(ctx, {
            type: 'polarArea',
            data: {
                labels: ['Yes', 'No', 'Abstain', 'No With Veto'],
                datasets: [{
                    data: [yes, no, abstain, veto],
                    backgroundColor: [
                        'rgba(76, 175, 80, 0.7)',
                        'rgba(244, 67, 54, 0.7)',
                        'rgba(158, 158, 158, 0.7)',
                        'rgba(255, 87, 34, 0.7)'
                    ],
                    borderColor: [
                        '#4CAF50',
                        '#f44336',
                        '#9E9E9E',
                        '#FF5722'
                    ],
                    borderWidth: 2
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'right'
                    },
                    tooltip: {
                        callbacks: {
                            label: (context) => {
                                const value = context.parsed.r;
                                const formatted = this.formatVotingPower(value);
                                const label = context.label;
                                return `${label}: ${formatted} PAW`;
                            }
                        }
                    }
                }
            }
        });
    }

    getChartOptions(isEmpty) {
        return {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    display: !isEmpty,
                    position: 'bottom',
                    labels: {
                        padding: 15,
                        font: {
                            size: 12
                        },
                        generateLabels: (chart) => {
                            const data = chart.data;
                            if (data.labels.length && data.datasets.length) {
                                return data.labels.map((label, i) => {
                                    const value = data.datasets[0].data[i];
                                    const total = data.datasets[0].data.reduce((a, b) => a + b, 0);
                                    const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : 0;
                                    const formatted = this.formatVotingPower(value);

                                    return {
                                        text: `${label}: ${formatted} PAW (${percentage}%)`,
                                        fillStyle: data.datasets[0].backgroundColor[i],
                                        hidden: false,
                                        index: i
                                    };
                                });
                            }
                            return [];
                        }
                    }
                },
                tooltip: {
                    enabled: !isEmpty,
                    callbacks: {
                        label: (context) => {
                            const value = context.parsed;
                            const total = context.dataset.data.reduce((a, b) => a + b, 0);
                            const percentage = total > 0 ? ((value / total) * 100).toFixed(2) : 0;
                            const formatted = this.formatVotingPower(value);
                            return `${context.label}: ${formatted} PAW (${percentage}%)`;
                        }
                    }
                }
            },
            cutout: '60%'
        };
    }

    formatVotingPower(amount) {
        const value = parseInt(amount) / 1000000;
        if (value >= 1000000) {
            return (value / 1000000).toFixed(2) + 'M';
        } else if (value >= 1000) {
            return (value / 1000).toFixed(2) + 'K';
        }
        return value.toLocaleString();
    }

    destroy(canvasId) {
        if (this.charts[canvasId]) {
            this.charts[canvasId].destroy();
            delete this.charts[canvasId];
        }
    }

    destroyAll() {
        Object.keys(this.charts).forEach(canvasId => {
            this.destroy(canvasId);
        });
    }
}
