import fs from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';
import { Icons } from '@/components/icons';
import type { DictItem, DictMap } from '@/features/dictionaries/api/types';
import { DICT_TYPES, createDictLookup, dictOptionsOr } from './dict';
import { TAG_COLORS } from './tag-color';

const dictionaryEntity = fs.readFileSync(
  path.resolve(__dirname, '../../../api/internal/domain/dictionary/entity.go'),
  'utf8'
);

function item(partial: Partial<DictItem>): DictItem {
  return {
    id: 1,
    typeCode: 'asset_status',
    label: { en: '', zh: '', ja: '', ko: '' },
    value: '',
    description: '',
    color: '',
    icon: '',
    isSystem: true,
    isDefault: false,
    isEnabled: true,
    sortOrder: 0,
    createdAt: '',
    updatedAt: '',
    ...partial
  };
}

const dicts: DictMap = {
  asset_status: [
    item({
      value: 'ACTIVE',
      label: { en: 'Active', zh: '生效', ja: '有効', ko: '활성' },
      color: 'green',
      icon: 'check'
    }),
    item({ value: 'DRAFT', label: { en: 'Draft', zh: '', ja: '', ko: '' } })
  ]
};

describe('createDictLookup', () => {
  it('localizes labels and falls back to English, then to the raw value', () => {
    const zh = createDictLookup(dicts, 'zh');
    expect(zh.label('asset_status', 'ACTIVE')).toBe('生效');
    expect(zh.label('asset_status', 'DRAFT')).toBe('Draft');
    // 停用或历史上存在过的取值仍会出现在旧数据里，不能抛错
    expect(zh.label('asset_status', 'RETIRED')).toBe('RETIRED');
  });

  it('exposes color and icon, and treats empty strings as absent', () => {
    const dict = createDictLookup(dicts, 'en');
    expect(dict.color('asset_status', 'ACTIVE')).toBe('green');
    expect(dict.icon('asset_status', 'ACTIVE')).toBe('check');
    expect(dict.color('asset_status', 'DRAFT')).toBeUndefined();
    expect(dict.icon('asset_status', 'DRAFT')).toBeUndefined();
  });

  it('builds localized options in item order', () => {
    expect(createDictLookup(dicts, 'ko').options('asset_status')).toEqual([
      { value: 'ACTIVE', label: '활성' },
      { value: 'DRAFT', label: 'Draft' }
    ]);
  });

  it('is empty rather than throwing before the dictionaries load', () => {
    const dict = createDictLookup(undefined, 'en');
    expect(dict.items('gender')).toEqual([]);
    expect(dict.options('gender')).toEqual([]);
    expect(dict.label('gender', 'MALE')).toBe('MALE');
  });
});

describe('dictOptionsOr', () => {
  const fallback = [{ value: 'X', label: 'X' }];

  it('prefers dictionary options and falls back when the type is empty', () => {
    const dict = createDictLookup(dicts, 'en');
    expect(dictOptionsOr(dict, 'asset_status', fallback)).toHaveLength(2);
    expect(dictOptionsOr(dict, 'gender', fallback)).toBe(fallback);
  });
});

/**
 * 字典的前后端契约。任何一条漂移都不会报错，只会让界面静默显示原始枚举值、
 * 丢失颜色或回退成默认图标，所以在这里把它们变成测试期失败。
 */
describe('dictionary contract with the backend seeds', () => {
  it('DICT_TYPES mirrors DefaultDictTypes', () => {
    const seeded = [...dictionaryEntity.matchAll(/\{Code:\s*"([a-z0-9_]+)"/g)].map((m) => m[1]);
    expect(seeded.length).toBeGreaterThan(0);
    expect([...DICT_TYPES].toSorted()).toEqual(seeded.toSorted());
  });

  it('tag colors mirror dictionary.Colors', () => {
    const declaration = dictionaryEntity.match(/var Colors = \[\]string\{([^}]+)\}/);
    expect(declaration).not.toBeNull();
    const colors = [...declaration![1].matchAll(/"([a-z]+)"/g)].map((m) => m[1]);
    expect(colors.toSorted()).toEqual(TAG_COLORS.toSorted());
  });

  it('every seeded icon is a key of Icons', () => {
    const icons = new Set([...dictionaryEntity.matchAll(/Icon:\s*"([^"]+)"/g)].map((m) => m[1]));
    expect(icons.size).toBeGreaterThan(0);
    expect([...icons].filter((icon) => !Object.hasOwn(Icons, icon))).toEqual([]);
  });
});
