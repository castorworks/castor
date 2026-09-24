import { queryOptions } from '@tanstack/react-query';
import { getLoginHistories } from './service';
import type { LoginHistoryFilters } from './types';

export const loginHistoryKeys = {
  all: ['login-histories'] as const,
  list: (filters: LoginHistoryFilters) => [...loginHistoryKeys.all, 'list', filters] as const
};

export const loginHistoriesQueryOptions = (filters: LoginHistoryFilters, options?: RequestInit) =>
  queryOptions({
    queryKey: loginHistoryKeys.list(filters),
    queryFn: () => getLoginHistories(filters, options)
  });
