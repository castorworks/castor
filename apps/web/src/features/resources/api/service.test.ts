import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/lib/api-client';
import { getResources } from './service';

vi.mock('@/lib/api-client', () => ({
  apiClient: vi.fn()
}));

const mockedApiClient = vi.mocked(apiClient);

const pagedResource = {
  total: 1,
  page: 1,
  pageSize: 20,
  totalPages: 1,
  list: [
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
};

/** 取出 apiClient 收到的 URL 的查询参数。 */
function requestedParams(): URLSearchParams {
  const url = mockedApiClient.mock.calls[0]?.[0] as string;
  return new URLSearchParams(url.slice(url.indexOf('?') + 1));
}

describe('getResources', () => {
  beforeEach(() => {
    mockedApiClient.mockReset();
  });

  it('reads castor paged responses from the list field', async () => {
    mockedApiClient.mockResolvedValueOnce(pagedResource);

    const result = await getResources({ page: 1, pageSize: 20 });

    expect(result.list).toHaveLength(1);
    expect(result.list[0]?.code).toBe('admin:users');
    expect(result.total).toBe(1);
    expect(result.totalPages).toBe(1);
  });

  // 资源列表与其它列表一样走 AGENTS.md 第 7 节的 `{field}-{op}=` 契约，不另设专用参数。
  it('sends filters using the shared {field}-{op} contract', async () => {
    mockedApiClient.mockResolvedValueOnce({ ...pagedResource, list: [] });

    await getResources({
      page: 2,
      pageSize: 20,
      category: 'admin',
      module: 'users',
      isEnabled: false,
      search: 'user'
    });

    const params = requestedParams();
    expect(params.get('page')).toBe('2');
    expect(params.get('category-eq')).toBe('admin');
    expect(params.get('module-eq')).toBe('users');
    expect(params.get('is_enabled-eq')).toBe('false');
    expect(params.get('searchText')).toBe('user');
    expect(params.get('searchFields')).toBe('name,code,path');
  });

  it('translates table sorting into a whitelisted order expression', async () => {
    mockedApiClient.mockResolvedValueOnce({ ...pagedResource, list: [] });

    await getResources({
      page: 1,
      pageSize: 20,
      sort: JSON.stringify([{ id: 'module', desc: true }])
    });

    expect(requestedParams().get('order')).toBe('module desc');
  });

  it('falls back to the curated catalog order', async () => {
    mockedApiClient.mockResolvedValueOnce({ ...pagedResource, list: [] });

    await getResources({ page: 1, pageSize: 20 });

    expect(requestedParams().get('order')).toBe('sort_order asc,id asc');
  });
});
