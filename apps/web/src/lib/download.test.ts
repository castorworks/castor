import { afterEach, describe, expect, it, vi } from 'vitest';
import { filenameFromDisposition } from './api-client';
import { exportQuery } from './download';

describe('exportQuery', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('keeps filters and sort, drops paging, adds format and time zone', () => {
    vi.spyOn(Intl.DateTimeFormat.prototype, 'resolvedOptions').mockReturnValue({
      timeZone: 'Asia/Shanghai'
    } as Intl.ResolvedDateTimeFormatOptions);
    const list = new URLSearchParams({
      page: '3',
      pageSize: '50',
      'log_type-in': 'LOGIN,LOGOUT',
      order: 'created_at desc'
    });

    const params = new URLSearchParams(exportQuery(list, 'csv'));

    expect(params.has('page')).toBe(false);
    expect(params.has('pageSize')).toBe(false);
    expect(params.get('log_type-in')).toBe('LOGIN,LOGOUT');
    expect(params.get('order')).toBe('created_at desc');
    expect(params.get('format')).toBe('csv');
    expect(params.get('tz')).toBe('Asia/Shanghai');
    // the list query itself is left untouched
    expect(list.get('page')).toBe('3');
  });
});

describe('filenameFromDisposition', () => {
  it('prefers the RFC 5987 UTF-8 filename', () => {
    expect(
      filenameFromDisposition(
        "attachment; filename*=UTF-8''users-20260923-101500.xlsx",
        'fallback.xlsx'
      )
    ).toBe('users-20260923-101500.xlsx');
    expect(
      filenameFromDisposition("attachment; filename*=UTF-8''%E7%94%A8%E6%88%B7.csv", 'x.csv')
    ).toBe('用户.csv');
  });

  it('falls back to a plain filename or the default', () => {
    expect(filenameFromDisposition('attachment; filename="a.csv"', 'x.csv')).toBe('a.csv');
    expect(filenameFromDisposition(null, 'x.csv')).toBe('x.csv');
    expect(filenameFromDisposition("attachment; filename*=UTF-8''%E0%A4%A", 'x.csv')).toBe('x.csv');
  });
});
