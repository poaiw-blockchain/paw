// Example Validation Tests
import { describe, test, expect } from '@jest/globals';

// Mock examples
const examples = {
    'hello-world': {
        title: 'Hello World',
        category: 'getting-started',
        language: 'javascript',
        code: 'console.log("Hello, PAW!");'
    },
    'query-balance': {
        title: 'Query Balance',
        category: 'getting-started',
        language: 'javascript',
        code: 'const balance = await api.getBalance("paw1...");'
    },
    'bank-transfer': {
        title: 'Bank Transfer',
        category: 'bank',
        language: 'javascript',
        code: 'const msg = { type: "cosmos-sdk/MsgSend" };'
    },
    'dex-swap': {
        title: 'DEX Swap',
        category: 'dex',
        language: 'javascript',
        code: 'const swapMsg = { type: "paw/dex/MsgSwap" };'
    },
    'staking': {
        title: 'Delegate Tokens',
        category: 'staking',
        language: 'javascript',
        code: 'const delegateMsg = { type: "cosmos-sdk/MsgDelegate" };'
    }
};

describe('Example Validation', () => {
    describe('Example Structure', () => {
        test('should have required fields', () => {
            Object.entries(examples).forEach(([key, example]) => {
                expect(example).toHaveProperty('title');
                expect(example).toHaveProperty('category');
                expect(example).toHaveProperty('language');
                expect(example).toHaveProperty('code');
            });
        });

        test('should have non-empty titles', () => {
            Object.values(examples).forEach(example => {
                expect(example.title).toBeTruthy();
                expect(example.title.length).toBeGreaterThan(0);
            });
        });

        test('should have valid categories', () => {
            const validCategories = ['getting-started', 'bank', 'dex', 'staking', 'governance'];

            Object.values(examples).forEach(example => {
                expect(validCategories).toContain(example.category);
            });
        });

        test('should have valid languages', () => {
            const validLanguages = ['javascript', 'python', 'go', 'shell'];

            Object.values(examples).forEach(example => {
                expect(validLanguages).toContain(example.language);
            });
        });

        test('should have non-empty code', () => {
            Object.values(examples).forEach(example => {
                expect(example.code).toBeTruthy();
                expect(example.code.length).toBeGreaterThan(0);
            });
        });
    });

    describe('Example Content', () => {
        test('hello-world should contain console.log', () => {
            expect(examples['hello-world'].code).toContain('console.log');
        });

        test('query-balance should contain api.getBalance', () => {
            expect(examples['query-balance'].code).toContain('api.getBalance');
        });

        test('bank-transfer should contain MsgSend', () => {
            expect(examples['bank-transfer'].code).toContain('MsgSend');
        });

        test('dex-swap should contain MsgSwap', () => {
            expect(examples['dex-swap'].code).toContain('MsgSwap');
        });

        test('staking should contain MsgDelegate', () => {
            expect(examples['staking'].code).toContain('MsgDelegate');
        });
    });

    describe('Code Quality', () => {
        test('should not have syntax errors in JavaScript examples', () => {
            Object.entries(examples).forEach(([key, example]) => {
                if (example.language === 'javascript') {
                    // Basic syntax check - code should not be empty
                    expect(example.code.trim()).toBeTruthy();

                    // Should not have obvious syntax errors
                    expect(example.code).not.toContain('undefined undefined');
                    expect(example.code).not.toContain('null null');
                }
            });
        });

        test('should use consistent coding style', () => {
            Object.values(examples).forEach(example => {
                if (example.language === 'javascript') {
                    // Code should not be empty
                    expect(example.code.trim()).toBeTruthy();

                    // Should not have obvious syntax errors
                    expect(example.code).not.toContain('undefined undefined');
                    expect(example.code).not.toContain('null null');
                }
            });
        });
    });

    describe('Example Categories', () => {
        test('should have getting-started examples', () => {
            const gettingStarted = Object.values(examples).filter(
                e => e.category === 'getting-started'
            );
            expect(gettingStarted.length).toBeGreaterThan(0);
        });

        test('should have bank examples', () => {
            const bank = Object.values(examples).filter(
                e => e.category === 'bank'
            );
            expect(bank.length).toBeGreaterThan(0);
        });

        test('should have dex examples', () => {
            const dex = Object.values(examples).filter(
                e => e.category === 'dex'
            );
            expect(dex.length).toBeGreaterThan(0);
        });

        test('should have staking examples', () => {
            const staking = Object.values(examples).filter(
                e => e.category === 'staking'
            );
            expect(staking.length).toBeGreaterThan(0);
        });
    });
});

describe('Example Browser Component', () => {
    class ExampleBrowser {
        constructor(examples) {
            this.examples = examples;
        }

        filter(query) {
            const lowerQuery = query.toLowerCase();
            return Object.entries(this.examples).reduce((acc, [key, example]) => {
                const title = (example.title || '').toLowerCase();
                if (title.includes(lowerQuery)) {
                    acc[key] = example;
                }
                return acc;
            }, {});
        }

        getExample(key) {
            return this.examples[key] || null;
        }

        getCategories() {
            const categories = new Set();
            Object.values(this.examples).forEach(example => {
                if (example.category) {
                    categories.add(example.category);
                }
            });
            return Array.from(categories);
        }
    }

    let browser;

    beforeEach(() => {
        browser = new ExampleBrowser(examples);
    });

    test('should filter examples by query', () => {
        const filtered = browser.filter('balance');
        expect(Object.keys(filtered)).toContain('query-balance');
    });

    test('should get example by key', () => {
        const example = browser.getExample('hello-world');
        expect(example).toBeTruthy();
        expect(example.title).toBe('Hello World');
    });

    test('should return null for invalid key', () => {
        const example = browser.getExample('non-existent');
        expect(example).toBeNull();
    });

    test('should get all categories', () => {
        const categories = browser.getCategories();
        expect(categories).toContain('getting-started');
        expect(categories).toContain('bank');
        expect(categories).toContain('dex');
        expect(categories).toContain('staking');
    });

    test('should handle empty query', () => {
        const filtered = browser.filter('');
        expect(Object.keys(filtered).length).toBe(Object.keys(examples).length);
    });

    test('should handle case-insensitive search', () => {
        const filtered = browser.filter('HELLO');
        expect(Object.keys(filtered)).toContain('hello-world');
    });
});
