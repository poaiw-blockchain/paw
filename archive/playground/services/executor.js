// Code Execution Service

export class CodeExecutor {
    constructor(apiClient, consoleOutput) {
        this.apiClient = apiClient;
        this.console = consoleOutput;
    }

    async execute(code, language, context = {}) {
        try {
            this.console.info(`Executing ${language} code...`);

            switch (language) {
                case 'javascript':
                    return await this.executeJavaScript(code, context);
                case 'python':
                    return await this.executePython(code, context);
                case 'go':
                    return await this.executeGo(code, context);
                case 'shell':
                    return await this.executeShell(code, context);
                default:
                    throw new Error(`Unsupported language: ${language}`);
            }
        } catch (error) {
            this.console.error(`Execution error: ${error.message}`);
            return {
                success: false,
                error: error.message
            };
        }
    }

    async executeJavaScript(code, context) {
        try {
            // Create a sandbox with PAW API helpers
            const sandbox = {
                console: {
                    log: (...args) => this.console.info(args.join(' ')),
                    error: (...args) => this.console.error(args.join(' ')),
                    warn: (...args) => this.console.warning(args.join(' '))
                },
                api: this.apiClient,
                wallet: {
                    address: context.walletAddress,
                    connected: context.walletConnected
                },
                fetch: fetch.bind(window)
            };

            // Wrap code in async function
            const wrappedCode = `
                (async () => {
                    ${code}
                })()
            `;

            // Execute with sandbox context
            const func = new Function(...Object.keys(sandbox), `return ${wrappedCode}`);
            const result = await func(...Object.values(sandbox));

            return {
                success: true,
                result: result,
                response: result
            };
        } catch (error) {
            throw new Error(`JavaScript execution error: ${error.message}`);
        }
    }

    async executePython(code, context) {
        this.console.warning('Python execution is simulated in this playground');

        // Parse and simulate Python code
        try {
            // Look for API calls in Python code
            const apiCallMatch = code.match(/requests\.get\(['"](.+?)['"]\)/);
            if (apiCallMatch) {
                const url = apiCallMatch[1];
                this.console.info(`Simulating API call: ${url}`);

                const response = await fetch(url);
                const data = await response.json();

                return {
                    success: true,
                    response: data
                };
            }

            this.console.info('Python code parsed (simulation mode)');
            return {
                success: true,
                response: { message: 'Python execution simulated' }
            };
        } catch (error) {
            throw new Error(`Python simulation error: ${error.message}`);
        }
    }

    async executeGo(code, context) {
        this.console.warning('Go execution is simulated in this playground');

        try {
            // Parse Go code for API calls
            const apiCallMatch = code.match(/http\.Get\("(.+?)"\)/);
            if (apiCallMatch) {
                const url = apiCallMatch[1];
                this.console.info(`Simulating API call: ${url}`);

                const response = await fetch(url);
                const data = await response.json();

                return {
                    success: true,
                    response: data
                };
            }

            this.console.info('Go code parsed (simulation mode)');
            return {
                success: true,
                response: { message: 'Go execution simulated' }
            };
        } catch (error) {
            throw new Error(`Go simulation error: ${error.message}`);
        }
    }

    async executeShell(code, context) {
        try {
            // Parse cURL commands
            const curlMatch = code.match(/curl\s+(?:-X\s+(\w+)\s+)?['"]?([^'"]+)['"]?/);
            if (!curlMatch) {
                throw new Error('Invalid cURL command');
            }

            const method = curlMatch[1] || 'GET';
            const url = curlMatch[2];

            this.console.info(`Executing cURL: ${method} ${url}`);

            const options = {
                method: method
            };

            // Parse headers
            const headerMatches = code.matchAll(/-H\s+['"](.+?)['"]/g);
            const headers = {};
            for (const match of headerMatches) {
                const [key, value] = match[1].split(':').map(s => s.trim());
                headers[key] = value;
            }

            if (Object.keys(headers).length > 0) {
                options.headers = headers;
            }

            // Parse body
            const bodyMatch = code.match(/-d\s+['"](.+?)['"]/);
            if (bodyMatch) {
                options.body = bodyMatch[1];
            }

            const response = await fetch(url, options);
            const data = await response.json();

            return {
                success: true,
                response: data
            };
        } catch (error) {
            throw new Error(`cURL execution error: ${error.message}`);
        }
    }

    extractTransaction(code, language) {
        try {
            if (language === 'javascript') {
                // Look for transaction objects
                const txMatch = code.match(/(?:const|let|var)\s+\w+\s*=\s*({[\s\S]+?});/);
                if (txMatch) {
                    const txStr = txMatch[1];
                    // Try to parse as JSON
                    return JSON.parse(txStr.replace(/(['"])?([a-zA-Z0-9_]+)(['"])?:/g, '"$2":'));
                }
            }

            return null;
        } catch (error) {
            console.error('Transaction extraction error:', error);
            return null;
        }
    }
}
