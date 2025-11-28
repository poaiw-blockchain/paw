// PAW Playground - Main Application
import { Editor } from './components/Editor.js';
import { Console } from './components/Console.js';
import { ResponseViewer } from './components/ResponseViewer.js';
import { ExampleBrowser } from './components/ExampleBrowser.js';
import { CodeExecutor } from './services/executor.js';
import { APIClient } from './services/apiClient.js';
import { examples } from './examples/index.js';

class PlaygroundApp {
    constructor() {
        this.editor = null;
        this.console = null;
        this.responseViewer = null;
        this.exampleBrowser = null;
        this.executor = null;
        this.apiClient = null;
        this.currentLanguage = 'javascript';
        this.currentNetwork = 'testnet';
        this.walletConnected = false;
        this.walletAddress = null;
        this.snippets = this.loadSnippets();

        this.init();
    }

    async init() {
        try {
            // Initialize components
            this.console = new Console('consoleMessages');
            this.responseViewer = new ResponseViewer('responseContent');
            this.exampleBrowser = new ExampleBrowser('examplesTab', examples);

            // Initialize API client
            this.apiClient = new APIClient(this.currentNetwork);

            // Initialize code executor
            this.executor = new CodeExecutor(this.apiClient, this.console);

            // Initialize Monaco Editor
            await this.initEditor();

            // Setup event listeners
            this.setupEventListeners();

            // Load default example
            this.loadExample('hello-world');

            this.console.info('Playground initialized successfully');
        } catch (error) {
            console.error('Failed to initialize playground:', error);
            this.showToast('Failed to initialize playground', 'error');
        }
    }

    async initEditor() {
        return new Promise((resolve, reject) => {
            require.config({ paths: { vs: 'https://cdn.jsdelivr.net/npm/monaco-editor@0.44.0/min/vs' } });

            require(['vs/editor/editor.main'], () => {
                try {
                    this.editor = new Editor('editorContainer', {
                        language: this.currentLanguage,
                        theme: 'vs-dark',
                        minimap: { enabled: false },
                        fontSize: 14,
                        lineNumbers: 'on',
                        roundedSelection: false,
                        scrollBeyondLastLine: false,
                        readOnly: false,
                        automaticLayout: true
                    });

                    // Listen to content changes
                    this.editor.onChange(() => {
                        this.updateEditorInfo();
                    });

                    resolve();
                } catch (error) {
                    reject(error);
                }
            });
        });
    }

