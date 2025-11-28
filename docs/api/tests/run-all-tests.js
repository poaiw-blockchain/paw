#!/usr/bin/env node

const { execSync } = require('child_process');
const path = require('path');

console.log('╔════════════════════════════════════════════════════════╗');
console.log('║   PAW Blockchain API Documentation Test Suite         ║');
console.log('╚════════════════════════════════════════════════════════╝\n');

const tests = [
  { name: 'OpenAPI Validation', file: 'openapi-validation.test.js' },
  { name: 'Code Examples Validation', file: 'examples.test.js' },
  { name: 'Links & Files Validation', file: 'links.test.js' }
];

let allPassed = true;
const results = [];

for (const test of tests) {
  console.log(`\n${'='.repeat(60)}`);
  console.log(`Running: ${test.name}`);
  console.log('='.repeat(60));

  try {
    execSync(`node ${path.join(__dirname, test.file)}`, {
      stdio: 'inherit',
      cwd: __dirname
    });
    results.push({ name: test.name, passed: true });
  } catch (error) {
    allPassed = false;
    results.push({ name: test.name, passed: false });
  }
}

// Print final summary
console.log('\n\n╔════════════════════════════════════════════════════════╗');
console.log('║              Final Test Summary                        ║');
console.log('╚════════════════════════════════════════════════════════╝\n');

results.forEach(result => {
  const status = result.passed ? '✅ PASSED' : '❌ FAILED';
  console.log(`${status}: ${result.name}`);
});

console.log('\n' + '='.repeat(60));

if (allPassed) {
  console.log('✅ All test suites passed!');
  console.log('='.repeat(60) + '\n');
  process.exit(0);
} else {
  console.log('❌ Some test suites failed');
  console.log('='.repeat(60) + '\n');
  process.exit(1);
}
