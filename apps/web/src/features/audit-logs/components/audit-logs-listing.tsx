import { HydrationBoundary, dehydrate } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { searchParamsCache } from '@/lib/searchparams';
import { auditLogsQueryOptions } from '../api/queries';
import { AuditLogsTable } from './audit-logs-table';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';

export default async function AuditLogsListingPage() {
  const page = searchParamsCache.get('page');
  const pageLimit = searchParamsCache.get('perPage');
  const logType = searchParamsCache.get('logType');
  const operator = searchParamsCache.get('operator');
  const target = searchParamsCache.get('target');
  const success = searchParamsCache.get('success');
  const sort = searchParamsCache.get('sort');

  const filters = {
    page,
    pageSize: pageLimit,
    ...(logType && { logType: String(logType) }),
    ...(operator && { operator: String(operator) }),
    ...(target && { target: String(target) }),
    ...(success && { success: String(success) }),
    ...(sort && { sort: String(sort) })
  };

  const queryClient = getQueryClient();
  const headers = await getServerAuthHeaders();

  try {
    await queryClient.prefetchQuery(auditLogsQueryOptions(filters, { headers }));
  } catch {
    // Prefetch failed — render empty shell; client will retry with refreshed token
  }

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <AuditLogsTable />
    </HydrationBoundary>
  );
}
