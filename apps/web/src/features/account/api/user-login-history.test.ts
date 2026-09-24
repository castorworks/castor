import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/lib/api-client';
import { getUserLoginHistories } from './user-login-history';

vi.mock('@/lib/api-client', () => ({
  apiClient: vi.fn()
}));

const mockedApiClient = vi.mocked(apiClient);

describe('getUserLoginHistories', () => {
  beforeEach(() => {
    mockedApiClient.mockReset();
  });

  it('reads castor paged responses from the list field', async () => {
    mockedApiClient.mockResolvedValueOnce({
      total: 1,
      list: [
        {
          id: 1,
          username: 'admin',
          ipAddr: '127.0.0.1',
          loginMethod: 'password',
          success: true,
          createdAt: '2026-05-06T00:00:00Z'
        }
      ]
    });

    const result = await getUserLoginHistories({ page: 1, pageSize: 20 });

    expect(result.list).toHaveLength(1);
    expect(result.list[0]?.username).toBe('admin');
    expect(result.total).toBe(1);
  });
});
