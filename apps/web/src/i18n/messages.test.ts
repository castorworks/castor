import fs from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';
import { locales } from './config';

const messagesDir = path.resolve(__dirname, '../../messages');

function flattenKeys(node: unknown, prefix = ''): string[] {
  if (!node || typeof node !== 'object' || Array.isArray(node)) {
    return [prefix];
  }
  return Object.entries(node as Record<string, unknown>).flatMap(([key, value]) =>
    flattenKeys(value, prefix ? `${prefix}.${key}` : key)
  );
}

function readKeys(locale: string): Set<string> {
  const file = path.join(messagesDir, `${locale}.json`);
  return new Set(flattenKeys(JSON.parse(fs.readFileSync(file, 'utf8'))));
}

describe('messages locale parity', () => {
  const [baseLocale, ...otherLocales] = locales;
  const baseKeys = readKeys(baseLocale);

  it('has translation keys', () => {
    expect(baseKeys.size).toBeGreaterThan(0);
  });

  it.each(otherLocales)('%s.json has exactly the same keys as the base locale', (locale) => {
    const keys = readKeys(locale);
    const missing = [...baseKeys].filter((key) => !keys.has(key));
    const extra = [...keys].filter((key) => !baseKeys.has(key));
    expect({ missing, extra }).toEqual({ missing: [], extra: [] });
  });
});
