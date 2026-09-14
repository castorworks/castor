import fs from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';
import { resourceValidationMessages } from './schemas/resource';

const rootDir = path.resolve(__dirname, '../../..');
const apiPermissionEntityPath = path.resolve(
  rootDir,
  '../api/internal/domain/permission/entity.go'
);
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

describe('resources i18n coverage', () => {
  it('translates every backend default resource seed key', () => {
    const entitySource = fs.readFileSync(apiPermissionEntityPath, 'utf8');
    const seedKeys = [...entitySource.matchAll(/Name:\s*"(seedResources[^"]+)"/g)].map(
      (match) => match[1]
    );

    expect(seedKeys.length).toBeGreaterThan(0);

    for (const locale of locales) {
      const messages = readMessages(locale);
      const missing = seedKeys.filter((key) => typeof getValue(messages, key) !== 'string');

      expect(missing, `${locale} missing seed resource translations`).toEqual([]);
    }
  });

  it('has localized labels for every default resource module slug', () => {
    const entitySource = fs.readFileSync(apiPermissionEntityPath, 'utf8');
    const modules = new Set(
      [...entitySource.matchAll(/Module:\s*"([^"]+)"/g)].map((match) => match[1])
    );

    expect(modules.size).toBeGreaterThan(0);

    for (const locale of locales) {
      const messages = readMessages(locale);
      const missing = [...modules].filter(
        (module) => typeof getValue(messages, `resources.modules.${module}`) !== 'string'
      );

      expect(missing, `${locale} missing resource module translations`).toEqual([]);
    }
  });

  it('uses translation keys for resource form validation messages', () => {
    expect(resourceValidationMessages).toMatchObject({
      codeMin: 'resources.validation.codeMin',
      nameMin: 'resources.validation.nameMin',
      pathRequired: 'resources.validation.pathRequired',
      actionsRequired: 'resources.validation.actionsRequired',
      categoryRequired: 'resources.validation.categoryRequired'
    });
  });
});
