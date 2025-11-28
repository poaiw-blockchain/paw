#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

console.log('=== Documentation Links Validation ===\n');

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

// Test main files exist
test('index.html exists', () => {
  const file = path.join(__dirname, '../index.html');
  if (!fs.existsSync(file)) throw new Error('index.html not found');
});

test('openapi.yaml exists', () => {
  const file = path.join(__dirname, '../openapi.yaml');
  if (!fs.existsSync(file)) throw new Error('openapi.yaml not found');
});

test('README.md exists', () => {
  const file = path.join(__dirname, '../README.md');
  if (!fs.existsSync(file)) throw new Error('README.md not found');
});

// Test Swagger UI exists
test('Swagger UI index.html exists', () => {
  const file = path.join(__dirname, '../swagger-ui/index.html');
  if (!fs.existsSync(file)) throw new Error('swagger-ui/index.html not found');
});

// Test Redoc exists
test('Redoc index.html exists', () => {
  const file = path.join(__dirname, '../redoc/index.html');
  if (!fs.existsSync(file)) throw new Error('redoc/index.html not found');
});

// Test Docker files
test('docker-compose.yml exists', () => {
  const file = path.join(__dirname, '../docker-compose.yml');
  if (!fs.existsSync(file)) throw new Error('docker-compose.yml not found');
});

test('nginx.conf exists', () => {
  const file = path.join(__dirname, '../nginx.conf');
  if (!fs.existsSync(file)) throw new Error('nginx.conf not found');
});

// Test Postman collection
test('Postman collection exists', () => {
  const file = path.join(__dirname, '../postman/PAW-API.postman_collection.json');
  if (!fs.existsSync(file)) throw new Error('Postman collection not found');
});

// Test internal links in index.html
test('index.html links to swagger-ui', () => {
  const file = path.join(__dirname, '../index.html');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('./swagger-ui/index.html')) {
    throw new Error('Missing link to swagger-ui');
  }
});

test('index.html links to redoc', () => {
  const file = path.join(__dirname, '../index.html');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('./redoc/index.html')) {
    throw new Error('Missing link to redoc');
  }
});

test('index.html links to openapi.yaml', () => {
  const file = path.join(__dirname, '../index.html');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('./openapi.yaml')) {
    throw new Error('Missing link to openapi.yaml');
  }
});

// Test swagger-ui references openapi.yaml
test('Swagger UI references openapi.yaml', () => {
  const file = path.join(__dirname, '../swagger-ui/index.html');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('../openapi.yaml')) {
    throw new Error('Swagger UI not referencing openapi.yaml');
  }
});

// Test redoc references openapi.yaml
test('Redoc references openapi.yaml', () => {
  const file = path.join(__dirname, '../redoc/index.html');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('../openapi.yaml')) {
    throw new Error('Redoc not referencing openapi.yaml');
  }
});

// Test README has proper sections
test('README has Quick Start section', () => {
  const file = path.join(__dirname, '../README.md');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('Quick Start')) {
    throw new Error('README missing Quick Start section');
  }
});

test('README has Usage Examples section', () => {
  const file = path.join(__dirname, '../README.md');
  const content = fs.readFileSync(file, 'utf8');
  if (!content.includes('Usage Examples')) {
    throw new Error('README missing Usage Examples section');
  }
});

// Test directory structure
test('examples directory exists', () => {
  const dir = path.join(__dirname, '../examples');
  if (!fs.existsSync(dir)) throw new Error('examples directory not found');
});

test('guides directory exists', () => {
  const dir = path.join(__dirname, '../guides');
  if (!fs.existsSync(dir)) throw new Error('guides directory not found');
});

test('tests directory exists', () => {
  const dir = path.join(__dirname, '../tests');
  if (!fs.existsSync(dir)) throw new Error('tests directory not found');
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
