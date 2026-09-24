'use client';

import { usePathname } from 'next/navigation';
import { useMemo } from 'react';
import type { NavGroup, NavItem } from '@/types';
import { useNavigationGroups } from '@/hooks/use-nav';

export type BreadcrumbItem = {
  title: string;
  /** Absent for menu groups and directories, which have no page of their own. */
  link?: string;
};

function matches(url: string, pathname: string) {
  return url !== '#' && (pathname === url || pathname.startsWith(`${url}/`));
}

/** How specific a matched menu path is: the length of its last item's URL. */
function depth(path: NavItem[]) {
  return path.at(-1)?.url.length ?? 0;
}

/** The menu path to the deepest nav item whose URL covers pathname (parents first). */
function trail(items: NavItem[], pathname: string): NavItem[] {
  let best: NavItem[] = [];
  for (const item of items) {
    const below = trail(item.items ?? [], pathname);
    const here = matches(item.url, pathname) ? [item] : [];
    const candidate = below.length > 0 ? [item, ...below] : here;
    if (candidate.length > 0 && depth(candidate) > depth(best)) best = candidate;
  }
  return best;
}

/**
 * Breadcrumbs derived from the navigation menu, so titles are the menu's localized titles
 * (never URL segments). Pages outside the menu get no breadcrumbs.
 */
export function breadcrumbTrail(groups: NavGroup[], pathname: string): BreadcrumbItem[] {
  for (const group of groups) {
    const path = trail(group.items, pathname);
    if (path.length === 0) continue;
    return [
      ...(group.label ? [{ title: group.label }] : []),
      ...path.map((item) =>
        item.url === '#' ? { title: item.title } : { title: item.title, link: item.url }
      )
    ];
  }
  return [];
}

export function useBreadcrumbs() {
  const pathname = usePathname();
  const groups = useNavigationGroups();
  return useMemo(() => breadcrumbTrail(groups, pathname), [groups, pathname]);
}