    setupEventListeners() {
        // Network selector
        const networkSelect = document.getElementById('network');
        networkSelect.addEventListener('change', (e) => {
            this.handleNetworkChange(e.target.value);
        });

        // Custom endpoint
        const setCustomEndpoint = document.getElementById('setCustomEndpoint');
        setCustomEndpoint.addEventListener('click', () => {
            this.setCustomEndpoint();
        });

        // Wallet connection
        const connectWallet = document.getElementById('connectWallet');
        connectWallet.addEventListener('click', () => {
            this.connectWallet();
        });

        const disconnectWallet = document.getElementById('disconnectWallet');
        disconnectWallet.addEventListener('click', () => {
            this.disconnectWallet();
        });

        // Editor tabs
        document.querySelectorAll('.editor-tab').forEach(tab => {
            tab.addEventListener('click', (e) => {
                this.switchLanguage(e.target.dataset.lang);
            });
        });

        // Editor actions
        document.getElementById('formatCode').addEventListener('click', () => {
            this.formatCode();
        });

        document.getElementById('shareCode').addEventListener('click', () => {
            this.shareCode();
        });

        document.getElementById('clearEditor').addEventListener('click', () => {
            this.clearEditor();
        });

        // Run code button
        document.getElementById('runCode').addEventListener('click', () => {
            this.runCode();
        });

        // Output tabs
        document.querySelectorAll('.output-tab').forEach(tab => {
            tab.addEventListener('click', (e) => {
                this.switchOutputTab(e.target.dataset.output);
            });
        });

        // Console actions
        document.getElementById('clearConsole').addEventListener('click', () => {
            this.console.clear();
        });

        // Response actions
        document.getElementById('copyResponse').addEventListener('click', () => {
            this.copyResponse();
        });

        // Sidebar tabs
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                this.switchSidebarTab(e.target.dataset.tab);
            });
        });

        // Example items
        document.querySelectorAll('.example-item').forEach(item => {
            item.addEventListener('click', (e) => {
                const exampleId = e.currentTarget.dataset.example;
                this.loadExample(exampleId);
            });
        });

        // Example search
        document.getElementById('exampleSearch').addEventListener('input', (e) => {
            this.searchExamples(e.target.value);
        });

        // Snippet actions
        document.getElementById('saveSnippet').addEventListener('click', () => {
            this.saveSnippet();
        });

        // Transaction builder
        document.getElementById('buildTx').addEventListener('click', () => {
            this.buildTransaction();
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
                e.preventDefault();
                this.runCode();
            }
        });
    }

    handleNetworkChange(network) {
        this.currentNetwork = network;

        if (network === 'custom') {
            document.getElementById('customEndpoint').style.display = 'flex';
        } else {
            document.getElementById('customEndpoint').style.display = 'none';
            this.apiClient.setNetwork(network);
            this.console.info(`Switched to ${network} network`);
        }
    }

    setCustomEndpoint() {
        const input = document.getElementById('customEndpointUrl');
        const url = input.value.trim();

        if (!url) {
            this.showToast('Please enter a valid endpoint URL', 'error');
            return;
        }

        try {
            new URL(url); // Validate URL
            this.apiClient.setCustomEndpoint(url);
            this.console.info(`Connected to custom endpoint: ${url}`);
            this.showToast('Custom endpoint set successfully', 'success');
        } catch (error) {
            this.showToast('Invalid endpoint URL', 'error');
        }
    }

    async connectWallet() {
        try {
            if (!window.keplr) {
                this.showToast('Please install Keplr wallet', 'error');
                window.open('https://www.keplr.app/', '_blank');
                return;
            }

            const chainId = this.apiClient.getChainId();
            await window.keplr.enable(chainId);

            const offlineSigner = window.keplr.getOfflineSigner(chainId);
            const accounts = await offlineSigner.getAccounts();

            this.walletAddress = accounts[0].address;
            this.walletConnected = true;

            // Update UI
            document.getElementById('connectWallet').style.display = 'none';
            document.getElementById('walletInfo').style.display = 'flex';
            document.getElementById('walletAddress').textContent =
                this.walletAddress.slice(0, 10) + '...' + this.walletAddress.slice(-6);

            this.console.success(`Wallet connected: ${this.walletAddress}`);
            this.showToast('Wallet connected successfully', 'success');
        } catch (error) {
            console.error('Wallet connection error:', error);
            this.showToast('Failed to connect wallet', 'error');
        }
    }

    disconnectWallet() {
        this.walletAddress = null;
        this.walletConnected = false;

        document.getElementById('connectWallet').style.display = 'block';
        document.getElementById('walletInfo').style.display = 'none';

        this.console.info('Wallet disconnected');
        this.showToast('Wallet disconnected', 'info');
    }

    switchLanguage(language) {
        this.currentLanguage = language;

        // Update active tab
        document.querySelectorAll('.editor-tab').forEach(tab => {
            tab.classList.toggle('active', tab.dataset.lang === language);
        });

        // Update editor language
        const languageMap = {
            'javascript': 'javascript',
            'python': 'python',
            'go': 'go',
            'shell': 'shell'
        };

        this.editor.setLanguage(languageMap[language]);

        // Update editor info
        const languageNames = {
            'javascript': 'JavaScript',
            'python': 'Python',
            'go': 'Go',
            'shell': 'cURL'
        };
        document.getElementById('editorLanguage').textContent = languageNames[language];

        this.console.info(`Switched to ${languageNames[language]}`);
    }

    formatCode() {
        this.editor.format();
        this.showToast('Code formatted', 'success');
    }

    async shareCode() {
        const code = this.editor.getValue();
        const shareData = {
            language: this.currentLanguage,
            code: code
        };

        const encoded = btoa(JSON.stringify(shareData));
        const url = `${window.location.origin}${window.location.pathname}?share=${encoded}`;

        try {
            await navigator.clipboard.writeText(url);
            this.showToast('Share URL copied to clipboard', 'success');
        } catch (error) {
            console.error('Copy error:', error);
            this.showToast('Failed to copy share URL', 'error');
        }
    }

    clearEditor() {
        this.editor.setValue('');
        this.console.info('Editor cleared');
    }

    async runCode() {
        const code = this.editor.getValue();

        if (!code.trim()) {
            this.showToast('Please enter some code to run', 'warning');
            return;
        }

        this.console.clear();
        this.console.info(`Running ${this.currentLanguage} code...`);

        try {
            const result = await this.executor.execute(code, this.currentLanguage, {
                walletAddress: this.walletAddress,
                walletConnected: this.walletConnected
            });

            if (result.success) {
                this.console.success('Execution completed successfully');

                if (result.response) {
                    this.responseViewer.setResponse(result.response);
                    this.switchOutputTab('response');
                }

                if (result.transaction) {
                    this.displayTransaction(result.transaction);
                }
            } else {
                this.console.error(`Error: ${result.error}`);
                this.showToast('Execution failed', 'error');
            }
        } catch (error) {
            console.error('Execution error:', error);
            this.console.error(`Execution error: ${error.message}`);
            this.showToast('Execution failed', 'error');
        }
    }

    switchOutputTab(tab) {
        // Update active tab
        document.querySelectorAll('.output-tab').forEach(t => {
            t.classList.toggle('active', t.dataset.output === tab);
        });

        // Update active panel
        document.querySelectorAll('.output-panel').forEach(p => {
            p.classList.toggle('active', p.id === `${tab}Output`);
        });
    }

    copyResponse() {
        const responseContent = document.getElementById('responseContent').textContent;

        if (!responseContent) {
            this.showToast('No response to copy', 'warning');
            return;
        }

        navigator.clipboard.writeText(responseContent)
            .then(() => {
                this.showToast('Response copied to clipboard', 'success');
            })
            .catch(() => {
                this.showToast('Failed to copy response', 'error');
            });
    }

    switchSidebarTab(tab) {
        // Update active tab button
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.tab === tab);
        });

        // Update active tab content
        document.getElementById('examplesTab').classList.toggle('active', tab === 'examples');
        document.getElementById('snippetsTab').classList.toggle('active', tab === 'snippets');
    }

    loadExample(exampleId) {
        const example = examples[exampleId];

        if (!example) {
            this.showToast('Example not found', 'error');
            return;
        }

        // Set language
        this.switchLanguage(example.language || 'javascript');

        // Set code
        this.editor.setValue(example.code);

        // Highlight selected example
        document.querySelectorAll('.example-item').forEach(item => {
            item.classList.toggle('active', item.dataset.example === exampleId);
        });

        this.console.info(`Loaded example: ${example.title}`);
    }

    searchExamples(query) {
        const lowerQuery = query.toLowerCase();

        document.querySelectorAll('.example-item').forEach(item => {
            const name = item.querySelector('.example-name').textContent.toLowerCase();
            const match = name.includes(lowerQuery);
            item.style.display = match ? 'flex' : 'none';
        });
    }

    saveSnippet() {
        const code = this.editor.getValue();

        if (!code.trim()) {
            this.showToast('Cannot save empty snippet', 'warning');
            return;
        }

        const name = prompt('Enter snippet name:');
        if (!name) return;

        const snippet = {
            id: Date.now().toString(),
            name: name,
            language: this.currentLanguage,
            code: code,
            created: new Date().toISOString()
        };

        this.snippets.push(snippet);
        this.saveSnippets();
        this.renderSnippets();

        this.showToast('Snippet saved successfully', 'success');
    }

    loadSnippets() {
        try {
            const saved = localStorage.getItem('paw-playground-snippets');
            return saved ? JSON.parse(saved) : [];
        } catch (error) {
            console.error('Failed to load snippets:', error);
            return [];
        }
    }

    saveSnippets() {
        try {
            localStorage.setItem('paw-playground-snippets', JSON.stringify(this.snippets));
        } catch (error) {
            console.error('Failed to save snippets:', error);
        }
    }

    renderSnippets() {
        const container = document.getElementById('snippetsList');

        if (this.snippets.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <p>No saved snippets yet</p>
                    <p class="text-muted">Save your code to access it later</p>
                </div>
            `;
            return;
        }

        container.innerHTML = this.snippets.map(snippet => `
            <div class="snippet-item" data-snippet-id="${snippet.id}">
                <div class="snippet-name">${snippet.name}</div>
                <div class="snippet-meta">${snippet.language} • ${new Date(snippet.created).toLocaleDateString()}</div>
            </div>
        `).join('');

        // Add click listeners
        container.querySelectorAll('.snippet-item').forEach(item => {
            item.addEventListener('click', () => {
                const snippetId = item.dataset.snippetId;
                const snippet = this.snippets.find(s => s.id === snippetId);
                if (snippet) {
                    this.switchLanguage(snippet.language);
                    this.editor.setValue(snippet.code);
                    this.showToast('Snippet loaded', 'success');
                }
            });
        });
    }

    buildTransaction() {
        const code = this.editor.getValue();

        try {
            // Parse transaction from code
            const txData = this.executor.extractTransaction(code, this.currentLanguage);

            if (!txData) {
                this.showToast('No transaction found in code', 'warning');
                return;
            }

            this.displayTransaction(txData);
            this.switchOutputTab('transaction');
        } catch (error) {
            console.error('Transaction build error:', error);
            this.showToast('Failed to build transaction', 'error');
        }
    }

    displayTransaction(txData) {
        const container = document.getElementById('transactionContent');
        container.innerHTML = `
            <div class="transaction-details">
                <h3>Transaction Details</h3>
                <pre>${JSON.stringify(txData, null, 2)}</pre>
            </div>
        `;
    }

    updateEditorInfo() {
        const lineCount = this.editor.getLineCount();
        document.getElementById('editorLines').textContent = `${lineCount} lines`;
    }

    showToast(message, type = 'info') {
        const container = document.getElementById('toastContainer');
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;

        const typeIcons = {
            success: '✅',
            error: '❌',
            warning: '⚠️',
            info: 'ℹ️'
        };

        toast.innerHTML = `
            <div class="toast-header">
                <span>${typeIcons[type]} ${type.charAt(0).toUpperCase() + type.slice(1)}</span>
                <button class="toast-close">&times;</button>
            </div>
            <div class="toast-message">${message}</div>
        `;

        container.appendChild(toast);

        // Auto remove after 5 seconds
        setTimeout(() => {
            toast.remove();
        }, 5000);

        // Manual close
        toast.querySelector('.toast-close').addEventListener('click', () => {
            toast.remove();
        });
    }
}

// Initialize app when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        window.playgroundApp = new PlaygroundApp();
    });
} else {
    window.playgroundApp = new PlaygroundApp();
}

export { PlaygroundApp };
