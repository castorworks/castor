import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/lib/api-client';
import { getEnabledDicts, updateDictItem } from './service';

vi.mock('@/lib/api-client', () => ({
  apiClient: vi.fn()
}));

const mockedApiClient = vi.mocked(apiClient);

describe('dictionaries service', () => {
  beforeEach(() => {
    mockedApiClient.mockReset();
  });

  it('reads dashboard dictionaries from the signed-in endpoint, not the public one', async () => {
    mockedApiClient.mockResolvedValueOnce({ asset_status: [] });
    const headers = { cookie: 'jwt=token' };

    await expect(getEnabledDicts({ headers })).resolves.toEqual({ asset_status: [] });

    // 免认证的 /v1/dictionaries 只返回 isPublic 的类型，拿它渲染后台会丢标签
    expect(mockedApiClient).toHaveBeenCalledWith('/v1/account/dictionaries', { headers });
  });

  it('sends an empty color so an edit can clear it', async () => {
    mockedApiClient.mockResolvedValueOnce({});

    await updateDictItem(7, { color: '', icon: '' });

    expect(mockedApiClient).toHaveBeenCalledWith('/v1/admin/dict-items/7', {
      method: 'PUT',
      body: JSON.stringify({ color: '', icon: '' })
    });
  });
});
