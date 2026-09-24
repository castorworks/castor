import { describe, expect, it } from 'vitest';
import type { NavGroup } from '@/types';
import { filterNavGroups } from './use-nav';

const groups: NavGroup[] = [
  {
    label: 'Overview',
    items: [
      { title: 'Dashboard', url: '/dashboard/overview' },
      { title: 'Users', url: '/dashboard/users', access: { permission: '/users:GET' } },
      { title: 'Resources', url: '/dashboard/resources', access: { permission: '/resources:GET' } },
      {
        title: 'Audit Logs',
        url: '/dashboard/audit-logs',
        access: { permissions: ['/audit:GET', '/audit:EXPORT'] }
      }
    ]
  },
  {
    label: 'Account',
    items: [{ title: 'Login', url: '/auth/sign-in' }]
  }
];

describe('filterNavGroups', () => {
  it('hides navigation without an effective permission', () => {
    const filtered = filterNavGroups(groups, []);

    expect(filtered.flatMap((group) => group.items.map((item) => item.title))).toEqual([
      'Dashboard',
      'Login'
    ]);
  });

  it('keeps navigation for matching effective permissions', () => {
    const filtered = filterNavGroups(groups, ['/users:GET', '/resources:GET', '/audit:GET']);

    expect(filtered.flatMap((group) => group.items.map((item) => item.title))).toEqual([
      'Dashboard',
      'Users',
      'Resources',
      'Audit Logs',
      'Login'
    ]);
  });

  it('accepts any permission listed by a navigation item', () => {
    const filtered = filterNavGroups(groups, ['/audit:EXPORT']);

    expect(filtered.flatMap((group) => group.items.map((item) => item.title))).toEqual([
      'Dashboard',
      'Audit Logs',
      'Login'
    ]);
  });
});
