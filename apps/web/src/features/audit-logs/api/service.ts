// ============================================================
// Audit Log Service — Data Access Layer
// ============================================================
// Connected to castor REST API via apiClient.
// ============================================================

import { apiClient, type CastorListResponse } from '@/lib/api-client';
import { buildCastorOrder } from '@/lib/castor-query';
import type {
  AuditLog,
  AuditLogCleanupResult,
  AuditLogFilters,
  AuditLogsPageResult
} from './types';

const AUDIT_LOG_ORDER_FIELDS: Record<string, string> = {
  logType: 'log_type',
  operator: 'operator',
  target: 'target',
  result: 'success',
  ipAddr: 'ip_addr',
  createdAt: 'created_at'
};

/** Fetch paginated audit log list */
export async function getAuditLogs(
  filters: AuditLogFilters,
  options?: RequestInit
): Promise<AuditLogsPageResult> {
  const params = new URLSearchParams();
  if (filters.page) params.set('page', String(filters.page));
  if (filters.pageSize) params.set('pageSize', String(filters.pageSize));
  if (filters.operator) params.set('operator-like', filters.operator);
  if (filters.operatorId) params.set('operator_id-eq', filters.operatorId);
  if (filters.logType) {
    if (filters.logType.includes(',')) {
      params.set('log_type-in', filters.logType);
    } else {
      params.set('log_type-eq', filters.logType);
    }
  }
  if (filters.target) params.set('target-like', filters.target);
  if (filters.success !== undefined) params.set('success-eq', filters.success);
  const order = buildCastorOrder(filters.sort, AUDIT_LOG_ORDER_FIELDS);
  if (order) params.set('order', order);

  const query = params.toString();
  const res = await apiClient<CastorListResponse<AuditLog>>(
    `/v1/admin/audit-logs${query ? `?${query}` : ''}`,
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

/** Cleanup audit logs older than the specified retention days */
export async function cleanupAuditLogs(retentionDays: number): Promise<AuditLogCleanupResult> {
  return apiClient<AuditLogCleanupResult>('/v1/admin/audit-logs/cleanup', {
    method: 'POST',
    body: JSON.stringify({ retentionDays })
  });
}
