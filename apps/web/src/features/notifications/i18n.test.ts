import fs from 'node:fs';
import path from 'node:path';
import { IntlMessageFormat } from 'intl-messageformat';
import { describe, expect, it } from 'vitest';

const rootDir = path.resolve(__dirname, '../../..');
const locales = ['zh', 'en', 'ja', 'ko'] as const;

function readMessages(locale: (typeof locales)[number]) {
  return JSON.parse(
    fs.readFileSync(path.resolve(rootDir, `messages/${locale}.json`), 'utf8')
  ) as Record<string, unknown>;
}

function getValue(messages: Record<string, unknown>, key: string): unknown {
  return key.split('.').reduce<unknown>((value, part) => {
    if (!value || typeof value !== 'object' || Array.isArray(value)) {
      return undefined;
    }

    return (value as Record<string, unknown>)[part];
  }, messages);
}

describe('notifications i18n', () => {
  it('formats the extra data placeholder as literal JSON in every locale', () => {
    for (const locale of locales) {
      const messages = readMessages(locale);
      const message = getValue(messages, 'notifications.admin.form.extraPlaceholder');

      expect(message, `${locale} extra placeholder`).toBeTypeOf('string');
      expect(new IntlMessageFormat(message as string, locale).format()).toBe('{"source":"system"}');
    }
  });
});
