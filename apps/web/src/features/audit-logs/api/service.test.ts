import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/lib/api-client';
import { getAuditLogs } from './service';

vi.mock('@/lib/api-client', () => ({
  apiClient: vi.fn()
}));

const mockedApiClient = vi.mocked(apiClient);

describe('getAuditLogs', () => {
  beforeEach(() => {
    mockedApiClient.mockReset();
  });

  it('reads castor paged responses from the list field', async () => {
    mockedApiClient.mockResolvedValueOnce({
      total: 1,
      page: 1,
      pageSize: 10,
      totalPages: 1,
      list: [
        {
          id: 1,
          logType: 'LOGIN',
          operator: 'admin',
          operatorId: 1,
          target: 'admin',
          details: 'Login via password',
          ipAddr: '127.0.0.1',
          success: true,
          createdAt: '2026-05-06T00:00:00Z'
        }
      ]
    });

    const result = await getAuditLogs({ page: 1, pageSize: 10 });

    expect(result.list).toHaveLength(1);
    expect(result.list[0]?.logType).toBe('LOGIN');
    expect(result.list[0]?.operator).toBe('admin');
    expect(result.total).toBe(1);
  });

  it('sends filters using backend audit log fields', async () => {
    mockedApiClient.mockResolvedValueOnce({
      total: 0,
      page: 1,
      pageSize: 10,
      totalPages: 0,
      list: []
    });

    await getAuditLogs({
      page: 1,
      pageSize: 10,
      logType: 'LOGIN',
      operator: 'admin',
      success: 'true',
      sort: JSON.stringify([{ id: 'logType', desc: false }])
    });

    expect(mockedApiClient).toHaveBeenCalledWith(
      '/v1/admin/audit-logs?page=1&pageSize=10&operator-like=admin&log_type-eq=LOGIN&success-eq=true&order=log_type+asc',
      undefined
    );
  });
});
