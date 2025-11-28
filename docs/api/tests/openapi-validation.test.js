#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const yaml = require('js-yaml');
const Ajv = require('ajv');
const addFormats = require('ajv-formats');

console.log('=== OpenAPI Specification Validation ===\n');

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

// Load OpenAPI spec
const specPath = path.join(__dirname, '../openapi.yaml');
let spec;

try {
  const fileContents = fs.readFileSync(specPath, 'utf8');
  spec = yaml.load(fileContents);
  console.log('✅ OpenAPI spec loaded successfully\n');
} catch (error) {
  console.error('❌ Failed to load OpenAPI spec:', error.message);
  process.exit(1);
}

// Test 1: Basic structure
test('OpenAPI spec has required fields', () => {
  if (!spec.openapi) throw new Error('Missing openapi version');
  if (!spec.info) throw new Error('Missing info section');
  if (!spec.paths) throw new Error('Missing paths section');
  if (!spec.components) throw new Error('Missing components section');
});

// Test 2: OpenAPI version
test('OpenAPI version is 3.0.x', () => {
  if (!spec.openapi.startsWith('3.0')) {
    throw new Error(`Expected OpenAPI 3.0.x, got ${spec.openapi}`);
  }
});

// Test 3: Info section
test('Info section is complete', () => {
  if (!spec.info.title) throw new Error('Missing title');
  if (!spec.info.version) throw new Error('Missing version');
  if (!spec.info.description) throw new Error('Missing description');
});

// Test 4: Servers defined
test('At least one server is defined', () => {
  if (!spec.servers || spec.servers.length === 0) {
    throw new Error('No servers defined');
  }
});

// Test 5: Server URLs are valid
test('Server URLs are valid', () => {
  spec.servers.forEach(server => {
    if (!server.url) throw new Error('Server missing URL');
    if (!server.description) throw new Error('Server missing description');
  });
});

// Test 6: Paths exist
test('API paths are defined', () => {
  const pathCount = Object.keys(spec.paths).length;
  if (pathCount === 0) throw new Error('No paths defined');
  console.log(`   Found ${pathCount} paths`);
});

// Test 7: DEX endpoints exist
test('DEX module endpoints are defined', () => {
  const dexPaths = [
    '/paw/dex/v1/pools',
    '/paw/dex/v1/pools/{pool_id}',
    '/paw/dex/v1/create_pool',
    '/paw/dex/v1/swap'
  ];

  dexPaths.forEach(p => {
    if (!spec.paths[p]) throw new Error(`Missing path: ${p}`);
  });
});

// Test 8: Oracle endpoints exist
test('Oracle module endpoints are defined', () => {
  const oraclePaths = [
    '/paw/oracle/v1/prices',
    '/paw/oracle/v1/prices/{asset}'
  ];

  oraclePaths.forEach(p => {
    if (!spec.paths[p]) throw new Error(`Missing path: ${p}`);
  });
});

// Test 9: Compute endpoints exist
test('Compute module endpoints are defined', () => {
  const computePaths = [
    '/paw/compute/v1/tasks',
    '/paw/compute/v1/tasks/{task_id}'
  ];

  computePaths.forEach(p => {
    if (!spec.paths[p]) throw new Error(`Missing path: ${p}`);
  });
});

// Test 10: Cosmos SDK endpoints exist
test('Cosmos SDK endpoints are defined', () => {
  const cosmosPaths = [
    '/cosmos/bank/v1beta1/balances/{address}',
    '/cosmos/staking/v1beta1/validators',
    '/cosmos/gov/v1beta1/proposals'
  ];

  cosmosPaths.forEach(p => {
    if (!spec.paths[p]) throw new Error(`Missing path: ${p}`);
  });
});

// Test 11: All paths have operations
test('All paths have at least one operation', () => {
  Object.entries(spec.paths).forEach(([path, pathItem]) => {
    const operations = ['get', 'post', 'put', 'delete', 'patch'];
    const hasOperation = operations.some(op => pathItem[op]);
    if (!hasOperation) {
      throw new Error(`Path ${path} has no operations`);
    }
  });
});

// Test 12: All operations have operationId
test('All operations have operationId', () => {
  Object.entries(spec.paths).forEach(([path, pathItem]) => {
    ['get', 'post', 'put', 'delete', 'patch'].forEach(method => {
      if (pathItem[method] && !pathItem[method].operationId) {
        throw new Error(`${method.toUpperCase()} ${path} missing operationId`);
      }
    });
  });
});

// Test 13: All operations have tags
test('All operations have tags', () => {
  Object.entries(spec.paths).forEach(([path, pathItem]) => {
    ['get', 'post', 'put', 'delete', 'patch'].forEach(method => {
      if (pathItem[method] && !pathItem[method].tags) {
        throw new Error(`${method.toUpperCase()} ${path} missing tags`);
      }
    });
  });
});

// Test 14: All operations have responses
test('All operations have responses', () => {
  Object.entries(spec.paths).forEach(([path, pathItem]) => {
    ['get', 'post', 'put', 'delete', 'patch'].forEach(method => {
      if (pathItem[method]) {
        if (!pathItem[method].responses) {
          throw new Error(`${method.toUpperCase()} ${path} missing responses`);
        }
        if (!pathItem[method].responses['200']) {
          throw new Error(`${method.toUpperCase()} ${path} missing 200 response`);
        }
      }
    });
  });
});

// Test 15: Schemas are defined
test('Component schemas are defined', () => {
  if (!spec.components.schemas) {
    throw new Error('No schemas defined in components');
  }
  const schemaCount = Object.keys(spec.components.schemas).length;
  if (schemaCount === 0) throw new Error('No schemas defined');
  console.log(`   Found ${schemaCount} schemas`);
});

// Test 16: Required schemas exist
test('Required schemas are defined', () => {
  const requiredSchemas = [
    'Pool',
    'PriceFeed',
    'ComputeTask',
    'Coin',
    'Validator',
    'TxResponse'
  ];

  requiredSchemas.forEach(schema => {
    if (!spec.components.schemas[schema]) {
      throw new Error(`Missing schema: ${schema}`);
    }
  });
});

// Test 17: All schemas have type
test('All schemas have type property', () => {
  Object.entries(spec.components.schemas).forEach(([name, schema]) => {
    if (!schema.type && !schema.$ref) {
      throw new Error(`Schema ${name} missing type`);
    }
  });
});

// Test 18: Path parameters are defined
test('Path parameters are properly defined', () => {
  Object.entries(spec.paths).forEach(([path, pathItem]) => {
    const params = path.match(/\{([^}]+)\}/g);
    if (params) {
      ['get', 'post', 'put', 'delete', 'patch'].forEach(method => {
        if (pathItem[method]) {
          const operation = pathItem[method];
          if (!operation.parameters) {
            throw new Error(`${method.toUpperCase()} ${path} missing parameters definition`);
          }
        }
      });
    }
  });
});

// Test 19: POST operations have requestBody
test('POST operations have requestBody', () => {
  Object.entries(spec.paths).forEach(([path, pathItem]) => {
    if (pathItem.post) {
      // Skip if path is just for query
      if (!path.includes('estimate') && !pathItem.post.requestBody) {
        throw new Error(`POST ${path} missing requestBody`);
      }
    }
  });
});

// Test 20: Error responses are defined
test('Common error responses are defined', () => {
  const errorResponses = ['NotFound', 'InternalError', 'BadRequest'];
  errorResponses.forEach(response => {
    if (!spec.components.responses[response]) {
      throw new Error(`Missing error response: ${response}`);
    }
  });
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
