import { HydrationBoundary, dehydrate } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';
import { searchParamsCache } from '@/lib/searchparams';
import { adminNotificationsQueryOptions } from '../api/queries';
import { AdminNotificationsTable } from './admin-notifications-table';

export default async function AdminNotificationListingPage() {
  const page = searchParamsCache.get('page');
  const pageLimit = searchParamsCache.get('perPage');
  const title = searchParamsCache.get('title');
  const type = searchParamsCache.get('type');
  const level = searchParamsCache.get('level');
  const sort = searchParamsCache.get('sort');

  const filters = {
    page,
    pageSize: pageLimit,
    ...(title && { title }),
    ...(type && { type }),
    ...(level && { level }),
    ...(sort && { sort })
  };

  const queryClient = getQueryClient();
  const headers = await getServerAuthHeaders();

  try {
    await queryClient.prefetchQuery(adminNotificationsQueryOptions(filters, { headers }));
  } catch {
    // Prefetch failed — render empty shell; client will retry with refreshed token
  }

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <AdminNotificationsTable />
    </HydrationBoundary>
  );
}
