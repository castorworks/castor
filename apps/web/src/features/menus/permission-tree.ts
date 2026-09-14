import type { MenuCatalog, MenuPermission } from './api/types';
import { menuTree, nodePermissions, permissionKey, type MenuNode } from './tree';

/** Keep every resource operation available, including operations outside menus. */
export function permissionTree(catalog: MenuCatalog, otherTitle: string): MenuNode[] {
  const nodes = menuTree(catalog.menus);
  const linked = new Set(nodes.flatMap((node) => nodePermissions(node).map(permissionKey)));
  const extra: MenuNode[] = catalog.resources.flatMap((resource) => {
    const permissions = resource.actions
      .map((action) => ({ resourceId: resource.id, action }))
      .filter((p) => !linked.has(permissionKey(p)));
    return permissions.length
      ? [
          {
            id: -resource.id,
            parentId: null,
            code: resource.code,
            kind: 'action' as const,
            titles: { en: resource.name, zh: resource.name, ja: resource.name, ko: resource.name },
            path: '',
            icon: 'page',
            sortOrder: resource.sortOrder ?? 0,
            isEnabled: resource.isEnabled,
            accessMode: 'permission' as const,
            permissions,
            children: []
          }
        ]
      : [];
  });
  if (extra.length)
    nodes.push({
      id: -2147483647,
      parentId: null,
      code: 'unmapped',
      kind: 'directory',
      titles: { en: otherTitle, zh: otherTitle, ja: otherTitle, ko: otherTitle },
      path: '',
      icon: 'workspace',
      sortOrder: 0,
      isEnabled: true,
      accessMode: 'authenticated',
      permissions: [],
      children: extra
    });
  return nodes;
}
export function togglePermissions(selection: Set<string>, refs: MenuPermission[]): Set<string> {
  const result = new Set(selection);
  const remove = refs.every((p) => result.has(permissionKey(p)));
  for (const p of refs) {
    if (remove) result.delete(permissionKey(p));
    else result.add(permissionKey(p));
  }
  return result;
}
export function selectionGrants(selection: Set<string>) {
  const byResource = new Map<number, string[]>();
  for (const key of selection) {
    const [id, action] = key.split(':');
    const resourceId = Number(id);
    byResource.set(resourceId, [...(byResource.get(resourceId) ?? []), action]);
  }
  return [...byResource].map(([resourceId, actions]) => ({ resourceId, actions }));
}
