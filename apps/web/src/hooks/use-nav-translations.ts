'use client';

import { useTranslations } from 'next-intl';
import type { NavGroup, NavItem } from '@/types';

/**
 * Resolves a nav item's display title using i18n.
 * Falls back to the English `title` if no `titleKey` is set.
 */
export function useNavItemTitle(item: NavItem, t: ReturnType<typeof useTranslations>): string {
  if (item.titleKey) {
    return t(item.titleKey);
  }
  return item.title;
}

/**
 * Translates a NavGroup's label. Falls back to the raw `label` string.
 */
export function useNavGroupLabel(group: NavGroup, t: ReturnType<typeof useTranslations>): string {
  if (group.labelKey) {
    return t(group.labelKey);
  }
  return group.label;
}

/**
 * Convenience hook that returns a translation function bound to the `nav` namespace,
 * plus a helper to resolve NavItem titles.
 *
 * Usage in sidebar/kbar:
 *   const { t, getTitle } = useNavTranslations();
 *   <span>{getTitle(item)}</span>
 */
export function useNavTranslations() {
  const t = useTranslations('nav');

  const getTitle = (item: NavItem): string => {
    if (item.titleKey) {
      // titleKey is a fully qualified key like 'nav.users' — use root t to resolve it
      return t(item.titleKey.replace(/^nav\./, ''));
    }
    return item.title;
  };

  const getLabel = (group: NavGroup): string => {
    if (group.labelKey) {
      return t(group.labelKey.replace(/^nav\./, ''));
    }
    return group.label;
  };

  return { t, getTitle, getLabel };
}
