import { HydrationBoundary, dehydrate } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { searchParamsCache } from '@/lib/searchparams';
import { assetsQueryOptions, assetStatsQueryOptions } from '../api/queries';
import { AssetsTable } from './assets-table';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';

export default async function AssetListingPage() {
  const page = searchParamsCache.get('page');
  const search = searchParamsCache.get('asset');
  const pageLimit = searchParamsCache.get('perPage');
  const category = searchParamsCache.get('category');
  const status = searchParamsCache.get('status');
  const sort = searchParamsCache.get('sort');

  const filters = {
    page,
    pageSize: pageLimit,
    ...(search && { search }),
    ...(category && { category }),
    ...(status && { status }),
    ...(sort && { sort })
  };

  const queryClient = getQueryClient();
  const headers = await getServerAuthHeaders();

  try {
    await Promise.all([
      queryClient.prefetchQuery(assetsQueryOptions(filters, { headers })),
      queryClient.prefetchQuery(assetStatsQueryOptions({ headers }))
    ]);
  } catch {
    // Prefetch failed — render empty shell; client will retry with refreshed token
  }

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <AssetsTable />
    </HydrationBoundary>
  );
}
