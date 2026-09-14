import { HydrationBoundary, dehydrate } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { searchParamsCache } from '@/lib/searchparams';
import { loginHistoriesQueryOptions } from '../api/queries';
import { LoginHistoriesTable } from './login-histories-table';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';

export default async function LoginHistoriesListingPage() {
  const page = searchParamsCache.get('page');
  const user = searchParamsCache.get('user');
  const pageLimit = searchParamsCache.get('perPage');
  const loginMethod = searchParamsCache.get('loginMethod');
  const result = searchParamsCache.get('result');
  const sort = searchParamsCache.get('sort');

  const filters = {
    page,
    pageSize: pageLimit,
    ...(user && { username: String(user) }),
    ...(loginMethod && { loginMethod: String(loginMethod) }),
    ...(result && { success: String(result) }),
    ...(sort && { sort: String(sort) })
  };

  const queryClient = getQueryClient();
  const headers = await getServerAuthHeaders();

  try {
    await queryClient.prefetchQuery(loginHistoriesQueryOptions(filters, { headers }));
  } catch {
    // Prefetch failed — render empty shell; client will retry with refreshed token
  }

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <LoginHistoriesTable />
    </HydrationBoundary>
  );
}
