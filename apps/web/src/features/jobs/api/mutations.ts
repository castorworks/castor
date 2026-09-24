import { mutationOptions } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { runJob, updateJob } from './service';
import { jobKeys } from './queries';
import type { JobUpdatePayload } from './types';

async function refreshJobs() {
  await getQueryClient().invalidateQueries({ queryKey: jobKeys.all });
}

export const updateJobMutation = mutationOptions({
  mutationFn: ({ key, data }: { key: string; data: JobUpdatePayload }) => updateJob(key, data),
  onSuccess: refreshJobs
});

export const runJobMutation = mutationOptions({
  mutationFn: (key: string) => runJob(key),
  onSuccess: refreshJobs
});
