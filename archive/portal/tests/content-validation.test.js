/**
 * Content Validation Tests
 * Validates content structure, formatting, and completeness
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const CONTENT_DIR = path.join(__dirname, '../content');
const files = [
    'getting-started.md',
    'user-guide.md',
    'developer-guide.md',
    'api-reference.md',
    'tutorials.md',
    'faq.md'
];

const results = {
    passed: 0,
    failed: 0,
    tests: []
};

function test(description, fn) {
    try {
        fn();
        results.passed++;
        results.tests.push({ description, status: 'PASS' });
        console.log(`✅ ${description}`);
    } catch (error) {
        results.failed++;
        results.tests.push({ description, status: 'FAIL', error: error.message });
        console.log(`❌ ${description}`);
        console.log(`   ${error.message}`);
    }
}

function assert(condition, message) {
    if (!condition) {
        throw new Error(message);
    }
}

function testContentValidation() {
    console.log('=== Content Validation Tests ===\n');

    // Test 1: All content files exist
    test('All content files exist', () => {
        files.forEach(file => {
            const filePath = path.join(CONTENT_DIR, file);
            assert(fs.existsSync(filePath), `File not found: ${file}`);
        });
    });

    // Test 2: Files have content
    test('All files have content', () => {
        files.forEach(file => {
            const filePath = path.join(CONTENT_DIR, file);
            const content = fs.readFileSync(filePath, 'utf8');
            assert(content.length > 100, `File ${file} is too short (< 100 chars)`);
        });
    });

    // Test 3: Files start with H1 heading
    test('All files start with H1 heading', () => {
        files.forEach(file => {
            const filePath = path.join(CONTENT_DIR, file);
            const content = fs.readFileSync(filePath, 'utf8');
            assert(content.trim().startsWith('# '), `File ${file} doesn't start with H1`);
        });
    });

    // Test 4: No broken code blocks
    test('All code blocks are properly closed', () => {
        files.forEach(file => {
            const filePath = path.join(CONTENT_DIR, file);
            const content = fs.readFileSync(filePath, 'utf8');
            const backticks = content.match(/```/g);
            if (backticks) {
                assert(backticks.length % 2 === 0,
                    `File ${file} has unclosed code block (${backticks.length} backticks)`);
            }
        });
    });

    // Test 5: Proper heading hierarchy
    test('Headings follow proper hierarchy', () => {
        files.forEach(file => {
            const filePath = path.join(CONTENT_DIR, file);
            const content = fs.readFileSync(filePath, 'utf8');
            const lines = content.split('\n');
            let lastLevel = 0;
            let inCodeBlock = false;

            lines.forEach((line, index) => {
                // Track code blocks
                if (line.trim().startsWith('```')) {
                    inCodeBlock = !inCodeBlock;
                    return;
                }

                // Skip lines in code blocks
                if (inCodeBlock) return;

                if (line.startsWith('#') && line.match(/^#+\s/)) {
                    const level = line.match(/^#+/)[0].length;
                    assert(level <= lastLevel + 1 || level === 1,
                        `File ${file} line ${index + 1}: Heading level jump (h${lastLevel} to h${level})`);
                    lastLevel = level;
                }
            });
        });
    });

    // Test 6: Getting Started has required sections
    test('Getting Started has required sections', () => {
        const content = fs.readFileSync(path.join(CONTENT_DIR, 'getting-started.md'), 'utf8');
        assert(content.includes('What is PAW'), 'Missing "What is PAW" section');
        assert(content.includes('Quick Start'), 'Missing "Quick Start" section');
        assert(content.includes('Installation') || content.includes('Install'), 'Missing Installation section');
    });

    // Test 7: User Guide has required sections
    test('User Guide has required sections', () => {
        const content = fs.readFileSync(path.join(CONTENT_DIR, 'user-guide.md'), 'utf8');
        assert(content.includes('Wallet'), 'Missing Wallet section');
        assert(content.includes('Sending') || content.includes('Transaction'), 'Missing Transaction section');
        assert(content.includes('DEX') || content.includes('Exchange'), 'Missing DEX section');
        assert(content.includes('Staking'), 'Missing Staking section');
    });

    // Test 8: Developer Guide has code examples
    test('Developer Guide has code examples', () => {
        const content = fs.readFileSync(path.join(CONTENT_DIR, 'developer-guide.md'), 'utf8');
        const codeBlocks = content.match(/```/g);
        assert(codeBlocks && codeBlocks.length >= 10,
            'Developer Guide should have at least 5 code examples');
    });

    // Test 9: API Reference has endpoints
    test('API Reference has endpoint documentation', () => {
        const content = fs.readFileSync(path.join(CONTENT_DIR, 'api-reference.md'), 'utf8');
        assert(content.includes('GET ') || content.includes('POST '),
            'API Reference missing HTTP method examples');
        assert(content.includes('Response') || content.includes('response'),
            'API Reference missing response examples');
    });

    // Test 10: Tutorials have step-by-step guides
    test('Tutorials have step-by-step structure', () => {
        const content = fs.readFileSync(path.join(CONTENT_DIR, 'tutorials.md'), 'utf8');
        assert(content.includes('Step') || content.includes('step'),
            'Tutorials should have step-by-step instructions');
        assert(content.includes('Tutorial') || content.includes('tutorial'),
            'Should have tutorial sections');
    });

    // Test 11: FAQ has questions
    test('FAQ has question-answer format', () => {
        const content = fs.readFileSync(path.join(CONTENT_DIR, 'faq.md'), 'utf8');
        const questions = content.match(/###\s+[^#\n]+\?/g);
        assert(questions && questions.length >= 5,
            'FAQ should have at least 5 questions');
    });

    // Test 12: No placeholder text
    test('No placeholder text remaining', () => {
        files.forEach(file => {
            const filePath = path.join(CONTENT_DIR, file);
            const content = fs.readFileSync(filePath, 'utf8').toLowerCase();
            assert(!content.includes('lorem ipsum'), `File ${file} contains placeholder text`);
            assert(!content.includes('todo'), `File ${file} contains TODO markers`);
            assert(!content.includes('tbd'), `File ${file} contains TBD markers`);
        });
    });

    // Test 13: Consistent terminology
    test('Consistent blockchain terminology', () => {
        files.forEach(file => {
            const filePath = path.join(CONTENT_DIR, file);
            const content = fs.readFileSync(filePath, 'utf8');

            // Check for consistent capitalization
            if (content.includes('Paw ') || content.includes('paw ')) {
                // Should be PAW (all caps)
                const wrongCases = content.match(/\b[Pp]aw\s/g);
                if (wrongCases && wrongCases.some(match => match !== 'PAW ')) {
                    results.tests.push({
                        description: `${file}: Inconsistent PAW capitalization`,
                        status: 'WARN'
                    });
                }
            }
        });
    });

    // Print summary
    console.log('\n=== Test Results ===\n');
    console.log(`Passed: ${results.passed}`);
    console.log(`Failed: ${results.failed}`);
    console.log(`Total: ${results.tests.length}`);

    if (results.failed === 0) {
        console.log('\n✅ All Content Validation Tests Passed');
    } else {
        console.log('\n❌ Some Tests Failed');
    }

    console.log('\n=== Test Complete ===\n');

    return results.failed === 0;
}

// Run tests
const passed = testContentValidation();
process.exit(passed ? 0 : 1);
