import { describe, expect, it } from 'vitest';
import { DEFAULT_REDIRECT_PATH, parseRelativePath, safeRedirectPath } from './safe-redirect';

describe('safeRedirectPath', () => {
  it('keeps same-origin paths with query and hash', () => {
    expect(safeRedirectPath('/dashboard/users')).toBe('/dashboard/users');
    expect(safeRedirectPath('/dashboard/users?page=2#top')).toBe('/dashboard/users?page=2#top');
    expect(safeRedirectPath('/')).toBe('/');
    expect(safeRedirectPath('/dashboard/users?q=a b')).toBe('/dashboard/users?q=a b');
  });

  it('falls back for missing values', () => {
    expect(safeRedirectPath(null)).toBe(DEFAULT_REDIRECT_PATH);
    expect(safeRedirectPath(undefined)).toBe(DEFAULT_REDIRECT_PATH);
    expect(safeRedirectPath('')).toBe(DEFAULT_REDIRECT_PATH);
  });

  it.each([
    'https://evil.example',
    'http://evil.example/dashboard',
    '//evil.example',
    '/\\evil.example',
    '\\\\evil.example',
    'javascript:alert(1)',
    'data:text/html,hi',
    'dashboard/overview',
    '/\t/evil.example',
    '/\n/evil.example',
    ' /dashboard',
    '/dashboard/../auth',
    '/./evil'
  ])('rejects %j', (value) => {
    expect(safeRedirectPath(value)).toBe(DEFAULT_REDIRECT_PATH);
  });

  it('uses a custom fallback', () => {
    expect(safeRedirectPath('//evil.example', '/403')).toBe('/403');
  });
});

describe('parseRelativePath', () => {
  it('splits pathname, search and hash', () => {
    expect(parseRelativePath('/a/b?x=1&y=?#h?#')).toEqual({
      pathname: '/a/b',
      search: '?x=1&y=?',
      hash: '#h?#'
    });
    expect(parseRelativePath('/a#h')).toEqual({ pathname: '/a', search: '', hash: '#h' });
  });
});
