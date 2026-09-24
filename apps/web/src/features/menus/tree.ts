import type { Menu, MenuPermission } from './api/types';
export interface MenuNode extends Menu {
  children: MenuNode[];
}
export function menuTree(items: Menu[]): MenuNode[] {
  const nodes = new Map(items.map((item) => [item.id, { ...item, children: [] } as MenuNode]));
  const roots: MenuNode[] = [];
  for (const node of nodes.values()) {
    if (node.parentId === null) roots.push(node);
    else nodes.get(node.parentId)?.children.push(node);
  }
  return sort(roots);
}
export function menuTitle(menu: Pick<Menu, 'titles'>, locale: string) {
  return menu.titles[locale as keyof Menu['titles']] || menu.titles.en;
}
export function permissionKey(p: MenuPermission) {
  return `${p.resourceId}:${p.action}`;
}
export function nodePermissions(node: MenuNode): MenuPermission[] {
  const refs = new Map(node.permissions.map((p) => [permissionKey(p), p]));
  for (const child of node.children)
    for (const p of nodePermissions(child)) refs.set(permissionKey(p), p);
  return [...refs.values()];
}

const sort = (nodes: MenuNode[]): MenuNode[] =>
  nodes
    .toSorted((a, b) => a.sortOrder - b.sortOrder || a.id - b.id)
    .map((n) => ({ ...n, children: sort(n.children) }));
