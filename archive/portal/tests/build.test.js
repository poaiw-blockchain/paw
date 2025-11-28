/**
 * Build Test
 *
 * Tests that documentation builds successfully
 */

import { exec } from 'child_process';
import { promisify } from 'util';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const execAsync = promisify(exec);
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const docsRoot = path.join(__dirname, '..');

class BuildTester {
  constructor() {
    this.errors = [];
    this.buildDir = path.join(docsRoot, '.vitepress', 'dist');
  }

  async run() {
    console.log('Testing documentation build...\n');

    try {
      // Clean previous build
      await this.cleanBuild();

      // Run build
      await this.build();

      // Verify build output
      await this.verifyBuild();

      // Report results
      this.report();

      if (this.errors.length > 0) {
        process.exit(1);
      }
    } catch (error) {
      console.error('Build failed:', error.message);
      process.exit(1);
    }
  }

  async cleanBuild() {
    console.log('Cleaning previous build...');
    if (fs.existsSync(this.buildDir)) {
      fs.rmSync(this.buildDir, { recursive: true, force: true });
    }
    console.log('✓ Clean complete\n');
  }

  async build() {
    console.log('Building documentation...');
    console.log('This may take a minute...\n');

    try {
      const { stdout, stderr } = await execAsync('npm run build', {
        cwd: docsRoot,
        maxBuffer: 1024 * 1024 * 10 // 10MB buffer
      });

      if (stderr && !stderr.includes('warnings')) {
        console.warn('Build warnings:', stderr);
      }

      console.log('✓ Build complete\n');
    } catch (error) {
      this.errors.push({
        type: 'build-error',
        message: `Build failed: ${error.message}`
      });
      throw error;
    }
  }

  async verifyBuild() {
    console.log('Verifying build output...\n');

    // Check build directory exists
    if (!fs.existsSync(this.buildDir)) {
      this.errors.push({
        type: 'missing-build',
        message: 'Build directory not found'
      });
      return;
    }

    // Check index.html exists
    const indexPath = path.join(this.buildDir, 'index.html');
    if (!fs.existsSync(indexPath)) {
      this.errors.push({
        type: 'missing-index',
        message: 'index.html not found in build output'
      });
    } else {
      console.log('✓ index.html found');
    }

    // Check assets directory
    const assetsDir = path.join(this.buildDir, 'assets');
    if (!fs.existsSync(assetsDir)) {
      this.errors.push({
        type: 'missing-assets',
        message: 'Assets directory not found'
      });
    } else {
      const files = fs.readdirSync(assetsDir);
      console.log(`✓ Assets directory found (${files.length} files)`);
    }

    // Check critical pages
    const criticalPages = [
      'guide/getting-started.html',
      'guide/wallets.html',
      'developer/quick-start.html',
      'validator/setup.html',
      'faq.html',
      'glossary.html'
    ];

    for (const page of criticalPages) {
      const pagePath = path.join(this.buildDir, page);
      if (!fs.existsSync(pagePath)) {
        this.errors.push({
          type: 'missing-page',
          page,
          message: `Critical page not built: ${page}`
        });
      } else {
        console.log(`✓ ${page} built successfully`);
      }
    }

    console.log('');

    // Calculate build size
    const buildSize = this.calculateDirSize(this.buildDir);
    console.log(`Build size: ${(buildSize / 1024 / 1024).toFixed(2)} MB\n`);
  }

  calculateDirSize(dirPath) {
    let totalSize = 0;

    const items = fs.readdirSync(dirPath);
    for (const item of items) {
      const itemPath = path.join(dirPath, item);
      const stats = fs.statSync(itemPath);

      if (stats.isDirectory()) {
        totalSize += this.calculateDirSize(itemPath);
      } else {
        totalSize += stats.size;
      }
    }

    return totalSize;
  }

  report() {
    console.log('\n' + '='.repeat(60));
    console.log('BUILD TEST REPORT');
    console.log('='.repeat(60) + '\n');

    if (this.errors.length > 0) {
      console.log(`Errors: ${this.errors.length}\n`);
      console.log('ERRORS:\n');
      this.errors.forEach((error, i) => {
        console.log(`${i + 1}. [${error.type}] ${error.message}\n`);
      });
    } else {
      console.log('✅ Build test passed!\n');
      console.log('Documentation built successfully with all critical pages.\n');
    }

    console.log('='.repeat(60) + '\n');
  }
}

// Run tester
const tester = new BuildTester();
tester.run().catch(console.error);
