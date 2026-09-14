import { HydrationBoundary, dehydrate } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { dictTypesQueryOptions } from '../api/queries';
import { DictionaryManager } from './dictionary-manager';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';

export default async function DictionaryListingPage() {
  const queryClient = getQueryClient();
  const headers = await getServerAuthHeaders();

  try {
    await queryClient.prefetchQuery(dictTypesQueryOptions({ headers }));
  } catch {
    // Prefetch failed — render empty shell; client will retry with refreshed token
  }

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <DictionaryManager />
    </HydrationBoundary>
  );
}
