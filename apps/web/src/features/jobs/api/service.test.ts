import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/lib/api-client';
import { getJobRuns, getJobs, runJob, updateJob } from './service';

vi.mock('@/lib/api-client', () => ({
  apiClient: vi.fn()
}));

const mockedApiClient = vi.mocked(apiClient);

function lastCall() {
  const call = mockedApiClient.mock.calls.at(-1);
  if (!call) throw new Error('apiClient was not called');
  return { endpoint: String(call[0]), init: call[1] ?? {} };
}

describe('jobs service', () => {
  beforeEach(() => {
    mockedApiClient.mockReset();
  });

  it('lists jobs and tolerates an empty response', async () => {
    mockedApiClient.mockResolvedValueOnce(null as never);
    expect(await getJobs()).toEqual({ timeZone: '', jobs: [] });
    expect(lastCall().endpoint).toBe('/v1/admin/jobs');
  });

  it('updates cron and enabled state', async () => {
    mockedApiClient.mockResolvedValueOnce({} as never);
    await updateJob('purgeExpiredSessions', { cron: '0 4 * * *', isEnabled: false });
    expect(lastCall()).toEqual({
      endpoint: '/v1/admin/jobs/purgeExpiredSessions',
      init: { method: 'PUT', body: JSON.stringify({ cron: '0 4 * * *', isEnabled: false }) }
    });
  });

  it('starts a manual run', async () => {
    mockedApiClient.mockResolvedValueOnce({ id: 1, status: 'RUNNING' } as never);
    const run = await runJob('sweepOrphanAttachments');
    expect(run.status).toBe('RUNNING');
    expect(lastCall()).toEqual({
      endpoint: '/v1/admin/jobs/sweepOrphanAttachments/run',
      init: { method: 'POST' }
    });
  });

  it('lists runs newest first by default', async () => {
    mockedApiClient.mockResolvedValueOnce({ total: 0, list: [] });
    await getJobRuns({ page: 1, pageSize: 10 });
    const params = new URL(lastCall().endpoint, 'http://x').searchParams;
    expect(lastCall().endpoint.startsWith('/v1/admin/job-runs?')).toBe(true);
    expect(params.get('order')).toBe('started_at desc');
  });

  it('maps run filters to the unified filter contract', async () => {
    mockedApiClient.mockResolvedValueOnce({ total: 0, list: [] });
    await getJobRuns({
      page: 2,
      pageSize: 20,
      jobKey: 'purgeExpiredSessions',
      status: 'FAILED,RUNNING',
      trigger: 'MANUAL',
      sort: JSON.stringify([{ id: 'affected', desc: true }])
    });
    const params = new URL(lastCall().endpoint, 'http://x').searchParams;
    expect(params.get('page')).toBe('2');
    expect(params.get('pageSize')).toBe('20');
    expect(params.get('job_key-eq')).toBe('purgeExpiredSessions');
    expect(params.get('status-in')).toBe('FAILED,RUNNING');
    expect(params.get('trigger-in')).toBe('MANUAL');
    expect(params.get('order')).toBe('affected desc');
  });
});
