import type { Department } from './api/types';

export interface DepartmentNode extends Department {
  children: DepartmentNode[];
}

/** Builds the tree from the flat list; orphans (missing parent) are shown as roots. */
export function departmentTree(items: Department[]): DepartmentNode[] {
  const nodes = new Map(
    items.map((item) => [item.id, { ...item, children: [] } as DepartmentNode])
  );
  const roots: DepartmentNode[] = [];
  for (const node of nodes.values()) {
    const parent = node.parentId === null ? undefined : nodes.get(node.parentId);
    if (parent) parent.children.push(node);
    else roots.push(node);
  }
  return sortNodes(roots);
}

const sortNodes = (nodes: DepartmentNode[]): DepartmentNode[] =>
  nodes
    .toSorted((a, b) => a.sortOrder - b.sortOrder || a.id - b.id)
    .map((node) => ({ ...node, children: sortNodes(node.children) }));

/** Depth-first order with the depth of each department, for indented selects. */
export function flattenDepartments(
  items: Department[]
): { department: Department; depth: number }[] {
  const out: { department: Department; depth: number }[] = [];
  const walk = (nodes: DepartmentNode[], depth: number) => {
    for (const node of nodes) {
      out.push({ department: node, depth });
      walk(node.children, depth + 1);
    }
  };
  walk(departmentTree(items), 0);
  return out;
}

/** The department and all its descendants. */
export function subtreeIds(items: Department[], rootId: number): number[] {
  const children = new Map<number, number[]>();
  for (const item of items) {
    if (item.parentId === null) continue;
    children.set(item.parentId, [...(children.get(item.parentId) ?? []), item.id]);
  }
  if (!items.some((item) => item.id === rootId)) return [];
  const result = [rootId];
  for (let i = 0; i < result.length; i++) {
    for (const child of children.get(result[i]) ?? []) {
      if (!result.includes(child)) result.push(child);
    }
  }
  return result;
}

/** Indents a label by depth for plain-text option lists. */
export function indentedLabel(name: string, depth: number): string {
  return `${'   '.repeat(depth)}${name}`;
}
