// ============================================================
// Online Session Service — Data Access Layer
// ============================================================

import { apiClient, type CastorListResponse } from '@/lib/api-client';
import { buildCastorOrder } from '@/lib/castor-query';
import type { Session, SessionFilters, SessionsPageResult } from './types';

const SESSION_ORDER_FIELDS: Record<string, string> = {
  user: 'username',
  createdAt: 'created_at',
  lastActiveAt: 'last_active_at',
  expiresAt: 'expires_at'
};

/** Fetch the paginated list of live sessions (most recently active first by default) */
export async function getSessions(
  filters: SessionFilters,
  options?: RequestInit
): Promise<SessionsPageResult> {
  const params = new URLSearchParams();
  if (filters.page) params.set('page', String(filters.page));
  if (filters.pageSize) params.set('pageSize', String(filters.pageSize));
  if (filters.username) params.set('username-like', filters.username);
  if (filters.ipAddr) params.set('ip_addr-like', filters.ipAddr);
  params.set(
    'order',
    buildCastorOrder(filters.sort, SESSION_ORDER_FIELDS) || 'last_active_at desc'
  );

  const res = await apiClient<CastorListResponse<Session>>(`/v1/admin/sessions?${params}`, options);
  return {
    list: res.list ?? [],
    total: res.total ?? 0,
    page: res.page ?? filters.page ?? 1,
    pageSize: res.pageSize ?? filters.pageSize ?? 10,
    totalPages:
      res.totalPages ?? Math.ceil((res.total ?? 0) / (res.pageSize ?? filters.pageSize ?? 10))
  };
}

/** Revoke a session: the user is signed out on their next request */
export async function revokeSession(id: string): Promise<void> {
  await apiClient<void>(`/v1/admin/sessions/${encodeURIComponent(id)}`, { method: 'DELETE' });
}
