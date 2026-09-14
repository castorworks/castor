// ============================================================
// Login History Service — Data Access Layer
// ============================================================
// Connected to castor REST API via apiClient.
// ============================================================

import { apiClient, type CastorListResponse } from '@/lib/api-client';
import { buildCastorOrder } from '@/lib/castor-query';
import type { LoginHistory, LoginHistoryFilters, LoginHistoriesPageResult } from './types';

const LOGIN_HISTORY_ORDER_FIELDS: Record<string, string> = {
  user: 'username',
  loginMethod: 'login_method',
  result: 'success',
  createdAt: 'created_at'
};

/** Fetch paginated login history list */
export async function getLoginHistories(
  filters: LoginHistoryFilters,
  options?: RequestInit
): Promise<LoginHistoriesPageResult> {
  const params = new URLSearchParams();
  if (filters.page) params.set('page', String(filters.page));
  if (filters.pageSize) params.set('pageSize', String(filters.pageSize));
  if (filters.username) params.set('username-like', filters.username);
  if (filters.loginMethod) {
    if (filters.loginMethod.includes(',')) {
      params.set('login_method-in', filters.loginMethod);
    } else {
      params.set('login_method-eq', filters.loginMethod);
    }
  }
  if (filters.success !== undefined) params.set('success-eq', filters.success);
  const order = buildCastorOrder(filters.sort, LOGIN_HISTORY_ORDER_FIELDS);
  if (order) params.set('order', order);

  const query = params.toString();
  const res = await apiClient<CastorListResponse<LoginHistory>>(
    `/v1/admin/login-histories${query ? `?${query}` : ''}`,
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

/** Delete a login history record */
export async function deleteLoginHistory(id: number): Promise<void> {
  await apiClient<void>(`/v1/admin/login-histories/${id}`, {
    method: 'DELETE'
  });
}

/** Batch delete login history records */
export async function batchDeleteLoginHistories(ids: number[]): Promise<void> {
  await apiClient<void>('/v1/admin/login-histories/batch/delete', {
    method: 'POST',
    body: JSON.stringify({ ids })
  });
}
