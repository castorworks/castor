import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/lib/api-client';
import { getResources } from './service';

vi.mock('@/lib/api-client', () => ({
  apiClient: vi.fn()
}));

const mockedApiClient = vi.mocked(apiClient);

describe('getResources', () => {
  beforeEach(() => {
    mockedApiClient.mockReset();
  });

  it('reads the backend resource list from the items field', async () => {
    mockedApiClient.mockResolvedValueOnce({
      total: 1,
      page: 1,
      pageSize: 20,
      items: [
        {
          id: 1,
          code: 'admin:users',
          name: 'Users',
          description: '',
          path: '/api/v1/admin/users',
          actions: ['GET'],
          category: 'admin',
          module: 'users',
          sortOrder: 1,
          isSystem: true,
          isEnabled: true
        }
      ]
    });

    const result = await getResources({ page: 1, pageSize: 20 });

    expect(result.list).toHaveLength(1);
    expect(result.list[0]?.code).toBe('admin:users');
    expect(result.total).toBe(1);
  });
});
