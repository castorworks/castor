import { queryOptions } from '@tanstack/react-query';
import { getResources } from './service';
import type { ResourceFilters } from './types';

export const resourceKeys = {
  all: ['resources'] as const,
  list: (filters: ResourceFilters) => [...resourceKeys.all, 'list', filters] as const,
  detail: (id: number) => [...resourceKeys.all, 'detail', id] as const
};

export const resourcesQueryOptions = (filters: ResourceFilters, options?: RequestInit) =>
  queryOptions({
    queryKey: resourceKeys.list(filters),
    queryFn: () => getResources(filters, options)
  });
