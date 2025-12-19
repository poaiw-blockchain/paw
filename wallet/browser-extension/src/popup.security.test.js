import { describe, it, expect, beforeAll } from 'vitest';
import { detectSignMode, enforceOriginHealth, enforceRateLimit, normalizeBytes } from './popup.js';

beforeAll(() => {
  if (typeof atob === 'undefined') {
    global.atob = (str) => Buffer.from(str, 'base64').toString('binary');
  }
});

describe('walletconnect security helpers', () => {
  it('detects direct sign mode from proto fields', () => {
    expect(detectSignMode({ bodyBytes: new Uint8Array([1, 2]) })).toBe('direct');
    expect(detectSignMode({ auth_info_bytes: 'Zm9v' })).toBe('direct');
    expect(detectSignMode({ msgs: [] })).toBe('amino');
  });

  it('enforces https origin policy with allowlisted localhost', () => {
    expect(() => enforceOriginHealth('https://safe.app')).not.toThrow();
    expect(() => enforceOriginHealth('http://localhost:8080')).not.toThrow();
    expect(() => enforceOriginHealth('http://evil.example.com')).toThrow();
  });

  it('rate limits repetitive signing requests', () => {
    const origin = 'https://rate-test.example';
    for (let i = 0; i < 5; i++) {
      enforceRateLimit(origin);
    }
    expect(() => enforceRateLimit(origin)).toThrow();
  });

  it('normalizes base64 strings into Uint8Array', () => {
    const bytes = normalizeBytes('YQ=='); // "a"
    expect(bytes).toBeInstanceOf(Uint8Array);
    expect(bytes[0]).toBe(97);
  });
});
