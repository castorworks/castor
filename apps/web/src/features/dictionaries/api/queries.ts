import { queryOptions } from '@tanstack/react-query';
import { getDictTypes, getDictItemsByType, getAllPublicDicts } from './service';

export const dictKeys = {
  all: ['dictionaries'] as const,
  types: ['dictionaries', 'types'] as const,
  items: (typeCode: string) => ['dictionaries', 'items', typeCode] as const,
  public: ['dictionaries', 'public'] as const
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

/** All public dicts — long staleTime since dictionary data rarely changes */
export const allPublicDictsQueryOptions = (options?: RequestInit) =>
  queryOptions({
    queryKey: dictKeys.public,
    queryFn: () => getAllPublicDicts(options),
    staleTime: 5 * 60 * 1000 // 5 minutes
  });
