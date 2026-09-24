import { queryOptions } from '@tanstack/react-query';
import { getJobRuns, getJobs } from './service';
import type { JobRunFilters } from './types';

/** Poll while something is running so the page shows the result without a reload. */
const RUNNING_POLL_MS = 3000;

export const jobKeys = {
  all: ['jobs'] as const,
  list: () => [...jobKeys.all, 'list'] as const,
  runs: (filters: JobRunFilters) => [...jobKeys.all, 'runs', filters] as const
};

export const jobsQueryOptions = (options?: RequestInit) =>
  queryOptions({
    queryKey: jobKeys.list(),
    queryFn: () => getJobs(options),
    refetchInterval: (query) =>
      query.state.data?.jobs.some((job) => job.lastRun?.status === 'RUNNING')
        ? RUNNING_POLL_MS
        : false
  });

export const jobRunsQueryOptions = (filters: JobRunFilters, options?: RequestInit) =>
  queryOptions({
    queryKey: jobKeys.runs(filters),
    queryFn: () => getJobRuns(filters, options),
    refetchInterval: (query) =>
      query.state.data?.list.some((run) => run.status === 'RUNNING') ? RUNNING_POLL_MS : false
  });
