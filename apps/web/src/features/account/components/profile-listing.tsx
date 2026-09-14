import { HydrationBoundary, dehydrate } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { accountInfoQueryOptions, userRolesQueryOptions } from '../api/queries';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';
import ProfilePage from './profile-page';

export default async function ProfileListingPage() {
  const queryClient = getQueryClient();
  const headers = await getServerAuthHeaders();

  await queryClient.prefetchQuery(accountInfoQueryOptions({ headers }));
  await queryClient.prefetchQuery(userRolesQueryOptions({ headers }));

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <ProfilePage />
    </HydrationBoundary>
  );
}
