import { queryOptions } from '@tanstack/react-query';
import { getSessions } from './service';
import type { SessionFilters } from './types';

export const sessionKeys = {
  all: ['sessions'] as const,
  list: (filters: SessionFilters) => [...sessionKeys.all, 'list', filters] as const
};

export const sessionsQueryOptions = (filters: SessionFilters, options?: RequestInit) =>
  queryOptions({
    queryKey: sessionKeys.list(filters),
    queryFn: () => getSessions(filters, options)
  });
