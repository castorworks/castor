import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/lib/api-client';
import { getAssetCategoryStats, getDashboardStats, getRecentActivities } from './service';

vi.mock('@/lib/api-client', () => ({
  apiClient: vi.fn()
}));

const mockedApiClient = vi.mocked(apiClient);

describe('dashboard service', () => {
  beforeEach(() => {
    mockedApiClient.mockReset();
  });

  it('reads dashboard stats from the backend aggregate endpoint', async () => {
    mockedApiClient.mockResolvedValueOnce({
      userCount: 12,
      assetCount: 7,
      assetTotalSizeFormatted: '42 MB',
      todayLoginCount: 3,
      recentAuditLogs: [],
      assetCategoryStats: {}
    });

    const result = await getDashboardStats();

    expect(mockedApiClient).toHaveBeenCalledWith('/v1/admin/dashboard/stats', undefined);
    expect(result).toEqual({
      totalUsers: 12,
      totalAssets: 7,
      totalAssetSize: '42 MB',
      todayLogins: 3,
      userGrowth: 0
    });
  });

  it('maps asset category stats from the dashboard aggregate endpoint', async () => {
    mockedApiClient.mockResolvedValueOnce({
      userCount: 0,
      assetCount: 0,
      assetTotalSizeFormatted: '0 B',
      todayLoginCount: 0,
      recentAuditLogs: [],
      assetCategoryStats: {
        IMAGE: 4,
        VIDEO: 0,
        DOCUMENT: 2
      }
    });

    await expect(getAssetCategoryStats()).resolves.toEqual([
      { category: 'IMAGE', count: 4 },
      { category: 'DOCUMENT', count: 2 }
    ]);
    expect(mockedApiClient).toHaveBeenCalledWith('/v1/admin/dashboard/stats', undefined);
  });

  it('maps recent activities from the dashboard aggregate endpoint', async () => {
    mockedApiClient.mockResolvedValueOnce({
      userCount: 0,
      assetCount: 0,
      assetTotalSizeFormatted: '0 B',
      todayLoginCount: 0,
      recentAuditLogs: [
        {
          id: 9,
          operator: 'admin',
          details: 'Created user',
          logType: 'user.create',
          createdAt: '2026-05-01T00:00:00Z'
        }
      ],
      assetCategoryStats: {}
    });

    await expect(getRecentActivities()).resolves.toEqual([
      {
        id: 9,
        username: 'admin',
        action: 'Created user',
        module: 'user.create',
        createdAt: '2026-05-01T00:00:00Z'
      }
    ]);
    expect(mockedApiClient).toHaveBeenCalledWith('/v1/admin/dashboard/stats', undefined);
  });
});
