import { Icons } from '@/components/icons';
import type { NavGroup, NavItem } from '@/types';
import type { Menu } from '@/features/menus/api/types';
import { menuTree, menuTitle, type MenuNode } from '@/features/menus/tree';

/** The API supplies the session's visible menu tree. No static menu fallback. */
export function buildNavigation(items: Menu[], locale: string): NavGroup[] {
  const toItem = (node: MenuNode): NavItem => ({
    title: menuTitle(node, locale),
    url: node.path || '#',
    icon: Object.hasOwn(Icons, node.icon) ? (node.icon as keyof typeof Icons) : 'page',
    items: node.children.filter((n) => n.kind !== 'action').map(toItem)
  });
  return menuTree(items)
    .filter((n) => n.kind !== 'action')
    .map((node) =>
      node.kind === 'directory'
        ? { label: menuTitle(node, locale), items: node.children.map(toItem) }
        : { label: '', items: [toItem(node)] }
    );
}
