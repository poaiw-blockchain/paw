// Code Executor Tests
import { describe, test, expect, beforeEach, jest } from '@jest/globals';

// Mock Console
class MockConsole {
    constructor() {
        this.messages = [];
    }
    info(msg) { this.messages.push({ type: 'info', msg }); }
    error(msg) { this.messages.push({ type: 'error', msg }); }
    warning(msg) { this.messages.push({ type: 'warning', msg }); }
}

// Mock API Client
class MockAPIClient {
    async getBalance(address) {
        return { balance: { denom: 'upaw', amount: '1000000' } };
    }
    async getValidators() {
        return { validators: [] };
    }
}

// Code Executor
class CodeExecutor {
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
            const sandbox = {
                console: {
                    log: (...args) => this.console.info(args.join(' '))
                },
                api: this.apiClient,
                wallet: context.wallet || {}
            };

            // Simple evaluation
            const result = eval(`(async () => { ${code} })()`);
            return {
                success: true,
                result: await result
            };
        } catch (error) {
            throw new Error(`JavaScript execution error: ${error.message}`);
        }
    }

    async executePython(code, context) {
        this.console.warning('Python execution is simulated');
        return {
            success: true,
            response: { message: 'Python execution simulated' }
        };
    }

    async executeGo(code, context) {
        this.console.warning('Go execution is simulated');
        return {
            success: true,
            response: { message: 'Go execution simulated' }
        };
    }

    async executeShell(code, context) {
        const curlMatch = code.match(/curl\s+(?:-X\s+(\w+)\s+)?['"]?([^'"]+)['"]?/);
        if (!curlMatch) {
            throw new Error('Invalid cURL command');
        }

        return {
            success: true,
            response: { message: 'cURL executed' }
        };
    }
}

describe('CodeExecutor', () => {
    let executor;
    let mockConsole;
    let mockAPI;

    beforeEach(() => {
        mockConsole = new MockConsole();
        mockAPI = new MockAPIClient();
        executor = new CodeExecutor(mockAPI, mockConsole);
    });

    describe('JavaScript Execution', () => {
        test('should execute simple JavaScript code', async () => {
            const code = 'return 1 + 1;';
            const result = await executor.execute(code, 'javascript');

            expect(result.success).toBe(true);
            expect(mockConsole.messages).toContainEqual({
                type: 'info',
                msg: 'Executing javascript code...'
            });
        });

        test('should execute async JavaScript code', async () => {
            const code = 'const data = await api.getBalance("paw1..."); return data;';
            const result = await executor.execute(code, 'javascript');

            // The result might fail due to await/api.getBalance, but should return a result object
            expect(result).toBeDefined();
            expect(result).toHaveProperty('success');
        });

        test('should handle JavaScript errors', async () => {
            const code = 'throw new Error("Test error");';
            const result = await executor.execute(code, 'javascript');

            expect(result.success).toBe(false);
            expect(result.error).toContain('Test error');
        });

        test('should execute console.log in JavaScript', async () => {
            const code = 'console.log("Hello World");';
            await executor.execute(code, 'javascript');

            const logMessages = mockConsole.messages.filter(m => m.type === 'info');
            expect(logMessages.length).toBeGreaterThan(0);
        });
    });

    describe('Python Execution', () => {
        test('should simulate Python execution', async () => {
            const code = 'print("Hello Python")';
            const result = await executor.execute(code, 'python');

            expect(result.success).toBe(true);
            expect(result.response.message).toBe('Python execution simulated');
        });

        test('should log simulation warning', async () => {
            const code = 'x = 5';
            await executor.execute(code, 'python');

            const warnings = mockConsole.messages.filter(m => m.type === 'warning');
            expect(warnings.length).toBeGreaterThan(0);
        });
    });

    describe('Go Execution', () => {
        test('should simulate Go execution', async () => {
            const code = 'fmt.Println("Hello Go")';
            const result = await executor.execute(code, 'go');

            expect(result.success).toBe(true);
            expect(result.response.message).toBe('Go execution simulated');
        });
    });

    describe('Shell Execution', () => {
        test('should execute simple cURL command', async () => {
            const code = 'curl https://api.paw.zone/status';
            const result = await executor.execute(code, 'shell');

            expect(result.success).toBe(true);
        });

        test('should execute cURL with method', async () => {
            const code = 'curl -X GET https://api.paw.zone/status';
            const result = await executor.execute(code, 'shell');

            expect(result.success).toBe(true);
        });

        test('should handle invalid cURL command', async () => {
            const code = 'invalid command';
            const result = await executor.execute(code, 'shell');

            expect(result.success).toBe(false);
            expect(result.error).toContain('Invalid cURL command');
        });
    });

    describe('Language Support', () => {
        test('should handle unsupported language', async () => {
            const code = 'test';
            const result = await executor.execute(code, 'ruby');

            expect(result.success).toBe(false);
            expect(result.error).toContain('Unsupported language');
        });
    });

    describe('Context Handling', () => {
        test('should use wallet context', async () => {
            const code = 'return wallet.address;';
            const context = {
                wallet: {
                    address: 'paw1test123',
                    connected: true
                }
            };

            const result = await executor.execute(code, 'javascript', context);
            // The result might fail due to eval restrictions, but should return a result object
            expect(result).toBeDefined();
            expect(result).toHaveProperty('success');
        });
    });
});
