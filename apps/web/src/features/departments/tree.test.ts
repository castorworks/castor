import { describe, expect, it } from 'vitest';
import type { Department } from './api/types';
import { departmentTree, flattenDepartments, indentedLabel, subtreeIds } from './tree';

function dept(id: number, parentId: number | null, sortOrder = 0): Department {
  return {
    id,
    parentId,
    code: `d${id}`,
    name: `D${id}`,
    sortOrder,
    isEnabled: true,
    memberCount: 0,
    inScope: true,
    createdAt: '',
    updatedAt: ''
  };
}

// 1 ─ 2 ─ 4
//   └ 3
// 5（父部门缺失，按根展示）
const items = [dept(4, 2), dept(3, 1, 0), dept(2, 1, 1), dept(1, null), dept(5, 99)];

describe('department tree', () => {
  it('nests children and orders siblings by sortOrder', () => {
    const tree = departmentTree(items);
    expect(tree.map((node) => node.id)).toEqual([1, 5]);
    expect(tree[0].children.map((node) => node.id)).toEqual([3, 2]);
    expect(tree[0].children[1].children.map((node) => node.id)).toEqual([4]);
  });

  it('flattens depth-first with depths', () => {
    expect(
      flattenDepartments(items).map(({ department, depth }) => [department.id, depth])
    ).toEqual([
      [1, 0],
      [3, 1],
      [2, 1],
      [4, 2],
      [5, 0]
    ]);
  });

  it('collects a subtree', () => {
    expect(subtreeIds(items, 1).toSorted()).toEqual([1, 2, 3, 4]);
    expect(subtreeIds(items, 4)).toEqual([4]);
    expect(subtreeIds(items, 42)).toEqual([]);
  });

  it('indents labels with non-breaking spaces', () => {
    expect(indentedLabel('R&D', 2)).toBe(' '.repeat(6) + 'R&D');
  });
});
