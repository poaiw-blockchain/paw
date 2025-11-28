// API Response Viewer Component

export class ResponseViewer {
    constructor(containerId) {
        this.containerId = containerId;
        this.container = document.getElementById(containerId);
        this.currentResponse = null;

        if (!this.container) {
            throw new Error(`Container with id "${containerId}" not found`);
        }
    }

    setResponse(response) {
        this.currentResponse = response;
        this.render();
    }

    render() {
        if (!this.currentResponse) {
            this.container.innerHTML = `
                <div class="empty-state">
                    <p>No response yet</p>
                    <p class="text-muted">Run code to see API responses</p>
                </div>
            `;
            return;
        }

        let formattedResponse;
        try {
            if (typeof this.currentResponse === 'string') {
                try {
                    const parsed = JSON.parse(this.currentResponse);
                    formattedResponse = JSON.stringify(parsed, null, 2);
                } catch {
                    formattedResponse = this.currentResponse;
                }
            } else {
                formattedResponse = JSON.stringify(this.currentResponse, null, 2);
            }
        } catch (error) {
            formattedResponse = String(this.currentResponse);
        }

        this.container.innerHTML = `<pre><code class="language-json">${this.escapeHtml(formattedResponse)}</code></pre>`;

        // Apply syntax highlighting if available
        if (typeof hljs !== 'undefined') {
            this.container.querySelectorAll('pre code').forEach(block => {
                hljs.highlightElement(block);
            });
        }
    }

    clear() {
        this.currentResponse = null;
        this.render();
    }

    getResponse() {
        return this.currentResponse;
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    download(filename = 'response.json') {
        if (!this.currentResponse) {
            return;
        }

        const data = typeof this.currentResponse === 'string'
            ? this.currentResponse
            : JSON.stringify(this.currentResponse, null, 2);

        const blob = new Blob([data], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        a.click();
        URL.revokeObjectURL(url);
    }
}
