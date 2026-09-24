import { queryOptions } from '@tanstack/react-query';
import { getMenus, getNavigation } from './service';
export const menuKeys = {
  all: ['menus'] as const,
  navigation: (sessionId?: string) => ['account', 'navigation', sessionId ?? null] as const
};
export const menuCatalogQuery = queryOptions({ queryKey: menuKeys.all, queryFn: () => getMenus() });
export const navigationQuery = (sessionId?: string) =>
  queryOptions({
    queryKey: menuKeys.navigation(sessionId),
    queryFn: getNavigation,
    staleTime: 0
  });
