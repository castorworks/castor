// ============================================================
// Scheduled Job Service — Data Access Layer
// ============================================================

import { apiClient, type CastorListResponse } from '@/lib/api-client';
import { buildCastorOrder } from '@/lib/castor-query';
import type {
  JobList,
  Job,
  JobRun,
  JobRunFilters,
  JobRunsPageResult,
  JobUpdatePayload
} from './types';

const RUN_ORDER_FIELDS: Record<string, string> = {
  startedAt: 'started_at',
  finishedAt: 'finished_at',
  affected: 'affected'
};

/** Jobs registered in code, with their settings, next run and latest run */
export async function getJobs(options?: RequestInit): Promise<JobList> {
  const res = await apiClient<JobList | null>('/v1/admin/jobs', options);
  return { timeZone: res?.timeZone ?? '', jobs: res?.jobs ?? [] };
}

export function updateJob(key: string, data: JobUpdatePayload) {
  return apiClient<Job>(`/v1/admin/jobs/${encodeURIComponent(key)}`, {
    method: 'PUT',
    body: JSON.stringify(data)
  });
}

/** Start a run in the background; resolves with the new (RUNNING) run */
export function runJob(key: string) {
  return apiClient<JobRun>(`/v1/admin/jobs/${encodeURIComponent(key)}/run`, { method: 'POST' });
}

/** Paginated run history, newest first by default */
export async function getJobRuns(
  filters: JobRunFilters,
  options?: RequestInit
): Promise<JobRunsPageResult> {
  const params = new URLSearchParams();
  if (filters.page) params.set('page', String(filters.page));
  if (filters.pageSize) params.set('pageSize', String(filters.pageSize));
  if (filters.jobKey) params.set('job_key-eq', filters.jobKey);
  if (filters.status) params.set('status-in', filters.status);
  if (filters.trigger) params.set('trigger-in', filters.trigger);
  params.set('order', buildCastorOrder(filters.sort, RUN_ORDER_FIELDS) || 'started_at desc');

  const res = await apiClient<CastorListResponse<JobRun>>(`/v1/admin/job-runs?${params}`, options);
  return {
    list: res.list ?? [],
    total: res.total ?? 0,
    page: res.page ?? filters.page ?? 1,
    pageSize: res.pageSize ?? filters.pageSize ?? 10,
    totalPages:
      res.totalPages ?? Math.ceil((res.total ?? 0) / (res.pageSize ?? filters.pageSize ?? 10))
  };
}
