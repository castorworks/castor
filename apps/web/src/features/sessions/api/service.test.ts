import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/lib/api-client';
import { getSessions, revokeSession } from './service';

vi.mock('@/lib/api-client', () => ({
  apiClient: vi.fn()
}));

const mockedApiClient = vi.mocked(apiClient);

function lastEndpoint() {
  const call = mockedApiClient.mock.calls.at(-1);
  if (!call) throw new Error('apiClient was not called');
  return { endpoint: String(call[0]), init: call[1] ?? {} };
}

describe('sessions service', () => {
  beforeEach(() => {
    mockedApiClient.mockReset();
  });

  it('lists live sessions, most recently active first by default', async () => {
    mockedApiClient.mockResolvedValueOnce({
      total: 1,
      page: 1,
      pageSize: 10,
      totalPages: 1,
      list: [{ id: 'abc', username: 'admin', current: true }]
    });

    const result = await getSessions({ page: 1, pageSize: 10 });

    expect(result.total).toBe(1);
    expect(result.list[0]?.current).toBe(true);
    const params = new URL(lastEndpoint().endpoint, 'http://x').searchParams;
    expect(params.get('order')).toBe('last_active_at desc');
  });

  it('maps filters to the unified filter contract', async () => {
    mockedApiClient.mockResolvedValueOnce({ total: 0, list: [] });

    await getSessions({
      page: 2,
      pageSize: 20,
      username: 'ali',
      ipAddr: '10.0.',
      sort: JSON.stringify([{ id: 'createdAt', desc: false }])
    });

    const params = new URL(lastEndpoint().endpoint, 'http://x').searchParams;
    expect(lastEndpoint().endpoint.startsWith('/v1/admin/sessions?')).toBe(true);
    expect(params.get('page')).toBe('2');
    expect(params.get('pageSize')).toBe('20');
    expect(params.get('username-like')).toBe('ali');
    expect(params.get('ip_addr-like')).toBe('10.0.');
    expect(params.get('order')).toBe('created_at asc');
  });

  it('revokes a session by id', async () => {
    mockedApiClient.mockResolvedValueOnce(undefined as never);
    await revokeSession('00000000-0000-0000-0000-000000000001');
    expect(lastEndpoint()).toEqual({
      endpoint: '/v1/admin/sessions/00000000-0000-0000-0000-000000000001',
      init: { method: 'DELETE' }
    });
  });
});
