import { describe, it, expect } from 'vitest';
import { buildNavigation } from '@/config/nav-config';
import { menuTree, nodePermissions, permissionKey } from './tree';
import { permissionTree, togglePermissions, selectionGrants } from './permission-tree';
import type { Menu, MenuCatalog } from './api/types';
import { menuSchema } from './schemas/menu';
const item = (id: number, parentId: number | null, kind: Menu['kind'], sortOrder = 0): Menu => ({
  id,
  parentId,
  kind,
  sortOrder,
  code: String(id),
  titles: { en: 'Reports', zh: '报表', ja: 'レポート', ko: '보고서' },
  path: kind === 'page' ? `/dashboard/page-${id}` : '',
  icon: 'page',
  isEnabled: true,
  accessMode: 'permission',
  permissions: kind === 'directory' ? [] : [{ resourceId: 1, action: 'GET' }]
});
describe('menu navigation and authorization', () => {
  it('builds sorted nested navigation with localized titles', () => {
    const groups = buildNavigation(
      [
        item(1, null, 'directory'),
        item(2, 1, 'directory'),
        item(3, 2, 'page'),
        item(4, 1, 'page', -1)
      ],
      'zh'
    );
    expect(groups[0].label).toBe('报表');
    expect(groups[0].items[0].url).toBe('/dashboard/page-4');
    expect(groups[0].items[1].items?.[0].url).toBe('/dashboard/page-3');
  });
  it('deduplicates shared operations when toggling a subtree', () => {
    const tree = menuTree([item(1, null, 'directory'), item(2, 1, 'page'), item(3, 2, 'action')]);
    const refs = nodePermissions(tree[0]);
    expect(refs).toHaveLength(1);
    const selected = togglePermissions(new Set(['9:POST']), refs);
    expect(selectionGrants(selected)).toEqual([
      { resourceId: 9, actions: ['POST'] },
      { resourceId: 1, actions: ['GET'] }
    ]);
    expect([...togglePermissions(selected, refs)]).toEqual(['9:POST']);
  });
  it('keeps unlinked and disabled resource operations in the authorization tree', () => {
    const catalog: MenuCatalog = {
      menus: [item(1, null, 'page')],
      resources: [
        {
          id: 1,
          code: 'resource',
          name: 'Resource',
          description: '',
          path: '/objects',
          actions: ['GET', 'POST'],
          category: 'admin',
          module: 'objects',
          sortOrder: 0,
          isSystem: false,
          isEnabled: false
        }
      ]
    };
    const refs = permissionTree(catalog, 'Other').flatMap((node) =>
      nodePermissions(node).map(permissionKey)
    );
    expect(refs).toEqual(['1:GET', '1:POST']);
  });
  it('requires four titles, an internal path, and permission bindings', () => {
    const valid = {
      code: 'reports',
      kind: 'page',
      parentId: 'root',
      en: 'Reports',
      zh: '报表',
      ja: 'レポート',
      ko: '보고서',
      path: '/dashboard/reports',
      icon: 'page',
      sortOrder: 0,
      isEnabled: true,
      accessMode: 'permission',
      permissions: [{ resourceId: 1, action: 'GET' }]
    };
    const schema = menuSchema((k) => k);
    expect(schema.safeParse(valid).success).toBe(true);
    for (const change of [{ ko: '' }, { path: 'https://example.com' }, { permissions: [] }])
      expect(schema.safeParse({ ...valid, ...change }).success).toBe(false);
  });
});
