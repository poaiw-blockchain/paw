/**
 * Log Viewer Component
 * Displays and manages log entries
 */

import monitoringService from '../services/monitoring.js';
import CONFIG from '../config.js';

class LogViewer {
    constructor() {
        this.logs = [];
        this.maxLogs = CONFIG.ui.logsMaxEntries;
        this.logsContainer = null;
    }

    /**
     * Initialize the component
     */
    init() {
        this.logsContainer = document.getElementById('logs-container');

        // Set up event listeners
        document.getElementById('clear-logs-btn')?.addEventListener('click', () => this.clearLogs());
        document.getElementById('export-logs-btn')?.addEventListener('click', () => this.exportLogs());

        // Listen to monitoring service for new logs
        monitoringService.on('newLog', (log) => this.addLog(log));

        // Add initial log
        this.addLog({
            level: 'info',
            message: 'Dashboard initialized',
            timestamp: new Date().toISOString()
        });
    }

    /**
     * Add log entry
     */
    addLog(log) {
        this.logs.push(log);

        // Limit number of logs
        if (this.logs.length > this.maxLogs) {
            this.logs.shift();
        }

        // Update UI
        this.renderLogs();

        // Auto-scroll to bottom
        if (this.logsContainer) {
            this.logsContainer.scrollTop = this.logsContainer.scrollHeight;
        }
    }

    /**
     * Render all logs
     */
    renderLogs() {
        if (!this.logsContainer) return;

        const html = this.logs.map(log => {
            const time = new Date(log.timestamp).toLocaleTimeString();
            return `
                <div class="log-entry ${log.level}">
                    <span class="log-time">${time}</span>
                    <span class="log-message">${this.escapeHtml(log.message)}</span>
                </div>
            `;
        }).join('');

        this.logsContainer.innerHTML = html || '<div class="log-entry info"><span class="log-message">No logs</span></div>';
    }

    /**
     * Clear all logs
     */
    clearLogs() {
        this.logs = [];
        this.renderLogs();
        monitoringService.addLog('info', 'Logs cleared');
    }

    /**
     * Export logs
     */
    exportLogs() {
        const data = this.logs.map(log => ({
            timestamp: log.timestamp,
            level: log.level,
            message: log.message
        }));

        const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `paw-logs-${Date.now()}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);

        monitoringService.addLog('success', 'Logs exported');
    }

    /**
     * Escape HTML to prevent XSS
     */
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    /**
     * Get all logs
     */
    getLogs() {
        return this.logs;
    }
}

export default new LogViewer();
