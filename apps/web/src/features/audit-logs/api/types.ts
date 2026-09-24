// ============================================================
// Audit Log types — aligned with castor backend DTOs
// ============================================================

export interface AuditLog {
  id: number;
  logType: string;
  operator: string;
  operatorId: number;
  target: string;
  details: string;
  ipAddr: string;
  success: boolean;
  createdAt: string;
}

export interface AuditLogFilters {
  page?: number;
  pageSize?: number;
  operator?: string;
  operatorId?: string;
  logType?: string;
  target?: string;
  success?: string;
  sort?: string;
}

export interface AuditLogsPageResult {
  list: AuditLog[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface AuditLogCleanupResult {
  deleted: number;
}
