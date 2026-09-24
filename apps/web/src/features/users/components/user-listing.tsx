import { HydrationBoundary, dehydrate } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { searchParamsCache } from '@/lib/searchparams';
import { usersQueryOptions } from '../api/queries';
import { UsersTable } from './users-table';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';

export default async function UserListingPage() {
  const page = searchParamsCache.get('page');
  const search = searchParamsCache.get('search');
  const pageLimit = searchParamsCache.get('perPage');
  const accountSource = searchParamsCache.get('accountSource');
  const status = searchParamsCache.get('status');
  const department = searchParamsCache.get('department');
  const sort = searchParamsCache.get('sort');

  const filters = {
    page,
    pageSize: pageLimit,
    ...(search && { search }),
    ...(accountSource && { accountSource }),
    ...(status && { status }),
    ...(department && { department }),
    ...(sort && { sort })
  };

  const queryClient = getQueryClient();
  const headers = await getServerAuthHeaders();

  try {
    await queryClient.prefetchQuery(usersQueryOptions(filters, { headers }));
  } catch {
    // Prefetch failed — render empty shell; client will retry with refreshed token
  }

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <UsersTable />
    </HydrationBoundary>
  );
}
