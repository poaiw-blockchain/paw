#!/usr/bin/env node

import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, dirname, extname } from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const extensionRoot = join(__dirname, '..');
const manifestPath = join(extensionRoot, 'manifest.json');
const srcDir = join(extensionRoot, 'src');

const failures = [];

function fail(message) {
  failures.push(message);
}

function readJson(path) {
  return JSON.parse(readFileSync(path, 'utf8'));
}

function checkManifest() {
  const manifest = readJson(manifestPath);
  const csp = manifest.content_security_policy?.extension_pages;
  if (!csp) {
    fail('manifest.json missing content_security_policy.extension_pages');
  } else {
    const unsafeTokens = ['unsafe-eval', 'unsafe-inline'];
    unsafeTokens.forEach(token => {
      if (csp.includes(token)) {
        fail(`content_security_policy contains forbidden token: ${token}`);
      }
    });
  }

  const hostPermissions = manifest.host_permissions || [];
  hostPermissions.forEach(host => {
    if (!/^https?:\/\//.test(host)) {
      fail(`host_permissions entry must be http(s): ${host}`);
    }
    if (host.includes('*') && !/(localhost|127\.0\.0\.1)/.test(host)) {
      fail(`wildcard host permissions not limited to localhost: ${host}`);
    }
  });

  const allowedPermissions = new Set(['storage', 'alarms']);
  (manifest.permissions || []).forEach(permission => {
    if (!allowedPermissions.has(permission)) {
      fail(`unexpected permission requested: ${permission}`);
    }
  });
  console.log('✓ Manifest hardening checks passed');
}

function collectJsFiles(dir) {
  const files = [];
  const entries = readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...collectJsFiles(fullPath));
    } else if (entry.isFile() && extname(entry.name) === '.js') {
      files.push(fullPath);
    }
  }
  return files;
}

function scanSource() {
  const files = collectJsFiles(srcDir);
  const bannedPatterns = [
    { regex: /\beval\s*\(/g, message: 'eval usage detected' },
    { regex: /new Function\s*\(/g, message: 'Function constructor detected' },
    { regex: /document\.write\s*\(/g, message: 'document.write detected' },
    { regex: /setTimeout\s*\(\s*['"`]/g, message: 'setTimeout string arguments detected' },
    { regex: /setInterval\s*\(\s*['"`]/g, message: 'setInterval string arguments detected' },
  ];

  files.forEach(file => {
    const contents = readFileSync(file, 'utf8');
    for (const pattern of bannedPatterns) {
      pattern.regex.lastIndex = 0;
      if (pattern.regex.test(contents)) {
        fail(`${pattern.message} (${file})`);
      }
    }
  });

  const popupPath = join(srcDir, 'popup.js');
  const popupContents = readFileSync(popupPath, 'utf8');
  if (!popupContents.includes('function escapeHtml(') || !popupContents.includes('function html(')) {
    fail('popup.js missing html sanitization helpers');
  }

  console.log(`✓ Static analysis completed for ${files.length} source files`);
}

function main() {
  checkManifest();
  scanSource();

  if (failures.length > 0) {
    console.error('✗ Browser extension security audit failed:');
    failures.forEach(message => {
      console.error(`  - ${message}`);
    });
    process.exit(1);
  }

  console.log('✓ Browser extension security audit passed');
}

main();
