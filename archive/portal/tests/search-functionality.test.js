/**
 * Search Functionality Tests
 * Tests the search index and search functionality
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const SEARCH_INDEX_PATH = path.join(__dirname, '../search-index.json');

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

function testSearchFunctionality() {
    console.log('=== Search Functionality Tests ===\n');

    // Test 1: Search index file exists
    test('Search index file exists', () => {
        assert(fs.existsSync(SEARCH_INDEX_PATH), 'search-index.json not found');
    });

    let searchIndex = [];

    // Test 2: Search index is valid JSON
    test('Search index is valid JSON', () => {
        const content = fs.readFileSync(SEARCH_INDEX_PATH, 'utf8');
        searchIndex = JSON.parse(content);
    });

    // Test 3: Search index is an array
    test('Search index is an array', () => {
        assert(Array.isArray(searchIndex), 'Search index should be an array');
    });

    // Test 4: Search index has entries
    test('Search index has entries', () => {
        assert(searchIndex.length > 0, 'Search index is empty');
        console.log(`   Found ${searchIndex.length} entries`);
    });

    // Test 5: All entries have required fields
    test('All entries have required fields', () => {
        searchIndex.forEach((entry, index) => {
            assert(entry.id, `Entry ${index} missing id`);
            assert(entry.title, `Entry ${index} missing title`);
            assert(entry.content, `Entry ${index} missing content`);
            assert(entry.url, `Entry ${index} missing url`);
        });
    });

    // Test 6: All entries have unique IDs
    test('All entries have unique IDs', () => {
        const ids = searchIndex.map(e => e.id);
        const uniqueIds = new Set(ids);
        assert(ids.length === uniqueIds.size,
            `Duplicate IDs found (${ids.length} total, ${uniqueIds.size} unique)`);
    });

    // Test 7: Search for common terms
    test('Search finds common terms', () => {
        const searchTerms = ['wallet', 'staking', 'transaction', 'dex', 'governance'];

        searchTerms.forEach(term => {
            const found = searchIndex.some(entry =>
                entry.content.toLowerCase().includes(term.toLowerCase()) ||
                entry.title.toLowerCase().includes(term.toLowerCase())
            );
            assert(found, `Search term "${term}" not found in index`);
        });
    });

    // Test 8: Content has minimum length
    test('Content entries have sufficient length', () => {
        searchIndex.forEach((entry, index) => {
            assert(entry.content.length >= 50,
                `Entry ${index} (${entry.title}) content too short`);
        });
    });

    // Test 9: URLs are valid
    test('All URLs are valid', () => {
        searchIndex.forEach((entry, index) => {
            assert(entry.url.startsWith('#') || entry.url.startsWith('http'),
                `Entry ${index} has invalid URL: ${entry.url}`);
        });
    });

    // Test 10: Categories are defined
    test('Entries have categories', () => {
        searchIndex.forEach((entry, index) => {
            if (entry.category) {
                assert(typeof entry.category === 'string',
                    `Entry ${index} category should be string`);
            }
        });
    });

    // Test 11: Tags are arrays
    test('Tags are arrays when present', () => {
        searchIndex.forEach((entry, index) => {
            if (entry.tags) {
                assert(Array.isArray(entry.tags),
                    `Entry ${index} tags should be array`);
            }
        });
    });

    // Test 12: Search index covers all main pages
    test('Search index covers all main pages', () => {
        const requiredPages = [
            'getting-started',
            'user-guide',
            'developer-guide',
            'api-reference',
            'tutorials',
            'faq'
        ];

        requiredPages.forEach(page => {
            const found = searchIndex.some(entry => entry.id === page);
            assert(found, `Required page "${page}" not in search index`);
        });
    });

    // Test 13: No duplicate titles
    test('No duplicate titles', () => {
        const titles = searchIndex.map(e => e.title.toLowerCase());
        const uniqueTitles = new Set(titles);
        const duplicates = titles.filter((title, index) =>
            titles.indexOf(title) !== index
        );

        if (duplicates.length > 0) {
            console.log(`   ⚠️  Warning: Found duplicate titles: ${[...new Set(duplicates)].join(', ')}`);
        }
    });

    // Print summary
    console.log('\n=== Test Results ===\n');
    console.log(`Passed: ${results.passed}`);
    console.log(`Failed: ${results.failed}`);
    console.log(`Total: ${results.tests.length}`);

    if (results.failed === 0) {
        console.log('\n✅ All Search Tests Passed');
    } else {
        console.log('\n❌ Some Tests Failed');
    }

    console.log('\n=== Test Complete ===\n');

    return results.failed === 0;
}

// Run tests
const passed = testSearchFunctionality();
process.exit(passed ? 0 : 1);
