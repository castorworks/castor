'use client';

import type { NavItem, NavGroup } from '@/types';
import { useAuthStore } from '@/stores/auth-store';
import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useLocale } from 'next-intl';
import { navigationQuery } from '@/features/menus/api/queries';
import { buildNavigation } from '@/config/nav-config';

export function canAccessNavItem(item: NavItem, permissions: string[]) {
  const required =
    item.access?.permissions ?? (item.access?.permission ? [item.access.permission] : []);
  if (!required.length) return true;
  return required.some((permission) => permissions.includes(permission));
}

export function filterNavItems(items: NavItem[], permissions: string[]): NavItem[] {
  return items
    .filter((item) => canAccessNavItem(item, permissions))
    .map((item) => {
      if (!item.items?.length) return item;

      return {
        ...item,
        items: filterNavItems(item.items, permissions)
      };
    });
}

export function filterNavGroups(groups: NavGroup[], permissions: string[]): NavGroup[] {
  return groups
    .map((group) => ({
      ...group,
      items: filterNavItems(group.items, permissions)
    }))
    .filter((group) => group.items.length > 0);
}

// Stable empty array reference to avoid infinite re-renders
const EMPTY_PERMISSIONS: string[] = [];

export function useFilteredNavItems(items: NavItem[]) {
  const grants = useAuthStore((s) => s.user?.permissions);
  const permissions = useMemo(
    () => grants?.map((grant) => `${grant.resourcePath}:${grant.action}`) ?? EMPTY_PERMISSIONS,
    [grants]
  );
  return useMemo(() => filterNavItems(items, permissions), [items, permissions]);
}

export function useFilteredNavGroups(groups: NavGroup[]) {
  const grants = useAuthStore((s) => s.user?.permissions);
  const permissions = useMemo(
    () => grants?.map((grant) => `${grant.resourcePath}:${grant.action}`) ?? EMPTY_PERMISSIONS,
    [grants]
  );
  return useMemo(() => filterNavGroups(groups, permissions), [groups, permissions]);
}

export function useNavigationGroups() {
  const user = useAuthStore((s) => s.user);
  const locale = useLocale();
  const query = useQuery({
    ...navigationQuery(user?.authorizationSessionId),
    enabled: !!user
  });
  return useMemo(() => buildNavigation(query.data?.items ?? [], locale), [query.data, locale]);
}
