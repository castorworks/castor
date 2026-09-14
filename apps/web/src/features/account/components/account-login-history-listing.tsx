import { HydrationBoundary, dehydrate } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { accountLoginHistoryQueryOptions } from '../api/user-login-history-queries';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';
import AccountLoginHistoryPage from './account-login-history-page';

export default async function AccountLoginHistoryListingPage() {
  const queryClient = getQueryClient();
  const headers = await getServerAuthHeaders();

  await queryClient.prefetchQuery(
    accountLoginHistoryQueryOptions({ page: 1, pageSize: 10 }, { headers })
  );

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <AccountLoginHistoryPage />
    </HydrationBoundary>
  );
}
