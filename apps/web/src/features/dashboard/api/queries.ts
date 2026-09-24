import { queryOptions } from '@tanstack/react-query';
import { getDashboardStats, getAssetCategoryStats, getRecentActivities } from './service';

export const dashboardKeys = {
  all: ['dashboard'] as const,
  stats: () => [...dashboardKeys.all, 'stats'] as const,
  assetCategories: () => [...dashboardKeys.all, 'asset-categories'] as const,
  recentActivities: (limit: number) => [...dashboardKeys.all, 'recent-activities', limit] as const
};

export function dashboardStatsQueryOptions(requestInit?: RequestInit) {
  return queryOptions({
    queryKey: dashboardKeys.stats(),
    queryFn: () => getDashboardStats(requestInit)
  });
}

export function assetCategoryStatsQueryOptions(requestInit?: RequestInit) {
  return queryOptions({
    queryKey: dashboardKeys.assetCategories(),
    queryFn: () => getAssetCategoryStats(requestInit)
  });
}

export function recentActivitiesQueryOptions(limit: number, requestInit?: RequestInit) {
  return queryOptions({
    queryKey: dashboardKeys.recentActivities(limit),
    queryFn: () => getRecentActivities(limit, requestInit)
  });
}
