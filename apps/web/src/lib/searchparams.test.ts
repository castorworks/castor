import fs from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';
import { searchParamGroups, searchParams } from './searchparams';

const srcDir = path.resolve(__dirname, '..');
const selfPath = path.join(srcDir, 'lib', 'searchparams.ts');

/** 递归收集 src 下的 ts/tsx 源码（跳过登记表自身与测试文件）。 */
function sourceFiles(dir: string): string[] {
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) return sourceFiles(full);
    if (!/\.tsx?$/.test(entry.name) || /\.test\.tsx?$/.test(entry.name)) return [];
    if (full === selfPath) return [];
    return [full];
  });
}

const sources = sourceFiles(srcDir).map((file) => fs.readFileSync(file, 'utf8'));

/**
 * 参数是否有真实使用方：服务端组件用 `searchParamsCache.get('x')`，
 * 客户端表格在 `useQueryStates` 里用 `x: parseAsXxx`。
 */
function isUsed(key: string): boolean {
  const ssr = `searchParamsCache.get('${key}')`;
  const client = new RegExp(`(^|[\\s{,])${key}:\\s*(parseAs|getSortingStateParser)`, 'm');
  return sources.some((text) => text.includes(ssr) || client.test(text));
}

describe('searchParams registry', () => {
  // 扁平表是各分组展开合并而来，重名会静默覆盖：后一个分组的 parser
  // 会取代前一个，而两个模块都以为自己那份生效。
  it('declares every key in exactly one group', () => {
    const owners = new Map<string, string[]>();
    for (const [group, params] of Object.entries(searchParamGroups)) {
      for (const key of Object.keys(params)) {
        owners.set(key, [...(owners.get(key) ?? []), group]);
      }
    }
    const duplicated = [...owners.entries()]
      .filter(([, groups]) => groups.length > 1)
      .map(([key, groups]) => `${key} → ${groups.join(', ')}`);

    expect(duplicated, '同名参数请合并到 sharedParams 并注明共用方').toEqual([]);
  });

  it('flattens to exactly the union of all groups', () => {
    const fromGroups = Object.values(searchParamGroups).flatMap((params) => Object.keys(params));
    expect(Object.keys(searchParams).sort()).toEqual([...new Set(fromGroups)].sort());
  });

  // 这条护栏针对的是实际发生过的腐化：登记表里长期留着
  // `name: parseAsString, // alias for search (used by products)`
  // 这类模板残留，以及 gender/role/requestMethod 等没有任何使用方的参数。
  it('has a real consumer for every declared key', () => {
    const orphans = Object.keys(searchParams).filter((key) => !isUsed(key));

    expect(orphans, '以下参数没有任何使用方，删除它们或接上对应的列表页').toEqual([]);
  });
});
