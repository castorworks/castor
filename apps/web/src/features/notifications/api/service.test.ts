import { beforeEach, describe, expect, it, vi } from 'vitest';
import { getAdminNotifications, getUserNotifications } from './service';
import { apiClient } from '@/lib/api-client';

vi.mock('@/lib/api-client', () => ({
  apiClient: vi.fn()
}));

const mockedApiClient = vi.mocked(apiClient);

describe('notification service', () => {
  beforeEach(() => {
    mockedApiClient.mockReset();
  });

  it('reads castor list responses from the list field for user notifications', async () => {
    mockedApiClient.mockResolvedValueOnce({
      total: 1,
      list: [
        {
          id: 1,
          title: 'System maintenance',
          content: 'Tonight at 10 PM',
          type: 'SYSTEM',
          level: 'INFO',
          link: null,
          extra: null,
          isGlobal: false,
          expireAt: null,
          isRead: false,
          readAt: null,
          createdAt: '2026-05-06T00:00:00Z',
          updatedAt: '2026-05-06T00:00:00Z'
        }
      ]
    });

    const result = await getUserNotifications({ page: 1, pageSize: 10 });

    expect(result.list).toHaveLength(1);
    expect(result.list[0]?.title).toBe('System maintenance');
    expect(result.total).toBe(1);
    expect(result.page).toBe(1);
    expect(result.pageSize).toBe(10);
    expect(result.totalPages).toBe(1);
  });

  it('reads castor paged list responses from the list field for admin notifications', async () => {
    mockedApiClient.mockResolvedValueOnce({
      total: 1,
      page: 2,
      pageSize: 20,
      totalPages: 3,
      list: [
        {
          id: 2,
          title: 'Admin alert',
          content: 'Review required',
          type: 'ALERT',
          level: 'WARNING',
          link: null,
          extra: null,
          isGlobal: true,
          expireAt: null,
          isRead: false,
          readAt: null,
          createdAt: '2026-05-06T00:00:00Z',
          updatedAt: '2026-05-06T00:00:00Z'
        }
      ]
    });

    const result = await getAdminNotifications({ page: 2, pageSize: 20 });

    expect(result.list).toHaveLength(1);
    expect(result.page).toBe(2);
    expect(result.pageSize).toBe(20);
    expect(result.totalPages).toBe(3);
  });
});
