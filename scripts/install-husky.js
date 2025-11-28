#!/usr/bin/env node
// PAW Blockchain - Husky Installation Helper
// This script is run automatically by npm after installing dependencies

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const huskyDir = path.join(__dirname, '..', '.husky');
const gitDir = path.join(__dirname, '..', '');

// Check if we're in a  repository
if (!fs.existsSync(gitDir)) {
  console.log('Not a  repository, skipping husky installation');
  process.exit(0);
}

// Check if husky is installed
try {
  require.resolve('husky');
} catch (e) {
  console.log('Husky not installed, skipping hook setup');
  process.exit(0);
}

console.log('Setting up  hooks with Husky...');

try {
  // Install husky hooks
  execSync('npx husky install', { stdio: 'inherit' });

  // Make hook files executable (Unix-like systems)
  if (process.platform !== 'win32') {
    const hooks = ['pre-commit', 'commit-msg', 'pre-push'];
    hooks.forEach(hook => {
      const hookPath = path.join(huskyDir, hook);
      if (fs.existsSync(hookPath)) {
        fs.chmodSync(hookPath, '755');
        console.log(`✓ Made ${hook} executable`);
      }
    });
  }

  console.log('✓ Husky hooks installed successfully');
} catch (error) {
  console.error('Failed to install husky hooks:', error.message);
  process.exit(1);
}
