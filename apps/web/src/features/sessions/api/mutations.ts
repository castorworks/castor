import { mutationOptions } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { dashboardKeys } from '@/features/dashboard/api/queries';
import { revokeSession } from './service';
import { sessionKeys } from './queries';

export const revokeSessionMutation = mutationOptions({
  mutationFn: (id: string) => revokeSession(id),
  onSuccess: async () => {
    const queryClient = getQueryClient();
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: sessionKeys.all }),
      queryClient.invalidateQueries({ queryKey: dashboardKeys.all })
    ]);
  }
});
