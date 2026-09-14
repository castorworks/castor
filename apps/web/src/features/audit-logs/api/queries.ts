import { queryOptions } from '@tanstack/react-query';
import { getAuditLogs } from './service';
import type { AuditLogFilters } from './types';

export const auditLogKeys = {
  all: ['audit-logs'] as const,
  list: (filters: AuditLogFilters) => [...auditLogKeys.all, 'list', filters] as const
};

export const auditLogsQueryOptions = (filters: AuditLogFilters, options?: RequestInit) =>
  queryOptions({
    queryKey: auditLogKeys.list(filters),
    queryFn: () => getAuditLogs(filters, options)
  });
