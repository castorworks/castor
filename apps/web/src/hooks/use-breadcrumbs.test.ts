import { describe, expect, it, vi } from 'vitest';
import type { NavGroup } from '@/types';

vi.mock('next/navigation', () => ({ usePathname: () => '/' }));
vi.mock('@/hooks/use-nav', () => ({ useNavigationGroups: () => [] }));

const { breadcrumbTrail } = await import('./use-breadcrumbs');

// buildNavigation 给出的已是当前语言的菜单标题
const groups: NavGroup[] = [
  { label: '', items: [{ title: '仪表盘', url: '/dashboard/overview', items: [] }] },
  {
    label: '系统管理',
    items: [
      {
        title: '运营管理',
        url: '#',
        items: [
          { title: '通知管理', url: '/dashboard/admin-notifications', items: [] },
          { title: '定时任务', url: '/dashboard/jobs', items: [] }
        ]
      }
    ]
  },
  {
    label: '账户',
    items: [{ title: '登录历史', url: '/dashboard/account/login-history', items: [] }]
  }
];

describe('breadcrumbTrail', () => {
  it('uses the localized menu titles, linking only real pages', () => {
    expect(breadcrumbTrail(groups, '/dashboard/admin-notifications')).toEqual([
      { title: '系统管理' },
      { title: '运营管理' },
      { title: '通知管理', link: '/dashboard/admin-notifications' }
    ]);
  });

  it('covers sub-pages of a menu entry without matching look-alike prefixes', () => {
    expect(breadcrumbTrail(groups, '/dashboard/jobs/42').at(-1)?.title).toBe('定时任务');
    expect(breadcrumbTrail(groups, '/dashboard/jobsx')).toEqual([]);
  });

  it('omits the label of an ungrouped top-level item', () => {
    expect(breadcrumbTrail(groups, '/dashboard/overview')).toEqual([
      { title: '仪表盘', link: '/dashboard/overview' }
    ]);
  });

  it('returns nothing for pages outside the menu instead of URL segments', () => {
    expect(breadcrumbTrail(groups, '/dashboard/unknown-page')).toEqual([]);
  });
});
