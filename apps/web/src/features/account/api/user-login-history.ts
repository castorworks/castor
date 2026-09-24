// ============================================================
// User Login History Service — Account-level (own history)
// ============================================================

import { apiClient, type CastorListResponse } from '@/lib/api-client';
import { buildCastorOrder } from '@/lib/castor-query';
import type {
  LoginHistory,
  LoginHistoryFilters,
  LoginHistoriesPageResult
} from '@/features/login-histories/api/types';

const USER_LOGIN_HISTORY_ORDER_FIELDS: Record<string, string> = {
  loginMethod: 'login_method',
  result: 'success',
  createdAt: 'created_at'
};

export async function getUserLoginHistories(
  filters: LoginHistoryFilters,
  options?: RequestInit
): Promise<LoginHistoriesPageResult> {
  const params = new URLSearchParams();
  if (filters.page) params.set('page', String(filters.page));
  if (filters.pageSize) params.set('pageSize', String(filters.pageSize));
  if (filters.loginMethod) params.set('login_method-eq', filters.loginMethod);
  if (filters.success !== undefined) params.set('success-eq', filters.success);
  const order = buildCastorOrder(filters.sort, USER_LOGIN_HISTORY_ORDER_FIELDS);
  if (order) params.set('order', order);

  const query = params.toString();
  const res = await apiClient<CastorListResponse<LoginHistory>>(
    `/v1/account/login-histories${query ? `?${query}` : ''}`,
    options
  );
  return {
    list: res.list ?? [],
    total: res.total ?? 0,
    page: res.page ?? filters.page ?? 1,
    pageSize: res.pageSize ?? filters.pageSize ?? 10,
    totalPages:
      res.totalPages ?? Math.ceil((res.total ?? 0) / (res.pageSize ?? filters.pageSize ?? 10))
  };
}
