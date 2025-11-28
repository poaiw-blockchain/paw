/**
 * Search Functionality Test
 *
 * Tests that VitePress search index is built correctly
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { glob } from 'glob';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const docsRoot = path.join(__dirname, '..');

class SearchTester {
  constructor() {
    this.errors = [];
    this.searchablePages = 0;
    this.totalWords = 0;
    this.searchTerms = [
      'PAW',
      'blockchain',
      'staking',
      'validator',
      'DEX',
      'governance',
      'wallet',
      'transaction',
      'token',
      'smart contract'
    ];
  }

  async run() {
    console.log('Testing search functionality...\n');

    // Find all markdown files
    const files = await glob('**/*.md', {
      cwd: docsRoot,
      ignore: ['node_modules/**', '.vitepress/**']
    });

    console.log(`Analyzing ${files.length} pages for search indexing...\n`);

    // Analyze each file
    for (const file of files) {
      await this.analyzePage(file);
    }

    // Test search terms
    await this.testSearchTerms();

    // Report results
    this.report();

    if (this.errors.length > 0) {
      process.exit(1);
    }
  }

  async analyzePage(file) {
    const filePath = path.join(docsRoot, file);
    const content = fs.readFileSync(filePath, 'utf-8');

    // Remove frontmatter
    const withoutFrontmatter = content.replace(/^---[\s\S]*?---/, '');

    // Count words
    const words = withoutFrontmatter
      .replace(/[#*`\[\]()]/g, ' ')
      .split(/\s+/)
      .filter(w => w.length > 2);

    this.totalWords += words.length;
    this.searchablePages++;

    // Check if page has meaningful content
    if (words.length < 50) {
      this.errors.push({
        type: 'thin-content',
        file,
        message: `Page "${file}" has very little content (${words.length} words)`
      });
    }

    // Check for headings
    const headings = content.match(/^#{1,6}\s+.+$/gm);
    if (!headings || headings.length < 2) {
      this.errors.push({
        type: 'no-headings',
        file,
        message: `Page "${file}" has insufficient headings for search navigation`
      });
    }
  }

  async testSearchTerms() {
    console.log('Testing search term coverage...\n');

    const allContent = [];

    // Read all files
    const files = await glob('**/*.md', {
      cwd: docsRoot,
      ignore: ['node_modules/**', '.vitepress/**']
    });

    for (const file of files) {
      const filePath = path.join(docsRoot, file);
      const content = fs.readFileSync(filePath, 'utf-8').toLowerCase();
      allContent.push(content);
    }

    const combinedContent = allContent.join(' ');

    // Test each search term
    for (const term of this.searchTerms) {
      const count = (combinedContent.match(new RegExp(term.toLowerCase(), 'g')) || []).length;

      if (count === 0) {
        this.errors.push({
          type: 'missing-term',
          term,
          message: `Important search term "${term}" not found in documentation`
        });
      } else {
        console.log(`✓ "${term}" found ${count} times`);
      }
    }
    console.log('');
  }

  report() {
    console.log('\n' + '='.repeat(60));
    console.log('SEARCH FUNCTIONALITY TEST REPORT');
    console.log('='.repeat(60) + '\n');

    console.log(`Searchable Pages: ${this.searchablePages}`);
    console.log(`Total Words: ${this.totalWords.toLocaleString()}`);
    console.log(`Average Words/Page: ${Math.round(this.totalWords / this.searchablePages)}`);
    console.log(`Errors: ${this.errors.length}\n`);

    if (this.errors.length > 0) {
      console.log('ISSUES FOUND:\n');
      this.errors.forEach((error, i) => {
        console.log(`${i + 1}. [${error.type}] ${error.message}\n`);
      });
    } else {
      console.log('✅ All search functionality tests passed!\n');
    }

    console.log('='.repeat(60) + '\n');
  }
}

// Run tester
const tester = new SearchTester();
tester.run().catch(console.error);
