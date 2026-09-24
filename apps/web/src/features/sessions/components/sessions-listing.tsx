import { HydrationBoundary, dehydrate } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { searchParamsCache } from '@/lib/searchparams';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';
import { sessionsQueryOptions } from '../api/queries';
import { SessionsTable } from './sessions-table';

export default async function SessionsListingPage() {
  const page = searchParamsCache.get('page');
  const pageLimit = searchParamsCache.get('perPage');
  const user = searchParamsCache.get('user');
  const ip = searchParamsCache.get('ip');
  const sort = searchParamsCache.get('sort');

  const filters = {
    page,
    pageSize: pageLimit,
    ...(user && { username: String(user) }),
    ...(ip && { ipAddr: String(ip) }),
    ...(sort && { sort: String(sort) })
  };

  const queryClient = getQueryClient();
  const headers = await getServerAuthHeaders();

  try {
    await queryClient.prefetchQuery(sessionsQueryOptions(filters, { headers }));
  } catch {
    // Prefetch failed — render empty shell; client will retry with refreshed token
  }

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <SessionsTable />
    </HydrationBoundary>
  );
}
