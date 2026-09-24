import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient, apiUpload } from '@/lib/api-client';
import { getAsset, getAssets, uploadAttachment } from './service';

vi.mock('@/lib/api-client', () => ({
  apiClient: vi.fn(),
  apiUpload: vi.fn()
}));

const mockedApiClient = vi.mocked(apiClient);

/** 取出 apiClient 收到的 URL 的查询参数。 */
function requestedParams(): URLSearchParams {
  const url = mockedApiClient.mock.calls[0]?.[0] as string;
  return new URLSearchParams(url.slice(url.indexOf('?') + 1));
}

describe('getAssets', () => {
  beforeEach(() => {
    mockedApiClient.mockReset();
    mockedApiClient.mockResolvedValue({ total: 0, list: [], page: 1, pageSize: 10, totalPages: 0 });
  });

  it('maps filters onto the unified {field}-{op} contract', async () => {
    await getAssets({
      page: 2,
      pageSize: 20,
      search: 'logo',
      category: 'IMAGE',
      status: 'ACTIVE,ARCHIVED',
      scope: 'LIBRARY'
    });

    const params = requestedParams();
    expect(params.get('page')).toBe('2');
    expect(params.get('pageSize')).toBe('20');
    expect(params.get('searchText')).toBe('logo');
    expect(params.get('category-eq')).toBe('IMAGE');
    expect(params.get('status-in')).toBe('ACTIVE,ARCHIVED');
    expect(params.get('scope-eq')).toBe('LIBRARY');
  });

  it('sends no scope filter unless asked, so callers choose what they list', async () => {
    await getAssets({ page: 1 });
    expect(requestedParams().has('scope-eq')).toBe(false);
  });

  it('translates table sorting into whitelisted order columns', async () => {
    await getAssets({ sort: JSON.stringify([{ id: 'createdAt', desc: true }]) });
    expect(requestedParams().get('order')).toBe('created_at desc');
  });
});

describe('getAsset', () => {
  beforeEach(() => {
    mockedApiClient.mockReset();
  });

  it('always exposes references as an array', async () => {
    mockedApiClient.mockResolvedValueOnce({ id: 7, name: 'logo', references: null });
    const detail = await getAsset(7);
    expect(mockedApiClient.mock.calls[0]?.[0]).toBe('/v1/admin/assets/7');
    expect(detail.references).toEqual([]);
  });
});

describe('uploadAttachment', () => {
  it("posts to the owning module's route and names the file as the uploader did", async () => {
    // 去重命中时后端返回的是已有资产，它的名字属于最早的上传者。
    vi.mocked(apiUpload).mockResolvedValueOnce({
      objectKey: 'abc.pdf',
      name: "someone-else's.pdf"
    });

    const file = new File(['x'], 'my-report.pdf');
    const attachment = await uploadAttachment('/v1/admin/notifications/attachments', file);

    const [path, formData] = vi.mocked(apiUpload).mock.calls[0] ?? [];
    expect(path).toBe('/v1/admin/notifications/attachments');
    expect((formData as FormData).get('file')).toBe(file);
    expect(attachment).toMatchObject({ objectKey: 'abc.pdf', name: 'my-report.pdf' });
  });
});
