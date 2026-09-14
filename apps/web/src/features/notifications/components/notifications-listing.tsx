import { HydrationBoundary, dehydrate } from '@tanstack/react-query';
import { redirect } from 'next/navigation';
import { getQueryClient } from '@/lib/query-client';
import { userNotificationsQueryOptions, unreadCountQueryOptions } from '../api/queries';
import { getLoginRedirectUrl } from '@/lib/auth';
import { getServerAuthHeaders, hasServerAuthHeaders } from '@/lib/server-auth-headers';
import NotificationsPage from './notifications-page';

export default async function NotificationsListingPage() {
  const queryClient = getQueryClient();
  const headers = await getServerAuthHeaders();

  if (!hasServerAuthHeaders(headers)) {
    redirect(getLoginRedirectUrl('/dashboard/notifications'));
  }

  await Promise.all([
    queryClient.prefetchQuery(
      userNotificationsQueryOptions({ page: 1, pageSize: 10 }, { headers })
    ),
    queryClient.prefetchQuery(unreadCountQueryOptions({ headers }))
  ]);

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <NotificationsPage />
    </HydrationBoundary>
  );
}
