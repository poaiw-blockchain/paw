#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

console.log('=== Code Examples Validation ===\n');

let passedTests = 0;
let failedTests = 0;

function test(name, fn) {
  try {
    fn();
    console.log(`✅ PASS: ${name}`);
    passedTests++;
  } catch (error) {
    console.log(`❌ FAIL: ${name}`);
    console.log(`   Error: ${error.message}`);
    failedTests++;
  }
}

// Test example files exist
const examplesDir = path.join(__dirname, '../examples');
const guideDir = path.join(__dirname, '../guides');

test('cURL examples file exists', () => {
  const file = path.join(examplesDir, 'curl.md');
  if (!fs.existsSync(file)) throw new Error('curl.md not found');
});

test('JavaScript examples file exists', () => {
  const file = path.join(examplesDir, 'javascript.md');
  if (!fs.existsSync(file)) throw new Error('javascript.md not found');
});

test('Python examples file exists', () => {
  const file = path.join(examplesDir, 'python.md');
  if (!fs.existsSync(file)) throw new Error('python.md not found');
});

test('Go examples file exists', () => {
  const file = path.join(examplesDir, 'go.md');
  if (!fs.existsSync(file)) throw new Error('go.md not found');
});

// Test guide files exist
test('Authentication guide exists', () => {
  const file = path.join(guideDir, 'authentication.md');
  if (!fs.existsSync(file)) throw new Error('authentication.md not found');
});

test('WebSocket guide exists', () => {
  const file = path.join(guideDir, 'websockets.md');
  if (!fs.existsSync(file)) throw new Error('websockets.md not found');
});

test('Rate limiting guide exists', () => {
  const file = path.join(guideDir, 'rate-limiting.md');
  if (!fs.existsSync(file)) throw new Error('rate-limiting.md not found');
});

test('Error codes guide exists', () => {
  const file = path.join(guideDir, 'errors.md');
  if (!fs.existsSync(file)) throw new Error('errors.md not found');
});

// Test curl examples content
test('cURL examples contain DEX endpoints', () => {
  const file = path.join(examplesDir, 'curl.md');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('/paw/dex/v1/pools')) {
    throw new Error('Missing DEX pool examples');
  }
});

test('cURL examples contain Oracle endpoints', () => {
  const file = path.join(examplesDir, 'curl.md');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('/paw/oracle/v1/prices')) {
    throw new Error('Missing Oracle price examples');
  }
});

// Test JavaScript examples content
test('JavaScript examples have client class', () => {
  const file = path.join(examplesDir, 'javascript.md');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('class PAWClient')) {
    throw new Error('Missing PAWClient class');
  }
});

test('JavaScript examples have async/await usage', () => {
  const file = path.join(examplesDir, 'javascript.md');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('async') || !content.includes('await')) {
    throw new Error('Missing async/await examples');
  }
});

// Test Python examples content
test('Python examples have client class', () => {
  const file = path.join(examplesDir, 'python.md');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('class PAWClient')) {
    throw new Error('Missing PAWClient class');
  }
});

test('Python examples use requests library', () => {
  const file = path.join(examplesDir, 'python.md');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('import requests')) {
    throw new Error('Missing requests import');
  }
});

// Test Go examples content
test('Go examples have client struct', () => {
  const file = path.join(examplesDir, 'go.md');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('type PAWClient struct')) {
    throw new Error('Missing PAWClient struct');
  }
});

test('Go examples have proper error handling', () => {
  const file = path.join(examplesDir, 'go.md');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('if err != nil')) {
    throw new Error('Missing error handling');
  }
});

// Test that examples reference correct API endpoints
test('Examples use correct base URL pattern', () => {
  const files = [
    path.join(examplesDir, 'curl.md'),
    path.join(examplesDir, 'javascript.md'),
    path.join(examplesDir, 'python.md'),
    path.join(examplesDir, 'go.md')
  ];

  files.forEach(file => {
    const content = fs.readFileSync(file, 'utf8');
    if (!content.includes('localhost:1317') && !content.includes('API_URL')) {
      throw new Error(`${path.basename(file)} missing API URL configuration`);
    }
  });
});

// Test guides content
test('Authentication guide covers transaction signing', () => {
  const file = path.join(guideDir, 'authentication.md');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('signing') && !content.includes('transaction')) {
    throw new Error('Missing transaction signing information');
  }
});

test('WebSocket guide has connection example', () => {
  const file = path.join(guideDir, 'websockets.md');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('WebSocket') && !content.includes('ws://')) {
    throw new Error('Missing WebSocket connection example');
  }
});

test('Rate limiting guide explains limits', () => {
  const file = path.join(guideDir, 'rate-limiting.md');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('100') || !content.includes('minute')) {
    throw new Error('Missing rate limit information');
  }
});

test('Error guide lists HTTP status codes', () => {
  const file = path.join(guideDir, 'errors.md');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('400') || !content.includes('404') || !content.includes('500')) {
    throw new Error('Missing HTTP status codes');
  }
});

// Print summary
console.log('\n=== Test Summary ===');
console.log(`Total tests: ${passedTests + failedTests}`);
console.log(`Passed: ${passedTests}`);
console.log(`Failed: ${failedTests}`);

if (failedTests > 0) {
  console.log('\n❌ Some tests failed');
  process.exit(1);
} else {
  console.log('\n✅ All tests passed!');
  process.exit(0);
}
