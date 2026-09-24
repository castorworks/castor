import { mutationOptions } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { deleteLoginHistory, batchDeleteLoginHistories } from './service';
import { loginHistoryKeys } from './queries';

export const deleteLoginHistoryMutation = mutationOptions({
  mutationFn: (id: number) => deleteLoginHistory(id),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: loginHistoryKeys.all });
  }
});

export const batchDeleteLoginHistoriesMutation = mutationOptions({
  mutationFn: (ids: number[]) => batchDeleteLoginHistories(ids),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: loginHistoryKeys.all });
  }
});
