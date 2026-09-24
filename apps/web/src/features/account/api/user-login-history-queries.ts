import { queryOptions } from '@tanstack/react-query';
import { getUserLoginHistories } from './user-login-history';
import type { LoginHistoryFilters } from '@/features/login-histories/api/types';

export const accountLoginHistoryKeys = {
  all: ['account', 'login-history'] as const,
  list: (filters: LoginHistoryFilters) => [...accountLoginHistoryKeys.all, filters] as const
};

export function accountLoginHistoryQueryOptions(
  filters: LoginHistoryFilters,
  requestInit?: RequestInit
) {
  return queryOptions({
    queryKey: accountLoginHistoryKeys.list(filters),
    queryFn: () => getUserLoginHistories(filters, requestInit)
  });
}
