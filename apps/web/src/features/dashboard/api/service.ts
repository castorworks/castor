// ============================================================
// Dashboard Stats Service — Uses castor's backend aggregate dashboard API
// ============================================================

import { apiClient } from '@/lib/api-client';
import type {
  DashboardStats,
  AssetCategoryStat,
  DashboardRecentActivity,
  DashboardStatsResponse
} from './types';

async function getDashboardStatsResponse(options?: RequestInit): Promise<DashboardStatsResponse> {
  return apiClient<DashboardStatsResponse>('/v1/admin/dashboard/stats', options);
}

/** Fetch aggregated dashboard stats */
export async function getDashboardStats(options?: RequestInit): Promise<DashboardStats> {
  const stats = await getDashboardStatsResponse(options);

  return {
    totalUsers: stats.userCount ?? 0,
    totalAssets: stats.assetCount ?? 0,
    totalAssetSize: stats.assetTotalSizeFormatted ?? '0 B',
    todayLogins: stats.todayLoginCount ?? 0,
    userGrowth: 0 // Placeholder — would need a dedicated growth API
  };
}

/** Fetch asset category distribution for pie chart */
export async function getAssetCategoryStats(options?: RequestInit): Promise<AssetCategoryStat[]> {
  const stats = await getDashboardStatsResponse(options);

  const categoryStats = stats.assetCategoryStats ?? {};
  return Object.entries(categoryStats)
    .map(([category, count]) => ({ category, count }))
    .filter((s) => s.count > 0);
}

/** Fetch recent audit activities for the activity feed */
export async function getRecentActivities(
  limit: number = 5,
  options?: RequestInit
): Promise<DashboardRecentActivity[]> {
  const stats = await getDashboardStatsResponse(options);

  return (stats.recentAuditLogs ?? []).slice(0, limit).map((log) => ({
    id: log.id,
    username: log.operator,
    action: log.details || log.logType,
    module: log.logType,
    createdAt: log.createdAt
  }));
}
