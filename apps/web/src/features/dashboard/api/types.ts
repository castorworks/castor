// ============================================================
// Dashboard Stats types — aligned with the backend aggregate dashboard API
// ============================================================

export interface DashboardStats {
  totalUsers: number;
  totalAssets: number;
  totalAssetSize: string;
  todayLogins: number;
  userGrowth: number; // percentage
}

export interface DashboardStatsResponse {
  userCount?: number;
  todayLoginCount?: number;
  recentAuditLogs?: DashboardAuditLog[];
  assetCount?: number;
  assetTotalSize?: number;
  assetTotalSizeFormatted?: string;
  assetCategoryStats?: Record<string, number>;
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
