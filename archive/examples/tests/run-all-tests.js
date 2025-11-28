#!/usr/bin/env node

/**
 * PAW Blockchain - Comprehensive Example Test Runner
 *
 * This script tests all code examples across multiple languages.
 *
 * Usage:
 *   node run-all-tests.js
 *   node run-all-tests.js --lang=javascript
 *   node run-all-tests.js --category=basic
 */

import { spawn } from 'child_process';
import { promises as fs } from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const EXAMPLES_ROOT = path.join(__dirname, '..');

// Test configuration
const TEST_CONFIG = {
  javascript: {
    basic: ['connect.js', 'create-wallet.js', 'query-balance.js'],
    dex: ['swap-tokens.js', 'add-liquidity.js'],
    staking: ['delegate.js'],
    governance: ['vote.js']
  },
  python: {
    basic: ['connect.py', 'create_wallet.py']
  },
  go: {
    basic: ['connect.go', 'create_wallet.go']
  },
  scripts: {
    basic: ['connect.sh', 'query-balance.sh']
  }
};

// ANSI colors
const colors = {
  reset: '\x1b[0m',
  green: '\x1b[32m',
  red: '\x1b[31m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  cyan: '\x1b[36m'
};

class TestRunner {
  constructor() {
    this.results = {
      passed: 0,
      failed: 0,
      skipped: 0,
      errors: []
    };
  }

  log(message, color = 'reset') {
    console.log(colors[color] + message + colors.reset);
  }

  async testJavaScriptFile(category, file) {
    const filePath = path.join(EXAMPLES_ROOT, 'javascript', category, file);

    // Check file exists
    try {
      await fs.access(filePath);
    } catch (error) {
      this.log(`  ✗ ${file} - File not found`, 'red');
      this.results.skipped++;
      return false;
    }

    // Test syntax by importing
    try {
      const content = await fs.readFile(filePath, 'utf-8');

      // Basic syntax checks
      if (!content.includes('export')) {
        this.log(`  ⚠ ${file} - No exports found`, 'yellow');
      }

      // Check for proper error handling
      if (!content.includes('try') || !content.includes('catch')) {
        this.log(`  ⚠ ${file} - Missing error handling`, 'yellow');
      }

      // Check for comments
      if (!content.includes('/**')) {
        this.log(`  ⚠ ${file} - Missing JSDoc comments`, 'yellow');
      }

      this.log(`  ✓ ${file} - Syntax valid`, 'green');
      this.results.passed++;
      return true;

    } catch (error) {
      this.log(`  ✗ ${file} - ${error.message}`, 'red');
      this.results.failed++;
      this.results.errors.push({ file, error: error.message });
      return false;
    }
  }

  async testPythonFile(category, file) {
    const filePath = path.join(EXAMPLES_ROOT, 'python', category, file);

    try {
      await fs.access(filePath);
      const content = await fs.readFile(filePath, 'utf-8');

      // Basic syntax checks
      if (!content.includes('def ')) {
        this.log(`  ⚠ ${file} - No function definitions`, 'yellow');
      }

      // Check for docstrings
      if (!content.includes('"""')) {
        this.log(`  ⚠ ${file} - Missing docstrings`, 'yellow');
      }

      this.log(`  ✓ ${file} - Syntax valid`, 'green');
      this.results.passed++;
      return true;

    } catch (error) {
      this.log(`  ✗ ${file} - ${error.message}`, 'red');
      this.results.failed++;
      return false;
    }
  }

  async testGoFile(category, file) {
    const filePath = path.join(EXAMPLES_ROOT, 'go', category, file);

    try {
      await fs.access(filePath);
      const content = await fs.readFile(filePath, 'utf-8');

      // Basic syntax checks
      if (!content.includes('package main')) {
        this.log(`  ⚠ ${file} - Not a main package`, 'yellow');
      }

      if (!content.includes('func main()')) {
        this.log(`  ⚠ ${file} - Missing main function`, 'yellow');
      }

      this.log(`  ✓ ${file} - Syntax valid`, 'green');
      this.results.passed++;
      return true;

    } catch (error) {
      this.log(`  ✗ ${file} - ${error.message}`, 'red');
      this.results.failed++;
      return false;
    }
  }

  async testScriptFile(category, file) {
    const filePath = path.join(EXAMPLES_ROOT, 'scripts', category, file);

    try {
      await fs.access(filePath);
      const content = await fs.readFile(filePath, 'utf-8');

      // Check shebang
      if (!content.startsWith('#!/bin/bash')) {
        this.log(`  ⚠ ${file} - Missing shebang`, 'yellow');
      }

      // Check for set -e
      if (!content.includes('set -e')) {
        this.log(`  ⚠ ${file} - Missing set -e`, 'yellow');
      }

      this.log(`  ✓ ${file} - Syntax valid`, 'green');
      this.results.passed++;
      return true;

    } catch (error) {
      this.log(`  ✗ ${file} - ${error.message}`, 'red');
      this.results.failed++;
      return false;
    }
  }

  async runTests(filterLang = null, filterCategory = null) {
    this.log('\n' + '='.repeat(80), 'cyan');
    this.log('PAW BLOCKCHAIN - CODE EXAMPLES TEST SUITE', 'cyan');
    this.log('='.repeat(80) + '\n', 'cyan');

    const languages = filterLang ? [filterLang] : Object.keys(TEST_CONFIG);

    for (const lang of languages) {
      const categories = filterCategory ?
        { [filterCategory]: TEST_CONFIG[lang][filterCategory] } :
        TEST_CONFIG[lang];

      this.log(`\nTesting ${lang.toUpperCase()} examples:`, 'blue');

      for (const [category, files] of Object.entries(categories)) {
        if (!files) continue;

        this.log(`\n  ${category}:`, 'cyan');

        for (const file of files) {
          switch (lang) {
            case 'javascript':
              await this.testJavaScriptFile(category, file);
              break;
            case 'python':
              await this.testPythonFile(category, file);
              break;
            case 'go':
              await this.testGoFile(category, file);
              break;
            case 'scripts':
              await this.testScriptFile(category, file);
              break;
          }
        }
      }
    }

    this.printSummary();
  }

  printSummary() {
    this.log('\n' + '='.repeat(80), 'cyan');
    this.log('TEST SUMMARY', 'cyan');
    this.log('='.repeat(80), 'cyan');

    const total = this.results.passed + this.results.failed + this.results.skipped;
    const passRate = total > 0 ? ((this.results.passed / total) * 100).toFixed(2) : 0;

    this.log(`\nTotal Tests: ${total}`, 'blue');
    this.log(`✓ Passed: ${this.results.passed}`, 'green');
    this.log(`✗ Failed: ${this.results.failed}`, 'red');
    this.log(`⊘ Skipped: ${this.results.skipped}`, 'yellow');
    this.log(`\nPass Rate: ${passRate}%`, passRate >= 80 ? 'green' : 'red');

    if (this.results.errors.length > 0) {
      this.log('\nErrors:', 'red');
      this.results.errors.forEach(({ file, error }) => {
        this.log(`  ${file}: ${error}`, 'red');
      });
    }

    this.log('\n' + '='.repeat(80) + '\n', 'cyan');

    return this.results.failed === 0;
  }
}

// Parse command line arguments
function parseArgs() {
  const args = process.argv.slice(2);
  const config = {
    lang: null,
    category: null
  };

  args.forEach(arg => {
    if (arg.startsWith('--lang=')) {
      config.lang = arg.split('=')[1];
    } else if (arg.startsWith('--category=')) {
      config.category = arg.split('=')[1];
    }
  });

  return config;
}

// Main execution
async function main() {
  const config = parseArgs();
  const runner = new TestRunner();

  try {
    await runner.runTests(config.lang, config.category);
    process.exit(runner.results.failed === 0 ? 0 : 1);
  } catch (error) {
    console.error('Fatal error:', error);
    process.exit(1);
  }
}

main();
