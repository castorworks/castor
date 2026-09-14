// ============================================================
// Login History types — aligned with castor backend DTOs
// ============================================================

export interface LoginHistory {
  id: number;
  userId: number;
  username: string;
  ipAddr: string;
  userAgent: string;
  loginMethod: string;
  success: boolean;
  createdAt: string;
}

export interface LoginHistoryFilters {
  page?: number;
  pageSize?: number;
  username?: string;
  loginMethod?: string;
  success?: string;
  sort?: string;
}

export interface LoginHistoriesPageResult {
  list: LoginHistory[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}
