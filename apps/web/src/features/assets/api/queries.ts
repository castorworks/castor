import { queryOptions } from '@tanstack/react-query';
import { getAsset, getAssets, getAssetStats } from './service';
import type { AssetFilters } from './types';

export const assetKeys = {
  all: ['assets'] as const,
  list: (filters: AssetFilters) => [...assetKeys.all, 'list', filters] as const,
  stats: ['assets', 'stats'] as const,
  detail: (id: number) => [...assetKeys.all, 'detail', id] as const
};

export const assetsQueryOptions = (filters: AssetFilters, options?: RequestInit) =>
  queryOptions({
    queryKey: assetKeys.list(filters),
    queryFn: () => getAssets(filters, options)
  });

export const assetStatsQueryOptions = (options?: RequestInit) =>
  queryOptions({
    queryKey: assetKeys.stats,
    queryFn: () => getAssetStats(options)
  });

export const assetDetailQueryOptions = (id: number) =>
  queryOptions({
    queryKey: assetKeys.detail(id),
    queryFn: () => getAsset(id)
  });
