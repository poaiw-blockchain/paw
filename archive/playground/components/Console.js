// Console Output Component

export class Console {
    constructor(containerId) {
        this.containerId = containerId;
        this.container = document.getElementById(containerId);
        this.messages = [];

        if (!this.container) {
            throw new Error(`Container with id "${containerId}" not found`);
        }
    }

    log(message, type = 'info') {
        const messageObj = {
            type,
            message,
            timestamp: new Date().toISOString()
        };

        this.messages.push(messageObj);
        this.render();
    }

    info(message) {
        this.log(message, 'info');
    }

    success(message) {
        this.log(message, 'success');
    }

    warning(message) {
        this.log(message, 'warning');
    }

    error(message) {
        this.log(message, 'error');
    }

    clear() {
        this.messages = [];
        this.render();
    }

    render() {
        if (this.messages.length === 0) {
            this.container.innerHTML = `
                <div class="console-message info">
                    <span class="console-icon">ℹ️</span>
                    <span>Console cleared</span>
                </div>
            `;
            return;
        }

        const icons = {
            info: 'ℹ️',
            success: '✅',
            warning: '⚠️',
            error: '❌'
        };

        this.container.innerHTML = this.messages.map(msg => {
            const time = new Date(msg.timestamp).toLocaleTimeString();
            return `
                <div class="console-message ${msg.type}">
                    <span class="console-icon">${icons[msg.type] || 'ℹ️'}</span>
                    <div>
                        <div>${this.escapeHtml(msg.message)}</div>
                        <small style="opacity: 0.6">${time}</small>
                    </div>
                </div>
            `;
        }).join('');

        // Auto-scroll to bottom
        this.container.scrollTop = this.container.scrollHeight;
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    getMessages() {
        return [...this.messages];
    }

    export() {
        return this.messages.map(msg => {
            const time = new Date(msg.timestamp).toLocaleTimeString();
            return `[${time}] ${msg.type.toUpperCase()}: ${msg.message}`;
        }).join('\n');
    }
}
