import { HydrationBoundary, dehydrate } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { settingsQueryOptions } from '../api/queries';
import { SettingsTable } from './settings-table';
import { getServerAuthHeaders } from '@/lib/server-auth-headers';

export default async function SettingsListingPage() {
  const queryClient = getQueryClient();
  const headers = await getServerAuthHeaders();

  try {
    await queryClient.prefetchQuery(settingsQueryOptions({}, { headers }));
  } catch {
    // Prefetch failed — render empty shell; client will retry with refreshed token
  }

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <SettingsTable />
    </HydrationBoundary>
  );
}
