import { describe, expect, it } from 'vitest';
import { buildUserListQuery } from './service';

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
