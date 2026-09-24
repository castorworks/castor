import { describe, expect, it, vi } from 'vitest';
import { buildUserListQuery, exportUsers, importUsers } from './service';

vi.mock('@/lib/api-client', () => ({
  apiClient: vi.fn(),
  apiUpload: vi.fn(),
  apiDownload: vi.fn()
}));

vi.mock('../lib/rsa', () => ({
  encryptPassword: vi.fn(async (plain: string) => `encrypted:${plain}`)
}));

describe('buildUserListQuery', () => {
  it('uses castor GenericGets query parameter names for search and order', () => {
    const query = buildUserListQuery({
      page: 2,
      pageSize: 20,
      search: 'alice',
      sort: JSON.stringify([{ id: 'createdAt', desc: true }])
    });

    expect(query.toString()).toBe(
      'page=2&pageSize=20&searchText=alice&searchFields=username%2Cname&order=created_at+desc'
    );
  });
});

describe('department filter', () => {
  it('matches the department and its sub-departments', () => {
    const query = buildUserListQuery({ department: '2' }, [2, 4, 5]);
    expect(query.get('department_id-in')).toBe('2,4,5');
  });

  it('falls back to the department itself when the tree is unavailable', () => {
    expect(buildUserListQuery({ department: '2' }).get('department_id-in')).toBe('2');
    expect(buildUserListQuery({ department: '2' }, []).get('department_id-in')).toBe('2');
  });
});

describe('user import and export requests', () => {
  it('sends the file and the RSA-encrypted shared password as multipart', async () => {
    const { apiUpload } = await import('@/lib/api-client');
    const upload = vi.mocked(apiUpload);
    upload.mockResolvedValueOnce({ created: 1, errors: null });

    const file = new File(['username\nalice\n'], 'users.csv', { type: 'text/csv' });
    await importUsers(file, 'Initial-Passw0rd');

    const [endpoint, form] = upload.mock.calls[0];
    expect(endpoint).toBe('/v1/admin/users/import');
    expect((form as FormData).get('file')).toBe(file);
    expect((form as FormData).get('password')).toBe('encrypted:Initial-Passw0rd');
  });

  it('exports with the list filters but without paging', async () => {
    const { apiDownload } = await import('@/lib/api-client');
    const download = vi.mocked(apiDownload);
    download.mockResolvedValueOnce({ blob: new Blob(), filename: 'users.xlsx' });

    await exportUsers({ page: 2, pageSize: 20, search: 'ali', accountSource: 'INTERNAL' }, 'xlsx');

    const endpoint = String(download.mock.calls[0][0]);
    const params = new URL(endpoint, 'http://x').searchParams;
    expect(endpoint.startsWith('/v1/admin/users/export?')).toBe(true);
    expect(params.get('searchText')).toBe('ali');
    expect(params.get('account_source-in')).toBe('INTERNAL');
    expect(params.get('format')).toBe('xlsx');
    expect(params.has('page')).toBe(false);
  });
});
