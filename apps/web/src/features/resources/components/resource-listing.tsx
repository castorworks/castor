import { HydrationBoundary, dehydrate } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { searchParamsCache } from '@/lib/searchparams';
import { resourcesQueryOptions } from '../api/queries';
import { ResourcesTable } from './resources-table';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';
import { getResourceModules } from '../api/service';
import { buildResourceFilters } from '../api/filters';

export default async function ResourceListingPage() {
  // 这里构造的 filters 必须与 ResourcesTable 客户端侧完全一致，
  // 否则 queryKey 对不上，预取的数据会被直接丢弃。
  const filters = buildResourceFilters({
    page: searchParamsCache.get('page'),
    perPage: searchParamsCache.get('perPage'),
    search: searchParamsCache.get('search'),
    category: searchParamsCache.get('category'),
    module: searchParamsCache.get('module'),
    status: searchParamsCache.get('status'),
    sort: searchParamsCache.get('sort')
  });

  const queryClient = getQueryClient();
  const headers = await getServerAuthHeaders();

  // Gracefully handle unauthorized SSR requests
  try {
    const [modules] = await Promise.all([
      getResourceModules({ headers }),
      queryClient.prefetchQuery(resourcesQueryOptions(filters, { headers }))
    ]);

    return (
      <HydrationBoundary state={dehydrate(queryClient)}>
        <ResourcesTable moduleOptions={modules} />
      </HydrationBoundary>
    );
  } catch {
    // Prefetch failed — render empty shell; client will retry with refreshed token
    return (
      <HydrationBoundary state={dehydrate(queryClient)}>
        <ResourcesTable />
      </HydrationBoundary>
    );
  }
}
