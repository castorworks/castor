// ============================================================
// Dashboard Stats Service — Uses castor's backend aggregate dashboard API
// ============================================================

import { apiClient } from '@/lib/api-client';
import { formatMonthName } from '@/lib/format';
import type {
  DashboardStats,
  AssetCategoryStat,
  DashboardMonthStat,
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
    activeSessions: stats.activeSessionCount ?? 0
  };
}

/** Fetch the monthly new-user / sign-in trend, aggregated on the server (oldest month first) */
export async function getMonthlyTrend(options?: RequestInit): Promise<DashboardMonthStat[]> {
  const stats = await getDashboardStatsResponse(options);
  return stats.monthlyTrend ?? [];
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

/**
 * Short month name for a `YYYY-MM` trend key. The key already names the month in the
 * backend's time zone, so it is formatted as a mid-month UTC date in UTC: no viewer
 * time zone can push it into the neighbouring month.
 */
export function trendMonthLabel(month: string, locale: string): string {
  const match = /^(\d{4})-(\d{2})$/.exec(month);
  if (!match) return month;
  const date = new Date(Date.UTC(Number(match[1]), Number(match[2]) - 1, 15));
  return formatMonthName(date, { locale, style: 'short', timeZone: 'UTC' }) || month;
}
