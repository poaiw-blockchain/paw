#!/usr/bin/env node
/**
 * Verify hardware-critical libraries are pinned to vetted versions.
 * Checks desktop and browser-extension package-lock files for Ledger deps.
 */
const fs = require('fs');
const path = require('path');

const expected = {
  '@ledgerhq/hw-app-cosmos': ['6.32.10'],
  '@ledgerhq/hw-transport-webhid': ['6.30.10'],
  '@ledgerhq/hw-transport-webusb': ['6.29.14'],
  '@ledgerhq/hw-transport-node-hid': ['6.29.15'],
};

const lockFiles = [
  path.join(__dirname, '..', 'browser-extension', 'package-lock.json'),
  path.join(__dirname, '..', 'desktop', 'package-lock.json'),
];

function checkLock(lockPath) {
  const data = JSON.parse(fs.readFileSync(lockPath, 'utf8'));
  const mismatches = [];
  Object.entries(expected).forEach(([pkg, range]) => {
    const entry = data.packages?.[`node_modules/${pkg}`] || data.dependencies?.[pkg];
    if (!entry) {
      // Not all packages are used in every project; skip if absent.
      return;
    }
    const allowed = Array.isArray(range) ? range : [range];
    const match = allowed.some((target) => entry.version && entry.version.startsWith(target));
    if (!match) {
      mismatches.push(`${pkg} expected ${allowed.join(' or ')}, found ${entry.version}`);
    }
  });
  return mismatches;
}

let errors = [];
lockFiles.forEach((file) => {
  if (fs.existsSync(file)) {
    errors = errors.concat(checkLock(file));
  } else {
    errors.push(`Lock file not found: ${file}`);
  }
});

if (errors.length) {
  console.error('[FAIL] Hardware library check failed:\n- ' + errors.join('\n- '));
  process.exit(1);
}

console.log('[OK] Hardware libraries match vetted versions:', Object.keys(expected).join(', '));
