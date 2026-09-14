import { HydrationBoundary, dehydrate } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { searchParamsCache } from '@/lib/searchparams';
import { resourcesQueryOptions } from '../api/queries';
import { ResourcesTable } from './resources-table';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';
import { getResourceModules } from '../api/service';

export default async function ResourceListingPage() {
  const page = searchParamsCache.get('page');
  const pageLimit = searchParamsCache.get('perPage');
  const category = searchParamsCache.get('category');
  const moduleFilter = searchParamsCache.get('module');

  const filters = {
    page,
    pageSize: pageLimit,
    ...(category && { category }),
    ...(moduleFilter && { module: moduleFilter as string })
  };

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
