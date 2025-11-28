#!/usr/bin/env node
/**
 * Test Verification Script
 * Validates that all components are properly structured
 */

const fs = require('fs');
const path = require('path');

console.log('='.repeat(80));
console.log('PAW Governance Portal - File Structure Verification');
console.log('='.repeat(80));

const baseDir = path.join(__dirname, '..');
const requiredFiles = [
    'index.html',
    'app.js',
    'services/governanceAPI.js',
    'components/ProposalList.js',
    'components/ProposalDetail.js',
    'components/CreateProposal.js',
    'components/VotingPanel.js',
    'components/TallyChart.js',
    'assets/css/styles.css',
    'tests/governance.test.js',
    'tests/test-runner.html',
    'README.md'
];

let passed = 0;
let failed = 0;

console.log('\nChecking required files...\n');

requiredFiles.forEach(file => {
    const filePath = path.join(baseDir, file);
    if (fs.existsSync(filePath)) {
        const stats = fs.statSync(filePath);
        console.log(`✓ PASS: ${file} (${stats.size} bytes)`);
        passed++;
    } else {
        console.log(`✗ FAIL: ${file} (missing)`);
        failed++;
    }
});

console.log('\n' + '='.repeat(80));
console.log(`Results: ${passed} passed, ${failed} failed, ${requiredFiles.length} total`);
console.log('='.repeat(80));

// Verify file contents
console.log('\nVerifying file contents...\n');

const validations = [
    {
        file: 'app.js',
        check: (content) => content.includes('class GovernanceApp'),
        desc: 'Main app class exists'
    },
    {
        file: 'services/governanceAPI.js',
        check: (content) => content.includes('class GovernanceAPI'),
        desc: 'API service class exists'
    },
    {
        file: 'components/ProposalList.js',
        check: (content) => content.includes('class ProposalList'),
        desc: 'ProposalList component exists'
    },
    {
        file: 'components/ProposalDetail.js',
        check: (content) => content.includes('class ProposalDetail'),
        desc: 'ProposalDetail component exists'
    },
    {
        file: 'components/CreateProposal.js',
        check: (content) => content.includes('class CreateProposal'),
        desc: 'CreateProposal component exists'
    },
    {
        file: 'components/VotingPanel.js',
        check: (content) => content.includes('class VotingPanel'),
        desc: 'VotingPanel component exists'
    },
    {
        file: 'components/TallyChart.js',
        check: (content) => content.includes('class TallyChart'),
        desc: 'TallyChart component exists'
    },
    {
        file: 'tests/governance.test.js',
        check: (content) => content.includes('class TestRunner'),
        desc: 'Test runner exists'
    },
    {
        file: 'index.html',
        check: (content) => content.includes('PAW Governance Portal'),
        desc: 'HTML title correct'
    },
    {
        file: 'assets/css/styles.css',
        check: (content) => content.includes('.governance-container'),
        desc: 'CSS styles defined'
    }
];

let contentPassed = 0;
let contentFailed = 0;

validations.forEach(({ file, check, desc }) => {
    const filePath = path.join(baseDir, file);
    if (fs.existsSync(filePath)) {
        const content = fs.readFileSync(filePath, 'utf8');
        if (check(content)) {
            console.log(`✓ PASS: ${desc}`);
            contentPassed++;
        } else {
            console.log(`✗ FAIL: ${desc}`);
            contentFailed++;
        }
    } else {
        console.log(`✗ FAIL: ${desc} (file missing)`);
        contentFailed++;
    }
});

console.log('\n' + '='.repeat(80));
console.log(`Content Validation: ${contentPassed} passed, ${contentFailed} failed, ${validations.length} total`);
console.log('='.repeat(80));

// Count lines of code
console.log('\nCode Statistics...\n');

let totalLines = 0;
let totalSize = 0;

requiredFiles.forEach(file => {
    const filePath = path.join(baseDir, file);
    if (fs.existsSync(filePath)) {
        const content = fs.readFileSync(filePath, 'utf8');
        const lines = content.split('\n').length;
        const size = fs.statSync(filePath).size;
        totalLines += lines;
        totalSize += size;
        console.log(`${file}: ${lines} lines, ${(size / 1024).toFixed(2)} KB`);
    }
});

console.log('\n' + '-'.repeat(80));
console.log(`Total: ${totalLines} lines, ${(totalSize / 1024).toFixed(2)} KB`);
console.log('='.repeat(80));

// Final verdict
if (failed === 0 && contentFailed === 0) {
    console.log('\n✓ All checks passed! Governance portal is ready for use.\n');
    process.exit(0);
} else {
    console.log('\n✗ Some checks failed. Please review the output above.\n');
    process.exit(1);
}
