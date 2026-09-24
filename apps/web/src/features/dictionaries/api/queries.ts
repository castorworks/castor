import { queryOptions } from '@tanstack/react-query';
import { getDictTypes, getDictItemsByType, getEnabledDicts } from './service';

export const dictKeys = {
  all: ['dictionaries'] as const,
  types: ['dictionaries', 'types'] as const,
  items: (typeCode: string) => ['dictionaries', 'items', typeCode] as const,
  enabled: ['dictionaries', 'enabled'] as const
};

export const dictTypesQueryOptions = (options?: RequestInit) =>
  queryOptions({
    queryKey: dictKeys.types,
    queryFn: () => getDictTypes(options)
  });

export const dictItemsQueryOptions = (typeCode: string, options?: RequestInit) =>
  queryOptions({
    queryKey: dictKeys.items(typeCode),
    queryFn: () => getDictItemsByType(typeCode, options),
    enabled: !!typeCode
  });

/**
 * Every enabled dictionary, used to render enum labels across the dashboard.
 * Prefetched by the dashboard layout and read through `useDict()`.
 * Long staleTime: dictionaries rarely change, and every dictionary mutation
 * invalidates `dictKeys.all`, which covers this key.
 */
export const enabledDictsQueryOptions = (options?: RequestInit) =>
  queryOptions({
    queryKey: dictKeys.enabled,
    queryFn: () => getEnabledDicts(options),
    staleTime: 5 * 60 * 1000
  });
