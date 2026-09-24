import { HydrationBoundary, dehydrate } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { rolesQueryOptions } from '../api/queries';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';
import { RoleListingContent } from './role-listing-content';

export default async function RoleListingPage() {
  const queryClient = getQueryClient();
  const headers = await getServerAuthHeaders();

  try {
    await queryClient.prefetchQuery(rolesQueryOptions({ headers }));
  } catch {
    // Prefetch failed — render empty shell; client will retry with refreshed token
  }

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <RoleListingContent />
    </HydrationBoundary>
  );
}
