// ============================================================
// Dashboard Stats types — aligned with the backend aggregate dashboard API
// ============================================================

export interface DashboardStats {
  totalUsers: number;
  totalAssets: number;
  totalAssetSize: string;
  todayLogins: number;
  activeSessions: number;
}

/** One calendar month (backend database time zone), oldest first. */
export interface DashboardMonthStat {
  /** `YYYY-MM` */
  month: string;
  newUsers: number;
  /** Successful sign-ins only */
  logins: number;
}

export interface DashboardStatsResponse {
  userCount: number;
  todayLoginCount: number;
  activeSessionCount: number;
  assetCount: number;
  assetTotalSize: number;
  assetTotalSizeFormatted: string;
  assetCategoryStats: Record<string, number> | null;
  monthlyTrend: DashboardMonthStat[] | null;
  recentAuditLogs: DashboardAuditLog[] | null;
}

export interface DashboardAuditLog {
  id: number;
  operator: string;
  details?: string;
  logType: string;
  createdAt: string;
}

export interface AssetCategoryStat {
  category: string;
  count: number;
}

export interface DashboardRecentActivity {
  id: number;
  username: string;
  action: string;
  module: string;
  createdAt: string;
}
