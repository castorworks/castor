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

describe('translation usage', () => {
  const srcDir = path.resolve(__dirname, '..');

  function sourceFiles(dir: string): string[] {
    return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) return sourceFiles(full);
      return /\.tsx?$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name) ? [full] : [];
    });
  }

  function findOffenders(pattern: RegExp): string[] {
    return sourceFiles(srcDir).flatMap((file) => {
      const matches = fs.readFileSync(file, 'utf8').match(pattern) ?? [];
      return matches.map(
        (match) => `${path.relative(srcDir, file)}: ${match.replace(/\s+/g, ' ')}`
      );
    });
  }

  // A translator function call: t('key'), tc('key'), ta('key', { ... }).
  const T_CALL = String.raw`\bt[a-zA-Z]*\('[\w.]+'(?:,\s*\{[^{}]*\})?\)`;

  // Word order and spacing differ per language ("邮箱 编码", "手机 号码"): every
  // phrase is one message, with ICU placeholders for the variable parts.
  it('does not compose phrases from adjacent translations', () => {
    const adjacent = new RegExp(String.raw`\{${T_CALL}\}\s*(?:\{' '\}\s*)?\{${T_CALL}\}`, 'g');
    const templated = new RegExp(String.raw`\$\{${T_CALL}\}[^\`]*\$\{${T_CALL}\}`, 'g');
    expect([...findOffenders(adjacent), ...findOffenders(templated)]).toEqual([]);
  });

  // A defaultValue hides a missing key behind English copy in every locale.
  it('does not fall back to inline default copy', () => {
    expect(
      findOffenders(new RegExp(String.raw`\bt[a-zA-Z]*\('[\w.]+',\s*\{[^{}]*\bdefaultValue:`, 'g'))
    ).toEqual([]);
  });
});
