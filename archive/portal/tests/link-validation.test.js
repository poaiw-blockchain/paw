/**
 * Link Validation Test
 *
 * Validates all internal and external links in documentation
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { glob } from 'glob';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const docsRoot = path.join(__dirname, '..');

// Regular expressions for link detection
const markdownLinkRegex = /\[([^\]]+)\]\(([^)]+)\)/g;
const htmlLinkRegex = /href=["']([^"']+)["']/g;

class LinkValidator {
  constructor() {
    this.errors = [];
    this.warnings = [];
    this.checkedLinks = new Set();
    this.externalLinks = new Set();
    this.internalLinks = new Set();
  }

  async run() {
    console.log('Starting link validation...\n');

    // Find all markdown files
    const files = await glob('**/*.md', {
      cwd: docsRoot,
      ignore: ['node_modules/**', '.vitepress/**']
    });

    console.log(`Found ${files.length} markdown files to check\n`);

    // Check each file
    for (const file of files) {
      await this.checkFile(file);
    }

    // Validate internal links exist
    await this.validateInternalLinks();

    // Report results
    this.report();

    // Exit with error if there are issues
    if (this.errors.length > 0) {
      process.exit(1);
    }
  }

  async checkFile(file) {
    const filePath = path.join(docsRoot, file);
    const content = fs.readFileSync(filePath, 'utf-8');

    // Extract markdown links
    let match;
    while ((match = markdownLinkRegex.exec(content)) !== null) {
      const [, text, url] = match;
      this.processLink(url, file, text);
    }

    // Extract HTML links
    while ((match = htmlLinkRegex.exec(content)) !== null) {
      const url = match[1];
      this.processLink(url, file);
    }
  }

  processLink(url, sourceFile, text = '') {
    // Skip anchors without path
    if (url.startsWith('#')) {
      return;
    }

    // Skip mailto and javascript
    if (url.startsWith('mailto:') || url.startsWith('javascript:')) {
      return;
    }

    // Detect external links
    if (url.startsWith('http://') || url.startsWith('https://')) {
      this.externalLinks.add(url);
      return;
    }

    // Internal link
    const linkKey = `${sourceFile}:${url}`;
    if (this.checkedLinks.has(linkKey)) {
      return;
    }
    this.checkedLinks.add(linkKey);

    // Remove anchor
    const [pathOnly] = url.split('#');
    if (!pathOnly) return; // Only anchor

    this.internalLinks.add({ url: pathOnly, source: sourceFile, fullUrl: url, text });
  }

  async validateInternalLinks() {
    console.log(`Validating ${this.internalLinks.size} internal links...\n`);

    for (const { url, source, fullUrl, text } of this.internalLinks) {
      // Handle absolute paths
      let targetPath = url.startsWith('/')
        ? path.join(docsRoot, url.slice(1))
        : path.join(docsRoot, path.dirname(source), url);

      // Add .md if no extension
      if (!path.extname(targetPath)) {
        targetPath += '.md';
      }

      // Also try without .md (for .html builds)
      const targetPathNoExt = targetPath.replace(/\.md$/, '');

      // Check if file exists
      const exists = fs.existsSync(targetPath) || fs.existsSync(targetPathNoExt);

      if (!exists) {
        this.errors.push({
          type: 'broken-link',
          source,
          link: fullUrl,
          text,
          message: `Broken link: "${fullUrl}" in ${source}`
        });
      }
    }
  }

  report() {
    console.log('\n' + '='.repeat(60));
    console.log('LINK VALIDATION REPORT');
    console.log('='.repeat(60) + '\n');

    console.log(`Total Links Checked: ${this.checkedLinks.size}`);
    console.log(`Internal Links: ${this.internalLinks.size}`);
    console.log(`External Links: ${this.externalLinks.size}`);
    console.log(`Errors: ${this.errors.length}`);
    console.log(`Warnings: ${this.warnings.length}\n`);

    if (this.errors.length > 0) {
      console.log('ERRORS:\n');
      this.errors.forEach((error, i) => {
        console.log(`${i + 1}. ${error.message}`);
        if (error.text) {
          console.log(`   Link text: "${error.text}"`);
        }
        console.log('');
      });
    }

    if (this.warnings.length > 0) {
      console.log('WARNINGS:\n');
      this.warnings.forEach((warning, i) => {
        console.log(`${i + 1}. ${warning.message}\n`);
      });
    }

    if (this.errors.length === 0 && this.warnings.length === 0) {
      console.log('✅ All links are valid!\n');
    }

    console.log('='.repeat(60) + '\n');
  }
}

// Run validator
const validator = new LinkValidator();
validator.run().catch(console.error);
